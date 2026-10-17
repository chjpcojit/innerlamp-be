# Design — lamp

Version: 2.0
Last updated: 2026-10-17
Status: v1 (domain, config, usecase, repository, migration `0027`, handler, wiring, OpenAPI) is implemented and verified (2026-10-10): unit, repository (real Postgres) and E2E (real binary + Postgres) tests pass. **v2.0 is design only, not implemented** (`tasks.md` Phase 6–8): optional recipient, `is_mine`, vector-file test, OpenAPI 3.0.0. Open `⚠️ Not confirmed`: garden latency target; retention of expired lamps. `erd.svg` not regenerated (no `mmdc`).
Module: `lamp`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Both `/v1/lamps` routes require authentication (wrapped in `authMiddleware.AuthenticateFunc`, `RequireTokenType: "access"`, like `/v1/memorials`), but `GET` reads the lamps of **every** user. The `userID` from `pkg/jwt/context` lights the lamp and, **since v2.0**, is also passed to the garden query to compute `is_mine`; it never filters the list. The feature touches all four layers of `services/appService` (`domain` → `usecase` → `repository` / `delivery/http`, wired in `cmd/main.go`), owns migration `0027_lamps.sql` (and the v2.0 migration) and the `lamp:` config block. The closest shape is `memorial` (create in one transaction that locks the `users` row, count the cap, insert) plus a cross-user list paged by **cursor** (deliberately different from the `page`/`total_pages` convention of `temple`; decided 2026-10-10). No external service, no `pkg/storage`.

What the behavior is lives in `docs/product/lamp/product.md` v1.0; this file only says how the backend builds it.

## Layers & components

