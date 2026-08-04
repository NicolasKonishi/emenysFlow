package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"buffetflow/internal/models"
)

type menuTemplateConfiguration struct {
	HasDecoration    bool `json:"has_decoration"`
	HasWelcomeDrinks bool `json:"has_welcome_drinks"`
	HasCoffeeTable   bool `json:"has_coffee_table"`
}

const menuTemplateSelect = `SELECT et.id,et.name,et.description,et.configuration_json,et.active,
	COUNT(linked.id),COALESCE(GROUP_CONCAT(linked.id),'')
	FROM event_templates et
	LEFT JOIN event_template_menu_items etmi ON etmi.template_id=et.id
	LEFT JOIN menu_items linked ON linked.id=etmi.menu_item_id AND linked.active=1`

func scanMenuTemplate(scanner interface{ Scan(...any) error }) (models.MenuTemplate, error) {
	var item models.MenuTemplate
	var configuration, itemIDs string
	var active int
	if err := scanner.Scan(&item.ID, &item.Name, &item.Description, &configuration, &active, &item.ItemCount, &itemIDs); err != nil {
		return item, err
	}
	var config menuTemplateConfiguration
	_ = json.Unmarshal([]byte(configuration), &config)
	item.HasDecoration = config.HasDecoration
	item.HasWelcomeDrinks = config.HasWelcomeDrinks
	item.HasCoffeeTable = config.HasCoffeeTable
	item.Active = active == 1
	item.ItemIDsCSV = itemIDs
	for _, raw := range strings.Split(itemIDs, ",") {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err == nil {
			item.ItemIDs = append(item.ItemIDs, id)
		}
	}
	return item, nil
}

