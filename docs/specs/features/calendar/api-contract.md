# Calendar image — API contract (for iOS)

Version: 1.1
Last updated: 2026-10-07
Status: implemented + verified against a running server
Code: `services/appService/delivery/http/calendar_handler.go`, `usecase/calendar_usecase.go`

Base URL (dev): `http://localhost:8080`. Công khai — không cần `Authorization`. Chỉ `GET`.
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