"v1" marks what exists; "v2.0" marks the planned change (all `[ ]` in `tasks.md`).

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/lamp.go` | `Lamp` (id, userID, `LampType`, `Intention`, recipient, `Prayer *string`, durationDays, isAnonymous, `LighterName *string`, litAt, endAt) + `DisplayName()`; `LampInput` (raw request); `GardenLamp` (id, `LampType`, recipient, `LighterName *string` + `DisplayName()` mapping `nil` to `Vô danh`, `LitAt` only to build the cursor — never leaves in the DTO, `EndAt`); `GardenFilter` (three raw queries, `nil` = absent); `GardenQuery`; `GardenPage` (items, total, counts `map[LampType]int`, hasMore, nextCursor *string); `Cursor{LitAt, ID}`; `LightLampParams{Lamp, MaxActive, DedupeWindow}`. Enums `LampType` (3, with `AllLampTypes`), `Intention` (4), `AnonymousLighterName = "Vô danh"`. No JSON/db tags. **v1 implemented.** **v2.0:** `Lamp.Recipient` and `GardenLamp.Recipient` become `*string` (`nil` = no recipient); `GardenLamp.IsMine bool`; `GardenQuery.ViewerID int64`. |
| Domain / Errors | `domain/lamp.go` | `ErrLampLimitReached`, `ErrInvalidLampFilter`, `ErrInvalidLimit`, `ErrInvalidCursor`. `ErrLampInvalid` + `*LampValidationError{Field, Message}` (with `Unwrap` to the sentinel) — the handler reads `Field` and returns `400 VALIDATION_ERROR` with `details.field`, same pattern as `MemorialValidationError`. **v1 implemented; v2.0 no change.** |
| Domain / Repository interface | `domain/lamp.go` | `LampRepository`: `Light(ctx, p LightLampParams) (lamp *Lamp, created bool, err error)` — one transaction (lock, snapshot the dharma name, dedupe, cap, insert); `Garden(ctx, q GardenQuery) (*GardenPage, error)` — one read transaction, same snapshot for the page and the counts, at most `Limit+1` rows. `LightLampParams` carries `Lamp.LitAt` (the shared clock), `MaxActive`, `DedupeWindow`, so the repository never reads the clock or the config. **v1 implemented; v2.0:** signatures unchanged, `GardenQuery` gains `ViewerID`. |
| Domain / Mock | `domain/mock_lamp_repository.go` | Hand-written mock for usecase tests. |
| Usecase | `services/appService/usecase/lamp_usecase.go` | `Light(ctx, userID, in LampInput)`: trim + NFC, validate AC 3–8 (enum, `durations_days`, `recipient_max_chars`, `prayer_max_chars` from config, lengths counted in characters), compute `litAt = u.now()` (truncated to microseconds) and `endAt`, call `LampRepository.Light`. `Garden(ctx, f GardenFilter)`: validate `lamp_type`/`limit`/`cursor` (AC 17, 19, 21), clamp `limit` to `PageSize`, take `now`, call `LampRepository.Garden`, trim to `limit` rows, build `has_more`/`next_cursor` (cursor encode/decode lives in the same file) and fill `0` for absent types in `counts`. `now func() time.Time` is injectable. **v1 implemented.** **v2.0:** `Light` treats an empty/whitespace-only recipient as absent (`nil`), no longer an error; `Garden` becomes `Garden(ctx, userID int64, f GardenFilter)` and puts `userID` into `GardenQuery.ViewerID`. |
| Repository | `services/appService/repository/postgres_lamp_repository.go` | SQL on `lamps`. `Light`: tx locks `users`, reads `dharma_name`, dedupe, cap, insert. `Garden`: `READ ONLY` + `REPEATABLE READ` tx, `GROUP BY` counts then keyset page. Every "is lit" comparison is `end_at > $now` with `$now` passed down by the usecase. **v1 implemented.** **v2.0:** dedupe compares `recipient IS NOT DISTINCT FROM $5` (`prayer` already does); the page query adds `(user_id = $viewer) AS is_mine` and still does **not** select `lighter_name` for anonymous lamps (the column is `NULL` for them). |
| Delivery / Handler | `services/appService/delivery/http/lamp_handler.go` | `Light`: `userID` from context → `DecodeJSONBodyStrict` → usecase → `201`/`200`. `Garden`: reads the `lamp_type`, `limit`, `cursor` queries (kept as-is even when empty so the usecase rejects them) → usecase. No business logic. **v1 implemented.** **v2.0:** `Garden` reads `userID` from context and passes it to the usecase. |
| Delivery / DTO | `delivery/http/lamp_handler.go` | `lampRequest`, `lampResponse`, `gardenLampResponse`, `gardenResponse` — JSON tags only here; `lit_at`/`end_at` RFC 3339 UTC; `items` always `[]`, never `null`; `counts` always has all 3 keys; `next_cursor` is a base64url string or `null`. **v2.0:** `lampRequest.Recipient` becomes `*string` (absent/`null`/blank all mean no recipient); `lampResponse.Recipient` and `gardenLampResponse.Recipient` become `*string` (JSON `null`, never `""`); `gardenLampResponse` gains `is_mine`. |
| Delivery / Error map | `delivery/http/errors.go` | `ErrLampLimitReached` → `409 LAMP_LIMIT_REACHED`; `ErrInvalidLampFilter` → `400 INVALID_FILTER` (`details.field = "lamp_type"`); `ErrInvalidLimit` → `400 INVALID_LIMIT`; `ErrInvalidCursor` → `400 INVALID_CURSOR`; `*LampValidationError` → `400 VALIDATION_ERROR` + `details.field`. **v1 implemented; v2.0 no change.** |
| Config | `services/appService/config/config.go`, `config.dev.yaml` | `LampConfig{MaxActivePerUser (20), PageSize (12), DedupeWindow (5s), DurationsDays ([1,7,21,49]), RecipientMaxChars (100), PrayerMaxChars (300)}`; `Validate` rejects `MaxActivePerUser < 1`, `PageSize < 1`, `DedupeWindow <= 0`, empty / `< 1` / duplicate `DurationsDays`, `RecipientMaxChars`/`PrayerMaxChars` `< 1` or **above the column size** (`LampRecipientColumnChars = 100`, `LampPrayerColumnChars = 300` — they match the `VARCHAR` sizes of `0027_lamps.sql` and are the only schema constants left; guards, not the values used for checking). The `lamp:` block is required in every `config.<env>.yaml`. **v1 implemented; v2.0 no change** (numbers unchanged, BR-LMP-03/04/06/07). |
| Wiring | `services/appService/cmd/main.go` | Builds repo → usecase (passes `cfg.Lamp`) → handler; registers `POST /v1/lamps` and `GET /v1/lamps`, **both wrapped in `authMiddleware.AuthenticateFunc`**. **v1 implemented; v2.0 no change.** |
| Migration | `migrations/0027_lamps.sql` | Table `lamps` (below) + three indexes. Idempotent (`IF NOT EXISTS`). `migrations/README.md` updated. **v1 implemented.** **v2.0:** a new migration (next free number; `0038` is the latest today) runs `ALTER TABLE lamps ALTER COLUMN recipient DROP NOT NULL`, idempotent; `0027` is never edited. Existing rows keep their recipients. |
| API spec | `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`) | 2 routes, Bearer security, schemas `Lamp`/`LampRequest`/`GardenLamp`/`GardenPage`. **v1 implemented (`info.version` 2.1.1).** **v2.0:** `recipient` optional in `LampRequest`, nullable in `Lamp` and `GardenLamp`, `is_mine` added to `GardenLamp` → `info.version` 3.0.0 (MAJOR: a field becomes nullable), a dated entry in `web/docs/src/content/docs/changelog.md`, re-sync with `node scripts/sync-openapi.mjs`. |
| DB spec | `docs/specs/db/erd.mmd`, `erd.svg` | Table `lamps` + relation `users 1—N lamps` exist in `erd.mmd`. **v2.0:** mark `recipient` nullable; regenerate `erd.svg` with `mmdc`. |
| Test / vectors | `services/appService/usecase/lamp_vectors_test.go` | **v2.0 (new):** runs `docs/product/lamp/rules/lamp-lifetime.json`, `lamp-input-validation.json`, `lamp-lighter-name.json` against the usecase. The loader expands the `{"repeat":..,"times":..}` and `{"concat":[..]}` notations and counts characters after NFC. |
| Test E2E | `services/appService/e2e/lamp_e2e_test.go` | Real binary + Postgres: 401 on both routes without/with junk/refresh token, cap of 20 including concurrent, dedupe, `counts` unchanged by the filter, cursor without repeats/gaps when lamps expire. **v2.0:** add no-recipient light and read-back (`null`), `is_mine` for the caller's lamp vs another user's, anonymous name absent from the garden. |

### Schema `lamps`

```
id             BIGSERIAL PRIMARY KEY
user_id        BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE   -- confirmed in v2.0: account deletion removes lamps (AC 29)
lamp_type      VARCHAR(16)  NOT NULL CHECK (IN 'OIL_LAMP','LOTUS_CANDLE','FLOWER_LANTERN')
intention      VARCHAR(20)  NOT NULL CHECK (IN 'PEACE','SALVATION','MERIT_DEDICATION','GRATITUDE')
recipient      VARCHAR(100)                 -- v2.0: NULL = no recipient (was NOT NULL)
prayer         VARCHAR(300)                 -- NULL = none
duration_days  SMALLINT     NOT NULL CHECK (duration_days > 0)
is_anonymous   BOOLEAN      NOT NULL DEFAULT false
lighter_name   VARCHAR(255)                 -- dharma name copied at lighting; NULL when anonymous or no dharma name
lit_at         TIMESTAMPTZ  NOT NULL        -- set by the usecase (server clock), no DEFAULT now()
end_at         TIMESTAMPTZ  NOT NULL

