# Product Spec — Dharma talks (pháp thoại)

| Field | Value |
|---|---|
| Document type | Product spec — Software Requirements Specification, IEEE 29148-2018 structure |
| Feature | A public library of dharma talks — audio, video or text — with a detail page (`dharmatalk`) |
| Feature code | `DHT` — IDs are `BR-DHT-NN`, `FR-DHT-NNN`, `NFR-DHT-NN` |
| Product | InnerLamp |
| Platforms in scope | Backend, Website, iOS, Android, Flutter |
| Version | 0.4 (draft) |
| Last updated | 2026-11-15 |
| Status | draft — written retroactively from an implemented feature whose product decisions are still open; 13 questions in Appendix A. v0.4 adds the cross-client audio delivery contract. |
| Owner / author | Product owner: anh Đức · written by Claude (`ba-writer`, retroactive from `docs/specs/features/dharmatalk/requirements.md` v2.0) |
| API contract | `docs/specs/api/openapi.yaml` @ v3.0.0 (owned by the backend; this spec never restates it) |
| Architecture decisions | None required; the catalogue is public and read-only, with seeded content. |
| Related product specs | `article` (the same public read-only catalogue shape), `sutra` (canonical texts, a separate library) |
| Downstream | Backend: `docs/specs/features/dharmatalk/` · Website: `Docs/specs/features/dharmatalk/` · iOS: `Docs/Specs/DharmaTalk/` · Android: `Docs/Specs/DharmaTalk/` · Flutter: `docs/specs/features/dharmatalk/` |

> See `changelog.md` in this folder for the version history. Platform specs cite this file as
> `docs/product/dharmatalk/product.md@<version>` plus the IDs they implement.

---

## 1. Introduction

### 1.1 Purpose

This document is the single source of product truth for the dharma-talk library: what a talk is, how it
is listed and opened, what its three media forms mean, and how its content reaches the product. It is
written as a **draft** because several product decisions for this feature were never confirmed.

### 1.2 Scope

**In scope:**
- A public, read-only catalogue of dharma talks: a paginated list and a detail page.
- Three media forms: an embedded YouTube video, an audio file, or text only.
- Playable audio for every client: an absolute public URL, byte-range and HEAD support, CORS, and a
  graceful player-unavailable state when the file is missing.
- The talk's speaker, summary, key points, quote, season, publication date, duration, tags, chapters,
  a follow-up meditation card, source and licence.
- Seeding the catalogue from a bundled data file on first run.

**Out of scope:**
- Creating, editing or deleting talks through an API or backoffice; content changes are manual today.
- Extracting audio from a YouTube video; YouTube is embedded only.
- Re-hosting licensed audio is **in** scope: local disk in dev, object storage or a CDN in production;
  only YouTube audio extraction is excluded.
- Search, filtering by category/tag/season, bookmarks, "already listened" state.
- A "featured/top" route.
- Sign-in — both routes are public like the article catalogue.
- Counting listens, the follow-up meditation mapping, chapter data, seasons and licence clearance beyond
  the questions in Appendix A.

### 1.3 Product overview

InnerLamp is a Buddhist-practice companion. The dharma-talk library collects lectures by named teachers
so a practitioner can listen, watch or read. It differs from the sutra library: a sutra is a canonical
text, while a dharma talk is one teacher's lecture, with a speaker, a duration and a media form.

### 1.4 Glossary

| Term | Definition | Not to be confused with |
|---|---|---|
| Dharma talk (pháp thoại) | A lecture by a named teacher, in one of three media forms. | A sutra (canonical text) |
| Media type | `youtube`, `audio` or `text`; decides the detail variant. | A category |
| Speaker | The teacher who gave the talk. | The source channel |
| Chapter (Các phần) | A time marker with a title inside a talk. | A section of a sutra |
| Season | An occasion such as Vu Lan or Tết. | A publication date |
| Follow-up meditation | A suggested sitting practice after the talk. | The meditation library |
| Needs review | A content flag that hides an unverified talk from users. | Draft |
| Source | Where the talk came from, with its URL and licence. | The speaker |

### 1.5 References

- `docs/specs/features/dharmatalk/requirements.md` v2.4 — the backend platform source for this spec.
- `docs/product/article/product.md` — the same public catalogue shape.
- Backend platform spec: `docs/specs/features/dharmatalk/` v2.4.
- API contract: `docs/specs/api/openapi.yaml` @ v3.0.0.

