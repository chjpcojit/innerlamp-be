# Requirements — gallery

Version: 2.0
Last updated: 2026-10-15
Status: in-progress — AC 1–2, 4–10, 12–15, 17–18 are implemented (top / paged list with `type` filter / detail, seed `cmd/seedgallery`); **v2.0 (2026-10-15) not implemented**: credit in the list items, `total` on `top`, the seeder requiring author and licence, the shared order vectors, E2E (AC 3, 11, 16, 19; `tasks.md` Phase 6–8). The content file is already committed (AC 20, be#35).
Module: `gallery`
Code: `services/appService/`
Product spec: `docs/product/gallery/product.md` @ v1.0
Contract: `docs/specs/api/openapi.yaml` (this repo)
Implements: BR-GAL-01, BR-GAL-02, BR-GAL-03, BR-GAL-04, BR-GAL-07, BR-GAL-09, FR-GAL-001, FR-GAL-002, FR-GAL-003, FR-GAL-006, FR-GAL-009, NFR-GAL-01, NFR-GAL-02, NFR-GAL-03

> See `changelog.md` in this folder for the version history of this spec.

This spec was first written prospectively (v1.0, 2026-10-06) and implemented the same day; from v2.0 the business source is the product spec above (Mixed mode: the shipped part is Retroactive, the **v2.0** part is Prospective). It does not restate the product's rules — it cites IDs and says how the backend does them. Where the spec and the shipped code disagree, the code is the source of truth for the shipped part. `api-contract.md` in this folder is an old client document, superseded by `openapi.yaml` (see its banner); it is deleted in the cleanup step.

The gallery is a **read-only, public** catalogue shaped like `temple`, `deity` and `article`: `top`, a paged list with an optional `type` filter, and a detail. One table `gallery_images` (migration `0017`) holds the four topics; the wire values map to the product's topics as `CHUA` → Chùa, `TUONG_PHAT` → Tượng Phật, `THIEN` → Cảnh thiền, `LE_HOI` → Lễ hội (BR-GAL-01; labels are the client's). Data is loaded by the manually run `cmd/seedgallery` from `assets/data/buddhist_gallery.json`.

Product IDs whose Backend column is `n/a` (not part of this spec): BR-GAL-05, BR-GAL-06, BR-GAL-08, FR-GAL-004, FR-GAL-005, FR-GAL-007, FR-GAL-008. BR-GAL-07 is computed by the apps; the backend only supplies the total (AC 3). BR-GAL-09 needs nothing beyond serving titles and places unchanged.

---

## US-GAL-01: View featured photos

## User Story

> As an **app or website user (no login needed)**,
> I want **to see a few Buddhist photos on the home screen and know how many the gallery holds**,
> So that **I can enjoy them and open the whole gallery**.

## Acceptance Criteria (EARS notation)

1. WHEN a `GET /v1/gallery/top` request arrives, THE SYSTEM SHALL return `200 OK` with `data.items` holding the first `gallery.top_limit` photos (10) of every type, ordered by `sort_order` ASC then `id` ASC. (No `Authorization` needed.) (FR-GAL-001, BR-GAL-02)
2. WHEN the table holds fewer photos than `gallery.top_limit`, or none, THE SYSTEM SHALL return all that exist, and `items: []` (not `null`) for an empty table. (BR-GAL-02)
3. **v2.0** WHEN `top` is served, THE SYSTEM SHALL also return `data.total` = the number of photos in the whole table (no filter), so an app computes "+N ảnh" without a second call. (BR-GAL-07, FR-GAL-001)

## US-GAL-02: Browse the gallery page by page

## User Story

> As an **app or website user (no login needed)**,
> I want **to scroll through the gallery and filter it by topic**,
> So that **I find the kind of photo I like**.

## Acceptance Criteria (EARS notation)

