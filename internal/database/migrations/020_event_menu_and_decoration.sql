-- Event-level menu customization, logistics and complete decoration planning.

ALTER TABLE menu_categories ADD COLUMN slug TEXT;
UPDATE menu_categories SET slug=CASE name
    WHEN 'Entradas' THEN 'starters'
    WHEN 'Pratos principais' THEN 'main_courses'
    WHEN 'Acompanhamentos' THEN 'sides'
    WHEN 'Mesa de café' THEN 'coffee_table'
    WHEN 'Sobremesas' THEN 'desserts'
    ELSE 'category-' || id END
WHERE slug IS NULL OR slug='';
CREATE UNIQUE INDEX IF NOT EXISTS idx_menu_categories_slug ON menu_categories(slug);

ALTER TABLE container_types ADD COLUMN quantity_mode TEXT NOT NULL DEFAULT 'per_event_type'
    CHECK(quantity_mode IN ('per_event_type','per_menu_item','fixed','manual','per_serving'));
ALTER TABLE container_types ADD COLUMN required_utensil_type TEXT NOT NULL DEFAULT 'none'
    CHECK(required_utensil_type IN ('spoon','fork','knife','none','custom'));
ALTER TABLE container_types ADD COLUMN custom_utensil_name TEXT NOT NULL DEFAULT '';
ALTER TABLE container_types ADD COLUMN fixed_quantity REAL CHECK(fixed_quantity IS NULL OR fixed_quantity >= 0);

UPDATE container_types SET quantity_mode='per_menu_item' WHERE name IN ('Cuba GN 1/1','Pote plástico');
UPDATE container_types SET quantity_mode='per_event_type' WHERE name IN ('Bowl','Copo de sobremesa');

