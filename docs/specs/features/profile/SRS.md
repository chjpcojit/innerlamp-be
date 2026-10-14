# Software Requirements Specification — User Profile (View and Edit)

| Field | Value |
|---|---|
| Document type | Software Requirements Specification (SRS) — IEEE 29148-2018 |
| Feature | User Profile: a signed-in user reads and edits their own profile |
| Product | InnerLamp |
| Version | 0.4 (draft) |
| Last updated | 2026-10-11 |
| Status | draft — 25 questions answered 2026-10-11; no open question left; 2 technical debts tracked (TD-001, TD-002) |
| Owner / author | duc@wee.vn (BA assist: Claude) |
| Downstream handoff | `il-spec-writer` → `requirements.md` / `design.md` / `flow.md` / `tasks.md` under `docs/specs/features/profile/` |

---

## 1. Introduction

### 1.1 Purpose

This document specifies, in business terms, how a signed-in InnerLamp user retrieves their own profile and edits the parts of it that are theirs to change: dharma name, ordination temple and avatar. It is written for product and engineering and drives the engineering spec that `il-spec-writer` produces next.

It **supersedes** the prospective `US-004` section of `docs/specs/features/onboarding/requirements.md` (a user story only, never designed). That section still describes the ordination place as free text; since onboarding v7.0 it is a reference to a temple. Once this SRS is approved, that section should point here.

### 1.2 Scope

**In scope:**
- Reading the signed-in user's own profile: email (with verification status), phone number, dharma name, ordination temple, avatar.
- Editing dharma name and ordination temple, including clearing either one.
- Setting, replacing and removing the avatar, using a direct-to-storage upload.
- Rejecting every one of these actions when the caller is not signed in.

**Out of scope:**
- Changing email or phone number (they are login identity and need their own verification flow).
- Changing the password while signed in (distinct from "forgot password").
- Deleting the account.
- Viewing another user's profile (this feature is strictly "my own").
- Email verification itself (the profile only reports its status).
- Any new profile field beyond the three editable ones above.

### 1.3 Product overview

InnerLamp is a practice companion for Buddhist practitioners. Users register by phone OTP, sign in, and then use features such as lamp lighting, memorials and genealogy. The profile is where a user keeps the personal details that describe their practice: the dharma name received at refuge and the temple where they took refuge or were ordained. A user often registers first and receives a dharma name later, so editing after registration is a real need.

### 1.4 Definitions, acronyms, abbreviations

| Term | Definition |
|---|---|
| Profile | The set of personal details of one user that this feature reads and edits. |
| Dharma name (pháp danh) | The name given to a practitioner at refuge. Optional free text. |
| Ordination temple (nơi quy y/xuất gia) | The temple where the user took refuge or was ordained, chosen from the existing temple catalogue. Optional. |
| Avatar | The user's profile picture, a single image. Optional. |
| Identity fields | Email and phone number; they identify the account and are not editable here. |
| Signed upload URL | A short-lived address the client uses to send the image straight to storage without the backend handling the file bytes. |
| Clear | Remove a stored optional value so the profile reads back as "not given". |
| SRS | Software Requirements Specification. |
| PII | Personally identifiable information. |

### 1.5 References

- `docs/specs/features/onboarding/requirements.md` — registration (US-001), the optional profile fields at sign-up, and the superseded US-004.
- `docs/specs/features/login/requirements.md` — sign-in and the access token this feature relies on.
- `docs/specs/features/temple/` — the temple catalogue the ordination temple points to.
- `migrations/0001_register_onboarding.sql`, `migrations/0023_registration_ordination_temple.sql` — current shape of the stored user.
- IEEE 29148-2018.

### 1.6 Document organization

Section 2 describes the feature in context. Section 3 lists interface, functional (with Gherkin acceptance criteria) and non-functional requirements. Section 4 gives verification and traceability. Appendix A tracks open and resolved questions.

---

## 2. Overall description

