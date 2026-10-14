# Software Requirements Specification — Temple Schedule

| Field | Value |
|---|---|
| Document type | Software Requirements Specification (SRS) — IEEE 29148-2018 |
| Feature | Temple Schedule (thời khoá tu tập của chùa) |
| Product | InnerLamp |
| Version | 0.3 |
| Last updated | 2026-10-13 |
| Status | confirmed 2026-10-13 — all questions answered; v0.3 adds instructor, location, description, steps and preparation; ready for `il-spec-writer` |
| Owner / author | duc@wee.vn (BA assist: Claude) |
| Downstream handoff | `il-spec-writer` → `requirements.md` / `design.md` / `flow.md` / `tasks.md` under `docs/specs/features/temple-schedule/` |

---

## 1. Introduction

### 1.1 Purpose

This document specifies, in business terms, the daily practice schedule of a temple (for example morning ceremony, noon meal, evening chanting) that the server provides per temple. It is written for product and engineering and drives the engineering spec.

### 1.2 Scope

**In scope:**
- A list of schedule items owned by one temple and linked to it: name, start time, optional end time, the days it applies to, optional description, optional instructor, optional location inside the temple, optional preparation notes, an optional ordered list of steps, display order, active flag.
- Reading the active schedule of one temple.
- Items that apply every day, or only on chosen weekdays.
- Items that cross midnight.
- Data kept in database rows so that back-office (BO) can add, edit and switch off each item later.
- A seed data file with **mock** schedules built on top of the existing temple catalogue, loaded into the database by a CLI.

**Out of scope:**
- Temple events (retreats, ceremonies on a specific date): a separate feature, `temple-event`. A schedule item is a standing routine; it has no date, no status, no registration and no occurrences. *(Decision 2026-10-13.)*
- Any link to the shared Buddhist calendar events (`calendar` module).
- BO screens and APIs to add, edit or switch off items, and their permissions (later). This release only needs the data to be in rows that BO can edit.
- Seasonal variants of the schedule (for example the summer retreat season, an cư kiết hạ) as a modelled concept. A temple changes its items instead. *(Decision 2026-10-13.)*
- Items that apply on chosen lunar days (for example 14 and 30): those are temple events (`temple-event`); a schedule item applies by weekday only. *(Decision 2026-10-13.)*
- A flag saying whether lay visitors may take part or an item is for monastics only; the description covers it. *(Decision 2026-10-13.)*
- An instructor or a location per step: the instructor and the location apply to the whole item, not to each step.
- Structured instructor records (a person entity, a monk directory) and structured locations (a catalogue of halls): both are free text in this release.
- Reminders, alarms and push notifications for schedule times.
- One-off changes ("no afternoon ceremony this week"): those are announcements, handled as a temple event.
- Languages other than Vietnamese.

### 1.3 Product overview

InnerLamp is a Buddhist-practice companion app (memorials, sutras, temples, genealogy, lamps, calendar). Users browse a catalogue of temples; each temple shows its address, opening hours and visiting notes. This feature adds what a temple does at which time of day, so a visitor knows when to arrive for a ceremony.

### 1.4 Definitions, acronyms, abbreviations

| Term | Definition |
|---|---|
| Temple | A record in the existing temple catalogue (chùa, thiền viện, đạo tràng, tịnh xá, thiền xá). |
| Schedule item (thời khoá) | One routine practice of a temple at a time of day, for example "Công phu sáng" (morning ceremony) at 04:00. Owned by exactly one temple. |
| Applies-on days | The weekdays on which a schedule item takes place. Default: every day. |
| Description | What the practice is, in a few sentences (for example the meaning of the ceremony, who may attend). |
| Instructor (người hướng dẫn) | The monastic or person who leads the item, as free text (for example "Thượng toạ Thích …"). |
| Location (địa điểm) | Where inside the temple the item takes place, as free text (for example "Chánh điện" — main hall, "Giảng đường" — lecture hall). |
| Step (trình tự) | One part of the course of an item, in order, with an optional time of day and a short text (for example 08:00 "Niệm Phật", 09:00 "Thuyết pháp"). |
| Preparation (cần chuẩn bị) | What a participant should bring or do beforehand (for example "Mặc áo tràng, mang theo kinh sách"). |
| Active | A schedule item shown to users. An inactive item is hidden; it models "the temple stopped this practice". |
| Temple event | A programme of a temple on specific dates; separate feature (`temple-event`). |
| Mock data | Invented schedules used for development; not announced by any real temple. |
| BO | Back-office. |

