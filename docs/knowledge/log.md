# Knowledge Bundle Update Log

## 2026-10-15

- Deleted `features/register.md` and `features/login.md` and removed them from `index.md`.
  - Reason: registration and sign-in move from phone + SMS to email + email code (ADR-0003, Accepted). The
    product specs (`docs/product/{onboarding,login}/product.md` @ v1.2) list both concepts as prior documents to
    delete once every platform spec was approved (Android, the last one, merged 2026-10-15).
  - Links to them in `packages/{ratelimit,middleware}.md`, `api/calendar-events.md` and
    `features/{deity,genealogy,attestation,temple,calendar,article,sutra}.md` now name the specs
    `docs/specs/features/login/` and `docs/specs/features/onboarding/` instead.
  - New concepts are written with `il-okf-writer` once the email flow is built and verified
    (`docs/specs/features/{login,onboarding}/tasks.md`).

## 2026-10-14

- Created `packages/ratelimit.md` (`status: draft`) and added it to `index.md`.
  - Documents `pkg/ratelimit` (stores, middleware, the new `FallbackStore`), how appService builds one shared
    store from `rate_limit_store` (Redis in dev, from `docker-compose.yml`), every route that has a request-rate
    limit with its key, budget and scope, the Postgres-backed OTP/login lockouts, and the routes with no limit.
  - Facts read from `pkg/ratelimit/{ratelimit.go,store/*.go,middleware/*.go}`,
    `services/appService/delivery/http/{client_ip_rate_limit,user_rate_limit,rate_limit_options}.go`,
    `services/appService/cmd/{main.go,ratelimit_store.go,profile_routes.go,temple_announcement_routes.go}`,
    `services/appService/config/{config.go,config.dev.yaml}` and `usecase/{otp_service,login_usecase}.go`; each file
    named in the concept was checked to exist.
  - Verification behind it: `go build ./...`, `go vet ./...`, `go test -short ./...` (only the pre-existing
    backofficeService config tests fail, for a missing untracked `config.dev.yaml`); the Redis store tests ran
    against a real Redis 7.0; the real binary ran against Postgres 16 + Redis (429 at the limit, two instances
    sharing one count, fallback to a per-instance count with Redis stopped, startup stopping without Redis).
  - Kept `draft`: the review's proposed fixes (per-IP limits on auth, attestation and admin-login routes, atomic
    lockouts, trusted-hop count behind a load balancer) are not implemented, the E2E suite uses the memory store,
    and Redis has no TLS (TD-001).
  - `verified:` was not present and was not added. Unlike most entries, this change also touched `.go` files
    (the Redis wiring itself); the concept was written after that code was built and run.

## 2026-10-11 (4)

