# Changelog — memorial product spec

## v2.0 — 2026-10-14

- **MAJOR — reminders move to the server** (ADR-0001, ADR-0002, both Accepted 2026-10-14).
- Changed: FR-MEM-008 rewritten. A reminder goes into the notification list on every platform, plus
  push on iOS and Android. The push shows the relationship only. Opening it opens the entry. Delete
  or turning the reminder off stops it everywhere. If push is not allowed, the reminder is still in the
  list, with a hint. Sign-out stops push to that device.
- Changed: BR-MEM-11 (reminder moment in the account time zone; no reminder when the moment has
  already passed at add/edit time). FR-MEM-007 (now on the website too; "Có nhắc" independent of
  device permission; nearest choice for old values). BR-MEM-05…07, BR-MEM-10 now apply to Backend.
- Added: BR-MEM-13 (late reminders sent while the anniversary day lasts, never twice); NFR-MEM-04
  (reminder within one hour); `rules/reminder-moment.json` (7 moment cases + 3 delivery cases).
- Removed: FR-MEM-009 (iOS reminders app), the "Website later" variance rows, the FR-MEM-004
  scenario about editing where reminder settings are hidden.
- Changed: FR-MEM-001 opens from the main navigation (website: profile page, §3.6); FR-MEM-002 empty
  book offers no family-tree import; FR-MEM-004/005 entry deleted elsewhere is removed with a message;
  FR-MEM-010 server error shows the offline copy with a message, sign-out removes the offline copy.
- Added: §3.1.1 user-facing copy table (client question 15) for review.
- Fixed: §4.2 matrix corrected against the code (BR-MEM-04 Backend done, NFR-MEM-01 Backend n/a, iOS
  FR-MEM-001/010 drift, FR-MEM-004/012/013 in-progress, Android drift notes).
- Open: does the account-wide notification switch also hide reminders from the notification list
  (Appendix A)? Recommendation: no, push only.
- Platforms to follow up:
  - Backend: account time zone; narrow reminder choices; drop per-entry time zone and the iOS reminders
    flag; planned reminder + hourly job; notification list and push registration (Appendix B 2, 3, 6–9);
    run `reminder-moment.json` and `next-anniversary.json`.
  - Website: reminder settings, notification list, report the time zone, profile-page entry.
  - iOS: remove local scheduling and the reminders-app option; push, notification list, time zone;
    offline copy tied to the account and removed on sign-out.
  - Android: remove the iOS-only option; push, notification list, time zone; hide the family-tree option.
  - Open client specs (web#4, iOS#5, Android#4) cite v1.0 and must move to v2.0.
- Reason: product owner decisions and answers to the client spec questions, 2026-10-14.

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
