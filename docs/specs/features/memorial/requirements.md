# Requirements — memorial

Version: 2.4
Last updated: 2026-10-07
Status: v1 (US-MEM-01..04, 5 route CRUD) đã implement 2026-10-07; có test usecase, handler, repository (Postgres thật) và E2E (binary thật). **Ảnh (US-MEM-05) hoãn vì chưa làm được GCS** — nợ ở `docs/tech-debt/memorial/TD-001`. Còn 1 marker `⚠️ Not confirmed` (bucket GCS, chỉ chặn Phase 6). `openapi.yaml` và `erd.svg` đã cập nhật.
Module: `memorial`
Code: `services/appService/`

> See `changelog.md` in this folder for the version history of this spec.

Spec này được viết **prospectively** (trước khi có code; v1 đã được implement theo đúng spec ở v2.2) cho tính năng **Sổ giỗ chạp**: mỗi user đăng nhập giữ một danh sách người thân đã mất, mỗi mục có ngày mất, ghi chú, ảnh và cấu hình nhắc nhở. Dữ liệu **riêng tư theo user** — khác hẳn nhóm `temple`/`deity`/`article` (công khai, chỉ đọc).


## Quyết định nền tảng (đã chốt với anh Đức, 2026-10-06)

- **Ngày mất chỉ lưu dương lịch.** Server lưu `death_date` (dương, `YYYY-MM-DD`) kèm `death_calendar` (`solar` | `lunar`) ghi lại *người dùng đã nhập theo lịch nào*. Nếu nhập âm lịch thì **client tự quy đổi âm → dương trước khi gửi**, và khi hiển thị lại thì client tự đổi dương → âm. Memorial **không** dùng lịch âm ở server và không tính ngày giỗ hằng năm. *(Ghi chú 2026-10-11: server đã có bộ chuyển đổi âm ↔ dương cho sự kiện lịch — `domain/lunarcal`, US-CAL-002 — nhưng memorial chưa dùng nó; ngày giỗ vẫn do client tính.)* **Năm mất là tuỳ chọn**: khi không biết năm thì không thể đổi âm → dương, nên server lưu *ngày + tháng (+ cờ tháng nhuận) theo đúng lịch đã nhập* thay cho ngày dương đầy đủ (AC6a); client hiểu ngày/tháng đó ở mọi năm.
- **Nhắc nhở là local-only.** Server chỉ lưu cấu hình nhắc (`reminder`, `add_to_os_reminder`); app trên máy tự đặt thông báo local và (nếu cờ bật) ghi vào Reminders/Calendar của OS. Không có push FCM/APNs, không có job quét hằng ngày ở backend.
- **Ảnh: mỗi mục tối đa 1 ảnh, tuỳ chọn.** Thiết kế đã chốt là upload multipart qua BE → GCS (bucket riêng tư, signed URL ngắn hạn), nhưng **tạm hoãn** vì chưa làm được GCS (2026-10-07): US-MEM-05 và các AC liên quan nằm ngoài v1, `photo_url` luôn trả `null` cho tới khi làm Phase 6. Ảnh đến từ cả iPhone lẫn Android; app tự đổi HEIC/HEIF → JPEG trước khi upload (xem Open questions).
- Mã user story `US-MEM-01..05` do spec tự chọn (theo tiền lệ `US-ART-01`), đã được anh Đức giao cho spec quyết.

---

## US-MEM-01: Thêm người vào sổ giỗ

## User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **thêm một người thân đã mất vào sổ giỗ của mình kèm ngày mất và cấu hình nhắc**,
> Để **không quên ngày giỗ chạp hằng năm**.

## Acceptance Criteria (EARS notation)

