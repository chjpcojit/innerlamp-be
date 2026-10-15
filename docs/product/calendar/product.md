# Product Spec — Buddhist events calendar (Sự kiện Phật giáo)

| Field | Value |
|---|---|
| Document type | Product spec — Software Requirements Specification, IEEE 29148-2018 structure |
| Feature | Calendar — Buddhist events / Sự kiện Phật giáo (`calendar`) |
| Feature code | `CAL` — IDs are `BR-CAL-NN`, `FR-CAL-NNN`, `NFR-CAL-NN` |
| Product | InnerLamp |
| Platforms in scope | Backend, Website, iOS, Android |
| Version | 1.0 |
| Last updated | 2026-10-15 |
| Status | draft for review — decisions of 2026-10-15 applied; the open items in Appendix A are confirmed by review of this PR |
| Owner / author | Product owner: anh Đức · written by Claude (`ba-writer`, retroactive mode) |
| API contract | `docs/specs/api/openapi.yaml` (owned by the backend; this spec never restates it) |
| Architecture decisions | None needed (decision 2026-10-15). The event data model (repeat rules, dates computed per year) was decided 2026-10-11 and is recorded as a constraint in §2.5 |
| Downstream | Backend: `docs/specs/features/calendar/` (`spec-writer`) · Website: `Docs/specs/features/calendar/` · iOS: `Docs/Specs/Calendar/` · Android: `Docs/Specs/Calendar/` |

> See `changelog.md` in this folder for the version history. Platform specs cite this file as
> `docs/product/calendar/product.md@<version>` plus the IDs they implement.

---

## 1. Introduction

### 1.1 Purpose

This document is the single source of product truth for the Buddhist events calendar on every
InnerLamp platform. It states what a user can observe and the business rules behind it, so the
backend, the website, the iOS app and the Android app show the same events on the same days. It
replaces the product decisions scattered across the backend calendar SRS and spec, the iOS
FestivalDetail spec and assumptions, the Android local event catalogue and the website code.

### 1.2 Scope

**In scope (v1.0):**
- A shared catalogue of recurring occasions: Buddhist festivals and observances (Phật đản, Vu Lan,
  vía Phật / Bồ Tát, Rằm, Mùng Một…), Vietnamese public holidays and folk festivals.
- The festival calendar view (Lịch lễ Phật giáo): every occurrence of a year, by month, with both
  the solar and the lunar date, how far away it is, and filters.
- The event detail view: what the occasion is, when it falls, and what Buddhists usually do.
- The "Buddhist events" block on the home screen.
- Everyone can read the calendar, signed in or not.

**Out of scope:**
- The lunar calendar screen, the lunar day detail and the daily Buddha picture. They share this
  feature slug and will be added to this spec as v1.x when they are rewritten.
- Temple events (sự kiện của chùa: retreats, courses, registration). Those belong to the sangha
  feature, keyed off the user's ordination temple.
- Reminders for an event ("Nhắc tôi ngày này"). When built, they use the notification list and push
  of ADR-0001.
- "Recommended sutras" and "Temples holding this event" on the detail view (no data source yet).
- Back-office screens to edit events (the data must allow it; the screens are a separate spec).
- Substitute days off (nghỉ bù) and per-year government changes to holidays.
- Personal events created by a user.
- Event content in languages other than Vietnamese.

### 1.3 Product overview

InnerLamp (Tâm Đăng) is a Vietnamese Buddhist practice app: practice tools, a lunar calendar, a
library, temples and family features. Most Buddhist festivals fall on a fixed lunar date, so their
solar date changes every year and people need the app to tell them "Phật đản is on Sunday 31 May
this year". This feature is the shared calendar of those occasions. It is reached from the home
screen (the "Sự kiện Phật giáo" block and its "Tất cả" link), from the temple tab, from a deity's
page, and from the website's events page.

### 1.4 Glossary

| Term | Definition | Not to be confused with |
|---|---|---|
| Sự kiện (event) | One occasion in the catalogue, e.g. "Phật đản", with its content and its repeat rule | An occurrence |
| Lần diễn ra (occurrence) | One event on one solar day, e.g. Phật đản on 2026-05-31 | The event itself |
| Quy tắc lặp (repeat rule) | How an event's date is found in any year (BR-CAL-01) | A date stored per year (never stored) |
| Lịch lễ Phật giáo (festival calendar) | The view listing every occurrence of a year by month | The lunar calendar screen (out of scope) |
| Nhóm (group) | A label an event belongs to: Phật giáo, Việt Nam, Dân gian; more can be added as data (BR-CAL-09) | Lễ lớn |
| Lễ lớn (major event) | An event marked as one of the main festivals of the year (BR-CAL-10) | Ngày nghỉ lễ |
| Ngày nghỉ lễ (public holiday) | An event whose own day is a public day off in Vietnam (BR-CAL-11) | Lễ lớn |
| Ngày trai giới (observance day) | Rằm (lunar 15th) and Mùng Một (lunar 1st) of every lunar month (BR-CAL-05) | A named festival on the same day |
| Âm lịch (lunar calendar) | The Vietnamese lunisolar calendar, computed at UTC+7 (BR-CAL-07) | The Chinese calendar, which can differ by a day or a month |
| Dương lịch (solar calendar) | The Gregorian calendar | — |
| Tháng nhuận (leap month) | An extra lunar month inserted in some years; it repeats the number of the month before it | A leap year in the solar calendar |
| Tháng thiếu / tháng đủ (short / full month) | A lunar month of 29 / 30 days; the length changes from year to year | — |
| Tiết khí (solar term) | One of 24 points of the sun's yearly path, every 15° of ecliptic longitude, e.g. Thanh minh (15°), Đông chí (270°) | A lunar date |
| Lễ nghi (guidance) | "What Buddhists usually do": a list of practices for the day | The description |