---

## 2. Overall description

### 2.1 Product perspective

A public content catalogue in the same shape as articles: a paginated list and a detail page, seeded
once from a bundled file. No user account is involved and nothing is written at request time.

### 2.2 Product functions

- Browse the talk list page by page (`FR-DHT-001`).
- Open one talk (`FR-DHT-002`).
- Present the right media form (`FR-DHT-003`).
- Give every client a playable audio URL (`FR-DHT-008`).
- Jump to a chapter (`FR-DHT-004`).
- Keep every missing field in a predictable shape (`FR-DHT-005`).
- Seed the catalogue on first run (`FR-DHT-006`).
- Hide talks flagged for review (`FR-DHT-007`).

### 2.3 User classes and characteristics

| Actor | Description | Permissions |
|---|---|---|
| App or web visitor | Anyone, signed in or not. | Read the public catalogue and a talk. |
| Content operator | A team member who prepares the bundled data and runs the seed. | Writes data outside the app. |
| Client developer | Builds the list and detail screens. | Consumes the fixed field shapes. |

### 2.4 Operating environment

| Platform | In scope | Notes |
|---|---|---|
| Backend | yes | Serves the catalogue and holds the content rows. |
| Website | yes | Talk list and detail. |
| iOS | yes | Talk list and detail; media variants. |
| Android | later | Same data and rules once the library screen is built. |
| Flutter | later | Web and mobile share the same contract; the library screen is not built yet. |

Network expectation: online; an embedded YouTube video also needs YouTube to be reachable.

### 2.5 Constraints

- The catalogue is read-only and public; there is no write route.
- One media type per talk; the three types have different field shapes.
- YouTube is embedded by id; no audio is extracted from a video.
- Licensed audio files are self-hosted: local disk in dev, object storage/CDN in production. The API
  always returns an absolute URL, so a client never builds a host itself.
- Content arrives through a first-run seed, never at service start.
- The seed is a fixed bundled file; changing it later needs a decision (Appendix A).
- Licence for every talk is `unknown` or a source copyright line today; this is a release blocker.

### 2.6 User documentation

N/A — the screens are self-explanatory.

### 2.7 Assumptions and dependencies

- **Assumptions:** a talk belongs to exactly one media type; the list is ordered by an editorial sort
  order; a talk's id is a stable slug.
- **Dependencies:** the bundled data file; the database; local audio files in dev or object storage/CDN
  in production; the client's media player and YouTube embed.

---

## 3. Specific requirements

### 3.1 External interface requirements

#### 3.1.1 User interfaces

A talk list with a card per talk (title, speaker, duration, media type, season) and paging; a detail page
with the media player (YouTube embed, audio player or text), summary, key points, quote, chapters, a
follow-up meditation card and the source at the end. The client hides a block when its data is missing.

#### 3.1.2 Hardware interfaces

The audio variant uses the device's audio output; the video variant uses the embedded player. No custom
hardware.

#### 3.1.3 Software interfaces

Two public read capabilities (a paginated list and a detail) plus a public media-file capability for
audio. Exact shapes and headers are in the contract and in `docs/specs/features/dharmatalk/design.md`.

#### 3.1.4 Communication interfaces

HTTPS, using the shared response and error conventions. A production audio URL is HTTPS; only a dev
server on the LAN may return HTTP, and the platform must allow that cleartext locally.

### 3.2 Business rules

**BR-DHT-01 — The catalogue is public and read-only**
- **Rule:** Anyone may read the list and a detail without signing in; no write route exists.
- **Examples:** a signed-out visitor browses the list; no client can create or edit a talk.
- **Computational:** no.
- **Applies to:** Backend, Website, iOS, Android, Flutter.
- **Source:** requirements v2.4.

**BR-DHT-02 — A talk flagged for review is invisible to users**
- **Rule:** A talk with the review flag set is stored but never returned by any public route; it counts as
  a stored row for seeding.
- **Examples:** 19 of 67 seeded talks are hidden, leaving 48 visible; a detail request for a hidden talk
  returns not found.
- **Computational:** no.
- **Applies to:** Backend.
- **Source:** requirements AC 4, 7.

**BR-DHT-03 — Exactly one media type per talk, with a fixed shape**
- **Rule:** A talk is `youtube`, `audio` or `text`. A YouTube talk carries an 11-character video id and no
  audio URL; an audio talk carries an absolute, public http(s) audio URL and no video id; a text talk
  carries neither.
