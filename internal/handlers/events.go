package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"buffetflow/internal/models"
)

func (a *App) events(writer http.ResponseWriter, request *http.Request) {
	data := a.baseData(request, "Eventos", "events")
	data.Query = request.URL.Query().Get("q")
	events, err := a.store.ListEvents(request.Context(), data.Query)
	if err != nil {
		data.Error = databaseErrorMessage(err)
	} else {
		data.Events = events
	}
	a.render(writer, request, "events", data)
}

func (a *App) eventForm(writer http.ResponseWriter, request *http.Request) {
	data := a.baseData(request, "Novo evento", "events")
	data.FormAction = "/events"
	now := time.Now().In(a.location).AddDate(0, 0, 7)
	data.Event = models.Event{StartsAt: time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, a.location), EndsAt: time.Date(now.Year(), now.Month(), now.Day()+1, 2, 0, 0, 0, a.location), GuestCount: 100, SafetyMarginPercent: 0, UsesGlassware: true}
	if request.PathValue("id") != "" {
		id, err := pathID(request)
		if err != nil {
			http.NotFound(writer, request)
			return
		}
		event, err := a.store.GetEvent(request.Context(), id)
		if err != nil {
			http.NotFound(writer, request)
			return
		}
		data.Title, data.Event, data.IsEdit, data.FormAction = "Editar evento", event, true, fmt.Sprintf("/events/%d", id)
	}
	data.EventMenu, _ = a.store.EventMenuSelection(request.Context(), data.Event.ID)
	data.MenuTemplates, _ = a.store.ListMenuTemplates(request.Context(), false)
	var modelID int64
	var selectedItemIDs, serviceIDs []int64
	if data.Event.ID > 0 {
		modelID, selectedItemIDs, _ = a.store.EventMenuModelSelection(request.Context(), data.Event.ID)
		serviceIDs, _ = a.store.EventServiceModelIDs(request.Context(), data.Event.ID)
		customItems, _ := a.store.EventMenuModelCustomItems(request.Context(), data.Event.ID)
		data.ModelCustomItems = strings.Join(customItems, "\n")
		data.MenuModelSnapshotVersion, data.MenuModelCurrentVersion, _, _ = a.store.EventMenuModelStatus(request.Context(), data.Event.ID)
		data.MenuModelOutdated = data.MenuModelCurrentVersion > data.MenuModelSnapshotVersion
	}
	if data.Event.ID > 0 {
		data.DecorationProfile, _ = a.store.GetDecorationProfile(request.Context(), data.Event.ID)
	}
	data.Decorations, _ = a.store.EventDecorationSelectionForWindow(request.Context(), data.Event.ID, data.Event.StartsAt, data.Event.EndsAt)
	a.populateEventModelData(request, &data, modelID, selectedItemIDs, serviceIDs)
	data.KitchenCooks, _ = a.store.ListKitchenCooks(request.Context(), false)
	if data.Event.ID > 0 && modelID > 0 {
		configurations, _ := a.store.EventMenuModelItemConfigurations(request.Context(), data.Event.ID)
		overlayMenuModelConfigurations(data.ModelSections, configurations)
	}
	data.MenuCustomized = data.IsEdit
	a.render(writer, request, "event_form", data)
}

