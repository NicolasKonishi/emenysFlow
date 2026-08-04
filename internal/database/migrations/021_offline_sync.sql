-- Server-side idempotency and conflict records. The operational queue itself lives in IndexedDB.

CREATE TABLE IF NOT EXISTS sync_devices (
    device_id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_name TEXT NOT NULL DEFAULT '',
    last_sync_at TEXT,
    last_seen_at TEXT NOT NULL,
    revoked_at TEXT,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sync_operations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    client_operation_id TEXT NOT NULL UNIQUE,
    device_id TEXT NOT NULL REFERENCES sync_devices(device_id),
    user_id INTEGER NOT NULL REFERENCES users(id),
    operation_type TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id INTEGER,
    base_version INTEGER NOT NULL DEFAULT 0,
    payload_json TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL CHECK(status IN ('applied','failed','conflict')),
    result_json TEXT NOT NULL DEFAULT '{}',
    server_snapshot_json TEXT,
    last_error TEXT NOT NULL DEFAULT '',
    submitted_at TEXT NOT NULL,
    applied_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sync_operations_device ON sync_operations(device_id,created_at);
CREATE INDEX IF NOT EXISTS idx_sync_operations_entity ON sync_operations(entity_type,entity_id,created_at);

