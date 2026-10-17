-- Calendar events v3.0: "major event" mark (BR-CAL-10, "Lễ lớn").
-- Spec: docs/specs/features/calendar/ (design.md, "Mô hình dữ liệu").
--
-- is_major is set by the content editor (the seed today, back-office later). Clients show major
-- events highlighted, under the "Lễ lớn" filter, and first on the home block. It is stored on the
-- event row, like is_public_holiday, so it changes without a release.
--
-- Idempotent (ADD COLUMN IF NOT EXISTS). Existing rows take false until the seed is re-run.
-- Depends on 0031.

ALTER TABLE calendar_events
    ADD COLUMN IF NOT EXISTS is_major BOOLEAN NOT NULL DEFAULT false;