### 1.5 References

- `docs/specs/features/temple-event/SRS.md` — sibling feature; boundary between an event and a schedule item.
- `docs/specs/features/temple/requirements.md` — the temple catalogue this feature attaches to; it already carries a free-text opening-hours field.

### 1.6 Document organization

Section 2 describes context and constraints, section 3 holds the numbered requirements with Gherkin acceptance criteria, section 4 the traceability matrix, and Appendix A the open and resolved questions.

---

## 2. Overall description

### 2.1 Product perspective

An enhancement of the existing temple area, shown with a temple's detail. It reads the temple catalogue (each item belongs to one existing temple). It is independent of `temple-event` and of `calendar`. It does not replace the temple's existing free-text opening hours, which describes when the temple is open, not what it practises.

### 2.2 Product functions

- List the active schedule items of a temple, in time order (`FR-001`, `FR-002`).
- Express items for every day or for chosen weekdays (`FR-003`).
- Express end time and items crossing midnight (`FR-004`).
- Describe each item: instructor, location, description, preparation (`FR-002`) and the ordered steps (`FR-010`).
- Hide an item without losing it (`FR-005`).
- Handle a temple with no schedule and an unknown temple (`FR-006`).
- Keep data in rows, loaded from a mock seed file (`FR-007`, `FR-008`).

### 2.3 User classes and characteristics

| Actor | Description | Characteristics / permissions |
|---|---|---|
| App user | Signed-in practitioner or visitor using the iOS app. | Read only; must be signed in (`FR-009`). |
| Team member | Prepares the seed file and runs the seed CLI. | Writes data outside the app. |
| Admin (future) | Back-office operator. | Adds, edits and switches off items; roles and permissions come later, not in this release. |

### 2.4 Operating environment

- Platform: iOS client calling the existing Go backend (`services/appService`).
- Network: online required.

### 2.5 Design and implementation constraints

- Backend follows the repo's Clean Architecture layout and the existing "read catalogue + seed by CLI upsert" shape used by `temple`, `calendar` and `temple-event`.
- Non-sensitive, environment-dependent values live in service config, not in domain constants.
- Content must be in database rows, not in code or embedded files.

### 2.6 User documentation

**N/A** — no separate manual; the client shows self-explanatory fields.

### 2.7 Assumptions and dependencies

- **Assumptions:** times are Vietnam local time (UTC+7) and carry no date; one row is one practice (a temple with two sessions enters two items); the schedule of a temple is small (a handful of items).
- **Dependencies:** the temple catalogue (`temples`) and the seed CLI pattern of existing catalogues.

---

## 3. Specific requirements

### 3.1 External interface requirements

#### 3.1.1 User interfaces
A "thời khoá" list inside the temple's detail: name and time of each item, the days it applies to when it is not every day, and the location when present. Opening an item shows the instructor, description, steps and preparation. Screens are designed by the client team.

#### 3.1.2 Hardware interfaces
**N/A** — no custom hardware interface.

#### 3.1.3 Software interfaces
One read operation, requiring sign-in: the active schedule of a given temple. Because the existing temple detail is public, the schedule is a separate operation, not part of that response; the contract is defined in the engineering spec.

#### 3.1.4 Communication interfaces
HTTPS, JSON, same envelope and error conventions as the other appService routes.

### 3.2 Functional requirements

#### Reading the schedule

**FR-001 — List a temple's schedule**
- **Priority:** Must
- **Description:** WHEN a user asks for the schedule of a temple THE SYSTEM SHALL return that temple's active items ordered by start time of day ascending, then by display order, then by a stable identifier.
- **Verification method:** unit test + black-box E2E
- **Source:** conversation 2026-10-13
- **Acceptance criteria:**
  ```gherkin
  Scenario: Temple has a schedule
    Given a temple has three active items at 04:00, 11:00 and 18:30
    When the user asks for the schedule of that temple
    Then the user sees the three items in that time order

  Scenario: Two items at the same time
    Given a temple has two active items both at 19:00 with display order 1 and 2
    When the user asks for the schedule of that temple
    Then the item with display order 1 comes first

  Scenario: Items of other temples are excluded
    Given temple A has two items and temple B has one item
    When the user asks for the schedule of temple A
    Then the user sees only the two items of temple A
  ```

