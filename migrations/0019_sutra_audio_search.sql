-- Sutras: audio_url, an estimate for every duration, and accent-insensitive search.
-- Spec: docs/specs/features/sutra/ (v1.1).
--
-- audio_url is '' (never NULL) when a sutra has no recording, so the API field is
-- always a string. No audio exists yet; the column is filled by hand/backoffice later.
--
-- unaccent backs GET /v1/sutras?q= (case- and Vietnamese-diacritic-insensitive match
-- on titles, collection and reference). It ships with Postgres contrib, and Cloud SQL
-- allows it. unaccent maps đ/Đ to d/D.
--
-- The duration backfill fills only rows still NULL (the texts with no Vietnamese
-- version): lines of the first translation / 4, rounded up, at least 1. 4 lines a
-- minute is the median of the hand-made estimates. Rows that already have an
-- estimate are untouched, so re-applying changes nothing. The seed JSON carries the
-- same estimate for a fresh database (assets/data/sutras/).
--
-- Idempotent like every other file here.
CREATE EXTENSION IF NOT EXISTS unaccent;

ALTER TABLE sutras ADD COLUMN IF NOT EXISTS audio_url TEXT NOT NULL DEFAULT '';

UPDATE sutras s
   SET duration_minutes = GREATEST(1, CEIL(t.line_count / 4.0)::int)
  FROM (SELECT DISTINCT ON (sutra_id) sutra_id, line_count
          FROM sutra_translations ORDER BY sutra_id, ordinal) t
 WHERE t.sutra_id = s.id AND s.duration_minutes IS NULL;
