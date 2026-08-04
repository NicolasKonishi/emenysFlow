package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"

	"buffetflow/internal/models"
)

func isCoffeeTableSection(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	return normalized == "mesa de café" || normalized == "mesa do café"
}

func isCakeSection(name string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(name)), "bolo")
}

func (s *Store) ValidateMenuModelSelections(ctx context.Context, modelID int64, selectedIDs []int64) error {
	selected := make(map[int64]bool, len(selectedIDs))
	for _, id := range selectedIDs {
		selected[id] = true
	}
	rows, err := s.db.QueryContext(ctx, `SELECT choice.id,choice.choice_group_name,choice.selection_min,choice.selection_max,item.id
		FROM menu_choice_groups choice
		JOIN menu_template_sections section ON section.id=choice.menu_template_section_id
		JOIN menu_choice_group_items link ON link.menu_choice_group_id=choice.id
		JOIN menu_template_items item ON item.id=link.menu_template_item_id
		WHERE section.menu_template_id=? AND choice.active=1 AND choice.deleted_at IS NULL
		AND item.active=1 AND item.deleted_at IS NULL ORDER BY choice.id`, modelID)
	if err != nil {
		return err
	}
	defer rows.Close()
	type groupState struct {
		name  string
		min   int
		max   sql.NullInt64
		count int
	}
	groups := map[int64]*groupState{}
	for rows.Next() {
		var groupID, itemID int64
		var name string
		var min int
		var max sql.NullInt64
		if err := rows.Scan(&groupID, &name, &min, &max, &itemID); err != nil {
			return err
		}
		group := groups[groupID]
		if group == nil {
			group = &groupState{name: name, min: min, max: max}
			groups[groupID] = group
		}
		if selected[itemID] {
			group.count++
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, group := range groups {
		if group.count < group.min {
			return fmt.Errorf("complete o grupo %q: escolha pelo menos %d opção(ões)", group.name, group.min)
		}
		if group.max.Valid && group.count > int(group.max.Int64) {
			return fmt.Errorf("o grupo %q permite no máximo %d opção(ões)", group.name, group.max.Int64)
		}
	}
	return nil
}

func (s *Store) MenuModelMenuItemIDs(ctx context.Context, modelID int64, selectedTemplateIDs []int64) ([]int64, error) {
	selected := make(map[int64]bool, len(selectedTemplateIDs))
	for _, id := range selectedTemplateIDs {
		selected[id] = true
	}
	rows, err := s.db.QueryContext(ctx, `SELECT item.id,item.menu_item_id,item.included
		FROM menu_template_items item
		JOIN menu_template_sections section ON section.id=item.menu_template_section_id
		WHERE section.menu_template_id=? AND item.active=1 AND item.deleted_at IS NULL
		AND item.menu_item_id IS NOT NULL`, modelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []int64
	for rows.Next() {
		var templateItemID, menuItemID int64
		var included int
		if err := rows.Scan(&templateItemID, &menuItemID, &included); err != nil {
			return nil, err
		}
		if selected[templateItemID] {
			result = append(result, menuItemID)
		}
	}
	return result, rows.Err()
}

func (s *Store) ApplyMenuModelSnapshot(ctx context.Context, eventID, modelID int64, selectedIDs []int64, customItems []string, userID int64) error {
	return s.ApplyMenuModelSnapshotWithCustomizations(ctx, eventID, modelID, selectedIDs, customItems, nil, nil, userID)
}

func (s *Store) ApplyMenuModelSnapshotWithCustomizations(ctx context.Context, eventID, modelID int64, selectedIDs []int64, customItems []string, itemNames map[int64]string, sectionCustomItems map[int64][]string, userID int64) error {
	if err := s.ValidateMenuModelSelections(ctx, modelID, selectedIDs); err != nil {
		return err
	}
	selected := make(map[int64]bool, len(selectedIDs))
	explicitSelection := selectedIDs != nil
	for _, id := range selectedIDs {
		selected[id] = true
	}
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		var hasCake int
		if err := tx.QueryRowContext(ctx, "SELECT has_cake FROM events WHERE id=?", eventID).Scan(&hasCake); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM event_menu_templates WHERE event_id=?", eventID); err != nil {
			return err
		}
		var name, description string
		var version int
		if err := tx.QueryRowContext(ctx, "SELECT name,description,current_version FROM menu_templates WHERE id=? AND active=1 AND deleted_at IS NULL", modelID).Scan(&name, &description, &version); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO event_menu_templates(event_id,source_menu_template_id,source_version,snapshot_name,snapshot_description,snapshot_json,applied_by,applied_at,updated_at)
			VALUES(?,?,?,?,?,json_object('model_id',?,'version',?),?,?,?)`, eventID, modelID, version, name, description, modelID, version, nullableUserID(userID), now, now)
		if err != nil {
			return err
		}
		snapshotID, err := result.LastInsertId()
		if err != nil {
			return err
		}
		sections, err := tx.QueryContext(ctx, `SELECT section.id,section.display_name,definition.section_type,section.sort_order,section.selection_min,section.selection_max,section.allow_event_changes,section.notes
			FROM menu_template_sections section JOIN menu_sections definition ON definition.id=section.menu_section_id
			WHERE section.menu_template_id=? AND section.active=1 AND section.deleted_at IS NULL ORDER BY section.sort_order`, modelID)
		if err != nil {
			return err
		}
		for sections.Next() {
			var sourceID int64
			var sectionName, sectionType, notes string
			var sortOrder, min, allow int
			var max sql.NullInt64
			if err := sections.Scan(&sourceID, &sectionName, &sectionType, &sortOrder, &min, &max, &allow, &notes); err != nil {
				sections.Close()
				return err
			}
			if isCoffeeTableSection(sectionName) {
				continue
			}
			sectionResult, err := tx.ExecContext(ctx, `INSERT INTO event_menu_sections(event_menu_template_id,source_template_section_id,name,section_type,sort_order,selection_min,selection_max,allow_event_changes,notes) VALUES(?,?,?,?,?,?,?,?,?)`, snapshotID, sourceID, sectionName, sectionType, sortOrder, min, nullInt64(max), allow, notes)
			if err != nil {
				sections.Close()
				return err
			}
			eventSectionID, err := sectionResult.LastInsertId()
			if err != nil {
				sections.Close()
				return err
			}
			items, err := tx.QueryContext(ctx, `SELECT id,menu_item_id,source_label,normalized_name,description,sort_order,included,notes
				FROM menu_template_items WHERE menu_template_section_id=? AND active=1 AND deleted_at IS NULL ORDER BY sort_order`, sourceID)
			if err != nil {
				sections.Close()
				return err
			}
			for items.Next() {
				var itemID int64
				var menuItemID sql.NullInt64
				var label, normalized, itemDescription, itemNotes string
				var itemSort, included int
				if err := items.Scan(&itemID, &menuItemID, &label, &normalized, &itemDescription, &itemSort, &included, &itemNotes); err != nil {
					items.Close()
					sections.Close()
					return err
				}
				isSelected := selected[itemID] || (!explicitSelection && included == 1)
				if isCakeSection(sectionName) && hasCake == 0 {
					isSelected = false
				}
				displayName := label
				if override := strings.TrimSpace(itemNames[itemID]); override != "" {
					displayName = override
				}
				result, insertErr := tx.ExecContext(ctx, `INSERT INTO event_menu_snapshot_items(event_menu_section_id,source_template_item_id,source_menu_item_id,source_label,normalized_name,display_name,description,sort_order,selected,notes,original_snapshot_json,changed_by,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,json_object('display_name',?,'description',?,'section_id',?,'sort_order',?),?,?,?)`, eventSectionID, itemID, nullInt64(menuItemID), label, normalized, displayName, itemDescription, itemSort, isSelected, itemNotes, label, itemDescription, eventSectionID, itemSort, nullableUserID(userID), now, now)
				err = insertErr
				if err != nil {
					items.Close()
					sections.Close()
					return err
				}
				snapshotItemID, _ := result.LastInsertId()
				if menuItemID.Valid {
					_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO event_menu_item_equipment(event_menu_snapshot_item_id,inventory_item_id,quantity,required,customized,notes,created_at,updated_at) SELECT ?,equipment.inventory_item_id,link.quantity,link.required,0,'',?,? FROM menu_item_equipment link JOIN equipment ON equipment.id=link.equipment_id WHERE link.menu_item_id=?`, snapshotItemID, now, now, menuItemID.Int64)
					if err != nil {
						items.Close()
						sections.Close()
						return err
					}
				}
			}
			if err := items.Err(); err != nil {
				items.Close()
				sections.Close()
				return err
			}
			items.Close()
			for index, customItem := range sectionCustomItems[sourceID] {
				customItem = strings.TrimSpace(customItem)
				if customItem == "" {
					continue
				}
				if _, err := tx.ExecContext(ctx, `INSERT INTO event_menu_snapshot_items(event_menu_section_id,source_label,normalized_name,display_name,sort_order,selected,custom_item,is_customized,original_snapshot_json,changed_by,created_at,updated_at) VALUES(?,?,?,?,?,1,1,1,json_object('display_name',?,'section_id',?,'sort_order',?),?,?,?)`, eventSectionID, customItem, customItem, customItem, index+1, customItem, eventSectionID, index+1, nullableUserID(userID), now, now); err != nil {
					return err
				}
			}
		}
		if err := sections.Err(); err != nil {
			sections.Close()
			return err
		}
		sections.Close()
		if len(customItems) > 0 {
			sectionResult, err := tx.ExecContext(ctx, `INSERT INTO event_menu_sections(event_menu_template_id,name,section_type,sort_order,selection_min,allow_event_changes,notes) VALUES(?,'Itens personalizados','food',999,0,1,'Adicionados especificamente neste evento')`, snapshotID)
			if err != nil {
				return err
			}
			sectionID, err := sectionResult.LastInsertId()
			if err != nil {
				return err
			}
			for index, customItem := range customItems {
				customItem = strings.TrimSpace(customItem)
				if customItem == "" {
					continue
				}
				if _, err := tx.ExecContext(ctx, `INSERT INTO event_menu_snapshot_items(event_menu_section_id,source_label,normalized_name,display_name,sort_order,selected,custom_item,is_customized,original_snapshot_json,changed_by,created_at,updated_at) VALUES(?,?,?,?,?,1,1,1,json_object('display_name',?,'section_id',?,'sort_order',?),?,?,?)`, sectionID, customItem, customItem, customItem, index+1, customItem, sectionID, index+1, nullableUserID(userID), now, now); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (s *Store) ClearMenuModelSnapshot(ctx context.Context, eventID int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM event_menu_templates WHERE event_id=?", eventID)
	return err
}

func (s *Store) EventMenuModelSelection(ctx context.Context, eventID int64) (int64, []int64, error) {
	var modelID int64
	err := s.db.QueryRowContext(ctx, "SELECT source_menu_template_id FROM event_menu_templates WHERE event_id=?", eventID).Scan(&modelID)
	if err == sql.ErrNoRows {
		return 0, nil, nil
	}
	if err != nil {
		return 0, nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT item.source_template_item_id FROM event_menu_snapshot_items item
		JOIN event_menu_sections section ON section.id=item.event_menu_section_id
		JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id
		WHERE snapshot.event_id=? AND LOWER(section.name) NOT IN ('mesa de café','mesa do café') AND item.selected=1 AND item.source_template_item_id IS NOT NULL`, eventID)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, nil, err
		}
		ids = append(ids, id)
	}
	return modelID, ids, rows.Err()
}

func (s *Store) EventMenuModelCustomItems(ctx context.Context, eventID int64) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT item.normalized_name FROM event_menu_snapshot_items item
		JOIN event_menu_sections section ON section.id=item.event_menu_section_id
		JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id
		WHERE snapshot.event_id=? AND LOWER(section.name) NOT IN ('mesa de café','mesa do café') AND item.custom_item=1 AND item.selected=1 ORDER BY item.sort_order,item.id`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []string
	for rows.Next() {
		var item string
		if err := rows.Scan(&item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) EventMenuModelStatus(ctx context.Context, eventID int64) (snapshotVersion, currentVersion int, modelName string, err error) {
	err = s.db.QueryRowContext(ctx, `SELECT snapshot.source_version,model.current_version,model.name FROM event_menu_templates snapshot JOIN menu_templates model ON model.id=snapshot.source_menu_template_id WHERE snapshot.event_id=?`, eventID).Scan(&snapshotVersion, &currentVersion, &modelName)
	if err == sql.ErrNoRows {
		return 0, 0, "", nil
	}
	return snapshotVersion, currentVersion, modelName, err
}

func (s *Store) EventMenuModelItemConfigurations(ctx context.Context, eventID int64) (map[int64]models.EventMenuItemConfiguration, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT item.source_template_item_id,item.portions,item.container_type_id FROM event_menu_snapshot_items item JOIN event_menu_sections section ON section.id=item.event_menu_section_id JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id WHERE snapshot.event_id=? AND LOWER(section.name) NOT IN ('mesa de café','mesa do café') AND item.source_template_item_id IS NOT NULL`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[int64]models.EventMenuItemConfiguration{}
	for rows.Next() {
		var item models.EventMenuItemConfiguration
		if err := rows.Scan(&item.TemplateItemID, &item.Portions, &item.ContainerTypeID); err != nil {
			return nil, err
		}
		result[item.TemplateItemID] = item
	}
	return result, rows.Err()
}

func (s *Store) UpdateEventMenuSnapshotConfigurations(ctx context.Context, eventID int64, configurations map[int64]models.EventMenuItemConfiguration, userID int64) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		for templateItemID, configuration := range configurations {
			_, err := tx.ExecContext(ctx, `UPDATE event_menu_snapshot_items SET portions=?,container_type_id=?,changed_by=?,updated_at=? WHERE source_template_item_id=? AND event_menu_section_id IN (SELECT section.id FROM event_menu_sections section JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id WHERE snapshot.event_id=?)`, nullableFloat64(configuration.Portions), nullInt64(configuration.ContainerTypeID), nullableUserID(userID), now, templateItemID, eventID)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func nullableFloat64(value sql.NullFloat64) any {
	if !value.Valid {
		return nil
	}
	return value.Float64
}

func (s *Store) CompareEventMenuModel(ctx context.Context, eventID int64) ([]models.ModelDifference, error) {
	modelID, _, err := s.EventMenuModelSelection(ctx, eventID)
	if err != nil || modelID == 0 {
		return nil, err
	}
	currentRows, err := s.db.QueryContext(ctx, `SELECT item.id,item.normalized_name FROM menu_template_items item JOIN menu_template_sections section ON section.id=item.menu_template_section_id WHERE section.menu_template_id=? AND LOWER(section.display_name) NOT IN ('mesa de café','mesa do café') AND item.active=1 AND item.deleted_at IS NULL`, modelID)
	if err != nil {
		return nil, err
	}
	current := map[int64]string{}
	for currentRows.Next() {
		var id int64
		var name string
		if err := currentRows.Scan(&id, &name); err != nil {
			currentRows.Close()
			return nil, err
		}
		current[id] = name
	}
	if err := currentRows.Err(); err != nil {
		currentRows.Close()
		return nil, err
	}
	currentRows.Close()
	snapshotRows, err := s.db.QueryContext(ctx, `SELECT item.source_template_item_id,item.normalized_name FROM event_menu_snapshot_items item JOIN event_menu_sections section ON section.id=item.event_menu_section_id JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id WHERE snapshot.event_id=? AND LOWER(section.name) NOT IN ('mesa de café','mesa do café') AND item.source_template_item_id IS NOT NULL`, eventID)
	if err != nil {
		return nil, err
	}
	snapshot := map[int64]string{}
	for snapshotRows.Next() {
		var id int64
		var name string
		if err := snapshotRows.Scan(&id, &name); err != nil {
			snapshotRows.Close()
			return nil, err
		}
		snapshot[id] = name
	}
	if err := snapshotRows.Err(); err != nil {
		snapshotRows.Close()
		return nil, err
	}
	snapshotRows.Close()
	var differences []models.ModelDifference
	for id, currentName := range current {
		snapshotName, existed := snapshot[id]
		if !existed {
			differences = append(differences, models.ModelDifference{Kind: "added", ItemName: currentName, Detail: "Adicionado na versão atual do modelo"})
		} else if snapshotName != currentName {
			differences = append(differences, models.ModelDifference{Kind: "changed", ItemName: currentName, Detail: "No evento: " + snapshotName})
		}
	}
	for id, snapshotName := range snapshot {
		if _, exists := current[id]; !exists {
			differences = append(differences, models.ModelDifference{Kind: "removed", ItemName: snapshotName, Detail: "Removido da versão atual do modelo"})
		}
	}
	sort.Slice(differences, func(i, j int) bool {
		if differences[i].Kind == differences[j].Kind {
			return differences[i].ItemName < differences[j].ItemName
		}
		return differences[i].Kind < differences[j].Kind
	})
	return differences, nil
}

func (s *Store) ApplyServiceSnapshots(ctx context.Context, eventID int64, serviceIDs []int64, userID int64) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		if _, err := tx.ExecContext(ctx, "DELETE FROM event_services WHERE event_id=?", eventID); err != nil {
			return err
		}
		seen := map[int64]bool{}
		for _, serviceID := range serviceIDs {
			if seen[serviceID] {
				continue
			}
			seen[serviceID] = true
			var name, config string
			var version int
			var duration sql.NullInt64
			var supplier sql.NullString
			if err := tx.QueryRowContext(ctx, "SELECT name,current_version,configuration_json,duration_minutes,supplier FROM service_templates WHERE id=? AND active=1 AND deleted_at IS NULL", serviceID).Scan(&name, &version, &config, &duration, &supplier); err != nil {
				return err
			}
			result, err := tx.ExecContext(ctx, `INSERT INTO event_services(event_id,source_service_template_id,source_version,snapshot_name,snapshot_json,duration_minutes,supplier,status,applied_by,applied_at,updated_at) VALUES(?,?,?,?,?,?,?,'planned',?,?,?)`, eventID, serviceID, version, name, config, nullInt64(duration), nullableSQLString(supplier), nullableUserID(userID), now, now)
			if err != nil {
				return err
			}
			eventServiceID, err := result.LastInsertId()
			if err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO event_service_components(event_service_id,source_template_component_id,name,description,selected,configuration_json,created_at,updated_at)
				SELECT ?,link.id,component.name,component.description,link.included,link.configuration_json,?,?
				FROM service_template_components link JOIN service_components component ON component.id=link.service_component_id
				WHERE link.service_template_id=? AND link.active=1 AND link.deleted_at IS NULL`, eventServiceID, now, now, serviceID)
			if err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO event_service_component_inventory_links(
				event_service_component_id,source_inventory_link_id,inventory_item_id,quantity_formula,ownership,supplier,pickup_notes,return_notes,active,created_at
			)
				SELECT event_component.id,source_link.id,source_link.inventory_item_id,source_link.quantity_formula,source_link.ownership,
					source_link.supplier,source_link.pickup_notes,source_link.return_notes,source_link.active,?
				FROM event_service_components event_component
				JOIN service_component_inventory_links source_link ON source_link.service_template_component_id=event_component.source_template_component_id
				WHERE event_component.event_service_id=? AND source_link.active=1`, now, eventServiceID)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func nullableSQLString(value sql.NullString) any {
	if !value.Valid {
		return nil
	}
	return value.String
}

func (s *Store) EventServiceModelIDs(ctx context.Context, eventID int64) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT source_service_template_id FROM event_services WHERE event_id=? AND deleted_at IS NULL", eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *Store) EventServiceRequirements(ctx context.Context, eventID int64, guests int) ([]models.AutomaticRequirement, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT link.id,link.inventory_item_id,link.quantity_formula,event_component.name
		FROM event_services service
		JOIN event_service_components event_component ON event_component.event_service_id=service.id AND event_component.selected=1
		JOIN event_service_component_inventory_links link ON link.event_service_component_id=event_component.id AND link.active=1
		WHERE service.event_id=? AND service.deleted_at IS NULL`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.AutomaticRequirement
	for rows.Next() {
		var id, inventoryID int64
		var formula, name string
		if err := rows.Scan(&id, &inventoryID, &formula, &name); err != nil {
			return nil, err
		}
		quantity := 1.0
		switch formula {
		case "ceil(guests/8)":
			quantity = math.Ceil(float64(guests) / 8)
		case "guests":
			quantity = float64(guests)
		}
		result = append(result, models.AutomaticRequirement{SourceKey: fmt.Sprintf("service:%d", id), InventoryItemID: inventoryID, Quantity: quantity, Origin: "Serviço contratado: " + name})
	}
	return result, rows.Err()
}
