# Tasks — lamp

Version: 2.0
Last updated: 2026-10-17
Status: Phase 1–5 (v1) implemented and verified 2026-10-10/11 — unit, repository (real Postgres) and E2E tests pass. **Phase 6–8 (v2.0) not started, every item `[ ]`.** Open `⚠️ Not confirmed`: garden latency target; retention of expired lamps. `erd.svg` not regenerated (no `mmdc`).
Module: `lamp`

> See `changelog.md` in this folder for the version history of this spec.

Each item traces to a row of the "Layers & components" table or a Step in `design.md`. Work test-first: the tests of each phase are written together with that phase's code.

## Phase 1 — Domain & Config

- [x] `domain/lamp.go`: entities `Lamp`, `LampInput`, `GardenLamp`, `GardenFilter`, `GardenQuery`, `GardenPage`, `Cursor`, `LightLampParams`; enums `LampType` (3 values), `Intention` (4 values); constant `AnonymousLighterName = "Vô danh"`
- [x] Sentinel errors `ErrLampLimitReached`, `ErrInvalidLampFilter`, `ErrInvalidLimit`, `ErrInvalidCursor`, `ErrLampInvalid` + `LampValidationError{Field, Message}`
- [x] Interface `LampRepository` (`Light`, `Garden`)
- [x] `domain/mock_lamp_repository.go`
- [x] `config/config.go`: `LampConfig{MaxActivePerUser, PageSize, DedupeWindow, DurationsDays}` + `Validate` (rejects `< 1`, window `<= 0`, empty/duplicate/`< 1` list) + tests in `config_test.go`
- [x] `config/config.dev.yaml`: `lamp:` block (`max_active_per_user: 20`, `page_size: 12`, `dedupe_window: "5s"`, `durations_days: [1, 7, 21, 49]`)
- [x] v1.3 — `config/config.go`: `RecipientMaxChars`, `PrayerMaxChars` in `LampConfig`, ceiling constants `LampRecipientColumnChars = 100` / `LampPrayerColumnChars = 300`, `Validate` rejects `< 1` or above the ceiling + tests
- [x] v1.3 — `config/config.dev.yaml`: `recipient_max_chars: 100` and `prayer_max_chars: 300` in the `lamp:` block
- [x] v1.3 — `usecase/lamp_usecase.go`: removed constants `lampRecipientMax`/`lampPrayerMax`, length checks use `cfg.RecipientMaxChars`/`cfg.PrayerMaxChars`; tests with limits other than 100/300 prove the values come from config
- [x] v1.3 — `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`) and the `lamps.md` web doc: 100/300 are described as config defaults

## Phase 2 — Usecase

- [x] `usecase/lamp_usecase.go`: `Light` (trim + NFC, validate AC 3–8, compute `litAt`/`endAt`), `Garden` (validate AC 17/19/21, clamp `limit`, decode cursor, build `next_cursor`/`has_more`, `NULL` → `Vô danh`); injectable `now`
- [x] Cursor encode/decode `<lit_at unix micro>.<id>` base64url (in `usecase/lamp_usecase.go`)
- [x] Table-driven unit tests (`testify`) with the mock repo: every validation AC, length boundaries 1/100/101 and 300/301 with composed and precomposed accents, whitespace-only strings, `end_at` exactly N×24h for all 4 durations, `limit` clamp, bad cursor, anonymous / no-dharma-name `lighter_name`

## Phase 3 — Repository

- [x] `ON DELETE` of `lamps.user_id` decided: `ON DELETE CASCADE` (same as `memorials`); confirmed as intended in v2.0 (AC 29)
- [x] `migrations/0027_lamps.sql` (idempotent `IF NOT EXISTS`; CHECKs for enums, `end_at > lit_at`, anonymous ⇒ `lighter_name IS NULL`; indexes `idx_lamps_lit`, `idx_lamps_type_lit`, `idx_lamps_user_end`); `migrations/README.md` updated — number `0027` because `0026` is taken by `0026_administrative_units.sql`
- [x] `docs/specs/db/erd.mmd` updated (table `lamps` + relation `users 1—N lamps`)
- [ ] Regenerate `docs/specs/db/erd.svg` from `erd.mmd` with `mmdc` — this machine has no `mmdc`
- [x] `repository/postgres_lamp_repository.go`: `Light` (one tx: lock `users` + read `dharma_name`, dedupe, cap, insert), `Garden` (`READ ONLY` `REPEATABLE READ` tx: `GROUP BY` counts + keyset page `LIMIT limit+1`); explicit column list, `context`, `%w`
- [x] Repository tests (`postgres_lamp_repository_test.go`, real Postgres): cap of 20 even with two concurrent `Light`, expired lamps not counted, dedupe inside/outside the window, anonymous keeps no name, changing `dharma_name` does not change old lamps, keyset without repeats/gaps when a lamp expires between pages, `counts` unchanged by `lamp_type`

## Phase 4 — Delivery

