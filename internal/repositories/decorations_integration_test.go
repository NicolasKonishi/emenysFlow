//go:build private_seeds

package repositories

import (
	"database/sql"
	"testing"
	"time"

	"buffetflow/internal/models"
)

func TestDecorationSelectionUsesColorAndEventAvailability(t *testing.T) {
	store, db, ctx := newModelWorkflowStore(t)
	event, err := store.GetEvent(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, "DELETE FROM inventory_reservations WHERE inventory_item_id=17"); err != nil {
		t.Fatal(err)
	}
	otherEvent := event
	otherEvent.ID = 0
	otherEvent.Name = "Evento concorrente da decoração"
	if err = store.SaveEvent(ctx, &otherEvent, 0); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err = db.ExecContext(ctx, `INSERT INTO inventory_reservations(event_id,inventory_item_id,quantity,starts_at,release_expected_at,status,created_at,updated_at) VALUES(?,17,4,?,?,'active',?,?)`, otherEvent.ID, event.StartsAt.UTC().Format(time.RFC3339), event.EndsAt.UTC().Format(time.RFC3339), now, now); err != nil {
		t.Fatal(err)
	}

	items, err := store.EventDecorationSelectionForWindow(ctx, event.ID, event.StartsAt, event.EndsAt)
	if err != nil {
		t.Fatal(err)
	}
	var vase models.EventDecoration
	for _, item := range items {
		if item.DecorationID == 2 {
			vase = item
			break
		}
	}
	if vase.DecorationID == 0 {
		t.Fatal("Vaso dourado não apareceu no catálogo de decoração")
	}
	if vase.Color != "Dourado" || vase.AvailableQuantity != 6 || vase.Quantity != 1 || !vase.Selectable {
		t.Fatalf("decoração com disponibilidade incorreta: %+v", vase)
	}

	vase.Selected = true
	vase.Quantity = 3
	if err = store.SaveEventDecorations(ctx, event.ID, []models.EventDecoration{vase}); err != nil {
		t.Fatal(err)
	}
	requirements, err := store.EventDecorationRequirements(ctx, event.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(requirements) != 1 || requirements[0].InventoryItemID != 17 || requirements[0].Quantity != 3 {
		t.Fatalf("requisito da decoração incorreto: %+v", requirements)
	}
}

func TestServiceCatalogExcludesInternalDecoration(t *testing.T) {
	store, _, ctx := newModelWorkflowStore(t)
	items, err := store.ListServiceModels(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if item.Category == "decoracao" {
			t.Fatalf("decoração interna ainda apareceu como serviço: %+v", item)
		}
	}
}

func TestDecorationCatalogCanCreateUpdateAndToggleItem(t *testing.T) {
	store, _, ctx := newModelWorkflowStore(t)
	item := models.EventDecoration{
		InventoryItemID: sql.NullInt64{Int64: 17, Valid: true},
		Name:            "Vaso de teste",
		UsageLocation:   "Mesa principal",
		Color:           "Rosé",
		Model:           "Boho",
		Ownership:       "owned",
	}
	if err := store.SaveDecorationCatalogItem(ctx, &item); err != nil {
		t.Fatal(err)
	}
	if item.DecorationID == 0 {
		t.Fatal("item de decoração não recebeu identificador")
	}

	item.Color = "Terracota"
	if err := store.SaveDecorationCatalogItem(ctx, &item); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListDecorationCatalog(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	var found models.EventDecoration
	for _, candidate := range items {
		if candidate.DecorationID == item.DecorationID {
			found = candidate
			break
		}
	}
	if found.DecorationID == 0 || found.Color != "Terracota" || !found.Active {
		t.Fatalf("item do catálogo não foi atualizado: %+v", found)
	}

	if err = store.ToggleDecorationCatalogItem(ctx, item.DecorationID); err != nil {
		t.Fatal(err)
	}
	items, err = store.ListDecorationCatalog(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	foundInactive := false
	for _, candidate := range items {
		if candidate.DecorationID == item.DecorationID {
			foundInactive = true
			if candidate.Active {
				t.Fatalf("item deveria estar inativo: %+v", candidate)
			}
		}
	}
	if !foundInactive {
		t.Fatal("item inativo não apareceu no catálogo completo")
	}
}

func TestEventRentedDecorationsCanBeSavedEditedAndRemoved(t *testing.T) {
	store, _, ctx := newModelWorkflowStore(t)
	items := []models.DecorationCompositionItem{{Name: "Cadeira Tiffany", Color: "Dourada", Quantity: 80, Origin: "rented"}}
	if err := store.SaveEventRentedDecorationItems(ctx, 1, items); err != nil {
		t.Fatal(err)
	}
	stored, err := store.EventRentedDecorationItems(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 1 || stored[0].Name != "Cadeira Tiffany" || stored[0].Color != "Dourada" || stored[0].Quantity != 80 {
		t.Fatalf("item alugado não foi salvo: %+v", stored)
	}

	stored[0].Color = "Rosé"
	stored[0].Quantity = 60
	stored = append(stored, models.DecorationCompositionItem{Name: "Arranjo alto", Quantity: 6, Origin: "rented"})
	if err = store.SaveEventRentedDecorationItems(ctx, 1, stored); err != nil {
		t.Fatal(err)
	}
	stored, err = store.EventRentedDecorationItems(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 2 || stored[0].Color != "Rosé" || stored[0].Quantity != 60 {
		t.Fatalf("itens alugados não foram atualizados: %+v", stored)
	}

	if err = store.SaveEventRentedDecorationItems(ctx, 1, nil); err != nil {
		t.Fatal(err)
	}
	stored, err = store.EventRentedDecorationItems(ctx, 1)
	if err != nil || len(stored) != 0 {
		t.Fatalf("itens alugados não foram removidos: %+v, err=%v", stored, err)
	}
}
