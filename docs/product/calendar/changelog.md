# Changelog — calendar product spec

## v1.0 — 2026-10-15

- First product spec for the Buddhist events calendar (retroactive mode), replacing the product
  decisions in the backend `docs/specs/features/calendar/SRS.md` v0.3, the iOS FestivalDetail spec
  and assumptions (AS-Festivals-001…002, AS-FestivalDetail-001…002, AS-Home-003), the Android local
  catalogue and the website code.
- Added BR-CAL-01…16, FR-CAL-001…010, NFR-CAL-01…03; vectors `rules/event-placement.json`
  (computed with the backend resolver), `rules/ordering.json`, `rules/countdown.json`,
  `rules/home-block.json`.
- Decisions of 2026-10-15: the calendar is public (FR-CAL-001); events get a major mark
  (BR-CAL-10); Rằm and Mùng Một come only from the catalogue (BR-CAL-05); the festival calendar shows
  every group with filters, the home block shows the Buddhist group only (BR-CAL-15); no ADR needed;
  slug `calendar`, lunar calendar screens come later as v1.x; the iOS "Events" screen belongs to the
  sangha feature. Remaining defaults from the question report (Q6–Q11, Q13) are applied and confirmed
  by review of the PR.
- Open: initial set of major events; who checks the seed content (Appendix A).
- Platforms to follow up: Backend (public route, major mark, seed in version control), Website
  ("Lễ lớn" filter, home block tiers, public access), iOS (major mark, stop local Rằm/Mùng Một,
  2100 limit, hide empty sections, offline copy, home block filter), Android (switch from the local
  catalogue to the backend, detail view, offline copy).
