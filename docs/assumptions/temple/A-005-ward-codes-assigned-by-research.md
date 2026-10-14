---
id: A-005
feature: temple
title: "The current ward of 25 temples was decided by research, not by the owner / Xã mới của 25 chùa do em tra nguồn quyết định, không phải chủ sản phẩm"
status: unvalidated
created: 2026-10-11
made-by: AI (Claude), while applying temple v1.7 data (Phase 10)
code: assets/data/temple.json (location.ward_code); docs/specs/features/temple/address-decisions.json
---

# A-005 — The current ward of 25 temples was decided by research, not by the owner / Xã mới của 25 chùa do em tra nguồn quyết định, không phải chủ sản phẩm

## 1. Assumption / Giả định

**EN** — 68 of the 93 temples were assigned automatically (their old ward maps to exactly one current ward in the dataset). The other 25 were decided by delegated web research against the 2025 merger resolutions and local sources, at the owner's instruction "decide everything yourself". Result: 19 high confidence, 5 medium (old ward split between several current wards, no street-level boundary text found), 1 unverified best guess (`temple_south_021`, An Khánh chosen between An Khánh and Bình Trưng). Evidence per temple is in `docs/specs/features/temple/address-decisions.json` and `address-decisions.md`.

**VI** — 68/93 chùa được gán tự động (xã cũ ánh xạ đúng một xã mới trong dataset). 25 chùa còn lại được quyết định bằng tra cứu nguồn (nghị quyết sáp nhập 2025 và nguồn địa phương), theo yêu cầu "em tự xử hết" của chủ sản phẩm. Kết quả: 19 tin cậy cao, 5 trung bình (xã cũ tách cho nhiều xã mới, không tìm được văn bản ranh giới cấp đường), 1 chưa xác minh (`temple_south_021`, chọn An Khánh trong An Khánh và Bình Trưng). Bằng chứng từng chùa ở `address-decisions.json` và `address-decisions.md`.

## 2. Rationale / Lý do

**EN** — The seed requires a ward for every temple (AC 34) and the owner delegated the review. Leaving the temples without codes would keep them out of the structured address; a flagged best effort is more useful than none.

**VI** — Seed bắt buộc có xã cho mọi chùa (AC 34) và chủ sản phẩm đã uỷ quyền duyệt. Để chùa không có mã sẽ loại chúng khỏi địa chỉ có cấu trúc; một kết quả tốt nhất có đánh dấu vẫn hữu ích hơn không có.

## 3. Impact if wrong / Hệ quả nếu sai

**EN** — A wrong `ward_code` shows the wrong ward in `full_address` for that temple and groups it with the wrong ward in any later ward-level filter. Province is unaffected (it comes from the temple's already-correct province).

**VI** — `ward_code` sai làm `full_address` của chùa đó hiện sai xã và xếp sai nhóm nếu sau này lọc theo xã. Tỉnh không bị ảnh hưởng (lấy từ tỉnh đã đúng của chùa).

## 4. How to validate / Cách xác nhận

**EN** — Check the 6 non-high rows in `address-decisions.md` against a map or the official boundary text; fix `ward_code` in `assets/data/temple.json` (or `address-decisions.json` and re-run `scripts/apply_temple_addresses.py` from `temple.json.pre-v1.7.bak`) and re-run `seedtemples`. Then set `status`.

**VI** — Đối chiếu 6 dòng không phải `high` trong `address-decisions.md` với bản đồ hoặc văn bản ranh giới chính thức; sửa `ward_code` trong `assets/data/temple.json` rồi chạy lại `seedtemples`. Sau đó đổi `status`.
