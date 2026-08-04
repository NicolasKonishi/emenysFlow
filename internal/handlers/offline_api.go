package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"buffetflow/internal/models"
)

type offlineEventBundle struct {
	Event      models.Event                      `json:"event"`
	Checklist  models.Checklist                  `json:"checklist"`
	Menu       []models.EventMenuSnapshotSection `json:"menu"`
	ServiceIDs []int64                           `json:"service_ids"`
	Shortages  []models.ChecklistShortage        `json:"shortages"`
}

func (a *App) offlineBootstrap(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	events, err := a.store.ListEvents(r.Context(), "")
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": databaseErrorMessage(err)})
		return
	}
	bundles := make([]offlineEventBundle, 0, len(events))
	for _, event := range events {
		checklist, _ := a.store.GetChecklistByEvent(r.Context(), event.ID)
		menu, _ := a.store.EventMenuSnapshotSections(r.Context(), event.ID)
		services, _ := a.store.EventServiceModelIDs(r.Context(), event.ID)
		shortages, _ := a.store.ListEventShortages(r.Context(), event.ID, true)
		bundles = append(bundles, offlineEventBundle{Event: event, Checklist: checklist, Menu: menu, ServiceIDs: services, Shortages: shortages})
	}
	inventory, _ := a.store.ListInventory(r.Context(), "", "", false)
	menus, _ := a.store.ListMenuModels(r.Context(), false)
	services, _ := a.store.ListServiceModels(r.Context(), false)
	settings, _ := a.store.OperationalSettings(r.Context())
	writeJSON(w, 200, map[string]any{"schema_version": 1, "synced_at": time.Now().UTC(), "offline_access_expires_at": time.Now().Add(12 * time.Hour).UTC(), "user": user, "events": bundles, "inventory": inventory, "menu_models": menus, "service_models": services, "operational_settings": settings})
}

func (a *App) syncOperations(w http.ResponseWriter, r *http.Request) {
	var requests []models.SyncOperationRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&requests); err != nil {
		writeJSON(w, 400, map[string]any{"error": "Conteúdo de sincronização inválido."})
		return
	}
	if len(requests) > 100 {
		writeJSON(w, 400, map[string]any{"error": "Envie no máximo 100 operações por vez."})
		return
	}
	user := currentUser(r)
	results := make([]models.SyncOperationResult, 0, len(requests))
	for _, request := range requests {
		if request.DeviceID == "" || request.ClientOperationID == "" {
			results = append(results, models.SyncOperationResult{ClientOperationID: request.ClientOperationID, Status: "failed", Error: "Identificação da operação ausente."})
			continue
		}
		_ = a.store.RegisterSyncDevice(r.Context(), request.DeviceID, r.UserAgent(), user.ID)
		if existing, found, _ := a.store.ExistingSyncOperation(r.Context(), request.ClientOperationID); found {
			results = append(results, existing)
			continue
		}
		result := a.applySyncOperation(r, request, user)
		_ = a.store.RecordSyncOperation(r.Context(), request, user.ID, result)
		_ = a.store.MarkDeviceSynced(r.Context(), request.DeviceID)
		results = append(results, result)
	}
	writeJSON(w, 200, map[string]any{"results": results, "synced_at": time.Now().UTC()})
}

func (a *App) applySyncOperation(r *http.Request, request models.SyncOperationRequest, user models.User) models.SyncOperationResult {
	result := models.SyncOperationResult{ClientOperationID: request.ClientOperationID, EntityID: request.EntityID}
	var payload map[string]any
	if err := json.Unmarshal(request.Payload, &payload); err != nil {
		result.Status = "failed"
		result.Error = "Conteúdo da operação inválido."
		return result
	}
	eventID := int64(numberValue(payload["event_id"]))
	var err error
	switch request.OperationType {
	case "update_quantity":
		stage := stringValue(payload["stage"])
		quantity := numberValue(payload["quantity"])
		notes := stringValue(payload["notes"])
		result.Version, err = a.store.SaveOperationalQuantity(r.Context(), eventID, request.EntityID, stage, quantity, notes, user.ID, request.BaseVersion)
	case "mark_shortage":
		shortage := models.ChecklistShortage{EventID: eventID, ChecklistItemID: request.EntityID, MissingQuantity: numberValue(payload["missing_quantity"]), Reason: stringValue(payload["reason"]), ResolutionType: stringValue(payload["resolution_type"]), ResponsibleName: stringValue(payload["responsible_name"]), SupplierName: stringValue(payload["supplier_name"]), Notes: stringValue(payload["notes"])}
		if value := stringValue(payload["due_at"]); value != "" {
			shortage.DueAt, _ = time.ParseInLocation("2006-01-02T15:04", value, a.location)
		}
		if value := stringValue(payload["estimated_cost"]); value != "" {
			shortage.EstimatedCostCents = sql.NullInt64{Int64: int64(numberValue(value)*100 + 0.5), Valid: true}
		}
		err = a.store.SaveChecklistShortage(r.Context(), shortage, user.ID)
	case "resolve_shortage":
		err = a.store.UpdateShortageStatus(r.Context(), eventID, request.EntityID, stringValue(payload["status"]), stringValue(payload["destination"]), stringValue(payload["notes"]), user.ID)
	case "add_manual_item":
		inventoryID := int64(numberValue(payload["inventory_item_id"]))
		item := models.ChecklistItem{InventoryItemID: sql.NullInt64{Int64: inventoryID, Valid: inventoryID > 0}, CategoryID: int64(numberValue(payload["category_id"])), Name: stringValue(payload["name"]), Unit: stringValue(payload["unit"]), RequiredQuantity: numberValue(payload["quantity"]), Notes: stringValue(payload["notes"]), ItemKind: "reusable"}
		result.EntityID, err = a.store.AddManualChecklistItem(r.Context(), eventID, item, user.ID)
	case "mobile_loading_decision":
		_, saveErr := a.store.UpdateMobileLoadingItem(r.Context(), eventID, request.EntityID, stringValue(payload["decision"]), numberValue(payload["missing_quantity"]), user.ID)
		err = saveErr
	case "update_event_draft":
		err = a.applyOfflineEventDraft(r, eventID, request.BaseVersion, payload, user.ID)
	default:
		err = fmt.Errorf("tipo de operação não permitido")
	}
	if err == nil {
		result.Status = "synced"
		if request.OperationType == "update_event_draft" {
			if event, loadErr := a.store.GetEvent(r.Context(), eventID); loadErr == nil {
				result.Version = event.RowVersion
			}
		}
		return result
	}
	if strings.Contains(err.Error(), "version conflict") {
		result.Status = "conflict"
		result.Error = "O servidor possui uma versão mais recente."
		if request.OperationType == "update_event_draft" {
			if event, loadErr := a.store.GetEvent(r.Context(), eventID); loadErr == nil {
				result.ServerSnapshot = event
				result.Version = event.RowVersion
			}
		} else if checklist, loadErr := a.store.GetChecklistByEvent(r.Context(), eventID); loadErr == nil {
			for _, item := range checklist.Items {
				if item.ID == request.EntityID {
					result.ServerSnapshot = item
					result.Version = item.RowVersion
					break
				}
			}
		}
		return result
	}
	result.Status = "failed"
	result.Error = databaseErrorMessage(err)
	return result
}

