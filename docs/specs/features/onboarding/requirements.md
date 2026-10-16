# Requirements — onboarding

Version: 8.2
Last updated: 2026-10-16
Status: implemented (v8.2) — đăng ký xác minh bằng email chạy thật (mailer `log` ở dev). Còn mở: giới hạn theo IP (AC22, login Phase 5), `erd.svg`, OpenAPI, OKF — xem `tasks.md`.
Module: `identity`
Code: `services/appService/`
Product spec: `docs/product/onboarding/product.md` @ v1.2
Contract: `docs/specs/api/openapi.yaml` (this repo) — cần cập nhật theo bảng API bên dưới (`tasks.md` Phase 9)
Implements: BR-ONB-04, BR-ONB-05, BR-ONB-06, BR-ONB-07, BR-ONB-08, BR-ONB-09, BR-ONB-10, FR-ONB-003, FR-ONB-004, FR-ONB-005, FR-ONB-006, FR-ONB-007, FR-ONB-008, FR-ONB-010, NFR-ONB-01, NFR-ONB-02, NFR-ONB-03
Architecture: ADR-0003 (Accepted 2026-10-14)

> See `changelog.md` in this folder for the version history of this spec.

v8.0 viết lại toàn bộ spec. Bản v7.5 (đăng ký bằng SĐT + OTP qua SMS) chỉ còn trong `changelog.md`
và git history. Luật nghiệp vụ nằm ở product spec; file này chỉ nói backend thực hiện chúng thế nào
và hợp đồng API ra sao. Cơ chế OTP dùng chung nằm ở [`../otp-mechanism/`](../otp-mechanism/requirements.md);
giới hạn tần suất cho mọi `/v1/auth/*` nằm ở [`../login/`](../login/requirements.md) (AC31–AC35).

Ngoài phạm vi backend của product spec: BR-ONB-01/02/03 (màn giới thiệu, chỉ app), BR-ONB-11
(điều khoản, chỉ client), FR-ONB-001/002/009 (chỉ client).

---

## US-001: Đăng ký tài khoản

**Priority:** P0

### User Story

> Là một **người dùng mới**,
> Tôi muốn **đăng ký bằng email, mật khẩu, số điện thoại và họ tên, rồi nhập mã gửi tới email**,
> Để **có tài khoản lưu hành trình tu tập trên mọi thiết bị mà không cần SMS**.

### Luồng (2 bước, giữ nguyên bất biến cũ)

1. Client gửi form → backend kiểm tra, lưu tạm vào `pending_registration` (khoá theo **email**),
   gửi mã 6 số tới email. Chưa có hàng `users` nào.
2. Client gửi email + mã → backend xác minh, **rồi mới** tạo `users` (`email_verified_at = now()`)
   và phiên đầu tiên trong **một transaction**, trả token.

Email đã có tài khoản: bước 1 trả **y hệt** email mới, nhưng thư gửi đi là thư báo "email này đã có
tài khoản" thay cho mã (BR-ONB-08, NFR-ONB-01). Không có mã nào dùng được cho email đó.

### Acceptance Criteria (EARS notation)

Cả bốn route đều public. Mã lỗi là nội dung `error.code` của envelope `utils.ApiResponse`.

**Bước 1 — `POST /v1/auth/register/request-otp`**

