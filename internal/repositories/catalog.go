package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"time"

	"buffetflow/internal/models"
)

func (s *Store) ListMenuCategories(ctx context.Context) ([]models.MenuCategory, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id,name,COALESCE(slug,''),sort_order FROM menu_categories WHERE active=1 ORDER BY sort_order,name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.MenuCategory
	for rows.Next() {
		var item models.MenuCategory
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.SortOrder); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) MenuCategorySlug(ctx context.Context, id int64) (string, error) {
	var slug string
	err := s.db.QueryRowContext(ctx, `SELECT slug FROM menu_categories WHERE id=?`, id).Scan(&slug)
	return slug, err
}

func (s *Store) ListContainerTypes(ctx context.Context, includeInactive bool) ([]models.ContainerType, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT c.id,c.name,c.capacity_portions,c.disposable,c.requires_lid,c.is_default,c.transport_notes,c.inventory_item_id,COALESCE(i.name,''),c.quantity_mode,c.required_utensil_type,c.custom_utensil_name,c.fixed_quantity,c.active
		FROM container_types c LEFT JOIN inventory_items i ON i.id=c.inventory_item_id WHERE (?=1 OR c.active=1) ORDER BY c.name`, includeInactive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.ContainerType
	for rows.Next() {
		var item models.ContainerType
		var disposable, lid, def, active int
		if err := rows.Scan(&item.ID, &item.Name, &item.CapacityPortions, &disposable, &lid, &def, &item.TransportNotes, &item.InventoryItemID, &item.InventoryItemName, &item.QuantityMode, &item.RequiredUtensilType, &item.CustomUtensilName, &item.FixedQuantity, &active); err != nil {
			return nil, err
		}
		item.Disposable, item.RequiresLid, item.IsDefault, item.Active = disposable == 1, lid == 1, def == 1, active == 1
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) GetContainerType(ctx context.Context, id int64) (models.ContainerType, error) {
	var item models.ContainerType
	var disposable, lid, def, active int
	err := s.db.QueryRowContext(ctx, `SELECT c.id,c.name,c.capacity_portions,c.disposable,c.requires_lid,c.is_default,c.transport_notes,c.inventory_item_id,COALESCE(i.name,''),c.quantity_mode,c.required_utensil_type,c.custom_utensil_name,c.fixed_quantity,c.active FROM container_types c LEFT JOIN inventory_items i ON i.id=c.inventory_item_id WHERE c.id=?`, id).Scan(&item.ID, &item.Name, &item.CapacityPortions, &disposable, &lid, &def, &item.TransportNotes, &item.InventoryItemID, &item.InventoryItemName, &item.QuantityMode, &item.RequiredUtensilType, &item.CustomUtensilName, &item.FixedQuantity, &active)
	item.Disposable, item.RequiresLid, item.IsDefault, item.Active = disposable == 1, lid == 1, def == 1, active == 1
	return item, err
}

func (s *Store) SaveContainerType(ctx context.Context, item *models.ContainerType) error {
	now := nowString()
	capacity, inventory := any(nil), any(nil)
	if item.CapacityPortions.Valid {
		capacity = item.CapacityPortions.Float64
	}
	if item.InventoryItemID.Valid {
		inventory = item.InventoryItemID.Int64
	}
	if item.QuantityMode == "" {
		item.QuantityMode = "per_event_type"
	}
	if item.RequiredUtensilType == "" {
		item.RequiredUtensilType = "none"
	}
	fixedQuantity := any(nil)
	if item.FixedQuantity.Valid {
		fixedQuantity = item.FixedQuantity.Float64
	}
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		if item.IsDefault {
			if _, err := tx.ExecContext(ctx, "UPDATE container_types SET is_default=0,updated_at=?", now); err != nil {
				return err
			}
		}
		if item.ID == 0 {
			result, err := tx.ExecContext(ctx, `INSERT INTO container_types(name,capacity_portions,disposable,requires_lid,is_default,transport_notes,inventory_item_id,quantity_mode,required_utensil_type,custom_utensil_name,fixed_quantity,active,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,1,?,?)`, item.Name, capacity, item.Disposable, item.RequiresLid, item.IsDefault, item.TransportNotes, inventory, item.QuantityMode, item.RequiredUtensilType, item.CustomUtensilName, fixedQuantity, now, now)
			if err != nil {
				return err
			}
			item.ID, err = result.LastInsertId()
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE container_types SET name=?,capacity_portions=?,disposable=?,requires_lid=?,is_default=?,transport_notes=?,inventory_item_id=?,quantity_mode=?,required_utensil_type=?,custom_utensil_name=?,fixed_quantity=?,updated_at=? WHERE id=?`, item.Name, capacity, item.Disposable, item.RequiresLid, item.IsDefault, item.TransportNotes, inventory, item.QuantityMode, item.RequiredUtensilType, item.CustomUtensilName, fixedQuantity, now, item.ID)
		return err
	})
}

