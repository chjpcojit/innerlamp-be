# Product Spec — Buddhist photo gallery (Ảnh đẹp Phật giáo)

| Field | Value |
|---|---|
| Document type | Product spec — Software Requirements Specification, IEEE 29148-2018 structure |
| Feature | Buddhist photo gallery: featured photos on the home view, the whole gallery with a topic filter, the photo viewer with credits and save (`gallery`) |
| Feature code | `GAL` — IDs are `BR-GAL-NN`, `FR-GAL-NNN`, `NFR-GAL-NN` |
| Product | InnerLamp |
| Platforms in scope | Backend, Website, iOS, Android |
| Version | 1.1 |
| Last updated | 2026-10-15 |
| Status | confirmed — 3 open items (Appendix A items 1, 2, 6) |
| Owner / author | Product owner: anh Đức · written by Claude (`ba-writer`, retroactive mode) |
| API contract | `docs/specs/api/openapi.yaml` (owned by the backend; this spec never restates it) |
| Architecture decisions | none needed for the feature (read-only public catalogue, no per-user data; see §2.7). Production photo storage is a pending cross-feature decision (Appendix A item 1) |
| Related product specs | `temple` (`TPL`), `deity` (`DTY`), `article` (`ART`) — same catalogue conventions |
| Downstream | Backend: `docs/specs/features/gallery/` · Website: `Docs/specs/features/gallery/` · iOS: `Docs/Specs/Gallery/` · Android: `Docs/Specs/Gallery/` |

> See `changelog.md` in this folder for the version history. Platform specs cite this file as
> `docs/product/gallery/product.md@<version>` plus the IDs they implement.

---

## 1. Introduction

### 1.1 Purpose

This document is the single source of product truth for InnerLamp's Buddhist photo gallery on every
platform: which photos a visitor sees and in which order, how the topic filter works, what the photo
viewer shows and allows, and how every photo is credited to its author under its licence.

### 1.2 Scope

**In scope:**

- A block of **featured photos** on the home view.
- The **whole gallery**, page by page, with a **topic filter**
  ("Tất cả · Chùa · Tượng Phật · Cảnh thiền · Lễ hội").
- A **photo viewer** (large photo) that pages through the whole filtered gallery, with the photo's
  topic, title, place and **credit** (author, licence, link to the source).
- **Save the photo** to the device on the apps.
- Loading the gallery content (operator task; no user-facing editing).

**Out of scope:**

- Favourite (heart) — removed for v1 (anh Đức, 2026-10-15); no "favourite photos" view exists.
- Set as wallpaper — removed (anh Đức, 2026-10-15); iOS does not allow it.
- Share — hidden on every platform for now (anh Đức, 2026-10-15).
- Adding, editing or hiding photos (back office) — later; v1 content comes from the content file
  (anh Đức, 2026-10-15).
- Staggered (masonry) layout — later; it needs each photo's dimensions (anh Đức, 2026-10-15).
- Search, zoom inside the viewer, a public web page per photo.
- Temple photos inside the temple page (`temple`) and deity images (`deity`).

### 1.3 Product overview

InnerLamp is a Buddhist companion app (website, iOS, Android) with a practice library, a lunar
calendar, temples and family features. The gallery is a public, read-only collection of beautiful
Buddhist photos — temples, Buddha statues, meditation scenes and festivals — gathered from Wikimedia
Commons under free licences. It is reachable from the home view without an account and is meant for
quiet browsing.

### 1.4 Glossary

| Term | Definition | Not to be confused with |
|---|---|---|
| Photo | One entry of the gallery: a picture, its topic, title, place and credit | A temple's or deity's own image |
| Topic (chủ đề) | One of four groups a photo belongs to: Chùa (temples), Tượng Phật (Buddha statues), Cảnh thiền (meditation scenes), Lễ hội (festivals) (BR-GAL-01) | "Thiền & đời sống" — former website label, replaced |
| Featured photos (nổi bật) | The first 10 photos of the gallery order (BR-GAL-02) | A popularity ranking |
| Gallery order | The order every list shows: topics rotate, the content file's order is kept within a topic (BR-GAL-02) | — |
| Photo viewer | The large-photo view opened from the gallery ("lightbox" on the website) | — |
| Credit (ghi công) | "Ảnh: {author} · {licence}" plus a link to the photo's source page (BR-GAL-04) | — |
| Licence | The free licence the author published the photo under (CC BY, CC BY-SA in several versions, CC0, public domain) | — |

### 1.5 References

- Design prototype: `InnerLamp Prototype.dc.html`, screen `photos` ("Ảnh chùa đẹp") with its viewer
  block (`pvOpen`), and the home section "Ảnh chùa đẹp"; feature description
  `innerlamp-iOS/Docs/innerlamp-project/project/docs/Mo-ta-tinh-nang.md` §"Ảnh chùa đẹp" (lines
  109, 197–198). The web adaptation `innerlamp-website/Docs/innerlamp-web-adaptation/` has the same
  sections.
- Prior documents (rewritten into this spec): backend `docs/specs/features/gallery/` v1.1
  (`requirements.md`, `api-contract.md`), `docs/assumptions/gallery/A-001`; iOS
  `Docs/Specs/Photos/` v1.2, `Docs/Specs/PhotoViewer/` v1.0, `AS-Photos-001`, `AS-PhotoViewer-001`,
  `TD-Photos-001`, `TD-Photos-002`, `TD-PhotoViewer-001`, `TD-PhotoViewer-002`; Android
  `Docs/home_assumptions.md` row 1. The website and Android had no gallery spec.
