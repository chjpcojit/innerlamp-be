-- US-002: Đăng nhập (login, refresh, logout).
--
-- Adds the three things US-001 deliberately left out: a device registry, a
-- per-phone brute-force counter, and the columns a rotating refresh token
-- needs. Nothing in 0001 is modified — every statement here is additive
-- (CREATE TABLE / ADD COLUMN / CREATE INDEX), so an existing database that has
-- already run 0001 picks this up without a rewrite of live data.
--
-- See docs/specs/features/login/design.md for the flows this schema serves.

-- user_devices -----------------------------------------------------------------
--
-- One row per (user, device) pair. device_id is the server-computed
-- fingerprint — SHA-256 of platform|os_version|device_model|user_agent, built in
-- repository, never sent by the client. A client-supplied id would be worthless
-- here: it would be the one fact an attacker gets to choose, and the whole point
-- of the fingerprint is to recognise "the same device again" without trusting
-- the caller's claim about which device it is.
--
-- UNIQUE(user_id, device_id) is the ON CONFLICT target of the login upsert, so
-- the same device logging in twice updates one row rather than stacking a second
-- — which is what makes AC4's "one active session per (user, device)" reachable
-- at all.
--
-- The four metadata columns are nullable except platform and device_model: the
-- use case rejects a request missing either (AC5), so they are guaranteed
-- present on insert, while os_version/app_version/device_name are optional in
-- the API and must be storable as absent.
CREATE TABLE IF NOT EXISTS user_devices (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id    VARCHAR(64) NOT NULL,
    platform     VARCHAR(32) NOT NULL,
    device_model VARCHAR(128) NOT NULL,
    os_version   VARCHAR(64),
    app_version  VARCHAR(32),
    device_name  VARCHAR(128),
    -- Server-observed, from the request rather than the body. 45 chars is the
    -- IPv6 maximum, so no address is ever truncated into a different one.
    ip_address   VARCHAR(45),
    user_agent   VARCHAR(512),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT user_devices_user_id_device_id_key UNIQUE (user_id, device_id)
);

CREATE INDEX IF NOT EXISTS idx_user_devices_user_id ON user_devices (user_id);

-- user_sessions: device binding + rotation history -----------------------------
--
-- device_id is NULLABLE on purpose, and ON DELETE SET NULL rather than CASCADE:
-- sessions created by US-001's verify-otp have no device at all, and AC12
-- requires them to refresh and log out exactly like any other session. NULL here
-- means "this session is not bound to a device", which is a legal state, not a
-- missing value. Deleting a device must not delete the session history with it
-- either — the session is the security artifact, the device row is metadata.
--
-- previous_refresh_token_hash holds the one generation before the current one.
-- It exists so a replayed token can be recognised as *reuse* rather than as
-- merely unknown (AC11): matching the previous generation is what proves the
-- caller is holding a token that was already exchanged, which is the theft
-- signal. One generation, not a full history — a token two rotations old simply
-- reads as "no session", which is the same 401 with no side effect.
ALTER TABLE user_sessions
    ADD COLUMN IF NOT EXISTS device_id BIGINT REFERENCES user_devices(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS previous_refresh_token_hash VARCHAR(64);

-- The partial unique index is what actually enforces AC4's "at most one active
-- session per (user, device)". The application also revokes the previous session
-- inside the login transaction, but that UPDATE and this INDEX are not redundant:
-- the index is the guarantee, the UPDATE is the orderly path that leaves
-- revoked_reason = 'superseded_by_login' for an operator to read. Without the
-- index, two concurrent logins from one device could both pass their UPDATE
-- (neither seeing the other's uncommitted row) and both insert.
--
-- Partial on is_revoked = false AND device_id IS NOT NULL: revoked sessions are
-- history and must not collide, and device-less US-001 sessions are excluded
-- entirely — otherwise every verify-otp session would collide with every other
-- on NULL.
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_sessions_active_device
    ON user_sessions (device_id)
    WHERE is_revoked = false AND device_id IS NOT NULL;

-- Serves FindSessionByRefreshHash's second lookup — the reuse probe, which asks
-- "is this token the generation before someone's current one?".
CREATE INDEX IF NOT EXISTS idx_user_sessions_previous_refresh_hash
    ON user_sessions (previous_refresh_token_hash)
    WHERE previous_refresh_token_hash IS NOT NULL;

-- login_attempts ---------------------------------------------------------------
--
-- The brute-force counter behind AC7. Keyed by phone_number, with NO foreign key
-- to users — deliberately. The lock must apply to numbers that have no account,
-- or the lock itself would answer "is this number registered?" for anyone
-- willing to fail five times. A FK would make that impossible to express.
--
-- In the database rather than in process memory because the counter has to be
-- correct across Cloud Run's several instances and to survive a restart: an
-- in-process map would give an attacker one fresh set of five attempts per
-- instance, and would forget every lock on deploy.
--
-- failed_count and locked_until are both written by one atomic
-- INSERT ... ON CONFLICT DO UPDATE (repository), so two concurrent failures
-- cannot both read the same starting count. last_failed_at is what makes
-- "consecutive" mean "each within failure_reset_window of the one before"
-- rather than "ever": a row whose last_failed_at is older than the window is
-- reset to 1 instead of incremented.
CREATE TABLE IF NOT EXISTS login_attempts (
    phone_number   VARCHAR(20) PRIMARY KEY,
    failed_count   INT NOT NULL DEFAULT 0,
    -- NULL while not locked. A deadline already in the past is read as unlocked
    -- by the use case; the row is not scrubbed, it is simply ignored — which is
    -- also what lets the UPSERT detect an expired lock without a second query.
    locked_until   TIMESTAMPTZ,
    last_failed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
