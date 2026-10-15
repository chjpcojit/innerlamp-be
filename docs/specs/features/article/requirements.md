# Requirements — article

Version: 3.0
Last updated: 2026-10-15
Status: in-progress — AC 1–19 are implemented (top / paged list / detail, seed `cmd/seedarticles`), tested at usecase, handler, seed-parser and repository level; no E2E yet. **v3.0 (2026-10-15) not implemented**: related articles `GET /v1/articles/{id}/related` (AC 20–25); the content file is committed to git (AC 26, done in the same change as this spec).
Module: `content`
Code: `services/appService/`
Product spec: `docs/product/article/product.md` @ v1.0
Contract: `docs/specs/api/openapi.yaml` (this repo)
Implements: BR-ART-01, BR-ART-02, BR-ART-03, BR-ART-04, FR-ART-001, FR-ART-002, FR-ART-003, FR-ART-005, FR-ART-009, NFR-ART-01, NFR-ART-02

> See `changelog.md` in this folder for the version history of this spec.

This spec was first written **retroactively** from the shipped "Bài viết Phật pháp hay" (Dharma
articles) catalogue; from v3.0 the business source is the product spec above (Mixed mode: the shipped
part is Retroactive, the related-articles part is Prospective). It does not restate product rules — it
cites IDs and says how the backend does it. Where this spec and the code disagree on the shipped part,
the code is the source of truth and the spec is corrected.

The catalogue is **read-only** and public, shaped like `temple` and `deity`: a featured list, a paged
list and a detail, plus (v3.0) the related articles of one article. Data is loaded by the manually run
`cmd/seedarticles` command, never when the service starts.

Product IDs whose Backend column is `n/a` (not part of this spec): BR-ART-05…09, FR-ART-004,
FR-ART-006, FR-ART-007, FR-ART-008. The server side of BR-ART-05/BR-ART-07 is only the deterministic
data shape of US-ART-04 (both image URLs, `null` for unknown values); choosing which image to show and
hiding empty values are client work. Sharing (ADR-0004) involves no backend.

---

## US-ART-01: View featured articles

## User Story

> As an **app or website user (no login needed)**,
> I want **to see a few curated Dharma articles on the home screen**,
> So that **I have something to read right away without searching**.

## Acceptance Criteria (EARS notation)

1. WHEN a `GET /v1/articles/top` request arrives, THE SYSTEM SHALL return `200 OK` with `data.items` holding the first `article.top_limit` articles (5) ordered by `sort_order` ASC then `id` ASC. No `Authorization` is needed. (FR-ART-001, BR-ART-01, BR-ART-02)
2. WHEN articles are returned by `top`, by the paged list or (v3.0) by `related`, THE SYSTEM SHALL return each one in the **summary** shape only — `id`, `title`, `category`, `author`, `author_initials`, `published_date`, `read_minutes`, `summary`, `tags`, `image_url`, `thumbnail_url` — and SHALL NOT include `body_paragraphs`, `quote`, `quote_source`, `language`, `source_name`, `source_url` or `license`. (FR-ART-001, FR-ART-002, FR-ART-005, NFR-ART-02)
3. WHEN the table holds fewer articles than `article.top_limit`, or none, THE SYSTEM SHALL return all that exist, and SHALL return `items: []` (not `null`) for an empty table. (BR-ART-02)

## US-ART-02: Browse all articles page by page

## User Story

> As an **app or website user (no login needed)**,
> I want **to scroll through every article page by page**,
> So that **I can read beyond the featured ones**.

## Acceptance Criteria (EARS notation)

4. WHEN a `GET /v1/articles` request arrives without `page` (or with an empty `page=`), THE SYSTEM SHALL serve page 1. The order is the same as `top`, so page 1 starts with exactly the `top` articles. (FR-ART-002, BR-ART-01)
5. WHEN a page is served, THE SYSTEM SHALL return `data` with `items` (summary shape), `page`, `page_size` (`article.page_size` = 10), `total` (number of articles in the table) and `total_pages` = `ceil(total / page_size)`. (FR-ART-002, BR-ART-03, NFR-ART-01)
6. WHEN `page` is greater than `total_pages`, THE SYSTEM SHALL return `200 OK` with `items: []` and the same `page_size`/`total`/`total_pages` — not an error. (BR-ART-03)
7. WHEN `page` is not a positive integer (`0`, negative, non-numeric, an integer that overflows, or larger than 2^31−1), THE SYSTEM SHALL return `400 INVALID_PAGE` with `details.field = "page"`. (platform)

## US-ART-03: Read one article

## User Story

> As an **app or website user (no login needed)**,
> I want **to open one article and read it in full**,
> So that **I can read its body, its quote and where it comes from**.

