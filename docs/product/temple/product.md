# Product Spec — Temple directory (Chùa Việt Nam)

| Field | Value |
|---|---|
| Document type | Product spec — Software Requirements Specification, IEEE 29148-2018 structure |
| Feature | Temple directory: featured temples, all temples with search and region filter, temple page (`temple`) |
| Feature code | `TPL` — IDs are `BR-TPL-NN`, `FR-TPL-NNN`, `NFR-TPL-NN` |
| Product | InnerLamp |
| Platforms in scope | Backend, Website, iOS, Android |
| Version | 1.2 |
| Last updated | 2026-10-15 |
| Status | confirmed — 1 open item (Appendix A) |
| Owner / author | Product owner: anh Đức · written by Claude (`ba-writer`, retroactive mode) |
| API contract | `docs/specs/api/openapi.yaml` (owned by the backend; this spec never restates it) |
| Architecture decisions | none needed (directory is read-only and public; see §2.7) |
| Related product specs | `onboarding` (`docs/product/onboarding/product.md`, code `ONB`) — picks the ordination temple from the short temple list of FR-TPL-005 (BR-ONB-10) |
| Downstream | Backend: `docs/specs/features/temple/` · Website: `Docs/specs/features/temple/` · iOS: `Docs/Specs/Temple/` · Android: `Docs/Specs/Temple/` |

> See `changelog.md` in this folder for the version history. Platform specs cite this file as
> `docs/product/temple/product.md@<version>` plus the IDs they implement.

---

## 1. Introduction

### 1.1 Purpose

This document is the single source of product truth for InnerLamp's temple directory on every
platform: what a visitor can browse, search, filter and read about Buddhist temples in Vietnam, and the
rules that make every platform show the same temples in the same order.

### 1.2 Scope

**In scope:**

- A short list of **featured temples** for the home view.
- The **whole directory**, page by page (12 temples per page), with **search by name** and a
  **region filter** (North / Central / South), which combine.
- A **temple page** with the temple's profile and a way to get directions.
- A **short temple list** (name and province only) for choosing a temple in another feature, such as
  the ordination temple at registration.
- Loading the directory content (operator task; no user-facing editing).

**Out of scope:**

- **Following a temple** ("Theo dõi") — a separate feature, specified later (anh Đức, 2026-10-15).
- **Reporting wrong information** ("Báo thông tin sai") — dropped (anh Đức, 2026-10-15).
- **Offering** ("Cúng dường") — the `offering` feature; this feature shows no offering action (BR-TPL-08).
- **Proposing a new temple** — `temple-suggestion`.
- **The user's own temple** (đạo tràng): schedule, temple events, temple announcements — `sangha`.
- Adding, editing or removing temples from the back office (anh Đức, 2026-10-15); temples are loaded by
  the operator or created from an approved temple proposal.
- A map of temples, "near me" search, sorting by distance, ratings and reviews.

### 1.3 Product overview

InnerLamp is a Buddhist companion app (website, iOS, Android) with a practice library, a lunar
calendar, family features and a community around temples. The temple directory is the public,
read-only catalogue of Buddhist places in Vietnam — chùa, thiền viện, thiền xá, tịnh xá, đạo tràng —
reachable from the home view without an account.

### 1.4 Glossary

| Term | Definition | Not to be confused with |
|---|---|---|
| Chùa / temple | Any Buddhist place in the directory; its **type** tells which kind (chùa, thiền viện, đạo tràng, tịnh xá, thiền xá) | — |
| Đạo tràng (practice-centre type) | One of the five temple types | "Đạo tràng" as the user's own temple (= ordination temple, `sangha`) |
| Chùa nổi bật (featured temples) | The first temples of the directory order (BR-TPL-01), shown on the home view | A curated or paid placement |
| Miền (region) | North (Bắc), Central (Trung) or South (Nam) — the only three regions (BR-TPL-04) | Province (tỉnh/thành) |
| Cấp di tích (heritage rank) | The temple's official recognition level, e.g. world heritage, special national relic | Temple type |
| Hệ phái (sect) | The Buddhist tradition, e.g. Bắc tông, Nam tông, Khất sĩ; an open list | — |
| Tên khác (alias) | A popular or historical name of the temple, shown as "Còn gọi: …" and matched by search | The official name |
| Nơi xuất gia (ordination temple) | The temple a user chose for their account (`onboarding`) | A temple the user only visits |

### 1.5 References

