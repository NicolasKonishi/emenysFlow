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

func (a *App) catalog(w http.ResponseWriter, r *http.Request) {
	data := a.baseData(r, "Cardápios e recipientes", "catalog")
	var err error
	data.MenuTemplates, err = a.store.ListMenuTemplates(r.Context(), true)
	if err != nil {
		data.Error = databaseErrorMessage(err)
	}
	data.MenuItems, err = a.store.ListMenuItems(r.Context(), true)
	if err != nil {
		data.Error = databaseErrorMessage(err)
	}
	data.Containers, _ = a.store.ListContainerTypes(r.Context(), true)
	a.render(w, r, "catalog", data)
}

func (a *App) menuTemplateForm(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", http.StatusForbidden)
		return
	}
	data := a.baseData(r, "Novo cardápio", "catalog")
	data.FormAction = "/catalog/templates"
	if r.PathValue("id") != "" {
		id, err := pathID(r)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		item, err := a.store.GetMenuTemplate(r.Context(), id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		data.MenuTemplate = item
		data.IsEdit = true
		data.Title = "Editar cardápio"
		data.FormAction = fmt.Sprintf("/catalog/templates/%d", id)
		data.MenuItems, _ = a.store.ListTemplateMenuItems(r.Context(), id, true)
	} else {
		data.EventMenu = a.globalMenuOptions(r)
	}
	a.render(w, r, "menu_template_form", data)
}

func (a *App) globalMenuOptions(r *http.Request) []models.EventMenuItem {
	items, _ := a.store.ListMenuItems(r.Context(), false)
	result := make([]models.EventMenuItem, 0, len(items))
	for _, item := range items {
		result = append(result, models.EventMenuItem{MenuItemID: item.ID, MenuItemName: item.Name, CategoryName: item.CategoryName})
	}
	return result
}

func parseMenuTemplate(r *http.Request, id int64) (models.MenuTemplate, []int64, error) {
	if err := r.ParseForm(); err != nil {
		return models.MenuTemplate{}, nil, err
	}
	item := models.MenuTemplate{
		ID:               id,
		Name:             strings.TrimSpace(r.FormValue("name")),
		Description:      strings.TrimSpace(r.FormValue("description")),
		HasDecoration:    boolForm(r.FormValue("has_decoration")),
		HasWelcomeDrinks: boolForm(r.FormValue("has_welcome_drinks")),
		HasCoffeeTable:   boolForm(r.FormValue("has_coffee_table")),
		Active:           true,
	}
	var itemIDs []int64
	for _, raw := range r.Form["menu_item_ids"] {
		menuItemID, err := strconv.ParseInt(raw, 10, 64)
		if err == nil && menuItemID > 0 {
			itemIDs = append(itemIDs, menuItemID)
		}
	}
	if item.Name == "" {
		return item, itemIDs, fmt.Errorf("Informe o nome do cardápio.")
	}
	if id == 0 && len(itemIDs) == 0 {
		return item, itemIDs, fmt.Errorf("Selecione pelo menos um item para o cardápio.")
	}
	return item, itemIDs, nil
}

func (a *App) saveMenuTemplate(w http.ResponseWriter, r *http.Request, id int64) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", http.StatusForbidden)
		return
	}
	item, itemIDs, err := parseMenuTemplate(r, id)
	if err == nil {
		err = a.store.SaveMenuTemplate(r.Context(), &item, itemIDs)
	}
	if err != nil {
		data := a.baseData(r, "Revisar cardápio", "catalog")
		data.MenuTemplate = item
		data.Error = err.Error()
		data.IsEdit = id > 0
		data.FormAction = "/catalog/templates"
		if id > 0 {
			data.FormAction = fmt.Sprintf("/catalog/templates/%d", id)
		}
		if id > 0 {
			data.MenuItems, _ = a.store.ListTemplateMenuItems(r.Context(), id, true)
		} else {
			data.EventMenu = a.globalMenuOptions(r)
			selected := map[int64]bool{}
			for _, menuItemID := range itemIDs {
				selected[menuItemID] = true
			}
			for index := range data.EventMenu {
				data.EventMenu[index].Selected = selected[data.EventMenu[index].MenuItemID]
			}
		}
		a.render(w, r, "menu_template_form", data)
		return
	}
	target := "/catalog?message=" + url.QueryEscape("Cardápio atualizado.")
	if id == 0 {
		target = fmt.Sprintf("/catalog/templates/%d/edit?message=%s", item.ID, url.QueryEscape("Cardápio criado. Agora você pode editar cada item separadamente."))
	}
	a.redirect(w, r, target, http.StatusSeeOther)
}

