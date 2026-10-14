# Changelog — temple-event Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version
first.

## v1.1 (2026-10-13) — implemented Phase 1–6

Code được viết theo TDD (domain + resolver → config → migration/repository → usecase → handler → seed → wiring) và E2E với binary thật. Spec được chỉnh cho khớp những gì đã build; không đổi hành vi, không có AC mới.

- `tasks.md`: tick Phase 1–6 và các mục E2E, `go build/vet/test`, ghi chú OKF; chừa kiểm tra `curl` bằng tay và `api-contract.md`.
- `design.md`: migration `0034` → **`0035`** (`temple_schedules` code trước nên lấy `0034`); `Verify(loc) error` thay `Verify()` (cần múi giờ để đọc sự kiện cả ngày) cùng `TempleEvent.Validate(loc)`; `InvalidDateFieldError` để `INVALID_DATE` nêu `from`/`to`; `ListActiveByTemple(ctx, templeID, window)` với bộ lọc thô rộng hơn cửa sổ một–hai ngày (repository không cần biết múi giờ); occurrence cả ngày kết thúc `23:59:59`; hình dạng file seed; CLI seed đọc config kể cả `-dry-run`.
- `requirements.md`, `flow.md`: header, trạng thái API contract (đã có trong openapi), tên tham số `window`/`Verify(loc)` ở `flow.md`. AC `TE1…TE25` không đổi.
- Hai `⚠️ Not confirmed` còn mở **vẫn mở và không đổi**: ngày 29 giữ nguyên ngày 29 trong tháng 30 ngày (đúng như đã code và test); ngưỡng thời gian phản hồi chưa đặt.

Đã làm (cross-file):
- `migrations/0035_temple_events.sql`, `migrations/README.md`, `docs/specs/db/erd.mmd`.
- `docs/specs/api/openapi.yaml`: route `GET /v1/temples/{id}/events`, schema `TempleEvent`, `TempleEventRecurrence`, `TempleEventOccurrence`, `TempleEventsEnvelope`; mã lỗi `INVALID_DATE_RANGE`, `DATE_RANGE_TOO_LARGE`.
- `config/config.go` + `config.dev.yaml`: khối `temple_event`; `Makefile`: `ensure-temple-events`, `seed-temple-events`; `assets/data/temple-events.json` (12 sự kiện mock, 5 chùa).
- Test: domain (resolver, Verify), usecase, repository (`TEST_DB_NAME=innerlamp_test`), handler, seed parse, `e2e/temple_event_e2e_test.go` (dữ liệu sinh theo đồng hồ thật nên không mục).

Follow-ups còn lại:
- `api-contract.md` cho iOS; kiểm tra `curl` bằng tay TE1–TE23.
- `docs/knowledge/features/temple-event.md` chưa có: gọi `il-okf-writer` sau khi chạy thử flow thật.
- Dữ liệu mock có sự kiện một lần ở các tháng 11–12/2026 và 01/2027; chúng hết hạn khỏi API khi qua ngày (TE14), cần đặt lại ngày hoặc thay bằng dữ liệu thật.

## v1.0 (2026-10-13) — initial

Initial spec written before implementation (prospective), as the work order for building temple events. Nguồn nghiệp vụ là `SRS.md` v0.2 (IEEE 29148), đã chốt hết câu hỏi chặn.

- `requirements.md`: US-TEV-001 và AC `TE1…TE25` truy vết sang `FR-*`/`NFR-*` của `SRS.md`; bảng API cho `GET /v1/temples/{id}/events` (cần đăng nhập, chưa có trong openapi); Out of scope và Pending; hai `⚠️ Not confirmed` còn mở (ngày 29 âm lịch, ngưỡng thời gian phản hồi), cả hai có mặc định và không chặn task nào.
- `design.md`: bảng Layers & components; mô hình dữ liệu một bảng `temple_events` có cột lặp; quy tắc sinh occurrence (một lần / weekly / lunar-monthly, ngày 30 → cuối tháng, loại trùng); response shape; hai Step-by-step Flow (đọc sự kiện một chùa; nạp seed); Known gaps.
- `flow.md`: **mới** — hai sequence diagram mirror 1:1 hai Step-by-step Flow của `design.md`.
- `tasks.md`: Phase 1–7, mọi item `[ ]`; Deferred liệt kê admin/phân quyền, bảng ngoại lệ, follow temple, ảnh, đa ngôn ngữ, dữ liệu thật.
- Reason: yêu cầu mới từ chủ sản phẩm ngày 2026-10-13 (sự kiện riêng của chùa, độc lập với sự kiện Phật giáo chung), đã qua BA (`SRS.md`).

Quyết định mới phát sinh khi viết design (không có trong SRS, cần anh biết):
- Tham số cửa sổ là `from`/`to` (`YYYY-MM-DD`), mã lỗi mới `INVALID_DATE_RANGE`, `DATE_RANGE_TOO_LARGE`; tái dùng `INVALID_DATE`.
- Config mới `temple_event.{timezone, default_window_months, max_window_months}` (không tái dùng `calendar.timezone` để hai module không phụ thuộc cấu hình của nhau).
- `temples` không có cột `is_active`, nên "chùa bị ẩn" của SRS FR-010 quy về "không tồn tại" (404).
- Seed không xoá và không ghi `is_active`, theo `calendar`.
- Sự kiện lặp có thời lượng mỗi lần phải dưới 24 giờ; ngày cả ngày chỉ cho `recurrence = NONE`.

Cross-file follow-ups (chưa làm ở đây, theo rule của skill):
- `docs/specs/api/openapi.yaml`: thêm route, schema, ba mã lỗi (Phase 6).
- `migrations/0034_temple_events.sql` + `migrations/README.md` + `docs/specs/db/erd.mmd`/`erd.svg` (Phase 4).
- `Makefile`: target seed/ensure (Phase 4).
- `assets/data/temple-events.json`: dữ liệu mock — chưa tồn tại (Phase 4).
- `docs/knowledge/features/temple-event.md`: chưa có; gọi `il-okf-writer` sau khi code + kiểm chứng.
- Không đổi `docs/specs/features/temple/*` và `docs/specs/features/calendar/*`.