4. WHEN a `GET /v1/gallery?page=N` request arrives, THE SYSTEM SHALL return `200 OK` with `data.items` (at most `gallery.page_size` = 20), `page`, `page_size`, `total`, `total_pages` = `ceil(total / page_size)`, ordered by `sort_order` ASC then `id` ASC (the same order as `top`, so page 1 starts with the photos of `top`). (FR-GAL-002, BR-GAL-02, BR-GAL-03)
5. WHEN `page` is omitted or empty, THE SYSTEM SHALL serve page 1. (FR-GAL-002)
6. WHEN `page` is not an integer, is less than 1, or is greater than 2^31−1, THE SYSTEM SHALL return `400 INVALID_PAGE` with `details.field = "page"`. (platform)
7. WHEN `page` is past the last page, THE SYSTEM SHALL return `200 OK` with `items: []` and the real `total` / `total_pages`. (BR-GAL-03)
8. WHEN `type` is exactly one of `CHUA`, `TUONG_PHAT`, `THIEN`, `LE_HOI`, THE SYSTEM SHALL return only photos of that type in the same order, and `total` / `total_pages` SHALL count only that type. (FR-GAL-003, BR-GAL-01, BR-GAL-02)
9. WHEN `type` is present, non-empty and not exactly one of the four values (including another case such as `chua`), THE SYSTEM SHALL return `400 INVALID_GALLERY_TYPE` with `details.field = "type"`; WHEN both parameters are invalid, `INVALID_PAGE` is reported first. (platform)
10. WHEN `type` is omitted or empty, THE SYSTEM SHALL return photos of every type. (FR-GAL-003)

## US-GAL-03: Credit every photo without a second call

## User Story

> As a **client developer**,
> I want **every photo in every list to carry its author, licence and source link**,
> So that **the viewer can credit the photo as the licence requires without fetching each detail**.

## Acceptance Criteria (EARS notation)

11. **v2.0** WHEN photos are returned by `top` or by the paged list, THE SYSTEM SHALL return each one with `id`, `type`, `title`, `location`, `thumbnail_url`, `image_url`, `author`, `license`, `source_url` — every field a string, never `null`. (BR-GAL-04, FR-GAL-006)
12. WHEN `thumbnail_url` / `image_url` are stored as root-relative paths (`/static/images/...`), THE SYSTEM SHALL return them as absolute URLs on `server.public_base_url`, or on the request's origin when that is empty; any other stored value is returned unchanged (`docs/assumptions/gallery/A-001`). (platform, NFR-GAL-02)

## US-GAL-04: View one photo

## User Story

> As a **client developer**,
> I want **to read one photo by its id**,
> So that **a client that only holds an id can still show and credit it**.

## Acceptance Criteria (EARS notation)

13. WHEN a `GET /v1/gallery/{id}` request arrives with an existing id, THE SYSTEM SHALL return `200 OK` with the fields of AC 11. (FR-GAL-006)
14. WHEN no photo has that id, THE SYSTEM SHALL return `404 GALLERY_IMAGE_NOT_FOUND`. (platform)
15. WHEN the literal path `/v1/gallery/top` is requested, THE SYSTEM SHALL route it to `top`, not to detail (`ServeMux` prefers the literal segment; ids have the form `gallery_NNN`). (platform)

## US-GAL-05: Load the gallery content

## User Story

> As the **operator**,
> I want **to load the gallery file into the database, all or nothing**,
> So that **the gallery never shows a half-loaded or uncredited set**.

## Acceptance Criteria (EARS notation)

