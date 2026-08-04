package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"buffetflow/internal/models"
)

func (s *Store) GetDecorationProfile(ctx context.Context, eventID int64) (models.DecorationProfile, error) {
	var profile models.DecorationProfile
	var active int
	err := s.db.QueryRowContext(ctx, `SELECT id,event_id,style,description,primary_colors,theme,notes,responsible_name,active,row_version FROM event_decoration_profiles WHERE event_id=?`, eventID).Scan(&profile.ID, &profile.EventID, &profile.Style, &profile.Description, &profile.PrimaryColors, &profile.Theme, &profile.Notes, &profile.ResponsibleName, &active, &profile.RowVersion)
	if err == sql.ErrNoRows {
		return models.DecorationProfile{EventID: eventID, Active: true}, nil
	}
	if err != nil {
		return profile, err
	}
	profile.Active = active == 1
	rows, err := s.db.QueryContext(ctx, `SELECT id,profile_id,name,composition_type,description,assembly_location,notes,sort_order,row_version FROM event_decoration_compositions WHERE profile_id=? ORDER BY sort_order,id`, profile.ID)
	if err != nil {
		return profile, err
	}
	for rows.Next() {
		var item models.DecorationComposition
		if err := rows.Scan(&item.ID, &item.ProfileID, &item.Name, &item.CompositionType, &item.Description, &item.AssemblyLocation, &item.Notes, &item.SortOrder, &item.RowVersion); err != nil {
			rows.Close()
			return profile, err
		}
		profile.Compositions = append(profile.Compositions, item)
	}
	if err := rows.Close(); err != nil {
		return profile, err
	}
	for index := range profile.Compositions {
		items, err := s.decorationCompositionItems(ctx, profile.Compositions[index].ID)
		if err != nil {
			return profile, err
		}
		profile.Compositions[index].Items = items
		photos, err := s.listReferencePhotos(ctx, eventID, profile.Compositions[index].ID, 0)
		if err != nil {
			return profile, err
		}
		profile.Compositions[index].Photos = photos
	}
	profile.Photos, err = s.listReferencePhotos(ctx, eventID, 0, 0)
	return profile, err
}

func (s *Store) SaveDecorationProfile(ctx context.Context, profile *models.DecorationProfile, userID int64) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		now := nowString()
		var id int64
		err := tx.QueryRowContext(ctx, `SELECT id FROM event_decoration_profiles WHERE event_id=?`, profile.EventID).Scan(&id)
		if err == sql.ErrNoRows {
			result, err := tx.ExecContext(ctx, `INSERT INTO event_decoration_profiles(event_id,style,description,primary_colors,theme,notes,responsible_name,active,row_version,created_at,updated_at) VALUES(?,?,?,?,?,?,?, ?,1,?,?)`, profile.EventID, profile.Style, profile.Description, profile.PrimaryColors, profile.Theme, profile.Notes, profile.ResponsibleName, profile.Active, now, now)
			if err != nil {
				return err
			}
			profile.ID, _ = result.LastInsertId()
			return nil
		}
		if err != nil {
			return err
		}
		profile.ID = id
		_, err = tx.ExecContext(ctx, `UPDATE event_decoration_profiles SET style=?,description=?,primary_colors=?,theme=?,notes=?,responsible_name=?,active=?,row_version=row_version+1,updated_at=? WHERE id=?`, profile.Style, profile.Description, profile.PrimaryColors, profile.Theme, profile.Notes, profile.ResponsibleName, profile.Active, now, id)
		return err
	})
}

