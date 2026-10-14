# Software Requirements Specification — Sign-out (Logout)

> Superseded by [docs/product/login/product.md](../../../product/login/product.md) (v1.0) — kept for history; do not update.

| Field | Value |
|---|---|
| Document type | Software Requirements Specification (SRS) — IEEE 29148-2018 |
| Feature | Sign-out (logout) for a signed-in user |
| Product | InnerLamp |
| Version | 0.1 |
| Last updated | 2026-10-11 |
| Status | retroactive — the feature is implemented and verified end to end (login spec v1.13, AC13–AC15); this SRS records it in IEEE 29148 form. 3 items are recorded as not confirmed (Appendix A) |
| Owner / author | duc@wee.vn (BA assist: Claude) |
| Downstream handoff | None pending. Engineering spec already exists: `docs/specs/features/login/{requirements,design,flow,tasks}.md` (current v1.13) |

---

## 1. Introduction

### 1.1 Purpose

This document states, in business terms, what must happen when a signed-in InnerLamp user chooses to sign out. It was written after the feature shipped, from the engineering spec, the code and the test-coverage report, so that the business view and the engineering spec can be compared.

### 1.2 Scope

**In scope:**
- Ending the session of the device the user is signing out from.
- Making that session's refresh token unusable afterwards.
- Behaviour when the session is already ended, unknown, or the request is not properly authenticated.
- Behaviour when the request comes from the website (web API key).

**Out of scope:**
- Signing out of all other devices ("logout everywhere") — planned for a later phase.
- Ending the *access* token immediately (see `FR-005`, decision: not done).
- Account deletion, password change or password reset (separate features).
- Sign-in and token refresh (specified in the same login spec, not repeated here).
- Screens and copy of the sign-out button and confirmation (client concern).

### 1.3 Product overview

InnerLamp is a Buddhist-practice companion app (iOS, plus a website). A user signs in with a phone number and password and gets a short-lived access token and a longer-lived refresh token. Each device has its own session. Sign-out is how a user ends the session on the device in their hand, for example on a shared or handed-over phone.

### 1.4 Definitions, acronyms, abbreviations

| Term | Definition |
|---|---|
| Session | The server-side record of one signed-in device, which the refresh token belongs to. |
| Access token | Short-lived credential (15 minutes) sent with each request to prove who the user is. |
| Refresh token | Longer-lived credential (7 days) used only to get a new access token. |
| `sid` | A value inside the access token that names the session the token belongs to. |
| Revoked session | A session that can no longer be refreshed. |
| Web API key | A shared secret held by the InnerLamp website server to identify the website as the caller. |
| Idempotent | Repeating the same request gives the same result and changes nothing more. |

### 1.5 References

- `docs/specs/features/login/requirements.md` — AC13–AC15 (logout), AC16–AC19 (web API key), AC10 (refresh failure).
- `docs/specs/features/login/design.md`, `flow.md`, `test-coverage-report.md`.
- `docs/specs/api/openapi.yaml` — operation `logout`.
- `docs/tech-debt/auth/TD-001-web-api-key-dev-plaintext-and-bypassable.md`.
- IEEE 29148-2018, *Systems and software engineering — Life cycle processes — Requirements engineering*.

### 1.6 Document organization

Section 2 gives context and users. Section 3 lists interfaces, functional and non-functional requirements. Section 4 gives verification and traceability. Appendix A tracks open and resolved questions.

---

## 2. Overall description

### 2.1 Product perspective

An enhancement of the existing sign-in feature (same spec folder, same user story US-002). It is the only authentication route that requires the caller to be signed in. It relies on the session record created at sign-in or registration and does not touch any third-party system.

### 2.2 Product functions

- End the current device's session (`FR-001`).
- Safe to repeat or to call for an already-ended session (`FR-002`).
- Refuse callers that are not properly signed in (`FR-003`).
- Never end another user's session (`FR-004`).
- Keep other devices of the same user signed in (`FR-005`).
- Accept the request from the website with its API key (`FR-006`).

### 2.3 User classes and characteristics

| Actor | Description | Characteristics / permissions |
|---|---|---|
| Signed-in user | Any user holding a valid access token for an active session. | Can end only the session named in their own token. |
| Website server | The InnerLamp website acting on behalf of a user. | Sends the web API key in addition to the user's token. |
| Mobile app | The iOS app. | Sends only the user's token. |

### 2.4 Operating environment

- Platform: iOS app and a website, both calling the same backend over HTTPS.
- Network: online required. There is no offline sign-out in the backend contract.

### 2.5 Design and implementation constraints

