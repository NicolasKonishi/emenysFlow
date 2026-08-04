-- Demonstration catalog. Rules remain ordinary database records and are editable.
INSERT OR IGNORE INTO inventory_categories (id, name, sort_order, active, created_at, updated_at) VALUES
 (1, 'Comidas', 10, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (2, 'Recipientes para transporte', 20, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (3, 'Cubas e utensílios de buffet', 30, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (4, 'Equipamentos de cozinha', 40, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (5, 'Louças', 50, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (6, 'Bebidas', 60, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (7, 'Descartáveis', 70, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (8, 'Itens dos garçons', 80, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (9, 'Mesa de café', 90, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (10, 'Bolo e doces', 100, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (11, 'Sobremesas', 110, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (12, 'Decoração', 120, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (13, 'Itens alugados', 130, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (14, 'Ferramentas', 140, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (15, 'Observações', 150, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT OR IGNORE INTO inventory_locations (id, name, description, active, created_at, updated_at) VALUES
 (1, 'Estoque de louças / Prateleira 2', 'Louças e itens de serviço', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (2, 'Estoque de descartáveis / Corredor A', 'Materiais consumíveis', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (3, 'Depósito de bebidas', 'Bebidas fechadas e suqueiras', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (4, 'Depósito de decoração', 'Peças decorativas e suportes', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (5, 'Área de equipamentos / Prateleira 1', 'Equipamentos e utensílios', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT OR IGNORE INTO inventory_items (id, name, description, category_id, unit, stock_quantity, minimum_stock, damaged_quantity, location_id, internal_code, item_kind, ownership, requires_return, replacement_value_cents, active, created_at, updated_at) VALUES
 (1, 'Garçom', 'Profissional de atendimento', 8, 'profissional', 30, 0, 0, NULL, 'STA-001', 'outsourced', 'outsourced', 0, 0, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (2, 'Jarra', 'Jarra de serviço', 8, 'unidade', 18, 10, 0, 1, 'GAR-001', 'reusable', 'owned', 1, 4500, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (3, 'Bandeja de garçom', 'Bandeja antiderrapante', 8, 'unidade', 16, 8, 0, 1, 'GAR-002', 'reusable', 'owned', 1, 6000, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (4, 'Copo descartável', 'Copo descartável 300 ml', 7, 'unidade', 1200, 300, 0, 2, 'DES-001', 'consumable', 'owned', 0, 20, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (5, 'Saco de lixo', 'Saco reforçado 100 L', 7, 'unidade', 80, 20, 0, 2, 'DES-002', 'consumable', 'owned', 0, 150, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (6, 'Guardanapo', 'Guardanapo de papel', 7, 'unidade', 1500, 500, 0, 2, 'DES-003', 'consumable', 'owned', 0, 10, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (7, 'Prato de jantar', 'Prato branco de porcelana', 5, 'unidade', 210, 100, 4, 1, 'LOU-001', 'reusable', 'owned', 1, 3500, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (8, 'Copo reutilizável', 'Copo de vidro', 5, 'unidade', 240, 100, 5, 1, 'LOU-002', 'reusable', 'owned', 1, 1800, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (9, 'Coca-Cola PET', 'Garrafa PET 2 L', 6, 'garrafa', 80, 20, 0, 3, 'BEB-001', 'consumable', 'owned', 0, 1200, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (10, 'Guaraná PET', 'Garrafa PET 2 L', 6, 'garrafa', 50, 15, 0, 3, 'BEB-002', 'consumable', 'owned', 0, 1000, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (11, 'Suco de laranja', 'Caixa de suco 1 L', 6, 'caixa', 60, 15, 0, 3, 'BEB-003', 'consumable', 'owned', 0, 900, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (12, 'Suco de uva', 'Caixa de suco 1 L', 6, 'caixa', 55, 15, 0, 3, 'BEB-004', 'consumable', 'owned', 0, 1000, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (13, 'Suqueira de vidro', 'Suqueira para welcome drinks', 6, 'unidade', 2, 1, 0, 3, 'BEB-005', 'reusable', 'owned', 1, 28000, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (14, 'Cuba GN 1/1', 'Cuba gastronômica de inox', 3, 'unidade', 20, 8, 1, 5, 'CUB-001', 'reusable', 'owned', 1, 13000, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (15, 'Liquidificador', 'Liquidificador industrial', 4, 'unidade', 2, 1, 0, 5, 'EQP-001', 'reusable', 'owned', 1, 45000, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (16, 'Toalha dourada', 'Toalha decorativa alugada', 12, 'unidade', 0, 0, 0, 4, 'DEC-001', 'rented', 'rented', 1, 12000, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (17, 'Vaso dourado', 'Vaso para mesa do bolo', 12, 'unidade', 12, 4, 0, 4, 'DEC-002', 'reusable', 'owned', 1, 8500, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (18, 'Bolo fake', 'Bolo cenográfico', 12, 'unidade', 1, 1, 0, 4, 'DEC-003', 'reusable', 'owned', 1, 30000, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (19, 'Bolsa de ferramentas', 'Ferramentas para montagem', 14, 'unidade', 2, 1, 0, 5, 'FER-001', 'reusable', 'owned', 1, 25000, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT OR IGNORE INTO calculation_rules (id, rule_key, name, description, category_id, calculation_type, base_value, divisor, multiplier, minimum_quantity, safety_percent, condition_json, result_inventory_item_id, priority, active, created_at, updated_at) VALUES
 (1, 'waiters', 'Garçons por convidados', 'Um garçom para cada grupo de 18 convidados.', 8, 'group_of_people', 0, 18, 1, NULL, 0, '{}', 1, 10, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (2, 'jugs_per_waiter', 'Jarras por garçom', 'Duas jarras para cada garçom.', 8, 'per_waiter', 0, 1, 2, NULL, 0, '{}', 2, 20, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (3, 'trays_per_waiter', 'Bandejas por garçom', 'Uma bandeja para cada garçom.', 8, 'per_waiter', 0, 1, 1, NULL, 0, '{}', 3, 30, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (4, 'disposable_cups', 'Copos descartáveis por convidado', 'Três copos por convidado.', 7, 'per_person', 0, 1, 3, NULL, 0, '{}', 4, 40, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (5, 'trash_bags', 'Mínimo de sacos de lixo', 'Quantidade fixa com mínimo de dez.', 7, 'fixed', 10, 1, 1, 10, 0, '{}', 5, 50, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (6, 'napkins', 'Guardanapos por convidado', 'Três guardanapos por convidado.', 7, 'per_person', 0, 1, 3, NULL, 0, '{}', 6, 60, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (7, 'dinner_plates', 'Pratos de jantar por convidado', 'Um prato por convidado; margem do evento aplicada.', 5, 'per_person', 0, 1, 1, NULL, 0, '{"use_event_safety_margin":true}', 7, 70, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (8, 'reusable_cups', 'Copos reutilizáveis por convidado', 'Um copo por convidado; margem do evento aplicada.', 5, 'per_person', 0, 1, 1, NULL, 0, '{"use_event_safety_margin":true}', 8, 80, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (9, 'soda_coke', 'Distribuição de Coca-Cola', '60% do total de refrigerantes.', 6, 'percentage_distribution', 0, 2, 1, NULL, 0, '{"distribution_group":"soda","percentage":60}', 9, 90, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (10, 'soda_guarana', 'Distribuição de Guaraná', '40% do total de refrigerantes.', 6, 'percentage_distribution', 0, 2, 1, NULL, 0, '{"distribution_group":"soda","percentage":40}', 10, 91, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (11, 'juice_orange', 'Distribuição de suco de laranja', '50% do total de sucos.', 6, 'percentage_distribution', 0, 2, 1, NULL, 0, '{"distribution_group":"juice","percentage":50}', 11, 100, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (12, 'juice_grape', 'Distribuição de suco de uva', '50% do total de sucos.', 6, 'percentage_distribution', 0, 2, 1, NULL, 0, '{"distribution_group":"juice","percentage":50}', 12, 101, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (13, 'juice_dispenser_without_welcome', 'Suqueira sem welcome drinks', 'Uma suqueira sem welcome drinks.', 6, 'fixed', 1, 1, 1, NULL, 0, '{"welcome_drinks":false}', 13, 110, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (14, 'juice_dispenser_with_welcome', 'Suqueira com welcome drinks', 'Duas suqueiras com welcome drinks.', 6, 'fixed', 2, 1, 1, NULL, 0, '{"welcome_drinks":true}', 13, 111, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT OR IGNORE INTO event_templates (id, name, description, configuration_json, active, created_at, updated_at) VALUES
 (1, 'Casamento com buffet completo', 'Buffet, welcome drinks, mesa de café e decoração.', '{"has_decoration":true,"has_welcome_drinks":true,"has_coffee_table":true}', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (2, 'Evento simples', 'Buffet sem decoração e sem welcome drinks.', '{"has_decoration":false,"has_welcome_drinks":false,"has_coffee_table":false}', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT OR IGNORE INTO events (id, template_id, client_name, name, venue, starts_at, ends_at, guest_count, has_decoration, has_welcome_drinks, has_coffee_table, starters_notes, main_courses_notes, sides_notes, beverages_notes, coffee_table_notes, cake_notes, sweets_notes, desserts_notes, notes, safety_margin_percent, status, active, created_at, updated_at) VALUES
 (1, 1, 'Íris do Campo', 'Casamento Íris do Campo', 'Espaço Jardim das Flores', strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '+14 days', '18 hours'), strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '+15 days', '2 hours'), 200, 1, 1, 1,
  'Caldinho de feijão\nBolinho de arroz\nBolinho de mandioca\nTorresmo',
  'Frango com quiabo\nVaca atolada\nRagu de calabresa',
  'Arroz branco\nArroz carreteiro\nFeijão tropeiro\nFarofa de couve\nMix de saladas',
  'Coca-Cola 60%\nGuaraná 40%\nSuco de laranja 50%\nSuco de uva 50%\nÁgua mineral',
  'Café\nMix de petit fours\nBolos\nBalas finas',
  'Doce de leite com abacaxi — 15 kg — 3 andares — requer refrigeração',
  'Mix de doces finos\nMini trufas\nCamafeus',
  'Doce de abóbora\nDoce de leite',
  'Evento de demonstração criado pela migration inicial.', 10, 'planning', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT OR IGNORE INTO menu_categories (id, name, sort_order, active, created_at, updated_at) VALUES
 (1, 'Entradas', 10, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (2, 'Pratos principais', 20, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (3, 'Acompanhamentos', 30, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (4, 'Mesa de café', 40, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (5, 'Sobremesas', 50, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT OR IGNORE INTO container_types (id, name, capacity_portions, disposable, requires_lid, is_default, transport_notes, active, created_at, updated_at) VALUES
 (1, 'Cuba GN 1/1', 40, 0, 1, 1, 'Transportar tampada em caixa térmica.', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (2, 'Bowl', 1, 0, 0, 0, 'Acondicionar em caixas divisórias.', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (3, 'Copo de sobremesa', 1, 1, 1, 0, 'Manter refrigerado.', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (4, 'Pote plástico', 10, 0, 1, 0, 'Identificar conteúdo e data.', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT OR IGNORE INTO menu_items (id, category_id, name, description, container_type_id, container_capacity_portions, active, created_at, updated_at) VALUES
 (1, 1, 'Caldinho de feijão', 'Entrada quente', 4, 10, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (2, 1, 'Bolinho de arroz', 'Entrada frita', 4, 20, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (3, 2, 'Frango com quiabo', 'Prato principal', 1, 40, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (4, 2, 'Vaca atolada', 'Prato principal', 1, 40, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (5, 3, 'Arroz branco', 'Acompanhamento', 1, 40, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (6, 3, 'Feijão tropeiro', 'Acompanhamento', 1, 40, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT OR IGNORE INTO event_menu_items (event_id, menu_item_id, portions, container_type_id, calculated_container_quantity, notes, created_at, updated_at)
 SELECT 1, id, 200, container_type_id, CEIL(200.0 / COALESCE(container_capacity_portions, 200)), '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP FROM menu_items WHERE id <= 6;

INSERT OR IGNORE INTO decorations (id, inventory_item_id, name, usage_location, color, model, ownership, rental_company, notes, active, created_at, updated_at) VALUES
 (1, 16, 'Toalha dourada', 'Mesas dos convidados', 'Dourado', 'Cetim', 'rented', 'Casa das Festas', 'Conferir manchas na retirada.', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (2, 17, 'Vaso dourado', 'Mesa do bolo', 'Dourado', 'Clássico', 'owned', '', '', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (3, 18, 'Bolo fake', 'Mesa do bolo', 'Branco e dourado', 'Três andares', 'owned', '', 'Transportar em caixa própria.', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
