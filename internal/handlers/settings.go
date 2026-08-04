package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"buffetflow/internal/models"
	"buffetflow/internal/services"
)

func (a *App) settings(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", 403)
		return
	}
	data := a.baseData(r, "Configurações", "settings")
	data.Users, _ = a.store.ListUsers(r.Context())
	data.Categories, _ = a.store.ListCategories(r.Context())
	data.Locations, _ = a.store.ListLocations(r.Context())
	data.Rules, _ = a.store.ListRules(r.Context(), true)
	data.Containers, _ = a.store.ListContainerTypes(r.Context(), true)
	settings, _ := a.store.OperationalSettings(r.Context())
	keys := make([]string, 0, len(settings))
	for key := range settings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		data.OperationalSettings = append(data.OperationalSettings, settings[key])
	}
	a.render(w, r, "settings", data)
}

func (a *App) operationalSettingSave(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", http.StatusForbidden)
		return
	}
	key := strings.TrimSpace(r.PathValue("key"))
	value := parseFloat(r.FormValue("value"))
	err := a.store.SaveOperationalSetting(r.Context(), key, value, currentUser(r).ID)
	target := "/settings?message=" + url.QueryEscape("Configuração operacional atualizada. Novos recálculos já usarão este valor.")
	if err != nil {
		target = "/settings?type=danger&message=" + url.QueryEscape("Não foi possível atualizar a configuração.")
	}
	a.redirect(w, r, target, http.StatusSeeOther)
}
func (a *App) userForm(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", 403)
		return
	}
	data := a.baseData(r, "Novo usuário", "settings")
	data.FormAction = "/settings/users"
	user := models.User{Role: "operational", Active: true}
	if r.PathValue("id") != "" {
		id, err := pathID(r)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		user, err = a.store.GetUser(r.Context(), id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		data.IsEdit = true
		data.Title = "Editar usuário"
		data.FormAction = fmt.Sprintf("/settings/users/%d", id)
	}
	data.Users = []models.User{user}
	a.render(w, r, "user_form", data)
}
func (a *App) userCreate(w http.ResponseWriter, r *http.Request) { a.saveUser(w, r, 0) }
func (a *App) userUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	a.saveUser(w, r, id)
}
func (a *App) saveUser(w http.ResponseWriter, r *http.Request, id int64) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", 403)
		return
	}
	_ = r.ParseForm()
	user := models.User{ID: id, Name: strings.TrimSpace(r.FormValue("name")), Email: strings.TrimSpace(r.FormValue("email")), Role: r.FormValue("role"), Active: true}
	password := r.FormValue("password")
	var hash string
	var err error
	if password != "" {
		hash, err = services.HashPassword(password)
	}
	if user.Name == "" || user.Email == "" {
		err = fmt.Errorf("nome e e-mail são obrigatórios")
	}
	if id == 0 && password == "" {
		err = fmt.Errorf("a senha inicial é obrigatória")
	}
	if err == nil {
		err = a.store.SaveUser(r.Context(), &user, hash)
	}
	if err != nil {
		data := a.baseData(r, "Revisar usuário", "settings")
		data.Users = []models.User{user}
		data.Error = err.Error()
		data.IsEdit = id > 0
		data.FormAction = "/settings/users"
		if id > 0 {
			data.FormAction = fmt.Sprintf("/settings/users/%d", id)
		}
		a.render(w, r, "user_form", data)
		return
	}
	a.redirect(w, r, "/settings?message="+url.QueryEscape("Usuário salvo."), http.StatusSeeOther)
}
func (a *App) userToggle(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", 403)
		return
	}
	id, err := pathID(r)
	current := currentUser(r)
	if err == nil {
		err = a.store.ToggleUser(r.Context(), id, current.ID)
	}
	target := "/settings?message=" + url.QueryEscape("Acesso do usuário atualizado.")
	if err != nil {
		target = "/settings?type=danger&message=" + url.QueryEscape(err.Error())
	}
	a.redirect(w, r, target, http.StatusSeeOther)
}
func (a *App) categorySave(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", 403)
		return
	}
	_ = r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	var err error
	if name == "" {
		err = fmt.Errorf("empty name")
	} else {
		err = a.store.SaveCategory(r.Context(), name)
	}
	target := "/settings?message=" + url.QueryEscape("Categoria criada.")
	if err != nil {
		target = "/settings?type=danger&message=" + url.QueryEscape("Não foi possível criar a categoria.")
	}
	a.redirect(w, r, target, http.StatusSeeOther)
}
func (a *App) locationSave(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", 403)
		return
	}
	_ = r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	var err error
	if name == "" {
		err = fmt.Errorf("empty name")
	} else {
		err = a.store.SaveLocation(r.Context(), name)
	}
	target := "/settings?message=" + url.QueryEscape("Localização criada.")
	if err != nil {
		target = "/settings?type=danger&message=" + url.QueryEscape("Não foi possível criar a localização.")
	}
	a.redirect(w, r, target, http.StatusSeeOther)
}
