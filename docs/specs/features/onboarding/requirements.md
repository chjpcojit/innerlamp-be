# Requirements — onboarding

Version: 6.1
Last updated: 2026-09-25
Status: implemented (US-001) — đăng ký 2 bước (request-otp → resend-otp → verify-otp) đã chạy thật end-to-end qua Postgres; AC1–AC15 đã có test (usecase, repository trên DB thật, handler, E2E). US-002 đã chuyển sang `login/`, US-003 sang `password-reset/`; **US-004 (hồ sơ) còn ở mức user story, chưa design/implement** (xem section cuối file).
Module: `identity`
Code: `services/appService/`
Feature folder: `services/appService/` + `docs/specs/features/onboarding/`

> See `changelog.md` in this folder for the version history of this spec.

Spec này viết **retroactive** từ feature đã implement (US-001). Nội dung còn lại của US-002/US-003 là
con trỏ chéo sang folder riêng (`login/`, `password-reset/`); US-004 giữ nguyên ở mức user story.
(v6.1: chuẩn hoá theo quy ước `il-spec-writer` hiện hành — viết lại AC sang EARS, thêm bảng API
contract, thêm `flow.md`, đồng bộ version 4 file; **không đổi behavior, không đổi AC, không đổi code**.
v6.0: US-003 chuyển sang `password-reset/`; v4.0: US-002 chuyển sang `login/`.)

---

## US-001: Đăng ký tài khoản

**Module:** `identity`
**Priority:** P0 (bắt buộc, chặn mọi tính năng cần user context)

### User Story

> Là một **người dùng mới**,
> Tôi muốn **đăng ký tài khoản bằng số điện thoại (xác thực OTP qua SMS), email và mật khẩu**,
> Để **có thể lưu và đồng bộ hành trình tu tập của mình qua nhiều thiết bị**.

### Các trường thông tin

| Trường | Bắt buộc? | Ghi chú |
|---|---|---|
| Số điện thoại | Bắt buộc | Chuẩn hóa định dạng E.164 (`+84...`), unique. **Phải verify OTP qua SMS** — đây là định danh chính ở MVP. Đã verify thì lưu thẳng vào `users.phone_number` |
| Mật khẩu | Bắt buộc | ≥ 8 ký tự |
| Email | Bắt buộc nhập | Unique toàn hệ thống. Client gửi lên ở bước 1, lưu tạm ở `pending_registration`; sau khi verify OTP mới copy sang `users.email`. Vì chưa verify nên `users.email_verified_at` = `NULL` |
| Pháp danh (dharma name) | Tùy chọn | Tên đạo, nếu đã quy y |
| Nơi quy y / xuất gia | Tùy chọn | Tên chùa/tổ đình nơi quy y hoặc xuất gia |

