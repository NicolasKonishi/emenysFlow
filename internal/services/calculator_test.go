package services

import (
	"database/sql"
	"testing"
	"time"

	"buffetflow/internal/models"
)

func float(value float64) sql.NullFloat64 { return sql.NullFloat64{Float64: value, Valid: true} }

func initialRules() []models.CalculationRule {
	welcome, noWelcome := true, false
	return []models.CalculationRule{
		{ID: 1, RuleKey: "waiters", CalculationType: "group_of_people", Divisor: 18, Multiplier: 1, Active: true, Priority: 10},
		{ID: 2, RuleKey: "jugs", CalculationType: "per_waiter", Divisor: 1, Multiplier: 2, Active: true, Priority: 20},
		{ID: 3, RuleKey: "trays", CalculationType: "per_waiter", Divisor: 1, Multiplier: 1, Active: true, Priority: 30},
		{ID: 4, RuleKey: "disposable_cups", CalculationType: "per_person", Divisor: 1, Multiplier: 3, Active: true, Priority: 40},
		{ID: 5, RuleKey: "trash_bags", CalculationType: "fixed", BaseValue: 1, Divisor: 1, Multiplier: 1, MinimumQuantity: float(10), Active: true, Priority: 50},
		{ID: 6, RuleKey: "dinner_plates", CalculationType: "per_person", Divisor: 1, Multiplier: 1, Condition: models.RuleCondition{UseEventSafetyMargin: true}, Active: true, Priority: 60},
		{ID: 7, RuleKey: "soda_coke", CalculationType: "percentage_distribution", Divisor: 2, Multiplier: 1, Condition: models.RuleCondition{DistributionGroup: "soda", Percentage: 60}, Active: true, Priority: 70},
		{ID: 8, RuleKey: "soda_guarana", CalculationType: "percentage_distribution", Divisor: 2, Multiplier: 1, Condition: models.RuleCondition{DistributionGroup: "soda", Percentage: 40}, Active: true, Priority: 71},
		{ID: 9, RuleKey: "juice_orange", CalculationType: "percentage_distribution", Divisor: 2, Multiplier: 1, Condition: models.RuleCondition{DistributionGroup: "juice", Percentage: 50}, Active: true, Priority: 80},
		{ID: 10, RuleKey: "juice_grape", CalculationType: "percentage_distribution", Divisor: 2, Multiplier: 1, Condition: models.RuleCondition{DistributionGroup: "juice", Percentage: 50}, Active: true, Priority: 81},
		{ID: 11, RuleKey: "welcome_yes", CalculationType: "fixed", BaseValue: 2, Divisor: 1, Multiplier: 1, Condition: models.RuleCondition{WelcomeDrinks: &welcome}, Active: true, Priority: 90},
		{ID: 12, RuleKey: "welcome_no", CalculationType: "fixed", BaseValue: 1, Divisor: 1, Multiplier: 1, Condition: models.RuleCondition{WelcomeDrinks: &noWelcome}, Active: true, Priority: 91},
	}
}

func resultMap(results []RuleResult) map[string]float64 {
	values := make(map[string]float64, len(results))
	for _, result := range results {
		values[result.Rule.RuleKey] = result.Quantity
	}
	return values
}

func TestInitialCalculationsForTwoHundredGuests(t *testing.T) {
	results, err := CalculateRules(CalculationInput{GuestCount: 200, HasWelcomeDrinks: true, SafetyMarginPercent: 10}, initialRules())
	if err != nil {
		t.Fatal(err)
	}
	values := resultMap(results)
	tests := map[string]float64{
		"waiters":         12,
		"jugs":            24,
		"trays":           12,
		"disposable_cups": 600,
		"trash_bags":      10,
		"dinner_plates":   220,
		"soda_coke":       60,
		"soda_guarana":    40,
		"juice_orange":    50,
		"juice_grape":     50,
		"welcome_yes":     2,
	}
	for key, want := range tests {
		if got := values[key]; got != want {
			t.Errorf("%s: got %.0f, want %.0f", key, got, want)
		}
	}
	if _, exists := values["welcome_no"]; exists {
		t.Error("welcome-no rule should not run")
	}
}