- Logout is a `POST` with no body; the session is identified from the access token, not from client-supplied data.
- The response uses the standard envelope used by all routes (`{"success": true}` on success).
- Only access tokens are accepted as the bearer; a refresh token is rejected.

### 2.6 User documentation

Developer-facing: `web/docs` guide "Xác thực" (section "Đăng xuất") and the `logout` operation in the OpenAPI reference. No end-user manual is required.

### 2.7 Assumptions and dependencies

- **Assumptions:** the client deletes both tokens from the device after sign-out, whatever the response was.
- **Dependencies:** sign-in and registration must put the session id into the access token (done, login v1.0 and onboarding Phase 7); the web API key must be configured for website callers.

---

## 3. Specific requirements

### 3.1 External interface requirements

#### 3.1.1 User interfaces
Not specified here. The client shows a sign-out action and returns the user to the signed-out state.

#### 3.1.2 Hardware interfaces
**N/A** — no custom hardware interface.

#### 3.1.3 Software interfaces

| Method | Path | Caller must send | Success | Failure |
|---|---|---|---|---|
| POST | `/v1/auth/logout` | `Authorization: Bearer <access token>`; no body; website also sends the web API key | `200` `{"success": true}` | `401` `UNAUTHORIZED`, `401` `INVALID_API_KEY`, `500` |

#### 3.1.4 Communication interfaces
HTTPS and JSON. The failure body uses the common error envelope; a `401` never says why the token was rejected.

### 3.2 Functional requirements

#### Ending a session

**FR-001 — Sign out of the current device**
- **Priority:** Must
- **Description:** WHEN a signed-in user asks to sign out with a valid, unexpired access token whose session is still active, THE SYSTEM SHALL end that session and report success.
- **Verification method:** automated unit, repository and end-to-end tests.
- **Source:** login spec AC13.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Sign out with an active session
    Given the user is signed in on a device
    When the user signs out
    Then the user is told that sign-out succeeded
    And the refresh token of that device can no longer be used to get a new access token
  ```

**FR-002 — Repeating sign-out is safe**
- **Priority:** Must
- **Description:** WHEN a user asks to sign out and the session named by the token is already ended, has expired, or is unknown, THE SYSTEM SHALL report success and SHALL NOT change anything.
- **Verification method:** automated unit, repository and end-to-end tests.
- **Source:** login spec AC14.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Sign out twice in a row
    Given the user has already signed out on a device
    When the same sign-out request is sent again
    Then the user is told that sign-out succeeded
    And no other session changes

  Scenario: Retry after a lost response
    Given the user signed out but the confirmation never reached the app
    When the app retries the sign-out
    Then the user is told that sign-out succeeded
  ```

**FR-003 — Refuse callers that are not properly signed in**
- **Priority:** Must
- **Description:** WHEN a sign-out request has no token, an invalid or expired token, a refresh token instead of an access token, or an access token with no session id, THE SYSTEM SHALL refuse it as unauthorized and SHALL NOT change anything.
- **Verification method:** automated handler and end-to-end tests (five cases over HTTP).
- **Source:** login spec AC15.
- **Acceptance criteria:**
  ```gherkin
  Scenario Outline: Reject a request that is not properly signed in
    Given the caller presents <credential>
    When the caller asks to sign out
    Then the request is refused as unauthorized
    And the reason is not revealed
    And the user's session still works afterwards

    Examples:
      | credential                              |
      | no token                                |
      | a token with a bad signature            |
      | an expired access token                 |
      | a refresh token                         |
      | an access token without a session id    |
  ```

