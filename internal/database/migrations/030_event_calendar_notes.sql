-- Event scheduling notes and the constrained event-creator role.
INSERT INTO roles(slug, name, description, sort_order) VALUES
    ('event_creator', 'Criador de eventos', 'Cria e edita informações, agenda e anotações dos eventos, sem acesso a estoque, catálogo ou configurações.', 3)
ON CONFLICT(slug) DO UPDATE SET name=excluded.name, description=excluded.description, sort_order=excluded.sort_order;

CREATE TABLE IF NOT EXISTS event_notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    category TEXT NOT NULL DEFAULT 'general' CHECK(category IN ('technical_visit','couple_request','cake','decoration','service','general')),
    title TEXT NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    visit_at TEXT,
    created_by INTEGER REFERENCES users(id),
    updated_by INTEGER REFERENCES users(id),
    row_version INTEGER NOT NULL DEFAULT 1 CHECK(row_version > 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_event_notes_event ON event_notes(event_id, visit_at DESC, created_at DESC);

CREATE TABLE IF NOT EXISTS event_note_photos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    client_upload_id TEXT UNIQUE,
    event_note_id INTEGER NOT NULL REFERENCES event_notes(id) ON DELETE CASCADE,
    storage_path TEXT NOT NULL,
    original_name TEXT NOT NULL,
    mime_type TEXT NOT NULL CHECK(mime_type IN ('image/jpeg','image/png','image/webp')),
    file_size INTEGER NOT NULL CHECK(file_size > 0),
    caption TEXT NOT NULL DEFAULT '',
    uploaded_by INTEGER REFERENCES users(id),
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_event_note_photos_note ON event_note_photos(event_note_id, created_at);
