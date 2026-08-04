package repositories

import (
	"context"
	"fmt"

	"buffetflow/internal/models"
)

func (s *Store) ListUsers(ctx context.Context) ([]models.User, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id,name,email,access_role,row_version,active FROM users ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.User
	for rows.Next() {
		var item models.User
		var active int
		if err := rows.Scan(&item.ID, &item.Name, &item.Email, &item.Role, &item.RowVersion, &active); err != nil {
			return nil, err
		}
		item.Active = active == 1
		item.AccessRole = item.Role
		result = append(result, item)
	}
	return result, rows.Err()
}
func (s *Store) GetUser(ctx context.Context, id int64) (models.User, error) {
	var item models.User
	var active int
	err := s.db.QueryRowContext(ctx, "SELECT id,name,email,access_role,row_version,active FROM users WHERE id=?", id).Scan(&item.ID, &item.Name, &item.Email, &item.Role, &item.RowVersion, &active)
	item.Active = active == 1
	item.AccessRole = item.Role
	return item, err
}
func (s *Store) SaveUser(ctx context.Context, user *models.User, passwordHash string) error {
	now := nowString()
	if user.Role != "admin" && user.Role != "organizer" && user.Role != "operational" {
		return fmt.Errorf("invalid role")
	}
	legacyRole := "employee"
	if user.Role == "admin" {
		legacyRole = "admin"
	}
	if user.ID == 0 {
		if passwordHash == "" {
			return fmt.Errorf("password required")
		}
		result, err := s.db.ExecContext(ctx, `INSERT INTO users(name,email,password_hash,role,access_role,active,created_at,updated_at) VALUES(?,?,?,?,?,1,?,?)`, user.Name, user.Email, passwordHash, legacyRole, user.Role, now, now)
		if err != nil {
			return err
		}
		user.ID, err = result.LastInsertId()
		return err
	}
	if passwordHash != "" {
		_, err := s.db.ExecContext(ctx, "UPDATE users SET name=?,email=?,password_hash=?,role=?,access_role=?,row_version=row_version+1,updated_at=? WHERE id=?", user.Name, user.Email, passwordHash, legacyRole, user.Role, now, user.ID)
		return err
	}
	_, err := s.db.ExecContext(ctx, "UPDATE users SET name=?,email=?,role=?,access_role=?,row_version=row_version+1,updated_at=? WHERE id=?", user.Name, user.Email, legacyRole, user.Role, now, user.ID)
	return err
}
func (s *Store) ToggleUser(ctx context.Context, id, currentUserID int64) error {
	if id == currentUserID {
		return fmt.Errorf("cannot disable current user")
	}
	_, err := s.db.ExecContext(ctx, "UPDATE users SET active=CASE active WHEN 1 THEN 0 ELSE 1 END,updated_at=? WHERE id=?", nowString(), id)
	return err
}
func (s *Store) SaveCategory(ctx context.Context, name, codePrefix string) error {
	now := nowString()
	_, err := s.db.ExecContext(ctx, `INSERT INTO inventory_categories(name,internal_code_prefix,sort_order,active,created_at,updated_at) VALUES(?,?,(SELECT COALESCE(MAX(sort_order),0)+10 FROM inventory_categories),1,?,?)`, name, codePrefix, now, now)
	return err
}
func (s *Store) SaveLocation(ctx context.Context, name string) error {
	now := nowString()
	_, err := s.db.ExecContext(ctx, `INSERT INTO inventory_locations(name,active,created_at,updated_at) VALUES(?,1,?,?)`, name, now, now)
	return err
}
