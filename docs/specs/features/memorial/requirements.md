# Requirements — memorial

Version: 4.0
Last updated: 2026-10-14
Status: v1 CRUD (US-MEM-01..04, 5 route) đã implement 2026-10-07 và có test unit/repository/E2E. **v4.0 đổi hành vi (chưa implement):** lựa chọn nhắc thu hẹp, múi giờ chuyển lên tài khoản, bỏ `reminder.timezone`/`add_to_os_reminder`, server lập lịch và gửi nhắc bằng job hằng giờ (US-MEM-06, US-MEM-07; `tasks.md` Phase 8–10, mọi mục `[ ]`). Ảnh (US-MEM-05) vẫn hoãn vì GCS (`docs/tech-debt/memorial/TD-001`). Marker mở: bucket GCS (Phase 6), feature notification chưa có spec (chặn Phase 10), hai câu hỏi product (xem Pending).
Module: `memorial`
Code: `services/appService/`
Product spec: `docs/product/memorial/product.md` @ v2.0 (câu trả lời về công tắc thông báo của tài khoản: product.md@2.1, FR-MEM-008)
Contract: `docs/specs/api/openapi.yaml` (this repo)
Architecture: ADR-0001 (inbox + push), ADR-0002 (server tính và gửi nhắc) — cả hai Accepted 2026-10-14
Implements: BR-MEM-01, BR-MEM-02, BR-MEM-03, BR-MEM-04, BR-MEM-05, BR-MEM-06, BR-MEM-07, BR-MEM-10, BR-MEM-11, BR-MEM-12, BR-MEM-13, FR-MEM-003, FR-MEM-004, FR-MEM-005, FR-MEM-007, FR-MEM-008, FR-MEM-009 (removed — bỏ field), FR-MEM-011 (planned), FR-MEM-012, FR-MEM-013, NFR-MEM-02, NFR-MEM-03, NFR-MEM-04

> See `changelog.md` in this folder for the version history of this spec.

Spec platform (layer 2) của **Sổ giỗ chạp** cho backend. Hành vi sản phẩm (sổ riêng theo tài khoản, nội dung và giới hạn một mục, ngày mất, tháng nhuận, kỳ giỗ tới, thời điểm nhắc, nhắc trễ, xoá vĩnh viễn) nằm ở product spec và **không** nhắc lại ở đây; file này chỉ nói backend hiện thực các `BR-MEM-*`/`FR-MEM-*` đó thế nào. Spec gốc viết prospectively (v1 implement đúng theo v2.2); v3.0 viết lại retroactively để trỏ lên product spec; v4.0 vừa retroactive (CRUD đã có) vừa prospective (phần nhắc từ server).

## Ràng buộc từ product spec

- **Lưu ngày mất như cũ, server giờ cũng tính kỳ giỗ để nhắc.** Backend vẫn lưu `death_date` dương lịch kèm `death_calendar` khi có năm, và ngày/tháng/cờ nhuận theo lịch đã nhập khi không có năm (AC6, AC6a). Theo ADR-0002 và product §2.7/§3.6 (dòng BR-MEM-05…10), server tự tính kỳ giỗ tới (BR-MEM-05…07) và thời điểm nhắc (BR-MEM-11) bằng `domain/lunarcal` — chỉ để nhắc; API **không** trả kỳ giỗ, số ngày còn lại hay lần giỗ (BR-MEM-08, 09 vẫn là việc của client).
- **Nhắc do server gửi** (ADR-0002, FR-MEM-008): mỗi mục có nhắc mang một kế hoạch dẫn xuất (`next_reminder_at`, `next_anniversary_date`); một Cloud Run Job chạy mỗi giờ gửi các nhắc đến hạn vào hộp thông báo (inbox) và đẩy push. **Inbox, đăng ký push token và `pkg/push`/`pkg/push/fcm` thuộc feature notification dùng chung (ADR-0001), không thuộc spec này** — memorial chỉ là producer đầu tiên.
- **Múi giờ thuộc tài khoản** (`users.timezone`, BR-MEM-11, Appendix B mục 3), không còn theo mục.
- **FR-MEM-009 bị bỏ**: `add_to_os_reminder` rời khỏi contract và schema (Appendix B mục 6).
- **Ảnh** (FR-MEM-011, planned): giới hạn 5 MB, JPEG/PNG/WebP khớp product spec; thiết kế upload multipart qua BE → GCS giữ ở US-MEM-05, hoãn tới Phase 6.
- Mã user story `US-MEM-01..05` và số AC giữ nguyên để không gãy tham chiếu trong code/test (AC10 bỏ nhưng giữ số); AC mới đánh số tiếp từ 34.

---

## US-MEM-01: Thêm người vào sổ giỗ

## User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **thêm một người thân đã mất vào sổ giỗ của mình kèm ngày mất và cấu hình nhắc**,
> Để **không quên ngày giỗ chạp hằng năm**.

## Acceptance Criteria (EARS notation)

