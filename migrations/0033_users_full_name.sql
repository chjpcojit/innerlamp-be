-- Registration + profile: capture the user's full name.
--
-- Registration requires full_name (US-001): it waits in pending_registration
-- between request-otp and verify-otp and is copied onto users once the OTP
-- matches. The profile (GET/PATCH /v1/users/me) reads and edits it.
--
-- Both columns are NULLable: accounts created before this migration have no
-- name and we do not invent one. The "required" rule lives in the use case
-- (domain.ErrInvalidFullName), not in the schema, so old rows stay valid.
--
-- VARCHAR(300): the effective limit is profile.full_name_max_chars (config,
-- 300 in dev), capped at this column size. The ALTER makes the file safe to
-- re-run on a database that already applied an earlier VARCHAR(255) version.
--
-- Depends on 0001 (users, pending_registration). Idempotent.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS full_name VARCHAR(300);
ALTER TABLE pending_registration
    ADD COLUMN IF NOT EXISTS full_name VARCHAR(300);

ALTER TABLE users ALTER COLUMN full_name TYPE VARCHAR(300);
ALTER TABLE pending_registration ALTER COLUMN full_name TYPE VARCHAR(300);