CREATE TABLE IF NOT EXISTS container_type_menu_categories (
    container_type_id INTEGER NOT NULL REFERENCES container_types(id) ON DELETE CASCADE,
    menu_category_id INTEGER NOT NULL REFERENCES menu_categories(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    PRIMARY KEY(container_type_id,menu_category_id)
);

INSERT OR IGNORE INTO container_type_menu_categories(container_type_id,menu_category_id,created_at)
SELECT container.id,category.id,CURRENT_TIMESTAMP
FROM container_types container CROSS JOIN menu_categories category
WHERE category.slug IN ('starters','desserts') AND container.name<>'Cuba GN 1/1';

ALTER TABLE event_menu_templates ADD COLUMN row_version INTEGER NOT NULL DEFAULT 1 CHECK(row_version > 0);
ALTER TABLE event_menu_templates ADD COLUMN updated_by INTEGER REFERENCES users(id);
ALTER TABLE event_menu_snapshot_items ADD COLUMN display_name TEXT NOT NULL DEFAULT '';
ALTER TABLE event_menu_snapshot_items ADD COLUMN is_customized INTEGER NOT NULL DEFAULT 0 CHECK(is_customized IN (0,1));
ALTER TABLE event_menu_snapshot_items ADD COLUMN was_removed INTEGER NOT NULL DEFAULT 0 CHECK(was_removed IN (0,1));
ALTER TABLE event_menu_snapshot_items ADD COLUMN original_snapshot_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE event_menu_snapshot_items ADD COLUMN customized_data_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE event_menu_snapshot_items ADD COLUMN row_version INTEGER NOT NULL DEFAULT 1 CHECK(row_version > 0);

UPDATE event_menu_snapshot_items SET display_name=source_label WHERE display_name='';

CREATE TABLE IF NOT EXISTS event_menu_item_containers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_menu_snapshot_item_id INTEGER NOT NULL REFERENCES event_menu_snapshot_items(id) ON DELETE CASCADE,
    purpose TEXT NOT NULL CHECK(purpose IN ('service','transport','cake_box','cake_base','cake_tray','cake_support')),
    container_type_id INTEGER REFERENCES container_types(id),
    inventory_item_id INTEGER REFERENCES inventory_items(id),
    quantity REAL CHECK(quantity IS NULL OR quantity >= 0),
    capacity_portions REAL CHECK(capacity_portions IS NULL OR capacity_portions > 0),
    requires_lid INTEGER NOT NULL DEFAULT 0 CHECK(requires_lid IN (0,1)),
    notes TEXT NOT NULL DEFAULT '',
    customized INTEGER NOT NULL DEFAULT 0 CHECK(customized IN (0,1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(event_menu_snapshot_item_id,purpose)
);

CREATE TABLE IF NOT EXISTS event_menu_item_equipment (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_menu_snapshot_item_id INTEGER NOT NULL REFERENCES event_menu_snapshot_items(id) ON DELETE CASCADE,
    inventory_item_id INTEGER NOT NULL REFERENCES inventory_items(id),
    quantity REAL NOT NULL DEFAULT 1 CHECK(quantity > 0),
    required INTEGER NOT NULL DEFAULT 1 CHECK(required IN (0,1)),
    customized INTEGER NOT NULL DEFAULT 0 CHECK(customized IN (0,1)),
    notes TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(event_menu_snapshot_item_id,inventory_item_id)
);

CREATE TABLE IF NOT EXISTS event_cake_configurations (
    event_id INTEGER PRIMARY KEY REFERENCES events(id) ON DELETE CASCADE,
    cake_count INTEGER NOT NULL DEFAULT 1 CHECK(cake_count >= 0),
    requires_refrigeration INTEGER NOT NULL DEFAULT 0 CHECK(requires_refrigeration IN (0,1)),
    notes TEXT NOT NULL DEFAULT '',
    row_version INTEGER NOT NULL DEFAULT 1 CHECK(row_version > 0),
    updated_by INTEGER REFERENCES users(id),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS event_menu_change_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    snapshot_item_id INTEGER REFERENCES event_menu_snapshot_items(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    before_json TEXT,
    after_json TEXT,
    changed_by INTEGER REFERENCES users(id),
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS event_decoration_profiles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id INTEGER NOT NULL UNIQUE REFERENCES events(id) ON DELETE CASCADE,
    style TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    primary_colors TEXT NOT NULL DEFAULT '',
    theme TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    responsible_user_id INTEGER REFERENCES users(id),
    responsible_name TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
    row_version INTEGER NOT NULL DEFAULT 1 CHECK(row_version > 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS event_decoration_compositions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    profile_id INTEGER NOT NULL REFERENCES event_decoration_profiles(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    composition_type TEXT NOT NULL DEFAULT 'other',
    description TEXT NOT NULL DEFAULT '',
    assembly_location TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    row_version INTEGER NOT NULL DEFAULT 1 CHECK(row_version > 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS event_decoration_composition_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    composition_id INTEGER NOT NULL REFERENCES event_decoration_compositions(id) ON DELETE CASCADE,
    decoration_id INTEGER REFERENCES decorations(id),
    inventory_item_id INTEGER REFERENCES inventory_items(id),
    custom_name TEXT NOT NULL DEFAULT '',
    quantity REAL NOT NULL DEFAULT 1 CHECK(quantity > 0),
    origin TEXT NOT NULL DEFAULT 'owned' CHECK(origin IN ('owned','rented','outsourced','client','produced')),
    supplier_id INTEGER REFERENCES suppliers(id),
    supplier_name TEXT NOT NULL DEFAULT '',
    estimated_cost_cents INTEGER CHECK(estimated_cost_cents IS NULL OR estimated_cost_cents >= 0),
    pickup_at TEXT,
    return_at TEXT,
    order_reference TEXT NOT NULL DEFAULT '',
    rental_status TEXT CHECK(rental_status IS NULL OR rental_status IN ('quote','awaiting_confirmation','confirmed','picked_up','delivered','returned','cancelled')),
    notes TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    row_version INTEGER NOT NULL DEFAULT 1 CHECK(row_version > 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CHECK(decoration_id IS NOT NULL OR inventory_item_id IS NOT NULL OR custom_name<>'')
);

CREATE TABLE IF NOT EXISTS event_reference_photos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    client_upload_id TEXT UNIQUE,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    composition_id INTEGER REFERENCES event_decoration_compositions(id) ON DELETE CASCADE,
    composition_item_id INTEGER REFERENCES event_decoration_composition_items(id) ON DELETE CASCADE,
    storage_path TEXT NOT NULL,
    original_name TEXT NOT NULL,
    mime_type TEXT NOT NULL CHECK(mime_type IN ('image/jpeg','image/png','image/webp')),
    file_size INTEGER NOT NULL CHECK(file_size > 0),
    caption TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_primary INTEGER NOT NULL DEFAULT 0 CHECK(is_primary IN (0,1)),
    uploaded_by INTEGER REFERENCES users(id),
    created_at TEXT NOT NULL,
    deleted_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_event_reference_photos_event ON event_reference_photos(event_id,deleted_at,sort_order);

-- Preserve legacy decoration selections inside a default composition.
INSERT OR IGNORE INTO event_decoration_profiles(event_id,style,description,active,created_at,updated_at)
SELECT DISTINCT event_id,'','Decoração migrada do cadastro anterior.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
FROM event_decorations;

INSERT INTO event_decoration_compositions(profile_id,name,composition_type,description,sort_order,created_at,updated_at)
SELECT profile.id,'Composição original','other','Itens preservados do cadastro anterior.',0,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
FROM event_decoration_profiles profile
WHERE EXISTS(SELECT 1 FROM event_decorations legacy WHERE legacy.event_id=profile.event_id)
  AND NOT EXISTS(SELECT 1 FROM event_decoration_compositions composition WHERE composition.profile_id=profile.id);

INSERT INTO event_decoration_composition_items(composition_id,decoration_id,inventory_item_id,custom_name,quantity,origin,supplier_name,notes,sort_order,created_at,updated_at)
SELECT composition.id,decoration.id,decoration.inventory_item_id,'',legacy.quantity,
       CASE decoration.ownership WHEN 'rented' THEN 'rented' ELSE 'owned' END,
       decoration.rental_company,legacy.notes,decoration.id,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
FROM event_decorations legacy
JOIN event_decoration_profiles profile ON profile.event_id=legacy.event_id
JOIN event_decoration_compositions composition ON composition.profile_id=profile.id AND composition.name='Composição original'
JOIN decorations decoration ON decoration.id=legacy.decoration_id
WHERE NOT EXISTS(
    SELECT 1 FROM event_decoration_composition_items item
    WHERE item.composition_id=composition.id AND item.decoration_id=decoration.id
);