- **Examples:** a YouTube talk never returns a full YouTube URL; an audio talk has no video id; a text
  talk has both empty.
- **Computational:** no.
- **Applies to:** Backend, Website, iOS, Android, Flutter.
- **Source:** requirements AC 8, 20.

**BR-DHT-04 — The list is ordered, paged and never errors past the end**
- **Rule:** The list is ordered by editorial sort order then id; the page size is configuration (10 in
  development); a page past the end returns an empty list with the same totals; an invalid page is a
  validation error.
- **Examples:** page 1 shows 10 of 36; page 4 shows the remaining 6; page 99 is empty, not an error;
  page 0 is refused.
- **Computational:** yes — paging arithmetic; covered by backend tests.
- **Applies to:** Backend, Website, iOS, Android, Flutter.
- **Source:** requirements AC 1–3, 5.

**BR-DHT-05 — Chapter markers are ordered and optional**
- **Rule:** Chapters are returned in ascending start time, each with a non-negative start and a title;
  no chapters returns an empty list, never null; a text talk always has no chapters.
- **Examples:** three chapters at 0, 600 and 1200 seconds; a talk with none hides the "Các phần" block.
- **Computational:** yes — ordering; covered by backend tests.
- **Applies to:** Backend, Website, iOS, Android, Flutter.
- **Source:** requirements AC 9–10.

**BR-DHT-06 — Missing data has a fixed shape**
- **Rule:** A missing date is null in `YYYY-MM-DD` form when present; a missing duration is null, never
  zero; a missing season or follow-up meditation is null; missing tags or key points are empty lists.
- **Examples:** a talk without a date returns null; a talk without a duration returns null, not 0.
- **Computational:** no.
- **Applies to:** Backend, Website, iOS, Android, Flutter.
- **Source:** requirements AC 11–14.

**BR-DHT-07 — The seed runs only into an empty catalogue and never at service start**
- **Rule:** The seed command inserts the bundled talks in one transaction only when the catalogue has no talk,
  with the file position as the sort order; running it again changes nothing; the service never reads
  the seed or writes a talk at startup; an existing id is left untouched.
- **Examples:** a fresh database gets 67 talks; a second run logs "already seeded"; restarting the
  service writes nothing.
- **Computational:** no.
- **Applies to:** Backend.
- **Source:** requirements AC 16–18.

**BR-DHT-08 — An invalid seed writes nothing**
- **Rule:** If any talk in the file is invalid — no talk, empty or duplicate id, empty title or speaker,
  unknown media type, a YouTube talk without a valid 11-character id, an audio talk without an http(s)
  URL or a text talk carrying either, a non-positive duration, a malformed date, or a bad chapter — the
  seed exits non-zero and leaves the catalogue unchanged.
- **Examples:** a duplicate id aborts the whole load; a missing speaker aborts it.
- **Computational:** no.
- **Applies to:** Backend.
- **Source:** requirements AC 19.

**BR-DHT-09 — Every talk names its source and licence**
- **Rule:** A talk returns the source name, source URL and licence, and the client shows the source at
  the end. A talk whose licence is not cleared must not be shown to users.
- **Examples:** the source card appears under the talk; an `unknown` licence blocks publication.
- **Computational:** no.
- **Applies to:** Backend, Website, iOS.
- **Source:** requirements AC 6; Appendix A (licence).

### 3.3 Functional requirements

**FR-DHT-001 — Browse the talk list**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS
- **Description:** WHEN a visitor asks for the talk list THE SYSTEM SHALL return page 1 by default,
  ordered by editorial sort order then id, with the page size, total visible talks and total pages; a
  page past the end returns an empty list with the same totals.
- **Business rules:** BR-DHT-01, BR-DHT-02, BR-DHT-04
- **Verification method:** unit, repository, end-to-end
- **Source:** requirements US-DT-01
- **Acceptance criteria:**
  ```gherkin
  Scenario: First page
    Given 48 visible talks and a page size of 10
    When a visitor opens the list without choosing a page
    Then the first 10 talks are shown in editorial order
    And the response reports 48 total and 5 pages

  Scenario: Page past the end
    Given 48 visible talks and a page size of 10
    When a visitor asks for page 99
    Then the list is empty
    And the response still reports 48 total and 5 pages

  Scenario: Invalid page
    When a visitor asks for page 0 or a non-numeric page
    Then the request is refused and names the page
  ```

