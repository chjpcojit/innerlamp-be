# Product Spec — Memorial book (Sổ giỗ chạp)

| Field | Value |
|---|---|
| Document type | Product spec — Software Requirements Specification, IEEE 29148-2018 structure |
| Feature | Memorial book / Sổ giỗ chạp (`memorial`) |
| Feature code | `MEM` — IDs are `BR-MEM-NN`, `FR-MEM-NNN`, `NFR-MEM-NN` |
| Product | InnerLamp |
| Platforms in scope | Backend, Website, iOS, Android |
| Version | 2.1 |
| Last updated | 2026-10-14 |
| Status | confirmed — reminders move to the server (ADR-0001, ADR-0002); no open items |
| Owner / author | Product owner: anh Đức · written by Claude (`ba-writer`, retroactive mode) |
| API contract | `docs/specs/api/openapi.yaml` (owned by the backend; this spec never restates it) |
| Architecture decisions | ADR-0001 (in-app notifications on every platform, push on the apps) · ADR-0002 (the backend computes and sends reminders) — both Accepted 2026-10-14 |
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
- InnerLamp reminds the user before each anniversary: in the user's notification list on every
  platform, and as a push notification on the apps.
- A portrait photo per relative (planned, not yet available — FR-MEM-011).

**Out of scope:**
- Sharing a book with other people, or editing it together (only the owner sees it).
- Creating entries from the family tree ("Lấy từ gia phả") — owned by the `genealogy` product spec.
- Adding anniversaries to the device's own reminders or calendar app (removed in v2.0, FR-MEM-009).
- The notification list itself (shared by several features; ADR-0001) — this spec only defines the
  memorial reminders that appear in it.
- Search, multiple photos per relative, soft delete / restore, export.

### 1.3 Product overview

InnerLamp (Tâm Đăng) is a Vietnamese Buddhist practice app: practice tools, a lunar calendar, a
library, temples and family features. Vietnamese families hold a death-anniversary ceremony (giỗ)
for each ancestor every year, usually on the lunar date of death. The memorial book is one of the
family features: it remembers each anniversary for the user, converts lunar dates to this year's
solar date, and reminds the user in time to prepare offerings. It is reached from the main navigation
(the home screen in the apps, next to the lunar calendar; the profile page on the website).

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
| Lời nhắc (reminder) | A notification InnerLamp sends before an anniversary (BR-MEM-11) | The device's own reminders app (not used, FR-MEM-009 removed) |
| Thông báo (notification list) | The user's list of notifications inside InnerLamp, the same on every platform (ADR-0001) | A push notification |
| Push notification | A notification the phone shows outside the app, on iOS and Android only | The notification list |
| Account time zone | The time zone InnerLamp uses for the user's reminders, taken from the user's device | The lunar calendar's UTC+7 (BR-MEM-10) |

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
- **Notifications** (ADR-0001): the notification list on every platform, push on the apps.

External capabilities, by role: push notifications on the apps, photo storage (planned).

### 2.2 Product functions

- Open the memorial book (FR-MEM-001)
- See every relative with the next anniversary, days left and which anniversary it is (FR-MEM-002)
- Add, edit and delete a relative (FR-MEM-003, FR-MEM-004, FR-MEM-005, FR-MEM-012, FR-MEM-013)
- Preview the next anniversary while entering a date (FR-MEM-006)
- Set and receive reminders (FR-MEM-007, FR-MEM-008)
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
| Backend | yes | Keeps each user's book privately, enforces BR-MEM-01…04 and BR-MEM-12, computes when reminders are due and sends them (ADR-0002) |
| Website | yes | Full book with reminder settings; reminders appear in the notification list; no push, no offline (§3.6) |
| iOS | yes | Full book, reminders (notification list + push), offline reading |
| Android | yes | Full book, reminders (notification list + push), offline reading |

Network expectation: online to change the book; the apps can read the last loaded book offline
(FR-MEM-010); the website is online only.

### 2.5 Constraints

- Cultural: the anniversary of a lunar date of death follows Vietnamese custom — always the regular
  (non-leap) lunar month with the same number (BR-MEM-05).
- Privacy: names and notes about deceased relatives are personal family data (NFR-MEM-02).

### 2.6 User documentation

N/A — the form explains itself (the preview of FR-MEM-006 states the day-30 rule).

### 2.7 Assumptions and dependencies

- **ADR-0002 (Accepted 2026-10-14):** InnerLamp's server decides when each reminder is due and sends
  it; the devices no longer schedule reminders themselves. This replaces the constraint of
  2026-10-06 ("the server only stores, the apps remind locally") recorded here in v1.x. The clients
  still compute the next anniversary to show it (BR-MEM-05…09); the server computes it too, for
  reminders, with the same rules and vectors.
- **ADR-0001 (Accepted 2026-10-14):** reminders reach the user through the notification list on every
  platform and, on the apps, a push notification.
- **Assumptions:**
  - The book is new on every platform except the backend and iOS; book data kept only in a browser
    (website) or in memory (Android) before this spec is discarded, as the product is not released
    (Q2, 2026-10-14).
