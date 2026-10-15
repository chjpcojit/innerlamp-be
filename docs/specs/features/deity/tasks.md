# Tasks — deity

Version: 3.0
Last updated: 2026-10-15
Status: in-progress — Phase 1–4 và 7 xong (trừ các mục `[ ]` ghi rõ); Phase 5 còn E2E `top`/`detail` và OKF; **Phase 8–10 (v3.0) chưa làm**; Phase 6 là việc hoãn có chủ đích.
Module: `deity`

> See `changelog.md` in this folder for the version history of this spec.

Mọi mục dưới đây trace về một hàng của bảng "Layers & components" hoặc một Step của "Step-by-step Flow" trong `design.md`. Chỉ `[x]` những gì đã kiểm chứng trong code. Lượt v3.0 (2026-10-15) đã đọc lại `domain/deity.go`, `usecase/deity_usecase.go`, `delivery/http/deity_handler.go`, `image_url.go`, `errors.go`, `cmd/seeddeities/parse.go`, `cmd/main.go`, migration `0015`/`0021`, và chạy `go test -short ./services/appService/cmd/seeddeities/ ./services/appService/usecase/ -run Deit` — usecase pass, `TestParseDeities_RealDataFile` **fail** vì file danh mục vắng.

## Phase 1 — Domain

- [x] `Deity`, `MemorialDay`, `Iconography`, `Mantras`, `DeityPage`, hằng `DeityBuddha`/`DeityBodhisattva`/`DeityDeva` trong `services/appService/domain/deity.go` (không tag)
- [x] `ErrDeityNotFound` trong `domain/deity.go`; tái dùng `ErrInvalidPage` (`domain/temple.go`)
- [x] Interface `DeityRepository` (4 method, thứ tự `sort_order ASC, id ASC` nằm trong hợp đồng)
- [x] `domain/mock_deity_repository.go` cho test usecase

## Phase 2 — Usecase

- [x] `usecase/deity_usecase.go`: `Top`, `ListDeities` (mọi loại, offset, `total_pages`, chặn `page < 1` / `> MaxInt32`, `items` nil → `[]`), `Get`
- [x] Unit tests: `usecase/deity_usecase_test.go`, mock tay, không DB thật

## Phase 3 — Repository và dữ liệu

- [x] `migrations/0007_deities.sql` — bảng `deities`, `CHECK` trên `type`, JSONB + `TEXT[]`, hai index; `0015_deities_deva_type.sql` mở `CHECK` cho `DEVA`
- [x] `repository/postgres_deity_repository.go`: `ListTop`, `List`, `FindByID`, `Upsert` (một transaction, `ON CONFLICT DO UPDATE`)
- [x] `cmd/seeddeities/main.go` + `parse.go`: đọc file, kiểm id/type (3 loại)/`name_vi`, `SortOrder` = vị trí, kết nối bằng config dev, gọi `Upsert`
- [ ] `assets/data/phat_botat.json` (20 mục) có trong repo — hiện bị `.gitignore` và vắng trong checkout này; xem Phase 8
- [x] Tests: `cmd/seeddeities/parse_test.go` (`_Validation`, `_OrderFollowsFile`, `_AcceptsDeva` pass ở `-short`; `_RealDataFile` **fail** khi thiếu file — Phase 8); `repository/postgres_deity_repository_test.go` có mặt, cần Postgres
- [x] Ảnh phục vụ được: đường dẫn gốc tương đối (`0021_relative_image_urls.sql`), `imageHost` ghép `server.public_base_url`, 20 ảnh `full` + 20 `thumbnails` trong `assets/images/buddhas/` (nội dung URL trong file danh mục chưa đối chiếu lại vì file vắng)
- [ ] Đưa seed ra khỏi config dev để dùng được ở staging/production — chặn bởi `⚠️ Not confirmed (product)` (product Appendix A mục 4) trong `requirements.md`

## Phase 4 — Delivery và tài liệu

- [x] `delivery/http/deity_handler.go`: `Top`, `ListDeities`, `Get` + DTO summary/detail (mảng không `null`; URL ảnh là chuỗi qua `imageHost`)
- [x] `delivery/http/errors.go`: `DEITY_NOT_FOUND`, `INVALID_PAGE`, `INVALID_DEITY_TYPE`
- [x] `config`: `DeityConfig`, block `deity:` trong `config.dev.yaml`
- [x] `cmd/main.go`: dựng dây và đăng ký 3 route (không gọi seed)
- [x] Handler tests: `delivery/http/deity_handler_test.go` + `mock_deity_usecase_test.go`
- [x] Ghi `0007` vào `migrations/README.md`
- [x] `docs/specs/api/openapi.yaml` và `web/docs/public/openapi.yaml`: 3 route + schema (tới v2.3)
- [x] Bảng `deities` trong `docs/specs/db/erd.mmd` và `erd.svg`

