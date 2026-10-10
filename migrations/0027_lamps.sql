-- Lamps (thắp đèn): a lamp a signed-in user lights, shown in a garden every signed-in user
-- can browse while it is lit. docs/specs/features/lamp/.
--
-- A lamp is "lit" while end_at is later than now; there is no status column and no cleanup job,
-- so an expired lamp simply stops matching. lit_at and end_at are absolute instants (UTC) set by
-- the use case from the server clock (end_at = lit_at + duration_days * 24h), never DEFAULT now():
-- the "is it lit" comparison and the stored times must come from the same clock.
--
-- Privacy: intention and prayer are only ever returned to the lighter. An anonymous lamp keeps no
-- name at all (chk_lamps_anonymous_no_name), so no other query can leak it.
-- lighter_name is the dharma name copied at lighting time; it does not follow profile changes.
--
-- user_id ON DELETE CASCADE mirrors memorials and is provisional: retention of lamps when an
-- account is deleted is not decided yet (requirements.md "Pending"). There is no delete-account
-- route yet, so nothing exercises it.
--
-- duration_days has no CHECK against {1,7,21,49}: the allowed set is config (lamp.durations_days).
--
-- Idempotent (IF NOT EXISTS) like every other file here. Depends on 0001 (users).
CREATE TABLE IF NOT EXISTS lamps (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    lamp_type     VARCHAR(16) NOT NULL
                  CONSTRAINT chk_lamps_lamp_type CHECK (lamp_type IN ('OIL_LAMP', 'LOTUS_CANDLE', 'FLOWER_LANTERN')),
    intention     VARCHAR(20) NOT NULL
                  CONSTRAINT chk_lamps_intention CHECK (intention IN ('PEACE', 'SALVATION', 'MERIT_DEDICATION', 'GRATITUDE')),
    recipient     VARCHAR(100) NOT NULL,
    prayer        VARCHAR(300),
    duration_days SMALLINT NOT NULL
                  CONSTRAINT chk_lamps_duration CHECK (duration_days > 0),
    is_anonymous  BOOLEAN NOT NULL DEFAULT false,
    lighter_name  VARCHAR(255),
    lit_at        TIMESTAMPTZ NOT NULL,
    end_at        TIMESTAMPTZ NOT NULL,
    CONSTRAINT chk_lamps_end_after_lit CHECK (end_at > lit_at),
    CONSTRAINT chk_lamps_anonymous_no_name CHECK (is_anonymous = false OR lighter_name IS NULL)
);

-- Garden, every type: ORDER BY lit_at DESC, id DESC with a keyset cursor.
CREATE INDEX IF NOT EXISTS idx_lamps_lit ON lamps (lit_at DESC, id DESC);
-- Garden filtered by lamp type.
CREATE INDEX IF NOT EXISTS idx_lamps_type_lit ON lamps (lamp_type, lit_at DESC, id DESC);
-- Per-user cap (count lit lamps) and double-tap lookup.
CREATE INDEX IF NOT EXISTS idx_lamps_user_end ON lamps (user_id, end_at);
