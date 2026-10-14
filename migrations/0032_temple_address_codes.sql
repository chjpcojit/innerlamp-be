-- Temples: structured address.
-- A temple used to carry its place as two display strings: `province` (short
-- name, one of the 34 current province-level units) and `address` (the full
-- address as compiled, still naming the old district and ward). It now also
-- carries the current-scheme (two-level, since 2025-07-01) province and ward
-- codes from the administrative-units dataset, plus a snapshot of the ward's
-- full name taken from that dataset when the temple was written. `address`
-- becomes the part below ward level (house number, street, hamlet), the same
-- meaning as temple_suggestions.address; the API composes the full address
-- from address + ward + province when it reads.
-- docs/specs/features/temple/ (v1.7, US-TMP-08, AC 32-38).
--
-- There is deliberately NO foreign key to addr_new_*: the dataset is reloaded by
-- upsert and a code only means something together with its scheme (same reason
-- as 0030). Validity is checked by cmd/seedtemples and by create-temple when a
-- temple is written. docs/assumptions/temple/A-001.
--
-- Existing rows keep their `province` and `address` untouched and get NULL
-- codes and ward; chk_temples_address_codes allows exactly that shape or a fully
-- populated one, never a half-filled row. There is NO backfill: a ward cannot be
-- inferred reliably from the old free-text address, so codes are assigned by a
-- reviewed data job and arrive with the next `cmd/seedtemples` run.
--
-- Depends on 0005 (temples). Idempotent (IF NOT EXISTS) like every other file here.
ALTER TABLE temples
    ADD COLUMN IF NOT EXISTS province_code VARCHAR(16),
    ADD COLUMN IF NOT EXISTS ward_code     VARCHAR(16),
    ADD COLUMN IF NOT EXISTS ward          VARCHAR(255);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_temples_address_codes'
    ) THEN
        ALTER TABLE temples ADD CONSTRAINT chk_temples_address_codes CHECK (
            (province_code IS NULL     AND ward_code IS NULL     AND ward IS NULL)
         OR (province_code IS NOT NULL AND ward_code IS NOT NULL AND ward IS NOT NULL)
        );
    END IF;
END $$;