func (a *App) applyOfflineEventDraft(r *http.Request, eventID int64, baseVersion int, payload map[string]any, userID int64) error {
	event, err := a.store.GetEvent(r.Context(), eventID)
	if err != nil {
		return err
	}
	if baseVersion > 0 && event.RowVersion != baseVersion {
		return fmt.Errorf("version conflict")
	}
	if event.Status != "planning" {
		return fmt.Errorf("only planning events can be edited offline")
	}
	if value := stringValue(payload["client_name"]); value != "" {
		event.ClientName = value
	}
	if value := stringValue(payload["name"]); value != "" {
		event.Name = value
	}
	if value := stringValue(payload["venue"]); value != "" {
		event.Venue = value
	}
	if value := int(numberValue(payload["guest_count"])); value > 0 {
		event.GuestCount = value
	}
	if value := stringValue(payload["starts_at"]); value != "" {
		if parsed, parseErr := time.ParseInLocation("2006-01-02T15:04", value, a.location); parseErr == nil {
			event.StartsAt = parsed
		}
	}
	if value := stringValue(payload["ends_at"]); value != "" {
		if parsed, parseErr := time.ParseInLocation("2006-01-02T15:04", value, a.location); parseErr == nil {
			event.EndsAt = parsed
		}
	}
	event.WaiterOverride = offlineOptionalInt(payload["waiter_override"])
	event.CoordinatorOverride = offlineOptionalInt(payload["coordinator_override"])
	event.LeaderOverride = offlineOptionalInt(payload["leader_override"])
	event.CoLeaderOverride = offlineOptionalInt(payload["co_leader_override"])
	event.KitchenCookID = offlineOptionalInt(payload["kitchen_cook_id"])
	event.AdditionalGuestMarginOverride = offlineOptionalFloat(payload["additional_guest_margin_override"])
	event.HasDecoration = boolValue(payload["has_decoration"])
	event.HasWelcomeDrinks = boolValue(payload["has_welcome_drinks"])
	event.HasCoffeeTable = boolValue(payload["has_coffee_table"])
	event.HasCake = boolValue(payload["has_cake"])
	event.CakeNotes = ""
	if event.HasCake {
		event.CakeNotes = strings.TrimSpace(stringValue(payload["cake_notes"]))
	}
	event.UsesGlassware = boolValue(payload["uses_glassware"])
	if observations, exists := payload["checklist_observations"]; exists {
		event.Notes = joinedStringValues(observations)
	} else {
		event.Notes = stringValue(payload["notes"])
	}
	if err := a.store.SaveEvent(r.Context(), &event, userID); err != nil {
		return err
	}
	if err := a.store.SyncEventCakePresence(r.Context(), event.ID, userID); err != nil {
		return err
	}
	_, err = a.checklist.GenerateTracked(r.Context(), event.ID, "offline_event_synced", userID)
	return err
}

func offlineOptionalInt(value any) sql.NullInt64 {
	if strings.TrimSpace(stringValue(value)) == "" {
		return sql.NullInt64{}
	}
	parsed := int64(numberValue(value))
	return sql.NullInt64{Int64: parsed, Valid: parsed >= 0}
}

func offlineOptionalFloat(value any) sql.NullFloat64 {
	if strings.TrimSpace(stringValue(value)) == "" {
		return sql.NullFloat64{}
	}
	parsed := numberValue(value)
	return sql.NullFloat64{Float64: parsed, Valid: parsed >= 0}
}
func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return ""
	}
}

func joinedStringValues(value any) string {
	var values []string
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			values = append(values, stringValue(item))
		}
	case []string:
		values = typed
	default:
		values = []string{stringValue(value)}
	}
	clean := make([]string, 0, len(values))
	for _, item := range values {
		if item = strings.TrimSpace(item); item != "" {
			clean = append(clean, item)
		}
	}
	return strings.Join(clean, "\n")
}

func numberValue(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case string:
		result, _ := strconv.ParseFloat(typed, 64)
		return result
	case json.Number:
		result, _ := typed.Float64()
		return result
	default:
		return 0
	}
}
func boolValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return typed == "true" || typed == "on" || typed == "1"
	default:
		return false
	}
}
