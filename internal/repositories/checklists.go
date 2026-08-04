package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"buffetflow/internal/models"
)

func (s *Store) RegenerateChecklist(ctx context.Context, eventID int64, generated []models.ChecklistItem) (int64, error) {
	var checklistID int64
	err := withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		err := tx.QueryRowContext(ctx, "SELECT id FROM checklists WHERE event_id=?", eventID).Scan(&checklistID)
		if err == sql.ErrNoRows {
			result, err := tx.ExecContext(ctx, "INSERT INTO checklists(event_id,version,generated_at,updated_at) VALUES(?,1,?,?)", eventID, now, now)
			if err != nil {
				return err
			}
			checklistID, err = result.LastInsertId()
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			if _, err := tx.ExecContext(ctx, "UPDATE checklists SET version=version+1,generated_at=?,updated_at=? WHERE id=?", now, now, checklistID); err != nil {
				return err
			}
		}

		activeKeys := make([]string, 0, len(generated))
		for _, item := range generated {
			activeKeys = append(activeKeys, item.SourceKey)
			var existingID int64
			var required float64
			var manualOverride int
			err := tx.QueryRowContext(ctx, "SELECT id,required_quantity,manual_override FROM checklist_items WHERE checklist_id=? AND source_key=?", checklistID, item.SourceKey).Scan(&existingID, &required, &manualOverride)
			if err != nil && err != sql.ErrNoRows {
				return err
			}
			if err == nil {
				if manualOverride == 1 {
					item.RequiredQuantity = required
					item.MissingQuantity = maxFloat(0, required-item.AvailableQuantity)
				}
				_, err = tx.ExecContext(ctx, `UPDATE checklist_items SET inventory_item_id=?,category_id=?,source_rule_id=?,name=?,unit=?,calculated_quantity=?,
					required_quantity=?,available_quantity=?,reserved_elsewhere_quantity=?,missing_quantity=?,calculation_origin=?,item_kind=?,location_snapshot=?,active=1,row_version=row_version+1,updated_at=? WHERE id=?`,
					nullInt64(item.InventoryItemID), item.CategoryID, nullInt64(item.SourceRuleID), item.Name, item.Unit, item.CalculatedQuantity, item.RequiredQuantity,
					item.AvailableQuantity, item.ReservedElsewhereQuantity, item.MissingQuantity, item.CalculationOrigin, item.ItemKind, item.LocationSnapshot, now, existingID)
				if err != nil {
					return err
				}
				continue
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO checklist_items(checklist_id,inventory_item_id,category_id,source_rule_id,source_key,name,unit,
				calculated_quantity,required_quantity,available_quantity,reserved_elsewhere_quantity,missing_quantity,calculation_origin,notes,status,
				item_kind,location_snapshot,manual_item,manual_override,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,'pending',?,?,0,0,?,?)`,
				checklistID, nullInt64(item.InventoryItemID), item.CategoryID, nullInt64(item.SourceRuleID), item.SourceKey, item.Name, item.Unit,
				item.CalculatedQuantity, item.RequiredQuantity, item.AvailableQuantity, item.ReservedElsewhereQuantity, item.MissingQuantity, item.CalculationOrigin,
				item.Notes, item.ItemKind, item.LocationSnapshot, now, now)
			if err != nil {
				return err
			}
		}

		if len(activeKeys) == 0 {
			_, err = tx.ExecContext(ctx, "UPDATE checklist_items SET active=0,row_version=row_version+1,updated_at=? WHERE checklist_id=? AND manual_item=0 AND manual_override=0", now, checklistID)
			return err
		}
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(activeKeys)), ",")
		args := make([]any, 0, len(activeKeys)+1)
		args = append(args, checklistID)
		for _, key := range activeKeys {
			args = append(args, key)
		}
		args = append([]any{now}, args...)
		_, err = tx.ExecContext(ctx, "UPDATE checklist_items SET active=0,row_version=row_version+1,updated_at=? WHERE checklist_id=? AND manual_item=0 AND manual_override=0 AND source_key NOT IN ("+placeholders+")", args...)
		return err
	})
	return checklistID, err
}

func (s *Store) GetChecklistByEvent(ctx context.Context, eventID int64) (models.Checklist, error) {
	var result models.Checklist
	var generated, updated string
	err := s.db.QueryRowContext(ctx, "SELECT id,event_id,version,generated_at,updated_at FROM checklists WHERE event_id=?", eventID).Scan(&result.ID, &result.EventID, &result.Version, &generated, &updated)
	if err != nil {
		return result, err
	}
	result.GeneratedAt, result.UpdatedAt = parseTime(generated), parseTime(updated)
	rows, err := s.db.QueryContext(ctx, `SELECT ci.id,ci.checklist_id,ci.inventory_item_id,ci.category_id,c.name,c.sort_order,ci.source_rule_id,ci.source_key,
		ci.name,ci.unit,ci.calculated_quantity,ci.required_quantity,ci.available_quantity,ci.reserved_elsewhere_quantity,ci.missing_quantity,
		ci.calculation_origin,ci.notes,ci.status,ci.item_kind,ci.location_snapshot,ci.manual_item,ci.manual_override,ci.override_reason,
		ci.separated_quantity,ci.loaded_quantity,COALESCE(ci.loading_decision,''),ci.loading_missing_quantity,
		ci.returned_quantity,ci.damaged_quantity,ci.lost_quantity,ci.active,ci.row_version,
		ci.separated_by,COALESCE(separated_user.name,''),ci.separated_at,ci.loaded_by,COALESCE(loaded_user.name,''),ci.loaded_at,
		ci.override_by,ci.override_at
		FROM checklist_items ci JOIN inventory_categories c ON c.id=ci.category_id
		LEFT JOIN users separated_user ON separated_user.id=ci.separated_by
		LEFT JOIN users loaded_user ON loaded_user.id=ci.loaded_by
		WHERE ci.checklist_id=? AND ci.active=1 ORDER BY c.sort_order,ci.name`, result.ID)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var item models.ChecklistItem
		var manualItem, manualOverride int
		var active int
		var separatedAt, loadedAt, overrideAt sql.NullString
		if err := rows.Scan(&item.ID, &item.ChecklistID, &item.InventoryItemID, &item.CategoryID, &item.CategoryName, &item.CategorySortOrder, &item.SourceRuleID, &item.SourceKey,
			&item.Name, &item.Unit, &item.CalculatedQuantity, &item.RequiredQuantity, &item.AvailableQuantity, &item.ReservedElsewhereQuantity, &item.MissingQuantity,
			&item.CalculationOrigin, &item.Notes, &item.Status, &item.ItemKind, &item.LocationSnapshot, &manualItem, &manualOverride, &item.OverrideReason,
			&item.SeparatedQuantity, &item.LoadedQuantity, &item.LoadingDecision, &item.LoadingMissingQuantity,
			&item.ReturnedQuantity, &item.DamagedQuantity, &item.LostQuantity, &active, &item.RowVersion,
			&item.SeparatedBy, &item.SeparatedByName, &separatedAt, &item.LoadedBy, &item.LoadedByName, &loadedAt,
			&item.OverrideBy, &overrideAt); err != nil {
			return result, err
		}
		item.ManualItem, item.ManualOverride = manualItem == 1, manualOverride == 1
		item.Active = active == 1
		if separatedAt.Valid {
			item.SeparatedAt = parseTime(separatedAt.String)
		}
		if loadedAt.Valid {
			item.LoadedAt = parseTime(loadedAt.String)
		}
		if overrideAt.Valid {
			item.OverrideAt = parseTime(overrideAt.String)
		}
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	result.Progress = checklistProgress(result.Items)
	return result, nil
}

func (s *Store) UpdateChecklistItemStatus(ctx context.Context, itemID int64, status string, userID int64) error {
	valid := map[string]bool{"pending": true, "separating": true, "separated": true, "checked": true, "loaded": true, "at_event": true, "returned": true, "damaged": true, "lost": true, "not_applicable": true}
	if !valid[status] {
		return fmt.Errorf("invalid status")
	}
	now := nowString()
	_, err := s.db.ExecContext(ctx, `UPDATE checklist_items SET status=?,
		separated_quantity=CASE WHEN ?='separated' THEN required_quantity WHEN ?='pending' THEN 0 ELSE separated_quantity END,
		separated_by=CASE WHEN ?='separated' THEN ? WHEN ?='pending' THEN NULL ELSE separated_by END,
		separated_at=CASE WHEN ?='separated' THEN ? WHEN ?='pending' THEN NULL ELSE separated_at END,
		row_version=row_version+1,updated_at=? WHERE id=?`,
		status, status, status, status, nullableUserID(userID), status, status, now, status, now, itemID)
	return err
}

func (s *Store) UpdateChecklistItemsStatus(ctx context.Context, eventID int64, itemIDs []int64, status string, userID int64) error {
	if status != "pending" && status != "separated" {
		return fmt.Errorf("invalid group status")
	}
	if len(itemIDs) == 0 {
		return fmt.Errorf("empty checklist group")
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(itemIDs)), ",")
	now := nowString()
	args := []any{status, status, status, status, nullableUserID(userID), status, status, now, status, now, eventID}
	for _, id := range itemIDs {
		args = append(args, id)
	}
	result, err := s.db.ExecContext(ctx, `UPDATE checklist_items SET status=?,
		separated_quantity=CASE WHEN ?='separated' THEN required_quantity WHEN ?='pending' THEN 0 ELSE separated_quantity END,
		separated_by=CASE WHEN ?='separated' THEN ? WHEN ?='pending' THEN NULL ELSE separated_by END,
		separated_at=CASE WHEN ?='separated' THEN ? WHEN ?='pending' THEN NULL ELSE separated_at END,
		row_version=row_version+1,updated_at=?
		WHERE checklist_id=(SELECT id FROM checklists WHERE event_id=?) AND id IN (`+placeholders+`)`, args...)
	if err != nil {
		return err
	}
	updated, _ := result.RowsAffected()
	if updated != int64(len(itemIDs)) {
		return fmt.Errorf("checklist group changed while updating")
	}
	return nil
}

func (s *Store) OverrideChecklistItem(ctx context.Context, itemID int64, quantity float64, reason string, userID int64) error {
	if quantity < 0 || strings.TrimSpace(reason) == "" {
		return fmt.Errorf("quantity and reason are required")
	}
	now := nowString()
	_, err := s.db.ExecContext(ctx, `UPDATE checklist_items SET required_quantity=?,missing_quantity=MAX(0,?-available_quantity),manual_override=1,
		override_reason=?,override_by=?,override_at=?,updated_at=? WHERE id=?`, quantity, quantity, reason, nullableUserID(userID), now, now, itemID)
	return err
}

func checklistProgress(items []models.ChecklistItem) models.ChecklistProgress {
	result := models.ChecklistProgress{Total: len(items)}
	for _, item := range items {
		if item.MissingQuantity > 0 {
			result.Missing++
		}
		switch item.Status {
		case "separated", "checked", "loaded", "at_event", "returned", "not_applicable":
			result.Completed++
		default:
			result.Pending++
		}
	}
	if result.Total > 0 {
		result.Percentage = int(float64(result.Completed)*100/float64(result.Total) + 0.5)
		separated, loaded := 0.0, 0.0
		for _, item := range items {
			if item.RequiredQuantity <= 0 {
				continue
			}
			separated += minFloat(1, item.SeparatedQuantity/item.RequiredQuantity)
			loaded += minFloat(1, item.LoadedQuantity/item.RequiredQuantity)
		}
		result.SeparationPercentage = int(separated*100/float64(result.Total) + 0.5)
		result.LoadingPercentage = int(loaded*100/float64(result.Total) + 0.5)
	}
	return result
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func nullInt64(value sql.NullInt64) any {
	if value.Valid {
		return value.Int64
	}
	return nil
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
