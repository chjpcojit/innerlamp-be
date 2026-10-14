---
title: Chùa
description: "Endpoint đọc danh mục chùa: nổi bật, danh sách phân trang và chi tiết."
---

Danh mục chùa **chỉ đọc**, không cần đăng nhập.

:::note
Hai route theo từng chùa nằm dưới cùng tiền tố nhưng **cần đăng nhập**: thời khoá tu tập `GET /v1/temples/{id}/schedule` (xem [Thời khoá tu tập của chùa](/guides/catalogues/temple-schedule/)) và sự kiện `GET /v1/temples/{id}/events` (xem [Sự kiện của chùa](/guides/catalogues/temple-events/)).
:::

| Nổi bật | Danh sách phân trang | Chi tiết | Danh sách rút gọn để chọn |
| --- | --- | --- | --- |
| `GET /v1/temples/top` | `GET /v1/temples?page=N` | `GET /v1/temples/{id}` | `GET /v1/temples/options?page=N&q=` |

## Luồng dùng thường gặp

1. Màn hình chủ: gọi `top` để hiển thị mục nổi bật (tối đa 10).
2. Màn hình danh sách: gọi danh sách với `page`, cuộn tới đâu tải trang kế tiếp tới đó.
3. Màn hình chi tiết: gọi `{id}` với id lấy từ `top` hoặc danh sách. Danh sách và `top` trả bản tóm tắt, chi tiết trả đầy đủ hơn.
4. Màn chọn chùa (ví dụ khi đăng ký): gọi `options` để lấy danh sách rút gọn, xem [Chọn chùa](#chọn-chùa-options).

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
| `location` | object | `{lat, lng, address, province, province_code, ward_code, ward, full_address}`. `address` chỉ là phần **dưới cấp xã** (số nhà, đường, thôn…), có thể rỗng. `province` là tên ngắn không tiền tố (vd. `Ninh Bình`, `Huế`, `Hồ Chí Minh`), một trong 34 đơn vị cấp tỉnh hiện hành. `province_code`, `ward_code`, `ward` là mã và tên **hệ mới** (`null` cùng lúc khi chùa chưa có địa chỉ có cấu trúc). `full_address` là chuỗi để hiển thị, không bao giờ `null`. |

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
    "location": { "lat": 20.2185, "lng": 105.9177, "address": "", "province": "Ninh Bình", "province_code": "37", "ward_code": "14359", "ward": "Phường Nam Hoa Lư", "full_address": "Phường Nam Hoa Lư, Ninh Bình" },
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

## Chọn chùa (`options`)

`GET /v1/temples/options` trả danh sách chùa **rút gọn** để người dùng chọn một ngôi chùa, ví dụ `ordination_temple_id` khi [đăng ký](/guides/register-and-login/). Công khai: **không cần đăng nhập và không cần `X-API-Key`**, nên gọi được ở màn đăng ký khi chưa có tài khoản.

Mỗi phần tử chỉ có đúng bốn field:

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `id` | string | Gửi làm `ordination_temple_id`. |
| `name` | string | Tên chùa. |
| `province` | string | Tên tỉnh/thành không tiền tố, một trong 34 đơn vị cấp tỉnh hiện hành. Dùng để hiện cạnh tên và phân biệt các chùa trùng tên. |
| `province_code` | string \| null | Mã tỉnh/thành hệ mới (khoá nối với `administrative-units`). `null` khi chùa chưa có địa chỉ có cấu trúc. |

Muốn địa chỉ đầy đủ, ảnh, toạ độ, loại, hệ phái hay hạng di tích thì gọi `GET /v1/temples` hoặc `GET /v1/temples/{id}`.

```bash
curl 'http://localhost:8080/v1/temples/options?page=1'
curl 'http://localhost:8080/v1/temples/options?q=bich%20dong'
```

```json
{
  "success": true,
  "data": {
    "items": [
      { "id": "temple_north_011", "name": "Chùa Bích Động", "province": "Ninh Bình", "province_code": "37" },
      { "id": "temple_central_001", "name": "Chùa Thiên Mụ", "province": "Huế", "province_code": "46" }
    ],
    "page": 1,
    "page_size": 100,
    "total": 93,
    "total_pages": 1
  }
}
```

- **Phân trang theo `page`**, cùng quy tắc với danh sách ở trên (bắt đầu từ 1, `400 INVALID_PAGE` khi không phải số nguyên dương, trang vượt cuối trả `items: []`). Trang lớn hơn danh sách duyệt (100 mục, cấu hình dev): dùng `page_size` và `total_pages` trong response, đừng hard-code.
- **Thứ tự** giống `top` và danh sách phân trang (di sản nổi bật trước, rồi năm thành lập, rồi `id`), **không** theo bảng chữ cái. Client tự sắp trong từng trang nếu cần.
- **Tìm theo tên** bằng `q`: khớp một phần tên, **không phân biệt hoa thường và dấu tiếng Việt** (`bich dong`, `Bích Động`, `BÍCH ĐỘNG` cho cùng kết quả; `đ` tính như `d`). `total` và `total_pages` đếm theo kết quả khớp. Chỉ tìm trong tên chùa, không tìm theo tên khác hay tỉnh. Không khớp chùa nào thì `200` với `items: []`.
- `q` dài 1 đến 100 ký tự sau khi bỏ khoảng trắng hai đầu. Muốn bỏ tìm kiếm thì **bỏ hẳn tham số `q`**: để trống hoặc chỉ khoảng trắng trả `400 INVALID_FILTER` (`details.field = "q"`).

## Ảnh

`thumbnail_url` và `image_url` là `null` khi chưa có ảnh (không phải chuỗi rỗng), hiện là `null` với mọi chùa. Hãy xử lý dạng Optional và hiển thị ảnh thay thế.

## Không tìm thấy

Id không tồn tại trả `404` với `TEMPLE_NOT_FOUND`.

Muốn đề xuất một ngôi chùa chưa có trong danh mục: [Đề xuất thêm chùa](/guides/catalogues/temple-suggestions/).

Xem cấu trúc đầy đủ của response trong [API Reference](/api-reference/). Envelope và mã lỗi chung: [Xử lý lỗi](/guides/error-handling/).
