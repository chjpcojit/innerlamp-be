# Software Requirements Specification — Temple Events

| Field | Value |
|---|---|
| Document type | Software Requirements Specification (SRS) — IEEE 29148-2018 |
| Feature | Temple Events (sự kiện của chùa) |
| Product | InnerLamp |
| Version | 0.2 |
| Last updated | 2026-10-13 |
| Status | confirmed 2026-10-13 — 2 low-impact items remain open (lunar day 29, response-time target); ready for `il-spec-writer` |
| Owner / author | duc@wee.vn (BA assist: Claude) |
| Downstream handoff | `il-spec-writer` → `requirements.md` / `design.md` / `flow.md` / `tasks.md` under `docs/specs/features/temple-event/` |

---

## 1. Introduction

### 1.1 Purpose

This document specifies, in business terms, the events that an individual temple organizes (retreats, ceremonies, regular practice sessions) and that the server provides to users per temple. It is written for product and engineering and drives the engineering spec.

### 1.2 Scope

**In scope:**
- Events owned by one temple and linked to it: title, description, kind, time, status, audience note, contact / registration information, image.
- One-off events (single day or several consecutive days).
- Recurring events with two recurrence kinds: every lunar month on given lunar days (for example 14 and 30), and weekly on a given weekday. A recurring event may have an end date.
- Reading the upcoming events of one temple. Past events are not shown.
- A seed data file with **mock** events built on top of the existing temple catalogue, loaded into the database by a CLI.
- Data kept in the database so that back-office can manage it later.

**Out of scope:**
- Any relation to the shared Buddhist / Vietnamese / folk calendar events (`calendar` module). Temple events are independent of them. *(Decision 2026-10-13.)*
- Who creates, edits or deletes temple events. For this release the only write path is the seed CLI; an admin role with permissions comes later. *(Decision 2026-10-13.)*
- Registering for an event inside the app, payments, capacity management, reminders and push notifications. Registration is shown as information only.
- Following a temple, and feeds across many temples.
- Skipping, postponing or cancelling a single occurrence of a recurring event (status applies to the whole event).
- Event photo upload (no object storage yet); the image is a URL.
- Languages other than Vietnamese.

### 1.3 Product overview

InnerLamp is a Buddhist-practice companion app (memorials, sutras, temples, genealogy, lamps, calendar). Users already browse a catalogue of temples. This feature adds, on each temple, what that temple is holding and when, so a user can decide whether to visit.

### 1.4 Definitions, acronyms, abbreviations

| Term | Definition |
|---|---|
| Temple | A record in the existing temple catalogue (chùa, thiền viện, đạo tràng, tịnh xá, thiền xá). |
| Temple event | A programme organized by one temple, for example a summer retreat or a regular repentance ceremony. Owned by exactly one temple. |
| Occurrence | A temple event on one concrete date. A one-off event has one (or one run of consecutive days); a recurring event has many. |
| Recurrence | The rule by which an event repeats: `none`, every lunar month on chosen lunar days, or weekly on a chosen weekday. |
| Lunar date (âm lịch) | Vietnamese lunar date: day, month, lunar year, and whether the month is a leap month. |
| Leap month (tháng nhuận) | An extra lunar month inserted in some lunar years, sharing its number with the previous month. |
| Status | `scheduled`, `postponed` or `cancelled`. |
| Mock data | Invented events used for development; not announced by any real temple. |
| BO | Back-office. |

### 1.5 References

- `docs/product/calendar/product.md` — shared calendar events, lunar conversion rules (reused here, not extended).
- `docs/specs/features/temple/requirements.md` — temple catalogue this feature attaches to.
- Web research 2026-10-13 (news and travel articles, not official temple announcements) on the kinds of events temples hold: periodic repentance on lunar 14 and 30, Bát quan trai retreats on lunar 15 and 30, summer retreats with registration, retreat postponement.

### 1.6 Document organization

Section 2 describes context and constraints, section 3 holds the numbered requirements with Gherkin acceptance criteria, section 4 the traceability matrix, and Appendix A the open and resolved questions.

---

## 2. Overall description

### 2.1 Product perspective

A new enhancement of the existing temple area. It reads the temple catalogue (each event belongs to one existing temple) and reuses the server's lunar-calendar conversion already built for the `calendar` module. It has no link to calendar events.

### 2.2 Product functions

