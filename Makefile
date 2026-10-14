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

.PHONY: help urls up down restart reset reseed status logs

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
	@echo "Redis       localhost:6379  (rate limit counters)"

up: ## Start the full stack: Postgres + migrations + Redis + Loki/Alloy/Grafana
	$(COMPOSE) up -d

down: ## Stop the stack, keep data volumes
	$(COMPOSE) down

restart: down up ## Stop then start again

reset: ## Stop and DELETE all data (Postgres/Loki/Grafana), then start clean
	$(COMPOSE) down -v
	$(COMPOSE) up -d

# `reseed` is the one-command "start the database over": it drops everything in the
# application database (schema public: every table, user data included), re-applies all
# migrations and loads every seed. Loki/Grafana volumes are left alone (that is `reset`).
# Stop appService first: its connection pool keeps handles on the dropped tables, so restart
# it afterwards (`make run` does everything above and starts it).
reseed: db-up ## WIPE the whole app database, re-apply migrations and seed everything (one command)
	@echo "Dropping schema public in $(DB_NAME): every table and row is deleted"
	$(COMPOSE) exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -v ON_ERROR_STOP=1 \
		-c 'DROP SCHEMA public CASCADE' -c 'CREATE SCHEMA public'
	$(COMPOSE) run --rm migrate
	$(MAKE) ensure-data
	@echo "reseed done: restart appService (make run)"

status: ## Show container status
	$(COMPOSE) ps

logs: ## Follow logs of every stack container
	$(COMPOSE) logs -f

# ── appService (runs on the host, not in compose) ───────────────────────────
#
# `run` depends on `up` and `ensure-data` so a fresh checkout needs one
# command: on its first start appService seeds the sutra library, and every sutra
# links to a deity, so an empty `deities` table would stop the service
# (`failed to seed sutras ... run cmd/seeddeities first`). `ensure-data` also
# restores temples and gallery when a test run wiped the dev database. Logs are teed
# into $(APP_LOG) because Alloy cannot read a host process's stdout — see
# docs/development/local-stack.md §"Vì sao appService không nằm trong compose".

.PHONY: run run-backoffice build test vet

run: up redis-up ensure-data ## Run appService on the host (re-seeds deities/temples/gallery if wiped), teeing logs into observability/logs/appservice.log
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

# ── redis (rate limit counters) ─────────────────────────────────────────────
#
# appService pings Redis at start and stops if it is not there, so `run` waits
# for the healthcheck (`up` alone returns before Redis answers).

.PHONY: redis-up redis-cli redis-ratelimits

REDIS_PASSWORD ?= innerlamp-dev-redis

redis-up: ## Start only Redis and wait until it answers
	$(COMPOSE) up -d --wait redis

redis-cli: redis-up ## Open an interactive redis-cli in the Redis container
	$(COMPOSE) exec redis redis-cli -a '$(REDIS_PASSWORD)' --no-auth-warning

redis-ratelimits: redis-up ## List live rate-limit counters with their TTL (ms)
	@$(COMPOSE) exec -T redis sh -c "redis-cli -a '$(REDIS_PASSWORD)' --no-auth-warning --scan --pattern 'innerlamp:*' | while read k; do echo \"\$$k  count=\$$(redis-cli -a '$(REDIS_PASSWORD)' --no-auth-warning zcard \"\$$k\")  pttl=\$$(redis-cli -a '$(REDIS_PASSWORD)' --no-auth-warning pttl \"\$$k\")\"; done"

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
# appService seeds nothing by itself: every table is loaded by a cmd/seed* command, and
# `make run` runs the ensure-* targets for them. Sutras link to deities, so deities
# must exist first:
#   ensure-deities  seed deities ONLY when the table is empty (what `run` uses)
#   seed-deities    always upsert from assets/data/phat_botat.json — this
#                   overwrites hand edits made to the deities table
#   seed-sutras     load the sutra library; does nothing when sutras has rows

.PHONY: ensure-deities ensure-temples ensure-gallery ensure-admin-units ensure-calendar-events seed-calendar-events ensure-temple-schedules seed-temple-schedules ensure-temple-events seed-temple-events ensure-articles ensure-dharma-talks ensure-sutras ensure-data seed-deities seed-gallery seed-admin-units seed-articles seed-dharma-talks seed-sutras seed

