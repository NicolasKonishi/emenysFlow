package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func modelChildID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("identificador inválido")
	}
	return id, nil
}

func (a *App) menuModelCreate(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", http.StatusForbidden)
		return
	}
	id, err := a.store.CreateMenuModel(r.Context(), currentUser(r).ID)
	if err != nil {
		a.redirect(w, r, "/models?type=danger&message="+url.QueryEscape(databaseErrorMessage(err)), http.StatusSeeOther)
		return
	}
	a.redirect(w, r, fmt.Sprintf("/models/menus/%d?message=%s", id, url.QueryEscape("Modelo criado. Adicione os itens de cada seção.")), http.StatusSeeOther)
}

func (a *App) serviceModelCreate(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", http.StatusForbidden)
		return
	}
	id, err := a.store.CreateServiceModel(r.Context(), currentUser(r).ID)
	if err != nil {
		a.redirect(w, r, "/models?tab=services&type=danger&message="+url.QueryEscape(databaseErrorMessage(err)), http.StatusSeeOther)
		return
	}
	a.redirect(w, r, fmt.Sprintf("/models/services/%d?message=%s", id, url.QueryEscape("Serviço criado. Adicione os componentes incluídos.")), http.StatusSeeOther)
}

func (a *App) modelsPage(w http.ResponseWriter, r *http.Request) {
	data := a.baseData(r, "Modelos", "models")
	data.ActiveTab = r.URL.Query().Get("tab")
	if data.ActiveTab != "services" {
		data.ActiveTab = "menus"
	}
	var err error
	data.MenuModels, err = a.store.ListMenuModels(r.Context(), true)
	if err == nil {
		data.ServiceModels, err = a.store.ListServiceModels(r.Context(), true)
	}
	if err != nil {
		data.Error = databaseErrorMessage(err)
	}
	a.render(w, r, "models", data)
}

func (a *App) menuModelPage(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	data := a.baseData(r, "Modelo de cardápio", "models")
	data.MenuModel, err = a.store.GetMenuModel(r.Context(), id)
	if err == nil {
		data.ModelSections, err = a.store.MenuModelSections(r.Context(), id)
	}
	if err != nil {
		http.NotFound(w, r)
		return
	}
	a.render(w, r, "menu_model", data)
}
func (a *App) menuModelUpdate(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", 403)
		return
	}
	id, err := pathID(r)
	_ = r.ParseForm()
	item, loadErr := a.store.GetMenuModel(r.Context(), id)
	if err == nil {
		err = loadErr
	}
	if err == nil {
		item.Name = strings.TrimSpace(r.FormValue("name"))
		item.Description = strings.TrimSpace(r.FormValue("description"))
		item.MenuType = strings.TrimSpace(r.FormValue("menu_type"))
		item.Active = boolForm(r.FormValue("active"))
		if item.Name == "" {
			err = fmt.Errorf("Informe o nome do modelo.")
		} else {
			err = a.store.UpdateMenuModel(r.Context(), item, currentUser(r).ID)
		}
	}
	target := fmt.Sprintf("/models/menus/%d?message=%s", id, url.QueryEscape("Modelo atualizado e nova versão registrada."))
	if err != nil {
		target = fmt.Sprintf("/models/menus/%d?type=danger&message=%s", id, url.QueryEscape(databaseErrorMessage(err)))
	}
	a.redirect(w, r, target, 303)
}
func (a *App) menuModelDuplicate(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", 403)
		return
	}
	id, err := pathID(r)
	var newID int64
	if err == nil {
		newID, err = a.store.DuplicateMenuModel(r.Context(), id, currentUser(r).ID)
	}
	if err != nil {
		a.redirect(w, r, "/models?type=danger&message="+url.QueryEscape(databaseErrorMessage(err)), 303)
		return
	}
	a.redirect(w, r, fmt.Sprintf("/models/menus/%d?message=%s", newID, url.QueryEscape("Cópia criada com seções, itens e grupos.")), 303)
}
func (a *App) menuModelToggle(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", 403)
		return
	}
	id, err := pathID(r)
	if err == nil {
		err = a.store.ToggleAdvancedMenuModel(r.Context(), id)
	}
	target := "/models?message=" + url.QueryEscape("Status do modelo atualizado.")
	if err != nil {
		target = "/models?type=danger&message=" + url.QueryEscape(databaseErrorMessage(err))
	}
	a.redirect(w, r, target, 303)
}