### 1.5 References

- Design prototype: `InnerLamp Prototype.dc.html` in the iOS/Android repos
  (`Docs/innerlamp-project/project/`), screens "Lịch lễ Phật giáo" (`festVals`) and "Chi tiết ngày
  lễ" (`festDetail`); `Mo-ta-tinh-nang.md` at `Docs/innerlamp-project/project/docs/` (Android copy),
  sections "Trang chủ" item 4 and "Chùa (tab)" common part.
- Prior documents:
  - Backend: `docs/specs/features/calendar/SRS.md` v0.3, `requirements.md` v2.1, `design.md`,
    `api-contract.md`; `docs/knowledge/features/calendar.md`, `docs/knowledge/api/calendar-events.md`.
  - iOS: `Docs/Specs/FestivalDetail/requirements.md` v1.0; `Docs/Assumptions/Festivals/AS-Festivals-001…002`,
    `Docs/Assumptions/FestivalDetail/AS-FestivalDetail-001…002`, `Docs/Assumptions/Home/AS-Home-003`;
    `Docs/Tech-Debt/FestivalDetail/TD-FestivalDetail-001…002`, `Docs/Tech-Debt/Home/TD-Home-003`;
    the Festivals screen README (no spec folder).
  - Android: no spec; local catalogue in `GetBuddhistEventMonthsUseCase`, `GetFestivalCalendarUseCase`;
    `Docs/home_assumptions.md` #11.
  - Website: no spec; `src/lib/server/api/calendar.ts`, routes `events/` and `events/[code]/`.
  - Reconciliation report: `/mnt/project-files/reports/product/calendar/2026-10-15-cau-hoi-truoc-khi-viet.md`.
- Platform folder names: Backend `calendar` · Website `events` (routes) · iOS `Festivals`,
  `FestivalDetail`, Home carousel · Android `festivals`. New platform specs use `calendar` / `Calendar`.

---

## 2. Overall description

### 2.1 Product perspective

Retroactive spec of a feature already built in part on every platform (see §4.2). It touches:
- **Home** (`home`): the "Buddhist events" block.
- **Sangha / temple tab**: links to the festival calendar; temple events are a different feature.
- **Deity** (`deity`): links to the festival calendar.
- **Lunar calendar** (this slug, later version): the same Vietnamese lunar conversion.
- **Memorial** (`memorial`): uses the same lunar conversion rules (BR-MEM-10 ≡ BR-CAL-07).
- **Back office**: will edit events, groups and flags later.

No external system is involved.

### 2.2 Product functions

- Read the calendar without signing in (FR-CAL-001)
- Browse the festival calendar of a year, by month, and load later years (FR-CAL-002, FR-CAL-004)
- Filter by major events or by group (FR-CAL-003)
- Read one event's detail (FR-CAL-005, FR-CAL-009)
- See upcoming Buddhist events on the home screen (FR-CAL-006)
- Keep reading the last loaded calendar offline (FR-CAL-007)
- Recover from load failures (FR-CAL-008)
- Change event content without an app release (FR-CAL-010)

### 2.3 User classes and characteristics

| Actor | Description | Permissions |
|---|---|---|
| Guest | Not signed in (guest mode is kept, ADR-0003) | Reads the whole calendar (FR-CAL-001) |
| Signed-in user | Any account | Same as guest; nothing in this feature is personal |
| Content editor | The product owner, later through the back office | Changes events, groups and flags (FR-CAL-010) |

### 2.4 Operating environment

| Platform | In scope | Notes |
|---|---|---|
| Backend | yes | Holds the catalogue, computes every occurrence of a requested year (BR-CAL-01…07), serves it to everyone |
| Website | yes | Festival calendar, detail with previous/next event, home block |
| iOS | yes | Festival calendar, detail, home block, offline reading |
| Android | yes | Festival calendar, detail, home block, offline reading (today: local catalogue, see §4.2) |

Network expectation: online to load a year; every platform shows the last loaded copy of a year
when the backend cannot be reached (FR-CAL-007).

### 2.5 Constraints

- **Dates come from repeat rules, not from per-year lists** (decision 2026-10-11). The catalogue keeps
  one repeat rule per event and InnerLamp computes the dates of the requested year, so a lunar
  festival never needs its solar dates maintained by hand. All platforms show the dates the backend
  computes; no client computes event dates of its own (decision 2026-10-15, BR-CAL-05).
- Years 1900–2100 only: the range in which the lunar conversion is verified (NFR-CAL-01).
- Religious content: names, descriptions and guidance follow the common calendar of the Vietnam
  Buddhist Sangha (GHPGVN) and are reviewed by the product owner before publishing.
- Vietnamese only in v1.0 (§1.2); the catalogue must allow adding a language later without
  changing existing events.

