package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"buffetflow/internal/models"
)

func bumpMenuModelVersion(ctx context.Context, tx *sql.Tx, modelID, userID int64, summary string) error {
	now := nowString()
	result, err := tx.ExecContext(ctx, "UPDATE menu_templates SET current_version=current_version+1,updated_at=? WHERE id=? AND deleted_at IS NULL", now, modelID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	var version int
	if err := tx.QueryRowContext(ctx, "SELECT current_version FROM menu_templates WHERE id=?", modelID).Scan(&version); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO menu_template_versions(menu_template_id,version,change_summary,snapshot_json,created_by,created_at) VALUES(?,?,?,'{}',?,?)`, modelID, version, summary, nullableUserID(userID), now)
	return err
}

func bumpServiceModelVersion(ctx context.Context, tx *sql.Tx, modelID, userID int64, summary string) error {
	now := nowString()
	result, err := tx.ExecContext(ctx, "UPDATE service_templates SET current_version=current_version+1,updated_at=? WHERE id=? AND deleted_at IS NULL", now, modelID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	var version int
	if err := tx.QueryRowContext(ctx, "SELECT current_version FROM service_templates WHERE id=?", modelID).Scan(&version); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO service_template_versions(service_template_id,version,change_summary,snapshot_json,created_by,created_at) VALUES(?,?,?,'{}',?,?)`, modelID, version, summary, nullableUserID(userID), now)
	return err
}

