# Software Requirements Specification — Lamp Lighting

| Field | Value |
|---|---|
| Document type | Software Requirements Specification (SRS) — IEEE 29148-2018 |
| Feature | Lamp Lighting (thắp đèn) and Lamp Garden (vườn đèn) |
| Product | InnerLamp |
| Version | 0.4 |
| Last updated | 2026-10-10 |
| Status | approved 2026-10-10 — 27 questions answered; implemented in spec v1.2 (domain to delivery, verified end to end). 2 items deliberately deferred (see Appendix A) |
| Owner / author | duc@wee.vn (BA assist: Claude) |
| Downstream handoff | `il-spec-writer` → `requirements.md` / `design.md` / `flow.md` / `tasks.md` under `docs/specs/features/lamp/` (written at v1.0, current v1.2) |

---

## 1. Introduction

### 1.1 Purpose

This document specifies, in business terms, how a signed-in InnerLamp user lights a virtual lamp with an intention and a prayer, and how every signed-in user browses the garden of lamps that are currently lit. It is written for product and engineering, and it drives the engineering spec that `il-spec-writer` produces next.

### 1.2 Scope

**In scope:**
- Lighting a lamp: choosing a lamp type, an intention, a recipient, a prayer text and a burn duration.
- A per-user cap on lamps lit at the same time.
- Automatic end of a lamp's life when its burn duration is over.
- The lamp garden: a list of all users' currently lit lamps, filterable by lamp type, loaded in pages ("load more").
- Counts shown with the garden: the total number of lit lamps and the number per lamp type.

**Out of scope:**
- Lighting a lamp at a temple with a fee (planned for v2; v1 is free and has no payment).
- Payment, donation or offering of any kind.
- Moderation of prayer text (decision: none in v1).
- Extending or relighting a lit lamp.
- Putting out or deleting a lamp before it expires (decision: not allowed).
- A "my lamps" screen, including expired lamps.
- Filtering the garden by intention.
- Notifications (for example "your lamp is about to go out").
- The screens, animation and artwork of the lamps (this document defines behaviour, not UI).

### 1.3 Product overview

InnerLamp is a Buddhist-practice companion app. Users already keep memorials, browse sutras and temples, and manage a genealogy. Lamp lighting adds a devotional action: a user lights a lamp on behalf of someone, and the lamp stays visible in a shared garden for the chosen number of days, so users can see a community of lit lamps.

### 1.4 Definitions, acronyms, abbreviations

| Term | Definition |
|---|---|
| Lamp (đèn) | A virtual lamp lit by a user. It has a type, an intention, a recipient, an optional prayer, a lit time and an end time. |
| Lamp type (loại đèn) | One of three fixed kinds: oil lamp (đèn dầu), lotus candle (nến sen), floating flower lantern (hoa đăng). |
| Intention (tâm nguyện) | The category of wish: peace (cầu an), salvation of the departed (cầu siêu), merit dedication (hồi hướng), gratitude (tri ân). |
| Recipient (dành cho ai) | Free text naming who the lamp is lit for. |
| Prayer (lời nguyện) | Optional free text with the user's wish. Not the same as the Intention. |
| Burn duration (thời gian đèn sáng) | How long the lamp stays lit: 1, 7, 21 or 49 days. |
| Lit lamp | A lamp whose end time has not yet passed. |
| Expired lamp | A lamp whose end time has passed. It no longer appears in the garden and no longer counts toward the cap. |
| Lamp garden (vườn đèn) | The list of all users' lit lamps. |
| Lighter | The user who lit the lamp. |
| Anonymous lamp (vô danh) | A lamp whose lighter chose to hide their name; the garden shows the label "Vô danh" (Anonymous) instead of the name. |

### 1.5 References

- User request, 2026-10-10 (feature description and answers to questions 1–27).
- `docs/specs/features/lamp/{requirements,design,flow,tasks}.md` — the engineering spec derived from this document (v1.2).
- `docs/specs/features/memorial/requirements.md` — existing user-scoped feature with a per-user limit, as a precedent for caps.
- `docs/specs/features/login/requirements.md` — authentication that this feature depends on.
- IEEE 29148-2018, *Systems and software engineering — Life cycle processes — Requirements engineering*.

### 1.6 Document organization

