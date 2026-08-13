package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"buffetflow/internal/models"
	"buffetflow/internal/repositories"
)

func (a *App) eventLayoutPage(w http.ResponseWriter, r *http.Request) {
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
	layout, err := a.store.GetEventFloorLayout(r.Context(), id)
	if err != nil {
		layout = models.EventFloorLayout{EventID: id, LayoutJSON: `{"version":2,"width":1400,"height":900,"waiters":[],"elements":[]}`}
	}
	data := a.baseData(r, "Layout do salão", "events")
	data.LayoutMode = "event"
	data.Event = event
	data.FloorLayout = layout
	data.StaffSummary, _ = a.checklist.StaffSummary(r.Context(), id)
	if data.StaffSummary.Waiters == 0 && event.GuestCount > 0 {
		data.StaffSummary.Waiters = (event.GuestCount + 17) / 18
	}
	if err != nil {
		data.Error = databaseErrorMessage(err)
	}
	a.render(w, r, "layout_form", data)
}

func (a *App) eventLayoutSave(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	layoutJSON := r.FormValue("layout_json")
	if !json.Valid([]byte(layoutJSON)) {
		a.redirect(w, r, fmt.Sprintf("/events/%d/layout?type=danger&message=%s", id, url.QueryEscape("Layout inválido. Tente salvar novamente.")), http.StatusSeeOther)
		return
	}
	err = a.store.SaveEventFloorLayout(r.Context(), id, layoutJSON, currentUser(r).ID)
	target := fmt.Sprintf("/events/%d/layout?message=%s", id, url.QueryEscape("Layout salvo com sucesso."))
	if errors.Is(err, repositories.ErrInvalidLayoutJSON) {
		target = fmt.Sprintf("/events/%d/layout?type=danger&message=%s", id, url.QueryEscape("Layout inválido. Tente salvar novamente."))
	} else if err != nil {
		target = fmt.Sprintf("/events/%d/layout?type=danger&message=%s", id, url.QueryEscape(databaseErrorMessage(err)))
	}
	a.redirect(w, r, target, http.StatusSeeOther)
}
