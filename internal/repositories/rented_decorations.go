package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"buffetflow/internal/models"
)

const quickRentalCompositionType = "quick_rentals"

func (s *Store) EventRentedDecorationItems(ctx context.Context, eventID int64) ([]models.DecorationCompositionItem, error) {
	if eventID <= 0 {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT item.id,item.composition_id,item.custom_name,item.color,item.quantity,item.origin,COALESCE(item.rental_status,''),item.notes,item.sort_order,item.row_version
		FROM event_decoration_composition_items item
		JOIN event_decoration_compositions composition ON composition.id=item.composition_id
		JOIN event_decoration_profiles profile ON profile.id=composition.profile_id
		WHERE profile.event_id=? AND composition.composition_type=? AND item.origin='rented'
		ORDER BY item.sort_order,item.id`, eventID, quickRentalCompositionType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.DecorationCompositionItem
	for rows.Next() {
		var item models.DecorationCompositionItem
		if err := rows.Scan(&item.ID, &item.CompositionID, &item.Name, &item.Color, &item.Quantity, &item.Origin, &item.RentalStatus, &item.Notes, &item.SortOrder, &item.RowVersion); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) SaveEventRentedDecorationItems(ctx context.Context, eventID int64, items []models.DecorationCompositionItem) error {
	if eventID <= 0 {
		return fmt.Errorf("event is required")
	}
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		var profileID int64
		err := tx.QueryRowContext(ctx, `SELECT id FROM event_decoration_profiles WHERE event_id=?`, eventID).Scan(&profileID)
		if err == sql.ErrNoRows {
			result, insertErr := tx.ExecContext(ctx, `INSERT INTO event_decoration_profiles(event_id,active,row_version,created_at,updated_at) VALUES(?,1,1,?,?)`, eventID, now, now)
			if insertErr != nil {
				return insertErr
			}
			profileID, _ = result.LastInsertId()
		} else if err != nil {
			return err
		}

		var compositionID int64
		err = tx.QueryRowContext(ctx, `SELECT id FROM event_decoration_compositions WHERE profile_id=? AND composition_type=? ORDER BY id LIMIT 1`, profileID, quickRentalCompositionType).Scan(&compositionID)
		if err == sql.ErrNoRows {
			if len(items) == 0 {
				return nil
			}
			result, insertErr := tx.ExecContext(ctx, `INSERT INTO event_decoration_compositions(profile_id,name,composition_type,description,assembly_location,notes,sort_order,row_version,created_at,updated_at) VALUES(?,'Itens alugados do evento',?,'Cadastro rápido feito na edição do evento.','','',900,1,?,?)`, profileID, quickRentalCompositionType, now, now)
			if insertErr != nil {
				return insertErr
			}
			compositionID, _ = result.LastInsertId()
		} else if err != nil {
			return err
		}

		existing := map[int64]bool{}
		rows, err := tx.QueryContext(ctx, `SELECT id FROM event_decoration_composition_items WHERE composition_id=? AND origin='rented'`, compositionID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			existing[id] = true
		}
		if err := rows.Close(); err != nil {
			return err
		}

		kept := map[int64]bool{}
		for index := range items {
			item := &items[index]
			item.Name = strings.TrimSpace(item.Name)
			item.Color = strings.TrimSpace(item.Color)
			if item.Name == "" || item.Quantity <= 0 {
				return fmt.Errorf("rented decoration name and quantity are required")
			}
			if item.ID > 0 && existing[item.ID] {
				result, updateErr := tx.ExecContext(ctx, `UPDATE event_decoration_composition_items SET custom_name=?,color=?,quantity=?,origin='rented',sort_order=?,row_version=row_version+1,updated_at=? WHERE id=? AND composition_id=?`, item.Name, item.Color, item.Quantity, index, now, item.ID, compositionID)
				if updateErr != nil {
					return updateErr
				}
				if changed, _ := result.RowsAffected(); changed > 0 {
					kept[item.ID] = true
					continue
				}
			}
			result, insertErr := tx.ExecContext(ctx, `INSERT INTO event_decoration_composition_items(composition_id,custom_name,color,quantity,origin,rental_status,notes,sort_order,row_version,created_at,updated_at) VALUES(?,?,?,?, 'rented','awaiting_confirmation','',?,1,?,?)`, compositionID, item.Name, item.Color, item.Quantity, index, now, now)
			if insertErr != nil {
				return insertErr
			}
			item.ID, _ = result.LastInsertId()
			kept[item.ID] = true
		}
		for id := range existing {
			if !kept[id] {
				if _, err := tx.ExecContext(ctx, `DELETE FROM event_decoration_composition_items WHERE id=? AND composition_id=?`, id, compositionID); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