1. WHEN a request arrives with valid fields and an email that has no account, THE SYSTEM SHALL normalise the email (trim, lower-case), store the payload with the password hash in `pending_registration` keyed by that email (replacing an earlier pending row of the same email only when that row has no live code or the new payload is the same — AC24), issue a 6-digit code for purpose `register`, email it, and return `200` with `expires_in_seconds` (= `otp.ttl`) and `resend_after_seconds` (= `otp.send_cooldown`), creating no `users` row. (FR-ONB-003, BR-ONB-08)
2. WHEN a request arrives with valid fields and an email that already belongs to an account, THE SYSTEM SHALL return the same `200` body as AC1, apply the same send limits, store the pending row marked `account_exists`, issue a code that is never disclosed, and email the account-exists notice instead of the code. Response body, status and the work done before answering (password hash, code hash, one mail call) SHALL be the same as AC1. (BR-ONB-08, NFR-ONB-01)
3. WHEN `email` is not valid per `rules/email.json` after normalisation, THE SYSTEM SHALL reply `400 INVALID_EMAIL` (details `{field: "email"}`) and write nothing. (BR-ONB-05)
4. WHEN `phone_country` + `phone_number` are not valid per `rules/phone-number.json` (country not in the list, wrong digit count after dropping separators and one leading 0), THE SYSTEM SHALL reply `400 INVALID_PHONE` (details `{field: "phone_number"}`) and write nothing; WHEN valid, THE SYSTEM SHALL store the E.164 form the vector gives. (BR-ONB-04)
5. WHEN `password` has fewer than 8 or more than 64 characters after NFC normalisation, THE SYSTEM SHALL reply `400 INVALID_PASSWORD` (details `{field: "password"}`) and write nothing; characters are counted per `rules/password.json`, nothing is trimmed, and no composition rule applies. (BR-ONB-06)
6. WHEN `full_name` is empty after trimming or longer than `profile.full_name_max_chars` (300), or `dharma_name` is longer than `profile.dharma_name_max_chars` (255) after trimming, THE SYSTEM SHALL reply `400 INVALID_FULL_NAME` / `400 INVALID_DHARMA_NAME` and write nothing; an empty `dharma_name` SHALL be stored as `NULL`. Counting follows `rules/names.json`. (BR-ONB-07)
7. WHEN the normalised phone number already belongs to an account, THE SYSTEM SHALL reply `409 PHONE_TAKEN` and send nothing. A phone number used only by another pending registration SHALL NOT block the request. (FR-ONB-005, BR-ONB-04)
8. WHEN `ordination_temple_id` is non-empty and matches no temple, THE SYSTEM SHALL reply `400 INVALID_TEMPLE` and write nothing; empty or absent means "Cư sĩ tu tại gia" and is stored as `NULL`. (FR-ONB-004, BR-ONB-10)
9. WHEN the send policy refuses the send for `(email, register)` — within the cooldown, or past the cap — THE SYSTEM SHALL reply `429 OTP_RESEND_TOO_SOON` (+`retry_after_seconds`) / `429 OTP_SEND_LIMITED` (+`locked_until`) and write nothing, identically for AC1 and AC2 emails. (BR-ONB-09)
10. WHEN the mailer fails after the rows were committed, THE SYSTEM SHALL release the send slot and reply `500 EMAIL_SEND_FAILED`. (platform)

**Gửi lại — `POST /v1/auth/register/resend-otp`**

11. WHEN a request arrives with an email that has a live pending registration, THE SYSTEM SHALL issue a new code that invalidates the previous one, email it (or the notice again when the row is `account_exists`), and return `200` with `expires_in_seconds`, `resend_after_seconds` and `resend_remaining`, under the same send policy as AC9. (FR-ONB-008, BR-ONB-09)
12. WHEN the email has no pending registration or it is older than `otp.pending_registration_ttl` (24h), THE SYSTEM SHALL reply `400 PENDING_REGISTRATION_EXPIRED`. (FR-ONB-010)
13. WHEN sending is locked for `(email, register)`, THE SYSTEM SHALL still accept the last valid code at verify; the send lock SHALL NOT block verification. (BR-ONB-09)

**Bước 2 — `POST /v1/auth/register/verify-otp`**

14. WHEN a request arrives with the right, unexpired, unused code for a pending row that is not `account_exists`, THE SYSTEM SHALL in one transaction create the `users` row from the pending payload with `email_verified_at = now()`, create the first session, mark the code used and delete the pending row, then return `201` with `access_token`, `refresh_token`, `expires_in` and `user`. (FR-ONB-007, BR-ONB-08)
15. WHEN the code is wrong, THE SYSTEM SHALL reply `400 OTP_INVALID`; WHEN it is expired or already used, `400 OTP_EXPIRED`; WHEN the third wrong code for that code is entered, THE SYSTEM SHALL lock it for `otp.lock_duration` (30m) and any later attempt SHALL get `429 OTP_LOCKED` (+`locked_until`). Each attempt SHALL be counted atomically before the code is compared, so concurrent guesses cannot exceed `otp.max_attempts`. (FR-ONB-007, BR-ONB-09)
16. WHEN the pending row is `account_exists`, THE SYSTEM SHALL treat every code as wrong (AC15), so no account or session is ever created for that email. (BR-ONB-08, NFR-ONB-01)
17. WHEN the pending row has expired, THE SYSTEM SHALL reply `400 PENDING_REGISTRATION_EXPIRED`. (FR-ONB-010)
18. WHEN, at account creation, the phone number has meanwhile been taken by another account, THE SYSTEM SHALL roll back and reply `409 PHONE_TAKEN`; WHEN the chosen temple no longer exists, `400 INVALID_TEMPLE`. (FR-ONB-005, FR-ONB-004)

**Chung**

