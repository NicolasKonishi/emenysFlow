package services

import (
	"context"
	"path/filepath"
	"testing"

	"buffetflow/internal/database"
	"buffetflow/internal/models"
	"buffetflow/internal/repositories"
)

func checklistStatusTestStore(t *testing.T) (*repositories.Store, func()) {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(context.Background(), db); err != nil {
		db.Close()
		t.Fatal(err)
	}
	return repositories.New(db), func() { _ = db.Close() }
}

func TestChecklistNotHavingAndSeparatedCloseActiveShortages(t *testing.T) {
	store, closeStore := checklistStatusTestStore(t)
	defer closeStore()
	ctx := context.Background()
	checklist, err := NewChecklistService(store).Generate(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(checklist.Items) == 0 {
		t.Fatal("generated checklist is empty")
	}
	item := checklist.Items[0]
	missing := item.RequiredQuantity
	if missing > 1 {
		missing = 1
	}
	shortage := models.ChecklistShortage{EventID: 1, ChecklistItemID: item.ID, MissingQuantity: missing, Reason: "Será comprado", ResolutionType: "purchase"}
	if err := store.SaveChecklistShortage(ctx, shortage, 0); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateChecklistItemStatus(ctx, item.ID, "not_applicable", 0); err != nil {
		t.Fatal(err)
	}
	var status string
	var separated, loaded float64
	if err := store.DB().QueryRowContext(ctx, "SELECT status,separated_quantity,loaded_quantity FROM checklist_items WHERE id=?", item.ID).Scan(&status, &separated, &loaded); err != nil {
		t.Fatal(err)
	}
	if status != "not_applicable" || separated != 0 || loaded != 0 {
		t.Fatalf("not-have state status=%q separated=%v loaded=%v", status, separated, loaded)
	}
	var activeShortages int
	if err := store.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM checklist_shortages WHERE checklist_item_id=? AND status NOT IN ('resolved','cancelled')", item.ID).Scan(&activeShortages); err != nil {
		t.Fatal(err)
	}
	if activeShortages != 0 {
		t.Fatalf("not-have state left %d active shortages", activeShortages)
	}

	if err := store.UpdateChecklistItemStatus(ctx, item.ID, "pending", 0); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveChecklistShortage(ctx, shortage, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveOperationalQuantity(ctx, 1, item.ID, "separation", item.RequiredQuantity, "", 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := store.DB().QueryRowContext(ctx, "SELECT status,separated_quantity FROM checklist_items WHERE id=?", item.ID).Scan(&status, &separated); err != nil {
		t.Fatal(err)
	}
	if status != "separated" || separated != item.RequiredQuantity {
		t.Fatalf("separated state status=%q separated=%v want=%v", status, separated, item.RequiredQuantity)
	}
	if err := store.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM checklist_shortages WHERE checklist_item_id=? AND status NOT IN ('resolved','cancelled')", item.ID).Scan(&activeShortages); err != nil {
		t.Fatal(err)
	}
	if activeShortages != 0 {
		t.Fatalf("separated state left %d active shortages", activeShortages)
	}
}
