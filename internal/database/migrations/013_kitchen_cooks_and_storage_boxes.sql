-- Schema das cozinheiras. Dados reais da equipe ficam em internal/database/seeds/private/013_kitchen_cooks.sql
CREATE TABLE IF NOT EXISTS kitchen_cooks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL UNIQUE COLLATE NOCASE,
    name TEXT NOT NULL UNIQUE COLLATE NOCASE,
    active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

INSERT OR IGNORE INTO kitchen_cooks(slug,name,active,created_at,updated_at) VALUES
    ('cozinheira-a','Cozinheira A',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
    ('cozinheira-b','Cozinheira B',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);

ALTER TABLE events ADD COLUMN kitchen_cook_id INTEGER REFERENCES kitchen_cooks(id);
CREATE INDEX IF NOT EXISTS idx_events_kitchen_cook ON events(kitchen_cook_id, starts_at);

INSERT OR IGNORE INTO inventory_items(
    name,description,category_id,subcategory,unit,stock_quantity,minimum_stock,damaged_quantity,location_id,
    internal_code,item_kind,ownership,requires_return,replacement_value_cents,notes,active,created_at,updated_at
)
SELECT 'Caixa da cozinheira — A','Caixa pessoal de utensílios e temperos.',id,'Caixas das cozinheiras','caixa',1,0,0,1,'COZ-A-BOX','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Equipamentos de cozinha'
UNION ALL SELECT 'Caixa da cozinheira — B','Caixa pessoal de utensílios e temperos.',id,'Caixas das cozinheiras','caixa',1,0,0,1,'COZ-B-BOX','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Equipamentos de cozinha';

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
FROM kitchen_cooks cook JOIN inventory_items item ON item.internal_code='COZ-' || UPPER(SUBSTR(cook.slug, -1)) || '-BOX';

CREATE INDEX IF NOT EXISTS idx_kitchen_cook_boxes_cook ON kitchen_cook_storage_boxes(kitchen_cook_id, active);