- Questions and answers: `/mnt/project-files/reports/product/gallery/2026-10-15-cau-hoi-truoc-khi-viet.md`
  (Q1–Q14, answered by anh Đức on 2026-10-15).
- v1.1: questions raised by the platform specs written against v1.0 — website
  `Docs/specs/features/gallery/requirements.md`, iOS `Docs/Specs/Gallery/requirements.md`, Android
  `Docs/Specs/Gallery/requirements.md` — answered by anh Đức on 2026-10-15 (P1–P6 shared, A1–A4 apps,
  W1–W7 website; Appendix A).
- Platform folder names: Backend `gallery` · Website `gallery` · iOS `Gallery` (merges the former
  `Photos` and `PhotoViewer` spec folders) · Android `Gallery`. Code folders keep their own names
  (iOS `Photos`, `PhotoViewer`; Android `photos`; website `gallery`).

---

## 2. Overall description

### 2.1 Product perspective

Retroactive spec of a feature that already runs in part: the backend serves the gallery; the website
and iOS show it from the backend; Android shows six sample photos. This version changes, for every
platform: one feature name and one set of topic labels, a credit on every large photo, a viewer that
pages through the whole filtered gallery, a home "+N" that counts the whole gallery, and no favourite,
wallpaper or share action.

Touches: `home` (the featured block). No other feature depends on the gallery.

### 2.2 Product functions

- Featured photos on the home view (`FR-GAL-001`)
- The whole gallery, page by page, with a topic filter (`FR-GAL-002`, `FR-GAL-003`)
- The photo viewer, its navigation and the credit (`FR-GAL-004` … `FR-GAL-006`)
- Save a photo on the apps (`FR-GAL-007`)
- Empty and error states (`FR-GAL-008`)
- Gallery content loading by the operator (`FR-GAL-009`)

### 2.3 User classes and characteristics

| Actor | Description | Permissions |
|---|---|---|
| Visitor | Anyone, signed in or not (anh Đức, 2026-10-15) | Browses everything in this feature; saves photos on the apps |
| Operator | Runs the content load | Loads and updates the gallery; not a user-facing role |

### 2.4 Operating environment

| Platform | In scope | Notes |
|---|---|---|
| Backend | yes | Stores the gallery and its credits, orders and filters it, serves pages and the photo files |
| Website | yes | Home mosaic, gallery pages with page numbers, lightbox |
| iOS | yes | Home block, gallery with continuous loading, viewer, save to the photo library |
| Android | yes | Same as iOS; today sample data only |

Network expectation: online. No offline requirement.

### 2.5 Constraints

- **Licences.** Every photo is published under a free licence that, for most of them (CC BY and
  CC BY-SA), requires crediting the author and naming the licence where the photo is shown. Showing a
  large photo without its credit is a licence breach, not a cosmetic defect (BR-GAL-04).
- Titles and places exist in Vietnamese only (BR-GAL-09).

### 2.6 User documentation

N/A — a read-only gallery needs no help content.

### 2.7 Assumptions and dependencies

- **Assumptions:** the gallery is public and read-only for users and changes only when the operator
  reloads it. It is small (86 photos on 2026-10-15: 41 Chùa, 21 Tượng Phật, 12 Cảnh thiền, 12 Lễ hội).
  No architecture decision is needed for the feature (anh Đức accepted the "no ADR" proposal,
  2026-10-15): favourites were removed, so there is no per-user data.
- **Dependencies:** the device photo library on the apps (FR-GAL-007); the browser to open a photo's
  source page.
- **Release dependency (apps):** the photo viewer on iOS and Android is not released to users until
  the backend carries the credit in every list (Appendix B item 1). The apps do not fetch each photo's
  credit separately in the meantime, since a large photo without its credit breaches its licence
  (§2.5, BR-GAL-04) (anh Đức, 2026-10-15, A2).
- Where the photo files live and how they are served in production is shared with `temple`, `deity`
  and `article` and is left to a separate ADR:
  > ⚠️ Not confirmed: architecture decision pending — where catalogue photos are stored and served in production (needs an ADR, shared with temple, deity, article) — blocks production release on the Backend, not the platform specs.

---

## 3. Specific requirements

### 3.1 External interface requirements

#### 3.1.1 User interfaces

| View | Platforms | Shows |
|---|---|---|
| Home block "Ảnh đẹp Phật giáo" | all clients | Featured photo tiles (FR-GAL-001) and "Xem tất cả" (see all) to the gallery |
| Gallery "Ảnh đẹp Phật giáo" | all clients | Title, subtitle with the photo count, chips "Tất cả · Chùa · Tượng Phật · Cảnh thiền · Lễ hội", a grid of photo tiles (title and place on each), more pages (FR-GAL-002, FR-GAL-003) |
| Photo viewer | all clients | Close, counter "N / total", the large photo, previous / next, topic (as a small heading), title, place, credit, and on the apps "Lưu ảnh" (FR-GAL-004…007) |
| Empty / error state | all clients | FR-GAL-008 |

Layout and styling are defined by the prototype, not here, except the tile shape of BR-GAL-05.

#### 3.1.2 Hardware interfaces

N/A — no device capability beyond the photo library of the apps (FR-GAL-007).

#### 3.1.3 Software interfaces

The backend must: give the featured photos; give one page of the gallery, optionally narrowed to one
topic, with the number of matches; give each photo's small and large picture, topic, title, place and
credit (author, licence, source link) **in every list**, so no client needs a second call to credit a
photo. Contract: `docs/specs/api/openapi.yaml`, tag Gallery. The lists, the filter and the photo detail
exist; the credit in the lists does **not** yet (Appendix B).

