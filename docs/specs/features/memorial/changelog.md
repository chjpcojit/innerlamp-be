# Changelog — memorial Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version
first.

## v4.0 (2026-10-14)

- `requirements.md`: trỏ lên product spec `docs/product/memorial/product.md` @ v2.0 (ADR-0001, ADR-0002 Accepted). `Implements:` thêm BR-MEM-05, 06, 07, 10, 13, FR-MEM-008, NFR-MEM-03, NFR-MEM-04; FR-MEM-009 ghi "removed". "Ràng buộc từ product spec" viết lại: server giờ tính kỳ giỗ để nhắc và gửi nhắc; inbox/push thuộc feature notification. AC8 ghi rõ mặc định nhắc là của form client. **AC9** thu hẹp về lead 0/1/3/7 × giờ 06:00/07:00/12:00/19:00. **AC10 bỏ** (giữ số). **AC12** từ chối `add_to_os_reminder`, `reminder.timezone` như field lạ. Thêm **AC17a** (giá trị nhắc cũ trả nguyên, job vẫn dùng), **AC17b** (`reminder` không còn `timezone`; không lộ `next_*`). AC18/AC20/AC22 nhắc tới kế hoạch nhắc. AC33 mở rộng cho job. Thêm **US-MEM-06** (AC34–AC39: lập kế hoạch khi tạo/sửa/báo múi giờ, mặc định `Asia/Ho_Chi_Minh`, lập bù cho mục chưa có kế hoạch) và **US-MEM-07** (AC40–AC46: job hằng giờ, `SKIP LOCKED`, idempotency key, gửi trễ/bỏ theo BR-MEM-13, push sau commit, metric, vector). Bảng contract: schema `reminder` mới, thêm dòng `PATCH /v1/users/me` (`timezone`, thuộc `profile`). Out of scope viết lại (feature notification, hạ tầng). Ghi chú: các quyết định platform bên dưới; công tắc thông báo tài khoản đã có câu trả lời. Open questions: 4 mục mới resolved; 5 mục cũ đánh dấu superseded.
- `design.md`: Overview + bảng Layers thêm `domain/memorial_schedule.go` (hàm thuần trên `lunarcal`), ports job (`MemorialReminderStore`, `ReminderPusher`, `MemorialReminderReplanner`), `usecase/memorial_reminder_usecase.go`, `repository/postgres_memorial_reminder_store.go`, hook `profile`, `cmd/memorialreminders`, config `Reminder{PlanBatch, MaxPerRun}`, metrics, migration `0037`, file vector mới. Schema: `users.timezone`, `memorials.next_reminder_at`/`next_anniversary_date`, bỏ `reminder_timezone`/`add_to_os_reminder`, CHECK + index partial mới, không backfill SQL. Mục mới "Phụ thuộc feature notification". Step-by-step Flow: **C-3** đọc `timezone` dưới khoá `users` và lập kế hoạch trong tx; **mới** flow Sửa (U-1..U-4, trước là ASCII một câu SQL), flow Báo múi giờ (T-1..T-3), flow Job (J-1..J-8). Request flow ASCII chỉ còn `Get`/`List`. Known gaps viết lại.
- `flow.md`: sơ đồ tạo đổi Step C-3; thêm sequence Sửa, Báo múi giờ, Job và flowchart quyết định J-4 → J-7; sơ đồ xoá thêm note kế hoạch mất cùng dòng (AC36); sơ đồ ảnh không đổi. Cả 7 khối Mermaid đã parse được bằng `mermaid@11.4.1`.
- `tasks.md`: Phase 7 bỏ phần mô tả `reminder.timezone` (field bị bỏ); thêm **Phase 8** (contract + migration `0037`), **Phase 9** (lập kế hoạch + vector `next-anniversary.json`, `reminder-moment.json` + hook `profile`), **Phase 10** (job, ⛔ chặn bởi feature notification). Mọi mục mới `[ ]`; Phase 1–6 không đổi trạng thái. Deferred: thay "push từ server" bằng "feature notification".
- Reason: product owner đưa nhắc giỗ lên server (product v2.0, ADR-0001/0002). Bump major vì đổi hành vi API, thêm bảng/cột, thêm flow và Phase.
- Quyết định platform (ghi ở Ghi chú / Open questions resolved của `requirements.md`):
  1. Giá trị nhắc cũ ngoài danh sách: **giữ trong DB, trả nguyên, job gửi theo giá trị cũ**; chỉ request ghi bị thu hẹp; CHECK DB giữ 0..30 (theo FR-MEM-007 "keep the stored value until the user saves").
  2. Bỏ `reminder.timezone`/`add_to_os_reminder` **không versioning**, ngay trên `/v1`; field cũ → 400; tăng major `info.version` của `openapi.yaml`.
  3. Báo múi giờ qua `PATCH /v1/users/me` field `timezone` (route `profile`), sau đăng nhập và khi đổi; mỗi lần báo đều tính lại kế hoạch (tự lành); lỗi tính lại → 500, múi giờ đã lưu.
  4. Ghi lời nhắc vào inbox **cùng tx** với cuộn kế hoạch; push **sau commit**; một tx cho một mục; mục lỗi bị loại khỏi phần còn lại của lượt.
  5. Không backfill kế hoạch bằng SQL; lượt job đầu tiên (Step J-2) lập kế hoạch cho mọi mục có nhắc mà `next_* IS NULL`.
  6. Thời điểm nhắc bằng đúng `now` không coi là "đã qua"; cuộn kế hoạch dùng `notBefore = next_anniversary_date + 1 ngày`.
  7. API không trả `next_reminder_at`/`next_anniversary_date`.
