package repositories

import (
	"context"
	"path/filepath"
	"testing"

	"buffetflow/internal/database"
	"buffetflow/internal/models"
)

func TestStandaloneFloorLayoutCRUD(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "standalone-layout.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	store := New(db)

	listed, err := store.ListStandaloneFloorLayouts(ctx, "")
	if err != nil {
		t.Fatalf("list empty layouts: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("expected no layouts, got %d", len(listed))
	}

	layout := models.StandaloneFloorLayout{
		Name:            "Salão principal",
		Venue:           "Espaço demonstração",
		GuestCount:      80,
		WaiterCount:     4,
		WaiterNamesJSON: `["Ana","Bruno","Carla","Diego"]`,
		LayoutJSON:      defaultStandaloneLayoutJSON,
	}
	if err := store.SaveStandaloneFloorLayout(ctx, &layout, 0); err != nil {
		t.Fatal(err)
	}
	if layout.ID == 0 {
		t.Fatal("expected saved layout id")
	}
	if layout.RowVersion != 1 {
		t.Fatalf("row version got %d", layout.RowVersion)
	}

	listed, err = store.ListStandaloneFloorLayouts(ctx, "principal")
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].Name != "Salão principal" {
		t.Fatalf("list after save got %#v", listed)
	}

	layout.Venue = "Salão atualizado"
	if err := store.SaveStandaloneFloorLayout(ctx, &layout, 0); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.GetStandaloneFloorLayout(ctx, layout.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Venue != "Salão atualizado" {
		t.Fatalf("venue got %q", loaded.Venue)
	}
	if loaded.RowVersion != 2 {
		t.Fatalf("row version after update got %d", loaded.RowVersion)
	}

	if err := store.ArchiveStandaloneFloorLayout(ctx, layout.ID, 0); err != nil {
		t.Fatal(err)
	}
	listed, err = store.ListStandaloneFloorLayouts(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 0 {
		t.Fatalf("archived layout still listed: %#v", listed)
	}
}