1. WHEN a `POST /v1/memorials` request arrives with a valid access token and a valid body, THE SYSTEM SHALL create the memorial owned by the token's user and return `201 Created` with the full `Memorial` (shape ở "API contract"). Ảnh, ghi chú, nhắc nhở và năm mất đều tuỳ chọn; mục mới luôn tạo **không có ảnh** (ảnh gắn sau qua route ảnh, US-MEM-05).
2. WHEN the request carries no valid access token (thiếu, hết hạn, sai loại token), THE SYSTEM SHALL return `401` and SHALL NOT create anything.
3. WHEN `full_name` is empty/blank after trim or longer than 100 characters, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "full_name"`.
4. WHEN `relationship` is not one of `GRANDFATHER_PATERNAL` (ông nội), `GRANDFATHER_MATERNAL` (ông ngoại), `GRANDMOTHER_PATERNAL` (bà nội), `GRANDMOTHER_MATERNAL` (bà ngoại), `FATHER` (cha), `MOTHER` (mẹ), `SPOUSE` (vợ/chồng), `SIBLING` (anh chị em), `CHILD` (con), `OTHER` (khác), THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "relationship"`.
5. WHEN `death_calendar` is not `solar` or `lunar`, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "death_calendar"`.
6. WHEN the request carries a full `death_date` (năm đã biết), THE SYSTEM SHALL require it to be a valid `YYYY-MM-DD` solar date not earlier than `1900-01-01` and not later than today in `Asia/Ho_Chi_Minh`, and SHALL require `death_day`, `death_month`, `death_leap_month` to be absent; otherwise it SHALL return `400 VALIDATION_ERROR` with `details.field = "death_date"`. (Áp dụng cho cả ngày đã được client quy đổi từ âm lịch — server chỉ thấy ngày dương.)
6a. WHEN the request has `death_date` absent or `null` (năm không rõ), THE SYSTEM SHALL require `death_day` and `death_month` and accept an optional `death_leap_month` (mặc định `false`), validated by `death_calendar`: **solar** → `death_month` 1..12, `death_day` 1..số ngày của tháng đó (tháng 2 cho phép 29), `death_leap_month` phải `false`; **lunar** → `death_month` 1..12, `death_day` 1..30. Otherwise it SHALL return `400 VALIDATION_ERROR` with `details.field` = `death_day` / `death_month` / `death_leap_month`. (Server không biết tháng âm có 29 hay 30 ngày — không kiểm sâu hơn.)
6b. WHEN the request has neither a `death_date` nor both `death_day` and `death_month`, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "death_date"` — một mục luôn có ít nhất ngày và tháng mất.
7. WHEN `note` is longer than 1000 characters, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "note"`; WHEN `note` is absent, `null` or blank, THE SYSTEM SHALL store `NULL` and return `note: null`.
8. WHEN `reminder` is `null` or absent, THE SYSTEM SHALL store no reminder and return `reminder: null`.
9. WHEN `reminder` is present, THE SYSTEM SHALL require `days_before` to be an integer in `0..30`, `time` to be `HH:mm` (24h, `00:00..23:59`) and `timezone` (optional, mặc định `Asia/Ho_Chi_Minh`) to be a valid IANA zone; otherwise it SHALL return `400 VALIDATION_ERROR` with `details.field` = `reminder.days_before` / `reminder.time` / `reminder.timezone`.
10. WHEN `add_to_os_reminder` is `true` AND `reminder` is `null`, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "add_to_os_reminder"` (không có gì để ghi vào OS). WHEN it is absent, THE SYSTEM SHALL treat it as `false`.
11. WHEN the user already owns 100 memorials, THE SYSTEM SHALL return `409 MEMORIAL_LIMIT_REACHED` and SHALL NOT insert — kể cả khi hai request tạo chạy đồng thời (không thể vượt 100).
12. WHEN the body has unknown fields, is empty, or is not valid JSON, THE SYSTEM SHALL return `400 VALIDATION_ERROR` (cùng cơ chế `DecodeJSONBodyStrict` như các route khác). WHEN the body sets `photo_url`, `id`, `user_id` or any server-owned field, THE SYSTEM SHALL reject it as an unknown field.

## US-MEM-02: Xem sổ giỗ của mình

## User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **xem danh sách và chi tiết các mục trong sổ giỗ của mình**,
> Để **biết ai đã ghi và cấu hình nhắc hiện tại**.

## Acceptance Criteria (EARS notation)

13. WHEN a `GET /v1/memorials` request arrives with a valid access token, THE SYSTEM SHALL return `200` with `data.items` holding **only** memorials owned by the token's user, ordered by `created_at` DESC then `id` DESC, and `data.total`. Không phân trang (tối đa 100 mục). WHEN the user has none, THE SYSTEM SHALL return `items: []` (không phải `null`).
14. WHEN a `GET /v1/memorials/{id}` request arrives AND the memorial exists AND belongs to the token's user, THE SYSTEM SHALL return `200` with the full `Memorial`.
15. WHEN `{id}` does not exist, is not a positive integer, OR belongs to another user, THE SYSTEM SHALL return the **same** `404 MEMORIAL_NOT_FOUND` for all three — không phân biệt "không tồn tại" với "của người khác" để không lộ sự tồn tại của dữ liệu.
16. WHEN a memorial has a photo, THE SYSTEM SHALL return `photo_url` as a signed GCS URL valid for 15 minutes; WHEN it has none, THE SYSTEM SHALL return JSON `null`. Client không được cache `photo_url` lâu hơn hạn của nó.
17. WHEN a memorial has a full death date, THE SYSTEM SHALL return `death_date` as a `"YYYY-MM-DD"` string in solar calendar (regardless of which calendar the user entered it in) with `death_day`, `death_month`, `death_leap_month` all `null`; WHEN the year is unknown, THE SYSTEM SHALL return `death_date: null` with `death_day`, `death_month`, `death_leap_month` (bool) in the stored `death_calendar`. `death_calendar` luôn có mặt.

