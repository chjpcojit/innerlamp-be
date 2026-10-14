-- Temple suggestions: structured address.
-- A suggestion used to carry the province as free text. It now carries the
-- current-scheme (two-level, since 2025-07-01) province and ward codes the
-- submitter picked from the administrative-units lookups, plus a snapshot of the
-- unit names taken from the dataset when the suggestion arrived. The snapshot is
-- what staff read, so a suggestion stays legible if the dataset is reloaded.
-- docs/specs/features/temple-suggestion/ (v3.0, N8).
--
-- There is deliberately NO foreign key to addr_new_*: the dataset is reloaded by
-- upsert and a code only means something together with its scheme. Validity is
-- checked by the use case at submission time.
--
-- Rows submitted before this migration keep their free-text `province` and get
-- NULL codes and ward; chk_temple_suggestions_address_codes allows exactly that
-- shape or a fully populated one, never a half-filled row. `province` still holds
-- the text in both cases (the snapshot name for new rows).
--
-- Idempotent (IF NOT EXISTS) like every other file here.
ALTER TABLE temple_suggestions
    ADD COLUMN IF NOT EXISTS province_code VARCHAR(16),
    ADD COLUMN IF NOT EXISTS ward_code     VARCHAR(16),
    ADD COLUMN IF NOT EXISTS ward          VARCHAR(255);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_temple_suggestions_address_codes'
    ) THEN
        ALTER TABLE temple_suggestions ADD CONSTRAINT chk_temple_suggestions_address_codes CHECK (
            (province_code IS NULL     AND ward_code IS NULL     AND ward IS NULL)
         OR (province_code IS NOT NULL AND ward_code IS NOT NULL AND ward IS NOT NULL)
        );
    END IF;
END $$;