- Design prototype: screens `temple` ("Trang chùa") and `temples` ("Tất cả chùa"), and the home block
  "Chùa Việt Nam" — `Mo-ta-tinh-nang.md` §6 "Trang chùa" and `Trang-chi-tiet-chua.md` (web layout), copy
  used: `innerlamp-Android/Docs/innerlamp-project/project/docs/`.
- Prior documents (rewritten into this spec; deleted after the platform specs are approved):
  backend `docs/specs/features/temple/` v1.14 (`requirements.md`, `api-contract.md`),
  `docs/assumptions/temple/A-001…A-005`; iOS `Docs/Specs/Temple/` (`SRS.md` v0.1, `requirements.md`
  v2.0), `Docs/Specs/Temples/` v1.0, `Docs/Assumptions/Temple/`, `Docs/Assumptions/Temples/`;
  Android `Docs/home_assumptions.md` rows 1, 3, 13; website: no spec.
- Question round: `reports/product/temple/2026-10-15-cau-hoi-truoc-khi-viet.md` (project files).
- Platform folder names: Backend `temple` · Website `temple` · iOS `Temple` (merges the former
  `Temple` and `Temples` folders) · Android `Temple`. Code folders keep their own names (iOS `Temple`,
  `Temples`; Android `temple`, `templelist`; website `temples`).

---

## 2. Overall description

### 2.1 Product perspective

Retroactive spec of a feature that already runs: the backend serves the directory; the website and iOS
show it from the backend; Android shows sample data. This version changes three things for every
platform: the directory gains search and a region filter on the server, a page holds 12 temples, and the
temple page shows no follow, offering or report-wrong-information action.