func (s *Store) ListMenuModels(ctx context.Context, includeInactive bool) ([]models.MenuModel, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT model.id,model.slug,model.name,model.description,model.menu_type,model.image_url,
		model.active,model.current_version,model.source_name,model.source_updated_month,
		(SELECT COUNT(*) FROM menu_template_sections section WHERE section.menu_template_id=model.id AND section.deleted_at IS NULL),
		(SELECT COUNT(*) FROM menu_template_items item JOIN menu_template_sections section ON section.id=item.menu_template_section_id WHERE section.menu_template_id=model.id AND item.active=1 AND item.deleted_at IS NULL),
		(SELECT COUNT(*) FROM menu_choice_groups choice JOIN menu_template_sections section ON section.id=choice.menu_template_section_id WHERE section.menu_template_id=model.id AND choice.active=1 AND choice.deleted_at IS NULL),
		COALESCE((SELECT GROUP_CONCAT(item.id) FROM menu_template_items item JOIN menu_template_sections section ON section.id=item.menu_template_section_id WHERE section.menu_template_id=model.id AND item.active=1 AND item.deleted_at IS NULL AND item.included=1),''),
		model.created_at,model.updated_at
		FROM menu_templates model
		WHERE model.deleted_at IS NULL AND (?=1 OR model.active=1)
		ORDER BY model.active DESC,model.name`, includeInactive)
	if err != nil {
		return nil, fmt.Errorf("list menu models: %w", err)
	}
	defer rows.Close()
	var result []models.MenuModel
	for rows.Next() {
		var item models.MenuModel
		var active int
		var createdAt, updatedAt string
		if err := rows.Scan(&item.ID, &item.Slug, &item.Name, &item.Description, &item.MenuType, &item.ImageURL,
			&active, &item.CurrentVersion, &item.SourceName, &item.SourceUpdatedMonth,
			&item.SectionCount, &item.ItemCount, &item.ChoiceGroupCount, &item.ItemIDsCSV, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		item.Active = active == 1
		item.CreatedAt, item.UpdatedAt = parseTime(createdAt), parseTime(updatedAt)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) GetMenuModel(ctx context.Context, id int64) (models.MenuModel, error) {
	items, err := s.ListMenuModels(ctx, true)
	if err != nil {
		return models.MenuModel{}, err
	}
	for _, item := range items {
		if item.ID == id {
			return item, nil
		}
	}
	return models.MenuModel{}, sql.ErrNoRows
}

func (s *Store) MenuModelSections(ctx context.Context, modelID int64) ([]models.MenuModelSection, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT section.id,section.menu_template_id,section.menu_section_id,section.display_name,definition.section_type,section.sort_order,section.required,section.selection_min,section.selection_max,section.allow_event_changes,section.notes FROM menu_template_sections section JOIN menu_sections definition ON definition.id=section.menu_section_id WHERE section.menu_template_id=? AND section.active=1 AND section.deleted_at IS NULL ORDER BY section.sort_order,section.id`, modelID)
	if err != nil {
		return nil, err
	}
	var result []models.MenuModelSection
	for rows.Next() {
		var item models.MenuModelSection
		var required, allow int
		if err := rows.Scan(&item.ID, &item.MenuModelID, &item.SectionID, &item.Name, &item.SectionType, &item.SortOrder, &required, &item.SelectionMin, &item.SelectionMax, &allow, &item.Notes); err != nil {
			rows.Close()
			return nil, err
		}
		item.Required, item.AllowEventChanges = required == 1, allow == 1
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	for sectionIndex := range result {
		section := &result[sectionIndex]
		itemRows, err := s.db.QueryContext(ctx, `SELECT id,menu_template_section_id,menu_item_id,slug,source_label,normalized_name,description,sort_order,included,optional,configurable,notes,active FROM menu_template_items WHERE menu_template_section_id=? AND deleted_at IS NULL ORDER BY sort_order,id`, section.ID)
		if err != nil {
			return nil, err
		}
		for itemRows.Next() {
			var child models.MenuModelItem
			var included, optional, configurable, active int
			if err := itemRows.Scan(&child.ID, &child.SectionID, &child.MenuItemID, &child.Slug, &child.SourceLabel, &child.NormalizedName, &child.Description, &child.SortOrder, &included, &optional, &configurable, &child.Notes, &active); err != nil {
				itemRows.Close()
				return nil, err
			}
			child.Included, child.Optional, child.Configurable, child.Active = included == 1, optional == 1, configurable == 1, active == 1
			section.Items = append(section.Items, child)
		}
		if err := itemRows.Err(); err != nil {
			itemRows.Close()
			return nil, err
		}
		itemRows.Close()

		groupRows, err := s.db.QueryContext(ctx, `SELECT id,menu_template_section_id,slug,choice_group_name,selection_min,selection_max,selection_required,allow_extra_items,allow_custom_item,configurable FROM menu_choice_groups WHERE menu_template_section_id=? AND active=1 AND deleted_at IS NULL ORDER BY sort_order,id`, section.ID)
		if err != nil {
			return nil, err
		}
		for groupRows.Next() {
			var group models.MenuChoiceGroup
			var required, extra, custom, config int
			if err := groupRows.Scan(&group.ID, &group.SectionID, &group.Slug, &group.Name, &group.SelectionMin, &group.SelectionMax, &required, &extra, &custom, &config); err != nil {
				groupRows.Close()
				return nil, err
			}
			group.Required, group.AllowExtraItems, group.AllowCustomItem, group.Configurable = required == 1, extra == 1, custom == 1, config == 1
			section.ChoiceGroups = append(section.ChoiceGroups, group)
		}
		if err := groupRows.Err(); err != nil {
			groupRows.Close()
			return nil, err
		}
		groupRows.Close()

		for groupIndex := range section.ChoiceGroups {
			group := &section.ChoiceGroups[groupIndex]
			choiceRows, choiceErr := s.db.QueryContext(ctx, `SELECT item.id,item.menu_template_section_id,item.menu_item_id,item.slug,item.source_label,item.normalized_name,item.description,item.sort_order,item.included,item.optional,item.configurable,item.notes,item.active FROM menu_choice_group_items choice JOIN menu_template_items item ON item.id=choice.menu_template_item_id WHERE choice.menu_choice_group_id=? AND item.active=1 AND item.deleted_at IS NULL ORDER BY choice.sort_order,item.id`, group.ID)
			if choiceErr != nil {
				return nil, choiceErr
			}
			for choiceRows.Next() {
				var child models.MenuModelItem
				var included, optional, configurable, active int
				if err := choiceRows.Scan(&child.ID, &child.SectionID, &child.MenuItemID, &child.Slug, &child.SourceLabel, &child.NormalizedName, &child.Description, &child.SortOrder, &included, &optional, &configurable, &child.Notes, &active); err != nil {
					choiceRows.Close()
					return nil, err
				}
				child.Included, child.Optional, child.Configurable, child.Active = included == 1, optional == 1, configurable == 1, active == 1
				group.Items = append(group.Items, child)
				for itemIndex := range section.Items {
					if section.Items[itemIndex].ID == child.ID {
						section.Items[itemIndex].InChoiceGroup = true
					}
				}
			}
			if err := choiceRows.Err(); err != nil {
				choiceRows.Close()
				return nil, err
			}
			choiceRows.Close()
		}
	}
	return result, nil
}

