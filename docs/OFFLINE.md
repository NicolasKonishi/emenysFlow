# Operação offline

O emenysFlow troca de área sozinho conforme a conexão com o serviço.

O **modo online** é o sistema completo. O **modo offline** usa só as checklists e o organizador de layout dos eventos salvos neste aparelho.

Nenhuma senha é persistida no navegador.

## Fluxo

1. O aparelho consulta `/api/health` de tempos em tempos.
2. Se o serviço responder, o modo online abre com tudo: eventos, estoque, modelos, checklists e layout.
3. Se o serviço não responder, o modo offline abre sozinho com as limitações de sempre.
4. Quando a conexão volta, aparece o convite para abrir o sistema completo. Dá para continuar offline se preferir.
5. Sincronizar alterações da checklist/layout **é opcional**.

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
