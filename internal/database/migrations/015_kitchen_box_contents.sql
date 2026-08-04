CREATE TABLE IF NOT EXISTS kitchen_cook_box_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kitchen_cook_storage_box_id INTEGER NOT NULL REFERENCES kitchen_cook_storage_boxes(id) ON DELETE CASCADE,
    inventory_item_id INTEGER NOT NULL REFERENCES inventory_items(id),
    quantity REAL NOT NULL DEFAULT 1 CHECK(quantity > 0),
    notes TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(kitchen_cook_storage_box_id, inventory_item_id)
);

CREATE INDEX IF NOT EXISTS idx_kitchen_box_items_box
    ON kitchen_cook_box_items(kitchen_cook_storage_box_id, active);

INSERT OR IGNORE INTO inventory_items(
    name,description,category_id,subcategory,unit,stock_quantity,minimum_stock,damaged_quantity,location_id,
    internal_code,item_kind,ownership,requires_return,replacement_value_cents,notes,active,created_at,updated_at
)
SELECT 'Faca de cozinha','Faca utilizada pelas cozinheiras no preparo.',id,'Utensílios das cozinheiras','unidade',3,0,0,5,'COZ-UT-FACA','reusable','owned',1,0,'Armazenada dentro da caixa pessoal de utensílios.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Equipamentos de cozinha'
UNION ALL SELECT 'Ralador','Ralador utilizado no preparo dos alimentos.',id,'Utensílios das cozinheiras','unidade',3,0,0,5,'COZ-UT-RALADOR','reusable','owned',1,0,'Armazenado dentro da caixa pessoal de utensílios.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Equipamentos de cozinha'
UNION ALL SELECT 'Colher de pau','Colher de pau utilizada no preparo.',id,'Utensílios das cozinheiras','unidade',3,0,0,5,'COZ-UT-COLHER-PAU','reusable','owned',1,0,'Armazenada dentro da caixa pessoal de utensílios.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Equipamentos de cozinha'
UNION ALL SELECT 'Espátula de cozinha','Espátula utilizada no preparo.',id,'Utensílios das cozinheiras','unidade',3,0,0,5,'COZ-UT-ESPATULA','reusable','owned',1,0,'Armazenada dentro da caixa pessoal de utensílios.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Equipamentos de cozinha'
UNION ALL SELECT 'Sal','Sal da caixa de temperos das cozinheiras.',id,'Temperos das cozinheiras','pacote',3,0,0,NULL,'COZ-TMP-SAL','consumable','owned',0,0,'Armazenado dentro da caixa pessoal de temperos.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Comidas'
UNION ALL SELECT 'Pimenta-do-reino','Pimenta-do-reino da caixa de temperos.',id,'Temperos das cozinheiras','pacote',3,0,0,NULL,'COZ-TMP-PIMENTA','consumable','owned',0,0,'Armazenada dentro da caixa pessoal de temperos.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Comidas'
UNION ALL SELECT 'Alho','Alho da caixa de temperos das cozinheiras.',id,'Temperos das cozinheiras','pacote',3,0,0,NULL,'COZ-TMP-ALHO','consumable','owned',0,0,'Armazenado dentro da caixa pessoal de temperos.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Comidas'
UNION ALL SELECT 'Colorau','Colorau da caixa de temperos das cozinheiras.',id,'Temperos das cozinheiras','pacote',3,0,0,NULL,'COZ-TMP-COLORAU','consumable','owned',0,0,'Armazenado dentro da caixa pessoal de temperos.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Comidas'
UNION ALL SELECT 'Tempero completo','Tempero completo da caixa das cozinheiras.',id,'Temperos das cozinheiras','pacote',3,0,0,NULL,'COZ-TMP-COMPLETO','consumable','owned',0,0,'Armazenado dentro da caixa pessoal de temperos.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Comidas';

INSERT OR IGNORE INTO kitchen_cook_box_items(kitchen_cook_storage_box_id,inventory_item_id,quantity,notes,active,created_at,updated_at)
SELECT box.id,item.id,1,'Item padrão da caixa de utensílios.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
FROM kitchen_cook_storage_boxes box
JOIN inventory_items item ON item.internal_code IN ('UTN-001','UTN-002','COZ-UT-FACA','COZ-UT-RALADOR','COZ-UT-COLHER-PAU','COZ-UT-ESPATULA')
WHERE box.box_type='utensils';

INSERT OR IGNORE INTO kitchen_cook_box_items(kitchen_cook_storage_box_id,inventory_item_id,quantity,notes,active,created_at,updated_at)
SELECT box.id,item.id,1,'Item padrão da caixa de temperos.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
FROM kitchen_cook_storage_boxes box
JOIN inventory_items item ON item.internal_code IN ('COZ-TMP-SAL','COZ-TMP-PIMENTA','COZ-TMP-ALHO','COZ-TMP-COLORAU','COZ-TMP-COMPLETO')
WHERE box.box_type='spices';
