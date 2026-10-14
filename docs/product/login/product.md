# Product Spec — Sign-in, session and password reset (Đăng nhập)

| Field | Value |
|---|---|
| Document type | Product spec — Software Requirements Specification, IEEE 29148-2018 structure |
| Feature | Sign-in, session, sign-out, forgot password, guest mode (`login`) |
| Feature code | `LGN` — IDs are `BR-LGN-NN`, `FR-LGN-NNN`, `NFR-LGN-NN` |
| Product | InnerLamp |
| Platforms in scope | Backend, Website, iOS, Android |
| Version | 1.0 |
| Last updated | 2026-10-14 |
| Status | draft — 2 small open items (Appendix A) |
| Owner / author | Product owner: anh Đức · written by Claude (`ba-writer`, retroactive mode) |
| API contract | `docs/specs/api/openapi.yaml` (owned by the backend; this spec never restates it) |
| Architecture decisions | ADR-0003 (accounts are identified by email and verified by an email code; SMS removed) — Accepted 2026-10-14 |
| Related product specs | `onboarding` (`docs/product/onboarding/product.md`, code `ONB`) — first run and registration; its password rule (BR-ONB-06) and code policy (BR-ONB-09) are reused here |
| Downstream | Backend: `docs/specs/features/login/` (absorbs `password-reset/`) · Website: `Docs/specs/features/login/` · iOS: `Docs/Specs/{Login,ForgotPassword}/` · Android: `Docs/Specs/Login/` (new) |

> See `changelog.md` in this folder for the version history. Platform specs cite this file as
> `docs/product/login/product.md@<version>` plus the IDs they implement.

---

## 1. Introduction

### 1.1 Purpose

This document is the single source of product truth for signing in, staying signed in, signing out,
resetting a forgotten password and browsing as a guest on every InnerLamp platform. Sign-in moves from
phone number to email (ADR-0003).

### 1.2 Scope

**In scope:**
- Signing in with email and password; the lock after repeated wrong passwords.
- Staying signed in on each device, and what the user sees when the session has ended.
- Several devices at once; signing out of one device.
- Forgot password: a code to the email, a new password, every device signed out.
- Guest mode ("Vào xem trước").

**Out of scope:**
- Registration and the first-run intro — `onboarding`.
- Signing in with a one-time code (SMS or email): removed; the "Gửi mã một lần qua SMS" button goes
  away on every client.
- Signing in with Google or Apple — later, separate ADR.
- Signing out of all other devices on demand, changing the password while signed in, changing email or
  phone, two-factor sign-in, deleting the account — later.
- Back-office (admin) sign-in — a different product surface (ADR-0003 rule 9).

### 1.3 Product overview

InnerLamp (Tâm Đăng) is a Vietnamese Buddhist practice app on the website, iOS and Android. An account
keeps the user's practice journey on every device; this feature lets the user get into that account
and stay in it safely, and lets visitors look around before they have one.

### 1.4 Glossary

| Term | Definition | Not to be confused with |
|---|---|---|
| Đăng nhập (sign-in) | Proving to InnerLamp who you are with email and password | Đăng ký (registration) |
| Phiên đăng nhập (session) | The signed-in state of one device; each device has its own | The account |
| Đăng xuất (sign-out) | Ending the session of the current device | Deleting the account |
| Khoá đăng nhập (sign-in lock) | A temporary refusal after too many wrong passwords for one email | A banned account |
| Quên mật khẩu (forgot password) | Setting a new password with a code sent to the account's email | Changing the password while signed in (later) |
| Mã xác minh (verification code) | A one-time 6-digit code InnerLamp emails (`onboarding` glossary) | The password |
| Khách (guest) | Someone using InnerLamp without signing in | A signed-in user |

### 1.5 References

- ADR-0003 `docs/adr/0003-email-otp-account-verification-without-sms.md` (Accepted 2026-10-14).
- Design prototype description `Mo-ta-tinh-nang.md` lines 48–96 (iOS copy); still phone-based,
  superseded by ADR-0003.
