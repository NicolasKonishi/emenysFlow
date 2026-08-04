package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"buffetflow/internal/models"
)

func (s *Store) EventHasMenuSnapshot(ctx context.Context, eventID int64) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM event_menu_templates WHERE event_id=?`, eventID).Scan(&count)
	return count > 0, err
}

func (s *Store) EventMenuSnapshotSections(ctx context.Context, eventID int64) ([]models.EventMenuSnapshotSection, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT section.id,section.name,section.section_type,section.sort_order,section.allow_event_changes,
		item.id,item.source_template_item_id,item.source_menu_item_id,item.display_name,item.description,item.sort_order,item.selected,item.custom_item,item.is_customized,item.was_removed,item.portions,item.container_type_id,item.notes,item.row_version,
		CASE WHEN LOWER(section.name) LIKE '%entrada%' OR LOWER(section.name) LIKE '%sobremesa%' OR LOWER(section.name) LIKE '%bolo%' THEN 1 ELSE 0 END
		FROM event_menu_sections section JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id
		LEFT JOIN event_menu_snapshot_items item ON item.event_menu_section_id=section.id
		WHERE snapshot.event_id=? AND LOWER(section.name) NOT IN ('mesa de café','mesa do café') ORDER BY section.sort_order,section.id,item.sort_order,item.id`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.EventMenuSnapshotSection
	byID := map[int64]int{}
	for rows.Next() {
		var sectionID int64
		var name, sectionType string
		var sortOrder, allow int
		var itemID sql.NullInt64
		var item models.EventMenuSnapshotItem
		var selected, custom, customized, removed, canChoose sql.NullInt64
		if err := rows.Scan(&sectionID, &name, &sectionType, &sortOrder, &allow, &itemID, &item.SourceTemplateItemID, &item.SourceMenuItemID, &item.DisplayName, &item.Description, &item.SortOrder, &selected, &custom, &customized, &removed, &item.Portions, &item.ContainerTypeID, &item.Notes, &item.RowVersion, &canChoose); err != nil {
			return nil, err
		}
		index, ok := byID[sectionID]
		if !ok {
			index = len(result)
			byID[sectionID] = index
			result = append(result, models.EventMenuSnapshotSection{ID: sectionID, Name: name, SectionType: sectionType, SortOrder: sortOrder, AllowEventChanges: allow == 1})
		}
		if itemID.Valid {
			item.ID = itemID.Int64
			item.EventMenuSectionID = sectionID
			item.SectionName = name
			item.SectionType = sectionType
			item.Selected = selected.Int64 == 1
			item.CustomItem = custom.Int64 == 1
			item.Customized = customized.Int64 == 1
			item.WasRemoved = removed.Int64 == 1
			item.CanChooseContainer = canChoose.Int64 == 1
			result[index].Items = append(result[index].Items, item)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	containers, err := s.eventMenuContainers(ctx, eventID)
	if err != nil {
		return nil, err
	}
	equipment, err := s.eventMenuEquipment(ctx, eventID)
	if err != nil {
		return nil, err
	}
	for sectionIndex := range result {
		for itemIndex := range result[sectionIndex].Items {
			item := &result[sectionIndex].Items[itemIndex]
			item.Containers = containers[item.ID]
			item.Equipment = equipment[item.ID]
		}
	}
	return result, nil
}

func (s *Store) AddEventMenuSnapshotItem(ctx context.Context, eventID, sectionID, sourceMenuItemID int64, name, description string, portions float64, userID int64) (int64, error) {
	name = strings.TrimSpace(name)
	if sourceMenuItemID <= 0 && name == "" {
		return 0, fmt.Errorf("item name is required")
	}
	if portions <= 0 {
		portions = 1
	}
	var itemID int64
	err := withTx(ctx, s.db, func(tx *sql.Tx) error {
		var snapshotEventID int64
		if err := tx.QueryRowContext(ctx, `SELECT snapshot.event_id FROM event_menu_sections section JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id WHERE section.id=?`, sectionID).Scan(&snapshotEventID); err != nil {
			return err
		}
		if snapshotEventID != eventID {
			return sql.ErrNoRows
		}
		var label, normalized, itemDescription string
		sourceID := any(nil)
		custom := 1
		if sourceMenuItemID > 0 {
			if err := tx.QueryRowContext(ctx, `SELECT COALESCE(NULLIF(display_name,''),name),COALESCE(NULLIF(normalized_name,''),name),description FROM menu_items WHERE id=? AND active=1`, sourceMenuItemID).Scan(&label, &normalized, &itemDescription); err != nil {
				return err
			}
			sourceID = sourceMenuItemID
			custom = 0
		} else {
			label = name
			normalized = name
			itemDescription = description
		}
		var sortOrder int
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sort_order),0)+10 FROM event_menu_snapshot_items WHERE event_menu_section_id=?`, sectionID).Scan(&sortOrder); err != nil {
			return err
		}
		now := nowString()
		original, _ := json.Marshal(map[string]any{"display_name": label, "description": itemDescription, "section_id": sectionID, "sort_order": sortOrder})
		result, err := tx.ExecContext(ctx, `INSERT INTO event_menu_snapshot_items(event_menu_section_id,source_menu_item_id,source_label,normalized_name,display_name,description,sort_order,selected,custom_item,is_customized,original_snapshot_json,portions,changed_by,created_at,updated_at) VALUES(?,?,?,?,?,?,?,1,?,1,?,?,?, ?,?)`, sectionID, sourceID, label, normalized, label, itemDescription, sortOrder, custom, string(original), portions, nullableUserID(userID), now, now)
		if err != nil {
			return err
		}
		itemID, _ = result.LastInsertId()
		_, err = tx.ExecContext(ctx, `INSERT INTO event_menu_change_history(event_id,snapshot_item_id,action,after_json,changed_by,created_at) VALUES(?,?,'added',json_object('name',?,'section_id',?),?,?)`, eventID, itemID, label, sectionID, nullableUserID(userID), now)
		return err
	})
	if err == nil {
		err = s.SyncLegacyEventMenuFromSnapshot(ctx, eventID)
	}
	return itemID, err
}

func (s *Store) UpdateEventMenuSnapshotItem(ctx context.Context, eventID, itemID, sectionID int64, name, description, notes string, sortOrder int, portions float64, containerTypeID sql.NullInt64, userID int64) error {
	name = strings.TrimSpace(name)
	if name == "" || portions <= 0 {
		return fmt.Errorf("name and portions are required")
	}
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		var canChoose int
		var before string
		if err := tx.QueryRowContext(ctx, `SELECT CASE WHEN LOWER(section.name) LIKE '%entrada%' OR LOWER(section.name) LIKE '%sobremesa%' OR LOWER(section.name) LIKE '%bolo%' THEN 1 ELSE 0 END,json_object('name',item.display_name,'description',item.description,'section_id',item.event_menu_section_id,'sort_order',item.sort_order,'portions',item.portions,'container_type_id',item.container_type_id) FROM event_menu_snapshot_items item JOIN event_menu_sections section ON section.id=item.event_menu_section_id JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id WHERE item.id=? AND snapshot.event_id=?`, itemID, eventID).Scan(&canChoose, &before); err != nil {
			return err
		}
		if canChoose == 0 {
			containerTypeID = sql.NullInt64{}
		}
		if containerTypeID.Valid {
			var allowed int
			if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM container_type_menu_categories link JOIN menu_categories category ON category.id=link.menu_category_id JOIN event_menu_snapshot_items item ON item.id=? LEFT JOIN menu_items source ON source.id=item.source_menu_item_id WHERE link.container_type_id=? AND (category.id=source.category_id OR LOWER(category.name) IN ('entradas','sobremesas'))`, itemID, containerTypeID.Int64).Scan(&allowed); err != nil {
				return err
			}
			if allowed == 0 {
				return fmt.Errorf("container is not allowed for this menu category")
			}
		}
		now := nowString()
		customized, _ := json.Marshal(map[string]any{"display_name": name, "description": description, "section_id": sectionID, "sort_order": sortOrder, "portions": portions, "notes": notes})
		result, err := tx.ExecContext(ctx, `UPDATE event_menu_snapshot_items SET event_menu_section_id=?,display_name=?,normalized_name=?,description=?,sort_order=?,portions=?,container_type_id=?,notes=?,selected=1,was_removed=0,is_customized=1,customized_data_json=?,changed_by=?,row_version=row_version+1,updated_at=? WHERE id=? AND event_menu_section_id IN (SELECT section.id FROM event_menu_sections section JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id WHERE snapshot.event_id=?)`, sectionID, name, name, description, sortOrder, portions, nullInt64(containerTypeID), notes, string(customized), nullableUserID(userID), now, itemID, eventID)
		if err != nil {
			return err
		}
		if count, _ := result.RowsAffected(); count == 0 {
			return sql.ErrNoRows
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO event_menu_change_history(event_id,snapshot_item_id,action,before_json,after_json,changed_by,created_at) VALUES(?,?,'updated',?,json_object('name',?,'section_id',?,'sort_order',?,'portions',?),?,?)`, eventID, itemID, before, name, sectionID, sortOrder, portions, nullableUserID(userID), now)
		return err
	})
}

func (s *Store) SetEventMenuSnapshotItemRemoved(ctx context.Context, eventID, itemID int64, removed bool, userID int64) error {
	now := nowString()
	value := 0
	action := "restored"
	if removed {
		value = 1
		action = "removed"
	}
	result, err := s.db.ExecContext(ctx, `UPDATE event_menu_snapshot_items SET selected=?,was_removed=?,is_customized=1,changed_by=?,row_version=row_version+1,updated_at=? WHERE id=? AND event_menu_section_id IN (SELECT section.id FROM event_menu_sections section JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id WHERE snapshot.event_id=?)`, !removed, value, nullableUserID(userID), now, itemID, eventID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	_, _ = s.db.ExecContext(ctx, `INSERT INTO event_menu_change_history(event_id,snapshot_item_id,action,changed_by,created_at) VALUES(?,?,?,?,?)`, eventID, itemID, action, nullableUserID(userID), now)
	return s.SyncLegacyEventMenuFromSnapshot(ctx, eventID)
}

func (s *Store) SyncLegacyEventMenuFromSnapshot(ctx context.Context, eventID int64) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM event_menu_items WHERE event_id=?`, eventID); err != nil {
			return err
		}
		now := nowString()
		_, err := tx.ExecContext(ctx, `INSERT INTO event_menu_items(event_id,menu_item_id,portions,container_type_id,calculated_container_quantity,notes,created_at,updated_at)
		SELECT ?,item.source_menu_item_id,CAST(COALESCE(item.portions,event.guest_count) AS INTEGER),item.container_type_id,
		CASE WHEN COALESCE(source.container_capacity_portions,container.capacity_portions,0)>0 THEN CEIL(COALESCE(item.portions,event.guest_count)/COALESCE(source.container_capacity_portions,container.capacity_portions)) ELSE 1 END,item.notes,?,?
		FROM event_menu_snapshot_items item JOIN event_menu_sections section ON section.id=item.event_menu_section_id JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id JOIN events event ON event.id=snapshot.event_id JOIN menu_items source ON source.id=item.source_menu_item_id LEFT JOIN container_types container ON container.id=COALESCE(item.container_type_id,source.container_type_id)
		WHERE snapshot.event_id=? AND LOWER(section.name) NOT IN ('mesa de café','mesa do café') AND item.selected=1 AND item.was_removed=0 AND item.source_menu_item_id IS NOT NULL`, eventID, now, now, eventID)
		return err
	})
}
