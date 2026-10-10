-- Administrative units of Việt Nam under two schemes, plus the ward-level mapping
-- between them. Spec: docs/specs/features/administrative-units/ (v1.3).
--
--   current scheme (since 1 July 2025): province → ward/commune/special zone
--   legacy scheme  (before it):          province → district → ward/commune/town
--
-- The schemes live in separate tables because the same numeric code can denote
-- different units in each (26737 is the legacy ward Đa Kao and the current ward
-- Tân Định); a code is only meaningful together with its scheme, so
-- the two must never share a column or a lookup namespace.
--
-- Populated from assets/data/administrative-units.json by
-- `go run ./services/appService/cmd/seedadminunits`. The seeder only upserts:
-- it never deletes, so a unit absent from a newer dataset stays as it was.
--
-- sort_order is the unit's 1-based Vietnamese alphabetical rank of `name`
-- inside its parent (provinces: global), computed when the dataset is built, so
-- ORDER BY sort_order does not depend on the database collation.
--
-- The `type` columns are the dataset's own labels (province/city, ward/commune/
-- special_zone, district/town/...) and are deliberately not CHECK-constrained:
-- a new label in a future dataset must not make the whole load fail.
--
-- Idempotent (IF NOT EXISTS) like every other file here.

CREATE TABLE IF NOT EXISTS addr_dataset_versions (
    version     VARCHAR(32) PRIMARY KEY,           -- the dataset's own date, e.g. 2026-09-25
    source_note TEXT NOT NULL DEFAULT '',          -- where it came from (repository + commit)
    loaded_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---- current scheme --------------------------------------------------------

CREATE TABLE IF NOT EXISTS addr_new_provinces (
    code       VARCHAR(16) PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,              -- "Hồ Chí Minh"
    full_name  VARCHAR(150) NOT NULL,              -- "Thành phố Hồ Chí Minh"
    type       VARCHAR(32) NOT NULL,
    sort_order INTEGER NOT NULL CHECK (sort_order >= 1)
);

CREATE TABLE IF NOT EXISTS addr_new_wards (
    code          VARCHAR(16) PRIMARY KEY,
    province_code VARCHAR(16) NOT NULL REFERENCES addr_new_provinces (code),
    name          VARCHAR(100) NOT NULL,
    full_name     VARCHAR(150) NOT NULL,
    type          VARCHAR(32) NOT NULL,
    sort_order    INTEGER NOT NULL CHECK (sort_order >= 1)
);

CREATE INDEX IF NOT EXISTS idx_addr_new_wards_province
    ON addr_new_wards (province_code, sort_order, code);

-- ---- legacy scheme ---------------------------------------------------------

CREATE TABLE IF NOT EXISTS addr_old_provinces (
    code       VARCHAR(16) PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    full_name  VARCHAR(150) NOT NULL,
    type       VARCHAR(32) NOT NULL,
    sort_order INTEGER NOT NULL CHECK (sort_order >= 1)
);

CREATE TABLE IF NOT EXISTS addr_old_districts (
    code          VARCHAR(16) PRIMARY KEY,
    province_code VARCHAR(16) NOT NULL REFERENCES addr_old_provinces (code),
    name          VARCHAR(100) NOT NULL,
    full_name     VARCHAR(150) NOT NULL,
    type          VARCHAR(32) NOT NULL,
    sort_order    INTEGER NOT NULL CHECK (sort_order >= 1)
);

CREATE INDEX IF NOT EXISTS idx_addr_old_districts_province
    ON addr_old_districts (province_code, sort_order, code);

-- A legacy ward's province is the province of its district; it is not stored a
-- second time (a stored copy could disagree with the district).
CREATE TABLE IF NOT EXISTS addr_old_wards (
    code          VARCHAR(16) PRIMARY KEY,
    district_code VARCHAR(16) NOT NULL REFERENCES addr_old_districts (code),
    name          VARCHAR(100) NOT NULL,
    full_name     VARCHAR(150) NOT NULL,
    type          VARCHAR(32) NOT NULL,
    sort_order    INTEGER NOT NULL CHECK (sort_order >= 1)
);

CREATE INDEX IF NOT EXISTS idx_addr_old_wards_district
    ON addr_old_wards (district_code, sort_order, code);

-- ---- mapping between the schemes ------------------------------------------

-- Many-to-many: a legacy ward may be spread over several current wards and a
-- current ward may gather several legacy wards. mapping_type and note are the
-- dataset's own labels (merged, province_changed, ambiguous, unchanged,
-- split_population, renamed) and are passed through, not interpreted.
--
-- Province-level correspondence is NOT stored: it is derived from these rows
-- (old → new is always exactly one province; new → old may be several).
CREATE TABLE IF NOT EXISTS addr_ward_mappings (
    old_ward_code VARCHAR(16) NOT NULL REFERENCES addr_old_wards (code),
    new_ward_code VARCHAR(16) NOT NULL REFERENCES addr_new_wards (code),
    mapping_type  VARCHAR(32) NOT NULL,
    note          TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (old_ward_code, new_ward_code)
);

-- The primary key serves old → new; this serves new → old.
CREATE INDEX IF NOT EXISTS idx_addr_ward_mappings_new
    ON addr_ward_mappings (new_ward_code);
