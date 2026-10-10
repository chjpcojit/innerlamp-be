-- Temples: the province/city becomes its own column instead of being read out of
-- the free-text address. Spec: docs/specs/features/temple/ (v1.5, US-TMP-07).
--
-- address stays the full display string ("Xã Ninh Hải, Huyện Hoa Lư, Tỉnh Ninh
-- Bình"); province is the short name without administrative prefix ("Ninh Bình",
-- "Hồ Chí Minh") that clients show and search/filter by. It is explicit data:
-- cmd/seedtemples takes it from location.province in assets/data/temple.json and
-- overwrites it on every run; the API never parses it out of address.
--
-- The backfill below fills ONLY rows whose province is still '' (the default), once,
-- for a database that was seeded before this column existed. It takes the last
-- comma-separated segment of address, drops a leading "Tỉnh " / "Thành phố " /
-- "TP." / "TP " (case-insensitive, and only when something follows it), and expands
-- the abbreviation HCM to "Hồ Chí Minh". Rows that already have a province are
-- untouched, so re-applying changes nothing. It does not map old province names to
-- the post-2025 units (docs/specs/features/temple/requirements.md, ⚠️ Not confirmed).
--
-- NOT NULL DEFAULT '' (never NULL), like image_url, so the API field is always a
-- string; width 100 matches temple_suggestions.province (0022).
--
-- Depends on 0005 (temples). Idempotent like every other file here.
ALTER TABLE temples ADD COLUMN IF NOT EXISTS province VARCHAR(100) NOT NULL DEFAULT '';

UPDATE temples
   SET province = CASE
         WHEN upper(seg.stripped) = 'HCM' THEN 'Hồ Chí Minh'
         ELSE seg.stripped
       END
  FROM (
        SELECT t.id,
               btrim(regexp_replace(
                       regexp_replace(
                         regexp_replace(btrim(t.address), '[,[:space:]]+$', ''),  -- drop trailing commas/spaces
                         '^.*,[[:space:]]*', ''),                                  -- keep the last segment
                       '^(thành phố[[:space:]]+|tp\.[[:space:]]*|tp[[:space:]]+|tỉnh[[:space:]]+)(?=.)',
                       '', 'i')) AS stripped
          FROM temples t
         WHERE t.province = ''
       ) seg
 WHERE temples.id = seg.id
   AND seg.stripped <> '';