CHECK (end_at > lit_at)
CHECK (is_anonymous = false OR lighter_name IS NULL)   -- anonymous means the name is never kept

idx_lamps_lit       (lit_at DESC, id DESC)                 -- garden, all types
idx_lamps_type_lit  (lamp_type, lit_at DESC, id DESC)      -- garden filtered by type
idx_lamps_user_end  (user_id, end_at)                      -- cap count and dedupe
```

`duration_days` has no `CHECK IN (1,7,21,49)` in the DB because the value set lives in config (`lamp.durations_days`); the DB only blocks absurd values. `recipient VARCHAR(100)` and `prayer VARCHAR(300)` are **hard ceilings**: config may set lower limits but `Validate` does not allow higher. `lit_at` has no `DEFAULT now()` so `lit_at`/`end_at` always share the clock of the `end_at > $now` comparison. `is_mine` is computed, never stored.

## Request flow

Both routes run two or more sequential DB statements in one transaction (count + insert; count + page), so they are in the Step-by-step Flows below.

> See `flow.md` for the Mermaid sequence diagrams (1:1 mirror of the Step-by-step Flows below).

## Step-by-step Flow — Light a lamp (POST /v1/lamps)

```
Step L-1 — Authenticate (middleware)
- Trigger: request to POST /v1/lamps
- Pre-condition: header Authorization: Bearer <access token>
- Call: authMiddleware.AuthenticateFunc
    - Success → userID in context
    - Failure (missing/expired/wrong token type) → 401, stop (AC 2)

