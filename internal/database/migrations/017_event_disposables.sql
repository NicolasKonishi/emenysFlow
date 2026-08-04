UPDATE inventory_items
SET name='Copo descartável',description='Copo descartável utilizado nos eventos.',subcategory='Descartáveis de evento',unit='unidade',updated_at=CURRENT_TIMESTAMP
WHERE internal_code='DES-001';

UPDATE inventory_items
SET name='Guardanapo de papel',description='Pacote de guardanapos de papel utilizado nos eventos.',subcategory='Descartáveis de evento',unit='pacote',updated_at=CURRENT_TIMESTAMP
WHERE internal_code='DES-003';

UPDATE inventory_items
SET name='Saco de lixo',description='Saco de lixo utilizado na operação dos eventos.',subcategory='Descartáveis de evento',unit='unidade',updated_at=CURRENT_TIMESTAMP
WHERE internal_code='DES-002';

INSERT OR IGNORE INTO inventory_items(
    name,description,category_id,subcategory,unit,stock_quantity,minimum_stock,damaged_quantity,location_id,
    internal_code,item_kind,ownership,requires_return,replacement_value_cents,notes,active,created_at,updated_at
)
SELECT 'Papel toalha','Pacote de papel toalha para a cozinha do evento.',id,'Descartáveis de evento','pacote',0,0,0,(SELECT id FROM inventory_locations WHERE name='Estoque de descartáveis / Corredor A'),'DES-004','consumable','owned',0,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Descartáveis'
UNION ALL SELECT 'Papel alumínio','Rolo de papel alumínio para a cozinha do evento.',id,'Descartáveis de evento','rolo',0,0,0,(SELECT id FROM inventory_locations WHERE name='Estoque de descartáveis / Corredor A'),'DES-005','consumable','owned',0,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Descartáveis'
UNION ALL SELECT 'Papel filme','Rolo de papel filme para a cozinha do evento.',id,'Descartáveis de evento','rolo',0,0,0,(SELECT id FROM inventory_locations WHERE name='Estoque de descartáveis / Corredor A'),'DES-006','consumable','owned',0,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Descartáveis'
UNION ALL SELECT 'Saco culinário','Pacote de sacos culinários para preparo e armazenamento.',id,'Descartáveis de evento','pacote',0,0,0,(SELECT id FROM inventory_locations WHERE name='Estoque de descartáveis / Corredor A'),'DES-007','consumable','owned',0,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Descartáveis'
UNION ALL SELECT 'Detergente','Frasco de detergente para a limpeza do evento.',id,'Descartáveis de evento','frasco',0,0,0,(SELECT id FROM inventory_locations WHERE name='Estoque de descartáveis / Corredor A'),'DES-008','consumable','owned',0,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Descartáveis'
UNION ALL SELECT 'Esponja','Esponja de limpeza utilizada no evento.',id,'Descartáveis de evento','unidade',0,0,0,(SELECT id FROM inventory_locations WHERE name='Estoque de descartáveis / Corredor A'),'DES-009','consumable','owned',0,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Descartáveis'
UNION ALL SELECT 'Palha de aço','Pacote de palha de aço para limpeza.',id,'Descartáveis de evento','pacote',0,0,0,(SELECT id FROM inventory_locations WHERE name='Estoque de descartáveis / Corredor A'),'DES-010','consumable','owned',0,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Descartáveis'
UNION ALL SELECT 'Álcool etílico','Frasco de álcool etílico para a operação do evento.',id,'Descartáveis de evento','frasco',0,0,0,(SELECT id FROM inventory_locations WHERE name='Estoque de descartáveis / Corredor A'),'DES-011','consumable','owned',0,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Descartáveis'
UNION ALL SELECT 'Álcool acendedor','Frasco de álcool acendedor para réchauds e equipamentos.',id,'Descartáveis de evento','frasco',0,0,0,(SELECT id FROM inventory_locations WHERE name='Estoque de descartáveis / Corredor A'),'DES-012','consumable','owned',0,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Descartáveis'
UNION ALL SELECT 'Sabão em pedra','Sabão em pedra utilizado na limpeza do evento.',id,'Descartáveis de evento','unidade',0,0,0,(SELECT id FROM inventory_locations WHERE name='Estoque de descartáveis / Corredor A'),'DES-013','consumable','owned',0,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Descartáveis'
UNION ALL SELECT 'Fósforo','Caixa de fósforos para a operação do evento.',id,'Descartáveis de evento','caixa',0,0,0,(SELECT id FROM inventory_locations WHERE name='Estoque de descartáveis / Corredor A'),'DES-014','consumable','owned',0,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Descartáveis'
UNION ALL SELECT 'Palito de dente','Caixa grande de palitos de dente.',id,'Descartáveis de evento','caixa grande',0,0,0,(SELECT id FROM inventory_locations WHERE name='Estoque de descartáveis / Corredor A'),'DES-015','consumable','owned',0,0,'Dobrar quando houver welcome drinks ou dadinho de tapioca.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Descartáveis'
UNION ALL SELECT 'Luvas descartáveis','Pares de luvas descartáveis para a equipe de cozinha.',id,'Descartáveis de evento','par',0,0,0,(SELECT id FROM inventory_locations WHERE name='Estoque de descartáveis / Corredor A'),'DES-016','consumable','owned',0,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Descartáveis'
UNION ALL SELECT 'Touca descartável','Touca descartável para a equipe de cozinha.',id,'Descartáveis de evento','unidade',0,0,0,(SELECT id FROM inventory_locations WHERE name='Estoque de descartáveis / Corredor A'),'DES-017','consumable','owned',0,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Descartáveis';

