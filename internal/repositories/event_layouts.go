package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"buffetflow/internal/models"
)

var ErrInvalidLayoutJSON = errors.New("invalid layout json")

const defaultFloorLayoutJSON = `{"version":2,"width":1400,"height":900,"waiters":[],"elements":[]}`

func (s *Store) GetEventFloorLayout(ctx context.Context, eventID int64) (models.EventFloorLayout, error) {
	var layout models.EventFloorLayout
	var createdAt, updatedAt string
	err := s.db.QueryRowContext(ctx, `SELECT id, event_id, layout_json, row_version, created_at, updated_at FROM event_floor_layouts WHERE event_id=?`, eventID).
		Scan(&layout.ID, &layout.EventID, &layout.LayoutJSON, &layout.RowVersion, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return models.EventFloorLayout{EventID: eventID, LayoutJSON: defaultFloorLayoutJSON}, nil
	}
	if err != nil {
		return layout, err
	}
	layout.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	layout.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return layout, nil
}

func (s *Store) SaveEventFloorLayout(ctx context.Context, eventID int64, layoutJSON string, userID int64) error {
	_, err := s.SaveEventFloorLayoutVersioned(ctx, eventID, layoutJSON, userID, 0)
	return err
}

func (s *Store) SaveEventFloorLayoutVersioned(ctx context.Context, eventID int64, layoutJSON string, userID int64, baseVersion int) (int, error) {
	if !json.Valid([]byte(layoutJSON)) {
		return 0, ErrInvalidLayoutJSON
	}
	var newVersion int
	err := withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		var version int
		err := tx.QueryRowContext(ctx, `SELECT row_version FROM event_floor_layouts WHERE event_id=?`, eventID).Scan(&version)
		if err == sql.ErrNoRows {
			if baseVersion > 0 {
				return fmt.Errorf("version conflict")
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO event_floor_layouts(event_id, layout_json, row_version, created_at, updated_at) VALUES(?,?,1,?,?)`, eventID, layoutJSON, now, now)
			newVersion = 1
			return err
		}
		if err != nil {
			return err
		}
		if baseVersion > 0 && baseVersion != version {
			return fmt.Errorf("version conflict")
		}
		_, err = tx.ExecContext(ctx, `UPDATE event_floor_layouts SET layout_json=?, row_version=row_version+1, updated_at=? WHERE event_id=?`, layoutJSON, now, eventID)
		newVersion = version + 1
		return err
	})
	return newVersion, err
}
