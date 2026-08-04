package repositories

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"buffetflow/internal/database"
	"buffetflow/internal/models"
)

func TestMenuItemCanBeCreatedUpdatedAndDeactivated(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "catalog-test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	store := New(db)

	var categoryID, inventoryID int64
	if err := db.QueryRowContext(ctx, "SELECT id FROM menu_categories WHERE name='Bebidas'").Scan(&categoryID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, "SELECT id FROM inventory_items WHERE internal_code='BEB-001'").Scan(&inventoryID); err != nil {
		t.Fatal(err)
	}
	item := models.MenuItem{
		CategoryID:            categoryID,
		Name:                  "Refrigerante teste",
		ResultInventoryItemID: sql.NullInt64{Int64: inventoryID, Valid: true},
		CalculationType:       "category_distribution",
		CalculationGroup:      "soda-test",
		CalculationDivisor:    2,
		CalculationMultiplier: 1,
		CalculationWeight:     70,
		Active:                true,
	}
	if err := store.SaveMenuItem(ctx, &item, nil); err != nil {
		t.Fatal(err)
	}
	created, err := store.GetMenuItem(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if created.Name != item.Name || created.CalculationWeight != 70 || created.ResultInventoryItemID.Int64 != inventoryID {
		t.Fatalf("created item did not preserve calculation fields: %+v", created)
	}

	item.Name = "Refrigerante atualizado"
	item.CalculationWeight = 55
	if err := store.SaveMenuItem(ctx, &item, nil); err != nil {
		t.Fatal(err)
	}
	updated, err := store.GetMenuItem(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != item.Name || updated.CalculationWeight != 55 {
		t.Fatalf("updated item did not preserve changes: %+v", updated)
	}
	if err := store.ToggleMenuItem(ctx, item.ID); err != nil {
		t.Fatal(err)
	}
	deactivated, err := store.GetMenuItem(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if deactivated.Active {
		t.Fatal("item should be inactive after removal")
	}
}

func TestMenuTemplateCanBeCreatedUpdatedAndDeactivated(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "menu-template-test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	store := New(db)
	items, err := store.ListMenuItems(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) < 2 {
		t.Fatal("expected seeded menu items")
	}

	template := models.MenuTemplate{
		Name:             "Cardápio de teste",
		Description:      "Modelo reutilizável",
		HasWelcomeDrinks: true,
		HasCoffeeTable:   true,
		Active:           true,
	}
	if err := store.SaveMenuTemplate(ctx, &template, []int64{items[0].ID, items[1].ID}); err != nil {
		t.Fatal(err)
	}
	created, err := store.GetMenuTemplate(ctx, template.ID)
	if err != nil {
		t.Fatal(err)
	}
	if created.ItemCount != 2 || !created.HasWelcomeDrinks || !created.HasCoffeeTable {
		t.Fatalf("created template did not preserve its configuration: %+v", created)
	}
	templateItems, err := store.ListTemplateMenuItems(ctx, template.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(templateItems) != 2 || !templateItems[0].TemplateOwnerID.Valid || templateItems[0].TemplateOwnerID.Int64 != template.ID {
		t.Fatalf("template items were not created as independent copies: %+v", templateItems)
	}
	originalSource, err := store.GetMenuItem(ctx, templateItems[0].SourceMenuItemID.Int64)
	if err != nil {
		t.Fatal(err)
	}
	templateItems[0].Name = "Item personalizado somente neste cardápio"
	if err := store.SaveTemplateMenuItem(ctx, template.ID, &templateItems[0], templateItems[0].Equipment); err != nil {
		t.Fatal(err)
	}
	unchangedSource, err := store.GetMenuItem(ctx, originalSource.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unchangedSource.Name != originalSource.Name {
		t.Fatalf("editing template item changed global catalog item from %q to %q", originalSource.Name, unchangedSource.Name)
	}
	if err := store.ToggleTemplateMenuItem(ctx, template.ID, templateItems[1].ID); err != nil {
		t.Fatal(err)
	}

	template.Name = "Cardápio atualizado"
	template.HasCoffeeTable = false
	if err := store.SaveMenuTemplate(ctx, &template, nil); err != nil {
		t.Fatal(err)
	}
	updated, err := store.GetMenuTemplate(ctx, template.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != template.Name || updated.ItemCount != 1 || updated.HasCoffeeTable {
		t.Fatalf("updated template did not preserve changes: %+v", updated)
	}
	if err := store.ToggleMenuTemplate(ctx, template.ID); err != nil {
		t.Fatal(err)
	}
	deactivated, err := store.GetMenuTemplate(ctx, template.ID)
	if err != nil {
		t.Fatal(err)
	}
	if deactivated.Active {
		t.Fatal("template should be inactive after removal")
	}
}