## US-MEM-03: Sửa một mục trong sổ giỗ

## User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **sửa thông tin một người đã ghi trong sổ giỗ**,
> Để **chỉnh lại thông tin sai hoặc đổi cấu hình nhắc**.

## Acceptance Criteria (EARS notation)

18. WHEN a `PUT /v1/memorials/{id}` request arrives with a valid body AND the memorial belongs to the token's user, THE SYSTEM SHALL replace `full_name`, `relationship`, `death_date`, `death_calendar`, `note`, `reminder`, `add_to_os_reminder` with the request values (full replacement — mọi field bắt buộc gửi như khi tạo, trừ field tuỳ chọn), refresh `updated_at`, and return `200` with the full `Memorial`.
19. WHEN a `PUT` is processed, THE SYSTEM SHALL NOT change `id`, `user_id`, `created_at` or the photo; ảnh chỉ đổi qua route ảnh (US-MEM-05).
20. WHEN a `PUT` body fails any validation of AC3–AC10 and AC12, THE SYSTEM SHALL return the same `400 VALIDATION_ERROR` as create and SHALL leave the row unchanged.
21. WHEN `{id}` does not exist or belongs to another user, THE SYSTEM SHALL return `404 MEMORIAL_NOT_FOUND` (AC15) and SHALL NOT modify any row.

## US-MEM-04: Xoá một mục khỏi sổ giỗ

## User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **xoá một người khỏi sổ giỗ của mình**,
> Để **dọn mục ghi nhầm hoặc không còn cần**.

## Acceptance Criteria (EARS notation)

22. WHEN a `DELETE /v1/memorials/{id}` request arrives AND the memorial belongs to the token's user, THE SYSTEM SHALL delete the row (xoá cứng) and return `200` `{"success": true}` (không `data`, theo `api-response-envelope` AC11).
23. WHEN a memorial with a photo is deleted, THE SYSTEM SHALL also delete the GCS object **after** the row is gone, best-effort: WHEN the object delete fails, THE SYSTEM SHALL still return `200`, log the failure with the object key (không log tên/ghi chú), and leave the orphan object for a cleanup job (⚠️ chưa có — xem Pending).
24. WHEN `{id}` does not exist or belongs to another user — kể cả gọi lại lần hai sau khi đã xoá — THE SYSTEM SHALL return `404 MEMORIAL_NOT_FOUND` and SHALL NOT delete anything.

## US-MEM-05: Ảnh của người đã mất

> ⏸ **HOÃN — chưa thuộc v1.** Chặn bởi hạ tầng GCS. AC25–AC32, nhánh "có ảnh" của AC16 và các nhánh xoá object ở AC23 chưa được implement; cho tới Phase 6, `photo_url` luôn là `null` và hai route ảnh chưa tồn tại. Nội dung dưới đây giữ nguyên để làm việc ngay khi GCS sẵn sàng.

## User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **tải, thay hoặc gỡ ảnh của người trong sổ giỗ**,
> Để **thấy di ảnh khi mở sổ giỗ**.

## Acceptance Criteria (EARS notation)

