-- Temple suggestions: a temple an app user says is missing from the catalogue.
-- Submitted anonymously (name, province/city, address) as 'pending'; back-office
-- staff complete the missing data and either create the official temple
-- ('created', temple_id set) or decline it ('rejected', optional reason).
-- docs/specs/features/temple-suggestion/.
--
-- Nothing identifying the submitter is stored (no user id, no IP): the route is
-- public and the per-IP rate limit only needs the address transiently.
-- Suggestions are kept indefinitely; there is no purge job.
--
-- chk_temple_suggestions_review ties status to the review columns so a row can
-- never be half-reviewed: pending has none, created has reviewer + time + temple,
-- rejected has reviewer + time and an optional reason. A rejected row that an
-- admin later turns into a temple becomes 'created' and reject_reason is cleared.
--
-- Idempotent (IF NOT EXISTS) like every other file here.
CREATE TABLE IF NOT EXISTS temple_suggestions (
    id            BIGSERIAL PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    province      VARCHAR(100) NOT NULL,
    address       VARCHAR(500) NOT NULL,
    status        VARCHAR(16)  NOT NULL DEFAULT 'pending'
                  CONSTRAINT chk_temple_suggestions_status
                  CHECK (status IN ('pending', 'created', 'rejected')),
    temple_id     VARCHAR(64)  REFERENCES temples(id),
    reviewed_by   BIGINT       REFERENCES admins(id),
    reviewed_at   TIMESTAMPTZ,
    reject_reason VARCHAR(500),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT chk_temple_suggestions_review CHECK (
        (status = 'pending'  AND temple_id IS NULL     AND reviewed_by IS NULL     AND reviewed_at IS NULL     AND reject_reason IS NULL)
     OR (status = 'created'  AND temple_id IS NOT NULL AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL AND reject_reason IS NULL)
     OR (status = 'rejected' AND temple_id IS NULL     AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL)
    )
);

-- One official temple comes from at most one suggestion.
CREATE UNIQUE INDEX IF NOT EXISTS uq_temple_suggestions_temple
    ON temple_suggestions (temple_id) WHERE temple_id IS NOT NULL;

-- Serves the review queue: WHERE status = $1 ORDER BY created_at, id.
CREATE INDEX IF NOT EXISTS idx_temple_suggestions_queue
    ON temple_suggestions (status, created_at, id);
