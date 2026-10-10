# Software Requirements Specification — Temple Suggestion

| Field | Value |
|---|---|
| Document type | Software Requirements Specification (SRS) — IEEE 29148-2018 |
| Feature | Temple Suggestion |
| Product | InnerLamp |
| Version | 1.0 |
| Last updated | 2026-10-10 |
| Status | approved 2026-10-10 — no open questions; ready for `il-spec-writer` |
| Owner / author | duc@wee.vn (BA assist: Claude) |
| Downstream handoff | `il-spec-writer` → `requirements.md` / `design.md` / `tasks.md` (already exist at v1.0 — see 2.7 for the deltas this SRS introduces) |

---

## 1. Introduction

### 1.1 Purpose

This document specifies, in business terms, how people using the InnerLamp app can suggest a temple that is missing from the catalogue, and how the back-office team turns such a suggestion into an official temple. It is written for product, back-office operators, and engineering, and it drives the engineering spec under `docs/specs/features/temple-suggestion/`.

### 1.2 Scope

**In scope:**
- Anonymous submission of a temple suggestion (temple name, province/city, address) from the app.
- A back-office review queue with three outcomes: waiting, converted into an official temple, rejected.
- Back-office completion of the missing temple information (including image paths) and conversion of a suggestion into an official temple in one atomic action.
- Rejection of junk or unwanted suggestions with a recorded reason.
- Abuse protection on the public submission, and role-based access on the review actions.

**Out of scope:**
- The back-office screens themselves (this document defines behaviour, not UI).
- Uploading image files; images are referenced by path only.
- Editing or deleting existing official temples and bulk seeding (owned by the Temple catalogue feature).
- Notifying the submitter of the outcome (submitters have no account and no contact channel).
- Duplicate detection at submission time (decision: staff filter duplicates, see FR-004).

### 1.3 Product overview

InnerLamp is a Buddhist-practice companion app. Its Temple catalogue lets users browse temples (top list, paged list, detail). The catalogue is currently populated only by an operator-run seed, so users cannot contribute. This feature opens a controlled contribution path: users propose, staff verify and enrich, and the catalogue grows.

### 1.4 Definitions, acronyms, abbreviations

| Term | Definition |
|---|---|
| Suggestion | A user-submitted proposal for a temple not yet in the catalogue. |
| Pending | Suggestion state: submitted, not yet reviewed. |
| Created | Suggestion state: converted into an official temple. |
| Rejected | Suggestion state: reviewed and declined, with a reason. |
| Official temple | A temple row in the catalogue visible to every app user. |
| Staff | A back-office account with the admin or content-editor role. |
| Viewer | A back-office account with read-only intent; may not review. |
| Heritage rank | Classification of a temple's heritage status, used to order the catalogue. |

### 1.5 References

- `docs/specs/features/temple/requirements.md` — existing read-only Temple catalogue.
- `docs/specs/features/admin/requirements.md` — back-office accounts, roles, sessions.
- `docs/specs/features/temple-suggestion/{requirements,design,flow,tasks}.md` — engineering spec derived from the same decisions (v1.0).
- IEEE 29148-2018, *Systems and software engineering — Life cycle processes — Requirements engineering*.

### 1.6 Document organization

Section 2 describes context and users. Section 3 lists interfaces, functional requirements (with Gherkin acceptance criteria), non-functional requirements and data needs. Section 4 gives verification and traceability. Appendix A tracks open and resolved questions.

---

## 2. Overall description

### 2.1 Product perspective

An enhancement to the existing Temple catalogue. It adds one public write capability to the app-facing service and a review capability to the back-office service. Both share the same database, so an official temple created by staff appears in the catalogue immediately. No third-party system is involved.

### 2.2 Product functions

- Submit a temple suggestion anonymously (`FR-001`–`FR-005`).
- Browse and inspect the review queue (`FR-006`, `FR-007`).
- Convert a suggestion into an official temple (`FR-008`–`FR-010`).
- Reject a suggestion with a reason (`FR-011`).
- Restrict review actions by role (`FR-012`).

