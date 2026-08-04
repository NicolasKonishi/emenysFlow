package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"buffetflow/internal/models"
)

func (a *App) eventMenuSnapshotItemAdd(w http.ResponseWriter, r *http.Request) {
	eventID, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	sectionID, _ := strconv.ParseInt(r.FormValue("section_id"), 10, 64)
	sourceID, _ := strconv.ParseInt(r.FormValue("source_menu_item_id"), 10, 64)
	_, err = a.store.AddEventMenuSnapshotItem(r.Context(), eventID, sectionID, sourceID, strings.TrimSpace(r.FormValue("name")), strings.TrimSpace(r.FormValue("description")), parseFloat(r.FormValue("portions")), currentUser(r).ID)
	if err == nil {
		_, err = a.checklist.GenerateTracked(r.Context(), eventID, "event_menu_item_added", currentUser(r).ID)
	}
	menuSnapshotRedirect(w, r, eventID, err, "Item adicionado ao cardápio do evento.")
}
func (a *App) eventMenuSnapshotItemUpdate(w http.ResponseWriter, r *http.Request) {
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
	sectionID, _ := strconv.ParseInt(r.FormValue("section_id"), 10, 64)
	sortOrder, _ := strconv.Atoi(r.FormValue("sort_order"))
	container := parseOptionalInt(r.FormValue("container_type_id"))
	err = a.store.UpdateEventMenuSnapshotItem(r.Context(), eventID, itemID, sectionID, strings.TrimSpace(r.FormValue("display_name")), strings.TrimSpace(r.FormValue("description")), strings.TrimSpace(r.FormValue("notes")), sortOrder, parseFloat(r.FormValue("portions")), container, currentUser(r).ID)
	if err == nil {
		err = a.store.SyncLegacyEventMenuFromSnapshot(r.Context(), eventID)
	}
	if err == nil {
		_, err = a.checklist.GenerateTracked(r.Context(), eventID, "event_menu_item_updated", currentUser(r).ID)
	}
	menuSnapshotRedirect(w, r, eventID, err, "Item atualizado somente neste evento.")
}
func (a *App) eventMenuSnapshotItemRemove(w http.ResponseWriter, r *http.Request) {
	a.setEventMenuSnapshotItemRemoved(w, r, true)
}
func (a *App) eventMenuSnapshotItemRestore(w http.ResponseWriter, r *http.Request) {
	a.setEventMenuSnapshotItemRemoved(w, r, false)
}
func (a *App) setEventMenuSnapshotItemRemoved(w http.ResponseWriter, r *http.Request, removed bool) {
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
	err = a.store.SetEventMenuSnapshotItemRemoved(r.Context(), eventID, itemID, removed, currentUser(r).ID)
	if err == nil {
		_, err = a.checklist.GenerateTracked(r.Context(), eventID, "event_menu_item_visibility", currentUser(r).ID)
	}
	message := "Item restaurado no cardápio do evento."
	if removed {
		message = "Item removido somente deste evento."
	}
	menuSnapshotRedirect(w, r, eventID, err, message)
}
func (a *App) eventMenuSnapshotRestoreModel(w http.ResponseWriter, r *http.Request) {
	eventID, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	modelID, selected, err := a.store.EventMenuModelSelection(r.Context(), eventID)
	if err == nil && modelID > 0 {
		err = a.store.ApplyMenuModelSnapshot(r.Context(), eventID, modelID, selected, nil, currentUser(r).ID)
	}
	if err == nil {
		err = a.store.SyncLegacyEventMenuFromSnapshot(r.Context(), eventID)
	}
	if err == nil {
		_, err = a.checklist.GenerateTracked(r.Context(), eventID, "event_menu_restored", currentUser(r).ID)
	}
	menuSnapshotRedirect(w, r, eventID, err, "Cardápio restaurado a partir do modelo.")
}

func (a *App) eventMenuContainerSave(w http.ResponseWriter, r *http.Request) {
	eventID, itemID, err := eventAndMenuItemIDs(r)
	if err == nil {
		containerID := parseOptionalInt(r.FormValue("container_type_id"))
		quantity := parseOptionalFloat(r.FormValue("quantity"))
		capacity := parseOptionalFloat(r.FormValue("capacity_portions"))
		container := models.EventMenuContainer{Purpose: r.FormValue("purpose"), ContainerTypeID: containerID, Quantity: quantity, CapacityPortions: capacity, RequiresLid: boolForm(r.FormValue("requires_lid")), Notes: strings.TrimSpace(r.FormValue("notes"))}
		err = a.store.SaveEventMenuContainer(r.Context(), eventID, itemID, container)
	}
	if err == nil {
		_, err = a.checklist.GenerateTracked(r.Context(), eventID, "event_menu_container_saved", currentUser(r).ID)
	}
	menuSnapshotRedirect(w, r, eventID, err, "Recipiente do evento atualizado.")
}