# A non-short `go test` against the dev database TRUNCATEs temples, deities, gallery_images,
# articles and sutras (repository tests connect to innerlamp, not a test database). The
# ensure-* targets below put that data back: each one seeds only when its table has no
# usable rows, so `make run` repairs a wiped database and leaves a healthy one alone.
# appService does not seed anything when it starts: articles, dharma talks and sutras are loaded by
# cmd/seedarticles, cmd/seeddharmatalks and cmd/seedsutras below, exactly like the other tables.

ensure-data: ensure-deities ensure-temples ensure-gallery ensure-admin-units ensure-calendar-events ensure-temple-schedules ensure-temple-events ensure-articles ensure-dharma-talks ensure-sutras ## Seed every table the service reads from when it is empty (what `run` uses; appService itself seeds nothing)

# "Usable" for deities means at least one row carries an image: the rows sutra seeding or a
# test leaves behind are name-only stubs, and a plain row count would treat them as seeded.
ensure-deities: db-up ## Seed deities only if none has an image (safe to repeat, keeps hand edits)
	@n=$$($(COMPOSE) exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -tAc "SELECT count(*) FROM deities WHERE COALESCE(image_url, '') <> ''" | tr -d '[:space:]'); \
	if [ "$$n" = "0" ]; then \
		echo "deities have no images (empty or stub rows): seeding from assets/data/phat_botat.json"; \
		go run ./services/appService/cmd/seeddeities; \
	else \
		echo "deities already seeded ($$n with images): leaving them as they are"; \
	fi

# Temple addresses are resolved against the administrative-units dataset, so those load first.
ensure-temples: db-up ensure-admin-units ## Seed temples only if the table is empty (safe to repeat, keeps hand edits)
	@n=$$($(COMPOSE) exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -tAc "SELECT count(*) FROM temples" | tr -d '[:space:]'); \
	if [ "$$n" = "0" ]; then \
		echo "temples table is empty: seeding from assets/data/temple.json"; \
		go run ./services/appService/cmd/seedtemples; \
	else \
		echo "temples already seeded ($$n rows): leaving them as they are"; \
	fi

ensure-gallery: db-up ## Seed gallery images only if the table is empty (safe to repeat, keeps hand edits)
	@n=$$($(COMPOSE) exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -tAc "SELECT count(*) FROM gallery_images" | tr -d '[:space:]'); \
	if [ "$$n" = "0" ]; then \
		echo "gallery_images table is empty: seeding from assets/data/buddhist_gallery.json"; \
		go run ./services/appService/cmd/seedgallery; \
	else \
		echo "gallery_images already seeded ($$n rows): leaving them as they are"; \
	fi

# Administrative units (tỉnh/quận/xã, hệ mới + cũ + bảng chuyển đổi) come from one file. Without
# them the lookups return empty lists and POST /v1/temple-suggestions refuses every province_code.
ensure-admin-units: db-up ## Seed administrative units only if no new-scheme province exists (safe to repeat)
	@n=$$($(COMPOSE) exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -tAc "SELECT count(*) FROM addr_new_provinces" | tr -d '[:space:]'); \
	if [ "$$n" = "0" ]; then \
		echo "addr_new_provinces is empty: seeding from assets/data/administrative-units.json"; \
		go run ./services/appService/cmd/seedadminunits; \
	else \
		echo "administrative units already seeded ($$n provinces): leaving them as they are"; \
	fi

seed-admin-units: db-up ## Upsert administrative units from assets/data/administrative-units.json (idempotent; -dry-run via go run)
	go run ./services/appService/cmd/seedadminunits

ensure-calendar-events: db-up ## Seed calendar events only if the table is empty (safe to repeat, keeps back-office edits)
	@n=$$($(COMPOSE) exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -tAc "SELECT count(*) FROM calendar_events" | tr -d '[:space:]'); \
	if [ "$$n" = "0" ]; then \
		echo "calendar_events is empty: seeding from assets/data/calendar-events.json"; \
		go run ./services/appService/cmd/seedcalendarevents; \
	else \
		echo "calendar events already seeded ($$n events): leaving them as they are"; \
	fi