1. WHEN a `POST /v1/memorials` request arrives with a valid access token and a valid body, THE SYSTEM SHALL create the memorial owned by the token's user and return `201 Created` with the full `Memorial` (shape ở "API contract"). Ảnh, ghi chú, nhắc nhở và năm mất đều tuỳ chọn; mục mới luôn tạo **không có ảnh** (ảnh gắn sau qua route ảnh, US-MEM-05). (FR-MEM-003, BR-MEM-01)
2. WHEN the request carries no valid access token (thiếu, hết hạn, sai loại token), THE SYSTEM SHALL return `401` and SHALL NOT create anything. (BR-MEM-01)
3. WHEN `full_name` is empty/blank after trim or longer than 100 characters, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "full_name"`. (BR-MEM-02, FR-MEM-012)
4. WHEN `relationship` is not one of `GRANDFATHER_PATERNAL` (ông nội), `GRANDFATHER_MATERNAL` (ông ngoại), `GRANDMOTHER_PATERNAL` (bà nội), `GRANDMOTHER_MATERNAL` (bà ngoại), `FATHER` (cha), `MOTHER` (mẹ), `SPOUSE` (vợ/chồng), `SIBLING` (anh chị em), `CHILD` (con), `OTHER` (khác), THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "relationship"`. (BR-MEM-02)
5. WHEN `death_calendar` is not `solar` or `lunar`, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "death_calendar"`. (BR-MEM-03)
6. WHEN the request carries a full `death_date` (năm đã biết), THE SYSTEM SHALL require it to be a valid `YYYY-MM-DD` solar date not earlier than `1900-01-01` and not later than today in `Asia/Ho_Chi_Minh`, and SHALL require `death_day`, `death_month`, `death_leap_month` to be absent; otherwise it SHALL return `400 VALIDATION_ERROR` with `details.field = "death_date"`. (Áp dụng cho cả ngày đã được client quy đổi từ âm lịch — server chỉ thấy ngày dương.) (BR-MEM-03)
6a. WHEN the request has `death_date` absent or `null` (năm không rõ), THE SYSTEM SHALL require `death_day` and `death_month` and accept an optional `death_leap_month` (mặc định `false`), validated by `death_calendar`: **solar** → `death_month` 1..12, `death_day` 1..số ngày của tháng đó (tháng 2 cho phép 29), `death_leap_month` phải `false`; **lunar** → `death_month` 1..12, `death_day` 1..30. Otherwise it SHALL return `400 VALIDATION_ERROR` with `details.field` = `death_day` / `death_month` / `death_leap_month`. (Server không biết tháng âm có 29 hay 30 ngày — không kiểm sâu hơn.) (BR-MEM-03, BR-MEM-04)
6b. WHEN the request has neither a `death_date` nor both `death_day` and `death_month`, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "death_date"` — một mục luôn có ít nhất ngày và tháng mất. (BR-MEM-03)
7. WHEN `note` is longer than 1000 characters, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "note"`; WHEN `note` is absent, `null` or blank, THE SYSTEM SHALL store `NULL` and return `note: null`. (BR-MEM-02, FR-MEM-013)
8. WHEN `reminder` is `null` or absent, THE SYSTEM SHALL store no reminder and return `reminder: null` (nhắc tắt). Mặc định "bật, trước 1 ngày, 07:00" của FR-MEM-003 là mặc định của form ở client; client gửi nó lên như mọi giá trị khác. (BR-MEM-11)
9. **(v4.0)** WHEN `reminder` is present, THE SYSTEM SHALL require `days_before` to be one of `0`, `1`, `3`, `7` and `time` to be one of `"06:00"`, `"07:00"`, `"12:00"`, `"19:00"`; otherwise it SHALL return `400 VALIDATION_ERROR` with `details.field` = `reminder.days_before` / `reminder.time`, and SHALL NOT write anything. (BR-MEM-11, FR-MEM-007)
10. *(Bỏ ở v4.0 — FR-MEM-009 removed. `add_to_os_reminder` không còn trong contract; gửi lên thì bị AC12 từ chối như field lạ.)*
11. WHEN the user already owns 100 memorials, THE SYSTEM SHALL return `409 MEMORIAL_LIMIT_REACHED` and SHALL NOT insert — kể cả khi hai request tạo chạy đồng thời (không thể vượt 100). (BR-MEM-02)
12. WHEN the body has unknown fields, is empty, or is not valid JSON, THE SYSTEM SHALL return `400 VALIDATION_ERROR` (cùng cơ chế `DecodeJSONBodyStrict` như các route khác). WHEN the body sets `photo_url`, `id`, `user_id`, any server-owned field, **or a field removed in v4.0 (`add_to_os_reminder`, `reminder.timezone`)**, THE SYSTEM SHALL reject it as an unknown field. (platform)

## US-MEM-02: Xem sổ giỗ của mình

## User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **xem danh sách và chi tiết các mục trong sổ giỗ của mình**,
> Để **biết ai đã ghi và cấu hình nhắc hiện tại**.

## Acceptance Criteria (EARS notation)

13. WHEN a `GET /v1/memorials` request arrives with a valid access token, THE SYSTEM SHALL return `200` with `data.items` holding **only** memorials owned by the token's user, ordered by `created_at` DESC then `id` DESC, and `data.total`. Không phân trang (tối đa 100 mục). WHEN the user has none, THE SYSTEM SHALL return `items: []` (không phải `null`). (BR-MEM-01)
14. WHEN a `GET /v1/memorials/{id}` request arrives AND the memorial exists AND belongs to the token's user, THE SYSTEM SHALL return `200` with the full `Memorial`. (BR-MEM-01)
15. WHEN `{id}` does not exist, is not a positive integer, OR belongs to another user, THE SYSTEM SHALL return the **same** `404 MEMORIAL_NOT_FOUND` for all three — không phân biệt "không tồn tại" với "của người khác" để không lộ sự tồn tại của dữ liệu. (BR-MEM-01)
16. WHEN a memorial has a photo, THE SYSTEM SHALL return `photo_url` as a signed GCS URL valid for 15 minutes; WHEN it has none, THE SYSTEM SHALL return JSON `null`. Client không được cache `photo_url` lâu hơn hạn của nó. (FR-MEM-011, NFR-MEM-02)
17. WHEN a memorial has a full death date, THE SYSTEM SHALL return `death_date` as a `"YYYY-MM-DD"` string in solar calendar (regardless of which calendar the user entered it in) with `death_day`, `death_month`, `death_leap_month` all `null`; WHEN the year is unknown, THE SYSTEM SHALL return `death_date: null` with `death_day`, `death_month`, `death_leap_month` (bool) in the stored `death_calendar`. `death_calendar` luôn có mặt. (BR-MEM-03, BR-MEM-04)
17a. **(v4.0)** WHEN a stored memorial has a `reminder` whose `days_before` or `time` is outside the AC9 choices (ghi trước v4.0), THE SYSTEM SHALL return the stored values unchanged in `GET` responses and SHALL keep planning and sending its reminders with those stored values until the entry is replaced by a `PUT`. (FR-MEM-007)
17b. **(v4.0)** WHEN any memorial is returned, THE SYSTEM SHALL return `reminder` as `{days_before, time}` (không còn `timezone`), SHALL NOT return `add_to_os_reminder`, and SHALL NOT expose the derived plan (`next_reminder_at`, `next_anniversary_date`). (platform, FR-MEM-009)

## US-MEM-03: Sửa một mục trong sổ giỗ

## User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **sửa thông tin một người đã ghi trong sổ giỗ**,
> Để **chỉnh lại thông tin sai hoặc đổi cấu hình nhắc**.

## Acceptance Criteria (EARS notation)

18. WHEN a `PUT /v1/memorials/{id}` request arrives with a valid body AND the memorial belongs to the token's user, THE SYSTEM SHALL replace `full_name`, `relationship`, `death_date`, `death_day`, `death_month`, `death_leap_month`, `death_calendar`, `note`, `reminder` with the request values (full replacement — mọi field bắt buộc gửi như khi tạo, trừ field tuỳ chọn; thiếu `reminder` = tắt nhắc), re-plan the reminder (AC34/AC35), refresh `updated_at`, and return `200` with the full `Memorial`. (FR-MEM-004)
19. WHEN a `PUT` is processed, THE SYSTEM SHALL NOT change `id`, `user_id`, `created_at` or the photo; ảnh chỉ đổi qua route ảnh (US-MEM-05). (FR-MEM-004)
20. WHEN a `PUT` body fails any validation of AC3–AC9 and AC12, THE SYSTEM SHALL return the same `400 VALIDATION_ERROR` as create and SHALL leave the row (kể cả kế hoạch nhắc) unchanged. (FR-MEM-004, BR-MEM-02, BR-MEM-03, BR-MEM-11)
21. WHEN `{id}` does not exist or belongs to another user, THE SYSTEM SHALL return `404 MEMORIAL_NOT_FOUND` (AC15) and SHALL NOT modify any row. (BR-MEM-01)

## US-MEM-04: Xoá một mục khỏi sổ giỗ

## User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **xoá một người khỏi sổ giỗ của mình**,
> Để **dọn mục ghi nhầm hoặc không còn cần**.

## Acceptance Criteria (EARS notation)

22. WHEN a `DELETE /v1/memorials/{id}` request arrives AND the memorial belongs to the token's user, THE SYSTEM SHALL delete the row (xoá cứng, kéo theo kế hoạch nhắc của nó) and return `200` `{"success": true}` (không `data`, theo `api-response-envelope` AC11). (FR-MEM-005, BR-MEM-12)
23. WHEN a memorial with a photo is deleted, THE SYSTEM SHALL also delete the GCS object **after** the row is gone, best-effort: WHEN the object delete fails, THE SYSTEM SHALL still return `200`, log the failure with the object key (không log tên/ghi chú), and leave the orphan object for a cleanup job (⚠️ chưa có — xem Pending). (FR-MEM-011, NFR-MEM-02)
24. WHEN `{id}` does not exist or belongs to another user — kể cả gọi lại lần hai sau khi đã xoá — THE SYSTEM SHALL return `404 MEMORIAL_NOT_FOUND` and SHALL NOT delete anything. (BR-MEM-01, BR-MEM-12)

## Áp dụng cho mọi route và cho job nhắc

33. WHEN any `/v1/memorials` route or the reminder job (US-MEM-07) handles a request, an entry or an error, THE SYSTEM SHALL NOT write `full_name`, `note` or photo content to logs, metrics or traces; only ids, dates and object keys may be logged. (NFR-MEM-02)

## US-MEM-05: Ảnh của người đã mất

> ⏸ **HOÃN — chưa thuộc v1.** Chặn bởi hạ tầng GCS. AC25–AC32, nhánh "có ảnh" của AC16 và các nhánh xoá object ở AC23 chưa được implement; cho tới Phase 6, `photo_url` luôn là `null` và hai route ảnh chưa tồn tại. Nội dung dưới đây giữ nguyên để làm việc ngay khi GCS sẵn sàng.

## User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **tải, thay hoặc gỡ ảnh của người trong sổ giỗ**,
> Để **thấy di ảnh khi mở sổ giỗ**.

## Acceptance Criteria (EARS notation)

25. WHEN a `PUT /v1/memorials/{id}/photo` request arrives as `multipart/form-data` with one file part `photo` AND the memorial belongs to the token's user, THE SYSTEM SHALL store the image in the private bucket, record its object key, and return `200` with the full `Memorial` (có `photo_url` mới). (FR-MEM-011)
26. WHEN the file is larger than 5 MB, THE SYSTEM SHALL return `413 PHOTO_TOO_LARGE` and SHALL NOT write to storage (giới hạn áp bằng `http.MaxBytesReader` trước khi đọc hết body). (FR-MEM-011)
27. WHEN the file's **detected** content type (sniff nội dung, không tin header `Content-Type` hay đuôi tên file) is not `image/jpeg`, `image/png` or `image/webp`, THE SYSTEM SHALL return `415 PHOTO_UNSUPPORTED_TYPE` and SHALL NOT write to storage. (FR-MEM-011)
28. WHEN the `photo` part is missing or empty, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "photo"`. (FR-MEM-011)
29. WHEN the memorial already has a photo, THE SYSTEM SHALL replace it: the new object is written and recorded first, the old object is deleted afterwards best-effort (AC23 áp dụng tương tự). Tên object **luôn mới** (`<uuid>`), không ghi đè object cũ. (FR-MEM-011)
30. WHEN `DELETE /v1/memorials/{id}/photo` arrives AND the memorial belongs to the token's user, THE SYSTEM SHALL clear the photo (row trước, object sau, best-effort) and return `200` with the full `Memorial` (`photo_url: null`); WHEN it had no photo, THE SYSTEM SHALL still return `200` (idempotent). (FR-MEM-011)
31. WHEN `{id}` does not exist or belongs to another user, THE SYSTEM SHALL return `404 MEMORIAL_NOT_FOUND` **before** reading the uploaded body into storage — người lạ không thể dùng route này để đẩy file lên bucket. (FR-MEM-011)
32. WHEN a storage call fails (ghi/ký URL), THE SYSTEM SHALL return `500 INTERNAL_ERROR` without leaking the cause, and the row's `photo_object_key` SHALL stay pointing at the previous object (hoặc `NULL`) — không bao giờ trỏ vào object chưa ghi được. (FR-MEM-011)

