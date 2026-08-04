-- Acervo próprio de decoração informado pela operação em agosto de 2026.
-- Itens sem contagem explícita foram registrados como um conjunto/lote editável.

UPDATE inventory_items
SET name='Toalha dourada',description='Toalha decorativa dourada própria',category_id=12,unit='unidade',stock_quantity=30,minimum_stock=0,location_id=4,item_kind='reusable',ownership='owned',requires_return=1,notes='Quantidade mínima informada: 30 unidades.',active=1,updated_at=CURRENT_TIMESTAMP
WHERE internal_code='DEC-001';

UPDATE inventory_items
SET name='Vaso dourado para mesa do bolo',description='Vaso dourado para composição da mesa do bolo',category_id=12,unit='unidade',stock_quantity=10,minimum_stock=0,location_id=4,item_kind='reusable',ownership='owned',requires_return=1,active=1,updated_at=CURRENT_TIMESTAMP
WHERE internal_code='DEC-002';

UPDATE inventory_items
SET name='Bolo fake',description='Bolo cenográfico para a mesa do bolo',category_id=12,unit='unidade',stock_quantity=10,minimum_stock=0,location_id=4,item_kind='reusable',ownership='owned',requires_return=1,active=1,updated_at=CURRENT_TIMESTAMP
WHERE internal_code='DEC-003';

