---
id: A-002
feature: temple-suggestion
title: "Suggestions submitted before v3.0 are not backfilled / Đề xuất gửi trước v3.0 không được backfill"
status: unvalidated
created: 2026-10-11
made-by: AI (Claude), while specifying temple-suggestion v3.0
code: docs/specs/features/temple-suggestion/design.md (Schema, Known gaps)
---

# A-002 — Suggestions submitted before v3.0 are not backfilled / Đề xuất gửi trước v3.0 không được backfill

## 1. Assumption / Giả định

**EN** — Existing rows keep their free-text `province` and get `NULL` `province_code`, `ward_code` and `ward`. The back-office list must show both shapes.

**VI** — Các dòng đã có giữ `province` chuỗi tự do và để `province_code`, `ward_code`, `ward` là `NULL`. Danh sách backoffice phải hiển thị được cả hai dạng.

## 2. Rationale / Lý do

**EN** — A ward cannot be derived reliably from a free-text province, and the app has not been released, so few or no real rows exist.

**VI** — Không thể suy ra xã từ tỉnh gõ tự do, và app chưa release nên có rất ít hoặc không có dòng thật.

## 3. Impact if wrong / Hệ quả nếu sai

**EN** — If real pending rows exist, an admin reviews them by hand like before; nothing is lost.

**VI** — Nếu có dòng pending thật, admin duyệt tay như trước; không mất dữ liệu.

## 4. How to validate / Cách xác nhận

**EN** — Count rows in dev/staging/production before the migration; owner confirms.

**VI** — Đếm số dòng ở dev/staging/production trước khi chạy migration; người dùng xác nhận.
