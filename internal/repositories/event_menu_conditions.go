package repositories

import "context"

// EventHasDadinhoTapioca checks both the legacy selectable menu and the
// versioned event snapshot, including custom items added by the customer.
func (s *Store) EventHasDadinhoTapioca(ctx context.Context, eventID int64) (bool, error) {
	var found int
	err := s.db.QueryRowContext(ctx, `SELECT CASE WHEN
		EXISTS (
			SELECT 1 FROM event_menu_items selected
			JOIN menu_items item ON item.id=selected.menu_item_id
			WHERE selected.event_id=? AND LOWER(COALESCE(NULLIF(item.display_name,''),item.name)) LIKE '%dadinho de tapioca%'
		)
		OR EXISTS (
			SELECT 1 FROM event_menu_snapshot_items item
			JOIN event_menu_sections section ON section.id=item.event_menu_section_id
			JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id
			WHERE snapshot.event_id=? AND item.selected=1 AND LOWER(item.normalized_name) LIKE '%dadinho de tapioca%'
		)
	THEN 1 ELSE 0 END`, eventID, eventID).Scan(&found)
	return found == 1, err
}
