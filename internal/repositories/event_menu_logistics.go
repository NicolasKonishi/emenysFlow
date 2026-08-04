package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"buffetflow/internal/models"
)

func (s *Store) eventMenuContainers(ctx context.Context, eventID int64) (map[int64][]models.EventMenuContainer, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT link.id,link.event_menu_snapshot_item_id,link.purpose,link.container_type_id,link.inventory_item_id,COALESCE(container.name,inventory.name,''),link.quantity,link.capacity_portions,link.requires_lid,link.notes FROM event_menu_item_containers link JOIN event_menu_snapshot_items item ON item.id=link.event_menu_snapshot_item_id JOIN event_menu_sections section ON section.id=item.event_menu_section_id JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id LEFT JOIN container_types container ON container.id=link.container_type_id LEFT JOIN inventory_items inventory ON inventory.id=link.inventory_item_id WHERE snapshot.event_id=? ORDER BY link.purpose`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[int64][]models.EventMenuContainer{}
	for rows.Next() {
		var item models.EventMenuContainer
		var lid int
		if err := rows.Scan(&item.ID, &item.SnapshotItemID, &item.Purpose, &item.ContainerTypeID, &item.InventoryItemID, &item.Name, &item.Quantity, &item.CapacityPortions, &lid, &item.Notes); err != nil {
			return nil, err
		}
		item.RequiresLid = lid == 1
		result[item.SnapshotItemID] = append(result[item.SnapshotItemID], item)
	}
	return result, rows.Err()
}

func (s *Store) eventMenuEquipment(ctx context.Context, eventID int64) (map[int64][]models.EventMenuEquipment, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT link.id,link.event_menu_snapshot_item_id,link.inventory_item_id,inventory.name,link.quantity,link.required,link.notes FROM event_menu_item_equipment link JOIN inventory_items inventory ON inventory.id=link.inventory_item_id JOIN event_menu_snapshot_items item ON item.id=link.event_menu_snapshot_item_id JOIN event_menu_sections section ON section.id=item.event_menu_section_id JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id WHERE snapshot.event_id=? ORDER BY inventory.name`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[int64][]models.EventMenuEquipment{}
	for rows.Next() {
		var item models.EventMenuEquipment
		var required int
		if err := rows.Scan(&item.ID, &item.SnapshotItemID, &item.InventoryItemID, &item.Name, &item.Quantity, &required, &item.Notes); err != nil {
			return nil, err
		}
		item.Required = required == 1
		result[item.SnapshotItemID] = append(result[item.SnapshotItemID], item)
	}
	return result, rows.Err()
}

func (s *Store) SaveEventMenuContainer(ctx context.Context, eventID, itemID int64, container models.EventMenuContainer) error {
	validPurpose := map[string]bool{"service": true, "transport": true, "cake_box": true, "cake_base": true, "cake_tray": true, "cake_support": true}
	if !validPurpose[container.Purpose] || !container.ContainerTypeID.Valid {
		return fmt.Errorf("container and purpose are required")
	}
	var inventoryID sql.NullInt64
	if err := s.db.QueryRowContext(ctx, `SELECT inventory_item_id FROM container_types WHERE id=? AND active=1`, container.ContainerTypeID.Int64).Scan(&inventoryID); err != nil {
		return err
	}
	var belongs int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM event_menu_snapshot_items item JOIN event_menu_sections section ON section.id=item.event_menu_section_id JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id WHERE item.id=? AND snapshot.event_id=?`, itemID, eventID).Scan(&belongs); err != nil || belongs == 0 {
		return sql.ErrNoRows
	}
	now := nowString()
	_, err := s.db.ExecContext(ctx, `INSERT INTO event_menu_item_containers(event_menu_snapshot_item_id,purpose,container_type_id,inventory_item_id,quantity,capacity_portions,requires_lid,notes,customized,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,1,?,?) ON CONFLICT(event_menu_snapshot_item_id,purpose) DO UPDATE SET container_type_id=excluded.container_type_id,inventory_item_id=excluded.inventory_item_id,quantity=excluded.quantity,capacity_portions=excluded.capacity_portions,requires_lid=excluded.requires_lid,notes=excluded.notes,customized=1,updated_at=excluded.updated_at`, itemID, container.Purpose, container.ContainerTypeID.Int64, nullInt64(inventoryID), nullableFloat64(container.Quantity), nullableFloat64(container.CapacityPortions), container.RequiresLid, strings.TrimSpace(container.Notes), now, now)
	return err
}

func (s *Store) RemoveEventMenuContainer(ctx context.Context, eventID, itemID, containerID int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM event_menu_item_containers WHERE id=? AND event_menu_snapshot_item_id=? AND event_menu_snapshot_item_id IN (SELECT item.id FROM event_menu_snapshot_items item JOIN event_menu_sections section ON section.id=item.event_menu_section_id JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id WHERE snapshot.event_id=?)`, containerID, itemID, eventID)
	if err == nil {
		if count, _ := result.RowsAffected(); count == 0 {
			return sql.ErrNoRows
		}
	}
	return err
}

