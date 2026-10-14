# Changelog — temple-schedule Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version
first.

## v1.1 (2026-10-13) — implemented Phase 1–5

Code được viết theo TDD (domain → usecase → repository → handler → wiring) và spec được chỉnh cho khớp những gì đã build. Không đổi hành vi so với v1.0; không có AC mới.

- `tasks.md`: tick Phase 1–5 và các mục E2E, `go build/vet/test`, ghi chú OKF của Phase 6; chừa kiểm tra `curl` bằng tay và `api-contract.md`.
- `design.md`: sửa số migration `0035` → **`0034`** (là số kế tiếp thực tế, `temple-event` chưa code); `Verify()` trả `error` bọc `ErrInvalidTempleSchedule` thay vì `[]string`; usecase không nhận `logger`; nêu hình dạng file seed `{meta:{format, source}, items}`.
- `requirements.md`, `flow.md`: chỉ đổi header (version, status) và dòng API contract nay đã có trong openapi. Không đổi nội dung AC hay sơ đồ (đã kiểm tra khớp code).
- Reason: hoàn tất implementation theo `tasks.md`.

Đã làm (cross-file):
- `migrations/0034_temple_schedules.sql`, `migrations/README.md`, `docs/specs/db/erd.mmd` (`erd.svg` không vẽ lại: `erd.mmd` là nguồn).
- `docs/specs/api/openapi.yaml`: route `GET /v1/temples/{id}/schedule`, schema `TempleScheduleItem`/`TempleScheduleStep`/`TempleScheduleEnvelope`.
- `Makefile`: `ensure-temple-schedules`, `seed-temple-schedules`, thêm vào `ensure-data`.
- `assets/data/temple-schedules.json`: dữ liệu mock (6 chùa, 15 buổi, 16 bước).
- Test: `domain/temple_schedule_test.go`, `usecase/temple_schedule_usecase_test.go`, `repository/postgres_temple_schedule_repository_test.go` (đã chạy với `TEST_DB_NAME=innerlamp_test`), `delivery/http/temple_schedule_handler_test.go`, `cmd/seedtempleschedules/parse_test.go`, `e2e/temple_schedule_e2e_test.go` (binary thật + Postgres thật, kiểm chứng route nằm sau middleware JWT).

Follow-ups còn lại:
- ~~`docs/specs/features/temple-event/*` đang ghi migration `0034`~~ → đã đổi sang `0035` khi implement `temple-event` (v1.1 của spec đó).
- `api-contract.md` cho iOS (ghi rõ `weekdays` `0` = Chủ nhật, `[]` = mỗi ngày).
- `docs/knowledge/features/temple-schedule.md` chưa có: gọi `il-okf-writer` sau khi người dùng chạy thử flow thật.

## v1.0 (2026-10-13) — initial

Initial spec written before implementation (prospective), as the work order for building the temple schedule (thời khoá tu tập của chùa). Nguồn nghiệp vụ là `SRS.md` v0.3 (IEEE 29148), đã chốt hết câu hỏi.

- `requirements.md`: US-TSC-001 và AC `TS1…TS19` truy vết sang `FR-*`/`NFR-*` của `SRS.md`; bảng API cho `GET /v1/temples/{id}/schedule` (cần đăng nhập, chưa có trong openapi); Out of scope và Pending; không còn `⚠️ Not confirmed` nào mở.
- `design.md`: bảng Layers & components; mô hình dữ liệu hai bảng cha/con (`temple_schedules`, `temple_schedule_steps`); quy ước thứ tự và chuẩn hoá (bước theo `position`, `weekdays` rỗng = mỗi ngày); response shape; hai Step-by-step Flow (đọc thời khoá một chùa; nạp seed); Known gaps.
- `flow.md`: **mới** — hai sequence diagram mirror 1:1 hai Step-by-step Flow của `design.md`.
- `tasks.md`: Phase 1–6, mọi item `[ ]`; Deferred liệt kê admin BO, người dẫn/địa điểm theo bước, danh mục sư/địa điểm, theo mùa, nhắc giờ, đa ngôn ngữ, dữ liệu thật.
- Reason: yêu cầu mới từ chủ sản phẩm ngày 2026-10-13 (thời khoá tu tập là dữ liệu riêng của chùa, không phải sự kiện; BO sửa được), đã qua BA (`SRS.md`).

Quyết định mới phát sinh khi viết design (không có nguyên văn trong SRS):
- **Hai bảng cha/con** thay vì một cột JSON cho bước, để BO sửa từng bước; khoá chính `(schedule_id, position)` chặn trùng vị trí ở mức DB.
- **Không có khối config mới và không có mã lỗi mới**: thời khoá không có múi giờ, cửa sổ hay giới hạn; lỗi duy nhất là `TEMPLE_NOT_FOUND` có sẵn.
- **Seed thay hoàn toàn bước** của buổi có trong file nhưng không xoá buổi vắng trong file và không ghi `is_active`.
- **`weekdays: []` nghĩa là mỗi ngày**; đủ 7 ngày được chuẩn hoá về `[]`; `0` = Chủ nhật.
- **Không đặt header cache** ở v1.
- Số migration `0035` giả định `temple-event` lấy `0034` (cả hai chưa được viết); phải dùng số kế tiếp thực tế khi tạo.

Cross-file follow-ups (chưa làm ở đây, theo rule của skill):
- `docs/specs/api/openapi.yaml`: thêm route và schema (Phase 5).
- `migrations/0035_temple_schedules.sql` + `migrations/README.md` + `docs/specs/db/erd.mmd`/`erd.svg` (Phase 2).
- `Makefile`: target seed/ensure (Phase 3).
- `assets/data/temple-schedules.json`: dữ liệu mock — chưa tồn tại (Phase 3).
- `docs/knowledge/features/temple-schedule.md`: chưa có; gọi `il-okf-writer` sau khi code + kiểm chứng.
- Không đổi `docs/specs/features/temple/*`, `temple-event/*` và `calendar/*`. Lưu ý `temple-event` và `temple-schedule` cùng thêm route dưới `/v1/temples/{id}/…`; kiểm tra không xung đột khi đăng ký trong `cmd/main.go`.
