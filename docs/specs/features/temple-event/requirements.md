# Requirements — temple-event

Version: 1.1
Last updated: 2026-10-13
Status: đã code (xem `tasks.md`). Dữ liệu ban đầu là **mock** nạp từ file JSON
Module: `temple-event`
Code: `services/appService/`

> See `changelog.md` in this folder for the version history of this spec.
> Nguồn nghiệp vụ: `SRS.md` v0.2 (IEEE 29148) trong thư mục này — mọi `FR-*`/`NFR-*` ở đó được truy vết bằng cột "SRS" của bảng AC bên dưới.

## User Story

### US-TEV-001 — Sự kiện của chùa

> Là một **người dùng đã đăng nhập (iOS)**,
> Tôi muốn **xem các sự kiện mà một ngôi chùa sắp tổ chức (khoá tu, pháp hội, lễ, sinh hoạt định kỳ)**,
> Để **biết chùa đó có chương trình gì, vào ngày giờ nào, để quyết định đi chùa**.

Sự kiện chùa là **của riêng từng chùa**, không liên quan tới sự kiện Phật giáo chung trong module `calendar` (quyết định 2026-10-13). Phiên bản này chỉ có đường đọc; ghi dữ liệu bằng CLI seed (admin và phân quyền làm sau).

## Acceptance Criteria (EARS notation)

Đánh số `TE1…`, mỗi dòng truy vết về `FR-*`/`NFR-*` của `SRS.md`.

