-- Identity moves from phone + SMS OTP to email + email code (ADR-0003).
-- docs/specs/features/onboarding/design.md, section "Migration 0037".
--
-- This file applies steps 1, 2, 3, 4 and 6 of that section only:
--   1. clear the transient OTP / pending tables
--   2. otp_codes keyed by email
--   3. otp_send_limits keyed by email
--   4. pending_registration keyed by email (+ account_exists)
--   6. users.email stored lower-case
-- DEFERRED to a later migration:
--   5. login_attempts -> email  (login Phase 1; the current login code still
--      keys login_attempts by phone_number, so it is not truncated here either)
--   7. auth_rate_limits table   (login Phase 5)
--
-- Depends on 0001, 0003. Idempotent (IF EXISTS / IF NOT EXISTS, constraints via
-- DO blocks), like every other file here.

-- 1. Transient data only, no real users yet (ADR-0003). Guarded on the old
-- otp_codes.phone_number column so the wipe happens once, on the first apply:
-- `make migrate` re-applies every file and must not clear live codes each time.
-- It also has to run before step 2/3, because ADD COLUMN ... NOT NULL without a
-- default fails on a non-empty table.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'otp_codes' AND column_name = 'phone_number'
    ) THEN
        TRUNCATE otp_codes, otp_send_limits, pending_registration;
    END IF;
END $$;

-- 2. otp_codes ----------------------------------------------------------------
-- Dropping phone_number also drops idx_otp_codes_phone_purpose; the explicit
-- DROP INDEX only documents it.
DROP INDEX IF EXISTS idx_otp_codes_phone_purpose;
ALTER TABLE otp_codes DROP COLUMN IF EXISTS phone_number;
ALTER TABLE otp_codes ADD COLUMN IF NOT EXISTS email VARCHAR(254) NOT NULL;

-- Serves FindActiveOTP: newest live row for (email, purpose).
CREATE INDEX IF NOT EXISTS idx_otp_codes_email_purpose
    ON otp_codes (email, purpose, created_at DESC);

-- 3. otp_send_limits ----------------------------------------------------------
-- Dropping phone_number drops the old UNIQUE (phone_number, purpose) with it.
ALTER TABLE otp_send_limits DROP COLUMN IF EXISTS phone_number;
ALTER TABLE otp_send_limits ADD COLUMN IF NOT EXISTS email VARCHAR(254) NOT NULL;

-- Target of INSERT ... ON CONFLICT (email, purpose) in the send policy, and the
-- lookup index. Name follows Postgres's default for an inline UNIQUE.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'otp_send_limits_email_purpose_key'
    ) THEN
        ALTER TABLE otp_send_limits
            ADD CONSTRAINT otp_send_limits_email_purpose_key UNIQUE (email, purpose);
    END IF;
END $$;

-- 4. pending_registration ------------------------------------------------------
-- phone_number stays NOT NULL (still captured at step 1) but is no longer the
-- ON CONFLICT target: email is. This reverses the 0001 rule "email deliberately
-- NOT unique": one pending row per email now, a fresh request overwrites it.
ALTER TABLE pending_registration
    DROP CONSTRAINT IF EXISTS pending_registration_phone_number_key;
DROP INDEX IF EXISTS idx_pending_registration_phone;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'pending_registration_email_key'
    ) THEN
        ALTER TABLE pending_registration
            ADD CONSTRAINT pending_registration_email_key UNIQUE (email);
    END IF;
END $$;

-- true when the email already belongs to an account: step 1 then sends the
-- "account exists" notice instead of a code, and verify-otp always fails.
ALTER TABLE pending_registration
    ADD COLUMN IF NOT EXISTS account_exists BOOLEAN NOT NULL DEFAULT false;

-- 6. users ---------------------------------------------------------------------
-- Email is the login identifier now; store it normalised so users_email_key
-- (kept, translated by name in repository/) is case-insensitive in effect.
-- Fails on a unique violation if two rows differ only in case/whitespace —
-- dev data only, resolve by hand if it happens.
UPDATE users SET email = lower(btrim(email)) WHERE email <> lower(btrim(email));

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_users_email_lowercase'
    ) THEN
        ALTER TABLE users
            ADD CONSTRAINT chk_users_email_lowercase CHECK (email = lower(email));
    END IF;
END $$;
