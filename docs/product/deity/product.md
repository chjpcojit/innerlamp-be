# Product Spec — Buddhas & Bodhisattvas (Chư Phật & Bồ Tát)

| Field | Value |
|---|---|
| Document type | Product spec — Software Requirements Specification, IEEE 29148-2018 structure |
| Feature | Buddhas & Bodhisattvas catalogue: featured deities, the whole catalogue with a type filter, the deity page (`deity`) |
| Feature code | `DTY` — IDs are `BR-DTY-NN`, `FR-DTY-NNN`, `NFR-DTY-NN` |
| Product | InnerLamp |
| Platforms in scope | Backend, Website, iOS, Android |
| Version | 1.0 |
| Last updated | 2026-10-15 |
| Status | confirmed — 5 open items (Appendix A) |
| Owner / author | Product owner: anh Đức · written by Claude (`ba-writer`, retroactive mode) |
| API contract | `docs/specs/api/openapi.yaml` (owned by the backend; this spec never restates it) |
| Architecture decisions | none needed (catalogue is read-only and public; see §2.7) |
| Related product specs | `calendar` (`docs/product/calendar/product.md`, code `CAL`) — the deity's vía days are calendar events (BR-DTY-05) · `temple` (`TPL`) — same catalogue conventions |
| Downstream | Backend: `docs/specs/features/deity/` · Website: `Docs/specs/features/deity/` · iOS: `Docs/Specs/Deity/` · Android: `Docs/Specs/Deity/` |

> See `changelog.md` in this folder for the version history. Platform specs cite this file as
> `docs/product/deity/product.md@<version>` plus the IDs they implement.

---

## 1. Introduction

### 1.1 Purpose

This document is the single source of product truth for InnerLamp's catalogue of Buddhas,
bodhisattvas and devas on every platform: what a visitor can browse and read about each of them, how
their vía (memorial) days are shown, and which practice actions the deity page offers.

### 1.2 Scope

**In scope:**

- A short list of **featured deities** for the home view and the library.
- The **whole catalogue**, page by page (12 per page), with a **type filter**
  ("Tất cả · Chư Phật · Bồ Tát").
- A **deity page**: profile, next vía day with countdown, biography, great vows, iconography,
  mantras and titles, related sutras.
- Two practice actions on the deity page: **recite with the mala** and **share**.
- Loading the catalogue content (operator task; no user-facing editing).

**Out of scope:**

- Search in the catalogue (anh Đức, 2026-10-15: not needed for a catalogue of about 20 entries).
- "Báo thông tin sai" (report wrong information) — dropped, as in `temple` (anh Đức, 2026-10-15).
- Favourite, follow, prayer requests, reminders for a deity's vía day.
- The festival calendar itself (`calendar`) and the sutra reader (`sutra`); this feature only links
  to them.
