-- Makes every event menu section selectable from an administrator-managed catalog.
ALTER TABLE menu_items ADD COLUMN result_inventory_item_id INTEGER REFERENCES inventory_items(id);
ALTER TABLE menu_items ADD COLUMN calculation_type TEXT NOT NULL DEFAULT 'menu_only';
ALTER TABLE menu_items ADD COLUMN calculation_group TEXT NOT NULL DEFAULT '';
ALTER TABLE menu_items ADD COLUMN calculation_divisor REAL NOT NULL DEFAULT 1;
ALTER TABLE menu_items ADD COLUMN calculation_multiplier REAL NOT NULL DEFAULT 1;
ALTER TABLE menu_items ADD COLUMN calculation_weight REAL NOT NULL DEFAULT 1;

INSERT OR IGNORE INTO menu_categories(name,sort_order,active,created_at,updated_at)
VALUES('Bebidas',40,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
UPDATE menu_categories SET sort_order=50 WHERE name='Mesa de café';
UPDATE menu_categories SET sort_order=60 WHERE name='Sobremesas';

INSERT OR IGNORE INTO menu_items(category_id,name,description,result_inventory_item_id,calculation_type,calculation_group,calculation_divisor,calculation_multiplier,calculation_weight,active,created_at,updated_at)
SELECT c.id,'Coca-Cola','Refrigerante PET',i.id,'category_distribution','soda',2,1,60,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM menu_categories c,inventory_items i WHERE c.name='Bebidas' AND i.internal_code='BEB-001';
INSERT OR IGNORE INTO menu_items(category_id,name,description,result_inventory_item_id,calculation_type,calculation_group,calculation_divisor,calculation_multiplier,calculation_weight,active,created_at,updated_at)
SELECT c.id,'Guaraná','Refrigerante PET',i.id,'category_distribution','soda',2,1,40,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM menu_categories c,inventory_items i WHERE c.name='Bebidas' AND i.internal_code='BEB-002';
INSERT OR IGNORE INTO menu_items(category_id,name,description,result_inventory_item_id,calculation_type,calculation_group,calculation_divisor,calculation_multiplier,calculation_weight,active,created_at,updated_at)
SELECT c.id,'Suco de laranja','Caixa de suco',i.id,'category_distribution','juice',2,1,50,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM menu_categories c,inventory_items i WHERE c.name='Bebidas' AND i.internal_code='BEB-003';
INSERT OR IGNORE INTO menu_items(category_id,name,description,result_inventory_item_id,calculation_type,calculation_group,calculation_divisor,calculation_multiplier,calculation_weight,active,created_at,updated_at)
SELECT c.id,'Suco de uva','Caixa de suco',i.id,'category_distribution','juice',2,1,50,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM menu_categories c,inventory_items i WHERE c.name='Bebidas' AND i.internal_code='BEB-004';

INSERT OR IGNORE INTO event_menu_items(event_id,menu_item_id,portions,calculated_container_quantity,notes,created_at,updated_at)
SELECT 1,m.id,200,0,'',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM menu_items m JOIN menu_categories c ON c.id=m.category_id WHERE c.name='Bebidas';

-- Beverage quantities are now derived only from the flavors selected in the event.
UPDATE calculation_rules SET active=0,updated_at=CURRENT_TIMESTAMP WHERE rule_key IN ('soda_coke','soda_guarana','juice_orange','juice_grape');