**FR-002 — Item information**
- **Priority:** Must
- **Description:** WHEN an item is returned THE SYSTEM SHALL include its name, start time, end time when it has one, the days it applies to, the temple it belongs to, and — each only when it has one — its description, instructor, location, preparation and steps.
- **Verification method:** unit test + E2E
- **Source:** conversation 2026-10-13
- **Acceptance criteria:**
  ```gherkin
  Scenario: Item with all fields
    Given an item has a name, a start time, an end time, chosen weekdays, a description, an instructor, a location, preparation notes and steps
    When the user views the schedule
    Then all of those fields are shown

  Scenario: Item with only the required fields
    Given an item has only a name and a start time
    When the user views the schedule
    Then it is shown with no end time, no description, no instructor, no location, no preparation and no steps, and nothing is invented

  Scenario: Instructor and location only
    Given an item has an instructor and a location but no description
    When the user views the schedule
    Then the instructor and the location are shown and no description is invented

  Scenario: Start time has no date
    Given an item starts at 04:00
    When the user views the schedule
    Then it is shown as a time of day without a date
  ```

**FR-003 — Days an item applies to**
- **Priority:** Must
- **Description:** WHEN an item has no chosen weekdays THE SYSTEM SHALL treat it as applying every day; WHEN it has chosen weekdays THE SYSTEM SHALL return exactly those weekdays so the client can show or filter by day.
- **Verification method:** unit test
- **Source:** conversation 2026-10-13
- **Acceptance criteria:**
  ```gherkin
  Scenario: Every day
    Given an item has no chosen weekdays
    When the user views the schedule
    Then the item is shown as applying every day

  Scenario: Weekend only
    Given an item applies on Saturday and Sunday
    When the user views the schedule
    Then the item is shown as applying on those two days only

  Scenario: All seven days chosen
    Given an item has all seven weekdays chosen
    When the user views the schedule
    Then it is shown as applying every day, the same as an item with none chosen
  ```

**FR-004 — End time and midnight**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL accept an item with no end time, an end time later than its start, or an end time earlier than its start (the item ends after midnight), and SHALL reject an end time equal to the start time.
- **Verification method:** unit test (data validation) + seed test
- **Source:** conversation 2026-10-13
- **Acceptance criteria:**
  ```gherkin
  Scenario: No end time
    Given an item has a start time and no end time
    When the data is loaded
    Then the item is stored

  Scenario: Ends after midnight
    Given an item starts at 22:00 and ends at 00:30
    When the user views the schedule
    Then it is shown from 22:00 to 00:30

  Scenario: End equals start
    Given an item is entered with the same start and end time
    When the data is loaded
    Then the whole load is rejected
  ```

**FR-010 — Steps of an item**
- **Priority:** Must
- **Description:** WHEN an item has steps THE SYSTEM SHALL return them in the explicit order set for them, each with its text and, when it has one, a time of day; the order SHALL NOT be derived from the times, so a course that passes midnight keeps its order. A step belongs to exactly one item and disappears from the result when its item is inactive.
- **Verification method:** unit test + seed test
- **Source:** conversation 2026-10-13 ("trình tự 8h làm gì, 9h làm gì")
- **Acceptance criteria:**
  ```gherkin
  Scenario: Steps in order
    Given an item has three steps at 08:00, 09:00 and 10:30
    When the user views the item
    Then the steps appear in that order, each with its time and text

  Scenario: Step without a time
    Given an item has a step with text only and no time
    When the user views the item
    Then the step appears in its place in the order, with no time

  Scenario: Course passing midnight
    Given an item has steps at 23:00, 23:45 and 00:15 in that order
    When the user views the item
    Then the steps appear in that order, not sorted by clock time

  Scenario: Item without steps
    Given an item has no steps
    When the user views the item
    Then it is shown with an empty list of steps, not an error

  Scenario: Step with an empty text
    Given a step is entered with an empty text
    When the data is loaded
    Then the whole load is rejected

  Scenario: Two steps with the same position
    Given two steps of one item are entered with the same position
    When the data is loaded
    Then the whole load is rejected

  Scenario: Item switched off
    Given an item with steps is switched off
    When the user views the schedule
    Then neither the item nor its steps appear
  ```

Step times are informative: they are not required to lie between the item's start and end time. *(Assumption 2026-10-13.)*

#### Switching items on and off

