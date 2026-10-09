-- Allow type DEVA in deities: a heavenly protector (Thiên) such as Đại Cát
-- Tường Thiên Nữ, who is neither a Buddha nor a bodhisattva but is the subject
-- of a sutra in the sutra catalogue (see sutra_deities).
--
-- 0007 creates the table with CHECK (BUDDHA, BODHISATTVA); this widens it.
-- Idempotent: the constraint is dropped if present, then re-added.
ALTER TABLE deities DROP CONSTRAINT IF EXISTS chk_deities_type;
ALTER TABLE deities
    ADD CONSTRAINT chk_deities_type CHECK (type IN ('BUDDHA', 'BODHISATTVA', 'DEVA'));
