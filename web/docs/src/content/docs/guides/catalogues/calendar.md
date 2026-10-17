---
title: Lịch âm (ảnh theo ngày và sự kiện)
description: "Ảnh Phật theo ngày dương lịch cho màn Lịch âm, và sự kiện lịch Phật giáo, Việt Nam, dân gian theo năm, kèm cả ngày dương và ngày âm (cả hai công khai)."
---

Module lịch có hai endpoint, đều công khai (không cần đăng nhập):

| Thao tác | Endpoint | Đăng nhập | Trạng thái |
| --- | --- | --- | --- |
| Ảnh Phật theo ngày | `GET /v1/calendar/image` | không | đang chạy |
| Sự kiện lịch của một năm | `GET /v1/calendar/events` | không (từ hợp đồng 2.2.0; trước đó cần) | đang chạy |

## Ảnh Phật theo ngày

Endpoint **công khai, chỉ đọc**:

```bash
curl 'http://localhost:8080/v1/calendar/image?date=2026-10-07'
```

### Hành vi

- `date` (tuỳ chọn): ngày dương `YYYY-MM-DD`, năm 1900 đến 2100. Sai định dạng hoặc ngoài khoảng trả `400 INVALID_DATE` (`details.field = "date"`).
- Cùng một ngày luôn cùng một ảnh; ngày liên tiếp cho ảnh khác nhau (chu kỳ 100 ngày).
- Bỏ trống `date` là **hôm nay theo giờ `Asia/Ho_Chi_Minh`**; `data.date` trả ngày đã dùng.
- Cache: có `date` thì `Cache-Control: public, max-age=86400`, không có thì `no-cache`.
- Ảnh không phụ thuộc lịch âm: server chọn theo số ngày dương, không theo ngày lễ.

### Response

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `date` | string | Ngày đã dùng. |
| `id` | string | Vd. `buddha_034`. |
| `image_url` | string | Ảnh đầy đủ. Chưa có `thumbnail_url`. |
| `author`, `license`, `license_url`, `source_url` | string | Ghi công. |

Ảnh CC BY / CC BY-SA bắt buộc hiện `author` và `license` trên màn hình.

## Sự kiện lịch

Các sự kiện Phật giáo (Phật đản, Vu lan, mùng 1, Rằm…), ngày lễ của Việt Nam (30/4, 2/9…) và lễ tết dân gian (Tết Nguyên Đán, Trung thu, Táo quân…). Mỗi lần xảy ra có **cả ngày dương lẫn ngày âm**, nên app không cần tự tính lịch âm để hiển thị sự kiện. **Công khai**: không cần đăng nhập, header `Authorization` nếu có thì bị bỏ qua. Client web gửi `X-API-Key` như các route khác (sai thì `401 INVALID_API_KEY`); mobile không gửi.

```bash
curl 'http://localhost:8080/v1/calendar/events?year=2026'
```

### Hành vi

- `year` (tuỳ chọn): năm dương, số nguyên 1900 đến 2100. Không phải số nguyên hoặc ngoài khoảng trả `400 INVALID_YEAR` (`details.field = "year"`). Bỏ trống là **năm hiện tại theo giờ `Asia/Ho_Chi_Minh`**; `data.year` trả năm đã dùng.
- Mỗi lần gọi trả **một năm** (1/1 đến 31/12). Hiển thị theo tháng thì app tự lọc `occurrences` theo `solar_date`. Muốn xem thêm thì gọi năm kế tiếp (`year + 1`); gọi năm trước cũng được.
- Server lưu **quy tắc lặp** của từng sự kiện rồi tính ngày cho năm được hỏi, nên mọi năm trong 1900..2100 đều có dữ liệu.
- Cache: có `year` thì `Cache-Control: public, max-age=<24 giờ>` (CDN / proxy cache được), không có thì `no-cache`.
- Giới hạn theo IP (dev: 60 lượt/phút); vượt thì `429 RATE_LIMITED` kèm `details.retry_after_seconds`. Request mang `X-API-Key` đúng không bị đếm, nên website nên gửi key và tự cache theo năm.

### Response

