package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"buffetflow/internal/models"
)

func (s *Store) OperationalSettings(ctx context.Context) (map[string]models.OperationalSetting, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT setting_key,name,description,numeric_value,unit,updated_at FROM operational_settings ORDER BY setting_key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]models.OperationalSetting{}
	for rows.Next() {
		var item models.OperationalSetting
		var updated string
		if err := rows.Scan(&item.Key, &item.Name, &item.Description, &item.Value, &item.Unit, &updated); err != nil {
			return nil, err
		}
		item.UpdatedAt = parseTime(updated)
		result[item.Key] = item
	}
	return result, rows.Err()
}

func (s *Store) SaveOperationalSetting(ctx context.Context, key string, value float64, userID int64) error {
	if value < 0 {
		return fmt.Errorf("setting value cannot be negative")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE operational_settings SET numeric_value=?,updated_by=?,updated_at=? WHERE setting_key=?`, value, nullableUserID(userID), nowString(), key)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) EventHasMenuCategory(ctx context.Context, eventID int64, slug string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM event_menu_items selected
		JOIN menu_items item ON item.id=selected.menu_item_id
		JOIN menu_categories category ON category.id=item.category_id
		WHERE selected.event_id=? AND category.slug=?`, eventID, slug).Scan(&count)
	return count > 0, err
}

func (s *Store) EventRequiresDessertSpoon(ctx context.Context, eventID int64) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM event_menu_items selected
		JOIN menu_items item ON item.id=selected.menu_item_id
		JOIN menu_categories category ON category.id=item.category_id
		JOIN container_types container ON container.id=COALESCE(selected.container_type_id,item.container_type_id)
		WHERE selected.event_id=? AND category.slug='desserts' AND container.required_utensil_type='spoon'`, eventID).Scan(&count)
	return count > 0, err
}

func normalizeSettingKey(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}
