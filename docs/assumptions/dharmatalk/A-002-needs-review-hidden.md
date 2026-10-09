---
id: A-002
feature: dharmatalk
title: "Talks flagged needs_review are hidden / Bài bị cờ needs_review bị ẩn"
status: unvalidated
created: 2026-10-03
made-by: AI (Claude), while writing the dharmatalk spec
code: docs/specs/features/dharmatalk/design.md
---

## Assumption / Giả định

**EN** — Public routes only expose `needs_review = false` (36 of 55 seeded talks); flagged rows are stored.
**VI** — Route công khai chỉ trả bài `needs_review = false` (36/55 bài); bài bị cờ vẫn được lưu.

## Rationale / Lý do

**EN** — Flagged talks have weak summaries/quotes, unverified content (3 Thông Triệt videos) or doubtful speaker identity; showing them risks wrong attribution.
**VI** — Bài bị cờ có tóm tắt/trích dẫn yếu, nội dung chưa xác minh (3 video Thông Triệt) hoặc nghi ngờ giảng sư; hiển thị có thể ghi sai nguồn.

## Impact if wrong / Hệ quả nếu sai

**EN** — If the user wants them visible, the list wrongly lacks 19 talks.
**VI** — Nếu người dùng muốn hiển thị thì danh sách đang thiếu 19 bài.

## How to validate / Cách xác nhận

**EN** — User reviews each flagged talk and decides keep/drop; then clear the flag via SQL.
**VI** — Người dùng duyệt từng bài bị cờ, quyết định giữ/loại; rồi gỡ cờ bằng SQL.
