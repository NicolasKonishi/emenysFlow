package repositories

import (
	"math"
	"testing"

	"buffetflow/internal/models"
)

func TestMenuRecipesCalculateIngredientsFromSelectedDishes(t *testing.T) {
	store, db, ctx := newModelWorkflowStore(t)
	var ribsID, stroganoffID int64
	if err := db.QueryRowContext(ctx, "SELECT id FROM menu_items WHERE name='Costelinha ao molho barbecue' LIMIT 1").Scan(&ribsID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, "SELECT id FROM menu_items WHERE name='Estrogonofe de frango' LIMIT 1").Scan(&stroganoffID); err != nil {
		t.Fatal(err)
	}
	selection := []models.EventMenuItem{
		{MenuItemID: ribsID, Selected: true, Portions: 100},
		{MenuItemID: stroganoffID, Selected: true, Portions: 80},
	}
	if err := store.SaveEventMenu(ctx, 1, selection); err != nil {
		t.Fatal(err)
	}
	requirements, err := store.EventMenuRecipeRequirements(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	quantities := map[string]float64{}
	for _, requirement := range requirements {
		var name string
		if err := db.QueryRowContext(ctx, "SELECT name FROM inventory_items WHERE id=?", requirement.InventoryItemID).Scan(&name); err != nil {
			t.Fatal(err)
		}
		quantities[name] = requirement.Quantity
	}
	want := map[string]float64{
		"Costela suína":  33.34,
		"Molho barbecue": 1,
		"Frango":         40,
		"Ketchup":        1,
		"Mostarda":       1,
	}
	for name, quantity := range want {
		if math.Abs(quantities[name]-quantity) > 0.001 {
			t.Errorf("ingredient %s got %.2f, want %.2f", name, quantities[name], quantity)
		}
	}
}

func TestMenuRecipeIngredientCRUDAndClone(t *testing.T) {
	store, db, ctx := newModelWorkflowStore(t)
	var menuItemID, inventoryItemID, templateID int64
	if err := db.QueryRowContext(ctx, "SELECT id FROM menu_items WHERE name='Arroz branco' AND template_owner_id IS NULL LIMIT 1").Scan(&menuItemID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, "SELECT id FROM inventory_items WHERE internal_code='COZ-TMP-SAL'").Scan(&inventoryItemID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, "SELECT id FROM event_templates ORDER BY id LIMIT 1").Scan(&templateID); err != nil {
		t.Fatal(err)
	}
	item := models.MenuItemIngredient{MenuItemID: menuItemID, InventoryItemID: inventoryItemID, CalculationType: "group_of_people", Quantity: 2, PeopleDivisor: 50, Notes: "Teste da receita"}
	if err := store.AddMenuItemIngredient(ctx, item); err != nil {
		t.Fatal(err)
	}
	ingredients, err := store.ListMenuItemIngredients(ctx, menuItemID)
	if err != nil {
		t.Fatal(err)
	}
	var ingredientID int64
	for _, ingredient := range ingredients {
		if ingredient.InventoryItemID == inventoryItemID {
			ingredientID = ingredient.ID
			item.ID = ingredient.ID
		}
	}
	if ingredientID == 0 {
		t.Fatal("added recipe ingredient was not listed")
	}
	item.Quantity, item.PeopleDivisor, item.Notes = 3, 40, "Atualizado"
	if err := store.UpdateMenuItemIngredient(ctx, item); err != nil {
		t.Fatal(err)
	}
	cloneID, err := store.CloneMenuItemToTemplate(ctx, templateID, menuItemID)
	if err != nil {
		t.Fatal(err)
	}
	cloneIngredients, err := store.ListMenuItemIngredients(ctx, cloneID)
	if err != nil {
		t.Fatal(err)
	}
	if len(cloneIngredients) != 1 || cloneIngredients[0].Quantity != 3 || cloneIngredients[0].PeopleDivisor != 40 {
		t.Fatal("cloned menu item did not preserve its recipe")
	}
	if err := store.RemoveMenuItemIngredient(ctx, menuItemID, ingredientID); err != nil {
		t.Fatal(err)
	}
	ingredients, err = store.ListMenuItemIngredients(ctx, menuItemID)
	if err != nil {
		t.Fatal(err)
	}
	for _, ingredient := range ingredients {
		if ingredient.ID == ingredientID {
			t.Fatal("removed ingredient remained active")
		}
	}
}
