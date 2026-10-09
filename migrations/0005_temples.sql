-- Temples (chùa / thiền viện / đạo tràng / tịnh xá / thiền xá) for the iOS
-- browse screens: top 10, paged list, detail. Populated from
-- assets/data/temple.json by `go run ./services/appService/cmd/seedtemples`.
--
-- id is the stable string id from the source data (temple_north_001 ...), not
-- a BIGSERIAL: the seeder upserts on it, so re-running never duplicates rows
-- and the id an iOS client cached stays valid across reseeds.
--
-- heritage_rank and type are CHECK-constrained to the values present in the
-- source data; adding a new rank is a migration plus a change to
-- domain/temple.go's ordering, not a config edit.
--
-- Idempotent (IF NOT EXISTS) like every other file here.
CREATE TABLE IF NOT EXISTS temples (
    id                  VARCHAR(64) PRIMARY KEY,
    name                VARCHAR(255) NOT NULL,
    type                VARCHAR(32) NOT NULL
                        CONSTRAINT chk_temples_type
                        CHECK (type IN ('CHUA', 'THIEN_VIEN', 'DAO_TRANG', 'TINH_XA', 'THIEN_XA')),
    region              VARCHAR(16) NOT NULL
                        CONSTRAINT chk_temples_region
                        CHECK (region IN ('north', 'central', 'south')),
    image_url           TEXT NOT NULL DEFAULT '',
    thumbnail_url       TEXT NOT NULL DEFAULT '',
    aliases             TEXT[] NOT NULL DEFAULT '{}',
    sect                VARCHAR(64) NOT NULL,
    founder             VARCHAR(255) NOT NULL DEFAULT '',
    current_abbot       VARCHAR(255) NOT NULL DEFAULT '',
    established_year    INTEGER NOT NULL,
    heritage_rank       VARCHAR(64) NOT NULL
                        CONSTRAINT chk_temples_heritage_rank
                        CHECK (heritage_rank IN (
                            'DI_SAN_THE_GIOI', 'DI_TICH_QUOC_GIA_DAC_BIET', 'KHU_DU_LICH_QUOC_GIA',
                            'DI_TICH_QUOC_GIA', 'DANH_LAM_THANG_CANH', 'DI_TICH_CAP_TINH',
                            'DI_TICH_CAP_THANH_PHO', 'CHUA_XEP_HANG')),
    -- Lower = more prominent. Derived from heritage_rank by the seeder (single
    -- mapping in domain.HeritagePriority) so top-10 / list ordering is an
    -- index-friendly column sort instead of a CASE repeated in every query.
    heritage_priority   SMALLINT NOT NULL,
    latitude            DOUBLE PRECISION NOT NULL,
    longitude           DOUBLE PRECISION NOT NULL,
    address             TEXT NOT NULL,
    operating_hours     VARCHAR(64) NOT NULL DEFAULT '',
    parking_car         BOOLEAN NOT NULL DEFAULT FALSE,
    parking_motorcycle  BOOLEAN NOT NULL DEFAULT FALSE,
    vegetarian_meals    BOOLEAN NOT NULL DEFAULT FALSE,
    summer_retreat      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Serves both ORDER BY heritage_priority, established_year, id (top + list).
CREATE INDEX IF NOT EXISTS idx_temples_ranking
    ON temples (heritage_priority, established_year, id);
