package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"buffetflow/internal/models"
	"buffetflow/internal/repositories"
)

func (a *App) layoutsPage(w http.ResponseWriter, r *http.Request) {
	data := a.baseData(r, "Layouts do salão", "layouts")
	data.Query = r.URL.Query().Get("q")
	layouts, err := a.store.ListStandaloneFloorLayouts(r.Context(), data.Query)
	if err != nil {
		data.Error = databaseErrorMessage(err)
	} else {
		data.StandaloneLayouts = layouts
	}
	a.render(w, r, "layouts", data)
}

func (a *App) layoutForm(w http.ResponseWriter, r *http.Request) {
	data := a.baseData(r, "Novo layout", "layouts")
	data.FormAction = "/layouts"
	data.StandaloneLayout = models.StandaloneFloorLayout{
		LayoutJSON:      defaultStandaloneLayoutDocument(),
		WaiterNamesJSON: `["Garçom 1","Garçom 2","Garçom 3","Garçom 4","Garçom 5","Garçom 6"]`,
		WaiterCount:     6,
	}
	data.LayoutMode = "standalone"
	if r.PathValue("id") != "" {
		id, err := pathID(r)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		layout, err := a.store.GetStandaloneFloorLayout(r.Context(), id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		data.Title = "Editar layout"
		data.StandaloneLayout = layout
		data.IsEdit = true
		data.FormAction = fmt.Sprintf("/layouts/%d", id)
	}
	a.render(w, r, "layout_form", data)
}

func (a *App) layoutCreate(w http.ResponseWriter, r *http.Request) {
	a.saveStandaloneLayout(w, r, 0)
}

func (a *App) layoutUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	a.saveStandaloneLayout(w, r, id)
}

func (a *App) layoutArchive(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	err = a.store.ArchiveStandaloneFloorLayout(r.Context(), id, currentUser(r).ID)
	target := "/layouts?message=" + url.QueryEscape("Layout removido.")
	if err != nil {
		target = fmt.Sprintf("/layouts/%d?type=danger&message=%s", id, url.QueryEscape(databaseErrorMessage(err)))
	}
	a.redirect(w, r, target, http.StatusSeeOther)
}

func (a *App) saveStandaloneLayout(w http.ResponseWriter, r *http.Request, id int64) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	layout := models.StandaloneFloorLayout{
		ID:              id,
		Name:            strings.TrimSpace(r.FormValue("name")),
		Venue:           strings.TrimSpace(r.FormValue("venue")),
		GuestCount:      parseIntDefault(r.FormValue("guest_count"), 0),
		WaiterCount:     parseIntDefault(r.FormValue("waiter_count"), 0),
		WaiterNamesJSON: strings.TrimSpace(r.FormValue("waiter_names_json")),
		LayoutJSON:      r.FormValue("layout_json"),
	}
	if layout.Name == "" {
		a.redirect(w, r, layoutFormRedirect(id, "danger", "Informe um nome para o layout."), http.StatusSeeOther)
		return
	}
	if layout.WaiterNamesJSON == "" {
		layout.WaiterNamesJSON = "[]"
	}
	if !json.Valid([]byte(layout.LayoutJSON)) {
		a.redirect(w, r, layoutFormRedirect(id, "danger", "Layout inválido. Tente salvar novamente."), http.StatusSeeOther)
		return
	}
	layout.LayoutJSON = mergeWaitersIntoLayoutJSON(layout.LayoutJSON, layout.WaiterNamesJSON)

	err := a.store.SaveStandaloneFloorLayout(r.Context(), &layout, currentUser(r).ID)
	target := fmt.Sprintf("/layouts/%d?message=%s", layout.ID, url.QueryEscape("Layout salvo com sucesso."))
	if errors.Is(err, repositories.ErrInvalidLayoutJSON) {
		target = layoutFormRedirect(layout.ID, "danger", "Layout inválido. Tente salvar novamente.")
	} else if err != nil {
		target = layoutFormRedirect(layout.ID, "danger", databaseErrorMessage(err))
	}
	a.redirect(w, r, target, http.StatusSeeOther)
}

func layoutFormRedirect(id int64, flashType, message string) string {
	if id > 0 {
		return fmt.Sprintf("/layouts/%d?type=%s&message=%s", id, flashType, url.QueryEscape(message))
	}
	return fmt.Sprintf("/layouts/new?type=%s&message=%s", flashType, url.QueryEscape(message))
}

func defaultStandaloneLayoutDocument() string {
	return `{"version":2,"width":1400,"height":900,"waiters":[],"elements":[]}`
}

func mergeWaitersIntoLayoutJSON(layoutJSON, waiterNamesJSON string) string {
	var document map[string]any
	if err := json.Unmarshal([]byte(layoutJSON), &document); err != nil {
		return layoutJSON
	}
	var waiters []string
	if json.Valid([]byte(waiterNamesJSON)) {
		_ = json.Unmarshal([]byte(waiterNamesJSON), &waiters)
	}
	clean := make([]string, 0, len(waiters))
	for _, name := range waiters {
		if name = strings.TrimSpace(name); name != "" {
			clean = append(clean, name)
		}
	}
	document["waiters"] = clean
	document["version"] = 2
	encoded, err := json.Marshal(document)
	if err != nil {
		return layoutJSON
	}
	return string(encoded)
}

func parseIntDefault(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}