**FR-005 — Inactive items are hidden**
- **Priority:** Must
- **Description:** WHEN an item is inactive THE SYSTEM SHALL NOT return it; switching an item off SHALL NOT delete it.
- **Verification method:** repository test + E2E
- **Source:** conversation 2026-10-13 ("huỷ một thời khoá")
- **Acceptance criteria:**
  ```gherkin
  Scenario: Item switched off
    Given a temple has an active item that is then switched off
    When the user views the schedule
    Then the item no longer appears

  Scenario: Item switched back on
    Given an item was switched off and is switched on again
    When the user views the schedule
    Then the item appears again with its previous content

  Scenario: Every item switched off
    Given all items of a temple are inactive
    When the user views the schedule
    Then the user sees an empty list
  ```

#### Empty and missing

**FR-006 — Temple without schedule or unknown temple**
- **Priority:** Must
- **Description:** WHEN the temple exists and has no active item THE SYSTEM SHALL return an empty list, not an error; WHEN no temple matches THE SYSTEM SHALL reject the request as "not found".
- **Verification method:** unit test + E2E
- **Source:** conversation 2026-10-13
- **Acceptance criteria:**
  ```gherkin
  Scenario: Temple has no schedule
    Given a temple exists and has no schedule items
    When the user asks for its schedule
    Then the user sees an empty list, not an error

  Scenario: Temple does not exist
    Given no temple matches the requested temple
    When the user asks for its schedule
    Then the request is rejected as "not found"
  ```

#### Data

**FR-007 — Data lives in the database**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL keep schedule items, their weekdays, steps, order and active flag in database rows, not in code or embedded files, so that a back-office can later add, edit and switch off each item without a release.
- **Verification method:** code review + repository test
- **Source:** conversation 2026-10-13
- **Acceptance criteria:**
  ```gherkin
  Scenario: Row edited directly
    Given an item's start time is changed in the database
    When the user views the schedule
    Then the new time is shown without any release

  Scenario: Item added directly
    Given a new item row is added for a temple in the database
    When the user views the schedule
    Then the new item appears in time order
  ```

**FR-008 — Mock seed data**
- **Priority:** Must
- **Description:** WHEN the seed CLI is run with a JSON file THE SYSTEM SHALL upsert the items keyed by stable ids, in one transaction, create no duplicates, overwrite existing rows with the file's content, leave the active flag of existing rows untouched, and write nothing if the file is invalid (unknown temple, duplicate id, end equal to start, malformed time, weekday outside Monday–Sunday, empty name, a step with empty text, a malformed step time, two steps of one item with the same position, unknown key).
- **Verification method:** repository test on a real Postgres + CLI test
- **Source:** conversation 2026-10-13
- **Acceptance criteria:**
  ```gherkin
  Scenario: First load
    Given no schedule items and a valid file
    When the seed is run
    Then every item in the file exists and belongs to its temple

  Scenario: Second load of the same file
    Given the file was already loaded
    When the seed is run again
    Then no duplicates exist and the items are unchanged

  Scenario: Switched-off item survives a reload
    Given an item was switched off after the first load
    When the seed is run again with the same file
    Then the item is still switched off

  Scenario: Unknown temple
    Given an item in the file refers to a temple that is not in the catalogue
    When the seed is run
    Then nothing is written and the error names the item

  Scenario: Malformed time
    Given an item in the file has a start time that is not a valid time of day
    When the seed is run
    Then nothing is written and the error names the item
  ```

The seed content is mock data for development; it must be replaced with real content before any release to real users.

**FR-009 — Sign-in required**
- **Priority:** Must
- **Description:** WHEN a request for a temple's schedule has no valid access token (missing, expired, or not an access token) THE SYSTEM SHALL reject it and return no items.
- **Verification method:** black-box E2E (route wired with the auth middleware)
- **Source:** conversation 2026-10-13
- **Acceptance criteria:**
  ```gherkin
  Scenario: Signed-in user
    Given the user is signed in
    When the user asks for the schedule of a temple
    Then the schedule is returned

  Scenario: Not signed in
    Given the user is not signed in
    When the user asks for the schedule of a temple
    Then the request is rejected as unauthorized and no items are shown

  Scenario: Expired session
    Given the user's session has expired
    When the user asks for the schedule of a temple
    Then the request is rejected as unauthorized and no items are shown
  ```

### 3.3 Performance requirements

