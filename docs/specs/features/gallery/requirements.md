# Requirements — gallery

Version: 1.1
Last updated: 2026-10-07
Status: implemented (2026-10-06) — 3 route đọc công khai, đã có code + unit/repository test + chạy thật trên binary và Postgres dev; chưa có E2E tự động. Còn 4 marker `⚠️ Not confirmed`.
Module: `content`
Code: `services/appService/`

> See `changelog.md` in this folder for the version history of this spec.

Spec này viết **prospectively** (trước khi có code) và đã được implement cùng ngày; các AC dưới đây khớp code đã chạy. Đây là danh mục **ảnh đẹp Phật giáo** (chùa, tượng Phật, thiền, lễ hội) chỉ đọc, công khai, cùng hình dạng với `temple` / `deity` / `article`: `top`, danh sách phân trang, chi tiết. Dữ liệu nguồn là `assets/data/buddhist_gallery.json` (86 mục: 41 `CHUA`, 21 `TUONG_PHAT`, 12 `THIEN`, 12 `LE_HOI`), ảnh gốc từ Wikimedia Commons kèm tác giả và giấy phép; **ở dev, ảnh được tải về và phục vụ từ file system** (`assets/images/gallery/`, route `/static/images/`) bằng URL trỏ tới LAN IP của máy chạy server, như `temple`/`deity`. Dữ liệu nạp bằng lệnh `cmd/seedgallery` chạy tay, không tự nạp khi service khởi động.

Khác `temple` (danh mục chùa có toạ độ, hạng di tích): gallery là **ảnh**, mỗi mục chỉ có tiêu đề, nơi chụp và thông tin ghi công; không có nội dung bài viết.

> ⚠️ Not confirmed: mã user story của tính năng này chưa được product chốt — spec tạm đánh `US-GAL-01..03` theo tiền lệ `US-DTY-01` của `deity`; đổi khi product chốt số.

---

## US-GAL-01: Xem ảnh nổi bật

> Là một **người dùng app (chưa cần đăng nhập)**,
> Tôi muốn **thấy vài tấm ảnh Phật giáo đẹp ngay ở màn hình chủ**,
> Để **thư giãn và bấm vào xem thêm**.

- WHEN a client sends `GET /v1/gallery/top`, THE SYSTEM SHALL respond `200` with `{"items": [...]}` holding the first `gallery.top_limit` images ordered by `sort_order ASC, id ASC`.
- WHEN the gallery table is empty, THE SYSTEM SHALL respond `200` with `"items": []`, not `null` and not an error.
- THE SYSTEM SHALL order `top` so that it mixes the four types (`CHUA`, `TUONG_PHAT`, `THIEN`, `LE_HOI`) instead of returning only the first type in the source file (xem `sort_order` ở `design.md`).

## US-GAL-02: Duyệt ảnh theo danh sách

> Là một **người dùng app (chưa cần đăng nhập)**,
> Tôi muốn **cuộn danh sách ảnh và lọc theo loại (chùa, tượng Phật, thiền, lễ hội)**,
> Để **tìm đúng loại ảnh mình thích**.

- WHEN a client sends `GET /v1/gallery?page=N`, THE SYSTEM SHALL respond `200` with `items` (at most `gallery.page_size`), `page`, `page_size`, `total`, `total_pages`, ordered by `sort_order ASC, id ASC` (cùng thứ tự với `top`, nên trang 1 bắt đầu bằng đúng các ảnh của `top`).
- WHEN `page` is omitted, THE SYSTEM SHALL treat it as `1`.
- WHEN `page` is not an integer, is less than `1`, or exceeds `MaxInt32`, THE SYSTEM SHALL respond `400 INVALID_PAGE` with `details.field = "page"`.
- WHEN `page` is past the last page, THE SYSTEM SHALL respond `200` with `items: []` and the real `total`.
- WHEN a client sends `GET /v1/gallery?type=T` with `T` one of `CHUA`, `TUONG_PHAT`, `THIEN`, `LE_HOI`, THE SYSTEM SHALL return only images of that type, and `total` / `total_pages` SHALL count only that type.
- WHEN `type` is present but not one of the four values, THE SYSTEM SHALL respond `400 INVALID_GALLERY_TYPE` with `details.field = "type"`.
- WHEN `type` is omitted or empty, THE SYSTEM SHALL return images of every type.

> ⚠️ Not confirmed: "lấy ảnh theo danh sách" được hiểu là danh sách phân trang có lọc `type` tuỳ chọn; chưa xác nhận có cần lọc theo `type` hay một bộ lọc khác (vd. `location`) — chặn tham số query của route danh sách.

## US-GAL-03: Xem chi tiết một ảnh

