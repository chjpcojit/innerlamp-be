# Product Spec — Lamp lighting and lamp garden (Thắp đèn và Vườn đèn)

| Field | Value |
|---|---|
| Document type | Product spec — Software Requirements Specification, IEEE 29148-2018 structure |
| Feature | Lamp lighting and lamp garden (`lamp`) |
| Feature code | `LMP` — IDs are `BR-LMP-NN`, `FR-LMP-NNN`, `NFR-LMP-NN` |
| Product | InnerLamp |
| Platforms in scope | Backend, Website, iOS, Android |
| Version | 1.0 |
| Last updated | 2026-10-16 |
| Status | confirmed by the product owner on 2026-10-16; open items are listed in Appendix A |
| Owner / author | Product owner: anh Đức · written by Claude (`ba-writer`, retroactive mode) |
| API contract | `docs/specs/api/openapi.yaml` (owned by the backend; this spec never restates it) |
| Architecture decisions | None needed — no hard-to-reverse decision is introduced (the data model and the service already exist). Lighting a lamp at a temple for a fee will need its own ADR when the offering feature is written. |
| Related product specs | `login` (`docs/product/login/product.md`) — sign-in, session ended, guest mode · `onboarding` — dharma name is optional on the account |
| Downstream | Backend: `docs/specs/features/lamp/` (`spec-writer`) · Website: `Docs/specs/features/lamp/` · iOS: `Docs/Specs/LightLamp/` · Android: `Docs/Specs/LightLamp/` |

> See `changelog.md` in this folder for the version history. Platform specs cite this file as
> `docs/product/lamp/product.md@<version>` plus the IDs they implement.

---

## 1. Introduction

### 1.1 Purpose

This document is the single source of product truth for lighting a virtual lamp and browsing the
garden of lamps on every InnerLamp platform. It replaces the backend SRS and the iOS SRS of the same
feature, and settles the places where the backend, the website and the iOS app had diverged.

### 1.2 Scope

**In scope:**
- Lighting a lamp: choosing a lamp type, an intention, a recipient, an optional prayer and how many
  days it burns, optionally hiding the user's name.
- A limit on how many lamps one user can have lit at once.
- A lamp going out by itself when its burning time is over.
- The lamp garden: every signed-in user sees the lit lamps of everyone, newest first, filtered by lamp
  type, loaded page by page, with the total and per-type counts.
- Marking the user's own lamps in the garden.
- Entering the feature from the main navigation of each platform.

**Out of scope:**
- "Thỉnh đèn tại chùa" (a temple lights a real lamp for a fee), payment, donation — `offering` feature.
  The lamp screens do not offer it until that feature is specified.
- A "my lamps" screen, expired-lamp history, notifications such as "your lamp is about to go out".
- Putting a lamp out early, extending, relighting, or editing a lamp.
- Moderating recipient or prayer text.
- Filtering the garden by intention.
- Lamp artwork and animation — the design prototype owns them.

### 1.3 Product overview

InnerLamp (Tâm Đăng) is a Vietnamese Buddhist practice app. Lighting a lamp is a small devotional act:
the user lights an oil lamp, a lotus candle or a floating flower lantern for someone, with a wish and a
prayer, and the lamp stays visible in a shared garden for the chosen number of days. The garden shows
the community that is practising at the same time. The lamp is symbolic and free.

### 1.4 Glossary

| Term | Definition | Not to be confused with |
|---|---|---|
| Đèn (lamp) | One virtual lamp lit by a user, with a type, an intention, a recipient, an optional prayer, a lighting moment and an end moment | A real lamp lit by a temple |
| Loại đèn (lamp type) | Đèn dầu (oil lamp), Nến sen (lotus candle) or Hoa đăng (floating flower lantern) | Tâm nguyện |
| Tâm nguyện (intention) | The kind of wish: Cầu an (peace), Cầu siêu (salvation of the departed), Hồi hướng (merit dedication), Tri ân (gratitude) | Lời nguyện |
| Dành cho (recipient) | Free text naming who the lamp is lit for; when blank the lamp is dedicated to all beings ("Tất cả chúng sanh") | The lighter |
| Lời nguyện (prayer) | Optional free text with the user's own words | Tâm nguyện |
| Thời gian đèn sáng (burn duration) | 1, 7, 21 or 49 days | The remaining time |
| Đèn đang sáng (lit lamp) | A lamp whose end moment has not passed | Expired lamp |
| Đèn đã tắt (expired lamp) | A lamp whose end moment has passed; it leaves the garden and no longer counts | — |
| Vườn đèn (lamp garden) | The list of everyone's lit lamps | The user's own lamps only |
| Người thắp (lighter) | The user who lit the lamp | The recipient |
| Vô danh (anonymous) | The label shown instead of a name when the lighter has no dharma name or chose to hide it | — |
| Pháp danh (dharma name) | The Buddhist name received at refuge; optional on the account | Họ và tên (full name) |

### 1.5 References

- Prior documents (deleted once the platform specs are approved, old versions stay in git history):
  backend `docs/specs/features/lamp/SRS.md` (v0.4, 27 decisions of 2026-10-10); iOS
  `Docs/Specs/LightLamp/SRS.md` and assumptions `AS-LightLamp-001…008`, tech debt
  `TD-LightLamp-001…005`; website note in `.assumptions.md`.
- Design prototype: the Thắp đèn screens of `InnerLamp Prototype` / `InnerLamp Web App` (copy of
  `Mo-ta-tinh-nang.md` in `innerlamp-iOS/Docs/innerlamp-project/project/docs/`; the Android copy is
  identical). The prototype says the center navigation button opens the meditation setup; the product
  owner decided on 2026-10-11 and 2026-10-16 that it opens Thắp đèn.
