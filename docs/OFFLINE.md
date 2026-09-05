# Operação offline

O emenysFlow divide o trabalho em duas áreas. O **modo online** cria eventos e administra o estoque. O **modo offline** usa as checklists e o organizador de layout dos eventos que foram salvos neste aparelho.

Nenhuma senha é persistida no navegador.

## Fluxo

1. Com conexão, entre no **modo online**, cadastre o evento e revise o estoque.
2. Abra o **modo offline** (ou use *Salvar eventos neste aparelho*) para baixar checklists e layouts. Os dados ficam disponíveis localmente por 12 horas.
3. Sem internet, a área offline continua usable: checklist de separação/carregamento e planta do salão.
4. Sincronizar com o online **é opcional**. O interruptor *Sincronizar com o online* fica desligado por padrão.
5. Se quiser enviar as alterações, ligue a sincronização ou toque em *Sincronizar agora*.

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