- List the upcoming events of a temple (`FR-001`, `FR-006`, `FR-011`).
- Show event details (`FR-002`).
- Express one-off, multi-day and recurring events (`FR-003`, `FR-004`, `FR-005`).
- Show status of an event (`FR-007`).
- Show the lunar date next to each occurrence (`FR-008`).
- Load mock data from a seed file (`FR-009`) and keep it in database rows (`FR-010`).

### 2.3 User classes and characteristics

| Actor | Description | Characteristics / permissions |
|---|---|---|
| App user | Signed-in practitioner or visitor using the iOS app. | Read only; must be signed in (`FR-012`). |
| Team member | Prepares the seed file and runs the seed CLI. | Writes data outside the app; no in-app permissions. |
| Admin (future) | Back-office operator. | Not in this release; roles and permissions come later. |

### 2.4 Operating environment

- Platform: iOS client calling the existing Go backend (`services/appService`).
- Network: online required.

### 2.5 Design and implementation constraints

- Backend follows the repo's Clean Architecture layout and the existing "read catalogue + seed by CLI upsert" shape used by `temple` and `calendar`.
- Lunar conversion is valid only for years 1900..2100 in UTC+7, same as `calendar`.
- Non-sensitive, environment-dependent values (time zone, maximum window) live in service config, not in domain constants.
- Content must be in database rows, not in code, so that BO can manage it later.

### 2.6 User documentation

**N/A** — no separate manual; the client shows self-explanatory fields.

### 2.7 Assumptions and dependencies

- **Assumptions:** one event has one start time; a temple holding two sessions in a day enters two events. Dates and times are in UTC+7. Registration information is free text or a link.
- **Dependencies:** the temple catalogue (`temples`), the lunar conversion of the `calendar` module, the seed CLI pattern of existing catalogues.

---

## 3. Specific requirements

### 3.1 External interface requirements

#### 3.1.1 User interfaces
A list of upcoming events inside a temple's detail, and an event detail. Screens are designed by the client team; this SRS only fixes the information shown.

#### 3.1.2 Hardware interfaces
**N/A** — no custom hardware interface.

#### 3.1.3 Software interfaces
One read operation: upcoming events of a given temple, optionally limited to a date window. Contract not yet confirmed; it is defined in the engineering spec.

#### 3.1.4 Communication interfaces
HTTPS, JSON, same envelope and error conventions as the other appService routes.

### 3.2 Functional requirements

#### Reading events

**FR-001 — List upcoming events of a temple**
- **Priority:** Must
- **Description:** WHEN a user asks for the events of a temple THE SYSTEM SHALL return the occurrences of that temple's active events that have not ended yet, ordered by start time ascending.
- **Verification method:** unit test + black-box E2E
- **Source:** conversation 2026-10-13
- **Acceptance criteria:**
  ```gherkin
  Scenario: Temple has upcoming events
    Given a temple has three active events starting on different future dates
    When the user asks for the events of that temple
    Then the user sees the three events, earliest first

  Scenario: Temple has no events
    Given a temple has no events
    When the user asks for the events of that temple
    Then the user sees an empty list, not an error

  Scenario: Temple does not exist
    Given no temple matches the requested temple
    When the user asks for its events
    Then the request is rejected as "not found"

  Scenario: Events of another temple are excluded
    Given temple A has two events and temple B has one event
    When the user asks for the events of temple A
    Then the user sees only the two events of temple A
  ```

**FR-002 — Event information**
- **Priority:** Must
- **Description:** WHEN an occurrence is returned THE SYSTEM SHALL include the event's title, description, kind, start, end (or all-day marker), status, audience note, contact or registration information, image, the temple it belongs to, and whether it recurs.
- **Verification method:** unit test + E2E
- **Source:** conversation 2026-10-13
- **Acceptance criteria:**
  ```gherkin
  Scenario: Full information is shown
    Given an event has a title, description, kind, audience note and registration contact
    When the user views its occurrence
    Then all of those fields are shown

  Scenario: Optional information is absent
    Given an event has no image and no audience note
    When the user views its occurrence
    Then the event is shown without those fields and nothing is invented
  ```

**FR-003 — One-off and multi-day events**
- **Priority:** Must
- **Description:** WHEN an event is not recurring THE SYSTEM SHALL treat it as one occurrence from its start to its end, and WHEN it spans several days THE SYSTEM SHALL show it on every day it covers.
- **Verification method:** unit test
- **Source:** conversation 2026-10-13
- **Acceptance criteria:**
  ```gherkin
  Scenario: Single-day event
    Given an event starts and ends on the same day
    When the user views events for that day
    Then the event appears once

  Scenario: Three-day retreat
    Given an event runs from the 10th to the 12th of a month
    When the user views events for the 11th
    Then the event appears

  Scenario: Event crossing midnight
    Given an event starts at 22:00 and ends at 02:00 the next day
    When the user views the event
    Then it is shown as starting on its first day and ending on the next day

  Scenario: End before start
    Given an event is entered with an end earlier than its start
    When the data is loaded
    Then the whole load is rejected
  ```