func (s *Store) SaveDecorationComposition(ctx context.Context, eventID int64, item *models.DecorationComposition) error {
	profile, err := s.GetDecorationProfile(ctx, eventID)
	if err != nil {
		return err
	}
	if profile.ID == 0 {
		profile.Active = true
		if err = s.SaveDecorationProfile(ctx, &profile, 0); err != nil {
			return err
		}
	}
	now := nowString()
	if item.ID == 0 {
		result, err := s.db.ExecContext(ctx, `INSERT INTO event_decoration_compositions(profile_id,name,composition_type,description,assembly_location,notes,sort_order,row_version,created_at,updated_at) VALUES(?,?,?,?,?,?,?,1,?,?)`, profile.ID, item.Name, item.CompositionType, item.Description, item.AssemblyLocation, item.Notes, item.SortOrder, now, now)
		if err != nil {
			return err
		}
		item.ID, _ = result.LastInsertId()
		return nil
	}
	result, err := s.db.ExecContext(ctx, `UPDATE event_decoration_compositions SET name=?,composition_type=?,description=?,assembly_location=?,notes=?,sort_order=?,row_version=row_version+1,updated_at=? WHERE id=? AND profile_id=?`, item.Name, item.CompositionType, item.Description, item.AssemblyLocation, item.Notes, item.SortOrder, now, item.ID, profile.ID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) RemoveDecorationComposition(ctx context.Context, eventID, compositionID int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM event_decoration_compositions WHERE id=? AND profile_id=(SELECT id FROM event_decoration_profiles WHERE event_id=?)`, compositionID, eventID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) decorationCompositionItems(ctx context.Context, compositionID int64) ([]models.DecorationCompositionItem, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT item.id,item.composition_id,item.decoration_id,item.inventory_item_id,COALESCE(NULLIF(item.custom_name,''),decoration.name,inventory.name,''),item.quantity,item.origin,item.supplier_id,item.supplier_name,item.estimated_cost_cents,item.pickup_at,item.return_at,item.order_reference,COALESCE(item.rental_status,''),item.notes,item.sort_order,item.row_version FROM event_decoration_composition_items item LEFT JOIN decorations decoration ON decoration.id=item.decoration_id LEFT JOIN inventory_items inventory ON inventory.id=item.inventory_item_id WHERE item.composition_id=? ORDER BY item.sort_order,item.id`, compositionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.DecorationCompositionItem
	for rows.Next() {
		var item models.DecorationCompositionItem
		var pickup, returned sql.NullString
		if err := rows.Scan(&item.ID, &item.CompositionID, &item.DecorationID, &item.InventoryItemID, &item.Name, &item.Quantity, &item.Origin, &item.SupplierID, &item.SupplierName, &item.EstimatedCostCents, &pickup, &returned, &item.OrderReference, &item.RentalStatus, &item.Notes, &item.SortOrder, &item.RowVersion); err != nil {
			return nil, err
		}
		if pickup.Valid {
			item.PickupAt = parseTime(pickup.String)
		}
		if returned.Valid {
			item.ReturnAt = parseTime(returned.String)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) SaveDecorationCompositionItem(ctx context.Context, eventID int64, item *models.DecorationCompositionItem) error {
	var valid int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM event_decoration_compositions composition JOIN event_decoration_profiles profile ON profile.id=composition.profile_id WHERE composition.id=? AND profile.event_id=?`, item.CompositionID, eventID).Scan(&valid); err != nil {
		return err
	}
	if valid == 0 {
		return sql.ErrNoRows
	}
	if item.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	if item.Origin == "" {
		item.Origin = "owned"
	}
	pickup := timeOrNil(item.PickupAt)
	returned := timeOrNil(item.ReturnAt)
	now := nowString()
	if item.ID == 0 {
		result, err := s.db.ExecContext(ctx, `INSERT INTO event_decoration_composition_items(composition_id,decoration_id,inventory_item_id,custom_name,quantity,origin,supplier_id,supplier_name,estimated_cost_cents,pickup_at,return_at,order_reference,rental_status,notes,sort_order,row_version,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,1,?,?)`, item.CompositionID, nullInt64(item.DecorationID), nullInt64(item.InventoryItemID), item.Name, item.Quantity, item.Origin, nullInt64(item.SupplierID), item.SupplierName, nullInt64(item.EstimatedCostCents), pickup, returned, item.OrderReference, nullIfEmpty(item.RentalStatus), item.Notes, item.SortOrder, now, now)
		if err != nil {
			return err
		}
		item.ID, _ = result.LastInsertId()
		return nil
	}
	result, err := s.db.ExecContext(ctx, `UPDATE event_decoration_composition_items SET decoration_id=?,inventory_item_id=?,custom_name=?,quantity=?,origin=?,supplier_id=?,supplier_name=?,estimated_cost_cents=?,pickup_at=?,return_at=?,order_reference=?,rental_status=?,notes=?,sort_order=?,row_version=row_version+1,updated_at=? WHERE id=? AND composition_id=?`, nullInt64(item.DecorationID), nullInt64(item.InventoryItemID), item.Name, item.Quantity, item.Origin, nullInt64(item.SupplierID), item.SupplierName, nullInt64(item.EstimatedCostCents), pickup, returned, item.OrderReference, nullIfEmpty(item.RentalStatus), item.Notes, item.SortOrder, now, item.ID, item.CompositionID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) RemoveDecorationCompositionItem(ctx context.Context, eventID, itemID int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM event_decoration_composition_items WHERE id=? AND composition_id IN (SELECT composition.id FROM event_decoration_compositions composition JOIN event_decoration_profiles profile ON profile.id=composition.profile_id WHERE profile.event_id=?)`, itemID, eventID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func timeOrNil(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value.UTC().Format(time.RFC3339)
}
func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (s *Store) listReferencePhotos(ctx context.Context, eventID, compositionID, itemID int64) ([]models.ReferencePhoto, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,COALESCE(client_upload_id,''),event_id,composition_id,composition_item_id,storage_path,original_name,mime_type,file_size,caption,sort_order,is_primary,created_at FROM event_reference_photos WHERE event_id=? AND deleted_at IS NULL AND ((?=0 AND composition_id IS NULL AND composition_item_id IS NULL) OR (?<>0 AND composition_id=?) OR (?<>0 AND composition_item_id=?)) ORDER BY is_primary DESC,sort_order,id`, eventID, compositionID, compositionID, compositionID, itemID, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.ReferencePhoto
	for rows.Next() {
		var item models.ReferencePhoto
		var primary int
		var created string
		if err := rows.Scan(&item.ID, &item.ClientUploadID, &item.EventID, &item.CompositionID, &item.CompositionItemID, &item.StoragePath, &item.OriginalName, &item.MIMEType, &item.FileSize, &item.Caption, &item.SortOrder, &primary, &created); err != nil {
			return nil, err
		}
		item.Primary = primary == 1
		item.CreatedAt = parseTime(created)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) SaveReferencePhoto(ctx context.Context, photo *models.ReferencePhoto, userID int64) error {
	now := nowString()
	return s.db.QueryRowContext(ctx, `INSERT INTO event_reference_photos(client_upload_id,event_id,composition_id,composition_item_id,storage_path,original_name,mime_type,file_size,caption,sort_order,is_primary,uploaded_by,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(client_upload_id) DO UPDATE SET caption=excluded.caption RETURNING id`, nullIfEmpty(photo.ClientUploadID), photo.EventID, nullInt64(photo.CompositionID), nullInt64(photo.CompositionItemID), photo.StoragePath, photo.OriginalName, photo.MIMEType, photo.FileSize, photo.Caption, photo.SortOrder, photo.Primary, nullableUserID(userID), now).Scan(&photo.ID)
}
func (s *Store) GetReferencePhoto(ctx context.Context, id int64) (models.ReferencePhoto, error) {
	var item models.ReferencePhoto
	var primary int
	var created string
	err := s.db.QueryRowContext(ctx, `SELECT id,COALESCE(client_upload_id,''),event_id,composition_id,composition_item_id,storage_path,original_name,mime_type,file_size,caption,sort_order,is_primary,created_at FROM event_reference_photos WHERE id=? AND deleted_at IS NULL`, id).Scan(&item.ID, &item.ClientUploadID, &item.EventID, &item.CompositionID, &item.CompositionItemID, &item.StoragePath, &item.OriginalName, &item.MIMEType, &item.FileSize, &item.Caption, &item.SortOrder, &primary, &created)
	item.Primary = primary == 1
	item.CreatedAt = parseTime(created)
	return item, err
}
func (s *Store) RemoveReferencePhoto(ctx context.Context, eventID, photoID int64) error {
	result, err := s.db.ExecContext(ctx, `UPDATE event_reference_photos SET deleted_at=? WHERE id=? AND event_id=? AND deleted_at IS NULL`, nowString(), photoID, eventID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) SyncDecorationRentalChecklist(ctx context.Context, eventID int64) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		var checklistID, categoryID int64
		if err := tx.QueryRowContext(ctx, `SELECT id FROM checklists WHERE event_id=?`, eventID).Scan(&checklistID); err != nil {
			return err
		}
		if err := tx.QueryRowContext(ctx, `SELECT id FROM inventory_categories WHERE name='Itens alugados'`).Scan(&categoryID); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT item.id,COALESCE(NULLIF(item.custom_name,''),decoration.name,inventory.name),item.quantity,COALESCE(item.rental_status,''),item.supplier_name,item.notes FROM event_decoration_composition_items item JOIN event_decoration_compositions composition ON composition.id=item.composition_id JOIN event_decoration_profiles profile ON profile.id=composition.profile_id LEFT JOIN decorations decoration ON decoration.id=item.decoration_id LEFT JOIN inventory_items inventory ON inventory.id=item.inventory_item_id WHERE profile.event_id=? AND profile.active=1 AND item.origin='rented'`, eventID)
		if err != nil {
			return err
		}
		type rental struct {
			id                      int64
			name                    string
			quantity                float64
			status, supplier, notes string
		}
		var rentals []rental
		for rows.Next() {
			var item rental
			if err := rows.Scan(&item.id, &item.name, &item.quantity, &item.status, &item.supplier, &item.notes); err != nil {
				rows.Close()
				return err
			}
			rentals = append(rentals, item)
		}
		if err := rows.Close(); err != nil {
			return err
		}
		now := nowString()
		activeKeys := map[string]bool{}
		for _, item := range rentals {
			key := fmt.Sprintf("decoration-rental:%d", item.id)
			activeKeys[key] = true
			confirmed := item.status == "confirmed" || item.status == "picked_up" || item.status == "delivered" || item.status == "returned"
			available := 0.0
			if confirmed {
				available = item.quantity
			}
			missing := item.quantity - available
			_, err := tx.ExecContext(ctx, `INSERT INTO checklist_items(checklist_id,category_id,source_key,name,unit,calculated_quantity,required_quantity,available_quantity,missing_quantity,calculation_origin,notes,status,item_kind,manual_item,active,row_version,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,? ,?,'pending','rented',0,1,1,?,?) ON CONFLICT(checklist_id,source_key) DO UPDATE SET name=excluded.name,calculated_quantity=excluded.calculated_quantity,required_quantity=excluded.required_quantity,available_quantity=excluded.available_quantity,missing_quantity=excluded.missing_quantity,notes=excluded.notes,active=1,row_version=checklist_items.row_version+1,updated_at=excluded.updated_at`, checklistID, categoryID, key, item.name, "unidade", item.quantity, item.quantity, available, missing, "Decoração alugada para o evento", item.notes, now, now)
			if err != nil {
				return err
			}
			if !confirmed {
				var checklistItemID int64
				if err := tx.QueryRowContext(ctx, `SELECT id FROM checklist_items WHERE checklist_id=? AND source_key=?`, checklistID, key).Scan(&checklistItemID); err != nil {
					return err
				}
				var count int
				if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM checklist_shortages WHERE checklist_item_id=? AND status NOT IN ('resolved','cancelled')`, checklistItemID).Scan(&count); err != nil {
					return err
				}
				if count == 0 {
					result, err := tx.ExecContext(ctx, `INSERT INTO checklist_shortages(checklist_item_id,event_id,missing_quantity,reason,resolution_type,status,supplier_name,notes,automatic,row_version,created_at,updated_at) VALUES(?,?,?,'Aluguel da decoração ainda não confirmado.','rental','renting',?,?,1,1,?,?)`, checklistItemID, eventID, item.quantity, item.supplier, item.notes, now, now)
					if err != nil {
						return err
					}
					shortageID, _ := result.LastInsertId()
					if _, err = tx.ExecContext(ctx, `INSERT INTO checklist_shortage_history(shortage_id,new_status,notes,created_at) VALUES(?,'renting','Aluguel pendente de confirmação.',?)`, shortageID, now); err != nil {
						return err
					}
				}
			}
		}
		rows, err = tx.QueryContext(ctx, `SELECT id,source_key FROM checklist_items WHERE checklist_id=? AND source_key LIKE 'decoration-rental:%'`, checklistID)
		if err != nil {
			return err
		}
		type stored struct {
			id  int64
			key string
		}
		var storedItems []stored
		for rows.Next() {
			var item stored
			if err := rows.Scan(&item.id, &item.key); err != nil {
				rows.Close()
				return err
			}
			storedItems = append(storedItems, item)
		}
		if err := rows.Close(); err != nil {
			return err
		}
		for _, item := range storedItems {
			if !activeKeys[item.key] {
				if _, err := tx.ExecContext(ctx, `UPDATE checklist_items SET active=0,row_version=row_version+1,updated_at=? WHERE id=?`, now, item.id); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
