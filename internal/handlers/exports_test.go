package handlers

import (
	"strings"
	"testing"
	"time"

	"buffetflow/internal/models"
)

func TestGroupChecklistForPDFSeparatesFoodAndDrinkFromMaterial(t *testing.T) {
	items := []models.ChecklistItem{
		{Name: "Prato de jantar", CategoryName: "Louças"},
		{Name: "Frango", CategoryName: "Comidas"},
		{Name: "Suco de laranja", CategoryName: "Bebidas"},
		{Name: "Vaso", CategoryName: "Decoração"},
		{Name: "Garçom", CategoryName: "Itens dos garçons"},
	}

	groups := groupChecklistForPDF(items)
	if len(groups) != 4 {
		t.Fatalf("PDF groups got %d, want 4: %#v", len(groups), groups)
	}
	wantKeys := []string{"material", "food_drink", "decoration", "team"}
	for index, key := range wantKeys {
		if groups[index].Key != key {
			t.Errorf("PDF group %d got key %q, want %q", index, groups[index].Key, key)
		}
	}
	if len(groups[0].Items) != 1 || groups[0].Items[0].Name != "Prato de jantar" {
		t.Errorf("material group got %#v", groups[0].Items)
	}
	if len(groups[1].Items) != 2 || groups[1].Items[0].Name != "Frango" || groups[1].Items[1].Name != "Suco de laranja" {
		t.Errorf("food/drink group got %#v", groups[1].Items)
	}
}

func TestBuildSimplePDFPrintsSeparateMaterialAndFoodDrinkSections(t *testing.T) {
	event := models.Event{ClientName: "Cliente", Name: "Evento", Venue: "Salão", GuestCount: 100, StartsAt: time.Date(2026, 8, 4, 18, 0, 0, 0, time.Local)}
	checklist := models.Checklist{Items: []models.ChecklistItem{
		{Name: "Panela grande", CategoryName: "Equipamentos de cozinha", RequiredQuantity: 5, Unit: "unidade"},
		{Name: "Frango", CategoryName: "Comidas", RequiredQuantity: 50, Unit: "kg"},
		{Name: "Água mineral", CategoryName: "Bebidas", RequiredQuantity: 20, Unit: "garrafa"},
	}}

	document := string(buildSimplePDF(event, checklist))
	material := strings.Index(document, "(MATERIAL)")
	foodDrink := strings.Index(document, "(COMIDA / BEBIDA)")
	if material < 0 || foodDrink < 0 {
		t.Fatalf("PDF is missing separate section headings: material=%d food/drink=%d", material, foodDrink)
	}
	if material >= foodDrink {
		t.Fatalf("PDF section order is incorrect: material=%d food/drink=%d", material, foodDrink)
	}
}

func TestASCIITextNormalizesUppercasePDFHeadings(t *testing.T) {
	if got := asciiText("DECORAÇÃO E OBSERVAÇÕES"); got != "DECORACAO E OBSERVACOES" {
		t.Fatalf("normalized PDF heading got %q", got)
	}
}
