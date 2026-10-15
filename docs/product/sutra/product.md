# Product Spec — Sutra Library (Kinh sách)

| Field | Value |
|---|---|
| Document type | Product spec — Software Requirements Specification, IEEE 29148-2018 structure |
| Feature | Sutra library: browse, filter and search sutras, verses and mantras; the sutra page; reading the text; the sutras related to a Buddha or bodhisattva; the quote of the day (`sutra`) |
| Feature code | `STR` — IDs are `BR-STR-NN`, `FR-STR-NNN`, `NFR-STR-NN` |
| Product | InnerLamp |
| Platforms in scope | Backend, Website, iOS, Android |
| Version | 1.0 |
| Last updated | 2026-10-15 |
| Status | confirmed — 15 open items (Appendix A) |
| Owner / author | Product owner: anh Đức · written by Claude (`ba-writer`, retroactive mode) |
| API contract | `docs/specs/api/openapi.yaml` (owned by the backend; this spec never restates it) |
| Architecture decisions | none needed (read-only public catalogue; audio, offline download and reading progress are out of scope and each needs its own ADR first; see §2.7) |
| Related product specs | `deity` (`docs/product/deity/product.md`, code `DTY`) — the deity page lists the sutras of this library (FR-DTY-007) · `calendar` (`CAL`) — the lunar day page shows the quote of the day · later specs not yet written: `library` (the hub tab), `downloads`, `home`, `settings` |
| Downstream | Backend: `docs/specs/features/sutra/` · Website: `Docs/specs/features/sutra/` · iOS: `Docs/Specs/Sutra/` · Android: `Docs/Specs/Sutra/` |

> See `changelog.md` in this folder for the version history. Platform specs cite this file as
> `docs/product/sutra/product.md@<version>` plus the IDs they implement.

---

## 1. Introduction

### 1.1 Purpose

This document is the single source of product truth for InnerLamp's sutra library on every platform:
what a visitor can browse, search and read, which facts about each text are shown, how the text is
presented for reading, and which quote of the day appears on the home and lunar-day views.

### 1.2 Scope

**In scope:**

- The **library list**: every sutra, verse (kệ) and mantra (chú), page by page, in a fixed editorial
  order, with a **topic chip**, a **duration chip** and a **search** by name.
- The **sutra page**: titles, kind, tradition, collection and citation code, summary, estimated
  duration, source facts (original title, author, Chinese translator), the versions of the text with
  translator, source and licence, the linked Buddhas and bodhisattvas, and a notice while the text is
  unreviewed.
- **Reading** the text: the whole text, a table of contents by section (phẩm), jumping to a section,
  choosing the version, font size and reading background.
- The **sutras related to a Buddha or bodhisattva** (the data behind the deity page).
- The **quote of the day** (câu kinh mỗi ngày) taken from a fixed set of 300 Dhammapada verses.
- Loading the library content (operator task; no user-facing editing).

**Out of scope:**

