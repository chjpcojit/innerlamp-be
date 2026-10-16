# Product Spec — Dharma articles (Bài viết Phật pháp hay)

| Field | Value |
|---|---|
| Document type | Product spec — Software Requirements Specification, IEEE 29148-2018 structure |
| Feature | Dharma articles: featured articles, all articles page by page, the article page with "Đọc tiếp" and Share (`article`) |
| Feature code | `ART` — IDs are `BR-ART-NN`, `FR-ART-NNN`, `NFR-ART-NN` |
| Product | InnerLamp |
| Platforms in scope | Backend, Website, iOS, Android |
| Version | 1.0 |
| Last updated | 2026-10-15 |
| Status | confirmed — 4 open items (Appendix A) |
| Owner / author | Product owner: anh Đức · written by Claude (`ba-writer`, retroactive mode) |
| API contract | `docs/specs/api/openapi.yaml` (owned by the backend; this spec never restates it) |
| Architecture decisions | ADR-0004 (shared links are website URLs opened by the apps) — Accepted; no other decision needed (§2.7) |
| Related product specs | `deity` (`DTY`), `temple` (`TPL`) — same catalogue conventions · `home` (not written yet) hosts the featured block |
| Downstream | Backend: `docs/specs/features/article/` · Website: `Docs/specs/features/article/` · iOS: `Docs/Specs/Article/` · Android: `Docs/Specs/Article/` |

> See `changelog.md` in this folder for the version history. Platform specs cite this file as
> `docs/product/article/product.md@<version>` plus the IDs they implement.

---

## 1. Introduction

### 1.1 Purpose

This document is the single source of product truth for InnerLamp's Dharma articles on every
platform: which articles a visitor sees first, how they browse the rest, what an article page shows,
what it suggests to read next, and how an article is shared.

### 1.2 Scope

**In scope:**

- **Featured articles** on the home view.
- **All articles** ("Bài viết Phật pháp hay"), page by page (10 per page).
- The **article page**: cover image, title, author, date, reading time, body, scripture quote, source
  attribution and summary notice.
- **"Đọc tiếp"** (read next): two related articles at the end of the article page.
- **Share** an article as a link that opens the app when installed, else the website (ADR-0004).
- Empty and error states.
- Loading the article content (operator task; no user-facing editing).

**Out of scope:**

- **Save / bookmark** ("Lưu bài" ↔ "Đã lưu") — not in this version; the action is not shown (anh Đức,
  2026-10-15).
- Search, filtering by category or tag, reading progress, font size, listening (text to speech),
  offline reading.
- Writing, editing or removing articles in the back office (later feature; anh Đức, 2026-10-15).
- The share-verse screen ("Chia sẻ câu kinh") of the prototype.

### 1.3 Product overview

InnerLamp is a Buddhist companion app (website, iOS, Android). Dharma articles are a curated,
public, read-only reading list: short summaries of Buddhist teachings from Vietnamese and English
sources, each crediting its original. They appear on the home view, can be browsed in full, and each
article leads to two more on the same topic. Visitors do not need an account.

### 1.4 Glossary

| Term | Definition | Not to be confused with |
|---|---|---|
| Article (bài viết) | One entry of the reading list: title, category, author, optional date, reading time, summary, body paragraphs, optional quote, source | A community post by a user (`community-post`) |
| Category (thể loại / chủ đề) | One free-text label per article, e.g. "Thiền định", "Chánh niệm", "Đời sống"; shown, never used as a filter in this version | A tag; an article can have several tags |
| Editorial order | The position the editors gave the article in the content file (BR-ART-01) | Publication date, popularity |
| Featured articles (nổi bật) | The first articles of the editorial order (BR-ART-02) | A popularity ranking |
| Related articles ("Đọc tiếp") | Two articles suggested at the end of an article page (BR-ART-04) | The next articles in the list |
| Source (nguồn) | The site and page the article summarises | The image credit |
| Summary notice | The line telling the reader the article is a summary of the original (BR-ART-06) | — |
| Cover image / thumbnail | The large picture on the article page / the small square picture on cards (BR-ART-05) | — |

