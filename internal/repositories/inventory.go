package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"buffetflow/internal/models"
)

func (s *Store) ListCategories(ctx context.Context) ([]models.Category, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, name, sort_order FROM inventory_categories WHERE active=1 ORDER BY sort_order, name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.Category
	for rows.Next() {
		var item models.Category
		if err := rows.Scan(&item.ID, &item.Name, &item.SortOrder); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) ListLocations(ctx context.Context) ([]models.Location, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, name FROM inventory_locations WHERE active=1 ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.Location
	for rows.Next() {
		var item models.Location
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) ListInventory(ctx context.Context, query, category string, includeInactive bool) ([]models.InventoryItem, error) {
	return s.listInventory(ctx, query, category, includeInactive, false)
}

// ListStockInventory omits the physical cook boxes and every item currently
// stored inside one. Those records remain available to rules and catalogs, but
// their day-to-day administration belongs exclusively to the boxes screen.
func (s *Store) ListStockInventory(ctx context.Context, query, category string, includeInactive bool) ([]models.InventoryItem, error) {
	return s.listInventory(ctx, query, category, includeInactive, true)
}

func (s *Store) listInventory(ctx context.Context, query, category string, includeInactive, hideKitchenBoxItems bool) ([]models.InventoryItem, error) {
	pattern := "%" + strings.TrimSpace(query) + "%"
	rows, err := s.db.QueryContext(ctx, `SELECT i.id, i.name, i.description, i.category_id, c.name, i.subcategory, i.unit,
		i.stock_quantity, i.minimum_stock, i.damaged_quantity,
		COALESCE((SELECT SUM(r.quantity) FROM inventory_reservations r WHERE r.inventory_item_id=i.id AND r.status='active'), 0),
		MAX(0, i.stock_quantity-i.damaged_quantity-COALESCE((SELECT SUM(r.quantity) FROM inventory_reservations r WHERE r.inventory_item_id=i.id AND r.status='active'), 0)),
		i.location_id, COALESCE(l.name,''), i.internal_code, COALESCE(i.barcode,''), COALESCE(i.photo_url,''),
		i.item_kind, i.ownership, i.requires_return, i.replacement_value_cents, i.notes, i.active
		FROM inventory_items i JOIN inventory_categories c ON c.id=i.category_id
		LEFT JOIN inventory_locations l ON l.id=i.location_id
		WHERE (? = 1 OR i.active=1) AND (? = '' OR CAST(i.category_id AS TEXT)=?)
		AND (? = '%%' OR i.name LIKE ? COLLATE NOCASE OR i.internal_code LIKE ? COLLATE NOCASE OR COALESCE(l.name,'') LIKE ? COLLATE NOCASE)
		AND (? = 0 OR (
			i.id NOT IN (SELECT inventory_item_id FROM kitchen_cook_storage_boxes)
			AND NOT EXISTS (
				SELECT 1 FROM kitchen_cook_box_items content
				JOIN kitchen_cook_storage_boxes box ON box.id=content.kitchen_cook_storage_box_id
				WHERE content.inventory_item_id=i.id AND content.active=1 AND box.active=1
			)
		))
		ORDER BY c.sort_order, i.name`, includeInactive, category, category, pattern, pattern, pattern, pattern, hideKitchenBoxItems)
	if err != nil {
		return nil, fmt.Errorf("list inventory: %w", err)
	}
	defer rows.Close()
	var result []models.InventoryItem
	for rows.Next() {
		var item models.InventoryItem
		var requiresReturn, active int
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.CategoryID, &item.CategoryName, &item.Subcategory, &item.Unit,
			&item.StockQuantity, &item.MinimumStock, &item.DamagedQuantity, &item.ReservedQuantity, &item.AvailableQuantity,
			&item.LocationID, &item.LocationName, &item.InternalCode, &item.Barcode, &item.PhotoURL,
			&item.ItemKind, &item.Ownership, &requiresReturn, &item.ReplacementValueCents, &item.Notes, &active); err != nil {
			return nil, err
		}
		item.RequiresReturn, item.Active = requiresReturn == 1, active == 1
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) GetInventoryItem(ctx context.Context, id int64) (models.InventoryItem, error) {
	var item models.InventoryItem
	var requiresReturn, active int
	err := s.db.QueryRowContext(ctx, `SELECT i.id, i.name, i.description, i.category_id, c.name, i.subcategory, i.unit,
		i.stock_quantity, i.minimum_stock, i.damaged_quantity,
		COALESCE((SELECT SUM(r.quantity) FROM inventory_reservations r WHERE r.inventory_item_id=i.id AND r.status='active'), 0),
		MAX(0, i.stock_quantity-i.damaged_quantity-COALESCE((SELECT SUM(r.quantity) FROM inventory_reservations r WHERE r.inventory_item_id=i.id AND r.status='active'), 0)),
		i.location_id, COALESCE(l.name,''), i.internal_code, COALESCE(i.barcode,''), COALESCE(i.photo_url,''),
		i.item_kind, i.ownership, i.requires_return, i.replacement_value_cents, i.notes, i.active
		FROM inventory_items i JOIN inventory_categories c ON c.id=i.category_id LEFT JOIN inventory_locations l ON l.id=i.location_id WHERE i.id=?`, id).Scan(
		&item.ID, &item.Name, &item.Description, &item.CategoryID, &item.CategoryName, &item.Subcategory, &item.Unit,
		&item.StockQuantity, &item.MinimumStock, &item.DamagedQuantity, &item.ReservedQuantity, &item.AvailableQuantity,
		&item.LocationID, &item.LocationName, &item.InternalCode, &item.Barcode, &item.PhotoURL,
		&item.ItemKind, &item.Ownership, &requiresReturn, &item.ReplacementValueCents, &item.Notes, &active)
	item.RequiresReturn, item.Active = requiresReturn == 1, active == 1
	return item, err
}

func (s *Store) SaveInventoryItem(ctx context.Context, item *models.InventoryItem, userID int64) error {
	now := nowString()
	location := any(nil)
	if item.LocationID.Valid {
		location = item.LocationID.Int64
	}
	barcode := any(nil)
	if strings.TrimSpace(item.Barcode) != "" {
		barcode = strings.TrimSpace(item.Barcode)
	}
	photo := any(nil)
	if strings.TrimSpace(item.PhotoURL) != "" {
		photo = strings.TrimSpace(item.PhotoURL)
	}
	if item.ID == 0 {
		result, err := s.db.ExecContext(ctx, `INSERT INTO inventory_items(name,description,category_id,subcategory,unit,stock_quantity,minimum_stock,damaged_quantity,
			location_id,internal_code,barcode,photo_url,item_kind,ownership,requires_return,replacement_value_cents,notes,active,created_at,updated_at)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,1,?,?)`, item.Name, item.Description, item.CategoryID, item.Subcategory, item.Unit,
			item.StockQuantity, item.MinimumStock, item.DamagedQuantity, location, item.InternalCode, barcode, photo, item.ItemKind,
			item.Ownership, item.RequiresReturn, item.ReplacementValueCents, item.Notes, now, now)
		if err != nil {
			return fmt.Errorf("create inventory item: %w", err)
		}
		item.ID, err = result.LastInsertId()
		return err
	}
	_, err := s.db.ExecContext(ctx, `UPDATE inventory_items SET name=?,description=?,category_id=?,subcategory=?,unit=?,stock_quantity=?,minimum_stock=?,damaged_quantity=?,
		location_id=?,internal_code=?,barcode=?,photo_url=?,item_kind=?,ownership=?,requires_return=?,replacement_value_cents=?,notes=?,updated_at=? WHERE id=?`,
		item.Name, item.Description, item.CategoryID, item.Subcategory, item.Unit, item.StockQuantity, item.MinimumStock, item.DamagedQuantity,
		location, item.InternalCode, barcode, photo, item.ItemKind, item.Ownership, item.RequiresReturn, item.ReplacementValueCents, item.Notes, now, item.ID)
	if err != nil {
		return fmt.Errorf("update inventory item: %w", err)
	}
	return nil
}

func (s *Store) ToggleInventoryItem(ctx context.Context, id, userID int64) error {
	_, err := s.db.ExecContext(ctx, "UPDATE inventory_items SET active=CASE active WHEN 1 THEN 0 ELSE 1 END, updated_at=? WHERE id=?", nowString(), id)
	return err
}

func (s *Store) InventorySnapshot(ctx context.Context, eventID, itemID int64, startsAt, endsAt time.Time) (models.InventoryItem, float64, error) {
	item, err := s.GetInventoryItem(ctx, itemID)
	if err != nil {
		return item, 0, err
	}
	var reserved float64
	err = s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(quantity),0) FROM inventory_reservations
		WHERE inventory_item_id=? AND event_id<>? AND status='active' AND datetime(starts_at) < datetime(?) AND datetime(release_expected_at) > datetime(?)`,
		itemID, eventID, endsAt.UTC().Format(time.RFC3339), startsAt.UTC().Format(time.RFC3339)).Scan(&reserved)
	if err != nil {
		return item, 0, err
	}
	return item, reserved, nil
}

func (s *Store) ReserveEvent(ctx context.Context, eventID, userID int64, allowShortage bool) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		var starts, ends string
		if err := tx.QueryRowContext(ctx, "SELECT starts_at, ends_at FROM events WHERE id=?", eventID).Scan(&starts, &ends); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT ci.id, ci.inventory_item_id, ci.required_quantity, ci.missing_quantity
			FROM checklists c JOIN checklist_items ci ON ci.checklist_id=c.id
			WHERE c.event_id=? AND ci.inventory_item_id IS NOT NULL AND ci.status<>'not_applicable'`, eventID)
		if err != nil {
			return err
		}
		type reservation struct {
			checklistID, itemID int64
			quantity, missing   float64
		}
		var reservations []reservation
		for rows.Next() {
			var value reservation
			if err := rows.Scan(&value.checklistID, &value.itemID, &value.quantity, &value.missing); err != nil {
				rows.Close()
				return err
			}
			if value.missing > 0 && !allowShortage {
				rows.Close()
				return fmt.Errorf("shortage_confirmation_required")
			}
			reservations = append(reservations, value)
		}
		rows.Close()
		now := nowString()
		for _, value := range reservations {
			result, err := tx.ExecContext(ctx, `UPDATE inventory_reservations SET checklist_item_id=?,quantity=?,starts_at=?,release_expected_at=?,updated_at=?
				WHERE event_id=? AND inventory_item_id=? AND status='active'`, value.checklistID, value.quantity, starts, ends, now, eventID, value.itemID)
			if err != nil {
				return err
			}
			updated, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if updated == 0 {
				_, err = tx.ExecContext(ctx, `INSERT INTO inventory_reservations(event_id,inventory_item_id,checklist_item_id,quantity,starts_at,release_expected_at,status,created_at,updated_at)
					VALUES(?,?,?,?,?,?,'active',?,?)`, eventID, value.itemID, value.checklistID, value.quantity, starts, ends, now, now)
				if err != nil {
					return err
				}
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO inventory_movements(inventory_item_id,event_id,movement_type,quantity,previous_stock,new_stock,reason,performed_by,created_at)
				SELECT id,?,'reservation',?,stock_quantity,stock_quantity,'Reserva confirmada para o evento.',?,? FROM inventory_items WHERE id=?`,
				eventID, value.quantity, nullableUserID(userID), now, value.itemID)
			if err != nil {
				return err
			}
		}
		var previous string
		if err := tx.QueryRowContext(ctx, "SELECT status FROM events WHERE id=?", eventID).Scan(&previous); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "UPDATE events SET status='reserved', updated_at=? WHERE id=?", now, eventID); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO event_status_history(event_id,previous_status,new_status,notes,changed_by,created_at) VALUES(?,?,'reserved','Estoque reservado.',?,?)`, eventID, previous, nullableUserID(userID), now)
		return err
	})
}