- The **library hub** (the "Thư viện" tab that gathers sutras, dharma talks, guided meditations and
  deities) and the **downloads** view: separate specs written later (anh Đức, 2026-10-15: "chúng ta
  đang làm phần kinh sách, còn việc client dùng cái gì load về và hiển thị thì tính sau").
- **Audio** and any listening or chant-along playback: no audio exists; the backend says "none" and
  what clients do about it is decided later (BR-STR-12, Appendix A).
- Offline download, bookmarks, favourites, reading progress and chanting history (journal / practice).
- Full-text search inside the text; matching lines between versions.
- Sharing a verse card; the quote of the day is shown, not shared, by this spec.
- Adding, editing or removing texts from a back office.
- Languages other than Vietnamese in the interface of the library (BR-STR-05).

### 1.3 Product overview

InnerLamp is a Buddhist companion app (website, iOS, Android) with a practice library, a lunar
calendar, temples and family features. The sutra library is the public, read-only part of the library
that holds 153 texts: sutras, verses and mantras from the Theravāda, Mahāyāna and Vajrayāna
traditions, each with one or more versions (translations, Sino-Vietnamese chant text, meaning,
verse). A visitor finds a text, reads it, and can follow its links to the Buddhas and bodhisattvas it
concerns. Separately, the app shows one Dhammapada verse a day.

### 1.4 Glossary

| Term | Definition | Not to be confused with |
|---|---|---|
| Kinh (sutra) | A prose discourse; kind "Kinh" | Kệ, Chú |
| Kệ (verse) | A text in verse form: Dhammapada, repentance, vows; kind "Kệ" | The quote of the day (one verse picked from the Dhammapada) |
| Chú (mantra, dhāraṇī) | A text chanted by sound; kind "Chú" | A deity's short mantra on the deity page |
| Truyền thống / tông phái (tradition) | Nam Tông (Theravāda), Bắc Tông (Mahāyāna), Kim Cang Thừa (Vajrayāna) | A temple's sect |
| Bộ kinh (collection) | The group a text belongs to, e.g. "Nhật tụng", "Trường Bộ" | A filter chip |
| Mã trích dẫn (citation code) | The text's catalogue code, e.g. "MN 86", "Taishō T0450"; empty when none | The original title |
| Nhật tụng (daily chanting) | A text that belongs to the daily chanting practice; the chip "Hằng ngày" | Duration |
| Mục đích (purpose) | What the text is used for; one of ten (BR-STR-03); a text has none, one or several | The topic chip as a whole |
| Phẩm (section) | A named part of a text; a long text has many, a short one none | A page of the list |
| Bản (version) | One rendering of a text: dịch (translation), phiên âm / Hán-Việt (chant), dịch nghĩa (meaning), thơ (poem), in one language | An edition of the app |
| Câu kinh mỗi ngày (quote of the day) | One of 300 Dhammapada verses shown for a calendar date (BR-STR-10) | A sutra's summary |
| Chưa duyệt (unreviewed) | Review state of every text today: nobody with religious knowledge has checked the labels, links and some titles | A licence state |

### 1.5 References

- Design prototype: `InnerLamp Prototype.dc.html`, screens "Thư viện" (the list of "Kinh thường
  tụng"), "Chi tiết bài kinh", "Tụng kinh", "Trình phát", "Tìm kiếm", "Không có kết quả", "Chia sẻ
  câu kinh", "Cỡ chữ kinh văn" (settings); copy used: `Mo-ta-tinh-nang.md` in
  `innerlamp-iOS/Docs/innerlamp-project/project/docs/` (the Android copy was not compared). The
  prototype shows a chant-along player with audio, a progress card and a download button; none is
  adopted (BR-STR-12, §1.2).
- Prior documents (rewritten into this spec): backend `docs/specs/features/sutra/` v3.0
  (`requirements.md`, 41 acceptance criteria), `docs/assumptions/sutra/A-001…A-011`,
  `docs/tech-debt/sutra/TD-001…TD-010`, `docs/knowledge/features/sutra.md`; iOS `Docs/Specs/Sutras`
  v1.2, `SutraDetail` v1.4, `Reader` v1.1, `Docs/Assumptions/{Sutras,SutraDetail,Reader}`,
  `Docs/Tech-Debt/{Sutras,SutraDetail,Reader}`, `Docs/Knowledge/features/library.md`. Website and
  Android had no sutra spec; Android `Docs/home_assumptions.md` row 3 and website `.assumptions.md`
  lines 6 and 23 mention the topic in passing.
- Platform folder names: Backend `sutra` · Website `sutra` · iOS `Sutra` (merges the former `Sutras`,
  `SutraDetail` and `Reader` folders) · Android `Sutra`. Code folders keep their own names (website
  `library/sutras`; iOS `Sutras`, `SutraDetail`, `Reader`; Android `sutralist`, `sutra`, `reader`).
  The iOS `Library` and `Downloads` spec folders belong to the later `library` and `downloads` specs
  and are not renamed.

---

## 2. Overall description

### 2.1 Product perspective

Retroactive spec of a feature that runs in part. The backend serves the whole library (list, filters,
search, sutra page, text by page of lines, sutras of a deity). The website shows the list, the page
and the full text from the backend. iOS shows the list and the page from the backend, but its reading
view and the library tiles still use sample texts, so a text can open the wrong content. Android shows
sample data everywhere. The quote of the day exists on iOS and Android from a bundled file and on the
website as four fixed lines.

This version changes, for every platform: one filter set, one search, Vietnamese-first titles and
versions, one reading experience built on the real text, one licence and unreviewed-content notice, no
simulated player, and one quote of the day for a date.

Touches: `deity` (related sutras, linked Buddhas and bodhisattvas), `calendar` (quote on the lunar day
page), `home` (quote block), `settings` (font size), and the later `library` and `downloads` specs.

### 2.2 Product functions

- Browse the library page by page (`FR-STR-001`)
- Filter by topic and by duration (`FR-STR-002`)
- Search by name (`FR-STR-003`)
- The sutra page (`FR-STR-004`)
- Read the text with a table of contents (`FR-STR-005`)
- Choose a version (`FR-STR-006`)
- Font size and reading background (`FR-STR-007`)
- Linked Buddhas and bodhisattvas, and the sutras of a deity (`FR-STR-008`, `FR-STR-009`)
- Quote of the day (`FR-STR-010`)
- Empty, error and loading states (`FR-STR-011`)
- Library content loading by the operator (`FR-STR-012`)

### 2.3 User classes and characteristics

| Actor | Description | Permissions |
|---|---|---|
| Visitor | Anyone, signed in or not (BR-STR-13) | Reads everything in this feature |
| Operator | Runs the content load | Loads the library; not a user-facing role |

### 2.4 Operating environment

| Platform | In scope | Notes |
|---|---|---|
| Backend | yes | Stores the library, orders, filters and searches it, serves pages of texts and pages of lines, links texts to Buddhas and bodhisattvas |
| Website | yes | List with page numbers; sutra page and reading view |
| iOS | yes | List with continuous loading; sutra page; reading view; quote of the day |
| Android | yes | Same as iOS; today sample data only |

Network expectation: online. The apps may keep the last loaded content readable offline; a proper
offline library is out of scope (§1.2). The quote of the day needs no network when the quote file is
bundled (Appendix A item 3).

### 2.5 Constraints

- Religious content must be accurate; an error is a sensitive content defect, not a cosmetic one. All
  153 texts are **unreviewed** today: the daily-chanting flag, the purposes, the links to Buddhas and
  bodhisattvas, the Vietnamese titles, the Chinese and Sanskrit titles and the Taishō numbers were
  assigned by the person who collected the data (an AI), not taken from the sources; source typos are
  kept as they are; the Great Compassion Mantra follows one numbered web version that other versions
  contradict in eight syllables. The text is shown exactly as stored (BR-STR-06).
  > ⚠️ Not confirmed: who reviews the sutra content (labels, links, titles, the Great Compassion Mantra) before public release, and how — blocks public release of the content, not the platform specs.
- Licences of the texts are unresolved: most Vietnamese versions have an unknown licence or only a
  "free to copy" notice; ten Vajrayāna texts are non-commercial (CC BY-NC 4.0); one English translator
  asks that his text not be used for AI training. The app is free, so anh Đức set this aside
  (2026-10-03, backend `A-007`/`TD-003`), and no text is hidden because of its licence (BR-STR-07). It
  must be cleared before any paid feature (the prototype has a paid plan that unlocks "the whole
  library of the Dharma") or any commercial release.
  > ⚠️ Not confirmed: who clears the licences and provenance of the texts before a commercial release or a paid plan — blocks any paid or commercial use of the library, not the platform specs.
- The library is read-only and public (BR-STR-13).

### 2.6 User documentation

N/A — a read-only library needs no help content.

### 2.7 Assumptions and dependencies

- **Assumptions:** the library is public and read-only for users and changes only when the operator
  reloads it. It holds 153 texts, 391 versions and about 64,000 lines; the longest version has 6,967
  lines. Every text has an estimated duration. No architecture decision is needed (anh Đức accepted
  the "no ADR" proposal, 2026-10-15): audio, offline download and reading progress/bookmarks are out
  of scope here and each needs an ADR (file storage, offline sync, per-account data) before it is
  specified.
- **Dependencies:** `deity` (the Buddha and bodhisattva pages that sutras link to and from);
  `calendar` and `home` (where the quote of the day appears); `settings` (the shared reading font
  size); the later `library` spec (the tab and tiles that open this feature).

---

## 3. Specific requirements

### 3.1 External interface requirements

#### 3.1.1 User interfaces

| View | Platforms | Shows |
|---|---|---|
| Sutra list | all clients | Title, "{N} bản kinh" (count of matches), search field, topic chips, duration chips, one card per text, more pages (FR-STR-001…003) |
| Sutra card | all clients | Vietnamese title, kind and tradition, collection / citation code, estimated duration, short summary, daily-chanting mark |
| Sutra page | all clients | Sections of FR-STR-004 and a "Đọc kinh" (read) action |
| Reading view | all clients | The text of one version, table of contents, jump to a section, version choice, font size, background (FR-STR-005…007) |
| Linked Buddhas and bodhisattvas | all clients | Names and images on the sutra page; the sutras of a deity on the deity page (FR-STR-008, FR-STR-009) |
| Quote of the day | all clients | Verse text and attribution on the home and lunar-day views (FR-STR-010) |
| Empty / error state | all clients | FR-STR-011 |

Layout and styling are defined by the prototype, not here.

#### 3.1.2 Hardware interfaces

N/A — no device capability is needed (the screen-awake behaviour of the former simulated reader is
dropped with it, BR-STR-12).

#### 3.1.3 Software interfaces

The backend must: give one page of the library in editorial order, narrowed by topic, duration and
search term, with the number of matches; give one text's page with all its versions and sections; give
the lines of one version page by page; give the sutras of one Buddha or bodhisattva with the deity's
role; say that a text has no audio. Contract: `docs/specs/api/openapi.yaml`, tag Sutras. All of this
exists; the quote of the day has no contract (Appendix B).

#### 3.1.4 Communication interfaces

N/A.

### 3.2 Business rules

**BR-STR-01 — Kind, tradition and their labels**
- **Rule:** Every text has exactly one **kind** and one **tradition**. Kinds are "Kinh" (sutra), "Kệ"
  (verse) and "Chú" (mantra). Traditions are "Nam Tông" (Theravāda), "Bắc Tông" (Mahāyāna) and "Kim
  Cang Thừa" (Vajrayāna). The labels are the same on every platform.
- **Examples:** Heart Sutra → Kinh, Bắc Tông; Dhammapada verses → Kệ, Nam Tông; Great Compassion
  Mantra → Chú, Bắc Tông.
- **Computational:** no
- **Applies to:** Backend, all clients
- > ⚠️ Not confirmed: the exact copy of the labels on the card and chips (kind, tradition, the ten purposes of BR-STR-03, the licence labels of BR-STR-07, the duration chips) — blocks label copy on Website, iOS, Android; the proposed labels in this spec stand until changed.
- **Source:** backend `sutra` v3.0; iOS `Sutras` v1.2 (labels follow the server codes)

**BR-STR-02 — Library order and page size**
- **Rule:** Texts are always listed in their **editorial position** in the library file (first
  first), then by their stable identifier so the order never changes between two loads. The same
  order applies to every page, every filter and every search result; search does not reorder by
  relevance. The list is served 20 texts per page.
- **Examples:** the position-3 text always precedes the position-4 text, with or without a chip; two
  texts at the same position are ordered by identifier.
- **Computational:** yes → `rules/sutra-browse.json`
- **Applies to:** Backend (orders and pages), all clients (show the order)
- > ⚠️ Not confirmed: whether the page size stays 20 or becomes 12 like `temple` and `deity` — blocks the page size on Backend, Website, iOS, Android; 20 is today's value.
- **Source:** backend `sutra` v3.0 AC1; iOS and website both use pages of 20

**BR-STR-03 — Filters: one topic chip, one duration chip**
- **Rule:** The list has two independent filters that combine (both must hold). The **topic chip**
  is one of: "Tất cả" (no filter), "Hằng ngày" (daily chanting texts), one of ten **purposes**, or one
  of the three **traditions**; only one topic chip is active at a time. The ten purposes are "Cầu
  an", "Cầu siêu", "Trì chú", "Sám hối", "Hồi hướng", "Phát nguyện", "Báo hiếu", "Thiền định", "Học
  giáo lý" and "Quy y thọ giới"; a text has none, one or several, and a purpose chip shows every text
  that has it. The **duration chip** is "Tất cả" or "Tối đa 5 / 15 / 30 / 60 phút"; it shows texts
  whose estimated duration is **at most** that many minutes (the limit itself is included). Changing
  a chip returns to the first page.
- **Examples:** "Hằng ngày" + "Tối đa 5 phút" → the daily texts of 5 minutes or less; "Cầu an" shows
  both the Great Compassion Mantra and the Medicine Buddha Sutra if both carry that purpose; a text
  of exactly 30 minutes appears under "Tối đa 30 phút".
- **Computational:** yes → `rules/sutra-browse.json`
- **Applies to:** Backend (filters), all clients (chips)
- **Source:** anh Đức, 2026-10-15 (Q2: "Đồng ý"); iOS `Sutras` v1.2 (chips, `AS-Sutras-002`: filters could not combine — drift); website (tradition and "Hằng ngày" only); Android sample (<15 / 15–30 / >30 minutes — drift). The daily-chanting flag and the purposes are unreviewed (§2.5).

**BR-STR-04 — Search matching**
- **Rule:** A search term matches a text when, after trimming, the term appears anywhere inside **any
  title of the text (in any language), its collection name or its citation code**, ignoring letter
  case and Vietnamese diacritics (đ and d are the same letter). The summary, the translator and the
  lines of the text are never searched. Wildcard characters in the term are literal. The term is 1–100
  characters after trimming; an empty term means "no search"; a longer term is cut by the client to
  100 characters. Search combines with the chips (BR-STR-03) and keeps the library order (BR-STR-02).
- **Examples:** "tam kinh", "TÂM KINH" and "tâm kinh" all match "Bát Nhã Ba La Mật Đa Tâm Kinh";
  "heart sutra" matches through the English title; "t0251" matches through the citation code
  "Taishō T0251"; "kinh tam" does not match "Tâm Kinh"; "tam%kinh" matches nothing.
- **Computational:** yes → `rules/sutra-search-match.json`
- **Applies to:** Backend (search runs on the server), all clients
- > ⚠️ Not confirmed: whether a search term narrows within the active chips (backend: both apply) or clears them on the apps — blocks the search behaviour on Website, iOS, Android; this spec states "both apply".
- **Source:** anh Đức, 2026-10-15 (Q3: search by sutra name on all three clients, Android and website "sẽ làm sau và phải làm"); backend search on titles, collection and reference; iOS `AS-Sutras-001` (titles, collection, reference). Android searched sample descriptions and tags — drift.

**BR-STR-05 — Vietnamese first: titles and the version that opens**
- **Rule:** Whatever language the app is set to, the interface of this feature shows the
  **Vietnamese title** of a text and opens the **Vietnamese version** first, because no other
  language is translated or reviewed yet. When a text has several Vietnamese versions, the one that
  opens first follows this order: translation, then chant, then meaning, then poem; two versions of
  the same variant keep their source order. Another title (original, Pali, Sanskrit, Chinese,
  English) may be shown beneath as the original title (BR-STR-07).
- **Examples:** a text with a Vietnamese translation and an English translation opens the Vietnamese
  translation; a text with a Vietnamese chant version and a Vietnamese meaning version opens the
  chant version; the Heart Sutra has two Vietnamese meaning versions and opens the first in source
  order.
- **Computational:** yes → `rules/sutra-default-version.json`
- **Applies to:** all clients
- > ⚠️ Not confirmed: which version opens for the ten Vajrayāna texts that have no Vietnamese version (English, or the first version in source order) — blocks the opening version of these texts on Website, iOS, Android; not covered by the vector file.
- **Source:** anh Đức, 2026-10-15 (Q4: "chưa có data nên dù khách chọn tiếng Anh vẫn hiển thị tiếng Việt"); the variant order is proposed from the website (translation before chant, meaning, poem); iOS used "the first Vietnamese version in server order" (`AS-Sutras-003`) and the website the visitor's locale — drift

**BR-STR-06 — The text is shown as stored**
- **Rule:** Lines are shown exactly as stored: no spelling correction, no re-wrapping that changes
  words, no translation. Lines are numbered 1..N without gaps inside one version; line numbers mean
  nothing across versions (versions are not aligned line by line). A section (phẩm) starts at a line
  and covers a number of lines; a version without sections has no table of contents.
- **Examples:** the source spelling "Quán Thế Ám" stays as written; line 12 of the Vietnamese version
  is unrelated to line 12 of the English one.
- **Computational:** no
- **Applies to:** Backend, all clients
- **Source:** backend `sutra` v3.0 AC19, `A-001`; `TD-002`

**BR-STR-07 — Sources, licences and the unreviewed notice**
- **Rule:** The sutra page shows, for each version, the **translator, the source (name and link where
  there is one) and the licence**. The licence is one of: "CC0", "CC BY-NC 4.0", "Phạm vi công cộng",
  "Được sao chép tự do", or "unknown"; when it is **unknown the licence line is not shown**, and the
  source's own statement is shown when there is one. A licence never hides a text. The page also shows
  the original title, the author, the Chinese translator and year, and a remark when the text has
  them; empty values are not shown. While a text is **unreviewed** (every text today) the page shows
  one short notice, "Nội dung chưa được người am hiểu duyệt" (the content has not yet been checked by
  someone knowledgeable); it disappears for a text once it is marked reviewed.
- **Examples:** a CC0 Pali version shows "CC0"; a Vietnamese version with an unknown licence shows
  translator and source and no licence line; a Vajrayāna text shows "CC BY-NC 4.0".
- **Computational:** no
- **Applies to:** Backend (gives the facts), all clients
- **Source:** anh Đức, 2026-10-15 (Q8: "Đồng ý"); backend `A-007`/`TD-003`; website hides an unknown licence and shows no notice, iOS shows the licence as stored and a "Lưu ý" notice — drift

**BR-STR-08 — Duration is an editorial estimate**
- **Rule:** Every text carries an estimated duration in minutes to read or chant it whole. It is an
  estimate written by the editor, not a measurement, and is shown as "khoảng N phút" (about N
  minutes). The duration chips of BR-STR-03 compare against this number.
- **Examples:** the Heart Sutra about 3–5 minutes, the Amitabha Sutra about 25; the long collections
  take hours and are above every chip.
- **Computational:** no (the comparison is covered by `rules/sutra-browse.json`)
- **Applies to:** Backend, all clients
- **Source:** backend `A-005`, `A-009`, `TD-007` (estimates unreviewed; known outliers accepted)

**BR-STR-09 — Links between texts and Buddhas / bodhisattvas**
- **Rule:** A text can be linked to Buddhas and bodhisattvas, each with a **role**: the text is *about*
  the deity (subject), the deity *speaks* in it (speaker), or the deity is *mentioned*. On the sutra
  page the linked deities are listed by role in that order (subject, speaker, mentioned), then in the
  order of the deity catalogue. On a deity page the related sutras are listed by the deity's role in
  each text in the same order, then in the library order (BR-STR-02). A deity with no sutra, or a text
  with no deity, shows no such section.
- **Examples:** Phật A Di Đà is the subject of the Amitabha Sutra and only mentioned in another; the
  Amitabha Sutra lists Phật A Di Đà first.
- **Computational:** no
- **Applies to:** Backend (orders), all clients
- > ⚠️ Not confirmed: whether the related-sutra list of a deity is capped and whether a failed load hides the section or offers "Thử lại" — shared with `deity` (A9, A10), blocks the related-sutra section on Website, iOS, Android.
- **Source:** backend `sutra` v3.0 AC7, AC20–AC24; links assigned by the collector, unreviewed (§2.5); `deity` FR-DTY-007

**BR-STR-10 — Quote of the day**
- **Rule:** The quote of the day is chosen from a fixed set of **300 Dhammapada verses**, numbered 1
  to 300 in the quote file. For a calendar date, the quote number is the **day of the year (1 to 366)
  of that Gregorian date, modulo 300, plus 1**. The same date therefore gives the same quote on every
  platform and on every view; "today" is the date on the user's device, and the lunar-day view uses the
  solar date of the day being shown. A quote shows the verse text and its attribution (Kinh Pháp Cú,
  translator, chapter and verse number).
- **Examples:** 1 January → day 1 → quote 2; 27 October (day 300 in a common year) → quote 1; 26 October
  (day 299) → quote 300; 31 December of a leap year (day 366) → quote 67.
- **Computational:** yes → `rules/sutra-daily-quote.json`
- **Applies to:** Website, iOS, Android (Backend does not choose it today)
- **Source:** anh Đức, 2026-10-15 (Q9: take the quote from the 300 verses); iOS `quote(for:)` and Android `quote(date)` use this rule on the lunar-day view; iOS and Android pick a **random** verse on the home view and the website shows four fixed lines chosen by lunar day + month — drift, replaced
- > ⚠️ Not confirmed: where the 300 quotes live — bundled in each app and the website as today (the iOS file is kept out of version control by the ignore rule for JSON files) or served by the backend — blocks the website quote and the single source of the file on every client.

**BR-STR-11 — Reading preferences**
- **Rule:** The reader can pick one of **four font sizes** for the text. The choice is remembered on
  the device and is the **same setting** as "Cỡ chữ kinh văn" in the settings. The reading background
  follows the app's light or dark appearance; there is no separate reading theme.
- **Examples:** choosing the largest size in the reading view makes the settings page show the largest
  size too, and the next text opens at that size.
- **Computational:** no
- **Applies to:** Website, iOS, Android
- **Source:** anh Đức, 2026-10-15 (Q7: "Đồng ý"); iOS offers four sizes and three backgrounds, not remembered, not linked to settings (`TD-Reader-003`); website offers three sizes, remembered and shared with settings, and a dark toggle that is not remembered — drift

**BR-STR-12 — No audio yet**
- **Rule:** No text has audio. The library says so explicitly for every text and the product offers no
  listening, chant-along playback, progress bar or "tụng theo" mode built on a fake timer. What the
  reading view offers instead is decided later.
- **Examples:** the sutra page shows no play button; the reading view shows no elapsed time.
- **Computational:** no
- **Applies to:** Backend (says "no audio"), all clients
- > ⚠️ Not confirmed: what the reading view offers while there is no audio — nothing, or the auto-scroll with a highlighted line at a chosen pace that anh Đức was offered — blocks the reading controls on Website, iOS, Android; anh Đức, 2026-10-15: "BE cứ trả không có, client thấy không có sẽ có hướng xử lý sau".
- **Source:** anh Đức, 2026-10-15 (Q6); backend `TD-005` (no audio); the simulated players on iOS (`TD-Reader-001`), Android and the website are drift until that question is answered

**BR-STR-13 — Public access**
- **Rule:** Everything in this feature is available without an account: a guest and a signed-in
  visitor see the same library, texts and quotes.
- **Examples:** a guest opens the Heart Sutra and reads it.
- **Computational:** no
- **Applies to:** Backend, all clients
- **Source:** anh Đức, 2026-10-15 (Q10: "Đồng ý"); backend `A-007`

### 3.3 Functional requirements

#### Browsing

**FR-STR-001 — Browse the library**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a visitor opens the sutra list THE SYSTEM SHALL show the first page of 20
  texts in library order with the number of texts that match, one card per text; WHEN the visitor
  asks for more THE SYSTEM SHALL show the next page, and the last page ends the list.
- **Business rules:** BR-STR-01, BR-STR-02, BR-STR-05, BR-STR-08, BR-STR-13
- **Verification method:** shared test vectors (order) + automated test on each platform
- **Source:** backend `sutra` v3.0 AC1, AC5; iOS `Sutras` v1.2
- **Acceptance criteria:**
  ```gherkin
  Scenario: First page
    Given the library holds 153 texts
    When a visitor opens the sutra list
    Then 20 cards appear in library order and "153 bản kinh" is shown

  Scenario: Next page
    Given the visitor sees the first page
    When the visitor asks for more
    Then the next 20 cards appear after the first 20, in the same order

  Scenario: Past the last page
    Given the visitor is on the last page
    When the visitor asks for more
    Then no card is added and no error appears
  ```

**FR-STR-002 — Filter by topic and duration**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a visitor chooses a topic chip or a duration chip THE SYSTEM SHALL show the
  first page of the texts that satisfy both chips, with the new count; "Tất cả" removes a chip.
- **Business rules:** BR-STR-02, BR-STR-03, BR-STR-08
- **Verification method:** shared test vectors (`sutra-browse`) + automated test on each platform
- **Source:** anh Đức, 2026-10-15 (Q2)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Topic chip
    When the visitor chooses "Nam Tông"
    Then only Theravāda texts appear, in library order

  Scenario: Both chips combine
    When the visitor chooses "Hằng ngày" and "Tối đa 5 phút"
    Then only daily-chanting texts of 5 minutes or less appear

  Scenario: Nothing matches
    When the chips match no text
    Then the empty state of FR-STR-011 appears with the count 0
  ```

**FR-STR-003 — Search by name**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a visitor types a search term THE SYSTEM SHALL show the first page of the
  texts that match it under BR-STR-04 and the active chips; WHEN the visitor clears the term THE
  SYSTEM SHALL show the list again.
- **Business rules:** BR-STR-04
- **Verification method:** shared test vectors (`sutra-search-match`, `sutra-browse`)
- **Source:** anh Đức, 2026-10-15 (Q3)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Search without diacritics
    When the visitor types "tam kinh"
    Then "Bát Nhã Ba La Mật Đa Tâm Kinh" appears

  Scenario: Search with a chip
    Given the chip "Bắc Tông" is active
    When the visitor types "kinh"
    Then only Mahāyāna texts whose names contain "kinh" appear

  Scenario: Nothing found
    When the visitor types a term that matches no text
    Then the empty state of FR-STR-011 for a search appears
  ```
- > ⚠️ Not confirmed: whether the "no result" view offers "Gửi yêu cầu bản kinh" (request a text) as the prototype does, and what it does — blocks the empty search state on Website, iOS, Android.

#### The sutra page and reading

**FR-STR-004 — The sutra page**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a visitor opens a text THE SYSTEM SHALL show its Vietnamese title (original
  title beneath when there is one), kind, tradition, collection and citation code, summary, estimated
  duration, the facts of BR-STR-07 for each version, the unreviewed notice while it applies, the linked
  Buddhas and bodhisattvas (FR-STR-008) and a "Đọc kinh" action that opens the reading view.
- **Business rules:** BR-STR-01, BR-STR-05, BR-STR-07, BR-STR-08, BR-STR-09, BR-STR-12
- **Verification method:** automated test on each platform + manual QA against the prototype
- **Source:** backend `sutra` v3.0 AC8–AC11, AC36–AC41; iOS `SutraDetail` v1.4
- **Acceptance criteria:**
  ```gherkin
  Scenario: Open a text
    When a visitor opens "Bát Nhã Ba La Mật Đa Tâm Kinh"
    Then its title, kind "Kinh", tradition "Bắc Tông", duration and summary appear with "Đọc kinh"

  Scenario: Unknown licence
    Given a version whose licence is unknown
    When the page is shown
    Then the version shows translator and source and no licence line

  Scenario: Text not found
    When a visitor opens a text that no longer exists
    Then the not-found state of FR-STR-011 appears
  ```

**FR-STR-005 — Read the text**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a visitor chooses "Đọc kinh" THE SYSTEM SHALL open the version of BR-STR-05 and
  show its lines in order, loading further lines as the visitor reads so a very long text never
  waits for its end; WHEN the version has sections THE SYSTEM SHALL offer a table of contents and,
  when a section is chosen, show that section from its first line.
- **Business rules:** BR-STR-05, BR-STR-06, BR-STR-12
- **Verification method:** automated test on each platform + manual QA with the longest text
- **Source:** backend `sutra` v3.0 AC12–AC19; website reading view; anh Đức, 2026-10-15 (Q5: the backend gives the sections for the table of contents)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Read a short text
    When a visitor opens the Heart Sutra and chooses "Đọc kinh"
    Then the Vietnamese version appears from its first line

  Scenario: Jump to a section
    Given a text of 28 sections
    When the visitor chooses section 12 in the table of contents
    Then the text continues from the first line of section 12

  Scenario: Very long text
    When a visitor opens the longest text (6,967 lines)
    Then the first lines appear without loading the whole text
  ```
- > ⚠️ Not confirmed: whether the apps show one section at a time like the website or scroll the whole text continuously — blocks the reading layout on iOS, Android; the table of contents and the jump are required either way.

**FR-STR-006 — Choose a version**
- **Priority:** Should
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a text has more than one version THE SYSTEM SHALL let the visitor switch
  between them in the reading view, keeping the table of contents of the chosen version; WHEN it has
  only one THE SYSTEM SHALL offer no switch.
- **Business rules:** BR-STR-05, BR-STR-06
- **Verification method:** automated test on each platform
- **Source:** website (translation selector); backend `sutra` v3.0 AC9, AC11
- **Acceptance criteria:**
  ```gherkin
  Scenario: Switch version
    Given a text with a Vietnamese and an English version
    When the visitor chooses the English version
    Then the reading view shows the English lines from their first line

  Scenario: Single version
    Given a text with one version
    Then no version switch appears
  ```
- > ⚠️ Not confirmed: whether iOS and Android offer the version switch in this release and how a version is named to a visitor (e.g. "Bản dịch", "Phiên âm Hán-Việt", "Dịch nghĩa", "Thơ", plus the language) — blocks the version switch on iOS, Android and its labels everywhere.

**FR-STR-007 — Font size and reading background**
- **Priority:** Should
- **Platforms:** Website, iOS, Android
- **Description:** WHEN a visitor changes the font size in the reading view THE SYSTEM SHALL apply it at
  once, remember it on the device and apply it to every later text and to the "Cỡ chữ kinh văn"
  setting; the reading background follows the app's appearance.
- **Business rules:** BR-STR-11
- **Verification method:** automated test on each platform
- **Source:** anh Đức, 2026-10-15 (Q7)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Change and remember
    When the visitor picks the largest of the four sizes
    Then the text grows at once and the next text opens at that size

  Scenario: Shared with settings
    Given the visitor chose the second size in the reading view
    When the visitor opens "Cỡ chữ kinh văn" in the settings
    Then the second size is selected
  ```

#### Linked Buddhas and bodhisattvas

**FR-STR-008 — Buddhas and bodhisattvas on the sutra page**
- **Priority:** Should
- **Platforms:** Backend, Website, iOS, Android
- **Description:** WHEN a text is linked to Buddhas or bodhisattvas THE SYSTEM SHALL list them on the
  sutra page by role (BR-STR-09), and choosing one SHALL open that deity's page; WHEN none is linked
  the section is not shown.
- **Business rules:** BR-STR-09
- **Verification method:** automated test on each platform
- **Source:** backend `sutra` v3.0 AC7; iOS `AS-SutraDetail-002` (plain text rows today)
- **Acceptance criteria:**
  ```gherkin
  Scenario: Open a linked deity
    Given the Amitabha Sutra is linked to Phật A Di Đà
    When the visitor chooses Phật A Di Đà on the sutra page
    Then the page of Phật A Di Đà opens

  Scenario: No linked deity
    Given a text with no linked deity
    Then no deity section appears
  ```

**FR-STR-009 — Sutras of a Buddha or bodhisattva**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android (shown by `deity` FR-DTY-007)
- **Description:** WHEN the deity page asks for the sutras of a deity THE SYSTEM SHALL give them with
  the deity's role in each, ordered by BR-STR-09, page by page; choosing one SHALL open that text's
  sutra page.
- **Business rules:** BR-STR-09
- **Verification method:** automated test on the backend; on clients through `deity`
- **Source:** backend `sutra` v3.0 AC20–AC24
- **Acceptance criteria:**
  ```gherkin
  Scenario: Related sutras by role
    Given Đức Phật Thích Ca is the speaker of many sutras and the subject of one
    When his sutras are listed
    Then the sutra where he is the subject comes before the sutras where he speaks

  Scenario: No sutra
    Given a deity linked to no sutra
    Then the list is empty and no error appears
  ```

#### Quote of the day

**FR-STR-010 — Quote of the day**
- **Priority:** Must
- **Platforms:** Website, iOS, Android
- **Description:** WHEN a view that shows the quote of the day is displayed (the home view, and the
  lunar-day view for the chosen day) THE SYSTEM SHALL show the quote of BR-STR-10 for that date, the
  same one however often the view is opened that day.
- **Business rules:** BR-STR-10
- **Verification method:** shared test vectors (`sutra-daily-quote`) on every platform
- **Source:** anh Đức, 2026-10-15 (Q9); `calendar` and `home`
- **Acceptance criteria:**
  ```gherkin
  Scenario: Same quote all day
    When the visitor opens the home view twice on 1 January
    Then both times show quote 2

  Scenario: Same quote on every platform
    Given the same date on the website, iOS and Android
    Then all three show the same verse

  Scenario: Lunar day view
    When the visitor chooses 27 October on the lunar-day view
    Then quote 1 is shown
  ```
- > ⚠️ Not confirmed: what the quote block shows when the quote file cannot be read — a built-in fallback line as the apps have today, or nothing — blocks the quote block on Website, iOS, Android.

#### States

**FR-STR-011 — Empty, error and loading states**
- **Priority:** Must
- **Platforms:** Website, iOS, Android
- **Description:** WHEN the list or a text cannot be loaded THE SYSTEM SHALL show an error state with
  "Thử lại" (retry), never sample content; WHEN the library has no text, or the chips or the search
  match none, THE SYSTEM SHALL show an empty state that says which of these it is; WHILE a page loads
  THE SYSTEM SHALL show a loading state and not an empty one.
- **Business rules:** BR-STR-02, BR-STR-03, BR-STR-04
- **Verification method:** automated test on each platform + manual QA
- **Source:** same decision as `deity` FR-DTY-010 (prototype empty state with retry, no static fallback); iOS and Android show sample texts today
- **Acceptance criteria:**
  ```gherkin
  Scenario: Backend unreachable
    Given the backend cannot be reached and nothing was loaded before
    When a visitor opens the sutra list
    Then an error state with "Thử lại" appears and no sample text

  Scenario: Retry succeeds
    Given the list shows its error state
    When the backend is reachable and the visitor chooses "Thử lại"
    Then the first page appears

  Scenario: Filter with no result
    Given the library has texts
    When the chips match none
    Then the empty state says no text matches the filters, not that the connection failed
  ```
- > ⚠️ Not confirmed: the copy of the error, empty, search-empty and not-found states — blocks the state copy on Website, iOS, Android.

#### Operator

**FR-STR-012 — Load library content**
- **Priority:** Must
- **Platforms:** Backend
- **Description:** WHEN the operator loads the library file into an empty library THE SYSTEM SHALL
  check the whole file first and add every text with its versions, lines and deity links all or
  nothing: an invalid file (a missing or duplicate identifier, an unknown kind, tradition, variant,
  role, purpose or licence code, a missing Vietnamese title, a version whose lines do not run 1..N, a
  deity that does not exist) changes nothing; WHEN the library already holds a text THE SYSTEM SHALL
  change nothing. The position in the file becomes the library order (BR-STR-02).
- **Business rules:** BR-STR-02, BR-STR-06
- **Verification method:** automated test on the backend (end-to-end against a real database)
- **Source:** backend `sutra` v3.0 AC29–AC35
- **Acceptance criteria:**
  ```gherkin
  Scenario: First load
    Given an empty library and a valid file of 153 texts
    When the operator loads it
    Then 153 texts exist in the file's order

  Scenario: Invalid file
    When the operator loads a file with a text of an unknown kind
    Then the load is refused and the library stays empty

  Scenario: Load again
    Given the library holds texts
    When the operator loads the file again
    Then nothing changes
  ```
- > ⚠️ Not confirmed: how a text is corrected or added after the first load (today only by hand-written database commands; no reload, no back office) and who runs the load in staging and production from which copy of the file (about 11 MB, kept outside version control by the ignore rule for JSON files) — blocks content fixes and production content, not the platform specs.

### 3.4 Non-functional requirements

- **NFR-STR-01 — Long texts read page by page:** no platform loads a whole long text before showing
  the first lines; the first lines of the longest text appear within 2 s on a 4G connection. Platforms:
  all.
- **NFR-STR-02 — Responsive list:** a chip change or a search shows its first page within 2 s on a 4G
  connection. Platforms: all.
- **NFR-STR-03 — Library size:** the list, search and reading stay within NFR-STR-01 and NFR-STR-02 at
  the current 153 texts and 64,000 lines and are not designed for a library several times larger
  without review. Platforms: Backend.

### 3.5 Data requirements

| Data item | Meaning | Owned by | Retention / deletion | Private or public |
|---|---|---|---|---|
| Text | Titles in several languages, kind, tradition, collection, citation code, summary, daily-chanting flag, purposes, estimated duration, original title, author, Chinese translator and year, remark, review state | InnerLamp library on the server | Kept; changed by the operator | Public |
| Version | One rendering of a text: language, variant, translator, source, licence and its explanation, lines, sections | Server | Kept with the text | Public |
| Stable text identifier | Never changes for a text; deity pages and shared links refer to it | Server | As long as the text exists | Public |
| Text ↔ deity link | Which Buddhas and bodhisattvas relate to the text, and in what role | Server | Kept with the text | Public |
| Quote | Verse text, translator, chapter, verse number; 300 in the quote file | The quote file (Appendix A item 3) | Kept | Public |
| Reading preferences | Chosen font size | This device only | Until the app data is cleared | Private to the device |

No personal data is involved except the device-only font size.

### 3.6 Platform variance

| ID | Platforms | Difference | Reason |
|---|---|---|---|
| FR-STR-001 | Website vs apps | Website: page numbers (shareable page links); a page number past the end leads to the last page. Apps: the next page loads at the end of the list | Web navigation and SEO vs mobile scrolling |
| FR-STR-005 | Website | Shows the text one section at a time with previous / next; apps: see the open question in FR-STR-005 | Reading on a wide screen |
| FR-STR-007 | Website | Offers the font size on the page and reads it from the same device setting as the settings page | No app settings screen |
| BR-STR-10 | Website | The quote file must be available to the website (Appendix A item 3) | Website had four fixed lines |
| §2.4 | iOS, Android | May keep the last loaded pages and texts readable offline; the website is online only | App caching |

### 3.7 Software system attributes

- **Reliability:** identical inputs give identical pages on every load (BR-STR-02 stable order); the
  same date gives the same quote everywhere (BR-STR-10).
- **Availability:** backend unreachable → error state with retry (FR-STR-011); apps may show cached
  content; the quote of the day does not depend on the backend if its file is bundled.
- **Security & privacy:** all data is public; no account, no personal data. Source links lead to
  third-party sites.
- **Maintainability / portability:** N/A.

---

## 4. Verification & traceability

### 4.1 Verification approach

Computational rules are verified by running `rules/*.json`: order, filters and search together
(`sutra-browse`), search matching (`sutra-search-match`) and the opening version (`sutra-default-version`)
on the backend and on every client that filters, searches or picks a version on the device; the quote
number (`sutra-daily-quote`) on every platform that shows the quote. Functional requirements are
verified by each platform's automated tests and manual QA against the prototype, and the backend by an
end-to-end test against a real database.

### 4.2 Traceability matrix

| ID | Backend | Website | iOS | Android | Note |
|---|---|---|---|---|---|
| BR-STR-01 | done | done | done | drift | Android sample chips (<15 / 15–30 / >30, own tags) |
| BR-STR-02 | done | done | done | planned | Android sample data |
| BR-STR-03 | done | drift | drift | drift | Website lacks purposes and duration; iOS chips do not combine; Android client-side sample filters |
| BR-STR-04 | done | planned | done | planned | Backend spec v3.0 does not list it yet; Android searches sample descriptions and tags |
| BR-STR-05 | n/a | drift | drift | planned | Website follows the visitor's locale; iOS uses the first Vietnamese version in server order |
| BR-STR-06 | done | done | planned | planned | iOS reader and Android reader use sample text |
| BR-STR-07 | done | drift | drift | planned | Website hides unknown licence and has no notice; iOS shows the licence as stored |
| BR-STR-08 | done | done | done | drift | Android sample durations |
| BR-STR-09 | done | planned | drift | planned | iOS shows plain rows, website does not list the deities |
| BR-STR-10 | n/a | drift | drift | drift | Home random on iOS and Android, four fixed lines on the website; the lunar-day view already uses the rule on iOS and Android |
| BR-STR-11 | n/a | drift | drift | planned | Website: three sizes; iOS: four sizes, three backgrounds, not remembered |
| BR-STR-12 | done | drift | drift | drift | Simulated players on all three clients |
| BR-STR-13 | done | done | done | done | |
| FR-STR-001 | done | done | done | planned | 3 scenarios; Android sample list |
| FR-STR-002 | done | drift | drift | drift | 3 scenarios |
| FR-STR-003 | done | planned | done | planned | 3 scenarios; the website and Android must add it |
| FR-STR-004 | done | done | done | drift | 3 scenarios; Android shows one fixed text |
| FR-STR-005 | done | done | drift | drift | 3 scenarios; iOS reader uses sample texts and can open the wrong text; Android shows one verse |
| FR-STR-006 | done | done | planned | planned | 2 scenarios |
| FR-STR-007 | n/a | drift | drift | planned | 2 scenarios |
| FR-STR-008 | done | planned | drift | planned | 2 scenarios |
| FR-STR-009 | done | planned | planned | planned | 2 scenarios; through `deity` |
| FR-STR-010 | n/a | drift | drift | drift | 3 scenarios |
| FR-STR-011 | n/a | drift | drift | drift | 3 scenarios; iOS and Android show sample content |
| FR-STR-012 | done | n/a | n/a | n/a | 3 scenarios |
| NFR-STR-01 | done | drift | drift | planned | Website loads up to 20 pages of 500 lines before showing a text |
| NFR-STR-02 | planned | planned | planned | planned | |
| NFR-STR-03 | planned | n/a | n/a | n/a | |

### 4.3 Test vectors

| File | Rule | Cases |
|---|---|---|
| `rules/sutra-browse.json` | BR-STR-02, BR-STR-03, BR-STR-04 | 13 |
| `rules/sutra-search-match.json` | BR-STR-04 | 13 |
| `rules/sutra-default-version.json` | BR-STR-05 | 6 |
| `rules/sutra-daily-quote.json` | BR-STR-10 | 8 |

---

## 5. Appendices

### Appendix A — Open questions

#### Open (unconfirmed)

1. > ⚠️ Not confirmed: who reviews the sutra content (labels, links, titles, the Great Compassion Mantra) before public release, and how — blocks public release of the content, not the platform specs.
2. > ⚠️ Not confirmed: who clears the licences and provenance of the texts before a commercial release or a paid plan — blocks any paid or commercial use of the library, not the platform specs.
3. > ⚠️ Not confirmed: where the 300 quotes live — bundled in each app and the website as today (the iOS file is kept out of version control by the ignore rule for JSON files) or served by the backend — blocks the website quote and the single source of the file on every client.
4. > ⚠️ Not confirmed: what the reading view offers while there is no audio — nothing, or the auto-scroll with a highlighted line at a chosen pace that anh Đức was offered — blocks the reading controls on Website, iOS, Android.
5. > ⚠️ Not confirmed: which version opens for the ten Vajrayāna texts that have no Vietnamese version (English, or the first version in source order) — blocks the opening version of these texts on Website, iOS, Android.
6. > ⚠️ Not confirmed: whether iOS and Android offer the version switch in this release and how a version is named to a visitor — blocks the version switch on iOS, Android and its labels everywhere.
7. > ⚠️ Not confirmed: whether the apps show one section at a time like the website or scroll the whole text continuously — blocks the reading layout on iOS, Android.
8. > ⚠️ Not confirmed: whether a search term narrows within the active chips (backend: both apply) or clears them on the apps — blocks the search behaviour on Website, iOS, Android.
9. > ⚠️ Not confirmed: whether the "no result" view offers "Gửi yêu cầu bản kinh" (request a text) as the prototype does, and what it does — blocks the empty search state on Website, iOS, Android.
10. > ⚠️ Not confirmed: the copy of the error, empty, search-empty and not-found states — blocks the state copy on Website, iOS, Android.
11. > ⚠️ Not confirmed: the exact copy of the labels on the card and chips (kind, tradition, purposes, licence labels, duration chips) — blocks label copy on Website, iOS, Android; the proposed labels stand until changed.
12. > ⚠️ Not confirmed: whether the page size stays 20 or becomes 12 like `temple` and `deity` — blocks the page size on Backend, Website, iOS, Android; 20 is today's value.
13. > ⚠️ Not confirmed: whether the related-sutra list of a deity is capped and whether a failed load hides the section or offers "Thử lại" — shared with `deity` (A9, A10), blocks the related-sutra section on Website, iOS, Android.
14. > ⚠️ Not confirmed: what the quote block shows when the quote file cannot be read — a built-in fallback line or nothing — blocks the quote block on Website, iOS, Android.
15. > ⚠️ Not confirmed: how a text is corrected or added after the first load, and who runs the load in staging and production from which copy of the file — blocks content fixes and production content, not the platform specs.

#### Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| Is an ADR needed? | No — read-only public catalogue; audio, offline and reading progress each need an ADR when they are specified | 2026-10-15 |
| Scope of the spec | Browse, sutra page, reading, deity links, quote of the day. The library hub and downloads are separate later specs | 2026-10-15 |
| Filters | One topic chip (Hằng ngày / ten purposes / three traditions) + one duration chip (5 / 15 / 30 / 60), combinable | 2026-10-15 |
| Search | Server search by name on all three clients; Android and website to be built | 2026-10-15 |
| Language | Vietnamese titles and versions for now, whatever the app language | 2026-10-15 |
| Reading preferences | Four font sizes, remembered, shared with the settings; background follows the app appearance | 2026-10-15 |
| Licence and unreviewed notice | Show translator, source and licence (hidden when unknown) and the unreviewed notice | 2026-10-15 |
| Quote of the day | From the 300 Dhammapada verses, one per date, the same on every platform | 2026-10-15 |
| Audio | None; the backend says so and clients decide later | 2026-10-15 |
| Access | Public, no account | 2026-10-15 |
| Table of contents | The backend gives the sections of a version for the table of contents (anh Đức asked, 2026-10-15; the answer was yes) | 2026-10-15 |

### Appendix B — Contract handoff (backend)

1. The backend sutra spec `docs/specs/features/sutra/` v3.0 is behind the code: search by name and the
   "no audio" value exist in the contract and the code but the spec still lists them as out of
   scope; `spec-writer` brings it in line when it cites this spec, including the 100-character limit of
   the search term (BR-STR-04) if the backend does not enforce it yet.
2. The duration chips (BR-STR-03) use "at most N minutes" for 5, 15, 30 and 60; the existing filter by
   maximum minutes already covers it.
3. Every text carries an estimated duration (BR-STR-08), including the ten Vajrayāna texts that had
   none; the contract should say the value is never empty.
4. The quote of the day has no contract. If the backend serves the 300 quotes (Appendix A item 3) it
   needs a list of quotes or one quote per date, with verse text, translator, chapter and verse number.
5. The library file (about 11 MB of JSON, kept outside version control by the ignore rule for JSON
   files) and the quote file are inputs of the content load in every environment (Appendix A items 3
   and 15).
6. The related-sutra listing for a deity is the source of FR-STR-009 and `deity` FR-DTY-007; the order
   and a possible cap are open (BR-STR-09, Appendix A item 13).
7. The sutra page needs the sections of each version (name, first line, number of lines) so every
   client can build the table of contents (FR-STR-005); this exists.
