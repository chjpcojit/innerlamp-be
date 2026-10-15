# Product Spec — Onboarding and registration (Giới thiệu và Đăng ký)

| Field | Value |
|---|---|
| Document type | Product spec — Software Requirements Specification, IEEE 29148-2018 structure |
| Feature | Onboarding and registration (`onboarding`) |
| Feature code | `ONB` — IDs are `BR-ONB-NN`, `FR-ONB-NNN`, `NFR-ONB-NN` |
| Product | InnerLamp |
| Platforms in scope | Backend, Website, iOS, Android |
| Version | 1.4 |
| Last updated | 2026-10-15 |
| Status | confirmed — no open items |
| Owner / author | Product owner: anh Đức · written by Claude (`ba-writer`, retroactive mode) |
| API contract | `docs/specs/api/openapi.yaml` (owned by the backend; this spec never restates it) |
| Architecture decisions | ADR-0003 (accounts are identified by email and verified by an email code; SMS removed) — Accepted 2026-10-14 |
| Related product specs | `login` (`docs/product/login/product.md`, code `LGN`) — sign-in, session, sign-out, forgot password, guest mode |
| Downstream | Backend: `docs/specs/features/onboarding/` (`spec-writer`) · Website: `Docs/specs/features/onboarding/` · iOS: `Docs/Specs/Onboarding/` · Android: `Docs/Specs/Onboarding/` |

> See `changelog.md` in this folder for the version history. Platform specs cite this file as
> `docs/product/onboarding/product.md@<version>` plus the IDs they implement.

---

## 1. Introduction

### 1.1 Purpose

This document is the single source of product truth for the first-run experience and account
registration on every InnerLamp platform. It replaces the phone-and-SMS registration described in the
backend, website and iOS specs: the account is now proven by an email code (ADR-0003).

### 1.2 Scope

**In scope:**
- The splash and the three intro screens shown the first time the mobile apps are opened, and where
  the app goes on later launches.
- Creating an account: full name, dharma name (optional), ordination temple (optional), phone number,
  email, password, acceptance of the terms.
- Verifying the email with a 6-digit code before the account exists; signing the user in right after.
- Proposing a temple that is missing from the list while registering.

**Out of scope:**
- Signing in, staying signed in, signing out, forgot password, guest mode — `login` product spec.
- Changing email or phone, changing password while signed in, deleting the account — later.
- Signing in with Google or Apple — later, separate ADR (ADR-0003 rule 9).
- Verifying the phone number — never (ADR-0003: no SMS).
- Editing the profile after registration — `profile` (legacy spec; not yet rewritten).

### 1.3 Product overview

InnerLamp (Tâm Đăng) is a Vietnamese Buddhist practice app: practice tools, a lunar calendar, a
library, temples and family features, on the website, iOS and Android. An account keeps the user's
practice journey on every device ("Hành trình tu tập sẽ được lưu lại, dù bạn đổi máy"). This feature
is the door into it: the apps introduce themselves once, then the user creates an account and proves
they own the email address they gave.

### 1.4 Glossary

| Term | Definition | Not to be confused with |
|---|---|---|
| Giới thiệu (intro screens) | The three screens shown on the first launch of the mobile apps | Splash, the brand screen shown on every launch |
| Splash | The brand screen with the wordmark and the tagline "Tự thắp đèn tâm, tự soi lối về" | Intro screens |
| Đăng ký (registration) | Creating an InnerLamp account | Ghi danh (event registration), a temple-event feature |
| Mã xác minh (verification code) | A one-time 6-digit code InnerLamp emails to prove the user owns the address | The password |
| Đăng ký dở (pending registration) | What the user filled in, kept until the email is verified or the time runs out; not an account | An account |
| Họ và tên (full name) | The user's legal or everyday name | Pháp danh |
| Pháp danh (dharma name) | The Buddhist name received at refuge (quy y); not everyone has one | Họ và tên |
| Nơi xuất gia / đạo tràng (ordination temple) | The temple where the user took refuge or ordained, picked from InnerLamp's temple list | A temple the user merely visits |
| Cư sĩ tu tại gia (lay practitioner) | The choice meaning "no ordination temple" | — |
| Đề xuất chùa (temple proposal) | A request to add a temple that is not in the list, reviewed by InnerLamp | A temple in the list |

### 1.5 References

- ADR-0003 `docs/adr/0003-email-otp-account-verification-without-sms.md` (Accepted 2026-10-14).
- Design prototype description `Mo-ta-tinh-nang.md` lines 48–96 (copy in `innerlamp-iOS/Docs/innerlamp-project/`; the Android copy is identical). It still shows phone + SMS; superseded by ADR-0003.
- Prior documents (rewritten in place or deleted on 2026-10-15, once the four platform specs were approved; old versions in git history): backend
  `docs/specs/features/{onboarding,otp-mechanism}/`, `docs/assumptions/onboarding/A-001`,
  `docs/knowledge/features/register.md`; website `Docs/specs/features/register/`; iOS
  `Docs/Specs/{SignUp,Otp,Onboarding,Splash}/`, `Docs/Assumptions/{SignUp,Splash}/`.
