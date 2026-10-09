-- Editorial content for the temple detail screen: introduction, architecture &
-- heritage highlights, and visiting tips. Populated from assets/data/temple.json
-- by `go run ./services/appService/cmd/seedtemples` (same upsert-by-id).
--
-- Defaults are empty ('' / '{}') so rows seeded before the content was
-- researched stay valid; the API returns them as empty, not null.
--
-- Idempotent (IF NOT EXISTS) like every other file here.
ALTER TABLE temples ADD COLUMN IF NOT EXISTS introduction           TEXT   NOT NULL DEFAULT '';
ALTER TABLE temples ADD COLUMN IF NOT EXISTS architecture_heritage  TEXT   NOT NULL DEFAULT '';
ALTER TABLE temples ADD COLUMN IF NOT EXISTS visiting_notes         TEXT[] NOT NULL DEFAULT '{}';