## US-MEM-06: Lập kế hoạch nhắc (v4.0, chưa implement)

## User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **mỗi mục có nhắc luôn có sẵn lần nhắc kế tiếp đúng với ngày giỗ, cấu hình nhắc và múi giờ tài khoản của tôi**,
> Để **server nhắc đúng lúc mà không cần máy tôi tự đặt lịch**.

## Acceptance Criteria (EARS notation)

34. WHEN a memorial is created (AC1) or replaced (AC18) with a `reminder`, THE SYSTEM SHALL, inside the same transaction as the write and after reading the account time zone under the `users` row lock, compute and store the entry's next reminder moment (`next_reminder_at`, UTC instant) and the anniversary it is for (`next_anniversary_date`, solar date) from the entry's date of death, its lead and time and the account time zone, per BR-MEM-05, BR-MEM-06, BR-MEM-07 and BR-MEM-11 — including the case where this anniversary's moment has already passed (no reminder for it; the following anniversary is planned) — and SHALL NOT create any notification during the request. (BR-MEM-05, BR-MEM-06, BR-MEM-07, BR-MEM-11, FR-MEM-004)
35. WHEN a memorial is created or replaced without a `reminder`, THE SYSTEM SHALL store `next_reminder_at` and `next_anniversary_date` as `NULL`, so that no reminder for it is sent by any job run that starts after the write commits. (FR-MEM-008)
36. WHEN a memorial is deleted (AC22), THE SYSTEM SHALL send no reminder for it from any job run that starts after the delete commits (kế hoạch mất cùng dòng). (FR-MEM-005, FR-MEM-008, BR-MEM-12)
37. WHEN the account time zone of a user is reported (`PATCH /v1/users/me` with `timezone` — route của feature `profile`, xem Ghi chú), THE SYSTEM SHALL validate it as an IANA zone name, store it in `users.timezone`, and recompute `next_reminder_at`/`next_anniversary_date` of every memorial of that user that has a reminder, using the new zone, then either:
    - on success: return the profile response as the `profile` spec defines, or
    - on failure (recompute fails after the zone was saved): return `500 INTERNAL_ERROR`, keep the saved zone, and recompute again on the next report of the same zone (lệnh báo múi giờ là idempotent; client báo lại khi đăng nhập). (BR-MEM-11)
