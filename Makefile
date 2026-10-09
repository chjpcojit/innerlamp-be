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
	@echo "backoffice  http://localhost:8081"
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
# `run` depends on `up` and `ensure-deities` so a fresh checkout needs one
# command: on its first start appService seeds the sutra library, and every sutra
# links to a deity, so an empty `deities` table would stop the service
# (`failed to seed sutras ... run cmd/seeddeities first`). Logs are teed
# into $(APP_LOG) because Alloy cannot read a host process's stdout — see
# docs/development/local-stack.md §"Vì sao appService không nằm trong compose".

.PHONY: run run-backoffice build test vet

run: up ensure-deities ## Run appService on the host (seeds deities first if empty), teeing logs into observability/logs/appservice.log
	@mkdir -p "$$(dirname '$(APP_LOG)')"
	@set -o pipefail; go run ./services/appService/cmd 2>&1 | tee -a '$(APP_LOG)'

run-backoffice: db-up ## Run backofficeService (admin API, :8081) on the host
	go run ./services/backofficeService/cmd

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

# ── seed data ───────────────────────────────────────────────────────────────
#
# appService seeds articles and the sutra library by itself on its first start,
# but deities and temples are loaded by commands. Sutras link to deities, so
# deities must exist first:
#   ensure-deities  seed deities ONLY when the table is empty (what `run` uses)
#   seed-deities    always upsert from assets/data/phat_botat.json — this
#                   overwrites hand edits made to the deities table
#   seed-sutras     load the sutra library; does nothing when sutras has rows

.PHONY: ensure-deities seed-deities seed-gallery seed-sutras seed

ensure-deities: db-up ## Seed deities only if the table is empty (safe to repeat, keeps hand edits)
	@n=$$($(COMPOSE) exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -tAc "SELECT count(*) FROM deities" | tr -d '[:space:]'); \
	if [ "$$n" = "0" ]; then \
		echo "deities table is empty: seeding from assets/data/phat_botat.json"; \
		go run ./services/appService/cmd/seeddeities; \
	else \
		echo "deities already seeded ($$n rows): leaving them as they are"; \
	fi

seed-deities: db-up ## Upsert deities from assets/data/phat_botat.json (overwrites hand edits)
	go run ./services/appService/cmd/seeddeities

seed-gallery: db-up ## Upsert gallery images from assets/data/buddhist_gallery.json (needs migrations/0017 applied)
	go run ./services/appService/cmd/seedgallery

seed-sutras: ensure-deities ## Load the sutra library by hand (no-op when the sutras table already has rows)
	go run ./services/appService/cmd/seedsutras

seed: seed-sutras ## Deities (if empty) then sutras — the order a fresh database needs

# ── LAN image URLs (real device, no CDN yet) ───────────────────────────────
#
# appService serves assets/images at /static/images/ (server.static_images_dir in
# config.dev.yaml). A phone cannot reach "localhost", so the image URLs stored in
# temple.json / phat_botat.json / buddhist_gallery.json / buddhist_articles.json (thumbnailURL) must carry this
# machine's LAN IP. The IP changes with the network, so this target rewrites the host in
# those files and re-seeds. Gallery files come from scripts/download_gallery_images.py. Article rows already in the DB are updated by SQL, because
# seedarticles never overwrites a stored value; restart appService afterwards (the
# articles JSON is embedded in the binary).
#   make lan-urls                   use this Mac's Wi-Fi IP (en0, else en1)
#   make lan-urls IP=192.168.1.50   use an explicit IP

.PHONY: lan-urls
LAN_IP ?= $(shell ipconfig getifaddr en0 2>/dev/null || ipconfig getifaddr en1 2>/dev/null)

lan-urls: db-up ## Point image URLs at this machine's LAN IP and re-seed deities + gallery + temples (DEV ONLY)
	@ip="$(if $(IP),$(IP),$(LAN_IP))"; \
	test -n "$$ip" || { echo "no LAN IP found; pass one: make lan-urls IP=192.168.x.x"; exit 1; }; \
	echo "image host -> http://$$ip:8080/static/images/"; \
	perl -pi -e "s#http://[^/\"]+/static/images/#http://$$ip:8080/static/images/#g" assets/data/temple.json assets/data/phat_botat.json assets/data/buddhist_articles.json assets/data/buddhist_gallery.json && \
	go run ./services/appService/cmd/seeddeities && \
	go run ./services/appService/cmd/seedgallery && \
	go run ./services/appService/cmd/seedtemples && \
	go run ./services/appService/cmd/seedarticles -fill-images && \
	$(COMPOSE) exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -c "UPDATE articles SET thumbnail_url = regexp_replace(thumbnail_url, '^http://[^/]+/static/images/', 'http://'||'$$ip'||':8080/static/images/') WHERE thumbnail_url LIKE 'http://%/static/images/%'" && \
	echo "done. Run: make run   (device must be on the same Wi-Fi; test: curl http://$$ip:8080/static/images/temples/full/temple_north_001.jpg -I)"

# ── observability ───────────────────────────────────────────────────────────

.PHONY: log-query

log-query: ## Print the 20 latest appService log lines from Loki
	@curl -sG 'http://localhost:3100/loki/api/v1/query_range' \
		--data-urlencode 'query={service="appservice"}' \
		--data-urlencode 'limit=20'
	@echo
