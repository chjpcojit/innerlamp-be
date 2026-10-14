-- Temple schedule (US-TSC-001): the standing practice routine of one temple
-- (công phu sáng, lễ Dược Sư, ...), as rows back-office can add, edit and switch
-- off. Spec: docs/specs/features/temple-schedule/ (design.md "Mô hình dữ liệu").
--
--   temple_schedules        one row per session: time of day, weekdays, free-text
--                           instructor/location, description, preparation
--   temple_schedule_steps   the session's running order; PK (schedule_id, position)
--
-- Not an event: no date, no occurrence, no status. Times are Vietnam wall-clock
-- (TIME, no zone). weekdays = '{}' means every day; 0 = Sunday .. 6 = Saturday
-- (time.Weekday). end_time may be earlier than start_time (crosses midnight) but
-- not equal to it. A step's position (not its time) is its order.
--
-- Populated from assets/data/temple-schedules.json by
-- `go run ./services/appService/cmd/seedtempleschedules`. The seed upserts
-- sessions by id, replaces the steps of the sessions in the file, never deletes a
-- session and never writes is_active (back-office turns a session off).
--
-- Idempotent (IF NOT EXISTS). Depends on 0005 (temples).

CREATE TABLE IF NOT EXISTS temple_schedules (
    id          VARCHAR(64) PRIMARY KEY,                -- stable key; the seed upserts by it
    temple_id   VARCHAR(64) NOT NULL REFERENCES temples (id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL
                CONSTRAINT chk_temple_schedules_name CHECK (btrim(name) <> ''),
    start_time  TIME NOT NULL,
    end_time    TIME,
    weekdays    SMALLINT[] NOT NULL DEFAULT '{}',       -- {} = every day
    description TEXT NOT NULL DEFAULT '',
    instructor  VARCHAR(255) NOT NULL DEFAULT '',
    location    VARCHAR(255) NOT NULL DEFAULT '',
    preparation TEXT NOT NULL DEFAULT '',
    sort_order  INTEGER NOT NULL DEFAULT 0,             -- tie-break for equal start_time
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,          -- off = schedule cancelled; the seed never writes it
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_temple_schedules_end CHECK (end_time IS NULL OR end_time <> start_time),
    CONSTRAINT chk_temple_schedules_weekdays CHECK (weekdays <@ ARRAY[0, 1, 2, 3, 4, 5, 6]::SMALLINT[])
);

CREATE INDEX IF NOT EXISTS idx_temple_schedules_temple
    ON temple_schedules (temple_id) WHERE is_active;

CREATE TABLE IF NOT EXISTS temple_schedule_steps (
    schedule_id VARCHAR(64) NOT NULL REFERENCES temple_schedules (id) ON DELETE CASCADE,
    position    SMALLINT NOT NULL
                CONSTRAINT chk_temple_schedule_steps_position CHECK (position >= 1),
    step_time   TIME,                                   -- informational; never the order
    text        TEXT NOT NULL
                CONSTRAINT chk_temple_schedule_steps_text CHECK (btrim(text) <> ''),
    PRIMARY KEY (schedule_id, position)
);