- Product owner answers of 2026-10-16 to the 14 questions in
  `/mnt/project-files/reports/product/lamp/2026-10-15-cau-hoi-truoc-khi-viet.md` (all defaults confirmed).
- Platform folder names: Backend `lamp` · Website `lamp` · iOS `LightLamp` · Android `LightLamp`
  (not built yet).

---

## 2. Overall description

### 2.1 Product perspective

A stand-alone devotional feature on top of the account: it needs a signed-in user and the account's
optional dharma name. It sits next to the practice tools and the home screen (entry points) and next to
the offering feature (a future paid temple lamp). Nothing outside InnerLamp is involved.

### 2.2 Product functions

- Open Thắp đèn from the main navigation (`FR-LMP-001`).
- Fill in and light a lamp (`FR-LMP-002`, `FR-LMP-003`).
- Limit lit lamps per user and absorb double submissions (`FR-LMP-004`, `FR-LMP-005`).
- Let a lamp go out by itself (`FR-LMP-006`).
- Browse, filter and page through the garden, see counts, see which lamps are mine
  (`FR-LMP-007` … `FR-LMP-011`).
- Handle empty, error and signed-out states (`FR-LMP-012`, `FR-LMP-013`).
- Keep paid temple lamps out of this version (`FR-LMP-014`).

### 2.3 User classes and characteristics

| Actor | Description | Permissions |
|---|---|---|
| Signed-in user | A person with an InnerLamp account who is signed in | Light lamps, browse the garden |
| Guest | Someone using the app or website without signing in | Sees a short introduction and a sign-in button; cannot light a lamp or see the garden |

### 2.4 Operating environment

| Platform | In scope | Notes |
|---|---|---|
| Backend | yes | Decides every time (lighting moment, end moment, which lamps are lit), enforces the limit, keeps lamps, serves the garden and the counts |
| Website | yes | Reached from the practice area and from the home celebration card |
| iOS | yes | Reached from the center navigation button and the home suggestion row |
| Android | yes | Not built yet; reached from the center navigation button |

Network expectation: online only on every platform. Lighting and the garden need the server; there is no
offline lamp and no local copy of lamps.

### 2.5 Constraints

- The set of lamp types, intentions and burn durations is closed; new values need a new version of this
  spec and new artwork.
- The lamp is free in this version; no payment concept appears in any lamp screen.
- The recipient is shown to every signed-in user, so the form must say so (`BR-LMP-10`).
- There is no moderation in this version (product decision 2026-10-10, confirmed 2026-10-16).

### 2.6 User documentation

N/A — a self-explanatory flow; the form carries the one notice that matters (the recipient is visible to others).

### 2.7 Assumptions and dependencies

- **Assumptions:**
  - Time is decided by the server. Clients never decide when a lamp was lit or when it goes out.
  - The account's full name is private; only the dharma name can appear in the garden.
- **Dependencies:**
  - `login` — the signed-in session and the "session ended" behavior.
  - `onboarding` — the dharma name is optional there.
  - `offering` (later) — the paid temple lamp and the "thỉnh đèn tại chùa" card.

---

## 3. Specific requirements

### 3.1 External interface requirements

#### 3.1.1 User interfaces

- **Light a lamp** — a form with: lamp type (3 choices, default oil lamp), intention (4 choices,
  default Cầu an), recipient ("Dành cho"), prayer ("Lời nguyện"), burn duration (1, 7, 21, 49 days,
  default 7), a switch "Ẩn tên tôi" (off by default). Under the recipient a notice: "Người khác sẽ thấy
  tên người nhận." A "Thắp đèn" button and a way to open the garden.
- **Lamp lit** — a confirmation showing the lamp, its type, intention and number of days, the
  recipient (or "Tất cả chúng sanh"), the prayer when there is one, until when it burns, and two
  actions: "Thắp thêm đèn" and "Vào vườn đèn".
