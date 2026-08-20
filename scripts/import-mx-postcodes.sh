#!/usr/bin/env bash
# Import Mexico postcode metadata (TSV) into PostgreSQL.
#
# Usage:
#   DATABASE_URL=postgres://... ./scripts/import-mx-postcodes.sh
#   DATABASE_URL=postgres://... CSV_FILE=./docs/mexico-zip-codes.csv ./scripts/import-mx-postcodes.sh
#
# Required:
#   DATABASE_URL - PostgreSQL DSN
#
# Optional:
#   CSV_FILE     - Source file (default: ./docs/mexico-zip-codes.csv)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

if [[ -f "${PROJECT_ROOT}/.env" ]]; then
    set -o allexport
    # shellcheck disable=SC1091
    source "${PROJECT_ROOT}/.env"
    set +o allexport
fi

if [[ -z "${DATABASE_URL:-}" ]]; then
    echo "ERROR: DATABASE_URL is not set."
    exit 1
fi

CSV_FILE="${CSV_FILE:-${PROJECT_ROOT}/docs/mexico-zip-codes.csv}"
if [[ ! -f "${CSV_FILE}" ]]; then
    echo "ERROR: CSV_FILE does not exist: ${CSV_FILE}"
    exit 1
fi

if ! command -v psql >/dev/null 2>&1; then
    echo "ERROR: psql is not installed."
    exit 1
fi

echo "▶ Importing postcode metadata from ${CSV_FILE} ..."

psql "${DATABASE_URL}" -v ON_ERROR_STOP=1 \
    -v csv_file="${CSV_FILE}" <<'SQL'
CREATE TEMP TABLE tmp_mx_postcodes_import (
    country_code       TEXT,
    postal_code        TEXT,
    place_name         TEXT,
    state_name         TEXT,
    state_code         TEXT,
    municipality_name  TEXT,
    municipality_code  TEXT,
    locality_name      TEXT,
    locality_code      TEXT,
    latitude           TEXT,
    longitude          TEXT,
    accuracy           TEXT
);

\copy tmp_mx_postcodes_import FROM :'csv_file' WITH (FORMAT csv, DELIMITER E'\t', QUOTE '"');

INSERT INTO mx_postcodes (
    country_code,
    postal_code,
    place_name,
    state_name,
    state_code,
    municipality_name,
    municipality_code,
    locality_name,
    locality_code,
    latitude,
    longitude,
    accuracy,
    updated_at
)
SELECT
    UPPER(TRIM(country_code))::CHAR(2),
    TRIM(postal_code),
    TRIM(place_name),
    TRIM(state_name),
    COALESCE(NULLIF(TRIM(state_code), ''), ''),
    TRIM(municipality_name),
    COALESCE(NULLIF(TRIM(municipality_code), ''), ''),
    TRIM(locality_name),
    COALESCE(NULLIF(TRIM(locality_code), ''), ''),
    NULLIF(TRIM(latitude), '')::NUMERIC(9, 6),
    NULLIF(TRIM(longitude), '')::NUMERIC(9, 6),
    NULLIF(TRIM(accuracy), '')::SMALLINT,
    NOW()
FROM tmp_mx_postcodes_import
WHERE COALESCE(TRIM(postal_code), '') <> ''
  AND COALESCE(TRIM(place_name), '') <> ''
ON CONFLICT (country_code, postal_code, place_name, state_code, municipality_code, locality_code)
DO UPDATE
SET state_name        = EXCLUDED.state_name,
    municipality_name = EXCLUDED.municipality_name,
    locality_name     = EXCLUDED.locality_name,
    latitude          = EXCLUDED.latitude,
    longitude         = EXCLUDED.longitude,
    accuracy          = EXCLUDED.accuracy,
    updated_at        = NOW();
SQL

echo "✓ Postcode import completed."