func (s *Store) ListMenuTemplates(ctx context.Context, includeInactive bool) ([]models.MenuTemplate, error) {
	rows, err := s.db.QueryContext(ctx, menuTemplateSelect+`
		WHERE (?=1 OR et.active=1)
		GROUP BY et.id
		ORDER BY et.active DESC,et.name`, includeInactive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.MenuTemplate
	for rows.Next() {
		item, err := scanMenuTemplate(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) GetMenuTemplate(ctx context.Context, id int64) (models.MenuTemplate, error) {
	return scanMenuTemplate(s.db.QueryRowContext(ctx, menuTemplateSelect+`
		WHERE et.id=?
		GROUP BY et.id`, id))
}

func (s *Store) MenuTemplateSelection(ctx context.Context, templateID int64) ([]models.EventMenuItem, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT m.id,COALESCE(NULLIF(m.display_name,''),m.name),c.name,m.template_owner_id
		FROM menu_items m
		JOIN menu_categories c ON c.id=m.category_id
		JOIN event_template_menu_items etmi ON etmi.menu_item_id=m.id AND etmi.template_id=?
		WHERE m.active=1 AND m.template_owner_id=?
		ORDER BY c.sort_order,COALESCE(NULLIF(m.display_name,''),m.name)`, templateID, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.EventMenuItem
	for rows.Next() {
		var item models.EventMenuItem
		if err := rows.Scan(&item.MenuItemID, &item.MenuItemName, &item.CategoryName, &item.TemplateOwnerID); err != nil {
			return nil, err
		}
		item.Selected = true
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) SaveMenuTemplate(ctx context.Context, item *models.MenuTemplate, menuItemIDs []int64) error {
	if strings.TrimSpace(item.Name) == "" {
		return fmt.Errorf("template name is required")
	}
	configuration, err := json.Marshal(menuTemplateConfiguration{
		HasDecoration:    item.HasDecoration,
		HasWelcomeDrinks: item.HasWelcomeDrinks,
		HasCoffeeTable:   item.HasCoffeeTable,
	})
	if err != nil {
		return err
	}
	now := nowString()
	isNew := item.ID == 0
	err = withTx(ctx, s.db, func(tx *sql.Tx) error {
		if isNew {
			result, err := tx.ExecContext(ctx, `INSERT INTO event_templates(name,description,configuration_json,active,created_at,updated_at)
				VALUES(?,?,?,1,?,?)`, item.Name, item.Description, string(configuration), now, now)
			if err != nil {
				return err
			}
			item.ID, err = result.LastInsertId()
			if err != nil {
				return err
			}
		} else {
			result, err := tx.ExecContext(ctx, `UPDATE event_templates SET name=?,description=?,configuration_json=?,updated_at=? WHERE id=?`, item.Name, item.Description, string(configuration), now, item.ID)
			if err != nil {
				return err
			}
			if count, _ := result.RowsAffected(); count == 0 {
				return sql.ErrNoRows
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if isNew {
		seen := map[int64]bool{}
		for _, menuItemID := range menuItemIDs {
			if menuItemID <= 0 || seen[menuItemID] {
				continue
			}
			seen[menuItemID] = true
			if _, err := s.CloneMenuItemToTemplate(ctx, item.ID, menuItemID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) ToggleMenuTemplate(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "UPDATE event_templates SET active=CASE active WHEN 1 THEN 0 ELSE 1 END,updated_at=? WHERE id=?", nowString(), id)
	return err
}

func (s *Store) ListTemplateMenuItems(ctx context.Context, templateID int64, includeInactive bool) ([]models.MenuItem, error) {
	rows, err := s.db.QueryContext(ctx, menuItemSelect+` WHERE m.template_owner_id=? AND (?=1 OR m.active=1) ORDER BY c.sort_order,COALESCE(NULLIF(m.display_name,''),m.name)`, templateID, includeInactive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.MenuItem
	for rows.Next() {
		item, err := scanMenuItem(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) CloneMenuItemToTemplate(ctx context.Context, templateID, sourceItemID int64) (int64, error) {
	source, err := s.GetMenuItem(ctx, sourceItemID)
	if err != nil {
		return 0, err
	}
	if source.TemplateOwnerID.Valid {
		return 0, fmt.Errorf("source item must belong to the global catalog")
	}
	source.ID = 0
	source.TemplateOwnerID = sql.NullInt64{Int64: templateID, Valid: true}
	source.SourceMenuItemID = sql.NullInt64{Int64: sourceItemID, Valid: true}
	source.Active = true
	if err := s.SaveMenuItem(ctx, &source, source.Equipment); err != nil {
		return 0, err
	}
	if _, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO menu_item_ingredients(menu_item_id,inventory_item_id,calculation_type,quantity,people_divisor,notes,active,created_at,updated_at)
		SELECT ?,inventory_item_id,calculation_type,quantity,people_divisor,notes,active,?,?
		FROM menu_item_ingredients WHERE menu_item_id=? AND active=1`, source.ID, nowString(), nowString(), sourceItemID); err != nil {
		return 0, err
	}
	if _, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO event_template_menu_items(template_id,menu_item_id,created_at) VALUES(?,?,?)`, templateID, source.ID, nowString()); err != nil {
		return 0, err
	}
	return source.ID, nil
}

func (s *Store) SaveTemplateMenuItem(ctx context.Context, templateID int64, item *models.MenuItem, equipment []models.EquipmentLink) error {
	if item.ID > 0 {
		var owner sql.NullInt64
		if err := s.db.QueryRowContext(ctx, "SELECT template_owner_id FROM menu_items WHERE id=?", item.ID).Scan(&owner); err != nil {
			return err
		}
		if !owner.Valid || owner.Int64 != templateID {
			return sql.ErrNoRows
		}
	}
	item.TemplateOwnerID = sql.NullInt64{Int64: templateID, Valid: true}
	if err := s.SaveMenuItem(ctx, item, equipment); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO event_template_menu_items(template_id,menu_item_id,created_at) VALUES(?,?,?)`, templateID, item.ID, nowString())
	return err
}

func (s *Store) ToggleTemplateMenuItem(ctx context.Context, templateID, itemID int64) error {
	result, err := s.db.ExecContext(ctx, `UPDATE menu_items SET active=CASE active WHEN 1 THEN 0 ELSE 1 END,updated_at=? WHERE id=? AND template_owner_id=?`, nowString(), itemID, templateID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	return nil
}
