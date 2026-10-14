# Design — memorial

Version: 4.2
Last updated: 2026-10-14
Status: v1 (5 route CRUD, Step C-1..C-4 bản v1 và D-1/D-2/D-4) đã implement và kiểm chứng 2026-10-07. **v4.0 (chưa implement):** contract nhắc mới, kế hoạch nhắc dẫn xuất (Step C-3 mở rộng, flow Sửa, flow Báo múi giờ) và job gửi nhắc hằng giờ (flow J-1..J-8). Phần ảnh/GCS (⏸ Phase 6) vẫn hoãn, nợ ở `docs/tech-debt/memorial/TD-001`. Marker mở: bucket GCS (Step P-2), feature notification chưa có spec (Step J-5, J-7).
Module: `memorial`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Bảy route `/v1/memorials/*` (5 route CRUD đã làm; hai route ảnh ⏸ Phase 6) đều **user-scoped**, bọc `authMiddleware.AuthenticateFunc` (`RequireTokenType: "access"`); `user_id` lấy từ context do middleware gắn (`pkg/jwt/context`), không bao giờ từ request. Feature chạm đủ bốn layer của `services/appService` (`domain` → `usecase` → `repository` / `delivery/http`, nối dây ở `cmd/main.go`) và sở hữu migration `0018_memorials.sql`.

**v4.0 (ADR-0002) thêm phần nhắc từ server:**

- Mỗi mục có nhắc mang hai cột dẫn xuất `next_reminder_at` (UTC) và `next_anniversary_date` (ngày dương của kỳ giỗ lời nhắc đó thuộc về). Chúng được tính bằng Go (hàm thuần trong `domain`, dùng `domain/lunarcal`) và ghi **trong cùng transaction** với lệnh tạo/sửa, khi múi giờ tài khoản được báo, và ngay sau khi job xử lý một lời nhắc (cuộn sang kỳ giỗ sau).
- Múi giờ chuyển lên tài khoản: `users.timezone` (IANA, mặc định `Asia/Ho_Chi_Minh`), client báo qua `PATCH /v1/users/me` (route của `profile`). Cột `memorials.reminder_timezone` và `add_to_os_reminder` bị bỏ.
- Một binary riêng `cmd/memorialreminders` chạy như **Cloud Run Job** do **Cloud Scheduler** gọi mỗi giờ (phút 0): lập kế hoạch cho mục chưa có kế hoạch, rồi nhận từng mục đến hạn bằng `FOR UPDATE SKIP LOCKED`, ghi lời nhắc vào inbox với idempotency key (`memorial_id`, `next_anniversary_date`), cuộn kế hoạch, commit, rồi mới gửi push.
- **Inbox, push token, `pkg/push`/`pkg/push/fcm` thuộc feature notification (ADR-0001)** — chưa có spec. Memorial chỉ định nghĩa những gì nó cần từ feature đó (mục "Phụ thuộc feature notification" bên dưới).

