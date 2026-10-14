# Software Requirements Specification — Temple Announcement

| Field | Value |
|---|---|
| Document type | Software Requirements Specification (SRS) — IEEE 29148-2018 |
| Feature | Temple Announcement (thông báo của chùa) |
| Product | InnerLamp |
| Version | 0.3 |
| Last updated | 2026-10-14 |
| Status | confirmed 2026-10-14 — open questions resolved by default; engineering spec written (v1.0, not yet implemented) |
| Owner / author | duc@wee.vn (BA assist: Claude) |
| Downstream handoff | `il-spec-writer` → `requirements.md` / `design.md` / `flow.md` / `tasks.md` under `docs/specs/features/temple-announcement/` (written at v1.0) |

---

## 1. Introduction

### 1.1 Purpose

This document specifies, in business terms, how a signed-in InnerLamp user sees the announcements published by the temple they are ordained at (for example "summer retreat registration closes in 10 days"), how they switch the announcements on or off, and how an administrator publishes them. It drives the engineering spec that `il-spec-writer` produces next.

### 1.2 Scope

**In scope:**
- A temple announcement: a record owned by one temple, with a title, a body, an issuing body inside the temple (for example Ban Trị sự, Ban Hoằng pháp), a publish time and an expiry time.
- A signed-in user with an ordination temple sees the unexpired announcements of that temple, on the website and in the mobile app, identically on every device.
- One global on/off switch per user for receiving announcements.
- Behaviour when the user changes their temple or leaves their temple.
- An administrator publishes, edits and revokes announcements (v1 source).
- A source field on every announcement so that an automatic system source can be added later.

**Out of scope:**
- OS-level push notifications (FCM, APNs, web push), SMS and email. An announcement is data stored in the temple's own records, not a message pushed to a device.
- Read / unread tracking and unread badges (decision 2026-10-14: the user always sees every active announcement).
- Per-type or per-temple switches.
- Scheduled publishing (publish later).
- Automatic generation of announcements by the system (only the data field is reserved).
- Temple events and the practice schedule; they stay in `temple-event` and `temple-schedule`.

### 1.3 Product overview

InnerLamp is a Buddhist-practice companion app. A user can record the temple where they were ordained (`ordination_temple_id`) in their profile and browse temples, events and schedules. This feature adds a channel for the temple to talk to its own followers: short announcements with a deadline.

### 1.4 Definitions, acronyms, abbreviations

| Term | Definition |
|---|---|
| Announcement (thông báo) | A temple-owned notice with a title, body, issuing body, publish time and expiry time. Distinct from a temple event and from an OS push notification. |
| Ordination temple (chùa quy y) | The single temple linked to the user profile as `ordination_temple_id`. In this document "the user's temple". |
| Issuing body (đơn vị phát hành) | The unit inside the temple that issues the announcement, for example Ban Trị sự, Ban Hoằng pháp. |
| Active announcement | An announcement that is published, not revoked, and whose expiry time has not passed. |
| Receive switch | A per-user setting: receive announcements (default) or do not receive. |
| Administrator | A backoffice admin who publishes announcements. |
| Source | Where an announcement came from: `admin` (v1) or `system` (reserved for later). |

### 1.5 References

- `docs/specs/features/profile/requirements.md` — ordination temple field and its change rules.
- `docs/specs/features/temple/requirements.md` — temple catalogue.
- `docs/specs/features/temple-event/SRS.md` — temple events (a separate concept).
- `docs/specs/features/admin/` — backoffice admin.

### 1.6 Document organization

Section 2 describes context and actors. Section 3 lists requirements with Gherkin acceptance criteria. Section 4 gives traceability. Appendix A lists open and resolved questions.

---

## 2. Overall description

### 2.1 Product perspective

New feature. It depends on the profile's ordination temple (read-only) and on the temple catalogue. It adds backoffice publishing. It does not touch push infrastructure.

### 2.2 Product functions

- View active announcements of the user's temple (`FR-001`, `FR-002`, `FR-003`).
- Turn receiving on or off (`FR-004`).
- Announcements disappear when they expire (`FR-005`).
- Administrator publishes, edits and revokes announcements (`FR-006`, `FR-007`).
- Show the issuing body on each announcement (`FR-008`).

### 2.3 User classes and characteristics

| Actor | Description | Characteristics / permissions |
|---|---|---|
| Follower | Signed-in user with an ordination temple | Reads the announcements of their own temple only; controls the receive switch. |
| User without temple | Signed-in user with no ordination temple | Sees no announcements, no error. |
| Administrator | Backoffice admin | Creates, edits, revokes announcements for any temple (v1). |
| System | Future automatic scanner | Reserved; creates announcements with source `system`. |

