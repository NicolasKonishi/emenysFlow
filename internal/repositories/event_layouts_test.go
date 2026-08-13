package repositories

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"buffetflow/internal/database"
)

func TestEventFloorLayoutSaveAndLoad(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "layout-test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	store := New(db)

	layoutJSON := `{"version":1,"width":1400,"height":900,"elements":[{"id":"t1","type":"table_round","x":100,"y":120,"width":88,"height":88,"label":"Mesa 1","waiter":"Ana","seats":8}]}`
	if err := store.SaveEventFloorLayout(ctx, 1, layoutJSON, 0); err != nil {
		t.Fatal(err)
	}
	layout, err := store.GetEventFloorLayout(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if layout.LayoutJSON != layoutJSON {
		t.Fatalf("layout json got %q", layout.LayoutJSON)
	}
	if layout.RowVersion != 1 {
		t.Fatalf("row version got %d", layout.RowVersion)
	}
	if err := store.SaveEventFloorLayout(ctx, 1, layoutJSON, 0); err != nil {
		t.Fatal(err)
	}
	layout, err = store.GetEventFloorLayout(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if layout.RowVersion != 2 {
		t.Fatalf("row version after update got %d", layout.RowVersion)
	}
}

func TestEventFloorLayoutVersionConflict(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "layout-version.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	store := New(db)

	layoutJSON := `{"version":2,"width":1400,"height":900,"waiters":[],"elements":[]}`
	if err := store.SaveEventFloorLayout(ctx, 1, layoutJSON, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveEventFloorLayoutVersioned(ctx, 1, layoutJSON, 0, 1); err != nil {
		t.Fatalf("expected update with matching version: %v", err)
	}
	if _, err := store.SaveEventFloorLayoutVersioned(ctx, 1, layoutJSON, 0, 1); err == nil || !strings.Contains(err.Error(), "version conflict") {
		t.Fatalf("expected version conflict, got %v", err)
	}
}

func TestEventFloorLayoutRejectsInvalidJSON(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "layout-invalid.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	store := New(db)
	if err := store.SaveEventFloorLayout(ctx, 1, "{invalid", 0); err != ErrInvalidLayoutJSON {
		t.Fatalf("expected ErrInvalidLayoutJSON, got %v", err)
	}
}