func (s *Store) UpdateMenuModel(ctx context.Context, item models.MenuModel, userID int64) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		result, err := tx.ExecContext(ctx, `UPDATE menu_templates SET name=?,description=?,menu_type=?,active=?,current_version=current_version+1,updated_at=? WHERE id=? AND deleted_at IS NULL`, item.Name, item.Description, item.MenuType, item.Active, now, item.ID)
		if err != nil {
			return err
		}
		if count, _ := result.RowsAffected(); count == 0 {
			return sql.ErrNoRows
		}
		var version int
		if err := tx.QueryRowContext(ctx, "SELECT current_version FROM menu_templates WHERE id=?", item.ID).Scan(&version); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO menu_template_versions(menu_template_id,version,change_summary,snapshot_json,created_by,created_at) VALUES(?,?,'Edição administrativa','{}',?,?)`, item.ID, version, nullableUserID(userID), now)
		return err
	})
}

func (s *Store) ToggleAdvancedMenuModel(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "UPDATE menu_templates SET active=CASE active WHEN 1 THEN 0 ELSE 1 END,updated_at=? WHERE id=? AND deleted_at IS NULL", nowString(), id)
	return err
}

func (s *Store) DuplicateMenuModel(ctx context.Context, id int64, userID int64) (int64, error) {
	var newID int64
	err := withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		result, err := tx.ExecContext(ctx, `INSERT INTO menu_templates(slug,name,description,menu_type,image_url,active,current_version,source_name,source_updated_month,created_at,updated_at) SELECT slug||'-copia-'||strftime('%s','now'),name||' — cópia',description,menu_type,image_url,1,1,source_name,source_updated_month,?,? FROM menu_templates WHERE id=?`, now, now, id)
		if err != nil {
			return err
		}
		newID, err = result.LastInsertId()
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO menu_template_sections(menu_template_id,menu_section_id,display_name,sort_order,required,selection_min,selection_max,allow_event_changes,notes,active,created_at,updated_at) SELECT ?,menu_section_id,display_name,sort_order,required,selection_min,selection_max,allow_event_changes,notes,active,?,? FROM menu_template_sections WHERE menu_template_id=? AND deleted_at IS NULL`, newID, now, now, id)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO menu_template_items(menu_template_section_id,menu_item_id,slug,source_label,normalized_name,description,sort_order,included,optional,configurable,notes,active,created_at,updated_at) SELECT ns.id,oi.menu_item_id,oi.slug,oi.source_label,oi.normalized_name,oi.description,oi.sort_order,oi.included,oi.optional,oi.configurable,oi.notes,oi.active,?,? FROM menu_template_items oi JOIN menu_template_sections os ON os.id=oi.menu_template_section_id JOIN menu_template_sections ns ON ns.menu_template_id=? AND ns.menu_section_id=os.menu_section_id WHERE os.menu_template_id=? AND oi.deleted_at IS NULL`, now, now, newID, id)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO menu_choice_groups(menu_template_section_id,slug,choice_group_name,selection_min,selection_max,selection_required,allow_extra_items,allow_custom_item,configurable,sort_order,active,created_at,updated_at) SELECT ns.id,og.slug,og.choice_group_name,og.selection_min,og.selection_max,og.selection_required,og.allow_extra_items,og.allow_custom_item,og.configurable,og.sort_order,og.active,?,? FROM menu_choice_groups og JOIN menu_template_sections os ON os.id=og.menu_template_section_id JOIN menu_template_sections ns ON ns.menu_template_id=? AND ns.menu_section_id=os.menu_section_id WHERE os.menu_template_id=? AND og.deleted_at IS NULL`, now, now, newID, id)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO menu_choice_group_items(menu_choice_group_id,menu_template_item_id,sort_order,initially_selected,created_at) SELECT new_group.id,new_item.id,choice.sort_order,choice.initially_selected,? FROM menu_choice_group_items choice JOIN menu_choice_groups old_group ON old_group.id=choice.menu_choice_group_id JOIN menu_template_sections old_section ON old_section.id=old_group.menu_template_section_id JOIN menu_template_items old_item ON old_item.id=choice.menu_template_item_id JOIN menu_template_sections new_section ON new_section.menu_template_id=? AND new_section.menu_section_id=old_section.menu_section_id JOIN menu_choice_groups new_group ON new_group.menu_template_section_id=new_section.id AND new_group.slug=old_group.slug JOIN menu_template_items new_item ON new_item.menu_template_section_id=new_section.id AND new_item.slug=old_item.slug WHERE old_section.menu_template_id=?`, now, newID, id)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO menu_template_item_inventory_links(menu_template_item_id,inventory_item_id,quantity_formula,ownership,notes,active,created_at)
			SELECT new_item.id,link.inventory_item_id,link.quantity_formula,link.ownership,link.notes,link.active,?
			FROM menu_template_item_inventory_links link
			JOIN menu_template_items old_item ON old_item.id=link.menu_template_item_id
			JOIN menu_template_sections old_section ON old_section.id=old_item.menu_template_section_id
			JOIN menu_template_sections new_section ON new_section.menu_template_id=? AND new_section.menu_section_id=old_section.menu_section_id
			JOIN menu_template_items new_item ON new_item.menu_template_section_id=new_section.id AND new_item.slug=old_item.slug
			WHERE old_section.menu_template_id=?`, now, newID, id)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO menu_template_shared_blocks(menu_template_id,menu_shared_block_id,sort_order,created_at) SELECT ?,menu_shared_block_id,sort_order,? FROM menu_template_shared_blocks WHERE menu_template_id=?`, newID, now, id)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO menu_template_versions(menu_template_id,version,change_summary,snapshot_json,created_by,created_at) VALUES(?,1,'Modelo duplicado','{}',?,?)`, newID, nullableUserID(userID), now)
		return err
	})
	return newID, err
}