Section 2 describes context and users. Section 3 lists interfaces, functional requirements (with Gherkin acceptance criteria), non-functional requirements and data needs. Section 4 gives verification and traceability. Appendix A tracks open and resolved questions.

---

## 2. Overall description

### 2.1 Product perspective

A new feature, independent of the existing catalogue features. It adds a write capability (light a lamp) and a read capability (garden listing with counts) to the app-facing service, both behind sign-in. It relies on the existing authentication and on the shared database. No third-party system is involved in v1.

### 2.2 Product functions

- Light a lamp (`FR-001`, `FR-002`).
- Limit concurrent lit lamps per user (`FR-003`).
- End a lamp's life when its duration is over (`FR-004`).
- Browse the garden of all users' lit lamps (`FR-005`).
- Filter the garden by lamp type (`FR-006`).
- Load more lamps page by page (`FR-007`).
- Show total and per-type counts (`FR-008`).
- Require sign-in for every lamp action (`FR-009`).

### 2.3 User classes and characteristics

| Actor | Description | Characteristics / permissions |
|---|---|---|
| Signed-in user | A person with an InnerLamp account who is signed in. | May light lamps and view the garden of all users' lit lamps. |
| Visitor (not signed in) | A person using the app without a session. | May do nothing in this feature: cannot light a lamp and cannot view the garden. |

### 2.4 Operating environment

- Platform: InnerLamp mobile app client, backed by the `appService` backend.
- Network: online required; there is no offline lighting or offline garden.
- Time: lamp end times are decided by the server clock, never by the device clock.

### 2.5 Design and implementation constraints

- The backend follows the project's Clean Architecture layout (domain / usecase / repository / delivery) and the existing API response envelope and JWT access-token middleware.
- The set of lamp types, intentions and burn durations is closed and fixed in v1.
- v1 must not preclude the v2 "light at a temple, with a fee" capability; no payment concept is introduced now.

### 2.6 User documentation

**N/A** — a self-explanatory consumer flow; no separate manual is required.

### 2.7 Assumptions and dependencies

