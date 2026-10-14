---
title: Phật và Bồ tát
description: "Endpoint đọc danh mục Phật và Bồ tát: nổi bật, danh sách Phật và Bồ tát phân trang và chi tiết."
---

Danh mục Phật và Bồ tát **chỉ đọc**, không cần đăng nhập.

| Nổi bật | Danh sách phân trang | Chi tiết |
| --- | --- | --- |
| `GET /v1/deities/top` | `GET /v1/deities?type=&page=N` | `GET /v1/deities/{id}` |

## Luồng dùng thường gặp

1. Màn hình chủ: gọi `top` để hiển thị mục nổi bật (tối đa 10).
2. Màn hình danh sách: gọi danh sách với `page`, cuộn tới đâu tải trang kế tiếp tới đó.
3. Màn hình chi tiết: gọi `{id}` với id lấy từ `top` hoặc danh sách. Danh sách và `top` trả bản tóm tắt, chi tiết trả đầy đủ hơn.

## `DeitySummary` (mỗi phần tử trong `items`)

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `id` | string | Vd. `bodhisattva_avalokiteshvara`. Dùng cho `GET /v1/deities/{id}`. |
| `type` | string | `BUDDHA` hoặc `BODHISATTVA` |
| `name_vi` | string | |
| `name_sanskrit` | string | |
| `name_chinese` | string | |
| `title_honorific` | string | |
| `thumbnail_url` | string | Đường dẫn tương đối từ dữ liệu nguồn, chưa tải được |
| `image_url` | string | URL giữ chỗ `cdn.yourapp.com`, chưa tải được |
| `core_virtue` | string | |
| `pure_land` | string | |

`top` và danh sách phân trang đều trộn cả Phật và Bồ tát (cùng một bảng); dùng `type` để phân biệt. Danh sách không có tham số lọc theo loại, muốn chỉ hiện một loại thì lọc phía client theo `type`.

:::caution
`GET /v1/bodhisattvas` đã bị xoá (không có alias). Đổi sang `GET /v1/deities?page=N`.
:::

## `GET /v1/deities/{id}` thêm

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `canonical_source` | string | Kinh điển gốc |
| `lunar_memorial_days` | object[] | `{lunar_day, lunar_month, event_name}` |
| `iconography` | object | `posture`, `mount_or_throne`, `sacred_artifacts` (string[]), `symbolic_meaning` |
| `biography_summary` | string | |
| `great_vows` | string[] | |
| `mantras` | object | `short_mantra_sanskrit`, `short_mantra_sino_vietnamese`, `long_dharani_name`, `dharani_phonetic_snippet` |

Mảng không bao giờ là `null`.

## Kinh, chú liên quan

`GET /v1/deities/{id}/sutras` trả các kinh, chú gắn với vị này. Xem [Kinh, kệ, chú](/guides/catalogues/sutras/#kinh-theo-phậtbồ-tát).

## Phân trang

```bash
curl 'http://localhost:8080/v1/deities?page=2'
```

`data` có dạng `{ "items": [...], "page", "page_size", "total", "total_pages" }`.

- `page` bắt đầu từ 1 và mặc định là 1. Giá trị không phải số nguyên dương trả `400 INVALID_PAGE`.
- Mỗi trang 20 mục (cấu hình dev). Dùng `total_pages` để biết khi nào dừng, đừng hard-code.
- Trang vượt quá cuối trả `200` với `items: []`, không phải lỗi.

## Lọc theo loại

```bash
curl 'http://localhost:8080/v1/deities?type=BUDDHA&page=1'
```

- `type` tuỳ chọn, nhận đúng một trong `BUDDHA`, `BODHISATTVA`, `DEVA` (phân biệt hoa/thường). Bỏ trống là mọi loại.
- Khi có `type`, `total` và `total_pages` chỉ đếm loại đó, `page` áp dụng trên danh sách đã lọc.
- Giá trị khác trả `400 INVALID_DEITY_TYPE` với `details.field = "type"`. Nếu cả `page` lẫn `type` sai, `INVALID_PAGE` được báo trước.

## Ảnh

`thumbnail_url` và `image_url` hiện chưa tải được (đường dẫn tương đối / URL giữ chỗ). Hãy xử lý lỗi tải ảnh và hiển thị ảnh thay thế.

## Không tìm thấy

Id không tồn tại trả `404` với `DEITY_NOT_FOUND`.

Xem cấu trúc đầy đủ của response trong [API Reference](/api-reference/). Envelope và mã lỗi chung: [Xử lý lỗi](/guides/error-handling/).
