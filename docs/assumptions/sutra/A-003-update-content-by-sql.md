---
id: A-003
feature: sutra
title: "Content is updated by hand-written SQL, as for articles / Nội dung được cập nhật bằng SQL tay, như article"
status: unvalidated
created: 2026-10-03
made-by: AI (Claude), while implementing the sutra feature
code: services/appService/usecase/sutra_usecase.go (SeedIfEmpty)
---

# A-003 — Content is updated by hand-written SQL, as for articles / Nội dung được cập nhật bằng SQL tay, như article

## 1. Assumption / Giả định

**EN** — The user "confirmed" the open point about how content changes after the first seed without naming an option. The default of the `article` feature was recorded: SQL by hand; reseed and backoffice CRUD deferred.

**VI** — Người dùng "chốt" điểm còn mở về cách đổi nội dung sau lần nạp đầu mà không nêu phương án. Mặc định của feature `article` được ghi lại: SQL tay; reseed và CRUD backoffice được hoãn.

## 2. Rationale / Lý do

**EN** — It is what the closest feature does and needs no new code; the user did not ask for more.

**VI** — Đó là điều feature gần nhất đang làm và không cần thêm code; người dùng không yêu cầu hơn.

## 3. Impact if wrong / Hệ quả nếu sai

**EN** — Corrections to the texts (see TD-002) cannot be shipped without SQL; a different mechanism would change the seed rules and their tests.

**VI** — Chỉnh sửa văn bản (xem TD-002) không phát hành được nếu thiếu SQL; một cơ chế khác sẽ đổi luật seed và các test của nó.

## 4. How to validate / Cách xác nhận

**EN** — Ask the user which mechanism they want (replace-one-sutra command, or CRUD in backofficeService). Related debt: TD-004.

**VI** — Hỏi người dùng muốn cơ chế nào (lệnh thay một kinh, hoặc CRUD ở backofficeService). Nợ liên quan: TD-004.