25. WHEN a `PUT /v1/memorials/{id}/photo` request arrives as `multipart/form-data` with one file part `photo` AND the memorial belongs to the token's user, THE SYSTEM SHALL store the image in the private bucket, record its object key, and return `200` with the full `Memorial` (có `photo_url` mới).
26. WHEN the file is larger than 5 MB, THE SYSTEM SHALL return `413 PHOTO_TOO_LARGE` and SHALL NOT write to storage (giới hạn áp bằng `http.MaxBytesReader` trước khi đọc hết body).
27. WHEN the file's **detected** content type (sniff nội dung, không tin header `Content-Type` hay đuôi tên file) is not `image/jpeg`, `image/png` or `image/webp`, THE SYSTEM SHALL return `415 PHOTO_UNSUPPORTED_TYPE` and SHALL NOT write to storage.
28. WHEN the `photo` part is missing or empty, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "photo"`.
29. WHEN the memorial already has a photo, THE SYSTEM SHALL replace it: the new object is written and recorded first, the old object is deleted afterwards best-effort (AC23 áp dụng tương tự). Tên object **luôn mới** (`<uuid>`), không ghi đè object cũ.
30. WHEN `DELETE /v1/memorials/{id}/photo` arrives AND the memorial belongs to the token's user, THE SYSTEM SHALL clear the photo (row trước, object sau, best-effort) and return `200` with the full `Memorial` (`photo_url: null`); WHEN it had no photo, THE SYSTEM SHALL still return `200` (idempotent).
31. WHEN `{id}` does not exist or belongs to another user, THE SYSTEM SHALL return `404 MEMORIAL_NOT_FOUND` **before** reading the uploaded body into storage — người lạ không thể dùng route này để đẩy file lên bucket.
32. WHEN a storage call fails (ghi/ký URL), THE SYSTEM SHALL return `500 INTERNAL_ERROR` without leaking the cause, and the row's `photo_object_key` SHALL stay pointing at the previous object (hoặc `NULL`) — không bao giờ trỏ vào object chưa ghi được.

## API contract (confirmed surface for this feature)

Đã có trong `docs/specs/api/openapi.yaml` (tag Memorials, 5 route; hai route ảnh chưa có vì hoãn) và `web/docs/public/openapi.yaml`. Mọi route đều `RequireAuth` (`authMiddleware.AuthenticateFunc`, `RequireTokenType: "access"`); `user_id` luôn lấy từ token, **không bao giờ** từ path/body.

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| POST | `/v1/memorials` | RequireAuth | `201` `Memorial` | 400 `VALIDATION_ERROR`, 401, 409 `MEMORIAL_LIMIT_REACHED`, 405, 500 |
| GET | `/v1/memorials` | RequireAuth | `200` `{items: [Memorial], total}` | 401, 405, 500 |
| GET | `/v1/memorials/{id}` | RequireAuth | `200` `Memorial` | 401, 404 `MEMORIAL_NOT_FOUND`, 405, 500 |
| PUT | `/v1/memorials/{id}` | RequireAuth | `200` `Memorial` | 400 `VALIDATION_ERROR`, 401, 404, 405, 500 |
| DELETE | `/v1/memorials/{id}` | RequireAuth | `200` `{"success": true}` | 401, 404, 405, 500 |
| PUT | `/v1/memorials/{id}/photo` ⏸ hoãn | RequireAuth | `200` `Memorial` | 400 `VALIDATION_ERROR`, 401, 404, 413 `PHOTO_TOO_LARGE`, 415 `PHOTO_UNSUPPORTED_TYPE`, 500 |
| DELETE | `/v1/memorials/{id}/photo` ⏸ hoãn | RequireAuth | `200` `Memorial` | 401, 404, 500 |

`Memorial`:

| Field | Kiểu | Ghi chú |
|---|---|---|
| `id` | int | |
| `full_name` | string | ≤ 100 ký tự |
| `relationship` | enum | 10 giá trị ở AC4 |
| `death_date` | `"YYYY-MM-DD"` \| `null` | dương lịch; `null` khi năm không rõ |
| `death_calendar` | `"solar"` \| `"lunar"` | lịch user đã nhập — client dùng để hiển thị lại; khi năm không rõ thì là lịch của `death_day`/`death_month` |
| `death_day`, `death_month` | int \| `null` | chỉ có khi năm không rõ, theo `death_calendar` |
| `death_leap_month` | bool \| `null` | chỉ có khi năm không rõ (và `lunar` mới có thể `true`) |
| `note` | string \| `null` | ≤ 1000 ký tự |
| `photo_url` | string \| `null` | signed URL, hạn 15 phút; **luôn `null` cho tới khi làm US-MEM-05** |
| `reminder` | `{days_before: 0..30, time: "HH:mm", timezone: IANA}` \| `null` | |
| `add_to_os_reminder` | bool | **ý định** của user, không phải trạng thái thật trên máy |
| `created_at`, `updated_at` | RFC 3339 | |

Request tạo/sửa gửi cùng các field trên trừ `id`, `photo_url`, `created_at`, `updated_at`.

## Out of scope (this feature boundary, not tracked here)

- **Server tính ngày giỗ / ngày nhắc kế tiếp** và mọi thứ cần lịch âm ở BE (`next_occurrence`, lọc "giỗ trong 30 ngày tới", sắp theo ngày giỗ gần nhất). Hệ quả trực tiếp của quyết định "chỉ lưu dương" — client làm.
- **Push notification** (FCM/APNs), đăng ký device token, job nhắc hằng ngày.
- Chia sẻ sổ giỗ cho người khác (gia đình cùng xem/sửa) — chỉ chủ sở hữu.
- Nhiều ảnh cho một mục; xoá mềm/khôi phục; tìm kiếm; phân trang.
- Ghi vào Reminders/Calendar của OS — **việc của app** (EventKit / Calendar Provider), BE chỉ lưu cờ.
- Xoá tài khoản kéo theo dọn ảnh trên GCS (chưa có feature xoá tài khoản).

## Deferred (có theo dõi, không chặn v1)

- **US-MEM-05 — ảnh** (route `PUT/DELETE /v1/memorials/{id}/photo`, adapter GCS, signed URL): hoãn vì chưa làm được GCS. Theo dõi ở `tasks.md` Phase 6.
- **Dọn object GCS mồ côi** (xoá object thất bại, hoặc xoá tài khoản sau này): "tính sau" — chỉ có ý nghĩa khi đã có ảnh, nên đi cùng Phase 6.

## Pending (not yet implemented)

- Ảnh (US-MEM-05) — `tasks.md` Phase 6, nợ `docs/tech-debt/memorial/TD-001`.

> ⚠️ Not confirmed: bucket GCS cho ảnh sổ giỗ — `config.dev.yaml` chỉ có `innerlamp-avatars-dev`; dùng chung bucket (prefix `memorials/`) hay tạo bucket riêng, và service account Cloud Run có quyền ký signed URL (`iam.serviceAccounts.signBlob`) chưa — chặn Phase 6 (ảnh), **không** chặn v1.

## Ghi chú kỹ thuật liên quan

- **Vì sao chỉ lưu dương lịch là ổn.** Ngày mất là *một ngày trong quá khứ*, đã cố định; quy đổi nó âm ↔ dương mang tính xác định, nên lưu dương + cờ `death_calendar` **không mất thông tin** — client đổi lại ra đúng ngày âm đã nhập (kể cả tháng nhuận, vì quy đổi là một-một giữa ngày dương và `(ngày, tháng, nhuận)` âm). Rủi ro duy nhất là **thuật toán âm lịch ở client**: sai một lần là dương lưu sai vĩnh viễn và server không phát hiện được (chỉ kiểm khoảng 1900..hôm nay). Giảm rủi ro: hai app dùng cùng thuật toán VN (múi giờ UTC+7, **không** dùng lịch Trung Quốc của thư viện hệ điều hành) và có test chung các ngày biên (tháng nhuận, Tết).
- Go stdlib không có lịch âm; vì ngày giỗ hằng năm do client tính nên BE không phải kéo thư viện âm lịch. Nếu sau này cần push từ server thì phải thêm thư viện âm lịch VN ở BE và đó là một thay đổi scope lớn (Out of scope hiện tại).
- **Quyền sở hữu** áp ở tầng SQL, không chỉ ở usecase: mọi truy vấn một mục đều có `WHERE id = $1 AND user_id = $2` (nên người lạ không bao giờ đọc được dòng); không có cách nào tạo/sửa mục dưới `user_id` khác vì `user_id` chỉ lấy từ token. id là số tự tăng (dễ đoán) nên 404 đồng nhất ở AC15 là biện pháp chống dò id.
- **Dữ liệu cá nhân nhạy cảm** (tên người thân, ghi chú, ảnh): không log `full_name`/`note`; ảnh không public, chỉ signed URL 15 phút; nhớ khi viết `openapi.yaml` đánh dấu route cần Bearer.
- **Giới hạn 100 mục/user** nhằm chặn lạm dụng bucket/DB, kiểm trong transaction có khoá dòng `users` (xem `design.md`) để request song song không vượt trần.
- **Không có unique** trên (`user_id`, `full_name`, `death_date`) — hai mục trùng được phép (ví dụ cùng tên); chống bấm đúp là việc của client.
- Giờ nhắc `time` là giờ treo tường theo `timezone` của mục (mặc định `Asia/Ho_Chi_Minh`); vì nhắc là local nên app đặt thông báo theo múi giờ này.
- Danh sách quan hệ: đã xác nhận "bà ngoại, cha, mẹ" là ba giá trị riêng; `OTHER` chỉ là một giá trị enum, không có nhãn tự do.
- **Hệ quả của "năm tuỳ chọn" lên quyết định "chỉ lưu dương":** quyết định đó vẫn đúng mỗi khi có năm. Khi không có năm, ngày âm không thể đổi sang dương (cần năm), nên server giữ nguyên ngày/tháng/nhuận user nhập. Mục có năm mất dạng "ngày dương đầy đủ", mục không có năm mất dạng "ngày + tháng theo lịch đã nhập" — hai dạng loại trừ nhau, được ép bằng CHECK ở DB.
- Hình dạng gần nhất để theo: user-scoped write của `onboarding`/`login` (middleware JWT) kết hợp upload của `pkg/storage`; chưa có route `/v1/users/me` nào để bắt chước.

## Open questions (resolved during implementation)

- ~~Thứ tự danh sách `GET /v1/memorials`?~~ → **`created_at` DESC, `id` DESC** (mục mới thêm hiện trước), như AC13. Backend không biết ngày giỗ kế tiếp nên client tự sắp lại nếu muốn theo ngày giỗ. Resolved 2026-10-07.
- ~~Mã user story?~~ → Spec tự chọn `US-MEM-01..05`. Resolved 2026-10-07.
- ~~Tháng nhuận: quy ước giỗ các năm không có tháng nhuận đó?~~ → Theo đề xuất: dùng tháng thường cùng số. Quy ước nằm ở client (iOS/Android dùng cùng thuật toán lịch VN UTC+7); BE chỉ lưu `death_leap_month`. Resolved 2026-10-07.
- ~~Ảnh HEIC/HEIF từ iPhone?~~ → Ảnh đến từ iPhone và Android; **app đổi HEIC/HEIF → JPEG trước khi upload**, BE giữ nguyên chỉ nhận jpeg/png/webp (không kéo thư viện giải mã HEIC vào BE). Đây là cách hiểu của spec từ câu trả lời "ảnh từ iPhone và Android" — `docs/assumptions/memorial/A-001-heic-client-converts.md` (đã xác nhận 2026-10-07). Resolved 2026-10-07.
- ~~Cờ `add_to_os_reminder` trên nhiều thiết bị / nhiều OS?~~ → **Client tự xử lý** (mỗi máy tự biết đã tạo sự kiện OS chưa); BE chỉ lưu cờ, không lưu event id. Resolved 2026-10-07.
- ~~Ảnh bằng GCS ngay bây giờ?~~ → **Tạm bỏ qua** vì chưa làm được; US-MEM-05 chuyển sang Deferred. Resolved 2026-10-07.
- ~~Dọn object GCS mồ côi?~~ → **Tính sau**, đi cùng ảnh (Deferred). Resolved 2026-10-07.
- ~~Ngày mất chỉ biết ngày/tháng, không biết năm?~~ → **Năm là tuỳ chọn.** Thiếu năm thì lưu `death_day`/`death_month`/`death_leap_month` theo lịch đã nhập, `death_date = NULL` (AC6a, AC6b, AC17). Resolved 2026-10-07.
- ~~`relationship = OTHER` có cần nhãn tự do?~~ → **Không**, chỉ là giá trị "khác". Resolved 2026-10-07.
- ~~Danh sách quan hệ "bà ngoại cha, me" nghĩa là gì?~~ → Ba giá trị riêng: bà ngoại, cha, mẹ. Resolved 2026-10-07.
- ~~Ảnh có bắt buộc không?~~ → **Tuỳ chọn** (hiểu "option luôn" là ảnh); mục tạo mới không có ảnh. Resolved 2026-10-07.
- ~~Lưu âm lịch hay chỉ dương?~~ → **Chỉ lưu dương lịch** + cờ `death_calendar`; client quy đổi hai chiều; server không tính ngày giỗ. Resolved 2026-10-06.
- ~~Ai gửi nhắc nhở?~~ → **Local-only** trên app; BE lưu cấu hình, không push. Resolved 2026-10-06.
- ~~Upload ảnh theo cách nào?~~ → **Multipart qua BE → GCS**. Resolved 2026-10-06.
- ~~Giới hạn và kiểu xoá?~~ → Mặc định đề xuất: 100 mục/user, 1 ảnh/mục, `full_name` ≤ 100, `note` ≤ 1000, nhắc 0–30 ngày, ngày mất ≥ 1900 và không ở tương lai, xoá cứng, 404 đồng nhất cho mục của người khác. Resolved 2026-10-06.
