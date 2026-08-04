CREATE TABLE IF NOT EXISTS kitchen_cooks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL UNIQUE COLLATE NOCASE,
    name TEXT NOT NULL UNIQUE COLLATE NOCASE,
    active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

INSERT OR IGNORE INTO kitchen_cooks(slug,name,active,created_at,updated_at) VALUES
    ('cris','Cris',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
    ('suelem','Suelem',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
    ('geriane','Geriane',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);

ALTER TABLE events ADD COLUMN kitchen_cook_id INTEGER REFERENCES kitchen_cooks(id);
CREATE INDEX IF NOT EXISTS idx_events_kitchen_cook ON events(kitchen_cook_id, starts_at);

INSERT OR IGNORE INTO inventory_items(
    name,description,category_id,subcategory,unit,stock_quantity,minimum_stock,damaged_quantity,location_id,
    internal_code,item_kind,ownership,requires_return,replacement_value_cents,notes,active,created_at,updated_at
)
SELECT 'Caixa de utensílios — Cris','Caixa pessoal da Cris com facas, colheres, ralador e demais utensílios de cozinha.',id,'Caixas das cozinheiras','caixa',1,0,0,5,'COZ-CRIS-UTENSILIOS','reusable','owned',1,0,'Conferir o conteúdo completo na saída e no retorno.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Equipamentos de cozinha'
UNION ALL SELECT 'Caixa de temperos — Cris','Caixa pessoal da Cris com os temperos utilizados no preparo.',id,'Caixas das cozinheiras','caixa',1,0,0,5,'COZ-CRIS-TEMPEROS','reusable','owned',1,0,'Conferir o conteúdo completo na saída e no retorno.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Equipamentos de cozinha'
UNION ALL SELECT 'Caixa de utensílios — Suelem','Caixa pessoal da Suelem com facas, colheres, ralador e demais utensílios de cozinha.',id,'Caixas das cozinheiras','caixa',1,0,0,5,'COZ-SUELEM-UTENSILIOS','reusable','owned',1,0,'Conferir o conteúdo completo na saída e no retorno.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Equipamentos de cozinha'
UNION ALL SELECT 'Caixa de temperos — Suelem','Caixa pessoal da Suelem com os temperos utilizados no preparo.',id,'Caixas das cozinheiras','caixa',1,0,0,5,'COZ-SUELEM-TEMPEROS','reusable','owned',1,0,'Conferir o conteúdo completo na saída e no retorno.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Equipamentos de cozinha'
UNION ALL SELECT 'Caixa de utensílios — Geriane','Caixa pessoal da Geriane com facas, colheres, ralador e demais utensílios de cozinha.',id,'Caixas das cozinheiras','caixa',1,0,0,5,'COZ-GERIANE-UTENSILIOS','reusable','owned',1,0,'Conferir o conteúdo completo na saída e no retorno.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Equipamentos de cozinha'
UNION ALL SELECT 'Caixa de temperos — Geriane','Caixa pessoal da Geriane com os temperos utilizados no preparo.',id,'Caixas das cozinheiras','caixa',1,0,0,5,'COZ-GERIANE-TEMPEROS','reusable','owned',1,0,'Conferir o conteúdo completo na saída e no retorno.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Equipamentos de cozinha';

CREATE TABLE IF NOT EXISTS kitchen_cook_storage_boxes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kitchen_cook_id INTEGER NOT NULL REFERENCES kitchen_cooks(id) ON DELETE CASCADE,
    inventory_item_id INTEGER NOT NULL UNIQUE REFERENCES inventory_items(id),
    box_type TEXT NOT NULL CHECK(box_type IN ('utensils','spices')),
    active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(kitchen_cook_id, box_type)
);

INSERT OR IGNORE INTO kitchen_cook_storage_boxes(kitchen_cook_id,inventory_item_id,box_type,active,created_at,updated_at)
SELECT cook.id,item.id,'utensils',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
FROM kitchen_cooks cook JOIN inventory_items item ON item.internal_code='COZ-' || UPPER(cook.slug) || '-UTENSILIOS';

INSERT OR IGNORE INTO kitchen_cook_storage_boxes(kitchen_cook_id,inventory_item_id,box_type,active,created_at,updated_at)
SELECT cook.id,item.id,'spices',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
FROM kitchen_cooks cook JOIN inventory_items item ON item.internal_code='COZ-' || UPPER(cook.slug) || '-TEMPEROS';

CREATE INDEX IF NOT EXISTS idx_kitchen_cook_boxes_cook ON kitchen_cook_storage_boxes(kitchen_cook_id, active);
