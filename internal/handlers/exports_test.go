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
		{Name: "Suco de laranja", CategoryName: "Bebidas", ItemKind: "consumable"},
		{Name: "Copo", CategoryName: "Descartáveis"},
		{Name: "Vaso", CategoryName: "Decoração"},
		{Name: "Jarra", CategoryName: "Itens dos garçons"},
		{Name: "Garçom", CategoryName: "Itens dos garçons", ItemKind: "outsourced"},
	}

	groups := groupChecklistForPDF(items)
	if len(groups) != 4 {
		t.Fatalf("PDF groups got %d, want 4: %#v", len(groups), groups)
	}
	wantKeys := []string{"food", "disposable", "material", "decoration"}
	for index, key := range wantKeys {
		if groups[index].Key != key {
			t.Errorf("PDF group %d got key %q, want %q", index, groups[index].Key, key)
		}
	}
	if len(groups[0].Items) != 2 || groups[0].Items[0].Name != "Frango" || groups[0].Items[1].Name != "Suco de laranja" {
		t.Errorf("food group got %#v", groups[0].Items)
	}
	if len(groups[1].Items) != 1 || groups[1].Items[0].Name != "Copo" {
		t.Errorf("disposable group got %#v", groups[1].Items)
	}
	if len(groups[2].Items) != 2 || groups[2].Items[0].Name != "Prato de jantar" || groups[2].Items[1].Name != "Jarra" {
		t.Errorf("material group got %#v", groups[2].Items)
	}
}

func TestBuildSimplePDFPrintsSeparateMaterialAndFoodDrinkSections(t *testing.T) {
	event := models.Event{ClientName: "Cliente", Name: "Evento", Venue: "Salão", GuestCount: 100, StartsAt: time.Date(2026, 8, 4, 18, 0, 0, 0, time.Local)}
	checklist := models.Checklist{Items: []models.ChecklistItem{
		{Name: "Panela grande", CategoryName: "Equipamentos de cozinha", RequiredQuantity: 5, Unit: "unidade"},
		{Name: "Frango", CategoryName: "Comidas", RequiredQuantity: 50, Unit: "kg"},
		{Name: "Água mineral", CategoryName: "Bebidas", RequiredQuantity: 20, Unit: "garrafa", ItemKind: "consumable"},
		{Name: "Copo", CategoryName: "Descartáveis", RequiredQuantity: 100, Unit: "unidade"},
	}}

	document := string(buildSimplePDF(event, checklist))
	food := strings.Index(document, "(COMIDA)")
	disposable := strings.Index(document, "(DESCARTAVEIS)")
	material := strings.Index(document, "(MATERIAL)")
	if food < 0 || disposable < 0 || material < 0 {
		t.Fatalf("PDF is missing separate section headings: food=%d disposable=%d material=%d", food, disposable, material)
	}
	if !(food < disposable && disposable < material) {
		t.Fatalf("PDF section order is incorrect: food=%d disposable=%d material=%d", food, disposable, material)
	}
}

func TestASCIITextNormalizesUppercasePDFHeadings(t *testing.T) {
	if got := asciiText("DECORAÇÃO E OBSERVAÇÕES"); got != "DECORACAO E OBSERVACOES" {
		t.Fatalf("normalized PDF heading got %q", got)
	}
}
