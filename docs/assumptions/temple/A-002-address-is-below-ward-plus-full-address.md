---
id: A-002
feature: temple
title: "`address` becomes the below-ward part; the API adds `full_address` / `address` chỉ còn phần dưới cấp xã; API thêm `full_address`"
status: unvalidated
created: 2026-10-11
made-by: AI (Claude), while specifying temple v1.7
code: docs/specs/features/temple/requirements.md (US-TMP-08, AC 33)
---

# A-002 — `address` becomes the below-ward part; the API adds `full_address` / `address` chỉ còn phần dưới cấp xã; API thêm `full_address`

## 1. Assumption / Giả định

**EN** — For a re-seeded temple, `location.address` holds only what is typed below ward level (house number, street, lane, hamlet) — the same meaning as `temple_suggestions.address`. The server also returns `location.full_address`, the non-empty parts `address`, `ward`, `province` joined by `", "`, for display. `province` stays the short name without `Tỉnh`/`Thành phố`, so `full_address` carries no province prefix. A row whose `ward` is NULL (not re-seeded) returns `full_address` equal to its stored `address` unchanged.

**VI** — Với chùa đã nạp lại, `location.address` chỉ còn phần gõ tay dưới cấp xã (số nhà, đường, ngõ, thôn) — cùng nghĩa với `temple_suggestions.address`. Server trả thêm `location.full_address` để hiển thị, ghép các phần không rỗng của `address`, `ward`, `province` bằng `", "`. `province` vẫn là tên ngắn không `Tỉnh`/`Thành phố`, nên `full_address` không có tiền tố cho tỉnh. Dòng có `ward` NULL (chưa nạp lại) trả `full_address` đúng bằng `address` đang lưu.

## 2. Rationale / Lý do

**EN** — One meaning of `address` across `temples`, `temple_suggestions` and genealogy; client no longer has to guess whether it contains the ward. The app is not released yet (temple-suggestion N8), so changing what `address` holds is acceptable.

**VI** — Một nghĩa của `address` trên `temples`, `temple_suggestions` và genealogy; client không còn phải đoán nó có chứa xã hay không. App chưa release (temple-suggestion N8) nên đổi nội dung `address` là chấp nhận được.

## 3. Impact if wrong / Hệ quả nếu sai

**EN** — If a client already displays `location.address` as the full address it will show only the street until it switches to `full_address`. If the owner wants `Tỉnh`/`Thành phố` prefixes, a `province_full_name` snapshot column is needed.

**VI** — Nếu client đang hiển thị `location.address` như địa chỉ đầy đủ thì sẽ chỉ thấy phần đường cho tới khi chuyển sang `full_address`. Nếu chủ sản phẩm muốn tiền tố `Tỉnh`/`Thành phố` thì cần thêm cột chụp `province_full_name`.

## 3a. Confirmed part / Phần đã xác nhận

**EN** — 2026-10-11 (owner): keep the short province name in `full_address`, no `Tỉnh`/`Thành phố` prefix (e.g. `Hồ Chí Minh`). The meaning change of `address` and the new `full_address` field are still unvalidated.

**VI** — 2026-10-11 (chủ sản phẩm): giữ tên ngắn của tỉnh trong `full_address`, không tiền tố `Tỉnh`/`Thành phố` (vd `Hồ Chí Minh`). Việc đổi nghĩa `address` và field mới `full_address` vẫn chưa xác nhận.

## 4. How to validate / Cách xác nhận

**EN** — Owner confirms or rejects (and decides the prefix); then set `status` and resolve the `⚠️ Not confirmed` marker in `requirements.md`.

**VI** — Người dùng xác nhận hoặc bác bỏ (và quyết định tiền tố); sau đó đổi `status` và gỡ marker `⚠️ Not confirmed` trong `requirements.md`.
