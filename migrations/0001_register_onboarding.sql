-- US-001: Đăng ký tài khoản (OTP qua SMS).
--
-- See docs/specs/features/onboarding/design.md for the flow this schema serves,
-- and migrations/README.md for why this file is applied with `psql -f` rather
-- than through scripts/migrate.sh.
--
-- Central invariant: no users row ever exists in an unverified state. The
-- account payload waits in pending_registration (password already bcrypt-hashed)
-- until the phone number is verified, at which point verify-otp inserts the
-- users row and deletes the pending row in one transaction. There is therefore
-- no email_verified_at IS NULL account sitting around waiting to be confirmed —
-- only a pending_registration row, which expires on its own.
--
-- Two-step registration is also why pending_registration exists as a table at
-- all: the email and password are captured at step 1 but the account is only
-- created at step 2, so between the steps something has to hold them.

-- users ----------------------------------------------------------------------
--
-- Created in full rather than ALTERed onto a pre-existing table: the previous
-- migrations assumed a `users` table shipped by an earlier feature, and that
-- assumption is what this file removes.
--
-- email UNIQUE is enforced here by Postgres's default constraint name
-- `users_email_key` (inline UNIQUE on the column). repository/ translates a
-- pq error 23505 by constraint NAME, so renaming this constraint — or creating
-- an equivalent one under a different name — silently turns a race-loser's 409
-- into a 500. Same for users_phone_number_key below.
CREATE TABLE IF NOT EXISTS users (
    id                BIGSERIAL PRIMARY KEY,
    email             VARCHAR(255) NOT NULL UNIQUE,
    -- NULL on purpose at registration: US-001 verifies the *phone number* by
    -- OTP, not the email, so this column records a step that does not exist
    -- yet. Login (US-002) rejects accounts whose email is still NULL here.
    email_verified_at TIMESTAMPTZ,
    phone_number      VARCHAR(20) NOT NULL UNIQUE,
    password_hash     VARCHAR(255) NOT NULL,
    -- Optional profile fields (AC9). NULL, never "", so "not given" reads back
    -- as absent rather than as a value the user cannot distinguish from empty.
    dharma_name       VARCHAR(255),
    ordination_place  VARCHAR(255),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- otp_codes ------------------------------------------------------------------
--
-- One table for every OTP flow, discriminated by purpose, so US-002 (login) and
-- US-003 (reset password) reuse it instead of each growing their own table —
-- and so the rate limiting below (attempts, resend_count, locked_until) is
-- written once.
--
-- otp_hash is a bcrypt hash, never the six-digit code. A 10^6 keyspace falls to
-- a precomputed table in seconds under a fast hash, so the OTP is hashed the
-- way a password is; the short lifetime does not make a fast hash safe.
--
-- The row carries its own rate limiting (AC5's resend cap, AC12's lockout), so
-- neither needs a second store and both survive a process restart.
CREATE TABLE IF NOT EXISTS otp_codes (
    id           BIGSERIAL PRIMARY KEY,
    phone_number VARCHAR(20) NOT NULL,
    -- The CHECK lists all four purposes, not just the one US-001 implements, so
    -- adding login/reset_password later does not need an ALTER on this
    -- constraint. The constant set lives in domain/otp.go.
    purpose      VARCHAR(20) NOT NULL
                 CONSTRAINT chk_otp_codes_purpose
                 CHECK (purpose IN ('register', 'login', 'reset_password', 'verify_email')),
    otp_hash     VARCHAR(255) NOT NULL,
    -- Count of wrong guesses; at the limit the use case locks the row.
    attempts     INT NOT NULL DEFAULT 0,
    -- Carried forward across resends, never reset — that is what makes the cap
    -- a cap. A fresh request-otp inherits the previous row's count.
    resend_count INT NOT NULL DEFAULT 0,
    -- Set together with used_at when the attempt limit is hit, so a locked row
    -- can never be replayed and reads as "no active OTP" to the finder.
    locked_until TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ NOT NULL,
    -- Set on success and on lock. Both mean "this code is done".
    used_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Serves FindActiveOTP: newest live row for (number, purpose). Partial on the
-- unspent rows because that read only ever wants one, and the table grows with
-- every issued code.
--
-- Guarded on the column: 0037 replaces otp_codes.phone_number with email (and
-- this index with idx_otp_codes_email_purpose), and every file is re-applied
-- on each `make migrate`, so an unguarded CREATE INDEX would fail on a
-- database that already ran 0037.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'otp_codes' AND column_name = 'phone_number'
    ) THEN
        CREATE INDEX IF NOT EXISTS idx_otp_codes_phone_purpose
            ON otp_codes (phone_number, purpose, created_at DESC);
    END IF;
END $$;

-- pending_registration -------------------------------------------------------
--
-- Holds the account payload between step 1 and step 2. phone_number is UNIQUE
-- because it is the ON CONFLICT target: a fresh request for the same number
-- overwrites the previous payload rather than stacking a second one, so an
-- abandoned attempt cannot leave two competing passwords behind.
--
-- email is deliberately NOT unique here. Only a completed users row is
-- authoritative for email uniqueness; two people may hold pending
-- registrations for the same unverified address, and the account insert
-- re-checks inside its own transaction. Making this column UNIQUE would instead
-- reject the second request with a database error the caller cannot act on.
--
-- expires_at equals the OTP's TTL and is pushed forward on each resend, so a
-- pending row and its code always expire together and an abandoned
-- registration disappears without a cleanup job. Nothing here is durable state:
-- a missing or expired row means "start over from step 1".
CREATE TABLE IF NOT EXISTS pending_registration (
    id               BIGSERIAL PRIMARY KEY,
    phone_number     VARCHAR(20) NOT NULL UNIQUE,
    email            VARCHAR(255) NOT NULL,
    password_hash    VARCHAR(255) NOT NULL,
    dharma_name      VARCHAR(255),
    ordination_place VARCHAR(255),
    expires_at       TIMESTAMPTZ NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_pending_registration_phone
    ON pending_registration (phone_number);

-- user_sessions --------------------------------------------------------------
--
-- Created at registration, so verifying the OTP is also logging in (AC2).
--
-- refresh_token_hash is a SHA-256 hex digest of the refresh token, never the
-- token itself: the token is a bearer credential, and a stolen database dump
-- must not yield one that works. The access token is not stored at all — it is
-- short-lived and verified by signature, so there is nothing to keep.
--
-- No device_id and no user_devices table in this story. Sessions are not tied
-- to a device yet; design.md lists that as a known gap, and adding a NOT NULL
-- device reference now would mean inventing data the client does not send.
CREATE TABLE IF NOT EXISTS user_sessions (
    id                 BIGSERIAL PRIMARY KEY,
    user_id            BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- Opaque identifier for the session, distinct from any token.
    session_id         VARCHAR(64) NOT NULL UNIQUE,
    refresh_token_hash VARCHAR(64) NOT NULL UNIQUE,
    token_version      INT NOT NULL DEFAULT 1,
    is_active          BOOLEAN NOT NULL DEFAULT true,
    is_revoked         BOOLEAN NOT NULL DEFAULT false,
    revoked_at         TIMESTAMPTZ,
    revoked_reason     VARCHAR(255),
    last_activity_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- now() + JWTConfig.RefreshTokenExpiration (168h in config.dev.yaml),
    -- passed in by the use case rather than defaulted here: the stored expiry
    -- and the refresh token's own expiry are the same number by construction.
    expires_at         TIMESTAMPTZ NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions (user_id);
