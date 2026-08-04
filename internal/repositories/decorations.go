package repositories

import (
	"buffetflow/internal/models"
	"context"
	"database/sql"
	"fmt"
	"time"
)

func (s *Store) EventDecorationSelection(ctx context.Context, eventID int64) ([]models.EventDecoration, error) {
	var startsAt, endsAt time.Time
	if eventID > 0 {
		event, err := s.GetEvent(ctx, eventID)
		if err != nil {
			return nil, err
		}
		startsAt, endsAt = event.StartsAt, event.EndsAt
	}
	return s.EventDecorationSelectionForWindow(ctx, eventID, startsAt, endsAt)
}

func (s *Store) EventDecorationSelectionForWindow(ctx context.Context, eventID int64, startsAt, endsAt time.Time) ([]models.EventDecoration, error) {
	starts, ends := startsAt.UTC().Format(time.RFC3339), endsAt.UTC().Format(time.RFC3339)
	rows, err := s.db.QueryContext(ctx, `WITH catalog AS (
		SELECT d.id,d.name,d.usage_location,d.color,d.model,d.ownership,d.rental_company,COALESCE(d.photo_url,'') photo_url,d.notes,d.inventory_item_id,
			COALESCE(i.unit,'unidade') unit,COALESCE(i.stock_quantity,0) stock_quantity,COALESCE(i.damaged_quantity,0) damaged_quantity,
			COALESCE((SELECT SUM(reservation.quantity) FROM inventory_reservations reservation
				WHERE reservation.inventory_item_id=i.id AND reservation.event_id<>? AND reservation.status='active'
				AND datetime(reservation.starts_at)<datetime(?) AND datetime(reservation.release_expected_at)>datetime(?)),0) reserved_quantity,
			CASE WHEN i.id IS NULL THEN 0 ELSE 1 END availability_tracked
		FROM decorations d LEFT JOIN inventory_items i ON i.id=d.inventory_item_id
		WHERE d.active=1 AND (i.id IS NULL OR i.active=1)
	)
	SELECT catalog.id,catalog.name,catalog.usage_location,catalog.color,catalog.model,catalog.ownership,catalog.rental_company,catalog.photo_url,catalog.notes,
		catalog.inventory_item_id,catalog.unit,catalog.stock_quantity,catalog.damaged_quantity,catalog.reserved_quantity,
		MAX(0,catalog.stock_quantity-catalog.damaged_quantity-catalog.reserved_quantity) available_quantity,catalog.availability_tracked,
		CASE WHEN event_selection.id IS NOT NULL OR (catalog.availability_tracked=1 AND catalog.stock_quantity-catalog.damaged_quantity-catalog.reserved_quantity>0) THEN 1 ELSE 0 END selectable,
		COALESCE(event_selection.quantity,CASE WHEN catalog.stock_quantity-catalog.damaged_quantity-catalog.reserved_quantity>0 THEN 1 ELSE 0 END),
		CASE WHEN event_selection.id IS NULL THEN 0 ELSE 1 END selected
	FROM catalog LEFT JOIN event_decorations event_selection ON event_selection.decoration_id=catalog.id AND event_selection.event_id=?
	ORDER BY catalog.usage_location,catalog.color,catalog.name`, eventID, ends, starts, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.EventDecoration
	for rows.Next() {
		var item models.EventDecoration
		var tracked, selectable, selected int
		if err := rows.Scan(&item.DecorationID, &item.Name, &item.UsageLocation, &item.Color, &item.Model, &item.Ownership, &item.RentalCompany, &item.PhotoURL, &item.Notes,
			&item.InventoryItemID, &item.Unit, &item.StockQuantity, &item.DamagedQuantity, &item.ReservedQuantity, &item.AvailableQuantity, &tracked, &selectable, &item.Quantity, &selected); err != nil {
			return nil, err
		}
		item.AvailabilityTracked, item.Selectable, item.Selected = tracked == 1, selectable == 1, selected == 1
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) ListDecorationCatalog(ctx context.Context, includeInactive bool) ([]models.EventDecoration, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT d.id,d.name,d.usage_location,d.color,d.model,d.ownership,d.rental_company,COALESCE(d.photo_url,''),d.notes,d.inventory_item_id,
		COALESCE(inventory.unit,'unidade'),COALESCE(inventory.stock_quantity,0),COALESCE(inventory.damaged_quantity,0),
		COALESCE((SELECT SUM(reservation.quantity) FROM inventory_reservations reservation WHERE reservation.inventory_item_id=inventory.id AND reservation.status='active'),0),
		CASE WHEN inventory.id IS NULL THEN 0 ELSE MAX(0,inventory.stock_quantity-inventory.damaged_quantity-COALESCE((SELECT SUM(reservation.quantity) FROM inventory_reservations reservation WHERE reservation.inventory_item_id=inventory.id AND reservation.status='active'),0)) END,
		CASE WHEN inventory.id IS NULL THEN 0 ELSE 1 END,d.active
	FROM decorations d LEFT JOIN inventory_items inventory ON inventory.id=d.inventory_item_id
	WHERE (?=1 OR d.active=1) ORDER BY d.active DESC,d.usage_location,d.color,d.name`, includeInactive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.EventDecoration
	for rows.Next() {
		var item models.EventDecoration
		var tracked, active int
		if err := rows.Scan(&item.DecorationID, &item.Name, &item.UsageLocation, &item.Color, &item.Model, &item.Ownership, &item.RentalCompany, &item.PhotoURL, &item.Notes,
			&item.InventoryItemID, &item.Unit, &item.StockQuantity, &item.DamagedQuantity, &item.ReservedQuantity, &item.AvailableQuantity, &tracked, &active); err != nil {
			return nil, err
		}
		item.AvailabilityTracked, item.Active = tracked == 1, active == 1
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) SaveDecorationCatalogItem(ctx context.Context, item *models.EventDecoration) error {
	if !item.InventoryItemID.Valid {
		return fmt.Errorf("decoration inventory item is required")
	}
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM inventory_items WHERE id=? AND active=1`, item.InventoryItemID.Int64).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return fmt.Errorf("decoration inventory item not found")
	}
	if item.Ownership != "rented" {
		item.Ownership = "owned"
	}
	now := nowString()
	if item.DecorationID == 0 {
		result, err := s.db.ExecContext(ctx, `INSERT INTO decorations(inventory_item_id,name,usage_location,color,model,ownership,rental_company,photo_url,notes,active,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,1,?,?)`, item.InventoryItemID.Int64, item.Name, item.UsageLocation, item.Color, item.Model, item.Ownership, item.RentalCompany, nullIfEmpty(item.PhotoURL), item.Notes, now, now)
		if err != nil {
			return err
		}
		item.DecorationID, err = result.LastInsertId()
		return err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE decorations SET inventory_item_id=?,name=?,usage_location=?,color=?,model=?,ownership=?,rental_company=?,photo_url=?,notes=?,updated_at=? WHERE id=?`, item.InventoryItemID.Int64, item.Name, item.UsageLocation, item.Color, item.Model, item.Ownership, item.RentalCompany, nullIfEmpty(item.PhotoURL), item.Notes, now, item.DecorationID)
	if err != nil {
		return err
	}
	if changed, _ := result.RowsAffected(); changed == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) ToggleDecorationCatalogItem(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `UPDATE decorations SET active=CASE active WHEN 1 THEN 0 ELSE 1 END,updated_at=? WHERE id=?`, nowString(), id)
	if err != nil {
		return err
	}
	if changed, _ := result.RowsAffected(); changed == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) SaveEventDecorations(ctx context.Context, eventID int64, items []models.EventDecoration) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM event_decorations WHERE event_id=?", eventID); err != nil {
			return err
		}
		for _, item := range items {
			if !item.Selected {
				continue
			}
			if item.Quantity <= 0 {
				return fmt.Errorf("decoration quantity must be positive")
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO event_decorations(event_id,decoration_id,quantity,notes) VALUES(?,?,?,'')`, eventID, item.DecorationID, item.Quantity); err != nil {
				return err
			}
		}
		return nil
	})
}
func (s *Store) EventDecorationRequirements(ctx context.Context, eventID int64, enabled bool) ([]models.AutomaticRequirement, error) {
	if !enabled {
		return nil, nil
	}
	var selectionCount int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM event_decorations WHERE event_id=?`, eventID).Scan(&selectionCount); err != nil {
		return nil, err
	}
	query := `SELECT item.inventory_item_id,SUM(item.quantity),GROUP_CONCAT(DISTINCT COALESCE(NULLIF(item.custom_name,''),decoration.name,inventory.name)) FROM event_decoration_composition_items item JOIN event_decoration_compositions composition ON composition.id=item.composition_id JOIN event_decoration_profiles profile ON profile.id=composition.profile_id LEFT JOIN decorations decoration ON decoration.id=item.decoration_id LEFT JOIN inventory_items inventory ON inventory.id=item.inventory_item_id WHERE profile.event_id=? AND profile.active=1 AND item.origin='owned' AND item.inventory_item_id IS NOT NULL GROUP BY item.inventory_item_id`
	if selectionCount > 0 {
		query = `SELECT d.inventory_item_id,SUM(ed.quantity),GROUP_CONCAT(DISTINCT d.name) FROM event_decorations ed JOIN decorations d ON d.id=ed.decoration_id WHERE ed.event_id=? AND d.inventory_item_id IS NOT NULL GROUP BY d.inventory_item_id`
	}
	rows, err := s.db.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.AutomaticRequirement
	for rows.Next() {
		var item models.AutomaticRequirement
		var names string
		if err := rows.Scan(&item.InventoryItemID, &item.Quantity, &names); err != nil {
			return nil, err
		}
		item.SourceKey = fmt.Sprintf("decoration:%d", item.InventoryItemID)
		item.Origin = "Decoração selecionada: " + names
		result = append(result, item)
	}
	return result, rows.Err()
}
