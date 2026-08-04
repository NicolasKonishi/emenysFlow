package services

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"sort"
	"strconv"

	"buffetflow/internal/models"
)

type CalculationInput struct {
	GuestCount            int
	HasWelcomeDrinks      bool
	HasDadinhoTapioca     bool
	HasDecoration         bool
	SafetyMarginPercent   float64
	WaiterOverride        *int
	CoordinatorOverride   *int
	LeaderOverride        *int
	CoLeaderOverride      *int
	AdditionalGuestMargin float64
	CoffeeKitDivisor      float64
	HasMainBuffet         bool
	UsesGlassware         bool
	HasCoffeeTable        bool
	RequiresDessertSpoon  bool
}

type RuleResult struct {
	Rule     models.CalculationRule
	Quantity float64
	Origin   string
}

type allocationPart struct {
	index     int
	base      int
	remainder float64
}

func CalculateRules(input CalculationInput, rules []models.CalculationRule) ([]RuleResult, error) {
	if input.GuestCount <= 0 {
		return nil, fmt.Errorf("guest count must be positive")
	}
	results := make([]RuleResult, 0, len(rules))
	waiterCount := 0.0
	if input.WaiterOverride != nil {
		waiterCount = float64(*input.WaiterOverride)
	}

	grouped := make(map[string][]models.CalculationRule)
	groupOrder := make([]string, 0)
	for _, rule := range rules {
		if !rule.Active || !conditionMatches(rule, input) {
			continue
		}
		if rule.CalculationType == "percentage_distribution" {
			group := rule.Condition.DistributionGroup
			if group == "" {
				if err := json.Unmarshal([]byte(rule.ConditionJSON), &rule.Condition); err != nil {
					return nil, fmt.Errorf("parse condition for %s: %w", rule.RuleKey, err)
				}
				group = rule.Condition.DistributionGroup
			}
			if _, exists := grouped[group]; !exists {
				groupOrder = append(groupOrder, group)
			}
			grouped[group] = append(grouped[group], rule)
			continue
		}

		quantity, origin, err := calculateSingleRule(input, rule, waiterCount)
		if err != nil {
			return nil, err
		}
		if rule.RuleKey == "waiters" && input.WaiterOverride == nil {
			waiterCount = quantity
		}
		if rule.RuleKey == "waiters" && input.WaiterOverride != nil {
			quantity = float64(*input.WaiterOverride)
			origin = "Quantidade de garçons alterada no evento"
			waiterCount = quantity
		}
		if override := staffOverrideForRule(input, rule.RuleKey); override != nil {
			quantity = float64(*override)
			origin = "Quantidade alterada especificamente neste evento"
		}
		results = append(results, RuleResult{Rule: rule, Quantity: quantity, Origin: origin})
	}

	for _, group := range groupOrder {
		groupRules := grouped[group]
		if len(groupRules) == 0 {
			continue
		}
		baseRule := groupRules[0]
		total := int(math.Ceil(float64(input.GuestCount)/baseRule.Divisor*baseRule.Multiplier + baseRule.BaseValue))
		percentages := make([]float64, len(groupRules))
		for i, rule := range groupRules {
			percentages[i] = rule.Condition.Percentage
		}
		allocated, err := AllocatePercentages(total, percentages)
		if err != nil {
			return nil, fmt.Errorf("allocate group %s: %w", group, err)
		}
		for i, rule := range groupRules {
			quantity := applyBoundsAndSafety(float64(allocated[i]), rule, 0)
			origin := fmt.Sprintf("%d convidados ÷ %.2g; %.2g%% da distribuição %s", input.GuestCount, rule.Divisor, rule.Condition.Percentage, group)
			results = append(results, RuleResult{Rule: rule, Quantity: quantity, Origin: origin})
		}
	}

	sort.SliceStable(results, func(i, j int) bool { return results[i].Rule.Priority < results[j].Rule.Priority })
	return results, nil
}