### 2.6 User documentation

N/A — each view explains itself; there is no separate help content.

### 2.7 Assumptions and dependencies

- **Assumptions:**
  - Every event's date follows from exactly one repeat rule; no year needs an exception.
  - The catalogue is small (tens to low hundreds of events), so one year is delivered as a whole.
- **Dependencies:**
  - The initial catalogue is collected by the team in a seed file and loaded into the backend.
    > ⚠️ Not confirmed: the seed file exists only on the product owner's computer (it is excluded
    > from version control) and its content has not been checked against the GHPGVN calendar —
    > who checks it, and by when? — blocks the "content reviewed" constraint in §2.5 and the initial
    > set of major events (BR-CAL-10); all platforms.
  - Event images: none supplied yet; a default image is shown until they are (BR-CAL-14).
  - Back office (later) for editing events, groups and flags.

---

## 3. Specific requirements

### 3.1 External interface requirements

#### 3.1.1 User interfaces

- **Festival calendar (Lịch lễ Phật giáo).** A list of months; under each month, one row per
  occurrence: solar day and weekday, event name, lunar date, countdown (BR-CAL-13), a mark for major
  events, and the group label(s). Filter chips at the top: "Tất cả", "Lễ lớn", then one per group.
  Opens on the current month of the current year (FR-CAL-002). Choosing a row opens the event detail.
- **Event detail (Chi tiết ngày lễ).** Image, group label(s) and major mark, name, lunar date, a
  solar date card with weekday and countdown, the description as paragraphs, and "Phật tử thường làm
  gì" as a numbered list (FR-CAL-005).
- **Home block (Sự kiện Phật giáo).** One card per month for the next six months, each with up to
  three events and "+N sự kiện khác"; a "Tất cả" link (FR-CAL-006).
- Entry points: home block, temple tab ("Lịch lễ"), deity page, website events page.

Layout and styling are defined by the design prototype, not here.

#### 3.1.2 Hardware interfaces

N/A — no device capability is used.

#### 3.1.3 Software interfaces

The product needs one capability from the backend: "give me every occurrence of solar year Y, with
each event's content, groups, major and public-holiday flags, and both dates", readable without
signing in. The contract is `docs/specs/api/openapi.yaml`; Appendix B lists what it must change.

#### 3.1.4 Communication interfaces

N/A — no message is sent to the user in v1.0.

### 3.2 Business rules

**BR-CAL-01 — One repeat rule per event**
- **Rule:** Every event has exactly one repeat rule, of one of five kinds: a fixed solar date
  (BR-CAL-03), a fixed lunar date (BR-CAL-04), every lunar month on a given day (BR-CAL-05), the last
  day of a given lunar month (BR-CAL-04), or a solar term (BR-CAL-06). The dates of a year are
  computed from the rule; no date is stored per year.
- **Examples:** 30/4 → fixed solar; Phật đản → lunar 15/4; Rằm → every lunar month, day 15;
  Giao thừa → last day of lunar month 12; Thanh minh → solar term 15°.
- **Computational:** yes → `rules/event-placement.json`
- **Applies to:** Backend (clients display what it computes)
- **Source:** decision 2026-10-11 (backend SRS FR-002…FR-005, FR-015)

**BR-CAL-02 — An occurrence belongs to the year of its solar date**
- **Rule:** A year's calendar holds every occurrence whose solar date falls between 1 January and
  31 December of that year, whatever its lunar year.
- **Examples:** Táo quân (lunar 23/12 of lunar year 2025) falls on 2026-02-10 → in the 2026
  calendar; lunar 23/12 of lunar year 2026 falls on 2027-01-30 → in the 2027 calendar, not 2026.
- **Computational:** yes → `rules/event-placement.json`
- **Applies to:** Backend
- **Source:** backend SRS FR-001 (E11)

**BR-CAL-03 — Fixed solar date that does not exist**
- **Rule:** A fixed solar date occurs on that month and day each year; in a year where that day does
  not exist (29 February in a common year) the event has no occurrence that year. No day is moved.
- **Examples:** 30/4 in 2026 → 2026-04-30; 29/2 in 2026 → none; 29/2 in 2028 → 2028-02-29.
- **Computational:** yes → `rules/event-placement.json`
- **Applies to:** Backend
- **Source:** backend SRS FR-002

**BR-CAL-04 — Fixed lunar date, leap months and day 30**
- **Rule:** A fixed lunar date occurs in the regular month of that number, never in its leap month,
  unless the event is defined as a leap-month event — then it occurs only in years that have that
  leap month. A lunar day 30 does not exist in a 29-day month: the event has no occurrence that
  year. Events that must fall on the last day of a month whatever its length (Giao thừa, vía Địa
  Tạng 30/7, vía Dược Sư 30/9) use the "last day of the month" rule instead, which falls on day 30
  in a full month and day 29 in a short one.
- **Examples:** Phật đản lunar 15/4 in 2026 → 2026-05-31; Vu Lan lunar 15/7 in 2025 (a year with a
  leap 6th month) → 2025-09-06 only; a leap-6th-month event in 2025 → 2025-08-08, in 2026 → none;
  last day of month 7 in 2025 → lunar 30/7 = 2025-09-21, in 2026 → lunar 29/7 = 2026-09-10;
  Giao thừa in 2026 → lunar 29/12/2025 = 2026-02-16.