### 2.4 Operating environment

- Platform: website and mobile app; both use the same backend.
- Network: online required.

### 2.5 Design and implementation constraints

- Go backend, Clean Architecture layout under `services/` (project convention).
- Visibility is decided at read time from the user's current ordination temple, not at publish time.
- Non-sensitive limits (title and body length, max validity) live in service config, not in code constants.

### 2.6 User documentation

N/A — the feature is self-explanatory in the UI; no separate manual.

### 2.7 Assumptions and dependencies

- **Assumptions:**
  - "The temple where I practise" means the existing `ordination_temple_id` (confirmed by the user).
  - "Leaving the temple" means `ordination_temple_id` is cleared (confirmed).
  - Announcements have no media or files in v1 (adopted default).
- **Dependencies:** `profile` (ordination temple), `temple` (catalogue), `admin` (backoffice auth).

---

## 3. Specific requirements

### 3.1 External interface requirements

#### 3.1.1 User interfaces
A list of the temple's active announcements (title, issuing body, publish time, expiry time, body) and a single receive switch. Screens are defined later in the design.

#### 3.1.2 Hardware interfaces
**N/A** — no hardware interface.

#### 3.1.3 Software interfaces
Contract not yet defined; it will be set by `il-spec-writer` (user read list, receive switch get/set, admin create/edit/revoke).

#### 3.1.4 Communication interfaces
HTTPS, JSON, bearer access token for users, admin authentication for backoffice.

### 3.2 Functional requirements

#### Follower

**FR-001 — View active announcements of my temple**
- **Priority:** Must
- **Description:** WHEN a signed-in user with an ordination temple and the receive switch on opens the announcements, THE SYSTEM SHALL return the active announcements of that temple, newest first, in pages of at most 20.
- **Verification method:** unit test, API test, E2E
- **Source:** user request 2026-10-14
- **Acceptance criteria:**
  ```gherkin
  Scenario: Follower sees announcements of their temple
    Given a user whose temple is "A" with receiving on
    And temple "A" has 2 active announcements and temple "B" has 1
    When the user opens the announcements
    Then the user sees the 2 announcements of "A", newest first
    And does not see the announcement of "B"

  Scenario: Same list on every device
    Given a user signed in on the website and on the app
    When the user opens the announcements on both
    Then both show the same announcements

  Scenario: Paging
    Given the user's temple has 25 active announcements
    When the user opens the announcements
    Then the first page shows 20 and a next page shows 5
  ```

**FR-002 — User without a temple**
- **Priority:** Must
- **Description:** WHEN a signed-in user without an ordination temple opens the announcements, THE SYSTEM SHALL return an empty list and SHALL NOT return an error.
- **Verification method:** unit test, API test
- **Source:** user request ("rời khỏi chùa")
- **Acceptance criteria:**
  ```gherkin
  Scenario: No temple
    Given a user with no ordination temple
    When the user opens the announcements
    Then the list is empty
    And no error is shown

  Scenario: Leaving the temple
    Given a user who sees announcements of temple "A"
    When the user clears their ordination temple
    Then the user no longer sees any announcement of "A"
  ```

**FR-003 — Changing temple changes what I see**
- **Priority:** Must
- **Description:** WHEN the user changes their ordination temple from A to B, THE SYSTEM SHALL show only the active announcements of B from the next read, and SHALL NOT show those of A, because an announcement belongs to the temple and not to the user.
- **Verification method:** unit test, API test, E2E
- **Source:** user request
- **Acceptance criteria:**
  ```gherkin
  Scenario: Switch temple
    Given a user in temple "A" who sees A's announcements
    When the user changes their temple to "B"
    Then the user sees the active announcements of "B"
    And sees none of "A"

  Scenario: Return to the previous temple
    Given the user moved from "A" to "B"
    When the user changes back to "A"
    Then the user sees the active announcements of "A" again
  ```

