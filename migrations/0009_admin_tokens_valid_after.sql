-- Backoffice: revoke an admin's outstanding access tokens.
--
-- A token whose iat is before tokens_valid_after is rejected. Set on password
-- change; NULL means no cut-off. Chosen over a per-token blacklist because the
-- server does not track issued tokens, so one timestamp per admin is the only
-- thing that can revoke "all of them".
ALTER TABLE admins
    ADD COLUMN IF NOT EXISTS tokens_valid_after TIMESTAMPTZ;