- **Computational:** yes → `rules/event-placement.json`
- **Applies to:** Backend
- **Source:** backend SRS FR-003, FR-015; backend changelog v2.1 (vía Địa Tạng / Dược Sư); Android
  `home_assumptions.md` #11 and iOS Festivals README agree that festivals do not repeat in a leap month

**BR-CAL-05 — Rằm and Mùng Một every lunar month, leap months included**
- **Rule:** Mùng Một (lunar day 1) and Rằm (lunar day 15) are events of the catalogue that occur in
  every lunar month, leap months included, labelled "(nhuận)" in a leap month. They come from the
  same catalogue as every other event; no platform adds them on its own. On a day that also has a
  named festival (e.g. Tết on Mùng Một tháng Giêng), both occurrences are shown, ordered by
  BR-CAL-12.
- **Examples:** Rằm in 2025 → 12 occurrences, including 2025-08-08 (lunar 15/6 leap); Mùng Một in
  2026 → 12 occurrences, the first on 2026-01-19 (lunar 1/12/2025).
- **Computational:** yes → `rules/event-placement.json`
- **Applies to:** Backend, Website, iOS, Android (clients must stop computing them locally)
- **Source:** backend SRS FR-004; decision 2026-10-15 (Q5: the server is the only source)

**BR-CAL-06 — Solar terms**
- **Rule:** A solar-term event falls on the solar date, in Vietnam time (UTC+7), on which the sun
  reaches that term's ecliptic longitude (a multiple of 15°).
- **Examples:** Thanh minh (15°) in 2026 → 2026-04-05; Đông chí (270°) in 2026 → 2026-12-22.
- **Computational:** yes → `rules/event-placement.json`
- **Applies to:** Backend
- **Source:** backend SRS FR-005

**BR-CAL-07 — Vietnamese lunar calendar**
- **Rule:** Lunar dates follow the Vietnamese lunar calendar computed at UTC+7, for years 1900–2100.
  This is the same rule as BR-MEM-10; the two features must never disagree on a lunar date.
- **Examples:** 2026-04-30 → lunar 14/3/2026; 2028-02-29 → lunar 5/2/2028.
- **Computational:** yes → `rules/event-placement.json`
- **Applies to:** Backend
- **Source:** backend SRS NFR-002; `docs/product/memorial/product.md` BR-MEM-10

**BR-CAL-08 — Both dates on every occurrence**
- **Rule:** Every occurrence is shown with its solar date and its lunar date (day, month, and
  "(nhuận)" for a leap month), whichever calendar the event is defined in.
- **Examples:** 30/4 → also shows lunar 14/3; Vu Lan → also shows 2025-09-06.
- **Computational:** no (the dates themselves are covered by BR-CAL-01…07)
- **Applies to:** Backend, Website, iOS, Android
- **Source:** backend SRS FR-006

**BR-CAL-09 — Groups are data**
- **Rule:** Every event belongs to one or more groups. The groups today are Phật giáo (Buddhist),
  Việt Nam (Vietnamese holidays) and Dân gian (folk); the content editor can add more without a
  release, and every platform labels and filters by whatever groups the catalogue has, in the
  catalogue's group order.
- **Examples:** Phật đản → Phật giáo; 30/4 → Việt Nam; Tết Nguyên Đán → Việt Nam + Dân gian.
- **Computational:** no
- **Applies to:** Backend, Website, iOS, Android
- **Source:** backend SRS FR-007; decision 2026-10-15 (Q3)

**BR-CAL-10 — Major events (lễ lớn) are data**
- **Rule:** Each event carries a yes/no "major event" mark set by the content editor. Major events
  are highlighted wherever they are listed, are the only ones under the "Lễ lớn" filter, and come
  first on the home block (BR-CAL-15). The detail view labels the event with its group names.
- **Examples:** Phật đản → major; Rằm tháng Hai → not major; Tết Dương lịch → not major.
- **Proposed initial set** (from the prototype and the Android catalogue):
  > ⚠️ Not confirmed: the initial major events are Rằm tháng Giêng (15/1), Phật nhập Niết bàn (15/2),
  > vía Quan Âm đản sanh (19/2), Phật đản (15/4), vía Quan Âm thành đạo (19/6), Vu Lan (15/7), vía
  > Quan Âm xuất gia (19/9) and Phật thành đạo (8/12) — add or remove any? — blocks the seed data
  > only; all platforms read the mark from the catalogue.
- **Computational:** no
- **Applies to:** Backend, Website, iOS, Android
- **Source:** decision 2026-10-15 (Q4, option 1); prototype `festVals`; Android `MAJOR_KEYS`

**BR-CAL-11 — Public holiday on its own day only**
- **Rule:** An event marked as a public holiday is a day off on its own occurrence only. InnerLamp
  never shows substitute days off (nghỉ bù).
- **Examples:** 30/4 falling on a Saturday → only 30/4 is shown as a holiday, not Monday 2/5.
- **Computational:** no
- **Applies to:** Backend, Website, iOS, Android
- **Source:** decision 2026-10-11 (backend SRS FR-008)

**BR-CAL-12 — Order of occurrences**
- **Rule:** Occurrences are ordered by solar date, then by the event's display order (set by the
  content editor), then by the event's stable identifier, so every platform shows the same order.
