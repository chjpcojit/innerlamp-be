---
title: Sổ giỗ chạp
description: "Endpoint quản lý sổ giỗ chạp của người dùng: thêm, xem, sửa, xoá người đã mất kèm ngày giỗ và nhắc nhở."
---

Sổ giỗ chạp là dữ liệu **riêng tư** của từng người dùng. Khác với các danh mục khác, **mọi endpoint đều cần đăng nhập**: gửi `Authorization: Bearer <access_token>` (xem [Xác thực](/guides/authentication/)). Chủ sở hữu luôn là `user_id` trong token; không có endpoint nào nhận user id từ client, và người này không đọc, sửa, xoá được mục của người khác.

| Thao tác | Endpoint |
| --- | --- |
| Thêm | `POST /v1/memorials` |
| Danh sách của tôi | `GET /v1/memorials` |
| Chi tiết | `GET /v1/memorials/{id}` |
| Sửa (thay thế toàn bộ) | `PUT /v1/memorials/{id}` |
| Xoá | `DELETE /v1/memorials/{id}` |

Mỗi người dùng tối đa **100 mục**.

## Server không tính ngày giỗ

Server chỉ **lưu** ngày mất và cấu hình nhắc. Server không tính ngày giỗ hằng năm, không quy đổi âm/dương và không gửi thông báo. App tự quy đổi lịch, tự sắp theo ngày giỗ kế tiếp và tự đặt nhắc local.

## Body khi thêm / sửa

`PUT` là **thay thế toàn bộ**: gửi đủ như khi tạo, field tuỳ chọn không gửi sẽ bị xoá. Field lạ (kể cả `id`, `user_id`, `photo_url`, `created_at`) bị từ chối `400 VALIDATION_ERROR`.

| Field | Kiểu | Bắt buộc | Ghi chú |
| --- | --- | --- | --- |
| `full_name` | string | có | Tối đa 100 ký tự sau khi cắt khoảng trắng đầu cuối. |
| `relationship` | string | có | `GRANDFATHER_PATERNAL`, `GRANDFATHER_MATERNAL`, `GRANDMOTHER_PATERNAL`, `GRANDMOTHER_MATERNAL`, `FATHER`, `MOTHER`, `SPOUSE`, `SIBLING`, `CHILD`, `OTHER`. |
| `death_calendar` | string | có | `solar` hoặc `lunar`: lịch người dùng đã nhập. |
| `death_date` | string | một trong hai dạng | Ngày mất **dương lịch** `YYYY-MM-DD`, từ `1900-01-01` đến hôm nay. Dùng khi **biết năm**. |
| `death_day`, `death_month`, `death_leap_month` | int, int, bool | một trong hai dạng | Dùng khi **không biết năm** (không gửi `death_date`). `lunar`: tháng 1–12, ngày 1–30, `death_leap_month` cho biết tháng nhuận. `solar`: không được `death_leap_month: true`. |
| `note` | string | không | Tối đa 1000 ký tự. Rỗng được lưu thành `null`. |
| `reminder` | object | không | `{days_before (0–30), time ("HH:mm"), timezone}`. `timezone` mặc định `Asia/Ho_Chi_Minh`. Không gửi hoặc `null` = không nhắc. |
| `add_to_os_reminder` | bool | không | Mặc định `false`; `true` chỉ hợp lệ khi có `reminder`. Chỉ là **ý định** của người dùng, app tự ghi vào Reminders/Calendar của máy. |

:::caution
Nếu người dùng nhập ngày **âm lịch** và biết năm, **client phải đổi sang dương lịch** rồi gửi vào `death_date` (`death_calendar` vẫn là `lunar` để nhớ lịch đã nhập).
:::

### Hai dạng ngày mất

```json title="Biết năm"
{
  "full_name": "Nguyễn Văn A",
  "relationship": "FATHER",
  "death_calendar": "lunar",
  "death_date": "2015-03-04",
  "note": "Nhớ ơn cha",
  "reminder": { "days_before": 3, "time": "07:30" },
  "add_to_os_reminder": true
}
```

```json title="Không biết năm, âm lịch, tháng nhuận"
{
  "full_name": "Bà ngoại",
  "relationship": "GRANDMOTHER_MATERNAL",
  "death_calendar": "lunar",
  "death_day": 15,
  "death_month": 7,
  "death_leap_month": true
}
```

## Response (`Memorial`)

Thêm trả `201`, sửa trả `200`, cùng một dạng `data`:

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `id` | int | Dùng cho `GET/PUT/DELETE /v1/memorials/{id}`. |
| `full_name`, `relationship`, `death_calendar` | string | Như đã gửi. |
| `death_date` | string \| null | Có khi biết năm. |
| `death_day`, `death_month`, `death_leap_month` | int/bool \| null | Có khi không biết năm. |
| `note` | string \| null | |
| `photo_url` | null | Chưa có ảnh (đang hoãn): luôn `null`. |
| `reminder` | object \| null | `{days_before, time, timezone}`. |
| `add_to_os_reminder` | bool | |
| `created_at`, `updated_at` | string | UTC, RFC 3339. |

`GET /v1/memorials` trả `data.items` (toàn bộ mục, không phân trang), mới thêm trước. Chưa có mục nào thì `items: []`.

`DELETE` là xoá cứng và trả `{"success": true}` (không có `data`). Gọi lại sau khi xoá trả `404`.

## Lỗi

| HTTP | code | Khi nào |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | Field sai (xem `details.field`, vd. `reminder.time`), body sai JSON, field lạ, sai Content-Type. |
| 401 | `UNAUTHORIZED` | Thiếu token, token sai/hết hạn, hoặc gửi refresh token thay cho access token. |
| 404 | `MEMORIAL_NOT_FOUND` | Id không tồn tại, không phải số nguyên dương, **hoặc thuộc người khác**. Cả ba trả đúng một phản hồi để không lộ id nào tồn tại. |
| 409 | `MEMORIAL_LIMIT_REACHED` | Đã đủ 100 mục. |
| 413 | `PAYLOAD_TOO_LARGE` | Body vượt 1 MB. |

Cấu trúc đầy đủ và nút gọi thử: [API Reference](/api-reference/). Envelope và mã lỗi chung: [Xử lý lỗi](/guides/error-handling/).
