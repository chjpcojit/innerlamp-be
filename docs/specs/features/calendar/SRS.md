# Software Requirements Specification — Calendar Events (Buddhist, Vietnamese and Folk)

> Superseded by [docs/product/calendar/product.md](../../../product/calendar/product.md) (v1.0) — kept for history; do not update.

| Field | Value |
|---|---|
| Document type | Software Requirements Specification (SRS) — IEEE 29148-2018 |
| Feature | Calendar Events (sự kiện lịch: Phật giáo, Việt Nam, dân gian) — extends the existing `calendar` module |
| Product | InnerLamp |
| Version | 0.3 |
| Last updated | 2026-10-11 |
| Status | confirmed 2026-10-11 — all questions answered; ready for `il-spec-writer` |
| Owner / author | duc@wee.vn (BA assist: Claude) |
| Downstream handoff | `il-spec-writer` → `requirements.md` / `design.md` / `flow.md` / `tasks.md` under `docs/specs/features/calendar/` (existing files describe only the daily image, US-CAL-001) |

---

## 1. Introduction

### 1.1 Purpose

This document specifies, in business terms, a calendar of events that the server provides to signed-in users: Buddhist observances, Vietnamese national days, and Vietnamese folk festivals. It is written for product and engineering and drives the engineering spec that `il-spec-writer` produces next.

### 1.2 Scope

**In scope:**
- A list of events for one calendar year, each with its solar (dương lịch) and lunar (âm lịch) date.
- Five kinds of recurrence: fixed solar date, fixed lunar date, every lunar month, last day of a lunar month, and solar term (tiết khí).
- Event content: name, short description, long description, image, and how-to-observe guidance (lễ nghi).
- One or more groups per event (Buddhist / Vietnam / folk).
- A flag saying whether the event is an official public holiday.
- Year-by-year "load more": the client asks for the current year, then the following years.
- A seed data file prepared by the team, loaded into the database.

**Out of scope:**
- Substitute days off (nghỉ bù) and government-adjusted working days. Only the day of the event itself is a holiday. *(Decision 2026-10-11.)*
- Back-office screens and APIs to create, edit or toggle events (a separate SRS later). This SRS only requires that the data model allows it, see FR-013.
- User-created personal events, reminders and push notifications.
- The daily Buddha image (US-CAL-001, already implemented, unchanged).
- Other countries' calendars and non-Vietnamese lunar conventions.

### 1.3 Product overview

InnerLamp is a Buddhist-practice companion app (memorials, sutras, temples, genealogy, lamps). The calendar screen already shows a lunar calendar and a daily image. This feature adds the events that fall on each day, so users know which Buddhist observances, national days and folk festivals are coming.

### 1.4 Definitions, acronyms, abbreviations