**FR-DHT-002 — Open one talk**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android, Flutter
- **Description:** WHEN a visitor opens a visible talk THE SYSTEM SHALL return the summary plus key
  points, quote, language, media fields, chapters, follow-up meditation, source and licence; an unknown
  or hidden id returns not found.
- **Business rules:** BR-DHT-02, BR-DHT-09
- **Verification method:** unit, end-to-end
- **Source:** requirements US-DT-02
- **Acceptance criteria:**
  ```gherkin
  Scenario: Open a visible talk
    Given a visible talk
    When a visitor opens it
    Then the detail shows the summary, key points, quote, media, chapters, source and licence

  Scenario: Hidden or unknown talk
    Given a talk flagged for review, or an unknown id
    When a visitor opens it
    Then the response is "not found" and no detail is shown
  ```

**FR-DHT-003 — Present the right media form**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android, Flutter
- **Description:** WHEN a talk is returned THE SYSTEM SHALL give the client exactly the fields its media
  type needs: a YouTube id for video, an absolute public audio URL for audio, neither for text; never a
  full YouTube URL and never a host the client must guess.
- **Business rules:** BR-DHT-03
- **Verification method:** unit, end-to-end
- **Source:** requirements AC 8
- **Acceptance criteria:**
  ```gherkin
  Scenario: YouTube talk
    Given a talk whose media type is youtube
    When a visitor opens it
    Then it carries an 11-character video id
    And it has no audio URL

  Scenario: Audio talk
    Given a talk whose media type is audio
    When a visitor opens it
    Then it carries an absolute http(s) audio URL
    And it has no video id

  Scenario: Text talk
    Given a talk whose media type is text
    When a visitor opens it
    Then it has neither a video id nor an audio URL
  ```

**FR-DHT-004 — Jump to a chapter**
- **Priority:** Should
- **Platforms:** Backend, Website, iOS
- **Description:** WHEN a talk has chapters THE SYSTEM SHALL return them ordered by start time with a
  title; no chapters returns an empty list; a text talk always has none.
- **Business rules:** BR-DHT-05
- **Verification method:** unit, end-to-end
- **Source:** requirements US-DT-03
- **Acceptance criteria:**
  ```gherkin
  Scenario: Chapters in order
    Given a talk with chapters at 1200, 0 and 600 seconds
    When a visitor opens it
    Then the chapters are ordered 0, 600, 1200
    And each has a title

  Scenario: No chapters
    Given a talk with no chapters
    When a visitor opens it
    Then the chapter list is empty
    And the client hides the section
  ```

**FR-DHT-005 — Predictable shapes for missing data**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS
- **Description:** WHEN a field has no value THE SYSTEM SHALL return null for date, duration, season and
  follow-up meditation, and an empty list for tags and key points; a duration is never zero.
- **Business rules:** BR-DHT-06
- **Verification method:** unit
- **Source:** requirements US-DT-04
- **Acceptance criteria:**
  ```gherkin
  Scenario: Missing optional data
    Given a talk with no date, no duration, no season and no tags
    When a visitor opens it
    Then those fields are null or empty as defined
    And the client can hide each block without a special case
  ```

**FR-DHT-006 — Seed the catalogue on first run**
- **Priority:** Must
- **Platforms:** Backend
- **Description:** WHEN the seed command runs against an empty catalogue THE SYSTEM SHALL insert every
  bundled talk in one transaction, using its file position as the sort order; a second run changes
  nothing; the service never seeds at startup.
- **Business rules:** BR-DHT-07, BR-DHT-08
- **Verification method:** repository test, command-line test
- **Source:** requirements US-DT-05
- **Acceptance criteria:**
  ```gherkin
  Scenario: First seed
    Given an empty catalogue
    When the seed command runs
    Then 67 talks are stored
    And the sort order follows the file

  Scenario: Second seed
    Given the catalogue already has talks
    When the seed command runs again
    Then it writes nothing and exits successfully

  Scenario: Invalid file
    Given a file with a duplicate talk id
    When the seed command runs
    Then it exits with an error and writes nothing
  ```

**FR-DHT-007 — Hide talks flagged for review**
- **Priority:** Must
- **Platforms:** Backend
- **Description:** WHEN a talk is flagged for review THE SYSTEM SHALL keep it stored but never return it
  from the list or the detail.
