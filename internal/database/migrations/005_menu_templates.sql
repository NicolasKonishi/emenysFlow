CREATE TABLE IF NOT EXISTS event_template_menu_items (
    template_id INTEGER NOT NULL REFERENCES event_templates(id) ON DELETE CASCADE,
    menu_item_id INTEGER NOT NULL REFERENCES menu_items(id),
    created_at TEXT NOT NULL,
    PRIMARY KEY (template_id, menu_item_id)
);

CREATE INDEX IF NOT EXISTS idx_event_template_menu_items_item
    ON event_template_menu_items(menu_item_id, template_id);

-- O modelo completo acompanha todos os itens iniciais do catálogo.
INSERT OR IGNORE INTO event_template_menu_items(template_id, menu_item_id, created_at)
SELECT 1, m.id, CURRENT_TIMESTAMP
FROM menu_items m
WHERE m.active = 1;

-- O modelo simples serve como ponto de partida mais enxuto.
INSERT OR IGNORE INTO event_template_menu_items(template_id, menu_item_id, created_at)
SELECT 2, m.id, CURRENT_TIMESTAMP
FROM menu_items m
WHERE m.name IN ('Caldinho de feijão', 'Frango com quiabo', 'Arroz branco', 'Coca-Cola', 'Guaraná');
