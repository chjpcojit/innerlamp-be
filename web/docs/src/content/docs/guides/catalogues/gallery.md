---
title: Ảnh đẹp Phật giáo
description: "Endpoint đọc thư viện ảnh Phật giáo: nổi bật, danh sách phân trang lọc theo loại và chi tiết kèm ghi công tác giả."
---

Thư viện ảnh **chỉ đọc**, không cần đăng nhập.

| Nổi bật | Danh sách phân trang | Chi tiết |
| --- | --- | --- |
| `GET /v1/gallery/top` | `GET /v1/gallery?page=N&type=` | `GET /v1/gallery/{id}` |

## Luồng dùng thường gặp

1. Màn hình chủ: gọi `top` (10 ảnh, trộn đủ bốn loại theo vòng).
2. Màn lưới ảnh: gọi danh sách với `page`; thêm `type` khi người dùng chọn một loại.
3. Màn xem ảnh: gọi `{id}` để lấy thêm thông tin ghi công.

## Loại ảnh (`type`)

`CHUA` (chùa), `TUONG_PHAT` (tượng Phật), `THIEN` (thiền), `LE_HOI` (lễ hội). Bỏ `type` là mọi loại; khi có thì `total` và `total_pages` chỉ đếm loại đó. Giá trị khác trả `400 INVALID_GALLERY_TYPE` (`details.field = "type"`).

## `GallerySummary` (mỗi phần tử trong `items`)

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `id` | string | Vd. `gallery_001`. Dùng cho `GET /v1/gallery/{id}`. |
| `type` | string | Một trong bốn loại trên. |
| `title` | string | |
| `location` | string | |
| `thumbnail_url` | string | Ảnh nhỏ cho lưới. |
| `image_url` | string | Ảnh đầy đủ. |

`top` trả `items`. Danh sách trả `items`, `page`, `page_size` (20), `total`, `total_pages`; trang vượt cuối trả `200` với `items: []`.

## Chi tiết và ghi công

`GET /v1/gallery/{id}` trả toàn bộ field của `GallerySummary`, thêm `author`, `license`, `source_url`. Giấy phép CC BY / CC BY-SA yêu cầu ghi công, nên **màn chi tiết phải hiển thị tác giả và giấy phép**.

## Lỗi

| HTTP | code | Khi nào |
| --- | --- | --- |
| 400 | `INVALID_PAGE` | `page` không phải số nguyên dương. |
| 400 | `INVALID_GALLERY_TYPE` | `type` không hợp lệ. |
| 404 | `GALLERY_IMAGE_NOT_FOUND` | Không có ảnh với id này. |

Cấu trúc đầy đủ: [API Reference](/api-reference/). Mã lỗi chung: [Xử lý lỗi](/guides/error-handling/).