### 2.3 User classes and characteristics

| Actor | Description | Characteristics / permissions |
|---|---|---|
| App user (anonymous) | Any person using the app, signed in or not. | May submit suggestions; cannot see any suggestion, including their own, after submission. |
| Content editor | Back-office staff member. | May list, view, convert and reject suggestions. |
| Admin | Back-office administrator. | Same as content editor, plus account management (outside this feature). |
| Viewer | Back-office account with read-only intent. | May not review suggestions. |

### 2.4 Operating environment

- Platform: server-side Go services (app-facing service and back-office service) on containerized hosting; consumed by the iOS app and the back-office web client.
- Devices: not applicable to this document (no device-side requirement beyond sending a request).
- Network: online required for every action.

### 2.5 Design and implementation constraints

- Clean Architecture layering per the project `CLAUDE.md` (domain / usecase / repository / delivery).
- Official temples must satisfy the same data rules as the existing catalogue (allowed types, regions, heritage ranks).
- Images are stored as relative paths (`/static/images/...`) or absolute http(s) URLs; no file upload path exists.
- Public submission goes through the same client-key mechanism as the other public write routes.

### 2.6 User documentation

N/A — the app shows a submission form with inline guidance; no manual is required. Back-office staff receive a short runbook as part of the back-office UI work (out of scope here).

### 2.7 Assumptions and dependencies

- **Assumptions:**
  - A suggestion needs only three facts from the user: temple name, province/city, address.
  - Staff, not the system, decide whether a suggestion duplicates an existing temple.
  - Image files are placed in the static image location by an operator before staff enter the path.
  - There are no personal data in a suggestion beyond what the user chooses to type into the three fields.
- **Dependencies:**
  - Temple catalogue (existing table and read endpoints).
  - Back-office authentication, session revocation and roles (existing).
  - Rate-limit capability (exists as a shared package but is not yet wired into the app-facing service).
- **Deltas against the engineering spec v1.0** (to be applied by `il-spec-writer` after this SRS is approved):
  1. Add a third suggestion state `rejected` with reviewer, time and an optional reason (max 500 characters), plus a reject action (was deferred as N4). The reject action itself is delivered in a later phase.
  2. A rejected suggestion can be reviewed again and converted into a temple by an admin (new transition `rejected` → `created`).
  3. Rate limit is per client IP, 5 submissions per hour (was N1).
  4. Permitted roles are admin and content editor (was N5).
  5. Duplicates are accepted at submission (confirms AC6).
  6. Province/city is free text (was N2); the submitter's IP is not stored; suggestions are kept indefinitely.
- **Design-level items still open in the engineering spec** (not business decisions; tracked there): format of the new temple identifier, and where the heritage-rank → priority mapping lives so two services cannot drift.

---

## 3. Specific requirements

### 3.1 External interface requirements

#### 3.1.1 User interfaces

- App: a "Suggest a temple" form with three inputs and a confirmation message. The app never lists suggestions.
- Back-office: a queue view filterable by state, a detail view, a "create temple" form pre-filled from the suggestion, and a "reject" dialog requiring a reason. Screens are designed separately.

#### 3.1.2 Hardware interfaces

**N/A** — no custom hardware interface.

#### 3.1.3 Software interfaces

Four capabilities are exposed, at contract level (exact paths and payloads live in the engineering spec and `openapi.yaml`, which are not yet confirmed):

- Submit a suggestion (public).
- List suggestions by state, paged (staff).
- Read one suggestion (staff).
- Convert one pending suggestion into an official temple, or reject it (staff).

The reject capability is required but its contract is deferred to a later phase (decision 2026-10-10); the first release ships submit, list, view and convert.

The reject action's contract (route and body) is designed in that later phase; until then it has no OpenAPI entry.

