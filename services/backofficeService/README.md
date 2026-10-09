# backofficeService

Admin/backoffice API for operating the innerlamp system. Same Clean Architecture
layout as `services/appService` (`domain` → `usecase` → `repository` / `delivery/http`,
wired in `cmd/main.go`). Shares the `innerlamp` Postgres with appService but is a
separate binary, port (`:8081`) and trust domain.

## Auth model

- Accounts live in the `admins` table (`migrations/0006_admins.sql`), **not** `users`. No self-registration.
- Admin tokens use their own JWT secret + audience (`innerlamp_backoffice`), so an appService token is rejected here and vice versa.
- Only access tokens are issued (no refresh/session yet). `/v1/admin/*` (except login) requires an `access` token **and** role `admin`.

## Bootstrap admin

On startup, if the `admins` table is **empty**, `bootstrap_admin` in the config (dev: `admin` / `admin`) is inserted with `must_change_password = TRUE` (atomic, safe with several instances). Its login token carries `must_change_password` and is refused everywhere except `POST /v1/admin/auth/change-password` (min 12 chars, must differ). Once any admin exists it never runs again. **Production: set `bootstrap_admin.enabled: false` and use `cmd/seedadmin`.**

## Routes

| Method | Path | Auth |
|---|---|---|
| GET | `/healthz` | public |
| POST | `/v1/admin/auth/login` | public |
| POST | `/v1/admin/auth/change-password` | admin (also accepts a must-change-password token) |
| GET | `/v1/admin/me` | admin (refused with `PASSWORD_CHANGE_REQUIRED` until the password is changed) |

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
- Real business routes (users, temples management).