func calculateSingleRule(input CalculationInput, rule models.CalculationRule, waiterCount float64) (float64, string, error) {
	basis := float64(input.GuestCount)
	if rule.Condition.UseAdditionalGuestMargin {
		basis += input.AdditionalGuestMargin
	}
	divisor := rule.Divisor
	if rule.Condition.UseCoffeeKitDivisor && input.CoffeeKitDivisor > 0 {
		divisor = input.CoffeeKitDivisor
	}
	var raw float64
	var origin string
	switch rule.CalculationType {
	case "fixed":
		raw = rule.BaseValue * rule.Multiplier
		origin = fmt.Sprintf("Quantidade fixa de %.2g por evento", raw)
	case "per_person":
		raw = rule.BaseValue + basis*rule.Multiplier
		if rule.Condition.UseAdditionalGuestMargin {
			origin = fmt.Sprintf("%d convidados + %.0f de margem operacional", input.GuestCount, input.AdditionalGuestMargin)
		} else {
			origin = fmt.Sprintf("%d convidados × %.2g", input.GuestCount, rule.Multiplier)
		}
	case "group_of_people":
		raw = math.Ceil(basis/divisor)*rule.Multiplier + rule.BaseValue
		origin = fmt.Sprintf("Arredondar para cima: %.0f pessoas atendidas ÷ %.2g × %.2g", basis, divisor, rule.Multiplier)
	case "per_waiter":
		raw = waiterCount*rule.Multiplier + rule.BaseValue
		origin = fmt.Sprintf("%.0f garçons × %.2g", waiterCount, rule.Multiplier)
	case "custom":
		condition := rule.Condition
		if rule.ConditionJSON != "" {
			_ = json.Unmarshal([]byte(rule.ConditionJSON), &condition)
		}
		if condition.Formula == "" {
			return 0, "", fmt.Errorf("custom formula is empty for rule %s", rule.RuleKey)
		}
		value, err := evaluateFormula(condition.Formula, map[string]float64{"guests": float64(input.GuestCount), "waiters": waiterCount, "safety_margin": input.SafetyMarginPercent, "additional_margin": input.AdditionalGuestMargin, "operational_guests": basis})
		if err != nil {
			return 0, "", fmt.Errorf("custom formula %s: %w", rule.RuleKey, err)
		}
		raw = value
		origin = "Fórmula personalizada: " + condition.Formula
	default:
		return 0, "", fmt.Errorf("calculation type %s requires a domain link not present in stage one", rule.CalculationType)
	}
	eventMargin := 0.0
	if rule.Condition.UseEventSafetyMargin {
		eventMargin = input.SafetyMarginPercent
	}
	return applyBoundsAndSafety(raw, rule, eventMargin), origin, nil
}

