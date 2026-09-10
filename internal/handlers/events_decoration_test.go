package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"buffetflow/internal/models"
)

func decorationFormRequest(t *testing.T, values url.Values) *http.Request {
	t.Helper()
	request := httptest.NewRequest("POST", "/events", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := request.ParseForm(); err != nil {
		t.Fatal(err)
	}
	return request
}

func TestApplyEventDecorationFormUsesChosenQuantity(t *testing.T) {
	items := []models.EventDecoration{{DecorationID: 7, Name: "Lanterna", Quantity: 1, AvailableQuantity: 6, AvailabilityTracked: true, Selectable: true}}
	request := decorationFormRequest(t, url.Values{"decoration_ids": {"7"}, "decoration_quantity_7": {"4"}})

	result, err := applyEventDecorationForm(request, items, true)
	if err != nil {
		t.Fatal(err)
	}
	if !result[0].Selected || result[0].Quantity != 4 {
		t.Fatalf("seleção inesperada: %+v", result[0])
	}
}

func TestApplyEventDecorationFormRejectsQuantityAboveAvailability(t *testing.T) {
	items := []models.EventDecoration{{DecorationID: 7, Name: "Lanterna", Quantity: 1, AvailableQuantity: 6, AvailabilityTracked: true, Selectable: true}}
	request := decorationFormRequest(t, url.Values{"decoration_ids": {"7"}, "decoration_quantity_7": {"7"}})

	if _, err := applyEventDecorationForm(request, items, true); err == nil {
		t.Fatal("quantidade acima do estoque deveria ser recusada")
	}
}

func TestParseRentedDecorationFormKeepsNameColorAndQuantity(t *testing.T) {
	request := decorationFormRequest(t, url.Values{
		"rented_decoration_id":       {"12", ""},
		"rented_decoration_name":     {"Cadeira Tiffany", "Arranjo alto"},
		"rented_decoration_color":    {"Dourada", ""},
		"rented_decoration_quantity": {"80", "6"},
	})
	items, err := parseRentedDecorationForm(request)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != 12 || items[0].Name != "Cadeira Tiffany" || items[0].Color != "Dourada" || items[0].Quantity != 80 || items[1].Quantity != 6 {
		t.Fatalf("itens alugados inesperados: %+v", items)
	}
}

func TestGroupChecklistUsesOperationalSectionsAndCompletion(t *testing.T) {
	items := []models.ChecklistItem{
		{Name: "Frango", CategoryName: "Comidas", Status: "separated"},
		{Name: "Copo", CategoryName: "Descartáveis", Status: "separated"},
		{Name: "Prato", CategoryName: "Louças", Status: "separated"},
		{Name: "Jarra", CategoryName: "Itens dos garçons", Status: "separated"},
		{Name: "Garçom", CategoryName: "Itens dos garçons", ItemKind: "outsourced", Status: "pending"},
		{Name: "Arranjo", CategoryName: "Acervo", SourceKey: "decoration:12", Status: "pending"},
	}

	groups := groupChecklist(items)
	if len(groups) != 4 {
		t.Fatalf("group count got %d, want 4: %+v", len(groups), groups)
	}
	wantKeys := []string{"food", "disposable", "material", "decoration"}
	wantCounts := []int{1, 1, 2, 1}
	for index, key := range wantKeys {
		if groups[index].Key != key || len(groups[index].Items) != wantCounts[index] {
			t.Errorf("group %d got key=%q items=%d, want key=%q items=%d", index, groups[index].Key, len(groups[index].Items), key, wantCounts[index])
		}
	}
	if groups[2].Items[0].Name != "Prato" || groups[2].Items[1].Name != "Jarra" {
		t.Errorf("material should keep team equipment and skip staff, got %#v", groups[2].Items)
	}
	if !groups[0].Completed || !groups[1].Completed || !groups[2].Completed || groups[3].Completed {
		t.Errorf("unexpected completed states: food=%v disposable=%v material=%v decoration=%v", groups[0].Completed, groups[1].Completed, groups[2].Completed, groups[3].Completed)
	}
}

func TestChecklistOperationalGroupClassifiesLoadClasses(t *testing.T) {
	cases := []struct {
		item models.ChecklistItem
		want string
	}{
		{item: models.ChecklistItem{Name: "Frango", CategoryName: "Comidas"}, want: "food"},
		{item: models.ChecklistItem{Name: "Suco", CategoryName: "Bebidas", ItemKind: "consumable"}, want: "food"},
		{item: models.ChecklistItem{Name: "Suqueira de vidro", CategoryName: "Bebidas", ItemKind: "reusable"}, want: "material"},
		{item: models.ChecklistItem{Name: "Açúcar", CategoryName: "Mesa de café", ItemKind: "consumable"}, want: "food"},
		{item: models.ChecklistItem{Name: "Molho", SourceKey: "menu-recipe:9", CalculationOrigin: "Receita do cardápio"}, want: "food"},
		{item: models.ChecklistItem{Name: "Copo descartável", CategoryName: "Descartáveis"}, want: "disposable"},
		{item: models.ChecklistItem{Name: "Prato", CategoryName: "Louças"}, want: "material"},
		{item: models.ChecklistItem{Name: "Cuba", CategoryName: "Cubas e utensílios de buffet"}, want: "material"},
		{item: models.ChecklistItem{Name: "Panela", CategoryName: "Equipamentos de cozinha"}, want: "material"},
		{item: models.ChecklistItem{Name: "Caixa", CategoryName: "Recipientes para transporte"}, want: "material"},
		{item: models.ChecklistItem{Name: "Jarra", CategoryName: "Itens dos garçons"}, want: "material"},
		{item: models.ChecklistItem{Name: "Bandeja de garçom", CategoryName: "Itens dos garçons"}, want: "material"},
		{item: models.ChecklistItem{Name: "Xícara", CategoryName: "Mesa de café", ItemKind: "reusable"}, want: "material"},
		{item: models.ChecklistItem{Name: "Garçom", CategoryName: "Itens dos garçons", ItemKind: "outsourced"}, want: ""},
		{item: models.ChecklistItem{Name: "Arranjo", SourceKey: "decoration:12"}, want: "decoration"},
		{item: models.ChecklistItem{Name: "Vaso", CategoryName: "Decoração"}, want: "decoration"},
	}
	for _, test := range cases {
		if got := checklistOperationalGroup(test.item); got != test.want {
			t.Errorf("%s (%s) got %q, want %q", test.item.Name, test.item.CategoryName, got, test.want)
		}
	}
}

func TestChecklistObservationsJoinEditableRowsAndIgnoreBlanks(t *testing.T) {
	values := url.Values{
		"checklist_observations": {"  Levar caixa térmica extra  ", "", "Avisar a equipe da separação"},
	}
	if got, want := checklistObservations(values), "Levar caixa térmica extra\nAvisar a equipe da separação"; got != want {
		t.Fatalf("checklistObservations() = %q, want %q", got, want)
	}
}

func TestChecklistObservationsKeepsLegacyNotesPayload(t *testing.T) {
	values := url.Values{"notes": {"Observação enviada pelo modo offline"}}
	if got, want := checklistObservations(values), values.Get("notes"); got != want {
		t.Fatalf("legacy checklist observation = %q, want %q", got, want)
	}
}

func TestJoinedStringValuesKeepsOfflineObservationRows(t *testing.T) {
	value := []any{"Levar prato extra", " ", "Avisar a equipe"}
	if got, want := joinedStringValues(value), "Levar prato extra\nAvisar a equipe"; got != want {
		t.Fatalf("joinedStringValues() = %q, want %q", got, want)
	}
}

func TestParseEventFormAllowsOptionalClientAndDateTime(t *testing.T) {
	app := &App{location: time.Local}
	request := decorationFormRequest(t, url.Values{
		"event_date":  {"2026-09-20"},
		"starts_time": {"18:00"},
		"guest_count": {"80"},
	})
	event, err := app.parseEventForm(request, 0)
	if err != nil {
		t.Fatal(err)
	}
	if event.ClientName != "" || event.Venue != "" {
		t.Fatalf("optional fields should stay empty: client=%q venue=%q", event.ClientName, event.Venue)
	}
	if event.Name != "Evento" {
		t.Fatalf("fallback name got %q", event.Name)
	}
	if event.WaiterOverride.Valid || event.CoordinatorOverride.Valid || event.LeaderOverride.Valid || event.CoLeaderOverride.Valid {
		t.Fatalf("staff overrides should stay calculated: %+v", event)
	}
	if event.StartsAt.Hour() != 18 || event.EndsAt.Sub(event.StartsAt) != 8*time.Hour {
		t.Fatalf("start/end got %v / %v", event.StartsAt, event.EndsAt)
	}
}

func TestParseMenuModelSectionCustomItemsKeepsMultipleRows(t *testing.T) {
	request := decorationFormRequest(t, url.Values{
		"model_custom_items_section_9": {"Salada extra", "Molho da casa"},
	})
	got := parseMenuModelSectionCustomItems(request)
	if len(got[9]) != 2 || got[9][0] != "Salada extra" || got[9][1] != "Molho da casa" {
		t.Fatalf("custom items got %#v", got)
	}
}
