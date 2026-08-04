-- Operational settings, roles, staff rules, shortages and recalculation audit.
-- Existing columns and workflow statuses are intentionally preserved for compatibility.

CREATE TABLE IF NOT EXISTS operational_settings (
    setting_key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    numeric_value REAL NOT NULL CHECK(numeric_value >= 0),
    unit TEXT NOT NULL DEFAULT 'unidade',
    updated_by INTEGER REFERENCES users(id),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

INSERT OR IGNORE INTO operational_settings(setting_key,name,description,numeric_value,unit,created_at,updated_at) VALUES
('additional_staff_margin','Margem operacional adicional','Quantidade fixa de pessoas acrescentada aos cálculos de louças e utensílios.',20,'pessoas',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('people_per_coffee_spoon_kit','Pessoas por kit de colheres de café','Quantidade de pessoas atendida por cada kit de colheres de café.',50,'pessoas',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('pans_per_rechaud','Cubas por rechaud','Quantidade de cubas atendida por cada rechaud.',1,'cubas',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);

ALTER TABLE users ADD COLUMN access_role TEXT NOT NULL DEFAULT 'operational'
    CHECK(access_role IN ('operational','organizer','admin'));
ALTER TABLE users ADD COLUMN row_version INTEGER NOT NULL DEFAULT 1 CHECK(row_version > 0);

UPDATE users SET access_role=CASE WHEN role='admin' THEN 'admin' ELSE 'operational' END;

ALTER TABLE events ADD COLUMN coordinator_override INTEGER CHECK(coordinator_override IS NULL OR coordinator_override >= 0);
ALTER TABLE events ADD COLUMN leader_override INTEGER CHECK(leader_override IS NULL OR leader_override >= 0);
ALTER TABLE events ADD COLUMN co_leader_override INTEGER CHECK(co_leader_override IS NULL OR co_leader_override >= 0);
ALTER TABLE events ADD COLUMN additional_guest_margin_override REAL CHECK(additional_guest_margin_override IS NULL OR additional_guest_margin_override >= 0);
ALTER TABLE events ADD COLUMN uses_glassware INTEGER NOT NULL DEFAULT 1 CHECK(uses_glassware IN (0,1));
ALTER TABLE events ADD COLUMN row_version INTEGER NOT NULL DEFAULT 1 CHECK(row_version > 0);
ALTER TABLE events ADD COLUMN updated_by INTEGER REFERENCES users(id);

ALTER TABLE checklists ADD COLUMN updated_by INTEGER REFERENCES users(id);
ALTER TABLE checklist_items ADD COLUMN active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1));
ALTER TABLE checklist_items ADD COLUMN row_version INTEGER NOT NULL DEFAULT 1 CHECK(row_version > 0);
ALTER TABLE checklist_items ADD COLUMN updated_by INTEGER REFERENCES users(id);

CREATE TABLE IF NOT EXISTS suppliers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE COLLATE NOCASE,
    contact_name TEXT NOT NULL DEFAULT '',
    phone TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS checklist_shortages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    checklist_item_id INTEGER NOT NULL REFERENCES checklist_items(id) ON DELETE CASCADE,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    missing_quantity REAL NOT NULL CHECK(missing_quantity > 0),
    reason TEXT NOT NULL,
    resolution_type TEXT NOT NULL CHECK(resolution_type IN ('purchase','rental','substitution','stock_transfer','production','wait_return','other')),
    status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','purchasing','renting','resolved','cancelled')),
    responsible_user_id INTEGER REFERENCES users(id),
    responsible_name TEXT NOT NULL DEFAULT '',
    due_at TEXT,
    supplier_id INTEGER REFERENCES suppliers(id),
    supplier_name TEXT NOT NULL DEFAULT '',
    estimated_cost_cents INTEGER CHECK(estimated_cost_cents IS NULL OR estimated_cost_cents >= 0),
    notes TEXT NOT NULL DEFAULT '',
    automatic INTEGER NOT NULL DEFAULT 0 CHECK(automatic IN (0,1)),
    resolution_destination TEXT CHECK(resolution_destination IS NULL OR resolution_destination IN ('separation','loading')),
    resolved_by INTEGER REFERENCES users(id),
    resolved_at TEXT,
    row_version INTEGER NOT NULL DEFAULT 1 CHECK(row_version > 0),
    created_by INTEGER REFERENCES users(id),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_checklist_shortages_active_item
    ON checklist_shortages(checklist_item_id)
    WHERE status NOT IN ('resolved','cancelled');
CREATE INDEX IF NOT EXISTS idx_checklist_shortages_event_status
    ON checklist_shortages(event_id,status,due_at);

CREATE TABLE IF NOT EXISTS checklist_shortage_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    shortage_id INTEGER NOT NULL REFERENCES checklist_shortages(id) ON DELETE CASCADE,
    previous_status TEXT,
    new_status TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    changed_by INTEGER REFERENCES users(id),
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS checklist_item_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    checklist_item_id INTEGER NOT NULL REFERENCES checklist_items(id) ON DELETE CASCADE,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    action TEXT NOT NULL,
    before_json TEXT,
    after_json TEXT,
    performed_by INTEGER REFERENCES users(id),
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_checklist_item_history_item
    ON checklist_item_history(checklist_item_id,created_at);

CREATE TABLE IF NOT EXISTS event_recalculations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    trigger_key TEXT NOT NULL,
    previous_checklist_version INTEGER NOT NULL DEFAULT 0,
    new_checklist_version INTEGER NOT NULL DEFAULT 0,
    added_count INTEGER NOT NULL DEFAULT 0,
    removed_count INTEGER NOT NULL DEFAULT 0,
    quantity_updated_count INTEGER NOT NULL DEFAULT 0,
    shortage_count INTEGER NOT NULL DEFAULT 0,
    reservation_updated_count INTEGER NOT NULL DEFAULT 0,
    summary_json TEXT NOT NULL DEFAULT '{}',
    requested_by INTEGER REFERENCES users(id),
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS event_recalculation_changes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    recalculation_id INTEGER NOT NULL REFERENCES event_recalculations(id) ON DELETE CASCADE,
    source_key TEXT NOT NULL,
    change_type TEXT NOT NULL CHECK(change_type IN ('added','removed','quantity_updated','shortage_created','reservation_updated')),
    before_json TEXT,
    after_json TEXT,
    created_at TEXT NOT NULL
);

