-- Calendar events (US-CAL-002): Buddhist, Vietnamese and folk occasions placed
-- on the calendar by a repeat RULE, not by year. Spec: docs/specs/features/calendar/
-- (design.md "Mô hình dữ liệu").
--
--   calendar_groups              labels (buddhist, vietnam, folk; back-office can add more)
--   calendar_events              one row per event: its rule, holiday flag, image, order
--   calendar_event_translations  name + descriptions + guidance per language (v1: 'vi')
--   calendar_event_groups        event <-> group links (every event has >= 1 group)
--
-- No table stores a year: the dates of a given year are computed on read from the
-- rule (solar date, lunar date, every lunar month, last day of a lunar month, or
-- solar term), so a lunar festival never needs its solar dates kept up to date.
--
-- `kind` decides which parameter columns are required; the rest must be NULL
-- (chk_calendar_events_params). is_leap_month only means something for kind =
-- 'lunar'. A longitude is a multiple of 15 degrees (the 24 solar terms).
--
-- "Every event has at least one group" cannot be a CHECK across tables; the seed
-- and the back-office enforce it. A group still referenced by an event cannot be
-- deleted (RESTRICT); deleting an event removes its translations and links.
--
-- Populated from assets/data/calendar-events.json by
-- `go run ./services/appService/cmd/seedcalendarevents`. The seed only upserts and
-- never deletes: back-office turns an event off with is_active = false.
--
-- Idempotent (IF NOT EXISTS) like every other file here. Depends on nothing.

CREATE TABLE IF NOT EXISTS calendar_groups (
    code       VARCHAR(32) PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,                -- Vietnamese; no translation table for groups yet
    sort_order INTEGER NOT NULL
               CONSTRAINT chk_calendar_groups_sort_order CHECK (sort_order >= 1)
);

CREATE TABLE IF NOT EXISTS calendar_events (
    code                 VARCHAR(64) PRIMARY KEY,    -- stable key; the seed upserts by it
    kind                 VARCHAR(16) NOT NULL
                         CONSTRAINT chk_calendar_events_kind
                         CHECK (kind IN ('solar', 'lunar', 'lunar_monthly', 'lunar_last_day', 'solar_term')),
    solar_month          SMALLINT,                   -- 1..12,  kind = solar
    solar_day            SMALLINT,                   -- 1..31,  kind = solar (29/2 allowed)
    lunar_month          SMALLINT,                   -- 1..12,  kind = lunar | lunar_last_day
    lunar_day            SMALLINT,                   -- 1..30,  kind = lunar | lunar_monthly
    is_leap_month        BOOLEAN NOT NULL DEFAULT false,
    solar_term_longitude SMALLINT,                   -- 0..345 step 15, kind = solar_term
    is_public_holiday    BOOLEAN NOT NULL DEFAULT false,  -- the event's own day only; no substitute days
    image_url            VARCHAR(500),               -- NULL until the owner supplies an image
    sort_order           INTEGER NOT NULL
                         CONSTRAINT chk_calendar_events_sort_order CHECK (sort_order >= 1),
    is_active            BOOLEAN NOT NULL DEFAULT true,   -- back-office turns off instead of deleting
    CONSTRAINT chk_calendar_events_ranges CHECK (
        (solar_month IS NULL OR solar_month BETWEEN 1 AND 12)
        AND (solar_day IS NULL OR solar_day BETWEEN 1 AND 31)
        AND (lunar_month IS NULL OR lunar_month BETWEEN 1 AND 12)
        AND (lunar_day IS NULL OR lunar_day BETWEEN 1 AND 30)
        AND (solar_term_longitude IS NULL
             OR (solar_term_longitude BETWEEN 0 AND 345 AND solar_term_longitude % 15 = 0))
    ),
    CONSTRAINT chk_calendar_events_params CHECK (
        (kind = 'solar'          AND solar_month IS NOT NULL AND solar_day IS NOT NULL
                                 AND lunar_month IS NULL AND lunar_day IS NULL
                                 AND solar_term_longitude IS NULL AND is_leap_month = false)
     OR (kind = 'lunar'          AND lunar_month IS NOT NULL AND lunar_day IS NOT NULL
                                 AND solar_month IS NULL AND solar_day IS NULL
                                 AND solar_term_longitude IS NULL)
     OR (kind = 'lunar_monthly'  AND lunar_day IS NOT NULL AND lunar_month IS NULL
                                 AND solar_month IS NULL AND solar_day IS NULL
                                 AND solar_term_longitude IS NULL AND is_leap_month = false)
     OR (kind = 'lunar_last_day' AND lunar_month IS NOT NULL AND lunar_day IS NULL
                                 AND solar_month IS NULL AND solar_day IS NULL
                                 AND solar_term_longitude IS NULL AND is_leap_month = false)
     OR (kind = 'solar_term'     AND solar_term_longitude IS NOT NULL
                                 AND solar_month IS NULL AND solar_day IS NULL
                                 AND lunar_month IS NULL AND lunar_day IS NULL AND is_leap_month = false)
    )
);

CREATE TABLE IF NOT EXISTS calendar_event_translations (
    event_code        VARCHAR(64) NOT NULL REFERENCES calendar_events (code) ON DELETE CASCADE,
    lang              VARCHAR(8)  NOT NULL,          -- v1: 'vi' only; adding a language adds rows
    name              VARCHAR(255) NOT NULL,
    short_description TEXT NOT NULL,
    long_description  TEXT NOT NULL,
    guidance          TEXT NOT NULL,                 -- lễ nghi
    PRIMARY KEY (event_code, lang)
);

CREATE TABLE IF NOT EXISTS calendar_event_groups (
    event_code VARCHAR(64) NOT NULL REFERENCES calendar_events (code) ON DELETE CASCADE,
    group_code VARCHAR(32) NOT NULL REFERENCES calendar_groups (code) ON DELETE RESTRICT,
    PRIMARY KEY (event_code, group_code)
);

-- Reverse lookup (events of a group) and the RESTRICT check on deleting a group.
CREATE INDEX IF NOT EXISTS idx_calendar_event_groups_group
    ON calendar_event_groups (group_code, event_code);
