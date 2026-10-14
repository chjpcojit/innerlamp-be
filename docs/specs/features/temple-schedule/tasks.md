# Tasks — temple-schedule

Version: 1.1
Last updated: 2026-10-13
Status: Phase 1–5 đã code + test (domain, migration, repository, seed + dữ liệu mock, usecase, handler, wiring, openapi); Phase 6 còn kiểm tra `curl` bằng tay và `api-contract.md`
Module: `temple-schedule`

> See `changelog.md` in this folder for the version history of this spec.

Thứ tự theo `design.md`: logic thuần trước (không DB), rồi schema, repository, seed + dữ liệu, usecase, delivery, wiring. Mỗi phase viết test cùng lúc (`golang-clean-arch-tdd`); đối chiếu TS1–TS19 ở `requirements.md`. Đường dẫn tính từ `services/appService/` trừ khi có ghi khác. Không có phase config vì feature không cần khối config mới.

## Phase 1 — Domain (thuần)

- [x] `domain/temple_schedule.go`: `TimeOfDay` (phút từ 00:00; `ParseTimeOfDay("HH:MM")`, `String()`), `TempleScheduleItem`, `TempleScheduleStep` (không tag JSON)
- [x] `domain/temple_schedule.go`: `ErrInvalidTempleSchedule`; dùng lại `ErrTempleNotFound`
- [x] `domain/temple_schedule.go`: `TempleScheduleRepository{ListActiveByTemple, ReplaceCatalogue}` và `TempleScheduleCatalogue{Source, Items}`
- [x] `domain/temple_schedule.go`: `TempleScheduleCatalogue.Verify() error` bọc `ErrInvalidTempleSchedule`, thông điệp liệt kê mọi lỗi kèm id buổi (TS17)
- [x] `domain/temple_schedule.go`: `SortTempleSchedule(items)` — buổi theo `start_time`, `sort_order`, `id`; bước theo `position`; `weekdays` bỏ trùng, sắp, đủ 7 → rỗng (TS6, TS9, TS12)
- [x] `domain/mock_temple_schedule_repository.go`: mock viết tay cho test usecase
- [x] `domain/temple_schedule_test.go` (bảng): parse giờ (hợp lệ, `24:00`, `7:5`, rỗng); sắp xếp (cùng giờ → `sort_order` → `id`; buổi qua nửa đêm; bước 23:00/23:45/00:15 giữ position); `weekdays` (trùng, đủ 7, rỗng); `Verify` (id trùng, name rỗng, `end == start`, thứ ngoài 0..6, bước text rỗng, hai bước cùng position, giờ bước sai)

## Phase 2 — Schema & repository

- [x] `migrations/0034_temple_schedules.sql`: hai bảng + CHECK (name/text không rỗng, `end_time <> start_time`, weekdays 0..6, position ≥ 1) + FK `ON DELETE CASCADE` + index; idempotent (`IF NOT EXISTS`); xác nhận số kế tiếp trước khi tạo
- [x] `repository/postgres_temple_schedule_repository.go`: `ListActiveByTemple` (hai truy vấn: buổi, rồi bước `ANY($1)`; cột liệt kê rõ), `ReplaceCatalogue` (một transaction, kiểm tra `temple_id`, upsert không ghi `is_active`, thay bước của các buổi trong file); `%w` bọc lỗi
- [x] `repository/postgres_temple_schedule_repository_test.go`: **đặt `TEST_DB_NAME=innerlamp_test`** (test repository xoá DB dev); upsert hai lần không trùng; `is_active` tắt tay được giữ sau khi nạp lại (TS16); bước bị thay (TS18); temple_id lạ rollback cả file; `ListActiveByTemple` lọc theo chùa, bỏ `is_active=false` kèm bước; chùa không có buổi trả rỗng và không chạy truy vấn bước
- [x] `docs/specs/db/erd.mmd`, `migrations/README.md`: thêm `temple_schedules`, `temple_schedule_steps` (`erd.svg` không vẽ lại: `erd.mmd` là nguồn, quyết định của chủ sản phẩm 2026-10-13)

