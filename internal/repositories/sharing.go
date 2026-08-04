package repositories

import (
	"context"
	"database/sql"
	"fmt"
)

func (s *Store) CreateEventShare(ctx context.Context, eventID int64, tokenHash string, userID int64) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO event_share_tokens(token_hash,event_id,active,created_by,created_at) VALUES(?,?,1,?,?)`, tokenHash, eventID, nullableUserID(userID), nowString())
	if err != nil {
		return fmt.Errorf("create event share: %w", err)
	}
	return nil
}

func (s *Store) EventByShareToken(ctx context.Context, tokenHash string) (modelsEventID int64, err error) {
	err = s.db.QueryRowContext(ctx, `SELECT event_id FROM event_share_tokens WHERE token_hash=? AND active=1 AND (expires_at IS NULL OR expires_at>?)`, tokenHash, nowString()).Scan(&modelsEventID)
	if err == sql.ErrNoRows {
		return 0, err
	}
	return modelsEventID, err
}