### 2.1 Product perspective

This is an enhancement to the existing identity area, on top of registration and login. It adds no new actor. It depends on:

- the signed-in session produced by login (the user is identified from their access token, never from an identifier they send);
- the temple catalogue, to validate and describe the ordination temple;
- object storage for avatars, which **does not exist yet** in the codebase (no storage code is wired and the stored user has no avatar column). This is the largest piece of new infrastructure in the feature.

### 2.2 Product functions

- View own profile (`FR-001`).
- Edit dharma name (`FR-002`) and ordination temple (`FR-003`) with partial-update semantics (`FR-004`).
- Keep identity fields read-only (`FR-005`).
- Set, replace and remove the avatar (`FR-006`, `FR-007`, `FR-008`).
- Require sign-in for all of the above (`FR-009`).

### 2.3 User classes and characteristics

| Actor | Description | Characteristics / permissions |
|---|---|---|
| Signed-in user | A registered user holding a valid access token. | Can read and edit only their own profile. |
| Anonymous caller | No token, an expired token or a refresh token used as a bearer. | Can do nothing in this feature. |

### 2.4 Operating environment

- Clients: the InnerLamp mobile and web clients calling the backend API.
- Network: online required; nothing is cached for offline editing.
- Avatar upload goes from the client directly to cloud object storage (GCP primary target).

### 2.5 Design and implementation constraints

- Clean Architecture layering and the project conventions in `CLAUDE.md` (domain / usecase / repository / delivery, wired from the composition root).
- Non-sensitive limits and lifetimes (avatar size, allowed types, upload-URL lifetime) are configuration values, not constants in the domain.
- Optional text is stored as absent (NULL), never as an empty string, consistent with registration.
- The profile belongs to a user-scoped, token-authenticated surface; the owner is always the token's user.

### 2.6 User documentation

**N/A** — no separate manual; the client's screen copy is covered by the client's own design. Vietnamese user-facing messages must be listed in the engineering spec.

### 2.7 Assumptions and dependencies

- **Assumptions:**
  - A user has exactly one profile and one avatar.
  - Email verification status is reported as-is; email is currently never verified, so the status is "not verified" for every user today.
  - Existing users who registered before this feature simply have an empty avatar.
- **Dependencies:**
  - Login / token validation (implemented).
  - Temple catalogue (implemented).
  - Object storage and a way to issue signed upload URLs (not built — blocking for the avatar requirements).
  - A new avatar reference stored with the user (schema change).

---

## 3. Specific requirements

### 3.1 External interface requirements

#### 3.1.1 User interfaces

One profile screen showing email with its verification status, phone number, dharma name, ordination temple (name) and avatar. Empty optional values read as clearly "not updated", never as "null" or an error. Edit actions: change dharma name, choose or clear the ordination temple (pick from the temple list), choose, replace or remove the avatar. Email and phone are displayed but not editable. Layout follows the client's design system.

#### 3.1.2 Hardware interfaces

**N/A** — no custom hardware interface.

#### 3.1.3 Software interfaces

Contract level only; shapes are fixed by the engineering spec.

- A read operation returns the signed-in user's profile.
- An update operation changes dharma name and/or ordination temple (partial update).
- A pair of avatar operations: request permission to upload, then confirm the upload; plus a remove operation.
- The temple catalogue's existing read operations supply the choices for the ordination temple.
- Errors reuse the project's common response envelope.

Route names, methods and payloads are chosen by engineering in `design.md` and the OpenAPI file (decided 2026-10-11).

#### 3.1.4 Communication interfaces

HTTPS and JSON for the API; the signed upload goes over HTTPS directly to storage. All profile calls carry the access token as a bearer credential.

### 3.2 Functional requirements

#### Reading the profile

