package repositories

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"buffetflow/internal/database"
	"buffetflow/internal/models"
)

func newModelWorkflowStore(t *testing.T) (*Store, *sql.DB, context.Context) {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "model-workflow.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	ctx := context.Background()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	return New(db), db, ctx
}

func TestMenuModelChoiceValidationAndSnapshotIndependence(t *testing.T) {
	store, db, ctx := newModelWorkflowStore(t)
	var modelID int64
	if err := db.QueryRowContext(ctx, "SELECT id FROM menu_templates WHERE slug='buffet-carnes'").Scan(&modelID); err != nil {
		t.Fatal(err)
	}
	rows, err := db.QueryContext(ctx, `SELECT choice.id,item.id FROM menu_choice_groups choice
		JOIN menu_template_sections section ON section.id=choice.menu_template_section_id
		JOIN menu_choice_group_items link ON link.menu_choice_group_id=choice.id
		JOIN menu_template_items item ON item.id=link.menu_template_item_id
		WHERE section.menu_template_id=? ORDER BY choice.id,link.sort_order`, modelID)
	if err != nil {
		t.Fatal(err)
	}
	choices := map[int64][]int64{}
	var groupOrder []int64
	for rows.Next() {
		var groupID, itemID int64
		if err := rows.Scan(&groupID, &itemID); err != nil {
			t.Fatal(err)
		}
		if _, exists := choices[groupID]; !exists {
			groupOrder = append(groupOrder, groupID)
		}
		choices[groupID] = append(choices[groupID], itemID)
	}
	rows.Close()
	var selected []int64
	for _, groupID := range groupOrder {
		if len(choices[groupID]) < 2 {
			t.Fatalf("choice group %d has fewer than two options", groupID)
		}
		selected = append(selected, choices[groupID][:2]...)
	}
	if err := store.ValidateMenuModelSelections(ctx, modelID, selected); err != nil {
		t.Fatalf("valid exact choices rejected: %v", err)
	}
	if err := store.ValidateMenuModelSelections(ctx, modelID, selected[1:]); err == nil {
		t.Fatal("invalid choice count should be rejected")
	}
	if err := store.ApplyMenuModelSnapshot(ctx, 1, modelID, selected, []string{"Pedido exclusivo do cliente"}, 0); err != nil {
		t.Fatal(err)
	}
	var sourceItemID int64
	var originalName string
	var included, configurable int
	if err := db.QueryRowContext(ctx, `SELECT item.id,item.normalized_name,item.included,item.configurable FROM menu_template_items item JOIN menu_template_sections section ON section.id=item.menu_template_section_id WHERE section.menu_template_id=? ORDER BY item.id LIMIT 1`, modelID).Scan(&sourceItemID, &originalName, &included, &configurable); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateMenuModelItem(ctx, modelID, sourceItemID, originalName+" atualizado", "", included == 1, configurable == 1, 0); err != nil {
		t.Fatal(err)
	}
	var snapshotName string
	if err := db.QueryRowContext(ctx, `SELECT item.normalized_name FROM event_menu_snapshot_items item JOIN event_menu_sections section ON section.id=item.event_menu_section_id JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id WHERE snapshot.event_id=1 AND item.source_template_item_id=?`, sourceItemID).Scan(&snapshotName); err != nil {
		t.Fatal(err)
	}
	if snapshotName != originalName {
		t.Fatalf("event snapshot changed with source model: got %q want %q", snapshotName, originalName)
	}
	snapshotVersion, currentVersion, _, err := store.EventMenuModelStatus(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if snapshotVersion != 1 || currentVersion != 2 {
		t.Fatalf("version comparison got snapshot=%d current=%d", snapshotVersion, currentVersion)
	}
	differences, err := store.CompareEventMenuModel(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(differences) == 0 {
		t.Fatal("updated source item should appear in model comparison")
	}
	configurations := map[int64]models.EventMenuItemConfiguration{sourceItemID: {TemplateItemID: sourceItemID, Portions: sql.NullFloat64{Float64: 150, Valid: true}}}
	if err := store.UpdateEventMenuSnapshotConfigurations(ctx, 1, configurations, 0); err != nil {
		t.Fatal(err)
	}
	if err := store.ApplyMenuModelSnapshot(ctx, 1, modelID, selected, []string{"Pedido exclusivo do cliente"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateEventMenuSnapshotConfigurations(ctx, 1, configurations, 0); err != nil {
		t.Fatal(err)
	}
	var portions float64
	if err := db.QueryRowContext(ctx, `SELECT item.portions FROM event_menu_snapshot_items item JOIN event_menu_sections section ON section.id=item.event_menu_section_id JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id WHERE snapshot.event_id=1 AND item.source_template_item_id=?`, sourceItemID).Scan(&portions); err != nil {
		t.Fatal(err)
	}
	if portions != 150 {
		t.Fatalf("manual portions got %.0f, want 150 after reapply", portions)
	}
	var customCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM event_menu_snapshot_items item JOIN event_menu_sections section ON section.id=item.event_menu_section_id JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id WHERE snapshot.event_id=1 AND item.custom_item=1 AND item.normalized_name='Pedido exclusivo do cliente'`).Scan(&customCount); err != nil {
		t.Fatal(err)
	}
	if customCount != 1 {
		t.Fatalf("custom snapshot items got %d, want 1", customCount)
	}
}

func TestCakeSectionFollowsEventCakeOption(t *testing.T) {
	store, db, ctx := newModelWorkflowStore(t)
	if _, err := db.ExecContext(ctx, "UPDATE events SET has_cake=0,cake_notes='' WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	var modelID int64
	if err := db.QueryRowContext(ctx, "SELECT id FROM menu_templates WHERE slug='buffet-brunch'").Scan(&modelID); err != nil {
		t.Fatal(err)
	}
	if err := store.ApplyMenuModelSnapshot(ctx, 1, modelID, nil, nil, 0); err != nil {
		t.Fatal(err)
	}
	selectedCakeItems := func() int {
		t.Helper()
		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM event_menu_snapshot_items item JOIN event_menu_sections section ON section.id=item.event_menu_section_id JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id WHERE snapshot.event_id=1 AND LOWER(section.name) LIKE '%bolo%' AND item.selected=1 AND item.was_removed=0`).Scan(&count); err != nil {
			t.Fatal(err)
		}
		return count
	}
	if got := selectedCakeItems(); got != 0 {
		t.Fatalf("cake items selected without cake = %d", got)
	}
	if _, err := db.ExecContext(ctx, "UPDATE events SET has_cake=1,cake_notes='Limão' WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	if err := store.SyncEventCakePresence(ctx, 1, 0); err != nil {
		t.Fatal(err)
	}
	if got := selectedCakeItems(); got == 0 {
		t.Fatal("cake items should be selected after enabling cake")
	}
	var cakeCount int
	if err := db.QueryRowContext(ctx, "SELECT cake_count FROM event_cake_configurations WHERE event_id=1").Scan(&cakeCount); err != nil {
		t.Fatal(err)
	}
	if cakeCount < 1 {
		t.Fatalf("cake count got %d, want at least one", cakeCount)
	}
}

func TestEventDadinhoConditionReadsSelectedMenuSnapshot(t *testing.T) {
	store, db, ctx := newModelWorkflowStore(t)
	var modelID int64
	if err := db.QueryRowContext(ctx, "SELECT id FROM menu_templates WHERE slug='buffet-nordestino'").Scan(&modelID); err != nil {
		t.Fatal(err)
	}
	if err := store.ApplyMenuModelSnapshot(ctx, 1, modelID, nil, nil, 0); err != nil {
		t.Fatal(err)
	}
	found, err := store.EventHasDadinhoTapioca(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("selected Dadinho de tapioca was not detected in event menu snapshot")
	}
	if _, err := db.ExecContext(ctx, `UPDATE event_menu_snapshot_items SET selected=0 WHERE normalized_name='Dadinho de tapioca' AND event_menu_section_id IN (SELECT section.id FROM event_menu_sections section JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id WHERE snapshot.event_id=1)`); err != nil {
		t.Fatal(err)
	}
	found, err = store.EventHasDadinhoTapioca(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("unselected Dadinho de tapioca still triggered event condition")
	}
}

func TestDuplicatedModelsKeepChoiceAndInventoryLinks(t *testing.T) {
	store, db, ctx := newModelWorkflowStore(t)
	var menuID, serviceID int64
	if err := db.QueryRowContext(ctx, "SELECT id FROM menu_templates WHERE slug='buffet-carnes'").Scan(&menuID); err != nil {
		t.Fatal(err)
	}
	menuCopyID, err := store.DuplicateMenuModel(ctx, menuID, 0)
	if err != nil {
		t.Fatal(err)
	}
	var originalChoices, copiedChoices int
	choiceCount := `SELECT COUNT(*) FROM menu_choice_group_items membership JOIN menu_choice_groups choice ON choice.id=membership.menu_choice_group_id JOIN menu_template_sections section ON section.id=choice.menu_template_section_id WHERE section.menu_template_id=?`
	if err := db.QueryRowContext(ctx, choiceCount, menuID).Scan(&originalChoices); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, choiceCount, menuCopyID).Scan(&copiedChoices); err != nil {
		t.Fatal(err)
	}
	if originalChoices == 0 || copiedChoices != originalChoices {
		t.Fatalf("menu choice links original=%d copied=%d", originalChoices, copiedChoices)
	}
	if err := db.QueryRowContext(ctx, "SELECT id FROM service_templates WHERE slug='celebrante'").Scan(&serviceID); err != nil {
		t.Fatal(err)
	}
	serviceCopyID, err := store.DuplicateServiceModel(ctx, serviceID, 0)
	if err != nil {
		t.Fatal(err)
	}
	var copiedServiceChoices int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM service_choice_group_components membership JOIN service_choice_groups choice ON choice.id=membership.service_choice_group_id WHERE choice.service_template_id=?`, serviceCopyID).Scan(&copiedServiceChoices); err != nil {
		t.Fatal(err)
	}
	if copiedServiceChoices != 2 {
		t.Fatalf("duplicated celebrant choices got %d, want 2", copiedServiceChoices)
	}
}

func TestServiceSnapshotsProduceOperationalRequirements(t *testing.T) {
	store, db, ctx := newModelWorkflowStore(t)
	rows, err := db.QueryContext(ctx, "SELECT id FROM service_templates WHERE slug IN ('bar','robo-de-led','pista-de-led','decoracao') ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	var serviceIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		serviceIDs = append(serviceIDs, id)
	}
	rows.Close()
	if err := store.ApplyServiceSnapshots(ctx, 1, serviceIDs, 0); err != nil {
		t.Fatal(err)
	}
	requirements, err := store.EventServiceRequirements(ctx, 1, 17)
	if err != nil {
		t.Fatal(err)
	}
	quantities := map[string]float64{}
	for _, requirement := range requirements {
		var code string
		if err := db.QueryRowContext(ctx, "SELECT internal_code FROM inventory_items WHERE id=?", requirement.InventoryItemID).Scan(&code); err != nil {
			t.Fatal(err)
		}
		quantities[code] = requirement.Quantity
	}
	for _, code := range []string{"LED-ROBO", "LED-CANHAO", "LED-PISTA", "DEC-MESA-8"} {
		if quantities[code] <= 0 {
			t.Fatalf("missing operational requirement %s: %#v", code, quantities)
		}
	}
	for _, code := range []string{"BAR-GELO", "BAR-CANUDO", "BAR-COPO"} {
		if quantities[code] > 0 {
			t.Fatalf("rare bar material %s should remain optional: %#v", code, quantities)
		}
	}
	if quantities["DEC-MESA-8"] != 3 {
		t.Fatalf("table formula got %v, want 3 for 17 guests", quantities["DEC-MESA-8"])
	}
	if _, err := db.ExecContext(ctx, "UPDATE service_component_inventory_links SET active=0"); err != nil {
		t.Fatal(err)
	}
	preserved, err := store.EventServiceRequirements(ctx, 1, 17)
	if err != nil {
		t.Fatal(err)
	}
	if len(preserved) != len(requirements) {
		t.Fatalf("service snapshot requirements changed with source model: got %d, want %d", len(preserved), len(requirements))
	}
	var barDuration, robotDuration int
	if err := db.QueryRowContext(ctx, "SELECT duration_minutes FROM event_services WHERE event_id=1 AND snapshot_name='Bar'").Scan(&barDuration); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, "SELECT duration_minutes FROM event_services WHERE event_id=1 AND snapshot_name='Robô de LED'").Scan(&robotDuration); err != nil {
		t.Fatal(err)
	}
	if barDuration != 240 || robotDuration != 60 {
		t.Fatalf("snapshot durations got bar=%d robot=%d", barDuration, robotDuration)
	}
}

func TestAdministrativeModelItemCRUDVersionsEveryChange(t *testing.T) {
	store, db, ctx := newModelWorkflowStore(t)
	modelID, err := store.CreateMenuModel(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	var sectionID int64
	if err := db.QueryRowContext(ctx, "SELECT id FROM menu_template_sections WHERE menu_template_id=? ORDER BY sort_order LIMIT 1", modelID).Scan(&sectionID); err != nil {
		t.Fatal(err)
	}
	if err := store.AddMenuModelItem(ctx, modelID, sectionID, "Item administrativo", "Primeira versão", true, true, 0); err != nil {
		t.Fatal(err)
	}
	var itemID int64
	if err := db.QueryRowContext(ctx, "SELECT id FROM menu_template_items WHERE menu_template_section_id=? AND normalized_name='Item administrativo'", sectionID).Scan(&itemID); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateMenuModelItem(ctx, modelID, itemID, "Item administrativo atualizado", "Segunda versão", false, true, 0); err != nil {
		t.Fatal(err)
	}
	if err := store.RemoveMenuModelItem(ctx, modelID, itemID, 0); err != nil {
		t.Fatal(err)
	}
	if err := store.AddMenuModelSection(ctx, modelID, "Estação especial", "food", 0); err != nil {
		t.Fatal(err)
	}
	var administrativeSectionID int64
	if err := db.QueryRowContext(ctx, "SELECT id FROM menu_template_sections WHERE menu_template_id=? AND display_name='Estação especial'", modelID).Scan(&administrativeSectionID); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateMenuModelSection(ctx, modelID, administrativeSectionID, "Estação premium", 90, 1, sql.NullInt64{Int64: 2, Valid: true}, true, true, 0); err != nil {
		t.Fatal(err)
	}
	if err := store.RemoveMenuModelSection(ctx, modelID, administrativeSectionID, 0); err != nil {
		t.Fatal(err)
	}
	var version, versionRows, activeRows int
	if err := db.QueryRowContext(ctx, "SELECT current_version FROM menu_templates WHERE id=?", modelID).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM menu_template_versions WHERE menu_template_id=?", modelID).Scan(&versionRows); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM menu_template_items WHERE id=? AND deleted_at IS NULL", itemID).Scan(&activeRows); err != nil {
		t.Fatal(err)
	}
	if version != 7 || versionRows != 7 || activeRows != 0 {
		t.Fatalf("CRUD versioning got version=%d rows=%d active=%d", version, versionRows, activeRows)
	}
}
