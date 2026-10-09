---
id: A-001
feature: dharmatalk
title: "Feature name, routes, no top route / Tên feature, route, không có route top"
status: unvalidated
created: 2026-10-03
made-by: AI (Claude), while writing the dharmatalk spec
code: docs/specs/features/dharmatalk/design.md
---

## Assumption / Giả định

**EN** — Feature folder `dharmatalk`; routes `GET /v1/dharma-talks` and `/{id}`; no `top` route.
**VI** — Thư mục feature `dharmatalk`; route `GET /v1/dharma-talks` và `/{id}`; không có route `top`.

## Rationale / Lý do

**EN** — Mirrors `article`; the mockup is only a detail screen, so YAGNI drops `top`.
**VI** — Theo khuôn `article`; mockup chỉ có màn chi tiết nên bỏ `top` (YAGNI).

## Impact if wrong / Hệ quả nếu sai

**EN** — Client paths change; adding `top` later is additive.
**VI** — Đổi đường dẫn phía client; thêm `top` sau này chỉ là thêm mới.

## How to validate / Cách xác nhận

**EN** — Ask the user and the iOS developer; check the home screen design.
**VI** — Hỏi người dùng và dev iOS; xem thiết kế màn chủ.