- **Dependencies:** sign-in on every platform; one shared Vietnamese lunar conversion (BR-MEM-10);
  the notification list and push (ADR-0001); the account time zone (BR-MEM-11); photo storage for
  FR-MEM-011 (not available yet).

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
- **Navigation.** Main navigation → book → add/edit → back to book with a confirmation message
  (where the book is reached from: §3.6). Choosing a memorial reminder in the notification list, or
  opening its push notification, opens that entry.

Layout: the design prototype (§1.5). User-facing copy (Vietnamese, with an English gloss). A platform
may adjust punctuation and line breaks but not the meaning; `{…}` is filled in by InnerLamp.

| Situation | Copy |
|---|---|
| Saved | "Đã lưu vào sổ giỗ." (saved to the memorial book) |
| Deleted | "Đã xóa {tên} khỏi sổ giỗ." (removed {name} from the memorial book) |
| Confirm delete | "Xóa {tên} khỏi sổ giỗ?" with "Xóa" (delete) and "Hủy" (cancel); "Không thể khôi phục sau khi xóa." (cannot be restored) |
| Book is full | "Sổ giỗ đã đủ 100 người. Hãy xóa bớt trước khi thêm mới." (the book holds 100 people; delete one first) |
| Name empty | "Vui lòng nhập họ và tên" (please enter the full name) |
| Name too long | "Họ và tên tối đa 100 ký tự" (at most 100 characters) |
| Note too long | "Ghi chú tối đa 1000 ký tự" (at most 1000 characters) |
| Date does not exist | "Ngày này không có trong lịch. Vui lòng kiểm tra lại." (this date does not exist in the calendar) |
| Leap month not in that year | "Năm {năm} không có tháng {tháng} nhuận." (year {year} has no leap month {month}) |
| Year before 1900 | "Năm mất phải từ 1900 trở về sau." (the year must be 1900 or later) |
| Date after today | "Ngày mất không thể sau hôm nay." (the date of death cannot be after today) |
| Could not save | "Chưa lưu được. Vui lòng thử lại." (not saved, please try again) |
| Could not delete | "Chưa xóa được. Vui lòng thử lại." (not deleted, please try again) |
| Connection needed to change | "Cần kết nối mạng để thay đổi sổ giỗ." (a connection is needed to change the book) |
| Showing the offline copy | "Đang hiện sổ giỗ đã tải lần trước. Không kết nối được máy chủ." (showing the last loaded book; the server cannot be reached) |
| Cannot load, no offline copy | "Chưa tải được sổ giỗ. Vui lòng kết nối mạng và thử lại." (the book cannot be loaded; connect and try again) |
| Entry no longer exists | "Mục này không còn trong sổ giỗ." (this entry is no longer in the book) |
| Leap-month choice | "Tháng nhuận" (leap month); a leap date is shown as "10/6 nhuận" |
| Push / notification, on the day | "Hôm nay là giỗ {quan hệ}" (today is the anniversary of {relationship}) |
| Push / notification, 1 day before | "Ngày mai là giỗ {quan hệ}" (tomorrow is the anniversary of {relationship}) |
| Push / notification, 3 or 7 days before | "Còn {n} ngày nữa là giỗ {quan hệ}" ({n} days until the anniversary of {relationship}) |
| Notification list detail (inside the app only) | "{tên} · {ngày dương, thứ}" ({name} · {solar date, weekday}) |
| Notifications not allowed on the device | "Bạn sẽ chỉ thấy lời nhắc trong mục Thông báo. Mở Cài đặt để bật thông báo đẩy." (you will only see reminders in the notification list; open Settings to allow push), with "Mở Cài đặt" (open Settings) |

{quan hệ} is the relationship in lower case (cha, mẹ, ông nội, …); for "khác" (other) it is "người
thân" (relative). The relative's name never appears in a push notification (NFR-MEM-02).

#### 3.1.2 Hardware interfaces

Push notifications on iOS and Android (FR-MEM-008); the photo library or camera for the portrait
(FR-MEM-011, planned).

#### 3.1.3 Software interfaces

The product needs the backend to: keep a signed-in user's memorial entries privately; create, read,
update and delete one entry; list all of a user's entries; reject invalid entries per BR-MEM-02…04;
know each user's account time zone; decide when each reminder is due and put it in the user's
notification list and push it to the user's apps (ADR-0001, ADR-0002); and (later) store one portrait
per entry. The contract is `docs/specs/api/openapi.yaml`; open contract needs are in Appendix B.

#### 3.1.4 Communication interfaces

Push notifications to iOS and Android through the operating system's push service (ADR-0001). No SMS
or email.

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
- **Applies to:** Backend (reminders, ADR-0002), Website, iOS, Android
- **Source:** Q1 answered 2026-10-14; prototype `gioNext`; backend resolved question 2026-10-07

**BR-MEM-06 — Day 30 in a short lunar month**
- **Rule:** When a lunar anniversary is on day 30 and that lunar month has 29 days in the target year,
  the anniversary falls on day 29 of that month.
