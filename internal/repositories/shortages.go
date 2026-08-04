package repositories

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"buffetflow/internal/models"
)

func (s *Store) ListEventShortages(ctx context.Context, eventID int64, includeClosed bool) ([]models.ChecklistShortage, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,checklist_item_id,event_id,missing_quantity,reason,resolution_type,status,
		responsible_user_id,responsible_name,due_at,supplier_id,supplier_name,estimated_cost_cents,notes,automatic,
		COALESCE(resolution_destination,''),resolved_by,resolved_at,row_version,created_at,updated_at
		FROM checklist_shortages WHERE event_id=? AND (?=1 OR status NOT IN ('resolved','cancelled')) ORDER BY status,due_at,created_at`, eventID, includeClosed)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.ChecklistShortage
	for rows.Next() {
		var item models.ChecklistShortage
		var due, resolved sql.NullString
		var automatic int
		var created, updated string
		if err := rows.Scan(&item.ID, &item.ChecklistItemID, &item.EventID, &item.MissingQuantity, &item.Reason, &item.ResolutionType, &item.Status,
			&item.ResponsibleUserID, &item.ResponsibleName, &due, &item.SupplierID, &item.SupplierName, &item.EstimatedCostCents, &item.Notes, &automatic,
			&item.ResolutionDestination, &item.ResolvedBy, &resolved, &item.RowVersion, &created, &updated); err != nil {
			return nil, err
		}
		if due.Valid {
			item.DueAt = parseTime(due.String)
		}
		if resolved.Valid {
			item.ResolvedAt = parseTime(resolved.String)
		}
		item.CreatedAt, item.UpdatedAt = parseTime(created), parseTime(updated)
		item.Automatic = automatic == 1
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) EnsureCalculatedShortages(ctx context.Context, eventID int64) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		rows, err := tx.QueryContext(ctx, `SELECT item.id,item.missing_quantity FROM checklist_items item JOIN checklists checklist ON checklist.id=item.checklist_id WHERE checklist.event_id=? AND item.active=1 AND item.missing_quantity>0`, eventID)
		if err != nil {
			return err
		}
		type missingItem struct {
			id       int64
			quantity float64
		}
		var missing []missingItem
		for rows.Next() {
			var item missingItem
			if err := rows.Scan(&item.id, &item.quantity); err != nil {
				rows.Close()
				return err
			}
			missing = append(missing, item)
		}
		if err := rows.Close(); err != nil {
			return err
		}
		for _, item := range missing {
			var count int
			if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM checklist_shortages WHERE checklist_item_id=? AND status NOT IN ('resolved','cancelled')`, item.id).Scan(&count); err != nil {
				return err
			}
			if count > 0 {
				continue
			}
			result, err := tx.ExecContext(ctx, `INSERT INTO checklist_shortages(checklist_item_id,event_id,missing_quantity,reason,resolution_type,status,notes,automatic,row_version,created_at,updated_at) VALUES(?,?,?,'Estoque disponível insuficiente.','other','pending','Gerada automaticamente durante o recálculo.',1,1,?,?)`, item.id, eventID, item.quantity, now, now)
			if err != nil {
				return err
			}
			shortageID, _ := result.LastInsertId()
			if _, err = tx.ExecContext(ctx, `INSERT INTO checklist_shortage_history(shortage_id,new_status,notes,created_at) VALUES(?,'pending','Falta detectada automaticamente.',?)`, shortageID, now); err != nil {
				return err
			}
		}
		_, err = tx.ExecContext(ctx, `UPDATE checklist_shortages SET status='resolved',resolution_destination='separation',resolved_at=?,notes=notes||' Resolvida automaticamente após recálculo.',row_version=row_version+1,updated_at=? WHERE event_id=? AND automatic=1 AND status NOT IN ('resolved','cancelled') AND checklist_item_id IN (SELECT item.id FROM checklist_items item JOIN checklists checklist ON checklist.id=item.checklist_id WHERE checklist.event_id=? AND (item.active=0 OR item.missing_quantity<=0))`, now, now, eventID, eventID)
		return err
	})
}

