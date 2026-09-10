# Publicação segura

Esta aplicação é um servidor Go com SQLite. Ela precisa de uma máquina com
disco persistente; GitHub Pages, Vercel e outros hosts puramente estáticos não
servem para ela. Os arquivos nesta pasta configuram uma implantação simples em
VPS com Docker, Caddy e certificado HTTPS automático.

## Antes de publicar

1. Não envie `data/`, `.env` ou os seeds de `internal/database/seeds/private/`
   para Git, Docker ou um serviço de terceiros.
2. Se algum banco já foi enviado ao Git, remova-o do histórico antes de tornar
   o repositório público. Removê-lo do último commit não impede acesso aos
   commits antigos.
3. Crie uma VPS atualizada, aponte o DNS do subdomínio para o IP dela e libere
   somente as portas TCP 80 e 443 no firewall. Não exponha a porta 8080.
4. Instale Docker Engine e o plugin Docker Compose na VPS.

## Subida inicial

No servidor, clone o repositório e crie o arquivo de segredos com permissões
restritas:

```bash
cp .env.example .env
chmod 600 .env
openssl rand -base64 32
# Edite .env e use o valor gerado em APP_ADMIN_PASSWORD.
docker compose up -d --build
docker compose logs -f caddy app
```

O Caddy só emite o certificado quando o DNS já aponta para a VPS. A aplicação
é acessível em `https://DOMAIN`; o primeiro administrador de uma base nova é
criado a partir de `APP_ADMIN_EMAIL` e `APP_ADMIN_PASSWORD`. Essa senha não é
registrada em log nem fica na imagem. Guarde-a num gerenciador de senhas.

Depois do primeiro acesso, crie contas individuais e desative contas que não
devem mais ter acesso. Não compartilhe links em `/share/`: eles funcionam como
uma credencial de leitura do evento.

## Dados reais e backup

O volume Docker `app_data` guarda o SQLite e as fotos enviadas. Ele não sai
automaticamente com um novo deploy. Faça backup cifrado, testado, fora da VPS.
Para garantir consistência, faça-o durante uma breve janela de manutenção:

```bash
docker compose stop app
mkdir -p backups
docker compose cp app:/var/lib/emenys ./backups/emenys-$(date +%F)
tar czf backups/emenys-$(date +%F).tgz -C backups emenys-$(date +%F)
docker compose start app
```

Copie o arquivo gerado para um destino protegido e apague o arquivo local quando
concluir. Para uma operação já existente, importe uma cópia consistente do
banco e das fotos no volume antes de iniciar a aplicação, ou faça a migração
da base dentro de uma janela de manutenção.

## Atualizações

```bash
git pull --ff-only
docker compose up -d --build
docker compose logs --tail=100 app
```

O contêiner executa como usuário não-root, usa sistema de arquivos somente
leitura fora do volume de dados e fica acessível apenas pelo Caddy na rede
interna. Em produção a aplicação exige HTTPS configurado, marca cookies como
`Secure`, aplica HSTS, valida a origem de requisições que alteram dados, limita
tentativas de login e usa uma senha inicial definida fora do código.
