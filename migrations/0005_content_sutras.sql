-- US-005: Xem danh sách kinh điển — creates the sutras table.
-- Schema per docs/db/content-erd.md (source: tam-dang-project-notes.md
-- mục 5, module content). sect and updated_at are indexed since they're
-- the two filters US-005 needs (AC5 sect filter, AC2/AC3/AC6 delta sync).
-- gen_random_uuid() requires the pgcrypto extension (same convention as
-- 0003_password_reset.sql).

CREATE TABLE IF NOT EXISTS sutras (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug        VARCHAR(255) NOT NULL UNIQUE,
    title       VARCHAR(255) NOT NULL,
    sect        VARCHAR(100),
    language    VARCHAR(10) NOT NULL DEFAULT 'vi',
    body        TEXT NOT NULL,
    audio_url   TEXT,
    version     INT NOT NULL DEFAULT 1,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sutras_sect ON sutras(sect);
CREATE INDEX IF NOT EXISTS idx_sutras_updated_at ON sutras(updated_at);
