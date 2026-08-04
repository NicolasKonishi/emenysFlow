ALTER TABLE events ADD COLUMN has_cake INTEGER NOT NULL DEFAULT 0 CHECK(has_cake IN (0,1));

-- Eventos antigos com sabor informado ou bolo selecionado mantêm a opção ativa.
UPDATE events
SET has_cake=1
WHERE TRIM(cake_notes)<>''
   OR EXISTS (
       SELECT 1
       FROM event_menu_templates snapshot
       JOIN event_menu_sections section ON section.event_menu_template_id=snapshot.id
       JOIN event_menu_snapshot_items item ON item.event_menu_section_id=section.id
       WHERE snapshot.event_id=events.id
         AND LOWER(section.name) LIKE '%bolo%'
         AND item.selected=1
         AND item.was_removed=0
   );

UPDATE event_cake_configurations
SET cake_count=0,requires_refrigeration=0,notes='',updated_at=CURRENT_TIMESTAMP
WHERE event_id IN (SELECT id FROM events WHERE has_cake=0);
