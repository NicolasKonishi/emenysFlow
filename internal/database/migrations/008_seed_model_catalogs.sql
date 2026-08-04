-- Modelos, seções e blocos compartilhados. Todos os registros usam chaves naturais únicas.
INSERT INTO menu_templates(slug,name,description,menu_type,active,current_version,source_name,source_updated_month,created_at,updated_at)
VALUES
('buffet-especial','Buffet Especial','Modelo de buffet com entradas, dois tipos de carne, acompanhamentos, bebidas, mesa de café, bolo, doces, materiais e equipe.','buffet',1,1,'Emeny''s Eventos - Cardápios','2025-01',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('buffet-crepe-frances','Buffet Crepe Francês','Buffet de crepes salgados e doces com sabores configuráveis.','buffet',1,1,'Emeny''s Eventos - Cardápios','2025-01',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('buffet-feijoada','Buffet Feijoada','Buffet de feijoada com entradas, acompanhamentos e sobremesas.','buffet',1,1,'Emeny''s Eventos - Cardápios','2025-01',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('buffet-boteco','Buffet Boteco','Buffet inspirado em comidas de boteco.','buffet',1,1,'Emeny''s Eventos - Cardápios','2025-01',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('buffet-comida-mineira','Buffet Comida Mineira','Buffet de pratos tradicionais da cozinha mineira.','buffet',1,1,'Emeny''s Eventos - Cardápios','2025-01',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('buffet-nordestino','Buffet Nordestino','Buffet de pratos tradicionais da cozinha nordestina.','buffet',1,1,'Emeny''s Eventos - Cardápios','2025-01',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('buffet-carnes','Buffet Carnes','Buffet com escolhas configuráveis de entradas, carnes e sobremesas.','buffet',1,1,'Emeny''s Eventos - Cardápios','2025-01',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('buffet-da-casa','Buffet da Casa','Buffet completo da casa com entradas, massas, carnes e acompanhamentos.','buffet',1,1,'Emeny''s Eventos - Cardápios','2025-01',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('buffet-brunch','Buffet Brunch','Brunch com frios, pães, salgados, bebidas, frutas e cereais.','brunch',1,1,'Emeny''s Eventos - Cardápios','2025-01',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('buffet-finger-foods','Buffet Finger Foods','Buffet de pequenas porções, fritos e assados.','finger-foods',1,1,'Emeny''s Eventos - Cardápios','2025-01',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('buffet-de-massas','Buffet de Massas','Buffet de massas com sabores configuráveis, molhos e acompanhamentos.','buffet',1,1,'Emeny''s Eventos - Cardápios','2025-01',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('buffet-de-churrasco','Buffet de Churrasco','Churrasco com carnes, entradas e acompanhamentos.','churrasco',1,1,'Emeny''s Eventos - Cardápios','2025-01',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('buffet-carnes-e-massas','Buffet Carnes e Massas','Buffet combinado de carnes, massas e acompanhamentos.','buffet',1,1,'Emeny''s Eventos - Cardápios','2025-01',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('buffet-churrasco-fogo-de-chao','Buffet Churrasco Fogo de Chão','Churrasco com cortes variados, entradas e acompanhamentos.','churrasco',1,1,'Emeny''s Eventos - Cardápios','2025-01',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
ON CONFLICT(slug) DO UPDATE SET name=excluded.name,description=excluded.description,menu_type=excluded.menu_type,source_name=excluded.source_name,source_updated_month=excluded.source_updated_month,updated_at=CURRENT_TIMESTAMP;

INSERT OR IGNORE INTO menu_template_versions(menu_template_id,version,change_summary,snapshot_json,created_at)
SELECT id,1,'Importação inicial da fonte','{}',CURRENT_TIMESTAMP FROM menu_templates;

INSERT INTO menu_sections(slug,name,section_type,active,created_at,updated_at) VALUES
('entradas','Entradas','food',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('buffet-principal','Buffet principal','food',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('acompanhamentos','Acompanhamentos','food',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('crepes-salgados','Crepes salgados','food',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('crepes-doces','Crepes doces','dessert',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('massas','Massas','food',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('molhos','Molhos','food',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('fritos','Fritos','food',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('assados','Assados','food',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('frios','Frios','food',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('paes','Pães','food',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('salgados','Salgados','food',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('frutas','Frutas','food',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('cereais','Cereais e complementos','food',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('bebidas','Bebidas','beverage',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('mesa-de-cafe','Mesa de café','service',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('bolo','Bolo','dessert',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('doces','Doces','dessert',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('sobremesas','Sobremesas','dessert',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('materiais','Materiais','material',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('equipe','Equipe','staff',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('salada','Salada','food',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('temperos-e-molhos','Temperos e molhos','food',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
ON CONFLICT(slug) DO UPDATE SET name=excluded.name,section_type=excluded.section_type,updated_at=CURRENT_TIMESTAMP;

INSERT OR IGNORE INTO menu_shared_blocks(slug,name,description,active,created_at,updated_at) VALUES
('bebidas-padrao','Bebidas padrão','Bebidas incluídas com quantidades calculadas pelas regras do evento.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('mesa-de-cafe-padrao','Mesa de café padrão','Itens padrão da mesa de café.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('doces-padrao','Doces padrão','Seleção padrão de doces.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('bolo-padrao','Bolo padrão','Bolo com sabor e características definidos no evento.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('materiais-padrao','Materiais padrão','Materiais calculados pelo motor de regras.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('materiais-crepe','Materiais para crepe','Materiais específicos do buffet de crepes.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('equipe-padrao','Equipe padrão','Funções dimensionadas pelas regras do evento.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);

-- Inventário necessário aos blocos compartilhados; saldos começam em zero e sem valor inventado.
INSERT OR IGNORE INTO inventory_items(name,description,category_id,unit,stock_quantity,minimum_stock,damaged_quantity,location_id,internal_code,item_kind,ownership,requires_return,replacement_value_cents,active,created_at,updated_at)
SELECT 'Rechaud','Equipamento de serviço',id,'unidade',0,0,0,NULL,'MAT-RECHAUD','reusable','owned',1,0,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Cubas e recipientes'
UNION ALL SELECT 'Talher de inox','Talher de serviço',id,'unidade',0,0,0,NULL,'MAT-TALHER','reusable','owned',1,0,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Louças'
UNION ALL SELECT 'Taça de vidro','Taça para bebidas',id,'unidade',0,0,0,NULL,'MAT-TACA','reusable','owned',1,0,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Louças'
UNION ALL SELECT 'Crepeira','Equipamento para preparo de crepes',id,'unidade',0,0,0,NULL,'EQP-CREPEIRA','reusable','owned',1,0,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Equipamentos de cozinha'
UNION ALL SELECT 'Cozinheira','Profissional de cozinha',id,'profissional',0,0,0,NULL,'STA-COZINHEIRA','outsourced','outsourced',0,0,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Equipe'
UNION ALL SELECT 'Metriê','Profissional responsável pelo salão',id,'profissional',0,0,0,NULL,'STA-METRIE','outsourced','outsourced',0,0,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Equipe'
UNION ALL SELECT 'Copeira','Profissional de copa',id,'profissional',0,0,0,NULL,'STA-COPEIRA','outsourced','outsourced',0,0,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Equipe'
UNION ALL SELECT 'Coordenador','Profissional de coordenação',id,'profissional',0,0,0,NULL,'STA-COORDENADOR','outsourced','outsourced',0,0,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM inventory_categories WHERE name='Equipe';
