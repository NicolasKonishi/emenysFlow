package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"buffetflow/internal/models"
)

func scanRule(scanner interface{ Scan(...any) error }) (models.CalculationRule, error) {
	var rule models.CalculationRule
	var active int
	err := scanner.Scan(&rule.ID, &rule.RuleKey, &rule.Name, &rule.Description, &rule.CategoryID, &rule.CategoryName,
		&rule.TriggerEvent, &rule.CalculationType, &rule.BaseValue, &rule.Divisor, &rule.Multiplier,
		&rule.MinimumQuantity, &rule.MaximumQuantity, &rule.SafetyPercent, &rule.ConditionJSON,
		&rule.ResultInventoryItemID, &rule.ResultItemName, &rule.ResultUnit, &rule.Priority, &active)
	rule.Active = active == 1
	if strings.TrimSpace(rule.ConditionJSON) != "" {
		_ = json.Unmarshal([]byte(rule.ConditionJSON), &rule.Condition)
	}
	return rule, err
}

const ruleSelect = `SELECT r.id,r.rule_key,r.name,r.description,r.category_id,c.name,r.trigger_event,r.calculation_type,
	r.base_value,r.divisor,r.multiplier,r.minimum_quantity,r.maximum_quantity,r.safety_percent,r.condition_json,
	r.result_inventory_item_id,i.name,i.unit,r.priority,r.active
	FROM calculation_rules r JOIN inventory_categories c ON c.id=r.category_id JOIN inventory_items i ON i.id=r.result_inventory_item_id`

func (s *Store) ListRules(ctx context.Context, includeInactive bool) ([]models.CalculationRule, error) {
	rows, err := s.db.QueryContext(ctx, ruleSelect+` WHERE (?=1 OR r.active=1) ORDER BY r.priority,r.name`, includeInactive)
	if err != nil {
		return nil, fmt.Errorf("list rules: %w", err)
	}
	defer rows.Close()
	var result []models.CalculationRule
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, rule)
	}
	return result, rows.Err()
}

func (s *Store) GetRule(ctx context.Context, id int64) (models.CalculationRule, error) {
	return scanRule(s.db.QueryRowContext(ctx, ruleSelect+` WHERE r.id=?`, id))
}

func (s *Store) SaveRule(ctx context.Context, rule *models.CalculationRule, userID int64) error {
	if rule.Divisor <= 0 {
		return fmt.Errorf("divisor must be greater than zero")
	}
	if strings.TrimSpace(rule.ConditionJSON) == "" {
		rule.ConditionJSON = "{}"
	}
	if !json.Valid([]byte(rule.ConditionJSON)) {
		return fmt.Errorf("condition must be valid JSON")
	}
	now := nowString()
	minimum, maximum := any(nil), any(nil)
	if rule.MinimumQuantity.Valid {
		minimum = rule.MinimumQuantity.Float64
	}
	if rule.MaximumQuantity.Valid {
		maximum = rule.MaximumQuantity.Float64
	}
	if rule.ID == 0 {
		result, err := s.db.ExecContext(ctx, `INSERT INTO calculation_rules(rule_key,name,description,category_id,trigger_event,calculation_type,
			base_value,divisor,multiplier,minimum_quantity,maximum_quantity,safety_percent,condition_json,result_inventory_item_id,priority,active,created_at,updated_at)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, rule.RuleKey, rule.Name, rule.Description, rule.CategoryID, "checklist_generation", rule.CalculationType,
			rule.BaseValue, rule.Divisor, rule.Multiplier, minimum, maximum, rule.SafetyPercent, rule.ConditionJSON, rule.ResultInventoryItemID, rule.Priority, rule.Active, now, now)
		if err != nil {
			return fmt.Errorf("create rule: %w", err)
		}
		rule.ID, err = result.LastInsertId()
		return err
	}
	_, err := s.db.ExecContext(ctx, `UPDATE calculation_rules SET rule_key=?,name=?,description=?,category_id=?,calculation_type=?,base_value=?,divisor=?,
		multiplier=?,minimum_quantity=?,maximum_quantity=?,safety_percent=?,condition_json=?,result_inventory_item_id=?,priority=?,active=?,updated_at=? WHERE id=?`,
		rule.RuleKey, rule.Name, rule.Description, rule.CategoryID, rule.CalculationType, rule.BaseValue, rule.Divisor, rule.Multiplier, minimum, maximum,
		rule.SafetyPercent, rule.ConditionJSON, rule.ResultInventoryItemID, rule.Priority, rule.Active, now, rule.ID)
	if err != nil {
		return fmt.Errorf("update rule: %w", err)
	}
	return nil
}

func (s *Store) ToggleRule(ctx context.Context, id, userID int64) error {
	_, err := s.db.ExecContext(ctx, "UPDATE calculation_rules SET active=CASE active WHEN 1 THEN 0 ELSE 1 END,updated_at=? WHERE id=?", nowString(), id)
	return err
}

func nullFloat(value string) sql.NullFloat64 {
	if strings.TrimSpace(value) == "" {
		return sql.NullFloat64{}
	}
	var result sql.NullFloat64
	if _, err := fmt.Sscan(value, &result.Float64); err == nil {
		result.Valid = true
	}
	return result
}
