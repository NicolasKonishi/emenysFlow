-- Seed público mínimo: estrutura genérica para demonstrar o sistema.
-- Catálogo completo, regras calibradas e eventos reais ficam em internal/database/seeds/private/.

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
 (1, 'Depósito principal', 'Estoque geral', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (2, 'Área de descartáveis', 'Materiais consumíveis', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (3, 'Depósito de bebidas', 'Bebidas fechadas', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (4, 'Depósito de decoração', 'Peças decorativas', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (5, 'Área de equipamentos', 'Equipamentos e utensílios', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT OR IGNORE INTO inventory_items (id, name, description, category_id, unit, stock_quantity, minimum_stock, damaged_quantity, location_id, internal_code, item_kind, ownership, requires_return, replacement_value_cents, active, created_at, updated_at) VALUES
 (1, 'Garçom', 'Profissional de atendimento', 8, 'profissional', 10, 0, 0, NULL, 'STA-001', 'outsourced', 'outsourced', 0, 0, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (2, 'Jarra', 'Jarra de serviço', 8, 'unidade', 10, 2, 0, 1, 'GAR-001', 'reusable', 'owned', 1, 4500, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (3, 'Bandeja de garçom', 'Bandeja antiderrapante', 8, 'unidade', 8, 2, 0, 1, 'GAR-002', 'reusable', 'owned', 1, 6000, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (4, 'Copo descartável', 'Copo descartável 300 ml', 7, 'unidade', 500, 100, 0, 2, 'DES-001', 'consumable', 'owned', 0, 20, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (5, 'Saco de lixo', 'Saco reforçado 100 L', 7, 'unidade', 40, 10, 0, 2, 'DES-002', 'consumable', 'owned', 0, 150, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (6, 'Guardanapo', 'Guardanapo de papel', 7, 'unidade', 800, 200, 0, 2, 'DES-003', 'consumable', 'owned', 0, 10, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (7, 'Prato de jantar', 'Prato branco', 5, 'unidade', 100, 20, 0, 1, 'LOU-001', 'reusable', 'owned', 1, 3500, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (8, 'Copo reutilizável', 'Copo de vidro', 5, 'unidade', 100, 20, 0, 1, 'LOU-002', 'reusable', 'owned', 1, 1800, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (9, 'Refrigerante PET', 'Garrafa PET 2 L', 6, 'garrafa', 40, 10, 0, 1, 'BEB-001', 'consumable', 'owned', 0, 1200, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (10, 'Suco', 'Caixa de suco 1 L', 6, 'caixa', 30, 10, 0, 1, 'BEB-003', 'consumable', 'owned', 0, 900, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (11, 'Cuba GN 1/1', 'Cuba gastronômica de inox', 3, 'unidade', 10, 2, 0, 5, 'CUB-001', 'reusable', 'owned', 1, 13000, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (15, 'Liquidificador', 'Liquidificador industrial', 4, 'unidade', 2, 1, 0, 5, 'EQP-001', 'reusable', 'owned', 1, 45000, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (12, 'Toalha decorativa', 'Toalha para mesas', 12, 'unidade', 5, 0, 0, 4, 'DEC-001', 'rented', 'rented', 1, 12000, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (13, 'Vaso decorativo', 'Vaso para mesa do bolo', 12, 'unidade', 5, 0, 0, 4, 'DEC-002', 'reusable', 'owned', 1, 8500, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (14, 'Bolo cenográfico', 'Bolo decorativo', 12, 'unidade', 1, 0, 0, 4, 'DEC-003', 'reusable', 'owned', 1, 30000, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT OR IGNORE INTO calculation_rules (id, rule_key, name, description, category_id, calculation_type, base_value, divisor, multiplier, minimum_quantity, safety_percent, condition_json, result_inventory_item_id, priority, active, created_at, updated_at) VALUES
 (1, 'waiters', 'Garçons por convidados', 'Um garçom para cada grupo de 20 convidados.', 8, 'group_of_people', 0, 20, 1, NULL, 0, '{}', 1, 10, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (2, 'jugs_per_waiter', 'Jarras por garçom', 'Duas jarras para cada garçom.', 8, 'per_waiter', 0, 1, 2, NULL, 0, '{}', 2, 20, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (3, 'trays_per_waiter', 'Bandejas por garçom', 'Uma bandeja para cada garçom.', 8, 'per_waiter', 0, 1, 1, NULL, 0, '{}', 3, 30, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (4, 'disposable_cups', 'Copos descartáveis por convidado', 'Dois copos por convidado.', 7, 'per_person', 0, 1, 2, NULL, 0, '{}', 4, 40, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (5, 'napkins', 'Guardanapos por convidado', 'Dois guardanapos por convidado.', 7, 'per_person', 0, 1, 2, NULL, 0, '{}', 6, 50, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (6, 'dinner_plates', 'Pratos por convidado', 'Um prato por convidado.', 5, 'per_person', 0, 1, 1, NULL, 0, '{}', 7, 60, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT OR IGNORE INTO event_templates (id, name, description, configuration_json, active, created_at, updated_at) VALUES
 (1, 'Evento completo', 'Buffet com decoração e welcome drinks.', '{"has_decoration":true,"has_welcome_drinks":true,"has_coffee_table":true}', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (2, 'Evento simples', 'Buffet enxuto sem extras.', '{"has_decoration":false,"has_welcome_drinks":false,"has_coffee_table":false}', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT OR IGNORE INTO events (id, template_id, client_name, name, venue, starts_at, ends_at, guest_count, has_decoration, has_welcome_drinks, has_coffee_table, notes, safety_margin_percent, status, active, created_at, updated_at) VALUES
 (1, 1, 'Cliente Demonstração', 'Evento de demonstração', 'Salão exemplo', strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '+14 days', '18 hours'), strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '+15 days', '2 hours'), 120, 1, 1, 0,
  'Evento genérico para demonstrar o fluxo do sistema.', 10, 'planning', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT OR IGNORE INTO menu_categories (id, name, sort_order, active, created_at, updated_at) VALUES
 (1, 'Entradas', 10, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (2, 'Pratos principais', 20, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (3, 'Acompanhamentos', 30, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT OR IGNORE INTO container_types (id, name, capacity_portions, disposable, requires_lid, is_default, transport_notes, active, created_at, updated_at) VALUES
 (1, 'Cuba GN 1/1', 40, 0, 1, 1, 'Transportar tampada.', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (2, 'Bowl', 1, 0, 0, 0, 'Acondicionar em caixas divisórias.', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (3, 'Copo de sobremesa', 1, 1, 1, 0, 'Manter refrigerado.', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (4, 'Pote plástico', 10, 0, 1, 0, 'Identificar conteúdo.', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT OR IGNORE INTO menu_items (id, category_id, name, description, container_type_id, container_capacity_portions, active, created_at, updated_at) VALUES
 (1, 1, 'Entrada quente', 'Entrada de demonstração', 4, 10, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (2, 2, 'Prato principal', 'Prato de demonstração', 1, 40, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (3, 3, 'Acompanhamento', 'Acompanhamento de demonstração', 1, 40, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT OR IGNORE INTO decorations (id, inventory_item_id, name, usage_location, color, model, ownership, rental_company, notes, active, created_at, updated_at) VALUES
 (1, 12, 'Toalha decorativa', 'Mesas dos convidados', 'Neutro', 'Tecido', 'rented', '', '', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (2, 13, 'Vaso decorativo', 'Mesa do bolo', 'Neutro', 'Clássico', 'owned', '', '', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
 (3, 14, 'Bolo cenográfico', 'Mesa do bolo', 'Branco', 'Decorativo', 'owned', '', '', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