seed-calendar-events: db-up ## Upsert calendar events from assets/data/calendar-events.json (idempotent; -dry-run via go run)
	go run ./services/appService/cmd/seedcalendarevents

ensure-temple-schedules: db-up ## Seed temple schedules only if the table is empty (safe to repeat, keeps back-office edits; needs temples seeded first)
	@n=$$($(COMPOSE) exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -tAc "SELECT count(*) FROM temple_schedules" | tr -d '[:space:]'); \
	if [ "$$n" = "0" ]; then \
		echo "temple_schedules is empty: seeding from assets/data/temple-schedules.json"; \
		go run ./services/appService/cmd/seedtempleschedules; \
	else \
		echo "temple schedules already seeded ($$n items): leaving them as they are"; \
	fi

seed-temple-schedules: db-up ## Upsert temple schedules from assets/data/temple-schedules.json (idempotent; -dry-run via go run)
	go run ./services/appService/cmd/seedtempleschedules

ensure-temple-events: db-up ## Seed temple events only if the table is empty (safe to repeat, keeps back-office edits; needs temples seeded first)
	@n=$$($(COMPOSE) exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -tAc "SELECT count(*) FROM temple_events" | tr -d '[:space:]'); \
	if [ "$$n" = "0" ]; then \
		echo "temple_events is empty: seeding from assets/data/temple-events.json"; \
		go run ./services/appService/cmd/seedtempleevents; \
	else \
		echo "temple events already seeded ($$n events): leaving them as they are"; \
	fi

seed-temple-events: db-up ## Upsert temple events from assets/data/temple-events.json (idempotent; -dry-run via go run)
	go run ./services/appService/cmd/seedtempleevents

ensure-articles: db-up ## Seed articles only if the table is empty (safe to repeat, keeps hand edits)
	@n=$$($(COMPOSE) exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -tAc "SELECT count(*) FROM articles" | tr -d '[:space:]'); \
	if [ "$$n" = "0" ]; then \
		echo "articles table is empty: seeding from assets/data/buddhist_articles.json"; \
		go run ./services/appService/cmd/seedarticles; \
	else \
		echo "articles already seeded ($$n rows): leaving them as they are"; \
	fi

# Talks flagged needs_review count as rows (they are stored, just never served), so a table whose
# every talk is under review is not seen as empty. cmd/seeddharmatalks applies the same rule.
ensure-dharma-talks: db-up ## Seed dharma talks only if the table is empty (safe to repeat, keeps hand edits)
	@n=$$($(COMPOSE) exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -tAc "SELECT count(*) FROM dharma_talks" | tr -d '[:space:]'); \
	if [ "$$n" = "0" ]; then \
		echo "dharma_talks table is empty: seeding from assets/data/dharma_talks.json"; \
		go run ./services/appService/cmd/seeddharmatalks; \
	else \
		echo "dharma talks already seeded ($$n rows): leaving them as they are"; \
	fi

# Every sutra links to deities, so deities first. cmd/seedsutras itself does nothing when sutras has rows.
ensure-sutras: ensure-deities ## Seed the sutra library only if the table is empty (safe to repeat)
	@n=$$($(COMPOSE) exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -tAc "SELECT count(*) FROM sutras" | tr -d '[:space:]'); \
	if [ "$$n" = "0" ]; then \
		echo "sutras table is empty: seeding the library (assets/data/sutras)"; \
		go run ./services/appService/cmd/seedsutras; \
	else \
		echo "sutras already seeded ($$n rows): leaving them as they are"; \
	fi

seed-articles: db-up ## Insert articles from assets/data/buddhist_articles.json that are not in the table yet (never modifies existing rows)
	go run ./services/appService/cmd/seedarticles

seed-dharma-talks: db-up ## Load dharma talks from assets/data/dharma_talks.json (does nothing when the table has rows)
	go run ./services/appService/cmd/seeddharmatalks

seed-temples: db-up ## Upsert temples from assets/data/temple.json (overwrites hand edits)
	go run ./services/appService/cmd/seedtemples

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
