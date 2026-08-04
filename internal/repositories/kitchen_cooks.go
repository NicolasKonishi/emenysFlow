package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"buffetflow/internal/models"
)

func (s *Store) ListKitchenCooks(ctx context.Context, includeInactive bool) ([]models.KitchenCook, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,slug,name,active FROM kitchen_cooks WHERE (?=1 OR active=1) ORDER BY name`, includeInactive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.KitchenCook
	for rows.Next() {
		var cook models.KitchenCook
		var active int
		if err := rows.Scan(&cook.ID, &cook.Slug, &cook.Name, &active); err != nil {
			return nil, err
		}
		cook.Active = active == 1
		result = append(result, cook)
	}
	return result, rows.Err()
}

func (s *Store) ValidateKitchenCook(ctx context.Context, cookID int64) error {
	var active int
	if err := s.db.QueryRowContext(ctx, "SELECT active FROM kitchen_cooks WHERE id=?", cookID).Scan(&active); err != nil {
		return err
	}
	if active != 1 {
		return fmt.Errorf("cozinheira inativa")
	}
	return nil
}

func (s *Store) ListKitchenCookBoxes(ctx context.Context) ([]models.KitchenCookBox, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT box.id,box.kitchen_cook_id,cook.name,box.inventory_item_id,item.name,box.box_type,box.active,
		(SELECT COUNT(*) FROM kitchen_cook_box_items content WHERE content.kitchen_cook_storage_box_id=box.id AND content.active=1),
		COALESCE((SELECT SUM(content.quantity) FROM kitchen_cook_box_items content WHERE content.kitchen_cook_storage_box_id=box.id AND content.active=1),0)
		FROM kitchen_cook_storage_boxes box
		JOIN kitchen_cooks cook ON cook.id=box.kitchen_cook_id
		JOIN inventory_items item ON item.id=box.inventory_item_id
		WHERE box.active=1 AND cook.active=1
		ORDER BY cook.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.KitchenCookBox
	for rows.Next() {
		var box models.KitchenCookBox
		var active int
		if err := rows.Scan(&box.ID, &box.KitchenCookID, &box.KitchenCookName, &box.InventoryItemID, &box.InventoryItemName, &box.BoxType, &active, &box.ItemCount, &box.TotalQuantity); err != nil {
			return nil, err
		}
		box.Active = active == 1
		result = append(result, box)
	}
	return result, rows.Err()
}

func (s *Store) GetKitchenCookBox(ctx context.Context, boxID int64) (models.KitchenCookBox, error) {
	var box models.KitchenCookBox
	var active int
	err := s.db.QueryRowContext(ctx, `SELECT box.id,box.kitchen_cook_id,cook.name,box.inventory_item_id,item.name,box.box_type,box.active,
		(SELECT COUNT(*) FROM kitchen_cook_box_items content WHERE content.kitchen_cook_storage_box_id=box.id AND content.active=1),
		COALESCE((SELECT SUM(content.quantity) FROM kitchen_cook_box_items content WHERE content.kitchen_cook_storage_box_id=box.id AND content.active=1),0)
		FROM kitchen_cook_storage_boxes box
		JOIN kitchen_cooks cook ON cook.id=box.kitchen_cook_id
		JOIN inventory_items item ON item.id=box.inventory_item_id
		WHERE box.id=?`, boxID).Scan(&box.ID, &box.KitchenCookID, &box.KitchenCookName, &box.InventoryItemID, &box.InventoryItemName, &box.BoxType, &active, &box.ItemCount, &box.TotalQuantity)
	if err != nil {
		return box, err
	}
	box.Active = active == 1
	rows, err := s.db.QueryContext(ctx, `SELECT content.id,content.kitchen_cook_storage_box_id,content.inventory_item_id,item.name,category.name,item.unit,content.quantity,content.notes,content.active
		FROM kitchen_cook_box_items content
		JOIN inventory_items item ON item.id=content.inventory_item_id
		JOIN inventory_categories category ON category.id=item.category_id
		WHERE content.kitchen_cook_storage_box_id=? AND content.active=1
		ORDER BY category.sort_order,item.name`, boxID)
	if err != nil {
		return box, err
	}
	defer rows.Close()
	for rows.Next() {
		var item models.KitchenCookBoxItem
		var itemActive int
		if err := rows.Scan(&item.ID, &item.BoxID, &item.InventoryItemID, &item.InventoryName, &item.CategoryName, &item.Unit, &item.Quantity, &item.Notes, &itemActive); err != nil {
			return box, err
		}
		item.Active = itemActive == 1
		box.Items = append(box.Items, item)
	}
	return box, rows.Err()
}

func (s *Store) ListKitchenBoxContentOptions(ctx context.Context, boxID int64) ([]models.InventoryItem, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT item.id,item.name,category.name,item.unit,item.active
		FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id
		WHERE item.active=1
		AND item.id NOT IN (SELECT inventory_item_id FROM kitchen_cook_storage_boxes)
		AND NOT EXISTS (SELECT 1 FROM kitchen_cook_box_items content WHERE content.kitchen_cook_storage_box_id=? AND content.inventory_item_id=item.id AND content.active=1)
		AND item.item_kind<>'outsourced'
		ORDER BY category.sort_order,item.name`, boxID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.InventoryItem
	for rows.Next() {
		var item models.InventoryItem
		var active int
		if err := rows.Scan(&item.ID, &item.Name, &item.CategoryName, &item.Unit, &active); err != nil {
			return nil, err
		}
		item.Active = active == 1
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) AddKitchenCookBoxItem(ctx context.Context, boxID, inventoryItemID int64, quantity float64, notes string, userID int64) error {
	if quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		var exists int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM kitchen_cook_storage_boxes WHERE id=? AND active=1", boxID).Scan(&exists); err != nil || exists == 0 {
			if err != nil {
				return err
			}
			return sql.ErrNoRows
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO kitchen_cook_box_items(kitchen_cook_storage_box_id,inventory_item_id,quantity,notes,active,created_at,updated_at)
			VALUES(?,?,?,?,1,?,?) ON CONFLICT(kitchen_cook_storage_box_id,inventory_item_id) DO UPDATE SET quantity=excluded.quantity,notes=excluded.notes,active=1,updated_at=excluded.updated_at`, boxID, inventoryItemID, quantity, notes, now, now)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO audit_log(user_id,entity_type,entity_id,action,after_json,created_at) VALUES(?,'kitchen_cook_box',?,'add_item',json_object('inventory_item_id',?,'quantity',?),?)`, nullableUserID(userID), boxID, inventoryItemID, quantity, now)
		return err
	})
}

