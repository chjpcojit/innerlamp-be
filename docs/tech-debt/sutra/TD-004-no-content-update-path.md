---
id: TD-004
feature: sutra
title: "No way to update sutra content after the first seed / Không có cách cập nhật nội dung kinh sau lần nạp đầu"
severity: medium
status: open
created: 2026-10-03
owner: ducnguyen-wee
code: services/appService/usecase/sutra_usecase.go (SeedIfEmpty); services/appService/cmd/seedsutras/main.go
---

# TD-004 — No way to update sutra content after the first seed / Không có cách cập nhật nội dung kinh sau lần nạp đầu

## 1. Debt / Nợ gì

**EN** — Seeding runs only when the `sutras` table is empty, in both the service and `cmd/seedsutras`. Editing the JSON does not reach an existing database; there is no replace-one-sutra, no content version or `updated_at` per sutra, and no admin CRUD. Fixing a typo in a deployed database means hand-written SQL across `sutra_lines`.

**VI** — Việc nạp chỉ chạy khi bảng `sutras` rỗng, cả ở service lẫn `cmd/seedsutras`. Sửa file JSON không đến được DB đã có dữ liệu; không có lệnh thay một kinh, không có version nội dung hay `updated_at` theo từng kinh, và không có CRUD cho admin. Sửa một lỗi chính tả trên DB đã triển khai nghĩa là viết SQL tay trên `sutra_lines`.

## 2. Why accepted / Vì sao chấp nhận

**EN** — Same as `article`; the user confirmed the point without naming a mechanism, so SQL by hand was recorded as the default (assumption A-003). Reseed and backoffice CRUD were deferred.

**VI** — Giống `article`; người dùng chốt điểm này mà không nêu cơ chế nên SQL tay được ghi làm mặc định (giả định A-003). Reseed và CRUD backoffice được hoãn.

## 3. Risk if unpaid / Rủi ro nếu không trả

**EN** — The corrections that TD-002 will produce cannot be shipped safely; development databases drift from the JSON; clients cannot tell that a text changed.

**VI** — Các chỉnh sửa mà TD-002 sẽ tạo ra không thể phát hành an toàn; DB dev lệch khỏi JSON; client không biết một văn bản đã đổi.

## 4. Exit criteria / Điều kiện trả nợ

- [ ] Pick a mechanism: `cmd/seedsutras -replace <id>` (replace one sutra in one transaction) or CRUD in backofficeService / Chọn cơ chế: `cmd/seedsutras -replace <id>` (thay một kinh trong một transaction) hoặc CRUD ở backofficeService
- [ ] Add a per-sutra `updated_at`/version the client can cache against / Thêm `updated_at`/version theo từng kinh để client cache
- [ ] Cover it with an E2E test like the existing seed tests / Có test E2E giống các test seed hiện có

## 5. Estimate / Ước lượng

**EN** — 1–2 days.

**VI** — 1–2 ngày.
