# expense-tracker — common developer targets (PR01+)
MODULE      := github.com/krishnamadhavan/expense-tracker
GO          ?= go
GOFLAGS     ?=
BIN_DIR     := bin
SERVER_BIN  := $(BIN_DIR)/expense-tracker
ET_HTTP_ADDR ?= :8080

.PHONY: all help tidy build test vet fmt check run clean ci

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