- Prior documents (to be deleted after the platform specs are approved): backend
  `docs/specs/features/{login,password-reset}/` (incl. `login/SRS.md`, sign-out),
  `docs/knowledge/features/login.md`; website `Docs/specs/features/login/`; iOS
  `Docs/Specs/{Login,ForgotPassword}/`.
- Inventory of what each platform does today: `/mnt/project-files/reports/product/auth/2026-10-14-kiem-ke-client-auth.md`.
- Questions and answers: `/mnt/project-files/reports/product/auth/2026-10-14-cau-hoi-product-spec.md`.
- Platform folder names: Backend `login`, `password-reset` · Website `login` · iOS `Login`, `ForgotPassword` · Android `login` (code only).

---

## 2. Overall description

### 2.1 Product perspective

A replacement. Sign-in by phone and password exists on the backend, website and iOS; Android shows
the screen without a backend connection. Forgot password exists only as an unfinished iOS screen.
This version signs in by email and adds password reset by email code.

Touches: `onboarding` (creates the account; launch routing), `profile` (sign-out button), every
feature that needs an account (guest mode prompts), `genealogy` (unchanged; still matched on phone).

External systems by role: an email-sending service for the reset code (mocked until a sending domain
exists, ADR-0003 rule 7).

### 2.2 Product functions

- Sign in, wrong credentials, sign-in lock (`FR-LGN-001` … `FR-LGN-003`)
- Stay signed in, session ended, several devices, sign-out (`FR-LGN-004` … `FR-LGN-007`)
- Forgot password (`FR-LGN-008`, `FR-LGN-009`)
- Guest mode (`FR-LGN-010`)

### 2.3 User classes and characteristics

| Actor | Description | Permissions |
|---|---|---|
| Guest | Not signed in | Public content only; account features ask to sign in |
| Signed-in user | Has a session on this device | Everything the account allows |
| User who forgot the password | Has an account, cannot sign in | Requests a code, sets a new password |

### 2.4 Operating environment

| Platform | In scope | Notes |
|---|---|---|
| Backend | yes | Checks credentials, keeps sessions, locks, sends reset codes |
| Website | yes | Sign-in page, forgot-password pages (new), guest browsing |
| iOS | yes | Sign-in, forgot password (to finish), guest mode |
| Android | yes | Same as iOS; today UI only |

Network expectation: online only for sign-in and reset. A signed-in app opened offline stays signed in
and shows what it can until the network returns.

### 2.5 Constraints

- No SMS anywhere (ADR-0003).
- Nothing on screen or in timing may tell a visitor whether an email has an account (ADR-0003 rule 8).
- The backend is the authority for every rule (owner, 2026-10-14).

### 2.6 User documentation

N/A.

### 2.7 Assumptions and dependencies

- **Assumptions:** accounts were created through `onboarding`, so every account has a verified email.
- **Dependencies:** `onboarding` BR-ONB-05 (email form), BR-ONB-06 (password rule), BR-ONB-09 (code
  policy); an email-sending service and a sending domain for production (ADR-0003 rule 7).

---

## 3. Specific requirements

### 3.1 External interface requirements

#### 3.1.1 User interfaces

| View | Platforms | Must show |
|---|---|---|
| Sign-in | all clients | Email, password (can be revealed), "Đăng nhập", "Quên mật khẩu?", link to registration, "Vào xem trước, chưa đăng nhập" |
| Forgot password — request | all clients | Email field, "Gửi mã", "Trở lại đăng nhập" |
| Forgot password — new password | all clients | Where the code went, 6-digit code, new password, time left, "Gửi lại mã" after the wait |
| Session ended | all clients | On the sign-in view: "Phiên đăng nhập đã hết, vui lòng đăng nhập lại." |
| Sign-out | all clients | A confirmation before signing out (from the profile) |
| Sign-in prompt for guests | all clients | Explains that the feature needs an account; offers sign-in and registration |

