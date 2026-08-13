-- Seed público mínimo: serviços genéricos de demonstração.
INSERT INTO service_templates(slug,name,description,category,duration_minutes,billing_unit,configuration_json,active,current_version,source_name,source_updated_month,created_at,updated_at) VALUES
('fotografia','Fotografia','Serviço de fotografia para o evento.','imagem',NULL,'serviço','{}',1,1,'Demonstração pública','',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('totem-fotografico','Totem fotográfico','Totem com fotos e impressão na hora.','imagem',NULL,'serviço','{}',1,1,'Demonstração pública','',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
ON CONFLICT(slug) DO UPDATE SET name=excluded.name,description=excluded.description,updated_at=CURRENT_TIMESTAMP;

INSERT OR IGNORE INTO service_template_versions(service_template_id,version,change_summary,snapshot_json,created_at)
SELECT id,1,'Versão inicial de demonstração','{}',CURRENT_TIMESTAMP FROM service_templates;

INSERT OR IGNORE INTO service_components(slug,name,description,category,source_label,normalized_name,search_aliases,configurable,active,created_at,updated_at) VALUES
('component-fotos','Fotografias do evento','','entrega','Fotografias do evento','Fotografias do evento','',0,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('component-totem','Estrutura de totem','','equipamento','Estrutura de totem','Estrutura de totem','',0,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);

INSERT OR IGNORE INTO service_template_components(service_template_id,service_component_id,sort_order,included,optional,configuration_json,notes,active,created_at,updated_at)
SELECT service.id,component.id,10,1,0,'{}','',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
FROM service_templates service JOIN service_components component ON
 (service.slug='fotografia' AND component.slug='component-fotos') OR
 (service.slug='totem-fotografico' AND component.slug='component-totem');
