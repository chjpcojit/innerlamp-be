-- Temple events (US-TEV-001): events owned by one temple (khoá tu, pháp hội, lễ,
-- thuyết pháp…), one-off or recurring. Spec: docs/specs/features/temple-event/
-- (design.md "Mô hình dữ liệu").
--
-- Independent of calendar_events (the shared Buddhist calendar): there is no link
-- between the two. One table, with recurrence columns; no occurrence is stored —
-- the occurrences in a window (and their lunar dates) are computed on read.
--
--   recurrence = NONE           one occurrence [starts_at, ends_at]
--   recurrence = WEEKLY         every recurrence_weekday (0 = Sunday .. 6)
--   recurrence = LUNAR_MONTHLY  every lunar month on recurrence_lunar_days (1..30;
--                               day 30 in a 29-day month lands on day 29, by the reader)
--
-- For a recurring event ends_at - starts_at is the length of each occurrence and
-- the clock time of starts_at is the time of every occurrence; recurrence_until
-- (inclusive) ends the repetition, NULL = forever. An all-day event has starts_at
-- and ends_at at 00:00 (temple_event.timezone) of its first and last day.
--
-- Populated from assets/data/temple-events.json by
-- `go run ./services/appService/cmd/seedtempleevents`. The seed upserts by id,
-- never deletes and never writes is_active (back-office turns an event off).
--
-- Idempotent (IF NOT EXISTS). Depends on 0005 (temples).

CREATE TABLE IF NOT EXISTS temple_events (
    id                    VARCHAR(64) PRIMARY KEY,         -- stable key; the seed upserts by it
    temple_id             VARCHAR(64) NOT NULL REFERENCES temples (id) ON DELETE CASCADE,
    title                 VARCHAR(255) NOT NULL
                          CONSTRAINT chk_temple_events_title CHECK (btrim(title) <> ''),
    description           TEXT NOT NULL DEFAULT '',
    kind                  VARCHAR(32) NOT NULL
                          CONSTRAINT chk_temple_events_kind
                          CHECK (kind IN ('KHOA_TU', 'PHAP_HOI', 'LE_CAU_AN', 'LE_CAU_SIEU', 'LE_HOI', 'THUYET_PHAP', 'KHAC')),
    starts_at             TIMESTAMPTZ NOT NULL,
    ends_at               TIMESTAMPTZ NOT NULL,
    is_all_day            BOOLEAN NOT NULL DEFAULT FALSE,
    status                VARCHAR(16) NOT NULL DEFAULT 'SCHEDULED'
                          CONSTRAINT chk_temple_events_status
                          CHECK (status IN ('SCHEDULED', 'POSTPONED', 'CANCELLED')),
    audience_note         TEXT NOT NULL DEFAULT '',
    contact_info          TEXT NOT NULL DEFAULT '',
    image_url             TEXT,                            -- NULL until the temple supplies an image
    recurrence            VARCHAR(16) NOT NULL DEFAULT 'NONE'
                          CONSTRAINT chk_temple_events_recurrence_kind
                          CHECK (recurrence IN ('NONE', 'LUNAR_MONTHLY', 'WEEKLY')),
    recurrence_lunar_days SMALLINT[] NOT NULL DEFAULT '{}',
    recurrence_weekday    SMALLINT,
    recurrence_until      DATE,
    is_active             BOOLEAN NOT NULL DEFAULT TRUE,   -- off = hidden; the seed never writes it
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_temple_events_span CHECK (ends_at >= starts_at),
    CONSTRAINT chk_temple_events_weekday CHECK (recurrence_weekday IS NULL OR recurrence_weekday BETWEEN 0 AND 6),
    CONSTRAINT chk_temple_events_lunar_days
        CHECK (recurrence_lunar_days <@ ARRAY[1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
                                              21, 22, 23, 24, 25, 26, 27, 28, 29, 30]::SMALLINT[]),
    CONSTRAINT chk_temple_events_recurrence CHECK (
        (recurrence = 'NONE'          AND cardinality(recurrence_lunar_days) = 0 AND recurrence_weekday IS NULL
                                      AND recurrence_until IS NULL)
        OR (recurrence = 'LUNAR_MONTHLY' AND cardinality(recurrence_lunar_days) > 0 AND recurrence_weekday IS NULL)
        OR (recurrence = 'WEEKLY'        AND cardinality(recurrence_lunar_days) = 0 AND recurrence_weekday IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_temple_events_temple_start
    ON temple_events (temple_id, starts_at) WHERE is_active;
