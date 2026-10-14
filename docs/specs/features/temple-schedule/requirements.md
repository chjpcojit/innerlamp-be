# Requirements — temple-schedule

Version: 1.1
Last updated: 2026-10-13
Status: Phase 1–5 đã code (xem `tasks.md`). Dữ liệu ban đầu là **mock** nạp từ file JSON
Module: `temple-schedule`
Code: `services/appService/`

> See `changelog.md` in this folder for the version history of this spec.
> Nguồn nghiệp vụ: `SRS.md` v0.3 (IEEE 29148) trong thư mục này — mọi `FR-*`/`NFR-*` ở đó được truy vết bằng cột "SRS" của bảng AC bên dưới.

## User Story

### US-TSC-001 — Thời khoá tu tập của chùa

> Là một **người dùng đã đăng nhập (iOS)**,
> Tôi muốn **xem thời khoá tu tập hằng ngày của một ngôi chùa: buổi nào, giờ nào, ai hướng dẫn, ở đâu, trình tự ra sao và cần chuẩn bị gì**,
> Để **biết giờ nào đến chùa để tham dự và chuẩn bị cho phù hợp**.

Thời khoá là nếp sinh hoạt thường ngày của chùa, **không phải sự kiện** (không có ngày cụ thể, không có trạng thái hoãn/huỷ, không sinh occurrence) và không liên quan tới `temple-event` hay `calendar`. Phiên bản này chỉ có đường đọc; ghi dữ liệu bằng CLI seed (admin BO và phân quyền làm sau).

## Acceptance Criteria (EARS notation)

Đánh số `TS1…`, mỗi dòng truy vết về `FR-*`/`NFR-*` của `SRS.md`.

| # | Acceptance criterion | SRS |
|---|---|---|
| TS1 | WHEN a request for a temple's schedule carries a valid access token and an existing temple id, THE SYSTEM SHALL return `200 OK` with `temple_id` and `items`. | FR-001 |
| TS2 | WHEN the request has no valid access token (không có `Authorization`, token hết hạn, hoặc không phải loại `access`), THE SYSTEM SHALL reject it through the existing JWT middleware and return no items. | FR-009 |
| TS3 | WHEN no temple has the requested id, THE SYSTEM SHALL return `404 TEMPLE_NOT_FOUND` and no items. | FR-006 |
| TS4 | WHEN the temple exists but has no active item, THE SYSTEM SHALL return `200 OK` with `items: []` (không phải lỗi). | FR-006 |
| TS5 | THE SYSTEM SHALL return only items whose `temple_id` equals the requested temple. | FR-001 |
| TS6 | THE SYSTEM SHALL sort `items` by `start_time` ascending, then by display order (`sort_order`), then by `id`. | FR-001, NFR-002 |
| TS7 | WHEN an item is returned, THE SYSTEM SHALL include `id`, `name`, `start_time` (`HH:MM`, 24 giờ, không ngày), `end_time`, `weekdays`, `description`, `instructor`, `location`, `preparation` and `steps`. | FR-002 |
| TS8 | WHEN an optional field has no value, THE SYSTEM SHALL return `""` for a text field, `null` for `end_time` and `[]` for `steps`, never an invented value. | FR-002 |
| TS9 | WHEN an item has no chosen weekdays, or has all seven, THE SYSTEM SHALL return `weekdays: []`, meaning every day; otherwise THE SYSTEM SHALL return the distinct chosen weekdays ascending (`0` = Chủ nhật … `6` = Thứ bảy). | FR-003 |
| TS10 | WHEN an item has an `end_time` earlier than its `start_time`, THE SYSTEM SHALL return it as is (buổi qua nửa đêm); an `end_time` equal to `start_time` SHALL NOT be accepted at seed. | FR-004 |
| TS11 | WHEN an item is inactive (`is_active = false`), THE SYSTEM SHALL NOT return it nor its steps; switching it off SHALL NOT delete it. | FR-005, FR-010 |
| TS12 | WHEN an item has steps, THE SYSTEM SHALL return them ordered by `position` ascending, each with `position`, `time` (`HH:MM` hoặc `null`) and `text`; THE SYSTEM SHALL NOT order steps by `time`. | FR-010 |
| TS13 | WHEN an item has no step, THE SYSTEM SHALL return `steps: []`. | FR-010 |
| TS14 | WHEN two requests carry the same temple and the data has not changed, THE SYSTEM SHALL return identical `items` in identical order. | NFR-002 |
| TS15 | THE SYSTEM SHALL keep schedule items and their steps in database rows (không hằng số/file nhúng trong code), so back-office can add, edit and switch off each item without a release. | FR-007 |
| TS16 | WHEN the seed command is run, THE SYSTEM SHALL upsert items keyed by stable `id` inside one transaction — creating no duplicates and overwriting existing rows with the file's content — and SHALL NOT write `is_active`, so an item switched off by hand stays off. | FR-008 |
| TS17 | WHEN the seed file is invalid (mã trùng, temple không tồn tại, `name` rỗng, giờ sai định dạng, `end_time` bằng `start_time`, thứ ngoài 0..6, bước có `text` rỗng, giờ bước sai định dạng, hai bước cùng `position`, khoá lạ), THE SYSTEM SHALL write nothing and report the offending item. | FR-008, FR-010 |
| TS18 | WHEN an item is seeded, THE SYSTEM SHALL replace that item's steps with the steps in the file (bước vắng trong file bị xoá khỏi item đó). | FR-008, FR-010 |
| TS19 | THE SYSTEM SHALL return all active items of the temple in one response, without pagination and without a cap on the number of items. | NFR-001 |