INSERT OR IGNORE INTO inventory_items(name,description,category_id,subcategory,unit,stock_quantity,minimum_stock,damaged_quantity,location_id,internal_code,item_kind,ownership,requires_return,replacement_value_cents,notes,active,created_at,updated_at) VALUES
('Vaso de vidro para mesa do bolo','Vaso de vidro para composição da mesa do bolo',12,'Mesa do bolo','unidade',10,0,0,4,'DEC-004','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Vaso de porcelana para mesa do bolo','Vaso de porcelana para composição da mesa do bolo',12,'Mesa do bolo','unidade',10,0,0,4,'DEC-005','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Bandeja dourada para doces','Bandeja dourada para servir doces na mesa principal',12,'Mesa do bolo','unidade',10,0,0,4,'DEC-006','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Vaso dourado para mesa dos convidados','Vaso dourado para composição das mesas dos convidados',12,'Mesas dos convidados','unidade',10,0,0,4,'DEC-007','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Lanterna de madeira','Lanterna de madeira para composição decorativa',12,'Composição geral','unidade',6,0,0,4,'DEC-008','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Lanterna branca','Lanterna branca para composição decorativa',12,'Composição geral','unidade',12,0,0,4,'DEC-009','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Samambaia','Samambaia para composição decorativa',12,'Composição geral','unidade',6,0,0,4,'DEC-010','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Sousplat de rattan','Sousplat de rattan para mesa dos convidados',12,'Mesas dos convidados','unidade',180,0,0,4,'DEC-011','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Sousplat dourado','Sousplat dourado para mesa dos convidados',12,'Mesas dos convidados','unidade',20,0,0,4,'DEC-012','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Sousplat verde','Sousplat verde para mesa dos convidados',12,'Mesas dos convidados','unidade',20,0,0,4,'DEC-013','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Lounge para decoração','Conjunto de lounge para decoração',12,'Lounge','conjunto',1,0,0,4,'DEC-014','reusable','owned',1,0,'Composição: 1 sofá, 1 tapete, 1 balança, 2 poltronas e 1 mesa de apoio.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Varal de luzes','Varal de luzes para composição decorativa',12,'Composição geral','conjunto',1,0,0,4,'DEC-015','reusable','owned',1,0,'Quantidade não informada; cadastrado como um conjunto editável.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Arranjo permanente','Arranjo floral permanente para decoração',12,'Composição floral','unidade',20,0,0,4,'DEC-016','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Flores azuis permanentes','Lote de flores azuis permanentes',12,'Composição floral','lote',1,0,0,4,'DEC-017','reusable','owned',1,0,'Quantidade não informada; cadastrado como um lote editável.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Flores vermelhas permanentes','Lote de flores vermelhas permanentes',12,'Composição floral','lote',1,0,0,4,'DEC-018','reusable','owned',1,0,'Quantidade não informada; cadastrado como um lote editável.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Flores rosas permanentes','Lote de flores rosas permanentes',12,'Composição floral','lote',1,0,0,4,'DEC-019','reusable','owned',1,0,'Quantidade não informada; cadastrado como um lote editável.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Flores laranjas permanentes','Lote de flores laranjas permanentes',12,'Composição floral','lote',1,0,0,4,'DEC-020','reusable','owned',1,0,'Quantidade não informada; cadastrado como um lote editável.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Toalha verde','Toalha decorativa verde própria',12,'Mesas dos convidados','unidade',30,0,0,4,'DEC-021','reusable','owned',1,0,'Quantidade mínima informada: 30 unidades.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Toalha azul','Toalha decorativa azul própria',12,'Mesas dos convidados','unidade',30,0,0,4,'DEC-022','reusable','owned',1,0,'Quantidade mínima informada: 30 unidades.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Toalha rosa','Toalha decorativa rosa própria',12,'Mesas dos convidados','unidade',30,0,0,4,'DEC-023','reusable','owned',1,0,'Quantidade mínima informada: 30 unidades.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Toalha branca','Toalha decorativa branca própria',12,'Mesas dos convidados','unidade',30,0,0,4,'DEC-024','reusable','owned',1,0,'Quantidade mínima informada: 30 unidades.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Toalha pérola','Toalha decorativa pérola própria',12,'Mesas dos convidados','unidade',30,0,0,4,'DEC-025','reusable','owned',1,0,'Quantidade mínima informada: 30 unidades.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Mesa para decoração da mesa do bolo','Mesa de apoio para composição da mesa do bolo',12,'Mesa do bolo','unidade',2,0,0,4,'DEC-026','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Mesa grande para o bolo','Mesa grande destinada ao bolo',12,'Mesa do bolo','unidade',2,0,0,4,'DEC-027','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Voal com LED','Voal com iluminação LED para fundo da mesa do bolo',12,'Fundo da mesa do bolo','unidade',1,0,0,4,'DEC-028','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Boleira de vidro','Boleira de vidro para a mesa do bolo',12,'Mesa do bolo','unidade',1,0,0,4,'DEC-029','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Bola de cipó','Bola de cipó para composição decorativa',12,'Composição geral','unidade',3,0,0,4,'DEC-030','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Vaso para samambaia','Vaso de apoio para samambaias',12,'Composição geral','unidade',10,0,0,4,'DEC-031','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Mesa de cerimônia','Mesa destinada à cerimônia',12,'Cerimônia','unidade',1,0,0,4,'DEC-032','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Voal','Voal para composição e fundo decorativo',12,'Composição geral','unidade',20,0,0,4,'DEC-033','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Tapete vermelho para cerimônia','Tapete vermelho para corredor da cerimônia',12,'Cerimônia','unidade',1,0,0,4,'DEC-034','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Tapete branco para cerimônia','Tapete branco para corredor da cerimônia',12,'Cerimônia','unidade',1,0,0,4,'DEC-035','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Tapete areia de 1 m para cerimônia','Tapete areia com metragem de 1 metro',12,'Cerimônia','unidade',1,0,0,4,'DEC-036','reusable','owned',1,0,'Metragem: 1 m.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Tapete areia de 2 m para cerimônia','Tapete areia com metragem de 2 metros',12,'Cerimônia','unidade',1,0,0,4,'DEC-037','reusable','owned',1,0,'Metragem: 2 m.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Tapete preto para cerimônia','Tapete preto para corredor da cerimônia',12,'Cerimônia','unidade',1,0,0,4,'DEC-038','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Cachepô de madeira para cerimônia','Cachepô de madeira para composição da cerimônia',12,'Cerimônia','unidade',12,0,0,4,'DEC-039','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Pranchão para buffet','Pranchão para montagem do buffet',12,'Buffet','unidade',6,0,0,4,'DEC-040','reusable','owned',1,0,'',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('Fogão portátil de apoio','Fogão levado quando o espaço do evento não possui equipamento',12,'Apoio do buffet','unidade',2,0,0,4,'DEC-041','reusable','owned',1,0,'Levar somente quando o local do evento não possuir fogão.',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);

UPDATE decorations
SET name='Toalha dourada',usage_location='Mesas dos convidados',color='Dourado',model='Tecido',ownership='owned',rental_company='',notes='Quantidade mínima informada: 30 unidades.',active=1,updated_at=CURRENT_TIMESTAMP
WHERE inventory_item_id=(SELECT id FROM inventory_items WHERE internal_code='DEC-001');

UPDATE decorations
SET name='Vaso dourado para mesa do bolo',usage_location='Mesa do bolo',color='Dourado',model='Vaso',ownership='owned',rental_company='',notes='',active=1,updated_at=CURRENT_TIMESTAMP
WHERE inventory_item_id=(SELECT id FROM inventory_items WHERE internal_code='DEC-002');

UPDATE decorations
SET name='Bolo fake',usage_location='Mesa do bolo',color='Variado',model='Cenográfico',ownership='owned',rental_company='',notes='Transportar em caixa própria.',active=1,updated_at=CURRENT_TIMESTAMP
WHERE inventory_item_id=(SELECT id FROM inventory_items WHERE internal_code='DEC-003');

WITH catalog(internal_code,name,usage_location,color,model,notes) AS (VALUES
('DEC-004','Vaso de vidro para mesa do bolo','Mesa do bolo','Transparente','Vidro',''),
('DEC-005','Vaso de porcelana para mesa do bolo','Mesa do bolo','Porcelana','Porcelana',''),
('DEC-006','Bandeja dourada para doces','Mesa do bolo','Dourado','Bandeja',''),
('DEC-007','Vaso dourado para mesa dos convidados','Mesas dos convidados','Dourado','Vaso',''),
('DEC-008','Lanterna de madeira','Composição geral','Madeira','Lanterna',''),
('DEC-009','Lanterna branca','Composição geral','Branco','Lanterna',''),
('DEC-010','Samambaia','Composição geral','Verde','Planta',''),
('DEC-011','Sousplat de rattan','Mesas dos convidados','Natural','Rattan',''),
('DEC-012','Sousplat dourado','Mesas dos convidados','Dourado','Sousplat',''),
('DEC-013','Sousplat verde','Mesas dos convidados','Verde','Sousplat',''),
('DEC-014','Lounge para decoração','Lounge','Variado','Conjunto','1 sofá, 1 tapete, 1 balança, 2 poltronas e 1 mesa de apoio.'),
('DEC-015','Varal de luzes','Composição geral','Luz quente','Varal de luzes','Quantidade não informada; cadastrado como um conjunto editável.'),
('DEC-016','Arranjo permanente','Composição floral','Variado','Arranjo permanente',''),
('DEC-017','Flores azuis permanentes','Composição floral','Azul','Flores permanentes','Quantidade não informada; cadastrado como um lote editável.'),
('DEC-018','Flores vermelhas permanentes','Composição floral','Vermelho','Flores permanentes','Quantidade não informada; cadastrado como um lote editável.'),
('DEC-019','Flores rosas permanentes','Composição floral','Rosa','Flores permanentes','Quantidade não informada; cadastrado como um lote editável.'),
('DEC-020','Flores laranjas permanentes','Composição floral','Laranja','Flores permanentes','Quantidade não informada; cadastrado como um lote editável.'),
('DEC-021','Toalha verde','Mesas dos convidados','Verde','Tecido','Quantidade mínima informada: 30 unidades.'),
('DEC-022','Toalha azul','Mesas dos convidados','Azul','Tecido','Quantidade mínima informada: 30 unidades.'),
('DEC-023','Toalha rosa','Mesas dos convidados','Rosa','Tecido','Quantidade mínima informada: 30 unidades.'),
('DEC-024','Toalha branca','Mesas dos convidados','Branco','Tecido','Quantidade mínima informada: 30 unidades.'),
('DEC-025','Toalha pérola','Mesas dos convidados','Pérola','Tecido','Quantidade mínima informada: 30 unidades.'),
('DEC-026','Mesa para decoração da mesa do bolo','Mesa do bolo','Variado','Mesa de apoio',''),
('DEC-027','Mesa grande para o bolo','Mesa do bolo','Variado','Mesa grande',''),
('DEC-028','Voal com LED','Fundo da mesa do bolo','Variado','Voal com LED',''),
('DEC-029','Boleira de vidro','Mesa do bolo','Transparente','Vidro',''),
('DEC-030','Bola de cipó','Composição geral','Natural','Cipó',''),
('DEC-031','Vaso para samambaia','Composição geral','Variado','Vaso',''),
('DEC-032','Mesa de cerimônia','Cerimônia','Variado','Mesa',''),
('DEC-033','Voal','Composição geral','Variado','Voal',''),
('DEC-034','Tapete vermelho para cerimônia','Cerimônia','Vermelho','Tapete',''),
('DEC-035','Tapete branco para cerimônia','Cerimônia','Branco','Tapete',''),
('DEC-036','Tapete areia de 1 m para cerimônia','Cerimônia','Areia','Tapete de 1 m','Metragem: 1 m.'),
('DEC-037','Tapete areia de 2 m para cerimônia','Cerimônia','Areia','Tapete de 2 m','Metragem: 2 m.'),
('DEC-038','Tapete preto para cerimônia','Cerimônia','Preto','Tapete',''),
('DEC-039','Cachepô de madeira para cerimônia','Cerimônia','Madeira','Cachepô',''),
('DEC-040','Pranchão para buffet','Buffet','Madeira','Pranchão',''),
('DEC-041','Fogão portátil de apoio','Apoio do buffet','Inox','Fogão','Levar somente quando o local do evento não possuir fogão.')
)
INSERT INTO decorations(inventory_item_id,name,usage_location,color,model,ownership,rental_company,notes,active,created_at,updated_at)
SELECT inventory.id,catalog.name,catalog.usage_location,catalog.color,catalog.model,'owned','',catalog.notes,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
FROM catalog JOIN inventory_items inventory ON inventory.internal_code=catalog.internal_code
WHERE NOT EXISTS(SELECT 1 FROM decorations decoration WHERE decoration.inventory_item_id=inventory.id);
