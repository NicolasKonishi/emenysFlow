ALTER TABLE menu_items ADD COLUMN display_name TEXT NOT NULL DEFAULT '';
ALTER TABLE menu_items ADD COLUMN template_owner_id INTEGER REFERENCES event_templates(id);
ALTER TABLE menu_items ADD COLUMN source_menu_item_id INTEGER REFERENCES menu_items(id);

CREATE INDEX IF NOT EXISTS idx_menu_items_template_owner
    ON menu_items(template_owner_id, active, category_id);

-- Cada vínculo antigo passa a apontar para uma cópia exclusiva do item.
INSERT INTO menu_items(
    category_id,name,display_name,description,container_type_id,container_capacity_portions,
    pan_inventory_item_id,transport_inventory_item_id,result_inventory_item_id,
    calculation_type,calculation_group,calculation_divisor,calculation_multiplier,
    calculation_weight,template_owner_id,source_menu_item_id,active,created_at,updated_at
)
SELECT
    source.category_id,
    '__template_' || etmi.template_id || '_source_' || source.id,
    COALESCE(NULLIF(source.display_name,''),source.name),
    source.description,source.container_type_id,source.container_capacity_portions,
    source.pan_inventory_item_id,source.transport_inventory_item_id,source.result_inventory_item_id,
    source.calculation_type,source.calculation_group,source.calculation_divisor,source.calculation_multiplier,
    source.calculation_weight,etmi.template_id,source.id,source.active,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
FROM event_template_menu_items etmi
JOIN menu_items source ON source.id=etmi.menu_item_id
WHERE source.template_owner_id IS NULL;

INSERT OR IGNORE INTO menu_item_equipment(menu_item_id,equipment_id,quantity,required)
SELECT clone.id,link.equipment_id,link.quantity,link.required
FROM menu_items clone
JOIN menu_item_equipment link ON link.menu_item_id=clone.source_menu_item_id
WHERE clone.template_owner_id IS NOT NULL;

-- Eventos já vinculados a um modelo também recebem as cópias exclusivas dele.
UPDATE event_menu_items
SET menu_item_id=(
    SELECT clone.id
    FROM events event
    JOIN menu_items clone ON clone.template_owner_id=event.template_id
    WHERE event.id=event_menu_items.event_id
      AND clone.source_menu_item_id=event_menu_items.menu_item_id
)
WHERE EXISTS(
    SELECT 1
    FROM events event
    JOIN menu_items clone ON clone.template_owner_id=event.template_id
    WHERE event.id=event_menu_items.event_id
      AND clone.source_menu_item_id=event_menu_items.menu_item_id
);

UPDATE event_template_menu_items
SET menu_item_id=(
    SELECT clone.id
    FROM menu_items clone
    WHERE clone.template_owner_id=event_template_menu_items.template_id
      AND clone.source_menu_item_id=event_template_menu_items.menu_item_id
)
WHERE EXISTS(
    SELECT 1
    FROM menu_items clone
    WHERE clone.template_owner_id=event_template_menu_items.template_id
      AND clone.source_menu_item_id=event_template_menu_items.menu_item_id
);
