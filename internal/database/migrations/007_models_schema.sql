-- Catálogo avançado de modelos de cardápio e serviços.
ALTER TABLE menu_items ADD COLUMN slug TEXT;
ALTER TABLE menu_items ADD COLUMN source_label TEXT NOT NULL DEFAULT '';
ALTER TABLE menu_items ADD COLUMN normalized_name TEXT NOT NULL DEFAULT '';
ALTER TABLE menu_items ADD COLUMN search_aliases TEXT NOT NULL DEFAULT '';
ALTER TABLE menu_items ADD COLUMN deleted_at TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_menu_items_slug
    ON menu_items(slug) WHERE slug IS NOT NULL AND slug <> '';

CREATE TABLE IF NOT EXISTS menu_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    menu_type TEXT NOT NULL DEFAULT 'buffet',
    image_url TEXT,
    active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
    current_version INTEGER NOT NULL DEFAULT 1 CHECK(current_version > 0),
    source_name TEXT NOT NULL,
    source_updated_month TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);

CREATE TABLE IF NOT EXISTS menu_template_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    menu_template_id INTEGER NOT NULL REFERENCES menu_templates(id),
    version INTEGER NOT NULL CHECK(version > 0),
    change_summary TEXT NOT NULL DEFAULT '',
    snapshot_json TEXT NOT NULL DEFAULT '{}',
    created_by INTEGER REFERENCES users(id),
    created_at TEXT NOT NULL,
    UNIQUE(menu_template_id, version)
);

CREATE TABLE IF NOT EXISTS menu_sections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    section_type TEXT NOT NULL DEFAULT 'food',
    active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);

CREATE TABLE IF NOT EXISTS menu_template_sections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    menu_template_id INTEGER NOT NULL REFERENCES menu_templates(id),
    menu_section_id INTEGER NOT NULL REFERENCES menu_sections(id),
    display_name TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    required INTEGER NOT NULL DEFAULT 0 CHECK(required IN (0,1)),
    selection_min INTEGER NOT NULL DEFAULT 0 CHECK(selection_min >= 0),
    selection_max INTEGER CHECK(selection_max IS NULL OR selection_max >= selection_min),
    allow_event_changes INTEGER NOT NULL DEFAULT 1 CHECK(allow_event_changes IN (0,1)),
    notes TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,
    UNIQUE(menu_template_id, menu_section_id)
);

CREATE TABLE IF NOT EXISTS menu_choice_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    menu_template_section_id INTEGER NOT NULL REFERENCES menu_template_sections(id) ON DELETE CASCADE,
    slug TEXT NOT NULL,
    choice_group_name TEXT NOT NULL,
    selection_min INTEGER NOT NULL DEFAULT 0 CHECK(selection_min >= 0),
    selection_max INTEGER CHECK(selection_max IS NULL OR selection_max >= selection_min),
    selection_required INTEGER NOT NULL DEFAULT 0 CHECK(selection_required IN (0,1)),
    allow_extra_items INTEGER NOT NULL DEFAULT 0 CHECK(allow_extra_items IN (0,1)),
    allow_custom_item INTEGER NOT NULL DEFAULT 0 CHECK(allow_custom_item IN (0,1)),
    configurable INTEGER NOT NULL DEFAULT 0 CHECK(configurable IN (0,1)),
    sort_order INTEGER NOT NULL DEFAULT 0,
    active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,
    UNIQUE(menu_template_section_id, slug)
);

CREATE TABLE IF NOT EXISTS menu_template_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    menu_template_section_id INTEGER NOT NULL REFERENCES menu_template_sections(id) ON DELETE CASCADE,
    menu_item_id INTEGER REFERENCES menu_items(id),
    slug TEXT NOT NULL,
    source_label TEXT NOT NULL,
    normalized_name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    included INTEGER NOT NULL DEFAULT 1 CHECK(included IN (0,1)),
    optional INTEGER NOT NULL DEFAULT 0 CHECK(optional IN (0,1)),
    configurable INTEGER NOT NULL DEFAULT 0 CHECK(configurable IN (0,1)),
    notes TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,
    UNIQUE(menu_template_section_id, slug)
);

CREATE TABLE IF NOT EXISTS menu_choice_group_items (
    menu_choice_group_id INTEGER NOT NULL REFERENCES menu_choice_groups(id) ON DELETE CASCADE,
    menu_template_item_id INTEGER NOT NULL REFERENCES menu_template_items(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    initially_selected INTEGER NOT NULL DEFAULT 0 CHECK(initially_selected IN (0,1)),
    created_at TEXT NOT NULL,
    PRIMARY KEY(menu_choice_group_id, menu_template_item_id)
);

CREATE TABLE IF NOT EXISTS menu_shared_blocks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);