func (a *App) menuTemplateCreate(w http.ResponseWriter, r *http.Request) {
	a.saveMenuTemplate(w, r, 0)
}

func (a *App) menuTemplateUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	a.saveMenuTemplate(w, r, id)
}

func (a *App) menuTemplateToggle(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", http.StatusForbidden)
		return
	}
	id, err := pathID(r)
	if err == nil {
		err = a.store.ToggleMenuTemplate(r.Context(), id)
	}
	target := "/catalog?message=" + url.QueryEscape("Status do cardápio atualizado.")
	if err != nil {
		target = "/catalog?type=danger&message=" + url.QueryEscape(databaseErrorMessage(err))
	}
	a.redirect(w, r, target, http.StatusSeeOther)
}

func templateIDFromPath(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("templateID"), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid template id")
	}
	return id, nil
}

func (a *App) populateTemplateMenuItemForm(r *http.Request, data *PageData) {
	data.MenuCategories, _ = a.store.ListMenuCategories(r.Context())
	data.Containers, _ = a.store.ListContainerTypes(r.Context(), false)
	data.Items, _ = a.store.ListInventory(r.Context(), "", "", false)
	data.MenuItems, _ = a.store.ListMenuItems(r.Context(), false)
	data.Equipment, _ = a.store.EquipmentOptions(r.Context())
	selected := map[int64]models.EquipmentLink{}
	for _, link := range data.MenuItem.Equipment {
		selected[link.EquipmentID] = link
	}
	for index := range data.Equipment {
		if link, ok := selected[data.Equipment[index].EquipmentID]; ok {
			data.Equipment[index].Selected = true
			data.Equipment[index].Quantity = link.Quantity
			data.Equipment[index].Required = link.Required
		}
	}
}

func (a *App) templateMenuItemForm(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", http.StatusForbidden)
		return
	}
	templateID, err := templateIDFromPath(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	template, err := a.store.GetMenuTemplate(r.Context(), templateID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	data := a.baseData(r, "Novo item exclusivo", "catalog")
	data.MenuTemplate = template
	data.FormAction = fmt.Sprintf("/catalog/templates/%d/items", templateID)
	data.MenuItem = models.MenuItem{
		TemplateOwnerID:       sql.NullInt64{Int64: templateID, Valid: true},
		Active:                true,
		CalculationType:       "menu_only",
		CalculationDivisor:    1,
		CalculationMultiplier: 1,
		CalculationWeight:     1,
	}
	if r.PathValue("id") != "" {
		itemID, itemErr := pathID(r)
		if itemErr != nil {
			http.NotFound(w, r)
			return
		}
		item, itemErr := a.store.GetMenuItem(r.Context(), itemID)
		if itemErr != nil || !item.TemplateOwnerID.Valid || item.TemplateOwnerID.Int64 != templateID {
			http.NotFound(w, r)
			return
		}
		data.MenuItem = item
		data.IsEdit = true
		data.Title = "Editar item do cardápio"
		data.FormAction = fmt.Sprintf("/catalog/templates/%d/items/%d", templateID, itemID)
	}
	a.populateTemplateMenuItemForm(r, &data)
	a.render(w, r, "template_menu_item_form", data)
}

func (a *App) saveTemplateMenuItem(w http.ResponseWriter, r *http.Request, templateID, itemID int64) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", http.StatusForbidden)
		return
	}
	item, equipment, err := a.parseMenuItem(r, itemID)
	item.TemplateOwnerID = sql.NullInt64{Int64: templateID, Valid: true}
	if err == nil {
		err = a.store.SaveTemplateMenuItem(r.Context(), templateID, &item, equipment)
	}
	if err != nil {
		data := a.baseData(r, "Revisar item do cardápio", "catalog")
		data.MenuTemplate, _ = a.store.GetMenuTemplate(r.Context(), templateID)
		data.MenuItem = item
		data.MenuItem.Equipment = equipment
		data.Error = databaseErrorMessage(err)
		data.IsEdit = itemID > 0
		data.FormAction = fmt.Sprintf("/catalog/templates/%d/items", templateID)
		if itemID > 0 {
			data.FormAction = fmt.Sprintf("/catalog/templates/%d/items/%d", templateID, itemID)
		}
		a.populateTemplateMenuItemForm(r, &data)
		a.render(w, r, "template_menu_item_form", data)
		return
	}
	a.redirect(w, r, fmt.Sprintf("/catalog/templates/%d/edit?message=%s", templateID, url.QueryEscape("Item do cardápio salvo.")), http.StatusSeeOther)
}

