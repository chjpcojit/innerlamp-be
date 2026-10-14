-- Temple announcements (US-TAN-001/002): notices a temple publishes to the people
-- whose ordination temple (users.ordination_temple_id) it is. Spec:
-- docs/specs/features/temple-announcement/ (design.md "Mô hình dữ liệu").
--
-- Not an event (temple_events), not a schedule (temple_schedules), not a push.
-- A user READS the announcements of their current temple; nothing is fanned out.
-- State (ACTIVE / EXPIRED / REVOKED) is derived from expires_at / revoked_at, never stored.
-- users.receive_temple_announcements is one switch per user, shared by all devices.
--
-- Idempotent (IF NOT EXISTS). Depends on 0005 (temples) and the users table.

CREATE TABLE IF NOT EXISTS temple_issuing_bodies (
    id         VARCHAR(64) PRIMARY KEY,                       -- server-generated, prefix tib_
    temple_id  VARCHAR(64) NOT NULL REFERENCES temples (id) ON DELETE CASCADE,
    name       VARCHAR(100) NOT NULL
               CONSTRAINT chk_temple_issuing_bodies_name CHECK (btrim(name) <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_temple_issuing_bodies_temple_name
    ON temple_issuing_bodies (temple_id, lower(name));

CREATE TABLE IF NOT EXISTS temple_announcements (
    id              VARCHAR(64) PRIMARY KEY,                  -- server-generated, prefix tann_
    temple_id       VARCHAR(64) NOT NULL REFERENCES temples (id) ON DELETE CASCADE,
    issuing_body_id VARCHAR(64) NOT NULL REFERENCES temple_issuing_bodies (id) ON DELETE RESTRICT,
    title           VARCHAR(150) NOT NULL
                    CONSTRAINT chk_temple_announcements_title CHECK (btrim(title) <> ''),
    body            TEXT NOT NULL
                    CONSTRAINT chk_temple_announcements_body CHECK (btrim(body) <> ''),
    source          VARCHAR(16) NOT NULL DEFAULT 'admin'
                    CONSTRAINT chk_temple_announcements_source CHECK (source IN ('admin', 'system')),
    published_at    TIMESTAMPTZ NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,                              -- set once, on revoke
    created_by      VARCHAR(64),                              -- admin id; NULL when source = system
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_temple_announcements_window CHECK (expires_at > published_at)
);

CREATE INDEX IF NOT EXISTS idx_temple_announcements_active
    ON temple_announcements (temple_id, published_at DESC, id)
    WHERE revoked_at IS NULL;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS receive_temple_announcements BOOLEAN NOT NULL DEFAULT TRUE;
