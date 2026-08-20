# migrations

Postgres SQL migrations for `services/appService`, run with `golang-migrate` via `scripts/migrate.sh`.

## Files

| File | User story |
|---|---|
| `0001_register_onboarding.sql` | US-001 Đăng ký — adds `phone_number`, `ordination_place` to `users`; creates `user_devices`, `user_sessions` |
| `0002_login_device_unique.sql` | US-002 Đăng nhập — uniqueness constraint on `user_devices(user_id, device_id)` so repeat logins upsert instead of duplicating |
| `0003_password_reset.sql` | US-003 Quên mật khẩu — password-reset token table |
| `0004_profile_avatar.sql` | US-004 Hồ sơ cá nhân — adds `avatar_url` to `users` |
| `0005_content_sutras.sql` | US-005 Xem danh sách kinh điển — creates `sutras` |
| `0006_content_bodhisattvas_mantras.sql` | US-007/US-008/US-009/US-010 — creates `bodhisattvas`, `mantras` |

Numbered sequentially, one migration per user story/AC batch — not per PR. Each file's header comment states the assumptions it makes about pre-existing schema (e.g. `0001` assumes `users` already has `id`, `email UNIQUE`, `password_hash`, `dharma_name`); read that comment before editing an early migration, since later migrations depend on those assumptions holding.

## Running

```bash
export DATABASE_URL=postgres://innerlamp:innerlamp@localhost:5432/innerlamp_dev?sslmode=disable
./scripts/migrate.sh up          # apply all pending
./scripts/migrate.sh down [N]    # rollback N steps (default 1)
./scripts/migrate.sh create NAME # scaffold a new migration pair
```

See `scripts/migrate.sh` header comment for full env var reference. `scripts/check-migrations.sh` and `scripts/cleanup-obsolete-migrations.sh` validate/maintain this directory as part of the repo's check scripts (`scripts/README.md`).

## Notes

Add a row to the table above for every new migration file — don't let this list drift from what's actually in the directory.