- Đã giải quyết trong phiên: công tắc thông báo của tài khoản chỉ tắt push, lời nhắc vẫn vào inbox (anh Đức, 2026-10-14; sẽ ghi ở product.md@2.1, FR-MEM-008) — không còn `⚠️ Not confirmed (product)` cho mục này; memorial luôn tạo notification, feature notification áp công tắc cho push.
- Vẫn mở: marker bucket GCS (Phase 6); ⚠️ feature notification chưa có spec (chặn Phase 10); ⚠️ (product) lời nhắc đã gửi của mục bị xoá và tên trong dòng chi tiết.
- Follow-ups (không sửa ở đây): `docs/specs/api/openapi.yaml` + `web/docs/public/openapi.yaml` (Phase 7, 8; `timezone` trên `/v1/users/me`); `migrations/0037_memorial_reminders.sql` + `migrations/README.md` (Phase 8); `docs/specs/db/erd.mmd`/`erd.svg` (Phase 8); `docs/specs/features/profile/` (field `timezone`, gọi `ReplanUser`); spec mới `docs/specs/features/notification/` (ADR-0001); `docs/knowledge/features/memorial.md` vẫn chưa có — tạo bằng `il-okf-writer` sau khi Phase 8–10 chạy end-to-end; hạ tầng Cloud Scheduler/Cloud Run Job (ADR-0002).
- Trả về `ba-writer` (product spec, không sửa ở đây):
  1. Lời nhắc đã gửi của một mục bị xoá: còn trong inbox không, và tên trong dòng chi tiết "{tên} · {ngày}" khi mục không còn (BR-MEM-12 vs §3.5).
  2. Ma trận §4.2: BR-MEM-08/09 cột Backend `n/a` đúng; FR-MEM-009 cột Backend nên ghi "drift → field bỏ ở BE v4.0" thay vì `n/a`, vì backend vẫn đang lưu cờ cho tới Phase 8.
  3. `rules/reminder-moment.json`: hai case "moment already passed" ghi nguồn BR-MEM-13 nhưng thực chất là quy tắc của BR-MEM-11 (add/edit sau thời điểm nhắc) — chỉ sửa nhãn.
  4. `rules/next-anniversary.json`, `death-date-conversion.json` còn `product_spec_version: "1.0"` — xác nhận vẫn đúng ở v2.0.
  5. Ghi câu trả lời công tắc thông báo vào product v2.1 (coordinator đã nhận làm).

## v3.0 (2026-10-14)