- Inventory of what each platform does today: `/mnt/project-files/reports/product/auth/2026-10-14-kiem-ke-client-auth.md` (iOS 24714c4, website 31eed2a, Android f0b3d02).
- Questions and answers: `/mnt/project-files/reports/product/auth/2026-10-14-cau-hoi-product-spec.md`.
- Platform spec folders: Backend `onboarding` · Website `onboarding` · iOS `Onboarding` · Android `Onboarding`. Code folders keep their own names (website `register`; iOS `Splash`, `Onboarding`, `SignUp`, `Otp`, `AddTemple`; Android `splash`, `onboarding`, `signup`).

---

## 2. Overall description

### 2.1 Product perspective

A replacement. Registration exists on the backend, website and iOS, verified by a code "sent by SMS"
to the phone (a fixed code in development); Android has the screens with no backend connection. This
version moves verification to email and makes the phone an unverified contact detail (ADR-0003).

Touches: `login` (the account created here signs in there), `temple` (the list of temples to pick
from), `temple-suggestion` (a proposed temple), `profile` (shows and edits these fields later),
`genealogy` (invitations are matched on the phone number, ADR-0003 rule 4).

External systems by role: an email-sending service delivers the verification code (mocked until a
sending domain exists, ADR-0003 rule 7). No SMS service.

### 2.2 Product functions

- Intro screens on first launch and launch routing on the mobile apps (`FR-ONB-001`, `FR-ONB-002`)
- Registration form, temple choice and temple proposal (`FR-ONB-003` … `FR-ONB-006`)
- Email verification code and account creation (`FR-ONB-007` … `FR-ONB-010`)

### 2.3 User classes and characteristics

| Actor | Description | Permissions |
|---|---|---|
| New visitor | Has never opened the app, or has no account | Sees the intro screens (apps), registers, or continues as a guest (`login`) |
| Registering user | Has submitted the form, has not entered the code yet | Enters or re-requests the code, goes back to fix the form; has no account yet |
| Signed-in user | Finished registration | Everything an account allows |

### 2.4 Operating environment

| Platform | In scope | Notes |
|---|---|---|
| Backend | yes | Validates every rule, keeps the pending registration, sends the code, creates the account |
| Website | yes | Registration and code entry; no splash, no intro screens (§3.6) |
| iOS | yes | Splash, intro, registration, code entry |
| Android | yes | Same as iOS; today UI only, no backend connection |

Network expectation: online only. Registration cannot start or finish offline; the form keeps what
the user typed while the request fails.

### 2.5 Constraints

- No SMS of any kind, and no phone verification (owner, 2026-10-14; ADR-0003).
- The phone number stays required and unique across InnerLamp although it is never verified
  (ADR-0003 rule 3). Accepted risk: someone can register another person's number first.
- The backend is the authority for every validation rule; clients may check earlier for a better
  experience but must not be stricter or looser (owner, 2026-10-14: "thống nhất ở BE").
- Terms of use and privacy policy must be accepted before an account is created.

### 2.6 User documentation

N/A — the screens explain themselves; the terms and privacy policy are separate content.

### 2.7 Assumptions and dependencies

- **Assumptions:** the user can read their email on the same or another device while registering.
- **Dependencies:** an email-sending service and a sending domain for production (ADR-0003 rule 7;
  none today, so the code is mocked in development); the temple list (`temple`); the temple proposal
  review (`temple-suggestion`).

---

## 3. Specific requirements

### 3.1 External interface requirements

#### 3.1.1 User interfaces

| View | Platforms | Must show |
|---|---|---|
| Splash | iOS, Android | Wordmark, tagline "Tự thắp đèn tâm, tự soi lối về"; advances on its own |
| Intro step 1 — "Thắp ngọn đèn trong tâm" | iOS, Android | Welcome text; "Bắt đầu"; "Tôi đã có tài khoản"; "Bỏ qua"; progress |
| Intro step 2 — "Bạn mong cầu điều gì?" | iOS, Android | 6 choices, several may be selected; "Tiếp tục"; "Quay lại"; "Bỏ qua" |
| Intro step 3 — "Chuông tỉnh thức" | iOS, Android | 3 bell times (05:30, 12:00, 21:30), each on/off; "Tạo tài khoản"; "Quay lại"; "Bỏ qua" |
| Registration form — "Tạo tài khoản" | all clients | Fields in this order: full name, dharma name (with "Gợi ý"), ordination temple, phone (with country), email, password (with strength hint), terms acceptance; "Đăng ký"; link to sign in |
| Temple proposal | all clients | Temple name; province and ward picked from the administrative-unit list; street line (optional) |
| Verification code | all clients | The email address the code went to; 6-digit entry; time left; "Gửi lại mã" after the wait; "Đổi email" back to the form |

