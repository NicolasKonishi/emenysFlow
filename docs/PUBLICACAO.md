# Publicar no GitHub sem expor a operação

Este guia explica o que vai (e o que **não** vai) para o repositório público.

## O que fica público

- Todo o código Go, templates, JavaScript e CSS
- Migrations de **esquema** (tabelas, índices, colunas)
- Seeds **mínimos** em `internal/database/migrations/` — dados genéricos só para demonstrar telas
- Documentação de arquitetura e testes automatizados

## O que fica privado (local)

Arquivos em `internal/database/seeds/private/*.sql` — **gitignored**:

| Conteúdo sensível | Por quê |
|-------------------|---------|
| Cardápios-modelo completos (14+ buffets) | Catálogo comercial da casa |
| Serviços terceirizados e componentes | Oferta real de parceiros |
| Regras de cálculo calibradas | Know-how operacional |
| Estoque e decoração reais | Inventário físico da empresa |
| Nomes da equipe (cozinheiras) | Dados pessoais / operacionais |
| Eventos com clientes reais | Privacidade |

## Antes do primeiro push

1. Confirme que `internal/database/seeds/private/` contém seus `.sql` completos (veja o README dessa pasta).
2. Verifique o `.gitignore` — especialmente `/data/` e `internal/database/seeds/private/*.sql`.
3. Rode `go test ./...` para garantir que os seeds públicos passam nos testes.
4. **Nunca** commite `data/buffetflow.db` — ele pode conter dados reais mesmo com seeds públicos.

### Se migrations antigas já foram commitadas com dados reais

Remova do histórico antes de tornar o repo público:

```powershell
# Exemplo: remover um arquivo do índice sem apagar localmente
git rm --cached internal/database/seeds/private/002_demo_catalog.sql
```

Para limpar histórico já publicado, use `git filter-repo` ou BFG — dados sensíveis em commits antigos continuam acessíveis até serem removidos do histórico.

## Desenvolvimento local com dados completos

```powershell
# Seeds públicos (migrations) + privados (pasta local)
go run ./cmd/server
```

Apague o banco para recarregar tudo:

```powershell
Remove-Item data/buffetflow.db
go run ./cmd/server
```

## Testes com catálogo completo

```powershell
go test -tags private_seeds ./internal/database/...
```

O teste `TestModelSeedsAreCompleteAndIdempotent` só roda com a tag `private_seeds`, pois valida contagens do catálogo real.

## Checklist de publicação

- [ ] `.gitignore` atualizado
- [ ] Seeds privados fora do Git
- [ ] README reflete versão de portfólio
- [ ] Nenhum `.db` ou `.env` staged
- [ ] `go test ./...` verde
- [ ] Repositório marcado como público somente após revisão