func (s *Store) SaveEventMenuEquipment(ctx context.Context, eventID, itemID, inventoryID int64, quantity float64, required bool, notes string) error {
	if quantity <= 0 {
		return fmt.Errorf("equipment quantity must be positive")
	}
	var belongs int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM event_menu_snapshot_items item JOIN event_menu_sections section ON section.id=item.event_menu_section_id JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id WHERE item.id=? AND snapshot.event_id=?`, itemID, eventID).Scan(&belongs); err != nil || belongs == 0 {
		return sql.ErrNoRows
	}
	var equipmentExists int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM equipment WHERE inventory_item_id=? AND active=1`, inventoryID).Scan(&equipmentExists); err != nil || equipmentExists == 0 {
		return fmt.Errorf("invalid equipment")
	}
	now := nowString()
	_, err := s.db.ExecContext(ctx, `INSERT INTO event_menu_item_equipment(event_menu_snapshot_item_id,inventory_item_id,quantity,required,customized,notes,created_at,updated_at) VALUES(?,?,?,?,1,?,?,?) ON CONFLICT(event_menu_snapshot_item_id,inventory_item_id) DO UPDATE SET quantity=excluded.quantity,required=excluded.required,customized=1,notes=excluded.notes,updated_at=excluded.updated_at`, itemID, inventoryID, quantity, required, strings.TrimSpace(notes), now, now)
	return err
}

func (s *Store) RemoveEventMenuEquipment(ctx context.Context, eventID, itemID, equipmentID int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM event_menu_item_equipment WHERE id=? AND event_menu_snapshot_item_id=? AND event_menu_snapshot_item_id IN (SELECT item.id FROM event_menu_snapshot_items item JOIN event_menu_sections section ON section.id=item.event_menu_section_id JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id WHERE snapshot.event_id=?)`, equipmentID, itemID, eventID)
	if err == nil {
		if count, _ := result.RowsAffected(); count == 0 {
			return sql.ErrNoRows
		}
	}
	return err
}

func (s *Store) GetEventCakeConfiguration(ctx context.Context, eventID int64) (models.EventCakeConfiguration, error) {
	result := models.EventCakeConfiguration{EventID: eventID, CakeCount: 1, RowVersion: 1}
	var refrigeration int
	err := s.db.QueryRowContext(ctx, `SELECT cake_count,requires_refrigeration,notes,row_version FROM event_cake_configurations WHERE event_id=?`, eventID).Scan(&result.CakeCount, &refrigeration, &result.Notes, &result.RowVersion)
	if err == sql.ErrNoRows {
		return result, nil
	}
	result.RequiresRefrigeration = refrigeration == 1
	return result, err
}

func (s *Store) SaveEventCakeConfiguration(ctx context.Context, configuration models.EventCakeConfiguration, userID int64) error {
	if configuration.CakeCount < 0 {
		return fmt.Errorf("cake count cannot be negative")
	}
	now := nowString()
	_, err := s.db.ExecContext(ctx, `INSERT INTO event_cake_configurations(event_id,cake_count,requires_refrigeration,notes,row_version,updated_by,created_at,updated_at) VALUES(?,?,?,?,1,?,?,?) ON CONFLICT(event_id) DO UPDATE SET cake_count=excluded.cake_count,requires_refrigeration=excluded.requires_refrigeration,notes=excluded.notes,row_version=event_cake_configurations.row_version+1,updated_by=excluded.updated_by,updated_at=excluded.updated_at`, configuration.EventID, configuration.CakeCount, configuration.RequiresRefrigeration, strings.TrimSpace(configuration.Notes), nullableUserID(userID), now, now)
	return err
}
