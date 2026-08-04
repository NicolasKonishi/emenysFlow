package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"buffetflow/internal/models"
)

const eventColumns = `e.id, e.template_id, e.client_name, e.name, e.venue, e.starts_at, e.ends_at,
	e.guest_count, e.has_decoration, e.has_welcome_drinks, e.has_coffee_table,
	e.starters_notes, e.main_courses_notes, e.sides_notes, e.beverages_notes,
	e.coffee_table_notes, e.cake_notes, e.sweets_notes, e.desserts_notes, e.notes,
	e.safety_margin_percent, e.waiter_override, e.kitchen_cook_id,
	COALESCE((SELECT cook.name FROM kitchen_cooks cook WHERE cook.id=e.kitchen_cook_id),''),
	e.coordinator_override,e.leader_override,e.co_leader_override,e.additional_guest_margin_override,e.uses_glassware,
	e.status, e.active, e.created_at, e.updated_at,e.row_version`

func scanEvent(scanner interface{ Scan(...any) error }) (models.Event, error) {
	var event models.Event
	var starts, ends, created, updated string
	var decoration, welcome, coffee, glassware, active int
	err := scanner.Scan(
		&event.ID, &event.TemplateID, &event.ClientName, &event.Name, &event.Venue, &starts, &ends,
		&event.GuestCount, &decoration, &welcome, &coffee,
		&event.StartersNotes, &event.MainCoursesNotes, &event.SidesNotes, &event.BeveragesNotes,
		&event.CoffeeTableNotes, &event.CakeNotes, &event.SweetsNotes, &event.DessertsNotes, &event.Notes,
		&event.SafetyMarginPercent, &event.WaiterOverride, &event.KitchenCookID, &event.KitchenCookName,
		&event.CoordinatorOverride, &event.LeaderOverride, &event.CoLeaderOverride, &event.AdditionalGuestMarginOverride, &glassware,
		&event.Status, &active, &created, &updated, &event.RowVersion,
	)
	event.StartsAt = parseTime(starts)
	event.EndsAt = parseTime(ends)
	event.CreatedAt = parseTime(created)
	event.UpdatedAt = parseTime(updated)
	event.HasDecoration = decoration == 1
	event.HasWelcomeDrinks = welcome == 1
	event.HasCoffeeTable = coffee == 1
	event.UsesGlassware = glassware == 1
	event.Active = active == 1
	return event, err
}