#### 3.1.2 Hardware interfaces

N/A.

#### 3.1.3 Software interfaces

The product needs the backend to: check email and password and open a session for this device; keep
the session alive while it is used; end the session of one device; send a reset code to an email;
check a reset code and set a new password, ending every session of the account. Contract handoff:
Appendix B.

#### 3.1.4 Communication interfaces

Email: the reset code, in Vietnamese. When the email has no account, nothing is sent (the screen does
not change).

### 3.2 Business rules

**BR-LGN-01 — Sign in with email and password**
- **Rule:** The user signs in with the account's email (compared as in BR-ONB-05: trimmed, case
  ignored) and the password. Any wrong combination — unknown email, wrong password, a registration
  that was never verified — gives the same message, "Email hoặc mật khẩu không đúng", in the same
  time. On the sign-in view the password only has to be non-empty.
- **Examples:** " An@Mail.com " + right password → signed in. Unknown email → "Email hoặc mật khẩu
  không đúng". Right email, wrong password → the same message.
- **Computational:** no.
- **Applies to:** Backend, Website, iOS, Android.
- **Source:** ADR-0003 rule 1; current backend AC2 (uniform message), now by email; anh Đức
  2026-10-14 (password non-empty at sign-in). Android requires 8 characters at sign-in (drift).

**BR-LGN-02 — Sign-in lock**
- **Rule:** 5 wrong attempts in a row for one email, each within 15 minutes of the previous one, lock
  sign-in for that email for 15 minutes, even with the right password. The lock applies to emails with
  no account too. The user is told when they can try again, using the time the backend gives.
- **Examples:** 5th wrong password at 10:00 → locked until 10:15; right password at 10:05 → still
  locked, "thử lại sau 10:15". 4 wrong, then 20 minutes pass, then 1 wrong → count restarts at 1.
- **Computational:** no — enforced by the backend only; verified by backend tests.
- **Applies to:** Backend; clients show the time.
- **Source:** current backend (per phone, now per email); anh Đức 2026-10-14 (answer 5). The website
  hard-codes "15 phút" (drift).

**BR-LGN-03 — Sessions**
- **Rule:** Each device has its own session. Signing in on a new device does not sign out the others.
  Signing in again on the same device replaces that device's previous session. A session stays alive
  as long as the device uses it at least once every 7 days; after 7 days unused it ends. A session also
  ends when the user signs out on that device, when the password is reset (BR-LGN-05), or when
  InnerLamp detects that its credentials were copied and reused.
- **Examples:** sign in on phone and website → both signed in. App unused for 8 days → next launch
  needs sign-in. Used every 3 days for a month → stays signed in.
- **Computational:** no.
- **Applies to:** Backend; clients follow.
- **Source:** current backend (login AC3, AC4, AC8–AC11, sliding 7 days).

**BR-LGN-04 — Session ended is explained**
- **Rule:** When a client finds that its session has ended without the user signing out on that
  device, it opens the sign-in view with "Phiên đăng nhập đã hết, vui lòng đăng nhập lại." It never
  drops the user to guest mode or to sign-in silently.
- **Computational:** no.
- **Applies to:** Website, iOS, Android.
- **Source:** anh Đức, 2026-10-14 ("có báo"). iOS and website are silent today (drift).

**BR-LGN-05 — Password reset**
- **Rule:** The user enters the account's email and receives a 6-digit code following the code policy
  of BR-ONB-09 (valid 10 minutes, a backend setting; resend after 60 seconds; at most 4 codes then 30
  minutes pause; 3 wrong entries lock the code for 30 minutes). The screen and its timing are the same
  whether or not the email has an account; no email is sent when it has none. With the right code the
  user sets a new password that follows BR-ONB-06. Then every session of the account, on every device,
  ends, and the user signs in again with the new password; they are not signed in automatically.
- **Examples:** right code + "matkhaumoi" → password changed, phone and website both signed out, the
  sign-in view opens. Email without an account → "Nếu email này có tài khoản, mã đã được gửi" and no
  email. New password of 7 characters → refused.