- **Examples:** 2026-02-17 holds Tết Nguyên Đán (display order 1) and Mùng Một (display order 40)
  → Tết first.
- **Computational:** yes → `rules/ordering.json`
- **Applies to:** Backend, Website, iOS, Android
- **Source:** backend SRS FR-014

**BR-CAL-13 — Countdown**
- **Rule:** The countdown of an occurrence is its solar date minus today's date on the device, in
  whole days: 0 → "Hôm nay" (today), 1 → "Ngày mai" (tomorrow), n > 1 → "Còn n ngày" (n days left),
  negative → "Đã qua" (passed).
- **Examples:** today 2026-10-15, occurrence 2026-10-15 → Hôm nay; 2026-10-16 → Ngày mai;
  2026-10-25 → Còn 10 ngày; 2026-10-10 → Đã qua.
- **Computational:** yes → `rules/countdown.json`
- **Applies to:** Website, iOS, Android
- **Source:** iOS FestivalDetail spec v1.0; prototype `festDetail`

**BR-CAL-14 — Content format and empty content**
- **Rule:** In an event's long description each line is one paragraph; in its guidance each line is
  one practice, shown as a numbered list (a leading "-", "•" or "*" is dropped). A section whose
  content is empty is hidden. An event without an image shows the shared default image.
- **Examples:** guidance "Tụng kinh\nPhóng sanh\nĂn chay" → three numbered rows; empty guidance →
  no "Phật tử thường làm gì" section.
- **Computational:** no
- **Applies to:** Backend (content rule for the editor), Website, iOS, Android
- **Source:** iOS AS-FestivalDetail-001; decision 2026-10-15 (Q7, Q11, Q13)

**BR-CAL-15 — What the home block shows**
- **Rule:** The home block shows the months from the current month for six months. Per month it
  shows Buddhist-group occurrences only, at most three: major events first, then other named events,
  then Rằm and Mùng Một; within each tier in BR-CAL-12 order. The rest is counted as
  "+N sự kiện khác". A month with no Buddhist occurrence still shows its card with an empty message.
- **Examples:** a month with Phật đản (major), vía Văn Thù (named), Rằm and Mùng Một → shows Phật
  đản, vía Văn Thù, Mùng Một, "+1"; a month with only Rằm and Mùng Một → both, "+0" not shown.
- **Computational:** yes → `rules/home-block.json`
- **Applies to:** Website, iOS, Android
- **Source:** prototype home block; decision 2026-10-15 (Q3, Q5); replaces iOS AS-Home-003

**BR-CAL-16 — Turned-off events are never shown**
- **Rule:** The content editor can turn an event off instead of deleting it; a turned-off event has
  no occurrence anywhere.
- **Examples:** Noel turned off → it disappears from every year on every platform.
- **Computational:** no
- **Applies to:** Backend
- **Source:** backend SRS FR-013

### 3.3 Functional requirements

#### Access

**FR-CAL-001 — Anyone can read the calendar**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a guest or a signed-in user opens any view of this feature THE SYSTEM SHALL
  show the same events, without asking them to sign in.
- **Business rules:** —
- **Verification method:** automated test on each platform; manual QA as guest
- **Source:** decision 2026-10-15 (Q2: the calendar becomes public)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Guest opens the festival calendar
    Given the user is not signed in
    When they open the festival calendar
    Then they see the occurrences of the current year
    And no sign-in prompt is shown

  Scenario: Signed-in user sees the same calendar
    Given one user is signed in and another is a guest
    When both open the festival calendar for the same year
    Then both see the same occurrences in the same order
  ```

#### Festival calendar

**FR-CAL-002 — Festival calendar of a year**
- **Priority:** Must
- **Platforms:** Website, iOS, Android (Backend provides the year)
- **Description:** WHEN the user opens the festival calendar THE SYSTEM SHALL load the current year
  (in Vietnam time), list its occurrences grouped by solar month from January to December, and bring
  the current month into view; each row shows the solar day and weekday, the event name, the lunar
  date, the countdown, the major mark and the group labels.
- **Business rules:** BR-CAL-02, BR-CAL-08, BR-CAL-09, BR-CAL-10, BR-CAL-12, BR-CAL-13
- **Verification method:** automated test on each platform; shared vectors for order and countdown
- **Source:** prototype `festVals`; iOS AS-Festivals-001; website `events/`
- **Acceptance criteria:**
  ```gherkin
  Scenario: Calendar opens on the current month
    Given today is 2026-10-15
    When the user opens the festival calendar
    Then the months January to December 2026 are listed
    And October 2026 is in view

  Scenario: A lunar festival shows both dates
    Given the calendar of 2026 is shown
    When the user looks at May
    Then Phật đản is listed on Sunday 31 May with lunar date 15/4

  Scenario: Past occurrences of the current year stay listed
    Given today is 2026-10-15
    When the user scrolls to February
    Then Tết Nguyên Đán is listed with the countdown "Đã qua"
  ```

**FR-CAL-003 — Filters**
- **Priority:** Must
- **Platforms:** Website, iOS, Android
- **Description:** WHEN the user chooses a filter THE SYSTEM SHALL show only the matching occurrences
  of the loaded years: "Tất cả" (all), "Lễ lớn" (major events only), or one group; each filter shows
  how many occurrences it holds.
- **Business rules:** BR-CAL-09, BR-CAL-10
- **Verification method:** automated test on each platform
- **Source:** decision 2026-10-15 (Q3 option 1, Q4 option 1); website group tabs; iOS tabs
- **Acceptance criteria:**
  ```gherkin
  Scenario: Filter by group
    Given the calendar of 2026 is shown
    When the user chooses the "Việt Nam" filter
    Then 30/4 is listed
    And Phật đản is not listed

  Scenario: Major events only
    Given the calendar of 2026 is shown
    When the user chooses the "Lễ lớn" filter
    Then Phật đản and Vu Lan are listed
    And no Rằm or Mùng Một that is not a major event is listed

  Scenario: Filter keeps working after loading another year
    Given the "Phật giáo" filter is chosen
    When the next year is loaded
    Then the new year's months show only Buddhist-group occurrences
  ```

**FR-CAL-004 — Load the following years**
- **Priority:** Must
- **Platforms:** Website, iOS, Android
- **Description:** WHEN the user reaches the end of the loaded years THE SYSTEM SHALL load the next
  year and append it, up to 2100; earlier years than the current one are not offered. IF loading the
  next year fails THEN THE SYSTEM SHALL keep what is shown and offer a retry at the end of the list.
- **Business rules:** BR-CAL-02
- **Verification method:** automated test on each platform
- **Source:** backend SRS FR-010 (one year per load); decision 2026-10-15 (Q9)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Next year is appended
    Given the calendar shows 2026
    When the user reaches the end of December 2026
    Then January to December 2027 are appended below

  Scenario: Last year
    Given the calendar shows up to 2100
    When the user reaches the end of December 2100
    Then no further year is loaded

  Scenario: Next year fails to load
    Given the calendar shows 2026 and the backend cannot be reached
    When the user reaches the end of December 2026
    Then 2026 stays on screen
    And a retry option is shown at the end of the list
  ```

