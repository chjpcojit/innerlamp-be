# Tasks — memorial

Version: 4.1
Last updated: 2026-10-14
Status: v1 (Phase 1–5, 5 route CRUD, không ảnh) đã implement và kiểm chứng 2026-10-07. v4.0 thêm Phase 8–10 (contract nhắc, kế hoạch nhắc, job gửi nhắc) — mọi mục `[ ]`; Phase 10 bị chặn bởi feature notification (ADR-0001, chưa có spec). Phase 6 (ảnh/GCS) hoãn (`docs/tech-debt/memorial/TD-001`). Phase 7 (vector `death-date-conversion.json`, mô tả `openapi.yaml`) chưa làm.
Module: `memorial`

> See `changelog.md` in this folder for the version history of this spec.

Mỗi mục truy về một dòng trong bảng "Layers & components" hoặc một Step trong `design.md`. Phase 1–5 ghi lại v1 như đã làm (kể cả `timezone`/`add_to_os_reminder`); những gì v4.0 đổi nằm ở Phase 8–10.

## Phase 1 — Domain

- [x] `domain/memorial.go`: entity `Memorial` (`DeathDate *time.Time`, `DeathDay/Month *int`, `DeathLeapMonth *bool`), `Reminder`, enum `Relationship` (10 giá trị), `DeathCalendar`
- [x] Sentinel errors `ErrMemorialNotFound`, `ErrMemorialLimitReached`, `ErrMemorialInvalid` + `MemorialValidationError{Field, Message}`
- [x] Interface `MemorialRepository` (`Create`, `ListByUser`, `FindByIDAndUser`, `Update`, `Delete`; mọi method một mục nhận `id` + `userID`). Đổi so với bản trước: `Delete` chỉ trả `error` và `SetPhotoKey` chưa khai báo — cả hai thay đổi chữ ký thuộc Phase 6 (v1 không có ảnh nên không có gì để trả/ghi)
- [x] `domain/mock_memorial_repository.go`

## Phase 2 — Usecase

- [x] `usecase/memorial_usecase.go`: `Create`, `List`, `Get`, `Update`, `Delete` (`photo_url` luôn `null` ở v1)
- [x] Validation AC3–AC10 (trim tên, enum, hai dạng ngày mất: đầy đủ 1900..hôm nay theo `Asia/Ho_Chi_Minh` hoặc ngày+tháng theo lịch đã nhập với quy tắc solar/lunar của AC6a/6b, `time.LoadLocation`, `HH:mm`, cờ OS cần reminder)
- [x] Unit tests table-driven (`testify`), mock repo: mỗi AC validation, 404 đồng nhất, trần 100

## Phase 3 — Repository

- [x] `migrations/0018_memorials.sql` (idempotent `IF NOT EXISTS`, gồm cột `photo_object_key` NULL; CHECK reminder all-or-none, CHECK OS cần reminder, CHECK hai dạng ngày mất loại trừ nhau + `death_leap_month` chỉ với `lunar`) + index `idx_memorials_user`; cập nhật `migrations/README.md`
- [x] Cập nhật `docs/specs/db/erd.mmd` (bảng `memorials` + quan hệ `users 1—N memorials`)
- [x] Sinh lại `docs/specs/db/erd.svg` từ `erd.mmd` bằng `mmdc` (mermaid-cli) — render thành công nên cú pháp `erd.mmd` hợp lệ; bản svg cũ còn thiếu cả `gallery_images`, bản mới có đủ
- [x] `repository/postgres_memorial_repository.go`: `Create` (tx: khoá `users`, đếm, chèn), `ListByUser`, `FindByIDAndUser`, `Update`, `Delete ... RETURNING`; `WHERE … AND user_id`
- [x] Repository tests (DB thật): trần 100 dưới tải song song, người lạ không đọc/sửa/xoá được, `ON DELETE CASCADE`, CHECK constraints bị DB từ chối

## Phase 4 — Delivery