-- Saldo inicial de demonstração; permanece totalmente editável no estoque.
UPDATE inventory_items
SET stock_quantity=100,updated_at=CURRENT_TIMESTAMP
WHERE internal_code IN ('DES-004','DES-005','DES-006','DES-007','DES-008','DES-009','DES-010','DES-011','DES-012','DES-013','DES-014','DES-015','DES-016','DES-017')
  AND stock_quantity=0;

UPDATE calculation_rules
SET name='Copos descartáveis por convidado',description='Três copos descartáveis por convidado.',calculation_type='per_person',base_value=0,divisor=1,multiplier=3,minimum_quantity=NULL,maximum_quantity=NULL,safety_percent=0,condition_json='{}',priority=40,active=1,updated_at=CURRENT_TIMESTAMP
WHERE rule_key='disposable_cups';

UPDATE calculation_rules
SET name='Guardanapo de papel por convidados',description='Um pacote para cada grupo de 18 convidados.',calculation_type='group_of_people',base_value=0,divisor=18,multiplier=1,minimum_quantity=NULL,maximum_quantity=NULL,safety_percent=0,condition_json='{}',priority=41,active=1,updated_at=CURRENT_TIMESTAMP
WHERE rule_key='napkins';

UPDATE calculation_rules
SET name='Sacos de lixo por convidados',description='Dez sacos para cada grupo de 100 convidados.',calculation_type='group_of_people',base_value=0,divisor=100,multiplier=10,minimum_quantity=NULL,maximum_quantity=NULL,safety_percent=0,condition_json='{}',priority=46,active=1,updated_at=CURRENT_TIMESTAMP
WHERE rule_key='trash_bags';

