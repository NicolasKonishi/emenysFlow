package repositories

import (
	"context"
	"fmt"

	"buffetflow/internal/models"
)

func (s *Store) Dashboard(ctx context.Context) (models.Dashboard, error) {
	var dashboard models.Dashboard
	events, err := s.ListEvents(ctx, "")
	if err != nil {
		return dashboard, err
	}
	if len(events) > 5 {
		events = events[:5]
	}
	dashboard.UpcomingEvents = events
	query := `SELECT
		(SELECT COUNT(DISTINCT c.event_id) FROM checklists c JOIN checklist_items ci ON ci.checklist_id=c.id JOIN events e ON e.id=c.event_id WHERE ci.missing_quantity>0 AND e.status<>'cancelled'),
		(SELECT COUNT(*) FROM events WHERE active=1 AND status IN ('planning','reserved')),
		(SELECT COUNT(*) FROM events WHERE active=1 AND status IN ('returning','post_event_check')),
		(SELECT COUNT(*) FROM inventory_items item WHERE active=1 AND stock_quantity-damaged_quantity < minimum_stock
			AND item.id NOT IN (SELECT inventory_item_id FROM kitchen_cook_storage_boxes)
			AND NOT EXISTS (SELECT 1 FROM kitchen_cook_box_items content JOIN kitchen_cook_storage_boxes box ON box.id=content.kitchen_cook_storage_box_id WHERE content.inventory_item_id=item.id AND content.active=1 AND box.active=1)),
		(SELECT COUNT(*) FROM inventory_items item WHERE active=1 AND damaged_quantity>0
			AND item.id NOT IN (SELECT inventory_item_id FROM kitchen_cook_storage_boxes)
			AND NOT EXISTS (SELECT 1 FROM kitchen_cook_box_items content JOIN kitchen_cook_storage_boxes box ON box.id=content.kitchen_cook_storage_box_id WHERE content.inventory_item_id=item.id AND content.active=1 AND box.active=1)),
		(SELECT COUNT(*) FROM rental_items WHERE returned_at IS NULL AND return_at < datetime('now','+3 days')),
		(SELECT COUNT(*) FROM checklist_shortages WHERE resolution_type='purchase' AND status NOT IN ('resolved','cancelled')),
		(SELECT COUNT(*) FROM checklist_shortages WHERE resolution_type='rental' AND status NOT IN ('resolved','cancelled'))`
	err = s.db.QueryRowContext(ctx, query).Scan(&dashboard.EventsWithShortages, &dashboard.AwaitingSeparation, &dashboard.AwaitingReturn, &dashboard.BelowMinimumStock, &dashboard.DamagedItems, &dashboard.RentalsDue, &dashboard.PendingPurchases, &dashboard.PendingRentals)
	if err != nil {
		return dashboard, fmt.Errorf("load dashboard: %w", err)
	}
	return dashboard, nil
}