- [x] DTO + handler `Create`, `List`, `Get`, `Update`, `Delete` trong `delivery/http/memorial_handler.go`
- [x] Map lỗi trong `delivery/http/errors.go` (404/409)
- [x] `config.go` + `config.dev.yaml`: `MemorialConfig{MaxPerUser}`
- [x] Đăng ký 5 route trong `cmd/main.go`, **mỗi route bọc `authMiddleware.AuthenticateFunc`**
- [x] Cập nhật `docs/specs/api/openapi.yaml` và `web/docs/public/openapi.yaml` cùng thay đổi handler (5 route, field `photo_url` luôn `null`)
- [x] Handler tests: thành công, từng nhánh lỗi, body sai/field lạ, hai dạng ngày mất

## Phase 5 — Integration & documentation

- [x] E2E (skill `golang-e2e-verification`): binary thật + Postgres — user B nhận 404 trên mục của A cho cả 5 route; thiếu token → 401 cho cả 5 route (chứng minh wiring middleware); tạo đồng thời > 100
- [x] `go test -short ./...` và `go vet ./...` sạch (toàn repo). Chạy đủ không `-short`: chỉ `TestSutraRepository_ShippedData` fail, không liên quan memorial (số `DeityLinks` của dữ liệu seed kinh: 191 thực tế so với 151 mong đợi)
- [ ] Marker `⚠️ Not confirmed` còn lại (bucket GCS) chỉ cần chốt trước Phase 6; ghi vào `changelog.md` khi chốt
- [x] Ghi vào changelog: concept OKF `docs/knowledge/features/memorial.md` cần tạo bởi `il-okf-writer` sau khi code được kiểm chứng

## Phase 6 — Ảnh (US-MEM-05) ⏸ HOÃN, chặn bởi GCS

Chỉ bắt đầu khi GCS dùng được và marker bucket/ký URL đã chốt.

- [ ] `domain/memorial.go`: port `MemorialPhotoStore` (`Put`/`Delete`/`SignedURL`), sentinel `ErrPhotoTooLarge`, `ErrPhotoUnsupportedType`; `domain/mock_memorial_photo_store.go`
- [ ] `repository/gcs_memorial_photo_store.go` bọc `pkg/storage` (constructor không mở kết nối; `Connect` ở `main`); `SetPhotoKey` trong repo
- [ ] Usecase `SetPhoto`/`RemovePhoto`, build `photo_url` ở `List`/`Get`, nhánh xoá object ở `Delete` (Step P-1..P-6, D-3); unit tests: thất bại GCS không đổi key trong DB, xoá object lỗi vẫn trả thành công
- [ ] Handler `UploadPhoto`/`DeletePhoto`: `MaxBytesReader` 5 MB, multipart, sniff content type, 404 trước khi đọc body (AC31); map lỗi 413/415
- [ ] `MemorialConfig`: `PhotoMaxBytes`, `PhotoURLTTL`, `PhotoBucket`, `PhotoPrefix`; storage client ở `cmd/main.go`; đăng ký 2 route ảnh bọc `authMiddleware.AuthenticateFunc`
- [ ] `openapi.yaml` (+ bản `web/`) thêm 2 route ảnh; E2E mở rộng cho 7 route
- [ ] Cơ chế dọn object mồ côi (đã chốt "tính sau")

## Phase 7 — Product spec alignment (`docs/product/memorial/product.md`, từ v1.0)

Không đổi hành vi; chỉ test và mô tả contract.

- [ ] `domain/lunarcal/memorial_vectors_test.go`: chạy mọi case của `docs/product/memorial/rules/death-date-conversion.json` (BR-MEM-03, BR-MEM-04, BR-MEM-10; AC46) trên `SolarToLunar`/`LunarToSolar`, đọc file từ gốc repo, không chép — theo dòng "Product vectors" của `design.md`
- [ ] `openapi.yaml` (+ `web/docs/public/openapi.yaml`): mô tả `death_leap_month` (`null` khi có `death_date`; client khôi phục cờ nhuận bằng cách đổi `death_date` dương → âm) và `PUT` thay toàn bộ (thiếu `reminder` = tắt nhắc) — product Appendix B mục 1. (v4.0: bỏ phần mô tả `reminder.timezone` — field bị bỏ ở Phase 8.)

## Phase 8 — Contract nhắc v4.0 (AC9, AC10 bỏ, AC12, AC17a, AC17b)

