package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"buffetflow/internal/models"
)

var allowedOperationStages = map[string]bool{"separating": true, "checking": true, "loading": true, "in_progress": true}

func (a *App) operationPage(w http.ResponseWriter, r *http.Request) {
	eventID, err := pathID(r)
	stage := r.PathValue("stage")
	if err != nil || !allowedOperationStages[stage] {
		http.NotFound(w, r)
		return
	}
	event, err := a.store.GetEvent(r.Context(), eventID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	data := a.baseData(r, operationTitle(stage), operationNav(r))
	data.Event = event
	data.Operation, err = a.store.GetEventOperation(r.Context(), eventID, stage)
	if err != nil {
		data.Error = databaseErrorMessage(err)
	}
	if data.Operation.ResponsibleName == "" {
		data.Operation.ResponsibleName = currentUser(r).Name
	}
	data.Checklist, err = a.store.OperationChecklist(r.Context(), eventID, stage)
	if err != nil {
		data.Error = databaseErrorMessage(err)
	}
	data.Groups = groupChecklist(data.Checklist.Items)
	a.render(w, r, "operation", data)
}

func (a *App) operationSave(w http.ResponseWriter, r *http.Request) {
	eventID, err := pathID(r)
	stage := r.PathValue("stage")
	if err != nil || !allowedOperationStages[stage] {
		http.NotFound(w, r)
		return
	}
	if err = r.ParseForm(); err != nil {
		http.Error(w, "Dados inválidos.", 400)
		return
	}
	checklist, err := a.store.OperationChecklist(r.Context(), eventID, stage)
	if err != nil {
		http.Error(w, databaseErrorMessage(err), 500)
		return
	}
	quantities := map[int64]float64{}
	for _, item := range checklist.Items {
		quantities[item.ID] = parseFloat(r.FormValue("quantity_" + strconv.FormatInt(item.ID, 10)))
	}
	occurred := time.Now().In(a.location)
	if raw := r.FormValue("occurred_at"); raw != "" {
		if parsed, e := time.ParseInLocation("2006-01-02T15:04", raw, a.location); e == nil {
			occurred = parsed
		}
	}
	operation := models.EventOperation{EventID: eventID, Stage: stage, ResponsibleName: strings.TrimSpace(r.FormValue("responsible_name")), Vehicle: strings.TrimSpace(r.FormValue("vehicle")), Notes: strings.TrimSpace(r.FormValue("notes")), PhotoURL: strings.TrimSpace(r.FormValue("photo_url")), OccurredAt: occurred}
	user := currentUser(r)
	err = a.store.SaveEventOperation(r.Context(), eventID, operation, quantities, user.ID)
	if err != nil {
		target := fmt.Sprintf("/events/%d/operation/%s?type=danger&message=%s", eventID, stage, url.QueryEscape(databaseErrorMessage(err)))
		a.redirect(w, r, target, http.StatusSeeOther)
		return
	}
	next := operationNext(stage, eventID)
	a.redirect(w, r, next+"?message="+url.QueryEscape(operationTitle(stage)+" registrada."), http.StatusSeeOther)
}

func (a *App) mobileLoadingItem(w http.ResponseWriter, r *http.Request) {
	eventID, err := pathID(r)
	itemID, itemErr := strconv.ParseInt(r.PathValue("itemID"), 10, 64)
	if err != nil || itemErr != nil || itemID <= 0 {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Dados inválidos.", http.StatusBadRequest)
		return
	}
	decision := strings.TrimSpace(r.FormValue("decision"))
	missingQuantity := parseFloat(r.FormValue("missing_quantity"))
	user := currentUser(r)
	loadedQuantity, err := a.store.UpdateMobileLoadingItem(r.Context(), eventID, itemID, decision, missingQuantity, user.ID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "Revise a quantidade faltante e tente novamente."})
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"decision":         decision,
		"missing_quantity": missingQuantity,
		"loaded_quantity":  loadedQuantity,
	})
}

func operationTitle(stage string) string {
	labels := map[string]string{"separating": "Separação", "checking": "Conferência", "loading": "Carregamento", "in_progress": "Evento em andamento"}
	return labels[stage]
}
func operationNext(stage string, eventID int64) string {
	switch stage {
	case "separating":
		return fmt.Sprintf("/events/%d/operation/checking", eventID)
	case "checking":
		return fmt.Sprintf("/events/%d/operation/loading", eventID)
	case "loading":
		return fmt.Sprintf("/events/%d/operation/in_progress", eventID)
	case "in_progress":
		return fmt.Sprintf("/events/%d/return", eventID)
	}
	return fmt.Sprintf("/events/%d", eventID)
}

func (a *App) returnPage(w http.ResponseWriter, r *http.Request) {
	eventID, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	event, err := a.store.GetEvent(r.Context(), eventID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	data := a.baseData(r, "Retorno e conferência", "events")
	data.Event = event
	data.ReturnItems, err = a.store.ReturnItems(r.Context(), eventID)
	if err != nil {
		data.Error = databaseErrorMessage(err)
	}
	a.render(w, r, "return", data)
}

func (a *App) returnSave(w http.ResponseWriter, r *http.Request) {
	eventID, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err = r.ParseForm(); err != nil {
		http.Error(w, "Dados inválidos.", 400)
		return
	}
	items, err := a.store.ReturnItems(r.Context(), eventID)
	if err != nil {
		http.Error(w, databaseErrorMessage(err), 500)
		return
	}
	for i := range items {
		key := strconv.FormatInt(items[i].ChecklistItemID, 10)
		items[i].ReturnedQuantity = parseFloat(r.FormValue("returned_" + key))
		items[i].DamagedQuantity = parseFloat(r.FormValue("damaged_" + key))
		items[i].LostQuantity = parseFloat(r.FormValue("lost_" + key))
		items[i].LaundryQuantity = parseFloat(r.FormValue("laundry_" + key))
		items[i].MaintenanceQuantity = parseFloat(r.FormValue("maintenance_" + key))
		items[i].Notes = strings.TrimSpace(r.FormValue("notes_" + key))
	}
	user := currentUser(r)
	err = a.store.SaveReturnInspections(r.Context(), eventID, items, user.ID, strings.TrimSpace(r.FormValue("general_notes")))
	target := fmt.Sprintf("/events/%d/return?message=%s", eventID, url.QueryEscape("Retorno salvo. Revise e finalize a conferência."))
	if err != nil {
		target = fmt.Sprintf("/events/%d/return?type=danger&message=%s", eventID, url.QueryEscape(err.Error()))
	}
	a.redirect(w, r, target, http.StatusSeeOther)
}

func (a *App) returnFinalize(w http.ResponseWriter, r *http.Request) {
	eventID, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	user := currentUser(r)
	err = a.store.FinalizeReturn(r.Context(), eventID, user.ID, strings.TrimSpace(r.FormValue("notes")))
	target := fmt.Sprintf("/events/%d?message=%s", eventID, url.QueryEscape("Evento finalizado, reservas liberadas e estoque atualizado."))
	if err != nil {
		target = fmt.Sprintf("/events/%d/return?type=danger&message=%s", eventID, url.QueryEscape(err.Error()))
	}
	a.redirect(w, r, target, http.StatusSeeOther)
}