| Term | Definition |
|---|---|
| Event (sự kiện) | A named observance, for example "Phật đản" (Buddha's Birthday) or "Quốc khánh" (National Day). |
| Occurrence | An event on one concrete date in one concrete year. |
| Recurrence rule | The stored description of when an event repeats every year. One per event. |
| Solar date (dương lịch) | Gregorian date. |
| Lunar date (âm lịch) | Vietnamese lunar date: day, month, lunar year, and whether the month is a leap month. |
| Leap month (tháng nhuận) | An extra lunar month inserted in some lunar years, sharing its number with the previous month. |
| Solar term (tiết khí) | One of 24 points of the sun's yearly path, for example Thanh minh (Clear and Bright) or Đông chí (Winter Solstice). Falls on a solar date that shifts slightly by year. |
| Group (nhóm) | A classification of an event: `buddhist`, `vietnam`, `folk`. |
| Public holiday | An official day off in Vietnam. |
| BO | Back-office. |

### 1.5 References

- `docs/specs/features/calendar/requirements.md` and `api-contract.md` (daily image, v1.1).
- `docs/specs/features/memorial/requirements.md` (states that the server did not know the lunar calendar until now).
- Existing year validation: 1900..2100 (daily image endpoint).

### 1.6 Document organization

Section 2 describes the context, section 3 lists the requirements (`FR-*`, `NFR-*`) with Gherkin criteria, section 4 maps requirements to verification, Appendix A lists open and resolved questions, and Appendix B gives an indicative event list.

---

## 2. Overall description

### 2.1 Product perspective

An enhancement of the existing `calendar` module. The server now owns a solar ↔ lunar conversion and a solar-term calculation, which the earlier spec left to the client. The client no longer needs its own lunar logic to display events, but may keep it for the day grid.

### 2.2 Product functions

- List all events of a year with both calendars (`FR-001`, `FR-006`, `FR-014`).
- Resolve each event's date by its recurrence rule (`FR-002`–`FR-005`, `FR-015`).
- Classify events and flag public holidays (`FR-007`, `FR-008`).
- Provide event content (`FR-009`).
- Validate and default the requested year (`FR-010`); require sign-in (`FR-011`); allow caching (`FR-012`).
- Load the seed data and keep it editable later by BO (`FR-013`).

### 2.3 User classes and characteristics

| Actor | Description | Characteristics / permissions |
|---|---|---|
| Signed-in user | Uses the iOS app. | Reads events. Needs a valid access token. |
| Anonymous visitor | No token. | Cannot read events. *(Decision 2026-10-11: auth required.)* |
| Back-office admin | Staff. | Will manage events and the holiday flag in a later SRS. |

### 2.4 Operating environment

- Platform: iOS client, Go backend (`services/appService`).
- Network: online required to fetch; the client may cache a fetched year for the session.
- Time zone: Vietnam (UTC+7), taken from `calendar.timezone`.

### 2.5 Design and implementation constraints

- Clean Architecture layout and per-package conventions of this repo (`AGENTS.md`).
- Lunar-calendar computation is done in-house without a new dependency (Principle: minimal dependencies). Vietnam's lunar calendar is computed at UTC+7, which differs from the Chinese calendar in some years.
- Valid years are 1900..2100, the range in which the conversion is verified.
- Non-sensitive, environment-dependent values live in `services/appService/config/`, not as constants in domain or usecase.

### 2.6 User documentation

**N/A** — the client app explains events on screen; no separate manual.

### 2.7 Assumptions and dependencies

- **Assumptions:**
  - A1: Every event's date can be derived from exactly one recurrence rule; no per-year exceptions exist (see Out of scope).
  - A2: The event list is small (tens to low hundreds of events), so one year is returned in a single response without paging inside the year.
  - A3: The client filters the returned year to the "current month" itself.
  - A4: The seed file is for first-time setup. In development it may be re-run and then overwrites existing events (keyed by stable code). Once BO exists, BO updates the data and the seed is no longer run. *(Decision 2026-10-11; no production guard is added while the project is in development.)*
- **Dependencies:**
  - The seed JSON file, collected and verified by the team before loading (task in `tasks.md`).
  - Image assets for events, if images are used (hosting and licence — see Appendix A).
  - A later BO SRS for editing events and toggling the holiday flag.

---

## 3. Specific requirements

### 3.1 External interface requirements

#### 3.1.1 User interfaces

The client shows the events of the current month on the calendar screen and fetches the next year when the user scrolls or navigates beyond the loaded year. Group and holiday information drive labels and colour. Screens are defined in the Design System, not here.

#### 3.1.2 Hardware interfaces

**N/A** — no custom hardware interface.

#### 3.1.3 Software interfaces

One new read endpoint under the `calendar` module returns the events of one year. Contract-level summary (the exact path and field names are fixed in `api-contract.md` by `il-spec-writer`):

- Input: a year (optional).
- Output: the year, and a list of occurrences with event content, groups, public-holiday flag, solar date and lunar date.
- Errors: invalid year; not signed in.

#### 3.1.4 Communication interfaces

HTTPS, JSON, the same response envelope as other routes, bearer access token.

### 3.2 Functional requirements

#### Listing

**FR-001 — List events of a year**
- **Priority:** Must
- **Description:** WHEN a signed-in user requests events for a valid year THE SYSTEM SHALL return every occurrence that falls in that solar year, each with its solar and lunar date.
- **Verification method:** unit test + black-box HTTP test
- **Source:** user request 2026-10-11 ("API lấy từ tháng 1 của năm hiện tại và load more các năm sau")
- **Acceptance criteria:**
  ```gherkin
  Scenario: Load the current year
    Given the user is signed in
    When the user asks for the events of the current year
    Then the user receives the occurrences from 1 January to 31 December of that year

  Scenario: Load more years
    Given the user has loaded the events of 2026
    When the user asks for the events of 2027
    Then the user receives the occurrences of 2027 only

  Scenario: Occurrence near year boundary
    Given an event whose lunar date falls in the next solar year
    When the user asks for the events of the earlier year
    Then that occurrence is not included in the earlier year
  ```

**FR-014 — Order of results**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL return occurrences sorted by solar date ascending, and by a stable event order when several events share a date.
- **Verification method:** unit test
- **Source:** inferred from client month display
- **Acceptance criteria:**
  ```gherkin
  Scenario: Sorted by date
    Given a year with events in different months
    When the user asks for that year
    Then the occurrences are in solar-date order

  Scenario: Two events on the same day
    Given two events fall on the same solar date
    When the user asks for that year twice
    Then both occurrences appear, in the same order both times
  ```

#### Recurrence rules

**FR-002 — Fixed solar date**
- **Priority:** Must
- **Description:** WHEN an event is defined by a solar month and day THE SYSTEM SHALL place it on that solar date in every requested year.
- **Verification method:** unit test
- **Source:** user decision 2026-10-11 (30/4, Noel, Tết Dương lịch are stored as solar)
- **Acceptance criteria:**
  ```gherkin
  Scenario Outline: Fixed solar events
    Given an event defined as solar <day>/<month>
    When the user asks for the year <year>
    Then the event appears on <year>-<month>-<day>
    And its lunar date is shown

    Examples:
      | day | month | year |
      | 30  | 4     | 2026 |
      | 25  | 12    | 2027 |
      | 2   | 9     | 2100 |

  Scenario: 29 February event in a common year
    Given an event defined as solar 29/2
    When the user asks for a year that is not a leap year
    Then the event does not appear that year
  ```
  > Decision 2026-10-11: no 29/2 event is seeded; if one is added it simply does not occur in common years.

**FR-003 — Fixed lunar date, with leap-month marker**
- **Priority:** Must
- **Description:** WHEN an event is defined by a lunar month and day THE SYSTEM SHALL compute its solar date in the requested year, and SHALL occur in the regular month by default; only an event marked for the leap month occurs in a leap month, and only in years that have that leap month.
- **Verification method:** unit test against a reference table
- **Source:** user decision 2026-10-11 (store lunar for lunar events; mark the leap month to compute)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Lunar event in a normal year
    Given an event defined as lunar 15/4
    When the user asks for the year 2026
    Then the occurrence's solar date is the date of lunar 15/4 in 2026
    And the lunar date shown is 15/4, not a leap month

  Scenario: Year with a leap month of the same number
    Given an event defined as lunar 15/4 not marked for leap month
    And the requested year has a leap 4th month
    When the user asks for that year
    Then the event appears once, in the regular 4th month

  Scenario: Event marked for the leap month
    Given an event defined as lunar 15/4 marked for the leap month
    When the user asks for a year that has no leap 4th month
    Then the event does not appear that year
  ```
  > Decision 2026-10-11: no seeded event is marked for the leap month; the mechanism exists for BO to use later.

**FR-004 — Every lunar month**
- **Priority:** Must
- **Description:** WHEN an event is defined as a lunar day repeating every month (mùng 1, Rằm) THE SYSTEM SHALL produce one occurrence per lunar month that starts in or reaches the requested year, including leap months.
- **Verification method:** unit test
- **Source:** user decision 2026-10-11 (include mùng 1 and Rằm by rule)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Rằm in a year without leap month
    Given the "Rằm" monthly event (lunar day 15)
    When the user asks for a year with 12 lunar months
    Then the user sees an occurrence for every lunar month's 15th that falls in that solar year

  Scenario: Leap month adds one occurrence
    Given the "Rằm" monthly event
    When the user asks for a year with a leap lunar month
    Then there is one extra occurrence, with the leap-month marker set
  ```

**FR-015 — Last day of a lunar month**
- **Priority:** Must
- **Description:** WHEN an event is defined as the last day of a lunar month (for example Giao thừa, the last day of month 12) THE SYSTEM SHALL place it on day 30 if that month has 30 days, otherwise on day 29.
- **Verification method:** unit test against a reference table
- **Source:** user decision 2026-10-11 (lunar day 30 does not exist in 29-day months)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Month 12 has 30 days
    Given the "Giao thừa" event defined as the last day of lunar month 12
    When the user asks for a year in which lunar month 12 has 30 days
    Then it appears on lunar 30/12

  Scenario: Month 12 has 29 days
    Given the "Giao thừa" event
    When the user asks for a year in which lunar month 12 has 29 days
    Then it appears on lunar 29/12

  Scenario: Day after Giao thừa
    Given the "Giao thừa" event
    When the user asks for a year
    Then the next day is lunar 1/1 of the following lunar year
  ```

**FR-005 — Solar term**
- **Priority:** Must
- **Description:** WHEN an event is defined by a solar term THE SYSTEM SHALL compute the solar date on which the sun reaches that term's longitude, at UTC+7, in the requested year.
- **Verification method:** unit test against a published table
- **Source:** user decision 2026-10-11 ("anh muốn có luôn mấy cái lễ đó")
- **Acceptance criteria:**
  ```gherkin
  Scenario: Thanh minh in a given year
    Given the "Thanh minh" event defined by its solar term
    When the user asks for the year 2026
    Then it appears on the date published for that term in 2026
    And its lunar date is shown

  Scenario: Date shifts between years
    Given the "Đông chí" event
    When the user asks for two consecutive years
    Then each year's date follows the published table, not a fixed day
  ```
  > Decision 2026-10-11: the assistant chooses the solar terms in the seed; missing ones are added later through BO.

#### Both calendars

**FR-006 — Always return both calendars**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL return, for every occurrence, both a solar date and a lunar date (day, month, lunar year, leap-month marker), regardless of which calendar the event is stored in.
- **Verification method:** unit test
- **Source:** user decision 2026-10-11 (store by type, compute the other on output)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Solar-stored event shows its lunar date
    Given an event stored as solar 30/4
    When the user asks for that year
    Then the occurrence shows the solar date and the matching lunar date

  Scenario: Lunar-stored event shows its solar date
    Given an event stored as lunar 15/7
    When the user asks for that year
    Then the occurrence shows the lunar date and the matching solar date
  ```

#### Content and classification

**FR-007 — Groups**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL return a list of one or more groups for each event, so that one event may belong to several groups, and SHALL treat groups as data that BO can extend.
- **Verification method:** unit test
- **Source:** user decision 2026-10-11 ("danh sách nhóm")
- **Acceptance criteria:**
  ```gherkin
  Scenario: Event in several groups
    Given Tết Nguyên Đán is in the groups "vietnam" and "folk"
    When the user asks for that year
    Then both groups are returned for that occurrence

  Scenario: Event with no group is not allowed
    Given an event definition with an empty group list
    When the seed data is loaded
    Then the load is rejected with a clear message
  ```
  > Decision 2026-10-11: BO may add groups. Groups are data, not a fixed list; the seed starts with `buddhist`, `vietnam`, `folk`.

**FR-008 — Public-holiday flag**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL return a public-holiday flag per event, which applies to the event's own day only and never to substitute days.
- **Verification method:** unit test
- **Source:** user decision 2026-10-11 (no substitute days)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Official holiday
    Given 30/4 is flagged as a public holiday
    When the user asks for that year
    Then the 30/4 occurrence is marked as a public holiday

  Scenario: Substitute day is not a holiday
    Given 30/4 and 1/5 are flagged and the days after them are not events
    When the user asks for that year
    Then no occurrence is created for the substitute days
  ```

**FR-009 — Event content**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL return, for each event, a name, a short description, a long description, an image (optional), and observance guidance (lễ nghi). Content is in Vietnamese for now; the data model SHALL allow adding other languages later without changing existing events.
- **Verification method:** unit test + manual review of seed data
- **Source:** user decision 2026-10-11 ("đầy đủ")
- **Acceptance criteria:**
  ```gherkin
  Scenario: Full content present
    Given an event with all content fields filled
    When the user asks for that year
    Then the occurrence carries name, short description, long description, image and guidance

  Scenario: Event without image
    Given an event has no image
    When the user asks for that year
    Then the occurrence is still returned and the image is shown as absent
  ```
  > Decision 2026-10-11: images are left empty in the seed; the owner will supply them later (source, licence and hosting to be settled then).

#### Request handling

**FR-010 — Year parameter**
- **Priority:** Must
- **Description:** WHEN the year is absent THE SYSTEM SHALL use the current year in the configured time zone; WHEN the year is not a whole number in 1900..2100 THE SYSTEM SHALL reject the request with a clear invalid-year error.
- **Verification method:** unit test + black-box HTTP test
- **Source:** user decision 2026-10-11
- **Acceptance criteria:**
  ```gherkin
  Scenario: Year omitted
    Given the user is signed in
    When the user asks for events without a year
    Then the user receives the current year's events

  Scenario Outline: Invalid year
    Given the user is signed in
    When the user asks for the year "<year>"
    Then the request is rejected as an invalid year
    And no events are returned

    Examples:
      | year   |
      | 1899   |
      | 2101   |
      | abc    |
      | 2026.5 |
  ```

**FR-011 — Authentication required**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL serve events only to requests carrying a valid access token.
- **Verification method:** black-box HTTP test
- **Source:** user decision 2026-10-11
- **Acceptance criteria:**
  ```gherkin
  Scenario: No token
    Given a request without sign-in
    When it asks for events
    Then it is rejected as unauthenticated

  Scenario: Wrong token type
    Given a request carrying a refresh token
    When it asks for events
    Then it is rejected as unauthenticated
  ```

**FR-012 — Caching**
- **Priority:** Should
- **Description:** WHEN the year is explicit THE SYSTEM SHALL allow the client to cache the response for one day, for that user only (private).
- **Verification method:** black-box HTTP test
- **Source:** user decision 2026-10-11
- **Acceptance criteria:**
  ```gherkin
  Scenario: Explicit year is cacheable by the user
    Given the user asks for the year 2027
    Then the response may be kept by the user's app for one day
    And it must not be stored by shared caches

  Scenario: Omitted year is not cached
    Given the user asks for events without a year
    Then the response must not be reused on the next day
  ```

#### Data

**FR-013 — Seed data and later BO editing**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL load events from a seed JSON file into the database keyed by a stable event code, so that re-running the seed creates no duplicates and overwrites an existing event with the file's content. The public-holiday flag, event content and groups SHALL be stored as database data, so that BO can change them later without a release.
- **Verification method:** integration test with an ephemeral database
- **Source:** user decision 2026-10-11 (JSON for first setup, re-run overwrites in dev, BO takes over later)
- **Acceptance criteria:**
  ```gherkin
  Scenario: First load
    Given an empty events table and a valid seed file
    When the seed is loaded
    Then every event in the file exists in the database

  Scenario: Re-running the seed in development
    Given the seed was loaded and an event's text was changed in the file
    When the seed is loaded again
    Then no duplicate events are created
    And that event shows the new text

  Scenario: Invalid seed file
    Given a seed file with a duplicate event code or a missing rule
    When the seed is loaded
    Then loading fails before any event is written
  ```

### 3.3 Performance requirements

- **NFR-001 — Response time:** one year of events is returned within 200 ms at the server (excluding network) on the reference dev environment.
- **NFR-002 — Conversion accuracy:** solar ↔ lunar conversion and solar-term dates match an authoritative reference table for every sample year in 1900..2100, with zero mismatches. The oracle is the Hồ Ngọc Đức lunar tables for lunar dates and a published Vietnam-time table for solar terms, cross-checked with a second source for a sample of years.
- **NFR-003 — Determinism:** the same year and the same data always return the same result, independent of the time of the request.

### 3.4 Logical database requirements

- Events persist in the server database: stable code, name, descriptions, image reference, observance guidance, groups (list), recurrence kind and its parameters, public-holiday flag, active flag, display order.
- No per-year rows are stored; occurrences are computed per request (FR-002–FR-005).
- No retention or deletion rules beyond BO deactivating an event.

### 3.5 Design constraints

- The seed file lives in the repository and is reviewed like code.
- Config values that change per environment (time zone, image base URL) go in `services/appService/config/`.

### 3.6 Software system attributes

- **Reliability:** an invalid seed or an invalid time-zone configuration fails at startup, not at request time (consistent with `calendar` v1.1).
- **Availability:** same as the rest of `appService`.
- **Security:** requires a valid access token; no personal data is stored or returned.
- **Maintainability:** adding an event of an existing kind needs data only, no code change.
- **Portability:** N/A.

---

## 4. Verification & traceability

### 4.1 Verification approach

Pure date logic (FR-002–FR-006, NFR-002) by unit tests against a reference table; HTTP behaviour (FR-001, FR-010–FR-012) by black-box tests against the real binary; seed (FR-013) by an integration test with an ephemeral database.

### 4.2 Traceability matrix

| Requirement ID | Source | Design | Test |
|---|---|---|---|
| FR-001 | user request 2026-10-11 | → il-spec-writer | 3 scenarios |
| FR-002 | decision 2026-10-11 | → il-spec-writer | 2 scenarios (1 outline) |
| FR-003 | decision 2026-10-11 | → il-spec-writer | 3 scenarios |
| FR-004 | decision 2026-10-11 | → il-spec-writer | 2 scenarios |
| FR-005 | decision 2026-10-11 | → il-spec-writer | 2 scenarios |
| FR-006 | decision 2026-10-11 | → il-spec-writer | 2 scenarios |
| FR-007 | decision 2026-10-11 | → il-spec-writer | 2 scenarios |
| FR-008 | decision 2026-10-11 | → il-spec-writer | 2 scenarios |
| FR-009 | decision 2026-10-11 | → il-spec-writer | 2 scenarios |
| FR-010 | decision 2026-10-11 | → il-spec-writer | 2 scenarios (1 outline) |
| FR-011 | decision 2026-10-11 | → il-spec-writer | 2 scenarios |
| FR-012 | decision 2026-10-11 | → il-spec-writer | 2 scenarios |
| FR-013 | decision 2026-10-11 | → il-spec-writer | 3 scenarios |
| FR-014 | inferred | → il-spec-writer | 2 scenarios |
| FR-015 | decision 2026-10-11 | → il-spec-writer | 3 scenarios |
| NFR-001 | decision 2026-10-11 | → il-spec-writer | measured in integration test |
| NFR-002 | decision 2026-10-11 | → il-spec-writer | via FR-003/004/005 tests |
| NFR-003 | inferred | → il-spec-writer | via FR-001 tests |

---

## 5. Appendices

### Appendix A — Open questions

#### Open (unconfirmed)

None.

#### Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| Who converts solar ↔ lunar? | Server. Store by the event's own calendar type, compute the other on output, return both. | 2026-10-11 |
| Leap month handling | Marked in the data; computed by the server (FR-003, FR-004). | 2026-10-11 |
| Avoid storing every year? | Yes, store recurrence rules only, occurrences are computed. | 2026-10-11 |
| Solar terms in scope? | Yes, compute them (FR-005). | 2026-10-11 |
| Substitute days off / per-year overrides | Out of scope: only the event's own day is a holiday. | 2026-10-11 |
| Data source | Team collects into a JSON seed file, loaded into the DB; BO SRS later. | 2026-10-11 |
| Holiday flag editable later | Yes, stored as data so BO can toggle it (FR-013). | 2026-10-11 |
| Pagination unit | One year per request; "load more" = next year; any year 1900–2100 allowed. | 2026-10-11 |
| Month filtering | Client filters; no month parameter. | 2026-10-11 |
| Out-of-range year | Invalid-year error. Omitted year = current year. | 2026-10-11 |
| Authentication | Required (access token). | 2026-10-11 |
| Cache | Private, one day, when the year is explicit. | 2026-10-11 |
| Where it lives | Extends the `calendar` module. | 2026-10-11 |
| Lunar day 30 in a 29-day month | New "last day of month" rule (FR-015). | 2026-10-11 |
| Seed re-run behaviour | Seed is for first setup; dev re-run overwrites; BO updates later and the seed is not run (FR-013). | 2026-10-11 |
| Cache when year is omitted | `no-cache`; explicit year gets private one-day cache (FR-012). | 2026-10-11 |
| Response-time target | 200 ms at the server (NFR-001). | 2026-10-11 |
| Reference table | Hồ Ngọc Đức tables plus a second source (NFR-002). | 2026-10-11 |
| Content language | Vietnamese only for now; other languages can be added later. | 2026-10-11 |
| Event images | Left empty in the seed; owner supplies them later. | 2026-10-11 |
| Which solar terms | Assistant chooses; BO adds missing ones later. | 2026-10-11 |
| Group set | Extensible by BO (groups are data). | 2026-10-11 |
| Groups | A list; an event may belong to several groups. | 2026-10-11 |
| Holiday flag | Yes. Buddhist events follow the Vietnam Buddhist Sangha's common calendar. | 2026-10-11 |
| mùng 1 / Rằm | Included, generated by the monthly rule. | 2026-10-11 |
| Content fields | Full: name, short and long description, image, observance guidance. | 2026-10-11 |

### Appendix B — Indicative event list (to be verified when the seed file is collected)

| Kind | Examples |
|---|---|
| Solar | Tết Dương lịch (1/1), 8/3, 30/4, 1/5, 1/6, 2/9, 20/10, 20/11, Noel (25/12) |
| Lunar | Tết Nguyên Đán (1/1), Rằm tháng Giêng (15/1), Hàn thực (3/3), Giỗ Tổ Hùng Vương (10/3), Phật đản (15/4), Đoan ngọ (5/5), Vu lan (15/7), Trung thu (15/8), Hạ nguyên (15/10), Táo quân (23/12) |
| Lunar monthly | Mùng 1, Rằm |
| Solar term | Thanh minh, Đông chí |

This list is a starting point for the data collection task, not a requirement.
