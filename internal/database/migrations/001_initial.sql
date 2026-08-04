-- Core migration for BuffetFlow. Values are stored in the smallest practical
-- units and timestamps use RFC 3339 UTC strings.
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE COLLATE NOCASE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin', 'employee')),
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    token_hash TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS inventory_categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE COLLATE NOCASE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS inventory_locations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE COLLATE NOCASE,
    description TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS inventory_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category_id INTEGER NOT NULL REFERENCES inventory_categories(id),
    subcategory TEXT NOT NULL DEFAULT '',
    unit TEXT NOT NULL DEFAULT 'unidade',
    stock_quantity REAL NOT NULL DEFAULT 0 CHECK (stock_quantity >= 0),
    minimum_stock REAL NOT NULL DEFAULT 0 CHECK (minimum_stock >= 0),
    damaged_quantity REAL NOT NULL DEFAULT 0 CHECK (damaged_quantity >= 0),
    location_id INTEGER REFERENCES inventory_locations(id),
    internal_code TEXT NOT NULL UNIQUE,
    barcode TEXT,
    photo_url TEXT,
    item_kind TEXT NOT NULL DEFAULT 'reusable' CHECK (item_kind IN ('reusable', 'consumable', 'rented', 'outsourced')),
    ownership TEXT NOT NULL DEFAULT 'owned' CHECK (ownership IN ('owned', 'rented', 'outsourced')),
    requires_return INTEGER NOT NULL DEFAULT 1 CHECK (requires_return IN (0, 1)),
    replacement_value_cents INTEGER NOT NULL DEFAULT 0 CHECK (replacement_value_cents >= 0),
    notes TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_inventory_items_category ON inventory_items(category_id, active);
CREATE INDEX IF NOT EXISTS idx_inventory_items_location ON inventory_items(location_id);
CREATE INDEX IF NOT EXISTS idx_inventory_items_name ON inventory_items(name COLLATE NOCASE);

CREATE TABLE IF NOT EXISTS event_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE COLLATE NOCASE,
    description TEXT NOT NULL DEFAULT '',
    configuration_json TEXT NOT NULL DEFAULT '{}',
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    template_id INTEGER REFERENCES event_templates(id),
    client_name TEXT NOT NULL,
    name TEXT NOT NULL,
    venue TEXT NOT NULL,
    starts_at TEXT NOT NULL,
    ends_at TEXT NOT NULL,
    guest_count INTEGER NOT NULL CHECK (guest_count > 0),
    has_decoration INTEGER NOT NULL DEFAULT 0 CHECK (has_decoration IN (0, 1)),
    has_welcome_drinks INTEGER NOT NULL DEFAULT 0 CHECK (has_welcome_drinks IN (0, 1)),
    has_coffee_table INTEGER NOT NULL DEFAULT 0 CHECK (has_coffee_table IN (0, 1)),
    starters_notes TEXT NOT NULL DEFAULT '',
    main_courses_notes TEXT NOT NULL DEFAULT '',
    sides_notes TEXT NOT NULL DEFAULT '',
    beverages_notes TEXT NOT NULL DEFAULT '',
    coffee_table_notes TEXT NOT NULL DEFAULT '',
    cake_notes TEXT NOT NULL DEFAULT '',
    sweets_notes TEXT NOT NULL DEFAULT '',
    desserts_notes TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    safety_margin_percent REAL NOT NULL DEFAULT 0 CHECK (safety_margin_percent >= 0),
    waiter_override INTEGER CHECK (waiter_override IS NULL OR waiter_override >= 0),
    status TEXT NOT NULL DEFAULT 'planning' CHECK (status IN ('planning','reserved','separating','checking','loading','in_progress','returning','post_event_check','completed','cancelled')),
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_by INTEGER REFERENCES users(id),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_events_starts ON events(starts_at, status);
CREATE INDEX IF NOT EXISTS idx_events_status ON events(status, active);

CREATE TABLE IF NOT EXISTS event_status_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    previous_status TEXT,
    new_status TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    changed_by INTEGER REFERENCES users(id),
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_event_history_event ON event_status_history(event_id, created_at);