**FR-004 — Receive switch**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL keep one receive switch per user, on by default. WHEN the switch is off, THE SYSTEM SHALL return no announcements. The switch value SHALL persist across devices, temple changes and leaving a temple.
- **Verification method:** unit test, API test, E2E
- **Source:** user request ("một công tắc chung")
- **Acceptance criteria:**
  ```gherkin
  Scenario: Turn receiving off
    Given a user in temple "A" with receiving on
    When the user turns receiving off
    Then the user sees no announcements

  Scenario: Turn receiving back on
    Given a user with receiving off
    When the user turns receiving on
    Then the user sees the active announcements of their temple

  Scenario: Setting survives a temple change
    Given a user with receiving off in temple "A"
    When the user changes to temple "B"
    Then receiving is still off

  Scenario: Setting is the same on all devices
    Given a user turns receiving off on the app
    When the user opens the website
    Then receiving is off there too
  ```
> Resolved 2026-10-14: "off" hides the list entirely.

**FR-005 — Expiry**
- **Priority:** Must
- **Description:** WHEN an announcement's expiry time has passed, THE SYSTEM SHALL stop returning it to users. The administrator SHALL still be able to see it.
- **Verification method:** unit test, API test
- **Source:** user request ("có hạn")
- **Acceptance criteria:**
  ```gherkin
  Scenario: Announcement expires
    Given an announcement that expires today at 18:00
    When a follower opens the announcements after 18:00
    Then the announcement is not shown

  Scenario: Admin still sees it
    Given an expired announcement
    When an administrator lists the announcements of that temple
    Then the expired announcement is listed
  ```

#### Administrator

**FR-006 — Publish an announcement**
- **Priority:** Must
- **Description:** WHEN an administrator submits a temple, an issuing body, a title, a body and an expiry time in the future, THE SYSTEM SHALL create the announcement with source `admin`, visible immediately to the followers of that temple.
- **Verification method:** unit test, API test, E2E
- **Source:** user request ("admin push thông báo")
- **Acceptance criteria:**
  ```gherkin
  Scenario: Publish
    Given an administrator and temple "A"
    When the administrator publishes an announcement with title, body, issuing body and a future expiry
    Then followers of "A" see it immediately

  Scenario: Expiry in the past
    When the administrator submits an expiry time in the past
    Then the request is rejected with a validation error

  Scenario: Missing required field
    When the administrator omits the title, body, issuing body or expiry
    Then the request is rejected with a validation error

  Scenario: Unknown temple
    When the administrator targets a temple that does not exist
    Then the request is rejected
  ```
> Resolved 2026-10-14: expiry is mandatory, at most 365 days ahead (configurable).
> Resolved 2026-10-14: v1 uses system-wide backoffice accounts; roles `admin` and `content_editor` may publish, `viewer` may not.
> Resolved 2026-10-14: no scheduled publishing; publish means visible now.

**FR-007 — Edit and revoke**
- **Priority:** Should
- **Description:** WHEN an administrator edits an announcement, THE SYSTEM SHALL show the new content to followers from the next read. WHEN an administrator revokes it, THE SYSTEM SHALL stop returning it to users while keeping it visible to administrators.
- **Verification method:** unit test, API test
- **Source:** adopted default
- **Acceptance criteria:**
  ```gherkin
  Scenario: Edit
    Given an active announcement
    When the administrator changes its body
    Then followers see the new body

  Scenario: Revoke
    Given an active announcement
    When the administrator revokes it
    Then followers no longer see it
    And administrators still see it as revoked
  ```

#### Display

**FR-008 — Issuing body on each announcement**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL store and return the issuing body of each announcement, so the user sees where it comes from (for example Ban Trị sự, Ban Hoằng pháp).
- **Verification method:** unit test, API test
- **Source:** user request 2026-10-14
- **Acceptance criteria:**
  ```gherkin
  Scenario: Issuing body shown
    Given an announcement issued by "Ban Hoằng pháp"
    When a follower opens the announcements
    Then the announcement shows "Ban Hoằng pháp" as its origin

  Scenario: Announcement without an issuing body is refused
    When the administrator publishes without an issuing body
    Then the request is rejected with a validation error
  ```
> Resolved 2026-10-14: the issuing body is chosen from a managed list per temple.

**FR-009 — Source field**
- **Priority:** Could
- **Description:** THE SYSTEM SHALL record the source of each announcement (`admin` or `system`), so that automatic generation can be added later without changing existing data.
- **Verification method:** code review
- **Source:** user request ("sau này có hệ thống auto quét")
- **Acceptance criteria:**
  ```gherkin
  Scenario: Source recorded
    When an administrator publishes an announcement
    Then its source is "admin"
  ```

### 3.3 Performance requirements

- **NFR-001 — List latency:** the announcement list responds in under 500 ms at the 95th percentile for a temple with up to 200 active announcements.
- **NFR-002 — Visibility delay:** a published, edited or revoked announcement is reflected for followers on their next read, within 5 seconds.
- **NFR-003 — Request limit:** the announcement routes have a per-account request limit of their own, separate from the profile routes; an account over the limit is refused until the window passes.