Touches: `onboarding` (short temple list for the ordination temple), `temple-suggestion` (an approved
proposal becomes a directory temple), `sangha` (the user's own temple opens this temple page), `calendar`
and `home` (open the temple page). Relies on the device's maps app for directions and, on the apps, the
device location for distance (§3.1.2).

### 2.2 Product functions

- Featured temples on the home view (`FR-TPL-001`)
- All temples, page by page, with search and region filter (`FR-TPL-002`, `FR-TPL-003`, `FR-TPL-004`)
- Short temple list for picking a temple (`FR-TPL-005`)
- Temple page and its sections (`FR-TPL-006` … `FR-TPL-009`)
- Directory content loading by the operator (`FR-TPL-010`)

### 2.3 User classes and characteristics

| Actor | Description | Permissions |
|---|---|---|
| Visitor | Anyone, signed in or not | Reads everything in this feature |
| Operator | Runs the content load | Loads and updates the directory; not a user-facing role |

### 2.4 Operating environment

| Platform | In scope | Notes |
|---|---|---|
| Backend | yes | Stores the directory, orders it, searches and filters it, serves pages |
| Website | yes | Directory pages with page numbers; temple page |
| iOS | yes | Home block, list with continuous loading, temple page |
| Android | yes | Same as iOS; today sample data only |

Network expectation: online. A platform may keep the last loaded content readable offline (§3.6).

### 2.5 Constraints

- Temple content (history, architecture, visiting notes, image links) comes from the JSON directory file
  loaded by the operator (FR-TPL-010). Crediting sources and image rights are not settled:
  > ⚠️ Not confirmed: whether edited texts must credit their sources and whether InnerLamp holds the rights to the temple images in the directory file — blocks public release of the content, not the platform specs.
- Province and ward follow the administrative units in force since 1 July 2025 (34 province-level units,
  two levels: province and ward). There is no district level.

### 2.6 User documentation

N/A — a read-only catalogue needs no help content.

### 2.7 Assumptions and dependencies

- **Assumptions:** the directory is public and read-only for users; it changes only when the operator
  reloads it or a temple proposal is approved. No architecture decision is needed (anh Đức accepted the
  "no ADR" proposal, 2026-10-15). A later follow feature or temple image storage may need one.
- **Dependencies:** administrative units (province and ward names); `temple-suggestion` (creates
  temples); the device maps app; the device location service (apps only).

---

## 3. Specific requirements

### 3.1 External interface requirements

#### 3.1.1 User interfaces

| View | Platforms | Shows |
|---|---|---|
| Home block "Chùa Việt Nam" | all clients | The featured temples (FR-TPL-001) and "Xem tất cả" (see all) to the directory |
| Directory "Chùa Việt Nam" / "Tất cả chùa" | all clients | Search field, region choices "Tất cả · Miền Bắc · Miền Trung · Miền Nam", the number of temples found, temple cards, more pages (FR-TPL-002…004) |
| Temple card | all clients | Image, name, type, province, heritage rank, sect |
| Temple page "Trang chùa" | all clients | Sections of FR-TPL-006…009, in the prototype's order |

#### 3.1.2 Hardware interfaces

Device location (iOS, Android), only to show the distance to a temple and only after the user allows it
(BR-TPL-07).

#### 3.1.3 Software interfaces

The backend must: give the featured temples; give one page of the directory, optionally narrowed by a
search term and a region, with the number of matches; give one temple's profile; give a short list of
temples (name and province) with search, for pickers. Contract: `docs/specs/api/openapi.yaml`, tag
Temples — search and region on the directory are **not yet** in it (Appendix B).

#### 3.1.4 Communication interfaces

N/A.

### 3.2 Business rules

**BR-TPL-01 — Directory order**
- **Rule:** Temples are always ordered by heritage rank (most important first: world heritage → special
  national relic → national tourist area → national relic → scenic landscape → provincial relic → city
  relic → ranked temple), then by founding year (older first), then by a stable identifier so the order
  never changes between two loads. The same order is used for the featured list, every directory page,
  every search or filter result, and the short list. Not personalised (no location, no history).
- **Examples:** a world-heritage temple founded in 1300 comes before a special national relic founded in
  1049; two national relics founded in 1600 and 1700 → 1600 first; a search for "trúc lâm" keeps this
  order among its matches.
- **Computational:** yes → `rules/temple-order.json`
- **Applies to:** Backend (clients show the order they receive)
- **Source:** backend v1.0 behaviour; anh Đức "Thứ tự chùa nổi bật giữ nguyên", 2026-10-15

**BR-TPL-02 — Featured temples**
- **Rule:** The featured list is the first N temples of the directory order (BR-TPL-01). N is set by the
  backend (today 10). Fewer temples than N → all of them; none → the block shows its empty state.
- **Examples:** 93 temples, N = 10 → the 10 first; 4 temples → 4.
- **Computational:** no (BR-TPL-01 vectors cover the order)
- **Applies to:** Backend, all clients
- **Source:** backend v1.0; anh Đức 2026-10-15 ("lấy top n chùa")

**BR-TPL-03 — Page size**
- **Rule:** One directory page holds **12** temples. The last page may hold fewer. Asking for a page past
  the last one gives an empty page, not an error.
- **Examples:** 93 temples → 8 pages, the 8th has 9; 0 temples → 0 pages; 12 matches → 1 page.
- **Computational:** no
- **Applies to:** Backend, all clients
- **Source:** anh Đức, 2026-10-15 ("1 page 12 items"; was 20 on the backend, 10 in the iOS SRS)

**BR-TPL-04 — Regions**
- **Rule:** Every temple belongs to exactly one of three regions: North, Central, South. There is no
  Highlands (Tây Nguyên) region; highland temples are Central (e.g. Thiền viện Trúc Lâm Đà Lạt).
  The filter offers "Tất cả" (all, no filter) plus the three regions; one choice at a time.
- **Examples:** filter South → only South temples; "Tất cả" → every temple.
- **Computational:** no
- **Applies to:** Backend, all clients
- **Source:** anh Đức, 2026-10-15 (Q3); iOS AS-Temples-001 (Android sample data had "Tây Nguyên" — drift)

**BR-TPL-05 — Search matching**
- **Rule:** A search term matches a temple when, after trimming, the term appears anywhere inside the
  temple's **name or one of its aliases**, ignoring letter case and Vietnamese diacritics (đ and d are
  the same letter). Address and province are not searched. Search and region filter combine (both must
  hold). The term is 1–100 characters after trimming; an empty term means "no search"; a longer term is
  cut by the client to 100 characters before it is sent. Matches keep the directory order (BR-TPL-01),
  not a relevance order.
- **Examples:** "bich dong" and "BÍCH ĐỘNG" both match "Chùa Bích Động"; "dien huu" matches "Chùa Một Cột" through its
  alias "Diên Hựu Tự"; "Hà Nội" matches no temple by province; "50%" matches only names
  containing "50%" literally.
- **Computational:** yes → `rules/temple-search-match.json`
- **Applies to:** Backend (search runs on the server); the short list (FR-TPL-005) matches **name only**
  (§3.6)
- **Source:** anh Đức, 2026-10-15 (Q2: server search on name + aliases, no address); backend v1.4 (`options`
  search); iOS `Temples` v1.0 searched name or address on the device — drift, replaced

