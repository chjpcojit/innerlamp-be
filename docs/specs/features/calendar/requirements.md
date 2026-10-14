# Requirements — calendar

Version: 2.1
Last updated: 2026-10-11
Status: US-CAL-001 và US-CAL-002 đã triển khai; US-CAL-002 verified bằng test từng tầng và E2E (binary thật + Postgres thật). Dữ liệu seed là bản nháp chưa đối chiếu nguồn chính thức
Module: `calendar`
Code: `services/appService/`

> See `changelog.md` in this folder for the version history of this spec.
> Nguồn nghiệp vụ của US-CAL-002: `SRS.md` v0.3 (IEEE 29148) trong thư mục này — mọi `FR-*`/`NFR-*` ở đó được truy vết bằng cột "SRS" của bảng AC bên dưới.

## User Story

### US-CAL-001 — Ảnh Phật cho khối Lịch âm

> Là một **người dùng app (iOS)**,
> Tôi muốn **thấy một ảnh Phật/Bồ Tát đổi theo từng ngày trên màn Lịch âm**,
> Để **mỗi ngày mở app có một hình ảnh an lạc khác nhau, và cùng một ngày thì luôn thấy cùng một ảnh**.

Nguồn ảnh: bộ 100 ảnh tại `assets/images/home-calendar/` (Wikimedia Commons, giấy phép tự do, thông tin ghi công ở `manifest.json` cùng thư mục).

### US-CAL-002 — Sự kiện lịch: Phật giáo, Việt Nam, dân gian

> Là một **người dùng đã đăng nhập (iOS)**,
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

### US-CAL-002 (sự kiện lịch) — mới

Đánh số `E1…`, riêng với `1…10` ở trên để các tham chiếu AC cũ trong `tasks.md` không bị đổi nghĩa.

| # | Acceptance criterion | SRS |
|---|---|---|
| E1 | WHEN a `GET /v1/calendar/events` request carries a valid access token and a valid `year`, THE SYSTEM SHALL return `200 OK` with `year` echoed back and every occurrence whose **solar date** falls in that year (1/1 → 31/12). | FR-001 |
| E2 | WHEN `year` is absent or empty, THE SYSTEM SHALL use the current year in the configured time zone (`calendar.timezone`) and return it in `year`. | FR-010 |
| E3 | WHEN `year` is not a whole number, or is outside 1900..2100, THE SYSTEM SHALL return `400 INVALID_YEAR` with `details.field = "year"` and no events. | FR-010 |
| E4 | WHEN the request has no valid access token (không có `Authorization`, token hết hạn, hoặc không phải loại `access`), THE SYSTEM SHALL reject it through the existing JWT middleware and return no events. | FR-011 |
| E5 | WHEN an occurrence is returned, THE SYSTEM SHALL include both `solar_date` (`YYYY-MM-DD`) and `lunar_date` (`day`, `month`, `year`, `is_leap_month`), whichever calendar the event is stored in. | FR-006 |
| E6 | WHEN an event's rule is a **fixed solar date**, THE SYSTEM SHALL place it on that solar month/day in the requested year, and SHALL omit it in a year where that day does not exist (29/2 năm không nhuận). | FR-002 |
| E7 | WHEN an event's rule is a **fixed lunar date**, THE SYSTEM SHALL convert that lunar month/day to its solar date; the event occurs in the regular month unless the rule is marked `is_leap_month`, in which case it occurs only in years that have that leap month. | FR-003 |
| E8 | WHEN an event's rule is **every lunar month** (mùng 1, Rằm), THE SYSTEM SHALL produce one occurrence for every lunar month — leap months included — whose solar date falls in the requested year. | FR-004 |
| E9 | WHEN an event's rule is the **last day of a lunar month** (Giao thừa = cuối tháng Chạp), THE SYSTEM SHALL place it on day 30 if that month has 30 days, otherwise on day 29. | FR-015 |
| E10 | WHEN an event's rule is a **solar term** (tiết khí: Thanh minh, Đông chí…), THE SYSTEM SHALL place it on the solar date, in UTC+7, on which the sun reaches that term's ecliptic longitude. | FR-005 |
| E11 | WHEN a lunar-stored event falls near the turn of the year (Táo quân 23/12 ÂL, Giao thừa), THE SYSTEM SHALL include it in the year of its **solar date**, not of its lunar year. | FR-001 |
| E12 | WHEN occurrences are returned, THE SYSTEM SHALL sort them by `solar_date` ascending, then by the event's display order, then by event `code`, so two requests give the same order. | FR-014 |
| E13 | WHEN an event is returned, THE SYSTEM SHALL include `code`, `name`, `short_description`, `long_description`, `guidance` (lễ nghi), `image_url` (`null` khi chưa có ảnh), `groups` and `is_public_holiday`, with text in the default language (`calendar.default_language`, hiện `vi`). | FR-009 |
| E14 | WHEN an event is returned, THE SYSTEM SHALL give it one or more groups (`buddhist`, `vietnam`, `folk` hoặc nhóm BO thêm sau) and SHALL list the `code` and `name` of every group in the response so the client can label them. | FR-007 |
| E15 | WHEN `is_public_holiday` is true, THE SYSTEM SHALL apply it to the event's own day only; THE SYSTEM SHALL NOT create occurrences for substitute days (nghỉ bù). | FR-008 |
| E16 | WHEN an event is inactive (`is_active = false`), THE SYSTEM SHALL NOT return it. | FR-013 |
| E17 | WHEN `year` is explicit, THE SYSTEM SHALL answer with `Cache-Control: private, max-age=<calendar.events_cache_max_age>`; WHEN `year` is omitted, THE SYSTEM SHALL answer with `Cache-Control: no-cache`. | FR-012 |
| E18 | WHEN the seed command is run, THE SYSTEM SHALL upsert events, translations, groups and event-group links keyed by stable codes — creating no duplicates and overwriting the existing rows with the file's content — inside one transaction, and SHALL write nothing if the file is invalid (mã trùng, thiếu tham số quy tắc, nhóm không tồn tại, khoá lạ). | FR-013 |
| E19 | THE SYSTEM SHALL keep the event's content, groups and `is_public_holiday` in database rows (không hằng số trong code) so that back-office can change them later without a release. | FR-013 |
| E20 | THE SYSTEM SHALL convert solar ↔ lunar dates and compute solar terms so that, for every year in 1900..2100, the result equals the reference tables (Hồ Ngọc Đức lunar tables + a second source), with zero mismatches in the verification sample. | NFR-002 |
| E21 | WHEN two requests carry the same `year` and the data has not changed, THE SYSTEM SHALL return identical `occurrences`, independent of the time of the request. | NFR-003 |