- **Assumptions:**
  - A lamp's burn duration is N×24 hours from the moment of lighting (user answer 3).
  - The 20-lamp cap counts only lit lamps; expired lamps never count (user answer 9).
  - Garden totals are global (all users' lit lamps) and do not change with the filter (user answer 10).
- **Dependencies:**
  - Login (`docs/specs/features/login`): the access token identifies the lighter.
  - Account profile: the lighter's name is the dharma name (pháp danh), which is optional on the account today. When there is none, or when the lighter hides their name, the garden shows "Vô danh" (Anonymous). No new profile field is needed.

---

## 3. Specific requirements

### 3.1 External interface requirements

#### 3.1.1 User interfaces

- A light-a-lamp form with: lamp type (3 choices), intention (4 choices), recipient (text), prayer (text), burn duration (4 choices), and a "hide my name" option.
- A lamp garden screen listing lit lamps, with an "All" filter plus one filter per lamp type, a "load more" action, and a header showing the total and per-type counts.
- Visual design follows the app's design system; this document does not specify pixels.

#### 3.1.2 Hardware interfaces

**N/A** — no custom hardware interface.

#### 3.1.3 Software interfaces

Backend contract, at business level (exact paths, field names and error codes are decided in the engineering spec):

- **Light a lamp** — input: lamp type, intention, recipient, prayer, burn duration, and whether to hide the lighter's name. Output: the created lamp, including its end time.
- **List the garden** — input: optional lamp-type filter, a page position and page size (at most 12). Output: a page of lit lamps (lighter's name, lamp type, recipient, end date), an indicator of whether more exist, the total number of lit lamps, and the number of lit lamps per lamp type, all in the same response. Intention and prayer are never returned to other users.

Paging uses an opaque **cursor** returned with each page (decision 27), not page numbers, so that "load more" stays stable while lamps expire or are lit (FR-007). A page holds at most 12 lamps.

#### 3.1.4 Communication interfaces

HTTPS with JSON, authenticated with the existing access token on every request.

### 3.2 Functional requirements

#### Lighting a lamp

**FR-001 — Light a lamp**
- **Priority:** Must
- **Description:** WHEN a signed-in user submits a lamp with a lamp type, an intention, a recipient and a burn duration, and optionally the choice to hide their name, THE SYSTEM SHALL create a lit lamp owned by that user whose end time is the lighting time plus the chosen burn duration.
- **Verification method:** unit test, E2E test
- **Source:** request bullets 1–5
- **Acceptance criteria:**
  ```gherkin
  Scenario: Light a lamp for 7 days
    Given a signed-in user with no lit lamps
    When the user lights an oil lamp for peace, for "Bà ngoại", with the prayer "Mong bà bình an", for 7 days
    Then the lamp is created
    And its end time is 7 days after the moment it was lit
    And the lamp appears in the garden

  Scenario Outline: Every burn duration is accepted
    Given a signed-in user with no lit lamps
    When the user lights a lotus candle for gratitude, for "Thầy", for <days> days
    Then the lamp is created with an end time <days> days after lighting

    Examples:
      | days |
      | 1    |
      | 7    |
      | 21   |
      | 49   |

  Scenario Outline: Every lamp type and intention is accepted
    Given a signed-in user with no lit lamps
    When the user lights a "<lamp_type>" lamp with the intention "<intention>" for 1 day
    Then the lamp is created

    Examples:
      | lamp_type             | intention                      |
      | oil lamp              | peace                          |
      | lotus candle          | salvation of the departed      |
      | floating flower lantern | merit dedication             |
      | oil lamp              | gratitude                      |

  Scenario: Light a lamp anonymously
    Given a signed-in user on the light-a-lamp form
    When the user chooses to hide their name and lights a lamp
    Then the lamp is created
    And it is marked as anonymous

  Scenario: Lighting the same lamp twice by repeated tap
    Given a signed-in user has just submitted a lamp
    When the user submits the identical lamp again immediately
    Then only one lamp exists
    And the user sees the lamp that was lit

  Scenario: Deliberately lighting the same lamp again later
    Given a signed-in user lit a lamp 6 seconds ago
    When the user submits an identical lamp
    Then a second lamp is created
  ```
  Two submissions count as the same lamp when they carry identical content (lamp type, intention, recipient, prayer, burn duration, hidden-name choice) from the same user within 5 seconds of each other.

**FR-002 — Validate the lamp input**
- **Priority:** Must
- **Description:** WHEN a user submits a lamp THE SYSTEM SHALL require a lamp type, an intention and a burn duration from the fixed sets, a recipient of 1 to 100 characters, and an optional prayer of at most 300 characters, and SHALL reject the lamp without creating it otherwise. Length is counted in characters, not bytes, after removing leading and trailing spaces.
- **Verification method:** unit test, E2E test
- **Source:** answers 4 and 5
- **Acceptance criteria:**
  ```gherkin
  Scenario: Recipient is empty
    Given a signed-in user on the light-a-lamp form
    When the user submits with an empty recipient
    Then no lamp is created
    And the user is told the recipient is required

  Scenario Outline: Recipient length boundaries
    Given a signed-in user on the light-a-lamp form
    When the user submits a recipient of <length> characters
    Then the lamp is <outcome>

    Examples:
      | length | outcome  |
      | 1      | created  |
      | 100    | created  |
      | 101    | rejected |

  Scenario Outline: Prayer length boundaries
    Given a signed-in user on the light-a-lamp form
    When the user submits a prayer of <length> characters
    Then the lamp is <outcome>

    Examples:
      | length | outcome  |
      | 300    | created  |
      | 301    | rejected |

  Scenario: Burn duration outside the allowed set
    Given a signed-in user on the light-a-lamp form
    When the user submits a burn duration of 5 days
    Then no lamp is created
    And the user is told the duration is not allowed

  Scenario: Recipient of only spaces
    Given a signed-in user on the light-a-lamp form
    When the user submits a recipient made only of spaces
    Then no lamp is created
    And the user is told the recipient is required

  Scenario: Spaces around the text are removed
    Given a signed-in user on the light-a-lamp form
    When the user submits the recipient "  Bà ngoại  "
    Then the lamp is created for "Bà ngoại"

  Scenario: Prayer is empty
    Given a signed-in user on the light-a-lamp form
    When the user submits with an empty prayer
    Then the lamp is created without a prayer

  Scenario: Vietnamese text is counted in characters
    Given a signed-in user on the light-a-lamp form
    When the user submits a recipient of 100 Vietnamese characters with diacritics
    Then the lamp is created
  ```

**FR-003 — Cap concurrent lit lamps per user**
- **Priority:** Must
- **Description:** WHEN a user tries to light a lamp while already having 20 lit lamps THE SYSTEM SHALL refuse the lamp; expired lamps SHALL NOT count toward the 20.
- **Verification method:** unit test, E2E test
- **Source:** answers 8 and 9
- **Acceptance criteria:**
  ```gherkin
  Scenario: Reach the cap
    Given a signed-in user with 19 lit lamps
    When the user lights one more lamp
    Then the lamp is created
    And the user now has 20 lit lamps

  Scenario: Exceed the cap
    Given a signed-in user with 20 lit lamps
    When the user lights another lamp
    Then no lamp is created
    And the user is told the limit of 20 lit lamps is reached

  Scenario: Expired lamps free capacity
    Given a signed-in user with 20 lamps of which 3 have expired
    When the user lights another lamp
    Then the lamp is created

  Scenario: The cap is per user
    Given user A has 20 lit lamps
    And user B has 0 lit lamps
    When user B lights a lamp
    Then the lamp is created

  Scenario: Two simultaneous lighting attempts at 19 lamps
    Given a signed-in user with 19 lit lamps
    When the user's two devices submit a lamp at the same moment
    Then at most one of the two lamps is created
    And the user never has more than 20 lit lamps
  ```

#### Lamp lifetime

**FR-004 — A lamp goes out when its duration is over**
- **Priority:** Must
- **Description:** WHEN a lamp's end time has passed THE SYSTEM SHALL treat it as expired: it SHALL no longer appear in the garden and SHALL NOT be counted in any total. Expired lamps SHALL NOT be extended or relit; the user lights a new lamp instead.
- **Verification method:** unit test, E2E test
- **Source:** answer 9
- **Acceptance criteria:**
  ```gherkin
  Scenario: Lamp disappears at its end time
    Given a lamp lit for 1 day
    When one day has passed since it was lit
    Then the lamp no longer appears in the garden
    And it is not counted in the total

  Scenario: End date follows the viewer's time zone
    Given a lamp whose end time is 2026-10-17 18:00 UTC
    When a viewer in Vietnam (UTC+7) opens the garden
    Then the lamp's end date is shown as 18 October 2026, 01:00
    And a viewer in UTC-5 sees 17 October 2026, 13:00

  Scenario: Lamp is still visible just before its end time
    Given a lamp lit for 1 day
    When one minute remains before its end time
    Then the lamp appears in the garden

  Scenario: An expired lamp cannot be extended
    Given a lamp that has expired
    When its owner tries to extend or relight it
    Then nothing is changed
    And the owner is directed to light a new lamp

  Scenario: A lit lamp cannot be put out early
    Given a lamp that is still lit
    When its owner tries to put it out or delete it
    Then nothing is changed
    And the lamp stays in the garden until its end time
  ```
  An end time is exactly N×24 hours after the moment of lighting (a 1-day lamp lit at 21:00 goes out at 21:00 the next day).

  Time is stored as absolute instants (UTC): the lighting time and the end time. They never change, whatever the user's location. The app converts them to the viewer's own time zone when showing the end date.

  The end date shown in the garden is displayed in the viewer's own device time zone.

#### Lamp garden

**FR-005 — Browse the garden of all users' lit lamps**
- **Priority:** Must
- **Description:** WHEN a signed-in user opens the lamp garden THE SYSTEM SHALL list the lit lamps of all users, newest lit first, each showing the lighter's name, the lamp type, the recipient and the end date.
- **Verification method:** unit test, E2E test
- **Source:** request bullet 6; answers 1, 3 and 11
- **Acceptance criteria:**
  ```gherkin
  Scenario: Garden shows everyone's lit lamps, newest first
    Given user A lit an oil lamp yesterday
    And user B lit a lotus candle today
    When a signed-in user opens the garden
    Then both lamps are listed
    And user B's lamp is listed before user A's lamp
    And each lamp shows the lighter's name, the lamp type, the recipient and the end date

  Scenario: Expired lamps are not listed
    Given a lamp whose end time has passed
    When a signed-in user opens the garden
    Then that lamp is not listed

  Scenario: Empty garden
    Given no lamps are lit by anyone
    When a signed-in user opens the garden
    Then the garden is empty
    And the total is 0

  Scenario: Lighter's name is the dharma name
    Given a user with a dharma name lit a lamp
    When another signed-in user opens the garden
    Then the lamp shows the lighter's dharma name

  Scenario: Intention and prayer are private
    Given a user lit a lamp with the intention "gratitude" and a prayer
    When another signed-in user opens the garden
    Then the lamp does not show the intention
    And the lamp does not show the prayer

  Scenario: Lighter without a dharma name
    Given a user with no dharma name lit a lamp
    When another signed-in user opens the garden
    Then the lamp shows the lighter's name as "Vô danh" (Anonymous)

  Scenario: Lighter hides their name
    Given a user with a dharma name lit an anonymous lamp
    When another signed-in user opens the garden
    Then the lamp shows the lighter's name as "Vô danh" (Anonymous)
    And the dharma name is not shown anywhere in the garden

  Scenario: Later change of dharma name does not alter the lamp
    Given a user lit a lamp while their dharma name was "Thích Minh An"
    When the user later changes their dharma name to "Thích Tuệ Minh"
    Then that lamp still shows "Thích Minh An" in the garden
    And lamps lit after the change show "Thích Tuệ Minh"
  ```

**FR-006 — Filter the garden by lamp type**
- **Priority:** Must
- **Description:** WHEN a user selects a lamp type in the garden THE SYSTEM SHALL list only lit lamps of that type; WHEN the user selects "All" THE SYSTEM SHALL list lit lamps of every type.
- **Verification method:** unit test, E2E test
- **Source:** request bullet 6
- **Acceptance criteria:**
  ```gherkin
  Scenario: Filter by one lamp type
    Given lit lamps of all three types exist
    When a signed-in user filters the garden by lotus candle
    Then only lotus candles are listed

  Scenario: Select All
    Given the garden is filtered by oil lamp
    When the user selects "All"
    Then lamps of every type are listed

  Scenario: Filter with no matching lamps
    Given no lotus candle is lit
    When a signed-in user filters the garden by lotus candle
    Then the list is empty
    And the totals still show the counts of all lit lamps

  Scenario: Unknown lamp type in the filter
    Given a signed-in user
    When the garden is requested with a lamp type outside the three allowed
    Then the request is rejected as invalid
  ```

**FR-007 — Load more**
- **Priority:** Must
- **Description:** WHEN the garden has more lit lamps than one page THE SYSTEM SHALL return the lamps in pages of at most 12 and let the user load the next page by sending back a position marker (cursor) issued with the previous page, without repeating or skipping a lamp that stays lit.
- **Verification method:** unit test, E2E test
- **Source:** request bullet 6; answer 18; decision 27
- **Acceptance criteria:**
  ```gherkin
  Scenario: Load the next page
    Given 30 lit lamps exist
    When a signed-in user opens the garden and then loads more
    Then the first page shows 12 lamps
    And the next 12 lamps are appended
    And no lamp appears twice

  Scenario: A page never exceeds 12 lamps
    Given 30 lit lamps exist
    When a signed-in user asks for a page of 50 lamps
    Then at most 12 lamps are returned

  Scenario: Last page
    Given the user has loaded every lit lamp
    When the user reaches the end of the list
    Then the user is told there are no more lamps

  Scenario: A lamp expires while the user is paging
    Given the user has loaded the first page
    And a lamp on a later page expires
    When the user loads more
    Then the expired lamp is not shown
    And no other lamp is skipped or repeated

  Scenario: A new lamp is lit while the user is paging
    Given the user has loaded the first page
    And another user lights a new lamp
    When the user loads more
    Then no lamp is repeated
  ```

**FR-008 — Show total and per-type counts**
- **Priority:** Must
- **Description:** WHEN the garden is returned THE SYSTEM SHALL include, in the same response, the total number of lit lamps and the number of lit lamps per lamp type, counted over all users and independent of the selected filter.
- **Verification method:** unit test, E2E test
- **Source:** request bullet 6; answer 10
- **Acceptance criteria:**
  ```gherkin
  Scenario: Counts for all lamp types
    Given 5 lit oil lamps, 3 lit lotus candles and 2 lit floating flower lanterns
    When a signed-in user opens the garden
    Then the total is 10
    And the counts are 5 oil lamps, 3 lotus candles and 2 floating flower lanterns

  Scenario: Counts ignore the filter
    Given 5 lit oil lamps and 3 lit lotus candles
    When the user filters the garden by lotus candle
    Then the listed lamps are the 3 lotus candles
    And the total is still 8
    And the counts are still 5 oil lamps and 3 lotus candles

  Scenario: Expired lamps are not counted
    Given 2 lit lamps and 4 expired lamps
    When a signed-in user opens the garden
    Then the total is 2

  Scenario: A lamp type with no lit lamps
    Given no floating flower lantern is lit
    When a signed-in user opens the garden
    Then the count for floating flower lanterns is 0
  ```

#### Access

**FR-009 — Sign-in required**
- **Priority:** Must
- **Description:** WHEN a request to light a lamp or to view the garden carries no valid sign-in THE SYSTEM SHALL refuse it and SHALL return no lamp data.
- **Verification method:** E2E test
- **Source:** request bullet 1; answer 2
- **Acceptance criteria:**
  ```gherkin
  Scenario: Visitor cannot view the garden
    Given a visitor who is not signed in
    When the visitor requests the garden
    Then the request is refused as unauthenticated
    And no lamp or count is returned

  Scenario: Visitor cannot light a lamp
    Given a visitor who is not signed in
    When the visitor submits a lamp
    Then the request is refused as unauthenticated
    And no lamp is created

  Scenario: Expired session
    Given a user whose sign-in has expired
    When the user requests the garden
    Then the request is refused as unauthenticated
  ```

### 3.3 Performance requirements

- **NFR-001 — Garden response time:** the garden request, including its counts, returns within a target to be decided.
  > ⚠️ Not confirmed: the response-time target and the expected number of simultaneously lit lamps (the product owner does not know yet; to be measured after launch) — it blocks index and count-strategy decisions, so the design should keep the counting approach easy to change.
- **NFR-002 — Counts consistent with the list:** the totals returned with a page SHALL be computed from the same moment as that page's list.

### 3.4 Logical database requirements

- The server persists every lamp: owner, lamp type, intention, recipient, prayer, whether the name is hidden, the lighter's displayed name as it was when the lamp was lit (a copy, not a live read of the profile), lit time, end time. Lit time and end time are stored as absolute instants (UTC), never as local dates.
- Expired lamps are kept for now (decision: "store first, decide retention later"), so v2 history features remain possible.
  > ⚠️ Not confirmed: how long expired lamps are kept and what happens to a user's lamps when the account is deleted — deferred by decision; it blocks the data-retention rule and account-deletion behaviour, and must be settled before go-live of any deletion feature.
- Nothing is stored on the device beyond what the app already stores.

### 3.5 Design constraints

- Follow the existing API envelope, error taxonomy and JWT middleware conventions of `appService`.
- The fixed sets (lamp types, intentions, durations) have a single source of truth on the server; the client does not invent values.
- The 20-lamp cap is configurable, not hard-coded across the code base.

### 3.6 Software system attributes

- **Reliability:** a lamp, once confirmed to the user, is not lost; an end time is never changed after lighting.
- **Availability:** same as the rest of `appService`; no separate target.
- **Security:**
  - Every lamp operation requires a valid access token; the lighter is taken from the token, never from the request body.
  - The garden is visible to every signed-in user, so the recipient is effectively public text; users should be told that what they type there is visible to others. The intention and prayer are private to the lighter. A hidden name is not returned to other users at all (not merely masked on the client).
  - Recipient and prayer may contain personal names (for example a deceased relative); they must not be written to logs.
  - There is no moderation in v1 (decision), so abusive text can appear in the garden. Accepted risk: see Appendix A.
- **Maintainability:** adding a new lamp type or duration later is a data/configuration change on the server plus client artwork.
- **Portability:** N/A.

---

## 4. Verification & traceability

### 4.1 Verification approach

Each requirement carries a verification method. Unit tests cover validation, the cap, expiry and counting rules. Black-box E2E tests (real binary, real Postgres) cover authentication wiring, the filter, paging and the counts, since those depend on route and middleware wiring.

### 4.2 Traceability matrix

| Requirement ID | Source | Design | Test |
|---|---|---|---|
| FR-001 | request bullets 1–5; answers 12, 23, 24 | → il-spec-writer | 6 scenarios |
| FR-002 | answers 4, 5, 13, 21 | → il-spec-writer | 8 scenarios |
| FR-003 | answers 8, 9 | → il-spec-writer | 5 scenarios |
| FR-004 | answers 9, 14, 15, 25 | → il-spec-writer | 5 scenarios |
| FR-005 | request bullet 6; answers 1, 3, 11, 16, 17, 22, 23, 26 | → il-spec-writer | 8 scenarios |
| FR-006 | request bullet 6 | → il-spec-writer | 4 scenarios |
| FR-007 | request bullet 6; answers 18, 27 | → il-spec-writer | 5 scenarios |
| FR-008 | request bullet 6; answer 10 | → il-spec-writer | 4 scenarios |
| FR-009 | request bullet 1; answer 2 | → il-spec-writer | 3 scenarios |
| NFR-001 | — | → il-spec-writer | TBD (⚠️ target not confirmed) |
| NFR-002 | answer 10 | → il-spec-writer | covered by FR-008 |

---

## 5. Appendices

### Appendix A — Open questions

#### Open (unconfirmed)

> ⚠️ Not confirmed: the garden response-time target and the expected volume of lit lamps (to be measured after launch) — it blocks index and count strategy (NFR-001).

> ⚠️ Not confirmed: how long expired lamps are kept and what happens to a user's lamps on account deletion (deferred by decision) — it blocks the retention rule (3.4).

#### Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| 1. Who sees which lamps in the garden? | All users' lit lamps are shown. | 2026-10-10 |
| 2. Can visitors view the garden? | No. Only signed-in users can view it. | 2026-10-10 |
| 3. What is displayed publicly for a lamp? | Lighter's name, lamp type, recipient, end date. | 2026-10-10 |
| 4. What is the recipient? | Free text, 1–100 characters. | 2026-10-10 |
| 5. Prayer length? | Maximum 300 characters. | 2026-10-10 |
| 6. Moderation in v1? | None. | 2026-10-10 |
| 7. Cost? | Free in v1. v2 will add lighting at a temple with a fee; out of scope here. | 2026-10-10 |
| 8. Lamp cap? | 20 lit lamps at the same time per user. | 2026-10-10 |
| 9. Expiry behaviour? | An expired lamp disappears; no extension; the user lights a new one. Expired lamps do not count toward the 20. | 2026-10-10 |
| 10. Count semantics? | Total of all lit lamps plus the count per lamp type, in the same response, independent of the filter. | 2026-10-10 |
| 11. Ordering? | Newest lit first. | 2026-10-10 |
| 12. Double tap on the same lamp? | Creates one lamp (window length still open). | 2026-10-10 |
| 13. Is the prayer required, and how is length counted? | Optional; counted in characters. | 2026-10-10 |
| 14. Meaning of "N days"? | N×24 hours from the moment of lighting. | 2026-10-10 |
| 15. Can the owner put out a lamp early? | No. | 2026-10-10 |
| 16. Which name is shown for the lighter? | Dharma name; fall back to the ordinary name when there is none (fallback field still open). | 2026-10-10 |
| 17. Are intention and prayer visible to others? | No. Only the lighter's own data; others do not see them. | 2026-10-10 |
| 18. Page size? | At most 12 per page. | 2026-10-10 |
| 19. Performance target and volume? | Unknown; to be measured (kept open). | 2026-10-10 |
| 20. Retention of expired lamps? | Keep them for now; decide retention later (kept open). | 2026-10-10 |
| 21. Are leading and trailing spaces trimmed? | Yes, before validation and storage. | 2026-10-10 |
| 22. Name shown when there is no dharma name? | "Vô danh" (Anonymous); no ordinary-name field is needed. | 2026-10-10 |
| 23. May the lighter hide their name? | Yes; the garden then shows "Vô danh". | 2026-10-10 |
| 24. De-duplication window for a double tap? | 5 seconds, for identical content from the same user. | 2026-10-10 |
| 25. Where is time stored and computed, and which zone shows the end date? | Stored and computed on the server as absolute instants (UTC); the end date is shown in the viewer's device time zone. | 2026-10-10 |
| 26. Does the lighter's name follow later dharma-name changes? | No. The name is frozen at the moment the lamp is lit. | 2026-10-10 |
| 27. Page by page number or by cursor? | By cursor. A page number shifts when lamps expire or are lit between two loads, which would repeat or skip lamps (FR-007); the cursor continues after the last lamp shown. Clients support one more paging style than the other lists. | 2026-10-10 |