> Số điện thoại **bắt buộc**; email **bắt buộc nhập**. Thiết kế bảng: `pending_registration` có cột
> `email`; bảng `users` cũng có cột `email` nhưng thêm cột `email_verified_at`. Client gửi email lên
> ở bước 1, hệ thống lưu tạm vào `pending_registration`; **sau khi verify OTP xong** mới copy sang
> `users.email`. Tại thời điểm copy email vẫn **chưa được verify**, nên `users.email_verified_at` để
> `NULL`. Không có bảng `pending_email` thường trú — email chưa verify nằm chung trong
> `pending_registration` với phần payload còn lại. Verify email (gửi mail xác thực + set
> `email_verified_at`) **không nằm trong phạm vi US-001** (xem [Out of scope](#out-of-scope-giai-đoạn-mvp)).

### Luồng đăng ký (2 bước)

Thứ tự: **verify OTP trước, tạo account sau** — không có tài khoản nào ở trạng thái "chưa verify" tồn
tại trong `users`.

1. Người dùng nhập số điện thoại + email + mật khẩu (+ pháp danh / nơi quy y nếu có) → hệ thống lưu
   tạm toàn bộ payload vào bảng `pending_registration`, **gửi OTP qua SMS**, chưa ghi gì vào `users`.
2. Người dùng nhập OTP → hệ thống xác thực, **rồi mới** tạo hàng `users` (lấy dữ liệu từ
   `pending_registration`) và cấp token; hàng tạm bị xoá sau khi tạo xong.

Dữ liệu đăng ký tạm (kể cả password **đã hash**) giữ ở **server**, trong bảng `pending_registration` —
không bắt client giữ plaintext password rồi gửi lại ở bước 2. Hệ quả: request bước 2 chỉ cần
`phone_number` + `otp_code`, và password không đi qua mạng lần thứ hai.

### Acceptance Criteria (EARS notation)

Cả **ba** route đều **public** (không `RequireAuth`) — người gọi chưa có tài khoản, nên không có gì để
xác thực. Success/failure code dưới đây là **nội dung `data`/`error` của envelope** `utils.ApiResponse`
(`{success, data}` / `{success:false, error:{code,message,type}, details}`).

**Bước 1 — `POST /v1/auth/register/request-otp`**

1. WHEN a `POST /v1/auth/register/request-otp` request arrives with a valid (E.164-normalised) `phone_number`, a valid `email`, and a `password` of at least 8 characters, THE SYSTEM SHALL store the registration payload — including the **bcrypt-hashed** password — in `pending_registration`, issue an OTP via SMS, and SHALL NOT create any `users` row yet. (AC1 cũ.)
2. WHEN a `request-otp` request arrives with a `password` shorter than 8 characters, THE SYSTEM SHALL reject it with `400 INVALID_PASSWORD` (details `{field: "password"}`) and SHALL NOT write any row. (AC7 cũ — client hiển thị lỗi ngay ở form trước khi gửi; server vẫn validate lại, không tin client.)
3. WHEN a `request-otp` request arrives with a malformed `phone_number` (even after E.164 normalisation) or a malformed `email`, THE SYSTEM SHALL reject it with `400 INVALID_PHONE` / `400 INVALID_EMAIL` (details `{field}`) and SHALL NOT write any row. (AC8 cũ.)
4. WHEN a `request-otp` request arrives with an empty `dharma_name` and/or `ordination_place`, THE SYSTEM SHALL still accept it — these are optional; an empty value SHALL be stored as SQL `NULL`. (AC9 cũ.)
5. WHEN a `request-otp` request arrives with a `phone_number` that already has a `users` row, THE SYSTEM SHALL reject it with `409 PHONE_TAKEN` and SHALL NOT create a duplicate. (AC6 cũ — SĐT chỉ bị coi là "đã dùng" khi có tài khoản; một lượt dở của **chính** số đó không chặn, xem AC15.)
6. WHEN a `request-otp` request arrives with an `email` that is already used — either by an existing `users.email` or by a **different** number's `pending_registration.email` (còn hạn) — THE SYSTEM SHALL reject it with `409 EMAIL_TAKEN` and SHALL NOT create a duplicate. The duplicate-email check SHALL query both sources, since an unverified email is not yet in `users`. (AC6 cũ.)
7. WHEN a `request-otp` request arrives within 60 seconds of the previous send for the same number, THE SYSTEM SHALL reject it with `429 OTP_RESEND_TOO_SOON` (with `retry_after_seconds`). (AC5 cũ.)

**Bước 2 — `POST /v1/auth/register/verify-otp`**

8. WHEN a `verify-otp` request arrives with a correct, unexpired (≤ 90s), unused OTP, THE SYSTEM SHALL create the `users` row with `phone_number` = the verified number, `email` = the submitted email, and `email_verified_at` = `NULL`, and SHALL return `201` with `access_token` + `refresh_token` (logging the user in immediately). An unverified email SHALL NOT be usable for login (xem AC13). (AC2 cũ.)
9. WHEN the account is created, THE SYSTEM SHALL copy the submitted (unverified) email into `users.email` with `email_verified_at` = `NULL`, and SHALL delete the `pending_registration` row **in the same transaction** — no temporary row SHALL be left behind. (AC3 cũ.)
10. WHEN a `verify-otp` request arrives with a wrong OTP, an expired OTP, or an already-used OTP, THE SYSTEM SHALL reject it (`400 OTP_INVALID` / `400 OTP_EXPIRED`) and SHALL NOT create a `users` row. (AC4 cũ.)
11. WHEN the 3rd wrong OTP is entered, THE SYSTEM SHALL lock OTP verification for that number for 30 minutes and invalidate the current OTP; a 4th attempt SHALL be rejected with `429 OTP_LOCKED` (with `locked_until`), forcing a restart from step 1 once the lock expires. (AC12 cũ.)
12. WHEN the account is created, THE SYSTEM SHALL store only the bcrypt hash of the password — never the plaintext. (AC10 cũ.)

**Gửi lại — `POST /v1/auth/resend-otp`**

13. WHEN a `resend-otp` request arrives for `purpose = "register"`, THE SYSTEM SHALL apply the same send policy as `request-otp` (60s cooldown; total 4 sends per registration = 1 initial + 3 counted; on the 4th send THE SYSTEM SHALL still send and lock sending for 30 minutes; sends 5+ SHALL be rejected with `429 OTP_SEND_LIMITED` + `locked_until`). After 30 minutes the counter SHALL reset. The send-lock SHALL NOT block entering the last valid OTP. (AC5 cũ.)
14. WHEN a `resend-otp` request arrives for a number whose `pending_registration` no longer exists or has expired, THE SYSTEM SHALL reject it with `400 OTP_INVALID` / `400 PENDING_REGISTRATION_EXPIRED`. The temporary payload SHALL live **24 hours**, and `resend-otp` SHALL issue a fresh OTP even if the previous OTP has expired, as long as the payload is still valid. (AC15 cũ.)

**Cross-cutting (gate email + phiên đăng nhập)**

15. WHEN an email with `email_verified_at IS NULL` is used to log in, THE SYSTEM SHALL reject it — an unverified email SHALL NOT be a valid login identity. (AC13 cũ; được thực thi bởi `login/` — US-002 chỉ hỗ trợ SĐT.)
16. WHEN any feature attempts to send mail to an email with `email_verified_at IS NULL`, THE SYSTEM SHALL refuse and direct the user to verify the email first — mail SHALL NOT be sent to an unverified address. (AC14 cũ; quy tắc gate áp lên các feature gửi thư — chưa có feature nào như vậy trong US-001.)
17. WHEN a user who just registered reopens the app, THE SYSTEM SHALL keep them signed in via the access token (valid 15 minutes). (AC11 cũ; vế "refresh token tự gia hạn" ngoài scope US-001 — theo dõi ở `login/` AC9–AC11.)

### API contract (confirmed surface for this feature)

Cả ba path đã có trong `docs/specs/api/openapi.yaml` (v3.7) — viết **sau khi** handler chạy thật qua
`curl`, theo tiền lệ US-001. Body thành công là nội dung `data`; body lỗi là nội dung `error` (cùng
envelope `utils.ApiResponse`).

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| POST | `/v1/auth/register/request-otp` | public | `200` `{message, expires_in_seconds: 90, resend_after_seconds: 60}` | `400 INVALID_PHONE / INVALID_EMAIL / INVALID_PASSWORD`, `409 PHONE_TAKEN / EMAIL_TAKEN`, `429 OTP_LOCKED / OTP_RESEND_TOO_SOON / OTP_SEND_LIMITED`, `500` |
| POST | `/v1/auth/resend-otp` | public | `200` `{message, expires_in_seconds: 90, resend_after_seconds: 60, resend_remaining}` | `400 INVALID_PHONE / VALIDATION_ERROR / OTP_INVALID / PENDING_REGISTRATION_EXPIRED`, `429 OTP_LOCKED / OTP_RESEND_TOO_SOON / OTP_SEND_LIMITED`, `500` |
| POST | `/v1/auth/register/verify-otp` | public | `201` `{access_token, refresh_token, user}` | `400 VALIDATION_ERROR / OTP_INVALID / OTP_EXPIRED / PENDING_REGISTRATION_EXPIRED`, `409 EMAIL_TAKEN / PHONE_TAKEN`, `429 OTP_LOCKED`, `500` |

### Gate email chưa verify

Luật chung áp cho mọi tính năng hiện tại và sau này: **`users.email_verified_at IS NULL` ⇒ email đó
bị coi là chưa sở hữu hợp lệ** — không dùng được làm định danh đăng nhập, không dùng được làm đích
nhận thư, không dùng được để khôi phục tài khoản. Lý do: ở bước 2, email được copy sang `users` mà
chưa hề có bằng chứng người đăng ký thực sự sở hữu hộp thư đó.

Hệ quả trực tiếp lên các US khác:

| US | Ảnh hưởng |
|---|---|
| US-002 (đăng nhập — nay ở [`../login/`](../login/requirements.md)) | Đăng nhập bằng email chỉ hợp lệ khi `email_verified_at IS NOT NULL`. Định danh chính là số điện thoại. **Đã áp dụng:** login spec chỉ hỗ trợ SĐT + mật khẩu, email login ngoài scope. |
| US-003 (quên mật khẩu — nay ở [`../password-reset/`](../password-reset/requirements.md)) | **Đã áp dụng:** reset dùng **SMS OTP** (`purpose = 'reset_password'`), không dùng email — vì email chưa verify bị gate, không còn đường gửi mail nào ở US-003. |
| US-004 (hồ sơ) | Màn hình hồ sơ phải hiển thị rõ trạng thái email (chưa xác thực / đã xác thực) và có nút "Xác thực email". |
| Tính năng tương lai dùng email | Phải kiểm tra gate trước khi gửi thư hoặc dùng email làm định danh. |

Ghi chú: chính tính năng **xác thực email** (gửi mail xác thực + set `email_verified_at`) và luồng
**đổi email** nằm ngoài phạm vi US-001 — đây mới chỉ là luật gate.

### Out of scope (giai đoạn MVP)

- **Verify email sau khi đăng ký** — US-001 copy email từ `pending_registration` sang `users.email`
  nhưng để `email_verified_at` = `NULL`, chưa gửi mail xác thực và chưa có luồng set
  `email_verified_at`. Để lại user story riêng, làm ngay sau US này (cùng dịp với US-003 vì cả hai
  đều cần dịch vụ gửi email).
- Đăng ký qua social login (Google/Apple) — để lại cho giai đoạn sau.
- Quên mật khẩu / reset password — đã chuyển sang [`../password-reset/`](../password-reset/requirements.md) (US-003, dùng SMS OTP).
- `POST /v1/auth/refresh` + rotate refresh token — ngoài scope US-001; theo dõi ở [`../login/`](../login/requirements.md).

### Pending

Feature đã implement và verify xong — xem `tasks.md` Phase 1–10 (không còn item `[ ]`). Hai việc còn
lại **ngoài task list**, không chặn tính năng:

- **OTP hằng số `111111`** + **SMS log-only** (`pkg/sms/logsms`) — placeholder dev theo Ghi chú kỹ
  thuật; chặn production, không chặn task.
- **Audit AC bằng E2E tự động** (`golang-e2e-verification`) — harness `services/appService/e2e/` đã
  có từ `login/`, chưa audit đủ AC1–AC15 cho US-001.

### Ghi chú kỹ thuật liên quan

- Endpoint (2 bước, khớp [Luồng đăng ký](#luồng-đăng-ký-2-bước)):
  ```
  POST /v1/auth/register/request-otp   { phone_number, email, password, dharma_name?, ordination_place? }
  POST /v1/auth/register/verify-otp    { phone_number, otp_code }
  POST /v1/auth/resend-otp             { phone_number, purpose }   -- dùng chung mọi purpose; US-001 chỉ hiện thực purpose=register
  ```
  Bước 1 **chưa ghi gì** vào `users` — chỉ ghi `pending_registration` + `otp_codes` (purpose =
  `register`); bước 2 mới tạo hàng `users` + cấp token.
- Schema (đã có code, xem `migrations/0001_register_onboarding.sql` + `0003_otp_send_limits.sql`):
  ```sql
  -- email vẫn nằm thẳng ở users: unique + NOT NULL giữ nguyên như thiết kế cũ.
  -- Chỉ thêm cột đánh dấu trạng thái verify.
  ALTER TABLE users ADD COLUMN email_verified_at TIMESTAMPTZ;
  ALTER TABLE users ADD COLUMN phone_number VARCHAR(20) UNIQUE NOT NULL;
  ALTER TABLE users ADD COLUMN ordination_place VARCHAR(255);
  -- (dharma_name đã có sẵn từ thiết kế trước, giữ nguyên)

  -- OTP dùng chung cho mọi mục đích (register / login / reset_password / verify_email),
  -- phân biệt bằng cột purpose. Bước 1 ghi vào đây với purpose = 'register'.
  CREATE TABLE otp_codes (
      id           BIGSERIAL PRIMARY KEY,
      phone_number VARCHAR(20) NOT NULL,
      purpose      VARCHAR(32) NOT NULL,
      otp_hash     VARCHAR(255) NOT NULL,
      attempts     INT NOT NULL DEFAULT 0,
      locked_until TIMESTAMPTZ,
      expires_at   TIMESTAMPTZ NOT NULL,
      used_at      TIMESTAMPTZ,
      created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
  );
  CREATE INDEX idx_otp_codes_phone_purpose ON otp_codes(phone_number, purpose);
  ALTER TABLE otp_codes ADD CONSTRAINT chk_otp_codes_purpose
      CHECK (purpose IN ('register', 'login', 'reset_password', 'verify_email'));

  -- v5.0: rate limit theo SĐT cho việc GỬI OTP, tách khỏi hàng otp_codes.
  CREATE TABLE otp_send_limits (
      id           BIGSERIAL PRIMARY KEY,
      phone_number VARCHAR(20) NOT NULL,
      purpose      VARCHAR(32) NOT NULL,
      send_count   INT NOT NULL DEFAULT 0,
      last_sent_at TIMESTAMPTZ,
      locked_until TIMESTAMPTZ,
      created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
      updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
      UNIQUE (phone_number, purpose)
  );

  -- dữ liệu đăng ký tạm giữa bước 1 và bước 2 (giữ ở server)
  CREATE TABLE pending_registration (
      id               BIGSERIAL PRIMARY KEY,
      phone_number     VARCHAR(20) NOT NULL UNIQUE,
      email            VARCHAR(255) NOT NULL,
      password_hash    VARCHAR(255) NOT NULL,
      dharma_name      VARCHAR(255),
      ordination_place VARCHAR(255),
      expires_at       TIMESTAMPTZ NOT NULL,  -- now() + PendingRegistrationTTL (24h)
      created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
  );
  CREATE INDEX idx_pending_registration_phone ON pending_registration(phone_number);
  ```
- **Không có bảng `pending_email` riêng.** Email chưa verify nằm chung trong `pending_registration`
  cùng phần payload còn lại, rồi mới copy sang `users.email` khi verify OTP xong. Trạng thái "chưa
  verify" ở `users` chỉ là `email_verified_at IS NULL`.
- **Email xuất hiện ở hai nơi trong khoảng giữa 2 bước**, nên check trùng phải quét cả hai:
  `users.email` (tài khoản đã tạo — unique constraint giữ nguyên, là chốt chặn cuối) và
  `pending_registration.email` (lượt đăng ký đang chờ verify).
- **Bảng `pending_registration`:** giữ tạm toàn bộ payload đăng ký ở **server** trong khoảng giữa bước
  1 và bước 2. Ở bước 2, hệ thống copy **cả `email` lẫn `password_hash`** sang `users` — email ở đây
  vẫn **chưa verify**, nên `users.email_verified_at` để `NULL`. Password đã **hash** sẵn từ bước 1,
  bước 2 chỉ copy chứ không hash lại. Hàng tạm bị xoá ngay sau khi tạo `users` thành công; hàng hết
  hạn (`expires_at`) dọn định kỳ cùng chỗ với `otp_codes`.
  - Lưu ý: `pending_registration.email` **không** đặt unique — trùng email phải được phát hiện bằng
    check tường minh khi tạo `users` (và qua unique constraint của `users.email`).
  - **TTL của payload tạm = 24 giờ** (v5.1; trước là 90 giây). Payload phải sống lâu hơn khoá-gửi 30
    phút thì người dùng mới "gửi lại / đăng ký lại" được sau khi khoá hết (AC15). `expires_at` là chốt
    chặn cuối.
- OTP lưu dạng **hash** (bcrypt — OTP chỉ 6 chữ số, không gian khoá nhỏ nên không dùng SHA-256),
  không plaintext.
- **Bảng `otp_codes` dùng chung, phân biệt bằng `purpose`.** Cột `purpose` cho biết OTP phục vụ luồng
  nào: `register` (US-001), `login` (đăng nhập bằng SĐT + OTP — **ngoài scope** của US-002 ở `login/`),
  `reset_password` (khôi phục mật khẩu qua SMS — ở [`../password-reset/`](../password-reset/requirements.md)),
  `verify_email` (xác thực email sau đăng ký). US-001 hiện chỉ dùng `purpose = 'register'`; các giá trị
  khác đã có trong `CHECK` constraint sẵn.
- **Tham số OTP:** TTL 90 giây; sau lần gửi đầu tiên, chặn gửi trong 60 giây; qua 60 giây, mỗi lần
  yêu cầu OTP mới **hoặc** gửi lại tính một lượt, tổng **4 lượt gửi** cho một lượt đăng ký (1 lần đầu
  + 3 lượt tính); khi thực hiện lượt thứ 4 thì **khóa gửi 30 phút** cho số điện thoại đó (khoá-gửi
  không chặn việc nhập OTP); nhập sai tối đa 3 lần thì khóa xác thực cho số điện thoại đó **30 phút**.
  Payload tạm `pending_registration` sống **24 giờ**. Các con số này đến từ `config.OTPConfig`
  (`services/appService/config/`), không hardcode trong usecase.
- **SMS gateway chưa có provider thật** ở giai đoạn này: `pkg/sms` theo pattern provider-agnostic
  interface + `Register`/`New` factory, trong đó `pkg/sms/logsms` là implementation **chỉ ghi log**
  nội dung SMS (không gửi thật). Provider thật (Twilio / AWS SNS / GCP tương đương) thêm sau bằng một
  subpackage `init()` tự đăng ký — không phải đổi interface.
- **OTP giai đoạn này là hằng số `111111`** (6 chữ số) để dev/test, chưa ngẫu nhiên. Khi chạy thật đổi
  sang sinh ngẫu nhiên bằng `crypto/rand` — tách riêng chỗ sinh OTP (`generateOTP()`) để chỉ đổi một
  điểm.
- Access token JWT sống 15 phút, refresh token sống **7 ngày** (`JWTConfig.RefreshTokenExpiration` =
  `168h`), rotate mỗi lần dùng.
- **Phụ thuộc chéo:** US-002 đã được viết lại và chuyển sang
  [`docs/specs/features/login/requirements.md`](../login/requirements.md): đăng nhập bằng **số điện
  thoại + mật khẩu**, kèm `refresh` và `logout`. Login bằng email và login bằng OTP để ngoài scope.

### Open questions (resolved during implementation)

- ~~Trần resend cũ chặn vĩnh viễn — xử lý thế nào?~~ → **Bỏ carry-forward, thay bằng `otp_send_limits` + khoá-gửi 30 phút** (v5.0). Resolved 2026-09-22.
- ~~Payload tạm sống bao lâu?~~ → **24 giờ**, tách khỏi OTP TTL 90 giây; `request-otp` ghi đè payload của chính SĐT, `resend-otp` giữ payload (v5.1). Resolved 2026-09-22.
- ~~Refresh token sống 7 hay 30 ngày?~~ → **7 ngày** (`168h`), khớp `config.dev.yaml`. Resolved 2026-09-22.
- ~~SMS lỗi thì trả lỗi hay log-rồi-trả-200?~~ → **500 `SMS_SEND_FAILED`** (không log-rồi-trả-200). Resolved 2026-09-17.
- ~~Chuẩn hoá số điện thoại đặt ở đâu?~~ → **`pkg/validation.NormalizePhone`**, E.164 + `+84` cho số local. Resolved 2026-09-17.

---

## US-002: Đăng nhập — đã chuyển sang `login/`

> US-002 (đăng nhập, kèm refresh token và logout) được đặc tả tại
> [`docs/specs/features/login/`](../login/requirements.md) từ v4.0 của thư mục này (2026-09-20). Không
> còn nội dung US-002 ở file này; các tham chiếu tới US-002 ở những chỗ khác trong file là tham chiếu
> chéo có chủ ý.

---

## US-003: Quên mật khẩu — đã chuyển sang `password-reset/`

> US-003 (quên mật khẩu, đặt lại bằng **SMS OTP**) được đặc tả tại
> [`docs/specs/features/password-reset/`](../password-reset/requirements.md) từ v6.0 của thư mục này
> (2026-09-25). Không còn nội dung US-003 ở file này; các tham chiếu tới US-003 ở những chỗ khác trong
> file là tham chiếu chéo có chủ ý.

---

## US-004: Cập nhật hồ sơ cá nhân

**Module:** `identity`
**Priority:** P1

> ⚠️ **US-004 chưa được design/implement** — còn ở mức user story (prospective). Chuẩn hoá lần này
> (v6.1) chỉ phủ US-001; US-004 giữ nguyên Given/When/Then bên dưới, chưa chuyển sang EARS và chưa có
> `design.md`/`flow.md`/task tương ứng. Khi đặc tả sẽ tách theo tiền lệ US-002/US-003 hoặc giữ tại đây.

### User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **cập nhật pháp danh, nơi quy y/xuất gia và ảnh đại diện của mình**,
> Để **hồ sơ tu tập phản ánh đúng thông tin cá nhân, đặc biệt khi tôi mới quy y sau khi đã tạo tài khoản**.

### Acceptance Criteria

1. **Given** tôi đã đăng nhập, **when** tôi mở màn hình hồ sơ, **then** tôi thấy đầy đủ thông tin hiện có: email, số điện thoại, pháp danh, nơi quy y/xuất gia, ảnh đại diện (các trường trống hiển thị rõ là chưa cập nhật, không hiện "null" hay lỗi).
2. **Given** tôi đang ở màn hình hồ sơ, **when** tôi sửa pháp danh và/hoặc nơi quy y/xuất gia rồi lưu, **then** thông tin được cập nhật thành công và phản ánh ngay trên UI, không cần tải lại app.
3. **Given** tôi tải lên một ảnh đại diện mới, **when** tôi lưu, **then** ảnh cũ (nếu có) được thay thế, ảnh mới hiển thị ở mọi nơi có avatar trong app.
4. **Given** tôi cố sửa email hoặc số điện thoại từ màn hình hồ sơ, **when** tôi thao tác, **then** hệ thống **không cho sửa trực tiếp ở đây** — đổi email/số điện thoại cần luồng xác thực riêng (US riêng, chưa làm ở giai đoạn này) vì đây là thông tin định danh đăng nhập, đổi tùy tiện có rủi ro bảo mật.
5. **Given** tôi để trống pháp danh hoặc nơi quy y/xuất gia và lưu, **when** hệ thống xử lý, **then** trường đó được xóa (set về rỗng) — cho phép người dùng "gỡ bỏ" thông tin đã nhập trước đó nếu muốn.
6. **Given** tôi chưa đăng nhập (access token không hợp lệ hoặc hết hạn), **when** tôi cố gọi API cập nhật hồ sơ, **then** hệ thống trả về lỗi 401 và không cho phép thao tác.

### Out of scope (giai đoạn này)

- Đổi email hoặc số điện thoại — cần luồng xác thực lại (verify email mới/OTP) riêng, chưa làm ở giai đoạn này
- Đổi mật khẩu khi đã đăng nhập (khác với "quên mật khẩu" ở US-003) — cần user story riêng
- Xóa tài khoản (account deletion) — cần user story riêng, có thể liên quan yêu cầu pháp lý (GDPR-like) nếu mở rộng quốc tế sau này

### Ghi chú kỹ thuật liên quan

- Endpoint: `PATCH /v1/users/me` (đã có sẵn trong thiết kế API ban đầu), yêu cầu `RequireAuth` middleware
- Ảnh đại diện: upload lên Cloud Storage, lưu `avatar_url` trong bảng `users` — cần endpoint upload riêng (`POST /v1/users/me/avatar` hoặc dùng signed upload URL) vì `PATCH /v1/users/me` chỉ nên nhận JSON, không nhận file trực tiếp
- Thuộc `UserService` trong `usecase/` (tách khỏi `AuthService` vì đây là CRUD hồ sơ, không phải nghiệp vụ xác thực — theo naming convention đã thống nhất)