**FR-001 — View own profile**
- **Priority:** Must
- **Description:** WHEN a signed-in user requests their profile THE SYSTEM SHALL return their email, whether the email is verified, their phone number, their dharma name, their ordination temple (identifier and name) and their avatar, omitting any value they have not given.
- **Verification method:** unit test (use case), integration test against a real database, end-to-end test.
- **Source:** user request "get their profile"; superseded US-004 AC1; answer 2026-10-11 (read shape).
- **Acceptance criteria:**
  ```gherkin
  Scenario: View a complete profile
    Given a signed-in user who has a dharma name, an ordination temple and an avatar
    When the user opens their profile
    Then the user sees their email, phone number, dharma name, temple name and avatar

  Scenario: View a profile with empty optional values
    Given a signed-in user who has no dharma name, no ordination temple and no avatar
    When the user opens their profile
    Then the empty values are shown as not updated
    And no value is shown as "null" and no error appears

  Scenario: Email is not verified
    Given a signed-in user whose email has never been verified
    When the user opens their profile
    Then the profile states that the email is not verified

  Scenario: Phone number is shown in full to its owner
    Given a signed-in user
    When the user opens their profile
    Then the phone number is shown unmasked

  Scenario: A user only ever sees their own profile
    Given two users A and B who are both signed in
    When user A opens their profile
    Then the profile shown is A's and contains none of B's data
  ```

#### Editing the profile

**FR-002 — Edit dharma name**
- **Priority:** Must
- **Description:** WHEN a signed-in user submits a new dharma name THE SYSTEM SHALL trim surrounding whitespace, store the value if it is within the allowed length and return the updated profile; WHEN the submitted value is empty THE SYSTEM SHALL clear the stored dharma name.
- **Verification method:** unit test, integration test, end-to-end test.
- **Source:** superseded US-004 AC2, AC5; answer 2026-10-11 (same rule as registration).
- **Acceptance criteria:**
  ```gherkin
  Scenario: Change the dharma name
    Given a signed-in user whose dharma name is "Minh Tâm"
    When the user saves the dharma name "Thanh Liên"
    Then the profile shows the dharma name "Thanh Liên"
    And the change is visible immediately without reloading the app

  Scenario: Surrounding whitespace is ignored
    Given a signed-in user
    When the user saves the dharma name "  Thanh Liên  "
    Then the profile shows the dharma name "Thanh Liên"

  Scenario: Clear the dharma name
    Given a signed-in user whose dharma name is "Minh Tâm"
    When the user saves an empty dharma name
    Then the profile shows the dharma name as not updated

  Scenario: A blank dharma name is the same as an empty one
    Given a signed-in user whose dharma name is "Minh Tâm"
    When the user saves a dharma name made only of spaces
    Then the profile shows the dharma name as not updated

  Scenario Outline: Reject a dharma name that is too long
    Given a signed-in user whose dharma name is "Minh Tâm"
    When the user saves a dharma name of <length> characters
    Then the user is told the dharma name is too long
    And the profile still shows the dharma name "Minh Tâm"

    Examples:
      | length |
      | 256    |
      | 1000   |

  Scenario: A name at the maximum length is accepted
    Given a signed-in user
    When the user saves a dharma name of 255 characters
    Then the profile shows that dharma name

  Scenario: Diacritics do not use up the limit
    Given a signed-in user
    When the user saves a dharma name of 255 Vietnamese letters that all carry diacritics
    Then the profile shows that dharma name
  ```
  > The 255 limit counts characters, not bytes, so each Vietnamese letter with diacritics counts as one (decided 2026-10-11). Registration today does not validate this length explicitly (a known gap tracked in `docs/HANDOFF.md`), so this requirement also asks that over-length input be a validation error, never a server error.

