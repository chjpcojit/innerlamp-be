---
id: TD-002
feature: config
title: "Repository tests TRUNCATE the default dev database / Test repository TRUNCATE database dev mặc định"
severity: medium
status: open
created: 2026-10-11
owner: ducnguyen-wee
code: services/appService/repository/postgres_registration_repository_test.go (connectTestDB, truncateAll)
---

# TD-002 — Repository tests TRUNCATE the default dev database / Test repository TRUNCATE database dev mặc định

## 1. Debt / Nợ gì

**EN** — `connectTestDB` (used by every `repository/*_test.go`) connects to `localhost:5432/innerlamp` unless `TEST_DB_NAME` is set, applies migrations, and runs `truncateAll`: `TRUNCATE users, user_sessions, temples, deities, gallery_images, articles, sutras, dharma_talks, ... CASCADE`. That is the same database `make run` uses, so a plain `go test ./services/appService/repository/` (without `-short`) empties the developer's seeded content and accounts. It happened on 2026-10-11 while the administrative-units tests were being checked; the seedable tables were reloaded with the repo's seed commands, user accounts, sessions, memorials, lamps and genealogies were not recoverable.

**VI** — `connectTestDB` (dùng bởi mọi `repository/*_test.go`) nối `localhost:5432/innerlamp` trừ khi đặt `TEST_DB_NAME`, áp migration, rồi chạy `truncateAll`: `TRUNCATE users, user_sessions, temples, deities, gallery_images, articles, sutras, dharma_talks, ... CASCADE`. Đó cũng là database mà `make run` dùng, nên `go test ./services/appService/repository/` (không `-short`) làm rỗng dữ liệu nội dung và tài khoản dev. Việc này đã xảy ra ngày 2026-10-11 khi kiểm tra test của administrative-units; các bảng seed được đã nạp lại bằng lệnh seed của repo, còn tài khoản, phiên, sổ giỗ, đèn và gia phả thì không khôi phục được.

## 2. Why accepted / Vì sao chấp nhận

**EN** — It is the documented design: repository tests run "against a manually-run local Postgres" so the SQL is proven on a real engine. Nobody chose to point them at the data database; the default name simply is that database.

**VI** — Đây là thiết kế đã ghi: test repository chạy "trên Postgres local chạy tay" để chứng minh SQL trên engine thật. Không ai chủ ý trỏ chúng vào database chứa dữ liệu; tên mặc định tình cờ là database đó.

## 3. Risk if unpaid / Rủi ro nếu không trả

**EN** — Any contributor (human or AI) running the full test command loses local data silently. The administrative-units tests now refuse to truncate a database holding a real dataset, but every other table is still wiped.

**VI** — Bất kỳ ai (người hay AI) chạy lệnh test đầy đủ đều mất dữ liệu local mà không có cảnh báo. Test administrative-units nay từ chối xoá database đang chứa bộ dữ liệu thật, nhưng mọi bảng khác vẫn bị xoá.

## 4. Exit criteria / Điều kiện trả nợ

- [ ] Default test database name differs from the run database (for example `innerlamp_test`), created by `make db-up` / Mặc định tên database test khác database chạy (vd. `innerlamp_test`), do `make db-up` tạo
- [ ] `connectTestDB` refuses to truncate a database that is not named like a test database, unless explicitly forced / `connectTestDB` từ chối TRUNCATE database không có tên kiểu database test, trừ khi ép rõ ràng
- [ ] `docs/development/local-stack.md` says which database the tests use / `docs/development/local-stack.md` ghi rõ test dùng database nào

## 5. Estimate / Ước lượng

**EN** — 0.5 day.

**VI** — 0.5 ngày.
