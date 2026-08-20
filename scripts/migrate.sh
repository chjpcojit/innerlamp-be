#!/usr/bin/env bash
# scripts/migrate.sh — PostgreSQL migration runner (golang-migrate)
#
# Usage:
#   ./scripts/migrate.sh up        — apply all pending migrations
#   ./scripts/migrate.sh down      — rollback one migration step
#   ./scripts/migrate.sh down N    — rollback N migration steps (default 1)
#   ./scripts/migrate.sh create NAME — create a new migration file pair
#
# Environment:
#   DATABASE_URL — PostgreSQL connection URL (required)
#                  Example: postgres://innerlamp:innerlamp@localhost:5432/innerlamp_dev?sslmode=disable
#   MIGRATIONS_DIR — path to migrations folder (default: ./migrations)
#
# The script loads .env if it exists in the project root.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-${PROJECT_ROOT}/migrations}"

# Load .env if it exists (local dev only)
if [[ -f "${PROJECT_ROOT}/.env" ]]; then
    set -o allexport
    # shellcheck disable=SC1091
    source "${PROJECT_ROOT}/.env"
    set +o allexport
fi

# Validate required env var
if [[ -z "${DATABASE_URL:-}" ]]; then
    echo "ERROR: DATABASE_URL is not set."
    echo ""
    echo "  For local development, copy .env.example to .env and set DATABASE_URL:"
    echo "    cp .env.example .env"
    echo ""
    echo "  Example value:"
    echo "    DATABASE_URL=postgres://innerlamp:innerlamp@localhost:5432/innerlamp_dev?sslmode=disable"
    exit 1
fi

# Validate golang-migrate is installed
if ! command -v migrate &>/dev/null; then
    echo "ERROR: golang-migrate is not installed."
    echo "  Install: brew install golang-migrate"
    echo "  Or: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
    exit 1
fi

COMMAND="${1:-}"
shift || true

case "${COMMAND}" in
    up)
        echo "▶  Applying migrations from ${MIGRATIONS_DIR} ..."
        migrate -database "${DATABASE_URL}" -path "${MIGRATIONS_DIR}" up
        echo "✓  Migrations applied."
        ;;
    down)
        STEPS="${1:-1}"
        echo "▶  Rolling back ${STEPS} migration(s) from ${MIGRATIONS_DIR} ..."
        migrate -database "${DATABASE_URL}" -path "${MIGRATIONS_DIR}" down "${STEPS}"
        echo "✓  Rollback complete."
        ;;
    create)
        NAME="${1:-}"
        if [[ -z "${NAME}" ]]; then
            echo "ERROR: NAME is required. Usage: make migrate-create NAME=create_users_table"
            exit 1
        fi
        echo "▶  Creating migration files for: ${NAME} ..."
        migrate create -ext sql -dir "${MIGRATIONS_DIR}" -seq "${NAME}"
        echo "✓  Migration files created in ${MIGRATIONS_DIR}/"
        ;;
    *)
        echo "Usage: $0 {up|down [N]|create NAME}"
        exit 1
        ;;
esac
