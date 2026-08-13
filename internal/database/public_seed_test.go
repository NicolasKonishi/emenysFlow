//go:build !private_seeds

package database

import (
	"context"
	"path/filepath"
	"testing"
)

func TestPublicMigrationsApplyWithMinimalSeed(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "public-seed.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}

	var templates, services, events int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM menu_templates").Scan(&templates); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM service_templates").Scan(&services); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM events WHERE active=1").Scan(&events); err != nil {
		t.Fatal(err)
	}
	if templates < 2 || services < 2 || events < 1 {
		t.Fatalf("minimal seed too small: templates=%d services=%d events=%d", templates, services, events)
	}
}