38. WHEN a user has never reported a time zone, THE SYSTEM SHALL plan that user's reminders in `Asia/Ho_Chi_Minh` (giá trị mặc định của `users.timezone`). (BR-MEM-11)
39. WHEN a memorial has a reminder but no plan (`next_reminder_at IS NULL` — dòng có từ trước migration v4.0, hoặc lần tính lại trước đó lỗi), THE SYSTEM SHALL plan it at the start of the next job run (AC40) with the same rule as AC34, before any delivery. (platform)

## US-MEM-07: Gửi nhắc giỗ (job hằng giờ, v4.0, chưa implement)

## User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **nhận đúng một lời nhắc cho mỗi kỳ giỗ trong mục Thông báo trên mọi nền tảng, và push trên điện thoại**,
> Để **chuẩn bị cúng giỗ kịp thời**.

## Acceptance Criteria (EARS notation)

40. WHEN the hourly reminder job runs (Cloud Scheduler → Cloud Run Job `memorial-reminders`, entry point riêng ở `cmd/`), THE SYSTEM SHALL claim memorials whose `next_reminder_at <= now()` one at a time with `FOR UPDATE SKIP LOCKED`, so that two overlapping executions never process the same entry at once, and for each claimed entry SHALL, in one database transaction: decide delivery per BR-MEM-13 in the account time zone (`rules/reminder-moment.json` delivery cases); when delivering, create one memorial reminder in the user's notification inbox with idempotency key (`memorial_id`, `next_anniversary_date`); then roll the plan forward to the anniversary after `next_anniversary_date` (AC34 rule); and commit. (FR-MEM-008, BR-MEM-11, BR-MEM-13)
41. WHEN a claimed entry's anniversary day has already ended in the account time zone, THE SYSTEM SHALL create no notification and send no push for that anniversary, and SHALL roll the plan forward. (BR-MEM-13)
42. WHEN a notification with the same idempotency key already exists (job chạy lại, retry sau lỗi giữa chừng), THE SYSTEM SHALL NOT create a second notification and SHALL NOT push again, and SHALL still roll the plan forward. (BR-MEM-13, FR-MEM-008)
43. WHEN the transaction of AC40 committed a newly created notification, THE SYSTEM SHALL then push it through the notification feature (ADR-0001) to every push token the user has registered, with the text of product §3.1.1 chosen by the days left at send time (anniversary date − today in the account time zone) and the relationship — never the relative's name — and WHEN the push fails THE SYSTEM SHALL log ids only and SHALL NOT remove the notification nor change the plan. Push bị tắt bởi công tắc thông báo của tài khoản (chỉ push dừng, nhắc vẫn vào inbox — product.md@2.1, FR-MEM-008) do feature notification áp, không phải job này. (FR-MEM-008, NFR-MEM-02)
44. WHEN planning or delivering one entry fails (lỗi tính lịch, lỗi DB), THE SYSTEM SHALL roll back that entry's transaction, log its id, count it in a failure metric, skip it for the rest of the run, and continue with the other entries; the entry stays due and is retried by the next run. (BR-MEM-13, NFR-MEM-04)
45. WHILE the job is scheduled hourly at minute 0, THE SYSTEM SHALL put every reminder whose moment is on a whole hour into the inbox within one hour of that moment, except during an outage (BR-MEM-13), and SHALL emit per-run counts of claimed, sent, skipped-late, already-sent and failed entries plus the count of entries overdue by more than one hour. (NFR-MEM-04)
46. THE SYSTEM's reminder planner SHALL pass every case of `docs/product/memorial/rules/reminder-moment.json` (moment and delivery cases) and the `next_date` of every case of `rules/next-anniversary.json`, and `domain/lunarcal` SHALL pass every case of `rules/death-date-conversion.json`, each loaded from the repo, never copied. (NFR-MEM-03, BR-MEM-03, BR-MEM-04, BR-MEM-05, BR-MEM-06, BR-MEM-07, BR-MEM-10, BR-MEM-11, BR-MEM-13)