func (a *App) parseEventForm(request *http.Request, id int64) (models.Event, error) {
	if err := request.ParseForm(); err != nil {
		return models.Event{}, err
	}
	guestCount, err := parsePositiveInt(request.FormValue("guest_count"))
	if err != nil {
		return models.Event{}, fmt.Errorf("Informe uma quantidade de convidados válida.")
	}
	starts, err := time.ParseInLocation("2006-01-02T15:04", request.FormValue("starts_at"), a.location)
	if err != nil {
		return models.Event{}, fmt.Errorf("Informe a data e o horário de início.")
	}
	ends, err := time.ParseInLocation("2006-01-02T15:04", request.FormValue("ends_at"), a.location)
	if err != nil || !ends.After(starts) {
		return models.Event{}, fmt.Errorf("O término precisa ser posterior ao início.")
	}
	event := models.Event{
		ID: id, TemplateID: parseOptionalInt(request.FormValue("template_id")), ClientName: strings.TrimSpace(request.FormValue("client_name")), Name: strings.TrimSpace(request.FormValue("name")),
		Venue: strings.TrimSpace(request.FormValue("venue")), StartsAt: starts, EndsAt: ends, GuestCount: guestCount,
		HasDecoration: boolForm(request.FormValue("has_decoration")), HasWelcomeDrinks: boolForm(request.FormValue("has_welcome_drinks")), HasCoffeeTable: boolForm(request.FormValue("has_coffee_table")),
		StartersNotes: strings.TrimSpace(request.FormValue("starters_notes")), MainCoursesNotes: strings.TrimSpace(request.FormValue("main_courses_notes")),
		SidesNotes: strings.TrimSpace(request.FormValue("sides_notes")), BeveragesNotes: strings.TrimSpace(request.FormValue("beverages_notes")),
		CoffeeTableNotes: strings.TrimSpace(request.FormValue("coffee_table_notes")), CakeNotes: strings.TrimSpace(request.FormValue("cake_notes")),
		SweetsNotes: strings.TrimSpace(request.FormValue("sweets_notes")), DessertsNotes: strings.TrimSpace(request.FormValue("desserts_notes")),
		Notes: strings.TrimSpace(request.FormValue("notes")), SafetyMarginPercent: parseFloat(request.FormValue("safety_margin_percent")),
		WaiterOverride: parseOptionalInt(request.FormValue("waiter_override")), CoordinatorOverride: parseOptionalInt(request.FormValue("coordinator_override")),
		LeaderOverride: parseOptionalInt(request.FormValue("leader_override")), CoLeaderOverride: parseOptionalInt(request.FormValue("co_leader_override")),
		AdditionalGuestMarginOverride: parseOptionalFloat(request.FormValue("additional_guest_margin_override")), UsesGlassware: boolForm(request.FormValue("uses_glassware")),
		KitchenCookID: parseOptionalInt(request.FormValue("kitchen_cook_id")),
	}
	if event.ClientName == "" || event.Name == "" || event.Venue == "" {
		return event, fmt.Errorf("Cliente, nome do evento e local são obrigatórios.")
	}
	if event.SafetyMarginPercent < 0 || event.SafetyMarginPercent > 100 {
		return event, fmt.Errorf("A margem deve estar entre 0%% e 100%%.")
	}
	return event, nil
}

func parsePositiveInt(value string) (int, error) {
	var result int
	_, err := fmt.Sscan(value, &result)
	if err != nil || result <= 0 {
		return 0, fmt.Errorf("invalid number")
	}
	return result, nil
}

func (a *App) eventCreate(writer http.ResponseWriter, request *http.Request) {
	a.saveEvent(writer, request, 0)
}
func (a *App) eventUpdate(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	a.saveEvent(writer, request, id)
}

