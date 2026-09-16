package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"buffetflow/internal/database"
	"buffetflow/internal/models"
	"buffetflow/internal/repositories"
	"buffetflow/internal/services"
)

const databasePath = "data/buffetflow.db"

type scenario struct {
	name, client, venue, notes               string
	start                                    time.Time
	guests                                   int
	decoration, welcome, coffee, cake, photo bool
	status                                   string
	noteItems                                []models.EventNote
}

func main() {
	apply := flag.Bool("apply", false, "replace events with the test scenarios")
	flag.Parse()

	ctx := context.Background()
	db, err := database.Open(databasePath)
	if err != nil {
		fatal(err)
	}
	defer db.Close()
	store := repositories.New(db)

	if !*apply {
		printCurrentData(ctx, store)
		return
	}

	backupPath, err := backupDatabase(ctx, db)
	if err != nil {
		fatal(err)
	}
	if err := database.Migrate(ctx, db); err != nil {
		fatal(err)
	}
	if err := removeExistingEvents(ctx, db, backupPath); err != nil {
		fatal(err)
	}

	location, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		fatal(err)
	}
	menuModels, err := store.ListMenuModels(ctx, false)
	if err != nil {
		fatal(err)
	}
	menuTemplates, err := store.ListMenuTemplates(ctx, false)
	if err != nil {
		fatal(err)
	}
	serviceModels, err := store.ListServiceModels(ctx, false)
	if err != nil {
		fatal(err)
	}
	userID := firstActiveUserID(ctx, db)
	checklist := services.NewChecklistService(store)

	scenarios := testScenarios(location)
	for index, item := range scenarios {
		menuLabel, err := createScenario(ctx, store, checklist, item, index, menuModels, menuTemplates, serviceModels, userID)
		if err != nil {
			fatal(fmt.Errorf("create %q: %w", item.name, err))
		}
		fmt.Printf("created: %s — %d guests — %s\n", item.name, item.guests, menuLabel)
	}
	if err := checkForeignKeys(ctx, db); err != nil {
		fatal(err)
	}
	printScenarioSummary(ctx, db, backupPath)
}

func printCurrentData(ctx context.Context, store *repositories.Store) {
	rows, err := store.DB().QueryContext(ctx, `SELECT id,name,client_name,venue,starts_at,guest_count,status FROM events ORDER BY starts_at,id`)
	if err != nil {
		fatal(err)
	}
	defer rows.Close()
	count := 0
	fmt.Println("Current events:")
	for rows.Next() {
		var id int64
		var name, client, venue, starts, status string
		var guests int
		if err := rows.Scan(&id, &name, &client, &venue, &starts, &guests, &status); err != nil {
			fatal(err)
		}
		fmt.Printf("  %d | %s | %s | %s | %d guests | %s | %s\n", id, name, client, venue, guests, status, starts)
		count++
	}
	if err := rows.Err(); err != nil {
		fatal(err)
	}
	if count == 0 {
		fmt.Println("  (none)")
	}
	models, err := store.ListMenuModels(ctx, false)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("Active menu models (%d):\n", len(models))
	for _, model := range models {
		fmt.Printf("  %d | %s\n", model.ID, model.Name)
	}
	templates, err := store.ListMenuTemplates(ctx, false)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("Legacy menu templates (%d):\n", len(templates))
	for _, item := range templates {
		fmt.Printf("  %d | %s\n", item.ID, item.Name)
	}
	services, err := store.ListServiceModels(ctx, false)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("Active service models (%d):\n", len(services))
	for _, service := range services {
		fmt.Printf("  %d | %s\n", service.ID, service.Name)
	}
}

func backupDatabase(ctx context.Context, db *sql.DB) (string, error) {
	if err := os.MkdirAll("data/backups", 0o750); err != nil {
		return "", err
	}
	backupPath := filepath.Join("data", "backups", "events-before-test-scenarios-"+time.Now().Format("20060102T150405")+".db")
	if _, err := db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		return "", err
	}
	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", backupPath); err != nil {
		return "", err
	}
	return backupPath, nil
}

func removeExistingEvents(ctx context.Context, db *sql.DB, backupPath string) error {
	var eventCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM events").Scan(&eventCount); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM events"); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM audit_log WHERE entity_type='event'"); err != nil {
		return err
	}

	eventUploads := filepath.Join("data", "uploads", "events")
	if _, err := os.Stat(eventUploads); err == nil {
		archiveUploads := strings.TrimSuffix(backupPath, ".db") + "-uploads"
		if err := os.Rename(eventUploads, archiveUploads); err != nil {
			return err
		}
	}
	fmt.Printf("removed %d existing events; database backup: %s\n", eventCount, backupPath)
	return nil
}