#### Recurrence

**FR-004 — Recurring events**
- **Priority:** Must
- **Description:** WHEN an event recurs every lunar month on chosen lunar days, or weekly on a chosen weekday, THE SYSTEM SHALL produce an occurrence for each matching date from the event's first date up to its optional end date, with the same start time on each occurrence.
- **Verification method:** unit test (table-driven over known years) + E2E
- **Source:** conversation 2026-10-13; temple research (lunar 14 and 30, 15 and 30)
- **Acceptance criteria:**
  ```gherkin
  Scenario Outline: Lunar monthly recurrence
    Given an event repeats on lunar days <days> every month
    When the user views the next three months
    Then there is one occurrence for each of those lunar days in each lunar month

    Examples:
      | days   |
      | 14, 30 |
      | 15     |

  Scenario: Weekly recurrence
    Given an event repeats every Sunday at 08:00
    When the user views the next four weeks
    Then four occurrences appear, each on a Sunday at 08:00

  Scenario: Leap month
    Given an event repeats on lunar day 15 every month
    And the viewed period contains a leap month
    When the user views the period
    Then the leap month also has an occurrence on its day 15

  Scenario: Recurrence has ended
    Given a recurring event ended last month
    When the user views its temple's events
    Then no occurrence of it appears

  Scenario: Recurrence end before first date
    Given a recurring event is entered with an end date before its first date
    When the data is loaded
    Then the whole load is rejected
  ```

**FR-005 — Day 30 in a 29-day lunar month**
- **Priority:** Must
- **Description:** WHEN a monthly lunar event is set on day 30 and a lunar month has only 29 days THE SYSTEM SHALL place the occurrence on the last day of that month. The server, not the client, computes this.
- **Verification method:** unit test
- **Source:** conversation 2026-10-13 (Q2)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Short month
    Given an event repeats on lunar day 30
    And a lunar month in the viewed period has 29 days
    When the user views that month
    Then the occurrence is on day 29, the last day of the month

  Scenario: Full month
    Given an event repeats on lunar day 30
    And a lunar month has 30 days
    When the user views that month
    Then the occurrence is on day 30

  Scenario: Event on days 29 and 30 in a short month
    Given an event repeats on lunar days 29 and 30
    And the lunar month has 29 days
    When the user views that month
    Then the event appears once in that month, not twice
  ```

> ⚠️ Not confirmed: should an event set on lunar day 29 also move to "last day" in months with 30 days, or stay on day 29 — blocks only the edge case where an event is set on day 29; default is stay on day 29.

**FR-006 — Past events are not shown**
- **Priority:** Must
- **Description:** WHEN the current moment (in the configured time zone) is after an occurrence's end THE SYSTEM SHALL NOT return that occurrence; an occurrence in progress is still returned.
- **Verification method:** unit test with a fixed clock + E2E
- **Source:** conversation 2026-10-13 (Q3)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Finished event is hidden
    Given an event ended yesterday
    When the user views the temple's events
    Then it does not appear

  Scenario: Event in progress is shown
    Given a three-day retreat is on its second day
    When the user views the temple's events
    Then it appears

  Scenario: Event later today is shown
    Given an event starts this evening
    When the user views events this morning
    Then it appears

  Scenario: Recurring event shows only future occurrences
    Given a weekly event had occurrences last week
    When the user views the temple's events
    Then only occurrences from now on appear
  ```

