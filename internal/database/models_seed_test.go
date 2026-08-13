//go:build private_seeds

package database

import (
	"context"
	"path/filepath"
	"testing"
)

func TestModelSeedsAreCompleteAndIdempotent(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "models-seed.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}

	assertCount := func(query string, want int) {
		t.Helper()
		var got int
		if err := db.QueryRowContext(ctx, query).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("query %q got %d, want %d", query, got, want)
		}
	}
	assertCount("SELECT COUNT(*) FROM menu_templates WHERE deleted_at IS NULL", 14)
	assertCount("SELECT COUNT(*) FROM service_templates WHERE deleted_at IS NULL", 11)
	assertCount("SELECT COUNT(*) FROM menu_template_versions WHERE version=1", 14)
	assertCount("SELECT COUNT(*) FROM service_template_versions WHERE version=1", 11)
	assertCount("SELECT COUNT(*) FROM kitchen_cooks WHERE active=1", 3)
	assertCount("SELECT COUNT(*) FROM kitchen_cook_storage_boxes WHERE active=1", 3)
	assertCount("SELECT COUNT(*) FROM kitchen_cooks cook WHERE (SELECT COUNT(*) FROM kitchen_cook_storage_boxes box WHERE box.kitchen_cook_id=cook.id AND box.active=1)=1", 3)
	assertCount("SELECT COUNT(*) FROM kitchen_cook_box_items WHERE active=1", 33)
	assertCount("SELECT COUNT(*) FROM kitchen_cook_storage_boxes box WHERE box.active=1 AND (SELECT COUNT(*) FROM kitchen_cook_box_items item WHERE item.kitchen_cook_storage_box_id=box.id AND item.active=1)=11", 3)
	assertCount("SELECT COUNT(*) FROM inventory_items WHERE active=1 AND name LIKE 'Caixa da cozinheira — %'", 3)
	assertCount("SELECT COUNT(*) FROM inventory_items WHERE active=1 AND internal_code BETWEEN 'DES-001' AND 'DES-017'", 17)
	assertCount("SELECT COUNT(*) FROM calculation_rules WHERE active=1 AND category_id=(SELECT id FROM inventory_categories WHERE name='Descartáveis')", 19)
	assertCount("SELECT COUNT(*) FROM inventory_items WHERE active=1 AND internal_code LIKE 'ING-%'", 5)
	assertCount("SELECT COUNT(*) FROM inventory_items WHERE active=1 AND internal_code BETWEEN 'DEC-001' AND 'DEC-041'", 41)
	assertCount("SELECT COUNT(*) FROM decorations WHERE active=1 AND inventory_item_id IN (SELECT id FROM inventory_items WHERE internal_code BETWEEN 'DEC-001' AND 'DEC-041')", 41)
	assertCount("SELECT COUNT(*) FROM inventory_items WHERE internal_code LIKE 'DEC-%' AND ownership='owned'", 41)
	assertCount("SELECT COUNT(*) FROM menu_templates WHERE source_name='Emeny''s Eventos - Cardápios' AND source_updated_month='2025-01'", 14)
	assertCount("SELECT COUNT(*) FROM service_templates WHERE source_name='Emeny''s Eventos - Serviços' AND source_updated_month='2025-01'", 11)
	assertCount("SELECT COUNT(*) FROM service_templates WHERE price_cents IS NOT NULL OR cost_cents IS NOT NULL OR commission_cents IS NOT NULL", 0)

	var menuItems, serviceComponents int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM menu_template_items").Scan(&menuItems); err != nil {
		t.Fatal(err)
	}
	if menuItems < 300 {
		t.Fatalf("seeded menu template items got %d, want at least 300", menuItems)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM service_template_components").Scan(&serviceComponents); err != nil {
		t.Fatal(err)
	}
	if serviceComponents < 80 {
		t.Fatalf("seeded service components got %d, want at least 80", serviceComponents)
	}
	var recipeIngredients int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM menu_item_ingredients WHERE active=1").Scan(&recipeIngredients); err != nil {
		t.Fatal(err)
	}
	if recipeIngredients < 25 {
		t.Fatalf("seeded recipe ingredients got %d, want at least 25", recipeIngredients)
	}

	assertCount(`SELECT COUNT(*) FROM menu_choice_groups choice JOIN menu_template_sections section ON section.id=choice.menu_template_section_id JOIN menu_templates model ON model.id=section.menu_template_id WHERE model.slug='buffet-carnes' AND choice.selection_min=2 AND choice.selection_max=2`, 3)
	assertCount(`SELECT COUNT(*) FROM menu_choice_groups choice JOIN menu_template_sections section ON section.id=choice.menu_template_section_id JOIN menu_templates model ON model.id=section.menu_template_id WHERE model.slug='buffet-especial' AND choice.selection_min=1 AND choice.selection_max=1`, 1)
	assertCount(`SELECT COUNT(*) FROM menu_choice_groups WHERE configurable=1 AND selection_max IS NULL`, 3)

	var barMinutes, robotMinutes int
	if err := db.QueryRowContext(ctx, "SELECT duration_minutes FROM service_templates WHERE slug='bar'").Scan(&barMinutes); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, "SELECT duration_minutes FROM service_templates WHERE slug='robo-de-led'").Scan(&robotMinutes); err != nil {
		t.Fatal(err)
	}
	if barMinutes != 240 || robotMinutes != 60 {
		t.Fatalf("seeded durations got bar=%d robot=%d", barMinutes, robotMinutes)
	}
	assertCount(`SELECT COUNT(*) FROM service_templates WHERE slug='pista-de-led' AND json_extract(configuration_json,'$.width_meters')=4 AND json_extract(configuration_json,'$.length_meters')=4`, 1)
	assertCount(`SELECT COUNT(*) FROM inventory_items WHERE active=1 AND internal_code IN ('EQP-PANELA-PRESSAO','EQP-CALDEIRAO','EQP-002','EQP-PANELA-MEDIA')`, 4)
	assertCount(`SELECT COUNT(*) FROM calculation_rules WHERE active=1 AND rule_key IN ('pressure_pans','cauldrons','large_pans','medium_pans')`, 4)
	assertCount(`SELECT COUNT(*) FROM inventory_items WHERE active=1 AND internal_code='CUB-CAIXA-PEGADORES' AND stock_quantity=3`, 1)
	assertCount(`SELECT COUNT(*) FROM calculation_rules WHERE active=1 AND rule_key='tongs_box' AND minimum_quantity=1`, 1)
	assertCount(`SELECT COUNT(*) FROM service_template_components component JOIN service_components item ON item.id=component.service_component_id WHERE item.name IN ('Gelo','Canudos','Copos de acrílico') AND component.included=1`, 0)
	assertCount(`SELECT COUNT(*) FROM calculation_rules WHERE rule_key='reusable_cups' AND active=1`, 0)
	assertCount(`SELECT COUNT(*) FROM inventory_items WHERE internal_code='CUB-002' AND active=1`, 0)
	assertCount(`SELECT COUNT(*) FROM events WHERE id=1 AND has_cake=1 AND TRIM(cake_notes)<>''`, 1)

	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	assertCount("SELECT COUNT(*) FROM menu_templates", 14)
	assertCount("SELECT COUNT(*) FROM service_templates", 11)
	assertCount("SELECT COUNT(*) FROM menu_template_versions", 14)
	assertCount("SELECT COUNT(*) FROM service_template_versions", 11)
	assertCount("SELECT COUNT(*) FROM kitchen_cook_box_items WHERE active=1", 33)
}