func createScenario(ctx context.Context, store *repositories.Store, checklist *services.ChecklistService, item scenario, index int, menuModels []models.MenuModel, menuTemplates []models.MenuTemplate, serviceModels []models.ServiceModel, userID int64) (string, error) {
	event := models.Event{
		ClientName: item.client, Name: item.name, Venue: item.venue,
		StartsAt: item.start, EndsAt: item.start.Add(7 * time.Hour), GuestCount: item.guests,
		HasDecoration: item.decoration, HasWelcomeDrinks: item.welcome, HasCoffeeTable: item.coffee, HasCake: item.cake,
		CakeNotes: cakeNotes(item.cake, index), Notes: item.notes, SafetyMarginPercent: safetyMargin(item.guests), UsesGlassware: item.guests >= 60,
	}
	if err := store.SaveEvent(ctx, &event, userID); err != nil {
		return "", err
	}

	menuLabel, err := attachMenu(ctx, store, event.ID, event.GuestCount, index, menuModels, menuTemplates, userID)
	if err != nil {
		return "", err
	}
	if err := store.SyncEventCakePresence(ctx, event.ID, userID); err != nil {
		return "", err
	}
	if len(serviceModels) > 0 {
		serviceID := serviceModels[index%len(serviceModels)].ID
		if err := store.ApplyServiceSnapshots(ctx, event.ID, []int64{serviceID}, userID); err != nil {
			return "", err
		}
	}
	if item.decoration {
		if err := attachDecoration(ctx, store, event.ID, index, userID); err != nil {
			return "", err
		}
	}
	for noteIndex := range item.noteItems {
		note := item.noteItems[noteIndex]
		note.EventID, note.Category = event.ID, "general"
		if err := store.SaveEventNote(ctx, &note, userID, 0); err != nil {
			return "", err
		}
		if item.photo && noteIndex == 0 {
			if err := attachExamplePhoto(ctx, store, event.ID, note.ID, noteIndex, userID); err != nil {
				return "", err
			}
		}
	}
	if item.status != "planning" {
		if _, err := store.DB().ExecContext(ctx, "UPDATE events SET status=? WHERE id=?", item.status, event.ID); err != nil {
			return "", err
		}
	}
	if _, err := checklist.GenerateTracked(ctx, event.ID, "test_scenarios_seed", userID); err != nil {
		return "", err
	}
	return menuLabel, nil
}

func attachMenu(ctx context.Context, store *repositories.Store, eventID int64, guests, index int, menuModels []models.MenuModel, menuTemplates []models.MenuTemplate, userID int64) (string, error) {
	if len(menuModels) > 0 {
		model := menuModels[index%len(menuModels)]
		selected, err := defaultModelSelections(ctx, store, model.ID)
		if err != nil {
			return "", err
		}
		if err := store.ApplyMenuModelSnapshot(ctx, eventID, model.ID, selected, nil, userID); err != nil {
			return "", err
		}
		if err := store.SyncLegacyEventMenuFromSnapshot(ctx, eventID); err != nil {
			return "", err
		}
		return "cardápio: " + model.Name, nil
	}
	if len(menuTemplates) > 0 {
		template := menuTemplates[index%len(menuTemplates)]
		items, err := store.MenuTemplateSelection(ctx, template.ID)
		if err != nil {
			return "", err
		}
		for itemIndex := range items {
			items[itemIndex].EventID, items[itemIndex].Selected, items[itemIndex].Portions = eventID, true, guests
		}
		if err := store.SaveEventMenu(ctx, eventID, items); err != nil {
			return "", err
		}
		return "cardápio: " + template.Name, nil
	}
	return "sem modelo de cardápio", nil
}

