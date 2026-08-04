package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"buffetflow/internal/models"
)

func (a *App) decorationCatalogPage(w http.ResponseWriter, r *http.Request) {
	data := a.baseData(r, "Decorações", "decorations")
	var err error
	data.Decorations, err = a.store.ListDecorationCatalog(r.Context(), r.URL.Query().Get("inactive") == "1")
	allItems, inventoryErr := a.store.ListInventory(r.Context(), "", "", false)
	for _, item := range allItems {
		if item.CategoryName == "Decoração" {
			data.Items = append(data.Items, item)
		}
	}
	if err == nil {
		err = inventoryErr
	}
	if err != nil {
		data.Error = databaseErrorMessage(err)
	}
	a.render(w, r, "decorations_catalog", data)
}

func parseDecorationCatalogForm(r *http.Request, id int64) (models.EventDecoration, error) {
	if err := r.ParseForm(); err != nil {
		return models.EventDecoration{}, err
	}
	item := models.EventDecoration{
		DecorationID:    id,
		Name:            strings.TrimSpace(r.FormValue("name")),
		UsageLocation:   strings.TrimSpace(r.FormValue("usage_location")),
		Color:           strings.TrimSpace(r.FormValue("color")),
		Model:           strings.TrimSpace(r.FormValue("model")),
		Ownership:       r.FormValue("ownership"),
		RentalCompany:   strings.TrimSpace(r.FormValue("rental_company")),
		PhotoURL:        strings.TrimSpace(r.FormValue("photo_url")),
		Notes:           strings.TrimSpace(r.FormValue("notes")),
		InventoryItemID: parseOptionalInt(r.FormValue("inventory_item_id")),
		Active:          true,
	}
	if item.Name == "" || !item.InventoryItemID.Valid {
		return item, fmt.Errorf("Informe o nome e vincule um item do estoque de decoração.")
	}
	if item.Ownership != "owned" && item.Ownership != "rented" {
		return item, fmt.Errorf("Informe se a peça é própria ou alugada.")
	}
	return item, nil
}

func (a *App) decorationCatalogCreate(w http.ResponseWriter, r *http.Request) {
	a.saveDecorationCatalogItem(w, r, 0)
}

func (a *App) decorationCatalogUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	a.saveDecorationCatalogItem(w, r, id)
}

func (a *App) saveDecorationCatalogItem(w http.ResponseWriter, r *http.Request, id int64) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito ao administrador.", http.StatusForbidden)
		return
	}
	item, err := parseDecorationCatalogForm(r, id)
	if err == nil {
		err = a.store.SaveDecorationCatalogItem(r.Context(), &item)
	}
	target := "/decorations?message=" + url.QueryEscape("Item de decoração salvo.")
	if err != nil {
		target = "/decorations?type=danger&message=" + url.QueryEscape(err.Error())
	}
	a.redirect(w, r, target, http.StatusSeeOther)
}

func (a *App) decorationCatalogToggle(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito ao administrador.", http.StatusForbidden)
		return
	}
	id, err := pathID(r)
	if err == nil {
		err = a.store.ToggleDecorationCatalogItem(r.Context(), id)
	}
	target := "/decorations?inactive=1&message=" + url.QueryEscape("Status da decoração atualizado.")
	if err != nil {
		target = "/decorations?inactive=1&type=danger&message=" + url.QueryEscape(databaseErrorMessage(err))
	}
	a.redirect(w, r, target, http.StatusSeeOther)
}