## API contract (confirmed surface for this feature)

Có trong `docs/specs/api/openapi.yaml` (tag Memorials, 5 route; hai route ảnh chưa có vì hoãn) và `web/docs/public/openapi.yaml`. **v4.0 đổi schema `Memorial`/`MemorialRequest` (bỏ `add_to_os_reminder`, bỏ `reminder.timezone`, thu hẹp `reminder`) — `openapi.yaml` chưa cập nhật, xem `tasks.md` Phase 8.** Mọi route đều `RequireAuth` (`authMiddleware.AuthenticateFunc`, `RequireTokenType: "access"`); `user_id` luôn lấy từ token, **không bao giờ** từ path/body.

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| POST | `/v1/memorials` | RequireAuth | `201` `Memorial` | 400 `VALIDATION_ERROR`, 401, 409 `MEMORIAL_LIMIT_REACHED`, 405, 500 |
| GET | `/v1/memorials` | RequireAuth | `200` `{items: [Memorial], total}` | 401, 405, 500 |
| GET | `/v1/memorials/{id}` | RequireAuth | `200` `Memorial` | 401, 404 `MEMORIAL_NOT_FOUND`, 405, 500 |
| PUT | `/v1/memorials/{id}` | RequireAuth | `200` `Memorial` | 400 `VALIDATION_ERROR`, 401, 404, 405, 500 |
| DELETE | `/v1/memorials/{id}` | RequireAuth | `200` `{"success": true}` | 401, 404, 405, 500 |
| PUT | `/v1/memorials/{id}/photo` ⏸ hoãn | RequireAuth | `200` `Memorial` | 400 `VALIDATION_ERROR`, 401, 404, 413 `PHOTO_TOO_LARGE`, 415 `PHOTO_UNSUPPORTED_TYPE`, 500 |
| DELETE | `/v1/memorials/{id}/photo` ⏸ hoãn | RequireAuth | `200` `Memorial` | 401, 404, 500 |
| PATCH | `/v1/users/me` (field `timezone`, **thuộc `profile`**) | RequireAuth | như `profile` | 400 (mã lỗi do spec `profile` chốt), 401, 500 (AC37) |

Không có route mới của memorial cho nhắc: job không phải HTTP. Inbox (`list`, `unread count`, `mark read`) và đăng ký push token là route của feature notification (ADR-0001).

`Memorial` (tóm tắt để đọc nhanh; nguồn đúng là schema `Memorial`/`MemorialRequest` trong `openapi.yaml`):