- **Computational:** no (the password check reuses `onboarding/rules/password.json`).
- **Applies to:** Backend, Website, iOS, Android.
- **Source:** anh Đức, 2026-10-14 (answer 7); ADR-0003 rule 5; pentest report item 19.

**BR-LGN-06 — Guest mode**
- **Rule:** "Vào xem trước, chưa đăng nhập" lets a visitor use InnerLamp without an account: public
  content only (library, calendar, temples and similar). Anything that belongs to an account (memorial
  book, family tree, practice records, profile, notifications) asks the guest to sign in or register
  instead of opening. A guest is never treated as signed in.
- **Examples:** guest opens the lunar calendar → it opens. Guest opens the memorial book → sign-in
  prompt. Android today opens Home as if signed in (drift).
- **Computational:** no.
- **Applies to:** Website, iOS, Android (the backend already refuses account data without a session).
- **Source:** anh Đức, 2026-10-14 (answer 6: keep guest mode).

**BR-LGN-07 — Sign-out**
- **Rule:** Sign-out asks for confirmation, ends the session of this device only, and removes this
  account's data kept on this device. Other devices stay signed in. Device-level settings that are not
  tied to the account (such as "intro seen", BR-ONB-01) stay.
  > ⚠️ Not confirmed: when the device cannot reach InnerLamp while signing out, does sign-out still complete on the device? Recommended: yes, the device forgets the session at once and the server session simply expires — Website, iOS, Android.
- **Computational:** no.
- **Applies to:** Backend, Website, iOS, Android.
- **Source:** backend `login/SRS.md` FR-001…FR-005; iOS AS-Profile-003.

### 3.3 Functional requirements

#### Sign-in

**FR-LGN-001 — Sign in**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the user submits a registered email and its password THE SYSTEM SHALL sign the
  user in on this device and open Home (apps) or the home page (website).
- **Business rules:** BR-LGN-01, BR-LGN-03
- **Verification method:** automated tests per platform; backend end-to-end test
- **Source:** current sign-in, now by email
- **Acceptance criteria:**
  ```gherkin
  Scenario: right email and password
    Given an account with email "an@mail.com"
    When the user signs in with " An@Mail.com " and the right password
    Then the user is signed in
    And Home opens

  Scenario: empty password
    Given the password field is empty
    Then the user cannot submit
  ```

**FR-LGN-002 — Wrong email or password**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the email is unknown or the password is wrong THE SYSTEM SHALL show "Email hoặc
  mật khẩu không đúng" and keep the email in the field.
- **Business rules:** BR-LGN-01
- **Verification method:** automated tests
- **Source:** current behavior
- **Acceptance criteria:**
  ```gherkin
  Scenario: unknown email and wrong password look the same
    When the user signs in with an email that has no account
    And later with a registered email and a wrong password
    Then both attempts show "Email hoặc mật khẩu không đúng"
  ```

**FR-LGN-003 — Sign-in lock**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN sign-in is locked for an email THE SYSTEM SHALL refuse every attempt for that
  email and tell the user the time they can try again.
- **Business rules:** BR-LGN-02
- **Verification method:** automated tests
- **Source:** current backend; anh Đức 2026-10-14
- **Acceptance criteria:**
  ```gherkin
  Scenario: fifth wrong password locks sign-in
    Given 4 wrong passwords for "an@mail.com" in the last 15 minutes
    When the user enters a 5th wrong password at 10:00
    Then the user is told sign-in is locked until 10:15

  Scenario: right password during the lock
    Given sign-in for "an@mail.com" is locked until 10:15
    When the user enters the right password at 10:05
    Then the user is still told to try again after 10:15
  ```

#### Session

**FR-LGN-004 — Stay signed in**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHILE the session of this device is alive THE SYSTEM SHALL keep the user signed in
  across app restarts and page reloads without asking for the password.
