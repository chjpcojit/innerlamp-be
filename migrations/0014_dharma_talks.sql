-- Dharma talks ("pháp thoại"): a teacher's talk, distinct from a sutra. Read-only
-- public catalogue for the paged list and the talk screen. Seeded once from
-- assets/data/dharma_talks.json when the table is empty on service start (see
-- usecase.DharmaTalkUsecase.SeedIfEmpty); after that the DB is the source of
-- truth and restarts never overwrite it.
--
-- id is the stable slug from the source data, so an id an iOS client cached
-- survives any reseed. sort_order is the position in the source file.
--
-- media_type picks the client layout: 'youtube' (embedded via youtube_id),
-- 'audio' (played from audio_url) or 'text' (read only). The CHECK keeps the
-- two media columns consistent with it even for hand-written SQL.
--
-- needs_review = true hides a row from every API read (it is still stored, and
-- still counted by the seed's emptiness check). Clear it by hand once reviewed.
--
-- duration_minutes, published_date, season, follow_up_meditation are NULLable:
-- the API returns JSON null, not a fake value. chapters is a JSON array,
-- '[]' when none: [{"start_seconds": 0, "title": "..."}].
--
-- Idempotent (IF NOT EXISTS) like every other file here.
CREATE TABLE IF NOT EXISTS dharma_talks (
    id                   VARCHAR(128) PRIMARY KEY,
    sort_order           INTEGER NOT NULL,
    title                TEXT NOT NULL,
    speaker              TEXT NOT NULL,
    speaker_initials     VARCHAR(8) NOT NULL DEFAULT '',
    category             VARCHAR(128) NOT NULL DEFAULT '',
    season               VARCHAR(64),
    published_date       DATE,
    duration_minutes     INTEGER CHECK (duration_minutes IS NULL OR duration_minutes > 0),
    summary              TEXT NOT NULL DEFAULT '',
    key_points           TEXT[] NOT NULL DEFAULT '{}',
    quote                TEXT NOT NULL DEFAULT '',
    quote_source         TEXT NOT NULL DEFAULT '',
    tags                 TEXT[] NOT NULL DEFAULT '{}',
    language             VARCHAR(8) NOT NULL DEFAULT 'vi',
    media_type           VARCHAR(16) NOT NULL CHECK (media_type IN ('youtube', 'audio', 'text')),
    youtube_id           VARCHAR(16),
    audio_url            TEXT,
    chapters             JSONB NOT NULL DEFAULT '[]',
    follow_up_meditation JSONB,
    needs_review         BOOLEAN NOT NULL DEFAULT false,
    review_note          TEXT NOT NULL DEFAULT '',
    source_name          TEXT NOT NULL DEFAULT '',
    source_url           TEXT NOT NULL DEFAULT '',
    license              TEXT NOT NULL DEFAULT '',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT dharma_talks_media_consistent CHECK (
        (media_type = 'youtube' AND youtube_id IS NOT NULL AND audio_url IS NULL) OR
        (media_type = 'audio'   AND audio_url  IS NOT NULL AND youtube_id IS NULL) OR
        (media_type = 'text'    AND youtube_id IS NULL     AND audio_url IS NULL)
    )
);

-- Serves the paged list (same ORDER BY as the repository).
CREATE INDEX IF NOT EXISTS idx_dharma_talks_order ON dharma_talks (sort_order, id);
