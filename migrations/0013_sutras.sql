-- Sutras ("kinh / kệ / chú"): the Buddhist text library for the iOS reader.
-- Spec: docs/specs/features/sutra/. Seeded once, on the service's first run, from
-- the JSON embedded in the binary (assets/data/sutras/, see
-- usecase.SutraUsecase.SeedIfEmpty). After that the database is the source of
-- truth and a restart never re-seeds.
--
-- sutras.id is the stable slug from the source data (tam-kinh, kinh-kalama, ...),
-- so an id an iOS client cached survives any reload.
--
-- sort_order is the position in assets/data/sutras/index.json (tradition, then
-- collection, then id). The source carries no popularity metric, so list order is
-- that curated order and nothing else.
--
-- reference is TEXT, not a short VARCHAR: it holds the citation as the source gives it,
-- and for lang-nghiem that is a 135-character title ("Taishō T0945 Đại Phật Đảnh ...").
--
-- titles is JSONB {"vi": "...", "en": "..."}: a short, always-read-whole map that
-- always holds "vi". duration_minutes is an editorial estimate, not a measurement,
-- and NULL when none was made (texts with no Vietnamese version).
--
-- sutra_translations holds one text of a sutra in one language. `key` is unique
-- within a sutra and is what the API selects by, because (lang, variant) is not
-- always unique. `sections` is precomputed at seed time as
-- [{"name": ..., "start_n": ..., "line_count": ...}] so the table of contents is
-- returned without scanning the lines. Its id changes if a sutra is ever reloaded,
-- so it is never exposed.
--
-- sutra_lines is one row per display line (~64,000 rows): a page of a long text
-- (the longest runs 6,967 lines) is a primary-key range read. n runs 1..line_count
-- without gaps within a translation, which is what makes that range read a page.
--
-- sutra_deities links a sutra to a Buddha / bodhisattva with a role. The deities
-- table (0007) must be seeded first; cmd/seeddeities does that.
--
-- Idempotent (IF NOT EXISTS) like every other file here.
CREATE TABLE IF NOT EXISTS sutras (
    id               VARCHAR(128) PRIMARY KEY,
    sort_order       INTEGER NOT NULL,
    tradition        VARCHAR(16) NOT NULL
                     CONSTRAINT chk_sutras_tradition CHECK (tradition IN ('THERAVADA', 'MAHAYANA', 'VAJRAYANA')),
    kind             VARCHAR(16) NOT NULL
                     CONSTRAINT chk_sutras_kind CHECK (kind IN ('SUTRA', 'VERSE', 'DHARANI')),
    collection       VARCHAR(128) NOT NULL DEFAULT '',
    reference        TEXT NOT NULL DEFAULT '',
    titles           JSONB NOT NULL,
    daily_chant      BOOLEAN NOT NULL DEFAULT false,
    purposes         TEXT[] NOT NULL DEFAULT '{}',
    duration_minutes INTEGER
                     CONSTRAINT chk_sutras_duration CHECK (duration_minutes IS NULL OR duration_minutes >= 1),
    review_status    VARCHAR(16) NOT NULL DEFAULT 'UNREVIEWED',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Serves the list (whole table, or filtered by tradition / kind) in curated order.
CREATE INDEX IF NOT EXISTS idx_sutras_order ON sutras (sort_order, id);
CREATE INDEX IF NOT EXISTS idx_sutras_tradition_kind ON sutras (tradition, kind, sort_order);
-- Serves the `purpose` filter (purposes @> ARRAY[$1]).
CREATE INDEX IF NOT EXISTS idx_sutras_purposes ON sutras USING GIN (purposes);

CREATE TABLE IF NOT EXISTS sutra_translations (
    id         BIGSERIAL PRIMARY KEY,
    sutra_id   VARCHAR(128) NOT NULL REFERENCES sutras (id) ON DELETE CASCADE,
    key        VARCHAR(64) NOT NULL,
    ordinal    INTEGER NOT NULL,
    lang       VARCHAR(8) NOT NULL,
    variant    VARCHAR(16) NOT NULL
               CONSTRAINT chk_sutra_translations_variant CHECK (variant IN ('translation', 'chant', 'meaning', 'poem')),
    translator  TEXT NOT NULL DEFAULT '',
    source_name TEXT NOT NULL DEFAULT '',
    source_url  TEXT NOT NULL DEFAULT '',
    license     TEXT NOT NULL DEFAULT '',
    note        TEXT NOT NULL DEFAULT '',
    line_count  INTEGER NOT NULL,
    sections    JSONB NOT NULL DEFAULT '[]',
    CONSTRAINT uq_sutra_translations_key UNIQUE (sutra_id, key)
);

CREATE TABLE IF NOT EXISTS sutra_lines (
    translation_id BIGINT NOT NULL REFERENCES sutra_translations (id) ON DELETE CASCADE,
    n              INTEGER NOT NULL,
    text           TEXT NOT NULL,
    section        TEXT,
    PRIMARY KEY (translation_id, n)
);

CREATE TABLE IF NOT EXISTS sutra_deities (
    sutra_id VARCHAR(128) NOT NULL REFERENCES sutras (id) ON DELETE CASCADE,
    deity_id VARCHAR(64)  NOT NULL REFERENCES deities (id),
    role     VARCHAR(16)  NOT NULL
             CONSTRAINT chk_sutra_deities_role CHECK (role IN ('SPEAKER', 'SUBJECT', 'MENTIONED')),
    PRIMARY KEY (sutra_id, deity_id)
);

-- Serves the reverse lookup (all sutras of one deity, optionally by role).
CREATE INDEX IF NOT EXISTS idx_sutra_deities_deity ON sutra_deities (deity_id, role);