- [x] `delivery/http/lamp_handler.go`: DTOs (`lampRequest`, `lampResponse`, `gardenLampResponse`, `gardenResponse`), `Light` (`201`/`200`), `Garden`; `items` always `[]`, `counts` always 3 keys, `lit_at`/`end_at` RFC 3339 UTC
- [x] `delivery/http/errors.go`: map `ErrLampLimitReached`, `ErrInvalidLampFilter`, `ErrInvalidLimit`, `ErrInvalidCursor`, `*LampValidationError`
- [x] Register `POST /v1/lamps` and `GET /v1/lamps` in `cmd/main.go`, **both wrapped in `authMiddleware.AuthenticateFunc`** — verified wrapped
- [x] `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`) updated with the handler change
- [x] Handler tests (`lamp_handler_test.go`): success, `201` vs `200`, each error branch, extra-field/empty/non-JSON body, bad `limit`/`cursor`/`lamp_type`

## Phase 5 — Integration & documentation

- [x] `e2e/lamp_e2e_test.go` (real binary + Postgres): 401 on both routes without a token, with a junk token, with a refresh token; light → appears in the garden; expired lamp disappears and does not count toward the cap; `total`/`counts` unchanged by the filter; anonymous does not leak the dharma name; intention/prayer absent from the garden response
- [x] `go vet ./services/appService/...` clean; `go test -short ./services/appService/...` and `go test ./services/appService/e2e/ -run TestLampFlow_E2E` pass (`usecase` ran through `-overlay` because another feature's `genealogy_member_usecase_test.go` does not compile)
- [ ] Refresh the OKF concept with `il-okf-writer` after the code is verified (`docs/knowledge/features/lamp.md`, does not exist yet) — now also covers v2.0

## Phase 6 — Optional recipient (v2.0, AC 6, 26; design: Layers rows Domain, Usecase, Repository, DTO, Migration; Step L-2, L-3c, L-4)

- [ ] New migration (next free number — `0038` is the latest at spec time; check at implementation) with `ALTER TABLE lamps ALTER COLUMN recipient DROP NOT NULL`, idempotent; update `migrations/README.md`; never edit `0027`
- [ ] `domain/lamp.go`: `Lamp.Recipient` and `GardenLamp.Recipient` become `*string`
- [ ] `usecase/lamp_usecase.go` (`Light`): after trim + NFC, an empty recipient becomes `nil`; the max-length check applies only when present; update the mock if its signature uses the field
- [ ] `repository/postgres_lamp_repository.go` (`Light`): dedupe compares `recipient IS NOT DISTINCT FROM $5`; insert/scan handle `NULL`
- [ ] `delivery/http/lamp_handler.go`: `lampRequest.Recipient *string` (absent, `null` and blank all mean none); `lampResponse` and `gardenLampResponse` return JSON `null`, never `""`
- [ ] Tests: usecase (blank/whitespace-only/absent recipient → `nil`, no `VALIDATION_ERROR`), repository on real Postgres (two identical no-recipient requests inside the window → one row; no-recipient and with-recipient lamps are not duplicates), handler (`null` in both responses), E2E (light without recipient → `201`, garden returns `recipient: null`)

## Phase 7 — `is_mine` and the vector files (v2.0, AC 25, 27, 30; design: Layers rows Domain, Usecase, Repository, Handler, DTO, Test/vectors; Step G-2, G-3c, G-4)

- [ ] `domain/lamp.go`: `GardenLamp.IsMine bool`, `GardenQuery.ViewerID int64`
- [ ] `usecase/lamp_usecase.go`: `Garden(ctx, userID, f)` passes `userID` as `ViewerID`
- [ ] `repository/postgres_lamp_repository.go` (`Garden`): select `(user_id = $viewer) AS is_mine`; do not load `user_id` into the entity
- [ ] `delivery/http/lamp_handler.go`: `Garden` reads `userID` from context; `gardenLampResponse.IsMine` → `is_mine`
- [ ] Tests: repository (own lamp `true`, other users' `false`, own anonymous lamp `true` with the name still absent, flag unchanged by `lamp_type`/`limit`/`cursor`), handler, E2E (two users)
- [ ] `usecase/lamp_vectors_test.go`: load `docs/product/lamp/rules/lamp-lifetime.json`, `lamp-input-validation.json`, `lamp-lighter-name.json`; support the `{"repeat":..,"times":..}` and `{"concat":[..]}` notations; count characters after NFC; every case passes

## Phase 8 — Contract and documentation (v2.0; design: Layers rows API spec, DB spec)

- [ ] `docs/specs/api/openapi.yaml`: `recipient` optional in `LampRequest`, nullable in `Lamp` and `GardenLamp`, `is_mine` in `GardenLamp`; bump `info.version` 2.1.1 → 3.0.0 (MAJOR: a response field becomes nullable)
- [ ] `web/docs/src/content/docs/changelog.md`: dated entry naming 3.0.0; re-sync `web/docs/public/openapi.yaml` (`cd web/docs && node scripts/sync-openapi.mjs`); update the `lamps.md` web doc
- [ ] `docs/specs/db/erd.mmd`: `recipient` nullable; regenerate `erd.svg` (see Phase 3)
- [ ] Delete the legacy `docs/specs/features/lamp/SRS.md` (it carries a "Superseded by" banner) in the separate cleanup PR, after the user approves the file list

## Deferred (separate feature, not blocking lamp)

- Lighting a lamp at a temple for a fee — hidden on every platform (product §3.5), no spec.
- "My lamps" screen and history of expired lamps — no spec.
- Moderation of prayers and taking lamps down through the back office — no spec; if built, it goes through `services/backofficeService`.
- Cleanup job for expired lamps / retention policy — waits for the retention marker.