**FR-007 — Status**
- **Priority:** Must
- **Description:** WHEN an event is `postponed` or `cancelled` THE SYSTEM SHALL still return it with that status, so the user is told instead of finding it missing.
- **Verification method:** unit test
- **Source:** temple research (a temple postponed registered retreats)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Postponed event
    Given an event is marked postponed
    When the user views the temple's events
    Then the event appears with a "postponed" status

  Scenario: Cancelled event
    Given an event is marked cancelled
    When the user views the temple's events
    Then the event appears with a "cancelled" status
  ```

**FR-008 — Lunar date**
- **Priority:** Must
- **Description:** WHEN an occurrence is returned THE SYSTEM SHALL include its lunar date (day, month, leap-month flag) as well as its solar date.
- **Verification method:** unit test against known dates
- **Source:** conversation 2026-10-13
- **Acceptance criteria:**
  ```gherkin
  Scenario: Both dates shown
    Given an event on a known solar date
    When the user views its occurrence
    Then the user sees the solar date and the matching lunar date

  Scenario: Lunar date outside supported years
    Given an event is entered with a date before 1900 or after 2100
    When the data is loaded
    Then the whole load is rejected
  ```

**FR-011 — Date window**
- **Priority:** Should
- **Description:** WHEN the request gives a start and end date THE SYSTEM SHALL return only occurrences within that window, and SHALL reject a window longer than the configured maximum (3 months). WHEN the request gives no window THE SYSTEM SHALL use today up to 3 months ahead. Both values come from service config.
- **Verification method:** unit test + E2E
- **Source:** technical limit discussion 2026-10-13
- **Acceptance criteria:**
  ```gherkin
  Scenario: Window within limit
    Given a window of one month
    When the user asks for it
    Then only occurrences in that month appear

  Scenario: Window too long
    Given a window longer than the allowed maximum
    When the user asks for it
    Then the request is rejected with a message about the maximum

  Scenario: Window in the past
    Given a window entirely before today
    When the user asks for it
    Then the user sees an empty list

  Scenario: No window given
    Given the user gives no window
    When the user asks for a temple's events
    Then occurrences from today up to three months ahead appear
  ```

#### Data

**FR-009 — Mock seed data**
- **Priority:** Must
- **Description:** WHEN the seed CLI is run with a JSON file THE SYSTEM SHALL upsert the temple events keyed by stable ids, in one transaction, create no duplicates, overwrite existing rows with the file's content, and write nothing if the file is invalid (unknown temple, duplicate id, missing recurrence parameters, end before start, unknown kind or status).
- **Verification method:** repository test on a real Postgres + CLI test
- **Source:** conversation 2026-10-13 (Q1)
- **Acceptance criteria:**
  ```gherkin
  Scenario: First load
    Given an empty set of temple events and a valid file
    When the seed is run
    Then every event in the file exists and belongs to its temple

  Scenario: Second load of the same file
    Given the file was already loaded
    When the seed is run again
    Then no duplicates exist and the events are unchanged

  Scenario: Edited event
    Given the file was loaded and an event's title was changed in the file
    When the seed is run again
    Then the stored event has the new title

  Scenario: Unknown temple
    Given an event in the file refers to a temple that is not in the catalogue
    When the seed is run
    Then nothing is written and the error names the event

  Scenario: Recurring event missing its parameters
    Given a monthly lunar event has no lunar days
    When the seed is run
    Then nothing is written and the error names the event
  ```

The seed content is mock data for development. No sample-content marking is required in this release *(decision 2026-10-13)*; the data must be replaced with real content before any release to real users.

**FR-010 — Data lives in the database**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL keep temple events, their recurrence and status in database rows, not in code or embedded files, so that a back-office can later change them without a release; an inactive event SHALL NOT be returned.
- **Verification method:** code review + repository test
- **Source:** conversation 2026-10-13
- **Acceptance criteria:**
  ```gherkin
  Scenario: Inactive event hidden
    Given an event is marked inactive
    When the user views the temple's events
    Then the event does not appear

  Scenario: Temple removed from the catalogue
    Given a temple is no longer active in the catalogue
    When the user asks for its events
    Then the request is treated as a temple that does not exist
  ```

**FR-012 — Sign-in required**
- **Priority:** Must
- **Description:** WHEN a request for temple events has no valid access token (missing, expired, or not an access token) THE SYSTEM SHALL reject it and return no events.
- **Verification method:** black-box E2E (route wired with the auth middleware)
- **Source:** conversation 2026-10-13
- **Acceptance criteria:**
  ```gherkin
  Scenario: Signed-in user
    Given the user is signed in
    When the user asks for the events of a temple
    Then the events are returned

  Scenario: Not signed in
    Given the user is not signed in
    When the user asks for the events of a temple
    Then the request is rejected as unauthorized and no events are shown

  Scenario: Expired session
    Given the user's session has expired
    When the user asks for the events of a temple
    Then the request is rejected as unauthorized and no events are shown
  ```

### 3.3 Performance requirements

- **NFR-001 — Bounded work per request:** one request covers exactly one temple and a window no longer than 3 months, so the number of occurrences computed is bounded. Response time target not confirmed.
- **NFR-002 — Determinism:** two requests with the same temple, window and "now" SHALL return identical occurrences in identical order.
- **NFR-003 — Lunar accuracy:** lunar dates SHALL match the reference used by the `calendar` module for every year 1900..2100.

> ⚠️ Not confirmed: response-time target (for example p95 under 300 ms for one temple and one month) — blocks NFR-001's measurable threshold.

### 3.4 Logical database requirements

- The server persists temple events (with their recurrence, status and activation flag), each linked to one catalogue temple.
- Lunar dates are **not** stored; they are computed when returned.
- No retention or deletion rule in this release; BO will define it later.

### 3.5 Design constraints

- Same layer layout, error envelope and seed-CLI conventions as `calendar` and `temple`.
- The seed file lives with the other seed data files under `assets/data/`.

### 3.6 Software system attributes

- **Reliability:** an invalid seed file never leaves partial data.
- **Availability:** follows the service's availability; no extra target.
- **Security:** events are public-interest, non-personal content; no personal data is stored. The read route requires sign-in (`FR-012`). The registration contact may be a phone number of a temple: treat it as business contact information, not user data.
- **Maintainability:** all content in rows so BO can take over writes.
- **Portability:** N/A.

---

## 4. Verification & traceability

### 4.1 Verification approach

Unit tests with a fixed clock for recurrence and date logic (including leap months and 29-day months), repository tests on a real Postgres for seed idempotency, and a black-box E2E over the compiled binary for the read route.

### 4.2 Traceability matrix

| Requirement ID | Source | Design | Test |
|---|---|---|---|
| FR-001 | Conversation 2026-10-13 | → il-spec-writer | 4 scenarios |
| FR-002 | Conversation 2026-10-13 | → il-spec-writer | 2 scenarios |
| FR-003 | Conversation 2026-10-13 | → il-spec-writer | 4 scenarios |
| FR-004 | Conversation 2026-10-13 | → il-spec-writer | 5 scenarios |
| FR-005 | Conversation 2026-10-13 (Q2) | → il-spec-writer | 3 scenarios |
| FR-006 | Conversation 2026-10-13 (Q3) | → il-spec-writer | 4 scenarios |
| FR-007 | Temple research | → il-spec-writer | 2 scenarios |
| FR-008 | Conversation 2026-10-13 | → il-spec-writer | 2 scenarios |
| FR-009 | Conversation 2026-10-13 (Q1) | → il-spec-writer | 5 scenarios |
| FR-010 | Conversation 2026-10-13 | → il-spec-writer | 2 scenarios |
| FR-011 | Technical limits | → il-spec-writer | 4 scenarios |
| FR-012 | Conversation 2026-10-13 | → il-spec-writer | 3 scenarios |
| NFR-001 | Technical limits | → il-spec-writer | TBD |
| NFR-002 | Technical limits | → il-spec-writer | TBD |
| NFR-003 | Reuse of `calendar` | → il-spec-writer | TBD |

---

## 5. Appendices

### Appendix A — Open questions

#### Open (unconfirmed)

Two low-impact items, recorded inline where they apply:

> ⚠️ Not confirmed: should an event set on lunar day 29 also move to "last day" in months with 30 days — blocks only the day-29 edge case (FR-005); default is stay on day 29.

> ⚠️ Not confirmed: response-time target (for example p95 under 300 ms for one temple and one month) — blocks the measurable threshold of NFR-001.

#### Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| Are temple events related to the shared Buddhist calendar events? | No. Temple events are the temple's own and independent. | 2026-10-13 |
| Who creates and edits temple events? | Not in this release; future admin with permissions. Data comes from a seed JSON. | 2026-10-13 |
| Where does the data come from for now? | Mock events on the existing temple list, in a JSON seed file loaded into the database. | 2026-10-13 |
| Day 30 in a 29-day lunar month? | Last day of the month. Computed by the server, not the client. | 2026-10-13 |
| Show past events? | No. | 2026-10-13 |
| Single table or separate table for regular schedules? | One table with recurrence fields; per-occurrence exceptions deferred. | 2026-10-13 |
| Registration in the app? | No; information only. | 2026-10-13 |
| Does the read route require sign-in? | Yes (`FR-012`). | 2026-10-13 |
| Show cancelled events? | Yes, with a "cancelled" status (`FR-007`). | 2026-10-13 |
| Maximum and default date window? | 3 months, from service config (`FR-011`). | 2026-10-13 |
| Mark mock events as sample content? | No; development only, replace with real content before release (`FR-009`). | 2026-10-13 |