## API contract (confirmed surface for this feature)

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| GET | `/v1/calendar/image?date=YYYY-MM-DD` | public | 200 `{date, id, image_url, author, license, license_url, source_url}` | 400 `INVALID_DATE`, 500 |
| GET | `/v1/calendar/events?year=YYYY` *(spec only)* | JWT `access` | 200 `{year, groups[], events[], occurrences[]}` | 400 `INVALID_YEAR`, 401 (middleware), 500 |

Chi tiết cho client: `api-contract.md` trong thư mục này. Hợp đồng của route thứ hai đã đối chiếu với server thật (2026-10-11).

## Out of scope (this feature boundary, not tracked here)

- Ảnh theo ngày lễ cụ thể (Phật đản, Vu lan…) cho **endpoint ảnh**: `/v1/calendar/image` vẫn chọn ảnh theo `dayNumber mod N`, không biết lịch âm. *(Server nay tính được lịch âm cho sự kiện — US-CAL-002 — nhưng không dùng nó để chọn ảnh.)*
- Upload / quản trị ảnh ngày: bộ ảnh cố định trong repo.
- CDN / object store: ảnh phục vụ qua `/static/images/` (dev-only) cho tới khi có CDN, giống gallery.
- Nghỉ bù và ngày làm bù do Chính phủ điều chỉnh; bảng ngoại lệ theo năm *(quyết định 2026-10-11)*.
- Back-office tạo/sửa sự kiện, nhóm, cờ nghỉ lễ: SRS và spec riêng. Spec này chỉ bảo đảm dữ liệu nằm trong DB để BO làm sau (E19).
- Sự kiện cá nhân của user, nhắc lịch, push notification.
- Lịch của nước khác / quy ước âm lịch Trung Quốc: chỉ lịch âm Việt Nam (UTC+7).
- Nội dung đa ngôn ngữ trong v1: chỉ tiếng Việt; **mô hình dữ liệu** phải cho thêm ngôn ngữ sau mà không đổi sự kiện cũ.

## Pending (not yet implemented)

- Đối chiếu nội dung `assets/data/calendar-events.json` với nguồn chính thức (lịch Phật sự GHPGVN, văn bản nghỉ lễ): hiện là bản nháp, ngày vía Phật/Bồ Tát mới đối chiếu với một trang tổng hợp.
- Bảng tham chiếu độc lập đầy đủ 1900..2100 cho E20 (hiện kiểm bằng các năm đã biết, khứ hồi từng ngày và so với bộ chuyển đổi cùng thuật toán của genealogy).
- `thumbnail_url` của ảnh ngày: chưa có, bộ ảnh chỉ có bản 1200px (~340 KB). Nếu iOS cần ảnh nhỏ thì phải sinh thumbnail (tách việc).
- Ảnh cho sự kiện: seed để trống (`image_url: null`); chủ sản phẩm sẽ bổ sung sau, khi đó phải chốt nguồn, giấy phép và nơi lưu.
- Tên nhóm đa ngôn ngữ: `calendar_groups.name` chỉ có tiếng Việt; thêm ngôn ngữ cần bảng dịch cho nhóm (xem `design.md` — Known gaps).

## Ghi chú kỹ thuật liên quan