Đổi hành vi API; làm cùng một PR với migration để code và schema khớp.

- [ ] `migrations/0037_memorial_reminders.sql` (số kế tiếp lúc implement): `users.timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Ho_Chi_Minh'`; `memorials.next_reminder_at TIMESTAMPTZ`, `next_anniversary_date DATE`; drop `chk_memorials_os_needs_reminder`, `chk_memorials_reminder_all_or_none`; drop cột `reminder_timezone`, `add_to_os_reminder`; CHECK all-or-none 2 cột, `chk_memorials_plan_pair`, `chk_memorials_plan_needs_reminder`; index `idx_memorials_next_reminder` (partial). Idempotent; không backfill bằng SQL (Step J-2 lập kế hoạch). Cập nhật `migrations/README.md`
- [ ] `docs/specs/db/erd.mmd` + sinh lại `erd.svg` bằng `mmdc`
- [ ] `domain/memorial.go`: `Reminder{DaysBefore, Time}` (bỏ `Timezone`), bỏ `AddToOSReminder` ở `Memorial`/`MemorialInput`/`ReminderInput`, thêm `Plan *ReminderPlan`, `ReminderLeadChoices`, `ReminderTimeChoices`
- [ ] `usecase/memorial_usecase.go`: validation AC9 theo tập lựa chọn; bỏ nhánh `timezone` (`time.LoadLocation`) và AC10; unit tests table-driven cho từng giá trị hợp lệ/không hợp lệ
- [ ] `repository/postgres_memorial_repository.go`: bỏ cột cũ khỏi SQL; đọc `next_*` vào `Plan`; repository test: dòng có `reminder_time` lẻ phút / `days_before` 14 vẫn đọc được (AC17a); CHECK mới bị DB từ chối khi vi phạm
- [ ] `delivery/http/memorial_handler.go`: DTO `reminder{days_before, time}`, bỏ `add_to_os_reminder`; handler tests: gửi `add_to_os_reminder` hoặc `reminder.timezone` → 400 `VALIDATION_ERROR` (AC12); response không có `next_*` (AC17b)
- [ ] `openapi.yaml` (+ `web/docs/public/openapi.yaml`): schema `Memorial`/`MemorialRequest` theo v4.0, tăng major `info.version`, ghi chú phá vỡ tương thích (sản phẩm chưa phát hành)
- [ ] E2E `e2e/memorial_e2e_test.go`: tạo/sửa với lựa chọn mới; field cũ bị từ chối

## Phase 9 — Lập kế hoạch nhắc (US-MEM-06: AC34–AC39, AC46)

Không cần feature notification.

- [ ] `domain/memorial_schedule.go`: `AnniversaryKey`, `NextAnniversary`, `PlanReminder`, `ShouldDeliver`, `DaysLeft`, `ReminderPlan` — hàm thuần, `now`/`loc` là tham số (Step C-3e, J-2, J-4)
- [ ] `domain/memorial_schedule_vectors_test.go`: chạy `rules/next-anniversary.json` (chỉ `next_date`) và `rules/reminder-moment.json` (`cases` + `delivery_cases`), đọc từ gốc repo, không chép (AC46, NFR-MEM-03)
- [ ] Unit tests bổ sung ngoài vector: lead/giờ cũ (14 ngày, 07:30); thời điểm đúng bằng `now` (không coi là đã qua); `notBefore` sau khi gửi; mục âm có năm chết trong tháng nhuận
- [ ] `domain.MemorialRepository`: `Create(…, plan PlanFunc)`, `Update(…, plan PlanFunc)`, `ReplanUser`; cập nhật `domain/mock_memorial_repository.go`
- [ ] Repository: `Create` đọc `timezone` trong `SELECT … FOR UPDATE` (Step C-3b/e/f); `Update` thành một tx `FOR SHARE` + UPDATE (Step U-3); `ReplanUser` (Step T-3). Repository tests (DB thật): kế hoạch được ghi cùng tx; lỗi `plan` → rollback, không có dòng nào; `reminder: null` → `next_*` NULL
- [ ] Usecase: clock tiêm vào; dựng `PlanFunc`; `ReplanUser` hiện thực `domain.MemorialReminderReplanner`; unit tests AC34/AC35/AC38
- [ ] Hook `profile` (phụ thuộc spec `profile` có field `timezone` — follow-up): `PATCH /v1/users/me` nhận `timezone`, validate, lưu, gọi `ReplanUser`; lỗi replan → 500 (AC37); `GET /v1/users/me` trả `timezone`; nối dây ở `cmd/main.go`
- [ ] E2E: báo `timezone` = `America/Los_Angeles` rồi tạo mục → kế hoạch đúng giờ địa phương (đọc thẳng DB trong test); báo lại sau lỗi giả lập → kế hoạch được tính lại