#### 3.1.4 Communication interfaces

HTTPS with JSON bodies. Public submission carries the client key header like other public routes; staff routes carry a back-office access token. Errors use the project's shared response envelope.

### 3.2 Functional requirements

#### Submitting a suggestion (app user)

**FR-001 — Submit a suggestion**
- **Priority:** Must
- **Description:** WHEN an app user submits a temple name, province/city and address, THE SYSTEM SHALL store a suggestion in the pending state and confirm receipt, without requiring the user to be signed in.
- **Verification method:** unit test, end-to-end test
- **Source:** Original request (client sends temple information to be added)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Anonymous user submits a valid suggestion
    Given a user who is not signed in
    When they submit a temple name, a province/city and an address
    Then the suggestion is stored as pending
    And the user receives a confirmation containing the suggestion identifier and the pending state

  Scenario: Signed-in user submits a valid suggestion
    Given a signed-in user
    When they submit a temple name, a province/city and an address
    Then the outcome is identical to an anonymous submission
  ```

**FR-002 — Validate submitted text**
- **Priority:** Must
- **Description:** WHEN a submission has a missing or blank value, or a value longer than the allowed length, THE SYSTEM SHALL refuse it, identify the offending value, and store nothing. Leading and trailing whitespace is ignored; accepted values are kept exactly as typed otherwise.
- **Verification method:** unit test
- **Source:** Normalization — input must be bounded
- **Acceptance criteria:**
  ```gherkin
  Scenario: Blank value
    Given a submission whose address contains only spaces
    When it is submitted
    Then it is refused and the address is named as the problem
    And nothing is stored

  Scenario: Value too long
    Given a submission whose temple name exceeds the maximum length
    When it is submitted
    Then it is refused and the temple name is named as the problem

  Scenario: Surrounding whitespace
    Given a submission whose temple name has spaces before and after it
    When it is submitted
    Then it is stored without those surrounding spaces and otherwise unchanged

  Scenario: Malformed request
    Given a request that is not well-formed or carries unexpected content
    When it is submitted
    Then it is refused and nothing is stored
  ```

**FR-003 — Limit submission rate per IP**
- **Priority:** Must
- **Description:** WHEN the same client IP submits suggestions faster than the permitted rate, THE SYSTEM SHALL refuse the excess submissions with a "too many requests" outcome and store nothing for them. The permitted rate is 5 submissions per IP per hour.
- **Verification method:** unit test, end-to-end test
- **Source:** Decision 2026-10-10 (per-IP rate limit)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Within the limit
    Given an IP that has submitted fewer suggestions than the permitted number in the window
    When it submits another valid suggestion
    Then the suggestion is stored

  Scenario: Over the limit
    Given an IP that has reached the permitted number in the window
    When it submits another suggestion
    Then it is refused as too many requests and nothing is stored

  Scenario: Window expires
    Given an IP that was refused for exceeding the limit
    When the window has passed and it submits again
    Then the suggestion is stored

  Scenario: Different IPs are independent
    Given one IP that has reached the limit
    When a different IP submits a valid suggestion
    Then it is stored
  ```
- **Known risk (accepted for v1):** many legitimate users may share one IP (carrier NAT, offices) and could be limited together; the limit is not loosened or keyed differently in v1.

