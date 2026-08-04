package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"buffetflow/internal/models"
)

func (s *Store) EnsureDemoAdmin(ctx context.Context, passwordHash string) error {
	now := nowString()
	_, err := s.db.ExecContext(ctx, `INSERT INTO users(name, email, password_hash, role, access_role, active, created_at, updated_at)
		VALUES('Administrador', 'admin@buffet.local', ?, 'admin', 'admin', 1, ?, ?)
		ON CONFLICT(email) DO NOTHING`, passwordHash, now, now)
	if err != nil {
		return fmt.Errorf("ensure demo admin: %w", err)
	}
	return nil
}

func (s *Store) UserByEmail(ctx context.Context, email string) (models.User, error) {
	var user models.User
	var active int
	err := s.db.QueryRowContext(ctx, `SELECT id, name, email, password_hash, access_role, row_version, active
		FROM users WHERE email = ? COLLATE NOCASE`, email).Scan(
		&user.ID, &user.Name, &user.Email, &user.Password, &user.Role, &user.RowVersion, &active,
	)
	user.AccessRole = user.Role
	user.Active = active == 1
	return user, err
}

func (s *Store) UserBySession(ctx context.Context, tokenHash string) (models.User, error) {
	var user models.User
	var active int
	err := s.db.QueryRowContext(ctx, `SELECT u.id, u.name, u.email, u.access_role, u.row_version, u.active
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ? AND s.expires_at > ?`, tokenHash, nowString()).Scan(
		&user.ID, &user.Name, &user.Email, &user.Role, &user.RowVersion, &active,
	)
	user.AccessRole = user.Role
	user.Active = active == 1
	return user, err
}

func (s *Store) CreateSession(ctx context.Context, tokenHash string, userID int64, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO sessions(token_hash, user_id, expires_at, created_at)
		VALUES(?, ?, ?, ?)`, tokenHash, userID, expiresAt.UTC().Format(time.RFC3339), nowString())
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE token_hash = ?", tokenHash)
	return err
}

func (s *Store) DeleteExpiredSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at <= ?", nowString())
	return err
}

func nullableUserID(userID int64) sql.NullInt64 {
	return sql.NullInt64{Int64: userID, Valid: userID > 0}
}