func (s *Store) ListServiceModels(ctx context.Context, includeInactive bool) ([]models.ServiceModel, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT service.id,service.slug,service.name,service.description,service.category,
		service.duration_minutes,service.billing_unit,service.price_cents,service.cost_cents,service.commission_cents,
		service.supplier,service.configuration_json,service.active,service.current_version,
		service.source_name,service.source_updated_month,
		(SELECT COUNT(*) FROM service_template_components component WHERE component.service_template_id=service.id AND component.active=1 AND component.deleted_at IS NULL),
		service.created_at,service.updated_at
		FROM service_templates service
		WHERE service.deleted_at IS NULL AND (?=1 OR service.active=1)
		ORDER BY service.active DESC,service.name`, includeInactive)
	if err != nil {
		return nil, fmt.Errorf("list service models: %w", err)
	}
	defer rows.Close()
	var result []models.ServiceModel
	for rows.Next() {
		var item models.ServiceModel
		var active int
		var createdAt, updatedAt string
		if err := rows.Scan(&item.ID, &item.Slug, &item.Name, &item.Description, &item.Category,
			&item.DurationMinutes, &item.BillingUnit, &item.PriceCents, &item.CostCents, &item.CommissionCents,
			&item.Supplier, &item.ConfigurationJSON, &active, &item.CurrentVersion,
			&item.SourceName, &item.SourceUpdatedMonth, &item.ComponentCount, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		item.Active = active == 1
		item.CreatedAt, item.UpdatedAt = parseTime(createdAt), parseTime(updatedAt)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) GetServiceModel(ctx context.Context, id int64) (models.ServiceModel, error) {
	items, err := s.ListServiceModels(ctx, true)
	if err != nil {
		return models.ServiceModel{}, err
	}
	for _, item := range items {
		if item.ID == id {
			return item, nil
		}
	}
	return models.ServiceModel{}, sql.ErrNoRows
}

func (s *Store) ServiceModelComponents(ctx context.Context, id int64) ([]models.ServiceComponent, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT link.id,component.id,component.slug,component.name,component.description,component.category,component.source_label,component.normalized_name,component.search_aliases,component.configurable,component.active,link.included,link.optional,link.configuration_json,link.notes FROM service_template_components link JOIN service_components component ON component.id=link.service_component_id WHERE link.service_template_id=? AND link.active=1 AND link.deleted_at IS NULL ORDER BY link.sort_order,link.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.ServiceComponent
	for rows.Next() {
		var item models.ServiceComponent
		var configurable, active, included, optional int
		if err := rows.Scan(&item.TemplateComponentID, &item.ID, &item.Slug, &item.Name, &item.Description, &item.Category, &item.SourceLabel, &item.NormalizedName, &item.SearchAliases, &configurable, &active, &included, &optional, &item.ConfigurationJSON, &item.Notes); err != nil {
			return nil, err
		}
		item.Configurable, item.Active, item.Included, item.Optional = configurable == 1, active == 1, included == 1, optional == 1
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) UpdateServiceModel(ctx context.Context, item models.ServiceModel, userID int64) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		result, err := tx.ExecContext(ctx, `UPDATE service_templates SET name=?,description=?,category=?,duration_minutes=?,billing_unit=?,supplier=?,active=?,current_version=current_version+1,updated_at=? WHERE id=? AND deleted_at IS NULL`, item.Name, item.Description, item.Category, nullInt64(item.DurationMinutes), item.BillingUnit, item.Supplier, item.Active, now, item.ID)
		if err != nil {
			return err
		}
		if count, _ := result.RowsAffected(); count == 0 {
			return sql.ErrNoRows
		}
		var version int
		if err := tx.QueryRowContext(ctx, "SELECT current_version FROM service_templates WHERE id=?", item.ID).Scan(&version); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO service_template_versions(service_template_id,version,change_summary,snapshot_json,created_by,created_at) VALUES(?,?,'Edição administrativa','{}',?,?)`, item.ID, version, nullableUserID(userID), now)
		return err
	})
}

