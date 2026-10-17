# Requirements — calendar

Version: 3.2
Last updated: 2026-10-17
Status: US-CAL-001 và US-CAL-002 (kể cả v3.0: route công khai, `is_major`, seed vào git) đã triển khai và có test, E2E với binary thật. Còn: đối chiếu nội dung seed với lịch GHPGVN
Module: `calendar`
Code: `services/appService/`

Product spec: `docs/product/calendar/product.md` @ v1.2 (US-CAL-002 only; US-CAL-001 ảnh theo ngày: none yet, product v1.x)
Contract: `docs/specs/api/openapi.yaml` (this repo)
Implements: BR-CAL-01…12, BR-CAL-14, BR-CAL-16, FR-CAL-001, FR-CAL-002, FR-CAL-004, FR-CAL-010, NFR-CAL-01, NFR-CAL-02

> See `changelog.md` in this folder for the version history of this spec.
> Từ v3.0 nguồn nghiệp vụ của US-CAL-002 là product spec ở trên. `SRS.md` và `api-contract.md` cũ đã xoá ở v3.1 (còn trong git history).

## User Story

### US-CAL-001 — Ảnh Phật cho khối Lịch âm

> Là một **người dùng app (iOS)**,
> Tôi muốn **thấy một ảnh Phật/Bồ Tát đổi theo từng ngày trên màn Lịch âm**,
> Để **mỗi ngày mở app có một hình ảnh an lạc khác nhau, và cùng một ngày thì luôn thấy cùng một ảnh**.

Nguồn ảnh: bộ 100 ảnh tại `assets/images/home-calendar/` (Wikimedia Commons, giấy phép tự do, thông tin ghi công ở `manifest.json` cùng thư mục).

### US-CAL-002 — Sự kiện lịch: Phật giáo, Việt Nam, dân gian

> Là một **khách hoặc người dùng đã đăng nhập (website, iOS, Android)**,
> Tôi muốn **xem các sự kiện Phật giáo, ngày lễ của Việt Nam và lễ tết dân gian trên lịch, theo tháng đang xem, và kéo thêm các năm sau**,
> Để **biết ngày Phật đản, Vu lan, Rằm, Tết, 30/4… rơi vào ngày dương/âm nào mà không phải tự tính lịch âm**.

Server giữ **quy tắc lặp** của từng sự kiện (không lưu từng năm) và tính ngày của năm được hỏi; mỗi sự kiện luôn trả cả ngày dương lẫn ngày âm.

## Acceptance Criteria (EARS notation)

### US-CAL-001 (ảnh theo ngày) — đã triển khai, không đổi

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

### US-CAL-002 (sự kiện lịch)

Đánh số `E1…`, riêng với `1…10` ở trên để các tham chiếu AC cũ trong `tasks.md` không bị đổi nghĩa. Từ v3.0 cột cuối là ID của product spec (trước đây là `FR-*` của `SRS.md`). Dòng có ghi **v3.0** là hành vi mới hoặc đổi, chưa triển khai (xem `tasks.md` Phase 14–18).