func evaluateFormula(expression string, variables map[string]float64) (float64, error) {
	node, err := parser.ParseExpr(expression)
	if err != nil {
		return 0, err
	}
	var eval func(ast.Expr) (float64, error)
	eval = func(expr ast.Expr) (float64, error) {
		switch value := expr.(type) {
		case *ast.BasicLit:
			if value.Kind != token.INT && value.Kind != token.FLOAT {
				return 0, fmt.Errorf("unsupported literal")
			}
			return strconv.ParseFloat(value.Value, 64)
		case *ast.Ident:
			result, ok := variables[value.Name]
			if !ok {
				return 0, fmt.Errorf("unknown variable %s", value.Name)
			}
			return result, nil
		case *ast.ParenExpr:
			return eval(value.X)
		case *ast.UnaryExpr:
			number, err := eval(value.X)
			if err != nil {
				return 0, err
			}
			if value.Op == token.SUB {
				return -number, nil
			}
			if value.Op == token.ADD {
				return number, nil
			}
			return 0, fmt.Errorf("unsupported unary operator")
		case *ast.BinaryExpr:
			left, err := eval(value.X)
			if err != nil {
				return 0, err
			}
			right, err := eval(value.Y)
			if err != nil {
				return 0, err
			}
			switch value.Op {
			case token.ADD:
				return left + right, nil
			case token.SUB:
				return left - right, nil
			case token.MUL:
				return left * right, nil
			case token.QUO:
				if right == 0 {
					return 0, fmt.Errorf("division by zero")
				}
				return left / right, nil
			}
			return 0, fmt.Errorf("unsupported operator")
		case *ast.CallExpr:
			name, ok := value.Fun.(*ast.Ident)
			if !ok {
				return 0, fmt.Errorf("unsupported function")
			}
			args := make([]float64, len(value.Args))
			for i, arg := range value.Args {
				args[i], err = eval(arg)
				if err != nil {
					return 0, err
				}
			}
			switch name.Name {
			case "ceil":
				if len(args) != 1 {
					return 0, fmt.Errorf("ceil expects one argument")
				}
				return math.Ceil(args[0]), nil
			case "floor":
				if len(args) != 1 {
					return 0, fmt.Errorf("floor expects one argument")
				}
				return math.Floor(args[0]), nil
			case "max":
				if len(args) != 2 {
					return 0, fmt.Errorf("max expects two arguments")
				}
				return math.Max(args[0], args[1]), nil
			case "min":
				if len(args) != 2 {
					return 0, fmt.Errorf("min expects two arguments")
				}
				return math.Min(args[0], args[1]), nil
			}
			return 0, fmt.Errorf("unsupported function %s", name.Name)
		}
		return 0, fmt.Errorf("unsupported expression")
	}
	return eval(node)
}

func conditionMatches(rule models.CalculationRule, input CalculationInput) bool {
	condition := rule.Condition
	if rule.ConditionJSON != "" && rule.ConditionJSON != "{}" {
		_ = json.Unmarshal([]byte(rule.ConditionJSON), &condition)
	}
	if condition.WelcomeDrinks != nil && *condition.WelcomeDrinks != input.HasWelcomeDrinks {
		return false
	}
	if condition.DadinhoTapioca != nil && *condition.DadinhoTapioca != input.HasDadinhoTapioca {
		return false
	}
	if condition.Decoration != nil && *condition.Decoration != input.HasDecoration {
		return false
	}
	if condition.HasMainBuffet != nil && *condition.HasMainBuffet != input.HasMainBuffet {
		return false
	}
	if condition.UsesGlassware != nil && *condition.UsesGlassware != input.UsesGlassware {
		return false
	}
	if condition.CoffeeTable != nil && *condition.CoffeeTable != input.HasCoffeeTable {
		return false
	}
	if condition.RequiresDessertSpoon != nil && *condition.RequiresDessertSpoon != input.RequiresDessertSpoon {
		return false
	}
	return true
}

func staffOverrideForRule(input CalculationInput, key string) *int {
	switch key {
	case "waiters":
		return input.WaiterOverride
	case "coordinators":
		return input.CoordinatorOverride
	case "leaders":
		return input.LeaderOverride
	case "co_leaders":
		return input.CoLeaderOverride
	default:
		return nil
	}
}

func StaffSummaryFromResults(results []RuleResult) models.StaffSummary {
	var summary models.StaffSummary
	for _, result := range results {
		value := int(math.Ceil(result.Quantity))
		switch result.Rule.RuleKey {
		case "waiters":
			summary.Waiters = value
		case "coordinators":
			summary.Coordinators = value
		case "leaders":
			summary.Leaders = value
		case "co_leaders":
			summary.CoLeaders = value
		}
	}
	summary.Total = summary.Waiters + summary.Coordinators + summary.Leaders + summary.CoLeaders
	return summary
}