- **Business rules:** BR-LGN-03
- **Verification method:** automated tests; manual QA
- **Source:** current behavior on iOS and website
- **Acceptance criteria:**
  ```gherkin
  Scenario: reopen the app the next day
    Given the user signed in yesterday on this phone
    When the user opens the app
    Then Home opens without asking for the password

  Scenario: two devices
    Given the user is signed in on the website
    When the user signs in on the phone
    Then the user is still signed in on the website
  ```

**FR-LGN-005 — Session ended**
- **Priority:** Must
- **Platforms:** Website, iOS, Android
- **Description:** WHEN a client finds that its session has ended (unused for more than 7 days, password
  reset on another device, or credentials reused) THE SYSTEM SHALL open the sign-in view with
  "Phiên đăng nhập đã hết, vui lòng đăng nhập lại."
- **Business rules:** BR-LGN-03, BR-LGN-04
- **Verification method:** automated tests; manual QA
- **Source:** anh Đức, 2026-10-14
- **Acceptance criteria:**
  ```gherkin
  Scenario: back after 8 days
    Given the user last used the app 8 days ago
    When the user opens it
    Then the sign-in view opens with "Phiên đăng nhập đã hết, vui lòng đăng nhập lại."

  Scenario: password reset elsewhere
    Given the user is signed in on the phone
    And the password is reset from the website
    When the phone next talks to InnerLamp
    Then the phone shows the sign-in view with the session-ended message
  ```

**FR-LGN-006 — Sign out**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the user confirms sign-out THE SYSTEM SHALL end this device's session, remove
  this account's data from the device and show the sign-in view (apps) or the home page as a guest
  (website).
- **Business rules:** BR-LGN-07
- **Verification method:** automated tests
- **Source:** backend sign-out SRS; iOS and website today
- **Acceptance criteria:**
  ```gherkin
  Scenario: sign out of the phone only
    Given the user is signed in on the phone and on the website
    When the user signs out on the phone and confirms
    Then the phone shows the sign-in view
    And the website is still signed in

  Scenario: cancel sign-out
    When the user chooses sign-out and then cancels
    Then the user stays signed in
  ```

**FR-LGN-007 — Signed-in visitor on the sign-in page (website)**
- **Priority:** Should
- **Platforms:** Website
- **Description:** WHEN a signed-in user opens the sign-in page THE SYSTEM SHALL take them to the home
  page instead.
  > ⚠️ Not confirmed: after signing in on the website, should the user return to the page they were on before? Recommended: yes — Website.
- **Business rules:** BR-LGN-03
- **Verification method:** automated tests
- **Source:** website open question (`Docs/specs/features/login/`)
- **Acceptance criteria:**
  ```gherkin
  Scenario: already signed in
    Given the user is signed in on the website
    When the user opens the sign-in page
    Then the home page opens
  ```

#### Forgot password

**FR-LGN-008 — Ask for a reset code**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the user enters an email on the forgot-password view and submits THE SYSTEM SHALL
  show the new-password view with "Nếu email này có tài khoản, mã đã được gửi", and email a code only
  when the email has an account.
- **Business rules:** BR-LGN-05, BR-ONB-09
- **Verification method:** automated tests; backend end-to-end test
- **Source:** anh Đức, 2026-10-14; ADR-0003 rule 5
- **Acceptance criteria:**
  ```gherkin
  Scenario: registered email
    Given an account with email "an@mail.com"
    When the user asks for a reset code for "an@mail.com"
    Then the new-password view opens
    And a 6-digit code arrives at "an@mail.com"

  Scenario: email with no account looks the same
    When the user asks for a reset code for "khong-co@mail.com"
    Then the same new-password view and message appear
    And no email is sent

  Scenario: resend after the wait
    Given 60 seconds have passed since the code was sent
    When the user chooses "Gửi lại mã"
    Then a new code replaces the old one
  ```

**FR-LGN-009 — Set a new password**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the user enters the right code and a valid new password THE SYSTEM SHALL change
  the password, end every session of the account and show the sign-in view with "Đã đổi mật khẩu, vui
  lòng đăng nhập lại."