CREATE TABLE IF NOT EXISTS menu_categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE COLLATE NOCASE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS container_types (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE COLLATE NOCASE,
    capacity_portions REAL,
    disposable INTEGER NOT NULL DEFAULT 0 CHECK (disposable IN (0, 1)),
    requires_lid INTEGER NOT NULL DEFAULT 0 CHECK (requires_lid IN (0, 1)),
    is_default INTEGER NOT NULL DEFAULT 0 CHECK (is_default IN (0, 1)),
    transport_notes TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS menu_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category_id INTEGER NOT NULL REFERENCES menu_categories(id),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    container_type_id INTEGER REFERENCES container_types(id),
    container_capacity_portions REAL,
    pan_inventory_item_id INTEGER REFERENCES inventory_items(id),
    transport_inventory_item_id INTEGER REFERENCES inventory_items(id),
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(category_id, name)
);

CREATE TABLE IF NOT EXISTS event_menu_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    menu_item_id INTEGER NOT NULL REFERENCES menu_items(id),
    portions INTEGER,
    container_type_id INTEGER REFERENCES container_types(id),
    calculated_container_quantity REAL,
    overridden_container_quantity REAL,
    notes TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(event_id, menu_item_id)
);

CREATE TABLE IF NOT EXISTS equipment (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    inventory_item_id INTEGER NOT NULL UNIQUE REFERENCES inventory_items(id),
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS menu_item_equipment (
    menu_item_id INTEGER NOT NULL REFERENCES menu_items(id) ON DELETE CASCADE,
    equipment_id INTEGER NOT NULL REFERENCES equipment(id),
    quantity REAL NOT NULL DEFAULT 1 CHECK (quantity > 0),
    required INTEGER NOT NULL DEFAULT 1 CHECK (required IN (0, 1)),
    PRIMARY KEY(menu_item_id, equipment_id)
);

CREATE TABLE IF NOT EXISTS calculation_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category_id INTEGER NOT NULL REFERENCES inventory_categories(id),
    trigger_event TEXT NOT NULL DEFAULT 'checklist_generation',
    calculation_type TEXT NOT NULL CHECK (calculation_type IN ('fixed','per_person','group_of_people','per_waiter','per_menu_item','per_table','per_dessert','per_starter','per_dish','per_equipment','percentage_distribution','custom')),
    base_value REAL NOT NULL DEFAULT 0,
    divisor REAL NOT NULL DEFAULT 1 CHECK (divisor > 0),
    multiplier REAL NOT NULL DEFAULT 1,
    minimum_quantity REAL,
    maximum_quantity REAL,
    safety_percent REAL NOT NULL DEFAULT 0,
    condition_json TEXT NOT NULL DEFAULT '{}',
    result_inventory_item_id INTEGER NOT NULL REFERENCES inventory_items(id),
    priority INTEGER NOT NULL DEFAULT 100,
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_rules_active_priority ON calculation_rules(active, priority);

CREATE TABLE IF NOT EXISTS checklists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id INTEGER NOT NULL UNIQUE REFERENCES events(id) ON DELETE CASCADE,
    version INTEGER NOT NULL DEFAULT 1,
    generated_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS checklist_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    checklist_id INTEGER NOT NULL REFERENCES checklists(id) ON DELETE CASCADE,
    inventory_item_id INTEGER REFERENCES inventory_items(id),
    category_id INTEGER NOT NULL REFERENCES inventory_categories(id),
    source_rule_id INTEGER REFERENCES calculation_rules(id),
    source_key TEXT NOT NULL,
    name TEXT NOT NULL,
    unit TEXT NOT NULL,
    calculated_quantity REAL NOT NULL DEFAULT 0,
    required_quantity REAL NOT NULL DEFAULT 0,
    available_quantity REAL NOT NULL DEFAULT 0,
    reserved_elsewhere_quantity REAL NOT NULL DEFAULT 0,
    missing_quantity REAL NOT NULL DEFAULT 0,
    calculation_origin TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','separating','separated','checked','loaded','at_event','returned','damaged','lost','not_applicable')),
    item_kind TEXT NOT NULL DEFAULT 'reusable' CHECK (item_kind IN ('reusable','consumable','rented','outsourced')),
    location_snapshot TEXT NOT NULL DEFAULT '',
    manual_item INTEGER NOT NULL DEFAULT 0 CHECK (manual_item IN (0, 1)),
    manual_override INTEGER NOT NULL DEFAULT 0 CHECK (manual_override IN (0, 1)),
    override_reason TEXT NOT NULL DEFAULT '',
    override_by INTEGER REFERENCES users(id),
    override_at TEXT,
    separated_quantity REAL NOT NULL DEFAULT 0,
    separated_by INTEGER REFERENCES users(id),
    separated_at TEXT,
    loaded_quantity REAL NOT NULL DEFAULT 0,
    returned_quantity REAL NOT NULL DEFAULT 0,
    damaged_quantity REAL NOT NULL DEFAULT 0,
    lost_quantity REAL NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(checklist_id, source_key)
);
CREATE INDEX IF NOT EXISTS idx_checklist_items_status ON checklist_items(checklist_id, status);
CREATE INDEX IF NOT EXISTS idx_checklist_items_inventory ON checklist_items(inventory_item_id);