## Phase 3 — Seed & dữ liệu

- [x] `cmd/seedtempleschedules/main.go` + `parse.go` + `parse_test.go`: strict decode, `-dry-run` (mẫu `seedcalendarevents`)
- [x] `assets/data/temple-schedules.json`: dữ liệu mock cho 6 chùa (15 buổi, 16 bước) lấy id từ `assets/data/temple.json`; phủ mọi dạng: mỗi ngày, chỉ cuối tuần, qua nửa đêm, không giờ kết thúc, buổi có đủ trường (người hướng dẫn, địa điểm, mô tả, chuẩn bị, bước), buổi chỉ có tên + giờ, bước không có giờ; `source` ghi rõ là mock
- [x] `Makefile`: `ensure-temple-schedules`, `seed-temple-schedules`, thêm vào `ensure-data` (mẫu `ensure-calendar-events`)

## Phase 4 — Usecase

- [x] `usecase/temple_schedule_usecase.go`: `NewTempleScheduleUsecase(templeRepo, scheduleRepo)` (không nhận logger: use case không có gì để log, lỗi được bọc và handler ánh xạ), `ScheduleForTemple` (Step 3–6); bọc lỗi để `errors.Is` hoạt động
- [x] `usecase/temple_schedule_usecase_test.go`: chùa không tồn tại; chùa không có buổi → rỗng, không lỗi; lỗi DB ở mỗi lần đọc; kết quả đã sắp xếp và chuẩn hoá

## Phase 5 — Delivery & wiring

- [x] `delivery/http/temple_schedule_handler.go`: parse `{id}`; DTO response (`""` cho text trống, `null` cho giờ trống, `[]` cho steps trống)
- [x] `delivery/http/errors.go`: kiểm tra `ErrTempleNotFound` → `404 TEMPLE_NOT_FOUND` đã có sẵn (không thêm mã lỗi mới)
- [x] `delivery/http/mock_temple_schedule_usecase_test.go` + `temple_schedule_handler_test.go`: 200, 404, 500, DTO null/rỗng
- [x] `cmd/main.go`: dựng repo/usecase/handler; `mux.HandleFunc("GET /v1/temples/{id}/schedule", authMiddleware.AuthenticateFunc(templeScheduleHandler.Schedule))`; **comment nêu route này cần đăng nhập, khác các route `/v1/temples` công khai**
- [x] `docs/specs/api/openapi.yaml`: route + schema `TempleScheduleResponse` / `TempleScheduleItem` / `TempleScheduleStep`

## Phase 6 — Integration & documentation

- [x] `e2e/temple_schedule_e2e_test.go` (binary thật + Postgres thật, theo `golang-e2e-verification`): 401 không token; 404 chùa lạ; 200 rỗng; buổi + bước đúng thứ tự; buổi tắt không hiện; route thật sự có middleware
- [ ] Chạy server thật + `curl` kiểm tra TS1–TS14 với dữ liệu seed
- [x] `go build ./... && go vet ./... && go test -short ./...` sạch (test repository chỉ chạy với `TEST_DB_NAME=innerlamp_test`)
- [ ] `api-contract.md` cho iOS (đối chiếu với server thật), ghi rõ `weekdays` `0` = Chủ nhật và `[]` = mỗi ngày
- [x] Ghi vào `changelog.md` rằng `docs/knowledge/features/temple-schedule.md` cần tạo bằng `il-okf-writer` sau khi code được kiểm chứng

## Deferred (separate feature, not blocking temple-schedule)

- Admin/backoffice thêm, sửa, tắt buổi và bước, và phân quyền.
- Người hướng dẫn và địa điểm theo từng bước.
- Danh mục có cấu trúc cho sư và địa điểm trong chùa.
- Thời khoá theo mùa (an cư kiết hạ) như khái niệm riêng.
- Nhắc giờ, báo thức, push notification.
- Đa ngôn ngữ cho nội dung thời khoá.
- Thay dữ liệu mock bằng nội dung thật do chùa cung cấp.