- Created `features/calendar.md` (`status: draft`), `api/calendar-events.md` (`status: stable`) and
  `migrations/0031-calendar-events.md` (`status: stable`); added them to `index.md` and moved `api/` and `migrations/`
  from "Planned, not yet populated" to the active sections.
  - `features/calendar.md` documents both halves of the module as built: the image of the day (US-CAL-001) and the
    events (US-CAL-002) — repeat rules (`solar`, `lunar`, `lunar_monthly`, `lunar_last_day`, `solar_term`) resolved per
    year by `domain/lunarcal` and `domain.ResolveOccurrences`, the `seedcalendarevents` command, and the config block.
    The pure `domain/lunarcal` package is described inside the feature rather than as a `Package` concept because it
    lives under `domain/`, not `pkg/`.
  - Facts read from `domain/lunarcal/*.go`, `domain/calendar_event*.go`, `domain/calendar_catalogue.go`,
    `usecase/calendar_event_usecase.go`, `repository/postgres_calendar_event_repository.go`,
    `delivery/http/calendar_event_handler.go` + `errors.go`, `cmd/seedcalendarevents/*`, the calendar blocks of
    `cmd/main.go` and `config/config.go`, and `migrations/0031_calendar_events.sql`; each file named in the tables was
    checked to exist.
  - Verification behind it: `go build ./...`, `go vet ./...` and `go test -short ./...` clean; the repository tests pass
    with `TEST_DB_NAME=innerlamp_test`; the full E2E suite (473s) passes, including
    `TestCalendarEventsFlow_E2E` on the real binary and a fresh Postgres 16; the route was also called with `curl`
    against the dev server. The migration was applied twice and its CHECK/FK behaviour tried with valid and invalid rows.
  - Kept the feature `draft`: the catalogue content is unreviewed, E20 lacks a full independent reference table, event
    images are `null`, and the seed is dev-only. Marked the endpoint and the migration `stable` because their contract
    and schema were exercised on a real database; the endpoint concept says the returned content is still a draft.
  - Recorded a defect found on the way: `usecase/genealogy_lunar.go` returns day `0` on 2054-05-07 and 2062-04-09
    (listed under the feature's Pending work).
  - `verified:` was not present and was not added. No `.go` file was changed by this entry.

## 2026-10-11 (3)

- Updated `features/genealogy.md` (still `status: draft`).
  - The last open spec question is settled: the profile returns both `age` (Western) and `traditional_age`
    (tuổi mụ = Western + 1). Facts re-read from `usecase/genealogy_kinship.go` (`TraditionalAge`),
    `usecase/genealogy_profile_usecase.go` and `delivery/http/genealogy_member_handler.go`.
  - Pending work no longer lists open spec questions; the concept stays `draft` for the reasons already listed
    (no invitation notification, dev OTP/SMS behind the E2E accounts, NFR targets not load-tested).
  - `verified:` was not present and was not added.

## 2026-10-11 (2)

- Updated `features/genealogy.md` (still `status: draft`).
  - Pending work: seven of the eight small spec questions were settled by the product owner (re-invite after decline,
    reassigned phone numbers, sibling order, viewer without a position, optional mother, no kind conversion,
    last-write-wins); only the age display (Western vs. traditional) remains open. No code behaviour changed.
  - Added a consequence worth knowing: because the client picks only "child of the father", `other_parent_id` is
    usually empty and the profile then has no `mother` and the "Mẹ" kinship term is `UNKNOWN`.
  - `verified:` was not present and was not added.

## 2026-10-11

- Created `features/genealogy.md` (`status: draft`).
  - Documents the genealogy feature as built: two roles (`owner` edits, `viewer` reads), the invitation journey
    (owner picks a living member and invites by phone → inbox → accept/decline → status visible to the owner),
    the "me" position matched on `phone_number`, the computed profile (age, relatives, lineage, chi, kinship),
    solar storage with a lunar flag and server-derived lunar date, and the 21 routes.
  - Facts read from `domain/genealogy.go`, `usecase/genealogy_*.go`, `repository/postgres_genealogy_*.go`,
    `delivery/http/genealogy_*.go` + `errors.go`, the genealogy block of `cmd/main.go`, `config.GenealogyConfig`,
    `migrations/0028_genealogy.sql` and `e2e/genealogy_e2e_test.go`; each file named in the Architecture table was
    checked to exist.
  - Verification behind it: `go build ./...`, `go vet ./...` and `go test -short ./...` clean for the whole repo;
    the repository tests and `TestGenealogy*` E2E pass on Postgres 16 / Docker; the acceptance map is in
    `docs/specs/features/genealogy/ac-audit.md` (AC1–AC82, no AC without a test).
  - Kept `draft`: invitees are only discoverable by polling the inbox (no push, no SMS), the E2E registers accounts
    through the dev OTP and log-only SMS, a few spec questions are open with defaults, and NFR-001/002 were never
    load-tested. Listed under Pending work.
  - `verified:` was not present and was not added. No `.go` file or `docs/specs/` file was changed by this entry.

## 2026-10-10

- Updated `features/register.md` (still `status: draft`).
  - Documented onboarding v7.0: `ordination_place` (text) was replaced by `ordination_temple_id`
    (FK `temples`), checked via `RegistrationRepository.TempleExists` after the conflict checks →
    `400 INVALID_TEMPLE`; migration `0023`; `user.ordination_temple_id` in the verify-otp response.
    Facts read from `domain/`, `usecase/register_usecase.go`, `repository/postgres_registration_repository.go`,
    `delivery/http/register_handler.go` + `errors.go` and `migrations/0023_*.sql`.
  - Fixed drift unrelated to v7.0 found while re-reading the code: the generic OTP work now lives in
    `usecase/otp_service.go` (`OTPService`) and `OTPRepository` (the old `ReissueOTP`/`FindActiveOTP`
    on `RegistrationRepository` and `generateOTP()` in `register_usecase.go` no longer exist there);
    the three routes sit behind `RequireWebAPIKey` (and `NoStore` for verify-otp), and the verify-otp
    body carries `expires_in`. Rewrote the Request flow, Architecture rows and Pending work accordingly.
  - Kept `draft`: OTP is still the constant `111111`, SMS is still log-only, and the US-001 ACs have not
    been through the `golang-e2e-verification` audit. Verification is the 2026-10-10 run listed in the
    concept (a full parallel run of the `e2e` package had 4 timeouts that pass individually).
  - `verified:` was not present and was not added.

## 2026-10-06

- Updated `features/article.md` (still `status: draft`).
  - Image data: all 60 seed articles now have an `imageURL` (38 `og:image`, 15 in-page images, 7 Wikimedia Commons illustrations, assumption A-001). Replaced the "38 of 60 / 22 null" statements; noted that already-seeded rows need `seedarticles -fill-images`.
  - Did not re-run the DB seed or any Postgres-backed tests.

## 2026-10-03 (2)

- Updated `features/sutra.md` (still `status: draft`).
  - Re-checked against the code and spec: `cmd/main.go` wiring and the four routes, the tasks list (Phase 1–5 done) and the data (153 files, 10 with `durationMinutes: null`). Spec version reference moved from v2.8 to v2.10.
  - `duration_minutes`: the owner chose to keep the generated values, so the Phase 0 hand-review is no longer pending work; the concept now states them as an editorial estimate and keeps the Dược Sư / very-long-text skew as a known limit (TD-007, A-005).
  - Did not re-run `go build` / `go test`; `status` stays `draft` because the content is unreviewed, `assets/` and `openapi.yaml` are not under git and licences are open.
- `index.md`: refreshed the sutra row.

## 2026-10-03

- Added `features/sutra.md` (`type: Feature`, `status: draft`).
  - Read the slice against the code: `migrations/0013_sutras.sql`, `domain/sutra.go`,
    `usecase/sutra_usecase.go`, `repository/postgres_sutra_repository.go` and `sutra_seed.go`,
    `delivery/http/sutra_handler.go` and `errors.go`, the `sutra:` config block, the wiring and
    `sutraSeedTimeout` in `cmd/main.go`, `cmd/seedsutras` and `assets/embed.go`.
  - Facts come from the spec v2.8 audit (`test-coverage-report.md`): `e2e/sutra_e2e_test.go` runs the real
    binary on Postgres (4 routes, first-run seed, second start, missing deities, invalid config) and every
    E2E test was shown able to fail by deliberate mutation. Counts (153 / 391 / 64,009 / 151) match the E2E
    check of `index.json`.
  - Marked `draft`: the code is exercised end to end, but the religious content is unreviewed, the dataset and
    OpenAPI file are not under version control, and licences are open. Debt is in `docs/tech-debt/sutra/`
    (TD-001…TD-010) and assumptions in `docs/assumptions/sutra/` (A-001…A-008).
- `index.md`: added the sutra row.

## 2026-09-30 (3)

- Updated `features/deity.md` for the paged list change: `GET /v1/bodhisattvas` (bodhisattvas only) was
  replaced by `GET /v1/deities?page=N`, which lists Buddhas and bodhisattvas together with no type filter.
  - Re-read the code: `delivery/http/deity_handler.go` (`ListDeities`), `usecase/deity_usecase.go`
    (`ListDeities`), `domain/deity.go` and `repository/postgres_deity_repository.go` (`ListByType` →
    `List`, count and page without `WHERE type`), and the route in `cmd/main.go`.
  - Ran `go build ./...`, `go vet ./services/appService/...` and `go test -short ./services/appService/...`: pass.
    Did **not** call the route on a running server, and the Postgres-backed repository tests are skipped
    under `-short`, so the concept stays `draft`.
  - Removed the "no Buddha list route" pending item; added the note that the old route is gone with no alias
    (breaking for iOS) and that `idx_deities_type_order` is no longer used by any query.
- `index.md`: refreshed the deity row.

## 2026-09-30 (2)

- Added `features/temple.md` and `features/deity.md` (`type: Feature`, `status: draft` both) and
  re-read `features/article.md` against the code.
  - Read each slice against the code: migrations (`0005`, `0011`, `0007`), `domain/{temple,deity}.go`,
    the usecases, the Postgres repositories, the handlers, `delivery/http/errors.go`, the
    `temple:`/`deity:` config blocks, the route wiring in `cmd/main.go` and the `seedtemples` /
    `seeddeities` commands.
  - Called the routes on the running dev server: temples `top`, `page=10` (empty page, `total: 179`),
    `page=0` (`400 INVALID_PAGE`), detail and unknown id (`404 TEMPLE_NOT_FOUND`); deities `top`,
    bodhisattvas page 1 and `page=abc`, unknown id (`404 DEITY_NOT_FOUND`). Counts checked against the
    data files and the dev DB: 179 temples, 18 deities (6 `BUDDHA`, 12 `BODHISATTVA`), 60 articles.
  - Ran `go test -short` for `usecase`, `delivery/http`, `repository`, `cmd/seedtemples` and
    `cmd/seeddeities`: all pass (cached results; the repository DB tests are skipped under `-short`).
  - Found and recorded: no temple has an image; every deity `image_url` is a `cdn.yourapp.com`
    placeholder and every `thumbnail_url` a relative path nothing serves; there is no Buddha list
    route; temples and deities are seeded only by a manual command on dev config, not at startup.
  - Marked both `draft`: no E2E test exists for either catalogue, and the seeding path is dev-only.
- Corrected `features/article.md`, which was stale in three places: it said there was no reseed
  command (`cmd/seedarticles` exists, with `InsertMissing` and `-fill-images` → `FillMissingImages`),
  and it said `openapi.yaml` and `erd.mmd` lack the article routes/table (both now list them).
  Status stays `draft`.
- `index.md`: added the two new rows and refreshed the article row.

## 2026-09-30

- Updated `features/article.md` for the article images: `image_url` is now set for 38 of 60 articles
  (source page `og:image`, hotlinked) and the seed file holds 60 articles, not 30. Counts checked against
  `assets/data/buddhist_articles.json` and the dev DB (`count(*)=60`, `count(image_url)=38`). Status stays `draft`.

- Documented the article catalogue and added `features/article.md` (`type: Feature`, `status: draft`),
  written from the spec `docs/specs/features/article/` created the same day.
  - Read the whole vertical slice against the code: `migrations/0012_articles.sql`, `assets/embed.go`,
    `repository/article_seed.go`, `domain/article.go`, `usecase/article_usecase.go`,
    `repository/postgres_article_repository.go`, `delivery/http/article_handler.go`,
    `delivery/http/errors.go`, the `article:` config block and the route/seed wiring in `cmd/main.go`.
  - Called the three routes on the running dev server (top returned 5 items, `page=4` returned an
    empty page with `total: 30`, `page=0` returned `400 INVALID_PAGE`, an unknown id returned
    `404 ARTICLE_NOT_FOUND`) and ran `go test -short` for `usecase`, `repository` and
    `delivery/http`. The first-run seed itself was not observed by this author.
  - Corrected a claim from the hand-off: `published_date` is `null` for half the articles (15 of 30
    have a date), not most; `image_url` is `null` for all 30.
  - Marked `draft` because no automated end-to-end test covers the routes or the seed, the API is
    not in `openapi.yaml`, and the repository tests need Postgres (they `TRUNCATE` the dev tables)
    and were not re-run. Pending work in the concept lists exactly what promotes it to `stable`.
- Linked `features/article.md` in `index.md`'s concept table.

## 2026-09-25 (2)

- Added `packages/attestation.md` (`type: Package`, `status: draft`) for `pkg/attestation`, the
  provider-agnostic verification package the attestation feature consumes.
  - Read the root (`pkg/attestation/attestation.go`) and all three drivers (`mock`, `playintegrity`,
    `appattest`) plus their tests, and confirmed the registry shape: `Verifier` interface +
    `Register(ProviderType, FactoryFunc)` / `New(Config)` backed by a package-level map, drivers
    self-registering in `init()`, the same pattern as `pkg/sms`/`pkg/database`.
  - Established the dependency graph from the real imports: the root is a **stdlib-only leaf** (no
    internal `pkg/` imports); the drivers import only their parent + external deps
    (`google.golang.org/api/playintegrity/v1` for playintegrity, `github.com/fxamacker/cbor/v2` for
    appattest). No `pkg/utils`/`pkg/logging` dependency, unlike `middleware`/`sms`.
  - Marked `draft` because the package is **not a pure library**: `playintegrity.VerifyAndroid` does
    the untested Google `DecodeIntegrityToken` call (only `mapPayload` is unit-tested), and no
    deployment selects the real drivers — `services/appService` wires `mock` for both platforms. The
    "own test suite passing" shortcut to `stable` therefore does not apply.
  - `index.md`: appended the `packages/attestation.md` row.
  - `features/attestation.md`: rewrote the Related-concepts bullet that previously said "no Package
    concept exists yet" into a link to the new file, and corrected its Pending-work list accordingly.
- This closes the scope question from the previous entry: the feature and package concepts are now
  both written.

## 2026-09-25

- Documented US-006 and added `features/attestation.md` (`type: Feature`, `status: draft`). This is
  the concept the spec's `changelog.md` has been asking for since v1.0 ("concept `attestation` cần
  `il-okf-writer` tạo sau khi flow chạy end-to-end").
  - Read the whole vertical slice against the code: `domain/attestation.go` (entities, enums, the
    `AttestationRepository` + `AttestationVerifier` ports, `Reason*` constants), `domain/errors.go`
    (the five attestation sentinels), `usecase/attestation_usecase.go`, `repository/
    postgres_attestation_repository.go` + `attestation_verifier.go` (adapter), `delivery/http/
    attestation_handler.go` + `attestation_middleware.go`, `cmd/main.go` wiring, `config/config.go`
    + `config.dev.yaml` (`attestation:` block), `pkg/attestation/*` (interface + `mock` /
    `playintegrity` / `appattest` drivers), `migrations/0004_attestation.sql`, and the
    `docs/specs/features/attestation/` spec (requirements/design/tasks/changelog).
  - Marked `draft` because the flow has only been exercised with the **mock** driver: both
    `attestation.android.provider` and `attestation.ios.provider` are `"mock"` in `config.dev.yaml`
    (⚠️ DEV ONLY), the real `playintegrity`/`appattest` drivers are implemented and unit-tested but
    never wired or run, and `RequireAttestation` is built but deliberately not attached to any route
    (gate policy is a deferred business decision). There is also no automated E2E test for
    attestation (`services/appService/e2e/` has only register/login). Promote to `stable` once the
    flow runs end-to-end against the real drivers / a staging backend and the ACs are audited.
  - Recorded the deferred items (fraud-risk receipt, `attestation_challenges` cleanup, iOS key
    lifecycle) and the still-open "⚠️ Not confirmed" design question (`environment=development`
    rejection) in Pending work / Known gaps.
  - `index.md`: appended the `features/attestation.md` row to the concept table.
- Scope note: wrote the **Feature** concept only. `pkg/attestation` (interface + 3 drivers, the
  provider-agnostic registry pattern shared with `pkg/sms`) deserves its own `packages/attestation.md`
  `Package` concept — flagged in the concept's Related concepts as not yet written.

## 2026-09-22 (4)

- Added `packages/middleware.md` (`type: Package`, `status: draft`) for `pkg/middleware`, which
  became wired into the server by `http-middleware-wiring` (implemented 2026-09-22). This populates
  the `packages/` section for the first time.
  - Documented the real HTTP surface (`Logger`, `CORS`, `Recovery`) and the gRPC interceptors, the
    `pkg/utils`-only internal dependency, and the consumer chain
    `Recovery(CORS(Logger(WithAPIFallback(mux))))` built by `newHandler` in
    `services/appService/cmd/main.go`, including why the order is load-bearing (Recovery catches
    every layer; CORS answers preflight before Logger, so preflight is not logged).
  - Marked `draft` because the chain and the new preflight envelope are proven by unit tests
    (`pkg/middleware/middleware_test.go`, `services/appService/cmd/main_test.go`) but were **not**
    exercised on a running binary + Postgres (the curl checks in the spec's Phase 3 are unrun);
    the gRPC `Recovery`/`StreamRecovery` interceptors are untested and unused. Promote to `stable`
    once a real server is exercised.
  - `index.md`: moved `packages/` from the "Planned" list into the active sections and added the
    concept row.
- Fixed drift the change created in the two feature concepts, after re-reading `cmd/main.go`:
  - `features/register.md`: its Wiring row still described `Recovery(WithAPIFallback(mux))`; rewritten
    to `newHandler(mux)` = `Recovery(CORS(Logger(WithAPIFallback(mux))))`, the "missing-middleware"
    sentence clarified to mean auth only, and a Related-concepts link added.
  - `features/login.md`: Related-concepts link added (its Wiring row was not wrong, so left as is).
  - Both `generated.at` timestamps refreshed; both keep their existing `status` (`register` draft,
    `login` stable) since the request/response status and bodies they document did not change.
- Sources read to write it: `pkg/middleware/{middleware,grpc}.go`, `pkg/middleware/README.md`,
  `pkg/middleware/middleware_test.go`, `services/appService/cmd/main.go` +
  `main_test.go`, and the `http-middleware-wiring` / `api-response-envelope` specs. No `resource:`
  path is guessed; `pkg/middleware/` was confirmed on disk and no `verified:` field was written.

## 2026-09-22 (3)

- Removed the `docs/ARCHITECTURE.md` reference from the bundle. That file has been deleted repo-wide
  (it was stale: §3–§4 still described removed US-003…US-010 work).
  - `index.md`: the introduction no longer points at it; it now tells the reader to use the code and
    the per-package/layer `README`s and to verify relationships from real imports.
  - `features/register.md` and `features/login.md`: dropped it from `sources:` and removed the body
    references (§4 gap list, §3.2 story table).
  - Also refreshed `features/register.md`'s "Known gaps vs. design" to cite onboarding v5.2 and
    corrected `appConfig.MockSecretGetter` → `appConfig.NewMockSecretGetterFromYAML` in `login.md`.
- Repo-wide, `AGENTS.md`, `CLAUDE.md` and the root `README.md` were updated to the current tree and
  no longer reference the deleted file.

## 2026-09-22 (2)

- Fixed two factual slips found in a full bundle-vs-code audit:
  - The send-cap wording in `features/register.md` (Summary + "Send policy") said "the fourth
    counted send" — only three sends are counted. The fourth send overall (the third counted) is
    still delivered and sets the 30-minute `otp_send_limits.locked_until`; the fifth is refused.
    The same wording is corrected in this log's `## 2026-09-22` entry below.
  - `features/register.md`'s Domain/Errors row omitted the input-validation sentinels
    (`ErrInvalidPhone`, `ErrInvalidEmail`, `ErrInvalidPassword`) and `ErrUnsupportedOTPPurpose`.
- Reworded the `index.md` introduction: `../ARCHITECTURE.md` §2.2 is the verified `pkg/` import
  graph, but §3–§4 have drifted, so the bundle no longer calls the whole file the source of truth.
- Corrected this log's `## 2026-09-21` entry: the login lock is recorded by the 5th consecutive
  failure (the 6th attempt is the one rejected), not "lockout on the 6th attempt".

## 2026-09-22

- Updated `features/register.md` after onboarding v5.0 (Phase 9) + v5.1 (Phase 10), re-reading the
  whole vertical slice against the code. What changed:
  - The send cap is no longer `otp_codes.resend_count` carried forward; it is a per-phone
    `otp_send_limits` row (`applySendLimits`, migration `0003`). One attempt gets four sends (first
    free + three counted), the 4th send (the 3rd counted) is still delivered and sets a 30-minute
    `locked_until`, and `request-otp`
    spends from the same counter as `resend-otp`. Removed the stale "resend_count never resets / SMS
    cap pending a decision" item — that was the bug v5.0 fixed.
  - Documented the two locks as independent despite both being 30 minutes: the send lock
    (`otp_send_limits.locked_until`) must not block `verify-otp`, while the wrong-guess lock
    (`otp_codes.locked_until`) does.
  - `pending_registration_ttl` is now 24h (was 90s). A pending row for the same phone no longer
    blocks `request-otp` (`FindConflicts` checks only `users` for the phone, plus another phone's
    live pending for the email); `resend-otp` issues a new code even when the previous code expired,
    and leaves the payload untouched. Added AC15 to the summary.
  - Refreshed the config table (`send_cooldown`/`max_send_attempts`/`send_lock_duration`,
    `lock_duration` 30m, `pending_registration_ttl` 24h), the Architecture table
    (`domain/otp_send_limit.go`, `ErrOTPSendLimited` replacing `ErrOTPResendExhausted`, five tables),
    all three Request-flow diagrams, the Validation rules (AC6), and Verification.
  - Added Verification for the three 2026-09-22 E2E tests and the new repository tests.
- Status stays `draft`: the OTP (`"111111"`) and SMS (`pkg/sms/logsms`) placeholders remain, the
  cleanup job is still deferred (now more load-bearing because the payload lives 24h), and the US-001
  ACs have not been audited end-to-end. Promote to `stable` once (a) the placeholders are replaced
  and (b) `golang-e2e-verification` maps the ACs to tests.
- Updated the `features/register.md` row in `index.md`.

## 2026-09-21 (5)

- Updated `features/register.md` after onboarding v4.5 (Phase 8): `FindConflicts` now counts only
  unexpired `pending_registration` rows, so an expired registration restarts via `request-otp`
  instead of returning `409` forever.
- Removed the corresponding Known gap and replaced the "expired rows are not inert" pending item.
- Added the new E2E test `register_e2e_test.go` to Verification, and noted the unresolved
  `resend_count` reset / SMS cap decision.
- Status stays `draft`: OTP/SMS placeholders remain, and the US-001 ACs are not fully audited.

## 2026-09-21 (4)

- Re-verified `features/register.md` and `features/login.md` line by line against the code.
  Both stay at their current `status`.
- `register.md` corrections:
  - A malformed `otp_code` is a `400` `VALIDATION_ERROR` from the struct tag. It does not reach
    the usecase and does not count as an attempt (the concept previously said it became
    `OTP_INVALID`).
  - Added the missing steps: `NormalizePhone` in resend/verify; no active OTP → `OTP_EXPIRED`;
    `ReissueOTP` extends `pending_registration.expires_at`; `SavePendingAndIssueOTP` carries
    `resend_count` forward; `LockOTP` also sets `used_at`.
  - Added the status/code on every branch, the `otp:` config values, and the phone-before-email
    conflict order.
- New Known gap in `register.md`, reproduced on the real binary: an expired `pending_registration`
  row is never deleted, and `FindConflicts` ignores `expires_at`. An abandoned registration
  therefore blocks its phone and email with `409` `PHONE_TAKEN` indefinitely, contradicting
  `design.md` flow 3 Step 2 and AC12's "làm lại từ bước 1". This also corrects the earlier claim
  (in this bundle and in the onboarding Deferred note) that expired rows are inert.
- `login.md` corrections: the lock is set on the 5th failure (not "6th failure"); documented the
  `RecordLoginFailure` reset rules, `FindSessionByRefreshHash` matching current or previous hash,
  and the `RotateSession` re-check.

## 2026-09-21 (3)

- Refreshed `features/register.md` from the code (`status: draft`, unchanged). Drift fixed:
  - `repository` does not import `usecase`.
  - `TokenIssuer` is now `func(userID, sessionID)` and the `verify-otp` access token carries `sid`.
  - `password` has a `required` tag.
  - `verify-otp` answers `201`.
  - The dev secret resolver is `NewMockSecretGetterFromYAML`.
  - Envelope, fallback and recovery wiring added.
- Added Verification: registration repository tests ran against Postgres and pass, and the E2E
  harness exercises `request-otp`/`verify-otp` as login setup.
- Dropped the resolved items: the `pkg/utils` failures are fixed, and the cleanup job was accepted
  for MVP.
- Linked [Login](features/login.md).
- Still `draft`: OTP/SMS are dev placeholders, and the US-001 ACs have no E2E audit yet.

## 2026-09-21 (2)

- Promoted `features/login.md` from `draft` to `stable`. The condition
  `docs/specs/features/login/requirements.md` set is now met: the `golang-e2e-verification` AC
  audit is done. `services/appService/e2e/login_e2e_test.go` runs the real binary against
  testcontainers Postgres and passes for AC1–AC15 (see
  `docs/specs/features/login/test-coverage-report.md`).
- In the concept: added a Verification bullet for the E2E test and the middleware-removal check,
  dropped the "no automated E2E" pending item, and updated the index row.

## 2026-09-21

- Documented US-002 (login, refresh, logout) and added `features/login.md` (`status: draft`).
  Checked against `domain/login.go`, `domain/session_token.go`, `usecase/login_usecase.go`,
  `repository/postgres_login_repository.go`, `delivery/http/login_handler.go` + `errors.go`,
  `cmd/main.go` wiring, and `migrations/0002_login_devices_and_rate_limit.sql`.
- Kept `draft` because `docs/specs/features/login/requirements.md` requires it until the AC audit
  (`golang-e2e-verification`) is done. The whole flow was run end-to-end in this pass against the real binary
  and Postgres 16 (Docker `innerlamp-pg`), using curl:
  - login and refresh, including rotation and reuse-revoke
  - same-device supersede
  - lockout: the 5th consecutive failure records the 15-minute lock, so the 6th attempt is rejected
    with `429 LOGIN_LOCKED`
  - logout returning `200` `{"success": true}`, idempotent
  - refresh after logout, and logout without a token, both rejected
- `TestLoginRepository_*` also ran against that database (not skipped) and passed.
- Still pending, listed in the concept: an automated E2E audit (`golang-e2e-verification`,
  blocked on the `testcontainers-go` decision), a cleanup job for `login_attempts`/sessions, and
  missing folder READMEs.
- Added the concept to `index.md`, and removed stray `****` markers from its heading and from the
  `api/` bullet.

## 2026-09-17

- Documented US-001's registration flow (`request-otp`/`resend-otp`/`verify-otp`) and added
  `features/register.md` (`status: draft`). Marked `draft` because `usecase/register_usecase.go`'s
  `generateOTP()` returns a hardcoded `"111111"` instead of a random code, `cmd/main.go` wires
  `pkg/sms/logsms` (logs the OTP rather than sending it) and `appConfig.MockSecretGetter` (both
  `⚠️ DEV ONLY`/`⚠️ DEV CONFIG`), there is no cleanup job for expired `pending_registration`/
  `otp_codes` rows, and this pass did not itself run `postgres_registration_repository_test.go`'s
  integration test against a live Postgres to confirm the end-to-end claim
  `docs/specs/features/onboarding/design.md` (v3.3) makes. Promote to `stable` once the OTP/SMS
  placeholders are replaced with production equivalents and the flow is independently re-exercised
  end-to-end against a real/staging Postgres.
- Created the `features/` section entry and the bundle intro/sections list in `index.md` (both
  files were zero-byte before this pass) and added `features/register.md` to its concept table.