16. **v2.0** WHEN `cmd/seedgallery` reads the file, THE SYSTEM SHALL refuse the whole file and touch no row when it is empty, an id is empty or repeated, a type is not one of the four values, a title or either picture path is empty, **or an author or licence is empty**. (FR-GAL-009, BR-GAL-04)
17. WHEN the file is valid, THE SYSTEM SHALL assign `sort_order` by rotating the types `CHUA` → `TUONG_PHAT` → `THIEN` → `LE_HOI`, keeping the file order within a type and skipping a type with no photo left. (BR-GAL-02)
18. WHEN the file is valid, THE SYSTEM SHALL insert new photos and update existing ones by `id` in **one** transaction (all or nothing); ids stay stable across loads. (FR-GAL-009)
19. **v2.0** THE SYSTEM SHALL pass every case of `docs/product/gallery/rules/gallery-order.json` (order and featured list) through the seeder's ordering plus the `top` limit, mapping the vector types `TEMPLE`/`STATUE`/`MEDITATION`/`FESTIVAL` to `CHUA`/`TUONG_PHAT`/`THIEN`/`LE_HOI`. (BR-GAL-02)
20. THE SYSTEM SHALL keep `assets/data/buddhist_gallery.json` in git as the default input of `cmd/seedgallery` (done in be#35). (FR-GAL-009)

> Numbering note: v1.0 used three sections of bullets without numbers; v2.0 numbers every criterion.

## API contract (confirmed surface)

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| GET | `/v1/gallery/top` | public | 200 `{items: GallerySummary[], total}` (**v2.0** `total`) | 500 |
| GET | `/v1/gallery?type=&page=` | public | 200 `{items, page, page_size, total, total_pages}` | 400 `INVALID_PAGE`, 400 `INVALID_GALLERY_TYPE`, 500 |
| GET | `/v1/gallery/{id}` | public | 200 `GalleryDetail` | 404 `GALLERY_IMAGE_NOT_FOUND`, 500 |

**v2.0** `GallerySummary` gains `author`, `license`, `source_url`, so `GalleryDetail` holds the same fields as the summary; both schema names stay for compatibility. Additive only: existing clients keep decoding. `openapi.yaml` is updated with the code (`tasks.md` Phase 7).

## Out of scope

- Writing photos through the API (back office) — later (product §1.2).
- Favourite, share, set as wallpaper (BR-GAL-08) — no backend part.
- Photo width/height for a masonry layout — later (product §1.2).
- Search.

## Pending (not yet implemented)

- v2.0 code: credit in the lists, `total` on `top`, seeder checks, vectors, E2E — `tasks.md` Phase 6–8.

> ⚠️ Not confirmed (product): architecture decision pending — where catalogue photos are stored and served in production (needs an ADR, shared with temple, deity, article) — blocks production release; today photos are served from `assets/images/gallery/` on the dev machine (`⚠️ DEV ONLY`, not in git).

> ⚠️ Not confirmed (product): who runs the load in staging and production and where the photo files come from there — blocks production content; `cmd/seedgallery` uses dev config like the other seeders.

## Open questions (resolved)

- ~~"Lấy ảnh theo danh sách" = paged list with an optional `type` filter?~~ → Yes, confirmed by product FR-GAL-003 (2026-10-15). No change.
- ~~User story codes US-GAL-01..03 unconfirmed~~ → Product IDs are `BR-/FR-GAL-*`; the US codes stay as local section names (2026-10-15).
- ~~Contract not in `openapi.yaml`~~ → It has been there since v1.0 (tag Gallery); the v1.1 note was stale (2026-10-15).
- ~~Base URL of images at staging/production~~ → Partly resolved by A-001 (`server.public_base_url`); where the files live is the pending ADR above.

## Technical notes

- `sort_order` is computed by the seeder, not stored in the file; the source file groups photos by type.
- `top` reuses `GalleryRepository.List(ctx, "", 0, top_limit)`, which already returns the total, so `total` on `top` needs no new repository method (`design.md`).
- Reuses `ErrInvalidPage`, `INVALID_PAGE`, the `{success, data}` envelope and `writeDomainError`.
- 86 photos on 2026-10-15 (41 `CHUA`, 21 `TUONG_PHAT`, 12 `THIEN`, 12 `LE_HOI`); every one has an author and a licence, so AC 16 refuses nothing in the current file.