## Phase 5 — Integration & documentation

- [ ] E2E black-box trong `services/appService/e2e/`: `top`, `detail`, `DEITY_NOT_FOUND`, DB chưa seed trả `items: []` (bộ lọc `type` đã có ở `deity_e2e_test.go`)
- [x] `go build ./...` và `go vet ./services/appService/...` sạch (lượt v2.3)
- [ ] Khái niệm OKF `docs/knowledge/features/deity.md` — `il-okf-writer` tạo sau khi v3.0 ship
- [ ] Rà `api-contract.md`: xoá file sau khi spec deity của website / iOS / Android được duyệt (tiền lệ `temple`, `calendar`) — Phase 10

## Phase 7 — Lọc theo `type` (v2.2, AC 20–22; hoàn tất 2026-10-12)

- [x] Domain: `ErrInvalidDeityType`, `IsValidDeityType`; `DeityRepository.List` có `deityType`; mock cập nhật
- [x] Usecase: `ListDeities(ctx, deityType, page)` — kiểm `page` trước rồi `type`
- [x] Repository: `WHERE type = $n` cho cả đếm và lấy trang; test lọc từng loại
- [x] Handler: đọc `type`; `ErrInvalidDeityType` → `400 INVALID_DEITY_TYPE`
- [x] `openapi.yaml` (2 bản): parameter `type`, response `400 INVALID_DEITY_TYPE`
- [x] `api-contract.md` (client iOS) v1.3
- [x] E2E `TestDeity_TypeFilter_E2E`

## Phase 8 — Dữ liệu, schema và seed (v3.0; AC 5, 15, 16, 18, 33)

Trace: hàng Migration, Data, Seed CLI, Repository (`Upsert`, `MissingCalendarEvents`), Config, Make, ERD của `design.md`; Step 1–6 của flow seed.

- [ ] `.gitignore`: thêm ngoại lệ `!assets/data/phat_botat.json` (cạnh `administrative-units.json`), commit file 20 mục (AC 33); `TestParseDeities_RealDataFile` pass lại
- [ ] File danh mục: thay `lunar_memorial_days` của từng mục bằng `via_event_codes` — **chặn bởi `⚠️ Not confirmed (product)` về ánh xạ ngày vía ↔ sự kiện lịch** trong `requirements.md`; cần cả `assets/data/calendar-events.json` (spec `calendar` E26) để biết mã
- [ ] `migrations/00NN_deity_via_events.sql` (số trống kế tiếp, hôm nay `0037`): bảng `deity_via_events` + hai FK + index `(event_code)`, `DROP COLUMN IF EXISTS lunar_memorial_days`; ghi vào `migrations/README.md`
- [ ] `docs/specs/db/erd.mmd` + `erd.svg`: thêm `deity_via_events`, bỏ `lunar_memorial_days`
- [ ] Domain: `Deity.ViaEventCodes`, bỏ `LunarMemorialDays` / `MemorialDay`; `DeityRepository.MissingCalendarEvents`; mock cập nhật
- [ ] `cmd/seeddeities/parse.go`: `via_event_codes` (không rỗng, không trùng trong một mục); test bảng cho hai lỗi mới
- [ ] `cmd/seeddeities/main.go`: gọi `MissingCalendarEvents` trước `Upsert`, có mã thiếu → `log.Fatalf` liệt kê mã (Step 4)
- [ ] Repository `Upsert`: trong cùng transaction, `DELETE` liên kết của các `id` trong file rồi `INSERT` lại; `MissingCalendarEvents`; test Postgres (`TEST_DB_NAME=innerlamp_test`): thay liên kết khi seed lại, FK lỗi → rollback cả deity, mã thiếu được báo đúng
- [ ] `Makefile`: `ensure-calendar-events` chạy trước `ensure-deities` trong `ensure-data`; `ensure-deities`/`seed-deities` phụ thuộc `ensure-calendar-events`
- [ ] `config.dev.yaml`: `deity.page_size: 12` (AC 5, BR-DTY-03)

## Phase 9 — Đọc: lần vía kế tiếp và hợp đồng (v3.0; AC 2, 8, 12, 22–32, 34)