func (s *Store) ToggleContainerType(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "UPDATE container_types SET active=CASE active WHEN 1 THEN 0 ELSE 1 END,updated_at=? WHERE id=?", nowString(), id)
	return err
}

const menuItemSelect = `SELECT m.id,m.category_id,c.name,COALESCE(NULLIF(m.display_name,''),m.name),m.description,m.container_type_id,COALESCE(ct.name,''),m.container_capacity_portions,m.pan_inventory_item_id,COALESCE(p.name,''),m.transport_inventory_item_id,COALESCE(t.name,''),m.result_inventory_item_id,COALESCE(ri.name,''),m.calculation_type,m.calculation_group,m.calculation_divisor,m.calculation_multiplier,m.calculation_weight,m.template_owner_id,m.source_menu_item_id,m.active FROM menu_items m JOIN menu_categories c ON c.id=m.category_id LEFT JOIN container_types ct ON ct.id=m.container_type_id LEFT JOIN inventory_items p ON p.id=m.pan_inventory_item_id LEFT JOIN inventory_items t ON t.id=m.transport_inventory_item_id LEFT JOIN inventory_items ri ON ri.id=m.result_inventory_item_id`

func scanMenuItem(scanner interface{ Scan(...any) error }) (models.MenuItem, error) {
	var item models.MenuItem
	var active int
	err := scanner.Scan(&item.ID, &item.CategoryID, &item.CategoryName, &item.Name, &item.Description, &item.ContainerTypeID, &item.ContainerTypeName, &item.ContainerCapacity, &item.PanInventoryItemID, &item.PanInventoryItemName, &item.TransportInventoryItemID, &item.TransportInventoryItemName, &item.ResultInventoryItemID, &item.ResultInventoryItemName, &item.CalculationType, &item.CalculationGroup, &item.CalculationDivisor, &item.CalculationMultiplier, &item.CalculationWeight, &item.TemplateOwnerID, &item.SourceMenuItemID, &active)
	item.Active = active == 1
	return item, err
}