func (a *App) saveEvent(writer http.ResponseWriter, request *http.Request, id int64) {
	event, err := a.parseEventForm(request, id)
	menuModelID := int64(0)
	if raw := request.FormValue("menu_model_id"); raw != "" {
		menuModelID, _ = strconv.ParseInt(raw, 10, 64)
	}
	modelItemIDs := formInt64Values(request, "model_item_ids")
	serviceModelIDs := formInt64Values(request, "service_model_ids")
	customItems := formLines(request.FormValue("model_custom_items"))
	itemNames := parseMenuModelItemNames(request)
	sectionCustomItems := parseMenuModelSectionCustomItems(request)
	itemConfigurations := parseMenuModelConfigurations(request)
	var decorationSelection []models.EventDecoration
	if err == nil {
		decorationSelection, err = a.store.EventDecorationSelectionForWindow(request.Context(), id, event.StartsAt, event.EndsAt)
		if err == nil {
			decorationSelection, err = applyEventDecorationForm(request, decorationSelection, event.HasDecoration)
		}
	}
	if err == nil && event.KitchenCookID.Valid {
		if cookErr := a.store.ValidateKitchenCook(request.Context(), event.KitchenCookID.Int64); cookErr != nil {
			err = fmt.Errorf("Selecione uma cozinheira ativa ou deixe o campo em branco.")
		}
	}
	if err == nil && menuModelID > 0 {
		err = a.store.ValidateMenuModelSelections(request.Context(), menuModelID, modelItemIDs)
	}
	selection, _ := a.store.EventMenuSelection(request.Context(), id)
	if menuModelID > 0 && err == nil {
		menuItemIDs, modelErr := a.store.MenuModelMenuItemIDs(request.Context(), menuModelID, modelItemIDs)
		if modelErr != nil {
			err = modelErr
		} else {
			selection = applySelectedMenuIDs(selection, menuItemIDs, event.GuestCount)
		}
	} else if event.TemplateID.Valid && len(request.Form["menu_item_ids"]) == 0 && request.FormValue("menu_customized") != "1" {
		selection = a.applyMenuTemplate(request, selection, event.TemplateID.Int64, event.GuestCount)
		if template, templateErr := a.store.GetMenuTemplate(request.Context(), event.TemplateID.Int64); templateErr == nil {
			event.HasDecoration = template.HasDecoration
			event.HasWelcomeDrinks = template.HasWelcomeDrinks
			event.HasCoffeeTable = template.HasCoffeeTable
		}
	} else {
		selection = applyEventMenuForm(request, selection, event.GuestCount)
	}
	applyMenuNotes(&event, selection)
	if err != nil {
		data := a.baseData(request, "Revisar evento", "events")
		data.Event = event
		data.Error = err.Error()
		data.IsEdit = id > 0
		if id > 0 {
			data.FormAction = fmt.Sprintf("/events/%d", id)
		} else {
			data.FormAction = "/events"
		}
		data.EventMenu = selection
		data.MenuTemplates, _ = a.store.ListMenuTemplates(request.Context(), false)
		data.KitchenCooks, _ = a.store.ListKitchenCooks(request.Context(), false)
		data.MenuCustomized = id > 0 || request.FormValue("menu_customized") == "1"
		data.ModelCustomItems = request.FormValue("model_custom_items")
		data.Decorations = decorationSelection
		a.populateEventModelData(request, &data, menuModelID, modelItemIDs, serviceModelIDs)
		a.render(writer, request, "event_form", data)
		return
	}
	user := currentUser(request)
	if err := a.store.SaveEvent(request.Context(), &event, user.ID); err != nil {
		data := a.baseData(request, "Revisar evento", "events")
		data.Event = event
		data.Error = databaseErrorMessage(err)
		data.IsEdit = id > 0
		data.FormAction = "/events"
		if id > 0 {
			data.FormAction = fmt.Sprintf("/events/%d", id)
		}
		data.EventMenu = selection
		data.MenuTemplates, _ = a.store.ListMenuTemplates(request.Context(), false)
		data.KitchenCooks, _ = a.store.ListKitchenCooks(request.Context(), false)
		data.MenuCustomized = id > 0 || request.FormValue("menu_customized") == "1"
		data.ModelCustomItems = request.FormValue("model_custom_items")
		data.Decorations = decorationSelection
		a.populateEventModelData(request, &data, menuModelID, modelItemIDs, serviceModelIDs)
		a.render(writer, request, "event_form", data)
		return
	}
	if event.HasDecoration || id > 0 {
		profile, _ := a.store.GetDecorationProfile(request.Context(), event.ID)
		profile.EventID = event.ID
		profile.Style = strings.TrimSpace(request.FormValue("decoration_style"))
		profile.Description = strings.TrimSpace(request.FormValue("decoration_description"))
		profile.PrimaryColors = strings.TrimSpace(request.FormValue("decoration_primary_colors"))
		profile.Theme = strings.TrimSpace(request.FormValue("decoration_theme"))
		profile.Notes = strings.TrimSpace(request.FormValue("decoration_notes"))
		profile.ResponsibleName = strings.TrimSpace(request.FormValue("decoration_responsible"))
		profile.Active = event.HasDecoration
		if event.HasDecoration || profile.ID > 0 {
			if profileErr := a.store.SaveDecorationProfile(request.Context(), &profile, user.ID); profileErr != nil {
				a.redirect(writer, request, fmt.Sprintf("/events/%d/edit?type=danger&message=%s", event.ID, url.QueryEscape(databaseErrorMessage(profileErr))), http.StatusSeeOther)
				return
			}
		}
	}
	if event.HasDecoration {
		if decorationErr := a.store.SaveEventDecorations(request.Context(), event.ID, decorationSelection); decorationErr != nil {
			a.redirect(writer, request, fmt.Sprintf("/events/%d/edit?type=danger&message=%s", event.ID, url.QueryEscape(databaseErrorMessage(decorationErr))), http.StatusSeeOther)
			return
		}
	}
	for index := range selection {
		selection[index].EventID = event.ID
	}
	if err := a.store.SaveEventMenu(request.Context(), event.ID, selection); err != nil {
		data := a.baseData(request, "Revisar evento", "events")
		data.Event = event
		data.EventMenu = selection
		data.Error = databaseErrorMessage(err)
		data.IsEdit = true
		data.FormAction = fmt.Sprintf("/events/%d", event.ID)
		data.MenuTemplates, _ = a.store.ListMenuTemplates(request.Context(), false)
		data.KitchenCooks, _ = a.store.ListKitchenCooks(request.Context(), false)
		data.MenuCustomized = true
		data.ModelCustomItems = request.FormValue("model_custom_items")
		data.Decorations = decorationSelection
		a.populateEventModelData(request, &data, menuModelID, modelItemIDs, serviceModelIDs)
		a.render(writer, request, "event_form", data)
		return
	}
	if menuModelID > 0 {
		currentModelID, _, _ := a.store.EventMenuModelSelection(request.Context(), event.ID)
		if currentModelID != menuModelID {
			err = a.store.ApplyMenuModelSnapshotWithCustomizations(request.Context(), event.ID, menuModelID, modelItemIDs, customItems, itemNames, sectionCustomItems, user.ID)
		}
		if err != nil {
			a.redirect(writer, request, fmt.Sprintf("/events/%d/edit?type=danger&message=%s", event.ID, url.QueryEscape(databaseErrorMessage(err))), http.StatusSeeOther)
			return
		}
		if err := a.store.UpdateEventMenuSnapshotConfigurations(request.Context(), event.ID, itemConfigurations, user.ID); err != nil {
			a.redirect(writer, request, fmt.Sprintf("/events/%d/edit?type=danger&message=%s", event.ID, url.QueryEscape(databaseErrorMessage(err))), http.StatusSeeOther)
			return
		}
	} else if err := a.store.ClearMenuModelSnapshot(request.Context(), event.ID); err != nil {
		a.redirect(writer, request, fmt.Sprintf("/events/%d/edit?type=danger&message=%s", event.ID, url.QueryEscape(databaseErrorMessage(err))), http.StatusSeeOther)
		return
	}
	if err := a.store.ApplyServiceSnapshots(request.Context(), event.ID, serviceModelIDs, user.ID); err != nil {
		a.redirect(writer, request, fmt.Sprintf("/events/%d/edit?type=danger&message=%s", event.ID, url.QueryEscape(databaseErrorMessage(err))), http.StatusSeeOther)
		return
	}
	if _, err := a.checklist.GenerateTracked(request.Context(), event.ID, "event_saved", user.ID); err != nil {
		a.logger.Error("generate checklist after event save", "event_id", event.ID, "error", err)
		target := fmt.Sprintf("/events/%d?type=danger&message=%s", event.ID, url.QueryEscape("Evento salvo, mas a checklist não pôde ser recalculada. Revise as regras."))
		a.redirect(writer, request, target, http.StatusSeeOther)
		return
	}
	target := fmt.Sprintf("/events/%d?message=%s", event.ID, url.QueryEscape("Evento salvo e checklist recalculada."))
	a.redirect(writer, request, target, http.StatusSeeOther)
}