## Acceptance Criteria (EARS notation)

8. WHEN a `GET /v1/articles/{id}` request arrives with an existing `id`, THE SYSTEM SHALL return `200 OK` with the summary fields plus `body_paragraphs`, `quote`, `quote_source`, `language`, `source_name`, `source_url` and `license`. (FR-ART-003)
9. WHEN `{id}` matches no article, THE SYSTEM SHALL return `404 ARTICLE_NOT_FOUND`. (FR-ART-003)
10. WHEN the literal path `/v1/articles/top` is requested, THE SYSTEM SHALL route it to the featured list, not to detail — so an article with `id = "top"` cannot be read through the detail route. (platform)

## US-ART-04: Consistent data shape

## User Story

> As a **client developer**,
> I want **missing values to always have a known shape**,
> So that **I can decode once without guessing between `null` and an empty string**.

## Acceptance Criteria (EARS notation)

11. WHEN an article has a publication date, THE SYSTEM SHALL return `published_date` as a `"YYYY-MM-DD"` string; WHEN it has none, THE SYSTEM SHALL return JSON `null` — never an empty string or a fake date. (platform)
12. WHEN an article has a cover image, THE SYSTEM SHALL return `image_url` as a string; WHEN it has none, THE SYSTEM SHALL return JSON `null`. (platform)
13. WHEN an article has a thumbnail, THE SYSTEM SHALL return `thumbnail_url` as an absolute URL of a small image this service hosts (a root-relative stored path is prefixed with `server.public_base_url`); WHEN it has none, THE SYSTEM SHALL return JSON `null`. The key is always present, in the summary and in the detail. (NFR-ART-02)
19. WHEN an article has no tags or no body paragraphs, THE SYSTEM SHALL return `[]` for `tags` / `body_paragraphs`, never `null`. (platform)
14. WHEN any article route fails, THE SYSTEM SHALL answer in the shared envelope `{"success": false, "error": {code, message, type}, "details"?}`; a non-`GET` request returns `405 METHOD_NOT_ALLOWED` (with header `Allow: GET, HEAD`) and an infrastructure failure returns `500 INTERNAL_ERROR` without leaking the cause. (platform)

> Numbering note: v2.0 had two criteria numbered 13; the second (empty arrays) is renumbered 19 so no older reference shifts. AC 20–26 were added in v3.0.

## US-ART-05: Load content with the seed command (the service never seeds)

## User Story

> As a **person deploying the system**,
> I want **one seed command that loads the articles into the database, which `make run` calls when the table is empty**,
> So that **a new environment has data without the service writing to the database on start, and re-running never touches edited rows**.

## Acceptance Criteria (EARS notation)

15. WHEN `cmd/seedarticles` runs, THE SYSTEM SHALL parse the JSON (embedded in the binary by default, or `-file`) and insert every article whose `id` is not in the `articles` table in **one transaction**, with `sort_order` equal to the article's position in the file (0, 1, 2, …), then log how many were inserted. WHEN appService starts, THE SYSTEM SHALL NOT read the seed or write any article. (FR-ART-009, BR-ART-01)
16. WHEN `cmd/seedarticles` runs and finds an article `id` already in the table, THE SYSTEM SHALL leave that row unmodified (only `-fill-images` sets a still-`NULL` `image_url` / `thumbnail_url`). `make ensure-articles` runs the command **only when the table has no row**, so the default flow never brings back a deleted article. (FR-ART-009)
17. WHEN a seed insert meets an `id` that already exists, THE SYSTEM SHALL leave that row untouched (`ON CONFLICT (id) DO NOTHING`), so two runs at the same time cannot overwrite each other or later edits. (FR-ART-009)
18. WHEN the seed is invalid (no article, an empty or duplicate `id`, an empty `title`, a `publishedDate` that is not `YYYY-MM-DD`) or the database rejects it (including when migration `0012` has not been applied), THE SYSTEM SHALL make `cmd/seedarticles` exit non-zero with a fatal log and SHALL leave the table unchanged (transaction rolled back). (FR-ART-009)
26. **v3.0** THE SYSTEM SHALL keep the content file `assets/data/buddhist_articles.json` under version control (`.gitignore` exception), so every environment and a fresh clone embed and load the same 60 articles. (FR-ART-009)

> ⚠️ Not confirmed (product): how an article that is already loaded is corrected or removed (the seed never changes an existing row; product Appendix A item 4) — blocks a content-update path, not this spec; hand back to ba-writer in innerlamp-be.

## US-ART-06: Related articles ("Đọc tiếp") — v3.0

## User Story