- **Business rules:** BR-DHT-02
- **Verification method:** repository, end-to-end
- **Source:** requirements AC 4, 7
- **Acceptance criteria:**
  ```gherkin
  Scenario: Flagged talk is hidden
    Given a stored talk flagged for review
    When a visitor browses the list or opens its id
    Then it never appears and its detail is "not found"
  ```

**FR-DHT-008 — Play a talk on any client**
- **Priority:** Must
- **Platforms:** Backend, Website, iOS, Android, Flutter
- **Description:** WHEN a talk has an audio file THE SYSTEM SHALL return an absolute, public `audio_url`
  that a media player can open without a session, serve it with `HEAD`, byte ranges and CORS, and keep
  the detail readable when the file is missing.
- **Business rules:** BR-DHT-03
- **Verification method:** unit and integration on the backend; manual playback on each client.
- **Source:** requirements AC 20–25
- **Acceptance criteria:**
  ```gherkin
  Scenario: Client gets a playable URL
    Given an audio talk stored with a root-relative path
    When a client opens its detail
    Then audio_url is an absolute http(s) URL
    And it opens without an Authorization header

  Scenario: Player seeks and probes
    Given the audio URL
    When the player sends a HEAD request
    Then it gets the size and "Accept-Ranges: bytes"
    When it sends a Range request
    Then it gets 206 with Content-Range

  Scenario: File is missing
    Given the file behind audio_url was removed
    When the player requests it
    Then the talk detail still shows the text and source
    And the player shows an unavailable state
  ```

### 3.4 Non-functional requirements

- **NFR-DHT-001 — Public availability:** both routes are public and answer without a session.
  Platforms: Backend.
- **NFR-DHT-002 — Deterministic order:** two requests with unchanged data return the same talks in the
  same order. Platforms: Backend, Website, iOS, Android, Flutter.
- **NFR-DHT-003 — No leaked internals:** a failed route answers in the shared envelope and never exposes
  the cause; a non-read request is refused with the allowed method. Platforms: Backend.
- **NFR-DHT-004 — Media delivery:** an audio response is cacheable for one day, needs no session,
  supports `HEAD` and byte ranges, carries `audio/mpeg` for mp3, and is readable from another origin.
  Platforms: Backend, Website, iOS, Android, Flutter.

### 3.5 Data requirements

| Data item | Meaning | Owned by | Retention / deletion | Private or public |
|---|---|---|---|---|
| Talk | A lecture in one media form | Backend | Kept; seed only adds | Public (unless flagged) |
| Speaker | Teacher's name | Backend | With the talk | Public |
| Media fields | Video id or absolute audio URL | Backend | With the talk | Public |
| Chapters | Time markers | Backend | With the talk | Public |
| Source and licence | Where it came from and its terms | Backend | With the talk | Public |
| Review flag | Hides an unverified talk | Backend | With the talk | Operational |

### 3.6 Platform variance

None intended: the same catalogue and field shapes serve every client. The player itself follows each
platform's media capabilities.

### 3.7 Software system attributes

- **Reliability:** a failed read returns an error; it never shows a hidden talk.
- **Availability:** same as the rest of the product; the video variant also depends on YouTube.
- **Security & privacy:** no personal data; the catalogue is public.
- **Maintainability:** a fixed field shape lets every client decode once and hide empty blocks.

---

## 4. Verification & traceability

### 4.1 Verification approach

Unit and repository tests for listing, detail, paging and the seed; end-to-end tests over the running
product for both public routes and the hidden-talk rule; a content review before any real release.

### 4.2 Traceability matrix

| ID | Backend | Website | iOS | Android | Flutter | Note |
|---|---|---|---|---|---|---|
| BR-DHT-01 | done | done | planned | later | later | |
| BR-DHT-02 | done | done | planned | later | later | 19 of 67 hidden |
| BR-DHT-03 | done | done | planned | later | later | |
| BR-DHT-04 | done | done | planned | later | later | |
| BR-DHT-05 | done | done | planned | later | later | Chapters empty in seed |
| BR-DHT-06 | done | done | planned | later | later | |
| BR-DHT-07 | done | n/a | n/a | n/a | n/a | Seed only |
| BR-DHT-08 | done | n/a | n/a | n/a | n/a | |
| BR-DHT-09 | in-progress | in-progress | planned | later | later | Licence unknown |
| FR-DHT-001 | done | done | planned | later | later | 3 scenarios |
| FR-DHT-002 | done | done | planned | later | later | 2 scenarios |
| FR-DHT-003 | done | done | planned | later | later | 3 scenarios |
| FR-DHT-004 | done | done | planned | later | later | 2 scenarios |
| FR-DHT-005 | done | done | planned | later | later | 1 scenario |
| FR-DHT-006 | done | n/a | n/a | n/a | n/a | 3 scenarios |
| FR-DHT-007 | done | done | planned | later | later | 1 scenario |
| FR-DHT-008 | done | done | planned | later | later | 6 criteria, 3 scenarios |
| NFR-DHT-001 | done | done | planned | later | later | |
| NFR-DHT-002 | done | done | planned | later | later | |
| NFR-DHT-003 | done | n/a | n/a | n/a | n/a | |
| NFR-DHT-004 | done | done | planned | later | later | |

