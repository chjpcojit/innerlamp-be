# Product Spec — Memorial book (Sổ giỗ chạp)

| Field | Value |
|---|---|
| Document type | Product spec — Software Requirements Specification, IEEE 29148-2018 structure |
| Feature | Memorial book / Sổ giỗ chạp (`memorial`) |
| Feature code | `MEM` — IDs are `BR-MEM-NN`, `FR-MEM-NNN`, `NFR-MEM-NN` |
| Product | InnerLamp |
| Platforms in scope | Backend, Website, iOS, Android |
| Version | 0.1 (draft) |
| Last updated | 2026-10-14 |
| Status | draft — every gathering question answered on 2026-10-14; awaiting the user's review of the written spec (→ v1.0) |
| Owner / author | Product owner: anh Đức · written by Claude (`ba-writer`, retroactive mode) |
| API contract | `docs/specs/api/openapi.yaml` (owned by the backend; this spec never restates it) |
| Architecture decisions | No ADR. The storage/reminder decision of 2026-10-06 is recorded as an existing constraint in §2.7 (user's choice, 2026-10-14) |
| Downstream | Backend: `docs/specs/features/memorial/` (`spec-writer`) · Website: `Docs/specs/features/memorial/` · iOS: `Docs/Specs/Memorial/` · Android: `Docs/Specs/Memorial/` |

> See `changelog.md` in this folder for the version history. Platform specs cite this file as
> `docs/product/memorial/product.md@<version>` plus the IDs they implement.

---

## 1. Introduction

### 1.1 Purpose

This document is the single source of product truth for the memorial book on every InnerLamp
platform. It states what a user can observe and the business rules behind it, so the backend, the
website, the iOS app and the Android app implement the same feature. It replaces the product
decisions scattered across the backend spec, the iOS spec and assumptions, the Android assumptions
and the website code.

### 1.2 Scope

**In scope:**
- A signed-in user keeps a private book of deceased relatives (người thân đã mất), each with a
  relationship, a date of death in the lunar or solar calendar, an optional note and reminder
  settings.
- The book shows every relative's next death anniversary (kỳ giỗ tới), the days left until it, and
  which anniversary it is (giỗ đầu, giỗ hết, giỗ lần thứ n).
- The user adds, edits and deletes entries; the book is the same on every device and platform the
  user signs in to.
- The apps remind the user before each anniversary.
- A portrait photo per relative (planned, not yet available — FR-MEM-011).

**Out of scope:**
- Sharing a book with other people, or editing it together (only the owner sees it).
- Creating entries from the family tree ("Lấy từ gia phả") — owned by the `genealogy` product spec.
- Reminders sent by the server (push); reminders on the website.
- Search, multiple photos per relative, soft delete / restore, export.

### 1.3 Product overview

InnerLamp (Tâm Đăng) is a Vietnamese Buddhist practice app: practice tools, a lunar calendar, a
library, temples and family features. Vietnamese families hold a death-anniversary ceremony (giỗ)
for each ancestor every year, usually on the lunar date of death. The memorial book is one of the
family features: it remembers each anniversary for the user, converts lunar dates to this year's
solar date, and reminds the user in time to prepare offerings. It is reached from the home screen,
next to the lunar calendar.

### 1.4 Glossary

| Term | Definition | Not to be confused with |
|---|---|---|
| Sổ giỗ chạp (memorial book) | The user's private list of deceased relatives and their anniversaries | Gia phả (family tree), a separate feature |
| Giỗ (death anniversary) | The yearly ceremony on the date a person died, in the calendar the family follows | Ngày mất (date of death), which happened once |
| Kỳ giỗ tới (next anniversary) | The first anniversary date on or after today, as a solar date | The date of death itself |
| Âm lịch (lunar calendar) | The Vietnamese lunisolar calendar, computed at UTC+7 (BR-MEM-10) | The Chinese calendar, which can differ by a day or a month |
| Dương lịch (solar calendar) | The Gregorian calendar | — |
| Tháng nhuận (leap month) | An extra lunar month inserted in some years; it repeats the number of the month before it | A leap year in the solar calendar |
| Tháng thiếu / tháng đủ (short / full month) | A lunar month of 29 / 30 days; the length of a given month changes from year to year | — |
| Giỗ đầu (first anniversary) | The anniversary one year after death | — |
| Giỗ hết (end-of-mourning anniversary) | The anniversary two years after death | Giỗ đầu |
| Giỗ lần thứ n (n-th anniversary) | The anniversary n years after death, n ≥ 3 | — |
| Lời nhắc (reminder) | A notification the app shows before an anniversary | The OS reminders app (iOS only, FR-MEM-009) |

### 1.5 References

- Design prototype: `InnerLamp Prototype.dc.html` in the iOS/Android repos (`Docs/innerlamp-project/project/`),
  screens "Sổ giỗ chạp" (`gio`, `gioEmpty`) and "Thêm / sửa ngày giỗ" (`gioEdit`), logic `gioNext`.
  Neither version of `Mo-ta-tinh-nang.md` (md5 `c6a951…`, `63347c…`) describes this feature.
- Prior documents:
  - Backend: `docs/specs/features/memorial/requirements.md` v2.4, `docs/assumptions/memorial/A-001`,
    `docs/tech-debt/memorial/TD-001`.
  - iOS: `Docs/Specs/Memorial/requirements.md` v1.3, `Docs/Assumptions/Memorial/AS-Memorial-001…006`,
    `Docs/Tech-Debt/Memorial/TD-Memorial-001…002`.
  - Android: `Docs/gio_assumptions.md`, `domain/calendar/MemorialSchedule.kt`.
  - Website: no spec; `src/lib/utils/memorial.ts`, `src/lib/content/memorial.ts`.
  - Reconciliation report: `/mnt/project-files/reports/product/memorial/2026-10-14-cau-hoi-truoc-khi-viet.md`.
- Platform folder names: Backend `memorial` · Website `memorial` (routes) · iOS `Memorial` · Android `gio`.
  New platform specs use `memorial` / `Memorial`.

---

## 2. Overall description

### 2.1 Product perspective

Retroactive spec of a feature already built in part on every platform (see §4.2). It touches:
- **Sign-in** (`login`, `onboarding`): the book belongs to the account.
- **Lunar calendar** (`calendar`): the same Vietnamese lunar conversion.
- **Home** (`home`): the entry point.
- **Family tree** (`genealogy`): may create entries later; not specified here.

External capabilities, by role: the device's local notifications (apps), the iOS reminders app
(iOS only), photo storage (planned).

### 2.2 Product functions

- Open the memorial book (FR-MEM-001)
- See every relative with the next anniversary, days left and which anniversary it is (FR-MEM-002)
- Add, edit and delete a relative (FR-MEM-003, FR-MEM-004, FR-MEM-005, FR-MEM-012, FR-MEM-013)
- Preview the next anniversary while entering a date (FR-MEM-006)
- Set and receive reminders (FR-MEM-007, FR-MEM-008, FR-MEM-009)
- Read the book offline in the apps (FR-MEM-010)
- Add a portrait (FR-MEM-011, planned)

### 2.3 User classes and characteristics

| Actor | Description | Permissions |
|---|---|---|
| Guest | Not signed in | Cannot open the book (FR-MEM-001) |
| Signed-in user | Owner of exactly one memorial book | Sees, adds, edits and deletes only their own entries (BR-MEM-01) |

### 2.4 Operating environment

| Platform | In scope | Notes |
|---|---|---|
| Backend | yes | Keeps each user's book privately, enforces BR-MEM-01…04 and BR-MEM-12; does not compute anniversaries or send reminders |
| Website | yes | Full book; no reminders delivered, no offline (§3.6) |
| iOS | yes | Full book, reminders, OS reminders app, offline reading |
| Android | yes | Full book, reminders, offline reading; no OS reminders app (§3.6) |

Network expectation: online to change the book; the apps can read the last loaded book offline
(FR-MEM-010); the website is online only.

### 2.5 Constraints

- Cultural: the anniversary of a lunar date of death follows Vietnamese custom — always the regular
  (non-leap) lunar month with the same number (BR-MEM-05).
- Privacy: names and notes about deceased relatives are personal family data (NFR-MEM-02).

### 2.6 User documentation

N/A — the form explains itself (the preview of FR-MEM-006 states the day-30 rule).

### 2.7 Assumptions and dependencies

- **Existing constraint (user decision 2026-10-06, confirmed as already decided on 2026-10-14; no ADR):**
  the server keeps the date of death and the reminder settings only. Each app computes the next
  anniversary and schedules its own reminders on the device; there is no server-sent reminder. This
  is why BR-MEM-05…09 apply to the clients and not the backend.
- **Assumptions:**
  - The book is new on every platform except the backend and iOS; book data kept only in a browser
    (website) or in memory (Android) before this spec is discarded, as the product is not released
    (Q2, 2026-10-14).
- **Dependencies:** sign-in on every platform; one shared Vietnamese lunar conversion (BR-MEM-10);
  photo storage for FR-MEM-011 (not available yet).

---

## 3. Specific requirements

### 3.1 External interface requirements

#### 3.1.1 User interfaces

- **Memorial book view.** A count of relatives ("N người thân"); when not empty, a highlight of the
  soonest anniversary (relative, relationship, which anniversary, the date of death as entered, the
  solar date it falls on this time with the weekday, reminder summary), then every relative sorted
  by next anniversary, each with relationship, date of death, days left and whether a reminder is on.
  An empty book shows an explanation and an invitation to add a relative.
- **Add / edit view.** Portrait (planned), full name, relationship (single choice), calendar
  (lunar / solar), day, month, year of death (or "unknown"), leap-month choice (lunar only), note,
  reminder settings, a live preview of the next anniversary, save; in edit mode also delete.
- **Navigation.** Home → book → add/edit → back to book with a confirmation message.

Layout and copy: the design prototype (§1.5).

#### 3.1.2 Hardware interfaces

Local notifications on iOS and Android (FR-MEM-008); the photo library or camera for the portrait
(FR-MEM-011, planned).

#### 3.1.3 Software interfaces

The product needs the backend to: keep a signed-in user's memorial entries privately; create, read,
update and delete one entry; list all of a user's entries; reject invalid entries per BR-MEM-02…04;
and (later) store one portrait per entry. The contract is `docs/specs/api/openapi.yaml`; open
contract needs are in Appendix B.

The iOS app additionally writes to the iOS reminders app (FR-MEM-009).

#### 3.1.4 Communication interfaces

N/A — no SMS, email or server push.

### 3.2 Business rules

**BR-MEM-01 — Private, account-owned book**
- **Rule:** A memorial entry belongs to the signed-in account that created it. Only that account can
  see, change or delete it, on any device or platform. Another user can never learn whether a given
  entry exists. The book is identical on every device and platform once the user is signed in.
- **Examples:** add on iPhone → visible on the website after sign-in; user B opening user A's entry →
  treated as not found.
- **Computational:** no
- **Applies to:** Backend, Website, iOS, Android
- **Source:** backend spec v2.4 (AC13–15); Q2 answered 2026-10-14

**BR-MEM-02 — Entry content and limits**
- **Rule:** An entry has: a full name (required, 1–100 characters after trimming leading and trailing
  spaces); a relationship from exactly this list — ông nội (paternal grandfather), bà nội (paternal
  grandmother), ông ngoại (maternal grandfather), bà ngoại (maternal grandmother), cha (father), mẹ
  (mother), vợ / chồng (spouse), anh chị em (sibling), con (child), khác (other; no free-text label);
  a date of death (BR-MEM-03); an optional note of at most 1000 characters; reminder settings
  (BR-MEM-11). A user has at most 100 entries. Two entries may be identical.
- **Examples:** name "  Nguyễn Văn A  " → saved as "Nguyễn Văn A"; name of only spaces → rejected;
  101st entry → rejected with a message that the book is full.
- **Computational:** no (limits verified by scenarios)
- **Applies to:** Backend, Website, iOS, Android
- **Source:** backend spec v2.4 (AC3, AC4, AC7, AC11); iOS spec v1.3

**BR-MEM-03 — Date of death**
- **Rule:** The date of death is entered in the lunar or the solar calendar and keeps that calendar.
  Day and month are required; the year is optional ("không rõ"). When the year is given it is
  between 1900 and the current year, and the full date must exist in that calendar and must not be
  after today (today in `Asia/Ho_Chi_Minh`). Without a year: solar day 1 to the month's length
  (29 February allowed), lunar day 1–30. Lunar years are lunar years: a death on lunar 3/12/2026 is in
  lunar year 2026 although it fell on 10 January 2027.
- **Examples:** solar 31/4 → rejected; solar 29/2/2025 → rejected; solar 29/2 with unknown year →
  accepted; lunar 30/7/2026 → rejected (lunar month 7 of 2026 has 29 days); year 1899 → rejected;
  tomorrow's date → rejected.
- **Computational:** yes (lunar ↔ solar conversion) → `rules/death-date-conversion.json`
- **Applies to:** Backend (validation), Website, iOS, Android (form and conversion)
- **Source:** backend spec v2.4 (AC5–6b); Q7 answered 2026-10-14 (1900 → current year on every platform)

**BR-MEM-04 — Death in a leap month**
- **Rule:** For a lunar date of death the user may say the month was a leap month (tháng nhuận). With
  a known year, this is allowed only if that lunar year has that leap month. The leap marker is part
  of the recorded date of death and is shown back exactly as entered; it does not change the
  anniversary (BR-MEM-05).
- **Examples:** 10/6 nhuận/2025 → accepted (2025 has a leap month 6); 10/6 nhuận/2026 → rejected
  (2026 has no leap month); 10/6 nhuận, year unknown → accepted.
- **Computational:** yes → `rules/death-date-conversion.json`
- **Applies to:** Backend, Website, iOS, Android
- **Source:** Q1 answered 2026-10-14; backend resolved question 2026-10-07

**BR-MEM-05 — Next anniversary**
- **Rule:** The next anniversary is the first date on or after today, in the entry's calendar, with
  the entry's day and month. For a lunar entry the anniversary is always in the regular (non-leap)
  lunar month with that number, every year — including when the person died in a leap month, and
  including years that have a leap month with that number. It is shown as a solar date.
- **Examples:** today 14/10/2026, lunar 15/10 → 23/11/2026; today 10/8/2025, died 10/6 nhuận/2025 →
  23/07/2026 (not 03/08/2025, which is the leap month 6); today 1/6/2025, lunar 10/6 → 04/07/2025
  (regular month 6, not the leap one).
- **Computational:** yes → `rules/next-anniversary.json`
- **Applies to:** Website, iOS, Android (Backend: n/a, §2.7)
- **Source:** Q1 answered 2026-10-14; prototype `gioNext`; backend resolved question 2026-10-07

**BR-MEM-06 — Day 30 in a short lunar month**
- **Rule:** When a lunar anniversary is on day 30 and that lunar month has 29 days in the target year,
  the anniversary falls on day 29 of that month.
- **Examples:** lunar 30/7, year 2026 (month 7 has 29 days) → lunar 29/7/2026 = 10/09/2026; lunar
  30/9, year 2026 (30 days) → 08/11/2026.
- **Computational:** yes → `rules/next-anniversary.json`
- **Applies to:** Website, iOS, Android
- **Source:** iOS spec v1.3; Android assumption #6; prototype

**BR-MEM-07 — 29 February in a common year**
- **Rule:** A solar anniversary on 29 February falls on 28 February in years that are not leap years.
- **Examples:** 29/2 in 2026 → 28/02/2026; in 2028 → 29/02/2028.
- **Computational:** yes → `rules/next-anniversary.json`
- **Applies to:** Website, iOS, Android
- **Source:** Android assumption #6; iOS spec v1.3

**BR-MEM-08 — Days left**
- **Rule:** Days left is the whole number of days from today to the next anniversary: 0 is shown as
  "Hôm nay" (today), 1 as "Ngày mai" (tomorrow), otherwise "còn N ngày" (N days left). Seven days or
  fewer is emphasised. "Today" is the date on the user's device.
- **Examples:** anniversary today → "Hôm nay"; 37 days → "còn 37 ngày".
- **Computational:** yes → `rules/next-anniversary.json` (days left)
- **Applies to:** Website, iOS, Android
- **Source:** iOS spec v1.3; Q10 answered 2026-10-14

**BR-MEM-09 — Which anniversary**
- **Rule:** When the year of death is known, the ordinal of the next anniversary is (year of the next
  anniversary − year of death), both counted in the entry's calendar (lunar years for a lunar entry).
  1 → "Giỗ đầu", 2 → "Giỗ hết", n ≥ 3 → "Giỗ lần thứ n", less than 1 → no label. Unknown year → no label.
- **Examples:** solar 20/11/2024, next 20/11/2026 → "Giỗ hết"; lunar 20/12/2025, next lunar
  20/12/2026 (27/01/2027) → "Giỗ đầu".
- **Computational:** yes → `rules/next-anniversary.json` (ordinal), `rules/anniversary-ordinal.json` (label)
- **Applies to:** Website, iOS, Android
- **Source:** iOS spec v1.3; Android assumption #8; prototype

> ⚠️ Not confirmed: a person who died today has a next anniversary of today with ordinal 0 (no
> label) under this rule; should the book instead show next year's "Giỗ đầu"? — affects the list and
> preview on Website, iOS, Android; the vector for this case is left out until answered.

**BR-MEM-10 — Calendar authority**
- **Rule:** All lunar ↔ solar conversion uses the Vietnamese lunar calendar computed at UTC+7
  (Hồ Ngọc Đức's algorithm), never a Chinese calendar from the operating system or a library. The
  backend's converter (lunar calendar feature) is the reference implementation from which the test
  vectors are generated; every platform must pass the same vectors.
- **Examples:** Tết 2026 = 17/02/2026; lunar 2025 has a leap month 6 starting 25/07/2025.
- **Computational:** yes → all files under `rules/`
- **Applies to:** Website, iOS, Android (and Backend if it ever converts memorial dates)
- **Source:** Q10 answered 2026-10-14; backend spec v2.4 technical note

**BR-MEM-11 — Reminder settings**
- **Rule:** Each entry has a reminder on or off. When on, the user chooses how early — on the day, 1,
  3 or 7 days before — and at what time — 06:00, 07:00, 12:00 or 19:00 — in the device's local time.
  A new entry starts with the reminder on, 1 day before, 07:00. The reminder fires once for each next
  anniversary (BR-MEM-05) at that lead and time, then for the following one.
- **Examples:** lunar 15/10, next 23/11/2026, 1 day before at 07:00 → 22/11/2026 07:00 device time;
  "on the day" at 06:00 → 23/11/2026 06:00.
- **Computational:** yes → `rules/next-anniversary.json` (the reminder moment is derived from the next
  anniversary; no separate vector)
- **Applies to:** Website (settings only), iOS, Android; Backend stores the settings
- **Source:** Q5 answered 2026-10-14; iOS spec v1.3; prototype

**BR-MEM-12 — Deletion is permanent**
- **Rule:** Deleting an entry removes it for good on every device; it cannot be restored. The user is
  asked to confirm before it happens.
- **Examples:** confirm → entry gone everywhere; cancel → nothing changes.
- **Computational:** no
- **Applies to:** Backend, Website, iOS, Android
- **Source:** Q6 answered 2026-10-14 (replaces the prototype's delete-without-asking, iOS `AS-Memorial-004`)

### 3.3 Functional requirements

#### Viewing the book

**FR-MEM-001 — Open the memorial book**
- **Priority:** Must
- **Platforms:** Website, iOS, Android
- **Description:** WHEN a signed-in user opens the memorial book from the home screen THE SYSTEM SHALL
  show that user's book; WHEN a guest tries to open it THE SYSTEM SHALL ask them to sign in first.
- **Business rules:** BR-MEM-01
- **Verification method:** automated UI/integration test per platform
- **Source:** iOS spec v1.3; Q2 answered 2026-10-14
- **Acceptance criteria:**
  ```gherkin
  Scenario: Signed-in user opens the book
    Given the user is signed in
    When the user opens the memorial book from the home screen
    Then the user's memorial book is shown

  Scenario: Guest is asked to sign in (BR-MEM-01)
    Given the user is not signed in
    When the user opens the memorial book
    Then the user is asked to sign in
    And no memorial entries are shown

  Scenario: Same book on another platform (BR-MEM-01)
    Given the user added "Nguyễn Văn A" on one device
    When the user signs in on another device or platform and opens the memorial book
    Then "Nguyễn Văn A" is in the book
  ```

**FR-MEM-002 — List by next anniversary**
- **Priority:** Must
- **Platforms:** Website, iOS, Android
- **Description:** WHEN the book has entries THE SYSTEM SHALL list every entry sorted by days left to
  its next anniversary (soonest first), show the soonest one highlighted, and for each entry show the relationship, the date of
  death as entered (calendar, leap marker), the days left and whether a reminder is on; the highlight
  also shows which anniversary it is and the solar date with its weekday. WHEN the book is empty THE
  SYSTEM SHALL show an empty state inviting the user to add a relative.
- **Business rules:** BR-MEM-05, BR-MEM-06, BR-MEM-07, BR-MEM-08, BR-MEM-09
- **Verification method:** shared vectors (`rules/next-anniversary.json`) + UI test per platform
- **Source:** iOS spec v1.3; prototype
- **Acceptance criteria:**
  ```gherkin
  Scenario: Sorted by soonest anniversary
    Given today is 14/10/2026
    And the book has a solar 20/11 entry and a lunar 15/10 entry
    When the user opens the memorial book
    Then the solar 20/11 entry (37 days left) is shown before the lunar 15/10 entry (40 days left)
    And the solar 20/11 entry is highlighted

  Scenario: Anniversary today (BR-MEM-08)
    Given today is the next anniversary of an entry
    When the user opens the memorial book
    Then that entry shows "Hôm nay" (today)

  Scenario: Which anniversary (BR-MEM-09)
    Given an entry died on solar 20/11/2024
    And today is 14/10/2026
    When the user opens the memorial book
    Then the entry's next anniversary is labelled "Giỗ hết"

  Scenario: Empty book
    Given the user has no entries
    When the user opens the memorial book
    Then the user sees that the book is empty and is invited to add a relative
  ```

#### Changing the book

**FR-MEM-003 — Add a relative**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the user adds a relative with valid content THE SYSTEM SHALL save it to the
  user's book, return to the book and confirm "Đã lưu vào sổ giỗ." (saved to the memorial book).
  A new entry starts as: relationship ông nội, lunar calendar, day 1, month 1, year unknown, not a leap
  month, reminder on, 1 day before, 07:00 (and, on iOS, adding to the reminders app on — FR-MEM-009).
- **Business rules:** BR-MEM-01, BR-MEM-02, BR-MEM-03, BR-MEM-04, BR-MEM-11
- **Verification method:** automated tests per platform
- **Source:** backend spec v2.4 (US-MEM-01); iOS spec v1.3; prototype
- **Acceptance criteria:**
  ```gherkin
  Scenario: Add a relative with a lunar date
    Given the user is signed in
    When the user adds "Nguyễn Văn A", relationship cha, lunar 15/10, year unknown, and saves
    Then the book lists "Nguyễn Văn A"
    And the user is told the entry was saved

  Scenario: Add a relative who died in a leap month (BR-MEM-04)
    Given the user is adding a relative
    When the user enters lunar 10/6 nhuận/2025 and saves
    Then the entry is saved
    And it shows its date of death as 10/6 nhuận/2025

  Scenario: Leap month the year does not have (BR-MEM-04)
    Given the user is adding a relative
    When the user enters lunar 10/6 nhuận/2026 and saves
    Then the user is told the date does not exist
    And nothing is added

  Scenario Outline: Invalid date of death (BR-MEM-03)
    Given the user is adding a relative
    When the user enters "<date>" and saves
    Then the user is told the date is not valid
    And nothing is added

    Examples:
      | date                            |
      | solar 29/2/2025                 |
      | lunar 30/7/2026                 |
      | a year before 1900              |
      | a date after today              |

  Scenario: Book is full (BR-MEM-02)
    Given the user already has 100 entries
    When the user adds another relative and saves
    Then the user is told the book is full
    And nothing is added

  Scenario: Server unreachable
    Given the user is adding a relative
    And InnerLamp cannot reach the server
    When the user saves
    Then the user is told the entry could not be saved
    And nothing is added
  ```

**FR-MEM-004 — Edit a relative**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the user opens an entry, changes it and saves with valid content THE SYSTEM
  SHALL replace the entry's content (photo excluded), return to the book with a confirmation and
  re-plan its reminder; WHEN the user goes back without saving THE SYSTEM SHALL keep the entry unchanged.
- **Business rules:** BR-MEM-01, BR-MEM-02, BR-MEM-03, BR-MEM-04, BR-MEM-11
- **Verification method:** automated tests per platform
- **Source:** backend spec v2.4 (US-MEM-03); iOS spec v1.3
- **Acceptance criteria:**
  ```gherkin
  Scenario: Change the date of death
    Given the book has "Nguyễn Văn A" on lunar 15/10
    When the user changes the date to lunar 20/10 and saves
    Then the book shows "Nguyễn Văn A" on lunar 20/10
    And the next anniversary is recomputed

  Scenario: Go back without saving
    Given the user is editing "Nguyễn Văn A"
    When the user changes the name and goes back without saving
    Then the book still shows "Nguyễn Văn A"

  Scenario: Entry deleted on another device
    Given the user is editing an entry that was deleted on another device
    When the user saves
    Then the user is told the entry no longer exists
    And the book no longer shows it
  ```

**FR-MEM-005 — Delete a relative**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the user chooses to delete an entry THE SYSTEM SHALL ask for confirmation;
  WHEN the user confirms THE SYSTEM SHALL delete it permanently, cancel its reminder, return to the
  book and confirm "Đã xóa <name> khỏi sổ giỗ." (removed from the memorial book); WHEN the user cancels
  THE SYSTEM SHALL change nothing.
- **Business rules:** BR-MEM-12
- **Verification method:** automated tests per platform
- **Source:** backend spec v2.4 (US-MEM-04); Q6 answered 2026-10-14
- **Acceptance criteria:**
  ```gherkin
  Scenario: Confirm deletion (BR-MEM-12)
    Given the user is editing "Nguyễn Văn A"
    When the user chooses delete and confirms
    Then "Nguyễn Văn A" is no longer in the book on any device
    And no reminder for "Nguyễn Văn A" fires afterwards

  Scenario: Cancel deletion
    Given the user is editing "Nguyễn Văn A"
    When the user chooses delete and cancels
    Then "Nguyễn Văn A" is still in the book
  ```

**FR-MEM-006 — Preview the next anniversary**
- **Priority:** Should
- **Platforms:** Website, iOS, Android
- **Description:** WHILE the user enters a day and month THE SYSTEM SHALL show the next anniversary
  as a solar date with weekday and days left; for a lunar date it SHALL add the lunar date and, when
  the year is known, which anniversary it is, and state that a day 30 falls on day 29 in a short month;
  for a solar date it SHALL add this year's equivalent lunar date.
- **Business rules:** BR-MEM-05, BR-MEM-06, BR-MEM-07, BR-MEM-08, BR-MEM-09
- **Verification method:** shared vectors + UI test per platform
- **Source:** iOS spec v1.3; prototype
- **Acceptance criteria:**
  ```gherkin
  Scenario: Preview a lunar date
    Given today is 14/10/2026
    When the user enters lunar 15/10
    Then the preview shows the next anniversary on Monday 23/11/2026, 40 days left

  Scenario: Preview day 30 in a short month (BR-MEM-06)
    Given today is 01/08/2026
    When the user enters lunar 30/7
    Then the preview shows the next anniversary on 10/09/2026
  ```

**FR-MEM-012 — Required name**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the user saves with a name that is empty after trimming THE SYSTEM SHALL keep
  the user on the form and show "Vui lòng nhập họ và tên" (please enter the full name) next to the
  name, clearing the message as soon as the user edits it.
- **Business rules:** BR-MEM-02
- **Verification method:** automated tests per platform
- **Source:** iOS spec v1.3; backend spec v2.4 (AC3)
- **Acceptance criteria:**
  ```gherkin
  Scenario Outline: Name empty after trimming
    Given the user is adding a relative
    When the user saves with the name "<name>"
    Then the user is told to enter the full name
    And nothing is added

    Examples:
      | name |
      |      |
      | "  " |

  Scenario: Name longer than 100 characters
    Given the user is adding a relative
    When the user enters a 101-character name and saves
    Then the user is told the name is too long
    And nothing is added
  ```

**FR-MEM-013 — Note**
- **Priority:** Could
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the user writes a note of up to 1000 characters THE SYSTEM SHALL keep it with
  the entry and show it when the entry is opened; a blank note is stored as no note.
- **Business rules:** BR-MEM-02
- **Verification method:** automated tests per platform
- **Source:** backend spec v2.4 (AC7)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Keep a note
    Given the user is adding a relative
    When the user writes the note "Giỗ tại nhà bác cả" (held at the eldest uncle's house) and saves
    Then opening the entry shows that note

  Scenario: Note too long
    Given the user is adding a relative
    When the user writes a 1001-character note and saves
    Then the user is told the note is too long
    And nothing is added
  ```

#### Reminders

**FR-MEM-007 — Reminder settings**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the reminder is on THE SYSTEM SHALL let the user choose the lead (on the day,
  1, 3, 7 days before) and the time (06:00, 07:00, 12:00, 19:00); WHEN it is off THE SYSTEM SHALL hide
  those choices. The book shows a "Có nhắc" (reminder on) mark on entries whose reminder is on.
- **Business rules:** BR-MEM-11
- **Verification method:** automated tests per platform
- **Source:** iOS spec v1.3; Q5 answered 2026-10-14
- **Acceptance criteria:**
  ```gherkin
  Scenario: Turn a reminder on
    Given the user is editing "Nguyễn Văn A"
    When the user turns the reminder on, chooses 3 days before at 19:00 and saves
    Then the entry shows that a reminder is on

  Scenario: Settings follow the account
    Given the user set a reminder on one device
    When the user opens the same entry on another device or platform
    Then the same reminder settings are shown
  ```

> ⚠️ Not confirmed: the website shows and lets the user edit reminder settings (they take effect in
> the apps) but delivers no reminder itself — is that the intended website behavior, or should the
> website hide reminder settings? — blocks the website platform spec for FR-MEM-007.

**FR-MEM-008 — Reminder delivery**
- **Priority:** Must
- **Platforms:** iOS, Android
- **Description:** WHEN an entry's reminder is on THE SYSTEM SHALL show a notification on every device
  where the user is signed in to the InnerLamp app, at the chosen lead and time before the next
  anniversary, and then plan the following anniversary; WHEN the reminder is turned off or the entry
  is deleted THE SYSTEM SHALL stop it. The user may receive the same reminder on several devices.
- **Business rules:** BR-MEM-05, BR-MEM-11
- **Verification method:** manual QA on device + unit tests of the reminder moment
- **Source:** Q5 answered 2026-10-14; existing constraint §2.7
- **Acceptance criteria:**
  ```gherkin
  Scenario: Reminder fires before the anniversary
    Given an entry on lunar 15/10 with a reminder 1 day before at 07:00
    And the next anniversary is 23/11/2026
    When it is 22/11/2026 07:00 on the device
    Then the device shows a reminder for that relative's anniversary

  Scenario: Reminder set on another device
    Given the user turned a reminder on from another device
    When the user next opens the memorial book on this device
    Then this device also reminds the user before that anniversary
  ```

**FR-MEM-009 — Add to the iOS reminders app**
- **Priority:** Could
- **Platforms:** iOS
- **Description:** WHEN the reminder is on THE SYSTEM SHALL offer "Thêm vào Lời nhắc (Reminders)"
  (add to the reminders app); WHEN it is on and the user saves THE SYSTEM SHALL add the anniversary to
  the iOS reminders app and confirm "Đã lưu. Lời nhắc giỗ <relationship> đã được thêm vào ứng dụng Lời
  nhắc." (saved, the reminder was added to the Reminders app).
- **Business rules:** BR-MEM-11
- **Verification method:** manual QA on device
- **Source:** iOS spec v1.3; Q4 answered 2026-10-14
- **Acceptance criteria:**
  ```gherkin
  @ios-only
  Scenario: Add to the reminders app
    Given the user is adding a relative with a reminder on
    And "add to the reminders app" is on
    When the user saves
    Then the anniversary appears in the iOS reminders app
    And the user is told it was added
  ```

#### Offline and media

**FR-MEM-010 — Read the book offline**
- **Priority:** Should
- **Platforms:** iOS, Android
- **Description:** WHEN the app cannot reach the server THE SYSTEM SHALL show the book as last loaded
  on that device, with next anniversaries computed for today, and SHALL refuse adding, editing and
  deleting with a message that a connection is needed.
- **Business rules:** BR-MEM-05
- **Verification method:** automated test + manual QA
- **Source:** iOS `AS-Memorial-006`; Q3 answered 2026-10-14
- **Acceptance criteria:**
  ```gherkin
  Scenario: Open the book offline
    Given the user opened the memorial book before
    And the device is offline
    When the user opens the memorial book
    Then the entries last loaded are shown

  Scenario: Change the book offline
    Given the device is offline
    When the user tries to save, edit or delete an entry
    Then the user is told a connection is needed
    And the book is unchanged

  Scenario: Never opened before, offline
    Given the user never opened the memorial book on this device
    And the device is offline
    When the user opens the memorial book
    Then the user is told the book cannot be loaded without a connection
  ```

**FR-MEM-011 — Portrait**
- **Priority:** Could (planned — blocked on photo storage)
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the user adds a portrait to an entry THE SYSTEM SHALL keep at most one photo per
  entry, show it in the book and the entry, let the user replace or remove it, and show the relative's
  initials when there is none. A photo is private like the rest of the entry.
- **Business rules:** BR-MEM-01
- **Verification method:** automated tests per platform once built
- **Source:** backend spec v2.4 (US-MEM-05, deferred); Q8 answered 2026-10-14
- **Acceptance criteria:**
  ```gherkin
  @not-confirmed
  Scenario: Add a portrait
    Given the user is editing "Nguyễn Văn A"
    When the user adds a photo and saves
    Then the book shows that photo for "Nguyễn Văn A"

  Scenario: No portrait
    Given an entry has no photo
    When the user opens the memorial book
    Then the entry shows the relative's initials
  ```

> ⚠️ Not confirmed: maximum photo size and accepted image types as the user experiences them (the
> backend spec plans 5 MB, JPEG/PNG/WebP, with the apps converting iPhone photos to JPEG) — blocks
> FR-MEM-011 on every platform; not needed until photo storage exists.

### 3.4 Non-functional requirements

- **NFR-MEM-01 — Responsiveness:** the book with 100 entries appears within 2 s on a 4G connection,
  and the preview of FR-MEM-006 updates as soon as a value changes. Platforms: Website, iOS, Android.
- **NFR-MEM-02 — Privacy:** names, notes and photos of relatives are never visible to anyone but the
  owner, never appear in logs or analytics, and photos are never publicly addressable. Platforms: all.
- **NFR-MEM-03 — Calendar correctness:** every platform passes every case in `rules/*.json`.
  Platforms: Website, iOS, Android.

### 3.5 Data requirements

| Data item | Meaning | Owned by | Retention / deletion | Private or public |
|---|---|---|---|---|
| Full name | Relative's name | User's account on the server | Until the user deletes the entry | Private |
| Relationship | One of BR-MEM-02's ten values | User's account | Same | Private |
| Date of death | Day, month, optional year, calendar, leap marker (BR-MEM-03/04) | User's account | Same | Private |
| Note | Free text ≤ 1000 characters | User's account | Same | Private |
| Reminder settings | On/off, lead, time | User's account | Same | Private |
| Add to iOS reminders app | The user's wish to also use the iOS reminders app | User's account (used on iOS only) | Same | Private |
| Portrait (planned) | One photo | User's account | Deleted with the entry | Private |
| Offline copy | Last loaded book | The device (apps only) | Replaced on each load | Private |
| Scheduled reminders | Device notifications | The device | Re-planned on each load | Private |

Data kept before this spec only in a browser (website) or app memory (Android) is not migrated (§2.7).

### 3.6 Platform variance

Intentional differences between platforms. Anything not listed here must behave identically.

| ID | Platforms | Difference | Reason |
|---|---|---|---|
| FR-MEM-008 | iOS, Android only | Reminders are delivered; the website delivers none | The website has no reliable notification channel (Q5, 2026-10-14) |
| FR-MEM-009 | iOS only | Can also add the anniversary to the iOS reminders app | Android and the website have no equivalent in this version (Q4, 2026-10-14) |
| FR-MEM-010 | iOS, Android only | Book readable offline | The website is online only (Q3, 2026-10-14) |
| BR-MEM-05…10 | Website, iOS, Android | Computed on the client; the backend does not compute anniversaries | Existing constraint (§2.7) |

### 3.7 Software system attributes

- **Reliability:** a wrong lunar conversion would store a wrong date for good; NFR-MEM-03 guards it.
- **Availability:** server unreachable → apps read offline (FR-MEM-010); the website shows an error;
  no platform loses data the user did not save.
- **Security & privacy:** NFR-MEM-02; another user's entry is indistinguishable from a missing one
  (BR-MEM-01).
- **Maintainability / portability:** the same vectors run on every platform (§4.3).

---

## 4. Verification & traceability

### 4.1 Verification approach

Computational rules (BR-MEM-03…11) are verified by running every case in `rules/*.json` in each
client's unit tests and, for BR-MEM-03/04 validation, the backend's. Expected values were generated
from the backend's lunar converter (BR-MEM-10). Functional requirements are verified by each
platform's automated tests mapped to the scenarios above; reminders and the iOS reminders app also
need manual QA on device.

### 4.2 Traceability matrix

Status per platform: `planned` · `in-progress` · `done` · `drift` (implemented differently — see
note) · `n/a` · `later`. Read on 2026-10-14 from each repo's `main`.

| ID | Backend | Website | iOS | Android | Note |
|---|---|---|---|---|---|
| BR-MEM-01 | done | drift | done | drift | Web keeps the book in the browser without sign-in; Android keeps it in memory only |
| BR-MEM-02 | done | in-progress | done | in-progress | Clients do not enforce the 100-entry limit message yet |
| BR-MEM-03 | done | in-progress | drift | drift | Clients offer only the last 96 years (needs 1900 → current year) |
| BR-MEM-04 | drift | drift | drift | drift | Backend accepts the leap marker only without a year (a known year is stored as a solar date, from which the marker is recoverable — Appendix B); clients have no leap choice and drop it |
| BR-MEM-05 | n/a | in-progress | in-progress | in-progress | Same algorithm on all three; not yet checked against vectors |
| BR-MEM-06 | n/a | in-progress | in-progress | in-progress | As above |
| BR-MEM-07 | n/a | in-progress | in-progress | in-progress | As above |
| BR-MEM-08 | n/a | in-progress | in-progress | in-progress | |
| BR-MEM-09 | n/a | in-progress | in-progress | in-progress | |
| BR-MEM-10 | n/a | in-progress | in-progress | in-progress | Each client has its own port of the algorithm |
| BR-MEM-11 | in-progress | in-progress | in-progress | in-progress | Backend accepts any lead 0–30 and any time; clients offer the fixed choices |
| BR-MEM-12 | done | planned | drift | planned | iOS deletes without asking |
| FR-MEM-001 | n/a | drift | done | drift | Web/Android do not require sign-in; 3 scenarios |
| FR-MEM-002 | n/a | in-progress | done | in-progress | Web/Android show local data; 4 scenarios |
| FR-MEM-003 | done | drift | done | drift | Not saved to the account on Web/Android; 6 scenarios |
| FR-MEM-004 | done | drift | done | drift | 3 scenarios |
| FR-MEM-005 | done | drift | drift | drift | No confirmation anywhere yet; 2 scenarios |
| FR-MEM-006 | n/a | in-progress | done | in-progress | 2 scenarios |
| FR-MEM-007 | done | in-progress | done | in-progress | 2 scenarios |
| FR-MEM-008 | n/a | n/a | planned | planned | iOS stub (`TD-Memorial-001`); 2 scenarios |
| FR-MEM-009 | n/a | n/a | planned | n/a | 1 scenario |
| FR-MEM-010 | n/a | n/a | done | planned | 3 scenarios |
| FR-MEM-011 | later | later | later | later | Blocked on photo storage (`docs/tech-debt/memorial/TD-001`); 2 scenarios |
| FR-MEM-012 | done | in-progress | done | in-progress | 2 scenarios |
| FR-MEM-013 | done | in-progress | done | in-progress | 2 scenarios |
| NFR-MEM-01 | planned | planned | planned | planned | Not measured |
| NFR-MEM-02 | done | planned | planned | planned | Backend does not log names/notes |
| NFR-MEM-03 | n/a | planned | planned | planned | Vectors added in v0.1 |

### 4.3 Test vectors

| File | Rule | Cases |
|---|---|---|
| `rules/next-anniversary.json` | BR-MEM-05, 06, 07, 08, 09 (ordinal) | 16 |
| `rules/anniversary-ordinal.json` | BR-MEM-09 (label) | 6 |
| `rules/death-date-conversion.json` | BR-MEM-03, 04, 10 | 9 |

---

## 5. Appendices

### Appendix A — Open questions

#### Open (unconfirmed)

> ⚠️ Not confirmed: a person who died today has a next anniversary of today with ordinal 0 (no
> label); should the book instead show next year's "Giỗ đầu"? — affects the list and preview on
> Website, iOS, Android (BR-MEM-09).

> ⚠️ Not confirmed: the website shows and lets the user edit reminder settings but delivers no
> reminder; is that intended, or should the website hide them? — blocks the website spec for FR-MEM-007.

> ⚠️ Not confirmed: maximum photo size and accepted image types as the user experiences them —
> blocks FR-MEM-011 on every platform (not needed until photo storage exists).

#### Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| The storage / client-computed / local-reminder decision has no ADR | Already decided on 2026-10-06; recorded as an existing constraint (§2.7), no ADR | 2026-10-14 |
| Leap months | A death may be recorded in a leap month; the anniversary is always in the regular month with the same number (BR-MEM-04, BR-MEM-05) | 2026-10-14 |
| Is the book tied to the account on every platform? | Yes; sign-in required on Website and Android too; earlier local data is discarded (BR-MEM-01, §2.7) | 2026-10-14 |
| Offline behavior | Apps read the last loaded book; no changes offline; website online only (FR-MEM-010) | 2026-10-14 |
| "Thêm vào Lời nhắc (Reminders)" | iOS only in this version (FR-MEM-009, §3.6) | 2026-10-14 |
| Reminders | Delivered on every signed-in app device; none on the website; fixed lead and time choices, device local time (BR-MEM-11, FR-MEM-008) | 2026-10-14 |
| Delete without asking? | Ask for confirmation (BR-MEM-12) | 2026-10-14 |
| Year of death range | 1900 to the current year on every platform (BR-MEM-03) | 2026-10-14 |
| Portrait | In the spec as planned (FR-MEM-011) | 2026-10-14 |
| Creating entries from the family tree | Out of scope; owned by `genealogy` | 2026-10-14 |
| Calendar authority and "today" | Backend lunar converter is the reference; today is the device's date (BR-MEM-08, BR-MEM-10) | 2026-10-14 |

### Appendix B — Contract handoff (backend)

For `spec-writer` (backend) and the contract; the product does not dictate the shape.

1. **Leap marker with a known year (BR-MEM-04).** No contract change is required: a known-year date
   is stored as a solar date, and converting it back yields the lunar day, month, year and leap marker
   one-to-one. The contract description should say so, so clients stop dropping the marker.
2. **Reminder choices (BR-MEM-11).** The product offers lead 0/1/3/7 days and times
   06:00/07:00/12:00/19:00. The contract currently accepts any lead 0–30 and any minute; the backend
   spec decides whether to keep that wider range (it must at least accept the product's choices).
3. **Reminder time zone.** The product uses the device's local time; the contract's time-zone value
   should be the device's zone (default `Asia/Ho_Chi_Minh` stays for clients that send none).
4. **Leap-month validation with a known year.** The backend currently cannot reject "10/6 nhuận/2026"
   because the client converts before sending; the client rejects it (BR-MEM-04). No server change
   unless the backend starts converting.
5. **Portrait (FR-MEM-011).** Already designed in the backend spec (deferred, `TD-001`).