Trace: hàng Domain / Vía, Usecase, Repository (đọc), Delivery / Handler, DTO, Wiring, API spec, E2E harness; Step 1–7 / 1–6 / 1–6 của ba flow đọc.

- [ ] `domain/deity_via.go`: `PickNextVia`, `ResolveDeityVias`, `NextVia`, `DeityVia`, `DeityWithVia`; unit test chạy **mọi case** của `docs/product/deity/rules/deity-next-via.json` đọc trực tiếp từ `docs/product/` (assert `event`, `solar_date`, `days`; `label` là của client — BR-CAL-13) (AC 32, BR-DTY-06); test riêng: sự kiện không có lần nào trong `Y..Y+1` → `Next = nil`, xếp cuối; `Y = 2100` không resolve `2101`
- [ ] Repository: `deityColumns` kèm `ARRAY(SELECT event_code …) AS via_event_codes` cho `ListTop`/`List`/`FindByID`; bỏ JSONB; test Postgres chạy **mọi case** của `docs/product/deity/rules/deity-order.json` (seed theo `position`/`type`, assert `List` và `ListTop` theo `order`/`featured`) (AC 32, BR-DTY-02)
- [ ] Usecase: constructor nhận `CalendarEventRepository`, `config.CalendarConfig`, `now`; `Top(ctx, date)`, `ListDeities(ctx, type, page, date)`, `Get(ctx, id, date)`; thứ tự kiểm `page` → `type` → `date`; bỏ qua `ListActiveEvents` khi không có mã; sự kiện inactive/không có trong danh sách bị bỏ (AC 24–27); test với mock: `date` rỗng dùng `calendar.timezone`, `date` sai → `ErrInvalidDate` không chạm repo, lỗi `ListActiveEvents` → lỗi bọc
- [ ] Handler + DTO: đọc `date` cho 3 route; `next_via` (summary), `via_events`, `lunar_memorial_days` dẫn xuất (detail); `lunar_date` cùng shape với `GET /v1/calendar/events`; mảng không `null`, `next_via`/`next_occurrence` là `null` khi không có (AC 2, 8, 12, 28–30); test handler cho 400 `INVALID_DATE` `details.field = "date"`
- [ ] `cmd/main.go`: truyền `calendarEventRepo`, `cfg.Calendar`, `time.Now` vào `NewDeityUsecase` (dựng `calendarEventRepo` trước deity nếu cần)
- [ ] `docs/specs/api/openapi.yaml` + `web/docs/public/openapi.yaml`: `page_size` 12, parameter `date` cho 3 route, `400 INVALID_DATE`, schema `DeityVia`, `next_via`, `via_events`, `lunar_memorial_days` `deprecated: true`, ví dụ ảnh không còn IP dev; mô tả tham chiếu `GET /v1/deities/{id}/sutras` cho kinh liên quan (AC 34)
- [ ] E2E (`e2e/harness.go` seed thêm vài sự kiện lịch + liên kết): `top`/list có `next_via` đúng với `date` cố định, detail có `via_events` theo AC 28, `date=2026-02-30` → `400 INVALID_DATE`, sự kiện inactive không xuất hiện, `page_size = 12`
- [ ] `go build ./...`, `go vet ./...`, `go test -short ./...` sạch

## Phase 10 — Dọn và tài liệu sau v3.0

- [ ] Xoá `lunar_memorial_days` khỏi `DeityDetail` (DTO, `openapi.yaml`) — chặn bởi `⚠️ Not confirmed` về thời điểm xoá trong `requirements.md`
- [ ] Xoá `api-contract.md` sau khi spec deity của website / iOS / Android được duyệt
- [ ] Báo `ba-writer` cập nhật cột Backend của `docs/product/deity/product.md` §4.2 (BR-DTY-03, BR-DTY-05, BR-DTY-06, FR-DTY-002, FR-DTY-006 → done) — skill này không sửa `docs/product/`
- [ ] Tài liệu web `web/docs/src/content/docs/guides/catalogues/deities.md`: trang 12, `next_via`, `via_events`, `date`
- [ ] OKF `docs/knowledge/features/deity.md` qua `il-okf-writer` (cùng mục Phase 5)

## Phase 6 / Deferred (separate feature, not blocking deity)

- Tạo/sửa/xoá mục qua backoffice (chưa có spec CRUD).
- Rà soát nội dung tôn giáo và quyền ảnh (product Appendix A mục 1).
- Lọc nhiều `type` cùng lúc; tìm kiếm (product §1.2: không làm).
