package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"buffetflow/internal/models"
)

const eventNoteColumns = `id,event_id,category,title,content,visit_at,created_at,updated_at,row_version`

func scanEventNote(scanner interface{ Scan(...any) error }) (models.EventNote, error) {
	var note models.EventNote
	var visitAt sql.NullString
	var createdAt, updatedAt string
	err := scanner.Scan(&note.ID, &note.EventID, &note.Category, &note.Title, &note.Content, &visitAt, &createdAt, &updatedAt, &note.RowVersion)
	if visitAt.Valid {
		note.VisitAt = parseTime(visitAt.String)
	}
	note.CreatedAt, note.UpdatedAt = parseTime(createdAt), parseTime(updatedAt)
	return note, err
}

func (s *Store) ListEventNotes(ctx context.Context, eventID int64) ([]models.EventNote, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+eventNoteColumns+` FROM event_notes WHERE event_id=? ORDER BY updated_at DESC, id DESC`, eventID)
	if err != nil {
		return nil, fmt.Errorf("list event notes: %w", err)
	}
	defer rows.Close()
	var notes []models.EventNote
	for rows.Next() {
		note, scanErr := scanEventNote(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		notes = append(notes, note)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(notes) == 0 {
		return notes, nil
	}
	photos, err := s.listEventNotePhotos(ctx, notes)
	if err != nil {
		return nil, err
	}
	byNote := map[int64][]models.EventNotePhoto{}
	for _, photo := range photos {
		byNote[photo.EventNoteID] = append(byNote[photo.EventNoteID], photo)
	}
	for index := range notes {
		notes[index].Photos = byNote[notes[index].ID]
	}
	return notes, nil
}

func (s *Store) listEventNotePhotos(ctx context.Context, notes []models.EventNote) ([]models.EventNotePhoto, error) {
	placeholders := ""
	args := make([]any, 0, len(notes))
	for index, note := range notes {
		if index > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args = append(args, note.ID)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,event_note_id,COALESCE(client_upload_id,''),storage_path,original_name,mime_type,file_size,caption,created_at
		FROM event_note_photos WHERE event_note_id IN (`+placeholders+`) ORDER BY id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var photos []models.EventNotePhoto
	for rows.Next() {
		var photo models.EventNotePhoto
		var createdAt string
		if err := rows.Scan(&photo.ID, &photo.EventNoteID, &photo.ClientUploadID, &photo.StoragePath, &photo.OriginalName, &photo.MIMEType, &photo.FileSize, &photo.Caption, &createdAt); err != nil {
			return nil, err
		}
		photo.CreatedAt = parseTime(createdAt)
		photos = append(photos, photo)
	}
	return photos, rows.Err()
}

func (s *Store) SaveEventNote(ctx context.Context, note *models.EventNote, userID int64, baseVersion int) error {
	now := nowString()
	visitAt := any(nil)
	if !note.VisitAt.IsZero() {
		visitAt = note.VisitAt.UTC().Format(time.RFC3339)
	}
	if note.ID == 0 {
		result, err := s.db.ExecContext(ctx, `INSERT INTO event_notes(event_id,category,title,content,visit_at,created_by,updated_by,created_at,updated_at)
			VALUES(?,?,?,?,?,?,?,?,?)`, note.EventID, note.Category, note.Title, note.Content, visitAt, nullableUserID(userID), nullableUserID(userID), now, now)
		if err != nil {
			return fmt.Errorf("create event note: %w", err)
		}
		note.ID, err = result.LastInsertId()
		return err
	}
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		// Category and technical-visit date were part of an earlier, classified
		// workflow. The current general list does not expose either field; keep
		// old values intact instead of silently discarding historic context.
		query := `UPDATE event_notes SET title=?,content=?,updated_by=?,updated_at=?,row_version=row_version+1 WHERE id=? AND event_id=?`
		args := []any{note.Title, note.Content, nullableUserID(userID), now, note.ID, note.EventID}
		if baseVersion > 0 {
			query += " AND row_version=?"
			args = append(args, baseVersion)
		}
		result, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return err
		}
		count, _ := result.RowsAffected()
		if count == 0 {
			if baseVersion > 0 {
				return fmt.Errorf("version conflict")
			}
			return sql.ErrNoRows
		}
		return nil
	})
}

func (s *Store) DeleteEventNote(ctx context.Context, eventID, noteID int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM event_notes WHERE id=? AND event_id=?`, noteID, eventID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) SaveEventNotePhoto(ctx context.Context, photo *models.EventNotePhoto, userID int64) error {
	clientUploadID := any(nil)
	if photo.ClientUploadID != "" {
		clientUploadID = photo.ClientUploadID
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO event_note_photos(client_upload_id,event_note_id,storage_path,original_name,mime_type,file_size,caption,uploaded_by,created_at)
		VALUES(?,?,?,?,?,?,?,?,?) ON CONFLICT(client_upload_id) DO NOTHING`, clientUploadID, photo.EventNoteID, photo.StoragePath, photo.OriginalName, photo.MIMEType, photo.FileSize, photo.Caption, nullableUserID(userID), nowString())
	if err != nil {
		return err
	}
	if photo.ClientUploadID == "" {
		photo.ID, _ = result.LastInsertId()
	}
	return nil
}

func (s *Store) GetEventNotePhoto(ctx context.Context, id int64) (models.EventNotePhoto, error) {
	var photo models.EventNotePhoto
	var createdAt string
	err := s.db.QueryRowContext(ctx, `SELECT id,event_note_id,COALESCE(client_upload_id,''),storage_path,original_name,mime_type,file_size,caption,created_at FROM event_note_photos WHERE id=?`, id).
		Scan(&photo.ID, &photo.EventNoteID, &photo.ClientUploadID, &photo.StoragePath, &photo.OriginalName, &photo.MIMEType, &photo.FileSize, &photo.Caption, &createdAt)
	photo.CreatedAt = parseTime(createdAt)
	return photo, err
}

func (s *Store) EventNoteBelongsToEvent(ctx context.Context, noteID, eventID int64) bool {
	var count int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM event_notes WHERE id=? AND event_id=?`, noteID, eventID).Scan(&count)
	return count > 0
}