"Bỏ qua" (skip) is shown on all three intro steps, as the iOS and Android apps already do.

The registration form's submit button reads "Đăng ký" (register) on every client, also while the
code is being sent. It is not "Gửi mã xác thực" (send verification code), the label all three
clients use today.

The email field explains that it is used to sign in and to receive the verification code. The old
hint "Để nhận biên nhận cúng dường" (for donation receipts only) is no longer true.

#### 3.1.2 Hardware interfaces

N/A.

#### 3.1.3 Software interfaces

The product needs the backend to: check a registration and hold it while the email is verified; send
a verification code to an email address and send it again on request; verify a code and create the
account, signing the user in; give the list of temples to choose from; accept a temple proposal from
a new account. Contract handoff: Appendix B.

#### 3.1.4 Communication interfaces

Email. InnerLamp sends (a) the verification code, and (b) when someone registers with an email that
already has an account, a notice to that address instead of a code (BR-ONB-08). Both emails are in
Vietnamese and never contain the password.

### 3.2 Business rules

**BR-ONB-01 — Intro screens once per installation**
- **Rule:** The intro screens are shown on the first launch after the app is installed. Reaching their
  end ("Tạo tài khoản" or "Tôi đã có tài khoản") records on the device that the intro has been seen;
  from then on the intro is not shown again on that installation. "Bỏ qua" (skip) counts as having seen
  it too. The record lives only on the device and is not tied to an account; reinstalling the app shows
  the intro again.
- **Examples:** first launch → intro. Intro finished, app reopened → no intro. Intro skipped, app
  reopened → no intro. App reinstalled → intro.
- **Computational:** no.
- **Applies to:** iOS, Android.
- **Source:** anh Đức, 2026-10-14 ("ghi 1 cờ nhớ ở client là đã xem"; skip counts: "ok").

**BR-ONB-02 — Launch routing on the apps**
- **Rule:** After the splash: a signed-in user goes to Home; otherwise, if the intro has not been seen
  (BR-ONB-01) the intro opens; otherwise the **sign-in** screen opens, with a link to registration.
- **Examples:** first install → intro. Intro seen, signed out → sign-in. Signed in → Home. Session
  ended while away → sign-in with the message of `login` FR-LGN-005.
- **Computational:** no.
- **Applies to:** iOS, Android.
- **Source:** anh Đức, 2026-10-14. Replaces iOS "→ registration" and Android "→ intro".

**BR-ONB-03 — Intro choices stay on the device**
- **Rule:** What the user picks on intro steps 2 and 3 is kept on that device only and is never sent to
  the account.
- **Computational:** no.
- **Applies to:** iOS, Android.
- **Source:** anh Đức, 2026-10-14 (answer 2).

**BR-ONB-04 — Phone number**
- **Rule:** Required. The user picks a country from the shared list below and types the national
  number; spaces, dots and dashes are ignored and one leading 0 is dropped. The remaining digits must
  be exactly the country's length. The number is stored with its country code and must be unique
  across InnerLamp. It is never verified.

  | Country | Code | Digits | | Country | Code | Digits |
  |---|---|---|---|---|---|---|
  | Việt Nam (default) | +84 | 9 | | Malaysia | +60 | 9 |
  | Hoa Kỳ | +1 | 10 | | Đài Loan | +886 | 9 |
  | Nhật Bản | +81 | 10 | | Úc | +61 | 9 |
  | Hàn Quốc | +82 | 9 | | Đức | +49 | 10 |
  | Thái Lan | +66 | 9 | | Pháp | +33 | 9 |
  | Singapore | +65 | 8 | | Canada | +1 | 10 |
- **Examples:** VN "0912 345 678" → +84912345678. VN "91234567" → invalid (8 digits). SG "8123 4567"
  → +6581234567. A number already used by another account → registration refused (FR-ONB-005).
- **Computational:** yes → `rules/phone-number.json`.
- **Applies to:** Backend, Website, iOS, Android.
- **Source:** anh Đức, 2026-10-14 (answer 3: per-country length as iOS, one shared list); list from
  iOS `ILCountry.swift`. Website today has China instead of Malaysia and accepts 9–10 digits for every
  country (drift).

**BR-ONB-05 — Email**
- **Rule:** Required; it is the sign-in identifier. Leading and trailing spaces are removed. Valid
  means: no spaces, exactly one "@", something before it, and after it a domain containing a dot
  followed by at least 2 characters; at most 254 characters. Upper and lower case are the same email
  ("An@Mail.com" = "an@mail.com"). One email belongs to at most one account.