#### Event detail

**FR-CAL-005 — Event detail**
- **Priority:** Must
- **Platforms:** Website, iOS, Android
- **Description:** WHEN the user chooses an occurrence THE SYSTEM SHALL show that event for that
  date: image, group labels and major mark, name, lunar date, solar date with weekday, countdown,
  description and guidance, following the content rules; sections without content are hidden, and
  no "recommended sutras" or "temples holding this event" section is shown in v1.0.
- **Business rules:** BR-CAL-08, BR-CAL-10, BR-CAL-13, BR-CAL-14
- **Verification method:** automated test on each platform; manual QA against the prototype
- **Source:** prototype `festDetail`; iOS FestivalDetail spec v1.0; decision 2026-10-15 (Q6, Q7)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Detail of a festival
    Given today is 2026-10-15
    When the user opens Phật đản of 2026-05-31
    Then the view shows "Phật đản", lunar date 15/4, Sunday 31 May 2026 and "Đã qua"
    And the guidance is shown as a numbered list

  Scenario: Event without image or guidance
    Given an event has no image and an empty guidance
    When the user opens it
    Then the default image is shown
    And the "Phật tử thường làm gì" section is not shown

  Scenario: Event no longer in the catalogue
    Given the user opens an occurrence that the catalogue no longer has
    Then the view says the event was not found and offers to go back
  ```

**FR-CAL-009 — Previous and next event from the detail**
- **Priority:** Could
- **Platforms:** Website (see §3.6)
- **Description:** WHEN the event detail is shown THE SYSTEM SHALL offer the previous and the next
  occurrence in BR-CAL-12 order, across the year boundary when the adjacent year can be loaded.
- **Business rules:** BR-CAL-12
- **Verification method:** automated test on the website
- **Source:** website `events/[code]`
- **Acceptance criteria:**
  ```gherkin
  Scenario: Next event across the new year
    Given the user is on the last occurrence of 2026
    When they choose "next"
    Then the first occurrence of 2027 opens
  ```

#### Home

**FR-CAL-006 — Buddhist events on the home screen**
- **Priority:** Must
- **Platforms:** Website, iOS, Android
- **Description:** WHEN the home screen is shown THE SYSTEM SHALL show the Buddhist events block per
  BR-CAL-15; choosing an event opens its detail, and choosing "Tất cả" or "+N sự kiện khác" opens the
  festival calendar.
- **Business rules:** BR-CAL-10, BR-CAL-12, BR-CAL-13, BR-CAL-15
- **Verification method:** shared vectors `rules/home-block.json`; automated test on each platform
- **Source:** prototype home item 4; decision 2026-10-15 (Q3, Q5)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Block spans six months
    Given today is 2026-10-15
    When the home screen is shown
    Then the block has cards for October 2026 to March 2027

  Scenario: Only Buddhist events
    Given November 2026 has Buddhist events and 20/11 (Việt Nam group)
    When the home screen is shown
    Then the November card does not list 20/11

  Scenario: Open the calendar from the block
    When the user chooses "Tất cả"
    Then the festival calendar opens on the current month
  ```

#### Availability

**FR-CAL-007 — Offline reading**
- **Priority:** Should
- **Platforms:** Website, iOS, Android
- **Description:** WHEN a year cannot be loaded and that year was loaded before on this device or
  browser session THE SYSTEM SHALL show the last loaded copy with a note that it may be out of date.