## Phase 10 — Job gửi nhắc (US-MEM-07: AC40–AC45) — ⛔ chặn bởi feature notification

Chỉ bắt đầu khi spec + migration notification (ADR-0001) có: bảng inbox với UNIQUE idempotency key, hàm ghi tx-aware, `ReminderPusher`, `pkg/push`/`pkg/push/fcm`. Riêng Step J-2 (`PlanUnplanned`) có thể làm sớm cùng Phase 9.

- [ ] Xoá mục: xoá các lời nhắc memorial của mục trong inbox cùng transaction (AC22, BR-MEM-12 @ product v2.1) qua hàm xoá tx-aware của feature notification
- [ ] `domain/memorial_reminder.go`: `MemorialReminderStore`, `DueReminder`, `DueHandler`, `DueResult`, `MemorialReminderNotice`, `ReminderPusher`, `MemorialReminderReplanner`; mocks
- [ ] `repository/postgres_memorial_reminder_store.go`: `PlanUnplanned` (Step J-2), `ProcessNextDue` (Step J-3 → J-6, một tx mỗi mục, `FOR UPDATE OF m SKIP LOCKED`, ghi inbox cùng tx, `ON CONFLICT DO NOTHING`)
- [ ] Repository tests (DB thật): hai `ProcessNextDue` song song không lấy cùng mục; key đã tồn tại → không tạo thêm, vẫn cuộn kế hoạch (AC42); lỗi UPDATE → rollback cả lời nhắc; mục bị xoá trong khi bị khoá → DELETE chờ, lượt sau không thấy (AC36)
- [ ] `usecase/memorial_reminder_usecase.go`: `MemorialReminderJob.Run` (J-2..J-8), `failedIDs`, `MaxPerRun`, copy push §3.1.1 theo `DaysLeft` lúc gửi và quan hệ (`OTHER` → "người thân"), không có tên (AC43, NFR-MEM-02); unit tests: gửi trễ trong ngày giỗ, bỏ khi đã hết ngày (AC41), lỗi push không đổi kế hoạch, lỗi một mục không dừng lượt (AC44)
- [ ] `config`: `MemorialConfig.Reminder{PlanBatch, MaxPerRun}` + `Validate` + `config.dev.yaml`
- [ ] `cmd/memorialreminders/main.go`: chạy một lần, `time/tzdata`, Connect fail-fast, exit code (Step J-1, J-8); test khởi động với config dev
- [ ] Metrics/log có cấu trúc: claimed, sent, skipped_late, already_sent, failed, overdue_1h (AC45); kiểm không log `full_name`/`note` (AC33)
- [ ] Integration test (testcontainers Postgres): chạy binary job trên dữ liệu seed — một lời nhắc mỗi (mục, kỳ giỗ) qua hai lượt chạy liên tiếp; lượt sau outage gửi bù trong ngày giỗ
- [ ] Ghi vào `docs/` hạ tầng (follow-up, ngoài repo nếu cần): Cloud Scheduler `0 * * * *` → Cloud Run Job `memorial-reminders`, service account riêng, quyền Cloud SQL + FCM (ADR-0002 Follow-ups)
- [ ] Concept OKF `docs/knowledge/features/memorial.md` (bởi `il-okf-writer`, sau khi job chạy end-to-end)

## Deferred (separate feature, not blocking memorial)

- Feature notification (inbox, push token, `pkg/push`) — spec riêng theo ADR-0001; Phase 10 phụ thuộc nó
- Chia sẻ sổ giỗ cho người thân khác
- Xoá tài khoản kéo theo dọn ảnh trên GCS
