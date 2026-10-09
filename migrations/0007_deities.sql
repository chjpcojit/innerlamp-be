-- Deities: Phật (BUDDHA) and Bồ tát (BODHISATTVA) for the iOS browse screens:
-- top 10, paged bodhisattva list, detail. Populated from
-- assets/data/phat_botat.json by `go run ./services/appService/cmd/seeddeities`.
--
-- id is the stable string id from the source data (buddha_amitabha, ...), so
-- the seeder upserts on it and an id an iOS client cached survives reseeds.
--
-- sort_order is the position in the source file, which is curated (the six
-- Buddhas, then the bodhisattvas by prominence). The source carries no
-- popularity metric, so "top" and paging both order by it instead of guessing
-- one. Reordering the file and reseeding reorders the app.
--
-- lunar_memorial_days is JSONB: a short, always-read-whole list of
-- {lunar_day, lunar_month, event_name}. Nothing queries inside it yet.
--
-- Idempotent (IF NOT EXISTS) like every other file here.
CREATE TABLE IF NOT EXISTS deities (
    id                          VARCHAR(64) PRIMARY KEY,
    type                        VARCHAR(16) NOT NULL
                                CONSTRAINT chk_deities_type CHECK (type IN ('BUDDHA', 'BODHISATTVA')),
    sort_order                  INTEGER NOT NULL,
    name_vi                     VARCHAR(255) NOT NULL,
    name_sanskrit               VARCHAR(255) NOT NULL DEFAULT '',
    name_chinese                VARCHAR(255) NOT NULL DEFAULT '',
    title_honorific             VARCHAR(255) NOT NULL DEFAULT '',
    thumbnail_url               TEXT NOT NULL DEFAULT '',
    image_url                   TEXT NOT NULL DEFAULT '',
    canonical_source            TEXT NOT NULL DEFAULT '',
    core_virtue                 TEXT NOT NULL DEFAULT '',
    pure_land                   TEXT NOT NULL DEFAULT '',
    lunar_memorial_days         JSONB NOT NULL DEFAULT '[]',
    iconography_posture         TEXT NOT NULL DEFAULT '',
    iconography_mount_or_throne TEXT NOT NULL DEFAULT '',
    sacred_artifacts            TEXT[] NOT NULL DEFAULT '{}',
    symbolic_meaning            TEXT NOT NULL DEFAULT '',
    biography_summary           TEXT NOT NULL DEFAULT '',
    great_vows                  TEXT[] NOT NULL DEFAULT '{}',
    short_mantra_sanskrit       TEXT NOT NULL DEFAULT '',
    short_mantra_sino_vietnamese TEXT NOT NULL DEFAULT '',
    long_dharani_name           TEXT NOT NULL DEFAULT '',
    dharani_phonetic_snippet    TEXT NOT NULL DEFAULT '',
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Serves top (whole table) and the per-type paged list.
CREATE INDEX IF NOT EXISTS idx_deities_type_order ON deities (type, sort_order, id);
CREATE INDEX IF NOT EXISTS idx_deities_order ON deities (sort_order, id);
