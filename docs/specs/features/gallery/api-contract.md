# Gallery (ảnh đẹp Phật giáo) — API contract (for iOS)

Version: 1.1
Last updated: 2026-10-07
Status: implemented + verified against a running server (dev DB, 86 ảnh)
Code: `services/appService/delivery/http/gallery_handler.go`, `usecase/gallery_usecase.go`, `repository/postgres_gallery_repository.go`
Data: `assets/data/buddhist_gallery.json` → table `gallery_images` via `make seed-gallery`

Base URL (dev): `http://localhost:8080`. Cả ba endpoint **công khai** — không cần `Authorization`. Chỉ `GET`.

## Envelope (như mọi route khác)

Success: `{"success": true, "data": <payload>}`
Error: `{"success": false, "error": {"code": "UPPER_SNAKE", "message": "...", "type": "business|validation|system"}, "details"?: {...}}`
Rẽ nhánh theo `error.code`, không theo `message`.

## Thứ tự

Theo `sort_order`, trộn bốn loại theo vòng: `CHUA`, `TUONG_PHAT`, `THIEN`, `LE_HOI`, `CHUA`, … Nên `top` và trang 1 của danh sách có đủ các loại. Khi lọc `type`, thứ tự trong loại giữ nguyên.

## Shared type: `GallerySummary`

| Field | Type | Notes |
|---|---|---|
| `id` | string | ổn định, vd. `gallery_001` |
| `type` | string | `CHUA` `TUONG_PHAT` `THIEN` `LE_HOI` — coi là enum mở, gặp giá trị lạ thì hiển thị như mục thường |
| `title` | string | |
| `location` | string | nơi chụp |
| `thumbnail_url` | string | ảnh nhỏ 500px, jpg |
| `image_url` | string | ảnh lớn 1280px, jpg |

Không field nào `null`. URL ảnh là URL tuyệt đối do server trả về, dùng nguyên chuỗi đó. **Dev**: ảnh phục vụ từ file system của máy chạy server qua `/static/images/gallery/...`, host là LAN IP của máy đó (hiện `192.168.1.186:8080`); thiết bị phải cùng Wi-Fi. Đổi mạng thì chạy `make lan-urls`.

## 1. GET /v1/gallery/top

10 ảnh đầu (`gallery.top_limit`). Không tham số.

```json
{"success": true, "data": {"items": [
  {"id": "gallery_001", "type": "CHUA", "title": "Chùa Một Cột", "location": "Hà Nội, Việt Nam",
   "thumbnail_url": "http://192.168.1.186:8080/static/images/gallery/thumb/gallery_001.jpg", "image_url": "http://192.168.1.186:8080/static/images/gallery/full/gallery_001.jpg"}
]}}
```

## 2. GET /v1/gallery?type=&page=

Danh sách phân trang, **20 ảnh mỗi trang** (`gallery.page_size`).

| Query | Notes |
|---|---|
| `page` | 1-based, mặc định 1. Không phải số nguyên dương → `400 INVALID_PAGE` (`details.field = "page"`) |
| `type` | Tuỳ chọn. `CHUA` / `TUONG_PHAT` / `THIEN` / `LE_HOI`. Bỏ trống = mọi loại. Giá trị khác → `400 INVALID_GALLERY_TYPE` (`details.field = "type"`) |

```json
{"success": true, "data": {"items": [ /* GallerySummary x20 */ ], "page": 1, "page_size": 20, "total": 86, "total_pages": 5}}
```

Khi có `type`, `total` / `total_pages` chỉ đếm loại đó. Trang vượt cuối trả `200` với `items: []` (không phải lỗi) — phân trang đến khi gặp `items` rỗng hoặc `page > total_pages`.

## 3. GET /v1/gallery/{id}

`GalleryDetail` = `GallerySummary` + ghi công:

| Field | Type | Notes |
|---|---|---|
| `author` | string | tác giả ảnh |
| `license` | string | `CC BY 2.0`, `CC BY-SA 4.0`, `CC0`, `Public domain`, … |
| `source_url` | string | trang gốc trên Wikimedia Commons |

Giấy phép CC BY / CC BY-SA yêu cầu **ghi công**: màn chi tiết nên hiện `author` + `license` (và liên kết `source_url`).

Không có id → `404 GALLERY_IMAGE_NOT_FOUND`.

## Lưu ý

- Chưa có CDN: URL ảnh chỉ dùng được trong mạng dev (xem `requirements.md`). Client nên cache ảnh.