func formInt64Values(request *http.Request, name string) []int64 {
	var result []int64
	for _, raw := range request.Form[name] {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil && id > 0 {
			result = append(result, id)
		}
	}
	return result
}

func applyEventDecorationForm(request *http.Request, items []models.EventDecoration, enabled bool) ([]models.EventDecoration, error) {
	selected := map[int64]bool{}
	for _, id := range formInt64Values(request, "decoration_ids") {
		selected[id] = true
	}
	for index := range items {
		item := &items[index]
		item.Selected = enabled && selected[item.DecorationID]
		if !item.Selected {
			continue
		}
		if !item.AvailabilityTracked || !item.Selectable {
			return items, fmt.Errorf("%s não está disponível no estoque para a data do evento.", item.Name)
		}
		quantity := parseFloat(request.FormValue(fmt.Sprintf("decoration_quantity_%d", item.DecorationID)))
		if quantity <= 0 {
			quantity = item.Quantity
		}
		if quantity <= 0 || quantity > item.AvailableQuantity+0.0001 {
			return items, fmt.Errorf("%s possui somente %s disponível para a data do evento.", item.Name, strconv.FormatFloat(item.AvailableQuantity, 'f', -1, 64))
		}
		item.Quantity = quantity
	}
	return items, nil
}

func formLines(value string) []string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	var result []string
	for _, line := range strings.Split(value, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			result = append(result, line)
		}
	}
	return result
}