Cho tới Phase 6, memorial không import `pkg/storage` và `photo_url` luôn `null`; cột `photo_object_key` có sẵn (NULL).

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/memorial.go` | `Memorial` (id, userID, fullName, `Relationship`, `DeathDate *time.Time`, `DeathCalendar`, `DeathDay/DeathMonth *int` + `DeathLeapMonth *bool`, `Note *string`, `PhotoObjectKey *string`, `Reminder *Reminder`, timestamps). **v4.0:** `Reminder{DaysBefore int, Time string}` (bỏ `Timezone`); bỏ `AddToOSReminder` khỏi `Memorial` và `MemorialInput`; thêm `Plan *ReminderPlan` (chỉ đọc nội bộ, không ra API). Hằng enum `Relationship` (10 giá trị) và `DeathCalendar`. Không JSON/db tag. |
| Domain / Reminder choices (v4.0) | `domain/memorial.go` | Tập hợp hợp lệ khi **ghi**: `ReminderLeadChoices = {0,1,3,7}`, `ReminderTimeChoices = {"06:00","07:00","12:00","19:00"}` (AC9). Đọc/lập kế hoạch chấp nhận mọi giá trị DB cho phép (AC17a). |
| Domain / Schedule (v4.0) | `services/appService/domain/memorial_schedule.go` | Hàm thuần, không I/O, nhận `now` và `*time.Location` làm tham số: `AnniversaryKey(m) (calendar, day, month)` (mục âm có năm: `lunarcal.SolarToLunar(death_date)`; cờ nhuận bị bỏ qua — BR-MEM-05); `NextAnniversary(key, fromDate) (date, error)` — ngày dương đầu tiên ≥ `fromDate` (BR-MEM-05…07, dùng `lunarcal.LunarToSolar`/`DaysInLunarMonth`); `PlanReminder(m, loc, now, notBefore) (ReminderPlan, error)` — kỳ giỗ đầu tiên ≥ max(hôm nay theo `loc`, `notBefore`) mà thời điểm nhắc (kỳ giỗ − lead, lúc `time`, theo `loc`) **không trước** `now`, nếu không thì kỳ sau (BR-MEM-11); `ShouldDeliver(plan, loc, now) bool` — BR-MEM-13; `DaysLeft(plan, loc, now) int`. `ReminderPlan{At time.Time (UTC), AnniversaryDate time.Time (date)}`. |
| Domain / Errors | `domain/memorial.go` | `ErrMemorialNotFound`, `ErrMemorialLimitReached`, `ErrPhotoTooLarge`, `ErrPhotoUnsupportedType`. `ErrMemorialInvalid` + `*MemorialValidationError{Field, Message}` — handler đọc `Field` để trả `400 VALIDATION_ERROR` kèm `details.field`. |
| Domain / Repository interface | `domain/memorial.go` | `MemorialRepository`: `Create(ctx, m, max, plan PlanFunc)` (khoá `users`, đọc `timezone`, đếm, gọi `plan`, chèn — một tx), `ListByUser`, `FindByIDAndUser`, `Update(ctx, m, plan PlanFunc)` (v4.0: một tx đọc `timezone` + UPDATE), `Delete`, **v4.0** `ReplanUser(ctx, userID, plan PlanFunc)`. `PlanFunc = func(loc *time.Location) (*ReminderPlan, error)` — usecase dựng closure (đã gắn `now`), repository gọi nó bên trong tx sau khi đọc múi giờ dưới khoá (cùng kiểu closure `TokenIssuer` của onboarding). ⏸ Phase 6: `Delete` trả `photo_object_key`, thêm `SetPhotoKey`. |
| Domain / Reminder job ports (v4.0) | `domain/memorial_reminder.go` | `MemorialReminderStore`: `PlanUnplanned(ctx, limit, plan func(m, loc) (*ReminderPlan, error)) (n int, err error)`; `ProcessNextDue(ctx, now, exclude []int64, handle DueHandler) (DueResult, error)` — một tx cho một mục (Step J-3..J-6). `DueReminder{Memorial, Location}`; `DueHandler` trả quyết định thuần (`Deliver bool`, `Notice *MemorialReminderNotice`, `Next *ReminderPlan`). `MemorialReminderNotice{UserID, MemorialID, AnniversaryDate, Relationship, DaysLeft, IdempotencyKey}` — không có tên. `ReminderPusher.Push(ctx, notificationID) error` — cổng do feature notification hiện thực (Step J-7). |
| Domain / Replanner port (v4.0) | `domain/memorial_reminder.go` | `MemorialReminderReplanner.ReplanUser(ctx, userID) error` — để `ProfileUsecase` gọi khi nhận `timezone` mà không import usecase memorial. |
| Domain / Mock | `domain/mock_memorial_repository.go`, **v4.0** `domain/mock_memorial_reminder_store.go`, `domain/mock_reminder_pusher.go` | Mock viết tay cho test usecase. |
| Usecase / CRUD | `services/appService/usecase/memorial_usecase.go` | `Create`, `List`, `Get`, `Update`, `Delete`; **v4.0** validation AC9 (tập lựa chọn), bỏ nhánh `timezone`/`add_to_os_reminder`; dựng `PlanFunc` từ `domain.PlanReminder` với `now` từ clock tiêm vào (`func() time.Time`, để test); `ReplanUser` (hiện thực `MemorialReminderReplanner`). ⏸ Phase 6: `SetPhoto`, `RemovePhoto`. |
| Usecase / Reminder job (v4.0) | `services/appService/usecase/memorial_reminder_usecase.go` | `MemorialReminderJob.Run(ctx, now) (RunStats, error)`: Step J-2 → vòng J-3..J-7, gom thống kê, giới hạn `MaxPerRun`; `handle` gọi `ShouldDeliver`, dựng `MemorialReminderNotice` (copy push theo §3.1.1, `DaysLeft` lúc gửi), `PlanReminder(..., notBefore = next_anniversary_date + 1 ngày)`. Không log tên/ghi chú. |
| Repository / CRUD | `services/appService/repository/postgres_memorial_repository.go` | SQL bảng `memorials`; `Create`/`Update`/`ReplanUser` mỗi cái một tx đọc `users.timezone` dưới khoá (`FOR UPDATE` ở Create, `FOR SHARE` ở Update/ReplanUser); `time.LoadLocation` trên giá trị đọc được; mọi `WHERE` có `user_id`; `NULL` ↔ `nil`; map `reminder_*`, `next_*`. |
| Repository / Reminder store (v4.0) | `services/appService/repository/postgres_memorial_reminder_store.go` | Hiện thực `MemorialReminderStore`: claim `FOR UPDATE OF m SKIP LOCKED`, ghi notification **trong cùng tx** qua hàm ghi tx-aware của repository notification (cùng package `repository`; ⚠️ chờ spec notification), `UPDATE memorials SET next_*`. |
| Delivery / Handler | `services/appService/delivery/http/memorial_handler.go` | `Create`, `List`, `Get`, `Update`, `Delete`; **v4.0** DTO `reminderRequest/Response{days_before, time}`, bỏ `timezone`, bỏ `add_to_os_reminder` (strict decoder tự từ chối field cũ — AC12). ⏸ Phase 6: `UploadPhoto`, `DeletePhoto`. |
| Delivery / Error map | `delivery/http/errors.go` | `ErrMemorialNotFound` → 404; `ErrMemorialLimitReached` → 409; ⏸ Phase 6: 413/415. Không mã lỗi mới ở v4.0. |
| Profile hook (v4.0, thuộc `profile`) | `usecase/profile_usecase.go`, `delivery/http/profile_handler.go` | Field `timezone` trên `PATCH /v1/users/me` (validate `time.LoadLocation`, từ chối `""`/`Local`), lưu `users.timezone`, rồi gọi `MemorialReminderReplanner.ReplanUser` (Step T-1..T-3). **Spec `profile` phải ghi phần này** — follow-up, không sửa ở đây. |
| Job entry point (v4.0) | `services/appService/cmd/memorialreminders/main.go` | Binary chạy một lần rồi thoát (Cloud Run Job `memorial-reminders`): nạp config + secrets như `cmd/main.go`, `Connect` DB fail-fast, dựng repo/usecase/pusher, `Run(ctx, time.Now())`, ghi log/metric, exit 0; exit ≠ 0 chỉ khi không dựng được dependency hoặc mất DB giữa chừng. Import `time/tzdata` (như `cmd/main.go`). Không mở HTTP. |
| Config | `services/appService/config/config.go`, `config.dev.yaml` | `MemorialConfig{MaxPerUser (100)}`; **v4.0** `MemorialConfig.Reminder{PlanBatch (200), MaxPerRun (20000)}` — số 0 bị `Validate` từ chối. ⏸ Phase 6: `PhotoMaxBytes`, `PhotoURLTTL`, `PhotoBucket`, `PhotoPrefix`. |
| Metrics (v4.0) | job | `memorial_reminders_{claimed,sent,skipped_late,already_sent,failed}_total` mỗi lượt + `memorial_reminders_overdue_1h` (số mục `next_reminder_at < now − 1h` lúc kết thúc) — log có cấu trúc trước; `pkg/metrics` khi có exporter. |
| Wiring | `services/appService/cmd/main.go` | Dựng repo → usecase → handler; 5 route CRUD, mỗi route bọc `authMiddleware.AuthenticateFunc`. **v4.0:** tiêm clock vào `MemorialUsecase` và tiêm `memorialUsecase` (port `MemorialReminderReplanner`) vào `ProfileUsecase`. ⏸ Phase 6: storage client + 2 route ảnh. |
| Migration | `migrations/0018_memorials.sql` (v1), **v4.0** `migrations/0037_memorial_reminders.sql` (số kế tiếp lúc implement) | Xem "Schema" bên dưới. |
| API spec | `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`) | 5 route; **v4.0** schema `reminder` thu hẹp, bỏ `timezone`/`add_to_os_reminder`, tăng major `info.version`; `timezone` trên `/v1/users/me` (cùng follow-up `profile`). |
| DB spec | `docs/specs/db/erd.mmd`, `erd.svg` | **v4.0** `users.timezone`, `memorials.next_*`, bỏ 2 cột. |
| Product vectors | `domain/lunarcal/memorial_vectors_test.go` (BR-MEM-03, 04, 10); **v4.0** `domain/memorial_schedule_vectors_test.go` (BR-MEM-05…07, 11, 13) | Đọc `docs/product/memorial/rules/*.json` từ repo (đường dẫn tương đối lên gốc repo, không chép). `death-date-conversion.json` → `SolarToLunar`/`LunarToSolar`. `next-anniversary.json` → chỉ so `expected.next_date` với `NextAnniversary(key, today)` (bỏ qua `days_left`, `ordinal` — việc client). `reminder-moment.json` → `cases` so `anniversary_date` + `reminder_at` (so instant, không so chuỗi offset) với `PlanReminder(…, notBefore = today)`; `delivery_cases` so `ShouldDeliver`. Lệch vector = code sai hoặc vector cần sinh lại qua `ba-writer`; không sửa tay JSON. |

### Schema `memorials` (sau v4.0)

```
id                    BIGSERIAL PRIMARY KEY
user_id               BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE
full_name             VARCHAR(100) NOT NULL
relationship          VARCHAR(24)  NOT NULL CHECK (IN 10 giá trị enum)
death_date            DATE CHECK (death_date >= DATE '1900-01-01')   -- NULL khi năm không rõ
death_calendar        VARCHAR(5)   NOT NULL CHECK (IN ('solar','lunar'))
death_day             SMALLINT CHECK (BETWEEN 1 AND 31)              -- chỉ khi death_date IS NULL
death_month           SMALLINT CHECK (BETWEEN 1 AND 12)
death_leap_month      BOOLEAN
note                  VARCHAR(1000)
photo_object_key      TEXT
reminder_days_before  SMALLINT CHECK (BETWEEN 0 AND 30)   -- giữ rộng cho dòng cũ (AC17a); request mới chỉ 0/1/3/7 (usecase)
reminder_time         TIME                                -- dòng cũ có thể lẻ phút; request mới chỉ 06/07/12/19:00
next_reminder_at      TIMESTAMPTZ                         -- v4.0, dẫn xuất
next_anniversary_date DATE                                -- v4.0, dẫn xuất
created_at, updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
-- bỏ ở v4.0: reminder_timezone, add_to_os_reminder

CHECK (hai dạng ngày mất loại trừ nhau — như 0018)
CHECK (death_calendar = 'lunar' OR death_leap_month IS NOT TRUE)
CHECK (death_calendar = 'solar' OR death_day IS NULL OR death_day <= 30)
CHECK chk_memorials_reminder_all_or_none: (reminder_days_before, reminder_time) cùng NULL hoặc cùng NOT NULL   -- thay bản 3 cột
CHECK chk_memorials_plan_pair: (next_reminder_at IS NULL) = (next_anniversary_date IS NULL)
CHECK chk_memorials_plan_needs_reminder: next_reminder_at IS NULL OR reminder_days_before IS NOT NULL
INDEX idx_memorials_next_reminder ON memorials (next_reminder_at) WHERE next_reminder_at IS NOT NULL
```

`users`: thêm `timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Ho_Chi_Minh'` (AC38). Migration `0037` idempotent (`ADD COLUMN IF NOT EXISTS`, `DROP CONSTRAINT IF EXISTS`, `DROP COLUMN IF EXISTS`), thứ tự: thêm cột `users.timezone` → thêm `next_*` → drop `chk_memorials_os_needs_reminder`, `chk_memorials_reminder_all_or_none` → drop `reminder_timezone`, `add_to_os_reminder` → tạo lại CHECK all-or-none 2 cột + 2 CHECK kế hoạch + index. **Không backfill bằng SQL** (cần `lunarcal`): mọi dòng có nhắc bắt đầu với `next_* = NULL` và được Step J-2 lập kế hoạch ở lượt job đầu tiên (AC39). Không có CHECK "có nhắc ⇒ có kế hoạch" vì chính trạng thái chuyển tiếp này.

Ràng buộc "không ở tương lai" và tập lựa chọn nhắc **không** đặt ở DB; chỉ usecase kiểm.

## Phụ thuộc feature notification (ADR-0001)

Memorial là producer đầu tiên của inbox. Những gì memorial **cần**, để spec notification (chưa có) quyết định hình dạng cụ thể:

1. Một bảng inbox sở hữu theo tài khoản, có cột idempotency key với **UNIQUE**, và một hàm ghi **nhận transaction đang mở** (`INSERT … ON CONFLICT (idempotency_key) DO NOTHING RETURNING id`) để Step J-5 ghi cùng tx với việc cuộn kế hoạch (ADR-0002: "one transaction for the DB part").
2. Key: `memorial:<memorial_id>:<next_anniversary_date YYYY-MM-DD>`.
3. Dữ liệu lời nhắc memorial: loại `memorial_reminder`, `memorial_id`, `anniversary_date`, `relationship`, `days_left`, tiêu đề/nội dung push (§3.1.1, chỉ quan hệ + số ngày) — **không** chứa `full_name`/`note`. Dòng chi tiết "{tên} · {ngày}" trong app đọc tên từ mục lúc hiển thị (lời nhắc bị xoá cùng mục, product.md@2.1 BR-MEM-12).
6. Hàm xoá tx-aware theo `memorial_id` (`DELETE … WHERE type = 'memorial_reminder' AND memorial_id = $1`) để Step D xoá lời nhắc cùng transaction với mục (AC22).
4. Cổng `ReminderPusher.Push(ctx, notificationID)` gửi tới mọi token của user qua `pkg/push/fcm`, bỏ push khi công tắc thông báo của tài khoản tắt (product.md@2.1, FR-MEM-008), tự dọn token hỏng; lỗi trả về chỉ để job log/đếm.
5. Mở lời nhắc → mở mục (FR-MEM-008): client dùng `memorial_id`; mục không còn → client hiện "Mục này không còn trong sổ giỗ." qua `404` của `GET /v1/memorials/{id}` (đã có).

> ⚠️ Not confirmed: spec + migration của feature notification (bảng inbox, UNIQUE key, hàm ghi tx-aware, `ReminderPusher`, `pkg/push`) — chặn Step J-5, J-7 và `tasks.md` Phase 10; Phase 8–9 không bị chặn.

## Request flow

`Get`, `List` chỉ một câu SQL nên giữ dạng ASCII; `Create`, `Update`, `Delete`, báo múi giờ, job và hai flow ảnh có nhiều bước nên nằm ở Step-by-step Flow và `flow.md`.

```
GET /v1/memorials/{id}   (tương tự GET /v1/memorials)
  └─ authMiddleware.AuthenticateFunc  (401 nếu thiếu/sai token; gắn userID vào ctx)
       └─ MemorialHandler.Get   (delivery/http)
            ├─ userID từ context; parse {id} (không phải số dương → 404)
            └─ MemorialUsecase.Get(ctx, userID, id)
                 ├─ MemorialRepository.FindByIDAndUser → SELECT … WHERE id=$1 AND user_id=$2
                 │     └─ sql.ErrNoRows → ErrMemorialNotFound (404, giống "của người khác")
                 └─ MemorialPhotoStore.SignedURL (⏸ Phase 6, chỉ khi có photo_object_key)
```

## Step-by-step Flow — Thêm người vào sổ giỗ (POST /v1/memorials)

```
Step C-1 — Xác thực (middleware)
- Trigger: request đến POST /v1/memorials
- Pre-condition: header Authorization: Bearer <access token>
- Call: authMiddleware.AuthenticateFunc
    - Success → userID vào context
    - Failure (thiếu/hết hạn/sai loại token) → 401, dừng (AC2)

Step C-2 — Parse + validate (this module)
- Trigger: sau C-1
- Pre-condition: userID có trong context
- Call: Handler DecodeJSONBodyStrict → MemorialUsecase.Create validate AC3–AC9 (v4.0: AC9 tập lựa chọn; field cũ `timezone`/`add_to_os_reminder` đã bị decoder từ chối — AC12)
    - Success → entity Memorial + PlanFunc (closure của domain.PlanReminder, now = clock(), notBefore = hôm nay; nil khi không có reminder)
    - Failure → 400 VALIDATION_ERROR kèm details.field; chưa chạm DB

Step C-3 — Khoá user, đọc múi giờ, kiểm trần, lập kế hoạch, chèn (this module, MỘT transaction)
- Trigger: sau C-2
- Pre-condition: dữ liệu hợp lệ
- Call: MemorialRepository.Create(ctx, m, MaxPerUser, plan)
    a. BEGIN
    b. SELECT timezone FROM users WHERE id=$1 FOR UPDATE   (tuần tự hoá các Create của cùng user; v4.0 đọc thêm timezone dưới cùng khoá — đổi múi giờ song song phải chờ)
    c. SELECT count(*) FROM memorials WHERE user_id=$1
    d. nếu count >= 100 → ROLLBACK, ErrMemorialLimitReached
    e. (v4.0) nếu có reminder: time.LoadLocation(timezone) → plan(loc) → ReminderPlan{At, AnniversaryDate} (AC34); không có reminder → next_* = NULL (AC35)
    f. INSERT INTO memorials (..., next_reminder_at, next_anniversary_date) RETURNING id, created_at, updated_at
    g. COMMIT
    - Success → Memorial đã có id
    - Failure (đủ 100) → 409 MEMORIAL_LIMIT_REACHED (AC11)
    - Failure (timezone không nạp được / plan lỗi) → ROLLBACK, 500 INTERNAL_ERROR (không xảy ra với dữ liệu hợp lệ: timezone được validate khi ghi, plan luôn có nghiệm)
    - Failure (lỗi DB) → 500 INTERNAL_ERROR, rollback
    - Transaction boundary: b–g cùng một tx; lỗi ở bất kỳ bước nào rollback toàn bộ. Không tạo notification nào (AC34).

Step C-4 — Trả response (this module)
- Trigger: sau C-3 thành công
- Pre-condition: row đã commit
- Call: Handler map entity → memorialResponse (photo_url = null; không có next_* — AC17b)
    - Success → 201 Memorial (AC1)
```

## Step-by-step Flow — Sửa một mục (PUT /v1/memorials/{id}) — v4.0

v1 là một câu `UPDATE … WHERE id AND user_id`; v4.0 thêm việc đọc múi giờ và lập kế hoạch nên thành nhiều bước.

```
Step U-1 — Xác thực (middleware)
- Trigger: request đến PUT /v1/memorials/{id}
- Call: authMiddleware.AuthenticateFunc
    - Failure → 401, dừng

Step U-2 — Parse + validate (this module)
- Trigger: sau U-1
- Pre-condition: {id} là số dương (không thì 404)
- Call: DecodeJSONBodyStrict → MemorialUsecase.Update validate AC3–AC9 như C-2; dựng PlanFunc như C-2
    - Failure → 400 VALIDATION_ERROR, row không đổi (AC20)

Step U-3 — Đọc múi giờ, lập kế hoạch, ghi (this module, MỘT transaction)
- Trigger: sau U-2
- Call: MemorialRepository.Update(ctx, m, plan)
    a. BEGIN
    b. SELECT timezone FROM users WHERE id=$userID FOR SHARE   (chặn đổi múi giờ song song tới khi commit)
    c. nếu có reminder: LoadLocation → plan(loc) (AC34, kể cả "thời điểm kỳ này đã qua → kỳ sau"); không có → next_* = NULL (AC35)
    d. UPDATE memorials SET full_name…, reminder_days_before, reminder_time, next_reminder_at, next_anniversary_date, updated_at=now() WHERE id=$1 AND user_id=$2 RETURNING …
    e. COMMIT
    - Success → Memorial mới
    - Failure (0 dòng: không có / của người khác) → ROLLBACK, 404 MEMORIAL_NOT_FOUND (AC21)
    - Failure (lỗi DB / plan) → ROLLBACK, 500
    - Transaction boundary: b–e. Nếu job đang giữ khoá dòng này (J-3), UPDATE ở d chờ job commit rồi ghi đè kế hoạch bằng kế hoạch mới — đúng ý: thay đổi của user thắng.

Step U-4 — Trả response
- Call: Handler → 200 Memorial (AC18)
```

## Step-by-step Flow — Báo múi giờ tài khoản (PATCH /v1/users/me, field timezone) — v4.0

Route thuộc `profile`; chỉ bước T-3 thuộc memorial. Ghi ở đây vì nó là một trigger lập kế hoạch (AC37).

```
Step T-1 — Xác thực + validate (thuộc profile)
- Trigger: PATCH /v1/users/me có field timezone
- Call: authMiddleware → ProfileHandler (DecodeJSONBodyStrict) → ProfileUsecase validate time.LoadLocation(timezone), từ chối "" và "Local"
    - Failure → 401 / 400 (mã lỗi do spec profile chốt), dừng; không đổi gì

Step T-2 — Lưu múi giờ (thuộc profile)
- Trigger: sau T-1
- Call: ProfileRepository cập nhật users.timezone (cùng câu UPDATE với các field profile khác trong body)
    - Success → T-3
    - Failure (DB) → 500, dừng

Step T-3 — Lập lại kế hoạch mọi mục có nhắc (this module, MỘT transaction)
- Trigger: sau T-2 thành công — luôn chạy khi body có timezone, kể cả giá trị không đổi (tự lành, AC37)
- Pre-condition: users.timezone đã commit
- Call: MemorialReminderReplanner.ReplanUser(ctx, userID) → MemorialRepository.ReplanUser(ctx, userID, plan)
    a. BEGIN
    b. SELECT timezone FROM users WHERE id=$1 FOR SHARE
    c. SELECT … FROM memorials WHERE user_id=$1 AND reminder_days_before IS NOT NULL FOR UPDATE   (≤ 100 dòng)
    d. mỗi dòng: plan(loc) → UPDATE memorials SET next_reminder_at, next_anniversary_date WHERE id=$
    e. COMMIT
    - Success → profile trả 200 như spec profile
    - Failure → ROLLBACK (kế hoạch cũ giữ nguyên, timezone mới đã lưu) → 500 INTERNAL_ERROR; client gửi lại cùng timezone (sau lỗi, hoặc lần đăng nhập sau) sẽ chạy lại T-3. Log userID, không log gì khác.
    - Transaction boundary: b–e; T-2 nằm ở transaction riêng của profile (quyết định: không kéo SQL bảng memorials vào repository profile).
```

## Step-by-step Flow — Job gửi nhắc giỗ hằng giờ (cmd/memorialreminders) — v4.0

```
Step J-1 — Khởi động (job)
- Trigger: Cloud Scheduler (cron "0 * * * *", múi UTC) gọi Cloud Run Job memorial-reminders
- Pre-condition: image chứa binary cmd/memorialreminders; service account riêng có quyền Cloud SQL + FCM (hạ tầng)
- Call: nạp config/secrets, database Connect(ctx), dựng MemorialReminderStore, ReminderPusher (feature notification), MemorialReminderJob; now := time.Now()
    - Success → J-2
    - Failure (config/DB) → log, exit ≠ 0 (Cloud Run Job tự retry theo cấu hình; chạy lại an toàn nhờ J-3/J-5)

Step J-2 — Lập kế hoạch cho mục chưa có (this module, mỗi lô một transaction)
- Trigger: sau J-1
- Call: MemorialReminderStore.PlanUnplanned(ctx, PlanBatch, plan) lặp tới khi trả 0:
    BEGIN; SELECT m.…, u.timezone FROM memorials m JOIN users u ON u.id = m.user_id
           WHERE m.reminder_days_before IS NOT NULL AND m.next_reminder_at IS NULL
           LIMIT $PlanBatch FOR UPDATE OF m SKIP LOCKED;
    mỗi dòng: PlanReminder(m, loc, now, notBefore = hôm nay) → UPDATE next_*; COMMIT
    - Success → J-3 (các mục vừa lập có thời điểm ≥ now nên không bị gửi bù — AC39)
    - Failure (một dòng plan lỗi) → ROLLBACK lô, log id, đếm failed, loại id đó ở các lô sau trong lượt này, tiếp tục

Step J-3 — Nhận một mục đến hạn (this module, mở transaction của mục)
- Trigger: vòng lặp sau J-2, tới khi hết mục đến hạn hoặc đủ MaxPerRun
- Call: MemorialReminderStore.ProcessNextDue(ctx, now, failedIDs, handle):
    BEGIN;
    SELECT m.…, u.timezone FROM memorials m JOIN users u ON u.id = m.user_id
      WHERE m.next_reminder_at <= $now AND m.id <> ALL($failedIDs)
      ORDER BY m.next_reminder_at LIMIT 1 FOR UPDATE OF m SKIP LOCKED
    - Không có dòng → COMMIT, sang J-8
    - Có dòng → J-4 (vẫn trong tx)

Step J-4 — Quyết định gửi hay bỏ (pure, trong tx)
- Call: handle(due) → ShouldDeliver(plan, loc, now) (BR-MEM-13, delivery_cases)
    - deliver=true → dựng MemorialReminderNotice (DaysLeft = anniversary − hôm nay theo loc; copy §3.1.1 theo quan hệ, OTHER → "người thân"; key memorial:<id>:<date>) → J-5
    - deliver=false (ngày giỗ đã hết theo múi giờ tài khoản) → đếm skipped_late, sang J-6 (AC41)
    - Luôn tính Next = PlanReminder(m, loc, now, notBefore = next_anniversary_date + 1 ngày)

Step J-5 — Ghi lời nhắc vào inbox (feature notification, trong cùng tx)
- Pre-condition: deliver=true
- Call: INSERT INTO <inbox> (…, idempotency_key) VALUES (…) ON CONFLICT (idempotency_key) DO NOTHING RETURNING id
    ⚠️ Not confirmed: bảng, cột và hàm ghi tx-aware thuộc spec notification chưa có (xem "Phụ thuộc feature notification").
    - Có id → created = true (đếm sent)
    - Không có id (đã có, do lượt trước lỗi sau khi ghi) → created = false, đếm already_sent; không push (AC42)
    - Failure (DB) → ROLLBACK, sang J-3 với id này vào failedIDs (AC44)

Step J-6 — Cuộn kế hoạch + commit (this module)
- Call: UPDATE memorials SET next_reminder_at=$Next.At, next_anniversary_date=$Next.AnniversaryDate WHERE id=$1; COMMIT
    - Success → nếu created → J-7, không thì quay J-3
    - Failure (plan lỗi ở J-4 hoặc UPDATE lỗi) → ROLLBACK (lời nhắc ở J-5 cũng mất, an toàn), id vào failedIDs, đếm failed, quay J-3 (AC44)
    - Transaction boundary: J-3 → J-6 là MỘT transaction cho MỘT mục. Xoá mục (D-2) hoặc PUT (U-3) song song phải chờ tx này; sau khi nó commit, lượt chạy sau không còn thấy kế hoạch cũ (AC35, AC36).

Step J-7 — Gửi push sau commit (feature notification, best-effort)
- Trigger: J-6 commit với created = true
- Call: ReminderPusher.Push(ctx, notificationID) — tới mọi token của user qua pkg/push/fcm; công tắc thông báo của tài khoản tắt → feature notification không push, lời nhắc vẫn ở inbox (product.md@2.1, FR-MEM-008)
    ⚠️ Not confirmed: cổng ReminderPusher và pkg/push thuộc spec notification chưa có.
    - Success / Failure → đều quay J-3; lỗi chỉ log notificationID, memorialID, không đổi inbox hay kế hoạch (AC43). Crash giữa J-6 và J-7 = lời nhắc có trong inbox nhưng không push (chấp nhận: inbox là nguồn sự thật, ADR-0001).

Step J-8 — Kết thúc lượt chạy (job)
- Call: SELECT count(*) FROM memorials WHERE next_reminder_at < $now − interval '1 hour' → metric overdue_1h; ghi thống kê claimed/sent/skipped_late/already_sent/failed (AC45); exit 0
    - Nếu dừng vì MaxPerRun → log cảnh báo (mục còn lại để lượt sau; dấu hiệu "Revisit when" của ADR-0002)
```

## Step-by-step Flow — Tải/thay ảnh (PUT /v1/memorials/{id}/photo)

> ⏸ **HOÃN (Phase 6) — chặn bởi GCS.** Giữ nguyên để làm tiếp; chưa implement flow này.

```
Step P-1 — Xác thực + kiểm quyền sở hữu TRƯỚC khi đọc body (this module)
- Trigger: request đến PUT /v1/memorials/{id}/photo
- Pre-condition: access token hợp lệ (401 nếu không)
- Call: MemorialRepository.FindByIDAndUser(id, userID)
    - Success → có Memorial (và photo_object_key cũ, nếu có)
    - Failure (không có / của người khác) → 404 MEMORIAL_NOT_FOUND; body upload chưa được đọc (AC31)

Step P-2 — Đọc + kiểm file (this module)
- Trigger: sau P-1
- Pre-condition: memorial thuộc user
- Call: http.MaxBytesReader(5 MB) → multipart part "photo" → sniff 512 byte đầu bằng http.DetectContentType
    - Success → bytes ảnh + contentType ∈ {jpeg, png, webp}
    - Failure (thiếu/rỗng part) → 400 (AC28); (vượt 5 MB) → 413 (AC26); (sai kiểu) → 415 (AC27) — chưa ghi storage
    - ⚠️ Not confirmed: bucket/prefix riêng cho ảnh sổ giỗ và quyền ký signed URL của service account Cloud Run — chặn `MemorialConfig.PhotoBucket` và adapter GCS; dev tạm dùng `innerlamp-avatars-dev`.

Step P-3 — Ghi object mới (this module → GCS)
- Trigger: sau P-2
- Pre-condition: file hợp lệ
- Call: MemorialPhotoStore.Put(ctx, "memorials/<userID>/<id>/<uuid>.<ext>", file, contentType)
    - Success → newKey
    - Failure → 500 INTERNAL_ERROR; DB chưa đổi, ảnh cũ (nếu có) vẫn nguyên (AC32)

Step P-4 — Ghi key mới vào DB (this module)
- Trigger: sau P-3
- Pre-condition: newKey đã tồn tại trên GCS
- Call: MemorialRepository.SetPhotoKey(id, userID, newKey) — UPDATE … WHERE id AND user_id, trả oldKey; refresh updated_at
    - Success → oldKey (có thể rỗng)
    - Failure (lỗi DB) → best-effort MemorialPhotoStore.Delete(newKey) rồi 500; DB giữ key cũ (AC32)
    - Failure (0 dòng — vừa bị xoá song song) → xoá newKey best-effort, 404

Step P-5 — Dọn object cũ (this module → GCS, best-effort)
- Trigger: sau P-4 thành công AND oldKey khác rỗng
- Pre-condition: DB đã trỏ sang newKey
- Call: MemorialPhotoStore.Delete(oldKey)
    - Success → xong
    - Failure → chỉ log (key, không log tên/ghi chú), vẫn tiếp tục; object mồ côi chờ dọn (AC29)

Step P-6 — Trả response (this module)
- Trigger: sau P-4/P-5
- Call: SignedURL(newKey, 15m) → memorialResponse
    - Success → 200 Memorial có photo_url (AC25)
    - Failure (ký URL) → 500 INTERNAL_ERROR (ảnh đã lưu; client gọi GET lại để lấy URL)
```

`DELETE /v1/memorials/{id}/photo` dùng đúng thứ tự "DB trước, GCS sau": `SetPhotoKey(id, userID, nil)` → xoá object cũ best-effort → `200 Memorial` (`photo_url: null`); không có ảnh vẫn `200` (AC30).

## Step-by-step Flow — Xoá một mục (DELETE /v1/memorials/{id})

```
Step D-1 — Xác thực (middleware)
- Trigger: request đến DELETE /v1/memorials/{id}
- Call: authMiddleware.AuthenticateFunc
    - Failure → 401, dừng

Step D-2 — Xoá dòng (this module)
- Trigger: sau D-1
- Pre-condition: {id} là số dương (không thì 404)
- Call: MemorialRepository.Delete(id, userID) — DELETE FROM memorials WHERE id=$1 AND user_id=$2 (v1: kiểm RowsAffected; ⏸ Phase 6 đổi sang RETURNING photo_object_key). Kế hoạch nhắc (next_*) mất cùng dòng; nếu job đang giữ khoá dòng (J-3..J-6) thì DELETE chờ job commit (AC36).
    - Success → xoá được 1 dòng
    - Failure (0 dòng: không tồn tại/của người khác/đã xoá) → 404 MEMORIAL_NOT_FOUND (AC24)
    - Failure (lỗi DB) → 500, không xoá gì

Step D-3 — Xoá ảnh (this module → GCS, best-effort) — ⏸ HOÃN tới Phase 6; v1 bỏ qua bước này (`photo_object_key` luôn NULL)
- Trigger: sau D-2 AND key khác NULL
- Pre-condition: dòng đã mất
- Call: MemorialPhotoStore.Delete(key)
    - Success/Failure → vẫn tiếp tục; thất bại chỉ log (AC23)

Step D-4 — Trả response
- Call: Handler → 200 {"success": true} (AC22)
```

**Vì sao "DB trước, GCS sau" ở các flow ảnh/xoá:** nếu xoá/đổi object trước rồi DB lỗi, dòng sẽ trỏ vào object không còn → user thấy ảnh hỏng. Làm ngược lại thì xấu nhất là một object mồ côi user không thấy — chấp nhận được và dọn được sau.

**Vì sao "commit trước, push sau" ở job:** push trước commit rồi commit lỗi thì lượt sau tạo lại lời nhắc (key chưa có) và push lần hai — vi phạm BR-MEM-13. Commit trước thì xấu nhất là lời nhắc trong inbox mà thiếu push — chấp nhận được (ADR-0001).

## Known gaps carried by this feature

- **Feature notification chưa có spec/code** (ADR-0001): Phase 10 bị chặn ở J-5/J-7. Phase 8–9 (contract, kế hoạch, J-2) làm được trước; khi đó các mục đến hạn chỉ tích lại (`overdue_1h` tăng) — **không deploy job trước khi Phase 10 xong**.
- **Spec `profile` chưa có `timezone`**: AC37 cần nó; follow-up cho spec `profile` (không sửa ở đây).
- **Hạ tầng** (Cloud Scheduler job, Cloud Run Job `memorial-reminders`, service account, quyền FCM) nằm ngoài repo — ADR-0002 Follow-ups.
- Clock: lượt job dùng một `now` cho cả lượt; lượt kéo dài vài phút vẫn đúng vì mục đến hạn trong lúc chạy sẽ được lượt sau lấy (≤ 1 giờ, NFR-MEM-04).
- Khoảng hở nhỏ đã chấp nhận: lời nhắc của một mục đang được job xử lý đúng lúc user xoá/tắt nhắc vẫn có thể vào inbox (tx của job commit trước) — mọi lượt sau đó không gửi nữa.
- ⏸ Phase 6: chưa có storage client cho memorial; khi chưa có bucket thật, dev có thể chạy với fake store.
- 5 route v1 đã được bọc `authMiddleware.AuthenticateFunc` và E2E (`e2e/memorial_e2e_test.go`) chứng minh cả 5 trả 401 khi thiếu token, token rác hoặc refresh token. Hai route ảnh (Phase 6) phải được thêm vào đúng bài kiểm tra này.
- `docs/specs/db/erd.svg` phải sinh lại bằng `mmdc` sau migration `0037` (repo chưa có script).
- Chưa có cơ chế dọn object GCS mồ côi.
- `openapi.yaml` chưa mô tả contract v4.0 (Phase 8) và chưa nói cờ nhuận khôi phục từ `death_date` (Phase 7).