func TestWaiterOverrideDrivesDependentEquipment(t *testing.T) {
	override := 9
	results, err := CalculateRules(CalculationInput{GuestCount: 200, WaiterOverride: &override}, initialRules())
	if err != nil {
		t.Fatal(err)
	}
	values := resultMap(results)
	if values["waiters"] != 9 {
		t.Fatalf("waiters got %.0f", values["waiters"])
	}
	if values["jugs"] != 18 {
		t.Fatalf("jugs got %.0f", values["jugs"])
	}
	if values["trays"] != 9 {
		t.Fatalf("trays got %.0f", values["trays"])
	}
}

func TestStaffRolesAndOperationalMarginForTwoHundredGuests(t *testing.T) {
	rules := []models.CalculationRule{
		{RuleKey: "waiters", CalculationType: "group_of_people", Divisor: 18, Multiplier: 1, Active: true, Priority: 1},
		{RuleKey: "coordinators", CalculationType: "fixed", BaseValue: 1, Multiplier: 1, Active: true, Priority: 2},
		{RuleKey: "leaders", CalculationType: "fixed", BaseValue: 1, Multiplier: 1, Active: true, Priority: 3},
		{RuleKey: "co_leaders", CalculationType: "fixed", BaseValue: 1, Multiplier: 1, Active: true, Priority: 4},
		{RuleKey: "jugs", CalculationType: "per_waiter", Multiplier: 2, Active: true, Priority: 5},
		{RuleKey: "trays", CalculationType: "per_waiter", Multiplier: 1, Active: true, Priority: 6},
		{RuleKey: "dinner_plates", CalculationType: "per_person", Multiplier: 1, Condition: models.RuleCondition{UseAdditionalGuestMargin: true, HasMainBuffet: boolPointer(true)}, Active: true, Priority: 7},
		{RuleKey: "coffee_spoon_kits", CalculationType: "group_of_people", Divisor: 1, Multiplier: 1, Condition: models.RuleCondition{UseAdditionalGuestMargin: true, UseCoffeeKitDivisor: true, CoffeeTable: boolPointer(true)}, Active: true, Priority: 8},
	}
	results, err := CalculateRules(CalculationInput{GuestCount: 200, AdditionalGuestMargin: 20, CoffeeKitDivisor: 50, HasMainBuffet: true, HasCoffeeTable: true}, rules)
	if err != nil {
		t.Fatal(err)
	}
	values := resultMap(results)
	if values["waiters"] != 12 || values["jugs"] != 24 || values["trays"] != 12 {
		t.Fatalf("regular staff equipment got waiters=%.0f jugs=%.0f trays=%.0f", values["waiters"], values["jugs"], values["trays"])
	}
	staff := StaffSummaryFromResults(results)
	if staff.Waiters != 12 || staff.Coordinators != 1 || staff.Leaders != 1 || staff.CoLeaders != 1 || staff.Total != 15 {
		t.Fatalf("staff summary got %+v", staff)
	}
	if values["dinner_plates"] != 220 || values["coffee_spoon_kits"] != 5 {
		t.Fatalf("operational margin got plates=%.0f coffee kits=%.0f", values["dinner_plates"], values["coffee_spoon_kits"])
	}
}

func boolPointer(value bool) *bool { return &value }

func TestPercentageDistributionUsesLargestRemainder(t *testing.T) {
	got, err := AllocatePercentages(7, []float64{60, 40})
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != 4 || got[1] != 3 {
		t.Fatalf("got %v, want [4 3]", got)
	}
	sum := 0
	for _, v := range got {
		sum += v
	}
	if sum != 7 {
		t.Fatalf("sum got %d", sum)
	}
}