## API contract (confirmed surface for this feature)

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| GET | `/v1/temples/{id}/schedule` *(spec only)* | JWT `access` | 200 `{temple_id, items[]}` | 401 (middleware), 404 `TEMPLE_NOT_FOUND`, 500 |

Đã có trong `docs/specs/api/openapi.yaml` (operationId `templeSchedule`) và đã được kiểm chứng bằng E2E với binary thật. Không có tham số query. Hình dạng response chi tiết ở `design.md`.

## Out of scope (this feature boundary, not tracked here)

- Sự kiện có ngày cụ thể (khoá tu, pháp hội, lễ): feature `temple-event`. Hai feature độc lập, không có liên kết dữ liệu.
- Liên kết với sự kiện Phật giáo chung của `calendar`.
- Admin BO thêm/sửa/tắt buổi và bước, và phân quyền: làm sau; bản này chỉ có CLI seed.
- Thời khoá theo ngày âm lịch (14, 30…): đó là sự kiện lặp của `temple-event`; buổi thời khoá chỉ áp dụng theo thứ trong tuần.
- Thời khoá theo mùa (an cư kiết hạ) như một khái niệm: BO sửa buổi khi sang mùa.
- Người hướng dẫn và địa điểm theo **từng bước**: hai trường đó thuộc cả buổi.
- Danh mục có cấu trúc cho người hướng dẫn (thực thể sư) và địa điểm (danh mục điện, phòng): hiện là văn bản tự do.
- Cờ "Phật tử có được tham dự không": phần `description` nêu.
- Nhắc giờ, báo thức, push notification.
- Thay đổi một lần ("tuần này nghỉ công phu chiều"): thông báo, thuộc `temple-event`.
- Nhiều ngôn ngữ: chỉ tiếng Việt.
- Giờ mở cửa của chùa: đã có `operating_hours` trong feature `temple`, không đổi.

## Pending (not yet implemented)

- Phase 6 của `tasks.md`: `erd.svg`, kiểm tra `curl` bằng tay, `api-contract.md` cho iOS.
- File dữ liệu mock `assets/data/temple-schedules.json` (dựa trên `assets/data/temple.json`): Phase 3.
- Thay dữ liệu mock bằng nội dung thật trước khi phát hành cho người dùng thật.

Không còn `⚠️ Not confirmed` nào mở; mọi câu hỏi của `SRS.md` đã được trả lời.

## Ghi chú kỹ thuật liên quan

- Hình dạng: danh mục đọc + nạp bằng CLI upsert + route cần JWT, giống `temple-event` và `calendar` events, nhưng **không có phần tính** (không occurrence, không lịch âm, không múi giờ): giờ chỉ là giờ trong ngày của Việt Nam.
- Route nằm dưới `/v1/temples/{id}/…` nhưng **cần đăng nhập**, khác bốn route `/v1/temples` công khai và khác chi tiết chùa công khai; vì thế thời khoá là route riêng, không nhét vào response chi tiết chùa.
- Hai bảng cha/con: `temple_schedules` (buổi) và `temple_schedule_steps` (bước, khoá chính `(schedule_id, position)`), để BO sửa từng bước mà không viết lại cả buổi.
- Bước sắp theo `position` do người nhập đặt, **không** theo `time`, để một buổi qua nửa đêm (23:00, 23:45, 00:15) giữ đúng trình tự. Giờ của bước chỉ là thông tin, không bắt buộc nằm trong khoảng `start_time`–`end_time` của buổi.
- Thay trường "ghi chú" chung của SRS v0.2 bằng `description` và `preparation` (SRS v0.3).
- Không cần khối config mới: không có múi giờ, cửa sổ hay giới hạn.
- Migration dự kiến sau `temple_events` (xem `design.md`; kiểm tra số lớn nhất hiện có khi viết).

## Open questions (resolved during implementation)

- ~~Thời khoá là sự kiện có loại hay dữ liệu riêng?~~ → Bảng riêng gắn chùa, không phải sự kiện. Resolved 2026-10-13.
- ~~Route công khai hay cần đăng nhập?~~ → Cần đăng nhập; vì chi tiết chùa công khai nên là route riêng. Resolved 2026-10-13.
- ~~Giới hạn số buổi mỗi chùa?~~ → Không giới hạn. Resolved 2026-10-13.
- ~~Cờ Phật tử/xuất gia?~~ → Không; dùng `description`. Resolved 2026-10-13.
- ~~Buổi theo ngày âm lịch?~~ → Không; đó là sự kiện lặp. Resolved 2026-10-13.
- ~~Thời khoá theo mùa?~~ → Ngoài phạm vi; BO sửa buổi. Resolved 2026-10-13.
- ~~Thông tin thêm của một buổi?~~ → Người hướng dẫn, địa điểm, mô tả, cần chuẩn bị, danh sách bước. Resolved 2026-10-13.
- ~~Người hướng dẫn và địa điểm: có cấu trúc hay văn bản tự do, ở mức buổi hay bước?~~ → Văn bản tự do, ở mức buổi. Resolved 2026-10-13.
- ~~Bước sắp theo giờ hay theo vị trí?~~ → Theo `position`; giờ bước tuỳ chọn và mang tính thông tin. Resolved 2026-10-13.
