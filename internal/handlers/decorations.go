package handlers

import (
	"fmt"
	"net/http"
	"net/url"
)

func (a *App) eventDecorationsPage(w http.ResponseWriter, r *http.Request) {
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
	data := a.baseData(r, "Decoração do evento", "events")
	data.Event = event
	data.DecorationProfile, _ = a.store.GetDecorationProfile(r.Context(), id)
	data.Items, _ = a.store.ListInventory(r.Context(), "", "", false)
	data.Decorations, err = a.store.EventDecorationSelection(r.Context(), id)
	if err != nil {
		data.Error = databaseErrorMessage(err)
	}
	a.render(w, r, "event_decorations", data)
}
func (a *App) eventDecorationsSave(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	profile, _ := a.store.GetDecorationProfile(r.Context(), id)
	profile.EventID = id
	profile.Style = r.FormValue("style")
	profile.Description = r.FormValue("description")
	profile.PrimaryColors = r.FormValue("primary_colors")
	profile.Theme = r.FormValue("theme")
	profile.Notes = r.FormValue("notes")
	profile.ResponsibleName = r.FormValue("responsible_name")
	profile.Active = true
	err = a.store.SaveDecorationProfile(r.Context(), &profile, currentUser(r).ID)
	if err == nil {
		_, err = a.checklist.GenerateTracked(r.Context(), id, "decoration_saved", currentUser(r).ID)
	}
	target := fmt.Sprintf("/events/%d?message=%s", id, url.QueryEscape("Decoração salva e checklist atualizada."))
	if err != nil {
		target = fmt.Sprintf("/events/%d/decorations?type=danger&message=%s", id, url.QueryEscape(databaseErrorMessage(err)))
	}
	a.redirect(w, r, target, http.StatusSeeOther)
}
