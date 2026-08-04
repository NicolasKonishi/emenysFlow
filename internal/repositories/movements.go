package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"buffetflow/internal/models"
)

func (s *Store) ListInventoryMovements(ctx context.Context, itemID int64) ([]models.InventoryMovement, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT m.id,m.movement_type,m.quantity,m.previous_stock,m.new_stock,m.reason,COALESCE(e.name,''),m.created_at FROM inventory_movements m LEFT JOIN events e ON e.id=m.event_id WHERE m.inventory_item_id=? ORDER BY m.created_at DESC,m.id DESC LIMIT 200`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.InventoryMovement
	for rows.Next() {
		var item models.InventoryMovement
		var created string
		if err := rows.Scan(&item.ID, &item.MovementType, &item.Quantity, &item.PreviousStock, &item.NewStock, &item.Reason, &item.EventName, &created); err != nil {
			return nil, err
		}
		item.CreatedAt = parseTime(created)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) AdjustInventory(ctx context.Context, itemID int64, movementType string, quantity float64, reason string, userID int64) error {
	valid := map[string]bool{"in": true, "out": true, "adjustment": true, "damage": true, "loss": true}
	if !valid[movementType] || quantity < 0 || reason == "" {
		return fmt.Errorf("invalid inventory adjustment")
	}
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		var stock, damaged float64
		if err := tx.QueryRowContext(ctx, "SELECT stock_quantity,damaged_quantity FROM inventory_items WHERE id=? AND active=1", itemID).Scan(&stock, &damaged); err != nil {
			return err
		}
		newStock, newDamaged := stock, damaged
		movementQuantity := quantity
		switch movementType {
		case "in":
			newStock += quantity
		case "out", "loss":
			newStock -= quantity
		case "adjustment":
			newStock = quantity
			movementQuantity = quantity - stock
		case "damage":
			newDamaged += quantity
		}
		if newStock < 0 {
			return fmt.Errorf("stock cannot become negative")
		}
		if newDamaged > newStock {
			return fmt.Errorf("damaged quantity cannot exceed stock")
		}
		now := nowString()
		if _, err := tx.ExecContext(ctx, "UPDATE inventory_items SET stock_quantity=?,damaged_quantity=?,updated_at=? WHERE id=?", newStock, newDamaged, now, itemID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO inventory_movements(inventory_item_id,movement_type,quantity,previous_stock,new_stock,reason,performed_by,created_at) VALUES(?,?,?,?,?,?,?,?)`, itemID, movementType, movementQuantity, stock, newStock, reason, nullableUserID(userID), now)
		return err
	})
}
