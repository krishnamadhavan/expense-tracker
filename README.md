# expense-tracker

Personal finance tracker (income vs expenses) with API-first Go backend, automatic categorization that learns from moderations, monthly/FY reports, and a local-first deploy path (docker-compose → Minikube → GitOps).

**Status:** M1 in progress. Architecture and PR plan live in [`docs/DESIGN.md`](docs/DESIGN.md).

## Requirements

- Go **1.24+** (language version pinned in `go.mod` as `1.24.0`; newer toolchains such as Go 1.26 work via the standard toolchain)
- Make (optional, for convenience targets)

## Quick start (PR01)

```bash
# Install deps / sync module (none beyond stdlib yet)
make tidy

# Run tests + vet
make check

# Build and run health endpoints
make run
# → GET http://localhost:8080/healthz
# → GET http://localhost:8080/readyz
```

Or without Make:

```bash
go test ./...
go run ./cmd/server
```

Environment variables used so far:

| Variable       | Default | Description        |
| -------------- | ------- | ------------------ |
| `ET_HTTP_ADDR` | `:8080` | HTTP listen address |

## Run API (PR05)

```bash
# With Docker (db + migrate + api). Set a real password in compose for non-local use.
docker compose up --build

# Or local binary against make db-up + migrate-up:
export ET_DATABASE_URL='postgres://expense:expense@localhost:5432/expense_tracker?sslmode=disable'
export ET_BOOTSTRAP_PASSWORD='changeme'
export ET_COOKIE_SECURE=false
export ET_CORS_ORIGINS=http://localhost:5173
make run
# Login: POST /api/v1/auth/login {"email":"admin@localhost","password":"changeme"}
# OpenAPI: GET /api/openapi.yaml
```

## Database (PR03)

SQL migrations live in `migrations/` (golang-migrate). Optional local Postgres:

```bash
# requires Docker
make db-up
# install CLI once: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.3
make migrate-up
```

Default URL: `postgres://expense:expense@localhost:5432/expense_tracker?sslmode=disable`  
Seed household id: `11111111-1111-4111-8111-111111111111` (see `internal/db`).

## Module

```text
github.com/krishnamadhavan/expense-tracker
```

## License

MIT — see [LICENSE](LICENSE).

## Roadmap (high level)

| Milestone | Scope |
| --------- | ----- |
| **M1** (PR01–05) | Skeleton → domain → schema → repos → secure API + minimal compose |
| **M2** | Categorization + moderation learning |
| **M3** | Reports APIs, export, budgets |
| **M4** | Web SPA (transactions, review, charts) |
| **M5** | Minikube / GitOps (optional stretch) |

PR01 delivers: Go module, Makefile, MIT license, GitHub Actions CI smoke, and a minimal HTTP server with `/healthz` and `/readyz`.
