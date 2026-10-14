---
title: Thời khoá tu tập của chùa
description: "Endpoint xem thời khoá sinh hoạt thường ngày của một ngôi chùa: buổi, giờ, ai hướng dẫn, ở đâu, trình tự và cần chuẩn bị gì (cần đăng nhập)."
---

Thời khoá là **nếp sinh hoạt thường ngày** của một ngôi chùa (công phu sáng, lễ Dược Sư, toạ thiền…). Đây **không phải sự kiện**: không có ngày cụ thể, không có trạng thái hoãn/huỷ.

| Thao tác | Endpoint | Đăng nhập |
| --- | --- | --- |
| Thời khoá của một chùa | `GET /v1/temples/{id}/schedule` | **cần** |

:::caution[Khác các route chùa còn lại]
`GET /v1/temples`, `/top`, `/options` và `/{id}` là công khai. Route `/schedule` nằm dưới cùng tiền tố nhưng **bắt buộc** `Authorization: Bearer <access_token>`. Thiếu token, token hết hạn hoặc là refresh token trả `401 UNAUTHORIZED`.
:::

```bash
curl 'http://localhost:8080/v1/temples/temple_north_001/schedule' \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

## Response

`data` có dạng `{ "temple_id", "items": [...] }`. Không phân trang, không giới hạn số buổi.

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `id` | string | Mã ổn định của buổi, vd. `tsch_north_001_duoc_su`. |
| `name` | string | Tên buổi. |
| `start_time` | string | Giờ bắt đầu `HH:MM` (24 giờ, giờ Việt Nam), không kèm ngày. |
| `end_time` | string \| null | Giờ kết thúc `HH:MM`. `null` khi không có. **Nhỏ hơn `start_time` nghĩa là buổi qua nửa đêm** (vd. `23:00` đến `01:00`). |
| `weekdays` | int[] | **`[]` nghĩa là mỗi ngày.** Nếu không là các thứ áp dụng, tăng dần, `0` = Chủ nhật … `6` = Thứ bảy. |
| `description` | string | Mô tả buổi. `""` nếu không có. |
| `instructor` | string | Người hướng dẫn, văn bản tự do. `""` nếu không có. |
| `location` | string | Địa điểm trong chùa (Chánh điện, Thiền đường…), văn bản tự do. `""` nếu không có. |
| `preparation` | string | Cần chuẩn bị gì. `""` nếu không có. |
| `steps` | object[] | Trình tự của buổi. `[]` nếu không có. |

Mỗi phần tử của `steps`:

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `position` | int | Thứ tự hiển thị, bắt đầu từ 1. |
| `time` | string \| null | Giờ `HH:MM` của bước, `null` khi không có. Chỉ là thông tin. |
| `text` | string | Nội dung bước. |

```json
{
  "success": true,
  "data": {
    "temple_id": "temple_north_001",
    "items": [
      {
        "id": "tsch_north_001_cong_phu_sang",
        "name": "Công phu sáng",
        "start_time": "04:00",
        "end_time": "05:00",
        "weekdays": [],
        "description": "Tụng kinh, trì chú đầu ngày.",
        "instructor": "Chư Tăng trong chùa",
        "location": "Chánh điện",
        "preparation": "",
        "steps": [
          { "position": 1, "time": "04:00", "text": "Thỉnh chuông, đại hồng chung" },
          { "position": 2, "time": "04:15", "text": "Tụng kinh Lăng Nghiêm" },
          { "position": 3, "time": null, "text": "Hồi hướng" }
        ]
      },
      {
        "id": "tsch_north_001_duoc_su",
        "name": "Lễ Dược Sư",
        "start_time": "08:00",
        "end_time": "11:00",
        "weekdays": [0],
        "description": "Lễ cầu an, tụng kinh Dược Sư; Phật tử tham dự.",
        "instructor": "Thượng toạ trụ trì",
        "location": "Chánh điện",
        "preparation": "Mặc áo tràng, mang theo kinh sách.",
        "steps": []
      }
    ]
  }
}
```

## Quy ước client cần biết

- **Thứ tự**: `items` đã sắp theo `start_time`, rồi thứ tự hiển thị, rồi `id`. `steps` sắp theo `position`, **không** theo `time`: một buổi qua nửa đêm có thể có bước `23:45` rồi `00:15`.
- **`weekdays` dùng `0` = Chủ nhật** (khác ISO, nơi `1` là Thứ hai). Client cần ánh xạ sang `Calendar` của hệ điều hành. Chọn đủ 7 ngày cũng được trả về `[]`.
- **Chùa chưa có thời khoá** trả `200` với `items: []`, không phải lỗi.
- **Buổi bị tắt** không xuất hiện (BO huỷ một buổi bằng cách tắt, không xoá).
- **`instructor` và `location` là văn bản tự do** áp cho cả buổi; không có trang riêng cho vị sư hay địa điểm.
- Chưa có header cache; dữ liệu chỉ đổi khi vận hành viên nạp lại.
- Dữ liệu hiện là **mock** do đội phát triển soạn, chưa phải thời khoá thật của chùa.

## Lỗi

| HTTP | `error.code` | Khi nào |
| --- | --- | --- |
| 401 | `UNAUTHORIZED` | Thiếu token, token sai, hết hạn hoặc không phải access token. |
| 404 | `TEMPLE_NOT_FOUND` | Không có chùa với `id` này. |
| 500 | `INTERNAL_ERROR` | Lỗi phía server. |

Xem thêm [Chùa](/guides/catalogues/temples/) cho các route công khai và [Xử lý lỗi](/guides/error-handling/).
