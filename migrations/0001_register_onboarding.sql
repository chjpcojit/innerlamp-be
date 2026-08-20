-- US-001: Đăng ký tài khoản (see docs/db/register-erd.md)
--
-- Assumes `users` already exists (id, email UNIQUE, password_hash,
-- dharma_name, created_at, updated_at) per the onboarding spec's note that
-- dharma_name is pre-existing. This migration only ALTERs users and adds
-- the two new tables.
--
-- Assumes the pre-existing `email` column carries the default Postgres
-- unique-constraint name `users_email_key` (inline `UNIQUE` on the column).
-- If it was created with an explicit constraint name instead, update the
-- constraint name used in repository/postgres_user_repository.go to match.

ALTER TABLE users ADD COLUMN IF NOT EXISTS phone_number VARCHAR(20) UNIQUE NOT NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS ordination_place VARCHAR(255);

CREATE TABLE IF NOT EXISTS user_devices (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL REFERENCES users(id),
    device_id     VARCHAR(64) NOT NULL,
    platform      VARCHAR(20),
    os_version    VARCHAR(50),
    app_version   VARCHAR(50),
    device_model  VARCHAR(100),
    device_name   VARCHAR(100),
    ip_address    VARCHAR(45),
    user_agent    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_devices_user_id ON user_devices(user_id);

CREATE TABLE IF NOT EXISTS user_sessions (
    id               BIGSERIAL PRIMARY KEY,
    user_id          BIGINT NOT NULL REFERENCES users(id),
    device_id        BIGINT NOT NULL REFERENCES user_devices(id),
    session_id       VARCHAR(64) NOT NULL UNIQUE,
    access_token     TEXT NOT NULL,
    refresh_token    TEXT NOT NULL UNIQUE,
    token_version    INT NOT NULL DEFAULT 1,
    is_active        BOOLEAN NOT NULL DEFAULT true,
    is_revoked       BOOLEAN NOT NULL DEFAULT false,
    revoked_at       TIMESTAMPTZ,
    revoked_reason   VARCHAR(255),
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at       TIMESTAMPTZ NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_device_id ON user_sessions(device_id);