**BR-TPL-06 — Missing values on the temple page**
- **Rule:** A section of the temple page whose data is missing still shows: text values show
  "Đang cập nhật" (being updated), amenities show "Không có" (not available), the events section shows
  its empty state. A temple without an image shows a placeholder. Without aliases, the "Còn gọi: …" line
  is hidden.
- **Examples:** no abbot → "Trụ trì: Đang cập nhật"; no aliases → no "Còn gọi" line.
- **Computational:** no
- **Applies to:** all clients
- **Source:** prototype ("Trường trống … hiện 'Đang cập nhật', không ẩn dòng"); iOS v2.4 behaviour
- **Fields added now / later:** the directory now also carries the Rằm & Mùng Một hours, the contacts
  (hotline, fanpage, website, email), the Buddhist sangha body (giáo hội) and the month the temple's data
  was last updated. Architecture style, materials and highlights, the "Kinh sách" amenity and the photo
  set come later; until then their rows and sections stay visible and show "Đang cập nhật" — including
  the "Kinh sách" amenity and the photo set, which are never hidden or shown as "Không có" while the data
  does not exist (anh Đức, 2026-10-15).

**BR-TPL-07 — Distance**
- **Rule:** The temple page shows "Cách bạn X km" only when the device knows the user's location because
  the user allowed it; otherwise the line is hidden. Distance is the great-circle (haversine) distance
  from the user to the temple, shown with one decimal under 10 km and rounded to a whole number from
  10 km. The website never asks for the location and never shows distance. No fixed reference point.
- **Examples:** 3.46 km → "Cách bạn 3,5 km"; 12.6 km → "Cách bạn 13 km"; location refused → no line.
- **Computational:** yes → `rules/temple-distance.json`
- **Applies to:** iOS, Android (Website: n/a)
- **Source:** anh Đức, 2026-10-15 (Q7); prototype distance rule; replaces the prototype's fixed point
  (Quận 1, TP.HCM)

**BR-TPL-08 — No offering, follow or report action**
- **Rule:** The temple page shows no "Cúng dường" (offering) action until the `offering` feature exists,
  no "Báo thông tin sai" (report wrong information) link, and no follow action while following is not
  built.
- **Examples:** the action row of the temple page holds only "Chỉ đường" (directions).
- **Computational:** no
- **Applies to:** all clients
- **Source:** anh Đức, 2026-10-15 (Q8 hide offering; Q5 report not built; follow is a later feature)
- The follow action is hidden, not shown disabled (anh Đức, 2026-10-15).

