-- Cada cozinheira passa a ter uma única caixa física. O conteúdo das antigas
-- caixas de temperos é incorporado à caixa principal sem descartar ajustes.
INSERT INTO kitchen_cook_box_items(
    kitchen_cook_storage_box_id,inventory_item_id,quantity,notes,active,created_at,updated_at
)
SELECT destination.id,content.inventory_item_id,content.quantity,content.notes,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
FROM kitchen_cook_storage_boxes source
JOIN kitchen_cook_storage_boxes destination
  ON destination.kitchen_cook_id=source.kitchen_cook_id
 AND destination.box_type='utensils'
JOIN kitchen_cook_box_items content
  ON content.kitchen_cook_storage_box_id=source.id
WHERE source.box_type='spices' AND source.active=1 AND content.active=1
ON CONFLICT(kitchen_cook_storage_box_id,inventory_item_id) DO UPDATE SET
    quantity=CASE
        WHEN kitchen_cook_box_items.active=1 THEN kitchen_cook_box_items.quantity+excluded.quantity
        ELSE excluded.quantity
    END,
    notes=CASE
        WHEN kitchen_cook_box_items.notes='' THEN excluded.notes
        WHEN excluded.notes='' THEN kitchen_cook_box_items.notes
        ELSE kitchen_cook_box_items.notes || ' / ' || excluded.notes
    END,
    active=1,
    updated_at=CURRENT_TIMESTAMP;

UPDATE kitchen_cook_box_items
SET active=0,updated_at=CURRENT_TIMESTAMP
WHERE kitchen_cook_storage_box_id IN (
    SELECT id FROM kitchen_cook_storage_boxes WHERE box_type='spices'
);

UPDATE inventory_items
SET name='Caixa da cozinheira — ' || (
        SELECT cook.name
        FROM kitchen_cook_storage_boxes box
        JOIN kitchen_cooks cook ON cook.id=box.kitchen_cook_id
        WHERE box.inventory_item_id=inventory_items.id
    ),
    description='Caixa pessoal com os utensílios e temperos utilizados pela cozinheira.',
    notes='Conferir todo o conteúdo da caixa na saída e no retorno.',
    updated_at=CURRENT_TIMESTAMP
WHERE id IN (
    SELECT inventory_item_id FROM kitchen_cook_storage_boxes WHERE box_type='utensils'
);

UPDATE kitchen_cook_storage_boxes
SET active=0,updated_at=CURRENT_TIMESTAMP
WHERE box_type='spices';

UPDATE inventory_items
SET active=0,updated_at=CURRENT_TIMESTAMP
WHERE id IN (
    SELECT inventory_item_id FROM kitchen_cook_storage_boxes WHERE box_type='spices'
);