> Resolved 2026-10-14: volume is unknown; no fan-out is needed because announcements are read on demand, not pushed.

### 3.4 Logical database requirements

- Announcements are persisted server-side and belong to a temple.
- Each announcement stores: temple, issuing body, title, body, publish time, expiry time, source, revoked flag, creator, edit times.
- The receive switch is stored per user.
- Limits adopted by default: title up to 150 characters, body up to 2000 characters of plain text, no media. These live in config.
- Retention: expired announcements are kept for administrators; no automatic deletion in v1.

### 3.5 Design constraints

- Follow the Clean Architecture layering and the response envelope used by other features.
- Authorization: a user can only read announcements of their own temple, derived from their token's user id, never from a request parameter.

### 3.6 Software system attributes

- **Reliability:** a failed read returns an error; it never returns another temple's announcements.
- **Availability:** same as the rest of the API.
- **Security:** announcements are not public; they require a signed-in user. Admin write operations require admin authentication. No sensitive personal data in an announcement.
- **Maintainability:** announcement and receive switch are separate from temple events.
- **Portability:** the same contract serves website and app.

---

## 4. Verification & traceability

### 4.1 Verification approach

Each requirement lists its method. Unit tests for use cases, repository tests on a real database, handler tests, and E2E for the wiring of user and admin routes.

### 4.2 Traceability matrix

| Requirement ID | Source | Design | Test |
|---|---|---|---|
| FR-001 | user request 2026-10-14 | → il-spec-writer | 3 Gherkin scenarios |
| FR-002 | user request | → il-spec-writer | 2 Gherkin scenarios |
| FR-003 | user request | → il-spec-writer | 2 Gherkin scenarios |
| FR-004 | user request | → il-spec-writer | 4 Gherkin scenarios |
| FR-005 | user request | → il-spec-writer | 2 Gherkin scenarios |
| FR-006 | user request | → il-spec-writer | 4 Gherkin scenarios |
| FR-007 | adopted default | → il-spec-writer | 2 Gherkin scenarios |
| FR-008 | user request 2026-10-14 | → il-spec-writer | 2 Gherkin scenarios |
| FR-009 | user request | → il-spec-writer | 1 Gherkin scenario |
| NFR-001 | adopted default | → il-spec-writer | TBD |
| NFR-002 | adopted default | → il-spec-writer | TBD |
| NFR-003 | user decision 2026-10-14 | → il-spec-writer | TBD |

---

## 5. Appendices

### Appendix A — Open questions

#### Open (unconfirmed)

None at the business level. None at the technical level either (engineering spec v1.1).

#### Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| "My temple" = `ordination_temple_id`? | Yes. | 2026-10-14 |
| "Leaving the temple" = clearing `ordination_temple_id`? | Yes. | 2026-10-14 |
| Announcement vs temple event vs OS push | Announcement is temple-owned data with an expiry; not an event and not OS push. | 2026-10-14 |
| Source of announcements | Admin in v1; automatic system source later. | 2026-10-14 |
| One switch or per type | One global switch per user. | 2026-10-14 |
| Same list on all devices | Yes. | 2026-10-14 |
| On temple change, do old announcements stay | No; they belong to the temple, not the user. | 2026-10-14 |
| Read / unread tracking | Not needed; the user always sees all active announcements. | 2026-10-14 |
| Show where the announcement comes from | Yes; an issuing body (Ban Trị sự, Ban Hoằng pháp, ...) is shown. | 2026-10-14 |
| Receive switch "off" hides the list entirely or only hints | Hides the list entirely. | 2026-10-14 |
| Expiry mandatory? | Mandatory, at most 365 days. | 2026-10-14 |
| System-wide admin or admin per temple | System-wide backoffice accounts (v1). | 2026-10-14 |
| Scheduled publishing | No. | 2026-10-14 |
| Issuing body: managed list or free text | Managed list per temple. | 2026-10-14 |
| Followers per temple and volume | Unknown; no fan-out needed. | 2026-10-14 |
| Role `content_editor` may publish like `admin`? | Yes. | 2026-10-14 |
| Pagination shape and list response-time target | `page`/`page_size` + total; p95 under 500 ms at 200 active announcements. | 2026-10-14 |
| Request limit for the announcement routes | Separate per-account limit, independent of the profile routes. | 2026-10-14 |