- `requirements.md`: gắn vào product spec `docs/product/memorial/product.md` @ v1.0 — thêm header `Product spec` / `Contract` / `Implements`; mỗi AC kết thúc bằng ID product (`FR-MEM-*`, `BR-MEM-*`) hoặc `(platform)` (AC10 cờ OS, AC12 body strict). Bỏ mục "Quyết định nền tảng" (quyết định sản phẩm giờ nằm ở product spec §2.7/§3.6), thay bằng "Ràng buộc từ product spec" chỉ nói backend hiện thực thế nào. Thêm AC33 (không log tên/ghi chú — NFR-MEM-02, trước chỉ là ghi chú; code memorial hiện không log gì). Ghi chú kỹ thuật: thêm cách backend đáp ứng Appendix B (giữ khoảng nhắc 0..30/mọi `HH:mm`; `timezone` là múi giờ máy; cờ nhuận khôi phục từ `death_date`; tháng nhuận/năm âm khi có năm do client kiểm; `PUT` thay toàn bộ nên nền tảng không hiện nhắc phải gửi lại). Hai câu hỏi Appendix B chuyển vào Open questions (resolved). Số AC và `US-MEM-*` giữ nguyên. Không đổi hành vi.
- `design.md`: thêm dòng "Product vectors" vào bảng Layers (test `domain/lunarcal` chạy `death-date-conversion.json`); Overview trỏ sang product spec thay cho "Quyết định nền tảng"; Known gaps ghi hai khác biệt có chủ đích với product và follow-up `openapi.yaml`. Step-by-step Flow không đổi.
- `flow.md`: no change needed, verified — các flow tạo/ảnh/xoá không đổi bước; chỉ bump header.
- `tasks.md`: thêm Phase 7 (2 mục `[ ]`: chạy vector `death-date-conversion.json`, bổ sung mô tả `openapi.yaml`). Phase 1–6 không đổi.
- Reason: chuyển sang mô hình 3 lớp tài liệu (product spec → contract → platform spec). Bump major vì đổi cấu trúc spec và thêm Phase.
- Follow-ups (không sửa ở đây): `docs/specs/api/openapi.yaml` + `web/docs/public/openapi.yaml` (Phase 7); `migrations/`, `docs/specs/db/`: không ảnh hưởng; `docs/knowledge/features/memorial.md` vẫn chưa có (đã ghi từ v2.2), không cần làm mới vì hành vi không đổi.
- Trả về `ba-writer` (product spec, không sửa ở đây):
  1. Ma trận §4.2, BR-MEM-04 cột Backend đang `drift`: theo Appendix B mục 1 và 4, backend đã đúng thiết kế (cờ nhuận khôi phục từ `death_date`, kiểm nhuận là việc client) — đề xuất đổi sang `done`.
  2. NFR-MEM-01 ghi Platforms "Website, iOS, Android" nhưng ma trận có Backend `planned` — cần chốt backend có mục tiêu thời gian riêng không; spec này chưa có AC cho NFR-MEM-01.
- Vẫn mở: marker bucket GCS (chỉ chặn Phase 6).

## v2.4 (2026-10-07)

- `requirements.md`: câu trả lời cho HEIC chuyển từ "cách hiểu của spec" sang đã xác nhận — client đổi HEIC/HEIF sang JPEG, backend chỉ nhận jpeg/png/webp. AC27 không đổi.
- `design.md`: no change needed, verified — Step P-2 đã đúng (chỉ nhận jpeg/png/webp, sniff nội dung).
- `flow.md`: no change needed, verified.
- `tasks.md`: no change needed, verified — Phase 6 không có việc BE nào cho HEIC; phần client ghi ở `docs/tech-debt/memorial/TD-001`.
- Reason: anh Đức xác nhận client tự đổi sang JPEG. `docs/assumptions/memorial/A-001` đổi `status` sang `confirmed`. Bump minor vì không đổi hành vi.
- Vẫn mở: marker bucket GCS (chỉ chặn Phase 6).

## v2.3 (2026-10-07)

- `requirements.md`: no change to behaviour; Status/Pending cập nhật — `erd.svg` đã sinh lại nên bỏ khỏi Pending.
- `design.md`: mục "Known gaps" về `erd.svg` đổi từ "chưa sinh lại" sang "đã sinh lại"; không đổi thiết kế.
- `flow.md`: no change needed, verified.
- `tasks.md`: tick mục sinh lại `erd.svg`; bỏ chú thích "chưa kiểm cú pháp"; Status chỉ còn marker bucket.
- Reason: xử lý các điểm còn lại sau v2.2. `erd.svg` render lại bằng `mmdc` từ `erd.mmd` (render thành công = cú pháp hợp lệ; bản cũ còn thiếu `gallery_images`). Hai file `openapi.yaml` được bỏ ignore trong `.gitignore` (`!docs/specs/api/openapi.yaml`, `!web/docs/public/openapi.yaml`) nên giờ hiện trong `git status` và diff được.
- Vẫn mở (cần anh quyết, không tự làm): marker bucket GCS; cách hiểu HEIC của `A-001` chưa được anh xác nhận. OKF memorial chưa viết theo yêu cầu.

