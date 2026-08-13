# Operação offline

O emenysFlow sincroniza os eventos, cardápios, serviços, checklists, estoque e configurações operacionais do usuário autenticado para o IndexedDB do dispositivo. Nenhuma senha é persistida no navegador.

## Fluxo

1. Com conexão, abra o sistema e aguarde o indicador `Online e sincronizado`.
2. As páginas visitadas e o conjunto operacional ficam disponíveis localmente por 12 horas.
3. Sem internet, alterações de separação, carregamento, faltas, itens manuais, rascunhos de evento e layouts do salão são gravadas na fila local.
4. Fotos de referência permanecem no dispositivo até o servidor confirmar o upload.
5. Ao voltar a conexão, a sincronização inicia automaticamente. O botão `Sincronizar agora` permite antecipá-la.

Cada operação possui um identificador único; repetições não são aplicadas duas vezes no servidor. Quantidades e rascunhos utilizam a versão conhecida no momento da alteração. Se houver uma versão mais nova no servidor, o painel de conflitos mostra as duas cópias e permite manter o servidor, manter a edição local ou mesclar campos compatíveis.

## Segurança e limites

- regras globais, cadastros administrativos e exclusões destrutivas exigem conexão;
- o logout apaga IndexedDB, caches da aplicação e a identificação local do dispositivo;
- uma sessão offline expirada exige reconexão;
- uma falha não remove a operação ou a foto: o registro passa para `failed` e pode ser tentado novamente;
- para produção, a aplicação deve ser servida por HTTPS, requisito dos Service Workers fora de `localhost`.
