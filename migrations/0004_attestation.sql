-- US-006 attestation: Play Integrity (Android) + App Attest (iOS).
--
-- See docs/specs/features/attestation/design.md for the flow this schema serves,
-- and migrations/README.md for why this file is applied with `psql -f` rather
-- than through scripts/migrate.sh (every statement is idempotent
-- CREATE ... IF NOT EXISTS / CREATE INDEX IF NOT EXISTS).
--
-- Two tables, one per side of the platform asymmetry (design.md §Schema):
--   - attestation_challenges: single-use nonces with a TTL, shared by both the
--     verify flow (purpose = 'attest') and the assertion middleware
--     (purpose = 'assert'). Android needs only this table — it has no durable
--     key, every verify is a fresh integrity token bound to a challenge.
--   - attestation_keys: durable per-device iOS App Attest keys (key_id, the
--     ES256 public key, the receipt kept for the deferred fraud-risk lookup,
--     and the largest seen sign counter for replay protection). Android never
--     writes a row here.

-- attestation_challenges -----------------------------------------------------
--
-- challenge is the base64 encoding of ≥16 cryptographically-random bytes and is
-- the lookup key, so it is UNIQUE. purpose discriminates which flow a nonce
-- belongs to; FindChallenge always filters on it so a nonce minted for verify
-- cannot be consumed by the assertion middleware and vice versa.
--
-- used_at is set on first consume (success OR negative verdict — the nonce is
-- one-time, design.md decision). expires_at = now() + challenge_ttl, computed by
-- the use case and passed in rather than defaulted here (same reasoning as
-- user_sessions.expires_at).
CREATE TABLE IF NOT EXISTS attestation_challenges (
    id         BIGSERIAL PRIMARY KEY,
    challenge  VARCHAR(255) NOT NULL UNIQUE,
    platform   VARCHAR(16) NOT NULL
               CONSTRAINT chk_attestation_challenges_platform
               CHECK (platform IN ('android', 'ios')),
    purpose    VARCHAR(16) NOT NULL
               CONSTRAINT chk_attestation_challenges_purpose
               CHECK (purpose IN ('attest', 'assert')),
    used_at    TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Serves the deferred cleanup job's expiry scan; FindChallenge looks up by
-- challenge (unique), so this index is for deletion, not lookup.
CREATE INDEX IF NOT EXISTS idx_attestation_challenges_expires
    ON attestation_challenges (expires_at);

-- attestation_keys -----------------------------------------------------------
--
-- key_id is the base64 credentialId the client presents. It is the hash of the
-- public key, so it is naturally UNIQUE — which is what makes re-registration
-- (an idempotent retry, AC10) an upsert instead of a duplicate error, and what
-- lets one device hold several keys (each reinstall mints a new key_id).
--
-- public_key is the COSE-encoded ES256 key used to verify later assertions.
-- receipt is stored for the (deferred) fraud-risk lookup and is NOT sent to
-- Apple in this version. sign_counter is the largest authenticator counter seen,
-- the value that makes replay/rollback detectable (AC11/AC12).
--
-- user_id is nullable and device-scoped (not account-scoped): a key belongs to
-- a device, and is associated with a user only later, if at all. ON DELETE SET
-- NULL keeps the key alive if the user row is removed.
CREATE TABLE IF NOT EXISTS attestation_keys (
    id           BIGSERIAL PRIMARY KEY,
    key_id       VARCHAR(255) NOT NULL,
    platform     VARCHAR(16) NOT NULL
                 CONSTRAINT chk_attestation_keys_platform
                 CHECK (platform IN ('ios')),
    public_key   BYTEA NOT NULL,
    receipt      BYTEA,
    sign_counter BIGINT NOT NULL DEFAULT 0,
    environment  VARCHAR(16) NOT NULL
                 CONSTRAINT chk_attestation_keys_environment
                 CHECK (environment IN ('development', 'production')),
    user_id      BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Target of SaveKey's ON CONFLICT (key_id) DO UPDATE, and the lookup index for
-- FindKey / UpdateKeyCounter.
CREATE UNIQUE INDEX IF NOT EXISTS idx_attestation_keys_key_id
    ON attestation_keys (key_id);
