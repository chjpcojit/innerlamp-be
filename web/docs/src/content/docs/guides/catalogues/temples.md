---
title: Chùa
description: "Endpoint đọc danh mục chùa: nổi bật, danh sách phân trang và chi tiết."
---

Danh mục chùa **chỉ đọc**, không cần đăng nhập.

| Nổi bật | Danh sách phân trang | Chi tiết |
| --- | --- | --- |
| `GET /v1/temples/top` | `GET /v1/temples?page=N` | `GET /v1/temples/{id}` |

## Luồng dùng thường gặp

1. Màn hình chủ: gọi `top` để hiển thị mục nổi bật (tối đa 10).
2. Màn hình danh sách: gọi danh sách với `page`, cuộn tới đâu tải trang kế tiếp tới đó.
3. Màn hình chi tiết: gọi `{id}` với id lấy từ `top` hoặc danh sách. Danh sách và `top` trả bản tóm tắt, chi tiết trả đầy đủ hơn.

## `TempleSummary` (mỗi phần tử trong `items`)

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `id` | string | Vd. `temple_north_011`. Dùng cho `GET /v1/temples/{id}`. |
| `name` | string | |
| `type` | string | `CHUA`, `THIEN_VIEN`, `DAO_TRANG`, `TINH_XA`, `THIEN_XA` |
| `region` | string | `north`, `central`, `south` |
| `thumbnail_url` | string \| null | Hiện là `null` với mọi chùa. |
| `image_url` | string \| null | Như trên. |
| `sect` | string | Hệ phái, vd. `BẮC_TÔNG`, `TRUC_LAM_YEN_TU`, `KHAT_SI` |
| `heritage_rank` | string | Vd. `DI_SAN_THE_GIOI`, `DI_TICH_QUOC_GIA_DAC_BIET` |
| `established_year` | int | |
| `location` | object | `{lat, lng, address}` |

Các field dạng enum nên decode theo kiểu mở (giá trị lạ về case "khác"), không làm hỏng cả response.

## `GET /v1/temples/{id}` thêm

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `aliases` | string[] | `[]` nếu không có |
| `founder` | string | |
| `current_abbot` | string | |
| `operating_hours` | string | Vd. `"07:00 - 18:00"` |
| `amenities` | object | `parking_car`, `parking_motorcycle`, `vegetarian_meals`, `summer_retreat` (bool) |
| `introduction` | string | Giới thiệu chùa. `""` nếu chưa có |
| `architecture_heritage` | string | Kiến trúc và di sản nổi bật. `""` nếu chưa có |
| `visiting_notes` | string[] | Lưu ý khi viếng, mỗi phần tử một ý. `[]` nếu chưa có |

Ba field nội dung cuối không bao giờ là `null`.

```json
{
  "success": true,
  "data": {
    "id": "temple_north_011",
    "name": "Chùa Bích Động",
    "type": "CHUA",
    "region": "north",
    "thumbnail_url": null,
    "image_url": null,
    "sect": "BẮC_TÔNG",
    "heritage_rank": "DI_SAN_THE_GIOI",
    "established_year": 1428,
    "location": { "lat": 20.2185, "lng": 105.9177, "address": "Xã Ninh Hải, Huyện Hoa Lư, Tỉnh Ninh Bình" },
    "aliases": ["Bạch Ngọc Thạch Sơn Động"],
    "founder": "Hai nhà sư Trí Kiên và Trí Thể",
    "current_abbot": "Thích Đức Lợi",
    "operating_hours": "07:00 - 18:00",
    "amenities": { "parking_car": true, "parking_motorcycle": true, "vegetarian_meals": false, "summer_retreat": false },
    "introduction": "Chùa Bích Động nằm trong dãy núi đá vôi ở Hoa Lư, Ninh Bình, ...",
    "architecture_heritage": "Chùa xây dựa vào sườn núi đá với ba tầng: chùa Hạ, chùa Trung và chùa Thượng ...",
    "visiting_notes": [
      "Đường lên chùa Thượng có nhiều bậc đá dốc; nên đi giày dép chống trơn, mang nước.",
      "Trang phục kín đáo; giữ yên lặng khi vào các điện thờ trong hang."
    ]
  }
}
```

## Phân trang

```bash
curl 'http://localhost:8080/v1/temples?page=2'
```

`data` có dạng `{ "items": [...], "page", "page_size", "total", "total_pages" }`.

- `page` bắt đầu từ 1 và mặc định là 1. Giá trị không phải số nguyên dương trả `400 INVALID_PAGE`.
- Mỗi trang 20 mục (cấu hình dev). Dùng `total_pages` để biết khi nào dừng, đừng hard-code.
- Trang vượt quá cuối trả `200` với `items: []`, không phải lỗi.

## Ảnh

`thumbnail_url` và `image_url` là `null` khi chưa có ảnh (không phải chuỗi rỗng), hiện là `null` với mọi chùa. Hãy xử lý dạng Optional và hiển thị ảnh thay thế.

## Không tìm thấy

Id không tồn tại trả `404` với `TEMPLE_NOT_FOUND`.

Xem cấu trúc đầy đủ của response trong [API Reference](/api-reference/). Envelope và mã lỗi chung: [Xử lý lỗi](/guides/error-handling/).
