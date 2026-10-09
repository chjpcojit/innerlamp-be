-- Sutras: split the packed `reference` and `license` strings, and add a summary.
-- Spec: docs/specs/features/sutra/ (v2.12, AC36-AC40).
--
-- `reference` now holds only the citation code ("Taishō T0450", "MN 86"). What used
-- to be packed after it gets its own column: original_title (Hán-Việt / source title,
-- or the scripture a dharani is drawn from), author (who composed the text, not who
-- translated it), han_translator, han_translation_year (CE; NULL = unknown) and
-- reference_note (any remark that fits no other column).
--
-- `summary` is a short Vietnamese description, '' until one is written.
--
-- On a translation, `license` is now a code (CC0-1.0, CC-BY-NC-4.0, PUBLIC_DOMAIN,
-- COPY_ALLOWED, UNKNOWN) checked by the seed parser, not by a CHECK, so a code can be
-- added without a migration. license_url is the URL the source gives and license_note
-- the source's own words; both '' when there is none.
--
-- This only adds columns: rows already stored keep their old `reference`/`license`
-- text and get '' / NULL in the new columns. The seed never re-runs on a table that has
-- rows (spec AC32), so existing data must be reloaded or backfilled separately.
--
-- Idempotent like every other file here.
ALTER TABLE sutras
    ADD COLUMN IF NOT EXISTS original_title       TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS author               TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS han_translator       TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS han_translation_year INTEGER
        CONSTRAINT chk_sutras_han_translation_year CHECK (han_translation_year IS NULL OR han_translation_year >= 1),
    ADD COLUMN IF NOT EXISTS reference_note       TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS summary              TEXT NOT NULL DEFAULT '';

ALTER TABLE sutra_translations
    ADD COLUMN IF NOT EXISTS license_url  TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS license_note TEXT NOT NULL DEFAULT '';