- **Business rules:** BR-LGN-05, BR-ONB-06, BR-ONB-09
- **Verification method:** shared password vectors; automated tests; backend end-to-end test
- **Source:** anh Đức, 2026-10-14
- **Acceptance criteria:**
  ```gherkin
  Scenario: reset and sign in again
    Given the user received a reset code
    When the user enters it with the new password "matkhaumoi"
    Then the sign-in view opens with "Đã đổi mật khẩu, vui lòng đăng nhập lại."
    And signing in with "matkhaumoi" works
    And the old password no longer works

  Scenario: every device is signed out
    Given the user is signed in on the website
    When the password is reset from the phone
    Then the website shows the session-ended message on its next request

  Scenario: wrong code
    When the user enters a wrong code
    Then the password does not change and the user is told the code is wrong

  Scenario: new password too short
    When the user enters the right code and a 7-character password
    Then the user is told the password needs at least 8 characters
  ```

#### Guest

**FR-LGN-010 — Browse as a guest**
- **Priority:** Should
- **Platforms:** Website, iOS, Android
- **Description:** WHEN the user chooses "Vào xem trước, chưa đăng nhập" THE SYSTEM SHALL open Home as a
  guest; WHEN a guest opens something that belongs to an account THE SYSTEM SHALL offer sign-in or
  registration instead.
- **Business rules:** BR-LGN-06
- **Verification method:** automated UI tests; manual QA
- **Source:** anh Đức, 2026-10-14
- **Acceptance criteria:**
  ```gherkin
  Scenario: guest reads public content
    Given the user chose "Vào xem trước, chưa đăng nhập"
    When the user opens the lunar calendar
    Then it opens

  Scenario: guest reaches an account feature
    Given the user is a guest
    When the user opens the memorial book
    Then the user is asked to sign in or register
  ```

### 3.4 Non-functional requirements

- **NFR-LGN-01 — No account enumeration:** sign-in, the lock and forgot password never reveal whether an
  email has an account, by message or by timing. Platforms: Backend, all clients.
- **NFR-LGN-02 — Abuse limits:** sign-in, session renewal and reset requests are limited per device
  address and per email (ADR-0003 rule 8); a limit shows "try again later", never a technical error.
  Platforms: Backend, all clients.
- **NFR-LGN-03 — Credentials on the device:** a device keeps its session only in the platform's protected
  storage, never readable by page scripts on the website and never in backups of the apps' plain files.
  Platforms: Website, iOS, Android.

### 3.5 Data requirements

| Data item | Meaning | Owned by | Retention / deletion | Private or public |
|---|---|---|---|---|
| Session | Signed-in state of one device | Server + that device | Ends per BR-LGN-03 | Secret |
| Device description | Which kind of device a session belongs to (platform, model) | Account (server) | With the session history | Private |
| Wrong-attempt count | Counts towards the sign-in lock | Server, per email | 15 minutes after the last wrong attempt | Internal |
| Reset code | One-time code to set a new password | Server (never readable) | Used, replaced, or expired | Secret |

### 3.6 Platform variance

| ID | Platforms | Difference | Reason |
|---|---|---|---|
| FR-LGN-001, FR-LGN-006 | Website | Lands on the website home page; after sign-out stays on the site as a guest | No Home tab or sign-in-first launch on the web |
| FR-LGN-007 | Website only | Redirect of signed-in visitors away from the sign-in page | Only the website has addressable pages |

### 3.7 Software system attributes

- **Reliability:** a failed sign-in request keeps the typed email.
- **Availability:** when the backend is down nobody can sign in or reset; signed-in apps keep their
  session and retry later. When the email service is down, reset cannot finish.
- **Security & privacy:** NFR-LGN-01…03; ADR-0003 rules 5, 6 and 8.
- **Maintainability / portability:** N/A at product level.

---

## 4. Verification & traceability

### 4.1 Verification approach

