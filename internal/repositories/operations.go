package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"buffetflow/internal/models"
)

var operationStages = map[string]bool{"separating": true, "checking": true, "loading": true, "in_progress": true}

func (s *Store) GetEventOperation(ctx context.Context, eventID int64, stage string) (models.EventOperation, error) {
	var item models.EventOperation
	var occurred string
	err := s.db.QueryRowContext(ctx, `SELECT id,event_id,stage,responsible_name,vehicle,notes,COALESCE(photo_url,''),occurred_at FROM event_operations WHERE event_id=? AND stage=?`, eventID, stage).Scan(&item.ID, &item.EventID, &item.Stage, &item.ResponsibleName, &item.Vehicle, &item.Notes, &item.PhotoURL, &occurred)
	if err == sql.ErrNoRows {
		return models.EventOperation{EventID: eventID, Stage: stage, OccurredAt: time.Now()}, nil
	}
	item.OccurredAt = parseTime(occurred)
	return item, err
}

func (s *Store) OperationChecklist(ctx context.Context, eventID int64, stage string) (models.Checklist, error) {
	checklist, err := s.GetChecklistByEvent(ctx, eventID)
	if err != nil {
		return checklist, err
	}
	filtered := checklist.Items[:0]
	for _, item := range checklist.Items {
		include := true
		switch stage {
		case "loading":
			include = item.Status == "separated" || item.Status == "checked" || item.Status == "loaded"
		case "checking":
			include = item.Status == "separated" || item.Status == "checking" || item.Status == "checked"
		}
		if include {
			filtered = append(filtered, item)
		}
	}
	checklist.Items = filtered
	checklist.Progress = checklistProgress(filtered)
	return checklist, nil
}

