# Makefile — local development helpers for innerlamp.
#
# Thin wrappers over `docker compose` (docker-compose.yml) and plain `go`
# commands; nothing here you could not run by hand. Prose:
#   docs/development/local-stack.md   — compose services, how to run, log stack
#   docs/development/database-cli.md  — Docker CLI + inspecting Postgres data
#
# Run `make` (or `make help`) for the target list.

SHELL := /bin/bash

COMPOSE ?= docker compose
APP_LOG ?= observability/logs/appservice.log
DB_USER ?= postgres
DB_NAME ?= innerlamp

.DEFAULT_GOAL := help

# ── stack lifecycle ─────────────────────────────────────────────────────────

.PHONY: help urls up down restart reset status logs

help: ## List available targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "} {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

urls: ## Print the local endpoints
	@echo "appService  http://localhost:8080"
	@echo "Grafana     http://localhost:3000"
	@echo "Loki        http://localhost:3100"
	@echo "Alloy       http://localhost:12345"
	@echo "Postgres    localhost:5432  (user=$(DB_USER) db=$(DB_NAME))"

up: ## Start the full stack: Postgres + migrations + Loki/Alloy/Grafana
	$(COMPOSE) up -d

down: ## Stop the stack, keep data volumes
	$(COMPOSE) down

restart: down up ## Stop then start again

reset: ## Stop and DELETE all data (Postgres/Loki/Grafana), then start clean
	$(COMPOSE) down -v
	$(COMPOSE) up -d

status: ## Show container status
	$(COMPOSE) ps

logs: ## Follow logs of every stack container
	$(COMPOSE) logs -f

# ── appService (runs on the host, not in compose) ───────────────────────────
#
# `run` depends on `up` so a fresh checkout needs one command. Logs are teed
# into $(APP_LOG) because Alloy cannot read a host process's stdout — see
# docs/development/local-stack.md §"Vì sao appService không nằm trong compose".

.PHONY: run build test vet

run: up ## Run appService on the host, teeing logs into observability/logs/appservice.log
	@mkdir -p "$$(dirname '$(APP_LOG)')"
	@set -o pipefail; go run ./services/appService/cmd 2>&1 | tee -a '$(APP_LOG)'

build: ## Compile every package
	go build ./...

test: ## Run fast tests (skips Docker/cloud integration)
	go test -short ./...

vet: ## Run go vet
	go vet ./...

# ── database ────────────────────────────────────────────────────────────────
#
# Every read-only target depends on db-up so it works even when only Postgres
# was started. Mutating targets are deliberately NOT auto-run by `up`.

.PHONY: db-up migrate psql db-tables db-counts db-accounts db-sessions db-otp db-reset-phone db-dump

db-up: ## Start only Postgres and apply migrations (no log stack)
	$(COMPOSE) up -d --wait postgres
	$(COMPOSE) run --rm migrate

migrate: ## Re-apply migrations/ to the running Postgres (idempotent)
	$(COMPOSE) run --rm migrate

psql: db-up ## Open an interactive psql shell in the Postgres container
	$(COMPOSE) exec postgres psql -U $(DB_USER) -d $(DB_NAME)

db-tables: db-up ## List tables
	$(COMPOSE) exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -c '\dt'

db-counts: db-up ## Row estimate per table
	$(COMPOSE) exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) \
		-c "SELECT relname AS table_name, n_live_tup AS row_estimate FROM pg_stat_user_tables ORDER BY relname;"

db-accounts: db-up ## List registered accounts
	$(COMPOSE) exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) \
		-c "SELECT id, phone_number, email, dharma_name, created_at FROM users ORDER BY id;"

db-sessions: db-up ## Show the 20 most recent sessions
	$(COMPOSE) exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) \
		-c "SELECT id, user_id, device_id, is_active, is_revoked, revoked_reason, expires_at FROM user_sessions ORDER BY id DESC LIMIT 20;"

db-otp: db-up ## Show recent OTP rows and send limits
	$(COMPOSE) exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) \
		-c "SELECT id, phone_number, purpose, attempts, expires_at, used_at, locked_until FROM otp_codes ORDER BY id DESC LIMIT 10;" \
		-c "SELECT phone_number, purpose, send_count, last_sent_at, locked_until FROM otp_send_limits ORDER BY phone_number;"

db-reset-phone: db-up ## Wipe one number's state to re-test a flow: make db-reset-phone PHONE=+84901234567
	@test -n "$(PHONE)" || { echo "Usage: make db-reset-phone PHONE=+84901234567"; exit 1; }
	$(COMPOSE) exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -v ON_ERROR_STOP=1 \
		-c "DELETE FROM users WHERE phone_number = '$(PHONE)';" \
		-c "DELETE FROM pending_registration WHERE phone_number = '$(PHONE)';" \
		-c "DELETE FROM otp_codes WHERE phone_number = '$(PHONE)';" \
		-c "DELETE FROM otp_send_limits WHERE phone_number = '$(PHONE)';" \
		-c "DELETE FROM login_attempts WHERE phone_number = '$(PHONE)';"

db-dump: db-up ## Print a pg_dump of the database to stdout: make db-dump > backup.sql
	$(COMPOSE) exec -T postgres pg_dump -U $(DB_USER) -d $(DB_NAME)

# ── observability ───────────────────────────────────────────────────────────

.PHONY: log-query

log-query: ## Print the 20 latest appService log lines from Loki
	@curl -sG 'http://localhost:3100/loki/api/v1/query_range' \
		--data-urlencode 'query={service="appservice"}' \
		--data-urlencode 'limit=20'
	@echo
