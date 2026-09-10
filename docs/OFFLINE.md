# Operação offline

O emenysFlow escolhe o modo sozinho. Com conexão, abre o sistema completo. Sem serviço, fica no modo offline.

O **modo online** é o sistema completo. O **modo offline** usa só as checklists e o organizador de layout dos eventos salvos neste aparelho.

Nenhuma senha é persistida no navegador.

## Fluxo

1. Depois do login, o aparelho consulta `/api/health`.
2. Se o serviço responder, o sistema online abre.
3. Se o serviço não responder, o modo offline abre sozinho, limitado a checklists e layout.
4. Um watcher continua consultando a conexão.
5. Nas configurações (e no hub offline) dá para ligar **Ao reconectar, abrir o sistema online**. Ligado é o padrão: quando a conexão volta, o sistema completo abre. Desligado, o aparelho permanece offline.
6. Sincronizar alterações da checklist/layout **é opcional**.
7. Enquanto houver conexão, o aparelho baixa os eventos para o modo offline funcionar sem internet.

Cada operação possui um identificador único; repetições não são aplicadas duas vezes no servidor. Quantidades e rascunhos utilizam a versão conhecida no momento da alteração. Se houver uma versão mais nova no servidor, o painel de conflitos mostra as duas cópias e permite manter o servidor, manter a edição local ou mesclar campos compatíveis.

## O que fica em cada área

| Área | Quando usar | O que faz |
|------|-------------|-----------|
| Online | Há internet | Criar/editar eventos e CRUD do estoque |
| Offline | No salão, na van ou sem rede | Checklist dos eventos salvos e layout das festas |

## Segurança e limites

- regras globais, cadastros administrativos e exclusões destrutivas exigem o modo online;
- o logout apaga IndexedDB, caches da aplicação e a identificação local do dispositivo;
- uma sessão offline expirada exige reconexão;
- uma falha não remove a operação ou a foto: o registro passa para `failed` e pode ser tentado novamente;
- para produção, a aplicação deve ser servida por HTTPS, requisito dos Service Workers fora de `localhost`.
