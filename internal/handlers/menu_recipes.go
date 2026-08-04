package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"buffetflow/internal/models"
)

func recipeIngredientFromRequest(request *http.Request, menuItemID, ingredientID int64) (models.MenuItemIngredient, error) {
	if err := request.ParseForm(); err != nil {
		return models.MenuItemIngredient{}, err
	}
	inventoryItemID, err := strconv.ParseInt(request.FormValue("inventory_item_id"), 10, 64)
	if ingredientID > 0 && strings.TrimSpace(request.FormValue("inventory_item_id")) == "" {
		inventoryItemID = 0 // Update keeps the existing linked inventory item.
		err = nil
	}
	item := models.MenuItemIngredient{
		ID:              ingredientID,
		MenuItemID:      menuItemID,
		InventoryItemID: inventoryItemID,
		CalculationType: request.FormValue("calculation_type"),
		Quantity:        parseFloat(request.FormValue("quantity")),
		PeopleDivisor:   parseFloat(request.FormValue("people_divisor")),
		Notes:           strings.TrimSpace(request.FormValue("notes")),
		Active:          true,
	}
	if err != nil || (ingredientID == 0 && item.InventoryItemID <= 0) || item.Quantity <= 0 {
		return item, fmt.Errorf("Selecione o ingrediente e informe uma quantidade válida.")
	}
	if item.CalculationType == "fixed" && item.PeopleDivisor <= 0 {
		item.PeopleDivisor = 1
	}
	if item.PeopleDivisor <= 0 {
		return item, fmt.Errorf("O divisor de pessoas deve ser maior que zero.")
	}
	return item, nil
}

func (a *App) menuItemIngredientAdd(writer http.ResponseWriter, request *http.Request) {
	if err := a.requireAdmin(request); err != nil {
		http.Error(writer, "Acesso restrito.", http.StatusForbidden)
		return
	}
	menuItemID, err := pathID(request)
	var item models.MenuItemIngredient
	if err == nil {
		item, err = recipeIngredientFromRequest(request, menuItemID, 0)
	}
	if err == nil {
		err = a.store.AddMenuItemIngredient(request.Context(), item)
	}
	a.redirectMenuRecipeResult(writer, request, menuItemID, err, "Ingrediente adicionado à receita.")
}

func (a *App) menuItemIngredientUpdate(writer http.ResponseWriter, request *http.Request) {
	if err := a.requireAdmin(request); err != nil {
		http.Error(writer, "Acesso restrito.", http.StatusForbidden)
		return
	}
	menuItemID, err := pathID(request)
	ingredientID, ingredientErr := strconv.ParseInt(request.PathValue("ingredientID"), 10, 64)
	if err == nil && (ingredientErr != nil || ingredientID <= 0) {
		err = ingredientErr
	}
	var item models.MenuItemIngredient
	if err == nil {
		item, err = recipeIngredientFromRequest(request, menuItemID, ingredientID)
	}
	if err == nil {
		err = a.store.UpdateMenuItemIngredient(request.Context(), item)
	}
	a.redirectMenuRecipeResult(writer, request, menuItemID, err, "Ingrediente da receita atualizado.")
}

func (a *App) menuItemIngredientRemove(writer http.ResponseWriter, request *http.Request) {
	if err := a.requireAdmin(request); err != nil {
		http.Error(writer, "Acesso restrito.", http.StatusForbidden)
		return
	}
	menuItemID, err := pathID(request)
	ingredientID, ingredientErr := strconv.ParseInt(request.PathValue("ingredientID"), 10, 64)
	if err == nil && (ingredientErr != nil || ingredientID <= 0) {
		err = ingredientErr
	}
	if err == nil {
		err = a.store.RemoveMenuItemIngredient(request.Context(), menuItemID, ingredientID)
	}
	a.redirectMenuRecipeResult(writer, request, menuItemID, err, "Ingrediente removido da receita.")
}

func (a *App) redirectMenuRecipeResult(writer http.ResponseWriter, request *http.Request, menuItemID int64, err error, message string) {
	target := fmt.Sprintf("/catalog/items/%d/edit?message=%s#recipe", menuItemID, url.QueryEscape(message))
	if err != nil {
		target = fmt.Sprintf("/catalog/items/%d/edit?type=danger&message=%s#recipe", menuItemID, url.QueryEscape(databaseErrorMessage(err)))
	}
	a.redirect(writer, request, target, http.StatusSeeOther)
}