func (s *Store) SaveEventOperation(ctx context.Context, eventID int64, operation models.EventOperation, quantities map[int64]float64, userID int64) error {
	if !operationStages[operation.Stage] {
		return fmt.Errorf("invalid operation stage")
	}
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		occurred := operation.OccurredAt
		if occurred.IsZero() {
			occurred = time.Now()
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO event_operations(event_id,stage,responsible_user_id,responsible_name,vehicle,notes,photo_url,occurred_at,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?) ON CONFLICT(event_id,stage) DO UPDATE SET responsible_user_id=excluded.responsible_user_id,responsible_name=excluded.responsible_name,vehicle=excluded.vehicle,notes=excluded.notes,photo_url=excluded.photo_url,occurred_at=excluded.occurred_at,updated_at=excluded.updated_at`, eventID, operation.Stage, nullableUserID(userID), operation.ResponsibleName, operation.Vehicle, operation.Notes, nullString(operation.PhotoURL), occurred.UTC().Format(time.RFC3339), now, now)
		if err != nil {
			return err
		}
		for itemID, quantity := range quantities {
			if quantity < 0 {
				return fmt.Errorf("quantity cannot be negative")
			}
			switch operation.Stage {
			case "separating":
				_, err = tx.ExecContext(ctx, `UPDATE checklist_items SET separated_quantity=?,status=CASE WHEN ?>=required_quantity THEN 'separated' ELSE 'separating' END,separated_by=?,separated_at=?,updated_at=? WHERE id=? AND checklist_id=(SELECT id FROM checklists WHERE event_id=?)`, quantity, quantity, nullableUserID(userID), now, now, itemID, eventID)
			case "checking":
				_, err = tx.ExecContext(ctx, `UPDATE checklist_items SET status=CASE WHEN ?>=required_quantity THEN 'checked' ELSE 'separating' END,checked_by=?,checked_at=?,updated_at=? WHERE id=? AND checklist_id=(SELECT id FROM checklists WHERE event_id=?)`, quantity, nullableUserID(userID), now, now, itemID, eventID)
			case "loading":
				_, err = tx.ExecContext(ctx, `UPDATE checklist_items SET loaded_quantity=?,loading_decision=CASE WHEN ?>=required_quantity THEN 'complete' ELSE 'missing' END,loading_missing_quantity=MAX(0,required_quantity-?),status='loaded',loaded_by=?,loaded_at=?,updated_at=? WHERE id=? AND checklist_id=(SELECT id FROM checklists WHERE event_id=?)`, quantity, quantity, quantity, nullableUserID(userID), now, now, itemID, eventID)
			}
			if err != nil {
				return err
			}
		}
		if operation.Stage == "in_progress" {
			if _, err = tx.ExecContext(ctx, `UPDATE checklist_items SET status='at_event',updated_at=? WHERE checklist_id=(SELECT id FROM checklists WHERE event_id=?) AND status='loaded'`, now, eventID); err != nil {
				return err
			}
		}
		var previous string
		if err = tx.QueryRowContext(ctx, "SELECT status FROM events WHERE id=?", eventID).Scan(&previous); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "UPDATE events SET status=?,updated_at=? WHERE id=?", operation.Stage, now, eventID); err != nil {
			return err
		}
		if previous != operation.Stage {
			_, err = tx.ExecContext(ctx, `INSERT INTO event_status_history(event_id,previous_status,new_status,notes,changed_by,created_at) VALUES(?,?,?,?,?,?)`, eventID, previous, operation.Stage, operation.Notes, nullableUserID(userID), now)
		}
		return err
	})
}

func (s *Store) UpdateMobileLoadingItem(ctx context.Context, eventID, itemID int64, decision string, missingQuantity float64, userID int64) (float64, error) {
	if decision != "complete" && decision != "missing" {
		return 0, fmt.Errorf("invalid loading decision")
	}
	if missingQuantity < 0 {
		return 0, fmt.Errorf("missing quantity cannot be negative")
	}
	var loadedQuantity float64
	err := withTx(ctx, s.db, func(tx *sql.Tx) error {
		var requiredQuantity float64
		err := tx.QueryRowContext(ctx, `SELECT item.required_quantity
			FROM checklist_items item JOIN checklists checklist ON checklist.id=item.checklist_id
			WHERE item.id=? AND checklist.event_id=? AND item.status IN ('separated','checked','loaded')`, itemID, eventID).Scan(&requiredQuantity)
		if err != nil {
			return err
		}
		if decision == "complete" {
			missingQuantity = 0
		} else if missingQuantity <= 0 || missingQuantity > requiredQuantity {
			return fmt.Errorf("missing quantity must be greater than zero and no more than required")
		}
		loadedQuantity = requiredQuantity - missingQuantity
		now := nowString()
		result, err := tx.ExecContext(ctx, `UPDATE checklist_items SET loaded_quantity=?,loading_decision=?,loading_missing_quantity=?,status='loaded',loaded_by=?,loaded_at=?,updated_at=?
			WHERE id=? AND checklist_id=(SELECT id FROM checklists WHERE event_id=?)`, loadedQuantity, decision, missingQuantity, nullableUserID(userID), now, now, itemID, eventID)
		if err != nil {
			return err
		}
		if count, _ := result.RowsAffected(); count == 0 {
			return sql.ErrNoRows
		}
		return nil
	})
	return loadedQuantity, err
}

func (s *Store) ReturnItems(ctx context.Context, eventID int64) ([]models.ReturnInspection, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT ci.id,ci.name,ci.unit,ci.loaded_quantity,COALESCE(ri.returned_quantity,0),COALESCE(ri.damaged_quantity,0),COALESCE(ri.lost_quantity,0),COALESCE(ri.laundry_quantity,0),COALESCE(ri.maintenance_quantity,0),COALESCE(ri.notes,'') FROM checklists c JOIN checklist_items ci ON ci.checklist_id=c.id JOIN inventory_items ii ON ii.id=ci.inventory_item_id LEFT JOIN return_inspections ri ON ri.checklist_item_id=ci.id AND ri.event_id=? WHERE c.event_id=? AND ii.requires_return=1 AND ci.loaded_quantity>0 ORDER BY ci.name`, eventID, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.ReturnInspection
	for rows.Next() {
		var item models.ReturnInspection
		if err := rows.Scan(&item.ChecklistItemID, &item.Name, &item.Unit, &item.LoadedQuantity, &item.ReturnedQuantity, &item.DamagedQuantity, &item.LostQuantity, &item.LaundryQuantity, &item.MaintenanceQuantity, &item.Notes); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) SaveReturnInspections(ctx context.Context, eventID int64, items []models.ReturnInspection, userID int64, notes string) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		for _, item := range items {
			if item.ReturnedQuantity < 0 || item.DamagedQuantity < 0 || item.LostQuantity < 0 || item.LaundryQuantity < 0 || item.MaintenanceQuantity < 0 {
				return fmt.Errorf("negative return quantity")
			}
			if item.ReturnedQuantity+item.DamagedQuantity+item.LostQuantity > item.LoadedQuantity+0.0001 {
				return fmt.Errorf("return quantities exceed loaded quantity for %s", item.Name)
			}
			_, err := tx.ExecContext(ctx, `INSERT INTO return_inspections(event_id,checklist_item_id,sent_quantity,returned_quantity,damaged_quantity,lost_quantity,laundry_quantity,maintenance_quantity,notes,inspected_by,inspected_at) VALUES(?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(event_id,checklist_item_id) DO UPDATE SET sent_quantity=excluded.sent_quantity,returned_quantity=excluded.returned_quantity,damaged_quantity=excluded.damaged_quantity,lost_quantity=excluded.lost_quantity,laundry_quantity=excluded.laundry_quantity,maintenance_quantity=excluded.maintenance_quantity,notes=excluded.notes,inspected_by=excluded.inspected_by,inspected_at=excluded.inspected_at`, eventID, item.ChecklistItemID, item.LoadedQuantity, item.ReturnedQuantity, item.DamagedQuantity, item.LostQuantity, item.LaundryQuantity, item.MaintenanceQuantity, item.Notes, nullableUserID(userID), now)
			if err != nil {
				return err
			}
			status := "returned"
			if item.LostQuantity > 0 {
				status = "lost"
			} else if item.DamagedQuantity > 0 {
				status = "damaged"
			}
			if _, err = tx.ExecContext(ctx, `UPDATE checklist_items SET returned_quantity=?,damaged_quantity=?,lost_quantity=?,status=?,updated_at=? WHERE id=?`, item.ReturnedQuantity, item.DamagedQuantity, item.LostQuantity, status, now, item.ChecklistItemID); err != nil {
				return err
			}
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO event_operations(event_id,stage,responsible_user_id,responsible_name,notes,occurred_at,created_at,updated_at) VALUES(?,'returning',?,'',?,?,?,?) ON CONFLICT(event_id,stage) DO UPDATE SET responsible_user_id=excluded.responsible_user_id,notes=excluded.notes,occurred_at=excluded.occurred_at,updated_at=excluded.updated_at`, eventID, nullableUserID(userID), notes, now, now, now)
		if err != nil {
			return err
		}
		var previous string
		if err = tx.QueryRowContext(ctx, "SELECT status FROM events WHERE id=?", eventID).Scan(&previous); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "UPDATE events SET status='post_event_check',updated_at=? WHERE id=?", now, eventID); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO event_status_history(event_id,previous_status,new_status,notes,changed_by,created_at) VALUES(?,?,'post_event_check',?,?,?)`, eventID, previous, notes, nullableUserID(userID), now)
		return err
	})
}