**BR-TPL-09 — Open or closed now**
- **Rule:** When the temple has opening hours, the temple page shows whether it is open now ("Đang mở
  cửa · đóng lúc HH:MM" or "Đã đóng cửa · mở lại HH:MM"). "Now" is always **Vietnam time (UTC+7)**,
  whatever the device's time zone. On the 1st (Mùng Một) and 15th (Rằm) day of the lunar month, Vietnam
  calendar, the Rằm & Mùng Một hours apply when the temple has them, otherwise the normal hours. Open
  means opening time ≤ now < closing time. Before opening → "mở lại" today's opening time; after
  closing → "mở lại" the next day's opening time (lunar rule applied to that day). Both hour rows are
  shown, the one in force today marked "Hôm nay". When the temple has no Rằm & Mùng Một hours of its own,
  that row shows the normal hours (they are the hours that apply on those days), not "Đang cập nhật".
  Without opening hours, the status is hidden and the hours show "Đang cập nhật".
- **Examples:** hours 05:00–20:00, now 19:10 → "Đang mở cửa · đóng lúc 20:00"; now 21:00 →
  "Đã đóng cửa · mở lại 05:00"; Rằm with holiday hours 04:00–22:00, now 21:00 → "Đang mở cửa · đóng lúc
  22:00"; device set to Paris at 14:00 (= 19:00 Vietnam) → judged at 19:00.
- **Computational:** yes → `rules/temple-open-status.json`
- **Applies to:** all clients
- **Source:** prototype; iOS behaviour; anh Đức, 2026-10-15 (Vietnam time, Rằm / Mùng Một hours)

### 3.3 Functional requirements

#### Browsing

**FR-TPL-001 — Featured temples**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a visitor opens the home view THE SYSTEM SHALL show the featured temples
  (BR-TPL-02) as cards and a "Xem tất cả" action that opens the directory.
- **Business rules:** BR-TPL-01, BR-TPL-02
- **Verification method:** automated test on each platform; vectors `temple-order.json` on the backend
- **Source:** backend v1.0; prototype home block
- **Acceptance criteria:**
  ```gherkin
  Scenario: Featured temples on the home view
    Given the directory holds 93 temples and the featured count is 10
    When a visitor who is not signed in opens the home view
    Then 10 temple cards appear in the directory order
    And the block offers "Xem tất cả"

  Scenario: Empty directory
    Given the directory holds no temple
    When a visitor opens the home view
    Then the temple block shows its empty state and no error

  Scenario: Featured temples cannot be loaded
    Given the featured temples cannot be loaded (server error or no connection)
    When a visitor opens the home view
    Then the temple block shows the same empty state as an empty directory
    And no error message, retry action or sample temple is shown
    And the rest of the home view is unaffected
  ```

**FR-TPL-002 — All temples, page by page**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a visitor opens the directory THE SYSTEM SHALL show the first 12 temples in the
  directory order with the number of temples found, and SHALL show the next pages on request: the website
  by page numbers, the apps by loading the next page as the visitor reaches the end of the list (§3.6).
- **Business rules:** BR-TPL-01, BR-TPL-03
- **Verification method:** automated test on each platform
- **Source:** backend v1.0; website; iOS `Temples` v1.0 (loaded every page at once — drift, replaced)
- **Acceptance criteria:**
  ```gherkin
  Scenario: First page
    Given the directory holds 93 temples
    When a visitor opens the directory
    Then 12 temples appear, the first ones of the directory order
    And the screen says 93 temples

  Scenario: Last page
    Given the directory holds 93 temples
    When the visitor moves to page 8
    Then 9 temples appear and no further page is offered

  Scenario: Loading fails
    Given the backend cannot be reached
    When a visitor opens the directory
    Then an error message with "Thử lại" (retry) appears, worded differently when the device is offline
  ```

**FR-TPL-003 — Search by name**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a visitor types a term in the directory search THE SYSTEM SHALL show, from page 1,
  only the temples matching it (BR-TPL-05), with the number of matches, and SHALL show the whole directory
  again when the term is cleared.
- **Business rules:** BR-TPL-05, BR-TPL-01, BR-TPL-03
- **Verification method:** shared vectors `temple-search-match.json` on the backend; automated test on
  each client
- **Source:** anh Đức, 2026-10-15 (Q2)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Search without diacritics
    Given the directory holds "Chùa Bích Động"
    When the visitor searches "bich dong"
    Then "Chùa Bích Động" is in the results

  Scenario: Search matches an alias
    Given "Chùa Một Cột" has the alias "Diên Hựu Tự"
    When the visitor searches "dien huu"
    Then "Chùa Một Cột" is in the results

  Scenario: No match
    When the visitor searches a term no temple name or alias contains
    Then the list says "Không tìm thấy chùa" with a hint to change the keyword or region

  Scenario: Clearing the search
    Given the visitor searched "bich dong"
    When the visitor clears the search field
    Then the whole directory appears again from page 1
  ```

**FR-TPL-004 — Region filter**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a visitor chooses a region THE SYSTEM SHALL show, from page 1, only the temples
  of that region, combined with the current search; "Tất cả" removes the filter.
- **Business rules:** BR-TPL-04, BR-TPL-05
- **Verification method:** automated test on each platform
- **Source:** anh Đức, 2026-10-15 (Q2, Q3); iOS `Temples` v1.0 (filtered on the device — drift, replaced)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Filter by region
    When the visitor chooses "Miền Nam"
    Then only South temples appear, in the directory order, with their count

  Scenario: Region and search combine
    Given the visitor chose "Miền Bắc"
    When the visitor searches "tran quoc"
    Then only North temples whose name or alias contains "tran quoc" appear

  Scenario: Back to all regions
    Given the visitor chose "Miền Trung"
    When the visitor chooses "Tất cả"
    Then temples of every region appear, the search still applied
  ```

**FR-TPL-005 — Short temple list for pickers**
- **Priority:** Must
- **Platforms:** Backend (clients use it through `onboarding`, BR-ONB-10)
- **Description:** WHEN another feature asks the visitor to choose a temple THE SYSTEM SHALL offer a list
  showing only each temple's name and province, in the directory order, searchable by name (BR-TPL-05,
  name only), loaded page by page.
