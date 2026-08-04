-- Itens de uso eventual permanecem no estoque, mas deixam de entrar
-- automaticamente em toda checklist ou serviço de bar.
UPDATE calculation_rules
SET active=0,updated_at=CURRENT_TIMESTAMP
WHERE rule_key='reusable_cups';

UPDATE service_template_components
SET included=0,optional=1,updated_at=CURRENT_TIMESTAMP
WHERE service_component_id IN (
    SELECT id FROM service_components WHERE name IN ('Gelo','Canudos','Copos de acrílico')
);

UPDATE event_service_components
SET selected=0,updated_at=CURRENT_TIMESTAMP
WHERE name IN ('Gelo','Canudos','Copos de acrílico');

-- A colher de serviço solta é substituída pela caixa de pegadores.
DELETE FROM menu_item_equipment
WHERE equipment_id IN (
    SELECT equipment.id
    FROM equipment
    JOIN inventory_items item ON item.id=equipment.inventory_item_id
    WHERE item.internal_code='UTN-001'
);

DELETE FROM event_menu_item_equipment
WHERE inventory_item_id=(SELECT id FROM inventory_items WHERE internal_code='UTN-001');

UPDATE equipment
SET active=0,updated_at=CURRENT_TIMESTAMP
WHERE inventory_item_id=(SELECT id FROM inventory_items WHERE internal_code='UTN-001');

INSERT OR IGNORE INTO inventory_items(
    name,description,category_id,subcategory,unit,stock_quantity,minimum_stock,damaged_quantity,location_id,
    internal_code,item_kind,ownership,requires_return,replacement_value_cents,notes,active,created_at,updated_at
)
SELECT 'Caixa de pegadores','Caixa com os diversos pegadores utilizados nas comidas do buffet.',id,'Caixas do buffet','caixa',3,1,0,5,
       'CUB-CAIXA-PEGADORES','reusable','owned',1,0,'Conferir os pegadores na saída e no retorno.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
FROM inventory_categories WHERE name='Cubas e utensílios de buffet';

INSERT OR IGNORE INTO calculation_rules(
    rule_key,name,description,category_id,trigger_event,calculation_type,base_value,divisor,multiplier,
    minimum_quantity,maximum_quantity,safety_percent,condition_json,result_inventory_item_id,priority,active,created_at,updated_at
)
SELECT 'tongs_box','Caixa de pegadores por evento','Uma caixa de pegadores em toda festa.',item.category_id,
       'checklist_generation','fixed',1,1,1,1,NULL,0,'{}',item.id,35,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
FROM inventory_items item WHERE item.internal_code='CUB-CAIXA-PEGADORES';

-- Kit mínimo de quinze panelas por evento, dividido por tipo.
UPDATE inventory_items
SET name='Panela grande',description='Panela grande para produção do buffet.',stock_quantity=10,minimum_stock=5,active=1,updated_at=CURRENT_TIMESTAMP
WHERE internal_code='EQP-002';

INSERT OR IGNORE INTO inventory_items(
    name,description,category_id,subcategory,unit,stock_quantity,minimum_stock,damaged_quantity,location_id,
    internal_code,item_kind,ownership,requires_return,replacement_value_cents,notes,active,created_at,updated_at
)
SELECT 'Panela de pressão','Panela de pressão para produção do buffet.',id,'Panelas','unidade',6,3,0,5,'EQP-PANELA-PRESSAO','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Equipamentos de cozinha'
UNION ALL SELECT 'Caldeirão','Caldeirão para produção do buffet.',id,'Panelas','unidade',6,3,0,5,'EQP-CALDEIRAO','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Equipamentos de cozinha'
UNION ALL SELECT 'Panela média','Panela média para produção do buffet.',id,'Panelas','unidade',10,4,0,5,'EQP-PANELA-MEDIA','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Equipamentos de cozinha';

-- Os vínculos antigos apontavam todos os pratos para a mesma panela grande.
-- O kit fixo abaixo passa a representar a necessidade operacional real.
UPDATE menu_items
SET pan_inventory_item_id=NULL,updated_at=CURRENT_TIMESTAMP
WHERE pan_inventory_item_id=(SELECT id FROM inventory_items WHERE internal_code='EQP-002');

INSERT OR IGNORE INTO calculation_rules(
    rule_key,name,description,category_id,trigger_event,calculation_type,base_value,divisor,multiplier,
    minimum_quantity,maximum_quantity,safety_percent,condition_json,result_inventory_item_id,priority,active,created_at,updated_at
)
SELECT 'pressure_pans','Panelas de pressão por evento','Três panelas de pressão no kit mínimo do evento.',item.category_id,'checklist_generation','fixed',3,1,1,3,NULL,0,'{}',item.id,120,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item WHERE item.internal_code='EQP-PANELA-PRESSAO'
UNION ALL SELECT 'cauldrons','Caldeirões por evento','Três caldeirões no kit mínimo do evento.',item.category_id,'checklist_generation','fixed',3,1,1,3,NULL,0,'{}',item.id,121,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item WHERE item.internal_code='EQP-CALDEIRAO'
UNION ALL SELECT 'large_pans','Panelas grandes por evento','Cinco panelas grandes no kit mínimo do evento.',item.category_id,'checklist_generation','fixed',5,1,1,5,NULL,0,'{}',item.id,122,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item WHERE item.internal_code='EQP-002'
UNION ALL SELECT 'medium_pans','Panelas médias por evento','Quatro panelas médias no kit mínimo do evento.',item.category_id,'checklist_generation','fixed',4,1,1,4,NULL,0,'{}',item.id,123,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item WHERE item.internal_code='EQP-PANELA-MEDIA';

-- Rechauds deixam de ser uma unidade calculada: as cubas representam as
-- posições de serviço e serão duplicadas acima de cem convidados.
UPDATE inventory_items
SET active=0,updated_at=CURRENT_TIMESTAMP
WHERE internal_code='CUB-002';
