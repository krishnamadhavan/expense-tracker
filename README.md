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
