# Requirements — calendar

Version: 1.1
Last updated: 2026-10-07
Status: implemented + verified against a running server (curl, port tạm)
Module: `calendar`
Code: `services/appService/`

> See `changelog.md` in this folder for the version history of this spec.

## User Story

### US-CAL-001 — Ảnh Phật cho khối Lịch âm

> Là một **người dùng app (iOS)**,
> Tôi muốn **thấy một ảnh Phật/Bồ Tát đổi theo từng ngày trên màn Lịch âm**,
> Để **mỗi ngày mở app có một hình ảnh an lạc khác nhau, và cùng một ngày thì luôn thấy cùng một ảnh**.

Nguồn ảnh: bộ 100 ảnh tại `assets/images/home-calendar/` (Wikimedia Commons, giấy phép tự do, thông tin ghi công ở `manifest.json` cùng thư mục).

## Acceptance Criteria (EARS notation)

1. WHEN a `GET /v1/calendar/image` request arrives with a valid `date` (`YYYY-MM-DD`), THE SYSTEM SHALL return `200 OK` with `date` echoed back and exactly one image chosen by that date.
2. WHEN two requests carry the same `date`, THE SYSTEM SHALL return the same image, regardless of when, by whom or how often they are made (chọn ảnh là hàm thuần của `date` và danh sách ảnh, không ngẫu nhiên, không lưu trạng thái).
3. WHEN two requests carry consecutive dates, THE SYSTEM SHALL return different images, until the image list wraps (với 100 ảnh: chu kỳ 100 ngày).
4. WHEN the request has no `date` (absent or empty), THE SYSTEM SHALL use today's date in the configured time zone (`calendar.timezone`, dev: `Asia/Ho_Chi_Minh`) and return that date in `date`.
5. WHEN `date` is not a real calendar date in `YYYY-MM-DD` form (sai định dạng, `2026-02-30`, `2026-1-5`), or its year is outside 1900..2100, THE SYSTEM SHALL return `400 INVALID_DATE` with `details.field = "date"`.
6. WHEN an image is returned, THE SYSTEM SHALL include `image_url` as an absolute URL, plus the attribution the licence needs: `author`, `license`, `license_url`, `source_url`.
7. WHEN `calendar.image_base_url` is set, THE SYSTEM SHALL build `image_url` from it; WHEN it is empty, THE SYSTEM SHALL build it from the request's scheme and host followed by `/static/images/home-calendar/<file>` (dev-only, như các ảnh gallery).
8. WHEN the request carries no `Authorization` header, THE SYSTEM SHALL still serve it (route công khai).
9. WHEN the request carries an explicit `date`, THE SYSTEM SHALL answer with `Cache-Control: public, max-age=86400`; WHEN it omits `date`, THE SYSTEM SHALL answer with `Cache-Control: no-cache` (kết quả đổi khi sang ngày mới).
10. WHEN the server starts and the image manifest is empty, malformed, lists a duplicate file, or `calendar.timezone` is not a valid IANA zone, THE SYSTEM SHALL fail at startup rather than at request time.

## API contract (confirmed surface for this feature)

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| GET | `/v1/calendar/image?date=YYYY-MM-DD` | public | 200 `{date, id, image_url, author, license, license_url, source_url}` | 400 `INVALID_DATE`, 500 |

Chi tiết cho client: `api-contract.md` trong thư mục này.

## Out of scope (this feature boundary, not tracked here)

- Tính ngày âm lịch / ngày lễ: client tự làm (xem `memorial/requirements.md` — server không biết lịch âm). Endpoint này chỉ trả ảnh.
- Ảnh theo ngày lễ Phật giáo cụ thể (Phật đản, Vu lan…) — cần lịch âm ở BE, ngoài phạm vi.
- Upload / quản trị ảnh: bộ ảnh cố định trong repo.
- CDN / object store: ảnh phục vụ qua `/static/images/` (dev-only) cho tới khi có CDN, giống gallery.

## Pending (not yet implemented)

- `thumbnail_url`: chưa có, bộ ảnh chỉ có bản 1200px (~340 KB). Nếu iOS cần ảnh nhỏ thì phải sinh thumbnail (tách việc).

## Ghi chú kỹ thuật liên quan

- Không có bảng DB, không migration: danh sách ảnh nạp từ `manifest.json` nhúng vào binary (`assets/embed.go`), kiểm tra lúc khởi động.
- Cách chọn ảnh: `index = (số ngày kể từ 1970-01-01 của date) mod số ảnh`, ảnh sắp theo tên file. Thêm/bớt ảnh làm xáo trộn ánh xạ ngày → ảnh từ thời điểm đó; chấp nhận được, ghi ở `design.md`.
- 4 trong 7 loại giấy phép là CC BY / CC BY-SA, **bắt buộc ghi công**: client hiện `author` + `license` (liên kết `license_url`/`source_url`) ở nơi hiển thị ảnh.
- Route này cùng nhóm với `/v1/gallery/*`: công khai, envelope `utils.ApiResponse`.

## Open questions (resolved during implementation)

- ~~Có endpoint ảnh cho Lịch âm không, và hướng nào?~~ → Làm endpoint mới `GET /v1/calendar/image` (hướng 2), theo yêu cầu của session iOS. Resolved 2026-10-07.
- ~~Map date → ảnh thế nào?~~ → BE tự chọn: `dayNumber mod N`. Resolved 2026-10-07.

> ⚠️ Not confirmed: iOS có cần `thumbnail_url` (ảnh nhỏ) không — chặn việc sinh thumbnail; v1 chỉ trả `image_url`.
