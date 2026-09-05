# emenysFlow

Sistema web para **planejar, calcular e operar eventos de buffet** — **modo online** com o sistema completo, e **modo offline** com checklists salvas e organizador de layout. Você escolhe a área depois do login; se a conexão cair, o modo offline abre sozinho.

Projeto pessoal que demonstra backend em Go, domínio rico, PWA offline e um fluxo operacional completo para equipes de eventos. A interface está em português; código, migrations e nomes técnicos em inglês.

> **Repositório público:** o código e a arquitetura estão aqui. Cardápios completos, serviços terceirizados, estoque real e regras calibradas da operação ficam em seeds locais não versionados — veja [docs/PUBLICACAO.md](docs/PUBLICACAO.md).

---

## Destaques técnicos

| Área | O que foi feito |
|------|-----------------|
| **Backend** | Monólito modular em Go (`net/http`, `html/template`, SQLite) |
| **Domínio** | Motor de cálculo com regras configuráveis, distribuição percentual, margens e overrides por evento |
| **Cardápio** | Modelos versionados, snapshots por evento, grupos de escolha, receitas e personalização sem alterar o modelo original |
| **Operação** | Checklist idempotente, reserva de estoque, separação, carregamento mobile, retorno transacional |
| **Offline** | Área própria com checklists salvas, layout das festas, IndexedDB e sync opcional |
| **Layout** | Organizador de planta do salão (mesas, áreas, equipamentos) por evento ou avulso |
| **Qualidade** | Testes unitários e de integração nas regras críticas; migrations SQL versionadas |

---

## Fluxo operacional

```text
Online: Evento → Cálculo → Reserva → Estoque
Offline: Checklist salva → Separação → Carregamento → Layout do salão → sync opcional
```

O sistema cobre:

- autenticação com sessões e senha PBKDF2;
- dashboard com próximos eventos e alertas;
- eventos com cardápio-modelo, serviços terceirizados e decoração;
- estoque com reservas conflitantes, movimentações e caixas de cozinheiras;
- regras de cálculo editáveis (garçons, descartáveis, bebidas, equipamentos);
- checklist recalculável com ajustes manuais auditáveis;
- PDF, CSV, impressão e link público somente leitura;
- escolha entre modo online (sistema completo) e modo offline (checklists e layout);
- modo offline automático, limitado a checklists salvas e layout, quando o serviço some;
- aviso para reabrir o online assim que a conexão volta.

Detalhes em [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) e [docs/OFFLINE.md](docs/OFFLINE.md).

---

## Stack

- **Go 1.26** — servidor HTTP, regras de negócio, acesso a dados
- **SQLite** (driver puro Go) — persistência local, WAL, foreign keys
- **HTMX** — navegação e formulários sem SPA
- **PWA** — service worker, manifest, operação offline

---

## Executar (demonstração pública)

```powershell
go mod download
go run ./cmd/server
```

Abra [http://localhost:8080](http://localhost:8080).

| Campo | Valor |
|-------|-------|
| E-mail | `admin@buffet.local` |
| Senha | `admin123` |

O seed público traz um evento genérico, dois cardápios-modelo e dois serviços de exemplo — suficiente para explorar telas e fluxos, sem expor dados da operação real.

Banco local: `data/buffetflow.db` (ignorado pelo Git).

```powershell
go run ./cmd/server -database C:\caminho\outro.db -address :9090
```

### Ambiente completo (local)

Para carregar cardápios, serviços e estoque reais, coloque os arquivos SQL em `internal/database/seeds/private/` (veja o README dessa pasta). O servidor aplica esses seeds automaticamente após as migrations.

---

## Testes

```powershell
go test ./...
```

Com seeds privados completos (validação do catálogo real):

```powershell
go test -tags private_seeds ./internal/database/...
```

---

## Estrutura

```text
cmd/server/                 entrada e servidor HTTP
internal/handlers/          rotas, validação e view models
internal/services/          autenticação e motor de checklist/cálculo
internal/repositories/      SQL parametrizado e transações
internal/models/            entidades de domínio
internal/database/          conexão, migrations e seeds
internal/templates/         HTML renderizado no servidor
web/static/                 CSS, JS, PWA e layout editor
docs/                       arquitetura, offline e publicação
```

---

## Sobre este repositório

Este projeto foi desenvolvido para uma operação real de buffet. O código mostra **como** o sistema foi construído — camadas, regras, testes e decisões — enquanto os **dados operacionais** (cardápios completos, preços, nomes da equipe, acervo de decoração) permanecem fora do Git.

Se você é recrutador ou colega de desenvolvimento: clone, rode os testes, navegue pelo fluxo de evento e leia a arquitetura. Isso reflete o escopo real do trabalho.

---

## Licença

Código fonte disponível para **consulta e portfólio**. Os dados de negócio e seeds privados não estão incluídos. Uso comercial ou redistribuição do catálogo operacional não é permitido sem autorização.

---

**Nicolas Konishi** — desenvolvimento full-stack, domínio de eventos e operação offline.
