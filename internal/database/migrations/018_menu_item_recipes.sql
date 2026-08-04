CREATE TABLE IF NOT EXISTS menu_item_ingredients (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    menu_item_id INTEGER NOT NULL REFERENCES menu_items(id) ON DELETE CASCADE,
    inventory_item_id INTEGER NOT NULL REFERENCES inventory_items(id),
    calculation_type TEXT NOT NULL DEFAULT 'proportional' CHECK(calculation_type IN ('proportional','group_of_people','fixed')),
    quantity REAL NOT NULL DEFAULT 1 CHECK(quantity > 0),
    people_divisor REAL NOT NULL DEFAULT 1 CHECK(people_divisor > 0),
    notes TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(menu_item_id,inventory_item_id)
);

CREATE INDEX IF NOT EXISTS idx_menu_item_ingredients_menu
    ON menu_item_ingredients(menu_item_id,active);

INSERT OR IGNORE INTO inventory_items(
    name,description,category_id,subcategory,unit,stock_quantity,minimum_stock,damaged_quantity,location_id,
    internal_code,item_kind,ownership,requires_return,replacement_value_cents,notes,active,created_at,updated_at
)
SELECT 'Costela suína','Costela suína utilizada nas receitas de costela e costelinha.',id,'Ingredientes de receitas','kg',100,0,0,NULL,'ING-COSTELA','consumable','owned',0,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Comidas'
UNION ALL SELECT 'Frango','Frango utilizado no preparo dos pratos do cardápio.',id,'Ingredientes de receitas','kg',100,0,0,NULL,'ING-FRANGO','consumable','owned',0,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Comidas'
UNION ALL SELECT 'Molho barbecue','Molho barbecue levado sempre que houver costelinha.',id,'Ingredientes de receitas','frasco',30,0,0,NULL,'ING-BARBECUE','consumable','owned',0,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Comidas'
UNION ALL SELECT 'Ketchup','Ketchup levado sempre que houver estrogonofe.',id,'Ingredientes de receitas','frasco',30,0,0,NULL,'ING-KETCHUP','consumable','owned',0,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Comidas'
UNION ALL SELECT 'Mostarda','Mostarda levada sempre que houver estrogonofe.',id,'Ingredientes de receitas','frasco',30,0,0,NULL,'ING-MOSTARDA','consumable','owned',0,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Comidas';

-- 1 kg de costela para cada 3 porções de qualquer prato de costela/costelinha.
INSERT OR IGNORE INTO menu_item_ingredients(menu_item_id,inventory_item_id,calculation_type,quantity,people_divisor,notes,active,created_at,updated_at)
SELECT menu.id,ingredient.id,'proportional',1,3,'1 kg de costela para cada 3 pessoas.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
FROM menu_items menu JOIN inventory_items ingredient ON ingredient.internal_code='ING-COSTELA'
WHERE LOWER(COALESCE(NULLIF(menu.display_name,''),menu.name)) LIKE '%costel%';

-- Barbecue é uma necessidade fixa sempre que houver costelinha no cardápio.
INSERT OR IGNORE INTO menu_item_ingredients(menu_item_id,inventory_item_id,calculation_type,quantity,people_divisor,notes,active,created_at,updated_at)
SELECT menu.id,ingredient.id,'fixed',1,1,'Sempre levar quando houver costelinha.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
FROM menu_items menu JOIN inventory_items ingredient ON ingredient.internal_code='ING-BARBECUE'
WHERE LOWER(COALESCE(NULLIF(menu.display_name,''),menu.name)) LIKE '%costelinha%';

-- 1 kg de frango para cada 2 porções de pratos que usam frango.
INSERT OR IGNORE INTO menu_item_ingredients(menu_item_id,inventory_item_id,calculation_type,quantity,people_divisor,notes,active,created_at,updated_at)
SELECT menu.id,ingredient.id,'proportional',1,2,'1 kg de frango para cada 2 pessoas.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
FROM menu_items menu JOIN inventory_items ingredient ON ingredient.internal_code='ING-FRANGO'
WHERE LOWER(COALESCE(NULLIF(menu.display_name,''),menu.name)) LIKE '%frango%';

-- Ketchup e mostarda são necessidades fixas dos pratos de estrogonofe/strogonoff.
INSERT OR IGNORE INTO menu_item_ingredients(menu_item_id,inventory_item_id,calculation_type,quantity,people_divisor,notes,active,created_at,updated_at)
SELECT menu.id,ingredient.id,'fixed',1,1,'Sempre levar quando houver estrogonofe.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
FROM menu_items menu JOIN inventory_items ingredient ON ingredient.internal_code IN ('ING-KETCHUP','ING-MOSTARDA')
WHERE LOWER(COALESCE(NULLIF(menu.display_name,''),menu.name)) LIKE '%estrogonof%'
   OR LOWER(COALESCE(NULLIF(menu.display_name,''),menu.name)) LIKE '%strogonof%';