**FR-004 — A user can only end their own session**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL end only a session that belongs to the user in the token, even if the token names a session id that belongs to someone else.
- **Verification method:** repository and end-to-end tests (a validly signed token of user A naming user B's session).
- **Source:** login spec AC14, parenthetical.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Token names another user's session
    Given user A holds a valid token that names a session of user B
    When user A signs out
    Then user B's session is still active
    And user A's own other sessions are not changed
  ```

**FR-005 — Other devices stay signed in**
- **Priority:** Must
- **Description:** WHEN a user signs out on one device, THE SYSTEM SHALL leave the user's sessions on other devices unchanged. The access token already issued for the signed-out session remains accepted until it expires (up to 15 minutes).
- **Verification method:** design review; repository test (only the matching row changes).
- **Source:** login spec "Out of scope" (logout everywhere) and technical note "Access token sau logout / revoke (chốt 2026-09-21)".
- **Acceptance criteria:**
  ```gherkin
  Scenario: Signed in on two devices, sign out on one
    Given the user is signed in on a phone and on a tablet
    When the user signs out on the phone
    Then the tablet can still refresh and call the API
  ```

#### Website caller

**FR-006 — Website calls carry the API key**
- **Priority:** Must
- **Description:** WHEN a sign-out request carries a web API key, THE SYSTEM SHALL accept it only if the key is correct; a wrong or empty key SHALL be refused as an invalid API key before the user's token is looked at. A request with no key SHALL be treated as coming from the mobile app.
- **Verification method:** automated handler and middleware tests.
- **Source:** login spec AC16–AC18.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Website signs out with the correct key
    Given the website sends the correct API key and a valid access token
    When the user signs out
    Then the user is told that sign-out succeeded

  Scenario: Website sends a wrong key
    Given the caller sends a wrong API key and a valid access token
    When the user signs out
    Then the request is refused as an invalid API key
    And the session is not ended

  Scenario: Mobile app sends no key
    Given the caller sends no API key and a valid access token
    When the user signs out
    Then the user is told that sign-out succeeded
  ```

### 3.3 Performance requirements

- **NFR-001 — Sign-out latency:** the backend contract does not set a target.
  > ⚠️ Not confirmed: Is there a response-time target for sign-out (for example, completes within 1 s at the 95th percentile)? — blocks writing a performance test; nothing else depends on it.

### 3.4 Logical database requirements

- The server keeps one session record per signed-in device. Ending a session marks it revoked with the reason "logout"; the record is not deleted.
- Only a hash of the refresh token is stored.
- Retention of revoked session records: not specified in the login spec.
  > ⚠️ Not confirmed: How long are revoked sessions kept before deletion? — blocks a data-retention rule; nothing in sign-out depends on it.

### 3.5 Design constraints

- The route must be wrapped by the access-token check so that a missing or wrong-type token is refused before the handler runs.
- Sign-out identifies the session by the user id and session id from the token together, never by an id sent by the client.

### 3.6 Software system attributes

- **Reliability:** safe to retry (`FR-002`).
- **Availability:** same as the rest of the backend; no separate target.
- **Security:** a refused request never says whether the token was expired, forged or of the wrong type; no token or session id is returned in the response. Known limits: the access token stays valid up to 15 minutes after sign-out (`FR-005`); the web API key does not stop a caller that omits it (`TD-001`).
- **Maintainability:** the behaviour is covered by AC13–AC15 in the login spec and by the end-to-end test.
- **Portability:** one contract serves iOS and the website.
- **Offline:** the backend cannot end a session without a connection.
  > ⚠️ Not confirmed: When the device is offline or the request fails, should the client still show the user as signed out and retry later, or block sign-out? — blocks the client's sign-out behaviour; the backend contract is not affected.

---

## 4. Verification & traceability

### 4.1 Verification approach

Automated tests at three levels (usecase, repository on a real database, handler) plus black-box end-to-end tests on the real binary against Postgres, as recorded in `test-coverage-report.md`. The `FR-005` access-token window is verified by design review because it is a deliberate non-behaviour.

### 4.2 Traceability matrix

| Requirement ID | Source | Design | Test |
|---|---|---|---|
| FR-001 | login AC13 | `login/design.md`, `login/flow.md` | 1 Gherkin scenario; unit + repository + handler + E2E |
| FR-002 | login AC14 | `login/design.md` | 2 Gherkin scenarios; unit + repository + E2E |
| FR-003 | login AC15 | `login/design.md` (Auth model) | 1 outline, 5 examples; handler + E2E |
| FR-004 | login AC14 | `login/design.md` | 1 Gherkin scenario; repository + E2E |
| FR-005 | login Out of scope; technical note 2026-09-21 | `login/design.md` (Known gaps) | 1 Gherkin scenario; design review |
| FR-006 | login AC16–AC18 | `login/design.md` (Cross-cutting: web `X-API-Key`) | 3 Gherkin scenarios; handler + middleware tests |
| NFR-001 | — | — | TBD |

---

## 5. Appendices

### Appendix A — Open questions

#### Open (unconfirmed)

> ⚠️ Not confirmed: Is there a response-time target for sign-out? — blocks only a performance test (`NFR-001`).

> ⚠️ Not confirmed: How long are revoked session records retained? — blocks a retention rule (section 3.4).

> ⚠️ Not confirmed: What should the client do when sign-out cannot reach the server (offline or error)? — blocks the client's sign-out behaviour (section 3.6).

#### Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| Is there a logout in the login feature? | Yes, `POST /v1/auth/logout`. | 2026-09-20 |
| Does a session still-live access token get blocked after logout? | No; stateless, valid until expiry (≤ 15 minutes). | 2026-09-21 |
| How does logout find the session: refresh token or access token + `sid`? | Access token in the `Authorization` header + `sid` claim, route requires sign-in. | 2026-09-21 |
| Sign out of all other devices? | Not in this phase. | 2026-09-20 |