func (s *Store) ToggleServiceModel(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "UPDATE service_templates SET active=CASE active WHEN 1 THEN 0 ELSE 1 END,updated_at=? WHERE id=? AND deleted_at IS NULL", nowString(), id)
	return err
}

func (s *Store) DuplicateServiceModel(ctx context.Context, id, userID int64) (int64, error) {
	var newID int64
	err := withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		result, err := tx.ExecContext(ctx, `INSERT INTO service_templates(slug,name,description,category,duration_minutes,billing_unit,price_cents,cost_cents,commission_cents,supplier,required_team_json,included_materials,excluded_materials,image_url,notes,terms,configuration_json,active,current_version,source_name,source_updated_month,created_at,updated_at) SELECT slug||'-copia-'||strftime('%s','now'),name||' — cópia',description,category,duration_minutes,billing_unit,price_cents,cost_cents,commission_cents,supplier,required_team_json,included_materials,excluded_materials,image_url,notes,terms,configuration_json,1,1,source_name,source_updated_month,?,? FROM service_templates WHERE id=?`, now, now, id)
		if err != nil {
			return err
		}
		newID, err = result.LastInsertId()
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO service_template_components(service_template_id,service_component_id,sort_order,included,optional,configuration_json,notes,active,created_at,updated_at) SELECT ?,service_component_id,sort_order,included,optional,configuration_json,notes,active,?,? FROM service_template_components WHERE service_template_id=? AND deleted_at IS NULL`, newID, now, now, id)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO service_component_inventory_links(service_template_component_id,inventory_item_id,quantity_formula,ownership,supplier,pickup_notes,return_notes,active,created_at)
			SELECT new_component.id,inventory.inventory_item_id,inventory.quantity_formula,inventory.ownership,inventory.supplier,inventory.pickup_notes,inventory.return_notes,inventory.active,?
			FROM service_component_inventory_links inventory
			JOIN service_template_components old_component ON old_component.id=inventory.service_template_component_id
			JOIN service_template_components new_component ON new_component.service_template_id=? AND new_component.service_component_id=old_component.service_component_id
			WHERE old_component.service_template_id=?`, now, newID, id)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO service_choice_groups(service_template_id,slug,name,selection_min,selection_max,selection_required,configurable,created_at,updated_at)
			SELECT ?,slug,name,selection_min,selection_max,selection_required,configurable,?,? FROM service_choice_groups WHERE service_template_id=?`, newID, now, now, id)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO service_choice_group_components(service_choice_group_id,service_template_component_id,sort_order,created_at)
			SELECT new_group.id,new_component.id,membership.sort_order,?
			FROM service_choice_group_components membership
			JOIN service_choice_groups old_group ON old_group.id=membership.service_choice_group_id
			JOIN service_template_components old_component ON old_component.id=membership.service_template_component_id
			JOIN service_choice_groups new_group ON new_group.service_template_id=? AND new_group.slug=old_group.slug
			JOIN service_template_components new_component ON new_component.service_template_id=? AND new_component.service_component_id=old_component.service_component_id
			WHERE old_group.service_template_id=?`, now, newID, newID, id)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO service_template_versions(service_template_id,version,change_summary,snapshot_json,created_by,created_at) VALUES(?,1,'Modelo duplicado','{}',?,?)`, newID, nullableUserID(userID), now)
		return err
	})
	return newID, err
}

func (s *Store) CreateMenuModel(ctx context.Context, userID int64) (int64, error) {
	var modelID int64
	err := withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		slug := fmt.Sprintf("novo-cardapio-%d", time.Now().UnixNano())
		result, err := tx.ExecContext(ctx, `INSERT INTO menu_templates(slug,name,description,menu_type,active,current_version,source_name,source_updated_month,created_at,updated_at) VALUES(?,'Novo modelo de cardápio','Personalize as seções e os itens deste modelo.','buffet',1,1,'Cadastro administrativo','',?,?)`, slug, now, now)
		if err != nil {
			return err
		}
		modelID, err = result.LastInsertId()
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO menu_template_sections(menu_template_id,menu_section_id,display_name,sort_order,required,selection_min,allow_event_changes,active,created_at,updated_at)
			SELECT ?,id,name,CASE slug WHEN 'entradas' THEN 10 WHEN 'buffet-principal' THEN 20 WHEN 'acompanhamentos' THEN 30 ELSE 40 END,0,0,1,1,?,?
			FROM menu_sections WHERE slug IN ('entradas','buffet-principal','acompanhamentos','bebidas')`, modelID, now, now)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO menu_template_versions(menu_template_id,version,change_summary,snapshot_json,created_by,created_at) VALUES(?,1,'Modelo criado','{}',?,?)`, modelID, nullableUserID(userID), now)
		return err
	})
	return modelID, err
}

func (s *Store) CreateServiceModel(ctx context.Context, userID int64) (int64, error) {
	var modelID int64
	err := withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		slug := fmt.Sprintf("novo-servico-%d", time.Now().UnixNano())
		result, err := tx.ExecContext(ctx, `INSERT INTO service_templates(slug,name,description,category,billing_unit,configuration_json,active,current_version,source_name,source_updated_month,created_at,updated_at) VALUES(?,'Novo modelo de serviço','Cadastre os componentes incluídos neste serviço.','geral','serviço','{}',1,1,'Cadastro administrativo','',?,?)`, slug, now, now)
		if err != nil {
			return err
		}
		modelID, err = result.LastInsertId()
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO service_template_versions(service_template_id,version,change_summary,snapshot_json,created_by,created_at) VALUES(?,1,'Modelo criado','{}',?,?)`, modelID, nullableUserID(userID), now)
		return err
	})
	return modelID, err
}

