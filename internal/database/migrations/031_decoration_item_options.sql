-- Structured choices used during event-specific decoration planning.
ALTER TABLE event_decoration_composition_items ADD COLUMN arrangement_kind TEXT NOT NULL DEFAULT ''
    CHECK(arrangement_kind IN ('', 'permanent', 'natural'));
ALTER TABLE event_decoration_composition_items ADD COLUMN fake_cake_type TEXT NOT NULL DEFAULT '';
