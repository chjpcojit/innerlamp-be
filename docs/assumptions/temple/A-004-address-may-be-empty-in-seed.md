---
id: A-004
feature: temple
title: "`location.address` may be empty in the seed file / `location.address` được phép rỗng trong file seed"
status: unvalidated
created: 2026-10-11
made-by: AI (Claude), while building scripts/build_temple_addresses.py (temple v1.10)
code: docs/specs/features/temple/requirements.md (AC 34)
---

# A-004 — `location.address` may be empty in the seed file / `location.address` được phép rỗng trong file seed

## 1. Assumption / Giả định

**EN** — When a temple's old address has nothing below ward level (20 of the 93 temples, e.g. `Xã Hương Sơn, Huyện Mỹ Đức, Hà Nội`), the seed accepts `location.address` as an empty string. The full address is then just `ward, province`.

**VI** — Khi địa chỉ cũ của chùa không có gì dưới cấp xã (20/93 chùa, vd `Xã Hương Sơn, Huyện Mỹ Đức, Hà Nội`), seed chấp nhận `location.address` là chuỗi rỗng. Địa chỉ đầy đủ khi đó chỉ là `xã, tỉnh`.

## 2. Rationale / Lý do

**EN** — Inventing a street part would be false data; rejecting the whole seed would block 20 temples that are correctly located by ward. `full_address` already joins only the non-empty parts.

**VI** — Bịa phần đường là dữ liệu sai; từ chối cả file seed sẽ chặn 20 chùa vốn đã định vị đúng theo xã. `full_address` vốn chỉ ghép các phần không rỗng.

## 3. Impact if wrong / Hệ quả nếu sai

**EN** — If the owner wants every temple to have a street/place line, the reviewer fills those 20 by hand (a place name such as the mountain or hamlet is enough) and AC 34 goes back to "non-blank".

**VI** — Nếu chủ sản phẩm muốn mọi chùa đều có dòng đường/nơi chốn, người duyệt điền tay 20 chùa này (tên núi, thôn là đủ) và AC 34 trở lại "không rỗng".

## 4. How to validate / Cách xác nhận

**EN** — Owner confirms or rejects; then set `status` and update `requirements.md` AC 34.

**VI** — Người dùng xác nhận hoặc bác bỏ; sau đó đổi `status` và cập nhật AC 34 trong `requirements.md`.
