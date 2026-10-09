---
title: Ảnh Phật theo ngày (Lịch âm)
description: "Endpoint trả ảnh Phật theo ngày dương lịch cho màn Lịch âm, kèm ghi công tác giả."
---

Một endpoint **công khai, chỉ đọc**:

```bash
curl 'http://localhost:8080/v1/calendar/image?date=2026-10-07'
```

## Hành vi

- `date` (tuỳ chọn): ngày dương `YYYY-MM-DD`, năm 1900 đến 2100. Sai định dạng hoặc ngoài khoảng trả `400 INVALID_DATE` (`details.field = "date"`).
- Cùng một ngày luôn cùng một ảnh; ngày liên tiếp cho ảnh khác nhau (chu kỳ 100 ngày).
- Bỏ trống `date` là **hôm nay theo giờ `Asia/Ho_Chi_Minh`**; `data.date` trả ngày đã dùng.
- Cache: có `date` thì `Cache-Control: public, max-age=86400`, không có thì `no-cache`.

## Response

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `date` | string | Ngày đã dùng. |
| `id` | string | Vd. `buddha_034`. |
| `image_url` | string | Ảnh đầy đủ. Chưa có `thumbnail_url`. |
| `author`, `license`, `license_url`, `source_url` | string | Ghi công. |

Ảnh CC BY / CC BY-SA bắt buộc hiện `author` và `license` trên màn hình.

Cấu trúc đầy đủ: [API Reference](/api-reference/). Mã lỗi chung: [Xử lý lỗi](/guides/error-handling/).
