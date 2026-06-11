# freehire

[freehire.dev](https://freehire.dev) — open-source IT job aggregator: many source parsers, normalization into a single schema, and an AI enrichment layer on top. Fully open and transparent, designed to make adding new sources easy.

> This is a backend scaffold. For now it only contains the skeleton: Fiber + Postgres + sqlc. Parsers, the pipeline, and the AI layer come next.

## Stack

- **Go** + [Fiber](https://gofiber.io/) — HTTP server
- **PostgreSQL** — storage and filtering
- **[sqlc](https://sqlc.dev/)** — type-safe DB access from SQL (no ORM)
- **Docker Compose** — local development

## Quick start

```bash
make up        # start app + postgres in Docker
curl localhost:8080/health
curl localhost:8080/api/v1/jobs
```

Migrations are applied automatically on first Postgres volume init
(the `migrations/` folder is mounted into `/docker-entrypoint-initdb.d`).

If port 8080 is already taken, pick another host port:

```bash
HIRE_HOST_PORT=8090 make up
```

## Local development

```bash
docker compose up -d db   # database only
make run                  # server on host, reads DATABASE_URL
```

## Commands

```bash
make help      # list all commands
make sqlc      # regenerate code from SQL (via Docker, no local sqlc needed)
make tidy      # go mod tidy
make psql      # psql inside the DB container
```

## Layout

```
cmd/server/        entry point
internal/
  config/          env configuration
  database/        pgxpool connection pool
  db/              generated sqlc code + queries/*.sql
  handlers/        HTTP handlers
migrations/        SQL schema (source for both sqlc and initdb)
```

## API

| Method | Path                | Description                     |
|--------|---------------------|---------------------------------|
| GET    | `/health`           | Service and DB status           |
| GET    | `/api/v1/jobs`      | List jobs (`limit`/`offset`)    |
| GET    | `/api/v1/jobs/:id`  | Job by id                       |

## Adding a source (planned)

The parser architecture lands next: each source implements a common interface
and registers itself in a registry — adding one comes down to a single file in
`internal/sources/`.