func (s *Store) ListEvents(ctx context.Context, query string) ([]models.Event, error) {
	pattern := "%" + strings.TrimSpace(query) + "%"
	rows, err := s.db.QueryContext(ctx, `SELECT `+eventColumns+`,
		COALESCE((SELECT 100.0 * SUM(MIN(ci.separated_quantity / NULLIF(ci.required_quantity,0),1)) / NULLIF(COUNT(*),0) FROM checklists c JOIN checklist_items ci ON ci.checklist_id=c.id WHERE c.event_id=e.id AND ci.active=1),0),
		COALESCE((SELECT 100.0 * SUM(MIN(ci.separated_quantity / NULLIF(ci.required_quantity,0),1)) / NULLIF(COUNT(*),0) FROM checklists c JOIN checklist_items ci ON ci.checklist_id=c.id WHERE c.event_id=e.id AND ci.active=1),0),
		COALESCE((SELECT 100.0 * SUM(MIN(ci.loaded_quantity / NULLIF(ci.required_quantity,0),1)) / NULLIF(COUNT(*),0) FROM checklists c JOIN checklist_items ci ON ci.checklist_id=c.id WHERE c.event_id=e.id AND ci.active=1),0),
		COALESCE((SELECT COUNT(*) FROM checklist_shortages shortage WHERE shortage.event_id=e.id AND shortage.status NOT IN ('resolved','cancelled')),0),
		COALESCE((SELECT COUNT(*) FROM checklist_shortages shortage WHERE shortage.event_id=e.id AND shortage.resolution_type='purchase' AND shortage.status NOT IN ('resolved','cancelled')),0),
		COALESCE((SELECT COUNT(*) FROM checklist_shortages shortage WHERE shortage.event_id=e.id AND shortage.resolution_type='rental' AND shortage.status NOT IN ('resolved','cancelled')),0)
		FROM events e WHERE e.active = 1 AND (? = '%%' OR e.name LIKE ? COLLATE NOCASE OR e.client_name LIKE ? COLLATE NOCASE OR e.venue LIKE ? COLLATE NOCASE)
		ORDER BY e.starts_at`, pattern, pattern, pattern, pattern)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()
	var events []models.Event
	for rows.Next() {
		var event models.Event
		var starts, ends, created, updated string
		var decoration, welcome, coffee, glassware, active int
		err := rows.Scan(
			&event.ID, &event.TemplateID, &event.ClientName, &event.Name, &event.Venue, &starts, &ends,
			&event.GuestCount, &decoration, &welcome, &coffee,
			&event.StartersNotes, &event.MainCoursesNotes, &event.SidesNotes, &event.BeveragesNotes,
			&event.CoffeeTableNotes, &event.CakeNotes, &event.SweetsNotes, &event.DessertsNotes, &event.Notes,
			&event.SafetyMarginPercent, &event.WaiterOverride, &event.KitchenCookID, &event.KitchenCookName,
			&event.CoordinatorOverride, &event.LeaderOverride, &event.CoLeaderOverride, &event.AdditionalGuestMarginOverride, &glassware,
			&event.Status, &active, &created, &updated, &event.RowVersion,
			&event.ChecklistProgress, &event.SeparationProgress, &event.LoadingProgress, &event.MissingItems, &event.PendingPurchases, &event.PendingRentals,
		)
		if err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		event.StartsAt, event.EndsAt = parseTime(starts), parseTime(ends)
		event.CreatedAt, event.UpdatedAt = parseTime(created), parseTime(updated)
		event.HasDecoration, event.HasWelcomeDrinks, event.HasCoffeeTable = decoration == 1, welcome == 1, coffee == 1
		event.UsesGlassware = glassware == 1
		event.Active = active == 1
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) GetEvent(ctx context.Context, id int64) (models.Event, error) {
	return scanEvent(s.db.QueryRowContext(ctx, `SELECT `+eventColumns+` FROM events e WHERE e.id = ?`, id))
}

func (s *Store) SaveEvent(ctx context.Context, event *models.Event, userID int64) error {
	if event.EndsAt.Before(event.StartsAt) || event.EndsAt.Equal(event.StartsAt) {
		return fmt.Errorf("end date must be after start date")
	}
	now := nowString()
	starts := event.StartsAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	ends := event.EndsAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	waiterOverride := any(nil)
	if event.WaiterOverride.Valid {
		waiterOverride = event.WaiterOverride.Int64
	}
	kitchenCookID := nullInt64(event.KitchenCookID)
	coordinatorOverride := nullInt64(event.CoordinatorOverride)
	leaderOverride := nullInt64(event.LeaderOverride)
	coLeaderOverride := nullInt64(event.CoLeaderOverride)
	additionalMarginOverride := any(nil)
	if event.AdditionalGuestMarginOverride.Valid {
		additionalMarginOverride = event.AdditionalGuestMarginOverride.Float64
	}
	if event.ID == 0 {
		result, err := s.db.ExecContext(ctx, `INSERT INTO events(
			template_id, client_name, name, venue, starts_at, ends_at, guest_count, has_decoration, has_welcome_drinks,
			has_coffee_table, starters_notes, main_courses_notes, sides_notes, beverages_notes,
			coffee_table_notes, cake_notes, sweets_notes, desserts_notes, notes, safety_margin_percent,
			waiter_override, kitchen_cook_id, coordinator_override,leader_override,co_leader_override,additional_guest_margin_override,uses_glassware,status, active, created_by,updated_by, created_at, updated_at)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,'planning',1,?,?,?,?)`,
			nullInt64(event.TemplateID), event.ClientName, event.Name, event.Venue, starts, ends, event.GuestCount,
			event.HasDecoration, event.HasWelcomeDrinks, event.HasCoffeeTable,
			event.StartersNotes, event.MainCoursesNotes, event.SidesNotes, event.BeveragesNotes,
			event.CoffeeTableNotes, event.CakeNotes, event.SweetsNotes, event.DessertsNotes, event.Notes,
			event.SafetyMarginPercent, waiterOverride, kitchenCookID, coordinatorOverride, leaderOverride, coLeaderOverride, additionalMarginOverride, event.UsesGlassware, nullableUserID(userID), nullableUserID(userID), now, now)
		if err != nil {
			return fmt.Errorf("create event: %w", err)
		}
		event.ID, err = result.LastInsertId()
		return err
	}

	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		var before string
		_ = tx.QueryRowContext(ctx, `SELECT json_object('guest_count', guest_count, 'welcome_drinks', has_welcome_drinks, 'decoration', has_decoration, 'kitchen_cook_id', kitchen_cook_id, 'updated_at', updated_at) FROM events WHERE id = ?`, event.ID).Scan(&before)
		result, err := tx.ExecContext(ctx, `UPDATE events SET template_id=?,client_name=?, name=?, venue=?, starts_at=?, ends_at=?, guest_count=?,
			has_decoration=?, has_welcome_drinks=?, has_coffee_table=?, starters_notes=?, main_courses_notes=?,
			sides_notes=?, beverages_notes=?, coffee_table_notes=?, cake_notes=?, sweets_notes=?, desserts_notes=?,
			notes=?, safety_margin_percent=?, waiter_override=?, kitchen_cook_id=?,coordinator_override=?,leader_override=?,co_leader_override=?,additional_guest_margin_override=?,uses_glassware=?,updated_by=?,row_version=row_version+1, updated_at=? WHERE id=? AND active=1`,
			nullInt64(event.TemplateID), event.ClientName, event.Name, event.Venue, starts, ends, event.GuestCount,
			event.HasDecoration, event.HasWelcomeDrinks, event.HasCoffeeTable,
			event.StartersNotes, event.MainCoursesNotes, event.SidesNotes, event.BeveragesNotes,
			event.CoffeeTableNotes, event.CakeNotes, event.SweetsNotes, event.DessertsNotes, event.Notes,
			event.SafetyMarginPercent, waiterOverride, kitchenCookID, coordinatorOverride, leaderOverride, coLeaderOverride, additionalMarginOverride, event.UsesGlassware, nullableUserID(userID), now, event.ID)
		if err != nil {
			return fmt.Errorf("update event: %w", err)
		}
		if count, _ := result.RowsAffected(); count == 0 {
			return sql.ErrNoRows
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO audit_log(user_id, entity_type, entity_id, action, before_json, after_json, created_at)
			VALUES(?, 'event', ?, 'update', ?, json_object('guest_count', ?, 'welcome_drinks', ?, 'decoration', ?, 'kitchen_cook_id', ?), ?)`,
			nullableUserID(userID), event.ID, before, event.GuestCount, event.HasWelcomeDrinks, event.HasDecoration, kitchenCookID, now)
		return err
	})
}

func (s *Store) CancelEvent(ctx context.Context, eventID, userID int64) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		var previous string
		if err := tx.QueryRowContext(ctx, "SELECT status FROM events WHERE id=? AND active=1", eventID).Scan(&previous); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE events SET status='cancelled', updated_at=? WHERE id=?", now, eventID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE inventory_reservations SET status='cancelled', updated_at=? WHERE event_id=? AND status='active'", now, eventID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO event_status_history(event_id, previous_status, new_status, notes, changed_by, created_at)
			VALUES(?, ?, 'cancelled', 'Evento cancelado pela interface.', ?, ?)`, eventID, previous, nullableUserID(userID), now)
		return err
	})
}

func (s *Store) DuplicateEvent(ctx context.Context, eventID, userID int64) (int64, error) {
	event, err := s.GetEvent(ctx, eventID)
	if err != nil {
		return 0, err
	}
	event.ID = 0
	event.Name += " — cópia"
	event.Status = "planning"
	event.StartsAt = event.StartsAt.AddDate(0, 0, 7)
	event.EndsAt = event.EndsAt.AddDate(0, 0, 7)
	if err := s.SaveEvent(ctx, &event, userID); err != nil {
		return 0, err
	}
	return event.ID, nil
}
