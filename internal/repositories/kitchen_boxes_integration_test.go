package repositories

import "testing"

func TestKitchenCookBoxContentsCRUD(t *testing.T) {
	store, db, ctx := newModelWorkflowStore(t)

	boxes, err := store.ListKitchenCookBoxes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(boxes) != 3 {
		t.Fatalf("boxes got %d, want 3", len(boxes))
	}

	var boxID, itemID int64
	if err := db.QueryRowContext(ctx, `SELECT box.id FROM kitchen_cook_storage_boxes box JOIN kitchen_cooks cook ON cook.id=box.kitchen_cook_id WHERE cook.slug='cris' AND box.box_type='utensils'`).Scan(&boxID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, "SELECT id FROM inventory_items WHERE internal_code='EQP-001'").Scan(&itemID); err != nil {
		t.Fatal(err)
	}
	if err := store.AddKitchenCookBoxItem(ctx, boxID, itemID, 2, "Teste de armazenamento", 0); err != nil {
		t.Fatal(err)
	}

	box, err := store.GetKitchenCookBox(ctx, boxID)
	if err != nil {
		t.Fatal(err)
	}
	var contentID int64
	for _, item := range box.Items {
		if item.InventoryItemID == itemID {
			contentID = item.ID
			if item.Quantity != 2 || item.Notes != "Teste de armazenamento" {
				t.Fatalf("added item got quantity %.0f and notes %q", item.Quantity, item.Notes)
			}
		}
	}
	if contentID == 0 {
		t.Fatal("added inventory item was not found inside box")
	}
	if err := store.UpdateKitchenCookBoxItem(ctx, boxID, contentID, 3, "Quantidade atualizada", 0); err != nil {
		t.Fatal(err)
	}
	box, err = store.GetKitchenCookBox(ctx, boxID)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range box.Items {
		if item.ID == contentID && (item.Quantity != 3 || item.Notes != "Quantidade atualizada") {
			t.Fatalf("updated item got quantity %.0f and notes %q", item.Quantity, item.Notes)
		}
	}
	if err := store.RemoveKitchenCookBoxItem(ctx, boxID, contentID, 0); err != nil {
		t.Fatal(err)
	}
	box, err = store.GetKitchenCookBox(ctx, boxID)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range box.Items {
		if item.ID == contentID {
			t.Fatal("removed item remained active inside box")
		}
	}
	if err := store.AddKitchenCookBoxItem(ctx, boxID, itemID, 4, "Reativado", 0); err != nil {
		t.Fatal(err)
	}
	box, err = store.GetKitchenCookBox(ctx, boxID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range box.Items {
		if item.InventoryItemID == itemID && item.Quantity == 4 && item.Notes == "Reativado" {
			found = true
		}
	}
	if !found {
		t.Fatal("re-added item was not reactivated with its new values")
	}
}

func TestStockListHidesBoxesAndTheirActiveContents(t *testing.T) {
	store, db, ctx := newModelWorkflowStore(t)

	stock, err := store.ListStockInventory(ctx, "", "", true)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range stock {
		if item.Name == "Colher de serviço" || item.Name == "Sal" || item.Name == "Caixa da cozinheira — Cris" {
			t.Fatalf("box-only item %q appeared in the general stock list", item.Name)
		}
	}

	allItems, err := store.ListInventory(ctx, "Colher de serviço", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(allItems) != 1 || allItems[0].Name != "Colher de serviço" {
		t.Fatal("box contents must remain available to internal catalogs and selectors")
	}

	var spoonID int64
	if err := db.QueryRowContext(ctx, "SELECT id FROM inventory_items WHERE name='Colher de serviço'").Scan(&spoonID); err != nil {
		t.Fatal(err)
	}
	rows, err := db.QueryContext(ctx, `SELECT content.kitchen_cook_storage_box_id,content.id
		FROM kitchen_cook_box_items content
		JOIN kitchen_cook_storage_boxes box ON box.id=content.kitchen_cook_storage_box_id
		WHERE content.inventory_item_id=? AND content.active=1 AND box.active=1`, spoonID)
	if err != nil {
		t.Fatal(err)
	}
	type storedContent struct{ boxID, contentID int64 }
	var contents []storedContent
	for rows.Next() {
		var content storedContent
		if err := rows.Scan(&content.boxID, &content.contentID); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		contents = append(contents, content)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	for _, content := range contents {
		if err := store.RemoveKitchenCookBoxItem(ctx, content.boxID, content.contentID, 0); err != nil {
			t.Fatal(err)
		}
	}

	stock, err = store.ListStockInventory(ctx, "Colher de serviço", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(stock) != 1 || stock[0].Name != "Colher de serviço" {
		t.Fatal("item removed from every active box should return to the general stock list")
	}
}
