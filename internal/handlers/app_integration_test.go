//go:build private_seeds

package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"buffetflow/internal/database"
	"buffetflow/internal/repositories"
	"buffetflow/internal/services"
)

func TestMainPagesRenderAfterLogin(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "web-test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	store := repositories.New(db)
	auth := services.NewAuthService(store)
	if err := auth.EnsureDemoAdmin(ctx); err != nil {
		t.Fatal(err)
	}
	checklist := services.NewChecklistService(store)
	if err := checklist.EnsureDemoChecklist(ctx); err != nil {
		t.Fatal(err)
	}
	location, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatal(err)
	}
	app := New(store, auth, checklist, slog.New(slog.NewTextHandler(io.Discard, nil)), location)
	server := httptest.NewServer(app.Routes())
	defer server.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	response, err := client.PostForm(server.URL+"/login", url.Values{"email": {"admin@buffet.local"}, "password": {"admin123"}})
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("login final status %d", response.StatusCode)
	}

	checks := map[string]string{
		"/":                    "Próximos eventos",
		"/online":              "Próximos eventos",
		"/offline":             "Checklists e layout das festas",
		"/api/health":          `"ok":true`,
		"/events":              "Cliente Demonstração",
		"/models":              "Buffet Demonstração",
		"/models?tab=services": "Totem fotográfico",
		"/models/menus/1":      "Adicionar à seção",
		"/models/services/2":   "Adicionar componente",
		"/events/new":          `type="number" min="1" max="10" step="1" name="decoration_quantity_2"`,
		"/events/menu-model-preview?menu_model_id=2": "Escolha de entradas",
		"/events/1":                      "Cliente Demonstração",
		"/events/1/pdf":                  "Compartilhar / WhatsApp",
		"/inventory":                     "Copo descartável",
		"/inventory/kitchen-boxes":       "Caixas das cozinheiras",
		"/inventory/kitchen-boxes/box/1": "Utensílios e temperos dentro da caixa",
		"/inventory/new":                 "Novo item",
		"/rules":                         "Garçons por convidados",
		"/rules/new":                     "Nova regra",
		"/catalog":                       "Cardápios e recipientes",
		"/catalog/templates/new":         "Crie o modelo",
		"/catalog/templates/1/items/new": "Copiar do catálogo geral",
		"/catalog/items/new":             "Novo item do cardápio",
		"/catalog/items/1/edit":          "Receita e ingredientes",
		"/catalog/containers/new":        "Novo recipiente",
		"/events/1/menu":                 "Cardápio do evento",
		"/events/1/decorations":          "Decoração do evento",
		"/events/1/layout":               "Layout do salão",
		"/layouts":                       "Layouts do salão",
		"/layouts/new":                   "Novo layout avulso",
		"/events/1/operation":            "Sincronizar agora",
		"/events/1/operation/separating": "Separação do estoque",
		"/events/1/operation/loading":    "Checklist rápido para a van",
		"/events/1/return":               "Retorno dos itens",
		"/inventory/2/movements":         "Últimas movimentações",
		"/settings":                      "Configurações",
		"/settings/users/new":            "Novo usuário",
		"/manifest.webmanifest":          "emenysFlow",
		"/static/offline.html":           "Conexão restabelecida",
		"/static/js/offline.js":          "/api/health",
		"/static/js/layout-division.js":  "suggestFloorWaiterDivision",
		"/sw.js":                         `CACHE_VERSION = "v32"`,
		"/api/offline/bootstrap":         `"schema_version":2`,
		"/static/css/app.css":            "--brand",
	}
	for path, expected := range checks {
		response, err := client.Get(server.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		body, _ := io.ReadAll(response.Body)
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Errorf("GET %s status %d", path, response.StatusCode)
		}
		if !strings.Contains(string(body), expected) {
			t.Errorf("GET %s missing %q in response", path, expected)
		}
		if strings.Contains(string(body), "onsubmit=") {
			t.Errorf("GET %s rendered an inline script handler", path)
		}
		if path == "/online" {
			policy := response.Header.Get("Content-Security-Policy")
			if !strings.Contains(policy, "script-src 'self'") || strings.Contains(policy, "unpkg.com") {
				t.Errorf("unexpected Content-Security-Policy: %q", policy)
			}
			if strings.Contains(string(body), "data-update-notice") || strings.Contains(string(body), "Uma nova versão do emenysFlow") {
				t.Error("online dashboard still rendered the obsolete application update notice")
			}
		}
		if path == "/events/new" && (!strings.Contains(string(body), "Itens que serão alugados") || !strings.Contains(string(body), `name="rented_decoration_color"`)) {
			t.Error("new event form is missing the rented decoration editor")
		}
		if path == "/events/new" && (!strings.Contains(string(body), `name="checklist_observations"`) || !strings.Contains(string(body), `data-add-checklist-observation`) || !strings.Contains(string(body), `data-remove-checklist-observation`)) {
			t.Error("new event form is missing the editable checklist observations")
		}
		if path == "/events/new" && (!strings.Contains(string(body), `name="has_cake"`) || !strings.Contains(string(body), `<strong>Tem bolo</strong>`) || !strings.Contains(string(body), `name="cake_notes"`) || !strings.Contains(string(body), `class="cake-flavor-field" data-cake-flavor-field hidden`)) {
			t.Error("new event form is missing the optional cake and flavor controls")
		}
		if path == "/inventory/new" && (!strings.Contains(string(body), `data-code-prefix="CUB"`) || !strings.Contains(string(body), `data-inventory-code-mode="create"`) || !strings.Contains(string(body), `name="internal_code"`)) {
			t.Error("new inventory item form is missing automatic internal code metadata")
		}
		if path == "/events/1" && (!strings.Contains(string(body), `/checklist/groups/material/status`) || !strings.Contains(string(body), `data-group-check`)) {
			t.Error("event checklist is missing the whole-group check control")
		}
	}

	response, err = client.PostForm(server.URL+"/inventory", url.Values{
		"name":            {"Cuba de Réchaud"},
		"internal_code":   {"CODIGO-ALTERADO-NO-NAVEGADOR"},
		"category_id":     {"3"},
		"unit":            {"unidade"},
		"item_kind":       {"reusable"},
		"ownership":       {"owned"},
		"requires_return": {"on"},
	})
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("create inventory item final status %d", response.StatusCode)
	}
	var generatedInternalCode string
	if err := store.DB().QueryRowContext(ctx, "SELECT internal_code FROM inventory_items WHERE name=?", "Cuba de Réchaud").Scan(&generatedInternalCode); err != nil {
		t.Fatal(err)
	}
	if generatedInternalCode != "CUB-cuba-de-rechaud" {
		t.Fatalf("generated inventory code = %q, want %q", generatedInternalCode, "CUB-cuba-de-rechaud")
	}

	demoChecklist, err := store.GetChecklistByEvent(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	var materialItemIDs []int64
	for _, group := range groupChecklist(demoChecklist.Items) {
		if group.Key == "material" {
			for _, item := range group.Items {
				materialItemIDs = append(materialItemIDs, item.ID)
			}
		}
	}
	if len(materialItemIDs) == 0 {
		t.Fatal("demo checklist should contain material items")
	}
	response, err = client.PostForm(server.URL+"/events/1/checklist/groups/material/status", url.Values{"status": {"separated"}})
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("whole material group check final status %d", response.StatusCode)
	}
	for _, itemID := range materialItemIDs {
		var status string
		var required, separated float64
		if err := store.DB().QueryRowContext(ctx, "SELECT status,required_quantity,separated_quantity FROM checklist_items WHERE id=?", itemID).Scan(&status, &required, &separated); err != nil {
			t.Fatal(err)
		}
		if status != "separated" || separated != required {
			t.Fatalf("material item %d status=%q separated=%.0f required=%.0f", itemID, status, separated, required)
		}
	}

	var syncItemID int64
	var syncVersion int
	if err := store.DB().QueryRowContext(ctx, `SELECT item.id,item.row_version FROM checklist_items item JOIN checklists checklist ON checklist.id=item.checklist_id WHERE checklist.event_id=1 AND item.active=1 ORDER BY item.id LIMIT 1`).Scan(&syncItemID, &syncVersion); err != nil {
		t.Fatal(err)
	}
	syncPayload := fmt.Sprintf(`[{"client_operation_id":"integration-operation-1","device_id":"integration-device","operation_type":"update_quantity","entity_type":"checklist_item","entity_id":%d,"base_version":%d,"payload":{"event_id":1,"stage":"separation","quantity":0,"notes":"teste offline"},"local_date":"2026-08-03T12:00:00Z"}]`, syncItemID, syncVersion)
	for attempt := 0; attempt < 2; attempt++ {
		request, err := http.NewRequest(http.MethodPost, server.URL+"/api/sync/operations", strings.NewReader(syncPayload))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Content-Type", "application/json")
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(response.Body)
		response.Body.Close()
		if response.StatusCode != http.StatusOK || !strings.Contains(string(body), `"status":"synced"`) {
			t.Fatalf("offline sync attempt %d got status %d: %s", attempt+1, response.StatusCode, body)
		}
	}
	var recordedOperations int
	if err := store.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM sync_operations WHERE client_operation_id='integration-operation-1'").Scan(&recordedOperations); err != nil || recordedOperations != 1 {
		t.Fatalf("idempotent sync operation count=%d err=%v", recordedOperations, err)
	}

	layoutJSON := `{"version":2,"width":1400,"height":900,"waiters":[],"elements":[{"id":"t1","type":"table_round","x":100,"y":120,"width":88,"height":88,"label":"Mesa 1","seats":8}]}`
	layoutSyncPayload := fmt.Sprintf(`[{"client_operation_id":"integration-layout-1","device_id":"integration-device","operation_type":"save_event_layout","entity_type":"event_floor_layout","entity_id":1,"base_version":0,"payload":{"event_id":1,"layout_json":%q,"layout_key":"event:1"},"local_date":"2026-08-03T12:00:00Z"}]`, layoutJSON)
	layoutRequest, err := http.NewRequest(http.MethodPost, server.URL+"/api/sync/operations", strings.NewReader(layoutSyncPayload))
	if err != nil {
		t.Fatal(err)
	}
	layoutRequest.Header.Set("Content-Type", "application/json")
	layoutResponse, err := client.Do(layoutRequest)
	if err != nil {
		t.Fatal(err)
	}
	layoutBody, _ := io.ReadAll(layoutResponse.Body)
	layoutResponse.Body.Close()
	if layoutResponse.StatusCode != http.StatusOK || !strings.Contains(string(layoutBody), `"status":"synced"`) {
		t.Fatalf("layout offline sync got status %d: %s", layoutResponse.StatusCode, layoutBody)
	}
	syncedLayout, err := store.GetEventFloorLayout(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if syncedLayout.LayoutJSON != layoutJSON {
		t.Fatalf("synced layout json got %q", syncedLayout.LayoutJSON)
	}
	if syncedLayout.RowVersion != 1 {
		t.Fatalf("synced layout version got %d", syncedLayout.RowVersion)
	}

	starts := time.Now().In(location).AddDate(0, 1, 0).Truncate(time.Minute)
	ends := starts.Add(8 * time.Hour)
	response, err = client.PostForm(server.URL+"/events", url.Values{
		"template_id":           {"2"},
		"client_name":           {"Cliente do modelo"},
		"name":                  {"Festa criada pelo cardápio-base"},
		"venue":                 {"Salão de testes"},
		"starts_at":             {starts.Format("2006-01-02T15:04")},
		"ends_at":               {ends.Format("2006-01-02T15:04")},
		"guest_count":           {"100"},
		"safety_margin_percent": {"10"},
	})
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("create event from menu template final status %d", response.StatusCode)
	}
	var createdEventID int64
	if err := store.DB().QueryRowContext(ctx, "SELECT id FROM events WHERE name=?", "Festa criada pelo cardápio-base").Scan(&createdEventID); err != nil {
		t.Fatal(err)
	}
	createdEvent, err := store.GetEvent(ctx, createdEventID)
	if err != nil {
		t.Fatal(err)
	}
	if !createdEvent.TemplateID.Valid || createdEvent.TemplateID.Int64 != 2 {
		t.Fatalf("event template got %+v, want 2", createdEvent.TemplateID)
	}
	if createdEvent.HasCake || createdEvent.CakeNotes != "" {
		t.Fatalf("event without cake got has_cake=%v flavor=%q", createdEvent.HasCake, createdEvent.CakeNotes)
	}
	templateSelection, err := store.MenuTemplateSelection(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	eventSelection, err := store.EventMenuSelection(ctx, createdEventID)
	if err != nil {
		t.Fatal(err)
	}
	templateSelected, eventSelected := 0, 0
	for _, item := range templateSelection {
		if item.Selected {
			templateSelected++
		}
	}
	for _, item := range eventSelection {
		if item.Selected {
			eventSelected++
		}
	}
	if templateSelected == 0 || eventSelected != templateSelected {
		t.Fatalf("event copied %d template items, want %d", eventSelected, templateSelected)
	}
	for index := range eventSelection {
		if eventSelection[index].Selected {
			eventSelection[index].Selected = false
			break
		}
	}
	if err := store.SaveEventMenu(ctx, createdEventID, eventSelection); err != nil {
		t.Fatal(err)
	}
	templateAfterEventEdit, err := store.MenuTemplateSelection(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	selectedAfterEventEdit := 0
	for _, item := range templateAfterEventEdit {
		if item.Selected {
			selectedAfterEventEdit++
		}
	}
	if selectedAfterEventEdit != templateSelected {
		t.Fatalf("editing event changed its template: got %d items, want %d", selectedAfterEventEdit, templateSelected)
	}

	var advancedModelID, barServiceID, fixedTemplateItemID, containerID, adminID, kitchenCookID int64
	if err := store.DB().QueryRowContext(ctx, "SELECT id FROM menu_templates WHERE slug='buffet-carnes'").Scan(&advancedModelID); err != nil {
		t.Fatal(err)
	}
	if err := store.DB().QueryRowContext(ctx, "SELECT id FROM service_templates WHERE slug='bar'").Scan(&barServiceID); err != nil {
		t.Fatal(err)
	}
	if err := store.DB().QueryRowContext(ctx, `SELECT item.id FROM menu_template_items item JOIN menu_template_sections section ON section.id=item.menu_template_section_id WHERE section.menu_template_id=? AND item.included=1 ORDER BY item.id LIMIT 1`, advancedModelID).Scan(&fixedTemplateItemID); err != nil {
		t.Fatal(err)
	}
	if err := store.DB().QueryRowContext(ctx, "SELECT id FROM container_types WHERE active=1 ORDER BY id LIMIT 1").Scan(&containerID); err != nil {
		t.Fatal(err)
	}
	if err := store.DB().QueryRowContext(ctx, "SELECT id FROM users WHERE email='admin@buffet.local'").Scan(&adminID); err != nil {
		t.Fatal(err)
	}
	if err := store.DB().QueryRowContext(ctx, "SELECT id FROM kitchen_cooks WHERE slug='geriane'").Scan(&kitchenCookID); err != nil {
		t.Fatal(err)
	}
	choiceRows, err := store.DB().QueryContext(ctx, `SELECT choice.id,item.id FROM menu_choice_groups choice JOIN menu_template_sections section ON section.id=choice.menu_template_section_id JOIN menu_choice_group_items membership ON membership.menu_choice_group_id=choice.id JOIN menu_template_items item ON item.id=membership.menu_template_item_id WHERE section.menu_template_id=? ORDER BY choice.id,membership.sort_order`, advancedModelID)
	if err != nil {
		t.Fatal(err)
	}
	groupCounts := map[int64]int{}
	var advancedChoices []int64
	for choiceRows.Next() {
		var groupID, itemID int64
		if err := choiceRows.Scan(&groupID, &itemID); err != nil {
			t.Fatal(err)
		}
		if groupCounts[groupID] < 2 {
			advancedChoices = append(advancedChoices, itemID)
			groupCounts[groupID]++
		}
	}
	choiceRows.Close()
	advancedForm := url.Values{
		"menu_model_id":          {fmt.Sprint(advancedModelID)},
		"service_model_ids":      {fmt.Sprint(barServiceID)},
		"kitchen_cook_id":        {fmt.Sprint(kitchenCookID)},
		"checklist_observations": {"Levar caixa térmica extra", "Avisar a equipe da separação"},
		"has_cake":               {"on"},
		"cake_notes":             {"Chocolate com morango"},
		fmt.Sprintf("model_portions_%d", fixedTemplateItemID):  {"120"},
		fmt.Sprintf("model_container_%d", fixedTemplateItemID): {fmt.Sprint(containerID)},
		"client_name":           {"Cliente avançado"},
		"name":                  {"Evento com snapshot avançado"},
		"venue":                 {"Salão de integração"},
		"starts_at":             {starts.AddDate(0, 1, 0).Format("2006-01-02T15:04")},
		"ends_at":               {ends.AddDate(0, 1, 0).Format("2006-01-02T15:04")},
		"guest_count":           {"120"},
		"safety_margin_percent": {"10"},
	}
	for _, itemID := range advancedChoices {
		advancedForm.Add("model_item_ids", fmt.Sprint(itemID))
	}
	response, err = client.PostForm(server.URL+"/events", advancedForm)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("create event from advanced model final status %d", response.StatusCode)
	}
	var advancedEventID int64
	if err := store.DB().QueryRowContext(ctx, "SELECT id FROM events WHERE name='Evento com snapshot avançado'").Scan(&advancedEventID); err != nil {
		t.Fatal(err)
	}
	var customSnapshotCount, serviceSnapshotCount, checklistServiceCount, checklistCookBoxCount int
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM event_menu_snapshot_items item JOIN event_menu_sections section ON section.id=item.event_menu_section_id JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id WHERE snapshot.event_id=? AND item.custom_item=1`, advancedEventID).Scan(&customSnapshotCount); err != nil {
		t.Fatal(err)
	}
	if err := store.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM event_services WHERE event_id=?", advancedEventID).Scan(&serviceSnapshotCount); err != nil {
		t.Fatal(err)
	}
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM checklist_items item JOIN checklists checklist ON checklist.id=item.checklist_id WHERE checklist.event_id=? AND item.name='Gelo'`, advancedEventID).Scan(&checklistServiceCount); err != nil {
		t.Fatal(err)
	}
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM checklist_items item JOIN checklists checklist ON checklist.id=item.checklist_id WHERE checklist.event_id=? AND item.source_key LIKE 'kitchen-cook-box:%'`, advancedEventID).Scan(&checklistCookBoxCount); err != nil {
		t.Fatal(err)
	}
	if customSnapshotCount != 0 || serviceSnapshotCount != 1 || checklistServiceCount != 0 || checklistCookBoxCount != 1 {
		t.Fatalf("advanced workflow custom=%d services=%d checklist-service=%d cook-boxes=%d", customSnapshotCount, serviceSnapshotCount, checklistServiceCount, checklistCookBoxCount)
	}
	var checklistNotes string
	if err := store.DB().QueryRowContext(ctx, "SELECT notes FROM events WHERE id=?", advancedEventID).Scan(&checklistNotes); err != nil {
		t.Fatal(err)
	}
	if checklistNotes != "Levar caixa térmica extra\nAvisar a equipe da separação" {
		t.Fatalf("checklist notes = %q", checklistNotes)
	}
	advancedEvent, err := store.GetEvent(ctx, advancedEventID)
	if err != nil {
		t.Fatal(err)
	}
	if !advancedEvent.HasCake || advancedEvent.CakeNotes != "Chocolate com morango" {
		t.Fatalf("advanced event cake got has_cake=%v flavor=%q", advancedEvent.HasCake, advancedEvent.CakeNotes)
	}
	var selectedCakeItems int
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM event_menu_snapshot_items item JOIN event_menu_sections section ON section.id=item.event_menu_section_id JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id WHERE snapshot.event_id=? AND LOWER(section.name) LIKE '%bolo%' AND item.selected=1 AND item.was_removed=0`, advancedEventID).Scan(&selectedCakeItems); err != nil {
		t.Fatal(err)
	}
	if selectedCakeItems == 0 {
		t.Fatal("event with cake should keep the cake section selected")
	}
	var editableItemID int64
	var editableName string
	var included, configurable int
	if err := store.DB().QueryRowContext(ctx, `SELECT item.id,item.normalized_name,item.included,item.configurable FROM menu_template_items item JOIN menu_template_sections section ON section.id=item.menu_template_section_id WHERE section.menu_template_id=? ORDER BY item.id LIMIT 1`, advancedModelID).Scan(&editableItemID, &editableName, &included, &configurable); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateMenuModelItem(ctx, advancedModelID, editableItemID, editableName+" revisão", "", included == 1, configurable == 1, adminID); err != nil {
		t.Fatal(err)
	}
	response, err = client.Get(fmt.Sprintf("%s/events/%d/menu-model/compare", server.URL, advancedEventID))
	if err != nil {
		t.Fatal(err)
	}
	compareBody, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(string(compareBody), "Atualização disponível") || !strings.Contains(string(compareBody), "Personalizações preservadas") {
		t.Fatalf("model comparison did not render: status=%d", response.StatusCode)
	}

	request, _ := http.NewRequest(http.MethodGet, server.URL+"/events", nil)
	request.Header.Set("HX-Request", "true")
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if strings.Contains(string(body), "<!doctype html>") {
		t.Error("HTMX response should contain only page content")
	}
	if !strings.Contains(string(body), "Eventos") {
		t.Error("HTMX response is missing page content")
	}
	for _, path := range []string{"/events/1/export.csv", "/events/1/export.pdf"} {
		response, err = client.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ = io.ReadAll(response.Body)
		response.Body.Close()
		if response.StatusCode != http.StatusOK || len(body) < 20 {
			t.Errorf("export %s failed", path)
		}
	}
	response, err = client.Get(server.URL + "/events/1/export.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if disposition := response.Header.Get("Content-Disposition"); !strings.HasPrefix(disposition, "inline;") {
		t.Errorf("inline PDF disposition got %q", disposition)
	}
	if framePolicy := response.Header.Get("X-Frame-Options"); framePolicy != "SAMEORIGIN" {
		t.Errorf("PDF frame policy got %q", framePolicy)
	}
	response.Body.Close()
	response, err = client.Get(server.URL + "/events/1/export.pdf?download=1")
	if err != nil {
		t.Fatal(err)
	}
	if disposition := response.Header.Get("Content-Disposition"); !strings.HasPrefix(disposition, "attachment;") {
		t.Errorf("download PDF disposition got %q", disposition)
	}
	response.Body.Close()
	token := "integration-share-token"
	sum := sha256.Sum256([]byte(token))
	if err := store.CreateEventShare(ctx, 1, hex.EncodeToString(sum[:]), 1); err != nil {
		t.Fatal(err)
	}
	response, err = client.Get(server.URL + "/share/" + token)
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	response.Body.Close()
	if !strings.Contains(string(body), "Somente leitura") {
		t.Error("shared event did not render")
	}
}