func parseMenuModelItemNames(request *http.Request) map[int64]string {
	result := map[int64]string{}
	for key, values := range request.Form {
		if !strings.HasPrefix(key, "model_item_name_") || len(values) == 0 {
			continue
		}
		id, err := strconv.ParseInt(strings.TrimPrefix(key, "model_item_name_"), 10, 64)
		if err == nil && id > 0 && strings.TrimSpace(values[0]) != "" {
			result[id] = strings.TrimSpace(values[0])
		}
	}
	return result
}

func parseMenuModelSectionCustomItems(request *http.Request) map[int64][]string {
	result := map[int64][]string{}
	for key, values := range request.Form {
		if !strings.HasPrefix(key, "model_custom_items_section_") || len(values) == 0 {
			continue
		}
		sectionID, err := strconv.ParseInt(strings.TrimPrefix(key, "model_custom_items_section_"), 10, 64)
		if err == nil && sectionID > 0 {
			result[sectionID] = formLines(values[0])
		}
	}
	return result
}

func applySelectedMenuIDs(selection []models.EventMenuItem, ids []int64, guests int) []models.EventMenuItem {
	selected := map[int64]bool{}
	for _, id := range ids {
		selected[id] = true
	}
	for index := range selection {
		selection[index].Selected = selected[selection[index].MenuItemID]
		if selection[index].Selected {
			selection[index].Portions = guests
		}
	}
	return selection
}

func markMenuModelSelections(sections []models.MenuModelSection, itemIDs []int64) {
	selected := map[int64]bool{}
	for _, id := range itemIDs {
		selected[id] = true
	}
	for sectionIndex := range sections {
		for itemIndex := range sections[sectionIndex].Items {
			item := &sections[sectionIndex].Items[itemIndex]
			item.Selected = (item.Included && itemIDs == nil) || selected[item.ID]
		}
		for groupIndex := range sections[sectionIndex].ChoiceGroups {
			for itemIndex := range sections[sectionIndex].ChoiceGroups[groupIndex].Items {
				item := &sections[sectionIndex].ChoiceGroups[groupIndex].Items[itemIndex]
				item.Selected = (item.Included && itemIDs == nil) || selected[item.ID]
			}
		}
	}
}

func overlayMenuModelConfigurations(sections []models.MenuModelSection, configurations map[int64]models.EventMenuItemConfiguration) {
	for sectionIndex := range sections {
		for itemIndex := range sections[sectionIndex].Items {
			if configuration, exists := configurations[sections[sectionIndex].Items[itemIndex].ID]; exists {
				sections[sectionIndex].Items[itemIndex].Portions = configuration.Portions
				sections[sectionIndex].Items[itemIndex].ContainerTypeID = configuration.ContainerTypeID
			}
		}
		for groupIndex := range sections[sectionIndex].ChoiceGroups {
			for itemIndex := range sections[sectionIndex].ChoiceGroups[groupIndex].Items {
				if configuration, exists := configurations[sections[sectionIndex].ChoiceGroups[groupIndex].Items[itemIndex].ID]; exists {
					sections[sectionIndex].ChoiceGroups[groupIndex].Items[itemIndex].Portions = configuration.Portions
					sections[sectionIndex].ChoiceGroups[groupIndex].Items[itemIndex].ContainerTypeID = configuration.ContainerTypeID
				}
			}
		}
	}
}

