# Requirements — lamp

Version: 2.0
Last updated: 2026-10-17
Status: v1 shipped — `POST /v1/lamps` and `GET /v1/lamps` (domain, usecase, repository, migration `0027`, handler, wiring, OpenAPI) are implemented and covered by unit, repository (real Postgres) and E2E tests. **v2.0 not implemented**: the recipient becomes optional (AC 6, 26), garden items gain `is_mine` (AC 25), account deletion removes lamps (AC 29), and the shared vector files are not run yet (AC 30) — `tasks.md` Phase 6–8, every item `[ ]`. Open: the garden latency target (NFR-LMP-01) and how long expired lamps are kept.
Module: `lamp`
Code: `services/appService/`
Product spec: `docs/product/lamp/product.md` @ v1.0
Contract: `docs/specs/api/openapi.yaml` (this repo)
Implements: BR-LMP-01, BR-LMP-02, BR-LMP-03, BR-LMP-04, BR-LMP-05, BR-LMP-06, BR-LMP-07, BR-LMP-08, BR-LMP-09, BR-LMP-10, BR-LMP-11, BR-LMP-13, BR-LMP-14, FR-LMP-003, FR-LMP-004, FR-LMP-005, FR-LMP-006, FR-LMP-007, FR-LMP-008, FR-LMP-009, FR-LMP-010, FR-LMP-011, FR-LMP-013, NFR-LMP-01, NFR-LMP-02, NFR-LMP-03

> See `changelog.md` in this folder for the version history of this spec.

