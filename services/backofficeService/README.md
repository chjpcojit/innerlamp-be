# backofficeService

Admin/backoffice API for operating the innerlamp system. Same Clean Architecture
layout as `services/appService` (`domain` → `usecase` → `repository` / `delivery/http`,
wired in `cmd/main.go`). Shares the `innerlamp` Postgres with appService but is a
separate binary, port (`:8081`) and trust domain.

## Auth model

- Accounts live in the `admins` table (`migrations/0006_admins.sql`), **not** `users`. No self-registration.
- Admin tokens use their own JWT secret + audience (`innerlamp_backoffice`), so an appService token is rejected here and vice versa.
- Only access tokens are issued (no refresh/session yet). `/v1/admin/*` (except login) requires an `access` token and a role the route allows: `admin` for account management, `admin` or `content_editor` for the temple-suggestion review routes.

## Bootstrap admin

On startup, if the `admins` table is **empty**, `bootstrap_admin` in the config (dev: `admin` / `admin`) is inserted with `must_change_password = TRUE` (atomic, safe with several instances). Its login token carries `must_change_password` and is refused everywhere except `POST /v1/admin/auth/change-password` (min 12 chars, must differ). Once any admin exists it never runs again. **Production: set `bootstrap_admin.enabled: false` and use `cmd/seedadmin`.**

## Routes

| Method | Path | Auth |
|---|---|---|
| GET | `/healthz` | public |
| POST | `/v1/admin/auth/login` | public |
| POST | `/v1/admin/auth/change-password` | admin (also accepts a must-change-password token) |
| GET | `/v1/admin/me` | admin (refused with `PASSWORD_CHANGE_REQUIRED` until the password is changed) |
| GET | `/v1/admin/temple-suggestions` | admin or content_editor — review queue (`?status=&page=`) |
| GET | `/v1/admin/temple-suggestions/{id}` | admin or content_editor |
| POST | `/v1/admin/temple-suggestions/{id}/create-temple` | admin or content_editor (a `rejected` suggestion: admin only) |

## Run

```bash
make migrate                                   # applies 0006_admins.sql
ADMIN_PASSWORD='at-least-12-chars' go run ./services/backofficeService/cmd/seedadmin -email you@example.com -name "You"
make run-backoffice
```

## Not done yet (deliberately)

- Login brute-force lockout / rate limiting (appService has it for users; admins do not yet).
- Refresh tokens / sessions / logout, roles beyond `admin`, audit log.
- `⚠️ DEV ONLY` config/secrets (`config.dev.yaml`, `MockSecretGetter`) — swap for `pkg/secrets/gcp|aws` before production.
- Reject action for temple suggestions (`docs/specs/features/temple-suggestion`, Phase 6), editing or deleting an existing temple.
- Real business routes beyond users and temple-suggestion review.

## Tests

`go test ./services/backofficeService/...` runs unit tests; `-short` skips the rest. The repository tests
TRUNCATE `temples` and `temple_suggestions`, so they run only when `TEST_DB_NAME` names a scratch database
other than `innerlamp` (e.g. `TEST_DB_NAME=innerlamp_test`). `e2e/` builds the real binary and runs it
against a testcontainers Postgres (needs Docker).