func (a *App) templateMenuItemCreate(w http.ResponseWriter, r *http.Request) {
	templateID, err := templateIDFromPath(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	a.saveTemplateMenuItem(w, r, templateID, 0)
}

func (a *App) templateMenuItemUpdate(w http.ResponseWriter, r *http.Request) {
	templateID, err := templateIDFromPath(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	itemID, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	a.saveTemplateMenuItem(w, r, templateID, itemID)
}

func (a *App) templateMenuItemClone(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", http.StatusForbidden)
		return
	}
	templateID, err := templateIDFromPath(r)
	if err == nil {
		_ = r.ParseForm()
		sourceID, parseErr := strconv.ParseInt(r.FormValue("source_item_id"), 10, 64)
		if parseErr != nil || sourceID <= 0 {
			err = fmt.Errorf("Selecione um item do catálogo para copiar.")
		} else {
			_, err = a.store.CloneMenuItemToTemplate(r.Context(), templateID, sourceID)
		}
	}
	target := fmt.Sprintf("/catalog/templates/%d/edit?message=%s", templateID, url.QueryEscape("Item copiado. A cópia agora é exclusiva deste cardápio."))
	if err != nil {
		target = fmt.Sprintf("/catalog/templates/%d/items/new?type=danger&message=%s", templateID, url.QueryEscape(databaseErrorMessage(err)))
	}
	a.redirect(w, r, target, http.StatusSeeOther)
}

func (a *App) templateMenuItemToggle(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", http.StatusForbidden)
		return
	}
	templateID, err := templateIDFromPath(r)
	itemID, itemErr := pathID(r)
	if err == nil && itemErr == nil {
		err = a.store.ToggleTemplateMenuItem(r.Context(), templateID, itemID)
	} else if err == nil {
		err = itemErr
	}
	target := fmt.Sprintf("/catalog/templates/%d/edit?message=%s", templateID, url.QueryEscape("Status do item atualizado."))
	if err != nil {
		target = fmt.Sprintf("/catalog/templates/%d/edit?type=danger&message=%s", templateID, url.QueryEscape(databaseErrorMessage(err)))
	}
	a.redirect(w, r, target, http.StatusSeeOther)
}