| Field | Kiểu | Ghi chú |
|---|---|---|
| `id` | int | |
| `full_name` | string | ≤ 100 ký tự |
| `relationship` | enum | 10 giá trị ở AC4 |
| `death_date` | `"YYYY-MM-DD"` \| `null` | dương lịch; `null` khi năm không rõ |
| `death_calendar` | `"solar"` \| `"lunar"` | lịch user đã nhập — client dùng để hiển thị lại; khi năm không rõ thì là lịch của `death_day`/`death_month` |
| `death_day`, `death_month` | int \| `null` | chỉ có khi năm không rõ, theo `death_calendar` |
| `death_leap_month` | bool \| `null` | chỉ có khi năm không rõ (và `lunar` mới có thể `true`); khi có `death_date`, cờ nhuận khôi phục bằng cách đổi `death_date` dương → âm |
| `note` | string \| `null` | ≤ 1000 ký tự |
| `photo_url` | string \| `null` | signed URL, hạn 15 phút; **luôn `null` cho tới khi làm US-MEM-05** |
| `reminder` | `{days_before: 0\|1\|3\|7, time: "06:00"\|"07:00"\|"12:00"\|"19:00"}` \| `null` | v4.0; dòng cũ có thể trả giá trị ngoài danh sách (AC17a) |
| `created_at`, `updated_at` | RFC 3339 | |

Request tạo/sửa gửi cùng các field trên trừ `id`, `photo_url`, `created_at`, `updated_at`. Bỏ ở v4.0: `add_to_os_reminder`, `reminder.timezone`.

## Out of scope (this feature boundary, not tracked here)

- **Feature notification** (ADR-0001): bảng inbox, API list/unread/mark-read, đăng ký/xoá push token, `pkg/push` + `pkg/push/fcm`, dọn token khi đăng xuất hoặc FCM báo token hỏng, công tắc thông báo của tài khoản. Memorial chỉ gọi vào nó (xem Pending).
- Trả kỳ giỗ tới / số ngày còn lại / lần giỗ trong API, lọc "giỗ trong 30 ngày tới", sắp theo ngày giỗ — client tính (BR-MEM-08, 09; product §3.6).
- Hạ tầng: tạo Cloud Scheduler job, Cloud Run Job, service account và quyền FCM — làm ở hạ tầng (ADR-0002 Follow-ups), spec này chỉ định nghĩa binary.
- Chia sẻ sổ giỗ cho người khác (gia đình cùng xem/sửa) — chỉ chủ sở hữu.
- Nhiều ảnh cho một mục; xoá mềm/khôi phục; tìm kiếm; phân trang.
- Hỏi xác nhận trước khi xoá (BR-MEM-12), xem offline (FR-MEM-010), preview kỳ giỗ (FR-MEM-006), hiện lựa chọn gần nhất cho giá trị nhắc cũ (FR-MEM-007) — việc của client.
- Xoá tài khoản kéo theo dọn ảnh trên GCS (chưa có feature xoá tài khoản).

## Deferred (có theo dõi, không chặn v1)

- **US-MEM-05 — ảnh** (route `PUT/DELETE /v1/memorials/{id}/photo`, adapter GCS, signed URL): hoãn vì chưa làm được GCS. Theo dõi ở `tasks.md` Phase 6.
- **Dọn object GCS mồ côi** (xoá object thất bại, hoặc xoá tài khoản sau này): "tính sau" — chỉ có ý nghĩa khi đã có ảnh, nên đi cùng Phase 6.

## Pending (not yet implemented)

- Ảnh (US-MEM-05) — `tasks.md` Phase 6, nợ `docs/tech-debt/memorial/TD-001`.
- Contract nhắc v4.0 (AC9, AC10 bỏ, AC12, AC17a/17b) — Phase 8.
- Lập kế hoạch nhắc (US-MEM-06) — Phase 9.
- Job gửi nhắc (US-MEM-07) — Phase 10, bị chặn bởi feature notification.
- Chạy vector (AC46) — Phase 7 (`death-date-conversion.json`) và Phase 9 (`next-anniversary.json`, `reminder-moment.json`).

> ⚠️ Not confirmed: bucket GCS cho ảnh sổ giỗ — `config.dev.yaml` chỉ có `innerlamp-avatars-dev`; dùng chung bucket (prefix `memorials/`) hay tạo bucket riêng, và service account Cloud Run có quyền ký signed URL (`iam.serviceAccounts.signBlob`) chưa — chặn Phase 6 (ảnh), **không** chặn v1.

> ⚠️ Not confirmed: feature notification (ADR-0001) chưa có spec (`docs/specs/features/notification/` chưa tồn tại) — schema inbox, cột/ràng buộc UNIQUE cho idempotency key, cách ghi một notification **trong cùng transaction** với job (AC40), cổng gửi push sau commit (AC43), dữ liệu một notification loại memorial mang theo — chặn Phase 10 (AC40–AC43); spec này chỉ nêu những gì memorial cần (xem `design.md` "Phụ thuộc feature notification").

> ⚠️ Not confirmed (product): khi một mục bị xoá, các lời nhắc **đã gửi** của nó trong mục Thông báo còn ở đó không, và nếu còn thì dòng chi tiết "{tên} · {ngày dương, thứ}" (product §3.1.1) lấy tên từ đâu khi mục không còn — BR-MEM-12 nói xoá "for good on every device" còn §3.5 nói lời nhắc trong inbox "kept after reading" — chặn cách feature notification lưu tên (lưu bản sao hay đọc lúc hiển thị); hand back to ba-writer in innerlamp-be.

## Ghi chú kỹ thuật liên quan

