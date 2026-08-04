CREATE TABLE IF NOT EXISTS event_service_component_inventory_links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_service_component_id INTEGER NOT NULL REFERENCES event_service_components(id) ON DELETE CASCADE,
    source_inventory_link_id INTEGER,
    inventory_item_id INTEGER NOT NULL REFERENCES inventory_items(id),
    quantity_formula TEXT NOT NULL DEFAULT '',
    ownership TEXT NOT NULL DEFAULT 'owned' CHECK(ownership IN ('owned','rented','outsourced','consumable')),
    supplier TEXT,
    pickup_notes TEXT NOT NULL DEFAULT '',
    return_notes TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
    created_at TEXT NOT NULL,
    UNIQUE(event_service_component_id, inventory_item_id)
);

CREATE INDEX IF NOT EXISTS idx_event_service_inventory_component
    ON event_service_component_inventory_links(event_service_component_id, active);

-- Preserve the operational requirements of snapshots created before this migration.
INSERT OR IGNORE INTO event_service_component_inventory_links(
    event_service_component_id,
    source_inventory_link_id,
    inventory_item_id,
    quantity_formula,
    ownership,
    supplier,
    pickup_notes,
    return_notes,
    active,
    created_at
)
SELECT
    event_component.id,
    source_link.id,
    source_link.inventory_item_id,
    source_link.quantity_formula,
    source_link.ownership,
    source_link.supplier,
    source_link.pickup_notes,
    source_link.return_notes,
    source_link.active,
    COALESCE(event_component.created_at, CURRENT_TIMESTAMP)
FROM event_service_components event_component
JOIN service_component_inventory_links source_link
    ON source_link.service_template_component_id = event_component.source_template_component_id;