| # | Acceptance criterion | Product |
|---|---|---|
| E1 | WHEN a `GET /v1/calendar/events` request carries a valid `year`, THE SYSTEM SHALL return `200 OK` with `year` echoed back and every occurrence whose **solar date** falls in that year (1/1 → 31/12). | FR-CAL-002, BR-CAL-02 |
| E2 | WHEN `year` is absent or empty, THE SYSTEM SHALL use the current year in the configured time zone (`calendar.timezone`) and return it in `year`. | FR-CAL-002 |
| E3 | WHEN `year` is not a whole number, or is outside 1900..2100, THE SYSTEM SHALL return `400 INVALID_YEAR` with `details.field = "year"` and no events. | FR-CAL-004, BR-CAL-07 |
| E4 | **v3.0** — WHEN the request carries no `Authorization` header, or carries one that is expired or invalid, THE SYSTEM SHALL serve it exactly as for a signed-in caller (route công khai, không chạy JWT middleware). | FR-CAL-001 |
| E5 | WHEN an occurrence is returned, THE SYSTEM SHALL include both `solar_date` (`YYYY-MM-DD`) and `lunar_date` (`day`, `month`, `year`, `is_leap_month`), whichever calendar the event is stored in. | BR-CAL-08 |
| E6 | WHEN an event's rule is a **fixed solar date**, THE SYSTEM SHALL place it on that solar month/day in the requested year, and SHALL omit it in a year where that day does not exist. | BR-CAL-03 |
| E7 | WHEN an event's rule is a **fixed lunar date**, THE SYSTEM SHALL convert that lunar month/day to its solar date in the regular month, or only in the leap month when the rule is marked `is_leap_month`, and SHALL omit it in a year where that day does not exist. | BR-CAL-04 |
| E8 | WHEN an event's rule is **every lunar month** (`lunar_monthly`), THE SYSTEM SHALL produce one occurrence for every lunar month, leap months included, whose solar date falls in the requested year. | BR-CAL-05 |
| E9 | WHEN an event's rule is the **last day of a lunar month** (`lunar_last_day`), THE SYSTEM SHALL place it on day 30 if that month has 30 days, otherwise on day 29. | BR-CAL-04 |
| E10 | WHEN an event's rule is a **solar term**, THE SYSTEM SHALL place it on the solar date, in UTC+7, on which the sun reaches that term's ecliptic longitude. | BR-CAL-06 |
| E11 | WHEN a lunar-stored event falls near the turn of the year, THE SYSTEM SHALL include it in the year of its **solar date**, not of its lunar year. | BR-CAL-02 |
| E12 | WHEN occurrences are returned, THE SYSTEM SHALL sort them by `solar_date` ascending, then by the event's `sort_order`, then by event `code`. | BR-CAL-12 |
| E13 | WHEN an event is returned, THE SYSTEM SHALL include `code`, `name`, `short_description`, `long_description`, `guidance`, `image_url` (`null` khi chưa có ảnh), `groups`, `is_public_holiday` and, **v3.0**, `is_major`, `sort_order` and `recurrence` (kind của quy tắc lặp), with text in `calendar.default_language` and line breaks of `long_description` / `guidance` kept as stored. | BR-CAL-10, BR-CAL-12, BR-CAL-14, FR-CAL-010 |
| E14 | WHEN an event is returned, THE SYSTEM SHALL give it one or more groups and SHALL list the `code` and `name` of every group in the response, in the catalogue's group order. | BR-CAL-09 |
| E15 | WHEN `is_public_holiday` is true, THE SYSTEM SHALL apply it to the event's own day only and SHALL NOT create occurrences for substitute days. | BR-CAL-11 |
| E16 | WHEN an event is inactive (`is_active = false`), THE SYSTEM SHALL NOT return it. | BR-CAL-16 |
| E17 | **v3.0** — WHEN `year` is explicit, THE SYSTEM SHALL answer with `Cache-Control: public, max-age=<calendar.events_cache_max_age>`; WHEN `year` is omitted, THE SYSTEM SHALL answer with `Cache-Control: no-cache`. | (platform) |
| E18 | WHEN the seed command is run, THE SYSTEM SHALL upsert events, translations, groups and event-group links keyed by stable codes, inside one transaction, and SHALL write nothing if the file is invalid. | FR-CAL-010 |
| E19 | THE SYSTEM SHALL keep each event's content, groups, `sort_order`, `is_public_holiday` and, **v3.0**, `is_major` in database rows, not in code, so they change without a release. | FR-CAL-010, BR-CAL-10 |
| E20 | THE SYSTEM SHALL convert solar ↔ lunar dates and compute solar terms so that, for every year in 1900..2100, the result equals the reference tables, with zero mismatches. | NFR-CAL-01, BR-CAL-07 |
| E21 | WHEN two requests carry the same `year` and the data has not changed, THE SYSTEM SHALL return identical `occurrences`, whoever asks and whenever. | NFR-CAL-02 |
| E22 | **v3.0** — WHEN the request carries an `X-API-Key` header that does not match `server.web_api_key`, THE SYSTEM SHALL return `401` and no events; WHEN the header is absent, THE SYSTEM SHALL serve the request (app di động). | (platform) |
| E23 | **v3.0, đổi ở v3.1** — WHEN a request without a valid `X-API-Key` comes from a client IP that exceeds `calendar.events_rate_limit` within its window, THE SYSTEM SHALL return `429` until the window passes; WHEN the request carries the valid website key, THE SYSTEM SHALL NOT count it against the limit. | (platform) |
| E24 | **v3.0** — WHEN the seed file gives an event `is_major`, THE SYSTEM SHALL store it and overwrite it on every later seed run, like `is_public_holiday`; an event without `is_major` in the file SHALL be stored as `false`. | BR-CAL-10, FR-CAL-010 |
| E25 | **v3.0** — WHEN the seed file gives a `lunar_monthly` event a `sort_order` not greater than that of every other event, THE SYSTEM SHALL refuse the file without writing. | BR-CAL-05 |
| E26 | **v3.0** — THE SYSTEM SHALL keep the seed file `assets/data/calendar-events.json` in version control, so `make ensure-calendar-events` on a fresh clone loads the catalogue. | (platform) |
| E27 | **v3.0** — THE SYSTEM SHALL pass every case of `docs/product/calendar/rules/event-placement.json` and `rules/ordering.json` in its unit tests, loading the files from `docs/product/`, not from a copy. | BR-CAL-01…07, BR-CAL-12 |