func (s *Store) SyncEventReservations(ctx context.Context, eventID int64) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		var starts, ends string
		if err := tx.QueryRowContext(ctx, "SELECT starts_at,ends_at FROM events WHERE id=?", eventID).Scan(&starts, &ends); err != nil {
			return err
		}
		now := nowString()
		_, err := tx.ExecContext(ctx, `UPDATE inventory_reservations SET status='released',released_at=?,updated_at=?
			WHERE event_id=? AND status='active' AND inventory_item_id NOT IN (
				SELECT ci.inventory_item_id FROM checklists c JOIN checklist_items ci ON ci.checklist_id=c.id
				WHERE c.event_id=? AND ci.inventory_item_id IS NOT NULL AND ci.status<>'not_applicable'
			)`, now, now, eventID, eventID)
		if err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT ci.id,ci.inventory_item_id,ci.required_quantity FROM checklists c
			JOIN checklist_items ci ON ci.checklist_id=c.id WHERE c.event_id=? AND ci.inventory_item_id IS NOT NULL AND ci.status<>'not_applicable'`, eventID)
		if err != nil {
			return err
		}
		type reservation struct {
			checklistID, itemID int64
			quantity            float64
		}
		var values []reservation
		for rows.Next() {
			var value reservation
			if err := rows.Scan(&value.checklistID, &value.itemID, &value.quantity); err != nil {
				rows.Close()
				return err
			}
			values = append(values, value)
		}
		if err := rows.Close(); err != nil {
			return err
		}
		for _, value := range values {
			result, err := tx.ExecContext(ctx, `UPDATE inventory_reservations SET checklist_item_id=?,quantity=?,starts_at=?,release_expected_at=?,updated_at=?
				WHERE event_id=? AND inventory_item_id=? AND status='active'`, value.checklistID, value.quantity, starts, ends, now, eventID, value.itemID)
			if err != nil {
				return err
			}
			updated, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if updated == 0 {
				_, err = tx.ExecContext(ctx, `INSERT INTO inventory_reservations(event_id,inventory_item_id,checklist_item_id,quantity,starts_at,release_expected_at,status,created_at,updated_at)
				VALUES(?,?,?,?,?,?,'active',?,?)`, eventID, value.itemID, value.checklistID, value.quantity, starts, ends, now, now)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}
