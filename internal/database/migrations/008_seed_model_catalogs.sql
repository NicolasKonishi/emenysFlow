-- Seed público mínimo: um cardápio-modelo genérico.
INSERT INTO menu_templates(slug,name,description,menu_type,active,current_version,source_name,source_updated_month,created_at,updated_at)
VALUES
('buffet-demonstracao','Buffet Demonstração','Modelo genérico com entradas, prato principal e acompanhamentos.','buffet',1,1,'Demonstração pública','',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('buffet-com-escolhas','Buffet com Escolhas','Modelo com grupos de escolha configuráveis.','buffet',1,1,'Demonstração pública','',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
ON CONFLICT(slug) DO UPDATE SET name=excluded.name,description=excluded.description,updated_at=CURRENT_TIMESTAMP;

INSERT OR IGNORE INTO menu_template_versions(menu_template_id,version,change_summary,snapshot_json,created_at)
SELECT id,1,'Versão inicial de demonstração','{}',CURRENT_TIMESTAMP FROM menu_templates;

INSERT INTO menu_sections(slug,name,section_type,active,created_at,updated_at) VALUES
('entradas','Entradas','food',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('buffet-principal','Buffet principal','food',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('acompanhamentos','Acompanhamentos','food',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('bebidas','Bebidas','beverage',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
ON CONFLICT(slug) DO UPDATE SET name=excluded.name,section_type=excluded.section_type,updated_at=CURRENT_TIMESTAMP;

INSERT OR IGNORE INTO menu_shared_blocks(slug,name,description,active,created_at,updated_at) VALUES
('bebidas-padrao','Bebidas padrão','Bebidas calculadas pelas regras do evento.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