- **Garden ("Vườn đèn")** — a header with the number of lit lamps ("N ngọn đèn đang sáng"), filter
  chips ("Tất cả" and one per lamp type, each with its count), one card per lamp (artwork, lamp type,
  lighter's name, recipient, remaining time), a "Xem thêm đèn" action, and a way back to lighting.
  The user's own lamps are highlighted and labelled "Của bạn" instead of the lighter's name.
- **Guest** — a short introduction to the lamp and a "Đăng nhập để thắp đèn" button instead of the form
  and the garden.
- **Messages** (Vietnamese copy; an English copy is each platform's concern):
  - limit reached: "Bạn đã có 20 ngọn đèn đang sáng, mức tối đa cho mỗi người. Hãy thắp thêm khi có đèn tắt."
  - invalid input: "Thông tin chưa hợp lệ. Vui lòng kiểm tra lại người nhận và lời nguyện."
  - lighting failed: "Chưa thắp được đèn. Vui lòng thử lại."
  - garden failed: "Chưa tải được vườn đèn. Vui lòng thử lại sau."
  - empty garden: "Vườn đèn còn trống" with "Thắp một ngọn đèn để nó sáng ở đây." and a button to light one.
  - session ended: the copy defined by the `login` spec.

#### 3.1.2 Hardware interfaces

N/A — no device capability beyond a screen.

#### 3.1.3 Software interfaces

The product needs from the backend: light a lamp for the signed-in user (and tell the user it was
lit, with its end moment); list the lit lamps of everyone, newest first, filtered by lamp type,
page by page, together with the total and per-type counts taken at the same moment, and telling which
lamps belong to the signed-in user. Intention and prayer of a lamp are never part of the garden. The
contract is `docs/specs/api/openapi.yaml`; see Appendix B for what changes.

#### 3.1.4 Communication interfaces

N/A — no email, SMS or push is part of this feature.

### 3.2 Business rules

**BR-LMP-01 — Lamp types and intentions are closed sets**
- **Rule:** A lamp has exactly one lamp type from {oil lamp, lotus candle, floating flower lantern}
  and exactly one intention from {peace, salvation of the departed, merit dedication, gratitude}.
  Anything else is not accepted.
- **Examples:** oil lamp + peace → accepted; "candle" → rejected; "wealth" → rejected.
- **Computational:** yes → `rules/lamp-input-validation.json`
- **Applies to:** Backend, Website, iOS, Android
- **Source:** backend SRS decision 2026-10-10, confirmed 2026-10-16

**BR-LMP-02 — Burn durations**
- **Rule:** The burn duration is 1, 7, 21 or 49 days. Any other number is not accepted.
- **Examples:** 7 → accepted; 5 → rejected; 0 → rejected.
- **Computational:** yes → `rules/lamp-input-validation.json`
- **Applies to:** Backend, Website, iOS, Android
- **Source:** backend SRS, iOS SRS, prototype; confirmed 2026-10-16

**BR-LMP-03 — The end moment**
- **Rule:** A lamp's end moment is its lighting moment plus N × 24 hours, N being the burn duration.
  Both moments are absolute instants decided by the server clock; the user's device clock and time
  zone never change them. Showing the end as a date and time uses the viewer's own device time zone.
- **Examples:** lit 2026-10-10 14:00 UTC for 1 day → ends 2026-10-11 14:00 UTC; the same lamp ends at
  21:00 on 11 October for a viewer in Vietnam (UTC+7); 2028-02-28 10:00 UTC + 1 day → 2028-02-29 10:00 UTC.
- **Computational:** yes → `rules/lamp-lifetime.json`
- **Applies to:** Backend (decides), Website, iOS, Android (display only)
- **Source:** backend SRS decisions 14 and 25; replaces the iOS calendar-day rule `AS-LightLamp-003`; confirmed 2026-10-16

**BR-LMP-04 — Lit and out**
- **Rule:** A lamp is lit while the current moment is earlier than its end moment. At the end moment
  and after, it is out: it no longer appears in the garden, no longer counts in any total or toward the
  user's limit, and cannot be brought back.
- **Examples:** one second before the end moment → lit; exactly at the end moment → out.
- **Computational:** yes → `rules/lamp-lifetime.json`
- **Applies to:** Backend, Website, iOS, Android
- **Source:** backend SRS decision 9, confirmed 2026-10-16

**BR-LMP-05 — At most 20 lit lamps per user**
- **Rule:** A user can have at most 20 lit lamps at the same time. Lamps that are out do not count.
  Two submissions at the same moment (for example from two devices) never push the user above 20.
  The number 20 is a setting of the service, not a constant clients embed.
- **Examples:** 19 lit + 1 new → accepted (20); 20 lit + 1 new → refused; 20 lamps of which 3 are out → accepted.
- **Computational:** no — verified by scenarios (it depends on the user's other lamps).
- **Applies to:** Backend (enforces), Website, iOS, Android (show the refusal)
- **Source:** backend SRS decisions 8 and 9, confirmed 2026-10-16

**BR-LMP-06 — The recipient**
- **Rule:** The recipient is optional. Spaces at both ends are removed first; at most 100 characters
  remain, counted as characters (after Unicode NFC normalization), not bytes. A blank recipient means
  the lamp is dedicated to all beings; every platform shows it as "Tất cả chúng sanh" in the viewer's
  language (the stored lamp does not carry that text).
- **Examples:** "  Bà ngoại  " → "Bà ngoại"; "" or "   " → all beings; 100 characters → accepted; 101 → rejected;
  100 Vietnamese letters typed as base letter plus combining marks → 100 characters, accepted.
- **Computational:** yes → `rules/lamp-input-validation.json`
- **Applies to:** Backend, Website, iOS, Android
- **Source:** product owner 2026-10-16 (supersedes the backend rule "recipient required"); limit from backend SRS decision 4

**BR-LMP-07 — The prayer**
- **Rule:** The prayer is optional. Spaces at both ends are removed; a blank prayer means no prayer.
  At most 300 characters, counted like the recipient.
- **Examples:** 300 characters → accepted; 301 → rejected; "  " → no prayer.
- **Computational:** yes → `rules/lamp-input-validation.json`
- **Applies to:** Backend, Website, iOS, Android
- **Source:** backend SRS decisions 5 and 13, confirmed 2026-10-16

**BR-LMP-08 — A double submission is one lamp**
- **Rule:** If the same user submits two lamps with identical content (lamp type, intention, recipient
  after trimming, prayer after trimming, burn duration, hidden-name choice) within 5 seconds, only one
  lamp is created and the user sees that lamp as lit, even when the second submission would have
  exceeded the limit. After 5 seconds an identical submission is a new lamp. The 5 seconds is a setting
  of the service.
- **Examples:** same content twice within 2 s → one lamp; same content after 6 s → two lamps;
  different recipient within 2 s → two lamps.
- **Computational:** no — verified by scenarios (stateful).
- **Applies to:** Backend (enforces), Website, iOS, Android (also ignore a second submission while lighting)
- **Source:** backend SRS decisions 12 and 24, confirmed 2026-10-16

**BR-LMP-09 — The lighter's name**
- **Rule:** The name other users see on a lamp is the lighter's dharma name as it was at the moment of
  lighting. It never changes afterwards, even if the dharma name is edited. When the lighter has no
  dharma name, or chose to hide the name, the label is "Vô danh". The account's full name is never shown.
  A hidden name is not sent to other users at all.
- **Examples:** dharma name "Thích Minh An", not hidden → "Thích Minh An"; hidden → "Vô danh"; no
  dharma name → "Vô danh"; name later changed to "Thích Tuệ Minh" → older lamps still read "Thích Minh An".
- **Computational:** yes → `rules/lamp-lighter-name.json`
- **Applies to:** Backend (decides), Website, iOS, Android (display)
- **Source:** backend SRS decisions 16, 22, 23, 26; full-name exclusion product owner 2026-10-16

**BR-LMP-10 — What other users see**
- **Rule:** The garden shows, for each lit lamp: the lighter's name (`BR-LMP-09`), the lamp type, the
  recipient (or all beings) and the end moment. The intention and the prayer are never shown in the
  garden, to anyone, including the lighter's own lamps there. The lighter sees their own intention and
  prayer only on the confirmation right after lighting. The lighting form tells the user that the
  recipient is visible to others.
- **Examples:** another user's lamp card has no intention and no prayer; the user's own lamp card
  has no intention and no prayer either.
- **Computational:** no — verified by scenarios.
- **Applies to:** Backend, Website, iOS, Android
- **Source:** backend SRS decisions 3 and 17; own lamps and notice: product owner 2026-10-16

**BR-LMP-11 — Garden order and counts**
- **Rule:** The garden lists lit lamps of all users, the most recently lit first. Filtering by lamp type
  narrows the list only. The total of lit lamps and the number of lit lamps per lamp type always cover
  all users and all types, ignore the filter, show 0 for a type with no lit lamp, and are taken at the
  same moment as the list they accompany. Paging never repeats or skips a lamp that stays lit, even
  while other lamps expire or are lit; a page holds at most 12 lamps.
- **Examples:** 5 oil lamps + 3 lotus candles lit, filter lotus candle → 3 listed, total still 8,
  counts 5 / 3 / 0; 30 lit lamps → pages of 12, 12, 6.
- **Computational:** no — verified by scenarios.
- **Applies to:** Backend (decides), Website, iOS, Android
- **Source:** backend SRS decisions 10, 11, 18, 27; confirmed 2026-10-16

**BR-LMP-12 — The remaining-time label**
- **Rule:** Each lit lamp card shows how long it still burns, from the lamp's end moment and the
  current moment. If 24 hours or more remain, the label is the remaining time in whole days rounded up
  ("Còn N ngày"). If less than 24 hours remain, the label is "Còn dưới 1 ngày". A lamp with no time left
  is out and is not shown (`BR-LMP-04`). The label is informational: which lamps are lit is always the
  server's answer when the garden loads.
- **Examples:** 168 h left → "Còn 7 ngày"; exactly 24 h → "Còn 1 ngày"; 24 h + 1 s → "Còn 2 ngày";
  23 h 59 m 59 s → "Còn dưới 1 ngày".
- **Computational:** yes → `rules/lamp-remaining-label.json`
- **Applies to:** Website, iOS, Android
- **Source:** product owner 2026-10-16 (replaces the iOS calendar-day label `AS-LightLamp-003`)

**BR-LMP-13 — A lamp is final**
- **Rule:** Once lit, a lamp cannot be put out, edited, extended or relit. To light more, the user lights
  a new lamp.
- **Examples:** a lit lamp has no "put out" action; an expired lamp has no "relight" action.
- **Computational:** no — policy.
- **Applies to:** Backend, Website, iOS, Android
- **Source:** backend SRS decisions 9 and 15, confirmed 2026-10-16

**BR-LMP-14 — Sign-in required**
- **Rule:** Lighting a lamp and seeing the garden (with its counts) need a signed-in user. A guest sees
  no lamp and no count. The lighter is always the signed-in user, never a name typed in the form.
- **Examples:** guest opens Thắp đèn → introduction and sign-in button; expired session → the
  session-ended behavior of `login`.
- **Computational:** no — policy.
- **Applies to:** Backend, Website, iOS, Android
- **Source:** backend SRS decision 2, confirmed 2026-10-16

### 3.3 Functional requirements

#### Entering the feature

**FR-LMP-001 — Open Thắp đèn**
- **Priority:** Must
- **Platforms:** Website, iOS, Android
- **Description:** WHEN the user chooses the lamp button in the main navigation (mobile apps), or the
  lamp in the practice area (website), THE SYSTEM SHALL open the light-a-lamp view. The home screen also
  offers a way in (the Vu Lan suggestion on iOS, the celebration card after finishing the day's practice
  on the website).
- **Business rules:** BR-LMP-14
- **Verification method:** manual QA per platform
- **Source:** product owner 2026-10-11 and 2026-10-16
- **Acceptance criteria:**
  ```gherkin
  Scenario: Center navigation button opens Thắp đèn
    Given a signed-in user on a mobile app main screen
    When the user chooses the lamp button in the main navigation
    Then the light-a-lamp view opens

  Scenario: Meditation setup remains reachable
    Given a signed-in user on a mobile app
    When the user looks for meditation
    Then it is reachable from the home screen and the meditation area, not from the lamp button
  ```

#### Lighting

**FR-LMP-002 — The light-a-lamp form**
- **Priority:** Must
- **Platforms:** Website, iOS, Android
- **Description:** WHEN the light-a-lamp view opens for a signed-in user, THE SYSTEM SHALL show an
  unlit lamp, the form of §3.1.1 with oil lamp, Cầu an and 7 days preselected and "Ẩn tên tôi" off,
  and the notice that the recipient is visible to others.
- **Business rules:** BR-LMP-01, BR-LMP-02, BR-LMP-06, BR-LMP-07, BR-LMP-09, BR-LMP-10
- **Verification method:** manual QA per platform
- **Source:** iOS SRS FR-001/002, website, product owner 2026-10-16
- **Acceptance criteria:**
  ```gherkin
  Scenario: Defaults
    Given a signed-in user
    When the light-a-lamp view opens
    Then oil lamp, Cầu an and 7 days are selected
    And the hide-name switch is off
    And the notice that the recipient is visible to others is shown

  Scenario: Choosing options
    Given the form is open
    When the user chooses the lotus candle
    Then only the lotus candle is selected and the lamp artwork shows a lotus candle
  ```

**FR-LMP-003 — Light the lamp**
- **Priority:** Must
- **Platforms:** Website, iOS, Android
- **Description:** WHEN the user submits the form, THE SYSTEM SHALL light the lamp for the signed-in
  user, show the lamp igniting and then the confirmation of §3.1.1. WHILE lighting, THE SYSTEM SHALL
  ignore further submissions. IF it fails, THE SYSTEM SHALL show the matching message and keep what the
  user typed.
- **Business rules:** BR-LMP-01 … BR-LMP-03, BR-LMP-06, BR-LMP-07, BR-LMP-09, BR-LMP-10
- **Verification method:** shared test vectors (`lamp-input-validation`, `lamp-lifetime`) + manual QA
- **Source:** backend SRS FR-001/002, iOS SRS FR-004/005/012
- **Acceptance criteria:**
  ```gherkin
  Scenario: Light a lamp for seven days
    Given a signed-in user with no lit lamps
    When the user lights an oil lamp for peace, for "Bà ngoại", with the prayer "Mong bà bình an", for 7 days
    Then the lamp ignites
    And the confirmation shows "Bà ngoại", the prayer, and that it burns until 7 days from now
    And the lamp appears in the garden

  Scenario: Recipient left empty
    Given the form is open
    When the user lights a lamp without a recipient
    Then the confirmation says the lamp is dedicated to "Tất cả chúng sanh"

  Scenario: Hide my name
    Given the user has the dharma name "Thích Minh An"
    When the user switches "Ẩn tên tôi" on and lights a lamp
    Then the garden shows "Vô danh" for that lamp to every user

  Scenario: Too long recipient
    Given the form is open
    When the user types a recipient of 101 characters
    Then the lamp is not lit
    And the invalid-input message is shown and the form keeps its values

  Scenario: Lighting fails
    Given the server cannot be reached
    When the user lights a lamp
    Then "Chưa thắp được đèn. Vui lòng thử lại." is shown
    And the form keeps its values and no lamp is added

  Scenario: Light another lamp
    Given a lamp has just been lit
    When the user chooses "Thắp thêm đèn"
    Then an empty form opens with an unlit lamp
    And the previous lamp type, intention and duration stay selected
  ```

**FR-LMP-004 — Limit of lit lamps**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the user already has 20 lit lamps and lights another, THE SYSTEM SHALL refuse
  it and show the limit message.
- **Business rules:** BR-LMP-04, BR-LMP-05
- **Verification method:** automated test on the backend, manual QA on clients
- **Source:** backend SRS FR-003
- **Acceptance criteria:**
  ```gherkin
  Scenario: Reach the limit
    Given a user with 19 lit lamps
    When the user lights one more
    Then the lamp is lit and the user has 20 lit lamps

  Scenario: Exceed the limit
    Given a user with 20 lit lamps
    When the user lights another
    Then no lamp is added
    And "Bạn đã có 20 ngọn đèn đang sáng…" is shown

  Scenario: Expired lamps free room
    Given a user with 20 lamps of which 3 are out
    When the user lights another
    Then the lamp is lit

  Scenario: Two devices at once
    Given a user with 19 lit lamps and two devices
    When both devices light a lamp at the same moment
    Then at most one lamp is added and the user never has more than 20
  ```

**FR-LMP-005 — Double submission is one lamp**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the same content is submitted twice by the same user within 5 seconds, THE
  SYSTEM SHALL keep one lamp and show it as lit.
- **Business rules:** BR-LMP-08
- **Verification method:** automated test on the backend, manual QA on clients
- **Source:** backend SRS FR-001
- **Acceptance criteria:**
  ```gherkin
  Scenario: Repeated submission
    Given a user has just submitted a lamp
    When the identical lamp is submitted again within 5 seconds
    Then only one lamp exists
    And the user sees the lamp that was lit

  Scenario: Repeated submission at the limit
    Given a user whose last lamp made 20 lit lamps, submitted 2 seconds ago
    When the identical lamp is submitted again
    Then the user sees that lamp as lit, not the limit message

  Scenario: Same lamp lit again deliberately
    Given a user lit a lamp 6 seconds ago
    When the user submits an identical lamp
    Then a second lamp is lit
  ```

#### Lifetime

**FR-LMP-006 — A lamp goes out by itself**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a lamp's end moment passes, THE SYSTEM SHALL stop listing it, stop counting it,
  and offer no way to extend or relight it.
- **Business rules:** BR-LMP-03, BR-LMP-04, BR-LMP-13
- **Verification method:** shared test vectors (`lamp-lifetime`) + automated test on the backend
- **Source:** backend SRS FR-004
- **Acceptance criteria:**
  ```gherkin
  Scenario: Lamp disappears at its end moment
    Given a lamp lit for 1 day
    When 24 hours have passed since it was lit
    Then it is not in the garden and not in the total

  Scenario: Still visible just before the end
    Given a lamp lit for 1 day
    When one minute remains
    Then it is in the garden with the label "Còn dưới 1 ngày"

  Scenario: End date follows the viewer's time zone
    Given a lamp whose end moment is 2026-10-17 18:00 UTC
    When a viewer in Vietnam opens the garden
    Then the end is shown as 18 October 2026, 01:00
  ```

#### Garden

**FR-LMP-007 — Browse the garden**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a signed-in user opens the garden, THE SYSTEM SHALL list the lit lamps of all
  users, most recent first, each with artwork, lamp type, lighter's name, recipient and remaining time.
- **Business rules:** BR-LMP-09, BR-LMP-10, BR-LMP-11, BR-LMP-12
- **Verification method:** shared test vectors (`lamp-remaining-label`, `lamp-lighter-name`) + manual QA
- **Source:** backend SRS FR-005, iOS SRS FR-008
- **Acceptance criteria:**
  ```gherkin
  Scenario: Everyone's lamps, newest first
    Given user A lit an oil lamp yesterday and user B lit a lotus candle today
    When a signed-in user opens the garden
    Then both lamps are listed with B's first
    And each shows lighter's name, lamp type, recipient and remaining time

  Scenario: Intention and prayer are private
    Given a user lit a lamp with the intention "Tri ân" and a prayer
    When another user opens the garden
    Then the card shows neither the intention nor the prayer

  Scenario: Lighter without a dharma name
    Given a user with no dharma name lit a lamp
    When another user opens the garden
    Then the card shows "Vô danh"

  Scenario: Later change of dharma name
    Given a lamp was lit while the dharma name was "Thích Minh An"
    When the dharma name is changed to "Thích Tuệ Minh"
    Then that lamp still shows "Thích Minh An"

  Scenario: Remaining time
    Given a lamp has 3 days and 2 hours left
    When the garden shows it
    Then the label reads "Còn 4 ngày"
  ```

**FR-LMP-008 — My lamps are marked**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the garden shows a lamp lit by the signed-in user, THE SYSTEM SHALL highlight it
  and label it "Của bạn" instead of the lighter's name. Intention and prayer stay hidden
  (`BR-LMP-10`). The mark does not depend on the device: the same account sees its lamps marked on
  every device and after reinstalling.
- **Business rules:** BR-LMP-10
- **Verification method:** automated test on the backend, manual QA on clients
- **Source:** product owner 2026-10-16
- **Acceptance criteria:**
  ```gherkin
  Scenario: Own lamp
    Given a user lit a lamp that is still burning
    When the user opens the garden
    Then that lamp is highlighted and labelled "Của bạn"

  Scenario: Another device
    Given a user lit a lamp on one device
    When the same user opens the garden on another device
    Then the lamp is highlighted there as well

  Scenario: Hidden name on an own lamp
    Given a user lit an anonymous lamp
    When the user opens the garden
    Then the lamp is labelled "Của bạn"
    And other users see "Vô danh"
  ```

**FR-LMP-009 — Filter by lamp type**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the user chooses a lamp type chip, THE SYSTEM SHALL list only lit lamps of that
  type from the first page; WHEN the user chooses "Tất cả", every type.
- **Business rules:** BR-LMP-11
- **Verification method:** automated test on the backend, manual QA on clients
- **Source:** backend SRS FR-006, iOS SRS FR-009
- **Acceptance criteria:**
  ```gherkin
  Scenario: Filter
    Given lit lamps of all three types exist
    When the user chooses "Nến sen"
    Then only lotus candles are listed, from the first page

  Scenario: No lamp of the type
    Given no lotus candle is lit
    When the user chooses "Nến sen"
    Then the list is empty and the counts still show every type
  ```

**FR-LMP-010 — Load more**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN more lamps exist than one page of 12, THE SYSTEM SHALL show a "Xem thêm đèn"
  action that appends the next page without repeating or skipping a lamp.
- **Business rules:** BR-LMP-11
- **Verification method:** automated test on the backend, manual QA on clients
- **Source:** backend SRS FR-007
- **Acceptance criteria:**
  ```gherkin
  Scenario: Next page
    Given 30 lit lamps exist
    When the user opens the garden and chooses "Xem thêm đèn"
    Then the first page shows 12 lamps and the next 12 are appended
    And no lamp appears twice

  Scenario: Last page
    Given the user has loaded every lit lamp
    Then "Xem thêm đèn" is no longer offered

  Scenario: A lamp expires between two loads
    Given the user has loaded the first page and a lamp on a later page expires
    When the user loads more
    Then the expired lamp is not shown and no other lamp is skipped or repeated

  Scenario: Filter resets paging
    Given the user has loaded two pages
    When the user changes the filter
    Then the list starts again from the first page of that type
  ```

**FR-LMP-011 — Counts**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN the garden is shown, THE SYSTEM SHALL show "N ngọn đèn đang sáng" with N the
  total of lit lamps, and on each chip the count of that type.
- **Business rules:** BR-LMP-11
- **Verification method:** automated test on the backend, manual QA on clients
- **Source:** backend SRS FR-008
- **Acceptance criteria:**
  ```gherkin
  Scenario: Counts
    Given 5 oil lamps, 3 lotus candles and 2 floating flower lanterns are lit
    When a signed-in user opens the garden
    Then the header reads "10 ngọn đèn đang sáng"
    And the chips show 5, 3 and 2

  Scenario: Counts ignore the filter
    Given 5 oil lamps and 3 lotus candles are lit
    When the user filters by lotus candle
    Then 3 lamps are listed and the header still reads 8

  Scenario: Only real lamps
    Given only 2 lamps are lit
    When the garden opens
    Then the header reads 2 — no sample lamps and no added base number
  ```

**FR-LMP-012 — Empty and error states**
- **Priority:** Must
- **Platforms:** Website, iOS, Android
- **Description:** WHEN no lit lamp matches, THE SYSTEM SHALL show the empty state with a way to light a
  lamp. WHEN the garden cannot be loaded, THE SYSTEM SHALL show the garden error message and keep
  lighting available.
- **Business rules:** BR-LMP-11
- **Verification method:** manual QA per platform
- **Source:** iOS SRS FR-013, website
- **Acceptance criteria:**
  ```gherkin
  Scenario: Empty garden
    Given no lamp is lit by anyone
    When the user opens the garden
    Then "Vườn đèn còn trống" is shown with a button to light a lamp

  Scenario: Garden cannot load
    Given the server cannot be reached
    When the user opens the garden
    Then "Chưa tải được vườn đèn. Vui lòng thử lại sau." is shown
    And the user can still open the light-a-lamp form
  ```

#### Access

**FR-LMP-013 — Guests and ended sessions**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a guest opens Thắp đèn, THE SYSTEM SHALL show the introduction and a sign-in
  button and no lamp data. WHEN the session has ended, THE SYSTEM SHALL behave as the `login` spec
  says.
- **Business rules:** BR-LMP-14
- **Verification method:** automated test on the backend, manual QA on clients
- **Source:** backend SRS FR-009, website
- **Acceptance criteria:**
  ```gherkin
  Scenario: Guest
    Given a guest
    When the guest opens Thắp đèn
    Then an introduction and "Đăng nhập để thắp đèn" are shown
    And no garden and no count are shown

  Scenario: Server refuses an unauthenticated request
    Given a request without a valid sign-in
    When it asks for the garden or to light a lamp
    Then it is refused and returns no lamp data
  ```

#### Scope boundary

**FR-LMP-014 — No paid temple lamp in this version**
- **Priority:** Must
- **Platforms:** Website, iOS, Android
- **Description:** THE SYSTEM SHALL NOT show "Thỉnh đèn tại chùa", a donation or payment control on any
  lamp screen until the offering feature is specified.
- **Business rules:** —
- **Verification method:** manual QA per platform
- **Source:** product owner 2026-10-16
- **Acceptance criteria:**
  ```gherkin
  Scenario: Confirmation has no temple lamp card
    Given a lamp has just been lit
    Then the confirmation offers only "Thắp thêm đèn" and "Vào vườn đèn"
  ```

### 3.4 Non-functional requirements

- **NFR-LMP-01 — Garden speed:** the garden with its counts appears within a target to be decided.
  Platforms: all.
  > ⚠️ Not confirmed: the garden response-time target and the expected number of lit lamps (to be measured after launch) — blocks the performance target and the counting strategy.
- **NFR-LMP-02 — Consistent counts:** the counts shown with a page come from the same moment as that
  page's list (`BR-LMP-11`). Platforms: Backend.
- **NFR-LMP-03 — Time is the server's:** no platform uses its own clock to decide whether a lamp is
  lit; the device clock only formats the remaining-time label and the end date (`BR-LMP-03`,
  `BR-LMP-12`). Platforms: all.

### 3.5 Data requirements

| Data item | Meaning | Owned by | Retention / deletion | Private or public |
|---|---|---|---|---|
| Lamp (type, recipient, lighting moment, end moment) | One lit lamp | User's account on the server | Kept after it goes out (history may come later); deleted when the account is deleted | Type, recipient, end moment, lighter's name: visible to all signed-in users while lit |
| Intention and prayer | The user's wish | User's account on the server | As the lamp | Private to the lighter; never in the garden |
| Lighter's name on the lamp | The dharma name at lighting, or none | The lamp | As the lamp | Visible to all signed-in users; absent when hidden |
| Hidden-name choice | Whether the lighter hid the name | The lamp | As the lamp | Not shown as such to others |

> ⚠️ Not confirmed: how long expired lamps are kept — blocks the retention rule (nothing is deleted until it is decided).

### 3.6 Platform variance

| ID | Platforms | Difference | Reason |
|---|---|---|---|
| FR-LMP-001 | Website vs mobile apps | Entry points differ: the website has no bottom navigation, so it opens from the practice area and the home celebration card | Each platform's navigation |

Anything not listed here behaves identically. Today's differences between the platforms and this spec
are drift (see §4.2), not variance.

### 3.7 Software system attributes

- **Reliability:** a lamp the user has been told is lit is not lost; its end moment never changes.
- **Availability:** with the server unreachable nothing in this feature works; lighting fails with a
  message and no lamp is stored on the device.
- **Security & privacy:** every request needs a signed-in user; the lighter comes from the session; a
  hidden name never leaves the server; recipient and prayer can contain a deceased relative's name, so
  they are not written to logs. No moderation means abusive text can appear in the garden — accepted risk.
- **Maintainability / portability:** a new lamp type or duration is a change to this spec, the
  service's settings and the artwork.

---

## 4. Verification & traceability

### 4.1 Verification approach

Computational rules run from `rules/*.json` on every platform that implements them (the backend runs
the end-moment, validation and name rules; the clients run the label rule and the validation rule for
their form checks). Stateful rules (limit, double submission, paging, counts) are verified by backend
tests and by scenarios on the clients. UI behavior is verified manually per platform.

### 4.2 Traceability matrix

Status per platform: `planned` · `in-progress` · `done` · `drift` · `n/a` · `later`. State as read on
2026-10-15 (backend, website, iOS, Android code).

| ID | Backend | Website | iOS | Android | Note |
|---|---|---|---|---|---|
| BR-LMP-01 | done | done | done | planned | |
| BR-LMP-02 | done | done | done | planned | |
| BR-LMP-03 | done | n/a | drift | planned | iOS counts calendar days locally |
| BR-LMP-04 | done | done | drift | planned | iOS expires by calendar day |
| BR-LMP-05 | done | done | drift | planned | iOS has no limit (local only) |
| BR-LMP-06 | drift | drift | drift | planned | Backend requires a recipient; web and iOS fill "Tất cả chúng sanh" on the client |
| BR-LMP-07 | done | done | drift | planned | iOS has no length limit |
| BR-LMP-08 | done | done | drift | planned | |
| BR-LMP-09 | done | drift | drift | planned | Web shows "Ẩn danh"; iOS shows "Của bạn" with no name |
| BR-LMP-10 | done | drift | drift | planned | iOS shows the intention on own cards; no recipient-visible notice anywhere |
| BR-LMP-11 | done | done | drift | planned | iOS lists own lamps only |
| BR-LMP-12 | n/a | drift | drift | planned | Web shows "còn 1 ngày" for any partial day; iOS counts calendar days |
| BR-LMP-13 | done | done | done | planned | |
| BR-LMP-14 | done | done | drift | planned | iOS works without sign-in |
| FR-LMP-001 | n/a | in-progress | done | planned | Android spec says the center button opens meditation |
| FR-LMP-002 | n/a | drift | drift | planned | No hide-name switch, no visibility notice; 2 scenarios |
| FR-LMP-003 | done | drift | drift | planned | iOS saves locally; 6 scenarios |
| FR-LMP-004 | done | done | drift | planned | |
| FR-LMP-005 | done | done | drift | planned | |
| FR-LMP-006 | done | done | drift | planned | |
| FR-LMP-007 | done | drift | drift | planned | Web adds 16 sample lamps and a base count of 1,284 |
| FR-LMP-008 | drift | drift | drift | planned | Backend has no own-lamp mark; web guesses from ids stored on the device |
| FR-LMP-009 | done | done | drift | planned | |
| FR-LMP-010 | done | done | drift | planned | |
| FR-LMP-011 | done | drift | drift | planned | See FR-LMP-007 for web |
| FR-LMP-012 | n/a | done | done | planned | |
| FR-LMP-013 | done | done | drift | planned | |
| FR-LMP-014 | n/a | drift | done | planned | Web links to the offering page |
| NFR-LMP-01 | planned | n/a | n/a | n/a | Target not confirmed |
| NFR-LMP-02 | done | n/a | n/a | n/a | |
| NFR-LMP-03 | done | done | drift | planned | |

### 4.3 Test vectors

| File | Rule | Cases |
|---|---|---|
| `rules/lamp-lifetime.json` | BR-LMP-03, BR-LMP-04 | 10 |
| `rules/lamp-remaining-label.json` | BR-LMP-12 | 11 |
| `rules/lamp-input-validation.json` | BR-LMP-01, BR-LMP-02, BR-LMP-06, BR-LMP-07 | 26 |
| `rules/lamp-lighter-name.json` | BR-LMP-09 | 5 |

---

## 5. Appendices

### Appendix A — Open questions

#### Open (unconfirmed)

> ⚠️ Not confirmed: the garden response-time target and the expected number of lit lamps — blocks NFR-LMP-01 and the counting strategy (to be measured after launch).

> ⚠️ Not confirmed: how long expired lamps are kept — blocks the retention rule in §3.5; deleting the account removes the lamps in the meantime.

> ⚠️ Not confirmed: whether a lit lamp can be shared (the website offers a "Chia sẻ" action on the confirmation when the browser supports it; iOS does not) — blocks FR-LMP-003 on every platform; no share is specified here until decided (see ADR-0004 for shared links).

> ⚠️ Not confirmed: English copy for every message of §3.1.1 and the English name of the anonymous label — blocks the English locales of the website and the apps.

#### Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| Scope: paid "thỉnh đèn tại chùa"? | Out of scope; hidden on every platform until the offering feature is specified; the website drops its link to the offering page | 2026-10-16 |
| Can guests see the garden? | No; guests see an introduction and a sign-in button | 2026-10-16 |
| Recipient left empty | Allowed; the lamp is dedicated to all beings, shown localized by each client; the backend stops requiring a recipient | 2026-10-16 |
| Which name is shown for the lighter? | Dharma name; "Vô danh" when none; the full name is never shown | 2026-10-16 |
| Hide-name switch | On every platform, labelled "Ẩn tên tôi", off by default | 2026-10-16 |
| How are own lamps marked? | The backend says which lamps are the user's; clients label them "Của bạn"; intention and prayer stay hidden; no "my lamps" screen | 2026-10-16 |
| How is the remaining time shown? | By the server's end moment; "Còn N ngày" rounded up, "Còn dưới 1 ngày" under 24 hours; the iOS calendar-day count is dropped | 2026-10-16 |
| What does the center navigation button open? | Thắp đèn on iOS and Android; meditation setup stays reachable from the home screen and the meditation area | 2026-10-16 |
| Lamps already stored on iOS | Dropped, not uploaded (the app is not released); iOS stops deleting lamps on sign-out | 2026-10-16 |
| Sample lamps and base count | No platform shows sample lamps or a base count | 2026-10-16 |
| Account deletion | Deletes the user's lamps | 2026-10-16 |
| Moderation | None in this version; the form tells the user the recipient is public | 2026-10-16 |
| Numbers | 20 lit lamps per user, 1/7/21/49 days, 100/300 characters, 12 per page, 5 seconds for a double submission — unchanged | 2026-10-16 |
| Android folder name | `Docs/Specs/LightLamp`; the website uses `Docs/specs/features/lamp` | 2026-10-16 |
| Page size, cursor paging, counts independent of the filter | Page of 12, paging that never repeats or skips a lamp, counts over all lit lamps | 2026-10-10 |

### Appendix B — Contract handoff (backend)

For `spec-writer` (backend) and the contract `docs/specs/api/openapi.yaml`:

1. The recipient becomes optional. A blank recipient is stored as "none"; the garden and the lighting
   response then carry no recipient text, and each client shows "Tất cả chúng sanh" in its language.
   (Today the service rejects a blank recipient.)
2. Each garden item says whether the signed-in user lit it (`BR-LMP-10`, `FR-LMP-008`); intention and
   prayer stay out of garden items.
3. Everything else already exists: lighting returns the lamp with its end moment and own intention and
   prayer; the garden returns a page, the total and per-type counts from one moment, with a cursor;
   the limit (20), the double-submission window (5 s), page size (12) and text limits are service settings.
4. A hidden name must stay absent from the garden response (it is not merely masked by the client).
