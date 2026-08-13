-- Seed público mínimo: itens de demonstração para os modelos genéricos.
CREATE TEMP TABLE IF NOT EXISTS seed_menu_content(template_slug TEXT,section_slug TEXT,item_name TEXT,included INTEGER,optional INTEGER,configurable INTEGER,group_slug TEXT,sort_order INTEGER,notes TEXT);
DELETE FROM seed_menu_content;
INSERT INTO seed_menu_content VALUES
('buffet-demonstracao','entradas','Entrada quente',1,0,0,NULL,10,''),
('buffet-demonstracao','buffet-principal','Prato principal',1,0,0,NULL,10,''),
('buffet-demonstracao','acompanhamentos','Acompanhamento',1,0,0,NULL,10,''),
('buffet-com-escolhas','entradas','Mini salgado A',0,1,0,'escolha-de-entradas',10,''),
('buffet-com-escolhas','entradas','Mini salgado B',0,1,0,'escolha-de-entradas',20,''),
('buffet-com-escolhas','entradas','Mini salgado C',0,1,0,'escolha-de-entradas',30,''),
('buffet-com-escolhas','buffet-principal','Carne A',0,1,0,'escolha-de-carnes',10,''),
('buffet-com-escolhas','buffet-principal','Carne B',0,1,0,'escolha-de-carnes',20,''),
('buffet-com-escolhas','acompanhamentos','Arroz branco',1,0,0,NULL,10,'');

INSERT OR IGNORE INTO menu_categories(name,sort_order,active,created_at,updated_at)
SELECT DISTINCT section.name,100+section.id,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM seed_menu_content data JOIN menu_sections section ON section.slug=data.section_slug;

INSERT OR IGNORE INTO menu_items(category_id,name,slug,source_label,normalized_name,search_aliases,description,calculation_type,calculation_divisor,calculation_multiplier,calculation_weight,active,created_at,updated_at)
SELECT category.id,data.item_name,data.section_slug||'-'||lower(hex(data.item_name)),data.item_name,data.item_name,'',MAX(data.notes),'menu_only',1,1,1,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
FROM seed_menu_content data JOIN menu_sections section ON section.slug=data.section_slug JOIN menu_categories category ON category.name=section.name
GROUP BY category.id,data.section_slug,data.item_name;

INSERT OR IGNORE INTO menu_template_sections(menu_template_id,menu_section_id,display_name,sort_order,required,selection_min,selection_max,allow_event_changes,notes,active,created_at,updated_at)
SELECT DISTINCT model.id,section.id,section.name,section.id*10,1,0,NULL,1,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
FROM seed_menu_content data JOIN menu_templates model ON model.slug=data.template_slug JOIN menu_sections section ON section.slug=data.section_slug;

INSERT OR IGNORE INTO menu_template_items(menu_template_section_id,menu_item_id,slug,source_label,normalized_name,description,sort_order,included,optional,configurable,notes,active,created_at,updated_at)
SELECT template_section.id,item.id,lower(hex(data.item_name)),data.item_name,data.item_name,item.description,data.sort_order,data.included,data.optional,data.configurable,data.notes,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
FROM seed_menu_content data
JOIN menu_templates model ON model.slug=data.template_slug
JOIN menu_sections section ON section.slug=data.section_slug
JOIN menu_template_sections template_section ON template_section.menu_template_id=model.id AND template_section.menu_section_id=section.id
JOIN menu_categories category ON category.name=section.name
JOIN menu_items item ON item.category_id=category.id AND item.name=data.item_name;

CREATE TEMP TABLE IF NOT EXISTS seed_choice_groups(template_slug TEXT,section_slug TEXT,group_slug TEXT,group_name TEXT,min_choices INTEGER,max_choices INTEGER,required INTEGER,allow_extra INTEGER,allow_custom INTEGER,configurable INTEGER);
DELETE FROM seed_choice_groups;
INSERT INTO seed_choice_groups VALUES
('buffet-com-escolhas','entradas','escolha-de-entradas','Escolha de entradas',2,2,1,0,0,0),
('buffet-com-escolhas','buffet-principal','escolha-de-carnes','Escolha de carnes',1,1,1,0,0,0);

INSERT OR IGNORE INTO menu_choice_groups(menu_template_section_id,slug,choice_group_name,selection_min,selection_max,selection_required,allow_extra_items,allow_custom_item,configurable,sort_order,active,created_at,updated_at)
SELECT template_section.id,data.group_slug,data.group_name,data.min_choices,data.max_choices,data.required,data.allow_extra,data.allow_custom,data.configurable,10,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
FROM seed_choice_groups data JOIN menu_templates model ON model.slug=data.template_slug JOIN menu_sections section ON section.slug=data.section_slug JOIN menu_template_sections template_section ON template_section.menu_template_id=model.id AND template_section.menu_section_id=section.id;

INSERT OR IGNORE INTO menu_choice_group_items(menu_choice_group_id,menu_template_item_id,sort_order,initially_selected,created_at)
SELECT choice.id,template_item.id,data.sort_order,0,CURRENT_TIMESTAMP
FROM seed_menu_content data JOIN menu_templates model ON model.slug=data.template_slug JOIN menu_sections section ON section.slug=data.section_slug JOIN menu_template_sections template_section ON template_section.menu_template_id=model.id AND template_section.menu_section_id=section.id JOIN menu_template_items template_item ON template_item.menu_template_section_id=template_section.id AND template_item.normalized_name=data.item_name JOIN menu_choice_groups choice ON choice.menu_template_section_id=template_section.id AND choice.slug=data.group_slug
WHERE data.group_slug IS NOT NULL;

DROP TABLE seed_choice_groups;
DROP TABLE seed_menu_content;
