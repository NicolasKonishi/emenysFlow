package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode"

	"buffetflow/internal/models"
)

func (a *App) inventory(writer http.ResponseWriter, request *http.Request) {
	data := a.baseData(request, "Estoque", "inventory")
	data.Query = request.URL.Query().Get("q")
	data.Filter = request.URL.Query().Get("category")
	items, err := a.store.ListStockInventory(request.Context(), data.Query, data.Filter, request.URL.Query().Get("inactive") == "1")
	if err != nil {
		data.Error = databaseErrorMessage(err)
	} else {
		data.Items = items
	}
	data.Categories, _ = a.store.ListCategories(request.Context())
	a.render(writer, request, "inventory", data)
}

func (a *App) inventoryForm(writer http.ResponseWriter, request *http.Request) {
	if err := a.requireAdmin(request); err != nil {
		http.Error(writer, "Acesso restrito ao administrador.", http.StatusForbidden)
		return
	}
	data := a.baseData(request, "Novo item", "inventory")
	data.FormAction = "/inventory"
	data.Item = models.InventoryItem{Unit: "unidade", ItemKind: "reusable", Ownership: "owned", RequiresReturn: true, Active: true}
	if request.PathValue("id") != "" {
		id, err := pathID(request)
		if err != nil {
			http.NotFound(writer, request)
			return
		}
		item, err := a.store.GetInventoryItem(request.Context(), id)
		if err != nil {
			http.NotFound(writer, request)
			return
		}
		data.Title = "Editar item"
		data.Item = item
		data.IsEdit = true
		data.FormAction = fmt.Sprintf("/inventory/%d", id)
	}
	data.Categories, _ = a.store.ListCategories(request.Context())
	data.Locations, _ = a.store.ListLocations(request.Context())
	a.render(writer, request, "inventory_form", data)
}

func (a *App) parseInventoryForm(request *http.Request, id int64) (models.InventoryItem, error) {
	if err := request.ParseForm(); err != nil {
		return models.InventoryItem{}, err
	}
	category := parseOptionalInt(request.FormValue("category_id"))
	if !category.Valid {
		return models.InventoryItem{}, fmt.Errorf("Selecione uma categoria.")
	}
	item := models.InventoryItem{ID: id, Name: strings.TrimSpace(request.FormValue("name")), Description: strings.TrimSpace(request.FormValue("description")), CategoryID: category.Int64,
		Subcategory: strings.TrimSpace(request.FormValue("subcategory")), Unit: strings.TrimSpace(request.FormValue("unit")), StockQuantity: parseFloat(request.FormValue("stock_quantity")),
		MinimumStock: parseFloat(request.FormValue("minimum_stock")), DamagedQuantity: parseFloat(request.FormValue("damaged_quantity")), LocationID: parseOptionalInt(request.FormValue("location_id")),
		InternalCode: strings.TrimSpace(request.FormValue("internal_code")), Barcode: strings.TrimSpace(request.FormValue("barcode")), PhotoURL: strings.TrimSpace(request.FormValue("photo_url")),
		ItemKind: request.FormValue("item_kind"), Ownership: request.FormValue("ownership"), RequiresReturn: boolForm(request.FormValue("requires_return")),
		ReplacementValueCents: int64(parseFloat(request.FormValue("replacement_value")) * 100), Notes: strings.TrimSpace(request.FormValue("notes")), Active: true}
	if id == 0 {
		prefix, err := a.store.InventoryCategoryCodePrefix(request.Context(), category.Int64)
		if err != nil {
			return item, fmt.Errorf("Selecione uma categoria válida.")
		}
		item.InternalCode = buildInventoryInternalCode(prefix, item.Name)
	}
	if item.Name == "" || item.InternalCode == "" || item.Unit == "" {
		return item, fmt.Errorf("Nome, código interno e unidade são obrigatórios.")
	}
	if item.StockQuantity < 0 || item.MinimumStock < 0 || item.DamagedQuantity < 0 || item.DamagedQuantity > item.StockQuantity {
		return item, fmt.Errorf("Revise as quantidades de estoque e itens danificados.")
	}
	return item, nil
}

var inventoryCodeAccents = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u", "ç", "c",
)

