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

func (s *Store) ListMenuItemIngredients(ctx context.Context, menuItemID int64) ([]models.MenuItemIngredient, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT recipe.id,recipe.menu_item_id,recipe.inventory_item_id,item.name,item.unit,
		recipe.calculation_type,recipe.quantity,recipe.people_divisor,recipe.notes,recipe.active
		FROM menu_item_ingredients recipe
		JOIN inventory_items item ON item.id=recipe.inventory_item_id
		WHERE recipe.menu_item_id=? AND recipe.active=1
		ORDER BY item.name`, menuItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.MenuItemIngredient
	for rows.Next() {
		var item models.MenuItemIngredient
		var active int
		if err := rows.Scan(&item.ID, &item.MenuItemID, &item.InventoryItemID, &item.InventoryItemName, &item.Unit, &item.CalculationType, &item.Quantity, &item.PeopleDivisor, &item.Notes, &active); err != nil {
			return nil, err
		}
		item.Active = active == 1
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) ListRecipeIngredientOptions(ctx context.Context) ([]models.InventoryItem, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT item.id,item.name,category.name,item.unit,item.active
		FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id
		WHERE item.active=1 AND item.item_kind='consumable'
		ORDER BY category.sort_order,item.name`)
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

func validateRecipeIngredient(item models.MenuItemIngredient) error {
	if item.MenuItemID <= 0 || item.Quantity <= 0 || item.PeopleDivisor <= 0 {
		return fmt.Errorf("invalid recipe ingredient")
	}
	switch item.CalculationType {
	case "proportional", "group_of_people", "fixed":
		return nil
	default:
		return fmt.Errorf("invalid recipe calculation type")
	}
}

func (s *Store) AddMenuItemIngredient(ctx context.Context, item models.MenuItemIngredient) error {
	if item.InventoryItemID <= 0 {
		return fmt.Errorf("invalid recipe ingredient")
	}
	if err := validateRecipeIngredient(item); err != nil {
		return err
	}
	now := nowString()
	_, err := s.db.ExecContext(ctx, `INSERT INTO menu_item_ingredients(menu_item_id,inventory_item_id,calculation_type,quantity,people_divisor,notes,active,created_at,updated_at)
		VALUES(?,?,?,?,?,?,1,?,?)
		ON CONFLICT(menu_item_id,inventory_item_id) DO UPDATE SET calculation_type=excluded.calculation_type,quantity=excluded.quantity,people_divisor=excluded.people_divisor,notes=excluded.notes,active=1,updated_at=excluded.updated_at`,
		item.MenuItemID, item.InventoryItemID, item.CalculationType, item.Quantity, item.PeopleDivisor, strings.TrimSpace(item.Notes), now, now)
	return err
}

func (s *Store) UpdateMenuItemIngredient(ctx context.Context, item models.MenuItemIngredient) error {
	if item.ID <= 0 {
		return fmt.Errorf("invalid recipe ingredient")
	}
	if err := validateRecipeIngredient(item); err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE menu_item_ingredients SET calculation_type=?,quantity=?,people_divisor=?,notes=?,updated_at=?
		WHERE id=? AND menu_item_id=? AND active=1`, item.CalculationType, item.Quantity, item.PeopleDivisor, strings.TrimSpace(item.Notes), nowString(), item.ID, item.MenuItemID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) RemoveMenuItemIngredient(ctx context.Context, menuItemID, ingredientID int64) error {
	result, err := s.db.ExecContext(ctx, `UPDATE menu_item_ingredients SET active=0,updated_at=? WHERE id=? AND menu_item_id=? AND active=1`, nowString(), ingredientID, menuItemID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) EventMenuRecipeRequirements(ctx context.Context, eventID int64) ([]models.AutomaticRequirement, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT recipe.inventory_item_id,recipe.calculation_type,recipe.quantity,recipe.people_divisor,
		event_item.portions,COALESCE(NULLIF(menu.display_name,''),menu.name)
		FROM event_menu_items event_item
		JOIN menu_items menu ON menu.id=event_item.menu_item_id
		JOIN menu_item_ingredients recipe ON recipe.menu_item_id=menu.id AND recipe.active=1
		JOIN inventory_items ingredient ON ingredient.id=recipe.inventory_item_id AND ingredient.active=1
		WHERE event_item.event_id=?
		ORDER BY recipe.inventory_item_id,menu.name`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type aggregate struct {
		quantity float64
		dishes   []string
	}
	values := map[int64]aggregate{}
	for rows.Next() {
		var inventoryID int64
		var calculationType, dish string
		var quantity, divisor float64
		var portions int
		if err := rows.Scan(&inventoryID, &calculationType, &quantity, &divisor, &portions, &dish); err != nil {
			return nil, err
		}
		calculated := quantity
		switch calculationType {
		case "proportional":
			calculated = math.Ceil((float64(portions)/divisor*quantity)*100-1e-9) / 100
		case "group_of_people":
			calculated = math.Ceil(float64(portions)/divisor) * quantity
		}
		value := values[inventoryID]
		if calculationType == "fixed" {
			value.quantity = math.Max(value.quantity, calculated)
		} else {
			value.quantity += calculated
		}
		seen := false
		for _, existing := range value.dishes {
			if existing == dish {
				seen = true
				break
			}
		}
		if !seen {
			value.dishes = append(value.dishes, dish)
		}
		values[inventoryID] = value
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(values))
	for inventoryID := range values {
		ids = append(ids, inventoryID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	result := make([]models.AutomaticRequirement, 0, len(values))
	for _, inventoryID := range ids {
		value := values[inventoryID]
		result = append(result, models.AutomaticRequirement{
			SourceKey:       fmt.Sprintf("menu-recipe:%d", inventoryID),
			InventoryItemID: inventoryID,
			Quantity:        value.quantity,
			Origin:          "Receita dos pratos: " + strings.Join(value.dishes, ", "),
		})
	}
	return result, nil
}
