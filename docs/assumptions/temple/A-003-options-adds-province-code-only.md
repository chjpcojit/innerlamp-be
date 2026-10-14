---
id: A-003
feature: temple
title: "`options` adds only `province_code`, no filter / `options` chỉ thêm `province_code`, không thêm lọc"
status: unvalidated
created: 2026-10-11
made-by: AI (Claude), while specifying temple v1.7
code: docs/specs/features/temple/requirements.md (AC 20)
---

# A-003 — `options` adds only `province_code`, no filter / `options` chỉ thêm `province_code`, không thêm lọc

## 1. Assumption / Giả định

**EN** — `GET /v1/temples/options` returns four fields per item: `id`, `name`, `province`, `province_code`. No `ward_code`/`ward` (the pick-list shows name and province only; the rest is in `GET /v1/temples/{id}`) and no `?province_code=` filter.

**VI** — `GET /v1/temples/options` trả bốn field mỗi phần tử: `id`, `name`, `province`, `province_code`. Không có `ward_code`/`ward` (pick-list chỉ hiện tên và tỉnh; phần còn lại ở `GET /v1/temples/{id}`) và không có lọc `?province_code=`.

## 2. Rationale / Lý do

**EN** — YAGNI: the user asked to normalise the data, not to add filtering. The code lets a client group or filter the list locally and is the stable join key to `administrative-units`.

**VI** — YAGNI: người dùng yêu cầu chuẩn hoá dữ liệu, không yêu cầu lọc. Mã cho phép client tự nhóm/lọc cục bộ và là khoá nối ổn định với `administrative-units`.

## 3. Impact if wrong / Hệ quả nếu sai

**EN** — Adding `ward_code` or a filter later is additive and non-breaking.

**VI** — Thêm `ward_code` hoặc bộ lọc sau này chỉ là thêm, không phá hợp đồng.

## 4. How to validate / Cách xác nhận

**EN** — Owner confirms when the iOS registration screen is built; then set `status`.

**VI** — Người dùng xác nhận khi làm màn đăng ký iOS; sau đó đổi `status`.