### 4.3 Test vectors

| File | Rule | Cases |
|---|---|---|
| — | — | No shared test vector. Paging arithmetic and chapter ordering are covered by backend tests. |

---

## 5. Appendices

### Appendix A — Open questions

#### Open (unconfirmed)

> ⚠️ Not confirmed: is the number of listens shown and counted — the original request was cut off; the
> backend has no field or route for it. This blocks any "2.4k lượt nghe" display.

> ⚠️ Not confirmed: the "listen, then sit" card — the app has no meditation entity for the follow-up to
> point at, so it is always null in the seed. This blocks the card and the talk → meditation mapping.

> ⚠️ Not confirmed: "Các phần" (chapters) — no seed talk has a time marker; the source (YouTube chapters
> or manual listening) and the owner are undecided. This blocks the chapter data.

> ⚠️ Not confirmed: the valid list of seasons and how a talk is assigned one — today it is free text and
> null everywhere. This blocks season filtering and display.

> ⚠️ Not confirmed: the review flag — 19 of 67 talks are hidden, including three unverified Thông Triệt
> talks; who decides and how a talk is un-flagged (a manual database edit today) is undecided. This blocks release.

> ⚠️ Not confirmed: audio sources — 8 of 67 talks have a valid Làng Mai mp3 and 12 more point at an Archive.org collection under CC BY-NC-SA; the owner confirmed the app is non-commercial, so these are visible, though the uploader's rights are not independently verified; other lawful sources
> and any permission from temples are not settled. This blocks the audio share of the catalogue.

> ⚠️ Not confirmed: licence — the owner confirmed on 2026-11-15 that InnerLamp is non-commercial and
> charges users nothing, so the twelve CC BY-NC-SA talks are shown with attribution. Still open: whether
> the app may show a summary or quote and embed the video, whether the source must be shown, and whether
> the uploader held the rights to license the recordings.

> ⚠️ Not confirmed: catalogue size — the target was 60 talks; 67 are stored because twelve Archive.org audio talks were added, 19 hidden. This blocks declaring the library complete.

> ⚠️ Not confirmed: the speaker's display name — the stored name keeps its title prefix ("Hòa thượng
> Thích …") while the mock says "Thầy …"; whether the app shortens it or the backend adds a display name
> is undecided.

> ⚠️ Not confirmed: how content is updated after the first seed — editing the bundled file and
> rebuilding does not change an existing catalogue; a manual database edit, a reseed command or a backoffice is
> undecided.

> ⚠️ Not confirmed: the user-story codes `US-DT-01..05` were never confirmed by product. This blocks
> stable traceability if they change.

> ⚠️ Not confirmed: a publication date for a YouTube talk is the upload date, not the lecture date. This
> affects how the date is labelled.

> ⚠️ Not confirmed: whether a talk may have no speaker at all (the seed refuses an empty speaker). This
> affects anonymous or collective talks.

#### Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| Is the catalogue public? | Yes, both routes, like articles. | 2026-10-03 |
| One media type or several per talk? | Exactly one of youtube, audio, text. | 2026-10-03 |
| Does the service seed at startup? | No; a separate seed command on first run. | 2026-10-11 |
| Is a "top" route needed? | No, for now. | 2026-10-03 |
| How are chapters ordered? | By start time; empty for text. | 2026-10-03 |
| Where are audio files hosted? | Local disk in dev, object storage/CDN through `server.audio_base_url` in production; the API always returns an absolute URL. | 2026-11-15 |

### Appendix B — Contract handoff

The contract at v3.0.0 covers the public list and detail. The exact shapes and error names live in
`docs/specs/api/openapi.yaml`; this appendix is kept until the client platform specs are rewritten
against this document.
