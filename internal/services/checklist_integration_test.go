package services

import (
	"context"
	"database/sql"
	"math"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"buffetflow/internal/database"
	"buffetflow/internal/models"
	"buffetflow/internal/repositories"
)

func testStore(t *testing.T) (*repositories.Store, func()) {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(context.Background(), db); err != nil {
		db.Close()
		t.Fatal(err)
	}
	return repositories.New(db), func() { db.Close() }
}

func TestFullOperationAndReturnUpdatesStock(t *testing.T) {
	store, closeStore := testStore(t)
	defer closeStore()
	service := NewChecklistService(store)
	ctx := context.Background()
	checklist, err := service.Generate(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.ReserveEvent(ctx, 1, 0, true); err != nil {
		t.Fatal(err)
	}
	quantities := map[int64]float64{}
	for _, item := range checklist.Items {
		quantities[item.ID] = item.RequiredQuantity
	}
	for _, stage := range []string{"separating", "checking", "loading", "in_progress"} {
		op := models.EventOperation{EventID: 1, Stage: stage, ResponsibleName: "Equipe teste", OccurredAt: time.Now()}
		if err = store.SaveEventOperation(ctx, 1, op, quantities, 0); err != nil {
			t.Fatalf("stage %s: %v", stage, err)
		}
		updated, loadErr := store.GetChecklistByEvent(ctx, 1)
		if loadErr != nil {
			t.Fatal(loadErr)
		}
		quantities = map[int64]float64{}
		for _, item := range updated.Items {
			quantities[item.ID] = item.RequiredQuantity
		}
	}
	returns, err := store.ReturnItems(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(returns) == 0 {
		t.Fatal("expected reusable return items")
	}
	for i := range returns {
		returns[i].ReturnedQuantity = returns[i].LoadedQuantity
	}
	returns[0].ReturnedQuantity = returns[0].LoadedQuantity - 1
	returns[0].LostQuantity = 1
	var inventoryID int64
	if err = store.DB().QueryRowContext(ctx, "SELECT inventory_item_id FROM checklist_items WHERE id=?", returns[0].ChecklistItemID).Scan(&inventoryID); err != nil {
		t.Fatal(err)
	}
	before, err := store.GetInventoryItem(ctx, inventoryID)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.SaveReturnInspections(ctx, 1, returns, 0, "Retorno de teste"); err != nil {
		t.Fatal(err)
	}
	if err = store.FinalizeReturn(ctx, 1, 0, "Finalizado em teste"); err != nil {
		t.Fatal(err)
	}
	after, err := store.GetInventoryItem(ctx, inventoryID)
	if err != nil {
		t.Fatal(err)
	}
	if after.StockQuantity != before.StockQuantity-1 {
		t.Fatalf("stock after loss got %.0f, want %.0f", after.StockQuantity, before.StockQuantity-1)
	}
	event, err := store.GetEvent(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if event.Status != "completed" {
		t.Fatalf("event status %s", event.Status)
	}
}

func TestChecklistRegenerationIsIdempotentAndPreservesOverride(t *testing.T) {
	store, closeStore := testStore(t)
	defer closeStore()
	service := NewChecklistService(store)
	ctx := context.Background()
	first, err := service.Generate(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) == 0 {
		t.Fatal("expected generated items")
	}
	firstID := first.Items[0].ID
	if err := store.OverrideChecklistItem(ctx, firstID, 99, "Teste de ajuste", 0); err != nil {
		t.Fatal(err)
	}
	second, err := service.Generate(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != len(first.Items) {
		t.Fatalf("item count changed from %d to %d", len(first.Items), len(second.Items))
	}
	seen := map[string]bool{}
	preserved := false
	for _, item := range second.Items {
		if seen[item.SourceKey] {
			t.Fatalf("duplicate source key %s", item.SourceKey)
		}
		seen[item.SourceKey] = true
		if item.ID == firstID {
			preserved = item.ManualOverride && item.RequiredQuantity == 99
		}
	}
	if !preserved {
		t.Fatal("manual override was not preserved")
	}
}

func TestSelectedCookAddsBoxesAndHidesTheirStoredItems(t *testing.T) {
	store, closeStore := testStore(t)
	defer closeStore()
	ctx := context.Background()
	service := NewChecklistService(store)

	before, err := service.Generate(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	looseBefore := map[string]bool{}
	for _, item := range before.Items {
		looseBefore[item.Name] = true
	}
	if !looseBefore["Colher de serviço"] || !looseBefore["Concha"] {
		t.Fatal("demo event should initially require the loose service spoon and ladle")
	}

	var crisID int64
	if err := store.DB().QueryRowContext(ctx, "SELECT id FROM kitchen_cooks WHERE slug='cris'").Scan(&crisID); err != nil {
		t.Fatal(err)
	}
	event, err := store.GetEvent(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	event.KitchenCookID = sql.NullInt64{Int64: crisID, Valid: true}
	if err := store.SaveEvent(ctx, &event, 0); err != nil {
		t.Fatal(err)
	}

	after, err := service.Generate(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	items := map[string]bool{}
	for _, item := range after.Items {
		items[item.Name] = true
	}
	if items["Colher de serviço"] || items["Concha"] {
		t.Fatal("items stored in the selected cook boxes must not appear loose in the checklist")
	}
	if !items["Caixa da cozinheira — Cris"] {
		t.Fatal("selected cook storage box must appear in the checklist")
	}
}

func TestStockReservationIsTransactional(t *testing.T) {
	store, closeStore := testStore(t)
	defer closeStore()
	service := NewChecklistService(store)
	ctx := context.Background()
	if _, err := service.Generate(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if err := store.ReserveEvent(ctx, 1, 0, true); err != nil {
		t.Fatal(err)
	}
	item, err := store.GetInventoryItem(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	if item.ReservedQuantity != 24 {
		t.Fatalf("jugs reserved got %.0f, want 24", item.ReservedQuantity)
	}
	event, err := store.GetEvent(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if event.Status != "reserved" {
		t.Fatalf("event status got %s", event.Status)
	}
	event.GuestCount = 100
	if err := store.SaveEvent(ctx, &event, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Generate(ctx, 1); err != nil {
		t.Fatal(err)
	}
	item, err = store.GetInventoryItem(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	if item.ReservedQuantity != 12 {
		t.Fatalf("updated jug reservation got %.0f, want 12", item.ReservedQuantity)
	}
}

func TestChecklistIncludesOnlySelectedBeverages(t *testing.T) {
	store, closeStore := testStore(t)
	defer closeStore()
	ctx := context.Background()
	initial, err := NewChecklistService(store).Generate(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	initialBeverages := map[string]float64{}
	for _, item := range initial.Items {
		if item.Name == "Coca-Cola PET" || item.Name == "Guaraná PET" || item.Name == "Suco de laranja" || item.Name == "Suco de uva" {
			initialBeverages[item.Name] = item.RequiredQuantity
		}
	}
	for name, want := range map[string]float64{"Coca-Cola PET": 60, "Guaraná PET": 40, "Suco de laranja": 50, "Suco de uva": 50} {
		if got := initialBeverages[name]; got != want {
			t.Fatalf("weighted quantity for %q got %.0f, want %.0f", name, got, want)
		}
	}

	selection, err := store.EventMenuSelection(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	for index := range selection {
		if selection[index].CategoryName == "Bebidas" {
			selection[index].Selected = selection[index].MenuItemName == "Coca-Cola"
		}
		if selection[index].Selected {
			selection[index].Portions = 200
		}
	}
	if err := store.SaveEventMenu(ctx, 1, selection); err != nil {
		t.Fatal(err)
	}

	checklist, err := NewChecklistService(store).Generate(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	beverages := map[string]float64{}
	for _, item := range checklist.Items {
		if item.Name == "Coca-Cola PET" || item.Name == "Guaraná PET" || item.Name == "Suco de laranja" || item.Name == "Suco de uva" {
			beverages[item.Name] = item.RequiredQuantity
		}
	}
	if got := beverages["Coca-Cola PET"]; got != 100 {
		t.Fatalf("selected Coca-Cola quantity got %.0f, want 100", got)
	}
	for _, name := range []string{"Guaraná PET", "Suco de laranja", "Suco de uva"} {
		if _, exists := beverages[name]; exists {
			t.Fatalf("unselected beverage %q was included in checklist", name)
		}
	}
}

func TestDisposableRulesMatchEventBriefing(t *testing.T) {
	store, closeStore := testStore(t)
	defer closeStore()
	ctx := context.Background()
	rules, err := store.ListRules(ctx, false)
	if err != nil {
		t.Fatal(err)
	}

	calculate := func(welcome, dadinho bool) map[string]float64 {
		t.Helper()
		results, err := CalculateRules(CalculationInput{GuestCount: 200, HasWelcomeDrinks: welcome, HasDadinhoTapioca: dadinho}, rules)
		if err != nil {
			t.Fatal(err)
		}
		values := map[string]float64{}
		for _, result := range results {
			values[result.Rule.RuleKey] = result.Quantity
		}
		return values
	}

	standard := calculate(false, false)
	want := map[string]float64{
		"disposable_cups":     600,
		"napkins":             12,
		"paper_towels":        7,
		"aluminum_foil":       4,
		"plastic_wrap":        4,
		"culinary_bags":       4,
		"trash_bags":          20,
		"detergent":           10,
		"sponges":             10,
		"steel_wool":          12,
		"ethyl_alcohol":       2,
		"firelighter_alcohol": 1,
		"bar_soap":            6,
		"matches":             10,
		"toothpicks_standard": 2,
		"gloves":              16,
		"hairnets":            12,
	}
	for ruleKey, quantity := range want {
		if got := standard[ruleKey]; got != quantity {
			t.Errorf("rule %s got %.0f, want %.0f", ruleKey, got, quantity)
		}
	}
	if _, exists := standard["toothpicks_welcome"]; exists {
		t.Error("welcome toothpick rule ran for a standard event")
	}
	if _, exists := standard["toothpicks_dadinho"]; exists {
		t.Error("dadinho toothpick rule ran for a standard event")
	}

	welcome := calculate(true, false)
	if got := welcome["toothpicks_welcome"]; got != 4 {
		t.Errorf("welcome toothpicks got %.0f, want 4", got)
	}
	if _, exists := welcome["toothpicks_standard"]; exists {
		t.Error("standard toothpick rule ran together with welcome drinks")
	}

	dadinho := calculate(false, true)
	if got := dadinho["toothpicks_dadinho"]; got != 4 {
		t.Errorf("dadinho toothpicks got %.0f, want 4", got)
	}
	if _, exists := dadinho["toothpicks_standard"]; exists {
		t.Error("standard toothpick rule ran together with dadinho de tapioca")
	}
}

func TestChecklistIncludesSelectedServiceMaterials(t *testing.T) {
	store, closeStore := testStore(t)
	defer closeStore()
	ctx := context.Background()
	models, err := store.ListServiceModels(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	var selected []int64
	for _, model := range models {
		if model.Slug == "bar" || model.Slug == "decoracao" || model.Slug == "robo-de-led" {
			selected = append(selected, model.ID)
		}
	}
	if err := store.ApplyServiceSnapshots(ctx, 1, selected, 0); err != nil {
		t.Fatal(err)
	}
	checklist, err := NewChecklistService(store).Generate(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	items := map[string]float64{}
	for _, item := range checklist.Items {
		items[item.Name] = item.RequiredQuantity
	}
	for _, name := range []string{"Gelo", "Canudo", "Copo de acrílico", "Robô de LED", "Canhão de CO2", "Mesa redonda"} {
		if items[name] <= 0 {
			t.Fatalf("service material %q missing from checklist: %#v", name, items)
		}
	}
	if items["Mesa redonda"] != 25 {
		t.Fatalf("table requirement got %.0f, want 25 for 200 guests", items["Mesa redonda"])
	}
}

func TestChecklistIncludesCalculatedMenuRecipeIngredients(t *testing.T) {
	store, closeStore := testStore(t)
	defer closeStore()
	ctx := context.Background()
	var ribsID, stroganoffID int64
	if err := store.DB().QueryRowContext(ctx, "SELECT id FROM menu_items WHERE name='Costelinha ao molho barbecue' LIMIT 1").Scan(&ribsID); err != nil {
		t.Fatal(err)
	}
	if err := store.DB().QueryRowContext(ctx, "SELECT id FROM menu_items WHERE name='Estrogonofe de frango' LIMIT 1").Scan(&stroganoffID); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveEventMenu(ctx, 1, []models.EventMenuItem{
		{MenuItemID: ribsID, Selected: true, Portions: 100},
		{MenuItemID: stroganoffID, Selected: true, Portions: 80},
	}); err != nil {
		t.Fatal(err)
	}
	checklist, err := NewChecklistService(store).Generate(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	quantities := map[string]float64{}
	for _, item := range checklist.Items {
		if strings.HasPrefix(item.SourceKey, "menu-recipe:") {
			quantities[item.Name] = item.RequiredQuantity
		}
	}
	want := map[string]float64{"Costela suína": 33.34, "Molho barbecue": 1, "Frango": 40, "Ketchup": 1, "Mostarda": 1}
	for name, quantity := range want {
		if math.Abs(quantities[name]-quantity) > 0.001 {
			t.Errorf("checklist recipe %s got %.2f, want %.2f", name, quantities[name], quantity)
		}
	}
}

func TestKitchenCookSelectionReplacesPersonalStorageBoxes(t *testing.T) {
	store, closeStore := testStore(t)
	defer closeStore()
	ctx := context.Background()

	var crisID, suelemID int64
	if err := store.DB().QueryRowContext(ctx, "SELECT id FROM kitchen_cooks WHERE slug='cris'").Scan(&crisID); err != nil {
		t.Fatal(err)
	}
	if err := store.DB().QueryRowContext(ctx, "SELECT id FROM kitchen_cooks WHERE slug='suelem'").Scan(&suelemID); err != nil {
		t.Fatal(err)
	}
	event, err := store.GetEvent(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	event.KitchenCookID = sql.NullInt64{Int64: crisID, Valid: true}
	if err := store.SaveEvent(ctx, &event, 0); err != nil {
		t.Fatal(err)
	}
	checklist, err := NewChecklistService(store).Generate(ctx, event.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertCookBoxes := func(wantCook, unwantedCook string, wantCount int) {
		t.Helper()
		count := 0
		for _, item := range checklist.Items {
			if strings.Contains(item.Name, "— "+wantCook) {
				count++
			}
			if strings.Contains(item.Name, "— "+unwantedCook) {
				t.Fatalf("found box from previous cook %s: %s", unwantedCook, item.Name)
			}
		}
		if count != wantCount {
			t.Fatalf("boxes for %s got %d, want %d", wantCook, count, wantCount)
		}
	}
	assertCookBoxes("Cris", "Suelem", 1)

	event.KitchenCookID = sql.NullInt64{Int64: suelemID, Valid: true}
	if err := store.SaveEvent(ctx, &event, 0); err != nil {
		t.Fatal(err)
	}
	checklist, err = NewChecklistService(store).Generate(ctx, event.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertCookBoxes("Suelem", "Cris", 1)

	event.KitchenCookID = sql.NullInt64{}
	if err := store.SaveEvent(ctx, &event, 0); err != nil {
		t.Fatal(err)
	}
	checklist, err = NewChecklistService(store).Generate(ctx, event.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range checklist.Items {
		if strings.HasPrefix(item.SourceKey, "kitchen-cook-box:") {
			t.Fatalf("optional cook cleared but box remained: %s", item.Name)
		}
	}
}

func TestMobileVanChecklistSavesCompleteAndMissingDecisions(t *testing.T) {
	store, closeStore := testStore(t)
	defer closeStore()
	ctx := context.Background()
	if _, err := NewChecklistService(store).Generate(ctx, 1); err != nil {
		t.Fatal(err)
	}
	var itemID int64
	var required float64
	if err := store.DB().QueryRowContext(ctx, `SELECT item.id,item.required_quantity FROM checklist_items item JOIN checklists checklist ON checklist.id=item.checklist_id WHERE checklist.event_id=1 AND item.required_quantity>=2 ORDER BY item.id LIMIT 1`).Scan(&itemID, &required); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB().ExecContext(ctx, "UPDATE checklist_items SET status='checked',separated_quantity=required_quantity WHERE id=?", itemID); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.UpdateMobileLoadingItem(ctx, 1, itemID, "complete", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != required {
		t.Fatalf("complete loaded quantity got %v, want %v", loaded, required)
	}
	var decision, status string
	var storedLoaded, missing float64
	if err := store.DB().QueryRowContext(ctx, "SELECT loading_decision,loading_missing_quantity,loaded_quantity,status FROM checklist_items WHERE id=?", itemID).Scan(&decision, &missing, &storedLoaded, &status); err != nil {
		t.Fatal(err)
	}
	if decision != "complete" || missing != 0 || storedLoaded != required || status != "loaded" {
		t.Fatalf("complete decision=%q missing=%v loaded=%v status=%q", decision, missing, storedLoaded, status)
	}
	loaded, err = store.UpdateMobileLoadingItem(ctx, 1, itemID, "missing", 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != required-2 {
		t.Fatalf("missing loaded quantity got %v, want %v", loaded, required-2)
	}
	if err := store.DB().QueryRowContext(ctx, "SELECT loading_decision,loading_missing_quantity,loaded_quantity,status FROM checklist_items WHERE id=?", itemID).Scan(&decision, &missing, &storedLoaded, &status); err != nil {
		t.Fatal(err)
	}
	if decision != "missing" || missing != 2 || storedLoaded != required-2 || status != "loaded" {
		t.Fatalf("missing decision=%q missing=%v loaded=%v status=%q", decision, missing, storedLoaded, status)
	}
	if _, err := store.UpdateMobileLoadingItem(ctx, 1, itemID, "missing", required+1, 0); err == nil {
		t.Fatal("missing quantity above required should be rejected")
	}
}

func TestMainCoursesAndSidesGenerateOneCubaEachWithoutDuplicates(t *testing.T) {
	store, closeStore := testStore(t)
	defer closeStore()
	ctx := context.Background()
	var selectedDishes int
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM event_menu_items selected JOIN menu_items item ON item.id=selected.menu_item_id JOIN menu_categories category ON category.id=item.category_id WHERE selected.event_id=1 AND category.slug IN ('main_courses','sides')`).Scan(&selectedDishes); err != nil {
		t.Fatal(err)
	}
	checklist, err := NewChecklistService(store).Generate(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	found, quantity := 0, 0.0
	for _, item := range checklist.Items {
		if item.Name == "Cuba GN 1/1" {
			found++
			quantity += item.RequiredQuantity
		}
	}
	if found != 1 || int(quantity) != selectedDishes {
		t.Fatalf("cubas got entries=%d quantity=%.0f, want one entry with %d", found, quantity, selectedDishes)
	}
}