- Adding, editing or removing deities from the back office.
- A dedicated share layout (the prototype's three-card share screen).

### 1.3 Product overview

InnerLamp is a Buddhist companion app (website, iOS, Android) with a practice library, a lunar
calendar, temples and family features. The Buddhas & Bodhisattvas catalogue is the public, read-only
part of the library that introduces each Buddha, bodhisattva and deva: who they are, their vows, their
vía days and the mantra used to recite their name. It is reachable from the home view and the library
without an account, and it leads into practice (mala), the festival calendar and the sutras.

### 1.4 Glossary

| Term | Definition | Not to be confused with |
|---|---|---|
| Deity | Any entry of the catalogue; its **type** is Phật, Bồ Tát or Chư Thiên (BR-DTY-01) | — |
| Phật (Buddha) | Type of an entry, e.g. Đức Phật Thích Ca, Phật A Di Đà; filter chip "Chư Phật" | — |
| Bồ Tát (bodhisattva) | Type of an entry, e.g. Quán Thế Âm, Địa Tạng; filter chip "Bồ Tát" | — |
| Chư Thiên (deva) | Type of an entry for heavenly protectors, e.g. Đại Cát Tường Thiên Nữ; no filter chip | "Thiên hộ pháp", "Thiên Nữ" — former platform labels, replaced (BR-DTY-01) |
| Ngày vía (vía day) | A yearly lunar day honouring the deity (birth, enlightenment, …); defined as an event of the festival calendar | A memorial anniversary in `memorial` |
| Lần vía kế tiếp (next vía) | The deity's vía occurrence nearest to today, today included (BR-DTY-06) | — |
| Đại nguyện (great vows) | The deity's numbered vows, e.g. the 48 vows of Phật A Di Đà | — |
| Hình tượng (iconography) | Posture, seat or mount, sacred objects held and their meaning | The deity's image |
| Thần chú & danh hiệu (mantras and titles) | The honorific title, the short mantra (Sino-Vietnamese and Sanskrit) and the dhāraṇī of the deity | The sutras linked to the deity |
| Kinh liên quan (related sutras) | Sutras of the sutra library linked to the deity | The canonical source text line |
| Featured deities (nổi bật) | The first entries of the catalogue order (BR-DTY-02) | A popularity ranking |

### 1.5 References

- Design prototype: `InnerLamp Prototype.dc.html`, screens `buddhas` ("Chư Phật & Bồ Tát"),
  `buddhasEmpty` ("Chư Phật & Bồ Tát · trống"), `buddha` ("Chi tiết Phật / Bồ Tát"), the home section
  "Thư viện Phật & Bồ Tát" and the library tile "Phật & Bồ Tát"; copy used:
  `innerlamp-Android/Docs/innerlamp-project/project/` (identical to the iOS copy). The web adaptation
  `innerlamp-website/Docs/innerlamp-web-adaptation/project/InnerLamp Web App.dc.html` has the same
  sections without the empty state. `Mo-ta-tinh-nang.md` (both versions) does not describe this
  feature.
- Prior documents (rewritten into this spec): backend `docs/specs/features/deity/` v2.3
  (`requirements.md`, `api-contract.md` v1.3); iOS `InnerLamp/Presentation/Scenes/Deity/README.md`,
  `Scenes/Deities/README.md`, `Docs/Assumptions/Deity/AS-Deity-DEVA-label.md`; website
  `.assumptions.md` lines 12, 13, 23 (stale); Android `Docs/home_assumptions.md` rows 1, 11, 13.
  No platform had a feature spec.
- Platform folder names: Backend `deity` · Website `deity` · iOS `Deity` (merges the former
  `Deity` and `Deities` scene docs) · Android `Deity`. Code folders keep their own names (iOS
  `Deity`, `Deities`; Android `buddha`; website `library/buddhas`).

---

## 2. Overall description

### 2.1 Product perspective

Retroactive spec of a feature that already runs in part: the backend serves the catalogue; the website
and iOS show it from the backend; Android shows nine sample entries. This version changes, for every
platform: one label for the deva type, pages of 12, the vía days taken from the festival calendar with
the next one counted down everywhere, related sutras taken from the sutra library's links, a real
empty / error state instead of placeholder or hard-coded content, and no "report wrong information".

Touches: `calendar` (vía days are its events; the deity page opens it), `sutra` (related sutras),
`practice` / mala (recite the mantra), `home` and `library` (featured blocks and entry tiles).

### 2.2 Product functions

- Featured deities on the home view and in the library (`FR-DTY-001`)
- The whole catalogue, page by page, with a type filter (`FR-DTY-002`, `FR-DTY-003`)
- The deity page and its sections (`FR-DTY-004` … `FR-DTY-007`)
- Recite with the mala and share (`FR-DTY-008`, `FR-DTY-009`)
- Empty and error states (`FR-DTY-010`)
- Catalogue content loading by the operator (`FR-DTY-011`)

### 2.3 User classes and characteristics

| Actor | Description | Permissions |
|---|---|---|
| Visitor | Anyone, signed in or not | Reads everything in this feature; uses mala and share |
| Operator | Runs the content load | Loads and updates the catalogue; not a user-facing role |

### 2.4 Operating environment

| Platform | In scope | Notes |
|---|---|---|
| Backend | yes | Stores the catalogue, orders and filters it, serves pages, links deities to their vía events and sutras |
| Website | yes | Catalogue pages with page numbers; deity page |
| iOS | yes | Home and library blocks, catalogue with continuous loading, deity page |
| Android | yes | Same as iOS; today sample data only |

Network expectation: online. The apps may keep the last loaded content readable offline (§3.6).

### 2.5 Constraints

- Religious content — vía days, mantras, dhāraṇī, great vows, biographies — must be accurate; an
  error is a sensitive content defect, not a cosmetic one. No review step exists yet:
  > ⚠️ Not confirmed: who reviews the deity content (biography, vows, mantras, iconography) and the image rights before public release, and how — blocks public release of the content, not the platform specs.
- Vía days follow the Vietnamese lunar calendar and its placement rules as defined by `calendar`
  (BR-CAL-04, BR-CAL-07); this feature never computes a lunar date on its own.

### 2.6 User documentation

N/A — a read-only catalogue needs no help content.

### 2.7 Assumptions and dependencies

- **Assumptions:** the catalogue is public and read-only for users and changes only when the operator
  reloads it. It is small (about 20 entries today). No architecture decision is needed (anh Đức
  accepted the "no ADR" proposal, 2026-10-15); linking a deity to calendar events and to sutras is a
  content link inside existing catalogues, not a new data model.
- **Dependencies:** `calendar` (the vía events and their occurrences); `sutra` (the deity ↔ sutra
  links and the sutra page); the mala practice tool on every client; the device share function on the
  apps.

---

## 3. Specific requirements

### 3.1 External interface requirements

#### 3.1.1 User interfaces

| View | Platforms | Shows |
|---|---|---|
| Home block "Thư viện Phật & Bồ Tát" | all clients | Featured deity cards (FR-DTY-001) and "Xem tất cả" (see all) to the catalogue |
| Library block "Chư Phật & Bồ Tát" and tile "Phật & Bồ Tát" | all clients | The same featured cards and an entry to the catalogue |
| Catalogue "Chư Phật & Bồ Tát" | all clients | Title, "{N} vị · tiểu sử, đại nguyện, thần chú" ({N} deities · biography, vows, mantras), chips "Tất cả · Chư Phật · Bồ Tát", deity cards, more pages (FR-DTY-002, FR-DTY-003) |
| Deity card | all clients | Image, type label, name, core virtue, "Vía dd/MM âm lịch" of the next vía (BR-DTY-06) |
| Deity page "Chi tiết Phật / Bồ Tát" | all clients | Sections of FR-DTY-004…009 in the order of FR-DTY-005 |
| Empty / error state | all clients | FR-DTY-010 |

Layout and styling are defined by the prototype, not here.

#### 3.1.2 Hardware interfaces

N/A — no device capability beyond the share function of the apps (FR-DTY-009).

#### 3.1.3 Software interfaces

The backend must: give the featured deities; give one page of the catalogue, optionally narrowed to one
type, with the number of matches; give one deity's profile; give for each deity its vía events with
their next occurrence (on cards and on the page); give the sutras linked to a deity. Contract:
`docs/specs/api/openapi.yaml`, tag Deities. The type filter, the profile and the linked sutras exist;
page size 12 and the vía days from `calendar` do **not** yet (Appendix B).

#### 3.1.4 Communication interfaces

N/A.

### 3.2 Business rules

**BR-DTY-01 — Types and their labels**
- **Rule:** Every deity has exactly one type: Phật, Bồ Tát or Chư Thiên. The type label shown on cards
  and on the deity page is "Phật", "Bồ Tát" or "Chư Thiên", the same on every platform. The filter
  offers "Tất cả" (all, no filter), "Chư Phật" and "Bồ Tát", one choice at a time; there is no
  "Chư Thiên" chip, so a Chư Thiên deity appears only under "Tất cả".
- **Examples:** Đại Cát Tường Thiên Nữ → label "Chư Thiên", listed under "Tất cả" only; Quán Thế Âm →
  "Bồ Tát", listed under "Tất cả" and "Bồ Tát".
- **Computational:** no
- **Applies to:** Backend, all clients
- **Source:** anh Đức, 2026-10-15; replaces the website label "Thiên hộ pháp" and the iOS label
  "Thiên Nữ" (AS-Deity-DEVA) — drift
- > ⚠️ Not confirmed: whether a "Chư Thiên" chip is added once the catalogue holds more than one deva — no platform is blocked now.

**BR-DTY-02 — Catalogue order and featured deities**
- **Rule:** Deities are always ordered by their editorial position in the catalogue file (first
  first), then by their stable identifier so the order never changes between two loads. The same order
  is used for the featured list, every catalogue page and every filtered result. The featured list is
  the first 10 deities of that order, whatever their type; fewer than 10 → all of them; none → the
  block shows its empty state. How many featured cards a block shows is a choice of each client's
  layout (§3.6); a client that shows fewer shows the first ones, in order.
- **Examples:** 20 deities → featured = positions 1–10 (today 6 Buddhas then the first 4 bodhisattvas); filter
  "Bồ Tát" keeps the editorial order among bodhisattvas; two deities with the same position → the one
  with the smaller identifier first.
- **Computational:** yes → `rules/deity-order.json`
- **Applies to:** Backend (clients show the order they receive)
- **Source:** backend v1.0 behaviour; anh Đức, 2026-10-15 ("Top trả 10 còn tuỳ nền tảng và UI hiển
  thị")
- > ⚠️ Not confirmed: whether the editorial file order is the intended "featured" criterion (the source has no popularity measure) — the order rule stands; only the content of the featured list may change.

**BR-DTY-03 — Page size**
- **Rule:** One catalogue page holds **12** deities. The last page may hold fewer. Asking for a page
  past the last one gives an empty page, not an error. The number of deities shown follows the
  current filter.
- **Examples:** 20 deities → 2 pages, the 2nd has 8; filter "Chư Phật" with 7 Buddhas → 1 page of 7;
  0 deities → 0 pages.
- **Computational:** no
- **Applies to:** Backend, all clients
- **Source:** anh Đức, 2026-10-15 (was 20 on the backend)

**BR-DTY-04 — Missing values**
- **Rule:** On the deity page a field or section without content is hidden, never shown empty: a
  missing honorific, Sanskrit or Chinese name, core virtue, pure land or canonical source line is
  left out; a section without content (biography, vows, iconography, mantras and titles, related
  sutras) is not shown; each mantra card without text is not shown. On a card, a missing core virtue
  is left out. A deity without an image shows a placeholder image. A deity without a vía day shows no
  vía line on its card and no vía card on its page.
- **Examples:** no dhāraṇī → the "Đà-la-ni" card is absent, the other mantra cards stay; no linked
  sutra → no "Kinh liên quan" section; no image → placeholder.
- **Computational:** no
- **Applies to:** all clients
- **Source:** website and iOS behaviour (both hide empty values); prototype (optional values omitted).
  Note: `temple` shows "Đang cập nhật" instead (BR-TPL-06) — a deliberate difference, since a deity
  page has no fixed set of rows.

**BR-DTY-05 — Vía days come from the festival calendar**
- **Rule:** A deity's vía days are the festival-calendar events linked to that deity (e.g. "Vía Quan
  Thế Âm đản sinh", "Vía Quan Thế Âm thành đạo", "Vía Quan Thế Âm xuất gia" for Quán Thế Âm). Their
  dates, their placement in leap and short months, and their solar dates each year are those of the
  calendar (BR-CAL-04, BR-CAL-07, BR-CAL-08). No platform keeps a separate list of vía dates and no
  client computes them.
- **Examples:** the calendar places "vía Dược Sư" on the last day of lunar month 9 → the deity page of
  Phật Dược Sư shows the same day; a correction of a vía date in the calendar shows on the deity page
  without any change to the deity.
- **Computational:** no (placement is covered by `calendar` vectors `event-placement.json`)
- **Applies to:** Backend (link and occurrences), all clients (show what they receive)
- **Source:** anh Đức, 2026-10-15 ("Theo calendar"); replaces the deity's own vía list on the backend,
  the hard-coded dates on the website and Android, and the iOS on-device lunar scan — drift

**BR-DTY-06 — Next vía**
- **Rule:** The next vía of a deity is, among the occurrences of all its vía events, the one with the
  earliest solar date on or after today (today included). Ties on the same day go to the event first in
  the calendar's display order (BR-CAL-12). "Today" is the device date, as for the calendar countdown
  (BR-CAL-13). The card shows the lunar date of the next vía ("Vía dd/MM âm lịch"); the page shows the
  next vía's name, lunar day and month, solar date with weekday, and the countdown of BR-CAL-13
  ("Hôm nay", "Ngày mai", "Còn n ngày").
- **Examples:** Quán Thế Âm with occurrences on 2026-04-05, 2026-08-01 and 2026-10-29, today
  2026-10-15 → next vía 2026-10-29, "Còn 14 ngày"; today equals an occurrence → that one, "Hôm nay";
  all occurrences of this year passed → the earliest occurrence of next year.
- **Computational:** yes → `rules/deity-next-via.json`
- **Applies to:** Backend (gives the next occurrence) or clients (pick it from the occurrences they
  receive) — the contract decides which; every platform that picks it runs the vectors
- **Source:** anh Đức, 2026-10-15 ("Lần vía gần nhất sắp tới"); replaces the iOS rule "first event
  whose name contains 'Đản' or 'Vía'" — drift
- The vectors' dates are inputs (occurrences as the calendar gives them); they assert only the choice
  and the countdown, never a lunar conversion.

**BR-DTY-07 — Great vows collapsed**
- **Rule:** When a deity has more than 5 great vows, the page shows the first 5 and an action
  "Xem đủ N đại nguyện" (show all N vows) that expands the list, then "Thu gọn" (collapse). With 5 or
  fewer, all are shown and no action appears. Vows are numbered from 1 in their stored order.
- **Examples:** 48 vows → 5 shown + "Xem đủ 48 đại nguyện"; 4 vows → 4 shown, no action.
- **Computational:** no
- **Applies to:** all clients
- **Source:** prototype `buddha` (ĐẠI NGUYỆN); anh Đức, 2026-10-15

**BR-DTY-08 — No report, favourite or follow action**
- **Rule:** The deity page shows no "Báo thông tin sai" link, no favourite and no follow action. The
  footer keeps only the editorial note "Nội dung do Ban biên tập tổng hợp từ kinh điển" (content
  compiled by the editorial board from the scriptures).
- **Examples:** the iOS footer loses its plain-text "Báo thông tin sai".
- **Computational:** no
- **Applies to:** all clients
- **Source:** anh Đức, 2026-10-15 (report dropped, as in `temple` BR-TPL-08)

### 3.3 Functional requirements

#### Browsing

**FR-DTY-001 — Featured deities**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a visitor opens the home view or the library THE SYSTEM SHALL show featured
  deity cards (BR-DTY-02) and a "Xem tất cả" action that opens the catalogue; choosing a card opens the
  deity page.
- **Business rules:** BR-DTY-02, BR-DTY-04, BR-DTY-06
- **Verification method:** automated test on each platform; vectors `deity-order.json` on the backend
- **Source:** backend v1.0; prototype home section and library block
- **Acceptance criteria:**
  ```gherkin
  Scenario: Featured deities on the home view
    Given the catalogue holds 20 deities
    When a visitor who is not signed in opens the home view
    Then featured deity cards appear in the catalogue order, starting with the first deity
    And the block offers "Xem tất cả"

  Scenario: A featured card opens the deity page
    When the visitor chooses the card of "Quán Thế Âm" on the home view
    Then the page of "Quán Thế Âm" appears
  ```

**FR-DTY-002 — The whole catalogue, page by page**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a visitor opens the catalogue THE SYSTEM SHALL show the first 12 deities in the
  catalogue order with the number of deities found, and SHALL show the next pages on request: the
  website by page numbers, the apps by loading the next page as the visitor reaches the end of the list
  (§3.6).
- **Business rules:** BR-DTY-02, BR-DTY-03, BR-DTY-04, BR-DTY-06
- **Verification method:** automated test on each platform
- **Source:** backend v1.0; website; iOS (loads every page at once — drift, replaced)
- **Acceptance criteria:**
  ```gherkin
  Scenario: First page
    Given the catalogue holds 20 deities
    When a visitor opens the catalogue
    Then 12 deity cards appear, the first ones of the catalogue order
    And the header says "20 vị"

  Scenario: Last page
    Given the catalogue holds 20 deities
    When the visitor moves to the second page
    Then 8 deity cards appear and no further page is offered

  Scenario: Card content
    Given "Quán Thế Âm" has vía days in the festival calendar
    When its card is shown
    Then it shows its image, the label "Bồ Tát", its name, its core virtue and "Vía dd/MM âm lịch" of its next vía
  ```

**FR-DTY-003 — Type filter**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a visitor chooses "Chư Phật" or "Bồ Tát" THE SYSTEM SHALL show, from page 1, only
  the deities of that type, with their count; "Tất cả" removes the filter.
- **Business rules:** BR-DTY-01, BR-DTY-02, BR-DTY-03
- **Verification method:** automated test on each platform
- **Source:** backend v2.2 (type filter); website; iOS (filtered on the device — drift, replaced)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Filter Buddhas
    When the visitor chooses "Chư Phật"
    Then only deities labelled "Phật" appear, in the catalogue order, and the header gives their count

  Scenario: A deva only under "Tất cả"
    Given "Đại Cát Tường Thiên Nữ" is of type Chư Thiên
    When the visitor chooses "Bồ Tát"
    Then "Đại Cát Tường Thiên Nữ" is not listed
    When the visitor chooses "Tất cả"
    Then "Đại Cát Tường Thiên Nữ" is listed with the label "Chư Thiên"
  ```

#### Deity page

**FR-DTY-004 — Open a deity page**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a visitor opens a deity from the home view, the library or the catalogue THE
  SYSTEM SHALL show that deity's page without signing in; WHEN the deity does not exist THE SYSTEM SHALL
  say it was not found and offer a way back; WHEN loading fails THE SYSTEM SHALL offer "Thử lại".
- **Business rules:** BR-DTY-04
- **Verification method:** automated test on each platform
- **Source:** backend v1.0; prototype `buddha`; website (502 page on failure — replaced by retry)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Open from the catalogue
    When the visitor opens "Phật A Di Đà" from the catalogue
    Then the page of "Phật A Di Đà" appears without signing in

  Scenario: Unknown deity
    When a visitor opens a deity that is not in the catalogue
    Then the page says the deity was not found and offers a way back

  Scenario: Loading fails
    Given the backend cannot be reached
    When a visitor opens a deity page
    Then an error message with "Thử lại" appears
  ```

**FR-DTY-005 — Deity profile sections**
- **Priority:** Must
- **Platforms:** Website, iOS, Android (Backend provides the data)
- **Description:** WHEN a deity page is shown THE SYSTEM SHALL show, in this order: the image with the
  type label and pure land; the name, the honorific title, the Sanskrit name and the Chinese name; the
  next-vía card (FR-DTY-006); the core virtue and the canonical source line; "Tiểu sử" (biography),
  collapsed with "Đọc thêm / Thu gọn" when long; "Đại nguyện" (BR-DTY-07); "Hình tượng" (posture, seat
  or mount, sacred objects, symbolic meaning); "Thần chú & danh hiệu" with up to three cards — "Danh
  hiệu" (honorific title), "Thần chú" (short mantra, Sino-Vietnamese then Sanskrit), "Đà-la-ni"
  (dhāraṇī name and excerpt) — followed by the actions of FR-DTY-008 and FR-DTY-009; "Kinh liên quan"
  (FR-DTY-007); and the footer of BR-DTY-08. Missing values follow BR-DTY-04.
- **Business rules:** BR-DTY-04, BR-DTY-07, BR-DTY-08
- **Verification method:** automated test on each client; manual QA against the prototype
- **Source:** prototype `buddha`; website deity page (adds "Hình tượng", kept by anh Đức
  2026-10-15); iOS Deity README
- **Acceptance criteria:**
  ```gherkin
  Scenario: Full profile
    Given a deity has a biography, 48 great vows, iconography and a short mantra
    When its page is shown
    Then "Tiểu sử", "Đại nguyện" with 5 vows and "Xem đủ 48 đại nguyện", "Hình tượng" and "Thần chú & danh hiệu" appear in that order

  Scenario: Long biography
    Given the biography is longer than the collapsed height
    When the page is shown
    Then the biography is collapsed with "Đọc thêm"

  Scenario: Missing values are hidden
    Given a deity has no iconography and no dhāraṇī
    When its page is shown
    Then no "Hình tượng" section and no "Đà-la-ni" card appear
  ```

**FR-DTY-006 — Next vía on the deity page**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a deity page is shown and the deity has vía days THE SYSTEM SHALL show a card
  with the next vía (BR-DTY-06): its name, lunar day and month ("THÁNG GIÊNG" … "THÁNG CHẠP" for months
  1 and 12), its solar date with weekday and its countdown, the list of all the deity's vía days, and a
  "Lịch lễ" action that opens the festival calendar.
- **Business rules:** BR-DTY-05, BR-DTY-06
- **Verification method:** shared vectors `deity-next-via.json`; automated test on each client
- **Source:** prototype `buddha` (vía card); iOS and Android (countdown); website (lunar date only —
  drift, replaced); anh Đức, 2026-10-15
- **Acceptance criteria:**
  ```gherkin
  Scenario: Next vía with countdown
    Given Quán Thế Âm's next vía occurs 14 days from today
    When the page of Quán Thế Âm is shown
    Then the vía card shows that vía's name, its lunar date, its solar date with weekday and "Còn 14 ngày"
    And all three vía days of Quán Thế Âm are listed

  Scenario: Vía today
    Given a deity's vía occurs today
    When its page is shown
    Then the vía card shows "Hôm nay"

  Scenario: Open the festival calendar
    When the visitor chooses "Lịch lễ" on a deity page
    Then the festival calendar opens
  ```

**FR-DTY-007 — Related sutras**
- **Priority:** Should
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a deity page is shown and the sutra library links sutras to that deity THE SYSTEM
  SHALL list them under "Kinh liên quan" in the sutra library's order, and choosing one SHALL open that
  sutra's page; WHEN no sutra is linked the section is not shown.
- **Business rules:** BR-DTY-04
- **Verification method:** automated test on each platform
- **Source:** backend `sutra` (deity ↔ sutra links, unused by clients); iOS (matched by title against a
  hard-coded table — drift, replaced); Android sample; anh Đức, 2026-10-15
- **Acceptance criteria:**
  ```gherkin
  Scenario: Linked sutra opens
    Given the sutra library links "Kinh A Di Đà" to Phật A Di Đà
    When the visitor chooses "Kinh A Di Đà" on the page of Phật A Di Đà
    Then the page of "Kinh A Di Đà" opens

  Scenario: No linked sutra
    Given no sutra is linked to a deity
    When its page is shown
    Then no "Kinh liên quan" section appears
  ```

#### Practice actions

**FR-DTY-008 — Recite with the mala**
- **Priority:** Should
- **Platforms:** Website, iOS, Android
- **Description:** WHEN the visitor chooses "Trì niệm với tràng hạt" (recite with the mala) on a deity
  page THE SYSTEM SHALL open the mala practice with the deity's short mantra in Sino-Vietnamese as the
  recitation text and the count at 0; the action is shown only when the deity has a short mantra.
- **Business rules:** BR-DTY-04
- **Verification method:** automated test on each client
- **Source:** prototype `buddha` (mala, text set, beads reset); iOS behaviour; Android (text not passed
  — drift); anh Đức, 2026-10-15
- **Acceptance criteria:**
  ```gherkin
  Scenario: Recite the mantra
    Given Phật A Di Đà's short mantra is "Nam mô A Di Đà Phật"
    When the visitor chooses "Trì niệm với tràng hạt" on his page
    Then the mala opens with "Nam mô A Di Đà Phật" and a count of 0

  Scenario: No mantra
    Given a deity has no short mantra
    When its page is shown
    Then "Trì niệm với tràng hạt" is not offered
  ```

**FR-DTY-009 — Share**
- **Priority:** Could
- **Platforms:** Website, iOS, Android
- **Description:** WHEN the visitor chooses "Chia sẻ" (share) on a deity page THE SYSTEM SHALL share the
  deity's name and short mantra (Sino-Vietnamese, then Sanskrit when present) through the device's
  share function (apps) or the browser's share or copy function (website).
- **Business rules:** BR-DTY-04
- **Verification method:** automated test on each client; manual QA
- **Source:** iOS behaviour (system share); Android (no-op — drift); website (none); anh Đức,
  2026-10-15
- > ⚠️ Not confirmed: whether the shared text also carries a link to the deity's web page — blocks the shared content on iOS and Android, and what the website copies.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Share a deity
    When the visitor chooses "Chia sẻ" on the page of Phật A Di Đà
    Then the share function opens with "Phật A Di Đà" and "Nam mô A Di Đà Phật"
  ```

#### States

**FR-DTY-010 — Empty and error states**
- **Priority:** Must
- **Platforms:** Website, iOS, Android
- **Description:** WHEN the catalogue cannot be loaded or holds no deity THE SYSTEM SHALL show, in the
  catalogue, the empty state "Chưa có thông tin chư Phật" (no information on the Buddhas yet) with the
  hint to check the connection, a "Thử lại" (retry) action and a "Xem lịch lễ vía" action that opens
  the festival calendar; the home and library blocks show their own empty state. No platform shows
  placeholder or hard-coded deities instead of real content.
- **Business rules:** BR-DTY-02
- **Verification method:** automated test on each client
- **Source:** prototype `buddhasEmpty`; anh Đức, 2026-10-15; replaces the website's static fallback
  list and the iOS silent empty block — drift
- **Acceptance criteria:**
  ```gherkin
  Scenario: Backend unreachable
    Given the backend cannot be reached and nothing was loaded before
    When a visitor opens the catalogue
    Then "Chưa có thông tin chư Phật" appears with "Thử lại" and "Xem lịch lễ vía"

  Scenario: Retry succeeds
    Given the catalogue shows its error state
    When the backend is reachable again and the visitor chooses "Thử lại"
    Then the first page of the catalogue appears

  Scenario: No fake content on the home view
    Given the backend cannot be reached
    When a visitor opens the home view
    Then the "Thư viện Phật & Bồ Tát" block shows its empty state and no deity card
  ```

#### Operator

**FR-DTY-011 — Load catalogue content**
- **Priority:** Must
- **Platforms:** Backend
- **Description:** WHEN the operator loads a catalogue file THE SYSTEM SHALL add new deities and update
  existing ones by their stable identifier, all or nothing: an invalid file (no deity, missing or
  duplicate identifier, unknown type, missing Vietnamese name) changes nothing. The editorial position
  in the file becomes the catalogue order (BR-DTY-02). Deities keep their identifier across loads, so
  links (vía events, sutras, shared pages) stay valid.
- **Business rules:** BR-DTY-01, BR-DTY-02
- **Verification method:** automated test on the backend
- **Source:** backend v1.0 (seed)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Reload after reordering
    Given "Quán Thế Âm" is 8th in the catalogue
    When the operator loads a file where "Quán Thế Âm" is 2nd
    Then "Quán Thế Âm" is 2nd in the catalogue and no second "Quán Thế Âm" appears

  Scenario: Invalid file
    When the operator loads a file with a deity of an unknown type
    Then the load is refused and the catalogue is unchanged
  ```
- > ⚠️ Not confirmed: who runs the load in staging and production and from which copy of the file (today a dev-only command and a file kept outside version control) — blocks production content, not the platform specs.

### 3.4 Non-functional requirements

- **NFR-DTY-01 — Page by page:** no platform loads the whole catalogue at once; browsing and the
  filter work page by page whatever the number of deities. Platforms: all.
- **NFR-DTY-02 — Responsive filter:** a filter change shows its first page within 2 s on a 4G
  connection. Platforms: all.

### 3.5 Data requirements

| Data item | Meaning | Owned by | Retention / deletion | Private or public |
|---|---|---|---|---|
| Deity profile | Type, names, honorific title, core virtue, pure land, canonical source, biography, vows, iconography, mantras, images | InnerLamp catalogue on the server | Kept; changed by the operator | Public |
| Stable deity identifier | Never changes for a deity; vía events, sutra links and shared pages refer to it | Server | As long as the deity exists | Public |
| Deity ↔ vía event link | Which festival-calendar events are the deity's vía days | Server (catalogue content) | Kept with the deity | Public |
| Deity ↔ sutra link | Which sutras relate to the deity | Server (sutra library) | Kept with the sutra | Public |

No personal data is involved.

### 3.6 Platform variance

| ID | Platforms | Difference | Reason |
|---|---|---|---|
| FR-DTY-001 / BR-DTY-02 | all clients | Each block shows as many of the 10 featured deities as its layout fits (today: website 6) | anh Đức, 2026-10-15: "tuỳ nền tảng và UI hiển thị" |
| FR-DTY-002 | Website vs apps | Website: page numbers (shareable page links); a page number past the end leads to the last page. Apps: next page loads at the end of the list | Web navigation and SEO vs mobile scrolling |
| FR-DTY-009 | Website | Uses the browser's share function, or copies the text when there is none | No OS share sheet on desktop browsers |
| §2.4 | iOS, Android | May keep the last loaded featured list, pages and deity pages readable offline; the website is online only | App caching |

### 3.7 Software system attributes

- **Reliability:** identical inputs give identical pages on every load (BR-DTY-02 stable order).
- **Availability:** backend unreachable → error state with retry (FR-DTY-004, FR-DTY-010); apps may
  show cached content.
- **Security & privacy:** all data is public; no personal data.
- **Maintainability / portability:** N/A.

---

## 4. Verification & traceability

### 4.1 Verification approach

Computational rules are verified by running `rules/*.json`: the order (BR-DTY-02) on the backend; the
next vía (BR-DTY-06) on every platform that picks it. Vía placement and countdown are verified by the
`calendar` vectors. Functional requirements are verified by each platform's automated tests and manual
QA against the prototype.

### 4.2 Traceability matrix

| ID | Backend | Website | iOS | Android | Note |
|---|---|---|---|---|---|
| BR-DTY-01 | done | drift | drift | planned | Web label "Thiên hộ pháp", iOS "Thiên Nữ"; Android sample has no deva |
| BR-DTY-02 | done | done | done | planned | Android sample data |
| BR-DTY-03 | drift | drift | drift | planned | Backend pages of 20; website shows backend pages; iOS loads every page |
| BR-DTY-04 | n/a | done | done | planned | |
| BR-DTY-05 | planned | drift | drift | drift | Backend stores its own vía list; web and Android hard-code dates; iOS computes on the device |
| BR-DTY-06 | planned | drift | drift | drift | Web: no next vía; iOS: "Đản/Vía" name match; Android: sample |
| BR-DTY-07 | n/a | drift | done | done | Website shows every vow |
| BR-DTY-08 | n/a | done | drift | drift | iOS and Android footer show "Báo thông tin sai" |
| FR-DTY-001 | done | done | done | planned | 2 scenarios |
| FR-DTY-002 | drift | done | drift | planned | 3 scenarios; page size 12; vía on cards |
| FR-DTY-003 | done | done | drift | planned | 2 scenarios; iOS filters on the device |
| FR-DTY-004 | done | drift | done | planned | 3 scenarios; website shows an error page instead of retry |
| FR-DTY-005 | done | drift | drift | drift | 3 scenarios; web lacks the vía card order; iOS hides iconography; Android has no iconography |
| FR-DTY-006 | planned | drift | drift | drift | 3 scenarios; vía from `calendar` |
| FR-DTY-007 | done | planned | drift | drift | 2 scenarios; backend links exist |
| FR-DTY-008 | n/a | planned | done | drift | 2 scenarios; Android does not pass the mantra |
| FR-DTY-009 | n/a | planned | done | drift | 1 scenario; Android no-op |
| FR-DTY-010 | n/a | drift | drift | planned | 3 scenarios; web static fallback; iOS silent empty block |
| FR-DTY-011 | done | n/a | n/a | n/a | 2 scenarios |
| NFR-DTY-01 | done | done | drift | planned | iOS loads every page at once |
| NFR-DTY-02 | planned | planned | planned | planned | |

### 4.3 Test vectors

| File | Rule | Cases |
|---|---|---|
| `rules/deity-order.json` | BR-DTY-02 | 5 |
| `rules/deity-next-via.json` | BR-DTY-06 | 7 |

---

## 5. Appendices

### Appendix A — Open questions

#### Open (unconfirmed)

1. > ⚠️ Not confirmed: who reviews the deity content (biography, vows, mantras, iconography) and the image rights before public release, and how — blocks public release of the content, not the platform specs.
2. > ⚠️ Not confirmed: whether the editorial file order is the intended "featured" criterion (the source has no popularity measure) — the order rule stands; only the content of the featured list may change.
3. > ⚠️ Not confirmed: whether the shared text also carries a link to the deity's web page — blocks the shared content on iOS and Android, and what the website copies.
4. > ⚠️ Not confirmed: who runs the load in staging and production and from which copy of the file (today a dev-only command and a file kept outside version control) — blocks production content, not the platform specs.
5. > ⚠️ Not confirmed: whether a "Chư Thiên" chip is added once the catalogue holds more than one deva — no platform is blocked now.

#### Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| Is an ADR needed? | No — read-only public catalogue | 2026-10-15 |
| Deva type label and filter | Label "Chư Thiên"; no chip, listed under "Tất cả" | 2026-10-15 |
| Featured count | Backend gives 10; each client shows as many as its layout fits | 2026-10-15 |
| Page size and search | 12 per page; no search | 2026-10-15 |
| Source of vía days | The festival calendar's events; next vía counted down on every platform | 2026-10-15 |
| Several vía days | The nearest upcoming one; the page lists all | 2026-10-15 |
| Related sutras | From the sutra library's deity links; no title matching | 2026-10-15 |
| Page actions and sections | Keep mala, share and iconography; drop "Báo thông tin sai" | 2026-10-15 |
| Error and empty state | Prototype empty state with retry; no static fallback content | 2026-10-15 |
| Great vows collapsed, offline | 5 shown + "Xem đủ"; apps may cache, website online only | 2026-10-15 |
| Content review and image rights | Left open (item 1) | 2026-10-15 |

### Appendix B — Contract handoff (backend)

1. Catalogue page size becomes 12 (BR-DTY-03); the featured count stays 10.
2. Each deity is linked to its vía events of the festival calendar (BR-DTY-05). The deity's own list of
   vía days is retired or kept strictly identical to the calendar; the calendar is the only source.
3. Featured cards and catalogue cards carry the next vía (lunar day and month at least, BR-DTY-06);
   the deity profile carries every linked vía event with its next occurrence (name, lunar date, solar
   date), so no client converts lunar dates or fetches each profile for a card.
4. Related sutras: the existing deity → sutras listing is the source of FR-DTY-007; clients start using
   it.
5. The catalogue file (today kept outside version control by the ignore rule for JSON files) is
   committed with the backend and is the input of the content load in every environment (Appendix A
   item 4).
6. The backend platform spec `docs/specs/features/deity/` v2.3 is behind the code (20 deities including
   Thất Phật Quá Khứ and Đại Cát Tường Thiên Nữ, images now served under the backend's own static path);
   `spec-writer` brings it in line when it cites this spec.