- **Examples:** lunar 30/7, year 2026 (month 7 has 29 days) → lunar 29/7/2026 = 10/09/2026; lunar
  30/9, year 2026 (30 days) → 08/11/2026.
- **Computational:** yes → `rules/next-anniversary.json`
- **Applies to:** Backend (reminders), Website, iOS, Android
- **Source:** iOS spec v1.3; Android assumption #6; prototype

**BR-MEM-07 — 29 February in a common year**
- **Rule:** A solar anniversary on 29 February falls on 28 February in years that are not leap years.
- **Examples:** 29/2 in 2026 → 28/02/2026; in 2028 → 29/02/2028.
- **Computational:** yes → `rules/next-anniversary.json`
- **Applies to:** Backend (reminders), Website, iOS, Android
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
  20/12/2026 (27/01/2027) → "Giỗ đầu"; died on solar 14/10/2026 and today is 14/10/2026 → next
  anniversary today, ordinal 0, no label.
- **Computational:** yes → `rules/next-anniversary.json` (ordinal), `rules/anniversary-ordinal.json` (label)
- **Applies to:** Website, iOS, Android
- **Source:** iOS spec v1.3; Android assumption #8; prototype

**BR-MEM-10 — Calendar authority**
- **Rule:** All lunar ↔ solar conversion uses the Vietnamese lunar calendar computed at UTC+7
  (Hồ Ngọc Đức's algorithm), never a Chinese calendar from the operating system or a library. The
  backend's converter (lunar calendar feature) is the reference implementation from which the test
  vectors are generated; every platform must pass the same vectors.
- **Examples:** Tết 2026 = 17/02/2026; lunar 2025 has a leap month 6 starting 25/07/2025.
- **Computational:** yes → all files under `rules/`
- **Applies to:** Backend, Website, iOS, Android
- **Source:** Q10 answered 2026-10-14; backend spec v2.4 technical note

**BR-MEM-11 — Reminder settings and moment**
- **Rule:** Each entry has a reminder on or off. When on, the user chooses how early — on the day, 1,
  3 or 7 days before — and at what time — 06:00, 07:00, 12:00 or 19:00. A new entry starts with the
  reminder on, 1 day before, 07:00. The reminder moment is the next anniversary (BR-MEM-05…07) minus
  the lead, at the chosen time, in the **account time zone**; the lunar date itself is always the
  Vietnamese calendar (BR-MEM-10). The account time zone is the time zone of the device the user last
  used; it is Vietnam time (UTC+7) until a device reports one. One reminder is sent per anniversary,
  then the next one is planned. When an entry is added or changed after this anniversary's reminder
  moment has passed, no reminder is sent for this anniversary and the following one is planned.
- **Examples:** lunar 15/10, next 23/11/2026, 1 day before at 07:00, account in Vietnam →
  22/11/2026 07:00 Vietnam time; same entry, account in Los Angeles → 22/11/2026 07:00 Los Angeles
  time; entry added on 22/11/2026 at 20:00 → no reminder for 23/11/2026, next reminder 11/11/2027 07:00.
- **Computational:** yes → `rules/reminder-moment.json`
- **Applies to:** Backend, Website, iOS, Android
- **Source:** Q5 answered 2026-10-14; ADR-0002 (account time zone, already-passed moment); Q10

**BR-MEM-12 — Deletion is permanent**
- **Rule:** Deleting an entry removes it for good on every device; it cannot be restored. Its reminders
  already in the notification list are removed with it. The user is asked to confirm before it happens.
- **Examples:** confirm → entry gone everywhere; cancel → nothing changes.
- **Computational:** no
- **Applies to:** Backend, Website, iOS, Android
- **Source:** Q6 answered 2026-10-14 (replaces the prototype's delete-without-asking, iOS `AS-Memorial-004`)

**BR-MEM-13 — Late reminders**
- **Rule:** If InnerLamp could not send a reminder at its moment (for example during an outage), it
  sends it as soon as it can, as long as the anniversary day has not ended in the account time zone;
  after that the reminder for this anniversary is skipped and the following one is planned. A reminder
  is never sent twice for the same entry and anniversary.
- **Examples:** reminder due 22/11/2026 07:00, sent at 23/11/2026 09:00 (anniversary day) → sent;
  checked at 24/11/2026 00:30 → skipped, next reminder for 2027.
- **Computational:** yes → `rules/reminder-moment.json` (delivery cases)
- **Applies to:** Backend
- **Source:** ADR-0002 resolved question 2026-10-14

### 3.3 Functional requirements

#### Viewing the book

**FR-MEM-001 — Open the memorial book**
- **Priority:** Must
- **Platforms:** Website, iOS, Android
- **Description:** WHEN a signed-in user opens the memorial book from the platform's main navigation
  (§3.6) THE SYSTEM SHALL show that user's book; WHEN a guest tries to open it THE SYSTEM SHALL ask
  them to sign in first.
- **Business rules:** BR-MEM-01
- **Verification method:** automated UI/integration test per platform
- **Source:** iOS spec v1.3; Q2 answered 2026-10-14; client question 1 answered 2026-10-14
- **Acceptance criteria:**
  ```gherkin
  Scenario: Signed-in user opens the book
    Given the user is signed in
    When the user opens the memorial book from the main navigation
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
  SYSTEM SHALL show an empty state inviting the user to add a relative (and no option to import from
  the family tree, which is out of scope).
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
    And no option to take relatives from the family tree is offered
  ```

#### Changing the book

**FR-MEM-003 — Add a relative**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the user adds a relative with valid content THE SYSTEM SHALL save it to the
  user's book, return to the book and confirm "Đã lưu vào sổ giỗ." (saved to the memorial book).
  A new entry starts as: relationship ông nội, lunar calendar, day 1, month 1, year unknown, not a leap
  month, reminder on, 1 day before, 07:00 — on every platform.
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

  Scenario: Default reminder on every platform (BR-MEM-11)
    Given the user adds a relative on any platform without changing the reminder
    When the user saves
    Then the entry has a reminder on, 1 day before, at 07:00
  ```

**FR-MEM-004 — Edit a relative**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the user opens an entry, changes it and saves with valid content THE SYSTEM
  SHALL replace the entry's content (photo excluded), return to the book with a confirmation and
  re-plan its reminder (BR-MEM-11); WHEN the user goes back without saving THE SYSTEM SHALL keep the
  entry unchanged; WHEN the entry was deleted on another device THE SYSTEM SHALL say so and remove it
  from the book.
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

  Scenario: Edit after this anniversary's reminder moment (BR-MEM-11)
    Given today is 22/11/2026 at 20:00 and "Nguyễn Văn A" has its anniversary on 23/11/2026
    When the user changes the reminder to 1 day before at 07:00 and saves
    Then no reminder is sent for 23/11/2026
    And the next reminder is planned for the following anniversary
  ```

**FR-MEM-005 — Delete a relative**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the user chooses to delete an entry THE SYSTEM SHALL ask for confirmation;
  WHEN the user confirms THE SYSTEM SHALL delete it permanently, cancel its reminder, return to the
  book and confirm "Đã xóa {tên} khỏi sổ giỗ." (removed from the memorial book); WHEN the user cancels
  THE SYSTEM SHALL change nothing; WHEN the entry was already deleted on another device THE SYSTEM
  SHALL remove it from the book and say it no longer exists.
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

  Scenario: Delete an entry already deleted elsewhere
    Given "Nguyễn Văn A" was deleted on another device
    When the user deletes "Nguyễn Văn A" on this device and confirms
    Then "Nguyễn Văn A" is removed from the book
    And the user is told the entry is no longer in the book
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
  those choices. The book shows a "Có nhắc" (reminder on) mark on entries whose reminder is on, whether
  or not the device allows push notifications (the setting belongs to the account). WHEN an entry has
  a lead or time outside these choices (made before v2.0) THE SYSTEM SHALL show the nearest choice and
  keep the stored value until the user saves. Nearest choice: the closest lead in days, a tie going to
  the earlier reminder (2 → 3 days, 5 → 7 days); the closest time, a tie going to the earlier time
  (09:30 → 07:00), anything before 06:00 → 06:00, anything after 19:00 → 19:00. The book's reminder
  summary shows the same nearest choice.
- **Business rules:** BR-MEM-11
- **Verification method:** automated tests per platform
- **Source:** iOS spec v1.3; Q5 answered 2026-10-14; ADR-0002; client questions 2, 5, 8 answered 2026-10-14
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

  Scenario: Reminder settings on the website
    Given the user is editing "Nguyễn Văn A" on the website
    When the user chooses 7 days before at 06:00 and saves
    Then the apps show the same reminder settings for "Nguyễn Văn A"
  ```

**FR-MEM-008 — Reminder delivery**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN an entry's reminder moment arrives (BR-MEM-11, BR-MEM-13) THE SYSTEM SHALL put
  one reminder in the user's notification list, seen on every platform, and on the apps also send it
  as a push notification to every device where the user is signed in; the push shows only the
  relationship and the days left (§3.1.1), never the name. WHEN the user opens the reminder THE
  SYSTEM SHALL open that entry. WHEN the reminder is turned off or the entry is deleted THE SYSTEM
  SHALL send no further reminders for it, at once and on every device. WHEN the device does not allow
  push notifications THE SYSTEM SHALL still show the reminder in the notification list and tell the
  user how to allow push. WHEN the user signs out of a device THE SYSTEM SHALL stop pushing to it.
  WHEN the user turns notifications off in InnerLamp's account settings THE SYSTEM SHALL stop push
  notifications on every device of the account and still put reminders in the notification list; to
  stop a relative's reminders entirely the user turns that entry's reminder off (FR-MEM-007).
- **Business rules:** BR-MEM-05, BR-MEM-11, BR-MEM-13
- **Verification method:** shared vectors (`rules/reminder-moment.json`) on the backend + automated
  tests per platform + manual QA of push on device
- **Source:** Q5 answered 2026-10-14; ADR-0001; ADR-0002; client questions 5, 7, 11, 12 answered 2026-10-14; account switch answered 2026-10-14
- **Acceptance criteria:**
  ```gherkin
  Scenario: Reminder before the anniversary
    Given an entry "Nguyễn Văn A", relationship cha, on lunar 15/10 with a reminder 1 day before at 07:00
    And the account time zone is Vietnam time
    And the next anniversary is 23/11/2026
    When it is 22/11/2026 07:00 Vietnam time
    Then the user's notification list on every platform shows a reminder for "Nguyễn Văn A"
    And each signed-in app device receives a push notification "Ngày mai là giỗ cha"

  Scenario: Open a reminder
    Given the user has a reminder for "Nguyễn Văn A" in the notification list
    When the user opens it
    Then the entry "Nguyễn Văn A" is shown

  Scenario: Open a reminder for a deleted entry
    Given "Nguyễn Văn A" was deleted after its reminder was sent
    When the user opens that reminder
    Then the memorial book is shown
    And the user is told the entry is no longer in the book

  Scenario: Push not allowed on the device
    Given the device does not allow push notifications for InnerLamp
    When a reminder for "Nguyễn Văn A" is sent
    Then no push notification appears on that device
    And the reminder is in the notification list
    And the reminder settings show how to allow push notifications

  Scenario: Deleted entry is never reminded (BR-MEM-12)
    Given "Nguyễn Văn A" had a reminder planned
    When the user deletes "Nguyễn Văn A" on any device
    Then no reminder for "Nguyễn Văn A" is sent afterwards on any device

  Scenario: One reminder per anniversary (BR-MEM-13)
    Given the reminder for "Nguyễn Văn A" for 23/11/2026 was sent
    When InnerLamp checks reminders again that day
    Then no second reminder for 23/11/2026 is sent

  Scenario: Signed out device
    Given the user signed out of InnerLamp on a phone
    When a reminder is sent
    Then that phone receives no push notification

  Scenario: Notifications turned off for the account
    Given the user turned notifications off in InnerLamp's account settings
    When a reminder moment arrives for "Nguyễn Văn A"
    Then no device of the account receives a push notification
    And the reminder is in the notification list on every platform
  ```

**FR-MEM-009 — Add to the iOS reminders app**
- **Status:** Removed (v2.0) — reminders are sent by InnerLamp (ADR-0002); the app no longer adds
  anniversaries to the device's reminders app.

#### Offline and media

**FR-MEM-010 — Read the book offline**
- **Priority:** Should
- **Platforms:** iOS, Android
- **Description:** WHEN the app cannot reach the server, or the server fails, THE SYSTEM SHALL show
  the book as last loaded on that device by the same account, with next anniversaries computed for
  today and a line saying it is the offline copy, and SHALL refuse adding, editing and deleting with a
  message that a connection is needed. WHEN the user signs out THE SYSTEM SHALL remove the offline
  copy from the device.
- **Business rules:** BR-MEM-05
- **Verification method:** automated test + manual QA
- **Source:** iOS `AS-Memorial-006`; Q3 answered 2026-10-14; client questions 9, 11 answered 2026-10-14
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

  Scenario: Server error
    Given the user opened the memorial book before
    And the server fails to answer
    When the user opens the memorial book
    Then the entries last loaded are shown
    And the user is told the server cannot be reached

  Scenario: Sign out removes the offline copy
    Given the user opened the memorial book before
    When the user signs out and another account signs in on the same device while offline
    Then none of the first user's entries are shown
  ```

**FR-MEM-011 — Portrait**
- **Priority:** Could (planned — blocked on photo storage)
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the user adds a portrait to an entry THE SYSTEM SHALL keep at most one photo per
  entry, show it in the book and the entry, let the user replace or remove it, and show the relative's
  initials when there is none. A photo is a JPEG, PNG or WebP image of at most 5 MB; the apps convert
  photos taken in other formats (such as iPhone HEIC) to JPEG before saving, and a photo over 5 MB is
  refused with a message. A photo is private like the rest of the entry.
- **Business rules:** BR-MEM-01
- **Verification method:** automated tests per platform once built
- **Source:** backend spec v2.4 (US-MEM-05, deferred); Q8 answered 2026-10-14
- **Acceptance criteria:**
  ```gherkin
  Scenario: Add a portrait
    Given the user is editing "Nguyễn Văn A"
    When the user adds a photo and saves
    Then the book shows that photo for "Nguyễn Văn A"

  Scenario: No portrait
    Given an entry has no photo
    When the user opens the memorial book
    Then the entry shows the relative's initials

  Scenario: Photo too large
    Given the user is editing "Nguyễn Văn A"
    When the user adds a photo larger than 5 MB and saves
    Then the user is told the photo is too large
    And the entry keeps its previous portrait
  ```

### 3.4 Non-functional requirements

- **NFR-MEM-01 — Responsiveness:** the book with 100 entries appears within 2 s on a 4G connection,
  and the preview of FR-MEM-006 updates as soon as a value changes. Platforms: Website, iOS, Android.
- **NFR-MEM-02 — Privacy:** names, notes and photos of relatives are never visible to anyone but the
  owner, never appear in logs, analytics or push notifications, and photos are never publicly
  addressable. Platforms: all.
- **NFR-MEM-03 — Calendar correctness:** every platform passes every case in `rules/*.json` that
  applies to it (§4.3). Platforms: Backend, Website, iOS, Android.
- **NFR-MEM-04 — Reminder punctuality:** a reminder is in the notification list within one hour after
  its moment (BR-MEM-11), except during an outage (BR-MEM-13). Platforms: Backend.

### 3.5 Data requirements

| Data item | Meaning | Owned by | Retention / deletion | Private or public |
|---|---|---|---|---|
| Full name | Relative's name | User's account on the server | Until the user deletes the entry | Private |
| Relationship | One of BR-MEM-02's ten values | User's account | Same | Private |
| Date of death | Day, month, optional year, calendar, leap marker (BR-MEM-03/04) | User's account | Same | Private |
| Note | Free text ≤ 1000 characters | User's account | Same | Private |
| Reminder settings | On/off, lead, time | User's account | Same | Private |
| Account time zone | Time zone used for reminders (BR-MEM-11) | User's account, reported by the device | Until the account is deleted | Private |
| Planned reminder | The next reminder moment of each entry | User's account on the server | Replaced when sent or when the entry changes; gone with the entry | Private |
| Memorial reminder in the notification list | One per entry and anniversary (ADR-0001) | User's account | Kept after reading; paged one month at a time; removed when the entry is deleted (BR-MEM-12) | Private |
| Portrait (planned) | One photo | User's account | Deleted with the entry | Private |
| Offline copy | Last loaded book | The device (apps only) | Replaced on each load; removed on sign-out | Private |

Data kept before this spec only in a browser (website) or app memory (Android) is not migrated (§2.7).

### 3.6 Platform variance

Intentional differences between platforms. Anything not listed here must behave identically.

| ID | Platforms | Difference | Reason |
|---|---|---|---|
| FR-MEM-001 | Website | The book is opened from the profile page; the apps open it from the home screen | Website navigation has no home grid (client question 1, 2026-10-14) |
| FR-MEM-008 | iOS, Android | Reminders also arrive as push notifications; the website shows them only in the notification list | The website has no push (ADR-0001) |
| FR-MEM-010 | iOS, Android only | Book readable offline; the website shows an error when the server cannot be reached | The website is online only (Q3, client question 9, 2026-10-14) |
| BR-MEM-05…10 | Backend vs clients | The clients compute the next anniversary to show it; the backend computes it to send reminders; both pass the same vectors | ADR-0002 |

### 3.7 Software system attributes

- **Reliability:** a wrong lunar conversion would store a wrong date for good; NFR-MEM-03 guards it.
- **Availability:** server unreachable → apps read offline (FR-MEM-010); the website shows an error;
  no platform loses data the user did not save. Reminders missed during an outage are caught up the
  same day (BR-MEM-13).
- **Reliability of reminders:** one reminder per entry and anniversary, never two (BR-MEM-13); a push
  that fails to arrive still leaves the reminder in the notification list (ADR-0001).
- **Security & privacy:** NFR-MEM-02; another user's entry is indistinguishable from a missing one
  (BR-MEM-01).
- **Maintainability / portability:** the same vectors run on every platform (§4.3).

---

## 4. Verification & traceability

### 4.1 Verification approach

Computational rules (BR-MEM-03…11, 13) are verified by running the `rules/*.json` cases that apply to
each platform (§4.3) in its unit tests. Expected values were generated from the backend's lunar
converter (BR-MEM-10). Functional requirements are verified by each platform's automated tests mapped
to the scenarios above; push notifications also need manual QA on device.

### 4.2 Traceability matrix

Status per platform: `planned` · `in-progress` · `done` · `drift` (implemented differently — see
note) · `n/a` · `later`. Read on 2026-10-14 from each repo's `main`.

| ID | Backend | Website | iOS | Android | Note |
|---|---|---|---|---|---|
| BR-MEM-01 | done | drift | done | drift | Web requires sign-in but keeps the book in the browser; Android keeps it in memory only |
| BR-MEM-02 | done | in-progress | done | in-progress | Clients do not enforce the 100-entry limit message yet |
| BR-MEM-03 | done | in-progress | drift | drift | Clients offer only the last 96 years (needs 1900 → current year); iOS turns lunar day 30 into 29 when saving |
| BR-MEM-04 | done | drift | drift | drift | Backend keeps the marker through the stored solar date (Appendix B); clients have no leap choice and drop it |
| BR-MEM-05 | planned | in-progress | in-progress | in-progress | Backend computes it for reminders from v2.0 (ADR-0002); clients not yet checked against vectors |
| BR-MEM-06 | planned | in-progress | in-progress | in-progress | As above |
| BR-MEM-07 | planned | in-progress | in-progress | in-progress | As above |
| BR-MEM-08 | n/a | in-progress | in-progress | in-progress | |
| BR-MEM-09 | n/a | in-progress | in-progress | in-progress | |
| BR-MEM-10 | done | in-progress | in-progress | in-progress | Backend converter is the reference; each client has its own port |
| BR-MEM-11 | drift | planned | in-progress | in-progress | Backend accepts any lead 0–30 and any time and keeps a time zone per entry (needs account time zone); website has no settings yet |
| BR-MEM-12 | done | planned | drift | planned | iOS deletes without asking |
| BR-MEM-13 | planned | n/a | n/a | n/a | New in v2.0 |
| FR-MEM-001 | n/a | drift | drift | drift | Web/Android do not use the account book; iOS shows an empty book to guests; 3 scenarios |
| FR-MEM-002 | n/a | in-progress | drift | drift | iOS and Android show a family-tree option on the empty book; 4 scenarios |
| FR-MEM-003 | done | drift | done | drift | Not saved to the account on Web/Android; 7 scenarios |
| FR-MEM-004 | done | drift | in-progress | drift | 4 scenarios |
| FR-MEM-005 | done | drift | drift | drift | No confirmation anywhere yet; 3 scenarios |
| FR-MEM-006 | n/a | in-progress | done | in-progress | 2 scenarios |
| FR-MEM-007 | drift | planned | in-progress | in-progress | Backend: wider choices than the product (Appendix B); 3 scenarios |
| FR-MEM-008 | planned | planned | drift | planned | iOS schedules local notifications (stub); all of it is new in v2.0 (ADR-0001/0002); 7 scenarios |
| FR-MEM-009 | drift | n/a | drift | drift | Removed (v2.0): the backend still stores the wish; iOS has the option; Android shows an iOS-only option — all to remove |
| FR-MEM-010 | n/a | n/a | drift | planned | iOS: offline copy not tied to the account and kept when the session expires (sign-out already clears it); 5 scenarios |
| FR-MEM-011 | later | later | later | later | Blocked on photo storage (`docs/tech-debt/memorial/TD-001`); 3 scenarios |
| FR-MEM-012 | done | in-progress | in-progress | in-progress | 2 scenarios |
| FR-MEM-013 | done | in-progress | in-progress | in-progress | 2 scenarios |
| NFR-MEM-01 | n/a | planned | planned | planned | Not measured; clients only |
| NFR-MEM-02 | done | planned | planned | drift | Backend does not log names/notes; Android debug log prints bodies and its database is in cloud backup |
| NFR-MEM-03 | planned | planned | planned | planned | Backend runs the vectors from v2.0 |
| NFR-MEM-04 | planned | n/a | n/a | n/a | New in v2.0 |

### 4.3 Test vectors

| File | Rule | Cases |
|---|---|---|
| `rules/next-anniversary.json` | BR-MEM-05, 06, 07, 08, 09 (ordinal) | 17 |
| `rules/anniversary-ordinal.json` | BR-MEM-09 (label) | 6 |
| `rules/death-date-conversion.json` | BR-MEM-03, 04, 10 | 9 |
| `rules/reminder-moment.json` | BR-MEM-11 (moment), BR-MEM-13 (late delivery) | 7 + 3 delivery |

Who runs what: the clients run `next-anniversary`, `anniversary-ordinal` and `death-date-conversion`;
the backend runs `death-date-conversion`, `next-anniversary` (anniversary date only) and
`reminder-moment`.

---

## 5. Appendices

### Appendix A — Open questions

#### Open (unconfirmed)

None.

#### Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| The storage / client-computed / local-reminder decision has no ADR | Already decided on 2026-10-06; recorded as an existing constraint (§2.7), no ADR — superseded by ADR-0002 in v2.0 | 2026-10-14 |
| Leap months | A death may be recorded in a leap month; the anniversary is always in the regular month with the same number (BR-MEM-04, BR-MEM-05) | 2026-10-14 |
| Is the book tied to the account on every platform? | Yes; sign-in required on Website and Android too; earlier local data is discarded (BR-MEM-01, §2.7) | 2026-10-14 |
| Offline behavior | Apps read the last loaded book; no changes offline; website online only (FR-MEM-010) | 2026-10-14 |
| "Thêm vào Lời nhắc (Reminders)" | iOS only in this version (FR-MEM-009, §3.6) | 2026-10-14 |
| Reminders | Delivered on every signed-in app device; none on the website; fixed lead and time choices, device local time (BR-MEM-11, FR-MEM-008) | 2026-10-14 |
| Delete without asking? | Ask for confirmation (BR-MEM-12) | 2026-10-14 |
| Year of death range | 1900 to the current year on every platform (BR-MEM-03) | 2026-10-14 |
| Portrait | In the spec as planned (FR-MEM-011) | 2026-10-14 |
| Creating entries from the family tree | Out of scope; owned by `genealogy` | 2026-10-14 |
| A person who died today: today with no label, or next year's "Giỗ đầu"? | Today, ordinal 0, no label (BR-MEM-09) | 2026-10-14 |
| Reminder settings on the website | Not shown on the website in this version; added later (§3.6) | 2026-10-14 |
| Portrait size and types | JPEG, PNG or WebP, at most 5 MB; the apps convert other formats to JPEG (FR-MEM-011) | 2026-10-14 |
| Calendar authority and "today" | Backend lunar converter is the reference; today is the device's date (BR-MEM-08, BR-MEM-10) | 2026-10-14 |
| Who sends reminders, and where? (supersedes "Reminders" and "Thêm vào Lời nhắc" above) | The server, into the notification list on every platform, plus push on the apps (ADR-0001, ADR-0002, FR-MEM-008); the device reminders app is dropped (FR-MEM-009 removed) | 2026-10-14 |
| Reminder settings on the website (supersedes the row above) | Shown on the website like on the apps (FR-MEM-007) | 2026-10-14 |
| Reminder time zone | The account time zone, from the user's device (BR-MEM-11, ADR-0002) | 2026-10-14 |
| Reminder moment already passed when an entry is added or changed | No reminder for this anniversary; plan the next (BR-MEM-11) | 2026-10-14 |
| Late reminder after an outage | Sent while the anniversary day lasts, otherwise skipped (BR-MEM-13) | 2026-10-14 |
| Where the website opens the book (client question 1) | From the profile page; the apps from the home screen (§3.6) | 2026-10-14 |
| Entry created on the website: which reminder? (client question 2) | The default, 1 day before at 07:00 (FR-MEM-003) | 2026-10-14 |
| Website copy "nhắc trước 1–7 ngày" (client question 4) | Kept, the website now has reminder settings | 2026-10-14 |
| Push not allowed on the device (client question 5) | Reminder still in the notification list; a hint to open Settings; "Có nhắc" still shown (FR-MEM-007, FR-MEM-008) | 2026-10-14 |
| Old reminder values outside the choices (client question 8) | Show the nearest choice, keep the value until saved (FR-MEM-007) | 2026-10-14 |
| Server error while loading (client question 9) | Apps show the offline copy with a message; the website shows an error (FR-MEM-010) | 2026-10-14 |
| Sign-out on a shared device (client question 11) | Stop push to that device and remove the offline copy (FR-MEM-008, FR-MEM-010) | 2026-10-14 |
| Lock-screen content (client question 12) | Relationship only, never the name (§3.1.1, FR-MEM-008) | 2026-10-14 |
| Saving or deleting an entry deleted elsewhere (client question 13) | Remove it from the book and say "Mục này không còn trong sổ giỗ." (FR-MEM-004, FR-MEM-005) | 2026-10-14 |
| "Lấy từ gia phả" on the empty book (client question 14) | Hidden (FR-MEM-002) | 2026-10-14 |
| User-facing copy (client question 15) | Drafted in §3.1.1 for review | 2026-10-14 |
| Turning notifications off in the account settings: push only, or the notification list too? | Push only, on every device of the account; reminders stay in the notification list. Turning push off in the phone's settings affects only that phone (FR-MEM-008) | 2026-10-14 |

### Appendix B — Contract handoff (backend)
For `spec-writer` (backend) and the contract; the product does not dictate the shape.

1. **Leap marker with a known year (BR-MEM-04).** No contract change is required: a known-year date
   is stored as a solar date, and converting it back yields the lunar day, month, year and leap marker
   one-to-one. The contract description should say so, so clients stop dropping the marker.
2. **Reminder choices (BR-MEM-11).** The product offers lead 0/1/3/7 days and times
   06:00/07:00/12:00/19:00, and the server now sends reminders at whole hours (ADR-0002). The contract
   should accept only these choices; how existing entries with other values are handled is the
   backend spec's call (the clients show the nearest choice, FR-MEM-007).
3. **Account time zone (BR-MEM-11, ADR-0002).** The clients report the device's time zone for the
   account (at sign-in and when it changes); the per-entry time zone goes away. Default: Vietnam time.
4. **Leap-month validation with a known year.** The backend currently cannot reject "10/6 nhuận/2026"
   because the client converts before sending; the client rejects it (BR-MEM-04). No server change
   unless the backend starts converting.
5. **Portrait (FR-MEM-011).** Already designed in the backend spec (deferred, `TD-001`); the product
   limits (5 MB, JPEG/PNG/WebP) match it.
6. **Device reminders app (FR-MEM-009 removed).** The entry no longer carries the wish to use the iOS
   reminders app; the contract drops it.
7. **Notification list (ADR-0001, FR-MEM-008).** List the user's notifications newest first, one month
   per page; unread count; mark one or all as read; a memorial reminder carries what the clients need
   to open the entry and to show the copy of §3.1.1.
8. **Push registration (ADR-0001).** The apps register their push address after sign-in and when it
   changes, and remove it at sign-out.
9. **Account notification switch (ADR-0001).** Turns off push for the whole account; the notification
   list keeps receiving reminders (FR-MEM-008).
