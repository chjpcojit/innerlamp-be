# Changelog — deity product spec

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