Step L-2 — Parse + validate + compute times (this module)
- Trigger: after L-1
- Pre-condition: userID in context
- Call: Handler DecodeJSONBodyStrict → LampUsecase.Light: trim + NFC on recipient/prayer, a blank recipient becomes nil (v2.0), validate AC 3–8, now := u.now(), litAt = now, endAt = now + durationDays×24h
    - Success → LightLampParams{lamp (no id/lighter_name yet), maxActive, dedupeWindow}
    - Failure → 400 VALIDATION_ERROR with details.field (AC 3–8, 11); the DB is not touched

Step L-3 — Lock, snapshot name, dedupe, check cap, insert (this module, ONE transaction)
- Trigger: after L-2
- Pre-condition: valid data
- Call: LampRepository.Light(ctx, p)
    a. BEGIN
    b. SELECT dharma_name FROM users WHERE id=$1 FOR UPDATE   (serializes one user's lightings — stops going over the cap and two double-submitted requests both inserting; also snapshots the dharma name in the same tx)
    c. SELECT … FROM lamps WHERE user_id=$1 AND lit_at > $now - $dedupeWindow AND lamp_type, intention, duration_days, is_anonymous equal AND recipient IS NOT DISTINCT FROM $5 AND prayer IS NOT DISTINCT FROM $6 ORDER BY lit_at DESC LIMIT 1
    d. if a row exists → COMMIT, return (that row, created=false)   (AC 10; checked before the cap)
    e. SELECT count(*) FROM lamps WHERE user_id=$1 AND end_at > $now
    f. if count >= maxActive → ROLLBACK, ErrLampLimitReached
    g. lighter_name := (is_anonymous ? NULL : dharma_name read in b)   (AC 12)
    h. INSERT INTO lamps (...) RETURNING id
    i. COMMIT → return (lamp, created=true)
    - Success (new) → created=true
    - Success (duplicate inside the window) → created=false, nothing written
    - Failure (cap reached) → 409 LAMP_LIMIT_REACHED (AC 9)
    - Failure (DB error) → 500 INTERNAL_ERROR, rolled back
    - Transaction boundary: a–i are one tx; an error at any step rolls everything back.

Step L-4 — Build the response (this module)
- Trigger: after a successful L-3
- Pre-condition: tx committed
- Call: Handler maps entity → lampResponse (lighter_name = "Vô danh" when NULL; recipient stays null when absent, v2.0)
    - Success (created=true) → 201 Lamp (AC 1)
    - Success (created=false) → 200 Lamp, the existing lamp (AC 10)
```

## Step-by-step Flow — See the lamp garden (GET /v1/lamps)

```
Step G-1 — Authenticate (middleware)
- Trigger: request to GET /v1/lamps
- Pre-condition: header Authorization: Bearer <access token>
- Call: authMiddleware.AuthenticateFunc
    - Success → userID in context (v2.0: used only for is_mine, never to filter)
    - Failure → 401, no lamp returned (AC 14)

Step G-2 — Parse query + validate (this module)
- Trigger: after G-1
- Pre-condition: authenticated request
- Call: Handler reads lamp_type, limit, cursor and userID → LampUsecase.Garden: validate AC 17/19/21, clamp limit to PageSize, decode cursor into (litAt, id), now := u.now()
    - Success → GardenQuery{lampType?, limit, cursor?, now, viewerID}
    - Failure (unknown/empty lamp_type) → 400 INVALID_FILTER; (limit not a positive integer) → 400 INVALID_LIMIT; (bad cursor) → 400 INVALID_CURSOR; the DB is not touched

Step G-3 — Count and fetch the page (this module, ONE read-only transaction)
- Trigger: after G-2
- Pre-condition: valid query
- Call: LampRepository.Garden(ctx, q)
    a. BEGIN READ ONLY, ISOLATION REPEATABLE READ   (counts and page share one snapshot — AC 18)
    b. SELECT lamp_type, count(*) FROM lamps WHERE end_at > $now GROUP BY lamp_type   → total = sum; absent types filled with 0 (independent of the filter)
    c. SELECT id, lamp_type, recipient, lighter_name, lit_at, end_at, (user_id = $viewer) AS is_mine FROM lamps WHERE end_at > $now [AND lamp_type=$t] [AND (lit_at, id) < ($cursorLitAt, $cursorId)] ORDER BY lit_at DESC, id DESC LIMIT $limit+1
    d. COMMIT
    - Success → rows, total, counts
    - Failure (DB error) → 500 INTERNAL_ERROR
    - Transaction boundary: a–d are one read snapshot; no writes.
    - ⚠️ Not confirmed: whether counting by scanning the lit rows with GROUP BY meets the response-time goal when lamps are many — the product owner has no target or volume yet (NFR-LMP-01); it blocks the choice of a pre-computed count table or a cache.

Step G-4 — Build the response (this module)
- Trigger: after G-3
- Pre-condition: rows, total, counts available
- Call: Usecase trims to `limit` rows (hasMore = limit+1 rows were read), builds nextCursor from (lit_at, id) of the last row, fills 0 for absent types in counts; Handler maps to gardenResponse (drops lit_at from each item, lighter_name NULL → "Vô danh" via `GardenLamp.DisplayName()`, recipient NULL → JSON null, is_mine as read)
    - Success → 200 {items, total, counts, has_more, next_cursor} (AC 13–22, 25–27)
```

**Why keyset (cursor) instead of `OFFSET`:** the order `lit_at DESC, id DESC` is stable; the anchor is the last row of the previous page, so a lamp that expires or is lit between two loads does not shift the page (AC 20). `OFFSET` would repeat or skip exactly in those two cases. The cursor is base64url of `<lit_at unix micro>.<id>`, opaque to clients; a bad decode returns `INVALID_CURSOR`.

**Why dedupe is in the same transaction as the cap:** locking the `users` row in step b already serializes two requests from one user, so "find a duplicate, else insert" cannot race between two double-submitted requests; no idempotency column or lock is needed and the client sends nothing extra.

**Why `is_mine` is computed in the page query:** the comparison `user_id = $viewer` happens in SQL, so `user_id` never has to be loaded into `GardenLamp` or reach the DTO; the only thing that leaves is a boolean, which cannot reveal another lamp's owner (AC 25).

**Why `IS NOT DISTINCT FROM` for the recipient (v2.0):** with a nullable recipient, `recipient = $5` is never true when both sides are `NULL`, so two identical no-recipient requests would both insert. `IS NOT DISTINCT FROM` treats `NULL` as equal to `NULL`, as `prayer` already needs.

## Deliberately stubbed for now (if applicable)

None. Retention of expired lamps (marker in `requirements.md`) has no route or job, so there is nothing to stub; nothing is deleted except by account deletion, which has no route yet either.

## Known gaps carried by this feature

- **v2.0 is not implemented**: the code still requires a recipient, has no `is_mine`, and the lamp vector files are not run by any test (`tasks.md` Phase 6–8).
- `lamps.user_id` is `ON DELETE CASCADE`, now confirmed as the intended behavior (AC 29). The expired-lamp retention period is still open (marker in `requirements.md`); changing it needs a new migration, never an edit of `0027`.
- Both routes are wrapped in `authMiddleware.AuthenticateFunc` in `cmd/main.go`, and `e2e/lamp_e2e_test.go` proves both return 401 without a token, with a junk token or with a refresh token.
- The 5-second dedupe window is checked at the repository level (real Postgres); E2E only proves `201` then `200`. The "after 5 seconds it is a new lamp" half needs a real wait and has no E2E test.
- `docs/specs/db/erd.mmd` has the `lamps` table, but `erd.svg` is not regenerated because this machine has no `mmdc`; the repo has no script for it.
- The `usecase` package does not compile under plain `go test` because `genealogy_member_usecase_test.go` (unfinished genealogy work) references a missing `GenealogyMemberUsecase`; the lamp tests ran through `-overlay` that drops that file. Not part of this feature.
- Cursor paging differs from the `page`/`total_pages` convention of the other list features (decided on purpose); clients must support both styles.
