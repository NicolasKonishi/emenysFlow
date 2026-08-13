# Seeds privados (não versionados)

Esta pasta contém dados operacionais reais do buffet — cardápios completos, serviços, estoque, regras calibradas e nomes da equipe. **Não deve ir para o GitHub.**

O repositório público traz apenas seeds mínimos para demonstrar a arquitetura. Com os arquivos `.sql` desta pasta, o sistema sobe com o catálogo completo localmente.

## Arquivos esperados

Copie (ou mantenha) estes arquivos aqui, na ordem alfabética em que são aplicados:

| Arquivo | Conteúdo |
|---------|----------|
| `002_demo_catalog.sql` | Catálogo completo, regras calibradas e evento de demonstração |
| `008_model_catalogs.sql` | Metadados dos cardápios-modelo |
| `009_menu_models.sql` | Itens, seções, grupos de escolha e blocos compartilhados |
| `010_service_models.sql` | Serviços terceirizados e componentes |
| `011_service_inventory_links.sql` | Vínculos serviço → estoque |
| `013_kitchen_cooks.sql` | Cozinheiras, caixas pessoais e conteúdo |
| `023_decoration_catalog.sql` | Acervo completo de decoração |

Se `009_menu_models.sql` ainda não existir aqui, recupere a versão anterior com:

```powershell
git show HEAD:internal/database/migrations/009_seed_menu_models.sql > internal/database/seeds/private/009_menu_models.sql
```

## Aplicação

Os seeds privados são carregados pelo servidor (`go run ./cmd/server`) após as migrations. Testes de integração com catálogo completo usam `-tags private_seeds`.
