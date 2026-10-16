# Changelog — sutra product spec

## v1.0 — 2026-10-15

- First product spec for the sutra library, written retroactively from the backend spec (`sutra` v3.0,
  41 acceptance criteria, assumptions `A-001…A-011`, tech debt `TD-001…TD-010`), the iOS specs
  (`Sutras` v1.2, `SutraDetail` v1.4, `Reader` v1.1) with their assumptions and tech debt, the website
  and Android code, the prototype, and anh Đức's answers of 2026-10-15.
- Decisions: no ADR; the library hub and downloads are separate later specs; one topic chip + one
  duration chip (5 / 15 / 30 / 60) that combine; server search by name on all clients; Vietnamese
  titles and versions whatever the app language; four remembered font sizes shared with the settings;
  licence and unreviewed notice shown; quote of the day from the 300 Dhammapada verses, one per date on
  every platform; no audio and no simulated player; public access.
- Drift settled: filters (iOS chips, website tradition only, Android sample tags), title and version
  choice (iOS first Vietnamese, website locale), licence display (iOS as stored, website hides unknown),
  reading preferences (iOS four sizes + three backgrounds, website three sizes), simulated players on
  all three clients, quote of the day (random on the apps' home views, four fixed lines on the website).
- Backend spec drift found: `sutra` v3.0 still lists search and the audio value as out of scope though
  both exist (Appendix B item 1).
- IDs: BR-STR-01…13, FR-STR-001…012, NFR-STR-01…03. Vectors: `sutra-browse` (13 cases),
  `sutra-search-match` (13), `sutra-default-version` (6), `sutra-daily-quote` (8).
- Platforms to follow up: Backend (`spec-writer`: bring the spec to the code, sections, search term
  limit, quote if served); Website, iOS, Android (their `spec-writer`: new `sutra` spec folders,
  drift listed in §4.2). 15 open questions in Appendix A.
