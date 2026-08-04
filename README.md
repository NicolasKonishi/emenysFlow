# emenysFlow

Sistema web para planejar eventos de buffet, calcular necessidades, reservar estoque e acompanhar a checklist operacional. A interface está em português do Brasil; o código, as migrations e os nomes técnicos estão em inglês.

O projeto implementa o fluxo funcional completo, do cadastro e cálculo até o retorno e a atualização final do estoque.

## O que já funciona

- autenticação simples com sessões persistidas e senha protegida por PBKDF2;
- dashboard com próximos eventos e alertas operacionais;
- cadastro, edição, duplicação e cancelamento de eventos;
- cadastro, edição, busca, filtro e desativação lógica de itens de estoque;
- cadastro, edição e ativação de regras de cálculo;
- cardápio normalizado e administrável, com inclusão, edição, desativação, recipientes, capacidades, panelas, transporte e equipamentos;
- cardápios-modelo reutilizáveis, escolhidos ao criar a festa para preencher automaticamente todas as seções e itens;
- serviços-modelo selecionáveis no evento, com componentes, duração, configuração e materiais operacionais;
- CRUD independente de seções, itens, grupos de escolha e componentes, com criação, edição, remoção lógica e duplicação;
- biblioteca versionada em **Modelos**, com 14 cardápios e 11 serviços carregados do banco;
- seções e grupos de escolha configuráveis, blocos compartilhados e regras de mínimo/máximo;
- snapshot versionado por evento: a festa preserva o cardápio, os serviços e seus materiais mesmo após o modelo original mudar;
- comparação manual com a versão atual do cardápio e reaplicação opcional, preservando escolhas e personalizações da festa;
- personalização por evento com CRUD independente de itens, seções, porções, recipientes de serviço/transporte/bolo e equipamentos, sem alterar o modelo original do buffet;
- seleção opcional da cozinheira responsável (Cris, Suelem ou Geriane), incluindo automaticamente sua caixa pessoal na checklist;
- caixa da cozinheira abrível com CRUD do conteúdo e da quantidade armazenada; utensílios e temperos guardados viajam dentro da caixa e deixam de aparecer soltos na checklist;
- conteúdos das caixas ficam ocultos da listagem geral de estoque e são administrados exclusivamente na área das cozinheiras;
- descartáveis de evento calculados automaticamente por convidados, incluindo a quantidade especial de palitos quando houver welcome drinks ou dadinho de tapioca;
- fichas de receita por prato, com ingredientes fixos, proporcionais ou por grupos de pessoas alimentando automaticamente a checklist;
- seleção múltipla de entradas, pratos principais, acompanhamentos e bebidas diretamente no formulário do evento;
- seleção de decoração por evento, condicionada à opção de decoração;
- geração e recálculo idempotente da checklist;
- cálculos de garçons, jarras, bandejas, descartáveis, louças, bebidas e welcome drinks;
- distribuição percentual apenas entre os sabores selecionados, com pesos configuráveis e garantia de fechamento do total;
- margem de segurança configurável por regra ou evento;
- alteração manual por evento, com motivo e preservação em novos cálculos;
- disponibilidade considerando estoque, itens danificados e reservas conflitantes;
- confirmação explícita para reservar um evento com faltas;
- fluxo operacional simplificado em Separação, Carregamento e Faltando, com quantidades parciais, responsáveis, prazos e resolução por compra/aluguel/substituição;
- registro de responsáveis, horários, veículo, observações e fotos por URL;
- retorno com danos, perdas, lavagem, manutenção e atualização transacional do estoque;
- entradas, saídas, ajustes, danos, perdas e histórico de movimentações;
- visualizador de PDF integrado ao navegador, com download opcional e compartilhamento do arquivo pelo menu do aparelho/WhatsApp;
- planilha CSV, impressão e link público somente leitura;
- usuários administradores e funcionários, categorias e localizações;
- fórmulas personalizadas seguras com `guests`, `waiters` e `safety_margin`;
- PWA operacional offline com IndexedDB, fila idempotente, fotos pendentes, conflitos, sincronização automática/manual, indicadores e aviso de nova versão;
- histórico auditável de cada recálculo, com versões e resumo das alterações;
- migrations e dados de demonstração idempotentes;
- testes unitários e de integração das principais regras.

## Requisitos

- Go 1.26 ou superior.
- Não é necessário instalar SQLite separadamente; o driver utilizado é implementado em Go.

## Executar

```powershell
go mod download
go run ./cmd/server
```