func (s *Store) FinalizeReturn(ctx context.Context, eventID, userID int64, notes string) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		var status string
		if err := tx.QueryRowContext(ctx, "SELECT status FROM events WHERE id=?", eventID).Scan(&status); err != nil {
			return err
		}
		if status == "completed" {
			return fmt.Errorf("event already completed")
		}
		rows, err := tx.QueryContext(ctx, `SELECT ci.inventory_item_id,ci.name,ri.sent_quantity,ri.returned_quantity,ri.damaged_quantity,ri.lost_quantity FROM return_inspections ri JOIN checklist_items ci ON ci.id=ri.checklist_item_id WHERE ri.event_id=? AND ci.inventory_item_id IS NOT NULL`, eventID)
		if err != nil {
			return err
		}
		type value struct {
			inventoryID                   int64
			name                          string
			sent, returned, damaged, lost float64
		}
		var values []value
		for rows.Next() {
			var v value
			if err := rows.Scan(&v.inventoryID, &v.name, &v.sent, &v.returned, &v.damaged, &v.lost); err != nil {
				rows.Close()
				return err
			}
			values = append(values, v)
		}
		if err := rows.Close(); err != nil {
			return err
		}
		now := nowString()
		for _, v := range values {
			accounted := v.returned + v.damaged + v.lost
			if accounted > v.sent+0.0001 {
				return fmt.Errorf("invalid return for %s", v.name)
			}
			if accounted < v.sent-0.0001 {
				return fmt.Errorf("há itens não conferidos em %s", v.name)
			}
			var stock, damagedStock float64
			if err := tx.QueryRowContext(ctx, "SELECT stock_quantity,damaged_quantity FROM inventory_items WHERE id=?", v.inventoryID).Scan(&stock, &damagedStock); err != nil {
				return err
			}
			newStock := stock - v.lost
			if newStock < 0 {
				return fmt.Errorf("stock would become negative for %s", v.name)
			}
			if _, err := tx.ExecContext(ctx, "UPDATE inventory_items SET stock_quantity=?,damaged_quantity=?,updated_at=? WHERE id=?", newStock, damagedStock+v.damaged, now, v.inventoryID); err != nil {
				return err
			}
			if v.damaged > 0 {
				if _, err := tx.ExecContext(ctx, `INSERT INTO inventory_movements(inventory_item_id,event_id,movement_type,quantity,previous_stock,new_stock,reason,performed_by,created_at) VALUES(?,?,'damage',?,?,?, ?,?,?)`, v.inventoryID, eventID, v.damaged, stock, stock, "Dano apurado na conferência pós-evento.", nullableUserID(userID), now); err != nil {
					return err
				}
			}
			if v.lost > 0 {
				if _, err := tx.ExecContext(ctx, `INSERT INTO inventory_movements(inventory_item_id,event_id,movement_type,quantity,previous_stock,new_stock,reason,performed_by,created_at) VALUES(?,?,'loss',?,?,?,?,?,?)`, v.inventoryID, eventID, v.lost, stock, newStock, "Perda apurada na conferência pós-evento.", nullableUserID(userID), now); err != nil {
					return err
				}
			}
		}
		consumableRows, err := tx.QueryContext(ctx, `SELECT ci.inventory_item_id,ci.name,SUM(ci.loaded_quantity) FROM checklists c JOIN checklist_items ci ON ci.checklist_id=c.id WHERE c.event_id=? AND ci.item_kind='consumable' AND ci.inventory_item_id IS NOT NULL AND ci.loaded_quantity>0 GROUP BY ci.inventory_item_id,ci.name`, eventID)
		if err != nil {
			return err
		}
		type consumable struct {
			id       int64
			name     string
			quantity float64
		}
		var consumables []consumable
		for consumableRows.Next() {
			var value consumable
			if err := consumableRows.Scan(&value.id, &value.name, &value.quantity); err != nil {
				consumableRows.Close()
				return err
			}
			consumables = append(consumables, value)
		}
		if err := consumableRows.Close(); err != nil {
			return err
		}
		for _, value := range consumables {
			var stock float64
			if err := tx.QueryRowContext(ctx, "SELECT stock_quantity FROM inventory_items WHERE id=?", value.id).Scan(&stock); err != nil {
				return err
			}
			newStock := stock - value.quantity
			if newStock < 0 {
				return fmt.Errorf("estoque insuficiente para baixar o consumo de %s", value.name)
			}
			if _, err := tx.ExecContext(ctx, "UPDATE inventory_items SET stock_quantity=?,updated_at=? WHERE id=?", newStock, now, value.id); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO inventory_movements(inventory_item_id,event_id,movement_type,quantity,previous_stock,new_stock,reason,performed_by,created_at) VALUES(?,?,'out',?,?,?,?,?,?)`, value.id, eventID, value.quantity, stock, newStock, "Consumo confirmado na finalização do evento.", nullableUserID(userID), now); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, "UPDATE inventory_reservations SET status='released',released_at=?,updated_at=? WHERE event_id=? AND status='active'", now, now, eventID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE events SET status='completed',updated_at=? WHERE id=?", now, eventID); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO event_operations(event_id,stage,responsible_user_id,responsible_name,notes,occurred_at,created_at,updated_at) VALUES(?,'post_event_check',?,'',?,?,?,?) ON CONFLICT(event_id,stage) DO UPDATE SET responsible_user_id=excluded.responsible_user_id,notes=excluded.notes,occurred_at=excluded.occurred_at,updated_at=excluded.updated_at`, eventID, nullableUserID(userID), notes, now, now, now)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO event_status_history(event_id,previous_status,new_status,notes,changed_by,created_at) VALUES(?,?,'completed',?,?,?)`, eventID, status, notes, nullableUserID(userID), now)
		return err
	})
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