func parseMenuModelConfigurations(request *http.Request) map[int64]models.EventMenuItemConfiguration {
	result := map[int64]models.EventMenuItemConfiguration{}
	for key, values := range request.Form {
		if len(values) == 0 {
			continue
		}
		if strings.HasPrefix(key, "model_portions_") {
			itemID, err := strconv.ParseInt(strings.TrimPrefix(key, "model_portions_"), 10, 64)
			if err != nil || itemID <= 0 {
				continue
			}
			configuration := result[itemID]
			configuration.TemplateItemID = itemID
			if portions, err := strconv.ParseFloat(values[0], 64); err == nil && portions > 0 {
				configuration.Portions = sql.NullFloat64{Float64: portions, Valid: true}
			}
			result[itemID] = configuration
			continue
		}
		if strings.HasPrefix(key, "model_container_") {
			itemID, err := strconv.ParseInt(strings.TrimPrefix(key, "model_container_"), 10, 64)
			if err != nil || itemID <= 0 {
				continue
			}
			configuration := result[itemID]
			configuration.TemplateItemID = itemID
			configuration.ContainerTypeID = parseOptionalInt(values[0])
			result[itemID] = configuration
		}
	}
	return result
}

func (a *App) populateEventModelData(request *http.Request, data *PageData, modelID int64, itemIDs, serviceIDs []int64) {
	data.MenuModels, _ = a.store.ListMenuModels(request.Context(), false)
	data.ServiceModels, _ = a.store.ListServiceModels(request.Context(), false)
	data.CurrentMenuModelID = modelID
	data.SelectedServiceIDs = map[int64]bool{}
	data.Containers, _ = a.store.ListContainerTypes(request.Context(), false)
	for _, id := range serviceIDs {
		data.SelectedServiceIDs[id] = true
	}
	if modelID > 0 {
		data.ModelSections, _ = a.store.MenuModelSections(request.Context(), modelID)
		data.ModelSections = withoutCoffeeTableSection(data.ModelSections)
		markMenuModelSelections(data.ModelSections, itemIDs)
		overlayMenuModelConfigurations(data.ModelSections, parseMenuModelConfigurations(request))
	}
}

func withoutCoffeeTableSection(sections []models.MenuModelSection) []models.MenuModelSection {
	filtered := make([]models.MenuModelSection, 0, len(sections))
	for _, section := range sections {
		name := strings.ToLower(strings.TrimSpace(section.Name))
		if name == "mesa de café" || name == "mesa do café" {
			continue
		}
		filtered = append(filtered, section)
	}
	return filtered
}

func (a *App) menuModelPreview(writer http.ResponseWriter, request *http.Request) {
	modelID, _ := strconv.ParseInt(request.URL.Query().Get("menu_model_id"), 10, 64)
	data := a.baseData(request, "Prévia do cardápio", "events")
	data.CurrentMenuModelID = modelID
	if modelID > 0 {
		data.ModelSections, _ = a.store.MenuModelSections(request.Context(), modelID)
		data.ModelSections = withoutCoffeeTableSection(data.ModelSections)
		markMenuModelSelections(data.ModelSections, nil)
	}
	data.Containers, _ = a.store.ListContainerTypes(request.Context(), false)
	a.render(writer, request, "menu_model_preview", data)
}

func (a *App) applyMenuTemplate(request *http.Request, selection []models.EventMenuItem, templateID int64, guestCount int) []models.EventMenuItem {
	templateItems, err := a.store.MenuTemplateSelection(request.Context(), templateID)
	if err != nil {
		return selection
	}
	selected := map[int64]bool{}
	for _, item := range templateItems {
		selected[item.MenuItemID] = item.Selected
	}
	for index := range selection {
		selection[index].Selected = selected[selection[index].MenuItemID]
		if selection[index].Selected {
			selection[index].Portions = guestCount
		}
	}
	return selection
}