Abra [http://localhost:8080](http://localhost:8080).

Credenciais de demonstração:

- e-mail: `admin@buffet.local`
- senha: `admin123`

O banco é criado em `data/buffetflow.db`. Para usar outro arquivo ou porta:

```powershell
go run ./cmd/server -database C:\dados\buffet.db -address :9090
```

## Migrations

As migrations SQL são carregadas da pasta do projeto e executadas automaticamente em ordem. Cada arquivo aplicado é registrado em `schema_migrations`.

Aplicar migrations e sair:

```powershell
go run ./cmd/server -migrate-only
```

Para criar uma migration, adicione um arquivo numerado em `internal/database/migrations`, por exemplo `003_event_returns.sql`. Migrations aplicadas nunca devem ser editadas; crie uma nova migration para alterar o esquema.

## Testes

```powershell
go test ./...
```

Com relatório de cobertura:

```powershell
go test -coverprofile coverage.out ./...
go tool cover -func coverage.out
```

## Dados iniciais

O seed inclui:

- evento `BUFFET — Íris do Campo — 200 pessoas` para daqui a aproximadamente 14 dias;
- cardápio com entradas, pratos principais e acompanhamentos;
- 14 cardápios-modelo e 11 serviços-modelo, prontos para seleção e personalização por evento;
- bebidas distribuídas entre Coca-Cola, Guaraná, suco de laranja e suco de uva;
- três itens de decoração;
- localizações e itens iniciais de estoque, incluindo cubas, panelas e utensílios;
- 14 regras de cálculo editáveis;
- modelos iniciais de evento e tipos de recipiente.

Para reiniciar somente o ambiente local de demonstração, pare a aplicação, remova manualmente `data/buffetflow.db` e execute o servidor novamente. Essa operação apaga todos os dados locais.

## Arquitetura

O projeto é um monólito modular. Essa escolha reduz a complexidade operacional e mantém limites claros para uma futura extração de módulos, se o volume exigir.

```text
cmd/server/                    inicialização e servidor HTTP
internal/database/             conexão, runner e migrations SQL
internal/handlers/             rotas, validação HTTP e view models
internal/models/               entidades do domínio
internal/repositories/         consultas parametrizadas e transações
internal/services/             autenticação e regras de negócio
internal/templates/            templates HTML renderizados pelo backend
web/static/                    CSS, JavaScript, manifest e service worker
docs/                          decisões de arquitetura e modelo de dados
data/                          banco SQLite local (ignorado pelo Git)
```

O fluxo principal é:

```text
HTTP → handler → service → repository → SQLite
                 ↓
            cálculo puro
```

Detalhes de arquitetura, telas, regras e decisões estão em [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md). O inventário inicial de tabelas está em [docs/DATABASE.md](docs/DATABASE.md).

## Decisões de implementação

- `net/http` e `html/template` evitam um framework web desnecessário.
- HTMX melhora navegação e formulários sem transformar o frontend em uma SPA.
- Datas são persistidas em UTC e exibidas em `America/Sao_Paulo` por padrão.
- Quantidades usam `REAL` para aceitar unidades fracionárias; valores monetários usam centavos inteiros.
- Exclusão lógica é a operação normal para itens já utilizados.
- Regras guardam parâmetros e condições no banco; o algoritmo interpreta os tipos de cálculo.
- O cardápio possui tabelas normalizadas e mantém os campos textuais do briefing como apoio operacional.
- Modelos são versionados; ao aplicá-los, o evento recebe snapshots próprios para evitar alterações retroativas.
- O PWA usa Cache First no app shell e Network First nos dados. Eventos previamente sincronizados permanecem consultáveis; ações operacionais offline entram numa fila idempotente com controle otimista de versão.
- A cópia offline não armazena senha. Ela expira em 12 horas sem uma nova validação online e é removida junto com os caches no logout.

## Preparação para produção

O núcleo funcional está concluído. Para operação pública, configure HTTPS, backup automatizado do arquivo SQLite, credenciais próprias, armazenamento externo de fotos e monitoramento. Operações administrativas globais e exclusões destrutivas continuam exigindo conexão; o modo offline é destinado à operação dos eventos.

## Segurança

Consultas usam parâmetros, operações críticas de estoque usam transações, cookies são `HttpOnly` e `SameSite=Strict`, respostas incluem cabeçalhos de segurança e alterações administrativas verificam o perfil do usuário. Antes de produção, altere a credencial inicial, use HTTPS e implemente rotação/recuperação de senha e proteção CSRF com token caso a aplicação precise aceitar origens adicionais.