Product IDs có cột Backend `n/a` (không thuộc spec này): BR-CAL-13, BR-CAL-15, FR-CAL-003, FR-CAL-005, FR-CAL-006, FR-CAL-007, FR-CAL-009, NFR-CAL-03. FR-CAL-008 (lỗi tải) phía server chỉ là mã lỗi đã có (`INVALID_YEAR`, `429`, `500`); thông báo và retry là việc của client.

## API contract (confirmed surface for this feature)

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| GET | `/v1/calendar/image?date=YYYY-MM-DD` | public | 200 `{date, id, image_url, author, license, license_url, source_url}` | 400 `INVALID_DATE`, 500 |
| GET | `/v1/calendar/events?year=YYYY` | **v3.0**: public, `X-API-Key` của website (tuỳ chọn) + giới hạn theo IP; trước v3.0: JWT `access` | 200 `{year, groups[], events[], occurrences[]}` | 400 `INVALID_YEAR`, 401 (sai `X-API-Key`), 429, 500 |

Hình dạng chi tiết: `docs/specs/api/openapi.yaml` (operation `calendarEvents`).

## Out of scope (this feature boundary, not tracked here)

- Ảnh theo ngày lễ cho **endpoint ảnh**: `/v1/calendar/image` vẫn chọn ảnh theo `dayNumber mod N`.
- Upload / quản trị ảnh ngày; CDN / object store (ảnh qua `/static/images/`, dev-only).
- Nghỉ bù, ngày làm bù, bảng ngoại lệ theo năm (BR-CAL-11).
- Back-office tạo/sửa sự kiện, nhóm, cờ: spec riêng. Spec này chỉ bảo đảm dữ liệu nằm trong DB (E19).
- Nhắc lịch, kinh nên tụng, chùa tổ chức (product spec §1.2: ngoài v1.0).
- Bộ lọc "Lễ lớn", khối Home, đếm ngược, tách dòng nội dung: client làm từ dữ liệu server trả (BR-CAL-13, BR-CAL-15, FR-CAL-003).
- Lịch nước khác; chỉ lịch âm Việt Nam (UTC+7). Nội dung chỉ tiếng Việt; mô hình dữ liệu cho thêm ngôn ngữ sau.

## Pending (not yet implemented)

- Đối chiếu nội dung seed với lịch GHPGVN: chưa có người nhận (product Appendix A). Giữ nhãn `DRAFT`.
- Sáu sự kiện thêm ngày 2026-10-17 cho các vị còn thiếu ngày trong lịch (Tỳ Lô Giá Na, Hư Không Tạng, Kim Cương Thủ, Lục Độ Mẫu, Bạch Độ Mẫu, Kim Cương Tát Đỏa) lấy ngày từ `phat_botat.json`, chưa có nguồn ngoài xác nhận (tìm web không thấy): cần người đối chiếu.
- Bảng tham chiếu độc lập đầy đủ 1900..2100 cho E20 (NFR-CAL-01 `in-progress`).
- `thumbnail_url` của ảnh ngày; ảnh cho sự kiện (seed để `null`, client hiện ảnh mặc định).
- Tên nhóm đa ngôn ngữ (`calendar_groups.name` chỉ tiếng Việt).
- `erd.svg` chưa vẽ lại cho cột `is_major` (`erd.mmd` là nguồn và đã cập nhật).

## Ghi chú kỹ thuật liên quan