**FR-003 — Edit ordination temple**
- **Priority:** Must
- **Description:** WHEN a signed-in user chooses an ordination temple THE SYSTEM SHALL verify that the temple exists in the catalogue and store it; WHEN the user clears the temple THE SYSTEM SHALL remove it; WHEN the chosen temple does not exist THE SYSTEM SHALL reject the whole update and change nothing.
- **Verification method:** unit test, integration test, end-to-end test.
- **Source:** superseded US-004 AC2, AC5 (updated for temple reference); answer 2026-10-11 (reject unknown temple).
- **Acceptance criteria:**
  ```gherkin
  Scenario: Choose an ordination temple
    Given a signed-in user with no ordination temple
    When the user chooses a temple from the catalogue and saves
    Then the profile shows that temple's name

  Scenario: Change the ordination temple
    Given a signed-in user whose ordination temple is temple X
    When the user chooses temple Y and saves
    Then the profile shows temple Y

  Scenario: Clear the ordination temple
    Given a signed-in user whose ordination temple is temple X
    When the user clears the ordination temple and saves
    Then the profile shows the ordination temple as not updated

  Scenario: Reject a temple that does not exist
    Given a signed-in user whose ordination temple is temple X
    When the user saves a temple that is not in the catalogue
    Then the user is told the temple is not valid
    And the profile still shows temple X

  Scenario: An invalid temple blocks the other changes in the same save
    Given a signed-in user whose dharma name is "Minh Tâm"
    When the user saves the dharma name "Thanh Liên" together with a temple that is not in the catalogue
    Then the user is told the temple is not valid
    And the profile still shows the dharma name "Minh Tâm"
  ```
  > Only a temple that appears in the catalogue's list of selectable temples can be chosen (decided 2026-10-11). A temple that exists but is not in that list is treated as not valid.

**FR-004 — Partial update**
- **Priority:** Must
- **Description:** WHEN a signed-in user submits an update THE SYSTEM SHALL leave unchanged every editable field the request does not mention, SHALL clear every field the request explicitly sets to empty, and SHALL apply the whole update or none of it.
- **Verification method:** unit test, integration test, end-to-end test.
- **Source:** answer 2026-10-11 (omitted = unchanged, empty = clear); superseded US-004 AC5.
- **Acceptance criteria:**
  ```gherkin
  Scenario: A field that is not mentioned stays unchanged
    Given a signed-in user with dharma name "Minh Tâm" and ordination temple X
    When the user saves only a new dharma name "Thanh Liên"
    Then the profile shows "Thanh Liên" and still shows temple X

  Scenario: Only one field is cleared
    Given a signed-in user with dharma name "Minh Tâm" and ordination temple X
    When the user clears only the ordination temple and saves
    Then the profile still shows "Minh Tâm"
    And the ordination temple is shown as not updated

  Scenario: Saving with nothing to change
    Given a signed-in user
    When the user saves without changing any field
    Then the profile is unchanged
    And no error appears

  Scenario: One invalid field blocks the whole update
    Given a signed-in user with dharma name "Minh Tâm"
    When the user saves a valid temple together with a dharma name that is too long
    Then the user is told the dharma name is too long
    And neither the dharma name nor the temple has changed
  ```
  ```gherkin
  Scenario: Two devices save at the same time
    Given a signed-in user who is editing the same profile on two devices
    When both devices save a different dharma name
    Then the profile shows the dharma name from the save that arrived last
    And neither save is rejected as out of date
  ```