| # | Acceptance criterion | SRS |
|---|---|---|
| TE1 | WHEN a request for a temple's events carries a valid access token and an existing temple id, THE SYSTEM SHALL return `200 OK` with `temple_id`, the effective `from`/`to` window, `events` and `occurrences`. | FR-001 |
| TE2 | WHEN the request has no valid access token (không có `Authorization`, token hết hạn, hoặc không phải loại `access`), THE SYSTEM SHALL reject it through the existing JWT middleware and return no events. | FR-012 |
| TE3 | WHEN no temple has the requested id, THE SYSTEM SHALL return `404 TEMPLE_NOT_FOUND` and no events. | FR-001 |
| TE4 | WHEN the temple exists but has no event with an occurrence in the window, THE SYSTEM SHALL return `200 OK` with empty `events` and `occurrences` (không phải lỗi). | FR-001 |
| TE5 | THE SYSTEM SHALL return only events whose `temple_id` equals the requested temple. | FR-001 |
| TE6 | WHEN an event is returned, THE SYSTEM SHALL include `id`, `title`, `description`, `kind`, `is_all_day`, `status`, `audience_note`, `contact_info`, `image_url` (`null` khi chưa có) and `recurrence`; WHEN an optional field has no value, THE SYSTEM SHALL return it empty or `null`, never an invented value. | FR-002 |
| TE7 | WHEN an occurrence is returned, THE SYSTEM SHALL include `event_id`, `starts_at`, `ends_at`, `solar_date` (the start date, `YYYY-MM-DD`) and `lunar_date` (`day`, `month`, `year`, `is_leap_month`) of the start date. | FR-002, FR-008 |
| TE8 | WHEN an event does not recur, THE SYSTEM SHALL produce exactly one occurrence from its `starts_at` to its `ends_at`, and SHALL return it for any window that overlaps that span (sự kiện nhiều ngày hiện ở mọi ngày nó trải qua). | FR-003 |
| TE9 | WHEN an event's `ends_at` is on a later calendar day than its `starts_at` (qua nửa đêm), THE SYSTEM SHALL return it as one occurrence that starts on the first day and ends on the later day. | FR-003 |
| TE10 | WHEN an event recurs **weekly** on a weekday, THE SYSTEM SHALL produce one occurrence for each matching date from the event's first date up to its optional `recurrence_until`, at the same time of day as `starts_at`. | FR-004 |
| TE11 | WHEN an event recurs **every lunar month** on given lunar days, THE SYSTEM SHALL produce one occurrence for each of those days in every lunar month of the window, leap months included, at the same time of day as `starts_at`. | FR-004 |
| TE12 | WHEN a lunar-monthly event is set on lunar day 30 and a lunar month has only 29 days, THE SYSTEM SHALL place that occurrence on day 29 (cuối tháng), and SHALL produce at most one occurrence per month per event date (ngày 29 và 30 trong tháng thiếu chỉ ra một lần). The computation is done by the server. | FR-005 |
| TE13 | WHEN a recurring event has a `recurrence_until`, THE SYSTEM SHALL NOT produce an occurrence whose date is after it. | FR-004 |
| TE14 | WHEN an occurrence's end is before the current moment (in the configured time zone), THE SYSTEM SHALL NOT return it; WHEN the occurrence is in progress or later today, THE SYSTEM SHALL return it. | FR-006 |
| TE15 | WHEN an event is `POSTPONED` or `CANCELLED`, THE SYSTEM SHALL still return it with that `status`. | FR-007 |
| TE16 | WHEN an event is inactive (`is_active = false`), THE SYSTEM SHALL NOT return it. | FR-010 |
| TE17 | WHEN the request gives neither `from` nor `to`, THE SYSTEM SHALL use today as `from` and `from` + `temple_event.default_window_months` as `to`. | FR-011 |
| TE18 | WHEN the window is longer than `temple_event.max_window_months`, THE SYSTEM SHALL return `400 DATE_RANGE_TOO_LARGE` and no events. | FR-011 |
| TE19 | WHEN `from` or `to` is not a real calendar date in `YYYY-MM-DD`, or its year is outside 1900..2100, THE SYSTEM SHALL return `400 INVALID_DATE` with `details.field` naming it; WHEN `to` is before `from`, THE SYSTEM SHALL return `400 INVALID_DATE_RANGE`. | FR-011, FR-008 |
| TE20 | WHEN the window ends before today, THE SYSTEM SHALL return `200 OK` with empty `events` and `occurrences`. | FR-011 |
| TE21 | THE SYSTEM SHALL sort `occurrences` by `starts_at` ascending, then by `event_id`, so two requests give the same order. | NFR-002 |
| TE22 | WHEN two requests carry the same temple, window and current moment and the data has not changed, THE SYSTEM SHALL return identical `events` and `occurrences`. | NFR-002 |
| TE23 | THE SYSTEM SHALL compute lunar dates with the same converter as `calendar` (`domain/lunarcal`), so they match its reference for every year 1900..2100. | NFR-003 |
| TE24 | WHEN the seed command is run, THE SYSTEM SHALL upsert temple events keyed by stable `id` inside one transaction — creating no duplicates and overwriting existing rows with the file's content — and SHALL write nothing if the file is invalid (mã trùng, temple không tồn tại, thiếu tham số lặp, `ends_at` trước `starts_at`, `recurrence_until` trước ngày đầu, kind/status lạ, khoá lạ, ngày ngoài 1900..2100). | FR-009 |
| TE25 | THE SYSTEM SHALL keep temple events, their recurrence, status and activation flag in database rows (không hằng số/file nhúng trong code), so back-office can manage them later without a release. | FR-010 |

## API contract (confirmed surface for this feature)

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| GET | `/v1/temples/{id}/events?from=YYYY-MM-DD&to=YYYY-MM-DD` | JWT `access` | 200 `{temple_id, from, to, events[], occurrences[]}` | 400 `INVALID_DATE`, 400 `INVALID_DATE_RANGE`, 400 `DATE_RANGE_TOO_LARGE`, 401 (middleware), 404 `TEMPLE_NOT_FOUND`, 500 |

Đã có trong `docs/specs/api/openapi.yaml` (operationId `templeEvents`) và được kiểm chứng bằng E2E với binary thật. Hình dạng response chi tiết ở `design.md`.

## Out of scope (this feature boundary, not tracked here)

