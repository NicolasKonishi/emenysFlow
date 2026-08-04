package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func kitchenBoxPathID(request *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(request.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid id")
	}
	return id, nil
}

func (a *App) kitchenBoxesPage(writer http.ResponseWriter, request *http.Request) {
	data := a.baseData(request, "Caixas das cozinheiras", "inventory")
	boxes, err := a.store.ListKitchenCookBoxes(request.Context())
	if err != nil {
		data.Error = databaseErrorMessage(err)
	} else {
		data.KitchenBoxes = boxes
	}
	a.render(writer, request, "kitchen_boxes", data)
}

func (a *App) kitchenBoxPage(writer http.ResponseWriter, request *http.Request) {
	boxID, err := kitchenBoxPathID(request, "boxID")
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	data := a.baseData(request, "Conteúdo da caixa", "inventory")
	data.KitchenBox, err = a.store.GetKitchenCookBox(request.Context(), boxID)
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	data.BoxInventoryOptions, _ = a.store.ListKitchenBoxContentOptions(request.Context(), boxID)
	a.render(writer, request, "kitchen_box", data)
}

func (a *App) kitchenBoxItemAdd(writer http.ResponseWriter, request *http.Request) {
	if err := a.requireAdmin(request); err != nil {
		http.Error(writer, "Acesso restrito ao administrador.", http.StatusForbidden)
		return
	}
	boxID, err := kitchenBoxPathID(request, "boxID")
	if err != nil || request.ParseForm() != nil {
		http.NotFound(writer, request)
		return
	}
	inventoryItemID, parseErr := strconv.ParseInt(request.FormValue("inventory_item_id"), 10, 64)
	quantity := parseFloat(request.FormValue("quantity"))
	if parseErr != nil || inventoryItemID <= 0 || quantity <= 0 {
		a.redirect(writer, request, fmt.Sprintf("/inventory/kitchen-boxes/box/%d?type=danger&message=%s", boxID, url.QueryEscape("Selecione o item e informe uma quantidade válida.")), http.StatusSeeOther)
		return
	}
	user := currentUser(request)
	err = a.store.AddKitchenCookBoxItem(request.Context(), boxID, inventoryItemID, quantity, strings.TrimSpace(request.FormValue("notes")), user.ID)
	a.redirectKitchenBoxResult(writer, request, boxID, err, "Item adicionado à caixa.")
}

func (a *App) kitchenBoxItemUpdate(writer http.ResponseWriter, request *http.Request) {
	if err := a.requireAdmin(request); err != nil {
		http.Error(writer, "Acesso restrito ao administrador.", http.StatusForbidden)
		return
	}
	boxID, err := kitchenBoxPathID(request, "boxID")
	contentID, contentErr := kitchenBoxPathID(request, "itemID")
	if err != nil || contentErr != nil || request.ParseForm() != nil {
		http.NotFound(writer, request)
		return
	}
	quantity := parseFloat(request.FormValue("quantity"))
	user := currentUser(request)
	err = a.store.UpdateKitchenCookBoxItem(request.Context(), boxID, contentID, quantity, strings.TrimSpace(request.FormValue("notes")), user.ID)
	a.redirectKitchenBoxResult(writer, request, boxID, err, "Conteúdo da caixa atualizado.")
}

func (a *App) kitchenBoxItemRemove(writer http.ResponseWriter, request *http.Request) {
	if err := a.requireAdmin(request); err != nil {
		http.Error(writer, "Acesso restrito ao administrador.", http.StatusForbidden)
		return
	}
	boxID, err := kitchenBoxPathID(request, "boxID")
	contentID, contentErr := kitchenBoxPathID(request, "itemID")
	if err != nil || contentErr != nil {
		http.NotFound(writer, request)
		return
	}
	user := currentUser(request)
	err = a.store.RemoveKitchenCookBoxItem(request.Context(), boxID, contentID, user.ID)
	a.redirectKitchenBoxResult(writer, request, boxID, err, "Item removido da caixa.")
}

func (a *App) redirectKitchenBoxResult(writer http.ResponseWriter, request *http.Request, boxID int64, err error, successMessage string) {
	target := fmt.Sprintf("/inventory/kitchen-boxes/box/%d?message=%s", boxID, url.QueryEscape(successMessage))
	if err != nil {
		target = fmt.Sprintf("/inventory/kitchen-boxes/box/%d?type=danger&message=%s", boxID, url.QueryEscape(databaseErrorMessage(err)))
	}
	a.redirect(writer, request, target, http.StatusSeeOther)
}