- **Business rules:** BR-TPL-01, BR-TPL-05
- **Verification method:** automated test on the backend; picker behaviour is tested under `onboarding`
- **Source:** backend v1.4–v1.7 (`options`)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Pick-list entries
    When a visitor opens the temple picker at registration
    Then each entry shows the temple name and its province only

  Scenario: Pick-list search ignores diacritics
    When the visitor types "vinh nghiem" in the picker
    Then "Chùa Vĩnh Nghiêm" is offered
  ```

#### Temple page

**FR-TPL-006 — Open a temple page**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a visitor opens a temple from the home view, the directory, their own temple
  (`sangha`) or the calendar THE SYSTEM SHALL show that temple's page; WHEN the temple does not exist THE
  SYSTEM SHALL say the temple was not found; WHEN loading fails THE SYSTEM SHALL offer "Thử lại".
- **Business rules:** BR-TPL-06
- **Verification method:** automated test on each platform
- **Source:** backend v1.0; prototype "Mở từ"
- **Acceptance criteria:**
  ```gherkin
  Scenario: Open from the directory
    When the visitor opens "Chùa Trấn Quốc" from the directory
    Then the page of "Chùa Trấn Quốc" appears without signing in

  Scenario: Unknown temple
    When a visitor opens a temple that is not in the directory
    Then the page says the temple was not found and offers a way back
  ```

**FR-TPL-007 — Temple profile sections**
- **Priority:** Must
- **Platforms:** Website, iOS, Android (Backend provides the data)
- **Description:** WHEN a temple page is shown THE SYSTEM SHALL show, in the prototype's order: image
  header with heritage rank, sect, name, "Còn gọi: …" aliases and place (ward, province); opening status
  and hours (BR-TPL-09) and distance (BR-TPL-07); founding year, heritage rank, sect and region; the
  introduction, collapsed with "Đọc thêm / Thu gọn" when long; abbot, founder and Buddhist sangha body;
  architecture and heritage; amenities (car parking, motorbike parking, vegetarian meals, retreats, and
  "Kinh sách", which shows "Đang cập nhật" until the data exists); numbered visiting notes "01, 02, …"; location and contacts (full address,
  region, coordinates, hotline, fanpage, website, email); and a footer with the update month when
  known. Missing values follow BR-TPL-06.
- **Business rules:** BR-TPL-06, BR-TPL-07, BR-TPL-09
- **Verification method:** automated test on each client; manual QA against the prototype
- **Source:** prototype "Trang chùa"; iOS v2.5; website `TempleDetail`
- **Acceptance criteria:**
  ```gherkin
  Scenario: Full profile
    Given a temple has aliases, an abbot and visiting notes
    When its page is shown
    Then "Còn gọi" lists the aliases, the abbot is named and the notes are numbered from "01"

  Scenario: Missing values stay visible
    Given a temple has no abbot and no hotline
    When its page is shown
    Then "Trụ trì" and "Hotline" show "Đang cập nhật" and the hotline cannot be used

  Scenario: Long introduction
    Given the introduction is longer than the collapsed height
    When the page is shown
    Then the introduction is collapsed with "Đọc thêm"
  ```

**FR-TPL-008 — Directions and contacts**
- **Priority:** Must
- **Platforms:** Website, iOS, Android
- **Description:** WHEN the visitor chooses "Chỉ đường" (or a maps action) THE SYSTEM SHALL open
  directions to the temple's coordinates in a maps app or maps website; WHEN the visitor chooses a contact
  that has a value THE SYSTEM SHALL call the number, open the page, or start an email; a contact without a
  value does nothing.
- **Business rules:** BR-TPL-06
- **Verification method:** automated test on each client
- **Source:** prototype; iOS (Google Maps and Apple Maps); website directions link
- **Acceptance criteria:**
  ```gherkin
  Scenario: Directions
    When the visitor chooses "Chỉ đường" on a temple page
    Then a maps app or maps website opens with the temple as destination

  Scenario: Contact without value
    Given the temple has no email
    When the visitor chooses the email row
    Then nothing opens
  ```

**FR-TPL-009 — Events section**
- **Priority:** Could
- **Platforms:** Website, iOS, Android
- **Description:** WHEN a temple page is shown THE SYSTEM SHALL show the "Sự kiện & khóa học" section; its
  content comes from `sangha` (temple events). Until `sangha` provides it on a platform, the section shows
  its empty state "Chùa chưa đăng sự kiện nào." without the follow hint of the prototype (BR-TPL-08).
- **Business rules:** BR-TPL-06, BR-TPL-08
- **Verification method:** manual QA
- **Source:** prototype; iOS (empty state today)
- **Acceptance criteria:**
  ```gherkin
  Scenario: No events
    Given no events are known for the temple
    When its page is shown
    Then the events section shows its empty state
  ```

#### Operator

**FR-TPL-010 — Load directory content**
- **Priority:** Must
- **Platforms:** Backend
- **Description:** WHEN the operator loads a directory file THE SYSTEM SHALL add new temples and update
  existing ones by their stable identifier, all or nothing: an invalid file (missing name, unknown region,
  heritage rank, province or ward, duplicate identifier) changes nothing. Temples keep their identifier
  across loads, so links and saved choices (e.g. an ordination temple) stay valid.
- **Business rules:** BR-TPL-04
- **Verification method:** automated test on the backend
- **Source:** backend v1.0–v1.7 (seed)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Reload after an edit
    Given temple "Chùa Hương" exists
    When the operator loads a file where its introduction changed
    Then "Chùa Hương" shows the new introduction and no second "Chùa Hương" appears

  Scenario: Invalid file
    When the operator loads a file with an unknown ward
    Then the load is refused and the directory is unchanged
  ```