func (a *App) menuItemForm(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", http.StatusForbidden)
		return
	}
	data := a.baseData(r, "Novo item do cardápio", "catalog")
	data.FormAction = "/catalog/items"
	data.MenuItem = models.MenuItem{
		Active:                true,
		CalculationType:       "menu_only",
		CalculationDivisor:    1,
		CalculationMultiplier: 1,
		CalculationWeight:     1,
	}
	if r.PathValue("id") != "" {
		id, err := pathID(r)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		item, err := a.store.GetMenuItem(r.Context(), id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		data.MenuItem = item
		data.IsEdit = true
		data.Title = "Editar item do cardápio"
		data.FormAction = fmt.Sprintf("/catalog/items/%d", id)
		data.MenuIngredients = item.Ingredients
	}
	data.MenuCategories, _ = a.store.ListMenuCategories(r.Context())
	data.Containers, _ = a.store.ListContainerTypes(r.Context(), false)
	data.Items, _ = a.store.ListInventory(r.Context(), "", "", false)
	data.Equipment, _ = a.store.EquipmentOptions(r.Context())
	data.RecipeIngredientOptions, _ = a.store.ListRecipeIngredientOptions(r.Context())
	selected := map[int64]models.EquipmentLink{}
	for _, link := range data.MenuItem.Equipment {
		selected[link.EquipmentID] = link
	}
	for i := range data.Equipment {
		if link, ok := selected[data.Equipment[i].EquipmentID]; ok {
			data.Equipment[i].Selected = true
			data.Equipment[i].Quantity = link.Quantity
			data.Equipment[i].Required = link.Required
		}
	}
	a.render(w, r, "menu_item_form", data)
}

func (a *App) parseMenuItem(r *http.Request, id int64) (models.MenuItem, []models.EquipmentLink, error) {
	if err := r.ParseForm(); err != nil {
		return models.MenuItem{}, nil, err
	}
	category := parseOptionalInt(r.FormValue("category_id"))
	item := models.MenuItem{
		ID:                       id,
		Name:                     strings.TrimSpace(r.FormValue("name")),
		Description:              strings.TrimSpace(r.FormValue("description")),
		ContainerTypeID:          parseOptionalInt(r.FormValue("container_type_id")),
		ContainerCapacity:        parseOptionalFloat(r.FormValue("container_capacity")),
		PanInventoryItemID:       parseOptionalInt(r.FormValue("pan_inventory_item_id")),
		TransportInventoryItemID: parseOptionalInt(r.FormValue("transport_inventory_item_id")),
		ResultInventoryItemID:    parseOptionalInt(r.FormValue("result_inventory_item_id")),
		CalculationType:          r.FormValue("menu_calculation_type"),
		CalculationGroup:         strings.TrimSpace(r.FormValue("calculation_group")),
		CalculationDivisor:       parseFloat(r.FormValue("calculation_divisor")),
		CalculationMultiplier:    parseFloat(r.FormValue("calculation_multiplier")),
		CalculationWeight:        parseFloat(r.FormValue("calculation_weight")),
		Active:                   true,
	}
	if item.CalculationType == "" {
		item.CalculationType = "menu_only"
	}
	if item.CalculationDivisor <= 0 {
		item.CalculationDivisor = 1
	}
	if item.CalculationMultiplier == 0 {
		item.CalculationMultiplier = 1
	}
	if item.CalculationWeight <= 0 {
		item.CalculationWeight = 1
	}
	if category.Valid {
		item.CategoryID = category.Int64
		if slug, slugErr := a.store.MenuCategorySlug(r.Context(), category.Int64); slugErr == nil && (slug == "main_courses" || slug == "sides") {
			item.ContainerTypeID = sql.NullInt64{}
			item.ContainerCapacity = sql.NullFloat64{}
		}
	}
	if item.Name == "" || item.CategoryID == 0 {
		return item, nil, fmt.Errorf("Nome e categoria são obrigatórios.")
	}
	options, _ := a.store.EquipmentOptions(r.Context())
	for i := range options {
		key := strconv.FormatInt(options[i].EquipmentID, 10)
		options[i].Selected = boolForm(r.FormValue("equipment_" + key))
		options[i].Quantity = parseFloat(r.FormValue("equipment_qty_" + key))
		if options[i].Quantity <= 0 {
			options[i].Quantity = 1
		}
		options[i].Required = !boolForm(r.FormValue("equipment_optional_" + key))
	}
	return item, options, nil
}

