-- US-007 AC3b / US-009 AC2b: Add sect column and index to mantras and bodhisattvas tables.
-- Follows the same pattern as sutras.sect in 0005_content_sutras.sql.

ALTER TABLE mantras ADD COLUMN IF NOT EXISTS sect VARCHAR(100);
CREATE INDEX IF NOT EXISTS idx_mantras_sect ON mantras(sect);

ALTER TABLE bodhisattvas ADD COLUMN IF NOT EXISTS sect VARCHAR(100);
CREATE INDEX IF NOT EXISTS idx_bodhisattvas_sect ON bodhisattvas(sect);
