-- Lamps v2.0: the recipient becomes optional (BR-LMP-06). A lamp with no recipient is dedicated
-- to all beings; clients show "Tất cả chúng sanh" (all beings) for a NULL recipient.
-- Spec: docs/specs/features/lamp/ (design.md, "Data model"; tasks.md Phase 6).
--
-- NULL is the only "no recipient" value: the use case stores a blank or whitespace-only recipient
-- as NULL, never ''. Existing rows keep their recipients. 0027 is not edited.
--
-- Idempotent: DROP NOT NULL on a column that is already nullable is a no-op. Depends on 0027.

ALTER TABLE lamps
    ALTER COLUMN recipient DROP NOT NULL;
