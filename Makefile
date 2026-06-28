# expense-tracker — common developer targets (PR01+)
MODULE      := github.com/krishnamadhavan/expense-tracker
GO          ?= go
GOFLAGS     ?=
BIN_DIR     := bin
SERVER_BIN  := $(BIN_DIR)/expense-tracker
ET_HTTP_ADDR ?= :8080

.PHONY: all help tidy build test vet fmt check run clean ci migrate-up migrate-down db-up db-down

all: check build

help: ## Show targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

tidy: ## Sync go.mod / go.sum
	$(GO) mod tidy

build: ## Build API server binary to bin/
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -o $(SERVER_BIN) ./cmd/server

test: ## Run unit tests
	$(GO) test $(GOFLAGS) ./...

vet: ## Run go vet
	$(GO) vet ./...

fmt: ## Format Go sources (fail if would change in CI via check)
	$(GO) fmt ./...

check: vet test ## Vet + test (CI gate)

run: build ## Build and run the server (ET_HTTP_ADDR, default :8080)
	ET_HTTP_ADDR=$(ET_HTTP_ADDR) $(SERVER_BIN)

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)

ci: tidy check build ## Local approximation of CI pipeline


# --- Database (PR03) ---
# Requires: Docker for db-up; golang-migrate CLI for migrate-* (see scripts/migrate.sh)

DATABASE_URL ?= postgres://expense:expense@localhost:5432/expense_tracker?sslmode=disable

db-up: ## Start Postgres 16 via docker-compose.db.yaml
	docker compose -f docker-compose.db.yaml up -d

db-down: ## Stop Postgres compose stack
	docker compose -f docker-compose.db.yaml down

migrate-up: ## Apply all migrations (DATABASE_URL or ET_*_DATABASE_URL)
	ET_DATABASE_URL="$(DATABASE_URL)" ./scripts/migrate.sh up

migrate-down: ## Roll back one migration
	ET_DATABASE_URL="$(DATABASE_URL)" ./scripts/migrate.sh down 1