`data` gồm bốn phần. Nội dung mỗi sự kiện chỉ xuất hiện **một lần** trong `events`; mỗi lần xảy ra là một phần tử của `occurrences`, trỏ về `events[].code`, nên mùng 1 và Rằm (khoảng 25 lần mỗi năm) không lặp mô tả dài.

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `year` | int | Năm dương đã dùng. |
| `groups` | mảng | `{code, name}` của các nhóm: ban đầu `buddhist` (Phật giáo), `vietnam` (Việt Nam), `folk` (dân gian). |
| `events` | mảng | Nội dung sự kiện (bảng dưới). Chỉ gồm sự kiện có lần xảy ra trong năm. |
| `occurrences` | mảng | Các lần xảy ra, đã sắp theo `solar_date`. |

`events[]`:

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `code` | string | Ổn định, khoá của sự kiện. |
| `name`, `short_description`, `long_description` | string | Tiếng Việt. |
| `guidance` | string | Việc nên làm; **mỗi dòng (`\n`) là một việc**. `long_description`: mỗi dòng là một đoạn. Tách theo xuống dòng, không theo dấu phẩy. |
| `image_url` | string hoặc `null` | `null` khi chưa có ảnh (bản đầu chưa có ảnh sự kiện). |
| `groups` | mảng string | Một hoặc nhiều mã nhóm; một sự kiện có thể thuộc nhiều nhóm. |
| `is_public_holiday` | bool | Nghỉ lễ chính thức, chỉ đúng cho **ngày của sự kiện**; ngày nghỉ bù không có lần xảy ra. |
| `is_major` | bool | **Mới 2.2.0.** "Lễ lớn" do biên tập đánh dấu (10 sự kiện: Rằm tháng Giêng, Đức Phật xuất gia, nhập Niết bàn, Quan Thế Âm đản sinh / thành đạo / xuất gia, Phật đản, Vu Lan, vía A Di Đà, Đức Phật thành đạo). Dùng cho bộ lọc "Lễ lớn" và khối Home. |
| `sort_order` | int | **Mới 2.2.0.** Thứ tự hiển thị trong cùng một ngày. Mùng 1 và Rằm lớn nhất nên đứng sau sự kiện có tên cùng ngày. |
| `recurrence` | string | **Mới 2.2.0.** Kiểu quy tắc lặp: `solar`, `lunar`, `lunar_monthly`, `lunar_last_day`, `solar_term`. Mùng 1 / Rằm là `lunar_monthly`; dùng nó để nhận biết thay vì so `code`. Decode kiểu mở. |

`occurrences[]`:

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `event_code` | string | Trỏ về `events[].code`. |
| `solar_date` | string | Ngày dương `YYYY-MM-DD`, luôn nằm trong `year`. |
| `lunar_date` | object | `{day, month, year, is_leap_month}`: ngày âm tương ứng. `is_leap_month` là `true` khi rơi vào tháng nhuận. |

Ví dụ rút gọn (nội dung văn bản là chỗ giữ chỗ; `2026-02-17` là mùng 1 Tết Bính Ngọ):

```json
{
  "success": true,
  "data": {
    "year": 2026,
    "groups": [{ "code": "folk", "name": "Dân gian" }],
    "events": [
      {
        "code": "tet-nguyen-dan",
        "name": "Tết Nguyên Đán",
        "short_description": "…",
        "long_description": "…",
        "guidance": "…",
        "image_url": null,
        "groups": ["vietnam", "folk"],
        "is_public_holiday": true,
        "is_major": false,
        "sort_order": 2,
        "recurrence": "lunar"
      }
    ],
    "occurrences": [
      {
        "event_code": "tet-nguyen-dan",
        "solar_date": "2026-02-17",
        "lunar_date": { "day": 1, "month": 1, "year": 2026, "is_leap_month": false }
      }
    ]
  }
}
```

### Lưu ý khi hiển thị

- Lễ âm lịch tháng Chạp (Táo quân, Giao thừa) có thể rơi vào tháng 1 hoặc 2 dương của `year`: hiển thị theo `solar_date`, không theo `lunar_date.year`.
- Nhóm có thể được back-office thêm về sau: lấy tên hiển thị từ `data.groups`, đừng hard-code ba mã nhóm.
- Chỉ có tiếng Việt ở bản đầu.
- Ngày vía của các vị Phật / Bồ Tát chỉ lấy từ lịch này: không giữ danh sách ngày vía riêng ở client.

Cấu trúc đầy đủ: [API Reference](/api-reference/). Mã lỗi chung: [Xử lý lỗi](/guides/error-handling/). Spec nghiệp vụ: [Lịch: thiết kế](/reference/nghiep-vu/calendar/design/).