func (s *Store) AddMenuModelItem(ctx context.Context, modelID, sectionID int64, name, description string, included, configurable bool, userID int64) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("informe o nome do item")
	}
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		var sectionSlug, sectionName string
		if err := tx.QueryRowContext(ctx, `SELECT definition.slug,definition.name FROM menu_template_sections section JOIN menu_sections definition ON definition.id=section.menu_section_id WHERE section.id=? AND section.menu_template_id=? AND section.deleted_at IS NULL`, sectionID, modelID).Scan(&sectionSlug, &sectionName); err != nil {
			return err
		}
		categoryName := sectionName
		switch sectionSlug {
		case "entradas", "salgados", "fritos", "assados":
			categoryName = "Entradas"
		case "buffet-principal", "massas", "molhos":
			categoryName = "Pratos principais"
		case "acompanhamentos":
			categoryName = "Acompanhamentos"
		case "bebidas":
			categoryName = "Bebidas"
		}
		var categoryID int64
		if err := tx.QueryRowContext(ctx, "SELECT id FROM menu_categories WHERE name=?", categoryName).Scan(&categoryID); err != nil {
			if err := tx.QueryRowContext(ctx, "SELECT id FROM menu_categories WHERE active=1 ORDER BY sort_order,id LIMIT 1").Scan(&categoryID); err != nil {
				return err
			}
		}
		now := nowString()
		unique := time.Now().UnixNano()
		storageName := fmt.Sprintf("__model_%d_%d", modelID, unique)
		slug := fmt.Sprintf("model-item-%d", unique)
		menuResult, err := tx.ExecContext(ctx, `INSERT INTO menu_items(category_id,name,display_name,description,slug,source_label,normalized_name,active,created_at,updated_at) VALUES(?,?,?,?,?,?,?,1,?,?)`, categoryID, storageName, name, description, slug, name, name, now, now)
		if err != nil {
			return err
		}
		menuItemID, err := menuResult.LastInsertId()
		if err != nil {
			return err
		}
		var sortOrder int
		if err := tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(sort_order),0)+10 FROM menu_template_items WHERE menu_template_section_id=?", sectionID).Scan(&sortOrder); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO menu_template_items(menu_template_section_id,menu_item_id,slug,source_label,normalized_name,description,sort_order,included,optional,configurable,active,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,1,?,?)`, sectionID, menuItemID, slug, name, name, description, sortOrder, included, !included, configurable, now, now)
		if err != nil {
			return err
		}
		return bumpMenuModelVersion(ctx, tx, modelID, userID, "Item adicionado: "+name)
	})
}