**FR-004 — Accept duplicates at submission**
- **Priority:** Must
- **Description:** WHEN a submission matches an existing temple or another suggestion, THE SYSTEM SHALL accept it exactly like any other, so that a submitter cannot learn what the catalogue or queue contains.
- **Verification method:** unit test
- **Source:** Decision 2026-10-10 (staff filter duplicates)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Suggestion matches an existing temple
    Given an official temple with the same name and province/city
    When a user submits that temple
    Then the suggestion is stored as pending
    And the response is indistinguishable from any other accepted submission

  Scenario: Suggestion matches a pending suggestion
    Given a pending suggestion with the same name and province/city
    When another user submits it
    Then a second pending suggestion is stored
  ```

**FR-005 — Keep suggestions private from the app**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL NOT offer any app-facing capability to list or read suggestions; the submitter receives only the confirmation at submission time.
- **Verification method:** end-to-end test
- **Source:** Normalization — public write must not become a public read
- **Acceptance criteria:**
  ```gherkin
  Scenario: App tries to read suggestions
    Given any app user, signed in or not
    When they attempt to list or open a suggestion through the app-facing service
    Then no such capability exists and nothing is returned
  ```

#### Reviewing suggestions (staff)

**FR-006 — List suggestions by state**
- **Priority:** Must
- **Description:** WHEN staff request the queue, THE SYSTEM SHALL return suggestions of the requested state (pending by default; created and rejected also available), oldest first, in pages, with the total count.
- **Verification method:** unit test, repository test, end-to-end test
- **Source:** Original request (staff review the proposals)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Default view is the pending queue
    Given pending, created and rejected suggestions exist
    When staff open the queue without choosing a state
    Then only pending suggestions are shown, oldest first

  Scenario: Choose another state
    Given rejected suggestions exist
    When staff ask for rejected suggestions
    Then only rejected suggestions are shown

  Scenario: Unknown state
    Given staff ask for a state that does not exist
    When the request is made
    Then it is refused and the state is named as the problem

  Scenario: Page beyond the end
    Given fewer suggestions than a full page
    When staff ask for a later page
    Then an empty page is returned with the same totals

  Scenario: Invalid page number
    Given a page number that is not a positive whole number
    When staff ask for it
    Then it is refused and the page is named as the problem

  Scenario: Empty queue
    Given no suggestions exist
    When staff open the queue
    Then an empty list is returned with a total of zero
  ```

**FR-007 — View one suggestion**
- **Priority:** Must
- **Description:** WHEN staff open a suggestion, THE SYSTEM SHALL show what the user submitted, its state, when it was received and, once reviewed, who reviewed it, when, the resulting temple (if created) or the reason (if rejected).
- **Verification method:** unit test, end-to-end test
- **Source:** Original request
- **Acceptance criteria:**
  ```gherkin
  Scenario: Open a pending suggestion
    Given a pending suggestion
    When staff open it
    Then they see the submitted values and the pending state
    And there is no reviewer, review time, temple or reason

  Scenario: Open a created suggestion
    Given a suggestion converted into a temple
    When staff open it
    Then they see the reviewer, the review time and a reference to the official temple

  Scenario: Open a rejected suggestion
    Given a rejected suggestion
    When staff open it
    Then they see the reviewer, the review time and the rejection reason

  Scenario: Suggestion does not exist
    Given an identifier that matches no suggestion
    When staff open it
    Then they are told it was not found
  ```

#### Converting a suggestion into an official temple (staff)

**FR-008 — Create an official temple from a pending or rejected suggestion**
- **Priority:** Must
- **Description:** WHEN staff submit a completed temple form for a pending suggestion (or an admin does so for a rejected one, see FR-010), THE SYSTEM SHALL create the official temple and mark the suggestion as created, recording the reviewer, the time and the resulting temple, as one indivisible action.
- **Verification method:** unit test, repository test, end-to-end test
- **Source:** Original request (staff add images/information like existing temples and submit; suggestion is marked "temple created")
- **Acceptance criteria:**
  ```gherkin
  Scenario: Staff convert a pending suggestion
    Given a pending suggestion and a complete, valid temple form
    When staff submit the form
    Then an official temple exists with the submitted information
    And the suggestion is marked created with the reviewer, the time and the temple reference

  Scenario: Temple appears in the catalogue immediately
    Given a suggestion that has just been converted
    When an app user browses the catalogue
    Then the new temple can be found in the paged list and opened in detail

  Scenario: Suggestion text is preserved
    Given a suggestion that has been converted
    When staff open it later
    Then it still shows exactly what the user originally submitted
  ```