- Không có bảng DB cho US-CAL-001: danh sách ảnh nạp từ `manifest.json` nhúng vào binary (`assets/embed.go`), kiểm tra lúc khởi động.
- Cách chọn ảnh: `index = (số ngày kể từ 1970-01-01 của date) mod số ảnh`, ảnh sắp theo tên file. 4 trong 7 loại giấy phép bắt buộc ghi công.
- US-CAL-002 lưu **quy tắc**, không lưu từng năm; ngày được tính khi trả API (product §2.5).
- Từ v3.0 route sự kiện công khai giống `administrative-units`: `webAPIKey` rồi giới hạn theo IP. Không có dữ liệu cá nhân nên không cần JWT (product §3.7). Response giống nhau cho mọi người nên đổi `private` sang `public` để CDN/proxy cache được.
- `recurrence` trả kind của quy tắc (`solar`, `lunar`, `lunar_monthly`, `lunar_last_day`, `solar_term`) để client biết sự kiện nào là Rằm / Mùng 1 (tầng 3 của BR-CAL-15) mà không phải so `code`. `sort_order` trả để client sắp lại khi gộp nhiều năm hoặc chạy `rules/ordering.json`.
- Seed sẽ vào git (bỏ khỏi `.gitignore` bằng ngoại lệ `!assets/data/calendar-events.json`, như `administrative-units.json`). Đã vào git (v3.2): 60 sự kiện, `DRAFT`.
- Năm hợp lệ 1900..2100 là phạm vi đã kiểm chứng của bộ chuyển đổi âm ↔ dương. BR-CAL-07 ≡ BR-MEM-10: memorial và calendar phải cho cùng ngày âm.

## Open questions (resolved during implementation)

- ~~Có endpoint ảnh cho Lịch âm không, và hướng nào?~~ → `GET /v1/calendar/image`. Resolved 2026-10-07.
- ~~Map date → ảnh thế nào?~~ → `dayNumber mod N`. Resolved 2026-10-07.
- ~~Ai đổi âm ↔ dương; tháng nhuận; lưu từng năm?~~ → Server; cờ `is_leap_month`; lưu quy tắc. Resolved 2026-10-11.
- ~~Tiết khí, nghỉ bù, nguồn dữ liệu, seed chạy lại, phân trang, năm sai, đặt ở đâu, nhóm, cờ nghỉ lễ~~ → xem changelog v2.0. Resolved 2026-10-11.
- ~~Xác thực?~~ → **Công khai** (FR-CAL-001), thay quyết định "cần JWT" ngày 2026-10-11. Resolved 2026-10-15.
- ~~Cờ lễ lớn?~~ → Cột `is_major` trên sự kiện, seed nạp, 10 sự kiện ban đầu (BR-CAL-10). Resolved 2026-10-15.
- ~~Rằm / Mùng 1 cùng ngày với lễ có tên?~~ → Lễ có tên trước; seed đưa Rằm / Mùng 1 xuống `sort_order` cuối, `Verify` chặn (E25). Resolved 2026-10-15.
- ~~Ai xuống dòng phần lễ nghi?~~ → Nội dung seed; server trả nguyên văn (E13). Resolved 2026-10-15.
- ~~Bảo vệ route công khai thế nào?~~ → Mặc định của spec này: `webAPIKey` + giới hạn theo IP riêng (`calendar.events_rate_limit`), cùng hình dạng `administrative-units`. Resolved 2026-10-15 (mặc định, merge PR = duyệt).
- ~~Website gọi qua server SvelteKit (một IP) nên bị giới hạn chung?~~ → Request có `X-API-Key` đúng không bị đếm; website tự cache theo năm. Resolved 2026-10-15 (v3.1, phát hiện khi viết spec website).
- ~~Mã nhóm `buddhist` có ổn định?~~ → Có: nhóm Phật giáo cố định (product BR-CAL-09 v1.2); client được dùng mã `buddhist` làm hằng. Resolved 2026-10-15.
- ~~Client nhận biết Rằm / Mùng 1 thế nào?~~ → Trường `recurrence` (mặc định của spec này). Resolved 2026-10-15 (mặc định, merge PR = duyệt).

> ⚠️ Not confirmed (product): ai đối chiếu nội dung seed với lịch GHPGVN và khi nào — chặn bỏ nhãn `DRAFT` của seed (Phase 16); câu hỏi đã nằm ở product Appendix A.

> ⚠️ Not confirmed: iOS có cần `thumbnail_url` (ảnh nhỏ) không — chặn việc sinh thumbnail; v1 chỉ trả `image_url`.
