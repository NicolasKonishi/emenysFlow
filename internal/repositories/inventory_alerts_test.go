package repositories

import (
	"context"
	"path/filepath"
	"testing"

	"buffetflow/internal/database"
)

func TestLoadingShortageAppearsInInventoryAlerts(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "inventory-alerts.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	store := New(db)

	result, err := db.ExecContext(ctx, `INSERT INTO checklists(event_id,version,generated_at,updated_at)
		VALUES(1,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatal(err)
	}
	checklistID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	result, err = db.ExecContext(ctx, `INSERT INTO checklist_items(checklist_id,inventory_item_id,category_id,source_key,name,unit,
		calculated_quantity,required_quantity,available_quantity,reserved_elsewhere_quantity,missing_quantity,
		calculation_origin,notes,status,item_kind,location_snapshot,separated_quantity,created_at,updated_at)
		VALUES(?,4,7,'loading-alert-test','Copo descartável','unidade',10,10,10,0,0,'Teste','',
			'separated','consumable','Área de descartáveis',10,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, checklistID)
	if err != nil {
		t.Fatal(err)
	}
	itemID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := store.UpdateMobileLoadingItem(ctx, 1, itemID, "missing", 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != 7 {
		t.Fatalf("loaded quantity = %v; want 7", loaded)
	}
	alerts, err := store.ListInventoryAlerts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 {
		t.Fatalf("alerts = %#v; want one loading shortage", alerts)
	}
	if alert := alerts[0]; alert.ChecklistItemID != itemID || alert.Loaded != 7 || alert.Required != 10 || alert.Missing != 3 {
		t.Fatalf("unexpected inventory alert: %#v", alert)
	}
	shortages, err := store.ListEventShortages(ctx, 1, false)
	if err != nil || len(shortages) != 1 {
		t.Fatalf("active shortages = %#v, %v; want one", shortages, err)
	}
	if err := store.UpdateShortageStatus(ctx, 1, shortages[0].ID, "cancelled", "", "", 0); err == nil {
		t.Fatal("an automatic loading shortage should stay open while the loaded quantity is incomplete")
	}

	if _, err := store.UpdateMobileLoadingItem(ctx, 1, itemID, "complete", 0, 0); err != nil {
		t.Fatal(err)
	}
	alerts, err = store.ListInventoryAlerts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 0 {
		t.Fatalf("alerts after completing load = %#v; want none", alerts)
	}
}
