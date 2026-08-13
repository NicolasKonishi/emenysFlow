package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"buffetflow/internal/models"
)

const defaultStandaloneLayoutJSON = `{"version":2,"width":1400,"height":900,"waiters":[],"elements":[]}`

func (s *Store) ListStandaloneFloorLayouts(ctx context.Context, query string) ([]models.StandaloneFloorLayout, error) {
	sqlQuery := `SELECT id,name,venue,guest_count,waiter_count,waiter_names_json,layout_json,active,row_version,created_at,updated_at
		FROM standalone_floor_layouts WHERE active=1`
	args := []any{}
	if query = strings.TrimSpace(query); query != "" {
		sqlQuery += ` AND (name LIKE ? OR venue LIKE ?)`
		like := "%" + query + "%"
		args = append(args, like, like)
	}
	sqlQuery += ` ORDER BY updated_at DESC, id DESC`
	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var layouts []models.StandaloneFloorLayout
	for rows.Next() {
		item, err := scanStandaloneFloorLayout(rows)
		if err != nil {
			return nil, err
		}
		layouts = append(layouts, item)
	}
	return layouts, rows.Err()
}

func (s *Store) GetStandaloneFloorLayout(ctx context.Context, id int64) (models.StandaloneFloorLayout, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id,name,venue,guest_count,waiter_count,waiter_names_json,layout_json,active,row_version,created_at,updated_at
		FROM standalone_floor_layouts WHERE id=? AND active=1`, id)
	return scanStandaloneFloorLayoutRow(row)
}

func scanStandaloneFloorLayout(rows *sql.Rows) (models.StandaloneFloorLayout, error) {
	var item models.StandaloneFloorLayout
	var active int
	var createdAt, updatedAt string
	err := rows.Scan(&item.ID, &item.Name, &item.Venue, &item.GuestCount, &item.WaiterCount, &item.WaiterNamesJSON, &item.LayoutJSON, &active, &item.RowVersion, &createdAt, &updatedAt)
	if err != nil {
		return item, err
	}
	item.Active = active == 1
	item.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	item.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return item, nil
}

func scanStandaloneFloorLayoutRow(row *sql.Row) (models.StandaloneFloorLayout, error) {
	var item models.StandaloneFloorLayout
	var active int
	var createdAt, updatedAt string
	err := row.Scan(&item.ID, &item.Name, &item.Venue, &item.GuestCount, &item.WaiterCount, &item.WaiterNamesJSON, &item.LayoutJSON, &active, &item.RowVersion, &createdAt, &updatedAt)
	if err != nil {
		return item, err
	}
	item.Active = active == 1
	item.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	item.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return item, nil
}

func (s *Store) SaveStandaloneFloorLayout(ctx context.Context, layout *models.StandaloneFloorLayout, userID int64) error {
	_, _, err := s.SaveStandaloneFloorLayoutVersioned(ctx, layout, userID, 0)
	return err
}

func (s *Store) SaveStandaloneFloorLayoutVersioned(ctx context.Context, layout *models.StandaloneFloorLayout, userID int64, baseVersion int) (int64, int, error) {
	if !json.Valid([]byte(layout.LayoutJSON)) {
		return 0, 0, ErrInvalidLayoutJSON
	}
	if !json.Valid([]byte(layout.WaiterNamesJSON)) {
		layout.WaiterNamesJSON = "[]"
	}
	var newVersion int
	err := withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		if layout.ID == 0 {
			if baseVersion > 0 {
				return fmt.Errorf("version conflict")
			}
			result, err := tx.ExecContext(ctx, `INSERT INTO standalone_floor_layouts(name,venue,guest_count,waiter_count,waiter_names_json,layout_json,active,row_version,created_by,updated_by,created_at,updated_at)
				VALUES(?,?,?,?,?,?,1,1,?,?,?,?)`, layout.Name, layout.Venue, layout.GuestCount, layout.WaiterCount, layout.WaiterNamesJSON, layout.LayoutJSON, nullableUserID(userID), nullableUserID(userID), now, now)
			if err != nil {
				return err
			}
			layout.ID, _ = result.LastInsertId()
			layout.RowVersion = 1
			newVersion = 1
			return nil
		}
		var version int
		if err := tx.QueryRowContext(ctx, `SELECT row_version FROM standalone_floor_layouts WHERE id=? AND active=1`, layout.ID).Scan(&version); err != nil {
			return err
		}
		if baseVersion > 0 && baseVersion != version {
			return fmt.Errorf("version conflict")
		}
		result, err := tx.ExecContext(ctx, `UPDATE standalone_floor_layouts SET name=?,venue=?,guest_count=?,waiter_count=?,waiter_names_json=?,layout_json=?,updated_by=?,row_version=row_version+1,updated_at=?
			WHERE id=? AND active=1`, layout.Name, layout.Venue, layout.GuestCount, layout.WaiterCount, layout.WaiterNamesJSON, layout.LayoutJSON, nullableUserID(userID), now, layout.ID)
		if err != nil {
			return err
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			return sql.ErrNoRows
		}
		layout.RowVersion = version + 1
		newVersion = layout.RowVersion
		return nil
	})
	return layout.ID, newVersion, err
}

func (s *Store) ArchiveStandaloneFloorLayout(ctx context.Context, id int64, userID int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE standalone_floor_layouts SET active=0,updated_by=?,updated_at=? WHERE id=? AND active=1`, nullableUserID(userID), nowString(), id)
	return err
}
