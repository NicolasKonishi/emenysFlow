INSERT OR IGNORE INTO inventory_items(name,description,category_id,unit,stock_quantity,minimum_stock,damaged_quantity,internal_code,item_kind,ownership,requires_return,replacement_value_cents,active,created_at,updated_at)
SELECT 'Gelo','Insumo para serviço de bar',id,'kg',0,0,0,'BAR-GELO','consumable','owned',0,0,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Descartáveis'
UNION ALL SELECT 'Canudo','Canudo para bebidas',id,'unidade',0,0,0,'BAR-CANUDO','consumable','owned',0,0,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Descartáveis'
UNION ALL SELECT 'Copo de acrílico','Copo para caipirinhas',id,'unidade',0,0,0,'BAR-COPO','consumable','owned',0,0,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Descartáveis'
UNION ALL SELECT 'Robô de LED','Estrutura do robô de LED',id,'unidade',0,0,0,'LED-ROBO','outsourced','outsourced',1,0,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Itens alugados'
UNION ALL SELECT 'Canhão de CO2','Canhão para apresentação',id,'unidade',0,0,0,'LED-CANHAO','outsourced','outsourced',1,0,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Itens alugados'
UNION ALL SELECT 'Pista de LED 4x4','Pista modular iluminada',id,'serviço',0,0,0,'LED-PISTA','outsourced','outsourced',1,0,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Itens alugados'
UNION ALL SELECT 'Mesa redonda','Mesa para oito convidados',id,'unidade',0,0,0,'DEC-MESA-8','rented','rented',1,0,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Itens alugados';

INSERT OR IGNORE INTO service_component_inventory_links(service_template_component_id,inventory_item_id,quantity_formula,ownership,created_at)
SELECT link.id,inventory.id,CASE component.name WHEN 'Guardanapos de papel' THEN 'guests' ELSE '' END,CASE inventory.item_kind WHEN 'consumable' THEN 'consumable' ELSE inventory.ownership END,CURRENT_TIMESTAMP
FROM service_template_components link JOIN service_components component ON component.id=link.service_component_id JOIN service_templates service ON service.id=link.service_template_id JOIN inventory_items inventory ON
 (service.slug='bar' AND ((component.name='Gelo' AND inventory.internal_code='BAR-GELO') OR (component.name='Canudos' AND inventory.internal_code='BAR-CANUDO') OR (component.name='Copos de acrílico' AND inventory.internal_code='BAR-COPO') OR (component.name='Guardanapos de papel' AND inventory.internal_code='DES-003')))
 OR (service.slug='robo-de-led' AND ((component.name='Robô de LED' AND inventory.internal_code='LED-ROBO') OR (component.name='Canhão de CO2' AND inventory.internal_code='LED-CANHAO')))
 OR (service.slug='pista-de-led' AND component.name='Pista de LED' AND inventory.internal_code='LED-PISTA')
 OR (service.slug='decoracao' AND component.name='Mesas redondas' AND inventory.internal_code='DEC-MESA-8');

UPDATE service_component_inventory_links SET quantity_formula='ceil(guests/8)'
WHERE service_template_component_id IN (
  SELECT link.id FROM service_template_components link
  JOIN service_templates service ON service.id=link.service_template_id
  JOIN service_components component ON component.id=link.service_component_id
  WHERE service.slug='decoracao' AND component.name='Mesas redondas'
);
