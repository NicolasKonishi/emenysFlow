package handlers

import (
	"testing"

	"buffetflow/internal/models"
)

func TestBuildInventoryInternalCode(t *testing.T) {
	tests := []struct {
		prefix string
		name   string
		want   string
	}{
		{prefix: "CUB", name: "Cuba de Réchaud", want: "CUB-cuba-de-rechaud"},
		{prefix: " beb ", name: "Água  com gás 500 ml", want: "BEB-agua-com-gas-500-ml"},
		{prefix: "Louças", name: "Prato / sobremesa", want: "LOUCAS-prato-sobremesa"},
	}

	for _, test := range tests {
		if got := buildInventoryInternalCode(test.prefix, test.name); got != test.want {
			t.Errorf("buildInventoryInternalCode(%q, %q) = %q; want %q", test.prefix, test.name, got, test.want)
		}
	}
}

func TestInventoryItemTabKeepsOperationalStockSeparated(t *testing.T) {
	tests := []struct {
		name string
		item models.InventoryItem
		want string
	}{
		{"food category", models.InventoryItem{CategoryName: "Bebidas", ItemKind: "consumable"}, "food"},
		{"food ingredient", models.InventoryItem{CategoryName: "Ingredientes de receitas", ItemKind: "consumable"}, "food"},
		{"disposable category", models.InventoryItem{CategoryName: "Descartáveis"}, "disposable"},
		{"disposable subcategory", models.InventoryItem{CategoryName: "Mesa de café", Subcategory: "Descartáveis de evento"}, "disposable"},
		{"durable material", models.InventoryItem{CategoryName: "Louças", ItemKind: "reusable"}, "material"},
	}

	for _, test := range tests {
		if got := inventoryItemTab(test.item); got != test.want {
			t.Errorf("%s: inventoryItemTab(%+v) = %q; want %q", test.name, test.item, got, test.want)
		}
	}
}
