# Calendar — API contract (for iOS)

Version: 2.1
Last updated: 2026-10-11
Status: `/v1/calendar/image` và `/v1/calendar/events` đã triển khai + đối chiếu với server thật (2026-10-11)
Code: `services/appService/delivery/http/calendar_handler.go`, `usecase/calendar_usecase.go` (ảnh); `calendar_event_handler.go`, `calendar_event_usecase.go` (sự kiện)

Base URL (dev): `http://localhost:8080`. Chỉ `GET`. Route ảnh công khai (không cần `Authorization`); route sự kiện **cần** `Authorization: Bearer <access_token>`.
Envelope như mọi route: `{"success": true, "data": ...}` / `{"success": false, "error": {"code", "message", "type"}, "details"?}`.

## GET /v1/calendar/image?date=YYYY-MM-DD

| Query | Notes |
|---|---|
| `date` | Tuỳ chọn, dương lịch `YYYY-MM-DD` (năm 1900..2100). Bỏ trống = hôm nay theo `Asia/Ho_Chi_Minh`. Sai → `400 INVALID_DATE` (`details.field = "date"`) |

Cùng `date` → luôn cùng ảnh. Ngày liên tiếp → ảnh khác nhau (chu kỳ 100 ngày). Truyền `date` là **ngày dương** đang hiển thị trên màn; server không biết lịch âm.

```json
{"success": true, "data": {
  "date": "2026-10-07",
  "id": "buddha_034",
  "image_url": "http://192.168.1.186:8080/static/images/home-calendar/buddha_034.jpg",
  "author": "Judgefloro",
  "license": "Public domain",
  "license_url": "https://commons.wikimedia.org/wiki/File:0395jfWisdom_Park_Namo_Amitabha_Buddha_New_Manila_Quezon_Cityfvf_06.jpg",
  "source_url": "https://commons.wikimedia.org/wiki/File:0395jfWisdom_Park_Namo_Amitabha_Buddha_New_Manila_Quezon_Cityfvf_06.jpg"
}}
```

(Ví dụ lấy từ server thật cho `2026-10-07`; `2026-10-08` → `buddha_035`.) Không field nào `null`. Ảnh JPEG ≤1200px, ~340 KB, dọc là chủ yếu. Chưa có `thumbnail_url`.

Lỗi: `400 INVALID_DATE` (`details.field = "date"`) khi `date` sai định dạng / ngày không tồn tại / ngoài 1900..2100.

Ghi công: CC BY / CC BY-SA bắt buộc hiện `author` + `license`; `license` = `CC0` / `Public domain` thì không bắt buộc nhưng hiện cũng không sao.

Cache: có `date` → `Cache-Control: public, max-age=86400`; không có `date` → `no-cache`. Không có ETag. Client nên luôn truyền `date` rõ ràng và cache ảnh theo `id`.

Dev: `image_url` dựng từ host của request (thiết bị phải cùng Wi-Fi với máy chạy server); production sẽ trỏ CDN.

---

## GET /v1/calendar/events?year=YYYY

Cần access token. Trả **cả một năm dương** (1/1 → 31/12). "Load more" = gọi năm kế tiếp (`year+1`); gọi ngược về năm trước cũng được. Client tự lọc theo tháng đang xem từ `occurrences`.

| Query | Notes |
|---|---|
| `year` | Tuỳ chọn, số nguyên 1900..2100. Bỏ trống = năm hiện tại theo `Asia/Ho_Chi_Minh` (và `year` trong response là năm đó). Sai → `400 INVALID_YEAR` (`details.field = "year"`) |

Response chuẩn hoá: nội dung mỗi sự kiện xuất hiện **một lần** trong `events`; mỗi lần xảy ra là một phần tử của `occurrences`, trỏ về `events[].code`. Cùng một sự kiện có thể có nhiều occurrence (mùng 1/Rằm mỗi tháng âm).

```json
{"success": true, "data": {
  "year": 2026,
  "groups": [
    {"code": "buddhist", "name": "Phật giáo"},
    {"code": "vietnam",  "name": "Việt Nam"},
    {"code": "folk",     "name": "Dân gian"}
  ],
  "events": [
    {
      "code": "tet-nguyen-dan",
      "name": "Tết Nguyên Đán",
      "short_description": "Mùng 1 Tết, ngày đầu năm âm lịch.",
      "long_description": "Tết Nguyên Đán là lễ tết lớn nhất của người Việt, khởi đầu năm âm lịch mới; gia đình sum họp, cúng tổ tiên, chúc nhau năm mới an lành.",
      "guidance": "Cúng gia tiên, chúc Tết, lễ chùa đầu năm; kiêng cãi vã, quét nhà đổ rác trong ngày.",
      "image_url": null,
      "groups": ["vietnam", "folk"],
      "is_public_holiday": true
    }
  ],
  "occurrences": [
    {"event_code": "tet-duong-lich", "solar_date": "2026-01-01", "lunar_date": {"day": 13, "month": 11, "year": 2025, "is_leap_month": false}},
    {"event_code": "tet-nguyen-dan", "solar_date": "2026-02-17", "lunar_date": {"day": 1, "month": 1, "year": 2026, "is_leap_month": false}}
  ]
}}
```

(Trích từ response thật của server dev, năm 2026: 51 sự kiện, 75 occurrence; rút gọn còn một sự kiện và hai occurrence. `2026-02-17` = mùng 1 Tết Bính Ngọ. Nội dung văn bản là **bản nháp** của dữ liệu seed.)

Quy ước cho client:
- `occurrences` đã **sắp xếp** theo `solar_date` tăng dần, rồi thứ tự hiển thị, rồi `code`; cùng `year` luôn cho cùng kết quả.
- Mọi occurrence đều có **cả** `solar_date` và `lunar_date`; `lunar_date.is_leap_month = true` khi rơi vào tháng nhuận.
- Sự kiện lễ âm lịch tháng Chạp (Táo quân, Giao thừa) có thể rơi vào tháng 1–2 dương của `year` — hiển thị theo `solar_date`, đừng theo `lunar_date.year`.
- `image_url` là `null` khi chưa có ảnh (v1 chưa có ảnh sự kiện); `guidance` là nội dung lễ nghi.
- `groups` của sự kiện là danh sách mã; tên hiển thị lấy từ `groups` ở cấp trên. Nhóm có thể được BO thêm về sau — client không nên hard-code ba mã.
- `is_public_holiday` chỉ đúng cho **ngày của sự kiện**; ngày nghỉ bù không có occurrence.
- Chỉ có tiếng Việt trong v1.

Lỗi: `400 INVALID_YEAR` (`details.field = "year"`) khi `year` không phải số nguyên hoặc ngoài 1900..2100. Thiếu/hết hạn/sai loại token → lỗi xác thực của middleware (như `/v1/lamps`).

Cache: có `year` → `Cache-Control: private, max-age=<giây theo config, dự kiến 86400>`; không có `year` → `no-cache`. Client nên luôn truyền `year` rõ ràng và cache theo năm.