19. WHEN the account is created, THE SYSTEM SHALL store the password only as `bcrypt(base64(SHA-256(NFC(password))))`, so all 64 characters count whatever their byte length, and SHALL never log, return or email the password. (BR-ONB-06, NFR-ONB-03)
20. WHEN a code is generated outside the dev environment, THE SYSTEM SHALL draw it from `crypto/rand` and store only its bcrypt hash; the code SHALL appear only in the email body. (NFR-ONB-03; ADR-0003 rule 6)
21. WHEN the service starts with `otp.dev_fixed_code` set or `mailer.provider = log` and `app.environment` is not `dev`, THE SYSTEM SHALL refuse to start. (ADR-0003 rules 6, 7)
22. WHEN any route of this feature exceeds its per-IP budget or the daily email budget, THE SYSTEM SHALL reply `429 RATE_LIMITED` with `retry_after_seconds` (login AC31–AC35). (NFR-ONB-02)
23. WHEN a temple proposal is sent after the account exists, THE SYSTEM SHALL handle it through the existing `POST /v1/temple-suggestions` unchanged (street line optional, province and ward required codes). (FR-ONB-006, BR-ONB-10)
24. WHEN a pending row exists for the email and a code for `(email, register)` is still live (unused and unexpired, a locked code included) and the request's payload differs from the stored one — phone, full name, dharma name, ordination temple, or a password that does not verify against the stored hash — THE SYSTEM SHALL reply `409 REGISTRATION_IN_PROGRESS` with `retry_after_seconds` (seconds until that code expires, rounded up, at least 1), and SHALL NOT change the pending row, the codes or the send counters, consume a send slot, or send any email. WHEN the payload is the same, or no live code exists, AC1/AC2 apply as before. The check and the write SHALL happen in one transaction under a lock on the email, and SHALL give the same answer for an email with and without an account. (BR-ONB-08, NFR-ONB-03; security review 2026-10-16)

### API contract (confirmed surface for this feature)

Thay đổi tại chỗ (cùng path, đổi body), trừ `resend-otp` chuyển path. Body thành công là `data`.

| Method | Path | Request | Success | Failure codes |
|---|---|---|---|---|
| POST | `/v1/auth/register/request-otp` | `{ email, password, phone_country, phone_number, full_name, dharma_name?, ordination_temple_id? }` | `200` `{ message, expires_in_seconds, resend_after_seconds }` | `400 VALIDATION_ERROR / INVALID_EMAIL / INVALID_PHONE / INVALID_PASSWORD / INVALID_FULL_NAME / INVALID_DHARMA_NAME / INVALID_TEMPLE`, `409 PHONE_TAKEN / REGISTRATION_IN_PROGRESS` (+`retry_after_seconds`), `429 OTP_RESEND_TOO_SOON / OTP_SEND_LIMITED / OTP_LOCKED / RATE_LIMITED`, `500 EMAIL_SEND_FAILED` |
| POST | `/v1/auth/register/resend-otp` (thay `/v1/auth/resend-otp`) | `{ email }` | `200` `{ message, expires_in_seconds, resend_after_seconds, resend_remaining }` | `400 VALIDATION_ERROR / INVALID_EMAIL / PENDING_REGISTRATION_EXPIRED`, `429 OTP_RESEND_TOO_SOON / OTP_SEND_LIMITED / OTP_LOCKED / RATE_LIMITED`, `500 EMAIL_SEND_FAILED` |
| POST | `/v1/auth/register/verify-otp` | `{ email, otp_code }` | `201` `{ access_token, refresh_token, expires_in, user }` | `400 VALIDATION_ERROR / INVALID_EMAIL / OTP_INVALID / OTP_EXPIRED / PENDING_REGISTRATION_EXPIRED / INVALID_TEMPLE`, `409 PHONE_TAKEN`, `429 OTP_LOCKED / RATE_LIMITED` |

- `phone_country` là mã ISO 3166 trong danh sách 12 nước của BR-ONB-04 (`VN`, `US`, `JP`, `KR`, `TH`, `SG`, `MY`, `TW`, `AU`, `DE`, `FR`, `CA`); `phone_number` là số người dùng gõ. `user.phone_number` trả về dạng E.164.
- `user` giữ shape cũ: `{ id, email, phone_number, full_name, dharma_name?, ordination_temple_id?, created_at }`.
- `expires_in_seconds` mặc định 600 (cấu hình `otp.ttl`). Client hiển thị số backend trả, không tự đặt.
- Header `X-API-Key` cho web và `Cache-Control: no-store` ở `verify-otp` giữ như cũ (login AC16–AC18).

### Thay đổi so với code hiện tại (v7.5)