func (a *App) menuModelItemAdd(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", http.StatusForbidden)
		return
	}
	modelID, err := pathID(r)
	_ = r.ParseForm()
	sectionID, sectionErr := strconv.ParseInt(r.FormValue("section_id"), 10, 64)
	if err == nil {
		err = sectionErr
	}
	if err == nil {
		err = a.store.AddMenuModelItem(r.Context(), modelID, sectionID, r.FormValue("name"), r.FormValue("description"), boolForm(r.FormValue("included")), boolForm(r.FormValue("configurable")), currentUser(r).ID)
	}
	message := "Item adicionado e nova versão registrada."
	flashType := ""
	if err != nil {
		message, flashType = databaseErrorMessage(err), "&type=danger"
	}
	a.redirect(w, r, fmt.Sprintf("/models/menus/%d?message=%s%s", modelID, url.QueryEscape(message), flashType), http.StatusSeeOther)
}

func (a *App) menuModelItemUpdate(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", http.StatusForbidden)
		return
	}
	modelID, err := pathID(r)
	itemID, childErr := modelChildID(r, "itemID")
	if err == nil {
		err = childErr
	}
	_ = r.ParseForm()
	if err == nil {
		err = a.store.UpdateMenuModelItem(r.Context(), modelID, itemID, r.FormValue("name"), r.FormValue("description"), boolForm(r.FormValue("included")), boolForm(r.FormValue("configurable")), currentUser(r).ID)
	}
	message := "Item atualizado e nova versão registrada."
	flashType := ""
	if err != nil {
		message, flashType = databaseErrorMessage(err), "&type=danger"
	}
	a.redirect(w, r, fmt.Sprintf("/models/menus/%d?message=%s%s", modelID, url.QueryEscape(message), flashType), http.StatusSeeOther)
}

func (a *App) menuModelItemRemove(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", http.StatusForbidden)
		return
	}
	modelID, err := pathID(r)
	itemID, childErr := modelChildID(r, "itemID")
	if err == nil {
		err = childErr
	}
	if err == nil {
		err = a.store.RemoveMenuModelItem(r.Context(), modelID, itemID, currentUser(r).ID)
	}
	message := "Item removido do modelo. Eventos existentes foram preservados."
	flashType := ""
	if err != nil {
		message, flashType = databaseErrorMessage(err), "&type=danger"
	}
	a.redirect(w, r, fmt.Sprintf("/models/menus/%d?message=%s%s", modelID, url.QueryEscape(message), flashType), http.StatusSeeOther)
}

func (a *App) menuChoiceGroupUpdate(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", http.StatusForbidden)
		return
	}
	modelID, err := pathID(r)
	groupID, childErr := modelChildID(r, "groupID")
	if err == nil {
		err = childErr
	}
	_ = r.ParseForm()
	min, minErr := strconv.Atoi(r.FormValue("selection_min"))
	if err == nil {
		err = minErr
	}
	max := parseOptionalInt(r.FormValue("selection_max"))
	if err == nil {
		err = a.store.UpdateMenuChoiceGroup(r.Context(), modelID, groupID, min, max, currentUser(r).ID)
	}
	message := "Regra de escolha atualizada."
	flashType := ""
	if err != nil {
		message, flashType = databaseErrorMessage(err), "&type=danger"
	}
	a.redirect(w, r, fmt.Sprintf("/models/menus/%d?message=%s%s", modelID, url.QueryEscape(message), flashType), http.StatusSeeOther)
}