func (s *Store) UpdateMenuModelItem(ctx context.Context, modelID, itemID int64, name, description string, included, configurable bool, userID int64) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("informe o nome do item")
	}
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		var menuItemID sql.NullInt64
		if err := tx.QueryRowContext(ctx, `SELECT item.menu_item_id FROM menu_template_items item JOIN menu_template_sections section ON section.id=item.menu_template_section_id WHERE item.id=? AND section.menu_template_id=? AND item.deleted_at IS NULL`, itemID, modelID).Scan(&menuItemID); err != nil {
			return err
		}
		now := nowString()
		if _, err := tx.ExecContext(ctx, `UPDATE menu_template_items SET source_label=?,normalized_name=?,description=?,included=?,optional=?,configurable=?,updated_at=? WHERE id=?`, name, name, description, included, !included, configurable, now, itemID); err != nil {
			return err
		}
		if menuItemID.Valid {
			if _, err := tx.ExecContext(ctx, `UPDATE menu_items SET display_name=CASE WHEN display_name='' THEN display_name ELSE ? END,description=?,source_label=?,normalized_name=?,updated_at=? WHERE id=?`, name, description, name, name, now, menuItemID.Int64); err != nil {
				return err
			}
		}
		return bumpMenuModelVersion(ctx, tx, modelID, userID, "Item atualizado: "+name)
	})
}

func (s *Store) RemoveMenuModelItem(ctx context.Context, modelID, itemID, userID int64) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		result, err := tx.ExecContext(ctx, `UPDATE menu_template_items SET active=0,deleted_at=?,updated_at=? WHERE id=? AND menu_template_section_id IN (SELECT id FROM menu_template_sections WHERE menu_template_id=?) AND deleted_at IS NULL`, now, now, itemID, modelID)
		if err != nil {
			return err
		}
		if count, _ := result.RowsAffected(); count == 0 {
			return sql.ErrNoRows
		}
		return bumpMenuModelVersion(ctx, tx, modelID, userID, "Item removido")
	})
}

func (s *Store) UpdateMenuChoiceGroup(ctx context.Context, modelID, groupID int64, min int, max sql.NullInt64, userID int64) error {
	if min < 0 || (max.Valid && max.Int64 < int64(min)) {
		return fmt.Errorf("limites de escolha inválidos")
	}
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE menu_choice_groups SET selection_min=?,selection_max=?,selection_required=?,updated_at=? WHERE id=? AND menu_template_section_id IN (SELECT id FROM menu_template_sections WHERE menu_template_id=?)`, min, nullInt64(max), min > 0, nowString(), groupID, modelID)
		if err != nil {
			return err
		}
		if count, _ := result.RowsAffected(); count == 0 {
			return sql.ErrNoRows
		}
		return bumpMenuModelVersion(ctx, tx, modelID, userID, "Regra de escolha atualizada")
	})
}

func (s *Store) AddMenuModelSection(ctx context.Context, modelID int64, name, sectionType string, userID int64) error {
	name = strings.TrimSpace(name)
	sectionType = strings.TrimSpace(sectionType)
	if name == "" {
		return fmt.Errorf("informe o nome da seção")
	}
	if sectionType == "" {
		sectionType = "food"
	}
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		slug := fmt.Sprintf("admin-section-%d", time.Now().UnixNano())
		result, err := tx.ExecContext(ctx, `INSERT INTO menu_sections(slug,name,section_type,active,created_at,updated_at) VALUES(?,?,?,1,?,?)`, slug, name, sectionType, now, now)
		if err != nil {
			return err
		}
		definitionID, err := result.LastInsertId()
		if err != nil {
			return err
		}
		var sortOrder int
		if err := tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(sort_order),0)+10 FROM menu_template_sections WHERE menu_template_id=?", modelID).Scan(&sortOrder); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO menu_template_sections(menu_template_id,menu_section_id,display_name,sort_order,required,selection_min,allow_event_changes,active,created_at,updated_at) VALUES(?,?,?,?,0,0,1,1,?,?)`, modelID, definitionID, name, sortOrder, now, now); err != nil {
			return err
		}
		return bumpMenuModelVersion(ctx, tx, modelID, userID, "Seção adicionada: "+name)
	})
}