func defaultModelSelections(ctx context.Context, store *repositories.Store, modelID int64) ([]int64, error) {
	sections, err := store.MenuModelSections(ctx, modelID)
	if err != nil {
		return nil, err
	}
	selected := map[int64]bool{}
	for _, section := range sections {
		for _, item := range section.Items {
			if item.Active && item.Included && !item.InChoiceGroup {
				selected[item.ID] = true
			}
		}
		for _, group := range section.ChoiceGroups {
			limit := len(group.Items)
			if group.SelectionMax.Valid && int(group.SelectionMax.Int64) < limit {
				limit = int(group.SelectionMax.Int64)
			}
			chosen := 0
			for _, candidate := range group.Items {
				if candidate.Active && candidate.Included && chosen < limit {
					selected[candidate.ID] = true
					chosen++
				}
			}
			for _, candidate := range group.Items {
				if chosen >= group.SelectionMin || chosen >= limit {
					break
				}
				if candidate.Active && !selected[candidate.ID] {
					selected[candidate.ID] = true
					chosen++
				}
			}
		}
	}
	result := make([]int64, 0, len(selected))
	for id := range selected {
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}

func attachDecoration(ctx context.Context, store *repositories.Store, eventID int64, index int, userID int64) error {
	profile := models.DecorationProfile{
		EventID: eventID, Active: true,
		Style:         []string{"Romântico contemporâneo", "Tropical elegante", "Minimalista dourado"}[index%3],
		PrimaryColors: []string{"Areia, branco e verde", "Terracota e rosé", "Azul e dourado"}[index%3],
		Theme:         "Cenário de teste", Notes: "Montar e conferir as referências antes de carregar.", ResponsibleName: "Equipe de decoração",
	}
	if err := store.SaveDecorationProfile(ctx, &profile, userID); err != nil {
		return err
	}
	selection, err := store.EventDecorationSelection(ctx, eventID)
	if err != nil {
		return err
	}
	chosen := make([]models.EventDecoration, 0, 2)
	for _, decoration := range selection {
		if len(chosen) == 2 {
			break
		}
		if decoration.Selectable {
			decoration.Selected, decoration.Quantity = true, float64(len(chosen)+1)
			chosen = append(chosen, decoration)
		}
	}
	if err := store.SaveEventDecorations(ctx, eventID, chosen); err != nil {
		return err
	}
	if err := store.EnsureDefaultDecorationCompositions(ctx, eventID, userID); err != nil {
		return err
	}
	profile, err = store.GetDecorationProfile(ctx, eventID)
	if err != nil {
		return err
	}
	for _, composition := range profile.Compositions {
		name := ""
		switch composition.CompositionType {
		case "cake_table":
			name = "Arranjo da mesa do bolo"
		case "guest_tables":
			name = "Número de mesa"
		case "ceremony":
			name = "Arranjo da cerimônia"
		}
		if name == "" {
			continue
		}
		piece := models.DecorationCompositionItem{CompositionID: composition.ID, Name: name, Quantity: 1, Origin: "owned", Color: profile.PrimaryColors, Notes: "Item de demonstração"}
		if err := store.SaveDecorationCompositionItem(ctx, eventID, &piece); err != nil {
			return err
		}
	}
	return store.SaveEventRentedDecorationItems(ctx, eventID, []models.DecorationCompositionItem{{Name: "Painel personalizado", Color: profile.PrimaryColors, Quantity: 1, Origin: "rented"}})
}

func attachExamplePhoto(ctx context.Context, store *repositories.Store, eventID, noteID int64, index int, userID int64) error {
	source := filepath.Join("web", "static", "icons", "emenys-mark.png")
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	directory := filepath.Join("data", "uploads", "events", fmt.Sprintf("%d", eventID), "notes")
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return err
	}
	destination := filepath.Join(directory, fmt.Sprintf("referencia-teste-%d.png", index+1))
	target, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(target, sourceFile)
	closeErr := target.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return store.SaveEventNotePhoto(ctx, &models.EventNotePhoto{
		EventNoteID: noteID, StoragePath: destination, OriginalName: "referencia-teste.png", MIMEType: "image/png", FileSize: written,
		Caption: "Referência visual de teste", ClientUploadID: fmt.Sprintf("test-event-%d-note-%d", eventID, noteID),
	}, userID)
}

func firstActiveUserID(ctx context.Context, db *sql.DB) int64 {
	var id int64
	_ = db.QueryRowContext(ctx, "SELECT id FROM users WHERE active=1 ORDER BY id LIMIT 1").Scan(&id)
	return id
}

func checkForeignKeys(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		return errors.New("foreign-key integrity check failed after creating test events")
	}
	return rows.Err()
}

func printScenarioSummary(ctx context.Context, db *sql.DB, backupPath string) {
	rows, err := db.QueryContext(ctx, `SELECT e.id,e.name,e.guest_count,e.venue,e.status,e.has_decoration,e.has_cake,
		(SELECT COUNT(*) FROM event_notes n WHERE n.event_id=e.id),
		(SELECT COUNT(*) FROM event_note_photos p JOIN event_notes n ON n.id=p.event_note_id WHERE n.event_id=e.id)
		FROM events e ORDER BY e.starts_at,e.id`)
	if err != nil {
		fatal(err)
	}
	defer rows.Close()
	fmt.Println("\nTest scenarios ready:")
	for rows.Next() {
		var id int64
		var name, venue, status string
		var guests, decoration, cake, notes, photos int
		if err := rows.Scan(&id, &name, &guests, &venue, &status, &decoration, &cake, &notes, &photos); err != nil {
			fatal(err)
		}
		fmt.Printf("  %d | %s | %d guests | decor=%t cake=%t | notes=%d photos=%d | %s | %s\n", id, name, guests, decoration == 1, cake == 1, notes, photos, status, venue)
	}
	if err := rows.Err(); err != nil {
		fatal(err)
	}
	fmt.Printf("\nBackup kept at %s\n", backupPath)
}