> Là một **người dùng app (chưa cần đăng nhập)**,
> Tôi muốn **mở một tấm ảnh để xem bản lớn và biết ảnh của ai, giấy phép nào**,
> Để **ngắm ảnh rõ hơn và ghi công đúng tác giả**.

- WHEN a client sends `GET /v1/gallery/{id}` with an existing id, THE SYSTEM SHALL respond `200` with the summary fields plus `author`, `license`, `source_url`.
- WHEN no image has that id, THE SYSTEM SHALL respond `404 GALLERY_IMAGE_NOT_FOUND`.

## API contract (confirmed surface for this feature)

> ⚠️ Not confirmed: contract chưa có trong `docs/specs/api/openapi.yaml` — bảng dưới là đề xuất của spec này, sẽ thành contract khi `openapi.yaml` được cập nhật cùng handler.

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| GET | `/v1/gallery/top` | public | 200 `{items: GallerySummary[]}` | 500 |
| GET | `/v1/gallery?type=&page=` | public | 200 `{items, page, page_size, total, total_pages}` | 400 `INVALID_PAGE`, 400 `INVALID_GALLERY_TYPE`, 500 |
| GET | `/v1/gallery/{id}` | public | 200 `GalleryDetail` | 404 `GALLERY_IMAGE_NOT_FOUND`, 500 |

`GallerySummary`: `id`, `type`, `title`, `location`, `thumbnail_url`, `image_url`. `GalleryDetail` = `GallerySummary` + `author`, `license`, `source_url`. Mọi trường là chuỗi không `null` (nguồn có đủ cả 86 mục).

## Out of scope (this feature boundary, not tracked here)

- Đưa ảnh lên GCS/CDN — ở dev ảnh nằm trên file system của máy dev; chuyển CDN là việc khi deploy (xem `⚠️ Not confirmed` bên dưới).
- Ghi / sửa / xoá ảnh qua API (backoffice) — danh mục chỉ đọc.
- Tìm kiếm full-text, yêu thích, tải ảnh xuống.

## Pending (not yet implemented)

- E2E tự động, render lại `erd.svg`, khái niệm OKF — xem `tasks.md` Phase 4–5.

> ⚠️ Not confirmed: base URL ảnh ở staging/production (CDN/GCS) chưa chốt — `image_url`/`thumbnail_url` hiện là `http://<LAN IP>:8080/static/images/gallery/...` chỉ chạy được trong cùng mạng với máy dev; chặn việc đưa vào môi trường thật.

> ⚠️ Not confirmed: quy trình nạp dữ liệu ngoài dev (staging/production) — `cmd/seedgallery` dùng cấu hình dev như `seeddeities`; chặn việc đưa vào môi trường thật.

## Ghi chú kỹ thuật liên quan

- Bảng mới `gallery_images` (migration `0017_gallery_images.sql`), khoá chính là id chuỗi ổn định của nguồn (`gallery_001`…) để seed upsert và id mà client đã lưu không đổi.
- **Ảnh dev**: `scripts/download_gallery_images.py` tải 86 ảnh về `assets/images/gallery/{full,thumb}/<id>.jpg` (full 1280px, thumb 500px, ~32 MB), ghi `credits.json`, và viết lại `image_url`/`thumbnail_url` trong `buddhist_gallery.json` thành URL LAN; URL Wikimedia gốc giữ ở `remote_image_url`/`remote_thumbnail_url` để chạy lại được. IP đổi theo mạng: `make lan-urls` viết lại host trong file JSON này (cùng temple/deity/article) và seed lại. Server chỉ phục vụ ảnh khi `server.static_images_dir` được đặt (`config.dev.yaml` đã đặt).
- `sort_order` do seeder tính: **xen kẽ** bốn loại theo vòng (CHUA, TUONG_PHAT, THIEN, LE_HOI, CHUA, …) vì file nguồn xếp theo nhóm (41 chùa liền nhau) nên nếu dùng nguyên vị trí thì `top` toàn chùa. Trong cùng một loại, thứ tự file được giữ. Nguồn không có chỉ số phổ biến nên không xếp theo độ nổi tiếng.
- Tái dùng `ErrInvalidPage`, `INVALID_PAGE`, envelope `{success, data}` và `writeDomainError` đã có.
- Route `/v1/gallery/top` là segment literal nên `ServeMux` ưu tiên hơn `/v1/gallery/{id}`.
- Giấy phép CC BY / CC BY-SA yêu cầu ghi công: client nên hiển thị `author` + `license` ở màn chi tiết; vì vậy chúng nằm trong detail.

## Open questions (resolved during implementation)

- (chưa có)
