---
id: A-003
feature: dharmatalk
title: "Seed emptiness check counts hidden rows / Kiểm tra bảng rỗng của seed tính cả dòng bị ẩn"
status: unvalidated
created: 2026-10-03
made-by: AI (Claude), while writing the dharmatalk spec
code: docs/specs/features/dharmatalk/design.md
---

## Assumption / Giả định

**EN** — `SeedIfEmpty` counts all rows, including `needs_review = true`.
**VI** — `SeedIfEmpty` đếm mọi dòng, kể cả `needs_review = true`.

## Rationale / Lý do

**EN** — Otherwise a table with every row flagged would look empty and be reseeded, overwriting review decisions.
**VI** — Nếu không, bảng mà mọi dòng đều bị cờ sẽ bị coi là rỗng và seed lại, ghi đè quyết định duyệt.

## Impact if wrong / Hệ quả nếu sai

**EN** — Low; only affects restart behavior.
**VI** — Thấp; chỉ ảnh hưởng hành vi khi khởi động lại.

## How to validate / Cách xác nhận

**EN** — Covered by a usecase test in tasks.md Phase 2.
**VI** — Có test usecase ở tasks.md Phase 2.