func (a *App) menuModelSectionAdd(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", http.StatusForbidden)
		return
	}
	modelID, err := pathID(r)
	_ = r.ParseForm()
	if err == nil {
		err = a.store.AddMenuModelSection(r.Context(), modelID, r.FormValue("name"), r.FormValue("section_type"), currentUser(r).ID)
	}
	message, flashType := "Seção adicionada e nova versão registrada.", ""
	if err != nil {
		message, flashType = databaseErrorMessage(err), "&type=danger"
	}
	a.redirect(w, r, fmt.Sprintf("/models/menus/%d?message=%s%s", modelID, url.QueryEscape(message), flashType), http.StatusSeeOther)
}

func (a *App) menuModelSectionUpdate(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", http.StatusForbidden)
		return
	}
	modelID, err := pathID(r)
	sectionID, childErr := modelChildID(r, "sectionID")
	if err == nil {
		err = childErr
	}
	_ = r.ParseForm()
	sortOrder, sortErr := strconv.Atoi(r.FormValue("sort_order"))
	selectionMin, minErr := strconv.Atoi(r.FormValue("selection_min"))
	if err == nil {
		err = sortErr
	}
	if err == nil {
		err = minErr
	}
	if err == nil {
		err = a.store.UpdateMenuModelSection(r.Context(), modelID, sectionID, r.FormValue("name"), sortOrder, selectionMin, parseOptionalInt(r.FormValue("selection_max")), boolForm(r.FormValue("required")), boolForm(r.FormValue("allow_event_changes")), currentUser(r).ID)
	}
	message, flashType := "Seção atualizada e nova versão registrada.", ""
	if err != nil {
		message, flashType = databaseErrorMessage(err), "&type=danger"
	}
	a.redirect(w, r, fmt.Sprintf("/models/menus/%d?message=%s%s", modelID, url.QueryEscape(message), flashType), http.StatusSeeOther)
}

func (a *App) menuModelSectionRemove(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", http.StatusForbidden)
		return
	}
	modelID, err := pathID(r)
	sectionID, childErr := modelChildID(r, "sectionID")
	if err == nil {
		err = childErr
	}
	if err == nil {
		err = a.store.RemoveMenuModelSection(r.Context(), modelID, sectionID, currentUser(r).ID)
	}
	message, flashType := "Seção removida. Eventos existentes foram preservados.", ""
	if err != nil {
		message, flashType = databaseErrorMessage(err), "&type=danger"
	}
	a.redirect(w, r, fmt.Sprintf("/models/menus/%d?message=%s%s", modelID, url.QueryEscape(message), flashType), http.StatusSeeOther)
}

