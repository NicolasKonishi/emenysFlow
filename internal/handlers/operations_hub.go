package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"buffetflow/internal/models"
)

func (a *App) operationHub(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	event, err := a.store.GetEvent(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	checklist, err := a.store.GetChecklistByEvent(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	shortages, err := a.store.ListEventShortages(r.Context(), id, true)
	if err != nil {
		http.Error(w, databaseErrorMessage(err), 500)
		return
	}
	activeShortages := map[int64]*models.ChecklistShortage{}
	for index := range shortages {
		if shortages[index].Status != "resolved" && shortages[index].Status != "cancelled" {
			activeShortages[shortages[index].ChecklistItemID] = &shortages[index]
		}
	}
	tab := r.URL.Query().Get("tab")
	if tab != "loading" && tab != "missing" {
		tab = "separation"
	}
	filtered := make([]models.ChecklistItem, 0, len(checklist.Items))
	for _, item := range checklist.Items {
		item.Shortage = activeShortages[item.ID]
		if tab == "missing" {
			continue
		}
		if item.Shortage != nil {
			continue
		}
		if tab == "loading" && item.SeparatedQuantity+0.0001 < item.RequiredQuantity {
			continue
		}
		filtered = append(filtered, item)
	}
	checklist.Items = filtered
	data := a.baseData(r, "Operação do evento", operationNav(r))
	data.Event = event
	data.Checklist = checklist
	data.Groups = groupChecklist(checklist.Items)
	data.Shortages = shortages
	data.ActiveTab = tab
	data.Categories, _ = a.store.ListCategories(r.Context())
	data.Items, _ = a.store.ListInventory(r.Context(), "", "", false)
	data.StaffSummary, _ = a.checklist.StaffSummary(r.Context(), id)
	a.render(w, r, "operations_hub", data)
}

func (a *App) operationQuantity(w http.ResponseWriter, r *http.Request) {
	eventID, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	itemID, err := strconv.ParseInt(r.PathValue("itemID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	stage := r.FormValue("stage")
	quantity := parseFloat(r.FormValue("quantity"))
	version, _ := strconv.Atoi(r.FormValue("version"))
	user := currentUser(r)
	newVersion, err := a.store.SaveOperationalQuantity(r.Context(), eventID, itemID, stage, quantity, strings.TrimSpace(r.FormValue("notes")), user.ID, version)
	if wantsJSON(r) {
		writeJSON(w, statusForOperationError(err), map[string]any{"ok": err == nil, "version": newVersion, "error": errorText(err)})
		return
	}
	tab := "separation"
	if stage == "loading" {
		tab = "loading"
	}
	operationRedirect(w, r, eventID, tab, err, "Quantidade atualizada.")
}

func (a *App) operationShortage(w http.ResponseWriter, r *http.Request) {
	eventID, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	itemID, err := strconv.ParseInt(r.PathValue("itemID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	shortage := models.ChecklistShortage{EventID: eventID, ChecklistItemID: itemID, MissingQuantity: parseFloat(r.FormValue("missing_quantity")), Reason: strings.TrimSpace(r.FormValue("reason")), ResolutionType: r.FormValue("resolution_type"), ResponsibleName: strings.TrimSpace(r.FormValue("responsible_name")), SupplierName: strings.TrimSpace(r.FormValue("supplier_name")), Notes: strings.TrimSpace(r.FormValue("notes"))}
	if raw := r.FormValue("due_at"); raw != "" {
		shortage.DueAt, _ = time.ParseInLocation("2006-01-02T15:04", raw, a.location)
	}
	if value := parseFloat(r.FormValue("estimated_cost")); value >= 0 && r.FormValue("estimated_cost") != "" {
		shortage.EstimatedCostCents = sql.NullInt64{Int64: int64(value*100 + 0.5), Valid: true}
	}
	err = a.store.SaveChecklistShortage(r.Context(), shortage, currentUser(r).ID)
	if wantsJSON(r) {
		writeJSON(w, statusForOperationError(err), map[string]any{"ok": err == nil, "error": errorText(err)})
		return
	}
	operationRedirect(w, r, eventID, "missing", err, "Item marcado como faltando.")
}

func (a *App) operationShortageStatus(w http.ResponseWriter, r *http.Request) {
	eventID, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	shortageID, err := strconv.ParseInt(r.PathValue("shortageID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	err = a.store.UpdateShortageStatus(r.Context(), eventID, shortageID, r.FormValue("status"), r.FormValue("destination"), strings.TrimSpace(r.FormValue("notes")), currentUser(r).ID)
	if wantsJSON(r) {
		writeJSON(w, statusForOperationError(err), map[string]any{"ok": err == nil, "error": errorText(err)})
		return
	}
	operationRedirect(w, r, eventID, "missing", err, "Situação da falta atualizada.")
}

func (a *App) operationManualItem(w http.ResponseWriter, r *http.Request) {
	eventID, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	inventoryID := parseOptionalInt(r.FormValue("inventory_item_id"))
	categoryID, _ := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
	item := models.ChecklistItem{InventoryItemID: inventoryID, CategoryID: categoryID, Name: strings.TrimSpace(r.FormValue("name")), Unit: strings.TrimSpace(r.FormValue("unit")), RequiredQuantity: parseFloat(r.FormValue("quantity")), Notes: strings.TrimSpace(r.FormValue("notes")), ItemKind: "reusable"}
	id, err := a.store.AddManualChecklistItem(r.Context(), eventID, item, currentUser(r).ID)
	if wantsJSON(r) {
		writeJSON(w, statusForOperationError(err), map[string]any{"ok": err == nil, "entity_id": id, "error": errorText(err)})
		return
	}
	operationRedirect(w, r, eventID, "separation", err, "Item manual adicionado.")
}

func wantsJSON(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/json") || r.Header.Get("X-BuffetFlow-Client") != ""
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
func statusForOperationError(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if strings.Contains(err.Error(), "version conflict") {
		return http.StatusConflict
	}
	return http.StatusBadRequest
}
func operationRedirect(w http.ResponseWriter, r *http.Request, eventID int64, tab string, err error, success string) {
	kind := "success"
	message := success
	if err != nil {
		kind = "danger"
		message = databaseErrorMessage(err)
	}
	target := fmt.Sprintf("/events/%d/operation?tab=%s&type=%s&message=%s", eventID, url.QueryEscape(tab), kind, url.QueryEscape(message))
	http.Redirect(w, r, target, http.StatusSeeOther)
}