func testScenarios(location *time.Location) []scenario {
	date := func(month time.Month, day, hour, minute int) time.Time {
		return time.Date(2026, month, day, hour, minute, 0, 0, location)
	}
	return []scenario{
		{
			name: "Casamento Laura & Rafael", client: "Laura e Rafael", venue: "Espaço Jardim das Flores", start: date(time.September, 19, 17, 0), guests: 220,
			decoration: true, welcome: true, coffee: true, cake: true, photo: true, status: "separating",
			notes:     "Chegar até 14h para montagem. Confirmar acesso pela portaria lateral.",
			noteItems: []models.EventNote{{Title: "Não terá lembrancinhas na mesa", Content: "Deixar somente o cardápio e os guardanapos em cada lugar."}, {Title: "Mesa do bolo em tons de areia", Content: "Usar a referência anexada como guia para flores e altura dos vasos."}, {Title: "Bebidas sem álcool para crianças", Content: "Separar sucos e água com gás na estação próxima ao salão."}},
		},
		{
			name: "Aniversário de 40 anos — Bruno", client: "Bruno Almeida", venue: "Casa da Família — Moema", start: date(time.September, 25, 19, 30), guests: 70,
			decoration: false, welcome: false, coffee: false, cake: false, status: "planning",
			notes: "Evento sem decoração e sem bolo. Foco no jantar e no bar.",
		},
		{
			name: "Formatura de Engenharia", client: "Comissão de Formatura", venue: "Clube Pinheiros — Salão Nobre", start: date(time.October, 3, 20, 0), guests: 480,
			decoration: true, welcome: true, coffee: false, cake: false, status: "reserved",
			notes:     "Fluxo grande de convidados; abrir duas ilhas de bebidas e separar equipe extra.",
			noteItems: []models.EventNote{{Title: "Não terá bolo", Content: "A sobremesa será servida em copinhos individuais."}, {Title: "Montagem liberada às 15h", Content: "Entrada de carga pelo portão de serviço, com lista de nomes na recepção."}},
		},
		{
			name: "Café corporativo — NovaTech", client: "NovaTech Brasil", venue: "Hub Paulista — Auditório 3", start: date(time.October, 8, 8, 0), guests: 35,
			decoration: false, welcome: false, coffee: true, cake: false, status: "planning",
			notes: "Café da manhã leve, com montagem silenciosa antes da palestra.",
		},
		{
			name: "Mini wedding Helena & Caio", client: "Helena e Caio", venue: "Fazenda Santa Clara", start: date(time.October, 17, 16, 0), guests: 120,
			decoration: true, welcome: true, coffee: true, cake: true, photo: true, status: "planning",
			notes:     "Cerimônia ao ar livre. Ter plano B para chuva e deixar o lounge próximo ao celeiro.",
			noteItems: []models.EventNote{{Title: "Usar velas somente na mesa do bolo", Content: "Não colocar velas nas mesas dos convidados por causa do vento."}, {Title: "Bolo de limão siciliano", Content: "Manter refrigerado até a montagem; servir depois do café."}},
		},
		{
			name: "Confraternização de fim de ano — Atlântica", client: "Grupo Atlântica", venue: "Galpão 21", start: date(time.December, 12, 18, 30), guests: 600,
			decoration: false, welcome: true, coffee: false, cake: false, status: "planning",
			notes:     "Maior evento do conjunto de testes. Separar carga em dois veículos e usar duas frentes de atendimento.",
			noteItems: []models.EventNote{{Title: "Não terá decoração de mesa", Content: "A cenografia será fornecida pelo cliente. Levar somente itens operacionais do buffet."}},
		},
		{
			name: "Almoço intimista de domingo", client: "Família Nogueira", venue: "Apartamento Vila Madalena", start: date(time.November, 22, 12, 0), guests: 18,
			decoration: false, welcome: false, coffee: true, cake: true, status: "planning",
			notes:     "Menor evento do conjunto de testes, pensado para conferir cálculos e organização para poucos convidados.",
			noteItems: []models.EventNote{{Title: "Servir almoço na varanda", Content: "Confirmar elevador de serviço e levar proteção para a mesa externa."}},
		},
	}
}

func cakeNotes(hasCake bool, index int) string {
	if !hasCake {
		return ""
	}
	return []string{"Baunilha com frutas vermelhas", "Limão siciliano", "Chocolate belga"}[index%3]
}

func safetyMargin(guests int) float64 {
	if guests >= 400 {
		return 8
	}
	if guests >= 150 {
		return 5
	}
	return 0
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "seed test events:", err)
	os.Exit(1)
}
