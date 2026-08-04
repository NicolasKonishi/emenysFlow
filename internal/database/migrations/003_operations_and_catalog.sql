-- Completes the operational workflow and normalized catalog links.
ALTER TABLE container_types ADD COLUMN inventory_item_id INTEGER REFERENCES inventory_items(id);
ALTER TABLE checklist_items ADD COLUMN checked_by INTEGER REFERENCES users(id);
ALTER TABLE checklist_items ADD COLUMN checked_at TEXT;
ALTER TABLE checklist_items ADD COLUMN loaded_by INTEGER REFERENCES users(id);
ALTER TABLE checklist_items ADD COLUMN loaded_at TEXT;

CREATE TABLE IF NOT EXISTS event_operations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    stage TEXT NOT NULL CHECK (stage IN ('separating','checking','loading','in_progress','returning','post_event_check')),
    responsible_user_id INTEGER REFERENCES users(id),
    responsible_name TEXT NOT NULL DEFAULT '',
    vehicle TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    photo_url TEXT,
    occurred_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(event_id, stage)
);

CREATE TABLE IF NOT EXISTS event_share_tokens (
    token_hash TEXT PRIMARY KEY,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0,1)),
    expires_at TEXT,
    created_by INTEGER REFERENCES users(id),
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_event_share_event ON event_share_tokens(event_id, active);

CREATE TABLE IF NOT EXISTS decoration_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE COLLATE NOCASE,
    description TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0,1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS decoration_template_items (
    template_id INTEGER NOT NULL REFERENCES decoration_templates(id) ON DELETE CASCADE,
    decoration_id INTEGER NOT NULL REFERENCES decorations(id),
    quantity REAL NOT NULL DEFAULT 1 CHECK (quantity > 0),
    PRIMARY KEY(template_id, decoration_id)
);

INSERT OR IGNORE INTO inventory_items(name,description,category_id,subcategory,unit,stock_quantity,minimum_stock,damaged_quantity,location_id,internal_code,item_kind,ownership,requires_return,replacement_value_cents,notes,active,created_at,updated_at)
SELECT 'Bowl de sobremesa','Bowl reutilizável para serviço individual',5,'Sobremesa','unidade',240,80,0,1,'LOU-003','reusable','owned',1,1200,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP;
INSERT OR IGNORE INTO inventory_items(name,description,category_id,subcategory,unit,stock_quantity,minimum_stock,damaged_quantity,location_id,internal_code,item_kind,ownership,requires_return,replacement_value_cents,notes,active,created_at,updated_at)
SELECT 'Pote plástico com tampa','Recipiente para transporte de alimentos',2,'Potes','unidade',60,15,0,5,'REC-001','reusable','owned',1,2200,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP;
INSERT OR IGNORE INTO inventory_items(name,description,category_id,subcategory,unit,stock_quantity,minimum_stock,damaged_quantity,location_id,internal_code,item_kind,ownership,requires_return,replacement_value_cents,notes,active,created_at,updated_at)
SELECT 'Panela grande','Panela para produção',4,'Panelas','unidade',8,3,0,5,'EQP-002','reusable','owned',1,35000,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP;
INSERT OR IGNORE INTO inventory_items(name,description,category_id,subcategory,unit,stock_quantity,minimum_stock,damaged_quantity,location_id,internal_code,item_kind,ownership,requires_return,replacement_value_cents,notes,active,created_at,updated_at)
SELECT 'Colher de serviço','Utensílio de buffet',3,'Utensílios','unidade',35,12,0,5,'UTN-001','reusable','owned',1,2800,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP;
INSERT OR IGNORE INTO inventory_items(name,description,category_id,subcategory,unit,stock_quantity,minimum_stock,damaged_quantity,location_id,internal_code,item_kind,ownership,requires_return,replacement_value_cents,notes,active,created_at,updated_at)
SELECT 'Concha','Utensílio para caldos e molhos',3,'Utensílios','unidade',24,8,0,5,'UTN-002','reusable','owned',1,3000,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP;

UPDATE container_types SET inventory_item_id=(SELECT id FROM inventory_items WHERE internal_code='CUB-001') WHERE name='Cuba GN 1/1';
UPDATE container_types SET inventory_item_id=(SELECT id FROM inventory_items WHERE internal_code='LOU-003') WHERE name='Bowl';
UPDATE container_types SET inventory_item_id=(SELECT id FROM inventory_items WHERE internal_code='DES-001') WHERE name='Copo de sobremesa';
UPDATE container_types SET inventory_item_id=(SELECT id FROM inventory_items WHERE internal_code='REC-001') WHERE name='Pote plástico';

UPDATE menu_items SET pan_inventory_item_id=(SELECT id FROM inventory_items WHERE internal_code='EQP-002') WHERE category_id IN (2,3);
UPDATE menu_items SET transport_inventory_item_id=(SELECT id FROM inventory_items WHERE internal_code='REC-001') WHERE category_id=1;

INSERT OR IGNORE INTO equipment(inventory_item_id,active,created_at,updated_at)
SELECT id,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items WHERE internal_code IN ('EQP-001','EQP-002','UTN-001','UTN-002');

INSERT OR IGNORE INTO menu_item_equipment(menu_item_id,equipment_id,quantity,required)
SELECT m.id,e.id,1,1 FROM menu_items m CROSS JOIN equipment e
JOIN inventory_items i ON i.id=e.inventory_item_id
WHERE m.name='Caldinho de feijão' AND i.internal_code IN ('EQP-001','UTN-002');
INSERT OR IGNORE INTO menu_item_equipment(menu_item_id,equipment_id,quantity,required)
SELECT m.id,e.id,1,1 FROM menu_items m CROSS JOIN equipment e
JOIN inventory_items i ON i.id=e.inventory_item_id
WHERE m.category_id IN (2,3) AND i.internal_code='UTN-001';

INSERT OR IGNORE INTO decoration_templates(name,description,active,created_at,updated_at)
VALUES('Decoração dourada','Composição dourada para mesa do bolo e convidados.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
      ('Decoração rústica','Composição com folhagens, madeira e luzes.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT OR IGNORE INTO decoration_template_items(template_id,decoration_id,quantity)
SELECT t.id,d.id,1 FROM decoration_templates t CROSS JOIN decorations d WHERE t.name='Decoração dourada';