func normalizeInventoryCodePart(value string) string {
	value = inventoryCodeAccents.Replace(strings.ToLower(strings.TrimSpace(value)))
	var result strings.Builder
	separator := false
	for _, character := range value {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			if separator && result.Len() > 0 {
				result.WriteByte('-')
			}
			result.WriteRune(character)
			separator = false
			continue
		}
		separator = result.Len() > 0
	}
	return result.String()
}

func normalizeInventoryCodePrefix(value string) string {
	return strings.ToUpper(normalizeInventoryCodePart(value))
}

func buildInventoryInternalCode(prefix, name string) string {
	cleanPrefix := normalizeInventoryCodePrefix(prefix)
	cleanName := normalizeInventoryCodePart(name)
	if cleanPrefix == "" {
		return cleanName
	}
	if cleanName == "" {
		return cleanPrefix
	}
	return cleanPrefix + "-" + cleanName
}

func (a *App) inventoryCreate(writer http.ResponseWriter, request *http.Request) {
	a.saveInventory(writer, request, 0)
}
func (a *App) inventoryUpdate(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	a.saveInventory(writer, request, id)
}
func (a *App) saveInventory(writer http.ResponseWriter, request *http.Request, id int64) {
	if err := a.requireAdmin(request); err != nil {
		http.Error(writer, "Acesso restrito ao administrador.", http.StatusForbidden)
		return
	}
	item, err := a.parseInventoryForm(request, id)
	if err != nil {
		data := a.baseData(request, "Revisar item", "inventory")
		data.Item = item
		data.Error = err.Error()
		data.IsEdit = id > 0
		data.FormAction = "/inventory"
		if id > 0 {
			data.FormAction = fmt.Sprintf("/inventory/%d", id)
		}
		data.Categories, _ = a.store.ListCategories(request.Context())
		data.Locations, _ = a.store.ListLocations(request.Context())
		a.render(writer, request, "inventory_form", data)
		return
	}
	user := currentUser(request)
	err = a.store.SaveInventoryItem(request.Context(), &item, user.ID)
	if err != nil {
		data := a.baseData(request, "Revisar item", "inventory")
		data.Item = item
		data.Error = databaseErrorMessage(err)
		data.IsEdit = id > 0
		data.Categories, _ = a.store.ListCategories(request.Context())
		data.Locations, _ = a.store.ListLocations(request.Context())
		a.render(writer, request, "inventory_form", data)
		return
	}
	a.redirect(writer, request, "/inventory?message="+url.QueryEscape("Item salvo no estoque."), http.StatusSeeOther)
}

func (a *App) inventoryToggle(writer http.ResponseWriter, request *http.Request) {
	if err := a.requireAdmin(request); err != nil {
		http.Error(writer, "Acesso restrito.", http.StatusForbidden)
		return
	}
	id, err := pathID(request)
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	user := currentUser(request)
	err = a.store.ToggleInventoryItem(request.Context(), id, user.ID)
	target := "/inventory?message=" + url.QueryEscape("Status do item atualizado.")
	if err != nil {
		target = "/inventory?type=danger&message=" + url.QueryEscape(databaseErrorMessage(err))
	}
	a.redirect(writer, request, target, http.StatusSeeOther)
}

func (a *App) inventoryMovements(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	item, err := a.store.GetInventoryItem(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	data := a.baseData(r, "Movimentações de estoque", "inventory")
	data.Item = item
	data.Movements, err = a.store.ListInventoryMovements(r.Context(), id)
	if err != nil {
		data.Error = databaseErrorMessage(err)
	}
	a.render(w, r, "inventory_movements", data)
}

func (a *App) inventoryAdjust(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", 403)
		return
	}
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	user := currentUser(r)
	err = a.store.AdjustInventory(r.Context(), id, r.FormValue("movement_type"), parseFloat(r.FormValue("quantity")), strings.TrimSpace(r.FormValue("reason")), user.ID)
	target := fmt.Sprintf("/inventory/%d/movements?message=%s", id, url.QueryEscape("Movimentação registrada."))
	if err != nil {
		target = fmt.Sprintf("/inventory/%d/movements?type=danger&message=%s", id, url.QueryEscape(err.Error()))
	}
	a.redirect(w, r, target, http.StatusSeeOther)
}

var _ = sql.ErrNoRows