| Hiện tại | v8.0 |
|---|---|
| OTP gửi qua SMS (`pkg/sms`, `repository/sms_otp_sender.go`) | Gửi qua email (`domain.AuthMailer`); **xoá** `pkg/sms`, `pkg/sms/logsms`, `sms_otp_sender.go`, `config.SMSConfig` |
| `generateOTP()` trả `111111` | `crypto/rand`; `111111` chỉ khi `otp.dev_fixed_code` và `app.environment = dev` (AC21) |
| `otp.ttl: 90s` | `600s` |
| `pending_registration`, `otp_codes`, `otp_send_limits` khoá theo `phone_number` | Khoá theo `email` (migration `0037`) |
| `409 EMAIL_TAKEN` | **Bỏ**; email đã có tài khoản đi đường AC2 |
| `409 PHONE_TAKEN` cả khi số nằm ở lượt đăng ký dở của email khác | Chỉ khi số thuộc một tài khoản (AC7) |
| `users.email_verified_at` luôn `NULL` | `now()` lúc tạo tài khoản |
| SĐT: `validation.NormalizePhone` (mặc định +84, nhận mọi số hợp lệ) | `phone_country` + bảng 12 nước, đúng số chữ số (AC4) |
| Mật khẩu ≥ 8 byte, bcrypt thẳng | 8–64 ký tự sau NFC, bcrypt trên SHA-256 (AC5, AC19) |
| `POST /v1/auth/resend-otp {phone_number, purpose}` | `POST /v1/auth/register/resend-otp {email}` |
| Không giới hạn theo IP | `429 RATE_LIMITED` (AC22) |
| Đếm nhập sai: đọc rồi tăng | Tăng nguyên tử trước khi so mã (AC15) |

## Out of scope (this feature boundary, not tracked here)

- Đăng nhập, phiên, quên mật khẩu, giới hạn tần suất dùng chung: [`../login/`](../login/requirements.md).
- Đổi email / SĐT, xoá tài khoản, Google/Apple: để sau (product §1.2).
- Xét duyệt đề xuất chùa: `temple-suggestion`.
- Lời mời gia phả khớp theo SĐT chưa xác minh: giữ nguyên, rủi ro đã chấp nhận (ADR-0003 rule 4).
- Chọn nhà cung cấp email thật: chờ có domain gửi thư (xem `design.md` "Mailer").

## Pending (not yet implemented)

- AC22 (giới hạn theo IP + ngân sách thư mỗi ngày): chưa gắn, làm ở login Phase 5.
- E2E, OpenAPI, OKF: `tasks.md` Phase 8–9.

`request-otp` / `resend-otp` trả `429 OTP_LOCKED` khi mã của email đang bị khoá do nhập sai: phát mã mới không được mở khoá 3 lượt đoán (thêm ở v8.1).

## Ghi chú kỹ thuật liên quan

- **Không lộ email đã đăng ký (AC2, AC16):** cả hai nhánh làm cùng việc (bcrypt mật khẩu, bcrypt mã,
  ghi `pending_registration` + `otp_codes`, một lần gọi mailer) nên body, status, giới hạn gửi và thời
  gian đều giống nhau. `resend-otp` cũng không lộ vì nhánh AC2 cũng có hàng pending. Ngoại lệ đã chấp
  nhận: `409 PHONE_TAKEN` (BR-ONB-04).
- **SĐT duy nhất nhưng không xác minh:** chỉ `users.phone_number` UNIQUE. Hai lượt đăng ký dở có thể
  cùng số; ai xác minh trước giữ số, người sau nhận `409 PHONE_TAKEN` ở bước 2 (AC18).
- **Không backfill** (không có người dùng thật, ADR-0003). Migration `0037` xoá dữ liệu tạm trong các
  bảng OTP/pending/khoá. Tài khoản dev tạo trước `0037` có `email_verified_at = NULL` và mật khẩu theo
  cách hash cũ nên không đăng nhập được: đăng ký lại.
- **Email lưu chữ thường** + `CHECK (email = lower(email))` để UNIQUE có nghĩa không phân biệt hoa thường.
- **Mailer đang mock** (ADR-0003 rule 7): dev dùng `LogMailer`, log cả mã. Production cần domain gửi
  thư (SPF/DKIM/DMARC) và driver thật trước khi bật.

## Open questions (resolved during implementation)

- ~~Xác minh tài khoản thế nào khi bỏ SMS?~~ → Mã 6 số qua email; SĐT bắt buộc, duy nhất, không xác minh (ADR-0003). Resolved 2026-10-14.
- ~~Thời hạn mã?~~ → 10 phút, cấu hình `otp.ttl` (anh Đức). Resolved 2026-10-14.
- ~~Địa chỉ đường của đề xuất chùa?~~ → Tuỳ chọn (product v1.2; backend vốn đã tuỳ chọn). Resolved 2026-10-14.
- ~~Cách kỹ thuật (đề xuất của Claude, anh Đức đồng ý 2026-10-14, duyệt chi tiết qua PR):~~ → sửa contract tại chỗ; migration `0037` không backfill; đếm giới hạn bằng Postgres, không Redis; mailer có interface + `LogMailer` tới khi có domain; hash SHA-256 trước bcrypt; mã cố định chỉ ở dev và chặn khởi động ngoài dev; bảng 12 nước nằm trong code (`pkg/validation`), chạy bằng vector. Resolved 2026-10-14.