func (a *App) menuItemCreate(w http.ResponseWriter, r *http.Request) { a.saveMenuItem(w, r, 0) }
func (a *App) menuItemUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	a.saveMenuItem(w, r, id)
}
func (a *App) saveMenuItem(w http.ResponseWriter, r *http.Request, id int64) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", http.StatusForbidden)
		return
	}
	item, equipment, err := a.parseMenuItem(r, id)
	if err == nil {
		err = a.store.SaveMenuItem(r.Context(), &item, equipment)
	}
	if err != nil {
		data := a.baseData(r, "Revisar item do cardápio", "catalog")
		data.MenuItem = item
		data.Equipment = equipment
		data.Error = databaseErrorMessage(err)
		if strings.Contains(err.Error(), "obrigatórios") {
			data.Error = err.Error()
		}
		data.IsEdit = id > 0
		data.FormAction = "/catalog/items"
		if id > 0 {
			data.FormAction = fmt.Sprintf("/catalog/items/%d", id)
		}
		data.MenuCategories, _ = a.store.ListMenuCategories(r.Context())
		data.Containers, _ = a.store.ListContainerTypes(r.Context(), false)
		data.Items, _ = a.store.ListInventory(r.Context(), "", "", false)
		data.MenuIngredients, _ = a.store.ListMenuItemIngredients(r.Context(), id)
		data.RecipeIngredientOptions, _ = a.store.ListRecipeIngredientOptions(r.Context())
		a.render(w, r, "menu_item_form", data)
		return
	}
	a.redirect(w, r, "/catalog?message="+url.QueryEscape("Item do cardápio salvo."), http.StatusSeeOther)
}
func (a *App) menuItemToggle(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", 403)
		return
	}
	id, err := pathID(r)
	if err == nil {
		err = a.store.ToggleMenuItem(r.Context(), id)
	}
	target := "/catalog?message=" + url.QueryEscape("Status do item atualizado.")
	if err != nil {
		target = "/catalog?type=danger&message=" + url.QueryEscape(databaseErrorMessage(err))
	}
	a.redirect(w, r, target, http.StatusSeeOther)
}

func (a *App) containerForm(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", 403)
		return
	}
	data := a.baseData(r, "Novo recipiente", "catalog")
	data.FormAction = "/catalog/containers"
	data.Container = models.ContainerType{Active: true}
	if r.PathValue("id") != "" {
		id, err := pathID(r)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		item, err := a.store.GetContainerType(r.Context(), id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		data.Container = item
		data.IsEdit = true
		data.Title = "Editar recipiente"
		data.FormAction = fmt.Sprintf("/catalog/containers/%d", id)
	}
	data.Items, _ = a.store.ListInventory(r.Context(), "", "", false)
	a.render(w, r, "container_form", data)
}
func (a *App) parseContainer(r *http.Request, id int64) (models.ContainerType, error) {
	if err := r.ParseForm(); err != nil {
		return models.ContainerType{}, err
	}
	item := models.ContainerType{ID: id, Name: strings.TrimSpace(r.FormValue("name")), CapacityPortions: parseOptionalFloat(r.FormValue("capacity_portions")), Disposable: boolForm(r.FormValue("disposable")), RequiresLid: boolForm(r.FormValue("requires_lid")), IsDefault: boolForm(r.FormValue("is_default")), TransportNotes: strings.TrimSpace(r.FormValue("transport_notes")), InventoryItemID: parseOptionalInt(r.FormValue("inventory_item_id")), QuantityMode: r.FormValue("quantity_mode"), RequiredUtensilType: r.FormValue("required_utensil_type"), CustomUtensilName: strings.TrimSpace(r.FormValue("custom_utensil_name")), FixedQuantity: parseOptionalFloat(r.FormValue("fixed_quantity")), Active: true}
	if item.Name == "" {
		return item, fmt.Errorf("Nome é obrigatório.")
	}
	return item, nil
}
func (a *App) containerCreate(w http.ResponseWriter, r *http.Request) { a.saveContainer(w, r, 0) }
func (a *App) containerUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	a.saveContainer(w, r, id)
}
func (a *App) saveContainer(w http.ResponseWriter, r *http.Request, id int64) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", 403)
		return
	}
	item, err := a.parseContainer(r, id)
	if err == nil {
		err = a.store.SaveContainerType(r.Context(), &item)
	}
	if err != nil {
		data := a.baseData(r, "Revisar recipiente", "catalog")
		data.Container = item
		data.Error = databaseErrorMessage(err)
		if strings.Contains(err.Error(), "obrigatório") {
			data.Error = err.Error()
		}
		data.IsEdit = id > 0
		data.FormAction = "/catalog/containers"
		if id > 0 {
			data.FormAction = fmt.Sprintf("/catalog/containers/%d", id)
		}
		data.Items, _ = a.store.ListInventory(r.Context(), "", "", false)
		a.render(w, r, "container_form", data)
		return
	}
	a.redirect(w, r, "/catalog?message="+url.QueryEscape("Recipiente salvo."), http.StatusSeeOther)
}
func (a *App) containerToggle(w http.ResponseWriter, r *http.Request) {
	if err := a.requireAdmin(r); err != nil {
		http.Error(w, "Acesso restrito.", 403)
		return
	}
	id, err := pathID(r)
	if err == nil {
		err = a.store.ToggleContainerType(r.Context(), id)
	}
	target := "/catalog?message=" + url.QueryEscape("Status do recipiente atualizado.")
	if err != nil {
		target = "/catalog?type=danger&message=" + url.QueryEscape(databaseErrorMessage(err))
	}
	a.redirect(w, r, target, http.StatusSeeOther)
}

