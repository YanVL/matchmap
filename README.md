# Matchmap

API em Go para localização, convites e partidas em tempo real (WebSocket), com PostgreSQL/PostGIS.

Os fluxos de desenvolvimento e migração passam pelo `Makefile`. Antes de qualquer comando, tenha o Docker Compose disponível e um arquivo `.env` na raiz (usado pela API e pelo banco).

Copie o exemplo e ajuste os valores se precisar:

```bash
cp .env.example .env
```

O `.env.example` documenta as variáveis:

| Variável | Uso |
| --- | --- |
| `POSTGRES_USER` | Usuário do Postgres (também no healthcheck e nas seeds) |
| `POSTGRES_PASSWORD` | Senha do Postgres |
| `POSTGRES_DB` | Nome do banco |
| `APP_PORT` | Porta publicada da API no host |
| `DB_PORT` | Porta publicada do Postgres no host |

`DATABASE_URL` não precisa estar no `.env` quando a API sobe pelo Compose: o `docker-compose.yml` monta a URL a partir das variáveis `POSTGRES_*`. Descomente `DATABASE_URL` no `.env` se for rodar a API fora do Docker.

## Comandos do Makefile

| Comando | O que faz |
| --- | --- |
| `make dev` | Sobe o ambiente de desenvolvimento (API + banco), reconstruindo as imagens |
| `make down` | Derruba o ambiente de desenvolvimento |
| `make migrate-up` | Aplica as migrations pendentes |
| `make migrate-force VERSION=<n>` | Força a versão das migrations (uso em estado sujo) |

### `make dev`

Sobe a stack com `docker-compose.yml` **e** `docker-compose.dev.yml`:

```bash
make dev
```

Equivalente a:

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build
```

O que sobe:

- **db** — PostGIS (`postgis/postgis:17-3.5`), porta `${DB_PORT}`
- **api** — imagem de desenvolvimento (`Dockerfile.dev`), porta `${APP_PORT}`

No overlay de dev a API monta o código em `/app` e roda o [Air](https://github.com/air-verse/air) (`air -c .air.toml`). Alterações em arquivos Go recarregam o processo sem rebuild da imagem.

A stack de dev **não** aplica migrations sozinha. Rode `make migrate-up` depois que o banco estiver saudável (ou em outro terminal, se `make dev` estiver em foreground).

### `make down`

Para e remove os containers da stack de desenvolvimento:

```bash
make down
```

O volume do Postgres (`matchmap-db-data`) permanece. Os dados do banco não são apagados.

### `make migrate-up`

Aplica todas as migrations em `migrations/` contra o banco do Compose:

```bash
make migrate-up
```

Usa o serviço `migrate` (`migrate/migrate`) definido em `docker-compose.yml`. O container sobe, espera o banco ficar healthy, roda `migrate up` e é removido (`--rm`).

Requer o banco acessível. Se nada estiver rodando, o Compose sobe o `db` para cumprir o `depends_on` e aplica as migrations.

### `make migrate-force`

Força a versão registrada pelo golang-migrate, **sem executar** os arquivos SQL. Serve para destravar o banco quando a tabela `schema_migrations` ficou em estado dirty (migration interrompida).

```bash
make migrate-force VERSION=4
```

`VERSION` é obrigatório: é o número da versão que deve ficar marcada como aplicada (o mesmo prefixo dos arquivos, por exemplo `0004_create_match_results`).

Use só quando souber qual versão está consistente com o schema real. Depois, se ainda houver migrations pendentes, rode `make migrate-up`.