**FR-005 — Identity fields are read-only**
- **Priority:** Must
- **Description:** WHEN a signed-in user tries to change their email or phone number through the profile THE SYSTEM SHALL reject the request and leave both unchanged.
- **Verification method:** unit test, end-to-end test.
- **Source:** superseded US-004 AC4.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Email cannot be edited from the profile
    Given a signed-in user
    When the user tries to save a different email
    Then the request is rejected
    And the email is unchanged

  Scenario: Phone number cannot be edited from the profile
    Given a signed-in user
    When the user tries to save a different phone number
    Then the request is rejected
    And the phone number is unchanged

  Scenario: The rejected request changes nothing else
    Given a signed-in user with dharma name "Minh Tâm"
    When the user saves a new dharma name together with a different email
    Then the request is rejected
    And the dharma name is still "Minh Tâm"
  ```
  > A request that includes email or phone is rejected as a whole (decided 2026-10-11).

#### Avatar

**FR-006 — Set the avatar**
- **Priority:** Must
- **Description:** WHEN a signed-in user wants to set an avatar THE SYSTEM SHALL issue a short-lived signed upload permission limited to one JPEG, PNG or WebP image of at most 5 MB, and WHEN the upload is confirmed THE SYSTEM SHALL check the object's declared type and size against those limits, copy it from the temporary upload location to the permanent one and make it the user's avatar.
- **Verification method:** unit test, integration test against storage, end-to-end test, manual QA on device.
- **Source:** user request; superseded US-004 AC3; answers 2026-10-11 (signed upload URL; JPEG/PNG/WebP, 5 MB).
- **Acceptance criteria:**
  ```gherkin
  Scenario: Set an avatar for the first time
    Given a signed-in user with no avatar
    When the user uploads a JPEG image of 2 MB and confirms
    Then the profile shows the new avatar

  Scenario Outline: Accepted image types
    Given a signed-in user
    When the user uploads a <type> image of 1 MB and confirms
    Then the profile shows the new avatar

    Examples:
      | type |
      | JPEG |
      | PNG  |
      | WebP |

  Scenario: Reject an unsupported image type
    Given a signed-in user
    When the user tries to upload a GIF image
    Then the user is told that only JPEG, PNG and WebP images are accepted
    And the avatar is unchanged

  Scenario: Reject an image that is too large
    Given a signed-in user
    When the user tries to upload an image larger than 5 MB
    Then the user is told the image is too large
    And the avatar is unchanged

  Scenario: An image of exactly the maximum size is accepted
    Given a signed-in user
    When the user uploads an image of exactly 5 MB and confirms
    Then the profile shows the new avatar

  Scenario: The upload permission expires
    Given a signed-in user who requested permission to upload
    When the user tries to upload after the permission has expired
    Then the upload is refused
    And the avatar is unchanged

  Scenario: Confirm without having uploaded
    Given a signed-in user who requested permission to upload
    When the user confirms without uploading any image
    Then the user is told no image was received
    And the avatar is unchanged

  Scenario: The permanent avatar is a copy of the confirmed upload
    Given a signed-in user who uploaded an image and confirmed
    When the user opens their profile
    Then the avatar shown is the confirmed image
    And the image is held in the permanent location, not the temporary one
  ```
  > Upload permission lifetime: 5 minutes, set in configuration (decided 2026-10-11).
  >
  > Moderation: uploaded avatars are not moderated or scanned (decided 2026-10-11). This is a conscious risk because avatars are shown to other users.
  >
  > Content check: the server does not inspect the uploaded file's content; it relies on the declared type and size only (decided 2026-10-11). A file with an image name but non-image content can therefore become an avatar. This is a conscious risk.
  >
  > Upload location: uploads first land in a temporary folder; on confirm the object is copied to the permanent folder (decided 2026-10-11).
  >
  > Deferred: nothing cleans the temporary folder for now (decided 2026-10-11). Uploads that are never confirmed stay there and storage grows until a cleanup is added. Revisit before go-live.

**FR-007 — Replace the avatar**
- **Priority:** Must
- **Description:** WHEN a signed-in user who already has an avatar confirms a new one THE SYSTEM SHALL make the new image the avatar everywhere it is shown and SHALL delete the previous image from storage once the change has succeeded.
- **Verification method:** integration test against storage, end-to-end test.
- **Source:** superseded US-004 AC3; answer 2026-10-11 (old file deleted).
- **Acceptance criteria:**
  ```gherkin
  Scenario: Replace an existing avatar
    Given a signed-in user with an avatar
    When the user uploads a new image and confirms
    Then the profile shows the new avatar
    And the previous image is no longer available

  Scenario: A failed replacement keeps the old avatar
    Given a signed-in user with an avatar
    When the user tries to replace it with an image that is rejected
    Then the profile still shows the previous avatar
    And the previous image is still available
  ```
  > If deleting the old image fails after the new one is accepted, the old image is left as garbage: the failure is logged and the request still succeeds, with no retry (decided 2026-10-11, technical debt `TD-002`).

**FR-008 — Remove the avatar**
- **Priority:** Should
- **Description:** WHEN a signed-in user removes their avatar THE SYSTEM SHALL clear the avatar from the profile and delete the stored image.
- **Verification method:** integration test, end-to-end test.
- **Source:** answer 2026-10-11 (remove allowed, old file deleted).
- **Acceptance criteria:**
  ```gherkin
  Scenario: Remove the avatar
    Given a signed-in user with an avatar
    When the user removes the avatar
    Then the profile shows the avatar as not updated
    And the previous image is no longer available

  Scenario: Remove when there is no avatar
    Given a signed-in user with no avatar
    When the user removes the avatar
    Then the profile is unchanged
    And no error appears
  ```
  ```gherkin
  Scenario: The avatar address is short-lived
    Given a signed-in user with an avatar
    When the user opens their profile and keeps the avatar address past its lifetime
    Then the old address no longer shows the avatar
    And opening the profile again returns a fresh working address
  ```
  > Avatar read access: the avatar is served only through a short-lived signed address, never a permanent public one (decided 2026-10-11). The read address lives 15 minutes, set in configuration (decided 2026-10-11).

#### Access control

**FR-009 — Sign-in required**
- **Priority:** Must
- **Description:** WHEN a caller without a valid access token requests any profile or avatar operation THE SYSTEM SHALL refuse it as unauthorized and SHALL change nothing.
- **Verification method:** end-to-end test (the route must actually be wired behind authentication), unit test.
- **Source:** superseded US-004 AC6.
- **Acceptance criteria:**
  ```gherkin
  Scenario Outline: Anonymous caller is refused
    Given a caller who is not signed in
    When the caller tries to <operation>
    Then the caller is told they are not authorized
    And nothing is changed

    Examples:
      | operation              |
      | view a profile         |
      | edit a profile         |
      | request an avatar upload |
      | remove an avatar       |

  Scenario: Expired access token
    Given a user whose access token has expired
    When the user opens their profile
    Then the user is told they are not authorized

  Scenario: A refresh token is not accepted as an access token
    Given a user presenting a refresh token as their credential
    When the user opens their profile
    Then the user is told they are not authorized

  Scenario: A user cannot target another user's profile
    Given two signed-in users A and B
    When user A edits their profile
    Then only A's profile changes
    And B's profile is unchanged
  ```

### 3.3 Performance requirements

- **NFR-001 — Response time:** no response-time target is set at this stage; a somewhat slower response is accepted (decided 2026-10-11). Revisit if the profile screen feels slow.
- **NFR-002 — Avatar limits:** a single avatar is at most 5 MB and one of JPEG, PNG or WebP; both values are configuration, not code.

### 3.4 Logical database requirements

- The server persists, per user: dharma name (optional), ordination temple reference (optional) and a new avatar reference (optional). Email, phone and email-verification status already exist.
- The avatar image itself lives in object storage; the user record holds only a reference to it.
- Optional values are stored as absent, never as empty text.
- Retention and deletion: the avatar is deleted with a replacement or removal (FR-007, FR-008).
  > Dependency for a future feature: whether the avatar must be deleted when the account is deleted is decided with account deletion, which is out of scope here (see `TD-002`, exit criteria).

### 3.5 Design constraints

- Follows the repo's Clean Architecture dependency direction and its provider-agnostic storage abstraction; no cloud-vendor code in the domain.
- Authentication reuses the existing token middleware and the project's common error envelope.
- The profile is a user-scoped surface: the owner is the token's user, there is no user identifier in the request.

### 3.6 Software system attributes

- **Reliability:** an update is all-or-nothing (FR-004); a failed avatar change never leaves the user without their previous avatar (FR-007).
- **Availability:** reads depend on the database only; avatar upload additionally depends on object storage. If storage is down, the user can still read the profile and edit text fields.
- **Security:**
  - Only the owner can read or change a profile; the owner comes from the token, never from client input.
  - The profile response contains PII (email, phone). It must not be cached by shared caches and must not appear in logs.
  - Upload permissions are scoped to one object, one user, one size and type limit, and are short-lived.
  - Avatars are user-supplied files that are neither moderated nor content-checked (see FR-006), so an avatar may be a non-image or inappropriate image; this is an accepted risk.
  - **Rate limit (NFR-003):** at most 10 requests per minute per account, counting every profile and avatar operation: read, edit, upload request, confirm and removal (decided 2026-10-11).
    ```gherkin
    Scenario: Eleventh request within a minute is refused
      Given a signed-in user who has made 10 profile or avatar requests in the last minute
      When the user makes an 11th request
      Then the user is told to try again later
      And the 11th request changes nothing

    Scenario: Reads count toward the limit
      Given a signed-in user who has opened their profile 10 times in the last minute
      When the user opens their profile again
      Then the user is told to try again later

    Scenario: The limit is per account
      Given user A has reached the limit
      When user B makes a profile request
      Then user B's request is accepted
    ```
    > Note for design: an avatar change is about 3 requests (request permission, confirm, and the profile read that shows it); raised from 5 to 10 on 2026-10-11 so a normal avatar change plus an edit fits. Revisit if the client hits it in normal use.
- **Maintainability:** limits (name length, avatar size/types, upload lifetime) are configuration.
- **Portability:** the storage provider is replaceable behind the project's storage interface.

---

## 4. Verification & traceability

### 4.1 Verification approach

Each requirement's verification method is given in section 3.2. Because the avatar flow crosses the client, the API and object storage, it needs a black-box end-to-end check against real services (or an emulator) in addition to unit tests. FR-009 specifically requires an end-to-end test that proves the routes are behind authentication, since layer tests cannot see a missing middleware.

### 4.2 Traceability matrix

| Requirement ID | Source | Design | Test |
|---|---|---|---|
| FR-001 | User request; US-004 AC1 | → il-spec-writer | 5 scenarios |
| FR-002 | US-004 AC2, AC5 | → il-spec-writer | 7 scenarios (one outline has 2 rows) |
| FR-003 | US-004 AC2, AC5 | → il-spec-writer | 5 scenarios |
| FR-004 | US-004 AC5; answer 2026-10-11 | → il-spec-writer | 5 scenarios |
| FR-005 | US-004 AC4 | → il-spec-writer | 3 scenarios |
| FR-006 | User request; US-004 AC3 | → il-spec-writer | 8 scenarios (one outline has 3 rows; the "not really an image" scenario was dropped, a copy-on-confirm one added) |
| FR-007 | US-004 AC3 | → il-spec-writer | 2 scenarios |
| FR-008 | Answer 2026-10-11 | → il-spec-writer | 3 scenarios |
| FR-009 | US-004 AC6 | → il-spec-writer | 4 scenarios (one outline has 4 rows) |
| NFR-001 | Open | → il-spec-writer | TBD (target unconfirmed) |
| NFR-002 | Answer 2026-10-11 | → il-spec-writer | covered by FR-006 |
| NFR-003 | Answer 2026-10-11 (rate limit) | → il-spec-writer | 3 scenarios |

---

## 5. Appendices

### Appendix A — Open questions

#### Open (unconfirmed)

_None._

#### Deferred (decided, not yet built)

- Cleanup of the temporary upload folder: nothing removes unconfirmed uploads for now. Revisit before go-live (FR-006, technical debt `TD-002`).
- Verification of signed URLs against the real bucket: dev and tests use a storage emulator, so the real bucket must be checked at deploy time (technical debt `TD-001`).

#### Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| Which fields are editable? | Dharma name, ordination temple and avatar. Email and phone stay read-only. | 2026-10-11 |
| How does a partial update treat omitted vs. cleared fields? | Omitted = unchanged; explicitly empty = cleared. | 2026-10-11 |
| What does the profile read return about email and phone? | Email, whether it is verified, full phone number, dharma name, ordination temple (identifier and name). | 2026-10-11 |
| Which rule applies to the dharma name on edit? | The same rule as registration (optional, trimmed, up to 255 characters, empty stored as absent). | 2026-10-11 |
| How does the avatar reach storage? | Signed upload URL, direct to storage. | 2026-10-11 |
| What are the avatar limits? | JPEG, PNG or WebP, maximum 5 MB. | 2026-10-11 |
| Can the avatar be removed, and what happens to the old file? | Removal is allowed; the old file is deleted from storage. | 2026-10-11 |
| What if the chosen ordination temple does not exist? | The whole update is rejected with a validation error; nothing is saved. | 2026-10-11 |
| How are concurrent edits from two devices resolved? | Last write wins; no version check, no rejection. | 2026-10-11 |
| Are avatars moderated or scanned? | No. Conscious risk accepted. | 2026-10-11 |
| Is the avatar publicly readable? | No; served only through a short-lived signed address. | 2026-10-11 |
| Lifetime of the signed upload permission? | 5 minutes, configured in `config.yaml`. | 2026-10-11 |
| Route names and payloads? | Chosen by engineering in `design.md` and OpenAPI. | 2026-10-11 |
| Old-image deletion fails after the new avatar is accepted? | Accept the orphaned file: log, still succeed, no retry (`TD-002`). | 2026-10-11 |
| Response-time target (NFR-001)? | None at this stage, a somewhat slower response is accepted. | 2026-10-11 |
| Storage for dev and tests? | A storage emulator; real-bucket verification at deploy (`TD-001`). | 2026-10-11 |
| Rate limit on profile changes? | 10 per minute per account (first set at 5, raised), counting every profile and avatar operation including reads. | 2026-10-11 |
| Where do uploads land and when are they cleaned? | Temporary folder first, copied to the permanent folder on confirm; temporary folder not cleaned for now (deferred). | 2026-10-11 |
| Which temples can be chosen? | Only those in the catalogue's list of selectable temples. | 2026-10-11 |
| Does the server inspect uploaded file content? | No; only declared type and size are checked. | 2026-10-11 |
| Lifetime of the avatar read address? | 15 minutes, configured in `config.yaml`. | 2026-10-11 |
| Does the 255 limit count characters or bytes? | Characters. | 2026-10-11 |
| Is a request containing email or phone rejected whole or partly applied? | Rejected as a whole. | 2026-10-11 |

### Appendix B — Normalization findings

| Finding | Where | Handling |
|---|---|---|
| Stale term: old US-004 calls the ordination place free text, but it is a temple reference since onboarding v7.0. | `onboarding/requirements.md` US-004 | This SRS uses "ordination temple". The onboarding section should be pointed here after approval. |
| Gap: the stored user has no avatar column and no object-storage code is wired. | `migrations/0001`, `CLAUDE.md` | Recorded as a dependency (section 2.7); avatar requirements are blocked on it. |
| Mismatch: old US-004 wanted an "Verify email" button on the profile, but no email-verification flow exists. | `onboarding/requirements.md` line 127 | Out of scope; the profile only reports verification status. |
| Ambiguity: old AC5 "leave blank = clear" did not separate omitted from cleared. | `onboarding/requirements.md` US-004 AC5 | Resolved by FR-004. |
| Gap: dharma name has no explicit length validation at registration (a known tracked gap). | `docs/HANDOFF.md` | FR-002 requires a validation error, never a server error; flagged for the same fix at registration. |