- **Business rules:** —
- **Verification method:** manual QA in airplane mode; automated test where the platform allows
- **Source:** decision 2026-10-15 (Q10); iOS TD-Home-003
- **Acceptance criteria:**
  ```gherkin
  Scenario: Offline after a previous visit
    Given the user loaded the 2026 calendar yesterday
    And the device is offline today
    When they open the festival calendar
    Then the 2026 occurrences are shown
    And a note says the data may be out of date

  Scenario: Home block offline
    Given the 2026 calendar was loaded before
    And the device is offline
    When the home screen is shown
    Then the Buddhist events block is filled from that copy, not empty
  ```

**FR-CAL-008 — Load failure**
- **Priority:** Must
- **Platforms:** Website, iOS, Android
- **Description:** IF a year cannot be loaded and no earlier copy exists THEN THE SYSTEM SHALL say so
  (with a distinct message when the device is offline) and offer a retry; the home block shows the
  same message in place of its cards instead of disappearing.
- **Business rules:** —
- **Verification method:** automated test on each platform
- **Source:** iOS FestivalDetail spec v1.0; website "unavailable" state
- **Acceptance criteria:**
  ```gherkin
  Scenario: First visit while offline
    Given the calendar was never loaded on this device
    And the device is offline
    When the user opens the festival calendar
    Then an offline message and a retry option are shown

  Scenario: Retry succeeds
    Given the load failure message is shown
    When the connection is back and the user retries
    Then the calendar of the current year is shown
  ```

#### Content management

**FR-CAL-010 — Content changes without a release**
- **Priority:** Must
- **Platforms:** Backend
- **Description:** WHEN the content editor changes an event's text, image, groups, display order,
  major mark, public-holiday mark or turns it off THE SYSTEM SHALL show the change on every platform
  on the next load, without an app or website release.
- **Business rules:** BR-CAL-09, BR-CAL-10, BR-CAL-11, BR-CAL-16
- **Verification method:** automated backend test; manual QA
- **Source:** backend SRS FR-013
- **Acceptance criteria:**
  ```gherkin
  Scenario: Mark an event as major
    Given vía Văn Thù is not a major event
    When the content editor marks it as major
    Then after the next load it appears under the "Lễ lớn" filter on every platform

  Scenario: Turn an event off
    When the content editor turns Noel off
    Then Noel no longer appears in any year on any platform
  ```

### 3.4 Non-functional requirements

- **NFR-CAL-01 — Correct lunar dates:** for every day of 1900–2100 the lunar date and the solar-term
  dates equal the reference tables (Hồ Ngọc Đức plus a second source), with zero mismatches.
  Platforms: Backend.
- **NFR-CAL-02 — Stable answer:** the same year asked twice, with no catalogue change in between,
  gives identical occurrences in identical order, whoever asks and whenever. Platforms: Backend.
- **NFR-CAL-03 — Fast enough:** the festival calendar of the current year appears within 2 s on a 4G
  connection, and from the device's copy within 0.5 s when one exists. Platforms: Website, iOS,
  Android.

### 3.5 Data requirements

| Data item | Meaning | Owned by | Retention / deletion | Private or public |
|---|---|---|---|---|
| Event | Name, short and long description, guidance, image, groups, display order, major mark, public-holiday mark, on/off | The InnerLamp catalogue on the server | Turned off, never deleted (BR-CAL-16) | Public |
| Repeat rule | How the event's date is found each year (BR-CAL-01) | Same | Same as the event | Public |
| Group | Name and order of a label | Same | Kept while an event uses it | Public |
| Occurrence | An event on one solar day with both dates | Computed on request, never stored | — | Public |
| Device copy | The last loaded year(s) for offline reading | The device or browser | Replaced on the next successful load | Public content, no personal data |

### 3.6 Platform variance

Intentional differences between platforms. Anything not listed here must behave identically.

| ID | Platforms | Difference | Reason |
|---|---|---|---|
| FR-CAL-009 | Website only | Previous / next event links on the detail | Already built on the website; the apps navigate back to the list. Other platforms may add it later without a product change |
| FR-CAL-007 | Website | "Device copy" means the copy kept by the website server or the browser for the session | The website has no persistent offline store |

### 3.7 Software system attributes

- **Reliability:** the same lunar rules as the memorial feature (BR-CAL-07 ≡ BR-MEM-10).
- **Availability:** when the backend is unreachable, the last loaded copy is shown (FR-CAL-007).
- **Security & privacy:** no personal data; the calendar is public content.
- **Maintainability / portability:** content and flags change as data (FR-CAL-010).

---

## 4. Verification & traceability

### 4.1 Verification approach

The backend verifies BR-CAL-01…07 by running `rules/event-placement.json` and NFR-CAL-01 against
reference tables. Every client runs `rules/countdown.json`, `rules/ordering.json` (where it orders
locally) and `rules/home-block.json`. FRs are verified by each platform's automated tests from the
Gherkin scenarios, plus manual QA against the prototype.

### 4.2 Traceability matrix

Status per platform: `planned` · `in-progress` · `done` · `drift` (implemented differently — see
note) · `n/a` · `later`.

