package main

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"buffetflow/internal/database"
)

// The export runs against the public seed: generic rules, two menu templates
// and the demo decorations. It must produce a bundle the CRM accepts.
func TestExportPublicSeed(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	exported, err := export(db)
	if err != nil {
		t.Fatal(err)
	}
	if exported.Version != 1 || len(exported.Menus) == 0 || len(exported.Decorations) == 0 || len(exported.Rules) == 0 {
		t.Fatalf("export is missing data: %d menus, %d decorations, %d rules", len(exported.Menus), len(exported.Decorations), len(exported.Rules))
	}

	byKey := map[string]rule{}
	menuKeys := map[string]bool{}
	for _, menu := range exported.Menus {
		menuKeys[menu.Key] = true
	}
	for _, item := range exported.Rules {
		byKey[item.Key] = item
		if item.Quantity <= 0 || item.ItemName == "" || item.MenuKeys == nil {
			t.Errorf("invalid rule %+v", item)
		}
		switch item.Category {
		case "equipment", "disposable", "decoration", "food":
		default:
			t.Errorf("rule %s has category %q", item.Key, item.Category)
		}
		for _, key := range item.MenuKeys {
			if !menuKeys[key] {
				t.Errorf("rule %s points to unknown menu %s", item.Key, key)
			}
		}
	}
	if _, ok := byKey["rule:1"]; ok {
		t.Error("waiters are staff, not material, and must not be exported")
	}
	if cups := byKey["rule:4"]; cups.Category != "disposable" || cups.Quantity != 3 || cups.PeoplePerUnit != 1 {
		t.Errorf("cups rule = %+v", cups)
	}
	if jugs := byKey["rule:2"]; jugs.Quantity != 2 || jugs.PeoplePerUnit != 20 {
		t.Errorf("two jugs per waiter, one waiter per 20 guests in the public seed; got %+v", jugs)
	}

	encoded, err := json.Marshal(exported)
	if err != nil {
		t.Fatal(err)
	}
	var generic map[string]any
	if err := json.Unmarshal(encoded, &generic); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"version", "source", "menus", "decorations", "rules", "layouts"} {
		if _, ok := generic[field]; !ok {
			t.Errorf("bundle misses %q", field)
		}
	}
}