- **Vì sao chỉ lưu dương lịch là ổn** (khi có năm). Ngày mất là *một ngày trong quá khứ*, đã cố định; quy đổi nó âm ↔ dương mang tính xác định, nên lưu dương + cờ `death_calendar` **không mất thông tin** — client (và từ v4.0 cả server, khi lập kế hoạch nhắc) đổi lại ra đúng ngày âm đã nhập, kể cả tháng nhuận. Rủi ro là **thuật toán âm lịch ở client** khi quy đổi trước lúc gửi: sai một lần là dương lưu sai vĩnh viễn. Giảm rủi ro: BR-MEM-10, NFR-MEM-03 và vector `rules/*.json`.
- **Server tính kỳ giỗ chỉ để nhắc.** Kế hoạch nhắc dùng `domain/lunarcal` (UTC+7, BR-MEM-10) cho ngày âm, còn múi giờ tài khoản chỉ quyết định "hôm nay" và giờ địa phương của lời nhắc (ADR-0002). Mục âm lịch có năm: server đổi `death_date` dương → âm để lấy ngày/tháng; cờ nhuận không đổi kỳ giỗ (BR-MEM-04, BR-MEM-05).
- **Giá trị nhắc cũ ngoài danh sách (quyết định platform v4.0).** Không viết lại dữ liệu khi migrate: product (FR-MEM-007) muốn giữ giá trị đã lưu tới khi user lưu lại, nên DB vẫn cho `reminder_days_before` 0..30 và mọi `reminder_time`; chỉ request ghi mới bị thu hẹp (AC9). Job gửi đúng giá trị cũ (AC17a): lead bất kỳ vẫn tính được, giờ lẻ như 07:30 được gửi ở lượt chạy đầu giờ kế tiếp (≤ 1 giờ, vẫn trong NFR-MEM-04). Sản phẩm chưa phát hành nên thực tế chỉ có dữ liệu dev.
- **Đổi contract không versioning (quyết định platform v4.0).** Sản phẩm chưa phát hành (product §2.7), nên bỏ `add_to_os_reminder` và `reminder.timezone` ngay trên `/v1`, không giữ field "nhận rồi bỏ qua", không mở `/v2`. Client gửi field cũ nhận `400 VALIDATION_ERROR` (AC12) — lỗi rõ ràng tốt hơn âm thầm bỏ qua. `info.version` của `openapi.yaml` tăng major khi cập nhật. Ba client đều phải theo product v2.0 nên phải đổi cùng lúc.
- **Báo múi giờ tài khoản (quyết định platform v4.0).** Client gửi `timezone` (tên IANA của máy) qua `PATCH /v1/users/me` ngay sau khi đăng nhập và khi múi giờ máy đổi (product Appendix B mục 3) — dùng lại route sẵn có của `profile` thay vì thêm field vào `POST /v1/auth/login`, để một đường duy nhất vừa đặt vừa đổi. `GET /v1/users/me` trả `timezone`. Spec `profile` phải thêm field này (follow-up, không sửa ở đây). Mỗi lần báo đều tính lại kế hoạch kể cả khi múi giờ không đổi — rẻ (≤ 100 mục) và giúp AC37 tự lành sau lỗi.
- **Công tắc thông báo của tài khoản** (ADR-0001; product.md@2.1, FR-MEM-008): chỉ tắt push trên mọi thiết bị của tài khoản, lời nhắc vẫn vào inbox. Nên job memorial **luôn** tạo notification khi đến hạn; việc bỏ push thuộc feature notification. Quyền thông báo ở OS của từng máy là chuyện riêng của máy đó.
- **Không gửi hai lần** dựa vào idempotency key (`memorial_id`, `next_anniversary_date`) có UNIQUE ở DB, không chỉ dựa vào khoá dòng — `FOR UPDATE SKIP LOCKED` chỉ để hai lượt chạy chồng nhau không làm cùng một việc.
- **Push chỉ có quan hệ và số ngày** (NFR-MEM-02, §3.1.1): `OTHER` hiện là "người thân". Số ngày tính lúc gửi, nên lời nhắc "trước 1 ngày" gửi trễ vào đúng ngày giỗ (BR-MEM-13) dùng câu "Hôm nay là giỗ …".
- **Quyền sở hữu** áp ở tầng SQL: mọi truy vấn một mục đều có `WHERE id = $1 AND user_id = $2`; `user_id` chỉ lấy từ token. id là số tự tăng (dễ đoán) nên 404 đồng nhất ở AC15 là biện pháp chống dò id.
- **Dữ liệu cá nhân nhạy cảm** (tên người thân, ghi chú, ảnh): không log `full_name`/`note` ở route lẫn job; ảnh không public, chỉ signed URL 15 phút.
- **Giới hạn 100 mục/user**, kiểm trong transaction có khoá dòng `users` (xem `design.md`) để request song song không vượt trần; cùng khoá đó giờ còn để đọc múi giờ nhất quán khi lập kế hoạch.
- **Không có unique** trên (`user_id`, `full_name`, `death_date`) — hai mục trùng được phép; chống bấm đúp là việc của client.
- Enum `relationship` (AC4) map 1:1 với 10 giá trị của BR-MEM-02; `OTHER` không kèm nhãn tự do.
- **Tháng nhuận khi biết năm (BR-MEM-04).** Server không nhận `death_leap_month` khi có `death_date` (AC6) mà không mất thông tin: cờ nhuận khôi phục được một-một khi đổi `death_date` dương → âm (Appendix B mục 1). Việc từ chối "10/6 nhuận/2026" và giới hạn "năm âm ≥ 1900" là của client, vì server chỉ thấy ngày dương đã quy đổi (Appendix B mục 4).
- **`PUT` thay toàn bộ (AC18).** Thiếu `reminder` nghĩa là tắt nhắc (và xoá kế hoạch). Mọi nền tảng giờ đều hiện cấu hình nhắc (FR-MEM-007), nên không còn trường hợp "nền tảng không hiện nhắc phải gửi lại".
- **Hai dạng ngày mất** (có năm: ngày dương đầy đủ; không năm: ngày + tháng theo lịch đã nhập) loại trừ nhau, ép bằng CHECK ở DB.