CREATE TABLE IF NOT EXISTS inventory_reservations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    inventory_item_id INTEGER NOT NULL REFERENCES inventory_items(id),
    checklist_item_id INTEGER REFERENCES checklist_items(id) ON DELETE SET NULL,
    quantity REAL NOT NULL CHECK (quantity >= 0),
    starts_at TEXT NOT NULL,
    release_expected_at TEXT NOT NULL,
    released_at TEXT,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','released','cancelled')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_reservations_window ON inventory_reservations(inventory_item_id, starts_at, release_expected_at, status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_reservations_active_unique ON inventory_reservations(event_id, inventory_item_id) WHERE status = 'active';

CREATE TABLE IF NOT EXISTS inventory_movements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    inventory_item_id INTEGER NOT NULL REFERENCES inventory_items(id),
    event_id INTEGER REFERENCES events(id),
    movement_type TEXT NOT NULL CHECK (movement_type IN ('in','out','adjustment','reservation','return','damage','loss','maintenance','laundry')),
    quantity REAL NOT NULL,
    previous_stock REAL NOT NULL,
    new_stock REAL NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    performed_by INTEGER REFERENCES users(id),
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_movements_item ON inventory_movements(inventory_item_id, created_at);

CREATE TABLE IF NOT EXISTS decorations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    inventory_item_id INTEGER REFERENCES inventory_items(id),
    name TEXT NOT NULL,
    usage_location TEXT NOT NULL DEFAULT '',
    color TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    ownership TEXT NOT NULL DEFAULT 'owned' CHECK (ownership IN ('owned','rented')),
    rental_company TEXT NOT NULL DEFAULT '',
    photo_url TEXT,
    notes TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS event_decorations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    decoration_id INTEGER NOT NULL REFERENCES decorations(id),
    quantity REAL NOT NULL DEFAULT 1,
    pickup_at TEXT,
    return_at TEXT,
    notes TEXT NOT NULL DEFAULT '',
    UNIQUE(event_id, decoration_id)
);

CREATE TABLE IF NOT EXISTS beverages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    inventory_item_id INTEGER REFERENCES inventory_items(id),
    name TEXT NOT NULL UNIQUE COLLATE NOCASE,
    beverage_type TEXT NOT NULL,
    package_size_ml INTEGER,
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS event_beverages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    beverage_id INTEGER NOT NULL REFERENCES beverages(id),
    percentage REAL,
    calculated_quantity REAL,
    overridden_quantity REAL,
    UNIQUE(event_id, beverage_id)
);

CREATE TABLE IF NOT EXISTS staff_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    guests_per_staff REAL NOT NULL,
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS rental_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    inventory_item_id INTEGER REFERENCES inventory_items(id),
    supplier TEXT NOT NULL,
    quantity REAL NOT NULL,
    pickup_at TEXT,
    return_at TEXT,
    returned_at TEXT,
    notes TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS return_inspections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    checklist_item_id INTEGER NOT NULL REFERENCES checklist_items(id),
    sent_quantity REAL NOT NULL DEFAULT 0,
    returned_quantity REAL NOT NULL DEFAULT 0,
    damaged_quantity REAL NOT NULL DEFAULT 0,
    lost_quantity REAL NOT NULL DEFAULT 0,
    laundry_quantity REAL NOT NULL DEFAULT 0,
    maintenance_quantity REAL NOT NULL DEFAULT 0,
    notes TEXT NOT NULL DEFAULT '',
    inspected_by INTEGER REFERENCES users(id),
    inspected_at TEXT NOT NULL,
    UNIQUE(event_id, checklist_item_id)
);

CREATE TABLE IF NOT EXISTS audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id),
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    action TEXT NOT NULL,
    before_json TEXT,
    after_json TEXT,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_entity ON audit_log(entity_type, entity_id, created_at);