func applyBoundsAndSafety(raw float64, rule models.CalculationRule, eventMargin float64) float64 {
	margin := rule.SafetyPercent + eventMargin
	if margin > 0 {
		raw *= 1 + margin/100
	}
	result := math.Ceil(raw - 1e-9)
	if rule.MinimumQuantity.Valid && result < rule.MinimumQuantity.Float64 {
		result = rule.MinimumQuantity.Float64
	}
	if rule.MaximumQuantity.Valid && result > rule.MaximumQuantity.Float64 {
		result = rule.MaximumQuantity.Float64
	}
	return result
}

func AllocatePercentages(total int, percentages []float64) ([]int, error) {
	if total < 0 {
		return nil, fmt.Errorf("total cannot be negative")
	}
	if len(percentages) == 0 {
		return nil, fmt.Errorf("at least one percentage is required")
	}
	sum := 0.0
	for _, percentage := range percentages {
		if percentage < 0 {
			return nil, fmt.Errorf("percentage cannot be negative")
		}
		sum += percentage
	}
	if sum <= 0 {
		return nil, fmt.Errorf("percentage sum must be positive")
	}
	result := make([]int, len(percentages))
	parts := make([]allocationPart, len(percentages))
	allocated := 0
	for i, percentage := range percentages {
		exact := float64(total) * percentage / sum
		base := int(math.Floor(exact))
		result[i], allocated = base, allocated+base
		parts[i] = allocationPart{index: i, base: base, remainder: exact - float64(base)}
	}
	sort.SliceStable(parts, func(i, j int) bool { return parts[i].remainder > parts[j].remainder })
	for i := 0; i < total-allocated; i++ {
		result[parts[i%len(parts)].index]++
	}
	return result, nil
}

func CapacityQuantity(portions int, capacity float64) int {
	if portions <= 0 {
		return 0
	}
	if capacity <= 0 {
		return 1
	}
	return int(math.Ceil(float64(portions) / capacity))
}

func AvailableQuantity(stock, reserved, damaged float64) float64 {
	return math.Max(0, stock-reserved-damaged)
}

func HasReservationConflict(existing []models.ReservationWindow, candidate models.ReservationWindow) bool {
	for _, current := range existing {
		if current.EventID == candidate.EventID {
			continue
		}
		if current.StartsAt.Before(candidate.EndsAt) && current.EndsAt.After(candidate.StartsAt) {
			return true
		}
	}
	return false
}

func ApplyReturn(stock, damagedStock, sent, returned, damaged, lost float64) (models.ReturnResult, error) {
	if stock < 0 || damagedStock < 0 || sent < 0 || returned < 0 || damaged < 0 || lost < 0 {
		return models.ReturnResult{}, fmt.Errorf("quantities cannot be negative")
	}
	if returned+damaged+lost > sent {
		return models.ReturnResult{}, fmt.Errorf("return quantities exceed sent quantity")
	}
	return models.ReturnResult{
		NewStock:          stock - lost,
		NewDamaged:        damagedStock + damaged,
		Unaccounted:       sent - returned - damaged - lost,
		ReturnedAvailable: returned,
	}, nil
}

func ReconcileChecklist(existing, calculated []models.ChecklistItem) []models.ChecklistItem {
	existingByKey := make(map[string]models.ChecklistItem, len(existing))
	for _, item := range existing {
		existingByKey[item.SourceKey] = item
	}
	result := make([]models.ChecklistItem, 0, len(calculated)+len(existing))
	seen := make(map[string]bool)
	for _, item := range calculated {
		if seen[item.SourceKey] {
			continue
		}
		seen[item.SourceKey] = true
		if old, found := existingByKey[item.SourceKey]; found {
			if old.ManualOverride {
				item.RequiredQuantity, item.ManualOverride, item.OverrideReason = old.RequiredQuantity, true, old.OverrideReason
			}
			item.Status = old.Status
		}
		result = append(result, item)
	}
	for _, item := range existing {
		if seen[item.SourceKey] {
			continue
		}
		if item.ManualItem || item.ManualOverride {
			result = append(result, item)
		}
	}
	return result
}
