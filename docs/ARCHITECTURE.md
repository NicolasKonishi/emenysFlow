# Arquitetura e regras do sistema

## 1. Arquitetura escolhida

O emenysFlow usa um monólito modular em Go. O processo HTTP, o motor de cálculo, o acesso a dados e a renderização HTML ficam no mesmo binário, mas separados por pacotes.

- **Handlers** convertem HTTP em comandos, fazem validação de formato e escolhem a resposta.
- **Services** executam cálculos e coordenam casos de uso.
- **Repositories** concentram SQL parametrizado e transações.
- **Models** representam o domínio sem depender da camada HTTP.
- **Templates** renderizam a interface no servidor; HTMX substitui somente o conteúdo principal durante a navegação.
- **SQLite** usa foreign keys, WAL, busy timeout e apenas uma conexão de escrita para reduzir contenção.

Essa estrutura permite testar os cálculos sem banco e testar reservas/checklists com um SQLite temporário real.

## 2. Estrutura de pastas

```text
cmd/server
internal/handlers
internal/services
internal/repositories
internal/models
internal/database/migrations
internal/templates/pages
web/static/css
web/static/js
docs
data
```

## 3. Modelo inicial do banco

O modelo separa seis áreas:

1. **Acesso:** `users`, `sessions`.
2. **Eventos e cardápio:** `events`, `event_status_history`, `event_templates`, `menu_categories`, `menu_items`, `event_menu_items`.
3. **Catálogo e estoque:** `inventory_categories`, `inventory_locations`, `inventory_items`, `inventory_movements`, `inventory_reservations`.
4. **Cálculo e operação:** `calculation_rules`, `checklists`, `checklist_items`, `equipment`, `menu_item_equipment`, `container_types`, `staff_rules`.
5. **Extensões operacionais:** `decorations`, `event_decorations`, `beverages`, `event_beverages`, `rental_items`, `return_inspections`, `audit_log`.
6. **Modelos versionados:** `menu_templates`, `service_templates`, suas versões, seções, itens, grupos, componentes e snapshots aplicados ao evento.

Todas as relações críticas usam foreign keys. Eventos, itens, regras e cadastros reutilizáveis aceitam desativação lógica. Movimentações, histórico e auditoria são registros append-only por intenção.

## 4. Principais regras de negócio

- A quantidade é calculada por regras ativas, em ordem de prioridade.
- `group_of_people` arredonda para cima antes de aplicar o multiplicador.
- Margens são aplicadas antes do arredondamento final.
- Distribuições percentuais usam o método dos maiores restos; a soma sempre fecha no total calculado.
- A quantidade de garçons alterada no evento alimenta também jarras e bandejas sem modificar a regra global.
- Quando a capacidade da cuba ou recipiente não existe, a função de capacidade retorna uma unidade por item.
- A disponibilidade é `estoque - reservas conflitantes de outros eventos - danificados`, limitada a zero na camada de cálculo.
- Janelas são conflitantes quando se sobrepõem; eventos que terminam exatamente no início do seguinte não conflitam.
- O recálculo identifica itens automáticos pela origem da regra e não cria duplicatas.
- Itens manuais são preservados. Overrides mantêm a quantidade ajustada e recebem apenas a nova quantidade calculada como referência.
- Um evento com faltas exige confirmação explícita antes da reserva.
- Cancelar um evento cancela suas reservas ativas e registra histórico de status.
- Consumíveis não devem ser aguardados no retorno; o esquema já carrega `item_kind` e `requires_return` para a etapa operacional.

## 5. Fluxo das telas

```text
Login
  └─ Visão geral
      ├─ Eventos
      │   ├─ Novo / editar
      │   │   ├─ Escolher cardápio e serviços
      │   │   ├─ Personalizar itens, porções e recipientes
      │   │   └─ Comparar / reaplicar nova versão do cardápio
      │   ├─ Duplicar / cancelar
      │   └─ Checklist
      │       ├─ Recalcular
      │       ├─ Ajustar quantidade
      │       ├─ Marcar separado
      │       └─ Reservar estoque
      ├─ Estoque
      │   ├─ Buscar / filtrar
      │   ├─ Novo / editar / desativar
      │   └─ Caixas das cozinheiras
      │       └─ Abrir / adicionar / editar / remover conteúdo
      ├─ Regras
      │   └─ Nova / editar / ativar
      └─ Modelos
          ├─ Cardápios: metadados, seções, itens e escolhas
          └─ Serviços: metadados e componentes
```

No celular, as quatro áreas principais ficam na barra inferior e a criação de evento recebe destaque central.

## 6. O que permanece configurável