Automated tests per platform for each scenario; backend end-to-end tests against a real database for
sign-in, the lock, sessions, sign-out and reset; the new-password check runs
`docs/product/onboarding/rules/password.json`.

### 4.2 Traceability matrix

| ID | Backend | Website | iOS | Android | Note |
|---|---|---|---|---|---|
| BR-LGN-01 | drift | drift | drift | drift | Sign-in by phone today; Android requires 8 characters |
| BR-LGN-02 | drift | drift | done | planned | Lock per phone today; website hard-codes 15 minutes |
| BR-LGN-03 | done | done | done | planned | |
| BR-LGN-04 | n/a | drift | drift | planned | Silent today |
| BR-LGN-05 | planned | planned | drift | planned | Not built; iOS screen is a phone/SMS stub |
| BR-LGN-06 | n/a | done | done | drift | Android enters Home as signed in |
| BR-LGN-07 | done | done | done | planned | Offline sign-out open |
| FR-LGN-001 | drift | drift | drift | planned | 2 scenarios |
| FR-LGN-002 | drift | drift | drift | planned | 1 scenario; message says "Số điện thoại" today |
| FR-LGN-003 | drift | drift | done | planned | 2 scenarios |
| FR-LGN-004 | done | done | done | planned | 2 scenarios |
| FR-LGN-005 | done | drift | drift | planned | 2 scenarios |
| FR-LGN-006 | done | done | done | planned | 2 scenarios |
| FR-LGN-007 | n/a | planned | n/a | n/a | 1 scenario |
| FR-LGN-008 | planned | planned | drift | planned | 3 scenarios |
| FR-LGN-009 | planned | planned | planned | planned | 4 scenarios |
| FR-LGN-010 | n/a | done | done | drift | 2 scenarios |
| NFR-LGN-01 | done | n/a | n/a | n/a | Uniform message and timing exist for phone |
| NFR-LGN-02 | planned | planned | planned | planned | No rate limit today (pentest H-2) |
| NFR-LGN-03 | n/a | done | done | planned | Android keeps tokens in memory only |

### 4.3 Test vectors

| File | Rule | Cases |
|---|---|---|
| `docs/product/onboarding/rules/password.json` | BR-LGN-05 (new password, via BR-ONB-06) | 10 |

---

## 5. Appendices

### Appendix A — Open questions

#### Open (unconfirmed)

> ⚠️ Not confirmed: when the device cannot reach InnerLamp while signing out, does sign-out still complete on the device? Recommended: yes — Website, iOS, Android (BR-LGN-07).

> ⚠️ Not confirmed: after signing in on the website, does the user return to the page they were on? Recommended: yes — Website (FR-LGN-007).

#### Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| Sign-in identifier? | Email + password (ADR-0003) | 2026-10-14 |
| One-time-code sign-in button? | Removed | 2026-10-14 |
| Sign-in lock? | 5 wrong in a row → 15 minutes per email; show the unlock time | 2026-10-14 |
| Session ended: silent or explained? | Explained with "Phiên đăng nhập đã hết" | 2026-10-14 |
| Forgot password? | Email code, new password, every device signed out, sign in again | 2026-10-14 |
| Guest mode? | Kept; public content only | 2026-10-14 |
| Change password/email/phone, delete account, Google/Apple, sign out everywhere? | Later | 2026-10-14 |

### Appendix B — Contract handoff (backend)

1. Sign-in takes email instead of phone; the lock counts per email (case ignored); the uniform message
   changes to "email".
2. New: request a reset code by email (same answer whether or not the email has an account, same send
   limits as registration), resend, and set a new password with the code; success ends every session of
   the account.
3. The new password follows BR-ONB-06 (8–64 characters).
4. Rate limits on sign-in, session renewal and reset answer with a "try later" status and the time
   (NFR-LGN-02).
5. Sign-in by one-time code is not offered; the reserved "login" code purpose is unused.
6. Sessions, renewal, reuse detection and sign-out are unchanged.
