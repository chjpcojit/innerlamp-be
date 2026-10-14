---
id: A-001
feature: administrative-units
title: "Cache lifetime is configuration, not a constant / Thời hạn cache là cấu hình, không phải hằng số"
status: unvalidated
created: 2026-10-11
made-by: AI (Claude), while implementing administrative-units
code: services/appService/config/config.go (AdministrativeUnitsConfig.CacheMaxAge), services/appService/delivery/http/administrative_unit_handler.go
---

# A-001 — Cache lifetime is configuration, not a constant / Thời hạn cache là cấu hình, không phải hằng số

## 1. Assumption / Giả định

**EN** — The "24 hours" the owner agreed to for client caching is stored as `administrative_units.cache_max_age` (dev: `24h`) and sent as `Cache-Control: public, max-age=<seconds>`, rather than a literal 86400 in the handler.

**VI** — Con số "24 giờ" người dùng đồng ý cho cache phía client được lưu ở `administrative_units.cache_max_age` (dev: `24h`) và gửi dưới dạng `Cache-Control: public, max-age=<giây>`, không phải số 86400 viết cứng trong handler.

## 2. Rationale / Lý do

**EN** — A cache lifetime is a per-environment tuning value (AGENTS.md §5.3: non-sensitive tunables belong in config). The agreed value still holds for dev.

**VI** — Một thời hạn cache là giá trị tinh chỉnh theo môi trường (AGENTS.md §5.3: giá trị không nhạy cảm, tinh chỉnh được thì ở config). Giá trị đã thống nhất vẫn đúng cho dev.

## 3. Impact if wrong / Hệ quả nếu sai

**EN** — None for clients; ops can lengthen or shorten it without a release. A zero or sub-second value is refused at startup by `Validate`.

**VI** — Client không thấy khác; vận hành đổi dài/ngắn được mà không cần release. Giá trị 0 hoặc dưới 1 giây bị `Validate` từ chối lúc khởi động.

## 4. How to validate / Cách xác nhận

**EN** — Owner confirms or rejects; then set `status` and note it in `docs/specs/features/administrative-units/changelog.md`.

**VI** — Người dùng xác nhận hoặc bác bỏ; sau đó đổi `status` và ghi vào `docs/specs/features/administrative-units/changelog.md`.