CREATE TABLE IF NOT EXISTS menu_shared_block_items (
    menu_shared_block_id INTEGER NOT NULL REFERENCES menu_shared_blocks(id) ON DELETE CASCADE,
    menu_item_id INTEGER NOT NULL REFERENCES menu_items(id),
    sort_order INTEGER NOT NULL DEFAULT 0,
    included INTEGER NOT NULL DEFAULT 1 CHECK(included IN (0,1)),
    created_at TEXT NOT NULL,
    PRIMARY KEY(menu_shared_block_id, menu_item_id)
);

CREATE TABLE IF NOT EXISTS menu_template_shared_blocks (
    menu_template_id INTEGER NOT NULL REFERENCES menu_templates(id) ON DELETE CASCADE,
    menu_shared_block_id INTEGER NOT NULL REFERENCES menu_shared_blocks(id),
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    PRIMARY KEY(menu_template_id, menu_shared_block_id)
);

CREATE TABLE IF NOT EXISTS menu_template_item_inventory_links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    menu_template_item_id INTEGER NOT NULL REFERENCES menu_template_items(id) ON DELETE CASCADE,
    inventory_item_id INTEGER NOT NULL REFERENCES inventory_items(id),
    quantity_formula TEXT NOT NULL DEFAULT '',
    ownership TEXT NOT NULL DEFAULT 'owned' CHECK(ownership IN ('owned','rented','outsourced','consumable')),
    notes TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
    created_at TEXT NOT NULL,
    UNIQUE(menu_template_item_id, inventory_item_id)
);

CREATE TABLE IF NOT EXISTS event_menu_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    source_menu_template_id INTEGER REFERENCES menu_templates(id),
    source_version INTEGER NOT NULL,
    snapshot_name TEXT NOT NULL,
    snapshot_description TEXT NOT NULL DEFAULT '',
    snapshot_json TEXT NOT NULL DEFAULT '{}',
    applied_by INTEGER REFERENCES users(id),
    applied_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(event_id)
);

CREATE TABLE IF NOT EXISTS event_menu_sections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_menu_template_id INTEGER NOT NULL REFERENCES event_menu_templates(id) ON DELETE CASCADE,
    source_template_section_id INTEGER,
    name TEXT NOT NULL,
    section_type TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    selection_min INTEGER NOT NULL DEFAULT 0,
    selection_max INTEGER,
    allow_event_changes INTEGER NOT NULL DEFAULT 1 CHECK(allow_event_changes IN (0,1)),
    notes TEXT NOT NULL DEFAULT '',
    UNIQUE(event_menu_template_id, sort_order, name)
);

CREATE TABLE IF NOT EXISTS event_menu_snapshot_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_menu_section_id INTEGER NOT NULL REFERENCES event_menu_sections(id) ON DELETE CASCADE,
    source_template_item_id INTEGER,
    source_menu_item_id INTEGER REFERENCES menu_items(id),
    source_choice_group_id INTEGER,
    source_label TEXT NOT NULL,
    normalized_name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    selected INTEGER NOT NULL DEFAULT 1 CHECK(selected IN (0,1)),
    custom_item INTEGER NOT NULL DEFAULT 0 CHECK(custom_item IN (0,1)),
    portions REAL,
    container_type_id INTEGER REFERENCES container_types(id),
    notes TEXT NOT NULL DEFAULT '',
    changed_by INTEGER REFERENCES users(id),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS service_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL,
    duration_minutes INTEGER,
    billing_unit TEXT NOT NULL DEFAULT 'serviço',
    price_cents INTEGER,
    cost_cents INTEGER,
    commission_cents INTEGER,
    supplier TEXT,
    required_team_json TEXT NOT NULL DEFAULT '{}',
    included_materials TEXT NOT NULL DEFAULT '',
    excluded_materials TEXT NOT NULL DEFAULT '',
    image_url TEXT,
    notes TEXT NOT NULL DEFAULT '',
    terms TEXT NOT NULL DEFAULT '',
    configuration_json TEXT NOT NULL DEFAULT '{}',
    active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
    current_version INTEGER NOT NULL DEFAULT 1 CHECK(current_version > 0),
    source_name TEXT NOT NULL,
    source_updated_month TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);