- divisor, multiplicador, valor base, mínimo, máximo e margem das regras;
- prioridade e ativação de cada regra;
- condições de welcome drinks, decoração e margem do evento;
- grupos e percentuais de distribuição de bebidas;
- item e categoria resultantes;
- margem e quantidade de garçons por evento;
- cozinheira responsável por evento, podendo ser definida ou alterada depois;
- conteúdo e quantidade de utensílios e temperos armazenados na caixa pessoal de cada cozinheira;
- proporções dos descartáveis por grupos de convidados e condição especial de palitos para welcome drinks ou dadinho de tapioca;
- ingredientes e proporções da ficha de receita de cada prato;
- estoque, mínimo, danos, localização, tipo e necessidade de retorno;
- tipos/capacidades de recipientes e relações de cardápio já modeladas no banco;
- modelos de evento e itens de decoração;
- cardápios e serviços reutilizáveis, suas seções, componentes, regras de escolha e estado ativo.

Os valores iniciais do briefing existem apenas na migration de seed e podem ser alterados pela interface de regras.

## Fluxo operacional concluído

O evento percorre planejamento, reserva, separação, conferência, carregamento, evento em andamento, retorno, conferência pós-evento e finalização. O estoque físico não é reduzido ao reservar: a disponibilidade considera reservas ativas. Na finalização, perdas reduzem o saldo, danos aumentam a quantidade indisponível e as reservas são liberadas em uma única transação.

Cardápios e decoração geram requisitos automáticos com chaves estáveis. Recipientes são somados conforme capacidade, panelas são vinculadas por prato e equipamentos compartilhados usam a maior quantidade obrigatória, evitando duplicação.

Ao selecionar um cardápio ou serviço, o evento recebe uma cópia versionada. Alterações posteriores no modelo não afetam festas existentes. A interface indica quando há uma versão nova do cardápio; a reaplicação é sempre manual e preserva escolhas válidas, itens personalizados, porções e recipientes. Os vínculos de estoque dos serviços também são copiados para o evento, incluindo fórmulas como quantidade por convidado.

Cada cozinheira possui uma única “Caixa da cozinheira” vinculada ao estoque, reunindo seus utensílios e temperos. A caixa pode ser aberta na área específica das cozinheiras e tem seu próprio conteúdo editável: é possível adicionar um item existente, alterar quantidade e observação, remover e reativar o item ao adicioná-lo novamente. A caixa física e todos os itens armazenados em alguma caixa ficam ocultos da listagem e dos indicadores gerais de estoque, embora continuem disponíveis para regras, catálogos e seletores internos. Um item removido de todas as caixas volta automaticamente ao estoque geral. A cozinheira é opcional no planejamento do evento; a geração da checklist inclui somente a caixa da responsável selecionada. Itens armazenados, como colher de serviço, concha e temperos, deixam de aparecer soltos na checklist, mas voltam a aparecer individualmente se forem removidos da caixa. Alterar ou remover a seleção e recalcular substitui ou retira automaticamente essa caixa.

No carregamento da van, a versão mobile troca a tabela operacional por cartões grandes com somente duas decisões: “Concluído” ou “Falta”, com a quantidade faltante. Cada decisão é salva imediatamente.

Os descartáveis operacionais são requisitos globais do evento e usam arredondamento para cima por grupo de convidados. Copos são calculados diretamente por pessoa. Palitos de dente usam regras mutuamente exclusivas: uma caixa grande por grupo de 100 em eventos comuns e duas caixas por grupo de 100 quando houver welcome drinks ou um item selecionado chamado “Dadinho de tapioca”. O motor verifica tanto o cardápio clássico quanto o snapshot versionado e itens personalizados do evento.

Cada item do cardápio possui uma ficha de receita administrativa. Ingredientes proporcionais calculam a quantidade por porções com duas casas decimais; ingredientes por grupos arredondam o número de grupos para cima; ingredientes fixos entram uma única vez se qualquer prato que os exige estiver selecionado. Exemplo inicial: costela suína a 1 kg para cada 3 pessoas, frango a 1 kg para cada 2 pessoas, barbecue para costelinha e ketchup/mostarda para estrogonofe. Ingredientes iguais de vários pratos são agregados em uma única linha da checklist. Ao copiar um prato para outro modelo, a receita é copiada junto.

O PDF abre em um visualizador incorporado ao sistema, sem download obrigatório. A mesma tela oferece download explícito e compartilhamento do arquivo pelo recurso nativo do aparelho, permitindo selecionar o WhatsApp; navegadores sem compartilhamento de arquivos usam download e abertura do WhatsApp como alternativa. Também estão disponíveis CSV compatível com planilhas, impressão e visualização pública somente leitura com token aleatório armazenado somente como hash.
