-- Backoffice: force a password change on first login.
--
-- Set TRUE for the bootstrap admin (created on first startup with a known
-- default password) and cleared when the admin sets a new password. While TRUE
-- the admin's access token carries `must_change_password` and is accepted only
-- by the change-password route.
ALTER TABLE admins
    ADD COLUMN IF NOT EXISTS must_change_password BOOLEAN NOT NULL DEFAULT FALSE;