- **NFR-001 — Bounded size:** one request covers exactly one temple and its few items; no pagination is needed. There is no cap on the number of items per temple *(decision 2026-10-13)*. A response-time target is not set.
- **NFR-002 — Determinism:** two requests for the same temple with unchanged data SHALL return identical items in identical order.

### 3.4 Logical database requirements

- The server persists schedule items, each linked to one catalogue temple, with start time of day, optional end time of day, chosen weekdays, description, instructor, location, preparation, display order and active flag.
- The server persists the steps of an item, each linked to one item, with its position, optional time of day and text.
- A time of day carries no date and no time zone: Vietnam local time.
- No retention or deletion rule in this release; BO will define it later. Switching off is the way to retire an item.

### 3.5 Design constraints

- Same layer layout, error envelope and seed-CLI conventions as `calendar`, `temple` and `temple-event`.
- The seed file lives with the other seed data files under `assets/data/`.

### 3.6 Software system attributes

- **Reliability:** an invalid seed file never leaves partial data.
- **Availability:** follows the service's availability.
- **Security:** non-personal content; no personal data stored. The read operation requires sign-in (FR-009).
- **Maintainability:** all content in rows so BO can take over writes.
- **Portability:** N/A.

---

## 4. Verification & traceability

### 4.1 Verification approach

Unit tests for ordering and validation, repository tests on a real Postgres for seed idempotency and the active flag, and a black-box E2E over the compiled binary for the read operation and its access rule.

### 4.2 Traceability matrix

| Requirement ID | Source | Design | Test |
|---|---|---|---|
| FR-001 | Conversation 2026-10-13 | → il-spec-writer | 3 scenarios |
| FR-002 | Conversation 2026-10-13 | → il-spec-writer | 3 scenarios |
| FR-010 | Conversation 2026-10-13 | → il-spec-writer | 7 scenarios |
| FR-003 | Conversation 2026-10-13 | → il-spec-writer | 3 scenarios |
| FR-004 | Conversation 2026-10-13 | → il-spec-writer | 3 scenarios |
| FR-005 | Conversation 2026-10-13 | → il-spec-writer | 3 scenarios |
| FR-006 | Conversation 2026-10-13 | → il-spec-writer | 2 scenarios |
| FR-007 | Conversation 2026-10-13 | → il-spec-writer | 2 scenarios |
| FR-008 | Conversation 2026-10-13 | → il-spec-writer | 5 scenarios (step rules covered by FR-010) |
| FR-009 | Conversation 2026-10-13 | → il-spec-writer | 3 scenarios |
| NFR-001 | Conversation 2026-10-13 | → il-spec-writer | TBD |
| NFR-002 | Conversation 2026-10-13 | → il-spec-writer | TBD |

---

## 5. Appendices

### Appendix A — Open questions

#### Open (unconfirmed)

None.

#### Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| Does the schedule require sign-in? | Yes (`FR-009`). As a consequence it is a separate operation, not part of the public temple detail. | 2026-10-13 |
| Maximum number of items per temple? | No cap (`NFR-001`). | 2026-10-13 |
| Flag for lay visitors vs monastics only? | Not in this release; the description covers it. | 2026-10-13 |
| Which extra information does an item carry? | Instructor, location inside the temple, description, preparation and an ordered list of steps (`FR-002`, `FR-010`). The earlier generic "note" is replaced by description and preparation. | 2026-10-13 |
| Instructor and location: structured or free text? | Free text, per item (not per step). | 2026-10-13 |
| Are steps ordered by their time? | No, by explicit position, so a course passing midnight keeps its order; step time is optional and informative. | 2026-10-13 |
| Items on chosen lunar days? | No; those are temple events. A schedule item applies by weekday only. | 2026-10-13 |
| Seasonal schedules (an cư kiết hạ)? | Out of scope; BO edits items when the season changes. | 2026-10-13 |
| Is a temple's practice schedule a temple event with a type, or its own data? | Its own table linked to the temple; not an event, no occurrences. | 2026-10-13 |
| Can the schedule change over time? | Yes. Items are rows that BO can add, edit and switch off. | 2026-10-13 |
| Where does the data come from for now? | Mock items on the existing temple list, in a JSON seed file loaded into the database (same approach as temple events). | 2026-10-13 |
| Who creates and edits items in this release? | No one in-app; admin back-office with permissions comes later. | 2026-10-13 |