INSERT OR IGNORE INTO calculation_rules(
    rule_key,name,description,category_id,trigger_event,calculation_type,base_value,divisor,multiplier,
    minimum_quantity,maximum_quantity,safety_percent,condition_json,result_inventory_item_id,priority,active,created_at,updated_at
)
SELECT 'paper_towels','Papel toalha por convidados','Um pacote para cada grupo de 30 convidados.',category.id,'checklist_generation','group_of_people',0,30,1,NULL,NULL,0,'{}',item.id,42,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id WHERE item.internal_code='DES-004'
UNION ALL SELECT 'aluminum_foil','Papel alumínio por convidados','Dois rolos para cada grupo de 100 convidados.',category.id,'checklist_generation','group_of_people',0,100,2,NULL,NULL,0,'{}',item.id,43,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id WHERE item.internal_code='DES-005'
UNION ALL SELECT 'plastic_wrap','Papel filme por convidados','Dois rolos para cada grupo de 100 convidados.',category.id,'checklist_generation','group_of_people',0,100,2,NULL,NULL,0,'{}',item.id,44,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id WHERE item.internal_code='DES-006'
UNION ALL SELECT 'culinary_bags','Saco culinário por convidados','Dois pacotes para cada grupo de 100 convidados.',category.id,'checklist_generation','group_of_people',0,100,2,NULL,NULL,0,'{}',item.id,45,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id WHERE item.internal_code='DES-007'
UNION ALL SELECT 'detergent','Detergente por convidados','Cinco frascos para cada grupo de 100 convidados.',category.id,'checklist_generation','group_of_people',0,100,5,NULL,NULL,0,'{}',item.id,47,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id WHERE item.internal_code='DES-008'
UNION ALL SELECT 'sponges','Esponjas por convidados','Cinco esponjas para cada grupo de 100 convidados.',category.id,'checklist_generation','group_of_people',0,100,5,NULL,NULL,0,'{}',item.id,48,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id WHERE item.internal_code='DES-009'
UNION ALL SELECT 'steel_wool','Palha de aço por convidados','Seis pacotes para cada grupo de 100 convidados.',category.id,'checklist_generation','group_of_people',0,100,6,NULL,NULL,0,'{}',item.id,49,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id WHERE item.internal_code='DES-010'
UNION ALL SELECT 'ethyl_alcohol','Álcool etílico por convidados','Um frasco para cada grupo de 150 convidados.',category.id,'checklist_generation','group_of_people',0,150,1,NULL,NULL,0,'{}',item.id,50,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id WHERE item.internal_code='DES-011'
UNION ALL SELECT 'firelighter_alcohol','Álcool acendedor por convidados','Um frasco para cada grupo de 200 convidados.',category.id,'checklist_generation','group_of_people',0,200,1,NULL,NULL,0,'{}',item.id,51,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id WHERE item.internal_code='DES-012'
UNION ALL SELECT 'bar_soap','Sabão em pedra por convidados','Três unidades para cada grupo de 100 convidados.',category.id,'checklist_generation','group_of_people',0,100,3,NULL,NULL,0,'{}',item.id,52,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id WHERE item.internal_code='DES-013'
UNION ALL SELECT 'matches','Fósforos por convidados','Cinco caixas para cada grupo de 100 convidados.',category.id,'checklist_generation','group_of_people',0,100,5,NULL,NULL,0,'{}',item.id,53,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id WHERE item.internal_code='DES-014'
UNION ALL SELECT 'toothpicks_standard','Palitos de dente por convidados','Uma caixa grande para cada grupo de 100 convidados.',category.id,'checklist_generation','group_of_people',0,100,1,NULL,NULL,0,'{"welcome_drinks":false,"dadinho_tapioca":false}',item.id,54,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id WHERE item.internal_code='DES-015'
UNION ALL SELECT 'toothpicks_welcome','Palitos de dente com welcome drinks','Duas caixas grandes para cada grupo de 100 convidados quando houver welcome drinks.',category.id,'checklist_generation','group_of_people',0,100,2,NULL,NULL,0,'{"welcome_drinks":true}',item.id,54,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id WHERE item.internal_code='DES-015'
UNION ALL SELECT 'toothpicks_dadinho','Palitos de dente com dadinho de tapioca','Duas caixas grandes para cada grupo de 100 convidados quando houver dadinho de tapioca sem welcome drinks.',category.id,'checklist_generation','group_of_people',0,100,2,NULL,NULL,0,'{"welcome_drinks":false,"dadinho_tapioca":true}',item.id,54,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id WHERE item.internal_code='DES-015'
UNION ALL SELECT 'gloves','Luvas descartáveis por convidados','Oito pares para cada grupo de 100 convidados.',category.id,'checklist_generation','group_of_people',0,100,8,NULL,NULL,0,'{}',item.id,55,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id WHERE item.internal_code='DES-016'
UNION ALL SELECT 'hairnets','Toucas descartáveis por convidados','Seis toucas para cada grupo de 100 convidados.',category.id,'checklist_generation','group_of_people',0,100,6,NULL,NULL,0,'{}',item.id,56,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_items item JOIN inventory_categories category ON category.id=item.category_id WHERE item.internal_code='DES-017';

-- Guardanapos agora são calculados globalmente por evento. Remove o vínculo
-- específico do serviço de bar para evitar duas linhas para o mesmo consumo.
UPDATE service_component_inventory_links
SET active=0
WHERE inventory_item_id=(SELECT id FROM inventory_items WHERE internal_code='DES-003');

UPDATE event_service_component_inventory_links
SET active=0
WHERE inventory_item_id=(SELECT id FROM inventory_items WHERE internal_code='DES-003');