| ID | Backend | Website | iOS | Android | Note |
|---|---|---|---|---|---|
| BR-CAL-01 | done | n/a | n/a | drift | Android computes from a local catalogue |
| BR-CAL-02 | done | n/a | n/a | drift | |
| BR-CAL-03 | done | n/a | n/a | drift | |
| BR-CAL-04 | done | n/a | n/a | drift | |
| BR-CAL-05 | done | done | drift | drift | iOS adds Rằm/Mùng Một locally when missing (AS-Festivals-002) |
| BR-CAL-06 | done | n/a | n/a | drift | |
| BR-CAL-07 | done | n/a | n/a | drift | |
| BR-CAL-08 | done | done | done | done | |
| BR-CAL-09 | done | done | done | drift | Android shows Buddhist events only |
| BR-CAL-10 | planned | planned | drift | drift | iOS "Lễ lớn" = every named event; Android fixed list |
| BR-CAL-11 | done | done | done | n/a | |
| BR-CAL-12 | done | done | done | drift | |
| BR-CAL-13 | n/a | done | done | done | |
| BR-CAL-14 | done | done | drift | n/a | iOS shows "Chưa có…" instead of hiding; no Android detail |
| BR-CAL-15 | n/a | planned | drift | drift | iOS/Android show every group, no tiering |
| BR-CAL-16 | done | n/a | n/a | n/a | |
| FR-CAL-001 | drift | drift | drift | done | Backend requires sign-in today; Android is local |
| FR-CAL-002 | done | done | done | drift | iOS loads up to 2200; Android local |
| FR-CAL-003 | n/a | done | drift | drift | Website filters by group only; iOS tabs differ |
| FR-CAL-004 | done | done | drift | planned | iOS: silent retry, limit 2200 |
| FR-CAL-005 | n/a | done | drift | planned | iOS shows sutra/temple sections |
| FR-CAL-006 | n/a | planned | drift | drift | |
| FR-CAL-007 | n/a | done | planned | planned | iOS TD-Home-003 |
| FR-CAL-008 | done | done | done | planned | |
| FR-CAL-009 | n/a | done | n/a | n/a | §3.6 |
| FR-CAL-010 | done | n/a | n/a | n/a | Back-office screens later |
| NFR-CAL-01 | in-progress | n/a | n/a | n/a | Full reference table pending |
| NFR-CAL-02 | done | n/a | n/a | n/a | |
| NFR-CAL-03 | planned | planned | planned | planned | |

### 4.3 Test vectors

| File | Rule | Cases |
|---|---|---|
| `rules/event-placement.json` | BR-CAL-01…07 | 19 |
| `rules/ordering.json` | BR-CAL-12 | 3 |
| `rules/countdown.json` | BR-CAL-13 | 5 |
| `rules/home-block.json` | BR-CAL-15 | 4 |

---

## 5. Appendices

### Appendix A — Open questions

#### Open (unconfirmed)

> ⚠️ Not confirmed: the initial set of major events (BR-CAL-10) — blocks the seed data only; all platforms.

> ⚠️ Not confirmed: who checks the seed content against the GHPGVN calendar, and by when (§2.7) — blocks publishing the content; all platforms.

#### Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| Is an ADR needed? | No: the repeat-rule data model was decided 2026-10-11; making the calendar public is a read route change; reminders later reuse ADR-0001 | 2026-10-15 |
| Slug and scope | `calendar`; v1.0 covers events only; lunar calendar, day detail and daily picture come as v1.x | 2026-10-15 |
| Can guests read the calendar? | Yes, everyone (FR-CAL-001) | 2026-10-15 |
| Which groups are shown? | Festival calendar: all groups with filters; home block: Buddhist group only | 2026-10-15 |
| How are major events known? | A yes/no mark on each event, set by the content editor (BR-CAL-10) | 2026-10-15 |
| Who produces Rằm and Mùng Một? | The catalogue only; clients stop computing them; they rank last on the home block | 2026-10-15 |
| Separate detail capability? | No: the detail uses the year already loaded | 2026-10-15 |
| Recommended sutras / hosting temples on the detail? | Out of scope for v1.0; hidden | 2026-10-15 |
| Reminders ("Nhắc tôi ngày này")? | Out of scope for v1.0; will use ADR-0001 | 2026-10-15 |
| Which years can be browsed? | Current year onward, one year at a time, up to 2100 | 2026-10-15 |
| Offline? | Show the last loaded copy with an "may be out of date" note | 2026-10-15 |
| Content format | One paragraph / practice per line | 2026-10-15 |
| Event images | Default image until images are supplied | 2026-10-15 |
| Is the iOS "Events" screen (courses, registration) part of this feature? | No: temple events of the user's sangha | 2026-10-15 |
| Substitute days off | Out of scope | 2026-10-11 |
| Who converts lunar ↔ solar? | The backend | 2026-10-11 |

### Appendix B — Contract handoff (backend)

What the backend contract must change for this spec (for `spec-writer`):

1. The year's events must be readable without signing in (FR-CAL-001); the website's API key rule
   for public routes applies as for other catalogue reads.
2. Each event carries the major mark (BR-CAL-10), settable as data and loaded by the seed.
3. The seed file must be kept in version control (today it is excluded) so a fresh setup has events.
4. Caching headers must suit a public, shared response.