## v2.2 (2026-10-07)

- `requirements.md`: v1 đã implement — cập nhật Status, đoạn mở đầu và mục Pending (chỉ còn ảnh và `erd.svg`); contract API đã có trong `openapi.yaml`. AC không đổi.
- `design.md`: sửa các dòng code đã làm sai: validation dùng `MemorialValidationError` thay vì "cơ chế có sẵn"; `MemorialRepository.Delete` ở v1 chỉ trả `error` và `SetPhotoKey` chưa khai báo (đổi sang Phase 6); Step D-2 dùng `RowsAffected` ở v1; mục "Known gaps" về middleware đã được giải quyết bằng E2E. Số bước và thứ tự các flow không đổi.
- `flow.md`: no change needed, verified — chỉ cập nhật header/Status; sơ đồ vẫn khớp từng bước với `design.md`.
- `tasks.md`: tick Phase 1–5 (trừ `erd.svg` và marker bucket); sửa chữ của mục interface cho khớp chữ ký thật; thêm mục "Sinh lại `erd.svg`"; ghi chú toàn bộ test sau khi chạy.
- Reason: implement v1 theo `golang-clean-arch-tdd`. Kiểm chứng: unit (usecase, handler), repository trên Postgres thật (gồm trần 100 dưới tải song song và CHECK của DB), E2E trên binary thật + testcontainers Postgres (5 route đều 401 khi thiếu/rác/refresh token; người lạ nhận 404 byte-giống id không tồn tại; trần 100 từ config thật), `go vet ./...` và `go test -short ./...` sạch.
- Ghi nhận khi chạy đủ test không `-short`: `TestSutraRepository_ShippedData` fail (số `DeityLinks` 191 thay vì 151) — không liên quan memorial, chưa sửa.
- Nợ kỹ thuật mới: `docs/tech-debt/memorial/TD-001-photo-upload-blocked-on-gcs.md` (ảnh/GCS hoãn theo quyết định của anh Đức).
- Cross-file follow-ups: `docs/specs/db/erd.svg` cần sinh lại từ `erd.mmd` (đã thêm bảng `memorials`, chưa render để kiểm cú pháp); `openapi.yaml` (+ bản `web/`) đã cập nhật nhưng bị `.gitignore` (`*.yaml`) bỏ qua như đã ghi ở `docs/tech-debt/sutra/TD-001`; concept OKF `docs/knowledge/features/memorial.md` do `il-okf-writer` tạo khi anh muốn (flow đã chạy E2E nhưng anh chưa yêu cầu), không thuộc skill này; `docs/knowledge/index.md`/`log.md` chưa đụng.

## v2.1 (2026-10-07)

- `requirements.md`: chốt thứ tự danh sách `GET /v1/memorials` là `created_at` DESC, `id` DESC (đúng như AC13 đang viết) — marker chuyển sang "Open questions (resolved)"; còn 1 marker (bucket GCS, chỉ chặn Phase 6).
- `design.md`: no change needed, verified — index `idx_memorials_user (user_id, created_at DESC, id DESC)` đã khớp thứ tự này.
- `flow.md`: no change needed, verified — không có flow nào phụ thuộc thứ tự danh sách.
- `tasks.md`: sửa mục Phase 5 về marker còn lại; không thêm/bỏ task, mọi mục vẫn `[ ]`.
- Reason: anh Đức đồng ý với thứ tự mặc định spec đề xuất. Bump minor vì chỉ chốt một câu hỏi, không đổi hành vi.

## v2.0 (2026-10-07)

