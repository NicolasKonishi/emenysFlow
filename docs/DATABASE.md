# Modelo de dados inicial

| Tabela | Responsabilidade | Integridade principal |
|---|---|---|
| `users` | Administradores e funcionários | e-mail único, perfil restrito |
| `sessions` | Sessões autenticadas | token hash único, expiração indexada |
| `events` | Briefing e ciclo de vida do evento | convidados positivos, status restrito |
| `kitchen_cooks` | Cozinheiras disponíveis para os eventos | nome e identificador únicos, ativação lógica |
| `kitchen_cook_storage_boxes` | Caixa pessoal de cada cozinheira | uma caixa ativa por cozinheira, vinculada a um item único de estoque |
| `kitchen_cook_box_items` | Conteúdo armazenado em cada caixa | item único por caixa, quantidade positiva, observação e exclusão lógica |
| `event_status_history` | Linha do tempo do status | referência obrigatória ao evento |
| `event_templates` | Modelos reutilizáveis | nome único, configuração JSON |
| `event_template_menu_items` | Itens exclusivos dos cardápios-modelo | combinação única de modelo e item; cada vínculo aponta para uma cópia própria |
| `menu_templates` / `menu_template_versions` | Biblioteca versionada de cardápios | slug único, fonte, versão e exclusão lógica |
| `menu_template_sections` / `menu_template_items` | Composição editável dos modelos | CRUD independente, ordem, obrigatoriedade e exclusão lógica |
| `menu_choice_groups` / `menu_choice_group_items` | Regras e opções de escolha por seção | mínimo, máximo opcional e personalização |
| `service_templates` / `service_template_versions` | Biblioteca versionada de serviços | preços inicialmente nulos e fonte rastreável |
| `service_template_components` | Componentes incluídos ou opcionais | CRUD, configuração por serviço e vínculo ao estoque |
| `service_component_inventory_links` | Materiais exigidos por componente do serviço | item, fórmula, propriedade e logística |
| `event_menu_templates` / `event_menu_sections` / `event_menu_snapshot_items` | Snapshot do cardápio aplicado | versão, escolhas, itens extras, porções e recipientes preservados |
| `event_services` / `event_service_components` | Snapshot dos serviços aplicados | versão, duração, fornecedor e componentes preservados |
| `event_service_component_inventory_links` | Snapshot dos materiais de cada serviço | requisitos imutáveis para a checklist da festa |
| `inventory_categories` | Categorias da checklist/estoque | nome único, ordenação |
| `inventory_locations` | Endereços físicos no estoque | nome único |
| `inventory_items` | Catálogo e saldo físico | código único, quantidades não negativas |
| `inventory_movements` | Livro de movimentações | tipo restrito, saldo anterior/novo |
| `inventory_reservations` | Bloqueio por evento e período | janela indexada, reserva ativa única |
| `calculation_rules` | Parâmetros do motor | chave única, divisor positivo |
| `checklists` | Versão da checklist do evento | uma checklist por evento |
| `checklist_items` | Itens calculados/manuais | origem única por checklist, status restrito |
| `menu_categories` | Entrada, principal, acompanhamento etc. | nome único |
| `menu_items` | Pratos e seus recipientes | nome único por categoria |
| `menu_item_ingredients` | Ficha de receita de cada prato | ingrediente único por prato, cálculo fixo, proporcional ou por grupos |
| `event_menu_items` | Cardápio selecionado no evento | item único por evento |
| `container_types` | Recipientes e capacidades | nome único |
| `equipment` | Equipamento controlado em estoque | item de estoque único |
| `menu_item_equipment` | Equipamentos por prato | chave composta |
| `decorations` | Catálogo decorativo | propriedade e dados de aluguel |
| `event_decorations` | Decoração por evento | peça única por evento |
| `beverages` | Catálogo de bebidas | nome único |
| `event_beverages` | Sabores, percentuais e overrides | bebida única por evento |
| `staff_rules` | Proporção de equipe | nome único |
| `rental_items` | Retiradas/devoluções de aluguel | evento obrigatório |
| `return_inspections` | Conferência pós-evento | item único por evento |
| `audit_log` | Histórico técnico de alterações | entidade, ação e instante indexáveis |
| `event_operations` | Responsáveis e dados de cada etapa operacional | uma ocorrência por evento/etapa |
| `event_share_tokens` | Links públicos somente leitura | token armazenado como hash |
| `decoration_templates` | Modelos reutilizáveis de decoração | nome único e exclusão lógica |
| `decoration_template_items` | Peças pertencentes ao modelo | chave composta |

## Convenções

- Chaves primárias são inteiros autoincrementais.
- Datas são textos RFC 3339 em UTC; o seed aceita o formato nativo do SQLite e o leitor trata ambos.
- Booleanos usam `INTEGER` com `CHECK (0, 1)`.
- Valores monetários usam centavos inteiros.
- Quantidades usam `REAL` para suportar peso, volume e unidades fracionárias.
- `active` implementa exclusão lógica nos cadastros principais.
- Foreign keys são ativadas na conexão e também declaradas na migration.