func (s *Store) ListMenuItems(ctx context.Context, includeInactive bool) ([]models.MenuItem, error) {
	rows, err := s.db.QueryContext(ctx, menuItemSelect+` WHERE m.template_owner_id IS NULL AND (?=1 OR m.active=1) ORDER BY c.sort_order,m.name`, includeInactive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.MenuItem
	for rows.Next() {
		item, err := scanMenuItem(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) GetMenuItem(ctx context.Context, id int64) (models.MenuItem, error) {
	item, err := scanMenuItem(s.db.QueryRowContext(ctx, menuItemSelect+` WHERE m.id=?`, id))
	if err != nil {
		return item, err
	}
	item.Equipment, err = s.MenuItemEquipment(ctx, id)
	if err != nil {
		return item, err
	}
	item.Ingredients, err = s.ListMenuItemIngredients(ctx, id)
	return item, err
}

func (s *Store) EquipmentOptions(ctx context.Context) ([]models.EquipmentLink, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT e.id,i.id,i.name,1,1 FROM equipment e JOIN inventory_items i ON i.id=e.inventory_item_id WHERE e.active=1 AND i.active=1 ORDER BY i.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.EquipmentLink
	for rows.Next() {
		var item models.EquipmentLink
		var required int
		if err := rows.Scan(&item.EquipmentID, &item.InventoryItemID, &item.Name, &item.Quantity, &required); err != nil {
			return nil, err
		}
		item.Required = required == 1
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) MenuItemEquipment(ctx context.Context, menuItemID int64) ([]models.EquipmentLink, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT e.id,i.id,i.name,me.quantity,me.required FROM menu_item_equipment me JOIN equipment e ON e.id=me.equipment_id JOIN inventory_items i ON i.id=e.inventory_item_id WHERE me.menu_item_id=? ORDER BY i.name`, menuItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.EquipmentLink
	for rows.Next() {
		var item models.EquipmentLink
		var required int
		if err := rows.Scan(&item.EquipmentID, &item.InventoryItemID, &item.Name, &item.Quantity, &required); err != nil {
			return nil, err
		}
		item.Required, item.Selected = required == 1, true
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) SaveMenuItem(ctx context.Context, item *models.MenuItem, equipment []models.EquipmentLink) error {
	now := nowString()
	container, capacity, pan, transport, resultItem := any(nil), any(nil), any(nil), any(nil), any(nil)
	if item.ContainerTypeID.Valid {
		container = item.ContainerTypeID.Int64
	}
	if item.ContainerCapacity.Valid {
		capacity = item.ContainerCapacity.Float64
	}
	if item.PanInventoryItemID.Valid {
		pan = item.PanInventoryItemID.Int64
	}
	if item.TransportInventoryItemID.Valid {
		transport = item.TransportInventoryItemID.Int64
	}
	if item.ResultInventoryItemID.Valid {
		resultItem = item.ResultInventoryItemID.Int64
	}
	if item.CalculationType == "" {
		item.CalculationType = "menu_only"
	}
	if item.CalculationDivisor <= 0 {
		item.CalculationDivisor = 1
	}
	if item.CalculationWeight <= 0 {
		item.CalculationWeight = 1
	}
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		if item.ID == 0 {
			storageName, displayName, templateOwner, sourceItem := item.Name, "", any(nil), any(nil)
			if item.TemplateOwnerID.Valid {
				storageName = fmt.Sprintf("__template_%d_%d", item.TemplateOwnerID.Int64, time.Now().UnixNano())
				displayName = item.Name
				templateOwner = item.TemplateOwnerID.Int64
			}
			if item.SourceMenuItemID.Valid {
				sourceItem = item.SourceMenuItemID.Int64
			}
			result, err := tx.ExecContext(ctx, `INSERT INTO menu_items(category_id,name,display_name,description,container_type_id,container_capacity_portions,pan_inventory_item_id,transport_inventory_item_id,result_inventory_item_id,calculation_type,calculation_group,calculation_divisor,calculation_multiplier,calculation_weight,template_owner_id,source_menu_item_id,active,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,1,?,?)`, item.CategoryID, storageName, displayName, item.Description, container, capacity, pan, transport, resultItem, item.CalculationType, item.CalculationGroup, item.CalculationDivisor, item.CalculationMultiplier, item.CalculationWeight, templateOwner, sourceItem, now, now)
			if err != nil {
				return err
			}
			item.ID, err = result.LastInsertId()
			if err != nil {
				return err
			}
		} else {
			nameColumn := "name"
			if item.TemplateOwnerID.Valid {
				nameColumn = "display_name"
			}
			query := fmt.Sprintf(`UPDATE menu_items SET category_id=?,%s=?,description=?,container_type_id=?,container_capacity_portions=?,pan_inventory_item_id=?,transport_inventory_item_id=?,result_inventory_item_id=?,calculation_type=?,calculation_group=?,calculation_divisor=?,calculation_multiplier=?,calculation_weight=?,updated_at=? WHERE id=?`, nameColumn)
			if _, err := tx.ExecContext(ctx, query, item.CategoryID, item.Name, item.Description, container, capacity, pan, transport, resultItem, item.CalculationType, item.CalculationGroup, item.CalculationDivisor, item.CalculationMultiplier, item.CalculationWeight, now, item.ID); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM menu_item_equipment WHERE menu_item_id=?", item.ID); err != nil {
			return err
		}
		for _, link := range equipment {
			if !link.Selected {
				continue
			}
			if link.Quantity <= 0 {
				link.Quantity = 1
			}
			if _, err := tx.ExecContext(ctx, "INSERT INTO menu_item_equipment(menu_item_id,equipment_id,quantity,required) VALUES(?,?,?,?)", item.ID, link.EquipmentID, link.Quantity, link.Required); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) ToggleMenuItem(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "UPDATE menu_items SET active=CASE active WHEN 1 THEN 0 ELSE 1 END,updated_at=? WHERE id=?", nowString(), id)
	return err
}

func (s *Store) EventMenuSelection(ctx context.Context, eventID int64) ([]models.EventMenuItem, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT m.id,COALESCE(NULLIF(m.display_name,''),m.name),c.name,m.template_owner_id,m.source_menu_item_id,COALESCE(em.id,0),COALESCE(em.portions,0)
		FROM menu_items m
		JOIN menu_categories c ON c.id=m.category_id
		LEFT JOIN event_menu_items em ON em.menu_item_id=m.id AND em.event_id=?
		WHERE (m.active=1 OR em.id IS NOT NULL) AND (
			(m.template_owner_id IS NULL AND (?=0 OR NOT EXISTS(
				SELECT 1 FROM events current_event JOIN menu_items clone ON clone.template_owner_id=current_event.template_id
				WHERE current_event.id=? AND clone.source_menu_item_id=m.id AND clone.active=1
			)))
			OR (?=0 AND m.template_owner_id IS NOT NULL AND EXISTS(SELECT 1 FROM event_templates et WHERE et.id=m.template_owner_id AND et.active=1))
			OR m.template_owner_id=(SELECT template_id FROM events WHERE id=?)
			OR em.id IS NOT NULL
		)
		ORDER BY c.sort_order,COALESCE(NULLIF(m.display_name,''),m.name)`, eventID, eventID, eventID, eventID, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.EventMenuItem
	for rows.Next() {
		var item models.EventMenuItem
		var selectedID int64
		if err := rows.Scan(&item.MenuItemID, &item.MenuItemName, &item.CategoryName, &item.TemplateOwnerID, &item.SourceMenuItemID, &selectedID, &item.Portions); err != nil {
			return nil, err
		}
		item.EventID = eventID
		item.Selected = selectedID > 0
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) SaveEventMenu(ctx context.Context, eventID int64, items []models.EventMenuItem) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM event_menu_items WHERE event_id=?", eventID); err != nil {
			return err
		}
		now := nowString()
		for _, item := range items {
			if !item.Selected {
				continue
			}
			if item.Portions <= 0 {
				continue
			}
			_, err := tx.ExecContext(ctx, `INSERT INTO event_menu_items(event_id,menu_item_id,portions,container_type_id,calculated_container_quantity,notes,created_at,updated_at) SELECT ?,m.id,?,m.container_type_id,CASE WHEN COALESCE(m.container_capacity_portions,ct.capacity_portions,0)>0 THEN CEIL(?/COALESCE(m.container_capacity_portions,ct.capacity_portions)) ELSE 1 END,'',?,? FROM menu_items m LEFT JOIN container_types ct ON ct.id=m.container_type_id WHERE m.id=?`, eventID, item.Portions, item.Portions, now, now, item.MenuItemID)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) EventMenuRequirements(ctx context.Context, eventID int64) ([]models.AutomaticRequirement, error) {
	type aggregate struct {
		quantity float64
		origin   string
	}
	values := map[string]aggregate{}
	inventoryByKey := map[string]int64{}
	var guests int
	var eventMargin sql.NullFloat64
	if err := s.db.QueryRowContext(ctx, `SELECT guest_count,additional_guest_margin_override FROM events WHERE id=?`, eventID).Scan(&guests, &eventMargin); err != nil {
		return nil, err
	}
	additionalMargin := 20.0
	_ = s.db.QueryRowContext(ctx, `SELECT numeric_value FROM operational_settings WHERE setting_key='additional_staff_margin'`).Scan(&additionalMargin)
	if eventMargin.Valid {
		additionalMargin = eventMargin.Float64
	}
	var cubaInventoryID int64
	_ = s.db.QueryRowContext(ctx, `SELECT id FROM inventory_items WHERE internal_code='CUB-001'`).Scan(&cubaInventoryID)
	cubaCount := 0.0
	rows, err := s.db.QueryContext(ctx, `SELECT em.id,COALESCE(NULLIF(m.display_name,''),m.name),category.slug,em.portions,COALESCE(em.overridden_container_quantity,em.calculated_container_quantity,1),ct.inventory_item_id,COALESCE(ct.quantity_mode,'per_event_type'),ct.fixed_quantity,m.pan_inventory_item_id,m.transport_inventory_item_id FROM event_menu_items em JOIN menu_items m ON m.id=em.menu_item_id JOIN menu_categories category ON category.id=m.category_id LEFT JOIN container_types ct ON ct.id=COALESCE(em.container_type_id,m.container_type_id) WHERE em.event_id=?`, eventID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id int64
		var name string
		var categorySlug string
		var portions int
		var containers float64
		var quantityMode string
		var fixedQuantity sql.NullFloat64
		var containerID, panID, transportID sql.NullInt64
		if err := rows.Scan(&id, &name, &categorySlug, &portions, &containers, &containerID, &quantityMode, &fixedQuantity, &panID, &transportID); err != nil {
			rows.Close()
			return nil, err
		}
		if categorySlug == "main_courses" || categorySlug == "sides" {
			cubaCount++
		} else if containerID.Valid {
			key := fmt.Sprintf("menu-container:%d", containerID.Int64)
			old := values[key]
			switch quantityMode {
			case "per_event_type":
				old.quantity = math.Max(old.quantity, float64(guests)+additionalMargin)
				old.origin = fmt.Sprintf("Selecionado como recipiente: %d convidados + %.0f de margem operacional", guests, additionalMargin)
			case "per_serving":
				old.quantity += float64(portions)
				old.origin = "Recipiente por porção do item selecionado"
			case "fixed":
				quantity := 1.0
				if fixedQuantity.Valid {
					quantity = fixedQuantity.Float64
				}
				old.quantity = math.Max(old.quantity, quantity)
				old.origin = "Quantidade fixa configurada no recipiente"
			case "manual":
				old.quantity += math.Ceil(containers)
				old.origin = "Quantidade manual de recipientes no evento"
			default:
				old.quantity += math.Ceil(containers)
				old.origin = "Recipientes conforme a capacidade de cada item"
			}
			values[key] = old
			inventoryByKey[key] = containerID.Int64
		}
		if panID.Valid {
			key := fmt.Sprintf("menu-pan:%d", panID.Int64)
			old := values[key]
			old.quantity++
			old.origin = "Cardápio: uma panela por prato vinculado"
			values[key] = old
			inventoryByKey[key] = panID.Int64
		}
		if transportID.Valid {
			key := fmt.Sprintf("menu-transport:%d", transportID.Int64)
			old := values[key]
			old.quantity++
			old.origin = "Cardápio: recipiente de transporte vinculado"
			values[key] = old
			inventoryByKey[key] = transportID.Int64
		}
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if cubaCount > 0 && cubaInventoryID > 0 {
		serviceLines := 1.0
		if guests > 100 {
			serviceLines = 2
		}
		cubaCount *= serviceLines
		key := "menu-cubas"
		origin := fmt.Sprintf("Uma cuba por prato principal ou acompanhamento = %.0f cubas", cubaCount)
		if serviceLines == 2 {
			origin = fmt.Sprintf("Evento acima de 100 convidados: duas filas de buffet = %.0f cubas", cubaCount)
		}
		values[key] = aggregate{quantity: cubaCount, origin: origin}
		inventoryByKey[key] = cubaInventoryID
	}
	rows, err = s.db.QueryContext(ctx, `SELECT m.id,m.result_inventory_item_id,m.calculation_type,m.calculation_group,m.calculation_divisor,m.calculation_multiplier,m.calculation_weight,em.portions,COALESCE(NULLIF(m.display_name,''),m.name) FROM event_menu_items em JOIN menu_items m ON m.id=em.menu_item_id WHERE em.event_id=? AND m.result_inventory_item_id IS NOT NULL AND m.calculation_type<>'menu_only' ORDER BY m.calculation_group,m.id`, eventID)
	if err != nil {
		return nil, err
	}
	type calculated struct {
		id, inventoryID             int64
		calcType, group, name       string
		divisor, multiplier, weight float64
		portions                    int
	}
	var calculatedItems []calculated
	for rows.Next() {
		var item calculated
		if err := rows.Scan(&item.id, &item.inventoryID, &item.calcType, &item.group, &item.divisor, &item.multiplier, &item.weight, &item.portions, &item.name); err != nil {
			rows.Close()
			return nil, err
		}
		calculatedItems = append(calculatedItems, item)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	groups := map[string][]calculated{}
	for _, item := range calculatedItems {
		if item.calcType == "category_distribution" {
			groups[item.group] = append(groups[item.group], item)
			continue
		}
		quantity := 1.0
		switch item.calcType {
		case "per_person":
			quantity = math.Ceil(float64(item.portions) * item.multiplier)
		case "group_of_people":
			quantity = math.Ceil(float64(item.portions)/item.divisor) * item.multiplier
		case "fixed":
			quantity = item.multiplier
		}
		key := fmt.Sprintf("menu-result:%d", item.id)
		values[key] = aggregate{quantity: quantity, origin: "Item selecionado no cardápio: " + item.name}
		inventoryByKey[key] = item.inventoryID
	}

	for group, groupItems := range groups {
		if len(groupItems) == 0 {
			continue
		}
		total := int(math.Ceil(float64(groupItems[0].portions)/groupItems[0].divisor) * groupItems[0].multiplier)
		totalWeight := 0.0
		for index := range groupItems {
			if groupItems[index].weight <= 0 {
				groupItems[index].weight = 1
			}
			totalWeight += groupItems[index].weight
		}
		type weightedShare struct {
			index     int
			quantity  int
			remainder float64
		}
		shares := make([]weightedShare, 0, len(groupItems))
		allocated := 0
		for index, item := range groupItems {
			exact := float64(total) * item.weight / totalWeight
			quantity := int(math.Floor(exact))
			allocated += quantity
			shares = append(shares, weightedShare{index: index, quantity: quantity, remainder: exact - float64(quantity)})
		}
		sort.SliceStable(shares, func(i, j int) bool { return shares[i].remainder > shares[j].remainder })
		for index := 0; index < total-allocated; index++ {
			shares[index%len(shares)].quantity++
		}
		for _, share := range shares {
			item := groupItems[share.index]
			key := fmt.Sprintf("menu-result:%d", item.id)
			values[key] = aggregate{quantity: float64(share.quantity), origin: fmt.Sprintf("Bebida selecionada: distribuição %s (peso %.2g)", group, item.weight)}
			inventoryByKey[key] = item.inventoryID
		}
	}
	rows, err = s.db.QueryContext(ctx, `SELECT e.inventory_item_id,MAX(me.quantity),GROUP_CONCAT(DISTINCT COALESCE(NULLIF(m.display_name,''),m.name)) FROM event_menu_items em JOIN menu_items m ON m.id=em.menu_item_id JOIN menu_item_equipment me ON me.menu_item_id=m.id JOIN equipment e ON e.id=me.equipment_id WHERE em.event_id=? AND me.required=1 AND NOT EXISTS(SELECT 1 FROM event_menu_item_equipment override JOIN event_menu_snapshot_items snapshot_item ON snapshot_item.id=override.event_menu_snapshot_item_id JOIN event_menu_sections section ON section.id=snapshot_item.event_menu_section_id JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id WHERE snapshot.event_id=em.event_id AND snapshot_item.source_menu_item_id=m.id) GROUP BY e.inventory_item_id`, eventID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var inventoryID int64
		var quantity float64
		var dishes string
		if err := rows.Scan(&inventoryID, &quantity, &dishes); err != nil {
			rows.Close()
			return nil, err
		}
		key := fmt.Sprintf("menu-equipment:%d", inventoryID)
		values[key] = aggregate{quantity: quantity, origin: "Equipamento obrigatório: " + dishes}
		inventoryByKey[key] = inventoryID
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	rows, err = s.db.QueryContext(ctx, `SELECT override.inventory_item_id,MAX(override.quantity),GROUP_CONCAT(DISTINCT snapshot_item.display_name) FROM event_menu_item_equipment override JOIN event_menu_snapshot_items snapshot_item ON snapshot_item.id=override.event_menu_snapshot_item_id JOIN event_menu_sections section ON section.id=snapshot_item.event_menu_section_id JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id WHERE snapshot.event_id=? AND snapshot_item.selected=1 AND snapshot_item.was_removed=0 AND override.required=1 GROUP BY override.inventory_item_id`, eventID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var inventoryID int64
		var quantity float64
		var dishes string
		if err := rows.Scan(&inventoryID, &quantity, &dishes); err != nil {
			rows.Close()
			return nil, err
		}
		key := fmt.Sprintf("event-menu-equipment:%d", inventoryID)
		values[key] = aggregate{quantity: quantity, origin: "Equipamento definido no evento: " + dishes}
		inventoryByKey[key] = inventoryID
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	rows, err = s.db.QueryContext(ctx, `SELECT link.id,COALESCE(link.inventory_item_id,container.inventory_item_id),COALESCE(link.quantity,CASE WHEN link.purpose LIKE 'cake_%' THEN COALESCE(cake.cake_count,1) ELSE 1 END),snapshot_item.display_name,link.purpose FROM event_menu_item_containers link JOIN event_menu_snapshot_items snapshot_item ON snapshot_item.id=link.event_menu_snapshot_item_id JOIN event_menu_sections section ON section.id=snapshot_item.event_menu_section_id JOIN event_menu_templates snapshot ON snapshot.id=section.event_menu_template_id LEFT JOIN container_types container ON container.id=link.container_type_id LEFT JOIN event_cake_configurations cake ON cake.event_id=snapshot.event_id WHERE snapshot.event_id=? AND snapshot_item.selected=1 AND snapshot_item.was_removed=0 AND COALESCE(link.inventory_item_id,container.inventory_item_id) IS NOT NULL`, eventID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var linkID, inventoryID int64
		var quantity float64
		var dish, purpose string
		if err := rows.Scan(&linkID, &inventoryID, &quantity, &dish, &purpose); err != nil {
			rows.Close()
			return nil, err
		}
		key := fmt.Sprintf("event-menu-container:%d", linkID)
		values[key] = aggregate{quantity: quantity, origin: fmt.Sprintf("%s de %s configurado no evento", purpose, dish)}
		inventoryByKey[key] = inventoryID
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]models.AutomaticRequirement, 0, len(keys))
	for _, key := range keys {
		value := values[key]
		result = append(result, models.AutomaticRequirement{SourceKey: key, InventoryItemID: inventoryByKey[key], Quantity: value.quantity, Origin: value.origin})
	}
	return result, nil
}