func (s *Store) SaveChecklistShortage(ctx context.Context, item models.ChecklistShortage, userID int64) error {
	if item.MissingQuantity <= 0 || strings.TrimSpace(item.Reason) == "" {
		return fmt.Errorf("missing quantity and reason are required")
	}
	validResolution := map[string]bool{"purchase": true, "rental": true, "substitution": true, "stock_transfer": true, "production": true, "wait_return": true, "other": true}
	if !validResolution[item.ResolutionType] {
		return fmt.Errorf("invalid resolution type")
	}
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		var required float64
		if err := tx.QueryRowContext(ctx, `SELECT checklist_item.required_quantity FROM checklist_items checklist_item JOIN checklists checklist ON checklist.id=checklist_item.checklist_id WHERE checklist_item.id=? AND checklist.event_id=? AND checklist_item.active=1`, item.ChecklistItemID, item.EventID).Scan(&required); err != nil {
			return err
		}
		if item.MissingQuantity > required+0.0001 {
			return fmt.Errorf("missing quantity exceeds required quantity")
		}
		now := nowString()
		var existingID int64
		err := tx.QueryRowContext(ctx, `SELECT id FROM checklist_shortages WHERE checklist_item_id=? AND status NOT IN ('resolved','cancelled')`, item.ChecklistItemID).Scan(&existingID)
		due := any(nil)
		if !item.DueAt.IsZero() {
			due = item.DueAt.UTC().Format(time.RFC3339)
		}
		estimated := any(nil)
		if item.EstimatedCostCents.Valid {
			estimated = item.EstimatedCostCents.Int64
		}
		if err == sql.ErrNoRows {
			result, insertErr := tx.ExecContext(ctx, `INSERT INTO checklist_shortages(checklist_item_id,event_id,missing_quantity,reason,resolution_type,status,responsible_user_id,responsible_name,due_at,supplier_id,supplier_name,estimated_cost_cents,notes,row_version,created_by,created_at,updated_at) VALUES(?,?,?,?,?,'pending',?,?,?,?,?,?,?,1,?,?,?)`, item.ChecklistItemID, item.EventID, item.MissingQuantity, item.Reason, item.ResolutionType, nullableUserID(item.ResponsibleUserID.Int64), item.ResponsibleName, due, nullInt64(item.SupplierID), item.SupplierName, estimated, item.Notes, nullableUserID(userID), now, now)
			if insertErr != nil {
				return insertErr
			}
			existingID, _ = result.LastInsertId()
		} else if err != nil {
			return err
		} else {
			if _, err = tx.ExecContext(ctx, `UPDATE checklist_shortages SET missing_quantity=?,reason=?,resolution_type=?,responsible_user_id=?,responsible_name=?,due_at=?,supplier_id=?,supplier_name=?,estimated_cost_cents=?,notes=?,row_version=row_version+1,updated_at=? WHERE id=?`, item.MissingQuantity, item.Reason, item.ResolutionType, nullInt64(item.ResponsibleUserID), item.ResponsibleName, due, nullInt64(item.SupplierID), item.SupplierName, estimated, item.Notes, now, existingID); err != nil {
				return err
			}
		}
		if _, err = tx.ExecContext(ctx, `UPDATE checklist_items SET loading_decision='missing',loading_missing_quantity=?,updated_by=?,row_version=row_version+1,updated_at=? WHERE id=?`, item.MissingQuantity, nullableUserID(userID), now, item.ChecklistItemID); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO checklist_shortage_history(shortage_id,previous_status,new_status,notes,changed_by,created_at) VALUES(?,NULL,'pending',?,?,?)`, existingID, item.Notes, nullableUserID(userID), now)
		return err
	})
}

func (s *Store) UpdateShortageStatus(ctx context.Context, eventID, shortageID int64, status, destination, notes string, userID int64) error {
	valid := map[string]bool{"pending": true, "purchasing": true, "renting": true, "resolved": true, "cancelled": true}
	if !valid[status] {
		return fmt.Errorf("invalid shortage status")
	}
	if status == "resolved" && destination != "separation" && destination != "loading" {
		return fmt.Errorf("resolution destination is required")
	}
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		var previous string
		var checklistItemID int64
		if err := tx.QueryRowContext(ctx, `SELECT status,checklist_item_id FROM checklist_shortages WHERE id=? AND event_id=?`, shortageID, eventID).Scan(&previous, &checklistItemID); err != nil {
			return err
		}
		now := nowString()
		resolvedAt := any(nil)
		resolvedBy := any(nil)
		resolutionDestination := any(nil)
		if status == "resolved" {
			resolvedAt = now
			resolvedBy = nullableUserID(userID)
			resolutionDestination = destination
		}
		if _, err := tx.ExecContext(ctx, `UPDATE checklist_shortages SET status=?,resolution_destination=?,resolved_by=?,resolved_at=?,notes=CASE WHEN ?<>'' THEN ? ELSE notes END,row_version=row_version+1,updated_at=? WHERE id=?`, status, resolutionDestination, resolvedBy, resolvedAt, notes, notes, now, shortageID); err != nil {
			return err
		}
		if status == "resolved" || status == "cancelled" {
			if destination == "loading" && status == "resolved" {
				if _, err := tx.ExecContext(ctx, `UPDATE checklist_items SET separated_quantity=required_quantity,separated_by=?,separated_at=?,status='separated',loading_decision=NULL,loading_missing_quantity=0,updated_by=?,row_version=row_version+1,updated_at=? WHERE id=?`, nullableUserID(userID), now, nullableUserID(userID), now, checklistItemID); err != nil {
					return err
				}
			} else {
				if _, err := tx.ExecContext(ctx, `UPDATE checklist_items SET status=CASE WHEN separated_quantity>=required_quantity THEN 'separated' ELSE 'separating' END,loading_decision=NULL,loading_missing_quantity=0,updated_by=?,row_version=row_version+1,updated_at=? WHERE id=?`, nullableUserID(userID), now, checklistItemID); err != nil {
					return err
				}
			}
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO checklist_shortage_history(shortage_id,previous_status,new_status,notes,changed_by,created_at) VALUES(?,?,?,?,?,?)`, shortageID, previous, status, notes, nullableUserID(userID), now)
		return err
	})
}

