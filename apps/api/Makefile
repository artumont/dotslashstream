.PHONY: help test test-v test-race test-short test-cover test-e2e lint vet build run tidy clean registry migrate

API_DIR := $(shell pwd)
COVER_PROFILE := coverage.out

## ── Development ──────────────────────────────────────────────────────────────

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the API binary
	go build -o bin/api ./cmd/api

run: ## Run the API server
	go run ./cmd/api

tidy: ## Tidy and verify go.mod
	go mod tidy
	go mod verify

clean: ## Remove build artifacts and Docker resources
	docker compose down -v --remove-orphans
	docker image rm -f dotslashstream-api 2>/dev/null || true
	rm -rf bin/ $(COVER_PROFILE) .tmp/

## ── Testing ──────────────────────────────────────────────────────────────────

test: ## Run all tests
	go test ./...

test-v: ## Run tests with verbose output
	go test -v ./...

test-race: ## Run tests with race detector
	go test -race ./...

test-short: ## Run tests (short mode, skip slow tests)
	go test -short ./...

test-cover: ## Run tests with coverage report
	go test -coverprofile=$(COVER_PROFILE) ./...
	go tool cover -func=$(COVER_PROFILE)
	@echo ""
	@echo "Open HTML report: go tool cover -html=$(COVER_PROFILE)"

test-bench: ## Run benchmarks
	go test -bench=. -benchmem ./...

test-e2e: ## Seed and run e2e tests against running API
	python3 scripts/seed.py
	python3 -m tests

## ── Code Quality ─────────────────────────────────────────────────────────────

vet: ## Run go vet
	go vet ./...

GOBIN := $(shell go env GOPATH)/bin

lint: ## Run staticcheck (install: go install honnef.co/go/tools/cmd/staticcheck@latest)
	$(GOBIN)/staticcheck ./...

## ── Code Generation ──────────────────────────────────────────────────────────

registry: ## Regenerate postgres table registry from models
	python3 scripts/gen_registry.py

seed: ## Seed database with test data
	python3 scripts/seed.py

seed-clean: ## Clean and re-seed test data
	python3 scripts/seed.py --clean

## ── Migrations ──────────────────────────────────────────────────────────────

migrate: ## Run pending migrations (unsafe needs MIGRATE_UNSAFE=true)
	go run ./cmd/migrate up

migrate-down: ## Rollback the last applied migration
	go run ./cmd/migrate down

migrate-status: ## Show applied migrations
	go run ./cmd/migrate status

migration-list: ## List all migrations with metadata
	go run ./cmd/migrate list

migrate-create: ## Create a safe migration (usage: make migrate-create name=add_foo desc="Add foo table")
	@if [ -z "$(name)" ] || [ -z "$(desc)" ]; then echo 'usage: make migrate-create name=<name> desc="<description>"'; exit 1; fi
	go run ./cmd/migrate create $(name) "$(desc)"

migrate-create-unsafe: ## Create an unsafe migration (usage: make migrate-create-unsafe name=drop_bar desc="Drop bar table")
	@if [ -z "$(name)" ] || [ -z "$(desc)" ]; then echo 'usage: make migrate-create-unsafe name=<name> desc="<description>"'; exit 1; fi
	go run ./cmd/migrate create $(name) "$(desc)" --unsafe

## ── Database ─────────────────────────────────────────────────────────────────

db-reset: ## Drop and recreate database, then re-run migrations (DANGER)
	@echo "⚠️  This will destroy all data. Press Ctrl+C to abort."
	@sleep 3
	docker compose exec db psql -U postgres -c "DROP DATABASE IF EXISTS dotslashstream;"
	docker compose exec db psql -U postgres -c "CREATE DATABASE dotslashstream;"
	make migrate

db-shell: ## Open psql shell
	docker compose exec db psql -U postgres -d dotslashstream

db-psql: ## Quick psql (usage: make db-psql SQL="SELECT * FROM users")
	docker compose exec db psql -U postgres -d dotslashstream -c "$(SQL)"

## ── Docker (Dev) ─────────────────────────────────────────────────────────────

dev: ## Start dev environment with hot reload
	docker compose up --build

dev-d: ## Start dev environment detached
	docker compose up --build -d

dev-down: ## Stop dev environment
	docker compose down

dev-logs: ## Tail dev logs
	docker compose logs -f api

dev-restart: ## Restart API only
	docker compose restart api

## ── Docker (Production) ──────────────────────────────────────────────────────

prod-build: ## Build production image
	docker build -t dotslashstream-api .

prod-run: ## Run production image
	docker run -p 8080:8080 --env-file .env dotslashstream-api

## ── Git Hooks ────────────────────────────────────────────────────────────────

pre-test: vet test ## Run vet + tests (use as pre-push check)

## ── All-in-One ───────────────────────────────────────────────────────────────

ci: registry tidy vet test-race ## Full CI pipeline: generate, tidy, vet, test with race
