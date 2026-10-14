-- Profile avatar: the object key of a user's avatar image. docs/specs/features/profile/.
--
-- Stores the KEY of the object in the permanent avatar folder of the bucket
-- (e.g. "avatars/42/7f3a....jpg"), never a URL: the read address is a signed URL
-- generated per response and expires in minutes, so it is not data worth storing.
-- NULL means "no avatar" and is what every existing account has, so there is
-- nothing to backfill.
--
-- The key is new on every avatar change. A replacement therefore never reuses the
-- previous name, and "point the row at the new object, then delete the old one"
-- cannot delete the object it just pointed at.
--
-- No index and no constraint: the column is only read together with the user's own
-- row, and the key format is owned by the use case, not by the schema.
--
-- Idempotent (IF NOT EXISTS) like every other file here. Depends on 0001 (users).
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS avatar_object VARCHAR(255);