- Liên kết với sự kiện Phật giáo chung của `calendar` (không có `calendar_event_code`): quyết định 2026-10-13.
- Ai tạo/sửa/xoá sự kiện: admin backoffice và phân quyền làm sau; bản này chỉ có CLI seed.
- Đăng ký tham dự trong app, thanh toán, quản lý số chỗ, nhắc lịch, push notification. `contact_info` chỉ là thông tin hiển thị.
- Theo dõi chùa (follow temple) và luồng sự kiện qua nhiều chùa.
- Hoãn/huỷ **một lần** của sự kiện lặp: `status` áp cho cả sự kiện.
- Upload ảnh sự kiện (chưa có object storage): `image_url` chỉ là đường dẫn.
- Nhiều ngôn ngữ: chỉ tiếng Việt.
- Phân trang kết quả: cửa sổ ngày tối đa đã chặn khối lượng (xem `design.md`).

## Pending (not yet implemented)

- Phase 7 của `tasks.md`: kiểm tra `curl` bằng tay, `api-contract.md` cho iOS.
- File dữ liệu mock `assets/data/temple-events.json` (dựa trên `assets/data/temple.json`): Phase 4.
- Thay dữ liệu mock bằng nội dung thật trước khi phát hành cho người dùng thật (Known gap trong `design.md`).

> ⚠️ Not confirmed: an event set on lunar day 29 should also move to "last day" in months with 30 days, or stay on day 29 — chỉ chặn trường hợp biên ngày 29; mặc định giữ nguyên ngày 29 (như thiết kế trong `design.md`).

> ⚠️ Not confirmed: response-time target for one temple and one window (ví dụ p95 < 300 ms) — chỉ chặn ngưỡng đo được của NFR-001; chưa có bài benchmark nào phụ thuộc vào nó.

## Ghi chú kỹ thuật liên quan

- Hình dạng giống `calendar` events: danh mục đọc + nạp bằng CLI upsert + route cần JWT + kết quả được **tính** (mở rộng quy tắc lặp) chứ không đọc nguyên dòng. Khác ở chỗ gắn `temple_id` và có cửa sổ ngày thay cho `year`.
- Route nằm dưới `/v1/temples/{id}/…` nhưng **cần đăng nhập**, khác bốn route `/v1/temples` công khai; ghi rõ ở wiring để không bị hiểu nhầm là sót middleware.
- Ngày âm không lưu; tính khi trả bằng `domain/lunarcal` (đã có, phạm vi 1900..2100, UTC+7).
- Một sự kiện = một giờ bắt đầu. Chùa có hai buổi trong ngày thì nhập thành hai sự kiện.
- Migration là `0035_temple_events.sql` (`0034` đã dùng cho `temple_schedules`).
- `temples` không có cột `is_active`; kịch bản "chùa bị ẩn khỏi catalogue" của `SRS.md` FR-010 quy về "chùa không tồn tại → 404" (TE3). Nếu sau này `temples` có cờ ẩn, thêm vào Step 4 của flow.

## Open questions (resolved during implementation)

- ~~Sự kiện chùa có liên quan sự kiện Phật giáo chung không?~~ → Không, độc lập. Resolved 2026-10-13.
- ~~Ai tạo/sửa sự kiện?~~ → Chưa tính; admin + phân quyền làm sau, bản này seed JSON. Resolved 2026-10-13.
- ~~Một bảng hay tách lịch định kỳ?~~ → Một bảng, có cột lặp; ngoại lệ từng lần để sau. Resolved 2026-10-13.
- ~~Ngày 30 trong tháng thiếu?~~ → Rơi vào ngày cuối tháng, BE tính. Resolved 2026-10-13.
- ~~Có hiện sự kiện đã qua không?~~ → Không. Resolved 2026-10-13.
- ~~Route công khai hay cần đăng nhập?~~ → Cần đăng nhập. Resolved 2026-10-13.
- ~~Huỷ/hoãn có hiện không?~~ → Có, kèm `status`. Resolved 2026-10-13.
- ~~Cửa sổ ngày?~~ → Mặc định và tối đa 3 tháng, lấy từ config. Resolved 2026-10-13.
- ~~Đánh dấu dữ liệu mock?~~ → Không cần (đang develop); thay bằng dữ liệu thật trước khi phát hành. Resolved 2026-10-13.
