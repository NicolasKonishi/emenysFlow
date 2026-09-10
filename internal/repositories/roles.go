package repositories

import (
	"context"
	"fmt"

	"buffetflow/internal/models"
)

func (s *Store) ListRoles(ctx context.Context) ([]models.Role, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, slug, name, description FROM roles ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.Role
	for rows.Next() {
		var role models.Role
		if err := rows.Scan(&role.ID, &role.Slug, &role.Name, &role.Description); err != nil {
			return nil, err
		}
		result = append(result, role)
	}
	return result, rows.Err()
}

func (s *Store) EnsureUserHasRoleByEmail(ctx context.Context, email, slug string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO user_roles(user_id, role_id)
		SELECT u.id, r.id FROM users u JOIN roles r ON r.slug=?
		WHERE u.email=? COLLATE NOCASE
		ON CONFLICT(user_id, role_id) DO NOTHING`, slug, email)
	if err != nil {
		return fmt.Errorf("ensure user role: %w", err)
	}
	return nil
}

func (s *Store) replaceUserRoles(ctx context.Context, userID int64, slugs []string) error {
	if _, err := s.db.ExecContext(ctx, "DELETE FROM user_roles WHERE user_id=?", userID); err != nil {
		return err
	}
	for _, slug := range slugs {
		if !models.IsKnownRole(slug) {
			return fmt.Errorf("função inválida")
		}
		result, err := s.db.ExecContext(ctx, `INSERT INTO user_roles(user_id, role_id)
			SELECT ?, id FROM roles WHERE slug=?`, userID, slug)
		if err != nil {
			return err
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			return fmt.Errorf("função %s não encontrada", slug)
		}
	}
	return nil
}

func (s *Store) attachUserRoles(ctx context.Context, users ...*models.User) error {
	ids := make([]int64, 0, len(users))
	index := map[int64]*models.User{}
	for _, user := range users {
		if user == nil || user.ID == 0 {
			continue
		}
		user.Roles = nil
		ids = append(ids, user.ID)
		index[user.ID] = user
	}
	if len(ids) == 0 {
		return nil
	}
	placeholders := ""
	args := make([]any, 0, len(ids))
	for i, id := range ids {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args = append(args, id)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT ur.user_id, r.slug
		FROM user_roles ur JOIN roles r ON r.id=ur.role_id
		WHERE ur.user_id IN (`+placeholders+`)
		ORDER BY r.sort_order, r.id`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var userID int64
		var slug string
		if err := rows.Scan(&userID, &slug); err != nil {
			return err
		}
		if user := index[userID]; user != nil {
			user.Roles = append(user.Roles, slug)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, user := range users {
		if user == nil {
			continue
		}
		if len(user.Roles) == 0 && user.Role != "" {
			switch user.Role {
			case "admin", "organizer":
				user.Roles = []string{models.RoleAdmin}
			case models.RoleAgent:
				user.Roles = []string{models.RoleAgent}
			default:
				user.Roles = []string{models.RoleCorre}
			}
		}
		user.Role = models.PrimaryRole(user.Roles)
		user.AccessRole = models.LegacyAccessRole(user.Roles)
	}
	return nil
}