func TestRoundingAndCapacities(t *testing.T) {
	if got := CapacityQuantity(200, 40); got != 5 {
		t.Errorf("cuba capacity: got %d", got)
	}
	if got := CapacityQuantity(201, 40); got != 6 {
		t.Errorf("cuba rounding: got %d", got)
	}
	if got := CapacityQuantity(200, 0); got != 1 {
		t.Errorf("missing capacity fallback: got %d", got)
	}
	if got := CapacityQuantity(77, 12); got != 7 {
		t.Errorf("container capacity: got %d", got)
	}
}

func TestInventoryAvailabilityAndReservationConflict(t *testing.T) {
	if got := AvailableQuantity(24, 8, 3); got != 13 {
		t.Fatalf("availability got %.0f", got)
	}
	base := time.Date(2026, 8, 10, 18, 0, 0, 0, time.UTC)
	existing := []models.ReservationWindow{{EventID: 1, StartsAt: base, EndsAt: base.Add(8 * time.Hour), Quantity: 10}}
	if !HasReservationConflict(existing, models.ReservationWindow{EventID: 2, StartsAt: base.Add(7 * time.Hour), EndsAt: base.Add(10 * time.Hour)}) {
		t.Error("expected overlap conflict")
	}
	if HasReservationConflict(existing, models.ReservationWindow{EventID: 2, StartsAt: base.Add(8 * time.Hour), EndsAt: base.Add(10 * time.Hour)}) {
		t.Error("touching windows should not conflict")
	}
}

func TestReturnDamageAndLoss(t *testing.T) {
	got, err := ApplyReturn(100, 2, 20, 16, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.NewStock != 99 {
		t.Errorf("new stock got %.0f", got.NewStock)
	}
	if got.NewDamaged != 4 {
		t.Errorf("damaged got %.0f", got.NewDamaged)
	}
	if got.Unaccounted != 1 {
		t.Errorf("unaccounted got %.0f", got.Unaccounted)
	}
	if _, err := ApplyReturn(100, 0, 10, 8, 2, 1); err == nil {
		t.Error("expected invalid return totals")
	}
}

func TestManualOverrideAndManualItemsSurviveRecalculationWithoutDuplicates(t *testing.T) {
	existing := []models.ChecklistItem{{SourceKey: "rule:1", RequiredQuantity: 15, ManualOverride: true, OverrideReason: "Acordado com a equipe", Status: "separated"}, {SourceKey: "manual:ice", Name: "Gelo", RequiredQuantity: 8, ManualItem: true}, {SourceKey: "rule:old", Name: "Antigo"}}
	calculated := []models.ChecklistItem{{SourceKey: "rule:1", CalculatedQuantity: 12, RequiredQuantity: 12}, {SourceKey: "rule:1", CalculatedQuantity: 12, RequiredQuantity: 12}, {SourceKey: "rule:2", CalculatedQuantity: 24, RequiredQuantity: 24}}
	got := ReconcileChecklist(existing, calculated)
	if len(got) != 3 {
		t.Fatalf("got %d items, want 3", len(got))
	}
	seen := map[string]int{}
	for _, item := range got {
		seen[item.SourceKey]++
		if item.SourceKey == "rule:1" {
			if item.RequiredQuantity != 15 || !item.ManualOverride || item.Status != "separated" {
				t.Errorf("manual override not preserved: %+v", item)
			}
		}
	}
	for key, count := range seen {
		if count != 1 {
			t.Errorf("duplicate %s", key)
		}
	}
	if seen["manual:ice"] != 1 {
		t.Error("manual item was not preserved")
	}
	if seen["rule:old"] != 0 {
		t.Error("obsolete automatic item should be removed")
	}
}

func TestCustomFormula(t *testing.T) {
	rule := models.CalculationRule{RuleKey: "custom", CalculationType: "custom", Divisor: 1, Multiplier: 1, Condition: models.RuleCondition{Formula: "max(10, ceil(guests / 25) * 2)"}, Active: true}
	results, err := CalculateRules(CalculationInput{GuestCount: 200}, []models.CalculationRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Quantity != 16 {
		t.Fatalf("custom formula result %+v", results)
	}
}
