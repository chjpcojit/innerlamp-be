# Changelog — dharmatalk product spec

Tracks version history for `docs/product/dharmatalk/product.md`.

## v0.3 — 2026-11-15

- Owner confirmed InnerLamp is non-commercial and charges users nothing. The six Archive.org talks are
  now visible, and six more were added: twelve CC BY-NC-SA 4.0 talks in total.
- Catalogue 61 → 67; hidden 25 → 19; visible 36 → 48; audio 14 → 20. Attribution recorded in
  `docs/legal/dharma-talks-attribution.md`.
- The uploader's rights are still not independently verified; the licence note keeps that caveat.

## v0.2 — 2026-11-15

- Data: six Archive.org audio talks by Hòa thượng Thích Trí Quảng added to the seed under
  CC BY-NC-SA 4.0, with the uploader's rights not independently verified; all six are hidden by the
  review flag. Catalogue 55 → 61, hidden 19 → 25, audio 8 → 14; the 36 visible talks are unchanged.
- No rule or acceptance criterion changed. Licence clearance remains an open question (Appendix A).
- Reason: content request of 2026-11-15.

## v0.1 — 2026-11-14

Initial draft, written retroactively from the implemented feature (`docs/specs/features/dharmatalk/`
requirements v2.0) because the feature had no product spec. The product decisions are still open, so
this version stays at 0.1.

- Added: `BR-DHT-01`…`BR-DHT-09`, `FR-DHT-001`…`FR-DHT-007`, `NFR-DHT-001`…`NFR-DHT-003`.
- Recorded: public read-only catalogue; one media type per talk with fixed field shapes; ordered paging;
  chapters by start time; predictable null/empty shapes; first-run seed only; invalid seed writes
  nothing; review flag hides a talk; source and licence shown.
- Recorded 13 open questions in Appendix A: listens, follow-up meditation, chapter data, seasons,
  the review flag, audio sources, licence, catalogue size, speaker display name, content updates,
  user-story codes, YouTube publication date, and whether a talk may lack a speaker.
- Platforms: Backend done; Website done; iOS planned; Android later. Licence remains a release blocker.
- Reason: documentation audit of 2026-11-14; the last user-facing feature without a product spec.
