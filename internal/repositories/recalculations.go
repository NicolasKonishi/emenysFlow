package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"

	"buffetflow/internal/models"
)

type recalculationItemSnapshot struct {
	Name       string  `json:"name"`
	Quantity   float64 `json:"quantity"`
	Available  float64 `json:"available"`
	Missing    float64 `json:"missing"`
	SourceKey  string  `json:"source_key"`
	RowVersion int     `json:"version"`
}

func (s *Store) RecordEventRecalculation(ctx context.Context, eventID int64, trigger string, userID int64, before, after models.Checklist) error {
	beforeItems := map[string]models.ChecklistItem{}
	afterItems := map[string]models.ChecklistItem{}
	for _, item := range before.Items {
		beforeItems[item.SourceKey] = item
	}
	for _, item := range after.Items {
		afterItems[item.SourceKey] = item
	}
	added, removed, updated, shortages := 0, 0, 0, 0
	for key, item := range afterItems {
		previous, exists := beforeItems[key]
		if !exists {
			added++
		} else if math.Abs(previous.RequiredQuantity-item.RequiredQuantity) > 0.0001 || math.Abs(previous.AvailableQuantity-item.AvailableQuantity) > 0.0001 {
			updated++
		}
		if item.MissingQuantity > 0 {
			shortages++
		}
	}
	for key := range beforeItems {
		if _, exists := afterItems[key]; !exists {
			removed++
		}
	}
	reservationsUpdated := 0
	if event, err := s.GetEvent(ctx, eventID); err == nil && event.Status != "planning" && event.Status != "cancelled" && event.Status != "completed" {
		reservationsUpdated = len(afterItems)
	}
	summary := models.RecalculationSummary{EventID: eventID, TriggerKey: trigger, PreviousVersion: before.Version, NewVersion: after.Version, Added: added, Removed: removed, QuantitiesUpdated: updated, Shortages: shortages, ReservationsUpdated: reservationsUpdated}
	summaryJSON, _ := json.Marshal(summary)
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `INSERT INTO event_recalculations(event_id,trigger_key,previous_checklist_version,new_checklist_version,added_count,removed_count,quantity_updated_count,shortage_count,reservation_updated_count,summary_json,requested_by,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, eventID, trigger, before.Version, after.Version, added, removed, updated, shortages, reservationsUpdated, string(summaryJSON), nullableUserID(userID), nowString())
		if err != nil {
			return err
		}
		recalculationID, err := result.LastInsertId()
		if err != nil {
			return err
		}
		for key, item := range afterItems {
			previous, exists := beforeItems[key]
			changeType := ""
			if !exists {
				changeType = "added"
			} else if math.Abs(previous.RequiredQuantity-item.RequiredQuantity) > 0.0001 || math.Abs(previous.AvailableQuantity-item.AvailableQuantity) > 0.0001 {
				changeType = "quantity_updated"
			}
			if changeType == "" {
				continue
			}
			beforeJSON := any(nil)
			if exists {
				encoded, _ := json.Marshal(recalculationSnapshot(previous))
				beforeJSON = string(encoded)
			}
			afterJSON, _ := json.Marshal(recalculationSnapshot(item))
			if _, err := tx.ExecContext(ctx, `INSERT INTO event_recalculation_changes(recalculation_id,source_key,change_type,before_json,after_json,created_at) VALUES(?,?,?,?,?,?)`, recalculationID, key, changeType, beforeJSON, string(afterJSON), nowString()); err != nil {
				return err
			}
		}
		for key, item := range beforeItems {
			if _, exists := afterItems[key]; exists {
				continue
			}
			beforeJSON, _ := json.Marshal(recalculationSnapshot(item))
			if _, err := tx.ExecContext(ctx, `INSERT INTO event_recalculation_changes(recalculation_id,source_key,change_type,before_json,created_at) VALUES(?,?,?,?,?)`, recalculationID, key, "removed", string(beforeJSON), nowString()); err != nil {
				return err
			}
		}
		return nil
	})
}

func recalculationSnapshot(item models.ChecklistItem) recalculationItemSnapshot {
	return recalculationItemSnapshot{Name: item.Name, Quantity: item.RequiredQuantity, Available: item.AvailableQuantity, Missing: item.MissingQuantity, SourceKey: item.SourceKey, RowVersion: item.RowVersion}
}

func (s *Store) LatestEventRecalculation(ctx context.Context, eventID int64) (models.RecalculationSummary, error) {
	var result models.RecalculationSummary
	var created string
	err := s.db.QueryRowContext(ctx, `SELECT id,event_id,trigger_key,previous_checklist_version,new_checklist_version,added_count,removed_count,quantity_updated_count,shortage_count,reservation_updated_count,created_at FROM event_recalculations WHERE event_id=? ORDER BY id DESC LIMIT 1`, eventID).Scan(&result.ID, &result.EventID, &result.TriggerKey, &result.PreviousVersion, &result.NewVersion, &result.Added, &result.Removed, &result.QuantitiesUpdated, &result.Shortages, &result.ReservationsUpdated, &created)
	result.CreatedAt = parseTime(created)
	return result, err
}
