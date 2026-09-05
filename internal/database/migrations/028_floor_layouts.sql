-- Floor plans for events and standalone layouts. The feature landed in
-- application code without a versioned schema, so listing /layouts failed
-- with a generic database error (no such table).

CREATE TABLE IF NOT EXISTS event_floor_layouts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id INTEGER NOT NULL UNIQUE REFERENCES events(id) ON DELETE CASCADE,
    layout_json TEXT NOT NULL,
    row_version INTEGER NOT NULL DEFAULT 1 CHECK(row_version > 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_event_floor_layouts_event ON event_floor_layouts(event_id);

CREATE TABLE IF NOT EXISTS standalone_floor_layouts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    venue TEXT NOT NULL DEFAULT '',
    guest_count INTEGER NOT NULL DEFAULT 0 CHECK(guest_count >= 0),
    waiter_count INTEGER NOT NULL DEFAULT 0 CHECK(waiter_count >= 0),
    waiter_names_json TEXT NOT NULL DEFAULT '[]',
    layout_json TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
    row_version INTEGER NOT NULL DEFAULT 1 CHECK(row_version > 0),
    created_by INTEGER REFERENCES users(id),
    updated_by INTEGER REFERENCES users(id),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_standalone_floor_layouts_active ON standalone_floor_layouts(active, updated_at);
