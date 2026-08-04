-- Preserve equipment choices inside every existing event menu snapshot.
INSERT OR IGNORE INTO event_menu_item_equipment(event_menu_snapshot_item_id,inventory_item_id,quantity,required,customized,notes,created_at,updated_at)
SELECT snapshot_item.id,equipment.inventory_item_id,link.quantity,link.required,0,'',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
FROM event_menu_snapshot_items snapshot_item
JOIN menu_item_equipment link ON link.menu_item_id=snapshot_item.source_menu_item_id
JOIN equipment ON equipment.id=link.equipment_id
WHERE snapshot_item.source_menu_item_id IS NOT NULL;

INSERT OR IGNORE INTO event_cake_configurations(event_id,cake_count,requires_refrigeration,notes,created_at,updated_at)
SELECT DISTINCT snapshot.event_id,1,0,'',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
FROM event_menu_templates snapshot
JOIN event_menu_sections section ON section.event_menu_template_id=snapshot.id
WHERE LOWER(section.name) LIKE '%bolo%';