- **Examples:** "an@mail.com" valid; " An@Mail.com " → "an@mail.com"; "an@mail" invalid; "an mail@x.vn"
  invalid; "a@b.c" invalid (1-character ending).
- **Computational:** yes → `rules/email.json`.
- **Applies to:** Backend, Website, iOS, Android.
- **Source:** ADR-0003 rule 1; the expression iOS and the website already use. Android today only
  checks for "@" and "." (drift).

**BR-ONB-06 — Password**
- **Rule:** Required; 8 to 64 characters, counted as characters after Unicode NFC normalisation (a
  letter with Vietnamese diacritics is one character, however it was typed), nothing trimmed. No required mix of digits, capitals or symbols. The strength hint
  shown while typing is advice only and never blocks submitting.
- **Examples:** "abcdefg" (7) invalid; "abcdefgh" (8) valid; "mậtkhẩuđẹp" (10 characters) valid;
  65 characters invalid.
- **Computational:** yes → `rules/password.json`.
- **Applies to:** Backend, Website, iOS, Android. Also used by `login` for a new password (BR-LGN-05).
- **Source:** anh Đức, 2026-10-14 (answer 3). The website counts bytes today and Android's hint needs
  12 characters for "strong" (drift / platform variance).

**BR-ONB-07 — Names**
- **Rule:** Full name is required: after removing leading and trailing spaces it has 1 to 300
  characters. Dharma name is optional: after removing spaces, empty means "no dharma name"; otherwise
  at most 255 characters. Characters typed in decomposed or precomposed Vietnamese count the same.
  "Gợi ý" fills in a suggested dharma name the user may keep or change; the field starts empty.
- **Examples:** full name "  Nguyễn Văn An " → "Nguyễn Văn An"; full name "   " invalid; dharma name
  empty → none; dharma name of 256 characters invalid.
- **Computational:** yes → `rules/names.json`.
- **Applies to:** Backend, Website, iOS, Android.
- **Source:** anh Đức, 2026-10-14 (answer 3: dharma name optional, full name required); limits from the
  backend (`docs/assumptions/onboarding/A-001`). iOS requires and pre-fills the dharma name, the website
  needs a 2-character full name, Android has no full-name field (drift).

