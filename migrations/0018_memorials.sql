-- Memorials (sổ giỗ chạp): the people a user keeps in their death-anniversary book.
-- Owned by one user; only that user may read or change a row (enforced in the repository
-- with WHERE ... AND user_id). docs/specs/features/memorial/.
--
-- Death date has two mutually exclusive forms (chk_memorials_death_form):
--   year known   -> death_date is the SOLAR date; death_day/month/leap_month are NULL.
--                   A date the user typed as lunar is converted to solar by the client.
--   year unknown -> death_date is NULL; death_day/death_month (+ death_leap_month) are in
--                   the calendar named by death_calendar. Lunar cannot be converted to
--                   solar without a year, so the server keeps what was typed.
-- death_calendar records which calendar the user typed in, so the client can show it back.
--
-- The reminder is local-only: the server stores the setting (reminder_* all set or all NULL)
-- and the app schedules notifications. add_to_os_reminder is the user's intent, not device state.
--
-- photo_object_key is NULL until the photo feature exists (tasks.md Phase 6, blocked on GCS);
-- it is created now so that feature needs no migration.
--
-- Idempotent (IF NOT EXISTS) like every other file here.
CREATE TABLE IF NOT EXISTS memorials (
    id                   BIGSERIAL PRIMARY KEY,
    user_id              BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    full_name            VARCHAR(100) NOT NULL,
    relationship         VARCHAR(24) NOT NULL
                         CONSTRAINT chk_memorials_relationship CHECK (relationship IN (
                             'GRANDFATHER_PATERNAL', 'GRANDFATHER_MATERNAL',
                             'GRANDMOTHER_PATERNAL', 'GRANDMOTHER_MATERNAL',
                             'FATHER', 'MOTHER', 'SPOUSE', 'SIBLING', 'CHILD', 'OTHER')),
    death_calendar       VARCHAR(5) NOT NULL
                         CONSTRAINT chk_memorials_death_calendar CHECK (death_calendar IN ('solar', 'lunar')),
    death_date           DATE
                         CONSTRAINT chk_memorials_death_date_min CHECK (death_date >= DATE '1900-01-01'),
    death_day            SMALLINT CONSTRAINT chk_memorials_death_day CHECK (death_day BETWEEN 1 AND 31),
    death_month          SMALLINT CONSTRAINT chk_memorials_death_month CHECK (death_month BETWEEN 1 AND 12),
    death_leap_month     BOOLEAN,
    note                 VARCHAR(1000),
    photo_object_key     TEXT,
    reminder_days_before SMALLINT
                         CONSTRAINT chk_memorials_reminder_days CHECK (reminder_days_before BETWEEN 0 AND 30),
    reminder_time        TIME,
    reminder_timezone    VARCHAR(64),
    add_to_os_reminder   BOOLEAN NOT NULL DEFAULT false,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_memorials_death_form CHECK (
        (death_date IS NOT NULL AND death_day IS NULL AND death_month IS NULL AND death_leap_month IS NULL)
        OR
        (death_date IS NULL AND death_day IS NOT NULL AND death_month IS NOT NULL AND death_leap_month IS NOT NULL)
    ),
    CONSTRAINT chk_memorials_leap_is_lunar CHECK (death_calendar = 'lunar' OR death_leap_month IS NOT TRUE),
    CONSTRAINT chk_memorials_lunar_day CHECK (death_calendar = 'solar' OR death_day IS NULL OR death_day <= 30),
    CONSTRAINT chk_memorials_reminder_all_or_none CHECK (
        (reminder_days_before IS NULL AND reminder_time IS NULL AND reminder_timezone IS NULL)
        OR
        (reminder_days_before IS NOT NULL AND reminder_time IS NOT NULL AND reminder_timezone IS NOT NULL)
    ),
    CONSTRAINT chk_memorials_os_needs_reminder CHECK (add_to_os_reminder = false OR reminder_days_before IS NOT NULL)
);

-- The list is "my memorials, newest first"; this index serves exactly that ordering.
CREATE INDEX IF NOT EXISTS idx_memorials_user ON memorials (user_id, created_at DESC, id DESC);