**FR-009 — Validate the temple form**
- **Priority:** Must
- **Description:** WHEN staff submit a temple form, THE SYSTEM SHALL require name, type, region, sect, founding year, heritage rank, coordinates and address, restrict type, region and heritage rank to the catalogue's allowed values, keep coordinates and year within valid ranges, accept image references only as a static-image path or an http(s) address, treat all other information as optional, and derive the catalogue ranking from the heritage rank rather than accept it from staff.
- **Verification method:** unit test
- **Source:** Normalization — official temples must meet the catalogue's data rules
- **Acceptance criteria:**
  ```gherkin
  Scenario: Missing required information
    Given a temple form without a sect
    When staff submit it
    Then it is refused, the sect is named as the problem, and nothing is created

  Scenario: Value outside the allowed set
    Given a temple form with a heritage rank that the catalogue does not define
    When staff submit it
    Then it is refused and the heritage rank is named as the problem

  Scenario: Coordinates out of range
    Given a temple form with a latitude outside the valid range
    When staff submit it
    Then it is refused and the latitude is named as the problem

  Scenario: Founding year in the future
    Given a temple form with a founding year later than the current year
    When staff submit it
    Then it is refused and the year is named as the problem

  Scenario: Image reference of an unsupported form
    Given a temple form whose image reference is neither a static-image path nor an http(s) address
    When staff submit it
    Then it is refused and the image reference is named as the problem

  Scenario: No images yet
    Given a valid temple form with no image references
    When staff submit it
    Then the temple is created and shown to app users as having no image

  Scenario: Ranking is system-derived
    Given a valid temple form
    When the temple is created
    Then its position in the catalogue follows its heritage rank and founding year
  ```

**FR-010 — One suggestion yields at most one temple**
- **Priority:** Must
- **Description:** WHEN a suggestion has already been converted, or two staff members act on it at the same moment, THE SYSTEM SHALL let at most one action succeed, leave no partial result, and tell the other staff member the suggestion was already handled.
- **Verification method:** repository test (concurrent), end-to-end test
- **Source:** Edge-case analysis (concurrency, retry)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Convert an already converted suggestion
    Given a suggestion already marked created
    When staff submit another temple form for it
    Then it is refused as already handled and no temple is created

  Scenario: Two staff convert at the same time
    Given one pending suggestion and two staff members submitting forms together
    When both actions run
    Then exactly one temple is created and one staff member is told it was already handled

  Scenario: Failure part-way
    Given a pending suggestion and a valid form
    When the action fails before it completes
    Then no temple exists and the suggestion is still pending

  Scenario: Admin converts a rejected suggestion
    Given a rejected suggestion that an admin has opened again in the review page
    When the admin submits a complete, valid temple form
    Then the temple is created
    And the suggestion is marked created with the admin as reviewer, replacing the rejection

  Scenario: Rejected suggestions stay reviewable
    Given rejected suggestions exist
    When an admin opens the rejected list
    Then each rejected suggestion can be opened with its reason and reviewer
  ```
- **Rule:** converting a rejected suggestion is allowed for admins only; a content editor may not (see FR-012).

#### Rejecting a suggestion (staff)

**FR-011 — Reject a suggestion with a reason**
- **Priority:** Should
- **Description:** WHEN staff reject a pending suggestion, THE SYSTEM SHALL mark it rejected and record the reviewer, the time and the optional reason (up to 500 characters), so the pending queue stays clean and the decision is auditable. Delivered in a later phase than FR-001–FR-010.
- **Verification method:** unit test, repository test, end-to-end test
- **Source:** Decision 2026-10-10 (add a rejected state); supersedes the earlier "pending/created only"
- **Acceptance criteria:**
  ```gherkin
  Scenario: Reject a pending suggestion
    Given a pending suggestion
    When staff reject it with a reason
    Then it is marked rejected with the reviewer, the time and the reason
    And it no longer appears in the pending queue

  Scenario: Reject without a reason
    Given a pending suggestion
    When staff reject it without giving a reason
    Then it is marked rejected with no reason

  Scenario: Reason too long
    Given a pending suggestion
    When staff reject it with a reason longer than 500 characters
    Then the rejection is refused, the reason is named as the problem, and the suggestion stays pending

  Scenario: Reject an already handled suggestion
    Given a suggestion that is already created or already rejected
    When staff reject it
    Then it is refused as already handled and nothing changes
  ```

#### Access control

**FR-012 — Restrict review actions by role**
- **Priority:** Must
- **Description:** WHEN a back-office request is made to list, view, convert or reject suggestions, THE SYSTEM SHALL require a valid, non-revoked staff session and allow it only for the admin and content-editor roles, except that converting a rejected suggestion into a temple is allowed for the admin role only.
- **Verification method:** unit test, end-to-end test
- **Source:** Decision 2026-10-10 (admin + content editor)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Content editor reviews
    Given a signed-in content editor
    When they list, view, convert or reject a suggestion
    Then the action is allowed

  Scenario: Content editor cannot convert a rejected suggestion
    Given a signed-in content editor and a rejected suggestion
    When they submit a temple form for it
    Then the action is forbidden and the suggestion stays rejected

  Scenario: Viewer is blocked
    Given a signed-in viewer
    When they attempt any review action
    Then the action is forbidden and nothing changes

  Scenario: No valid session
    Given a request with no token, a malformed token, a refresh token or a revoked session
    When any review action is attempted
    Then the request is rejected as unauthenticated and its content is not read

  Scenario: Unknown role
    Given a token whose role the system does not recognize
    When any review action is attempted
    Then the action is forbidden
  ```