## Open questions (resolved during implementation)

- ~~Giá trị nhắc ngoài 0/1/3/7 × 06:00/07:00/12:00/19:00 đã lưu trước v4.0 xử lý thế nào?~~ → **Giữ nguyên trong DB, trả nguyên, job gửi theo giá trị cũ; chỉ request ghi mới bị thu hẹp** (AC9, AC17a). Quyết định platform theo product Appendix B mục 2 và FR-MEM-007. Resolved 2026-10-14.
- ~~Bỏ `reminder.timezone`/`add_to_os_reminder` có cần versioning/tương thích ngược?~~ → **Không**: sản phẩm chưa phát hành, bỏ thẳng trên `/v1`, field cũ bị từ chối như field lạ (AC12). Quyết định platform theo ADR-0002 Follow-ups. Resolved 2026-10-14.
- ~~Client báo múi giờ tài khoản qua đâu?~~ → `PATCH /v1/users/me` field `timezone`, sau đăng nhập và khi đổi (AC37). Quyết định platform. Resolved 2026-10-14.
- ~~Công tắc thông báo của tài khoản có ẩn cả lời nhắc trong inbox?~~ → **Không — chỉ tắt push** trên mọi thiết bị; lời nhắc vẫn vào inbox (anh Đức, 2026-10-14; product.md@2.1, FR-MEM-008). Không đổi gì ở memorial: job luôn tạo notification; feature notification áp công tắc cho push (AC43). Resolved 2026-10-14.
- ~~Product spec Appendix B mục 2: giữ khoảng nhắc 0..30 ngày / mọi `HH:mm` hay thu hẹp về lựa chọn của product?~~ → v3.0 chọn **giữ rộng**; **v4.0 thay bằng thu hẹp** (product v2.0 Appendix B mục 2: server gửi nhắc theo giờ chẵn). Resolved 2026-10-14, superseded 2026-10-14.
- ~~Product spec Appendix B mục 1, 3, 4, 5, 6 có cần đổi backend?~~ → v3.0: không. **v4.0:** mục 3 (múi giờ tài khoản) và mục 6 (bỏ cờ iOS) giờ đổi backend (AC9–AC12, AC37); mục 1 vẫn chỉ cần mô tả trong `openapi.yaml` (Phase 7). Resolved 2026-10-14.
- ~~Thứ tự danh sách `GET /v1/memorials`?~~ → **`created_at` DESC, `id` DESC** (mục mới thêm hiện trước), như AC13. Backend không trả ngày giỗ kế tiếp nên client tự sắp lại theo ngày giỗ. Resolved 2026-10-07.
- ~~Mã user story?~~ → Spec tự chọn `US-MEM-01..05` (v4.0 thêm `US-MEM-06..07`). Resolved 2026-10-07.
- ~~Tháng nhuận: quy ước giỗ các năm không có tháng nhuận đó?~~ → Dùng tháng thường cùng số — nay là BR-MEM-05 của product; server áp khi lập kế hoạch nhắc (v4.0). Resolved 2026-10-07.
- ~~Ảnh HEIC/HEIF từ iPhone?~~ → **App đổi HEIC/HEIF → JPEG trước khi upload**, BE chỉ nhận jpeg/png/webp — `docs/assumptions/memorial/A-001-heic-client-converts.md` (đã xác nhận 2026-10-07). Resolved 2026-10-07.
- ~~Cờ `add_to_os_reminder` trên nhiều thiết bị / nhiều OS?~~ → Client tự xử lý. Resolved 2026-10-07; **superseded v4.0**: cờ bị bỏ (FR-MEM-009 removed).
- ~~Ảnh bằng GCS ngay bây giờ?~~ → **Tạm bỏ qua** vì chưa làm được; US-MEM-05 chuyển sang Deferred. Resolved 2026-10-07.
- ~~Dọn object GCS mồ côi?~~ → **Tính sau**, đi cùng ảnh (Deferred). Resolved 2026-10-07.
- ~~Ngày mất chỉ biết ngày/tháng, không biết năm?~~ → **Năm là tuỳ chọn** (AC6a, AC6b, AC17). Resolved 2026-10-07.
- ~~`relationship = OTHER` có cần nhãn tự do?~~ → **Không**. Resolved 2026-10-07.
- ~~Danh sách quan hệ "bà ngoại cha, me" nghĩa là gì?~~ → Ba giá trị riêng: bà ngoại, cha, mẹ. Resolved 2026-10-07.
- ~~Ảnh có bắt buộc không?~~ → **Tuỳ chọn**; mục tạo mới không có ảnh. Resolved 2026-10-07.
- ~~Lưu âm lịch hay chỉ dương?~~ → **Chỉ lưu dương lịch** khi có năm + cờ `death_calendar`. Resolved 2026-10-06 (phần "server không tính ngày giỗ" superseded v4.0 bởi ADR-0002).
- ~~Ai gửi nhắc nhở?~~ → Local-only trên app. Resolved 2026-10-06; **superseded v4.0** bởi ADR-0002: server gửi (US-MEM-06, US-MEM-07).
- ~~Upload ảnh theo cách nào?~~ → **Multipart qua BE → GCS**. Resolved 2026-10-06.
- ~~Giới hạn và kiểu xoá?~~ → 100 mục/user, 1 ảnh/mục, `full_name` ≤ 100, `note` ≤ 1000, ngày mất ≥ 1900 và không ở tương lai, xoá cứng, 404 đồng nhất cho mục của người khác. Resolved 2026-10-06 (khoảng nhắc 0–30 superseded v4.0 bởi AC9).