CREATE TABLE IF NOT EXISTS service_template_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_template_id INTEGER NOT NULL REFERENCES service_templates(id),
    version INTEGER NOT NULL CHECK(version > 0),
    change_summary TEXT NOT NULL DEFAULT '',
    snapshot_json TEXT NOT NULL DEFAULT '{}',
    created_by INTEGER REFERENCES users(id),
    created_at TEXT NOT NULL,
    UNIQUE(service_template_id, version)
);

CREATE TABLE IF NOT EXISTS service_components (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL,
    source_label TEXT NOT NULL DEFAULT '',
    normalized_name TEXT NOT NULL,
    search_aliases TEXT NOT NULL DEFAULT '',
    configurable INTEGER NOT NULL DEFAULT 0 CHECK(configurable IN (0,1)),
    active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);

CREATE TABLE IF NOT EXISTS service_template_components (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_template_id INTEGER NOT NULL REFERENCES service_templates(id) ON DELETE CASCADE,
    service_component_id INTEGER NOT NULL REFERENCES service_components(id),
    sort_order INTEGER NOT NULL DEFAULT 0,
    included INTEGER NOT NULL DEFAULT 1 CHECK(included IN (0,1)),
    optional INTEGER NOT NULL DEFAULT 0 CHECK(optional IN (0,1)),
    configuration_json TEXT NOT NULL DEFAULT '{}',
    notes TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,
    UNIQUE(service_template_id, service_component_id)
);

CREATE TABLE IF NOT EXISTS service_choice_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_template_id INTEGER NOT NULL REFERENCES service_templates(id) ON DELETE CASCADE,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    selection_min INTEGER NOT NULL DEFAULT 0 CHECK(selection_min >= 0),
    selection_max INTEGER CHECK(selection_max IS NULL OR selection_max >= selection_min),
    selection_required INTEGER NOT NULL DEFAULT 0 CHECK(selection_required IN (0,1)),
    configurable INTEGER NOT NULL DEFAULT 0 CHECK(configurable IN (0,1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(service_template_id, slug)
);

CREATE TABLE IF NOT EXISTS service_choice_group_components (
    service_choice_group_id INTEGER NOT NULL REFERENCES service_choice_groups(id) ON DELETE CASCADE,
    service_template_component_id INTEGER NOT NULL REFERENCES service_template_components(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    PRIMARY KEY(service_choice_group_id, service_template_component_id)
);

CREATE TABLE IF NOT EXISTS service_component_inventory_links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_template_component_id INTEGER NOT NULL REFERENCES service_template_components(id) ON DELETE CASCADE,
    inventory_item_id INTEGER NOT NULL REFERENCES inventory_items(id),
    quantity_formula TEXT NOT NULL DEFAULT '',
    ownership TEXT NOT NULL DEFAULT 'owned' CHECK(ownership IN ('owned','rented','outsourced','consumable')),
    supplier TEXT,
    pickup_notes TEXT NOT NULL DEFAULT '',
    return_notes TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
    created_at TEXT NOT NULL,
    UNIQUE(service_template_component_id, inventory_item_id)
);

CREATE TABLE IF NOT EXISTS event_services (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    source_service_template_id INTEGER REFERENCES service_templates(id),
    source_version INTEGER NOT NULL,
    snapshot_name TEXT NOT NULL,
    snapshot_json TEXT NOT NULL DEFAULT '{}',
    duration_minutes INTEGER,
    supplier TEXT,
    status TEXT NOT NULL DEFAULT 'planned',
    applied_by INTEGER REFERENCES users(id),
    applied_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);

CREATE TABLE IF NOT EXISTS event_service_components (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_service_id INTEGER NOT NULL REFERENCES event_services(id) ON DELETE CASCADE,
    source_template_component_id INTEGER,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    selected INTEGER NOT NULL DEFAULT 1 CHECK(selected IN (0,1)),
    ownership TEXT NOT NULL DEFAULT 'owned' CHECK(ownership IN ('owned','rented','outsourced','consumable')),
    supplier TEXT,
    configuration_json TEXT NOT NULL DEFAULT '{}',
    changed_by INTEGER REFERENCES users(id),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_menu_templates_active ON menu_templates(active, deleted_at, name);
CREATE INDEX IF NOT EXISTS idx_menu_template_sections_template ON menu_template_sections(menu_template_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_menu_template_items_section ON menu_template_items(menu_template_section_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_service_templates_active ON service_templates(active, deleted_at, name);
CREATE INDEX IF NOT EXISTS idx_service_components_category ON service_components(category, active);
CREATE INDEX IF NOT EXISTS idx_event_services_event ON event_services(event_id, deleted_at);
