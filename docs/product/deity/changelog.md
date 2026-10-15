# Changelog — deity product spec

## v1.1 — 2026-10-15

- Added: 12 open questions gathered while writing the four platform specs (Appendix A items 6–17),
  each also placed inline at its rule or requirement: vía mapping per deity (A6, most blocking),
  leap-month vía wording (A7), vía with no occurrence (A8), related-sutra order (A9) and list
  details (A10), iconography labels (A11), share without a mantra (A12), deity page error copy (A13),
  featured block empty state (A14), empty filter result (A15), offline passed vía (A16), unknown type
  label (A17).
- Clarified: BR-DTY-06 — the server picks the next vía; clients send their device date as "today" and
  only count down (backend deity spec v3.0).
- Matrix: FR-DTY-004 Website done (error page already offers retry); BR-DTY-08 Website drift (editorial
  footer note missing); BR-DTY-06 note updated.
- Platforms to follow up: none until the questions are answered; then Backend (A6, A9), all clients
  (A7, A8, A10–A17).

## v1.0 — 2026-10-15

- First product spec for the Buddhas & Bodhisattvas catalogue, written retroactively from the backend
  spec (`deity` v2.3, `api-contract.md` v1.3), the iOS scene READMEs and `AS-Deity-DEVA`, the website
  and Android code, and the prototype (`buddhas`, `buddhasEmpty`, `buddha`), with anh Đức's answers of
  2026-10-15.
- Decisions: no ADR; one deva label "Chư Thiên" with no chip; backend gives 10 featured, each client
  shows what its layout fits; pages of 12; no search; vía days come from the festival calendar and the
  next one is counted down on every platform; related sutras from the sutra library's links; keep mala,
  share and iconography; drop "Báo thông tin sai"; prototype empty / error state, no static fallback;
  5 vows collapsed; apps may cache.
- IDs: BR-DTY-01…08, FR-DTY-001…011, NFR-DTY-01…02. Vectors: `deity-order` (5 cases),
  `deity-next-via` (7 cases).
- Open: content review and image rights (A1), featured criterion (A2), link in shared text (A3),
  production content load (A4), future "Chư Thiên" chip (A5).
- Platforms to follow up: Backend (page size 12, deity ↔ calendar vía link, next vía on cards and
  profile, commit the catalogue file, bring spec v2.3 in line), Website (deva label, vía from calendar
  with countdown, related sutras, mala and share actions, collapsed vows, empty state instead of the
  static fallback, retry instead of an error page), iOS (deva label, server paging and filter, vía from
  the backend instead of the on-device scan, related sutras from the links, show iconography, remove
  the report text, empty state on home and library), Android (from sample data to the backend, deva
  type, iconography, empty state, pass the mantra to the mala, real share, remove the report text).
