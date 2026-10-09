---
id: TD-009
feature: sutra
title: "Behaviours the tests cannot reach / Các hành vi mà test chưa với tới"
severity: low
status: open
created: 2026-10-03
owner: ducnguyen-wee
code: services/appService/e2e/sutra_e2e_test.go; docs/specs/features/sutra/test-coverage-report.md
---

# TD-009 — Behaviours the tests cannot reach / Các hành vi mà test chưa với tới

## 1. Debt / Nợ gì

**EN** — (1) `500` without leaking the cause on the read routes is proven only at handler level. (2) The main binary aborting on invalid embedded data is covered through the parser and `cmd/seedsutras -dir`, not end to end. (3) The `seeded sutras on first run` log line is not asserted. (4) Two instances starting at the same moment are not tested (only "the row already exists"). (5) `index.json` duplicates per-file metadata and drift between them is caught only by the Docker-based E2E, not by `-short`. (6) No load or concurrent-read test; response times were measured once on loopback. (7) The repository and E2E tests need Postgres/Docker and are skipped under `-short`; repository tests `TRUNCATE` the dev database tables (existing pattern).

**VI** — (1) `500` không lộ nguyên nhân ở các route đọc chỉ được chứng minh ở tầng handler. (2) Binary chính abort khi dữ liệu nhúng sai được phủ qua parser và `cmd/seedsutras -dir`, chưa từ đầu đến cuối. (3) Dòng log `seeded sutras on first run` chưa có test kiểm. (4) Hai instance khởi động đồng thời chưa được test (chỉ có "hàng đã tồn tại"). (5) `index.json` nhân đôi metadata của từng file và lệch giữa chúng chỉ bị E2E dùng Docker bắt, không phải `-short`. (6) Chưa có test tải hay đọc đồng thời; thời gian phản hồi mới đo một lần trên loopback. (7) Test repository và E2E cần Postgres/Docker nên bị bỏ qua khi `-short`; test repository `TRUNCATE` các bảng của DB dev (mẫu có sẵn).

## 2. Why accepted / Vì sao chấp nhận

**EN** — These need fault injection or process orchestration that black-box tests cannot do cheaply; the gaps are listed in `test-coverage-report.md`.

**VI** — Cần chèn lỗi hoặc điều phối process mà test hộp đen khó làm rẻ; các khoảng trống đã được liệt kê trong `test-coverage-report.md`.

## 3. Risk if unpaid / Rủi ro nếu không trả

**EN** — A regression in these areas would not turn a test red; the first two matter most (leaking an internal error, a bad seed starting the service).

**VI** — Hồi quy ở các vùng này sẽ không làm test nào đỏ; hai điểm đầu quan trọng nhất (lộ lỗi nội bộ, seed sai vẫn cho service chạy).

## 4. Exit criteria / Điều kiện trả nợ

- [ ] Add a unit test that `index.json` metadata equals each file's metadata (runs under `-short`) / Thêm unit test kiểm metadata `index.json` bằng metadata từng file (chạy được với `-short`)
- [ ] Add a test that two concurrent `InsertAll` calls leave exactly one copy / Thêm test hai lần `InsertAll` đồng thời chỉ để lại đúng một bản
- [ ] Point repository tests at a dedicated test database instead of the dev one / Cho test repository dùng DB test riêng thay vì DB dev

## 5. Estimate / Ước lượng

**EN** — 0.5–1 day.

**VI** — 0,5–1 ngày.