func (a *App) eventMenuForm(w http.ResponseWriter, r *http.Request) {
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
	data := a.baseData(r, "Cardápio do evento", "events")
	data.Event = event
	if hasSnapshot, _ := a.store.EventHasMenuSnapshot(r.Context(), id); hasSnapshot {
		data.SnapshotSections, err = a.store.EventMenuSnapshotSections(r.Context(), id)
		data.MenuItems, _ = a.store.ListMenuItems(r.Context(), false)
		data.Containers, _ = a.store.ListContainerTypes(r.Context(), false)
		data.Equipment, _ = a.store.EquipmentOptions(r.Context())
		data.CakeConfiguration, _ = a.store.GetEventCakeConfiguration(r.Context(), id)
		if err != nil {
			data.Error = databaseErrorMessage(err)
		}
		a.render(w, r, "event_menu_snapshot", data)
		return
	}
	data.EventMenu, err = a.store.EventMenuSelection(r.Context(), id)
	if err != nil {
		data.Error = databaseErrorMessage(err)
	}
	a.render(w, r, "event_menu", data)
}
func (a *App) eventMenuSave(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err = r.ParseForm(); err != nil {
		http.Error(w, "Dados inválidos.", 400)
		return
	}
	selection, _ := a.store.EventMenuSelection(r.Context(), id)
	var chosen []models.EventMenuItem
	for _, item := range selection {
		key := strconv.FormatInt(item.MenuItemID, 10)
		item.Selected = boolForm(r.FormValue("menu_" + key))
		item.Portions = int(parseFloat(r.FormValue("portions_" + key)))
		chosen = append(chosen, item)
	}
	err = a.store.SaveEventMenu(r.Context(), id, chosen)
	if err == nil {
		_, err = a.checklist.GenerateTracked(r.Context(), id, "event_menu_saved", currentUser(r).ID)
	}
	target := fmt.Sprintf("/events/%d?message=%s", id, url.QueryEscape("Cardápio salvo e checklist atualizada."))
	if err != nil {
		target = fmt.Sprintf("/events/%d/menu?type=danger&message=%s", id, url.QueryEscape(databaseErrorMessage(err)))
	}
	a.redirect(w, r, target, http.StatusSeeOther)
}

var _ = sql.ErrNoRows