INSERT OR IGNORE INTO inventory_items(name,description,category_id,subcategory,unit,stock_quantity,minimum_stock,damaged_quantity,location_id,internal_code,item_kind,ownership,requires_return,replacement_value_cents,notes,active,created_at,updated_at)
SELECT 'Coordenador','Coordenador da equipe de atendimento.',id,'Equipe','profissional',30,0,0,NULL,'STA-002','outsourced','outsourced',0,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Itens dos garçons'
UNION ALL SELECT 'Líder','Líder da equipe de atendimento.',id,'Equipe','profissional',30,0,0,NULL,'STA-003','outsourced','outsourced',0,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Itens dos garçons'
UNION ALL SELECT 'Colíder','Colíder da equipe de atendimento.',id,'Equipe','profissional',30,0,0,NULL,'STA-004','outsourced','outsourced',0,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Itens dos garçons'
UNION ALL SELECT 'Taça','Taça utilizada no serviço do evento.',id,'Louças de serviço','unidade',300,100,0,NULL,'LOU-004','reusable','owned',1,2000,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Louças'
UNION ALL SELECT 'Kit de talheres','Kit de talheres para uma pessoa.',id,'Talheres','kit',300,100,0,NULL,'LOU-005','reusable','owned',1,3000,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Louças'
UNION ALL SELECT 'Colher de sobremesa','Colher individual para sobremesa.',id,'Talheres','unidade',300,100,0,NULL,'LOU-006','reusable','owned',1,700,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Louças'
UNION ALL SELECT 'Prato de sobremesa','Prato individual para sobremesa.',id,'Louças de sobremesa','unidade',300,100,0,NULL,'LOU-007','reusable','owned',1,1800,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Louças'
UNION ALL SELECT 'Copo de sobremesa','Copo individual para sobremesa.',id,'Louças de sobremesa','unidade',300,100,0,NULL,'LOU-008','reusable','owned',1,1000,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Louças'
UNION ALL SELECT 'Kit de colheres de café','Kit de colheres para o serviço de café.',id,'Talheres','kit',20,5,0,NULL,'CAF-001','reusable','owned',1,4000,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Mesa de café'
UNION ALL SELECT 'Rechaud','Rechaud para manutenção da temperatura das cubas.',id,'Rechauds','unidade',30,10,0,NULL,'CUB-002','reusable','owned',1,30000,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Cubas e utensílios de buffet';