### 3.3 Performance requirements

- **NFR-001 — Submission throughput guard:** the per-IP limit of FR-003 holds under concurrent requests from one IP (no burst beyond the limit by racing requests). The limit is 5 per IP per hour (see FR-003).
- **NFR-002 — Queue responsiveness:** a page of the queue loads within 1 second for staff with up to 10,000 suggestions stored; the index is sized for that volume.

### 3.4 Logical database requirements

- Persist, server-side, every suggestion with: submitted name, province/city and address; state (pending, created, rejected); received time; and, once reviewed, reviewer, review time, resulting temple (created) or reason (rejected).
- A temple may originate from at most one suggestion, and a created suggestion always references its temple.
- State and review data must be consistent: a pending suggestion has no review data; a created one has reviewer, time and temple; a rejected one has reviewer, time and reason.
- Official temples are stored in the existing catalogue table and follow its constraints.
- All suggestions, whatever their state, are kept indefinitely; there is no purge job.
- The submitter's IP address is not stored with a suggestion (the rate limit uses it transiently only).
- Province/city is stored as free text exactly as typed; consistent naming and province filtering are not guaranteed.

### 3.5 Design constraints

- Follow the existing response envelope, strict request-body decoding and error-code conventions.
- Role checks reuse the existing back-office session and role mechanism; no new RBAC model is introduced.
- Database changes ship as an idempotent migration like all others.

### 3.6 Software system attributes

- **Reliability:** conversion is all-or-nothing (FR-008, FR-010); a failed attempt leaves no temple and the suggestion pending.
- **Availability:** same as the hosting services; no extra availability target.
- **Security:** the public endpoint must not reveal catalogue or queue contents (FR-004, FR-005); errors never expose internal causes; review actions require a valid staff session and an allowed role (FR-012); the review is attributable to a named staff account (FR-007, FR-008, FR-011).
- **Maintainability:** the heritage-rank → priority mapping must exist in one place so the two services cannot rank differently.
- **Portability:** N/A — server-side feature.

---

## 4. Verification & traceability

### 4.1 Verification approach