### 1.5 References

- Design prototype: `InnerLamp Prototype.dc.html` and `Mo-ta-tinh-nang.md` (copy in
  `innerlamp-iOS/Docs/innerlamp-project/project/docs/`): home §11 "Bài viết Phật pháp hay" (5 cards),
  "Tất cả bài viết", "Bài viết Phật pháp". Web adaptation:
  `innerlamp-website/Docs/innerlamp-web-adaptation/project/InnerLamp Web App.dc.html`.
- Prior documents (rewritten into this spec): backend `docs/specs/features/article/` v2.0,
  `docs/assumptions/article/A-001` (Commons images), `A-002` (hosted thumbnails); iOS
  `Docs/Specs/Article/` v1.3 and `Docs/Specs/Articles/` v1.1; website `.assumptions.md` lines 12, 23
  (stale); Android `Docs/home_assumptions.md` rows 1, 3, 13. Website and Android had no feature spec.
- Questions and answers: `/mnt/project-files/reports/product/article/2026-10-15-cau-hoi-truoc-khi-viet.md`.
- Platform folder names: Backend `article` · Website `article` · iOS `Article` (merges the former
  `Article` and `Articles` spec folders) · Android `Article`. Code folders keep their own names.

---

## 2. Overall description

### 2.1 Product perspective

Retroactive spec of a feature that already runs in part: the backend serves 60 articles; the website
and iOS show them from the backend; Android shows five sample articles. This version settles, for every
platform: pages of 10, featured = editorial order, related articles from the same category, the image
order on cards and on the page, the source attribution, a link-based Share, no Save action and no
fake content when nothing can be loaded.

Touches: `home` (featured block), ADR-0004 (shared links).

### 2.2 Product functions

- Featured articles on the home view (`FR-ART-001`)
- All articles, page by page (`FR-ART-002`)
- The article page (`FR-ART-003`, `FR-ART-004`)
- "Đọc tiếp" (`FR-ART-005`)
- Share (`FR-ART-006`) and opening a shared link (`FR-ART-007`)
- Empty and error states (`FR-ART-008`)
- Content loading by the operator (`FR-ART-009`)

### 2.3 User classes and characteristics

| Actor | Description | Permissions |
|---|---|---|
| Visitor | Anyone, signed in or not | Reads everything in this feature; shares |
| Link recipient | Someone who opens a shared article link, with or without the app | Same as a visitor |
| Operator | Runs the content load | Loads the articles; not a user-facing role |

### 2.4 Operating environment

| Platform | In scope | Notes |
|---|---|---|
| Backend | yes | Stores the articles, orders them, serves pages, picks related articles |
| Website | yes | Home block, list with page numbers, article page; serves the shared links (ADR-0004) |
| iOS | yes | Home block, list with continuous loading, article page; opens shared links |
| Android | yes | Same as iOS; today sample data only |

Network expectation: online only. No platform keeps articles for offline reading (anh Đức,
2026-10-15).

### 2.5 Constraints

- Every article is a summary of a third-party page and most images are third-party pictures; seven
  images come from Wikimedia Commons under licences that require crediting the author. No permission
  has been obtained and no image credit is shown:
  > ⚠️ Not confirmed: whether InnerLamp may show these summaries, their images and the small copies of those images, and how image authors are credited — blocks public release of the content, not the platform specs.
- Article content is not translated: each article is shown in its own language (58 Vietnamese, 2
  English today) whatever the interface language.

### 2.6 User documentation

N/A — a read-only reading list needs no help content.

### 2.7 Assumptions and dependencies

- **Assumptions:** the articles are public and read-only for users and change only when the operator
  loads content. The list is small (60 today). No architecture decision is needed for the reading list
  itself (anh Đức accepted the "no ADR" proposal, 2026-10-15); the shared-link convention is ADR-0004
  (Accepted, 2026-10-15).