#### 3.1.4 Communication interfaces

N/A.

### 3.2 Business rules

**BR-GAL-01 — Feature name, topics and their labels**
- **Rule:** The feature is called "Ảnh đẹp Phật giáo" (Buddhist photos) on every platform, as the home
  block title and the gallery title. Every photo has exactly one topic. The topic labels are "Chùa",
  "Tượng Phật", "Cảnh thiền" and "Lễ hội", the same on every platform. The filter offers, in this
  order, "Tất cả" (all, no filter), "Chùa", "Tượng Phật", "Cảnh thiền", "Lễ hội", one choice at a time.
- **Examples:** a meditation photo shows "CẢNH THIỀN" as the small heading in the viewer; the website
  chip "Thiền & đời sống" becomes "Cảnh thiền" and moves before "Lễ hội".
- **Computational:** no
- **Applies to:** Backend (the four topics), all clients
- **Source:** anh Đức, 2026-10-15 (Q2: "Ảnh đẹp Phật giáo"; Q3: prototype labels and order). Replaces
  "Ảnh chùa đẹp" on iOS and the prototype, and the website label "Thiền & đời sống" and chip order —
  drift

**BR-GAL-02 — Gallery order and featured photos**
- **Rule:** Photos are ordered by rotating the topics in the fixed order Chùa → Tượng Phật → Cảnh thiền
  → Lễ hội → Chùa …, taking at each turn the next photo of that topic in the content file's order; a
  topic with no photo left is skipped and the others keep rotating. The same order is used for the
  featured list, every gallery page and every viewer. A topic filter keeps that order among the photos
  of the topic (which is the content file's order for that topic). The featured list is the first 10
  photos of the unfiltered order; fewer than 10 → all of them; none → the block shows its empty state
  (FR-GAL-008).
  How many featured photos a block shows is a choice of each client's layout (§3.6); a client that
  shows fewer shows the first ones, in order.
- **Examples:** 41 Chùa, 21 Tượng Phật, 12 Cảnh thiền, 12 Lễ hội → the first 48 photos rotate all four
  topics, then Chùa and Tượng Phật alternate, then only Chùa; featured = 3 rounds of the four topics
  minus the last two.
- **Computational:** yes → `rules/gallery-order.json`
- **Applies to:** Backend (clients show the order they receive)
- **Source:** backend v1.0 behaviour (the content file groups photos by topic, so its raw order would
  make the featured list all temples)

**BR-GAL-03 — Pages**
- **Rule:** The gallery is shown page by page. The apps load 20 photos at a time and load the next 20
  as the visitor reaches the end of the grid; the website shows 12 photos per numbered page. The last
  page may hold fewer. A page past the last one is empty, not an error. The count in the subtitle and
  the number of pages follow the current filter.
- **Examples:** 86 photos → apps: 5 loads (the 5th has 6); website: 8 pages (the 8th has 2); filter
  "Lễ hội" with 12 photos → apps 1 load, website 1 page.
- **Computational:** no
- **Applies to:** Backend, all clients (§3.6)
- **Source:** backend v1.0 (20); website (12, cut from the backend pages); iOS (20)

**BR-GAL-04 — Credit on every large photo**
- **Rule:** Wherever a photo is shown large (the viewer, the website lightbox) the credit is shown with
  it: "Ảnh: {author} · {licence}" (photo: author · licence) and a link that opens the photo's source
  page in the browser. The credit is shown for every licence, including CC0 and public domain, so the
  rule has no exception. Tiles in the grid and in the home block need no credit. The credit changes
  with the photo when the visitor pages through the viewer. A photo with no source link still shows
  its credit; the credit then opens nothing.
- **Examples:** "Chùa Một Cột" → "Ảnh: thalling55 · CC BY 2.0", the link opens its Wikimedia Commons
  page; a CC0 photo still shows its author and "CC0"; a photo without a source link shows
  "Ảnh: {author} · {licence}" that cannot be chosen.
- **Computational:** no
- **Applies to:** Backend (every photo carries its credit), all clients
- **Source:** anh Đức, 2026-10-15 (Q4; P6 photo without a source link); website already shows it; iOS
  shows none (`TD-Photos-001`, High) — drift

**BR-GAL-05 — Tile shape**
- **Rule:** Every tile in the gallery grid has the same 4:3 landscape shape; the photo fills the tile
  and is cropped at the edges when its own shape differs. The viewer shows the large photo in a 4:3
  frame filled the same way (cropped, never letterboxed), so opening a tile and closing the viewer keep
  the same picture.
- **Examples:** a portrait photo of a statue fills its tile and loses its top and bottom edges.
- **Computational:** no
- **Applies to:** all clients
- **Source:** anh Đức, 2026-10-15 (Q10: uniform grid, masonry later; P3: the viewer crops to 4:3, no
  letterbox); iOS `AS-Photos-001`, `AS-PhotoViewer-001`

**BR-GAL-06 — Viewer covers the whole filtered gallery**
- **Rule:** The viewer opened from a tile pages through the whole gallery under the current filter, in
  the gallery order, not only through the photos already loaded: when the visitor nears the last loaded
  photo the next ones are loaded. The counter shows the photo's position and the total number of
  photos under the current filter ("N / total"). Previous is not offered on the first photo, next is
  not offered on the last.
- **Examples:** filter "Tất cả", 86 photos, 20 loaded, the visitor opens the 3rd → "3 / 86"; paging to
  the 20th loads the next photos and "21 / 86" follows; filter "Lễ hội" → "1 / 12" … "12 / 12", no next
  on the 12th.
- **Computational:** no
- **Applies to:** all clients
- **Source:** anh Đức, 2026-10-15 (Q11). Replaces the iOS snapshot of loaded photos with a loaded-count
  total (`TD-PhotoViewer-002`) and the website lightbox limited to the current page — drift

**BR-GAL-07 — Home "+N ảnh"**
- **Rule:** On the apps, the home block shows the first 3 featured photos as tiles and covers the last
  tile with "+N ảnh" (N more photos), where N = the number of photos in the whole gallery minus the
  tiles shown. The overlay appears only when N > 0. The website home mosaic has no "+N" (§3.6).
- **Examples:** 86 photos → "+83 ảnh"; 3 photos → no overlay; 2 photos → 2 tiles, no overlay.
- **Computational:** yes → `rules/gallery-home-more.json`
- **Applies to:** iOS, Android (Backend gives the total)
- **Source:** anh Đức, 2026-10-15 (Q12). Replaces the iOS count of the remaining featured photos (at
  most 7) — drift

**BR-GAL-08 — No favourite, wallpaper or share**
- **Rule:** No platform offers a favourite (heart), "Đặt hình nền" (set as wallpaper) or "Chia sẻ"
  (share) action on a photo. The only photo action is "Lưu ảnh" (save) on the apps (FR-GAL-007).
- **Examples:** the iOS viewer loses its heart, wallpaper and share buttons and their toasts.
- **Computational:** no
- **Applies to:** all clients
- **Source:** anh Đức, 2026-10-15 (Q6 favourite removed, Q7 wallpaper removed, Q8 share hidden on
  every platform for now). The iOS viewer shows all four as stubs that claim success
  (`TD-PhotoViewer-001`) — drift

**BR-GAL-09 — Content language**
- **Rule:** Photo titles and places are in Vietnamese only and are shown as they are in every interface
  language; only the interface copy (labels, buttons, messages) is translated.
- **Examples:** the English website shows "Chùa Một Cột" and "Hà Nội, Việt Nam" unchanged, with the
  English chip labels.
- **Computational:** no
- **Applies to:** all clients
- **Source:** anh Đức, 2026-10-15 (Q14; W3: confirmed for the English website)

### 3.3 Functional requirements

#### Browsing

**FR-GAL-001 — Featured photos on the home view**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a visitor opens the home view THE SYSTEM SHALL show the "Ảnh đẹp Phật giáo"
  block with the first featured photos (BR-GAL-02) and a "Xem tất cả" action; on the apps the last tile
  carries "+N ảnh" (BR-GAL-07). Choosing a tile, "+N ảnh" or "Xem tất cả" opens the gallery.
- **Business rules:** BR-GAL-01, BR-GAL-02, BR-GAL-07
- **Verification method:** automated test on each platform; vectors `gallery-order.json` on the
  backend, `gallery-home-more.json` on the apps
- **Source:** backend v1.0; prototype home section; anh Đức, 2026-10-15 (Q12)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Featured photos on an app home view
    Given the gallery holds 86 photos
    When a visitor who is not signed in opens the home view of an app
    Then the "Ảnh đẹp Phật giáo" block shows the first 3 featured photos
    And the last tile carries "+83 ảnh"

  Scenario: A tile opens the gallery
    When the visitor chooses a photo tile in the home block
    Then the gallery "Ảnh đẹp Phật giáo" opens
  ```

**FR-GAL-002 — The whole gallery, page by page**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a visitor opens the gallery THE SYSTEM SHALL show the first photos in the
  gallery order as a grid of tiles, each with its title and place, with the number of photos in the
  subtitle, and SHALL show the next photos on request (BR-GAL-03).
- **Business rules:** BR-GAL-02, BR-GAL-03, BR-GAL-05
- **Verification method:** automated test on each platform
- **Source:** backend v1.0; website; iOS Photos v1.2
- **Acceptance criteria:**
  ```gherkin
  Scenario: First photos on an app
    Given the gallery holds 86 photos
    When a visitor opens the gallery on an app
    Then 20 tiles appear in the gallery order, each with its title and place
    And the subtitle gives 86 photos

  Scenario: Next photos on an app
    Given 20 photos are shown
    When the visitor reaches the end of the grid
    Then the next 20 photos are added after them and the 20 already shown stay

  Scenario: Last page on the website
    Given the gallery holds 86 photos
    When a visitor opens page 8 of the gallery on the website
    Then 2 tiles appear and no further page is offered
  ```

**FR-GAL-003 — Topic filter**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a visitor chooses a topic chip THE SYSTEM SHALL show, from the first photos,
  only the photos of that topic with their count in the subtitle; "Tất cả" removes the filter.
- **Business rules:** BR-GAL-01, BR-GAL-02, BR-GAL-03
- **Verification method:** automated test on each platform
- **Source:** backend v1.0; website; iOS Photos v1.2
- **Acceptance criteria:**
  ```gherkin
  Scenario: Filter festivals
    Given the gallery holds 12 festival photos
    When the visitor chooses "Lễ hội"
    Then only photos of the topic "Lễ hội" appear, from the first one, and the subtitle gives 12 photos

  Scenario: Back to all photos
    Given the filter "Lễ hội" is chosen
    When the visitor chooses "Tất cả"
    Then photos of every topic appear in the gallery order
  ```

#### Photo viewer

**FR-GAL-004 — Open and close the viewer**
- **Priority:** Must
- **Platforms:** Website, iOS, Android
- **Description:** WHEN the visitor chooses a tile in the gallery THE SYSTEM SHALL open the viewer on
  that photo, showing the counter, the large photo, its topic, title, place and credit; WHEN the
  visitor closes the viewer THE SYSTEM SHALL return to the gallery at the photo last shown.
- **Business rules:** BR-GAL-04, BR-GAL-05, BR-GAL-06
- **Verification method:** automated test on each client; manual QA against the prototype
- **Source:** prototype `photos` (`pvOpen`); iOS PhotoViewer v1.0; website lightbox
- **Acceptance criteria:**
  ```gherkin
  Scenario: Open a photo
    Given the gallery shows "Chùa Một Cột" as the 1st photo of 86
    When the visitor chooses its tile
    Then the viewer shows the large photo of "Chùa Một Cột", "CHÙA", "Hà Nội, Việt Nam" and "1 / 86"

  Scenario: Close after paging
    Given the viewer was opened on the 1st photo and the visitor moved to the 5th
    When the visitor closes the viewer
    Then the gallery shows the tile of the 5th photo
  ```

**FR-GAL-005 — Page through the viewer**
- **Priority:** Must
- **Platforms:** Website, iOS, Android
- **Description:** WHEN the visitor moves to the next or previous photo in the viewer THE SYSTEM SHALL
  show that photo with its own topic, title, place, credit and position, loading further photos when
  needed, through the whole gallery under the current filter (BR-GAL-06).
- **Business rules:** BR-GAL-02, BR-GAL-04, BR-GAL-06
- **Verification method:** automated test on each client
- **Source:** anh Đức, 2026-10-15 (Q11)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Page past the loaded photos
    Given the filter is "Tất cả", 20 of 86 photos are loaded and the viewer shows "20 / 86"
    When the visitor moves to the next photo
    Then the 21st photo of the gallery order appears with "21 / 86"

  Scenario: Last photo of a filter
    Given the filter is "Lễ hội" and the viewer shows "12 / 12"
    Then no next photo is offered
  ```

**FR-GAL-006 — Credit in the viewer**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a photo is shown in the viewer THE SYSTEM SHALL show its credit
  "Ảnh: {author} · {licence}" and, when the visitor chooses it, open the photo's source page in the
  browser; WHEN the photo has no source link THE SYSTEM SHALL still show the credit, which then cannot
  be chosen.
- **Business rules:** BR-GAL-04
- **Verification method:** automated test on each client; manual QA of every licence label
- **Source:** anh Đức, 2026-10-15 (Q4, P6); website lightbox; `TD-Photos-001`
- **Acceptance criteria:**
  ```gherkin
  Scenario: Credit shown
    When the viewer shows "Chùa Một Cột"
    Then "Ảnh: thalling55 · CC BY 2.0" appears with the photo

  Scenario: Open the source
    When the visitor chooses the credit of "Chùa Một Cột"
    Then its source page opens in the browser

  Scenario: Credit follows the photo
    Given the viewer shows a photo by "thalling55"
    When the visitor moves to a photo by another author
    Then the credit names the other author and that photo's licence

  Scenario: Photo without a source link
    Given a photo has an author and a licence but no source link
    When the viewer shows it
    Then its credit "Ảnh: {author} · {licence}" appears
    And choosing the credit opens nothing
  ```

**FR-GAL-007 — Save a photo**
- **Priority:** Should
- **Platforms:** iOS, Android
- **Description:** WHEN the visitor chooses "Lưu ảnh" (save photo) in the viewer THE SYSTEM SHALL save
  the large photo to the device photo library, asking, where the device requires it, only for
  permission to add photos (§3.6), and SHALL confirm "Đã lưu vào thư viện ảnh" (saved to the photo
  library) only after the photo is actually saved. Errors are shown as a short message (toast):
  - WHEN permission is refused THE SYSTEM SHALL show "Chưa có quyền lưu ảnh. Mở Cài đặt để cấp quyền."
    (no permission to save photos yet — open Settings to grant it) with an action that opens the
    device settings.
  - WHEN saving fails THE SYSTEM SHALL show "Không lưu được ảnh. Vui lòng thử lại." (couldn't save the
    photo — please try again).
- **Business rules:** BR-GAL-08
- **Verification method:** automated test on each app; manual QA on a device
- **Source:** anh Đức, 2026-10-15 (Q5; P2 error copy as a toast; A1 permission purpose text; A3 no
  permission on Android); prototype ("Lưu ảnh" and its toast); iOS stub (`TD-PhotoViewer-001`) — drift
- **Acceptance criteria:**
  ```gherkin
  Scenario: Save succeeds
    Given the visitor allows adding photos
    When the visitor chooses "Lưu ảnh" on "Chùa Một Cột"
    Then the large photo of "Chùa Một Cột" is in the device photo library
    And "Đã lưu vào thư viện ảnh" appears

  Scenario: Permission refused (where the device asks for it)
    Given the visitor refuses to allow adding photos
    When the visitor chooses "Lưu ảnh"
    Then no photo is saved
    And "Chưa có quyền lưu ảnh. Mở Cài đặt để cấp quyền." appears with an action that opens the device settings
    And "Đã lưu vào thư viện ảnh" does not appear

  Scenario: Save fails
    Given the large photo cannot be downloaded or written
    When the visitor chooses "Lưu ảnh"
    Then no photo is saved
    And "Không lưu được ảnh. Vui lòng thử lại." appears
    And "Đã lưu vào thư viện ảnh" does not appear
  ```

#### States

**FR-GAL-008 — Empty and error states**
- **Priority:** Must
- **Platforms:** Website, iOS, Android
- **Description:** A load failure is always shown, never hidden. The error message on every platform
  is "Không tải được danh sách ảnh. Vui lòng thử lại sau." (couldn't load the photo list — please try
  again later), with "Thử lại" (try again) wherever a retry is possible.
  - WHEN the gallery cannot be loaded THE SYSTEM SHALL show the error message with "Thử lại" and no
    photo tile.
  - WHEN loading more photos fails THE SYSTEM SHALL keep the photos already shown and show the error.
  - WHEN loading a newly chosen topic fails THE SYSTEM SHALL keep the photos already shown and move the
    chip selection back to the topic they belong to (website: §3.6).
  - WHEN the gallery or the chosen topic holds no photo THE SYSTEM SHALL show "Chưa có ảnh nào trong
    chủ đề này." (no photos in this topic yet).
  - WHEN the home block cannot be loaded THE SYSTEM SHALL still show the "Ảnh đẹp Phật giáo" block,
    with the error message and "Thử lại" in place of the tiles, on every client.
  - WHEN the gallery holds no photo THE SYSTEM SHALL still show the home block, with "Chưa có ảnh
    nào." (no photos yet) in place of the tiles, on every client.
  - WHEN the viewer cannot load the next photos THE SYSTEM SHALL stay on the current photo and show the
    error message as a short message (toast); moving to the next photo again retries the load.
- **Business rules:** BR-GAL-02, BR-GAL-06
- **Verification method:** automated test on each client
- **Source:** iOS Photos v1.2 (keep photos, revert chip); website messages; anh Đức, 2026-10-15 (P1
  error block everywhere and copy, P4/W2 empty home block, P5 viewer, W5 website variance). Replaces the
  v1.0 home block hidden on the apps and the website sample mosaic (Q12)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Backend unreachable
    Given the backend cannot be reached
    When a visitor opens the gallery
    Then an error message with "Thử lại" appears and no photo tile

  Scenario: Topic fails to load on an app
    Given photos of every topic are shown
    When the visitor chooses "Cảnh thiền" and loading fails
    Then the photos of every topic stay, the chip "Tất cả" is selected again and the error is shown

  Scenario: Home block when the backend is unreachable
    Given the backend cannot be reached
    When a visitor opens the home view on any client
    Then the "Ảnh đẹp Phật giáo" block appears with "Không tải được danh sách ảnh. Vui lòng thử lại sau." and "Thử lại"
    And no photo tile and no sample photo appears

  Scenario: Home block with an empty gallery
    Given the gallery holds no photo
    When a visitor opens the home view on any client
    Then the "Ảnh đẹp Phật giáo" block appears with "Chưa có ảnh nào."

  Scenario: Viewer cannot load the next photos
    Given the viewer shows "20 / 86" and the next photos are not loaded yet
    When the visitor moves to the next photo and loading fails
    Then the viewer stays on "20 / 86" and the error message appears briefly
    And moving to the next photo again retries the load

  Scenario: Topic fails to load on the website
    Given photos of every topic are shown on the website
    When the visitor chooses "Cảnh thiền" and loading fails
    Then the chip "Cảnh thiền" is selected, the error message appears with "Thử lại" and no photo tile
  ```

#### Operator

**FR-GAL-009 — Load gallery content**
- **Priority:** Must
- **Platforms:** Backend
- **Description:** WHEN the operator loads the gallery content file THE SYSTEM SHALL add new photos and
  update existing ones by their stable identifier, all or nothing: an invalid file (no photo, missing or
  duplicate identifier, unknown topic, missing title, missing picture, missing author or licence)
  changes nothing. The order of the file within each topic becomes the gallery order (BR-GAL-02).
  Photos keep their identifier across loads. The content file is kept with the backend source.
- **Business rules:** BR-GAL-01, BR-GAL-02, BR-GAL-04
- **Verification method:** automated test on the backend
- **Source:** backend v1.0 (seed); anh Đức, 2026-10-15 (Q9: content file committed, back office
  later). Requiring author and licence is new: it follows from BR-GAL-04 (all 86 photos have both)
- > ⚠️ Not confirmed: who runs the load in staging and production and where the photo files come from there (today a dev-only command and photo files kept on the developer's machine) — blocks production content, not the platform specs.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Reload after reordering
    Given "Chùa Một Cột" is the 1st temple photo
    When the operator loads a file where it is the 3rd temple photo
    Then it moves accordingly in the gallery order and no second "Chùa Một Cột" appears

  Scenario: Photo without credit
    When the operator loads a file with a photo that has no author
    Then the load is refused and the gallery is unchanged
  ```

### 3.4 Non-functional requirements

- **NFR-GAL-01 — Page by page:** no platform loads the whole gallery at once; browsing, the filter and
  the viewer work page by page whatever the number of photos. Platforms: all.
- **NFR-GAL-02 — Picture size:** tiles use the small picture; only the viewer and "Lưu ảnh" use the
  large one. One exception: the large tile of the website home mosaic may use the large picture
  (§3.6). Platforms: all clients.
- **NFR-GAL-03 — Responsive filter:** a topic change shows its first photos within 2 s on a 4G
  connection. Platforms: all.

### 3.5 Data requirements

| Data item | Meaning | Owned by | Retention / deletion | Private or public |
|---|---|---|---|---|
| Photo | Topic, title, place, small and large picture | InnerLamp gallery on the server | Kept; changed by the operator | Public |
| Credit | Author, licence, source page | Server, with the photo | As long as the photo exists | Public |
| Stable photo identifier | Never changes for a photo | Server | As long as the photo exists | Public |
| Gallery content file | The list of photos with their credits, in editorial order per topic | Backend source | Versioned with the code | Public |

No personal data is involved. A photo saved by a visitor belongs to the visitor's device; InnerLamp
keeps no record of it.

### 3.6 Platform variance

| ID | Platforms | Difference | Reason |
|---|---|---|---|
| BR-GAL-03 / FR-GAL-002 | Website vs apps | Website: 12 photos per numbered page (shareable page links). Apps: 20 photos per load, next load at the end of the grid | Web navigation vs mobile scrolling |
| FR-GAL-001 / BR-GAL-07 | Website vs apps | Website home: a mosaic of the first 4 featured photos, no "+N". Apps: 3 tiles with "+N ảnh" | Each layout shows what it fits |
| FR-GAL-007 | Website | No "Lưu ảnh"; the browser's own save function is available | Browsers save images natively |
| NFR-GAL-02 / FR-GAL-001 | Website | The large (double-size) tile of the home mosaic may use the large picture; the other tiles keep the small one | The small picture looks soft at that size; anh Đức, 2026-10-15 (W1) |
| FR-GAL-007 | iOS | Saving asks for add-only access to the photo library, with the purpose text "InnerLamp cần quyền thêm ảnh để lưu ảnh bạn chọn vào thư viện." (InnerLamp needs permission to add photos to save the photo you choose to your library); permission refused → the toast offers to open Settings | iOS requires permission to add photos; anh Đức, 2026-10-15 (A1) |
| FR-GAL-007 | Android | Saving asks for no permission, so the "permission refused → open Settings" branch exists only on iOS | Android lets an app add its own photos without a runtime permission; anh Đức, 2026-10-15 (A3) |
| FR-GAL-002 | Website | The gallery subtitle is one general sentence that lists no topic: "Những khoảnh khắc an lành từ chùa chiền khắp nơi" (peaceful moments from temples everywhere); the photo count under the current filter is still shown | The former subtitle listed every topic even under a filter; anh Đức, 2026-10-15 (W4) |
| FR-GAL-008 | Website vs apps | Website: choosing a topic or a page loads a new page, so a failed load shows the error message with "Thử lại" for the newly chosen topic or page (its chip selected) instead of keeping the photos shown and moving the chip back | Each website page has its own address; anh Đức, 2026-10-15 (W5) |

### 3.7 Software system attributes

- **Reliability:** identical content gives identical pages on every load (BR-GAL-02 stable order).
- **Availability:** backend unreachable → error state with retry (FR-GAL-008).
- **Security & privacy:** all data is public; no personal data; saving asks only for add-only access to
  the photo library.
- **Legal:** every large photo carries its credit (BR-GAL-04).
- **Maintainability / portability:** N/A.

---

## 4. Verification & traceability

### 4.1 Verification approach

Computational rules are verified by running `rules/*.json`: the gallery order and featured list
(BR-GAL-02) on the backend; the home "+N" (BR-GAL-07) on the apps. Functional requirements are verified
by each platform's automated tests and manual QA against the prototype; the credit (BR-GAL-04) is also
checked by hand on every licence label of the content file before release.

### 4.2 Traceability matrix

| ID | Backend | Website | iOS | Android | Note |
|---|---|---|---|---|---|
| BR-GAL-01 | done | drift | drift | drift | Web "Thiền & đời sống" and chip order; iOS and Android title "Ảnh chùa đẹp" |
| BR-GAL-02 | done | done | done | planned | Android sample data |
| BR-GAL-03 | done | done | done | planned | |
| BR-GAL-04 | drift | done | drift | planned | Backend gives the credit only on the photo detail; iOS shows none (`TD-Photos-001`) |
| BR-GAL-05 | n/a | drift | done | drift | Web tiles follow a mixed-size mosaic and the lightbox letterboxes; Android sample tiles have varying heights |
| BR-GAL-06 | n/a | drift | drift | planned | Web lightbox stays in the current page; iOS snapshot (`TD-PhotoViewer-002`) |
| BR-GAL-07 | done | n/a | drift | planned | iOS counts the remaining featured photos |
| BR-GAL-08 | n/a | done | drift | planned | iOS stub heart, wallpaper, share |
| BR-GAL-09 | done | done | done | planned | |
| FR-GAL-001 | done | drift | drift | planned | 2 scenarios; web home tiles do not open the gallery; iOS "+N" (BR-GAL-07) |
| FR-GAL-002 | done | done | done | planned | 3 scenarios |
| FR-GAL-003 | done | drift | done | planned | 2 scenarios; web labels (BR-GAL-01) |
| FR-GAL-004 | n/a | drift | done | planned | 2 scenarios; web counter within the page, no topic heading, close does not return to the last photo |
| FR-GAL-005 | n/a | drift | drift | planned | 2 scenarios; BR-GAL-06 |
| FR-GAL-006 | drift | done | drift | planned | 4 scenarios; credit in the lists (Appendix B); apps' viewer waits for it (§2.7) |
| FR-GAL-007 | n/a | n/a | drift | planned | 3 scenarios; iOS stub |
| FR-GAL-008 | n/a | drift | drift | planned | 6 scenarios; web: no "Thử lại", sample mosaic on home error/empty; iOS: no retry on the first load, home block hidden on error |
| FR-GAL-009 | drift | n/a | n/a | n/a | 2 scenarios; author and licence not yet required; file not yet committed |
| NFR-GAL-01 | done | done | done | planned | |
| NFR-GAL-02 | done | done | done | planned | |
| NFR-GAL-03 | planned | planned | planned | planned | |

### 4.3 Test vectors

| File | Rule | Cases |
|---|---|---|
| `rules/gallery-order.json` | BR-GAL-02 | 7 |
| `rules/gallery-home-more.json` | BR-GAL-07 | 5 |

---

## 5. Appendices

### Appendix A — Open questions

#### Open (unconfirmed)

1. > ⚠️ Not confirmed: architecture decision pending — where catalogue photos are stored and served in production (needs an ADR, shared with temple, deity, article) — blocks production release on the Backend, not the platform specs.
2. > ⚠️ Not confirmed: who runs the load in staging and production and where the photo files come from there (today a dev-only command and photo files kept on the developer's machine) — blocks production content, not the platform specs.
3. Resolved in v1.1 (viewer crops to 4:3, no letterbox) — see the table below (P3).
4. Resolved in v1.1 (save error copy, shown as a toast) — see the table below (P2).
5. Resolved in v1.1 (one load-error message on every platform) — see the table below (P1).
6. Content review: nobody has yet checked by hand that each of the 86 photos is what its title and place say and that its licence on the source page matches the content file. Not a spec blocker; recommended before public release.

#### Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| Is an ADR needed? | No for the feature; production photo storage left to a separate cross-feature ADR | 2026-10-15 |
| Folder names (Q1) | `gallery` / `Gallery`; iOS merges `Photos` and `PhotoViewer` | 2026-10-15 |
| Feature name (Q2) | "Ảnh đẹp Phật giáo" on every platform | 2026-10-15 |
| Topic labels and order (Q3) | Prototype: Tất cả, Chùa, Tượng Phật, Cảnh thiền, Lễ hội | 2026-10-15 |
| Credit (Q4) | Required on every large photo; carried in the lists | 2026-10-15 |
| Save (Q5) | Apps only, add-only permission | 2026-10-15 |
| Favourite, wallpaper, share (Q6–Q8) | Removed, removed, hidden on every platform for now | 2026-10-15 |
| Content management (Q9) | Content file, committed with the backend; back office later | 2026-10-15 |
| Grid shape (Q10) | Uniform 4:3; masonry later | 2026-10-15 |
| Viewer range (Q11) | Whole filtered gallery; total from the server | 2026-10-15 |
| Home block (Q12) | 3 tiles + "+N" (N = total − 3) on the apps; hidden on error (changed in v1.1 by P1: error block) | 2026-10-15 |
| Sign-in (Q13) | Not needed | 2026-10-15 |
| Content language (Q14) | Vietnamese only | 2026-10-15 |
| Load-error state (P1; item 5) | Always a visible error block, never hidden — home block on the apps and the website home included (no sample mosaic); copy "Không tải được danh sách ảnh. Vui lòng thử lại sau." with "Thử lại" where a retry is possible, on every platform (FR-GAL-008) | 2026-10-15 |
| Save errors on the apps (P2; item 4) | Toast: "Chưa có quyền lưu ảnh. Mở Cài đặt để cấp quyền." with an action to open Settings (iOS); "Không lưu được ảnh. Vui lòng thử lại." (FR-GAL-007) | 2026-10-15 |
| Viewer frame (P3; item 3) | Crops to 4:3, no letterbox (BR-GAL-05) | 2026-10-15 |
| Empty gallery on the home view (P4, W2) | The home block shows "Chưa có ảnh nào." on every client, not hidden, no sample mosaic (FR-GAL-008) | 2026-10-15 |
| Viewer cannot load the next photos (P5) | Error toast, stay on the current photo; moving next again retries (FR-GAL-008) | 2026-10-15 |
| Photo without a source link (P6) | Credit still shown, not choosable (BR-GAL-04, FR-GAL-006) | 2026-10-15 |
| iOS permission purpose text (A1) | "InnerLamp cần quyền thêm ảnh để lưu ảnh bạn chọn vào thư viện." (§3.6) | 2026-10-15 |
| Temporary per-photo credit on the apps (A2) | No; the apps' viewer is not released until the credit is in the lists (§2.7) | 2026-10-15 |
| Save permission on Android (A3) | None asked; the "open Settings" branch is iOS only (§3.6) | 2026-10-15 |
| Android album folder (A4) | Platform detail, left to the Android spec | 2026-10-15 |
| Large home tile on the website (W1) | May use the large picture (NFR-GAL-02 exception, §3.6) | 2026-10-15 |
| Sample mosaic copy on the English website (W3) | Moot: the sample mosaic is no longer shown (P1); real titles and places stay Vietnamese (BR-GAL-09) | 2026-10-15 |
| Website gallery subtitle (W4) | "Những khoảnh khắc an lành từ chùa chiền khắp nơi" (peaceful moments from temples everywhere) — one general sentence listing no topic (§3.6) | 2026-10-15 |
| Website failed topic or page load (W5) | Error block with "Thử lại" at the newly chosen topic or page (§3.6) | 2026-10-15 |
| Website filter address and closing the lightbox on another page (W6, W7) | Website navigation details, handed to the website platform spec | 2026-10-15 |

### Appendix B — Contract handoff (backend)

1. Every photo in the featured list and in the gallery pages carries its credit (author, licence,
   source link) in addition to what it carries today (BR-GAL-04, FR-GAL-006). Additive: existing
   clients keep working. The photo detail stays.
2. The content loader refuses a photo without author or licence (FR-GAL-009).
3. The content file `assets/data/buddhist_gallery.json` is committed (ignore-rule exception added with
   this spec) and is the input of the content load in every environment. The photo files (about 32 MB)
   stay outside version control until Appendix A item 1 is decided.
4. The backend platform spec `docs/specs/features/gallery/` v1.1 still describes the contract as "not
   in openapi" and keeps a client copy `api-contract.md`; `spec-writer` brings it in line when it cites
   this spec, and the copy is deleted in the cleanup step.