UPDATE container_types
SET inventory_item_id=(SELECT id FROM inventory_items WHERE internal_code='LOU-008'),updated_at=CURRENT_TIMESTAMP
WHERE name='Copo de sobremesa';

UPDATE calculation_rules
SET description='Um prato por convidado mais a margem operacional fixa.',condition_json='{"has_main_buffet":true,"use_additional_guest_margin":true}',safety_percent=0,updated_at=CURRENT_TIMESTAMP
WHERE rule_key='dinner_plates';

INSERT OR IGNORE INTO calculation_rules(rule_key,name,description,category_id,trigger_event,calculation_type,base_value,divisor,multiplier,minimum_quantity,maximum_quantity,safety_percent,condition_json,result_inventory_item_id,priority,active,created_at,updated_at)
SELECT 'coordinators','Coordenadores por evento','Quantidade fixa de coordenadores por evento.',category.id,'checklist_generation','fixed',1,1,1,NULL,NULL,0,'{}',item.id,11,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id WHERE item.internal_code='STA-002'
UNION ALL SELECT 'leaders','Líderes por evento','Quantidade fixa de líderes por evento.',category.id,'checklist_generation','fixed',1,1,1,NULL,NULL,0,'{}',item.id,12,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id WHERE item.internal_code='STA-003'
UNION ALL SELECT 'co_leaders','Colíderes por evento','Quantidade fixa de colíderes por evento.',category.id,'checklist_generation','fixed',1,1,1,NULL,NULL,0,'{}',item.id,13,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id WHERE item.internal_code='STA-004'
UNION ALL SELECT 'glassware','Taças por convidado','Uma taça por convidado mais a margem operacional.',category.id,'checklist_generation','per_person',0,1,1,NULL,NULL,0,'{"uses_glassware":true,"use_additional_guest_margin":true}',item.id,72,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id WHERE item.internal_code='LOU-004'
UNION ALL SELECT 'cutlery_kits','Kits de talheres','Um kit por convidado mais a margem operacional.',category.id,'checklist_generation','per_person',0,1,1,NULL,NULL,0,'{"has_main_buffet":true,"use_additional_guest_margin":true}',item.id,73,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id WHERE item.internal_code='LOU-005'
UNION ALL SELECT 'dessert_spoons','Colheres de sobremesa','Uma colher por pessoa quando a sobremesa exigir colher.',category.id,'checklist_generation','per_person',0,1,1,NULL,NULL,0,'{"requires_dessert_spoon":true,"use_additional_guest_margin":true}',item.id,74,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id WHERE item.internal_code='LOU-006'
UNION ALL SELECT 'coffee_spoon_kits','Kits de colheres de café','Um kit por grupo configurado de pessoas quando houver mesa de café.',category.id,'checklist_generation','group_of_people',0,50,1,NULL,NULL,0,'{"coffee_table":true,"use_additional_guest_margin":true,"use_coffee_kit_divisor":true}',item.id,75,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id WHERE item.internal_code='CAF-001';
