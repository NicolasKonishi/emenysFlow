package handlers

import (
	"net/http/httptest"
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
	wantKeys := []string{"material", "disposable", "food", "decoration"}
	for index, key := range wantKeys {
		if groups[index].Key != key {
			t.Errorf("PDF group %d got key %q, want %q", index, groups[index].Key, key)
		}
	}
	if len(groups[0].Items) != 2 || groups[0].Items[0].Name != "Prato de jantar" || groups[0].Items[1].Name != "Jarra" {
		t.Errorf("material group got %#v", groups[0].Items)
	}
	if len(groups[1].Items) != 1 || groups[1].Items[0].Name != "Copo" {
		t.Errorf("disposable group got %#v", groups[1].Items)
	}
	if len(groups[2].Items) != 2 || groups[2].Items[0].Name != "Frango" || groups[2].Items[1].Name != "Suco de laranja" {
		t.Errorf("food group got %#v", groups[2].Items)
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
	if !(material < disposable && disposable < food) {
		t.Fatalf("PDF section order is incorrect: food=%d disposable=%d material=%d", food, disposable, material)
	}
}

func TestASCIITextNormalizesUppercasePDFHeadings(t *testing.T) {
	if got := asciiText("DECORAÇÃO E OBSERVAÇÕES"); got != "DECORACAO E OBSERVACOES" {
		t.Fatalf("normalized PDF heading got %q", got)
	}
}

func TestChecklistPDFKindAndItemsKeepDecorationSeparate(t *testing.T) {
	request := httptest.NewRequest("GET", "/events/12/export.pdf?checklist=decoration", nil)
	if kind := checklistPDFKindFromRequest(request); kind != checklistPDFDecoration {
		t.Fatalf("PDF kind got %q, want decoration", kind)
	}
	items := []models.ChecklistItem{
		{Name: "Prato", CategoryName: "Louças"},
		{Name: "Vaso", SourceKey: "decoration-composition:4"},
		{Name: "Arranjo", CategoryName: "Decoração"},
	}
	operational := checklistItemsForPDF(items, checklistPDFOperational)
	decoration := checklistItemsForPDF(items, checklistPDFDecoration)
	if len(operational) != 1 || operational[0].Name != "Prato" {
		t.Fatalf("operational PDF items got %#v", operational)
	}
	if len(decoration) != 2 || decoration[0].Name != "Vaso" || decoration[1].Name != "Arranjo" {
		t.Fatalf("decoration PDF items got %#v", decoration)
	}
}

func TestDecorationPDFIncludesNotesAndReferenceCaptions(t *testing.T) {
	event := models.Event{ClientName: "Cliente", Name: "Evento", Venue: "Salão", GuestCount: 100, StartsAt: time.Date(2026, 8, 4, 18, 0, 0, 0, time.Local), Notes: "Levar velas extras."}
	notes := []models.EventNote{{
		Title:   "Alinhamento da cerimônia",
		Content: "Tapete será montado antes da chegada dos convidados.",
		Photos:  []models.EventNotePhoto{{Caption: "Foto do corredor"}},
	}}
	profile := models.DecorationProfile{
		PrimaryColors: "Azul e dourado",
		Photos:        []models.ReferencePhoto{{Caption: "Painel principal"}},
		Compositions: []models.DecorationComposition{{
			Name:   "Mesa do bolo",
			Photos: []models.ReferencePhoto{{Caption: "Referência da mesa"}},
			Items: []models.DecorationCompositionItem{{
				Name: "Vaso dourado", Quantity: 2, Color: "Dourado", ArrangementKind: "natural",
			}},
		}},
	}
	document := string(buildChecklistPDF(event, models.Checklist{Items: []models.ChecklistItem{{Name: "Vaso dourado", SourceKey: "decoration-composition:5", RequiredQuantity: 2, Unit: "unidade"}}}, notes, profile, checklistPDFDecoration))
	for _, expected := range []string{
		"(CHECKLIST DE DECORACAO)",
		"(OBSERVACOES DA CHECKLIST)",
		"(ANOTACOES DO EVENTO)",
		"(Foto de referencia: Foto do corredor)",
		"(CONFIGURACAO DA DECORACAO)",
		"(Foto de referencia: Painel principal)",
		"(Foto de referencia: Referencia da mesa)",
	} {
		if !strings.Contains(document, expected) {
			t.Errorf("PDF is missing %q", expected)
		}
	}
}
