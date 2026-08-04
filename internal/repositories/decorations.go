package repositories

import (
	"buffetflow/internal/models"
	"context"
	"database/sql"
	"fmt"
)

func (s *Store) EventDecorationSelection(ctx context.Context, eventID int64) ([]models.EventDecoration, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT d.id,d.name,d.usage_location,d.color,d.model,d.ownership,d.inventory_item_id,COALESCE(ed.quantity,1),CASE WHEN ed.id IS NULL THEN 0 ELSE 1 END FROM decorations d LEFT JOIN event_decorations ed ON ed.decoration_id=d.id AND ed.event_id=? WHERE d.active=1 ORDER BY d.name`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.EventDecoration
	for rows.Next() {
		var item models.EventDecoration
		var selected int
		if err := rows.Scan(&item.DecorationID, &item.Name, &item.UsageLocation, &item.Color, &item.Model, &item.Ownership, &item.InventoryItemID, &item.Quantity, &selected); err != nil {
			return nil, err
		}
		item.Selected = selected == 1
		result = append(result, item)
	}
	return result, rows.Err()
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
	var profileCount int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM event_decoration_profiles WHERE event_id=?`, eventID).Scan(&profileCount); err != nil {
		return nil, err
	}
	query := `SELECT d.inventory_item_id,SUM(ed.quantity),GROUP_CONCAT(DISTINCT d.name) FROM event_decorations ed JOIN decorations d ON d.id=ed.decoration_id WHERE ed.event_id=? AND d.inventory_item_id IS NOT NULL GROUP BY d.inventory_item_id`
	if profileCount > 0 {
		query = `SELECT item.inventory_item_id,SUM(item.quantity),GROUP_CONCAT(DISTINCT COALESCE(NULLIF(item.custom_name,''),decoration.name,inventory.name)) FROM event_decoration_composition_items item JOIN event_decoration_compositions composition ON composition.id=item.composition_id JOIN event_decoration_profiles profile ON profile.id=composition.profile_id LEFT JOIN decorations decoration ON decoration.id=item.decoration_id LEFT JOIN inventory_items inventory ON inventory.id=item.inventory_item_id WHERE profile.event_id=? AND profile.active=1 AND item.origin='owned' AND item.inventory_item_id IS NOT NULL GROUP BY item.inventory_item_id`
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
