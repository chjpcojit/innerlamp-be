-- Registration: the place of ordination becomes a reference to a temple.
--
-- Until now users.ordination_place / pending_registration.ordination_place held
-- free text typed by the user. Clients now pick a temple from GET /v1/temples and
-- send its id (docs/specs/features/onboarding/ v7.0), so the column is replaced
-- by ordination_temple_id -> temples(id). NULL means "no place of ordination".
--
-- The old text column is DROPPED without a backfill (decided 2026-10-10): free
-- text cannot be mapped reliably to temples.id and the app is pre-production.
--
-- No ON DELETE action: temples is read-only reference data; deleting a temple a
-- user points at should fail loudly, not silently null or cascade.
--
-- Depends on 0001 (users, pending_registration) and 0005 (temples).
-- Idempotent (IF NOT EXISTS / IF EXISTS) like every other file here.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS ordination_temple_id VARCHAR(64) REFERENCES temples(id);
ALTER TABLE pending_registration
    ADD COLUMN IF NOT EXISTS ordination_temple_id VARCHAR(64) REFERENCES temples(id);

ALTER TABLE users DROP COLUMN IF EXISTS ordination_place;
ALTER TABLE pending_registration DROP COLUMN IF EXISTS ordination_place;
