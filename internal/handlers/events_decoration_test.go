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
