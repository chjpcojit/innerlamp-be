# appService

The one consuming service in this repo (module `chjpcoj.com.innerlamp`). Implements the InnerLamp auth/onboarding user stories from `docs/specs/onboarding/onboarding.md`, plus the `content` module user stories from `docs/specs/content/content.md`, on top of the shared `pkg/` framework.

## Layout (Clean Architecture)

```
cmd/         entrypoint — wiring only (constructs pkg/* clients, usecases, handlers, starts http.Server)
config/      AppConfig — loads YAML via pkg/config, resolves sensitive fields (DB password, JWT key) via pkg/secrets
domain/      entities + sentinel errors, no framework imports
usecase/     business logic, depends on domain + repository interfaces (defined in usecase/domain, not concrete drivers)
repository/  concrete data-access implementations (Postgres, GCS) implementing domain/usecase-facing interfaces
delivery/http/  HTTP handlers — parse request, call usecase, map result/error to response
```

Dependency direction: `delivery → usecase → domain`, `repository → domain`. `domain` imports nothing from the other three.

## What's implemented

| User story | Usecase | Handler | Route |
|---|---|---|---|
| US-001 Đăng ký | `RegisterUsecase` | `register_handler.go` | `POST /v1/auth/register` |
| US-002 Đăng nhập | `LoginUsecase` | `login_handler.go` | `POST /v1/auth/login` |
| US-003 Quên mật khẩu | `PasswordResetUsecase` | `forgot_password_handler.go`, `reset_password_handler.go` | `POST /v1/auth/forgot-password`, `POST /v1/auth/reset-password` |
| US-004 Hồ sơ cá nhân | `ProfileUsecase`, `AvatarUsecase` | `profile_handler.go`, `avatar_handler.go` | `GET/PATCH /v1/users/me`, `POST /v1/users/me/avatar` |
| US-005 Xem danh sách kinh điển | `ListSutrasUsecase` | `sutras_handler.go` | `GET /v1/content/sutras` (public, no auth) |
| US-006 Xem chi tiết một bài kinh | `GetSutraUsecase` | `sutra_detail_handler.go` | `GET /v1/content/sutras/{slug}` (public, no auth) |
| US-007 Xem danh sách thần chú | `ListMantrasUsecase` | `mantras_handler.go` | `GET /v1/content/mantras` (public, no auth) |
| US-008 Xem chi tiết một thần chú | `GetMantraUsecase` | `mantra_detail_handler.go` | `GET /v1/content/mantras/{slug}` (public, no auth) |
| US-009 Xem danh sách Chư Phật & Bồ Tát | `ListBodhisattvasUsecase` | `bodhisattvas_handler.go` | `GET /v1/content/bodhisattvas` (public, no auth) |
| US-010 Xem chi tiết một vị Phật/Bồ Tát | `GetBodhisattvaUsecase` | `bodhisattva_detail_handler.go` | `GET /v1/content/bodhisattvas/{slug}` (public, no auth; response embeds related `mantras`) |

Schema for these lives in `../../migrations/` (see `migrations/README.md`; `sutras`/`mantras`/`bodhisattvas` schema also documented in `docs/db/content-erd.md`). Every domain/usecase file cross-references the user story + AC it implements in its doc comments — read those before extending a flow instead of re-deriving requirements here.

## Known gaps in `cmd/main.go`

Wiring is real (register through avatar upload run end-to-end against Postgres + GCS). `main.go` now loads non-sensitive settings from `config/config.dev.yaml` via `config.Load` (this package's `AppConfig`), and resolves `database.password_secret`/`jwt.secret_key_secret` via `config.MockSecretGetter` (`mock_secrets.go`) — a **dev-only** resolver that reads plaintext values from the YAML's `dev_secrets` section instead of a real secret manager. Set `CONFIG_PATH` to override the default file (also accepts `s3://`/`gs://` sources, per `pkg/config`). Before deploying: point `CONFIG_PATH` at a real config source and swap `MockSecretGetter` for a `pkg/secrets` provider (`pkg/secrets/gcp` or `pkg/secrets/aws`) — search `main.go` for `⚠️ DEV ONLY`. `mailer.NewLogMailer` (logs instead of sending) must also be swapped for a real provider.

## Notes

Update this README when a layer's responsibility changes or a new user story lands — it should always match the table above, not what onboarding.md originally scoped.