- `requirements.md`: **ảnh (US-MEM-05) chuyển sang Deferred** vì chưa làm được GCS — AC25–AC32 và các nhánh ảnh của AC16/AC23 không thuộc v1, `photo_url` luôn `null`, hai route ảnh đánh dấu ⏸; thêm mục "Deferred". Chốt: mã US tự chọn, tháng nhuận theo đề xuất (dùng tháng thường cùng số), HEIC → client đổi sang JPEG (cách hiểu của spec, ghi ở `docs/assumptions/memorial/A-001-heic-client-converts.md`), cờ OS đa thiết bị client tự xử lý, dọn object mồ côi "tính sau"; mỗi mục tối đa 1 ảnh tuỳ chọn. Còn 2 marker (thứ tự danh sách; bucket GCS — chỉ chặn Phase 6).
- `design.md`: v1 làm 5 route CRUD, không import `pkg/storage`, không dựng storage client; các layer/flow liên quan ảnh đánh dấu ⏸ Phase 6 (port `MemorialPhotoStore`, adapter GCS, config ảnh, flow tải/thay ảnh, Step D-3). Giữ cột `photo_object_key` (NULL) trong `0018` để khỏi cần migration khi làm ảnh. Số bước của flow tạo/xoá không đổi.
- `flow.md`: sơ đồ tải/thay ảnh và nhánh xoá-object trong sơ đồ xoá đánh dấu hoãn; không thêm/bỏ bước.
- `tasks.md`: Phase 1–5 thu về v1 (5 route, bỏ phần storage/ảnh); thêm Phase 6 — Ảnh ⏸ gom mọi mục ảnh, mọi mục vẫn `[ ]`.
- Reason: anh Đức quyết định tạm bỏ qua GCS. Đây là thay đổi scope (ảnh ra khỏi v1) nên bump major.
- Follow-up: `openapi.yaml` (+ bản `web/`) chỉ cần 5 route cho v1; `migrations/0018_memorials.sql` gồm 3 cột ngày-mất mới (v1.1) và `photo_object_key`; `docs/specs/db/erd.*`; OKF chưa tạo nên chưa có gì để làm lại.

## v1.1 (2026-10-07)

- `requirements.md`: năm mất thành tuỳ chọn — sửa AC6, thêm AC6a/6b, sửa AC17, thêm field `death_day`/`death_month`/`death_leap_month` vào `Memorial`; AC1 ghi rõ mục mới tạo không có ảnh; `OTHER` chỉ là một giá trị (không nhãn tự do); xác nhận "bà ngoại, cha, mẹ" là ba giá trị; ảnh tuỳ chọn. Hai marker (năm không rõ, nhãn `OTHER`) chuyển sang "Open questions (resolved)"; còn 7 marker.
- `design.md`: entity `DeathDate *time.Time` + `DeathDay/Month/LeapMonth`; schema `death_date` nullable + 3 cột mới + CHECK loại trừ hai dạng ngày mất; nhánh validate ở Step C-2. Số bước của các flow không đổi.
- `flow.md`: chỉ đổi nhãn validate ở Step C-2 của diagram tạo; không thêm/bỏ bước.
- `tasks.md`: bổ sung validation hai dạng ngày và CHECK vào Phase 2/3; cập nhật danh sách marker cần chốt ở Phase 5. Mọi mục vẫn `[ ]`.
- Reason: anh Đức cho năm mất tuỳ chọn. Vì ngày âm không đổi sang dương được khi thiếu năm, quyết định "chỉ lưu dương" giữ nguyên khi có năm, còn thiếu năm thì lưu ngày+tháng theo lịch đã nhập. Bump minor vì phần còn lại của luồng không đổi.
- Follow-up: `openapi.yaml` (+ `web/docs/public/openapi.yaml`), `migrations/0018_memorials.sql` + `docs/specs/db/erd.*` phải phản ánh 3 cột mới; OKF chưa tạo nên không có gì cần làm lại.

## v1.0 (2026-10-06)

- `requirements.md`: spec đầu tiên cho Sổ giỗ chạp, viết trước khi có code (Prospective): 5 user story `US-MEM-01..05`, 32 AC (EARS), contract 7 route đều `RequireAuth`, 9 marker `⚠️ Not confirmed`.
- `design.md`: 4 layer + storage adapter + migration `0018_memorials.sql`; ba Step-by-step Flow (tạo có khoá `users` để chặn vượt trần 100 khi chạy song song; upload ảnh DB-trước-GCS-sau; xoá).
- `flow.md`: ba sequence diagram mirror 1:1 các flow trên.
- `tasks.md`: 5 Phase, mọi mục `[ ]`.
- Reason: yêu cầu mới của anh Đức. Đã chốt trong phiên: chỉ lưu ngày dương + cờ `death_calendar` (client quy đổi âm↔dương), nhắc nhở local-only, upload ảnh multipart qua BE → GCS, giới hạn mặc định đề xuất.
- Follow-up chưa làm (không thuộc skill này): `docs/specs/api/openapi.yaml` + `web/docs/public/openapi.yaml`, `migrations/0018_memorials.sql` + `docs/specs/db/erd.*`, concept OKF `docs/knowledge/features/memorial.md` (sau khi code kiểm chứng).