func (s *Store) UpdateMenuModelSection(ctx context.Context, modelID, sectionID int64, name string, sortOrder, selectionMin int, selectionMax sql.NullInt64, required, allowChanges bool, userID int64) error {
	name = strings.TrimSpace(name)
	if name == "" || sortOrder < 0 || selectionMin < 0 || (selectionMax.Valid && selectionMax.Int64 < int64(selectionMin)) {
		return fmt.Errorf("configuração da seção inválida")
	}
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE menu_template_sections SET display_name=?,sort_order=?,required=?,selection_min=?,selection_max=?,allow_event_changes=?,updated_at=? WHERE id=? AND menu_template_id=? AND deleted_at IS NULL`, name, sortOrder, required, selectionMin, nullInt64(selectionMax), allowChanges, nowString(), sectionID, modelID)
		if err != nil {
			return err
		}
		if count, _ := result.RowsAffected(); count == 0 {
			return sql.ErrNoRows
		}
		return bumpMenuModelVersion(ctx, tx, modelID, userID, "Seção atualizada: "+name)
	})
}

func (s *Store) RemoveMenuModelSection(ctx context.Context, modelID, sectionID, userID int64) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		result, err := tx.ExecContext(ctx, `UPDATE menu_template_sections SET active=0,deleted_at=?,updated_at=? WHERE id=? AND menu_template_id=? AND deleted_at IS NULL`, now, now, sectionID, modelID)
		if err != nil {
			return err
		}
		if count, _ := result.RowsAffected(); count == 0 {
			return sql.ErrNoRows
		}
		return bumpMenuModelVersion(ctx, tx, modelID, userID, "Seção removida")
	})
}

func (s *Store) AddServiceModelComponent(ctx context.Context, modelID int64, name, description, category string, included, configurable bool, userID int64) error {
	name = strings.TrimSpace(name)
	category = strings.TrimSpace(category)
	if name == "" {
		return fmt.Errorf("informe o nome do componente")
	}
	if category == "" {
		category = "geral"
	}
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		var exists int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM service_templates WHERE id=? AND deleted_at IS NULL", modelID).Scan(&exists); err != nil || exists == 0 {
			if err != nil {
				return err
			}
			return sql.ErrNoRows
		}
		now := nowString()
		slug := fmt.Sprintf("admin-component-%d", time.Now().UnixNano())
		result, err := tx.ExecContext(ctx, `INSERT INTO service_components(slug,name,description,category,source_label,normalized_name,configurable,active,created_at,updated_at) VALUES(?,?,?,?,?,?,?,1,?,?)`, slug, name, description, category, name, name, configurable, now, now)
		if err != nil {
			return err
		}
		componentID, err := result.LastInsertId()
		if err != nil {
			return err
		}
		var sortOrder int
		if err := tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(sort_order),0)+10 FROM service_template_components WHERE service_template_id=?", modelID).Scan(&sortOrder); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO service_template_components(service_template_id,service_component_id,sort_order,included,optional,configuration_json,active,created_at,updated_at) VALUES(?,?,?,?,?,'{}',1,?,?)`, modelID, componentID, sortOrder, included, !included, now, now)
		if err != nil {
			return err
		}
		return bumpServiceModelVersion(ctx, tx, modelID, userID, "Componente adicionado: "+name)
	})
}

func (s *Store) UpdateServiceModelComponent(ctx context.Context, modelID, templateComponentID int64, name, description, category, configuration string, included, configurable bool, userID int64) error {
	name = strings.TrimSpace(name)
	category = strings.TrimSpace(category)
	if name == "" {
		return fmt.Errorf("informe o nome do componente")
	}
	if category == "" {
		category = "geral"
	}
	if strings.TrimSpace(configuration) == "" {
		configuration = "{}"
	}
	if !json.Valid([]byte(configuration)) {
		return fmt.Errorf("a configuração JSON é inválida")
	}
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		var currentID int64
		if err := tx.QueryRowContext(ctx, "SELECT service_component_id FROM service_template_components WHERE id=? AND service_template_id=? AND deleted_at IS NULL", templateComponentID, modelID).Scan(&currentID); err != nil {
			return err
		}
		now := nowString()
		slug := fmt.Sprintf("admin-component-%d", time.Now().UnixNano())
		result, err := tx.ExecContext(ctx, `INSERT INTO service_components(slug,name,description,category,source_label,normalized_name,configurable,active,created_at,updated_at) VALUES(?,?,?,?,?,?,?,1,?,?)`, slug, name, description, category, name, name, configurable, now, now)
		if err != nil {
			return err
		}
		newComponentID, err := result.LastInsertId()
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE service_template_components SET service_component_id=?,included=?,optional=?,configuration_json=?,updated_at=? WHERE id=?`, newComponentID, included, !included, configuration, now, templateComponentID); err != nil {
			return err
		}
		return bumpServiceModelVersion(ctx, tx, modelID, userID, "Componente atualizado: "+name)
	})
}

func (s *Store) RemoveServiceModelComponent(ctx context.Context, modelID, templateComponentID, userID int64) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		result, err := tx.ExecContext(ctx, `UPDATE service_template_components SET active=0,deleted_at=?,updated_at=? WHERE id=? AND service_template_id=? AND deleted_at IS NULL`, now, now, templateComponentID, modelID)
		if err != nil {
			return err
		}
		if count, _ := result.RowsAffected(); count == 0 {
			return sql.ErrNoRows
		}
		return bumpServiceModelVersion(ctx, tx, modelID, userID, "Componente removido")
	})
}