- **Content source:** in every environment the directory is loaded from the JSON directory file
  (anh Đức, 2026-10-15). The file is the source of truth and must be kept under version control with
  the backend (Appendix B).

### 3.4 Non-functional requirements

- **NFR-TPL-01 — Responsive search:** a search or region change shows its first page within 2 s on a 4G
  connection; clients wait for a short pause in typing before searching so a word does not send one
  request per letter. Platforms: all.
- **NFR-TPL-02 — Scales with the directory:** no platform loads the whole directory at once; browsing,
  search and filter work page by page whatever the number of temples. Platforms: all.

### 3.5 Data requirements

| Data item | Meaning | Owned by | Retention / deletion | Private or public |
|---|---|---|---|---|
| Temple profile | Name, aliases, type, region, sect, heritage rank, founding year, province, ward, street line, coordinates, hours, amenities, people, texts, contacts, images | InnerLamp directory on the server | Kept; changed by the operator or an approved proposal | Public |
| Stable temple identifier | Never changes for a temple; other features keep it (ordination temple) | Server | As long as the temple exists | Public |
| User location | Used on the device only to compute distance | The user's device | Not stored, not sent | Private |

### 3.6 Platform variance

| ID | Platforms | Difference | Reason |
|---|---|---|---|
| FR-TPL-002 | Website vs apps | Website: page numbers (shareable page links); apps: next page loads at the end of the list | Web navigation and SEO vs mobile scrolling |
| BR-TPL-07 | Website | No distance, never asks for location | Low value on desktop; avoids a browser permission prompt |
| FR-TPL-008 | iOS | Offers both Apple Maps and Google Maps; Android and website open the default maps app / Google Maps | OS maps apps |
| FR-TPL-005 | all | The short list searches name only (BR-TPL-05 adds aliases for the directory) | Picker keeps the existing behaviour of `onboarding` |
| §2.4 | iOS, Android | May keep the last loaded featured list, pages and temple pages readable offline | App caching; the website is online only |

### 3.7 Software system attributes

- **Reliability:** identical inputs give identical pages on every load (BR-TPL-01 stable order).
- **Availability:** backend unreachable → error with retry (FR-TPL-002, FR-TPL-006); apps may show cached
  content.
- **Security & privacy:** all data is public; the user's location never leaves the device.
- **Maintainability / portability:** N/A.

---

## 4. Verification & traceability

### 4.1 Verification approach

Computational rules (BR-TPL-01, BR-TPL-05, BR-TPL-07, BR-TPL-09) are verified by running `rules/*.json`:
order and search on the backend, distance on iOS and Android, open status on every client. Functional requirements are verified by each
platform's automated tests and manual QA against the prototype.

### 4.2 Traceability matrix

