# Changelog — memorial product spec

## v1.0 — 2026-10-14

- Confirmed: the product owner answered the three open questions; the spec has no open items.
- Resolved: a death on today's date shows today's anniversary with no label (BR-MEM-09); new case in
  `rules/next-anniversary.json` (now 17 cases).
- Changed: the website does not show reminder settings in this version (FR-MEM-007, BR-MEM-11 →
  Website `later`, new §3.6 row); editing there keeps the settings made in the apps (new FR-MEM-004
  scenario, Appendix B item 6).
- Resolved: portrait is JPEG, PNG or WebP, at most 5 MB, apps convert other formats to JPEG
  (FR-MEM-011, new scenario).
- Platforms to follow up: Website (hide reminder settings, keep them on save), all platforms start
  their memorial platform spec from `product.md@1.0` with `spec-writer`.
- Reason: product owner answers, 2026-10-14.

## v0.1 — 2026-10-14

- Added: first product spec for the memorial book, written retroactively from the backend spec v2.4,
  the iOS spec v1.3 and assumptions, the Android assumptions and the website code.
- Added: BR-MEM-01…12, FR-MEM-001…013, NFR-MEM-01…03.
- Added: test vectors `rules/next-anniversary.json` (16 cases), `rules/anniversary-ordinal.json` (6),
  `rules/death-date-conversion.json` (9), generated from the backend lunar converter (BR-MEM-10).
- Resolved: Q0–Q10 of the 2026-10-14 reconciliation (no ADR, existing constraint; leap month recorded,
  anniversary in the regular month; account-owned book on every platform; offline read-only in the apps;
  OS reminders app iOS-only; reminders on app devices only with fixed choices; confirm before delete;
  year of death 1900 → current year; portrait planned; family tree out of scope; backend converter as
  calendar authority).
- Open: ordinal 0 for a death on today's date (BR-MEM-09); reminder settings on the website (FR-MEM-007);
  photo size and types (FR-MEM-011).
- Platforms to follow up:
  - Backend: write `docs/specs/features/memorial/` against this spec with `spec-writer`; Appendix B items.
  - Website: move the book from browser storage to the account (BR-MEM-01, FR-MEM-001…005), leap choice
    (BR-MEM-04), year range (BR-MEM-03), delete confirmation (BR-MEM-12), run vectors.
  - iOS: keep the leap marker (BR-MEM-04), year range (BR-MEM-03), delete confirmation (BR-MEM-12),
    reminders (FR-MEM-008/009), run vectors.
  - Android: sign-in and account storage (BR-MEM-01), offline cache (FR-MEM-010), leap choice, year range,
    delete confirmation, reminders (FR-MEM-008), run vectors.
- Reason: pilot of the layer-0 product spec, user answers 2026-10-14.