func (s *Store) UpdateKitchenCookBoxItem(ctx context.Context, boxID, contentID int64, quantity float64, notes string, userID int64) error {
	if quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		result, err := tx.ExecContext(ctx, `UPDATE kitchen_cook_box_items SET quantity=?,notes=?,updated_at=? WHERE id=? AND kitchen_cook_storage_box_id=? AND active=1`, quantity, notes, now, contentID, boxID)
		if err != nil {
			return err
		}
		if count, _ := result.RowsAffected(); count == 0 {
			return sql.ErrNoRows
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO audit_log(user_id,entity_type,entity_id,action,after_json,created_at) VALUES(?,'kitchen_cook_box',?,'update_item',json_object('content_id',?,'quantity',?),?)`, nullableUserID(userID), boxID, contentID, quantity, now)
		return err
	})
}

func (s *Store) RemoveKitchenCookBoxItem(ctx context.Context, boxID, contentID, userID int64) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		result, err := tx.ExecContext(ctx, `UPDATE kitchen_cook_box_items SET active=0,updated_at=? WHERE id=? AND kitchen_cook_storage_box_id=? AND active=1`, now, contentID, boxID)
		if err != nil {
			return err
		}
		if count, _ := result.RowsAffected(); count == 0 {
			return sql.ErrNoRows
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO audit_log(user_id,entity_type,entity_id,action,after_json,created_at) VALUES(?,'kitchen_cook_box',?,'remove_item',json_object('content_id',?),?)`, nullableUserID(userID), boxID, contentID, now)
		return err
	})
}

func (s *Store) EventKitchenCookContainedInventoryIDs(ctx context.Context, eventID int64) (map[int64]bool, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT DISTINCT content.inventory_item_id
		FROM events event
		JOIN kitchen_cook_storage_boxes box ON box.kitchen_cook_id=event.kitchen_cook_id AND box.active=1
		JOIN kitchen_cook_box_items content ON content.kitchen_cook_storage_box_id=box.id AND content.active=1
		WHERE event.id=?`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result[id] = true
	}
	return result, rows.Err()
}

func (s *Store) EventKitchenCookRequirements(ctx context.Context, eventID int64) ([]models.AutomaticRequirement, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT box.id,box.inventory_item_id,cook.name,box.box_type,COUNT(content.id),COALESCE(SUM(content.quantity),0)
		FROM events event
		JOIN kitchen_cooks cook ON cook.id=event.kitchen_cook_id
		JOIN kitchen_cook_storage_boxes box ON box.kitchen_cook_id=cook.id AND box.active=1
		JOIN inventory_items item ON item.id=box.inventory_item_id AND item.active=1
		LEFT JOIN kitchen_cook_box_items content ON content.kitchen_cook_storage_box_id=box.id AND content.active=1
		WHERE event.id=?
		GROUP BY box.id,box.inventory_item_id,cook.name,box.box_type
		ORDER BY CASE box.box_type WHEN 'utensils' THEN 1 ELSE 2 END`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.AutomaticRequirement
	for rows.Next() {
		var boxID, inventoryID int64
		var cookName, boxType string
		var contentCount int
		var totalQuantity float64
		if err := rows.Scan(&boxID, &inventoryID, &cookName, &boxType, &contentCount, &totalQuantity); err != nil {
			return nil, err
		}
		result = append(result, models.AutomaticRequirement{
			SourceKey:       fmt.Sprintf("kitchen-cook-box:%d", boxID),
			InventoryItemID: inventoryID,
			Quantity:        1,
			Origin:          fmt.Sprintf("Cozinheira %s: caixa da cozinheira (%d tipos; %.2g itens)", cookName, contentCount, totalQuantity),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