Each requirement lists its verification method. Unit tests cover validation and orchestration with hand-written mocks; repository tests run against a real Postgres, including concurrency and constraint behaviour; the end-to-end suite runs the compiled services against an ephemeral Postgres to prove route wiring, authentication, roles and the full submit → review → catalogue journey.

### 4.2 Traceability matrix

| Requirement ID | Source | Design | Test |
|---|---|---|---|
| FR-001 | Original request | → il-spec-writer (`requirements.md` AC1) | 2 Gherkin scenarios |
| FR-002 | Normalization | → il-spec-writer (AC2–AC4) | 4 Gherkin scenarios |
| FR-003 | Decision 2026-10-10 | → il-spec-writer (AC7) | 4 Gherkin scenarios |
| FR-004 | Decision 2026-10-10 | → il-spec-writer (AC6) | 2 Gherkin scenarios |
| FR-005 | Normalization | → il-spec-writer (AC8) | 1 Gherkin scenario |
| FR-006 | Original request | → il-spec-writer (AC9–AC12) | 6 Gherkin scenarios |
| FR-007 | Original request | → il-spec-writer (AC13–AC14) | 4 Gherkin scenarios |
| FR-008 | Original request | → il-spec-writer (AC16, AC17, AC25, AC26) | 3 Gherkin scenarios |
| FR-009 | Normalization | → il-spec-writer (AC18–AC21) | 7 Gherkin scenarios |
| FR-010 | Edge-case analysis | → il-spec-writer (AC22, AC24) | 4 Gherkin scenarios |
| FR-011 | Decision 2026-10-10 | → il-spec-writer (**new** — not in v1.0) | 3 Gherkin scenarios |
| FR-012 | Decision 2026-10-10 | → il-spec-writer (AC15, AC27) | 4 Gherkin scenarios |
| NFR-001 | Decision 2026-10-10 | → il-spec-writer | TBD |
| NFR-002 | Normalization | → il-spec-writer | TBD |
| NFR-003 | Original request | → il-spec-writer (AC25) | 1 end-to-end test |

---

## 5. Appendices

### Appendix A — Open questions

#### Open (unconfirmed)

None.

#### Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| Who may submit a suggestion? | Anyone, no sign-in required. | 2026-10-10 |
| How do staff add images? | By entering image paths; no file upload in this version. | 2026-10-10 |
| How is the official temple created given the catalogue's mandatory fields? | Staff complete the full form; conversion creates the temple and marks the suggestion created in one atomic action. | 2026-10-10 |
| How is the public endpoint protected from abuse? | Per-IP rate limit. | 2026-10-10 |
| Which back-office roles may review? | Admin and content editor. | 2026-10-10 |
| What happens to junk or duplicate suggestions? | New `rejected` state with reviewer, time and reason. | 2026-10-10 |
| Detect duplicates at submission? | No; accepted like any other, staff filter. | 2026-10-10 |
| Permitted submissions per IP and window? | 5 per IP per hour. | 2026-10-10 |
| Can a rejected suggestion be reopened and converted? | Yes — an admin can review a rejected suggestion again in the admin page and convert it into a temple. | 2026-10-10 |
| Is a rejection reason mandatory, and how long may it be? | Optional, up to 500 characters. | 2026-10-10 |
| How long are reviewed suggestions kept? | Indefinitely. | 2026-10-10 |
| Store the submitter's IP? | No. | 2026-10-10 |
| Province/city free text or fixed list? | Free text. | 2026-10-10 |
| When is the reject action's API built? | Later, as a back-office API in a subsequent phase; the capability is still required. | 2026-10-10 |
| May a content editor convert a rejected suggestion? | No — admin only. | 2026-10-10 |
| How to treat many users behind one shared IP? | Accepted as a known risk for v1; no special handling. | 2026-10-10 |
| Queue-page speed target and volume? | Page loads within 1 second with up to 10,000 suggestions. | 2026-10-10 |
| Reject action's API contract? | Designed in the later phase that builds the reject action. | 2026-10-10 |