func (s *Store) SaveOperationalQuantity(ctx context.Context, eventID, itemID int64, stage string, quantity float64, notes string, userID int64, expectedVersion int) (int, error) {
	if stage != "separation" && stage != "loading" {
		return 0, fmt.Errorf("invalid operational stage")
	}
	if quantity < 0 {
		return 0, fmt.Errorf("quantity cannot be negative")
	}
	var newVersion int
	err := withTx(ctx, s.db, func(tx *sql.Tx) error {
		var required, separated, loaded float64
		var version int
		if err := tx.QueryRowContext(ctx, `SELECT item.required_quantity,item.separated_quantity,item.loaded_quantity,item.row_version FROM checklist_items item JOIN checklists checklist ON checklist.id=item.checklist_id WHERE item.id=? AND checklist.event_id=? AND item.active=1`, itemID, eventID).Scan(&required, &separated, &loaded, &version); err != nil {
			return err
		}
		if expectedVersion > 0 && expectedVersion != version {
			return fmt.Errorf("version conflict")
		}
		now := nowString()
		before := fmt.Sprintf(`{"separated_quantity":%g,"loaded_quantity":%g,"version":%d}`, separated, loaded, version)
		if stage == "separation" {
			if quantity+0.0001 < loaded {
				return fmt.Errorf("separated quantity cannot be less than loaded quantity")
			}
			_, err := tx.ExecContext(ctx, `UPDATE checklist_items SET separated_quantity=?,status=CASE WHEN ?>=required_quantity THEN 'separated' ELSE 'separating' END,separated_by=?,separated_at=CASE WHEN ?>=required_quantity THEN ? ELSE NULL END,notes=CASE WHEN ?<>'' THEN ? ELSE notes END,updated_by=?,row_version=row_version+1,updated_at=? WHERE id=?`, quantity, quantity, nullableUserID(userID), quantity, now, notes, notes, nullableUserID(userID), now, itemID)
			if err != nil {
				return err
			}
			separated = quantity
		} else {
			if separated+0.0001 < required {
				return fmt.Errorf("item is not ready for loading")
			}
			if quantity > separated+0.0001 {
				return fmt.Errorf("loaded quantity exceeds separated quantity")
			}
			_, err := tx.ExecContext(ctx, `UPDATE checklist_items SET loaded_quantity=?,status=CASE WHEN ?>=required_quantity THEN 'loaded' ELSE 'separated' END,loaded_by=?,loaded_at=CASE WHEN ?>=required_quantity THEN ? ELSE NULL END,loading_decision=CASE WHEN ?>=required_quantity THEN 'complete' ELSE NULL END,loading_missing_quantity=MAX(0,required_quantity-?),notes=CASE WHEN ?<>'' THEN ? ELSE notes END,updated_by=?,row_version=row_version+1,updated_at=? WHERE id=?`, quantity, quantity, nullableUserID(userID), quantity, now, quantity, quantity, notes, notes, nullableUserID(userID), now, itemID)
			if err != nil {
				return err
			}
			loaded = quantity
		}
		newVersion = version + 1
		after := fmt.Sprintf(`{"separated_quantity":%g,"loaded_quantity":%g,"version":%d}`, separated, loaded, newVersion)
		_, err := tx.ExecContext(ctx, `INSERT INTO checklist_item_history(checklist_item_id,event_id,action,before_json,after_json,performed_by,created_at) VALUES(?,?,?,?,?,?,?)`, itemID, eventID, stage, before, after, nullableUserID(userID), now)
		return err
	})
	return newVersion, err
}

func (s *Store) AddManualChecklistItem(ctx context.Context, eventID int64, item models.ChecklistItem, userID int64) (int64, error) {
	if strings.TrimSpace(item.Name) == "" || strings.TrimSpace(item.Unit) == "" || item.RequiredQuantity <= 0 {
		return 0, fmt.Errorf("name, unit and quantity are required")
	}
	var checklistID int64
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM checklists WHERE event_id=?`, eventID).Scan(&checklistID); err != nil {
		return 0, err
	}
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return 0, err
	}
	sourceKey := "manual:" + hex.EncodeToString(random)
	now := nowString()
	result, err := s.db.ExecContext(ctx, `INSERT INTO checklist_items(checklist_id,inventory_item_id,category_id,source_key,name,unit,calculated_quantity,required_quantity,available_quantity,missing_quantity,calculation_origin,notes,status,item_kind,location_snapshot,manual_item,manual_override,active,row_version,updated_by,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,'pending',?,?,1,0,1,1,?,?,?)`, checklistID, nullInt64(item.InventoryItemID), item.CategoryID, sourceKey, item.Name, item.Unit, item.RequiredQuantity, item.RequiredQuantity, item.AvailableQuantity, maxFloat(0, item.RequiredQuantity-item.AvailableQuantity), "Item adicionado manualmente ao evento", item.Notes, item.ItemKind, item.LocationSnapshot, nullableUserID(userID), now, now)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}