Backend platform spec (layer 2) for **lighting a lamp and the lamp garden**. The product behavior (closed sets, burn durations, the end moment, the limit of 20, the recipient and prayer rules, the double-submission window, the lighter's name, what other users see, garden order and counts, final lamps, sign-in) lives in the product spec and is **not** restated here; this file says how the backend implements those `BR-LMP-*` / `FR-LMP-*`. The spec was written prospectively (v1.0, 2026-10-10) and implemented the same day; v2.0 (2026-10-17) rebases it on the product spec, rewrites it in English per the Language policy and adds the two product decisions of 2026-10-16 (Retroactive for the shipped part, Prospective for the v2.0 part). Where this spec and the shipped code disagree on the shipped part, the code wins.

## Constraints from the product spec

- **Every route needs a signed-in user**, and the garden reads the lamps of **all** users (BR-LMP-14, BR-LMP-11). The user id from the token lights the lamp and, since v2.0, only marks which garden items are the caller's; it never filters the list.
- **Only four things leave the server about someone else's lamp**: the lighter's name, the lamp type, the recipient and the end moment (BR-LMP-10). Intention and prayer are returned only in the response of the lighting request itself. v2.0 adds one boolean about the caller's own lamps (`is_mine`, AC 25).
- **The lighter's name is the dharma name copied at lighting time** (BR-LMP-09). The account's full name is never read. A hidden name is not stored on the lamp at all.
- **Time is the server's** (BR-LMP-03, BR-LMP-04, NFR-LMP-03): `end_at = lit_at + duration_days × 24h`, both UTC; a lamp is lit while `end_at` is later than the request's `now`.
- **The recipient is optional** (BR-LMP-06, decision 2026-10-16): a blank recipient is stored as "none" and returned as `null`; the clients render "Tất cả chúng sanh" in the viewer's language. This reverses v1 (`recipient` required).
- **Remaining-time labels (BR-LMP-12) are the clients' work**; the server only returns `end_at`.
- **Limits are configuration** (BR-LMP-05, BR-LMP-06, BR-LMP-07, BR-LMP-08): `lamp.max_active_per_user`, `lamp.page_size`, `lamp.dedupe_window`, `lamp.durations_days`, `lamp.recipient_max_chars`, `lamp.prayer_max_chars`.
- User story codes `US-LAMP-01..02` and AC numbers 1–24 are kept so references in code and tests do not break; AC 6 changes meaning, AC 25–30 are new.

---

## US-LAMP-01: Light a lamp

## User Story

> As a **signed-in user**,
> I want **to light a lamp with a lamp type, an intention, an optional recipient, an optional prayer and a burn duration**,
> So that **my wish stays in the lamp garden for the days I chose**.

## Acceptance Criteria (EARS notation)

1. WHEN a `POST /v1/lamps` request arrives with a valid access token and a valid body, THE SYSTEM SHALL create a lit lamp owned by the token's user and return `201 Created` with the full `Lamp` (shape in "API contract"), where `lit_at` is the server time of the request and `end_at = lit_at + duration_days × 24 hours`. (FR-LMP-003, BR-LMP-03)
2. WHEN the request carries no valid access token (missing, expired, wrong token type), THE SYSTEM SHALL return `401` and SHALL NOT create anything. (FR-LMP-013, BR-LMP-14)
3. WHEN `lamp_type` is not one of `OIL_LAMP`, `LOTUS_CANDLE`, `FLOWER_LANTERN`, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "lamp_type"`. (FR-LMP-003, BR-LMP-01)
4. WHEN `intention` is not one of `PEACE`, `SALVATION`, `MERIT_DEDICATION`, `GRATITUDE`, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "intention"`. (FR-LMP-003, BR-LMP-01)
5. WHEN `duration_days` is not one of the configured durations (`lamp.durations_days`, dev: `1, 7, 21, 49`), THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "duration_days"`. (FR-LMP-003, BR-LMP-02)
6. WHEN `recipient` is absent, `null`, empty or blank after trimming leading and trailing whitespace, THE SYSTEM SHALL store no recipient and return `recipient: null`; WHEN the trimmed recipient is longer than `lamp.recipient_max_chars` (dev: 100) characters, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "recipient"`; otherwise the stored value is the trimmed text. Length is counted in characters (code points after NFC normalization), not bytes. *(v2.0: a blank recipient used to be rejected.)* (FR-LMP-003, BR-LMP-06)
7. WHEN `prayer` is longer than `lamp.prayer_max_chars` (dev: 300) characters after trimming, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "prayer"`; WHEN `prayer` is absent, `null` or blank after trimming, THE SYSTEM SHALL store `NULL` and return `prayer: null`. (FR-LMP-003, BR-LMP-07)
8. WHEN `is_anonymous` is absent, THE SYSTEM SHALL treat it as `false`; WHEN it is present and not a boolean, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "is_anonymous"`. (FR-LMP-003, BR-LMP-09)
9. WHEN the user already has `lamp.max_active_per_user` (dev: 20) lit lamps — a lit lamp being one whose `end_at` is later than now — THE SYSTEM SHALL return `409 LAMP_LIMIT_REACHED` and SHALL NOT insert; lamps whose `end_at` has passed SHALL NOT count, and two requests of the same user running concurrently SHALL NOT push the user above the limit. (FR-LMP-004, BR-LMP-04, BR-LMP-05)
10. WHEN the same user submits a lamp whose content (`lamp_type`, `intention`, `recipient`, `prayer`, `duration_days`, `is_anonymous`) is identical to a lamp they lit within the last `lamp.dedupe_window` (dev: 5s), THE SYSTEM SHALL NOT insert a second lamp and SHALL return `200 OK` with that existing lamp; WHEN the previous identical lamp is older than the window, THE SYSTEM SHALL create a new lamp. The duplicate check runs **before** the limit check, so a double submission at exactly 20 lamps gets its lamp back, not a `409`. Two blank recipients count as identical. (FR-LMP-005, BR-LMP-08)
11. WHEN the body is not valid JSON, is empty, has a wrong `Content-Type`, or carries a field the endpoint does not accept (`id`, `user_id`, `lit_at`, `end_at`, `lighter_name`, …), THE SYSTEM SHALL return `400` and SHALL NOT create anything. (platform)
12. WHEN `is_anonymous` is `true`, THE SYSTEM SHALL store no name on the lamp and return `lighter_name: "Vô danh"`; WHEN it is `false`, THE SYSTEM SHALL copy the user's current dharma name onto the lamp, or store no name when the user has none (returned as `"Vô danh"`). The account's full name SHALL NOT be read. (FR-LMP-003, BR-LMP-09)

## US-LAMP-02: See the lamp garden

## User Story

> As a **signed-in user**,
> I want **to see the lit lamps of everyone, filter them by lamp type, load more and see which are mine**,
> So that **I see the community lighting lamps and how many are lit**.

## Acceptance Criteria (EARS notation)

13. WHEN a `GET /v1/lamps` request arrives with a valid access token, THE SYSTEM SHALL return `200` with `data.items` holding the lit lamps of **all** users, ordered by `lit_at` DESC then `id` DESC (most recently lit first), each item carrying `id`, `lamp_type`, `recipient`, `lighter_name`, `end_at` and `is_mine`. (FR-LMP-007, BR-LMP-10, BR-LMP-11)
14. WHEN the request carries no valid access token, THE SYSTEM SHALL return `401` and SHALL NOT return any lamp or count. (FR-LMP-013, BR-LMP-14)
15. THE SYSTEM SHALL NOT return `intention`, `prayer`, `user_id`, `is_anonymous` or `lit_at` in any garden item. (FR-LMP-007, BR-LMP-10)
16. WHEN `lighter_name` is returned for a lamp with no stored name (anonymous lighter, or no dharma name), THE SYSTEM SHALL return the literal `Vô danh`; THE SYSTEM SHALL NOT return the dharma name of an anonymous lamp in any field. (FR-LMP-007, BR-LMP-09)
17. WHEN `lamp_type` is absent, THE SYSTEM SHALL list lamps of every type; WHEN it is one of the three types, THE SYSTEM SHALL list only that type; WHEN it is present but not one of the three (including empty), THE SYSTEM SHALL return `400 INVALID_FILTER` with `details.field = "lamp_type"`. (FR-LMP-009, BR-LMP-11)
18. THE SYSTEM SHALL return in the same response `data.total` (lit lamps of all users, all types) and `data.counts` with one key per lamp type (`OIL_LAMP`, `LOTUS_CANDLE`, `FLOWER_LANTERN`, `0` when none is lit), both independent of the `lamp_type` filter and read from the same snapshot as the page. (FR-LMP-011, BR-LMP-11, NFR-LMP-02)
19. WHEN `limit` is absent, THE SYSTEM SHALL return at most `lamp.page_size` (dev: 12) items; WHEN `limit` is greater than `lamp.page_size`, THE SYSTEM SHALL return at most `lamp.page_size` items (no error); WHEN `limit` is not a positive integer, THE SYSTEM SHALL return `400 INVALID_LIMIT` with `details.field = "limit"`. (FR-LMP-010, BR-LMP-11)
20. WHEN more lit lamps match than the page holds, THE SYSTEM SHALL return `has_more: true` and a `next_cursor`; WHEN the client sends that `cursor`, THE SYSTEM SHALL return the lamps strictly after the last one of the previous page in the same order, so that no lamp is repeated or skipped because other lamps expired or were lit in between. WHEN there is no further lamp, THE SYSTEM SHALL return `has_more: false` and `next_cursor: null`. (FR-LMP-010, BR-LMP-11)
21. WHEN `cursor` is present but not a value previously issued by this endpoint, THE SYSTEM SHALL return `400 INVALID_CURSOR` with `details.field = "cursor"`. (FR-LMP-010)
22. WHEN no lamp is lit, THE SYSTEM SHALL return `200` with `items: []` (not `null`), `total: 0`, every count `0`, `has_more: false`, `next_cursor: null`. (FR-LMP-007, FR-LMP-011)
23. WHEN a lamp's `end_at` is not later than the server time of the request, THE SYSTEM SHALL NOT list it, SHALL NOT count it in `total` or `counts`, and SHALL NOT count it toward the user's limit — checked at request time, with no cleanup job involved. (FR-LMP-006, BR-LMP-04)
24. THE SYSTEM SHALL write `end_at` (and `lit_at` in the `Lamp` response) as RFC 3339 in UTC (`2026-10-17T18:00:00Z`); THE SYSTEM SHALL NOT take the current time from the client. (FR-LMP-006, BR-LMP-03, NFR-LMP-03)
25. WHEN a garden item is a lamp lit by the token's user, THE SYSTEM SHALL return `is_mine: true` for it and `false` for every other item, on every device of the account and for anonymous lamps too. The flag SHALL NOT depend on `lamp_type`, `limit` or `cursor`, and SHALL NOT reveal any other lamp's owner. *(New in v2.0.)* (FR-LMP-008, BR-LMP-10)
26. WHEN a lamp has no recipient, THE SYSTEM SHALL return `recipient: null` in the `Lamp` response and in garden items, and SHALL NOT substitute any text such as "all beings" on the server. *(New in v2.0.)* (BR-LMP-06)
27. WHEN a lamp is lit anonymously, THE SYSTEM SHALL keep the lighter's name out of the garden response entirely (it is not stored; `is_mine` does not change that). (BR-LMP-09, BR-LMP-10)
28. THE SYSTEM SHALL offer no route to edit, extend, put out or delete a lamp. (BR-LMP-13)
29. WHEN a user's account is deleted, THE SYSTEM SHALL delete that user's lamps; expired lamps are kept until then. *(v2.0: confirms the existing `ON DELETE CASCADE`; there is no delete-account route yet, so nothing triggers it.)* (BR-LMP-04, product §3.5)
30. THE SYSTEM SHALL pass the shared test vectors `docs/product/lamp/rules/lamp-lifetime.json`, `lamp-input-validation.json` and `lamp-lighter-name.json` in a backend test. *(New in v2.0.)* (BR-LMP-01, BR-LMP-02, BR-LMP-03, BR-LMP-04, BR-LMP-06, BR-LMP-07, BR-LMP-09)

## API contract (confirmed surface for this feature)

> Present in `docs/specs/api/openapi.yaml` (tag `Lamps`, two routes, schemas `Lamp` / `LampRequest` / `GardenLamp` / `GardenPage`) and in the copy `web/docs/public/openapi.yaml`, at `info.version` 2.1.1. **v2.0 changes the contract** (see "Contract changes" below); the OpenAPI file is updated with the code, not by this spec. Envelope `{"success", "data"}` / `{"success": false, "error", "details"?}` as for every route.

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| POST | `/v1/lamps` | `authMiddleware` (access token) | `201` `Lamp` (new lamp) or `200` `Lamp` (the identical lamp lit within the dedupe window) | 400 `VALIDATION_ERROR` (`details.field`), 401, 405, 409 `LAMP_LIMIT_REACHED`, 500 |
| GET | `/v1/lamps?lamp_type=&limit=&cursor=` | `authMiddleware` (access token) | `200` `{items: [GardenLamp], total, counts, has_more, next_cursor}` | 400 `INVALID_FILTER`, 400 `INVALID_LIMIT`, 400 `INVALID_CURSOR`, 401, 405, 500 |

**Request `POST /v1/lamps`**

```json
{
  "lamp_type": "OIL_LAMP",
  "intention": "PEACE",
  "recipient": "Bà ngoại",
  "prayer": "Mong bà bình an",
  "duration_days": 7,
  "is_anonymous": false
}
```

Required: `lamp_type`, `intention`, `duration_days`. Optional: `recipient` (v2.0), `prayer`, `is_anonymous`.

**Response `Lamp`** (only to the lighter): `id`, `lamp_type`, `intention`, `recipient` (or `null`), `prayer` (or `null`), `duration_days`, `is_anonymous`, `lighter_name`, `lit_at`, `end_at`.

**Response `GardenLamp`**: `id`, `lamp_type`, `recipient` (or `null`), `lighter_name`, `end_at`, `is_mine`.

**Response `GET /v1/lamps`**

```json
{
  "success": true,
  "data": {
    "items": [
      { "id": 41, "lamp_type": "LOTUS_CANDLE", "recipient": "Thầy", "lighter_name": "Thích Minh An", "end_at": "2026-10-17T18:00:00Z", "is_mine": false },
      { "id": 40, "lamp_type": "OIL_LAMP", "recipient": null, "lighter_name": "Vô danh", "end_at": "2026-10-11T10:00:00Z", "is_mine": true }
    ],
    "total": 10,
    "counts": { "OIL_LAMP": 5, "LOTUS_CANDLE": 3, "FLOWER_LANTERN": 2 },
    "has_more": true,
    "next_cursor": "MTc5MDAwMDAwMDAwMDAwMC40MQ"
  }
}
```

### Contract changes in v2.0 (to land with the code)

| Change | Why | Semver effect |
|---|---|---|
| `Lamp.recipient` and `GardenLamp.recipient` become nullable | BR-LMP-06 | A type change a client must react to → **MAJOR** |
| `LampRequest.recipient` becomes optional | BR-LMP-06 | MINOR |
| `GardenLamp.is_mine` (boolean, always present) | FR-LMP-008 | MINOR |

Together: `info.version` 2.1.1 → **3.0.0**, a dated entry in `web/docs/src/content/docs/changelog.md`, and `web/docs/public/openapi.yaml` re-synced (`CLAUDE.md`, "OpenAPI versioning rule").

## Out of scope (this feature boundary, not tracked here)

- **Lighting a lamp at a temple for a fee**, payment, donation — the `offering` feature; no money or temple concept enters the `lamps` table.
- Moderating the recipient or the prayer, reporting, removing lamps from the back office (no moderation in this version).
- Extending, relighting, putting out or deleting a lamp (BR-LMP-13).
- A "my lamps" screen and the history of expired lamps.
- Filtering the garden by intention; "your lamp is about to go out" notifications.
- Artwork, animation and the remaining-time label (clients).
- Sharing a lit lamp.

## Pending (not yet implemented)

- Everything marked *v2.0* above (`tasks.md` Phase 6–8).
- Regenerating `docs/specs/db/erd.svg` from `erd.mmd` with `mmdc` (not installed on the machine that built v1) — `tasks.md` Phase 3.
- The OKF concept `docs/knowledge/features/lamp.md` (`il-okf-writer`, once the v2.0 flow is verified) — `tasks.md` Phase 5.

> ⚠️ Not confirmed: the response-time target of `GET /v1/lamps` and the expected number of lit lamps at the same time (the product owner does not know yet; to be measured after launch) — it blocks the decision between counting live rows and a pre-aggregated table or cache (the current design counts with `GROUP BY` over the lit rows, which is easy to replace). Product NFR-LMP-01.

> ⚠️ Not confirmed: how long expired lamps are kept (deferred on purpose: "store first, decide later") — it blocks any cleanup job and the retention rule. Meanwhile nothing is deleted except by account deletion (AC 29).

## Technical notes

- **Clock:** `lit_at`, `end_at` and every "is lit" comparison (`end_at > now`) use the single `now` the usecase takes from the server clock (injectable for tests), not scattered `now()` calls inside SQL, so `total`, `counts` and the page describe the same moment.
- **Limits live in configuration, not constants** in domain or usecase: `lamp.max_active_per_user`, `lamp.page_size`, `lamp.dedupe_window`, `lamp.durations_days`, `lamp.recipient_max_chars`, `lamp.prayer_max_chars` (see `design.md`, Config). The two length limits can only be **lowered** relative to the `VARCHAR(100)` / `VARCHAR(300)` columns: `Validate` rejects a value above the column at startup, because overflowing a column would become a `500` at write time. Raising a cap needs a migration that widens the column first.
- **Text normalization:** trim both ends, then NFC, before counting, so Vietnamese typed as base letter plus combining marks and as precomposed letters have the same length (vector `lamp-input-validation.json`). `golang.org/x/text` is already in `go.mod` (indirect); make it direct when used.
- **Privacy:** `recipient` can be a deceased relative's name and is shown to every signed-in user; neither `recipient` nor `prayer` is written to logs (ids only).
- **An anonymous lamp keeps no name in the database** (`lighter_name = NULL`, `is_anonymous = true`), so no other query can leak it. The users table only has an optional `dharma_name` (no ordinary-name field), so "no dharma name" always shows `Vô danh`.
- **`is_mine` is computed in the garden query** (`user_id = $viewer`), never from a second lookup, and the viewer id comes from the token only.
- **Relation to `memorial`:** same model — per-user data, a cap, the `users` row locked when creating; different in that the garden reads other users' lamps (five public fields).

## Open questions (resolved)

- ~~Who sees which lamps?~~ → Every signed-in user sees the lit lamps of all users. Resolved 2026-10-10.
- ~~Can guests see the garden?~~ → No. Resolved 2026-10-10 (reconfirmed 2026-10-16).
- ~~What is public about a lamp?~~ → Lighter's name, lamp type, recipient, end date; never intention and prayer. Resolved 2026-10-10.
- ~~Recipient and prayer text?~~ → Free text; recipient up to 100 characters, prayer up to 300, counted in characters, trimmed; the two numbers are configuration since v1.3. **Recipient became optional on 2026-10-16.**
- ~~Moderation in v1?~~ → None. Resolved 2026-10-10.
- ~~Cost?~~ → Free; the temple lamp with a fee belongs to the offering feature. Resolved 2026-10-10.
- ~~Cap?~~ → 20 lit lamps per user; expired ones do not count. Resolved 2026-10-10.
- ~~Expiry, extension, putting out early?~~ → The lamp disappears when it expires; no extension, no early removal. Resolved 2026-10-10.
- ~~Does `total` depend on the filter?~~ → No. Resolved 2026-10-10.
- ~~Order and page size?~~ → Most recently lit first; at most 12 per page. Resolved 2026-10-10.
- ~~Double submission?~~ → Identical content within 5 seconds is one lamp. Resolved 2026-10-10.
- ~~What does "N days" mean, which time zone?~~ → N × 24 hours from lighting; stored in UTC; the end date is shown in the viewer's device time zone. Resolved 2026-10-10.
- ~~Lighter's name?~~ → Dharma name; `Vô danh` when none; may be hidden; never changes after lighting. Resolved 2026-10-10.
- ~~Trimming?~~ → Yes, both ends, before validation and storage. Resolved 2026-10-10.
- ~~Garden paging: `page`/`total_pages` or cursor?~~ → **Cursor** (`cursor` / `next_cursor` / `has_more`), a deliberate deviation from the other lists (AC 20 forbids repeating or skipping lamps when others expire or are lit). Resolved 2026-10-10.
- ~~How does a client know which lamps are the user's?~~ → The server returns `is_mine`; no "my lamps" screen. Resolved 2026-10-16 (product FR-LMP-008).
- ~~Account deletion?~~ → Deletes the user's lamps. Resolved 2026-10-16 (product §3.5).