- **Dependencies:** a public website page per article (the shared link's fallback, ADR-0004); a
  domain for the website (ADR-0004 open question); the device share function on the apps.

---

## 3. Specific requirements

### 3.1 External interface requirements

#### 3.1.1 User interfaces

| View | Platforms | Shows |
|---|---|---|
| Home block "Bài viết Phật pháp hay" | all clients | Featured article cards (FR-ART-001) and "Xem tất cả" (see all) |
| All articles "Bài viết Phật pháp hay" | all clients | Title, "{loaded} / {total} bài" on the apps or the total on the website, article cards, more pages, end line "Bạn đã xem hết bài viết" (FR-ART-002) |
| Article card | all clients | Thumbnail (BR-ART-05), category, title (2 lines), author · "{n} phút đọc" |
| Article page "Bài viết Phật pháp" | all clients | FR-ART-003 … FR-ART-006 |
| Empty / error state | all clients | FR-ART-008 |

Layout and styling are defined by the prototype, not here.

#### 3.1.2 Hardware interfaces

N/A — no device capability beyond the share function of the apps (FR-ART-006).

#### 3.1.3 Software interfaces

The backend must: give the featured articles; give one page of all articles with the total; give one
article with its body; give the related articles of one article (BR-ART-04). Contract:
`docs/specs/api/openapi.yaml`, tag Articles. Featured, pages and the article exist; **related
articles do not yet** (Appendix B). The website must serve the association files of ADR-0004.

#### 3.1.4 Communication interfaces

Shared links: website URLs opened by the apps as verified links (ADR-0004).

### 3.2 Business rules

**BR-ART-01 — Editorial order**
- **Rule:** Articles are always ordered by their editorial position in the content file (first first),
  then by their stable identifier, so the order never changes between two loads. The same order is
  used for the featured list, every page of all articles and the related articles.
- **Examples:** two articles with the same position → the one with the smaller identifier first; an
  article added at the end of the file → last in the list.
- **Computational:** yes → `rules/article-order.json`
- **Applies to:** Backend (clients show the order they receive)
- **Source:** backend v1.0 behaviour; anh Đức, 2026-10-15 (Q5)

**BR-ART-02 — Featured articles**
- **Rule:** The featured list is the first 5 articles of the editorial order; fewer than 5 → all of
  them; none → the block shows its empty state (FR-ART-008). How many featured cards a block shows is a
  choice of each client's layout (§3.6); a client that shows fewer shows the first ones, in order. Not
  ranked by date or popularity.
- **Examples:** 60 articles → positions 1–5; the website home shows positions 1–4.
- **Computational:** no (a prefix of BR-ART-01)
- **Applies to:** Backend, all clients
- **Source:** backend v1.0; prototype home §11 (5 cards); anh Đức, 2026-10-15 (Q4, Q5)

**BR-ART-03 — Page size and end of list**
- **Rule:** One page holds **10** articles; the last page may hold fewer. A page past the last one is
  empty, not an error. The list shows how many articles exist; when the last page has been shown the
  apps show "Bạn đã xem hết bài viết" (you have seen every article).
- **Examples:** 60 articles → 6 pages; after page 6 the apps show the end line; 0 articles → empty
  state.
- **Computational:** no
- **Applies to:** Backend, all clients
- **Source:** backend v1.0 (10); anh Đức, 2026-10-15 (Q6: keep 10, drop the prototype's 20)

**BR-ART-04 — Related articles ("Đọc tiếp")**
- **Rule:** The related articles of an article are up to **2** other articles chosen in this order:
  1. articles of the **same category**, in editorial order (BR-ART-01);
  2. if fewer than 2, completed with the featured order (BR-ART-01 from the top), skipping articles
     already chosen.
  The article itself is never chosen. Two categories are the same when their labels are equal after
  trimming surrounding spaces and ignoring letter case. With only one article in the list there is no
  related article and the section is hidden.
- **Examples:** A (Thiền định) at position 7, B (Thiền định) at 3, C (Thiền định) at 12 → related of A
  = B, C; D is the only "Hiếu hạnh" article → related of D = positions 1 and 2 (or 1 and 3 when D is
  2nd); E (Chánh niệm) with one other "Chánh niệm" article F → related of E = F, then position 1.
- **Computational:** yes → `rules/article-related.json`
- **Applies to:** Backend (picks them); clients show what they receive
- **Source:** anh Đức, 2026-10-15 (Q7: "BE cung cấp thêm API lấy 2 bài cùng chủ đề"); fill-up and
  label matching are the defaults announced in the thread; replaces iOS (first 2 featured) and Android
  (next 2 in a cycle) — drift

**BR-ART-05 — Images**
- **Rule:** A card shows the article's thumbnail; without one, its cover image; without either, a
  placeholder. The article page shows the cover image; without one, the thumbnail; without either, a
  placeholder. An image that fails to load is replaced by the placeholder without any error message.
- **Examples:** article with only a cover → the card uses the cover; cover host down → placeholder on
  the page, the text still shows.
- **Computational:** no
- **Applies to:** all clients
- **Source:** website and iOS behaviour (cards); anh Đức, 2026-10-15 (Q8); iOS page shows only the cover
  — drift

**BR-ART-06 — Source attribution and summary notice**
- **Rule:** Every article page shows "Nguồn: {source name}" (source) with a link to the original page,
  and the notice "Đây là bản tóm tắt. Vui lòng đọc bài gốc để có đầy đủ nội dung." (this is a summary;
  please read the original for the full content). Without a source link the name is shown without a
  link; only web (http/https) links are offered.
- **Examples:** source "Giác Ngộ" with a link → "Nguồn: Giác Ngộ" opens the original page in the
  browser.
- **Computational:** no
- **Applies to:** all clients
- **Source:** website behaviour; iOS (source only, no notice) — drift; anh Đức, 2026-10-15 (Q9)

**BR-ART-07 — Missing values**
- **Rule:** On a card or the article page, a missing value is left out, never shown empty: no date →
  no date; no author → no author line; no quote → no quote block; no category → no category label; no
  tags → no tags. An article without body paragraphs shows its summary and the line "Bài viết chưa có
  nội dung chi tiết." (this article has no detailed content yet).
- **Examples:** 31 of the 60 articles have no date → their cards and pages show none.
- **Computational:** no
- **Applies to:** all clients
- **Source:** website and iOS behaviour (both hide empty values)

**BR-ART-08 — Language**
- **Rule:** Articles are listed and shown whatever the interface language, each in its own language,
  never translated. The article page declares the article's language to the reader's device (so
  hyphenation and screen readers use it). Interface text (headings, actions, states) follows the
  interface language where the platform has more than one (§3.6).
- **Examples:** interface in English → all 60 articles are listed; the 58 Vietnamese articles are shown
  in Vietnamese.
- **Computational:** no
- **Applies to:** all clients
- **Source:** website behaviour; anh Đức, 2026-10-15 (Q10)

**BR-ART-09 — No save action in this version**
- **Rule:** No platform shows a "Lưu bài" / "Đã lưu" action, nor a list of saved articles.
- **Examples:** the iOS and Android article pages lose their Save button.
- **Computational:** no
- **Applies to:** all clients
- **Source:** anh Đức, 2026-10-15 (Q2)

### 3.3 Functional requirements

#### Browsing

**FR-ART-001 — Featured articles on the home view**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a visitor opens the home view THE SYSTEM SHALL show featured article cards
  (BR-ART-02) and a "Xem tất cả" action that opens all articles; choosing a card opens the article page.
- **Business rules:** BR-ART-01, BR-ART-02, BR-ART-05, BR-ART-07
- **Verification method:** automated test on each platform
- **Source:** backend v1.0; prototype home §11
- **Acceptance criteria:**
  ```gherkin
  Scenario: Featured articles on the home view
    Given 60 articles exist
    When a visitor who is not signed in opens the home view
    Then article cards appear in the editorial order, starting with the first article
    And the block offers "Xem tất cả"

  Scenario: A featured card opens its article
    When the visitor chooses the first featured card
    Then the article page of that article appears
  ```

**FR-ART-002 — All articles, page by page**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a visitor opens all articles THE SYSTEM SHALL show the first 10 articles in the
  editorial order with the number of articles, and SHALL show the next pages on request: the website by
  page numbers, the apps by loading the next page as the visitor reaches the end of the list, showing
  "Đang tải thêm bài viết…" while loading, never loading the same page twice at once, and "Bạn đã xem
  hết bài viết" after the last page (§3.6).
- **Business rules:** BR-ART-01, BR-ART-03, BR-ART-05, BR-ART-07
- **Verification method:** automated test on each platform
- **Source:** backend v1.0; iOS; website; prototype "Tất cả bài viết"
- **Acceptance criteria:**
  ```gherkin
  Scenario: First page on the apps
    Given 60 articles exist
    When a visitor opens all articles on the app
    Then 10 article cards appear, the first of the editorial order
    And the header says "10 / 60 bài"

  Scenario: Reaching the end on the apps
    Given the visitor has loaded 50 of 60 articles
    When the visitor reaches the end of the list
    Then "Đang tải thêm bài viết…" appears, then 10 more cards
    And the header says "60 / 60 bài" and the list ends with "Bạn đã xem hết bài viết"

  Scenario: A page that fails to load keeps what was loaded
    Given the visitor has loaded 20 articles
    When loading the next page fails
    Then the 20 loaded cards stay on screen and the next page can be requested again
  ```

#### Article page

**FR-ART-003 — Open an article**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a visitor opens an article from the home view, all articles, "Đọc tiếp" or a
  shared link THE SYSTEM SHALL show its page without signing in; WHEN the article does not exist THE
  SYSTEM SHALL say it was not found and offer a way back; WHEN loading fails THE SYSTEM SHALL show the
  error state of FR-ART-008.
- **Business rules:** BR-ART-07
- **Verification method:** automated test on each platform
- **Source:** backend v1.0; iOS; website
- **Acceptance criteria:**
  ```gherkin
  Scenario: Open from the list
    When the visitor opens "Vipassana – nghệ thuật sống" from all articles
    Then its article page appears without signing in

  Scenario: Unknown article
    When a visitor opens an article that does not exist
    Then the page says the article was not found and offers a way back
  ```

**FR-ART-004 — Article page content**
- **Priority:** Must
- **Platforms:** Website, iOS, Android (Backend provides the data)
- **Description:** WHEN an article page is shown THE SYSTEM SHALL show, in this order: the category and
  "{n} phút đọc" (reading time); the cover image (BR-ART-05); the title; the author with initials and
  the date ("{d} tháng {m}" on the apps, the platform's date format on the website); the body paragraphs
  as plain text; the quote with its source; the source attribution and summary notice (BR-ART-06); the
  Share action (FR-ART-006); "Đọc tiếp" (FR-ART-005). Missing values follow BR-ART-07; the language
  follows BR-ART-08; there is no Save action (BR-ART-09).
- **Business rules:** BR-ART-05, BR-ART-06, BR-ART-07, BR-ART-08, BR-ART-09
- **Verification method:** automated test on each client; manual QA against the prototype
- **Source:** prototype "Bài viết Phật pháp"; iOS; website
- **Acceptance criteria:**
  ```gherkin
  Scenario: Full article
    Given an article has a cover, a date, 5 paragraphs and a quote
    When its page is shown
    Then the category, reading time, cover, title, author, date, 5 paragraphs, quote, source and summary notice appear in that order
    And no "Lưu bài" action appears

  Scenario: Article without a date or quote
    Given an article has no date and no quote
    When its page is shown
    Then no date and no quote block appear
  ```

**FR-ART-005 — "Đọc tiếp"**
- **Priority:** Should
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN an article page is shown THE SYSTEM SHALL show under "Đọc tiếp" the related
  articles of BR-ART-04 as cards; choosing one opens that article at its top. WHEN there is none, or
  they cannot be loaded, the section is hidden and the rest of the page is unaffected.
- **Business rules:** BR-ART-04, BR-ART-05
- **Verification method:** shared vectors `article-related.json` on the backend; automated test on each
  client
- **Source:** prototype ("2 bài kế"); iOS (first 2 featured — drift); Android (cycle — drift); website
  (none); anh Đức, 2026-10-15 (Q7)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Two articles of the same category
    Given "Vipassana – nghệ thuật sống" is in category "Thiền định" with two other "Thiền định" articles
    When its page is shown
    Then "Đọc tiếp" shows those two articles in the editorial order

  Scenario: Open a related article
    When the visitor chooses a card under "Đọc tiếp"
    Then that article's page appears scrolled to its top

  Scenario: Related articles fail to load
    Given the related articles cannot be loaded
    When an article page is shown
    Then the article is shown without the "Đọc tiếp" section
  ```

#### Sharing

**FR-ART-006 — Share an article**
- **Priority:** Should
- **Platforms:** Website, iOS, Android
- **Description:** WHEN the visitor chooses "Chia sẻ" on an article page THE SYSTEM SHALL share the
  article's title and its InnerLamp link (the website page of the article, ADR-0004) through the
  device's share function (apps) or the browser's share function, or copy the link when there is none
  (website).
- **Business rules:** —
- **Verification method:** automated test on each client; manual QA
- **Source:** prototype ("Chia sẻ"); iOS and Android (not built); anh Đức, 2026-10-15 (Q3); ADR-0004
- > ⚠️ Not confirmed: the website domain the link uses in each environment (ADR-0004 open question) — blocks the exact shared link on every client and FR-ART-007 end to end.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Share from the app
    When the visitor chooses "Chia sẻ" on an article page in the app
    Then the device share function opens with the article's title and its InnerLamp website link

  Scenario: Share on a desktop browser without a share function
    When the visitor chooses "Chia sẻ" on an article page of the website
    Then the article's link is copied and the visitor is told it was copied
  ```

**FR-ART-007 — Open a shared link**
- **Priority:** Should
- **Platforms:** Website, iOS, Android
- **Description:** WHEN someone opens a shared article link THE SYSTEM SHALL open that article's page in
  the app when the app is installed on the device, and the website page otherwise; WHEN the article no
  longer exists the app shows its not-found state (FR-ART-003).
- **Business rules:** —
- **Verification method:** manual QA on real devices (verified links need the real domain)
- **Source:** anh Đức, 2026-10-15 (Q3); ADR-0004
- **Acceptance criteria:**
  ```gherkin
  Scenario: Recipient has the app
    Given the app is installed on the recipient's phone
    When the recipient opens a shared article link
    Then the app opens on that article's page

  Scenario: Recipient has no app
    Given the app is not installed
    When the recipient opens a shared article link
    Then the website page of that article opens
  ```

#### States

**FR-ART-008 — Empty and error states**
- **Priority:** Must
- **Platforms:** Website, iOS, Android
- **Description:** WHEN articles cannot be loaded (no connection, backend unreachable) or none exist THE
  SYSTEM SHALL show an empty block in place of the content — the home block, all articles or the
  article page — and SHALL NOT show placeholder, sample or cached articles instead.
- **Business rules:** BR-ART-02
- **Verification method:** automated test on each client
- **Source:** anh Đức, 2026-10-15 (Q11: "mất mạng không tải được sẽ có 1 khối rỗng"); replaces the
  website's static home fallback and Android sample data — drift
- > ⚠️ Not confirmed: whether the empty block offers a "Thử lại" (retry) action and which text it shows — blocks the copy and the action on every client.
- **Acceptance criteria:**
  ```gherkin
  Scenario: No connection on the home view
    Given the device has no connection
    When a visitor opens the home view
    Then the "Bài viết Phật pháp hay" block is empty and no article card appears

  Scenario: No connection in all articles
    Given the device has no connection and nothing was loaded
    When a visitor opens all articles
    Then an empty block appears and no article card appears
  ```

#### Operator

**FR-ART-009 — Load article content**
- **Priority:** Must
- **Platforms:** Backend
- **Description:** WHEN the operator loads the article content file THE SYSTEM SHALL add the articles
  whose identifier is not stored yet, all or nothing: an invalid file (no article, missing or duplicate
  identifier, missing title, malformed date) changes nothing. The position in the file becomes the
  editorial order (BR-ART-01). Articles keep their identifier, so shared links stay valid. Loading the
  file is the only way to change articles in this version.
- **Business rules:** BR-ART-01
- **Verification method:** automated test on the backend
- **Source:** backend v2.0 (seed command); anh Đức, 2026-10-15 (Q14: "tạm thời toàn bộ là seed, sau
  này có BO")
- **Acceptance criteria:**
  ```gherkin
  Scenario: Load into an empty list
    Given no article exists
    When the operator loads a file of 60 articles
    Then 60 articles exist in the file's order

  Scenario: Invalid file
    When the operator loads a file with two articles of the same identifier
    Then the load is refused and the articles are unchanged
  ```
- > ⚠️ Not confirmed: how an article that is already loaded is corrected or removed (today the load never changes an existing article) — blocks content fixes after the first load, not the platform specs.

### 3.4 Non-functional requirements

- **NFR-ART-01 — Page by page:** no platform loads all articles at once; browsing works page by page
  whatever the number of articles. Platforms: all.
- **NFR-ART-02 — Light cards:** card images are small copies, not the full cover image, so a page of 10
  cards stays light on a mobile connection. Platforms: all.

### 3.5 Data requirements

| Data item | Meaning | Owned by | Retention / deletion | Private or public |
|---|---|---|---|---|
| Article | Title, category, author, date, reading time, summary, body, quote, tags, language, images, source, licence | InnerLamp content on the server | Kept; changed only by the operator | Public |
| Stable article identifier | Never changes; shared links and related articles refer to it | Server | As long as the article exists | Public |
| Editorial position | Order of the article in the content file | Server | Replaced at each load | Public |

No personal data is involved; sharing sends nothing to InnerLamp.

### 3.6 Platform variance

| ID | Platforms | Difference | Reason |
|---|---|---|---|
| FR-ART-001 / BR-ART-02 | Website | The home block shows 4 of the 5 featured articles | Grid of 4 columns; anh Đức, 2026-10-15 (Q4) |
| FR-ART-002 | Website vs apps | Website: page numbers (shareable page links), the first article of page 1 shown as a large card; a page number past the end leads to the last page. Apps: next page loads at the end of the list, header "{loaded} / {total} bài", end line | Web navigation and SEO vs mobile scrolling; anh Đức, 2026-10-15 (Q5) |
| FR-ART-006 | Website | Uses the browser's share function, or copies the link when there is none | No OS share sheet on desktop browsers |
| BR-ART-08 | iOS, Android | Interface text in Vietnamese only in this version; the website has Vietnamese and English | anh Đức, 2026-10-15 (Q10) |

### 3.7 Software system attributes

- **Reliability:** identical inputs give identical pages and related articles on every load (BR-ART-01).
- **Availability:** backend unreachable → empty block (FR-ART-008); no offline copy.
- **Security & privacy:** all data is public; no personal data. Shared links are verified for the
  apps (ADR-0004), so no other app can open them.
- **Maintainability / portability:** N/A.

---

## 4. Verification & traceability

### 4.1 Verification approach

Computational rules are verified by running `rules/*.json` on the backend: the editorial order
(BR-ART-01) and the related articles (BR-ART-04). Functional requirements are verified by each
platform's automated tests and manual QA against the prototype; shared links (FR-ART-007) need manual
QA on real devices once the domain exists.

### 4.2 Traceability matrix

| ID | Backend | Website | iOS | Android | Note |
|---|---|---|---|---|---|
| BR-ART-01 | done | done | done | planned | Android sample data |
| BR-ART-02 | done | done | done | planned | Website home shows 4 (variance) |
| BR-ART-03 | done | done | done | drift | Android pages of 20 (sample) |
| BR-ART-04 | planned | planned | drift | drift | iOS: first 2 featured; Android: next 2 in a cycle; website: none |
| BR-ART-05 | n/a | done | drift | planned | iOS page shows only the cover |
| BR-ART-06 | n/a | done | drift | planned | iOS shows no summary notice |
| BR-ART-07 | n/a | done | done | planned | |
| BR-ART-08 | n/a | done | done | planned | iOS and Android Vietnamese only (variance) |
| BR-ART-09 | n/a | done | drift | drift | iOS and Android show an in-memory Save |
| FR-ART-001 | done | done | done | planned | 2 scenarios |
| FR-ART-002 | done | done | done | planned | 3 scenarios |
| FR-ART-003 | done | done | done | planned | 2 scenarios |
| FR-ART-004 | done | done | drift | drift | 2 scenarios; iOS/Android Save button; iOS no notice |
| FR-ART-005 | planned | planned | drift | drift | 3 scenarios; needs the related-articles backend feature |
| FR-ART-006 | n/a | planned | planned | planned | 2 scenarios; iOS/Android share is a TODO |
| FR-ART-007 | n/a | planned | planned | planned | 2 scenarios; needs the domain (ADR-0004) |
| FR-ART-008 | n/a | drift | done | drift | 2 scenarios; website static home fallback; Android sample |
| FR-ART-009 | done | n/a | n/a | n/a | 2 scenarios |
| NFR-ART-01 | done | done | done | drift | Android builds 60 locally |
| NFR-ART-02 | done | done | done | planned | |

### 4.3 Test vectors

| File | Rule | Cases |
|---|---|---|
| `rules/article-order.json` | BR-ART-01 | 4 |
| `rules/article-related.json` | BR-ART-04 | 7 |

---

## 5. Appendices

### Appendix A — Open questions

#### Open (unconfirmed)

1. > ⚠️ Not confirmed: whether InnerLamp may show these summaries, their images and the small copies of those images, and how image authors are credited — blocks public release of the content, not the platform specs.
2. > ⚠️ Not confirmed: the website domain the link uses in each environment (ADR-0004 open question) — blocks the exact shared link on every client and FR-ART-007 end to end.
3. > ⚠️ Not confirmed: whether the empty block offers a "Thử lại" (retry) action and which text it shows — blocks the copy and the action on every client.
4. > ⚠️ Not confirmed: how an article that is already loaded is corrected or removed (today the load never changes an existing article) — blocks content fixes after the first load, not the platform specs.

#### Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| Is an ADR needed for the reading list? | No; the shared-link convention is ADR-0004 | 2026-10-15 |
| Scope | Featured, all articles, article page, "Đọc tiếp", Share | 2026-10-15 |
| Save / bookmark | Not in this version; action hidden | 2026-10-15 |
| Share | Title + InnerLamp website link; opens the app when installed (ADR-0004) | 2026-10-15 |
| Featured count and criterion | Backend gives 5 in editorial order; website home shows 4 | 2026-10-15 |
| Page size | 10; drop the prototype's 20 | 2026-10-15 |
| "Đọc tiếp" | 2 articles of the same category, picked by the backend | 2026-10-15 |
| Images | Card: thumbnail → cover → placeholder; page: cover → thumbnail → placeholder | 2026-10-15 |
| Source and notice | Always shown; rights left open (item 1) | 2026-10-15 |
| Language | Content not translated; apps Vietnamese only | 2026-10-15 |
| Offline | No offline copy; empty block | 2026-10-15 |
| Category | Label only, no filter | 2026-10-15 |
| Guests | Everything public | 2026-10-15 |
| Content updates | Content file load only; back office later | 2026-10-15 |

### Appendix B — Contract handoff (backend)

1. **Related articles** for one article (BR-ART-04): new read, public, up to 2 article summaries; not
   found when the article does not exist. Runs `rules/article-related.json`.
2. The article page needs the thumbnail too (BR-ART-05): already present in the backend response, but
   missing from the iOS contract copy — the backend platform spec states it.
3. The content file (`assets/data/buddhist_articles.json`, 60 articles) is committed with the backend
   (2026-10-15) and is the input of the content load in every environment (FR-ART-009).
4. The backend platform spec `docs/specs/features/article/` v2.0 is in Vietnamese and predates this
   spec; `spec-writer` rewrites it in English when it cites this spec (repo language policy).
