package models

import "testing"

func TestChecklistItemCanAwaitOnlyForPurchasesAndRentals(t *testing.T) {
	tests := []struct {
		name string
		item ChecklistItem
		want bool
	}{
		{name: "consumable", item: ChecklistItem{ItemKind: "consumable"}, want: true},
		{name: "rented", item: ChecklistItem{ItemKind: "rented"}, want: true},
		{name: "purchase shortage", item: ChecklistItem{ItemKind: "reusable", Shortage: &ChecklistShortage{ResolutionType: "purchase"}}, want: true},
		{name: "rental shortage", item: ChecklistItem{ItemKind: "reusable", Shortage: &ChecklistShortage{ResolutionType: "rental"}}, want: true},
		{name: "owned reusable", item: ChecklistItem{ItemKind: "reusable"}, want: false},
		{name: "unrelated shortage", item: ChecklistItem{ItemKind: "reusable", Shortage: &ChecklistShortage{ResolutionType: "other"}}, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.item.CanAwait(); got != test.want {
				t.Fatalf("CanAwait() = %v, want %v", got, test.want)
			}
		})
	}
}
