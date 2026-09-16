package handlers

import (
	"slices"
	"testing"

	"buffetflow/internal/models"
)

func TestSplitEventChecklistsKeepsDecorationSeparateFromOperationalWork(t *testing.T) {
	checklist := models.Checklist{
		ID:      91,
		EventID: 37,
		Version: 4,
		Items: []models.ChecklistItem{
			{ID: 1, Name: "Travessa", SourceKey: "rule:8", CategoryName: "Louças"},
			{ID: 2, Name: "Arranjo rastreado", SourceKey: "decoration:12", CategoryName: "Acervo"},
			{ID: 3, Name: "Bolo fake", SourceKey: "decoration-composition:24", ItemKind: "decoration"},
			{ID: 4, Name: "Cadeira Tiffany", SourceKey: "decoration-rental:31", ItemKind: "rented"},
			{ID: 5, Name: "Painel sem estoque", CategoryName: "Decoração"},
			{ID: 6, Name: "Peça planejada", ItemKind: "decoration"},
			{ID: 7, Name: "Garçom", SourceKey: "rule:99", ItemKind: "outsourced"},
			{ID: 8, Name: "Montador", SourceKey: "decoration:99", CategoryName: "Decoração", ItemKind: "outsourced"},
			{ID: 9, Name: "Auxiliar", SourceKey: "rule:100", Unit: "profissional"},
			{ID: 10, Name: "Coordenadora", SourceKey: "rule:101", CategoryName: "Equipe"},
		},
	}

	operational, decoration := splitEventChecklists(checklist)

	if operational.ID != checklist.ID || operational.EventID != checklist.EventID || operational.Version != checklist.Version {
		t.Fatalf("operational checklist lost its identity: %#v", operational)
	}
	if decoration.ID != checklist.ID || decoration.EventID != checklist.EventID || decoration.Version != checklist.Version {
		t.Fatalf("decoration checklist lost its identity: %#v", decoration)
	}
	if got, want := checklistItemIDs(operational.Items), []int64{1}; !slices.Equal(got, want) {
		t.Errorf("operational item IDs = %v, want %v", got, want)
	}
	if got, want := checklistItemIDs(decoration.Items), []int64{2, 3, 4, 5, 6}; !slices.Equal(got, want) {
		t.Errorf("decoration item IDs = %v, want %v", got, want)
	}
}

func TestChecklistIsDecorationRecognizesEveryGeneratedDecorationSource(t *testing.T) {
	cases := []struct {
		name string
		item models.ChecklistItem
	}{
		{name: "tracked event decoration", item: models.ChecklistItem{SourceKey: "decoration:12"}},
		{name: "composition item", item: models.ChecklistItem{SourceKey: "decoration-composition:24"}},
		{name: "rented composition item", item: models.ChecklistItem{SourceKey: "decoration-rental:31"}},
		{name: "planned decoration fallback", item: models.ChecklistItem{ItemKind: "decoration"}},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if !checklistIsDecoration(test.item) {
				t.Fatalf("%#v was not recognized as decoration", test.item)
			}
		})
	}
}

func TestSplitEventShortagesKeepsStaffOutOfBothChecklists(t *testing.T) {
	operational := models.Checklist{Items: []models.ChecklistItem{{ID: 1, Name: "Travessa"}}}
	decoration := models.Checklist{Items: []models.ChecklistItem{{ID: 2, Name: "Vaso", SourceKey: "decoration:2"}}}
	shortages := []models.ChecklistShortage{
		{ID: 10, ChecklistItemID: 1},
		{ID: 20, ChecklistItemID: 2},
		{ID: 30, ChecklistItemID: 3}, // equipe, excluída das duas listas
	}

	operationalShortages, decorationShortages := splitEventShortages(shortages, operational, decoration)
	if got, want := checklistShortageIDs(operationalShortages), []int64{10}; !slices.Equal(got, want) {
		t.Errorf("operational shortages = %v, want %v", got, want)
	}
	if got, want := checklistShortageIDs(decorationShortages), []int64{20}; !slices.Equal(got, want) {
		t.Errorf("decoration shortages = %v, want %v", got, want)
	}
}

func checklistItemIDs(items []models.ChecklistItem) []int64 {
	ids := make([]int64, len(items))
	for index, item := range items {
		ids[index] = item.ID
	}
	return ids
}

func checklistShortageIDs(shortages []models.ChecklistShortage) []int64 {
	ids := make([]int64, len(shortages))
	for index, shortage := range shortages {
		ids[index] = shortage.ID
	}
	return ids
}
