package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

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
		{Name: "Prato", CategoryName: "Louças", Status: "separated"},
		{Name: "Arranjo", CategoryName: "Acervo", SourceKey: "decoration:12", Status: "pending"},
		{Name: "Cozinheira", CategoryName: "Equipe", Status: "loaded"},
	}

	groups := groupChecklist(items)
	if len(groups) != 3 {
		t.Fatalf("group count got %d, want 3: %+v", len(groups), groups)
	}
	wantKeys := []string{"material", "decoration", "team"}
	for index, key := range wantKeys {
		if groups[index].Key != key || len(groups[index].Items) != 1 {
			t.Errorf("group %d got key=%q items=%d, want key=%q items=1", index, groups[index].Key, len(groups[index].Items), key)
		}
	}
	if !groups[0].Completed || groups[1].Completed || !groups[2].Completed {
		t.Errorf("unexpected completed states: material=%v decoration=%v team=%v", groups[0].Completed, groups[1].Completed, groups[2].Completed)
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