func (a *App) eventMenuContainerRemove(w http.ResponseWriter, r *http.Request) {
	eventID, itemID, err := eventAndMenuItemIDs(r)
	containerID, parseErr := strconv.ParseInt(r.PathValue("containerID"), 10, 64)
	if err == nil {
		err = parseErr
	}
	if err == nil {
		err = a.store.RemoveEventMenuContainer(r.Context(), eventID, itemID, containerID)
	}
	if err == nil {
		_, err = a.checklist.GenerateTracked(r.Context(), eventID, "event_menu_container_removed", currentUser(r).ID)
	}
	menuSnapshotRedirect(w, r, eventID, err, "Recipiente removido deste evento.")
}

func (a *App) eventMenuEquipmentSave(w http.ResponseWriter, r *http.Request) {
	eventID, itemID, err := eventAndMenuItemIDs(r)
	inventoryID, parseErr := strconv.ParseInt(r.FormValue("inventory_item_id"), 10, 64)
	if err == nil {
		err = parseErr
	}
	if err == nil {
		err = a.store.SaveEventMenuEquipment(r.Context(), eventID, itemID, inventoryID, parseFloat(r.FormValue("quantity")), boolForm(r.FormValue("required")), strings.TrimSpace(r.FormValue("notes")))
	}
	if err == nil {
		_, err = a.checklist.GenerateTracked(r.Context(), eventID, "event_menu_equipment_saved", currentUser(r).ID)
	}
	menuSnapshotRedirect(w, r, eventID, err, "Equipamento do evento atualizado.")
}

func (a *App) eventMenuEquipmentRemove(w http.ResponseWriter, r *http.Request) {
	eventID, itemID, err := eventAndMenuItemIDs(r)
	equipmentID, parseErr := strconv.ParseInt(r.PathValue("equipmentID"), 10, 64)
	if err == nil {
		err = parseErr
	}
	if err == nil {
		err = a.store.RemoveEventMenuEquipment(r.Context(), eventID, itemID, equipmentID)
	}
	if err == nil {
		_, err = a.checklist.GenerateTracked(r.Context(), eventID, "event_menu_equipment_removed", currentUser(r).ID)
	}
	menuSnapshotRedirect(w, r, eventID, err, "Equipamento removido deste evento.")
}

func (a *App) eventCakeConfigurationSave(w http.ResponseWriter, r *http.Request) {
	eventID, err := pathID(r)
	count, parseErr := strconv.Atoi(r.FormValue("cake_count"))
	if err == nil {
		err = parseErr
	}
	if err == nil {
		err = a.store.SaveEventCakeConfiguration(r.Context(), models.EventCakeConfiguration{EventID: eventID, CakeCount: count, RequiresRefrigeration: boolForm(r.FormValue("requires_refrigeration")), Notes: strings.TrimSpace(r.FormValue("notes"))}, currentUser(r).ID)
	}
	if err == nil {
		_, err = a.checklist.GenerateTracked(r.Context(), eventID, "event_cake_updated", currentUser(r).ID)
	}
	menuSnapshotRedirect(w, r, eventID, err, "Configuração do bolo atualizada.")
}

func eventAndMenuItemIDs(r *http.Request) (int64, int64, error) {
	eventID, err := pathID(r)
	if err != nil {
		return 0, 0, err
	}
	itemID, err := strconv.ParseInt(r.PathValue("itemID"), 10, 64)
	if err != nil || itemID <= 0 {
		return eventID, 0, sql.ErrNoRows
	}
	return eventID, itemID, nil
}
func menuSnapshotRedirect(w http.ResponseWriter, r *http.Request, eventID int64, err error, message string) {
	kind := "success"
	if err != nil {
		kind = "danger"
		message = databaseErrorMessage(err)
	}
	http.Redirect(w, r, fmt.Sprintf("/events/%d/menu?type=%s&message=%s", eventID, kind, url.QueryEscape(message)), http.StatusSeeOther)
}