> As an **app or website user (no login needed)**,
> I want **two articles on the same topic suggested at the end of the article I am reading**,
> So that **I can keep reading without going back to the list**.

## Acceptance Criteria (EARS notation)

20. WHEN a `GET /v1/articles/{id}/related` request arrives with an existing `id`, THE SYSTEM SHALL return `200 OK` with `data.items` holding up to `article.related_limit` (2) articles in the summary shape, chosen and ordered per BR-ART-04: first the other articles whose category equals the current one after `btrim` + `lower`, in `sort_order` ASC then `id` ASC; then, if fewer than the limit, the remaining articles in the same order. No `Authorization` is needed. (FR-ART-005, BR-ART-04, BR-ART-01)
21. WHEN choosing related articles, THE SYSTEM SHALL never return the requested article itself and never return the same article twice. (BR-ART-04)
22. WHEN the table holds no other article, THE SYSTEM SHALL return `items: []` (not `null`). (BR-ART-04)
23. WHEN `{id}` matches no article, THE SYSTEM SHALL return `404 ARTICLE_NOT_FOUND`. (FR-ART-005)
24. THE SYSTEM SHALL pass every case of `docs/product/article/rules/article-related.json` (and `article-order.json` for the shared order) in an automated test. (BR-ART-04, BR-ART-01)
25. WHEN a non-`GET` request reaches `/v1/articles/{id}/related`, or the database fails, THE SYSTEM SHALL answer as in AC 14. (platform)

## API contract (confirmed surface for this feature)

The contract is `docs/specs/api/openapi.yaml`, tag Articles. Summary of the routes this spec covers:

| Method | Path | Auth | Success | Failure codes | State |
|---|---|---|---|---|---|
| GET | `/v1/articles/top` | public | `200` `{items: [ArticleSummary]}` | 405, 500 | shipped |
| GET | `/v1/articles?page=N` | public | `200` `{items, page, page_size, total, total_pages}` | 400 `INVALID_PAGE`, 405, 500 | shipped |
| GET | `/v1/articles/{id}` | public | `200` `ArticleDetail` | 404 `ARTICLE_NOT_FOUND`, 405, 500 | shipped |
| GET | `/v1/articles/{id}/related` | public | `200` `{items: [ArticleSummary]}` (0–2) | 404 `ARTICLE_NOT_FOUND`, 405, 500 | **v3.0, to add to `openapi.yaml`** |

## Out of scope (this feature boundary, not tracked here)

- Creating, editing or removing articles through the API or the back office (product: later feature).
- Search, filtering by `category`/`tag`, bookmarks (BR-ART-09), read tracking.
- Share links and their association files (ADR-0004) — website and apps only.
- Copying cover images to GCS/CDN: `image_url` stays a hotlink to the source site or Wikimedia Commons.
- Authentication: every route is public, like `temple` and `deity`.

## Pending (not yet implemented)

- Related articles (AC 20–25) — `tasks.md` Phase 7.
- E2E test running the real binary + Postgres for the four routes — `tasks.md` Phase 5.

> ⚠️ Not confirmed (product): whether InnerLamp may show the summaries, their images and the hosted thumbnails, and how image authors are credited (product Appendix A item 1; A-001, A-002) — blocks public release of the content, not this spec.

## Technical notes

- Seed data: 60 articles, 58 Vietnamese and 2 English; 29 have a `publishedDate`; 60/60 have an `imageURL` (38 from the source page's `og:image`, 15 from an image inside the source page, 7 from Wikimedia Commons — A-001) and a `thumbnailURL` (root-relative `/static/images/articles/thumb/<id>.webp`, A-002). The 60 articles carry 44 distinct categories, most with 1–3 articles, so BR-ART-04 usually falls back to the featured order.
- `id` is the stable slug from the source data (e.g. `vipassana-nghe-thuat-song`), so ids cached by clients and shared links stay valid across reloads.
- `total` and the page are read by two separate statements (count, then page), not one snapshot — see `design.md`.
- Sizes live in the `article:` block of `config.<env>.yaml` (`top_limit: 5`, `page_size: 10`, v3.0 `related_limit: 2`); all are required — 0 means an empty result.

## Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| User story numbering | `US-ART-01…06` kept; product IDs are the cross-platform reference | 2026-10-15 |
| Featured criterion | Editorial order of the content file (BR-ART-02) | 2026-10-15 |
| Content update process after the first load | Content file + seed only; back office later (FR-ART-009); correcting a loaded row stays open (product A4) | 2026-10-15 |
| Display conditions and attribution | Clients always show source and summary notice (BR-ART-06); rights stay open (product A1) | 2026-10-15 |
| Related articles | Backend picks 2 of the same category, filled from the featured order (BR-ART-04) | 2026-10-15 |