**BR-ONB-08 — Account exists only after the email is verified**
- **Rule:** Submitting the form creates a pending registration and emails a code; no account exists
  yet. The account is created, and the user signed in, only when the right code is entered. A pending
  registration lasts 24 hours; after that the user starts again. Submitting again with the same email
  replaces the previous pending registration. If the email already belongs to an account, the screen
  behaves exactly as for a new email, and InnerLamp emails that address a notice ("this email already
  has an account — sign in or reset your password") instead of a code, so nobody can learn from the
  screen whether an email is registered (ADR-0003 rule 8). A phone number already used by an account
  is refused on the form with "Số điện thoại này đã được đăng ký", because the number must be unique
  (BR-ONB-04).
- **Examples:** new email → code emailed. Email of an existing account → same screen, notice emailed,
  no code works. Pending registration 25 hours old → "Thời gian đăng ký đã hết. Vui lòng đăng ký lại từ
  đầu."
- **Computational:** no.
- **Applies to:** Backend; clients show the outcomes.
- **Source:** anh Đức, 2026-10-14 (ADR question 7); ADR-0003 rules 2 and 8. Today the backend answers
  "email already used" openly (drift).

**BR-ONB-09 — Verification code policy**
- **Rule:**
  - The code has 6 digits and is valid for **10 minutes**; the duration is a backend setting, and clients
    show the time left that the backend reports.
  - A new code can be requested 60 seconds after the previous one. At most 4 codes are sent for one
    registration (the first plus 3 resends); after the 4th, sending is locked for 30 minutes. The lock
    on sending never stops the user from entering the last code they received.
  - 3 wrong entries lock the code for 30 minutes; the user then requests a new code.
  - A new code makes the previous one invalid. A code works once.
  - The time left, the wait before resending and any unlock time are shown to the user as given by the
    backend; clients never assume their own values.
- **Examples:** code entered at 9:59 → accepted. At 10:01 → "expired", request a new one. Third wrong
  code → locked, the screen shows when to try again.
- **Computational:** no — enforced by the backend only; verified by backend tests.
- **Applies to:** Backend; clients show the state.
- **Source:** anh Đức, 2026-10-14 (answer 4: longer validity, configurable); other values unchanged from
  the current backend (`docs/specs/features/onboarding/requirements.md` v7.5 AC7, AC11, AC13).

**BR-ONB-10 — Ordination temple and temple proposal**
- **Rule:** Optional. The user picks one temple from InnerLamp's temple list (searchable by name), or
  "Chưa xuất gia — Cư sĩ tu tại gia" for none. If the temple is not in the list, the user may propose
  it; the proposal is sent only after the account is created and is reviewed under `temple-suggestion`.
  A proposed temple is not the user's ordination temple until it is approved. A proposal needs the
  temple name and its province and ward, both **required** and chosen from InnerLamp's list of
  Vietnamese administrative units (the same list used across the product), never typed freely.
  The street line (số nhà, tên đường) is optional.
- **Computational:** no.
- **Applies to:** Backend, Website, iOS, Android.
- **Source:** backend v7.0 (temple chosen from the list); project decision 2026-10-14 ("đạo tràng" =
  ordination temple, a single field); anh Đức 2026-10-14 (province and ward from the administrative-unit
  list, required; street line optional). iOS never sends the proposal and Android takes free text (drift).

**BR-ONB-11 — Terms acceptance**
- **Rule:** The user must accept the terms of use and the privacy policy before submitting. The
  submit button stays unavailable until they do. Every client shows the **same** terms text from one
  source; how it is shown (in-app sheet, web page) is up to each platform (§3.6).
- **Computational:** no.
- **Applies to:** Website, iOS, Android.
- **Source:** current behavior on all three clients; anh Đức 2026-10-14 (one text: "có").

### 3.3 Functional requirements

#### First run (apps)

**FR-ONB-001 — Intro screens on first launch**
- **Priority:** Should
- **Platforms:** iOS, Android
- **Description:** WHEN the app is opened for the first time after installation THE SYSTEM SHALL show
  the splash and then the three intro steps; "Bỏ qua" or "Tạo tài khoản" opens registration, "Tôi đã
  có tài khoản" opens sign-in.
- **Business rules:** BR-ONB-01, BR-ONB-03
- **Verification method:** automated UI test on each app; manual QA
- **Source:** iOS and Android today (same copy); anh Đức 2026-10-14
- **Acceptance criteria:**
  ```gherkin
  Scenario: first launch shows the intro
    Given the app has just been installed
    When the user opens it
    Then the splash appears and then intro step 1

  Scenario: finishing the intro opens registration
    Given the user is on intro step 3
    When the user chooses "Tạo tài khoản"
    Then the registration form opens
    And the intro is recorded as seen on this device

  Scenario: existing user leaves the intro to sign in
    Given the user is on intro step 1
    When the user chooses "Tôi đã có tài khoản"
    Then the sign-in screen opens
    And the intro is recorded as seen on this device

  Scenario: intro choices are not sent anywhere
    Given the user selected two wishes and turned on the 05:30 bell
    When the user creates an account later
    Then those choices remain on this device only
  ```

**FR-ONB-002 — Where the app opens on later launches**
- **Priority:** Must
- **Platforms:** iOS, Android
- **Description:** WHEN the app is opened and the intro has been seen THE SYSTEM SHALL open Home for a
  signed-in user and the sign-in screen otherwise.
- **Business rules:** BR-ONB-02
- **Verification method:** automated UI test on each app
- **Source:** anh Đức, 2026-10-14
- **Acceptance criteria:**
  ```gherkin
  Scenario: signed out after the intro opens sign-in
    Given the intro was seen on this device
    And nobody is signed in
    When the user opens the app
    Then the sign-in screen opens
    And it offers a link to create an account

  Scenario: signed-in user goes home
    Given the user is signed in
    When the user opens the app
    Then Home opens

  Scenario: reinstall shows the intro again
    Given the user saw the intro, then deleted and reinstalled the app
    When the user opens it
    Then the intro opens
  ```

#### Registration form

**FR-ONB-003 — Fill in and submit the registration form**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the user submits the registration form with valid values THE SYSTEM SHALL keep
  the registration pending, email a verification code to the given address and open the verification
  code view; WHEN a value is invalid THE SYSTEM SHALL show which field is wrong and why, and keep
  everything the user typed.
- **Business rules:** BR-ONB-04, BR-ONB-05, BR-ONB-06, BR-ONB-07, BR-ONB-08, BR-ONB-11
- **Verification method:** shared test vectors on every platform; automated tests; manual QA
- **Source:** current registration on all platforms; ADR-0003
- **Acceptance criteria:**
  ```gherkin
  Scenario: valid form sends a code to the email
    Given the user filled full name, phone, email, a password of 8 characters and accepted the terms
    When the user submits
    Then the verification code view opens showing the email address
    And a 6-digit code arrives at that address

  Scenario: dharma name and temple may be left empty
    Given the user left dharma name empty and chose "Chưa xuất gia — Cư sĩ tu tại gia"
    When the user submits an otherwise valid form
    Then the registration continues

  Scenario: invalid field is explained and nothing is lost
    Given the user typed a password of 7 characters
    When the user submits
    Then the password field says it needs at least 8 characters
    And every other field keeps its value

  Scenario: terms not accepted
    Given every field is valid but the terms are not accepted
    Then the user cannot submit the form

  Scenario: the device is offline
    Given the device has no connection
    When the user submits a valid form
    Then the user is told the request could not be sent
    And every field keeps its value
  ```

**FR-ONB-004 — Choose the ordination temple**
- **Priority:** Should
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the user opens the ordination temple field THE SYSTEM SHALL list InnerLamp's
  temples, searchable by name, with "Chưa xuất gia — Cư sĩ tu tại gia" first.
- **Business rules:** BR-ONB-10
- **Verification method:** automated tests; manual QA
- **Source:** backend v7.0, iOS and website today
- **Acceptance criteria:**
  ```gherkin
  Scenario: pick a temple from the list
    Given the user types part of a temple name
    When the user chooses a temple from the results
    Then the field shows that temple

  Scenario: no ordination temple
    When the user chooses "Chưa xuất gia — Cư sĩ tu tại gia"
    Then the account is created without an ordination temple
  ```

**FR-ONB-005 — Phone number already in use**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the user submits a phone number that already belongs to an account THE SYSTEM
  SHALL refuse the registration with "Số điện thoại này đã được đăng ký" next to the phone field and
  suggest signing in.
- **Business rules:** BR-ONB-04, BR-ONB-08
- **Verification method:** automated tests
- **Source:** current behavior; ADR-0003 rule 3
- **Acceptance criteria:**
  ```gherkin
  Scenario: phone already used
    Given an account exists with phone +84912345678
    When someone submits the form with Việt Nam and "0912345678"
    Then the phone field says "Số điện thoại này đã được đăng ký"
    And no code is sent
  ```

**FR-ONB-006 — Propose a missing temple**
- **Priority:** Could
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the user cannot find their temple and fills in the temple proposal THE SYSTEM
  SHALL keep the proposal with the registration and send it for review once the account is created;
  the registration does not depend on the proposal being accepted.
- **Business rules:** BR-ONB-10
- **Verification method:** automated tests; manual QA
- **Source:** website today; iOS keeps it local (drift)
- **Acceptance criteria:**
  ```gherkin
  Scenario: proposal sent after the account exists
    Given the user proposed "Chùa Phước Hậu" while registering
    When the user verifies the email
    Then the account is created
    And the proposal is sent for review

  Scenario: abandoned registration sends nothing
    Given the user proposed a temple but never verified the email
    Then no proposal is sent
  ```

#### Email verification

**FR-ONB-007 — Enter the verification code**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the user enters the 6-digit code that was emailed, within its validity, THE
  SYSTEM SHALL create the account, sign the user in and open Home (apps) or the home page (website).
- **Business rules:** BR-ONB-08, BR-ONB-09
- **Verification method:** automated tests; end-to-end test on the backend
- **Source:** current flow, now by email (ADR-0003)
- **Acceptance criteria:**
  ```gherkin
  Scenario: right code creates the account
    Given a code was emailed 2 minutes ago
    When the user enters it
    Then the account exists with the details from the form
    And the user is signed in
    And Home opens

  Scenario: wrong code
    When the user enters a code that is not the one emailed
    Then the user is told the code is wrong
    And the account is not created

  Scenario: expired code
    Given the code was emailed 11 minutes ago
    When the user enters it
    Then the user is told the code has expired and can request a new one

  Scenario: third wrong code locks the code
    Given the user entered 2 wrong codes
    When the user enters a third wrong code
    Then the user is told to try again later, with the time it unlocks
  ```

**FR-ONB-008 — Time left and resending the code**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHILE the verification code view is open THE SYSTEM SHALL show the time the code
  remains valid and, after the wait, offer "Gửi lại mã"; WHEN the user resends THE SYSTEM SHALL email a
  new code that replaces the old one.
- **Business rules:** BR-ONB-09
- **Verification method:** automated tests
- **Source:** current behavior (website countdown; iOS shows a fixed text — drift)
- **Acceptance criteria:**
  ```gherkin
  Scenario: resend after the wait
    Given 60 seconds have passed since the code was sent
    When the user chooses "Gửi lại mã"
    Then a new code is emailed
    And the previous code no longer works

  Scenario: resend too soon is not offered
    Given 20 seconds have passed since the code was sent
    Then "Gửi lại mã" is not available and the remaining wait is shown

  Scenario: too many codes
    Given 4 codes were already sent for this registration
    When the user asks for another
    Then the user is told sending is paused, with the time it resumes
    And the last code received can still be entered
  ```

**FR-ONB-009 — Change the email**
- **Priority:** Must
- **Platforms:** Website, iOS, Android
- **Description:** WHEN the user chooses "Đổi email" on the verification code view THE SYSTEM SHALL go
  back to the registration form with every field still filled in.
- **Business rules:** BR-ONB-08
- **Verification method:** automated UI tests
- **Source:** current "Đổi số" behavior on iOS and website, renamed
- **Acceptance criteria:**
  ```gherkin
  Scenario: go back and fix the email
    Given the user is on the verification code view
    When the user chooses "Đổi email"
    Then the registration form opens with every field as the user left it
  ```

**FR-ONB-010 — Registration timed out**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the user enters or requests a code more than 24 hours after submitting the form
  THE SYSTEM SHALL say "Thời gian đăng ký đã hết. Vui lòng đăng ký lại từ đầu." and return to the form
  with the fields still filled in.
- **Business rules:** BR-ONB-08
- **Verification method:** automated tests
- **Source:** current backend (24 h); iOS message
- **Acceptance criteria:**
  ```gherkin
  Scenario: come back the next day
    Given the user submitted the form 25 hours ago and never verified
    When the user asks for a new code
    Then the user is told the registration has expired
    And the form opens again with the fields as the user left them
  ```

### 3.4 Non-functional requirements

- **NFR-ONB-01 — No account enumeration through registration:** nothing on screen, in timing or in
  the response tells a visitor whether an email already has an account (BR-ONB-08). The phone check
  is the accepted exception (BR-ONB-04). Platforms: Backend, all clients.
- **NFR-ONB-02 — Abuse limits:** registration, code sending and code checking are limited per device
  address and per email, with a daily cap on emails sent by InnerLamp (ADR-0003 rule 8). When a limit
  is hit the user sees a "try again later" message, never a technical error. Platforms: Backend, all
  clients.
- **NFR-ONB-03 — Secrets:** the password and the code are never shown back, never written in emails
  other than the code itself, and never kept in plain form by any platform. Platforms: all.

### 3.5 Data requirements

| Data item | Meaning | Owned by | Retention / deletion | Private or public |
|---|---|---|---|---|
| Email | Sign-in identifier, proven by the code | Account (server) | With the account | Private |
| Phone number | Contact; matches genealogy invitations; not proven | Account (server) | With the account | Private |
| Full name | The user's name | Account | With the account | Private (shown to people the user shares with) |
| Dharma name | Optional Buddhist name | Account | With the account | Private |
| Ordination temple | One temple from the list, or none | Account | With the account | Private |
| Password | Secret | Account (never readable) | Replaced on reset | Secret |
| Pending registration | Form data waiting for the code | Server | Deleted when the account is created or after 24 h | Private |
| Intro seen | Whether this installation showed the intro | This device | Until the app is removed | Device only |
| Intro choices | Wishes and bell times picked in the intro | This device | Until the app is removed | Device only |

### 3.6 Platform variance

| ID | Platforms | Difference | Reason |
|---|---|---|---|
| FR-ONB-001, FR-ONB-002, BR-ONB-01…03 | iOS, Android only | Splash, intro screens and launch routing | The website has no install and lands on its home page |
| BR-ONB-06 | each client | How the strength hint looks and its levels | Advice only, never blocks |
| BR-ONB-11 | each client | How the one terms text is shown (in-app sheet, web pages) | Platform convention |
| FR-ONB-007 | Website | After verification the user lands on the website home page | The website has no Home tab |
| FR-ONB-003 | Website only | A signed-in user who opens the registration page is taken to the home page | Only the website has addressable pages |

### 3.7 Software system attributes

- **Reliability:** a failed request never loses what the user typed.
- **Availability:** when the backend or the email service is down, registration cannot finish; the user
  is told to try again later. Already signed-in users are unaffected.
- **Security & privacy:** see NFR-ONB-01…03 and ADR-0003 rules 6 and 8. Personal data (name, phone,
  email, dharma name, temple) belongs to the account and is deleted with it.
- **Maintainability / portability:** N/A at product level.

---

## 4. Verification & traceability

### 4.1 Verification approach

Each client runs the vectors in `rules/` against its own early checks; the backend runs them against
its validation. Flows are covered by automated tests per platform and by the backend's end-to-end
tests against a real database. Email delivery is checked with the development mailer, which logs the
code instead of sending it.

### 4.2 Traceability matrix

Status per platform: `planned` · `in-progress` · `done` · `drift` · `n/a` · `later`.

| ID | Backend | Website | iOS | Android | Note |
|---|---|---|---|---|---|
| BR-ONB-01 | n/a | n/a | drift | drift | iOS records completion; Android keeps nothing |
| BR-ONB-02 | n/a | n/a | drift | drift | iOS → registration, Android → intro today |
| BR-ONB-03 | n/a | n/a | done | done | Android keeps choices in memory only |
| BR-ONB-04 | drift | drift | done | drift | Backend accepts any valid number; website 9–10 digits, China listed |
| BR-ONB-05 | drift | done | done | drift | Backend: email not the identifier, case handling to check |
| BR-ONB-06 | drift | drift | done | drift | Backend has no 64 limit; website counts bytes; Android hint levels differ |
| BR-ONB-07 | done | drift | drift | drift | Website full name ≥ 2; iOS dharma name required; Android no full name |
| BR-ONB-08 | drift | drift | drift | planned | Verification by SMS today; email-taken answered openly |
| BR-ONB-09 | drift | done | done | planned | Backend validity 90 s, fixed code |
| BR-ONB-10 | done | done | drift | drift | iOS never sends the proposal; Android free text |
| BR-ONB-11 | n/a | drift | drift | drift | Three separate texts today (iOS sheet, web pages, Android static) |
| FR-ONB-001 | n/a | n/a | done | done | 4 scenarios |
| FR-ONB-002 | n/a | n/a | drift | drift | 3 scenarios |
| FR-ONB-003 | drift | drift | drift | planned | 5 scenarios; email code instead of SMS; submit reads "Gửi mã xác thực" on all three clients |
| FR-ONB-004 | done | done | done | drift | 2 scenarios |
| FR-ONB-005 | done | done | done | planned | 1 scenario |
| FR-ONB-006 | done | done | drift | planned | 2 scenarios |
| FR-ONB-007 | drift | drift | drift | planned | 4 scenarios |
| FR-ONB-008 | drift | done | drift | planned | 3 scenarios |
| FR-ONB-009 | n/a | drift | drift | planned | 1 scenario; "Đổi số" today |
| FR-ONB-010 | done | drift | done | planned | 1 scenario; website stays on the code view |
| NFR-ONB-01 | drift | n/a | n/a | n/a | Email-taken is revealed today |
| NFR-ONB-02 | planned | planned | planned | planned | No rate limit today (pentest H-2) |
| NFR-ONB-03 | done | done | done | planned | |

### 4.3 Test vectors

| File | Rule | Cases |
|---|---|---|
| `rules/phone-number.json` | BR-ONB-04 | 13 |
| `rules/email.json` | BR-ONB-05 | 11 |
| `rules/password.json` | BR-ONB-06 (and BR-LGN-05) | 10 |
| `rules/names.json` | BR-ONB-07 | 11 |

---

## 5. Appendices

### Appendix A — Open questions

#### Open (unconfirmed)

None.

#### Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| Is "Bỏ qua" shown on intro step 3? | Yes, on all three steps, as the apps do today | 2026-10-15 |
| Registration submit label? | "Đăng ký", not "Gửi mã xác thực" | 2026-10-15 |
| Street line of a proposed temple? | Optional | 2026-10-14 |
| Does skipping the intro count as seen? | Yes | 2026-10-14 |
| Website: signed-in user opens the registration page? | Taken to the home page | 2026-10-14 |
| Address of a proposed temple? | Province and ward from the administrative-unit list, required | 2026-10-14 |
| One terms text for every client? | Yes, shown each platform's way | 2026-10-14 |
| How is the account proven without SMS? | Email code; phone kept, required, unique, unverified (ADR-0003) | 2026-10-14 |
| One spec or two? | Two: `onboarding` and `login` | 2026-10-14 |
| Intro choices: device or account? | Device only | 2026-10-14 |
| Launch after the intro when signed out? | Sign-in screen; the intro-seen record lives on the device | 2026-10-14 |
| Code validity? | Longer than 90 s: 10 minutes, a backend setting | 2026-10-14 |
| Dharma name / full name? | Dharma name optional, full name required | 2026-10-14 |
| Password rule? | 8–64 characters, no composition rule; strength hint is advice | 2026-10-14 |
| Phone check? | Exact length per country, one shared list of 12 countries | 2026-10-14 |
| Account created when? | Only after the email code is verified | 2026-10-14 |
| Change email/phone, delete account, Google/Apple? | Later | 2026-10-14 |

### Appendix B — Contract handoff (backend)

1. Registration takes the same fields as today; the code goes to the **email**, and pending
   registrations, resends and verification are keyed by email instead of phone.
2. Code validity becomes a backend setting, default 10 minutes; the response tells clients the time
   left and the wait before resending (clients show those values).
3. Registering with an email that already has an account answers exactly like a new email and sends the
   notice email (BR-ONB-08); the explicit "email already used" answer goes away. "Phone already used"
   stays.
4. Phone validation follows the 12-country list and digit counts of BR-ONB-04 (today any valid number).
5. Password limit 8–64 characters counted as characters. With multi-byte characters 64 characters can
   exceed what the current password hashing accepts in full; the backend spec must make every character
   count.
6. Email compared without case, stored trimmed; at most 254 characters.
7. Verified accounts record that the email is verified (ADR-0003 rule 1).
8. Rate limits on every registration and code request answer with a "try later" status and the time
   (NFR-ONB-02).
9. The temple proposal is sent after the account exists (today: website only).
10. SMS sending is removed (ADR-0003 code table).