- Không có bảng DB cho US-CAL-001: danh sách ảnh nạp từ `manifest.json` nhúng vào binary (`assets/embed.go`), kiểm tra lúc khởi động.
- Cách chọn ảnh: `index = (số ngày kể từ 1970-01-01 của date) mod số ảnh`, ảnh sắp theo tên file. Thêm/bớt ảnh làm xáo trộn ánh xạ ngày → ảnh từ thời điểm đó; chấp nhận được, ghi ở `design.md`.
- 4 trong 7 loại giấy phép là CC BY / CC BY-SA, **bắt buộc ghi công**: client hiện `author` + `license` (liên kết `license_url`/`source_url`) ở nơi hiển thị ảnh.
- US-CAL-002 là **hình dạng khác**: có DB (4 bảng, migration `0031_calendar_events.sql` dự kiến), có seed CLI, route cần JWT. Cùng hình dạng "danh mục đọc + nạp bằng CLI upsert" với `administrative-units` / `temple`, khác ở chỗ route cần đăng nhập và kết quả được **tính** theo năm chứ không đọc nguyên dòng.
- Lưu **quy tắc**, không lưu từng năm: ngày dương của lễ âm lịch đổi mỗi năm, nên lễ âm lưu `lunar_month/lunar_day`, lễ dương lưu `solar_month/solar_day`, lễ tiết khí lưu kinh độ Mặt Trời; cái còn lại được tính khi trả API.
- Năm hợp lệ 1900..2100 là phạm vi đã kiểm chứng của bộ chuyển đổi âm ↔ dương, trùng giới hạn của `/v1/calendar/image`.
- Khớp `docs/specs/features/memorial/requirements.md`: spec đó còn ghi "server không biết lịch âm" — câu đó **không còn đúng cho calendar events** (xem follow-up trong `changelog.md`).

## Open questions (resolved during implementation)

- ~~Có endpoint ảnh cho Lịch âm không, và hướng nào?~~ → Làm endpoint mới `GET /v1/calendar/image` (hướng 2), theo yêu cầu của session iOS. Resolved 2026-10-07.
- ~~Map date → ảnh thế nào?~~ → BE tự chọn: `dayNumber mod N`. Resolved 2026-10-07.
- ~~Ai đổi âm ↔ dương?~~ → Server; lưu theo loại lịch của sự kiện, tính loại còn lại khi trả, trả cả hai. Resolved 2026-10-11.
- ~~Tháng nhuận?~~ → Đánh dấu trong dữ liệu (`is_leap_month`), server tính; mặc định chỉ tháng thường. Resolved 2026-10-11.
- ~~Có phải lưu từng năm?~~ → Không: lưu quy tắc lặp, tính theo năm. Resolved 2026-10-11.
- ~~Tiết khí có trong phạm vi?~~ → Có, tính bằng kinh độ Mặt Trời; bộ tiết khí nào thành sự kiện do người soạn seed chọn, BO bổ sung sau. Resolved 2026-10-11.
- ~~Nghỉ bù / ngoại lệ theo năm?~~ → Ngoài phạm vi. Resolved 2026-10-11.
- ~~Nguồn dữ liệu?~~ → File JSON seed do team thu thập (`assets/data/calendar-events.json`), nạp vào DB bằng CLI; BO làm sau. Resolved 2026-10-11.
- ~~Seed chạy lại?~~ → Seed dùng cho setup lần đầu; ở dev chạy lại thì ghi đè; có BO thì BO cập nhật và không chạy seed (không thêm guard production khi đang develop). Resolved 2026-10-11.
- ~~Đơn vị phân trang, lọc tháng?~~ → Một năm mỗi lần (`?year=`), "load more" = năm kế tiếp, client tự lọc tháng. Resolved 2026-10-11.
- ~~Năm ngoài phạm vi / bỏ trống?~~ → `400 INVALID_YEAR` ngoài 1900..2100; bỏ trống = năm hiện tại. Resolved 2026-10-11.
- ~~Xác thực, cache?~~ → Cần JWT; `private, max-age` khi có `year`, `no-cache` khi bỏ trống. Resolved 2026-10-11.
- ~~Đặt ở đâu?~~ → Mở rộng module `calendar`. Resolved 2026-10-11.
- ~~Nhóm?~~ → Một sự kiện có danh sách nhóm; nhóm là dữ liệu, BO thêm được. Resolved 2026-10-11.
- ~~Cờ nghỉ lễ, mùng 1/Rằm, Giao thừa, nội dung, ngôn ngữ, ảnh~~ → Có cờ `is_public_holiday` (lễ Phật giáo theo lịch chung của GHPGVN); mùng 1/Rằm đưa vào bằng quy tắc lặp; Giao thừa bằng quy tắc "ngày cuối tháng"; nội dung đầy đủ, tiếng Việt, ảnh để trống. Resolved 2026-10-11.

> ⚠️ Not confirmed: iOS có cần `thumbnail_url` (ảnh nhỏ) không — chặn việc sinh thumbnail; v1 chỉ trả `image_url`.