func applyEventMenuForm(request *http.Request, selection []models.EventMenuItem, guestCount int) []models.EventMenuItem {
	selected := map[int64]bool{}
	for _, raw := range request.Form["menu_item_ids"] {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			selected[id] = true
		}
	}
	for i := range selection {
		selection[i].Selected = selected[selection[i].MenuItemID]
		if selection[i].Selected {
			selection[i].Portions = guestCount
		}
	}
	return selection
}

func applyMenuNotes(event *models.Event, selection []models.EventMenuItem) {
	groups := map[string][]string{}
	for _, item := range selection {
		if item.Selected {
			groups[item.CategoryName] = append(groups[item.CategoryName], item.MenuItemName)
		}
	}
	event.StartersNotes = strings.Join(groups["Entradas"], "\n")
	event.MainCoursesNotes = strings.Join(groups["Pratos principais"], "\n")
	event.SidesNotes = strings.Join(groups["Acompanhamentos"], "\n")
	event.BeveragesNotes = strings.Join(groups["Bebidas"], "\n")
}

func (a *App) eventShow(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	data := a.baseData(request, "Checklist do evento", "events")
	event, err := a.store.GetEvent(request.Context(), id)
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	data.Event = event
	checklist, err := a.store.GetChecklistByEvent(request.Context(), id)
	if err == sql.ErrNoRows {
		checklist, err = a.checklist.GenerateTracked(request.Context(), id, "initial_generation", currentUser(request).ID)
	}
	if err != nil {
		data.Error = databaseErrorMessage(err)
	} else {
		data.Checklist = checklist
		data.Groups = groupChecklist(checklist.Items)
		data.Recalculation, _ = a.store.LatestEventRecalculation(request.Context(), id)
	}
	a.render(writer, request, "event_show", data)
}

func groupChecklist(items []models.ChecklistItem) []models.ChecklistGroup {
	var groups []models.ChecklistGroup
	for _, item := range items {
		if len(groups) == 0 || groups[len(groups)-1].Category != item.CategoryName {
			groups = append(groups, models.ChecklistGroup{Category: item.CategoryName})
		}
		groups[len(groups)-1].Items = append(groups[len(groups)-1].Items, item)
	}
	return groups
}

func (a *App) eventGenerate(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	if _, err = a.checklist.GenerateTracked(request.Context(), id, "manual_recalculation", currentUser(request).ID); err != nil {
		a.redirect(writer, request, fmt.Sprintf("/events/%d?type=danger&message=%s", id, url.QueryEscape(databaseErrorMessage(err))), http.StatusSeeOther)
		return
	}
	a.redirect(writer, request, fmt.Sprintf("/events/%d?message=%s", id, url.QueryEscape("Checklist recalculada sem duplicar itens.")), http.StatusSeeOther)
}

func (a *App) eventReserve(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	_ = request.ParseForm()
	force := boolForm(request.FormValue("confirm_shortage"))
	user := currentUser(request)
	err = a.store.ReserveEvent(request.Context(), id, user.ID, force)
	if err != nil && strings.Contains(err.Error(), "shortage_confirmation_required") {
		target := fmt.Sprintf("/events/%d?type=danger&message=%s", id, url.QueryEscape("Há itens faltantes. Use “Reservar mesmo assim” para registrar a pendência."))
		a.redirect(writer, request, target, http.StatusSeeOther)
		return
	}
	if err != nil {
		a.redirect(writer, request, fmt.Sprintf("/events/%d?type=danger&message=%s", id, url.QueryEscape(databaseErrorMessage(err))), http.StatusSeeOther)
		return
	}
	a.redirect(writer, request, fmt.Sprintf("/events/%d?message=%s", id, url.QueryEscape("Estoque reservado para o período do evento.")), http.StatusSeeOther)
}

func (a *App) eventDuplicate(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	user := currentUser(request)
	newID, err := a.store.DuplicateEvent(request.Context(), id, user.ID)
	if err == nil {
		_, err = a.checklist.GenerateTracked(request.Context(), newID, "event_duplicated", user.ID)
	}
	if err != nil {
		a.redirect(writer, request, "/events?type=danger&message="+url.QueryEscape(databaseErrorMessage(err)), http.StatusSeeOther)
		return
	}
	a.redirect(writer, request, fmt.Sprintf("/events/%d/edit?message=%s", newID, url.QueryEscape("Cópia criada. Revise a data e os dados.")), http.StatusSeeOther)
}