func (a *App) serviceModelPage(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	data := a.baseData(r, "Modelo de serviço", "models")
	data.ServiceModel, err = a.store.GetServiceModel(r.Context(), id)
	if err == nil {
		data.ServiceComponents, err = a.store.ServiceModelComponents(r.Context(), id)
	}
	if err != nil {
		http.NotFound(w, r)
		return
	}
	a.render(w, r, "service_model", data)
}
func (a *App) serviceModelUpdate(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", 403)
		return
	}
	id, err := pathID(r)
	_ = r.ParseForm()
	item, loadErr := a.store.GetServiceModel(r.Context(), id)
	if err == nil {
		err = loadErr
	}
	if err == nil {
		item.Name = strings.TrimSpace(r.FormValue("name"))
		item.Description = strings.TrimSpace(r.FormValue("description"))
		item.Category = strings.TrimSpace(r.FormValue("category"))
		item.BillingUnit = strings.TrimSpace(r.FormValue("billing_unit"))
		item.DurationMinutes = parseOptionalInt(r.FormValue("duration_minutes"))
		supplier := strings.TrimSpace(r.FormValue("supplier"))
		item.Supplier = sql.NullString{String: supplier, Valid: supplier != ""}
		item.Active = boolForm(r.FormValue("active"))
		if item.Name == "" {
			err = fmt.Errorf("Informe o nome do serviço.")
		} else {
			err = a.store.UpdateServiceModel(r.Context(), item, currentUser(r).ID)
		}
	}
	target := fmt.Sprintf("/models/services/%d?message=%s", id, url.QueryEscape("Serviço atualizado e nova versão registrada."))
	if err != nil {
		target = fmt.Sprintf("/models/services/%d?type=danger&message=%s", id, url.QueryEscape(databaseErrorMessage(err)))
	}
	a.redirect(w, r, target, 303)
}
func (a *App) serviceModelDuplicate(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", 403)
		return
	}
	id, err := pathID(r)
	var newID int64
	if err == nil {
		newID, err = a.store.DuplicateServiceModel(r.Context(), id, currentUser(r).ID)
	}
	if err != nil {
		a.redirect(w, r, "/models?tab=services&type=danger&message="+url.QueryEscape(databaseErrorMessage(err)), 303)
		return
	}
	a.redirect(w, r, fmt.Sprintf("/models/services/%d?message=%s", newID, url.QueryEscape("Cópia do serviço criada.")), 303)
}
func (a *App) serviceModelToggle(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", 403)
		return
	}
	id, err := pathID(r)
	if err == nil {
		err = a.store.ToggleServiceModel(r.Context(), id)
	}
	target := "/models?tab=services&message=" + url.QueryEscape("Status do serviço atualizado.")
	if err != nil {
		target = "/models?tab=services&type=danger&message=" + url.QueryEscape(databaseErrorMessage(err))
	}
	a.redirect(w, r, target, 303)
}

func (a *App) serviceModelComponentAdd(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", http.StatusForbidden)
		return
	}
	modelID, err := pathID(r)
	_ = r.ParseForm()
	if err == nil {
		err = a.store.AddServiceModelComponent(r.Context(), modelID, r.FormValue("name"), r.FormValue("description"), r.FormValue("category"), boolForm(r.FormValue("included")), boolForm(r.FormValue("configurable")), currentUser(r).ID)
	}
	message, flashType := "Componente adicionado e nova versão registrada.", ""
	if err != nil {
		message, flashType = databaseErrorMessage(err), "&type=danger"
	}
	a.redirect(w, r, fmt.Sprintf("/models/services/%d?message=%s%s", modelID, url.QueryEscape(message), flashType), http.StatusSeeOther)
}

func (a *App) serviceModelComponentUpdate(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", http.StatusForbidden)
		return
	}
	modelID, err := pathID(r)
	componentID, childErr := modelChildID(r, "componentID")
	if err == nil {
		err = childErr
	}
	_ = r.ParseForm()
	if err == nil {
		err = a.store.UpdateServiceModelComponent(r.Context(), modelID, componentID, r.FormValue("name"), r.FormValue("description"), r.FormValue("category"), r.FormValue("configuration_json"), boolForm(r.FormValue("included")), boolForm(r.FormValue("configurable")), currentUser(r).ID)
	}
	message, flashType := "Componente atualizado e nova versão registrada.", ""
	if err != nil {
		message, flashType = databaseErrorMessage(err), "&type=danger"
	}
	a.redirect(w, r, fmt.Sprintf("/models/services/%d?message=%s%s", modelID, url.QueryEscape(message), flashType), http.StatusSeeOther)
}

func (a *App) serviceModelComponentRemove(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", http.StatusForbidden)
		return
	}
	modelID, err := pathID(r)
	componentID, childErr := modelChildID(r, "componentID")
	if err == nil {
		err = childErr
	}
	if err == nil {
		err = a.store.RemoveServiceModelComponent(r.Context(), modelID, componentID, currentUser(r).ID)
	}
	message, flashType := "Componente removido. Eventos existentes foram preservados.", ""
	if err != nil {
		message, flashType = databaseErrorMessage(err), "&type=danger"
	}
	a.redirect(w, r, fmt.Sprintf("/models/services/%d?message=%s%s", modelID, url.QueryEscape(message), flashType), http.StatusSeeOther)
}
