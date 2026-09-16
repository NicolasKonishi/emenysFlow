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
	a.renderEventChecklist(w, r)
}

func (a *App) renderEventChecklist(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	data := a.baseData(r, "Checklist do evento", operationNav(r))
	event, err := a.store.GetEvent(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	data.Event = event
	data.EventNotes, _ = a.store.ListEventNotes(r.Context(), id)
	checklist, err := a.store.GetChecklistByEvent(r.Context(), id)
	if err == sql.ErrNoRows {
		checklist, err = a.checklist.GenerateTracked(r.Context(), id, "initial_generation", currentUser(r).ID)
	}
	if err != nil {
		data.Error = databaseErrorMessage(err)
		a.render(w, r, "event_show", data)
		return
	}
	shortages, err := a.store.ListEventShortages(r.Context(), id, true)
	if err != nil {
		data.Error = databaseErrorMessage(err)
		a.render(w, r, "event_show", data)
		return
	}
	tab := operationTab(r)
	operationalChecklist, decorationChecklist := splitEventChecklists(checklist)
	operationalShortages, decorationShortages := splitEventShortages(shortages, operationalChecklist, decorationChecklist)
	data.Checklist = filterChecklistForTab(operationalChecklist, operationalShortages, tab)
	data.Groups = groupChecklist(data.Checklist.Items)
	data.DecorationChecklist = filterChecklistForTab(decorationChecklist, decorationShortages, tab)
	data.DecorationGroups = groupChecklist(data.DecorationChecklist.Items)
	data.Shortages = activeShortages(operationalShortages)
	data.DecorationShortages = activeShortages(decorationShortages)
	data.MissingCount = checklistMissingCount(operationalChecklist, operationalShortages) + checklistMissingCount(decorationChecklist, decorationShortages)
	if event.HasDecoration || len(decorationChecklist.Items) > 0 {
		data.DecorationProfile, _ = a.store.GetDecorationProfile(r.Context(), id)
	}
	data.ActiveTab = tab
	a.render(w, r, "event_show", data)
}

func operationTab(r *http.Request) string {
	tab := r.URL.Query().Get("tab")
	if tab != "loading" && tab != "missing" {
		return "separation"
	}
	return tab
}

func activeShortages(shortages []models.ChecklistShortage) []models.ChecklistShortage {
	active := make([]models.ChecklistShortage, 0, len(shortages))
	for _, shortage := range shortages {
		if shortage.Status != "resolved" && shortage.Status != "cancelled" {
			active = append(active, shortage)
		}
	}
	return active
}

func checklistMissingCount(checklist models.Checklist, shortages []models.ChecklistShortage) int {
	active := map[int64]bool{}
	for _, shortage := range shortages {
		if shortage.Status != "resolved" && shortage.Status != "cancelled" {
			active[shortage.ChecklistItemID] = true
		}
	}
	count := 0
	for _, item := range checklist.Items {
		if active[item.ID] || item.LoadingMissingQuantity > 0 {
			count++
		}
	}
	return count
}

func filterChecklistForTab(checklist models.Checklist, shortages []models.ChecklistShortage, tab string) models.Checklist {
	active := map[int64]*models.ChecklistShortage{}
	for index := range shortages {
		if shortages[index].Status != "resolved" && shortages[index].Status != "cancelled" {
			active[shortages[index].ChecklistItemID] = &shortages[index]
		}
	}
	filtered := make([]models.ChecklistItem, 0, len(checklist.Items))
	for _, item := range checklist.Items {
		item.Shortage = active[item.ID]
		if tab == "missing" {
			if item.Shortage != nil || item.LoadingMissingQuantity > 0 {
				filtered = append(filtered, item)
			}
			continue
		}
		// Purchases and rentals remain visible in the separation checklist so
		// the team can explicitly keep them as "Aguardando" until they arrive.
		if item.Shortage != nil && !item.CanAwait() {
			continue
		}
		if tab == "loading" && (item.Status == "not_applicable" || item.SeparatedQuantity+0.0001 < item.RequiredQuantity) {
			continue
		}
		filtered = append(filtered, item)
	}
	checklist.Items = filtered
	return checklist
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
	if shortage.Reason == "" {
		shortage.Reason = "Não tem no estoque"
	}
	if shortage.ResolutionType == "" {
		shortage.ResolutionType = "other"
	}
	if shortage.MissingQuantity <= 0 {
		if required, err := a.store.ChecklistItemRequired(r.Context(), eventID, itemID); err == nil {
			shortage.MissingQuantity = required
		}
	}
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
	if strings.Contains(err.Error(), "version conflict") {
		return "Este item foi atualizado em outro lugar. Atualize a lista e tente de novo."
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
	target := fmt.Sprintf("/events/%d?tab=%s&type=%s&message=%s", eventID, url.QueryEscape(tab), kind, url.QueryEscape(message))
	http.Redirect(w, r, target, http.StatusSeeOther)
}