func (a *App) eventMenuModelCompare(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	data := a.baseData(request, "Comparar modelo do evento", "events")
	data.Event, err = a.store.GetEvent(request.Context(), id)
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	modelID, _, err := a.store.EventMenuModelSelection(request.Context(), id)
	if err != nil || modelID == 0 {
		http.NotFound(writer, request)
		return
	}
	data.MenuModel, err = a.store.GetMenuModel(request.Context(), modelID)
	if err == nil {
		data.MenuModelSnapshotVersion, data.MenuModelCurrentVersion, _, err = a.store.EventMenuModelStatus(request.Context(), id)
	}
	if err == nil {
		data.ModelDifferences, err = a.store.CompareEventMenuModel(request.Context(), id)
	}
	if err != nil {
		data.Error = databaseErrorMessage(err)
	}
	data.MenuModelOutdated = data.MenuModelCurrentVersion > data.MenuModelSnapshotVersion
	a.render(writer, request, "event_menu_model_compare", data)
}

func (a *App) eventMenuModelReapply(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	modelID, selectedIDs, err := a.store.EventMenuModelSelection(request.Context(), id)
	customItems, customErr := a.store.EventMenuModelCustomItems(request.Context(), id)
	configurations, configErr := a.store.EventMenuModelItemConfigurations(request.Context(), id)
	if err == nil {
		err = customErr
	}
	if err == nil {
		err = configErr
	}
	user := currentUser(request)
	if err == nil {
		err = a.store.ApplyMenuModelSnapshot(request.Context(), id, modelID, selectedIDs, customItems, user.ID)
	}
	if err == nil {
		err = a.store.UpdateEventMenuSnapshotConfigurations(request.Context(), id, configurations, user.ID)
	}
	if err == nil {
		_, err = a.checklist.GenerateTracked(request.Context(), id, "menu_model_reapplied", currentUser(request).ID)
	}
	if err != nil {
		target := fmt.Sprintf("/events/%d/menu-model/compare?type=danger&message=%s", id, url.QueryEscape(databaseErrorMessage(err)))
		a.redirect(writer, request, target, http.StatusSeeOther)
		return
	}
	target := fmt.Sprintf("/events/%d/edit?message=%s", id, url.QueryEscape("Versão atual aplicada manualmente. Personalizações, quantidades e recipientes foram preservados."))
	a.redirect(writer, request, target, http.StatusSeeOther)
}

func (a *App) eventCancel(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	user := currentUser(request)
	if err = a.store.CancelEvent(request.Context(), id, user.ID); err != nil {
		a.redirect(writer, request, fmt.Sprintf("/events/%d?type=danger&message=%s", id, url.QueryEscape(databaseErrorMessage(err))), http.StatusSeeOther)
		return
	}
	a.redirect(writer, request, "/events?message="+url.QueryEscape("Evento cancelado e reservas liberadas."), http.StatusSeeOther)
}

func (a *App) checklistStatus(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	_ = request.ParseForm()
	user := currentUser(request)
	eventID := request.FormValue("event_id")
	if err = a.store.UpdateChecklistItemStatus(request.Context(), id, request.FormValue("status"), user.ID); err != nil {
		a.redirect(writer, request, "/events/"+eventID+"?type=danger&message="+url.QueryEscape(databaseErrorMessage(err)), http.StatusSeeOther)
		return
	}
	a.redirect(writer, request, "/events/"+eventID+"?message="+url.QueryEscape("Status atualizado."), http.StatusSeeOther)
}

func (a *App) checklistOverride(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	_ = request.ParseForm()
	user := currentUser(request)
	eventID := request.FormValue("event_id")
	err = a.store.OverrideChecklistItem(request.Context(), id, parseFloat(request.FormValue("quantity")), request.FormValue("reason"), user.ID)
	target := "/events/" + eventID + "?message=" + url.QueryEscape("Quantidade alterada somente neste evento.")
	if err != nil {
		target = "/events/" + eventID + "?type=danger&message=" + url.QueryEscape("Informe a quantidade e o motivo da alteração.")
	}
	a.redirect(writer, request, target, http.StatusSeeOther)
}