| ID | Backend | Website | iOS | Android | Note |
|---|---|---|---|---|---|
| BR-TPL-01 | done | done | done | planned | Android sample data |
| BR-TPL-02 | done | done | done | planned | |
| BR-TPL-03 | drift | drift | drift | planned | Backend pages of 20; website shows backend pages; iOS loads all pages |
| BR-TPL-04 | done | planned | done | drift | No region filter on web; Android sample has "Tây Nguyên" |
| BR-TPL-05 | planned | planned | drift | planned | Backend searches only the short list (name); iOS searches name or address on the device |
| BR-TPL-06 | done | drift | done | planned | Website hides empty sections |
| BR-TPL-07 | n/a | n/a | drift | planned | iOS leaves distance blank (no location request) |
| BR-TPL-08 | n/a | drift | drift | planned | Website and iOS show a follow action; iOS shows "Cúng dường" and the report link |
| BR-TPL-09 | n/a | drift | drift | planned | Website shows the hours text only; iOS uses device time and has no Rằm / Mùng Một hours yet |
| FR-TPL-001 | done | done | done | planned | 2 scenarios |
| FR-TPL-002 | drift | done | drift | planned | 3 scenarios; page size 12 |
| FR-TPL-003 | planned | planned | drift | planned | 4 scenarios |
| FR-TPL-004 | planned | planned | drift | planned | 3 scenarios |
| FR-TPL-005 | done | done | done | in-progress | 2 scenarios; via `onboarding` |
| FR-TPL-006 | done | done | done | planned | 2 scenarios |
| FR-TPL-007 | planned | drift | in-progress | planned | 3 scenarios; backend adds holiday hours, contacts, sangha body, update month |
| FR-TPL-008 | n/a | done | done | planned | 2 scenarios |
| FR-TPL-009 | n/a | planned | done | planned | 1 scenario |
| FR-TPL-010 | done | n/a | n/a | n/a | 2 scenarios |
| NFR-TPL-01 | planned | planned | planned | planned | |
| NFR-TPL-02 | done | done | drift | planned | iOS loads every page at once |

### 4.3 Test vectors

| File | Rule | Cases |
|---|---|---|
| `rules/temple-order.json` | BR-TPL-01 | 6 |
| `rules/temple-search-match.json` | BR-TPL-05 | 10 |
| `rules/temple-distance.json` | BR-TPL-07 | 6 |
| `rules/temple-open-status.json` | BR-TPL-09 | 10 |

---

## 5. Appendices

### Appendix A — Open questions

#### Open (unconfirmed)

1. > ⚠️ Not confirmed: whether edited texts must credit their sources and whether InnerLamp holds the rights to the temple images in the directory file — blocks public release of the content, not the platform specs.

#### Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| Is an ADR needed? | No — read-only public directory | 2026-10-15 |
| Scope | Featured top N, directory pages of 12, search, region filter; plus the existing temple page and picker list | 2026-10-15 |
| Search where and on what? | On the server, name + aliases, no address | 2026-10-15 |
| Regions | Three: North, Central, South; no Highlands | 2026-10-15 |
| Follow a temple | Separate feature, later | 2026-10-15 |
| Report wrong information | Not built | 2026-10-15 |
| Distance | Only with the user's location permission; website none | 2026-10-15 |
| Offering action | Hidden until the offering feature exists | 2026-10-15 |
| Back-office temple editing | Out of scope; content loaded by the operator | 2026-10-15 |
| Featured order | Unchanged (heritage rank, founding year) | 2026-10-15 |
| Spec folder names | `temple` / `Temple`; iOS merges `Temple` + `Temples` | 2026-10-15 |
| Temple-page fields | Add now: Rằm & Mùng Một hours, contacts, sangha body, update month; later: architecture details, "Kinh sách", photo set | 2026-10-15 |
| Open / closed time zone | Always Vietnam time; Rằm / Mùng Một hours on lunar 1st and 15th | 2026-10-15 |
| Follow action until the follow feature | Hidden | 2026-10-15 |
| Content loading | From the JSON directory file in every environment | 2026-10-15 |
| Rằm & Mùng Một row without holiday hours | Shows the normal hours | 2026-10-15 |
| Fields that come later (architecture, "Kinh sách", photo set) | Shown with "Đang cập nhật", not hidden | 2026-10-15 |
| Home featured block when loading fails | Same empty state as an empty directory; no error text, no sample temples | 2026-10-15 |

### Appendix B — Contract handoff (backend)

1. The directory page takes an optional search term (name + aliases, diacritic-insensitive, 1–100
   characters) and an optional region (north / central / south); both combine; the page count reflects
   the matches. Invalid region or term → a validation error naming the parameter.
2. Directory page size becomes 12 (BR-TPL-03); the featured count stays a backend setting.
3. The short list keeps its current shape and name-only search.
4. Temple profile gains: Rằm & Mùng Một opening and closing times, hotline, fanpage, website, email,
   Buddhist sangha body, and the month the data was last updated — each optional.
5. The JSON directory file is committed with the backend (today the ignore rule for JSON files keeps it
   out of version control) and is the input of the content load in every environment.
