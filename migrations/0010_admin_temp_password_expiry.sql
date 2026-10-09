-- Backoffice: temporary passwords expire.
--
-- Set when an account is created (or its password reset) by an admin, to
-- now + account.temp_password_ttl; cleared when the admin sets their own
-- password. NULL means no expiry (the bootstrap admin, or an account that has
-- already chosen its own password). Expiry is evaluated at login, no job.
ALTER TABLE admins
    ADD COLUMN IF NOT EXISTS temp_password_expires_at TIMESTAMPTZ;
