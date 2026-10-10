# Tasks — onboarding

Version: 7.3
Last updated: 2026-10-10
Status: US-001 Phase 1–10 xong (2026-09-22); **Phase 11 (v7.0, `ordination_temple_id`) xong (v7.2, 2026-10-10; `erd.svg` render lại ở v7.3).** **Phase 10 (v5.1)** tách `pending_registration_ttl` khỏi OTP TTL (24h) và mở lại đường làm lại (`request-otp` ghi đè payload của chính SĐT, `resend-otp` cấp OTP mới khi OTP cũ hết hạn). **Phase 9 (v5.0) đã viết code + test**: bỏ trần resend chặn vĩnh viễn, thay bằng `otp_send_limits` + khoá-gửi 30 phút, `request-otp` cũng tiêu lượt. Đã verify: `go test -short ./...` toàn repo xanh; `go test -count=1 ./services/appService/repository/...` + `./services/appService/e2e/...` xanh với Postgres/testcontainers thật; `check-architecture.sh` (ARCH_ERRORS=0) + `check_dependency_rule.sh` sạch. US-001 chạy thật end-to-end qua Postgres (xem Phase 6); job dọn hàng hết hạn chốt chấp nhận ở MVP (Deferred). Ngoài task list, chưa làm: audit AC bằng `golang-e2e-verification` cho US-001 (harness `services/appService/e2e/` đã có từ `login/`), và OTP `111111` + SMS log-only vẫn là placeholder dev theo Ghi chú kỹ thuật — chặn production, không chặn task. (v7.2: **Phase 11 implement xong**, tick item đã verify. v7.1: chốt 3 câu hỏi mở của Phase 11, không đổi task. v7.0: **thêm Phase 11** — thay `ordination_place` bằng `ordination_temple_id`; không tick/untick task cũ. v6.1: chuẩn hoá theo quy ước `il-spec-writer` hiện hành — thêm `flow.md` (bản vẽ Mermaid, **không phải việc implement** nên không thêm Phase/task), AC của `requirements.md` viết lại sang EARS; không tick/untick task nào, không đổi code. v5.2: sửa tài liệu cũ — `hashRefreshToken` nay là `domain.HashRefreshToken`; tên E2E Phase 8 đã đổi ở Phase 10; các item Phase 1–3 mô tả policy v4.5 được đánh dấu rõ; không đổi code/task. v5.1: **Phase 10 xong** — tách pending TTL khỏi OTP TTL (24h), request-otp ghi đè payload của chính SĐT, resend-otp cấp OTP mới khi OTP cũ hết hạn; xem changelog v5.1. v5.0: **Phase 9 xong** — bỏ trần resend chặn vĩnh viễn, thay bằng `otp_send_limits` + khoá-gửi 30 phút, `request-otp` cũng tiêu lượt; xem changelog v5.0. v4.5: thêm Phase 8 — fix pending hết hạn chặn đăng ký lại, xong; v4.4: tick `go test -short`, dọn hàng hết hạn → Deferred, cập nhật Status; v4.3: sửa mô tả envelope ở Phase 4 `errors.go`, không tick/untick; v4.2: **Phase 7 xong** — access token mang `sid`, verify thật (xem Phase 7); v4.1: thêm **Phase 7** — sửa `TokenIssuer` để access token mang `sid`, do `login/` yêu cầu; toàn bộ `[ ]`, không tick/untick task cũ; v4.0: chỉ đổi con trỏ US-002 → `login/`.)
Module: `identity` (spec folder `onboarding`)

> See `changelog.md` in this folder for the version history of this spec.

<!--
Mọi item dưới đây trace về một dòng trong bảng "Layers & components" của design.md hoặc một step
trong "Step-by-step Flow" của nó.

Item `[x]` = đã verify (code có thật + test tương ứng pass). Item `[ ]` = chưa làm, hoặc làm dở.
Checklist tick ngày 2026-09-17, verify bằng: `go build ./...` (sạch), `go vet ./...` (sạch),
`go test ./services/appService/...` (pass).

Không còn xung đột chưa chốt. Xung đột OTP policy vs. quy tắc config của skill đã giải quyết theo
hướng config (xem item Phase 2 về `otpPolicy`) — code đã refactor tương ứng.

design.md không còn marker `⚠️ Not confirmed` nào (v5.1 — nhánh reset bộ đếm gửi đã chốt,
§"OTP send-limit policy"); 5 câu hỏi mở ở v3.0 đã được chốt (xem changelog.md).
-->

## Phase 1 — Domain

- [x] `domain/user.go` — entity `User`: `ID`, `Email`, `EmailVerifiedAt *time.Time`, `PhoneNumber`, `PasswordHash`, `DharmaName *string`, `OrdinationPlace *string`, `CreatedAt`, `UpdatedAt`. Field thuần nghiệp vụ, không JSON tag.
- [x] `domain/pending_registration.go` — entity `PendingRegistration`: `PhoneNumber`, `Email`, `PasswordHash`, `DharmaName`, `OrdinationPlace`, `ExpiresAt`
- [x] `domain/otp.go` — entity `OTPCode` + type `OTPPurpose`. Khai **đủ bốn** hằng (`OTPPurposeRegister`, `OTPPurposeLogin`, `OTPPurposeResetPassword`, `OTPPurposeVerifyEmail`) để `chk_otp_codes_purpose` và endpoint `resend-otp` dùng chung được; US-001 chỉ **hiện thực** `register`. Không tạo entity `Device`/`Session` ở phạm vi US-001.
- [x] `domain/register.go` — input/output types `RequestRegisterOTPInput`, `ResendOTPInput`, `VerifyRegisterOTPInput`, `RegisterResult`; func type `TokenIssuer` (usecase truyền JWT minting xuống repository mà `domain/` không import `pkg/jwt`)
- [x] `domain/register.go` — interface `RegistrationRepository` (do `repository/` implement, khai ở `domain/`), gồm `ReissueOTP` cho flow resend
- [x] `domain/register.go` — port `OTPSender` interface **một method duy nhất**: `SendOTP(ctx, phoneNumber, otpCode string, purpose OTPPurpose) error`. Không tách `SendRegistrationOTP` riêng — flow 1 và flow resend dùng chung đường gửi; US-002/US-003 gọi lại đúng method này với `purpose` khác.
- [x] `domain/errors.go` — sentinel: `ErrPhoneTaken`, `ErrEmailTaken`, `ErrInvalidPhone`, `ErrInvalidEmail`, `ErrInvalidPassword`, `ErrOTPInvalid`, `ErrOTPExpired`, `ErrOTPLocked`, `ErrOTPResendTooSoon`, `ErrOTPResendExhausted`, `ErrPendingRegistrationNotFound`, `ErrPendingRegistrationExpired`, `ErrUnsupportedOTPPurpose`
      <!-- Ngoài danh sách trên, errors.go còn có `ErrSMSSendFailed` (AC: SMS lỗi → 500, chốt ở changelog v3.0 mục 4) và hai typed error `OTPLockedError` / `OTPResendTooSoonError` mang `locked_until` / `retry_after`. Bổ sung so với item gốc; Phase 4 phải map cả `ErrSMSSendFailed` → 500. -->
- [x] `domain/mock_registration_repository.go` — mock tay cho `RegistrationRepository`, dùng ở test usecase (không DB thật)
      <!-- Thêm cả `domain/mock_otp_sender.go` cho port `OTPSender` — cùng lý do, cùng package. -->

## Phase 2 — Usecase

- [x] `usecase/register_usecase.go` — `RequestOTP`: normalize email → normalize phone → validate (email, phone, password ≥ 8) → `FindConflicts` → bcrypt hash password → `SavePendingAndIssueOTP` (1 tx) → `OTPSender.SendOTP(..., OTPPurposeRegister)`
      <!-- design.md flow 1 Step 1-5. Thứ tự bắt buộc: gửi SMS là bước CUỐI, sau khi tx đã commit.
      Verify: TestRequestOTP_HappyPath — assert.Equal([]string{"save","send"}, order) pin đúng thứ tự này. -->
- [x] `usecase/register_usecase.go` — kiểm rate limit trước khi ghi: cooldown 60s kể từ hàng OTP mới nhất, trần `resend_count >= 3` (`ErrOTPResendExhausted`), và `locked_until > now()` (`ErrOTPLocked`). Cả ba nhánh đều KHÔNG ghi gì.
      <!-- Verify: TestRequestOTP_LockedNumber / _Cooldown / TestResendOTP_Blocked — mỗi nhánh assert AssertNotCalled("SavePendingAndIssueOTP"/"ReissueOTP") + AssertNotCalled("SendOTP"). -->
- [x] `usecase/register_usecase.go` — `ResendOTP`: normalize phone → validate `purpose` → `FindActiveOTP` (đọc cooldown / `resend_count` / `locked_until` từ chính hàng đó) → `ReissueOTP` (1 tx) → `OTPSender.SendOTP`
      <!-- design.md flow 3 Step 1-5. Dùng chung cho MỌI purpose; US-001 chỉ nhận purpose='register', các giá trị khác trả ErrUnsupportedOTPPurpose cho tới khi US-002/US-003 wire vào.
      Verify: TestResendOTP_HappyPath — order == ["reissue","send"], và AssertNotCalled("SavePendingAndIssueOTP") chứng minh payload không bị ghi lại. -->
- [x] `usecase/register_usecase.go` — `ResendOTP` khi `purpose != 'register'` trả `ErrUnsupportedOTPPurpose` → 400, KHÔNG chạm DB. Hằng `OTPPurpose` đã khai đủ bốn giá trị ở domain, nhưng usecase chỉ hiện thực nhánh `register`.
      <!-- Verify: TestResendOTP_UnsupportedPurpose — assert AssertNotCalled("FindPendingRegistration"). -->
- [x] `usecase/register_usecase.go` — `VerifyOTP`: `FindPendingRegistration` → `FindActiveOTP(purpose='register')` → check `locked_until` → bcrypt compare → sai thì `IncrementOTPAttempts` (+ `LockOTP` khi `attempts+1 >= 3`) → đúng thì `CreateUserAndSession` (1 tx)
      <!-- design.md flow 2 Step 1-6. AC12: khoá ở lần sai thứ 3, lần thử thứ 4 chặn ở Step 3
      Verify: TestVerifyOTP_HappyPath (đúng → CreateUserAndSession), _WrongOTP (sai → IncrementOTPAttempts), _Locked (attempts=3 → LockOTP rồi lần 4 chặn trước khi so hash). -->
- [x] `usecase/register_usecase.go` — OTP policy KHÔNG phải `const` trong usecase: `otpPolicy config.OTPConfig` là field của `RegisterUsecase`, nhận qua constructor. Sáu giá trị nằm ở `services/appService/config/config.go` (`OTPConfig`) + `config.dev.yaml` block `otp:` — `ttl` 90s, `resend_cooldown` 60s, `max_resend` 3, `max_attempts` 3, `lock_duration` 15m, `pending_registration_ttl` 90s.
      <!-- v5.0/v5.1 thay các key này: `resend_cooldown` → `send_cooldown` (60s); bỏ `max_resend`; thêm `max_send_attempts` (3) + `send_lock_duration` (30m); `lock_duration` 15m → 30m; `pending_registration_ttl` 90s → 24h. Xem Phase 9/10. -->
      <!-- design.md flow 1 Step 4 — đã chốt 90s. Resend gia hạn thêm 90s, xem flow 3 Step 4(c).
      Giải quyết xung đột cũ với `.claude/skills/golang-clean-arch-tdd/SKILL.md` §"Configuration":
      TTL/cooldown/cap/lock-duration là thứ dev cần rút ngắn và prod cần nới ra, nên thuộc config.
      Code đã refactor theo hướng đó.
      Verify: `register_usecase.go:84` field `otpPolicy config.OTPConfig`; đọc tại :184, :192,
      :261, :350, :351; `NewRegisterUsecase` nhận `otpPolicy config.OTPConfig` (:101); không còn
      const block cho sáu giá trị này.
      LƯU Ý: usecase import thẳng `services/appService/config` (:34) chứ không qua một struct
      `OTPPolicy` khai tại `usecase/` — xem design.md line 54. -->
- [x] `usecase/register_usecase.go` — `generateOTP()` tách riêng một hàm, hiện trả hằng số `"111111"`; đổi sang `crypto/rand` về sau chỉ sửa hàm này
- [x] `usecase/register_usecase.go` — `hashOTP()` / so khớp bằng **bcrypt** (OTP chỉ 6 chữ số, không gian khoá nhỏ nên không dùng SHA-256), cùng cost với password hash
      <!-- design.md flow 2 Step 4 — đã chốt bcrypt, không còn là câu hỏi mở
      Verify: TestRequestOTP_HappyPath assert OTP hash verify được bằng bcrypt.CompareHashAndPassword
      và khác plaintext; TestVerifyOTP_WrongOTP so sai trả ErrOTPInvalid. -->
- [x] `usecase/register_usecase.go` — gọi `validation.NormalizePhone` trước khi validate và trước khi lưu
      <!-- design.md flow 1 Step 1 — đã chốt: chuẩn hoá về E.164 (+84 cho số local), hàm sống ở `pkg/validation`, KHÔNG tự viết trong usecase. Giá trị lưu ở `pending_registration.phone_number` / `users.phone_number` là bản đã chuẩn hoá.
      Verify: TestRequestOTP_HappyPath nhận input raw "0901234567" và assert capturedPending.PhoneNumber == "+84901234567". -->
- [x] Unit tests `usecase/register_usecase_test.go` — table-driven, `testify/assert` + `testify/require`, mock `RegistrationRepository` + mock `OTPSender`, không DB thật
      <!-- 25 test func, chạy `go test ./services/appService/usecase/...` pass (1.765s). -->
- [x] Test phủ đủ nhánh lỗi của AC4/AC5/AC6/AC12: OTP sai, OTP hết hạn, OTP đã dùng, resend quá sớm, resend quá 3 lần, sai 3 lần → lock
      <!-- Verify: TestVerifyOTP_WrongOTP (OTP sai), TestVerifyOTP_NoActiveOTP (hết hạn/đã dùng — cùng map về ErrOTPExpired), TestResendOTP_Blocked (3 case: locked / cooldown / cap), TestVerifyOTP_Locked (sai 3 lần → lock). -->
- [x] Test khẳng định bước 1 **không** ghi gì vào `users` (AC1), và bước 2 xoá `pending_registration` cùng tx với `INSERT users` (AC3)
      <!-- CẢ HAI NỬA XONG 2026-09-17.
      AC1 (nửa usecase): TestRequestOTP_HappyPath assert AssertNotCalled("CreateUserAndSession") ✔
      AC3 (nửa repository, Phase 3 xong):
      repository/postgres_registration_repository_test.go
      TestRegistrationRepository_CreateUserAndSession/happy_path... assert
      FindPendingRegistration trả ErrPendingRegistrationNotFound ngay sau khi CreateUserAndSession
      COMMIT — chứng minh (c) DELETE pending_registration chạy cùng tx với (b) INSERT users.
      Subtest "token_issuer_failure_rolls_back_the_whole_transaction" chứng minh chiều ngược lại:
      issueTokens lỗi → ROLLBACK → pending_registration vẫn còn, users vẫn 0 hàng — không có trạng
      thái nửa vời. `go test ./services/appService/repository/...` chạy thật với Postgres (Docker,
      không lưu trong repo) pass. -->
- [x] Test `ResendOTP`: gửi lại thành công thì `resend_count` **+1** chứ không reset về 0; hàng OTP cũ bị `used_at`; gọi lần thứ 4 trả `ErrOTPResendExhausted`
      <!-- design.md flow 3 Step 3 — AC5. CẢ HAI NỬA XONG 2026-09-17.
      NỬA ĐẦU (usecase): TestResendOTP_HappyPath assert result.ResendRemaining == 1 với hàng vào có
      ResendCount=2, và AssertNotCalled("SavePendingAndIssueOTP") — tức payload không bị viết lại.
      TestResendOTP_Blocked case "resend cap reached" phủ ErrOTPResendExhausted ✔
      NỬA SAU (repository, Phase 3 xong):
      TestRegistrationRepository_ReissueOTP/invalidates_old_otp... assert hàng OTP cũ có used_at
      NOT NULL sau ReissueOTP, và issued.ResendCount == 2 (đọc trong tx, +1 từ hàng cũ có
      resend_count=1) — carry-forward, không reset về 0. -->

## Phase 3 — Repository

- [x] `migrations/0001_register_onboarding.sql` — đánh số lại từ `0001` vì `migrations/` hiện rỗng. Một file gồm:
      (a) `CREATE TABLE IF NOT EXISTS users` + `users.email_verified_at`;
      (b) `phone_number VARCHAR(20) UNIQUE NOT NULL`, `ordination_place VARCHAR(255)` — khai **inline trong `CREATE TABLE`**, không tách `ALTER TABLE` (xem ghi chú dưới);
      (c) `CREATE TABLE otp_codes` + `idx_otp_codes_phone_purpose` + `chk_otp_codes_purpose` (CHECK liệt kê **đủ bốn** purpose: `register`, `login`, `reset_password`, `verify_email`);
      (d) `CREATE TABLE pending_registration` + `idx_pending_registration_phone`;
      (e) `CREATE TABLE user_sessions`
      <!-- design.md "Known gaps": KHÔNG tạo `user_devices` — contract 2 bước không mang trường device
      nào, nên `user_sessions` cũng KHÔNG có `device_id`. US-002 sẽ thêm cột + FK bằng migration riêng. -->
      <!-- `user_sessions` lưu `refresh_token_hash` (SHA-256), KHÔNG lưu token thô — design.md flow 2 Step 6. -->
      <!-- `user_sessions.expires_at` = now() + 168h (7 ngày, đã chốt) — đọc từ JWTConfig.RefreshTokenExpiration, không hardcode. -->
      <!-- SỬA ITEM 2026-09-17: (b) gốc viết "ALTER TABLE users thêm phone_number, ordination_place",
      giả định có sẵn một bảng `users` từ feature trước. File thực tế dựng `users` trọn vẹn bằng
      `CREATE TABLE IF NOT EXISTS` — comment đầu file nói rõ đây là chủ ý ("the previous migrations
      assumed a users table shipped by an earlier feature, and that assumption is what this file
      removes"). Sửa text item cho khớp code, không sửa code.
      Verify: file tồn tại (7.9 KB) + soi từng mệnh đề (a)-(e) khớp. CHƯA apply lên Postgres thật —
      việc đó nằm ở Phase 6 item "chạy end-to-end". -->
- [x] `repository/postgres_registration_repository.go` — implement `RegistrationRepository`, gồm `FindConflicts` (MỘT query quét `users.phone_number` ∪ `users.email` ∪ `pending_registration.phone_number` ∪ `pending_registration.email`), `SavePendingAndIssueOTP`, `ReissueOTP`, `FindPendingRegistration`, `DeletePendingRegistration`, `FindActiveOTP`, `IncrementOTPAttempts`, `LockOTP`, `CreateUserAndSession`
      <!-- 2026-09-17: tất cả 9 method của domain.RegistrationRepository implement xong.
      FindActiveOTP có một tinh chỉnh so với câu SQL literal ở design.md flow 2 Step 3: điều kiện
      thật là `(locked_until > now() OR (used_at IS NULL AND expires_at > now()))`, không phải chỉ
      `used_at IS NULL` — bắt buộc để (a) một OTP đã hết hạn tự nhiên (chưa bị đánh used_at, vì
      không có cron dọn) không còn "active" (AC4), và (b) một hàng đang bị khoá vẫn được tìm thấy
      dù `expires_at` gốc (90s) đã qua từ lâu so với `locked_until` (15 phút), để verify-otp lần
      thứ 4 báo đúng OTP_LOCKED thay vì OTP_EXPIRED (AC12). Khớp với doc comment đã có sẵn ở
      `domain/otp.go`: "FindActiveOTP matches on UsedAt OR LockedUntil rather than UsedAt alone" —
      chỉ làm rõ thêm mốc `expires_at`/`locked_until > now()` mà comment đó chưa viết ra literal SQL.
      Test: TestRegistrationRepository_FindActiveOTP, 7 subtest phủ đủ các nhánh trên. -->
- [x] `SavePendingAndIssueOTP` — MỘT transaction: UPSERT `pending_registration` `ON CONFLICT (phone_number) DO UPDATE` → `UPDATE otp_codes SET used_at = now()` cho OTP cũ → `INSERT otp_codes` mới (`resend_count` carry-forward từ hàng mới nhất, không đếm lại từ 0)
      <!-- design.md flow 1 Step 4. AC5 = cơ chế invalidate-OTP-cũ nằm ở đây.
      Verify: TestRegistrationRepository_SavePendingAndIssueOTP, subtest "second call upserts
      payload, invalidates old otp, carries resend_count forward" — pass thật với Postgres. -->
- [x] `ReissueOTP` — MỘT transaction, KHÁC `SavePendingAndIssueOTP` ở hai điểm: KHÔNG UPSERT `pending_registration` (payload giữ nguyên), và **gia hạn `pending_registration.expires_at = now() + 90s`** (chỉ khi `purpose='register'`). Vẫn `UPDATE otp_codes SET used_at = now()` → `INSERT otp_codes` mới với `resend_count` đọc trong cùng tx rồi +1.
      <!-- design.md flow 3 Step 4. Gia hạn expires_at là khác biệt duy nhất so với flow 1 — thiếu nó thì resend thành công vẫn để payload chết ở giây 90.
      Verify: TestRegistrationRepository_ReissueOTP, hai subtest — gia hạn cho purpose=register,
      KHÔNG đụng pending_registration cho purpose khác (login) dù cùng file/hàm. -->
- [x] `CreateUserAndSession` — MỘT transaction: re-check `users.email` → `INSERT users` (`email_verified_at = NULL`) → `DELETE pending_registration` → `UPDATE otp_codes SET used_at = now()` → `INSERT user_sessions`; gọi closure `TokenIssuer` BÊN TRONG tx
      <!-- design.md flow 2 Step 5-6. Re-check (a) là bắt buộc vì `pending_registration.email` cố ý KHÔNG unique.
      Verify: TestRegistrationRepository_CreateUserAndSession, 4 subtest: happy path (user +
      session + email_verified_at NULL + refresh_token_hash != raw token), email re-check chặn
      trước insert, phone unique-violation chặn TẠI insert (re-check chỉ xét email), token issuer
      lỗi → rollback toàn bộ (users=0 hàng, pending vẫn còn). -->
- [x] Dịch `pq` unique violation (`23505` + tên constraint) thành `ErrEmailTaken` / `ErrPhoneTaken`. Đây là chốt chặn race của AC6 — check ở usecase chỉ để báo lỗi sớm, KHÔNG phải chốt chặn cuối.
      <!-- Tên constraint phải khớp CHÍNH XÁC `users_email_key` / `users_phone_number_key` do Postgres
      tự đặt — xem migrations/README.md §"Bất biến schema phải giữ".
      Verify: TestIsUniqueViolation (unit, 4 case) + TestRegistrationRepository_CreateUserAndSession
      subtest "phone_unique_violation_on_insert_is_translated..." — ép thật một 23505 trên
      `users_phone_number_key` bằng cách chèn trước một `users` row cùng SĐT nhưng khác email (qua
      được re-check email), rồi assert lỗi trả về đúng ErrPhoneTaken. -->
- [x] `nullableString()` — chuỗi rỗng của `dharma_name` / `ordination_place` ghi thành SQL NULL (AC9)
      <!-- Verify: TestNullableString (unit) + TestRegistrationRepository_SavePendingAndIssueOTP
      đọc lại `DharmaName` không rỗng qua FindPendingRegistration. -->
- [x] Repository tests `repository/postgres_registration_repository_test.go` — chạy được với `-short`; nhánh cần Postgres thật bị skip khi thiếu credential
      <!-- `go test -short ./services/appService/repository/...` → tất cả integration test SKIP
      (testing.Short() gate ở connectTestDB), 3 unit test (`nullableString`, `isUniqueViolation`,
      `TestHashRefreshToken` cho `domain.HashRefreshToken`) vẫn PASS. Không skip: chạy full `go test` với Postgres thật (container
      Docker tạm, khởi ephemeral lúc verify, không lưu trong repo/CI) — 10 TestRegistrationRepository_*
      func, tất cả pass. `applyMigration()` tự chạy migrations/0001 (idempotent, toàn CREATE ... IF
      NOT EXISTS) nên test tự bootstrap schema, không phụ thuộc bước migrate riêng. -->
- [x] Test `ReissueOTP` khẳng định `pending_registration.expires_at` được đẩy lên `now() + 90s` và `password_hash` không bị đụng
      <!-- design.md flow 3 Step 4(c) — đây là hành vi dễ bị viết thiếu nhất của flow resend.
      Verify: TestRegistrationRepository_ReissueOTP/invalidates_old_otp... assert pending.ExpiresAt
      == newExpiresAt (WithinDuration 2s) sau ReissueOTP với pending TTL ngắn ban đầu (5s) — chứng
      minh gia hạn thật sự xảy ra, không phải trùng hợp còn hạn sẵn. password_hash không có trong
      câu UPDATE nào của ReissueOTP (chỉ SET expires_at) nên không cần test riêng — không có câu
      lệnh nào có thể đụng vào nó. -->

## Phase 4 — Delivery

- [x] `delivery/http/register_handler.go` — DTO `requestOTPRequest` / `resendOTPRequest` / `verifyOTPRequest` + response types; JSON tag chỉ sống ở file này, không rò vào `domain/`
      <!-- 2026-09-17: thêm `userResponse` (không phải `domain.User` trực tiếp — xem layer_conventions
      "Entity vs response model"). `go vet` + code review xác nhận không file `domain/*.go` nào có
      json tag. -->
- [x] `delivery/http/register_handler.go` — `RequestOTP` decode strict, validate qua `validation.Validator` (tag `phone` đã đăng ký ở composition root), map domain error → status
      <!-- Verify: TestRegisterHandler_RequestOTP_Success, _MalformedJSON, _UnknownField,
      _MissingRequiredField, _MalformedPhoneCaughtByValidator, _DomainErrors (6 case bảng),
      _OTPLockedIncludesDeadline. -->
- [x] `delivery/http/register_handler.go` — `ResendOTP` decode strict, validate `purpose` bằng `oneof=register login reset_password verify_email`, map domain error → status
      <!-- Verify: TestRegisterHandler_ResendOTP_Success, _PurposeOutsideOneof (chặn ở validator,
      chưa chạm usecase — AssertNotCalled), _UnsupportedPurposeFromUsecase (purpose hợp lệ theo
      oneof nhưng usecase chưa hiện thực — "login"), _TooSoonIncludesRetryAfter, _Exhausted. -->
- [x] `delivery/http/register_handler.go` — `VerifyOTP` decode strict, map domain error → status
      <!-- Verify: TestRegisterHandler_VerifyOTP_Success, _OTPCodeWrongShape,
      _PendingRegistrationNotFoundMapsToOTPInvalidNot404, _Expired, _Locked. -->
- [x] `delivery/http/errors.go` — helper dùng chung map error → envelope `utils.ApiResponse` `{success, error: {code, message, type}, details}`; US-001 là consumer đầu tiên
      <!-- Sửa mô tả v4.3 (2026-09-21): bản cũ ghi `{error: {code, message, details}}` — sai so với code
      (`details` nằm cạnh `error`, có thêm `success` và `error.type`). Code không đổi; item vẫn `[x]`. -->
- [x] Map sentinel → HTTP: `ErrPhoneTaken` / `ErrEmailTaken` → 409; `ErrInvalidPhone` / `ErrInvalidEmail` / `ErrInvalidPassword` / `ErrOTPInvalid` / `ErrOTPExpired` / `ErrPendingRegistrationNotFound` / `ErrPendingRegistrationExpired` / `ErrUnsupportedOTPPurpose` → 400; `ErrOTPResendTooSoon` / `ErrOTPResendExhausted` / `ErrOTPLocked` → 429
      <!-- Bổ sung 2026-09-17: `ErrSMSSendFailed` → 500 `SMS_SEND_FAILED` (chốt ở changelog v3.0
      mục 4, sentinel đã có trong domain/errors.go). Item gốc thiếu nhánh này — implement xong,
      `writeDomainError` có case riêng, test "sms send failed" trong _DomainErrors. Default case
      (lỗi không nhận diện) → 500 INTERNAL_ERROR, test "unmapped infra error". -->
- [x] `VerifyOTP` trả 400 `OTP_INVALID` (không phải 404) khi không có `pending_registration` — tránh cho caller dò SĐT nào đang có lượt đăng ký dở
      <!-- design.md flow 2 Step 2. Verify: TestRegisterHandler_VerifyOTP_PendingRegistrationNotFoundMapsToOTPInvalidNot404
      assert rec.Code == 400 kèm message "must not be 404". -->
- [x] `ErrOTPLocked` trả kèm `locked_until` trong response body
      <!-- Dùng `errors.AsType[*domain.OTPLockedError]` (Go 1.26 generic form của errors.As), không
      phải `errors.Is` — sentinel không mang timestamp. Verify:
      TestRegisterHandler_RequestOTP_OTPLockedIncludesDeadline. -->
- [x] Bước 1 validate format phía server bằng `pkg/validation` — AC7/AC8 mô tả validate ở client, nhưng server vẫn phải tự validate lại, không tin client
      <!-- `requestOTPRequest.PhoneNumber` mang tag `phone` (PhoneRule), `Email` mang `email`. Server
      luôn validate bất kể client đã validate hay chưa — request test không giả lập client "đã
      validate", tự gửi input sai định dạng thẳng vào handler. -->
- [x] Route 3 endpoint trong `services/appService/cmd/main.go`: `POST /v1/auth/register/request-otp`, `POST /v1/auth/resend-otp`, `POST /v1/auth/register/verify-otp` — **cả ba public**, KHÔNG wrap `RequireAuth`
      <!-- `mux.HandleFunc("POST /v1/auth/...", handler.Method)` ×3, không middleware nào bọc quanh.
      Verify: chạy thật `go run ./services/appService/cmd` với Postgres Docker tạm — cả 3 route
      trả đúng response qua curl (xem log verify Phase 6 bên dưới), không cần Authorization header. -->
- [x] `cmd/main.go` — đăng ký `validation.PhoneTag` ở composition root (trước lần validate đầu tiên); constructor injection theo thứ tự phụ thuộc; chạm các block `⚠️ DEV ONLY` / `⚠️ DEV CONFIG`
      <!-- File đã bị xoá ở commit 0475281, viết lại từ template cũ (commit e622180, trước khi
      luồng 2 bước ra đời) — giữ style ⚠️ DEV CONFIG/⚠️ DEV ONLY, MockSecretGetter, logging.New,
      jwt.New. Bỏ các phần thuộc US-002/003/004/content (login/forgot-password/profile/avatar/
      sutras/mantras/bodhisattvas — chưa tồn tại code cho các usecase đó). Thêm mới so với template:
      `validator.Register(validation.PhoneTag, ...)` + `validator.SetFieldNameTag("json")`, wiring
      `pkg/sms` (blank import `pkg/sms/logsms` + `sms.New(cfg.SMS.Provider, logger)`),
      `repository.NewSMSOTPSender`. -->
- [x] Tạo `docs/specs/api/openapi.yaml` — `docs/specs/api/` hiện rỗng, phải dựng mới với 3 route của US-001
      <!-- OpenAPI 3.0.3, 3 path + schema request/response/error khớp code thật (không phải bản thiết
      kế lý thuyết) — viết SAU khi handler đã chạy thật qua curl, để mọi field/status/code trong file
      đều verify được bằng request thật. YAML syntax kiểm bằng gopkg.in/yaml.v3 (repo đã có dep này). -->
- [x] Handler tests `delivery/http/register_handler_test.go` — success mỗi bước (kể cả resend), từng nhánh lỗi, body sai định dạng / JSON hỏng, `purpose` ngoài danh sách `oneof`
      <!-- 21 test func (bao gồm bảng con _DomainErrors 6 case), `go test ./services/appService/delivery/...`
      pass. Mock usecase hand-written (`mock_register_usecase_test.go`) theo đúng convention
      testify.Mock trong cùng package với interface `registerUsecase`. -->

## Phase 5 — Hạ tầng dùng chung

- [x] Tạo `pkg/sms/sms.go` theo đúng pattern provider-agnostic của repo: interface `Sender` ở package root + `Register(type, FactoryFunc)` / `New(config)` backed by package-level map
      <!-- Verify: pkg/sms/sms_test.go — TestNew_NilConfig/_MissingProvider/_UnregisteredProvider/
      _UsesRegisteredFactory/_FactoryErrorPropagates. `Config` có thêm field `Logger logging.Logger`
      so với pkg/database's Config — lý do: impl duy nhất hiện có (log-only) *chính là* việc ghi log,
      nên cần một sink; field này optional cho provider thật sau này. -->
- [x] `pkg/sms/logsms/log_sms.go` — impl **chỉ ghi log** nội dung SMS, tự đăng ký qua `init()`. Đây là impl duy nhất ở giai đoạn này; provider thật (Twilio / AWS SNS / GCP tương đương) thêm sau bằng subpackage mới, **không đổi interface**.
      <!-- Verify: pkg/sms/logsms/log_sms_test.go. Adapter `services/appService/repository/sms_otp_sender.go`
      implement `domain.OTPSender` bằng cách bọc `sms.Sender` — KHÔNG có trong bảng Layers gốc của
      design.md, bổ sung vì domain.OTPSender cần otpCode+purpose còn pkg/sms.Sender chỉ biết
      "gửi text tới một số" (cố ý, để pkg/sms không phụ thuộc domain type của service này). Sống ở
      `repository/` vì đó là nơi mọi domain port khác đã được implement bằng infra thật. Verify:
      repository/sms_otp_sender_test.go. -->
- [x] Thêm `SMS` vào `services/appService/config/config.go` (`AppConfig.SMS`) + `config.dev.yaml` (cùng chỗ với các block `⚠️ DEV ONLY`)
      <!-- `SMSConfig{Provider string}`, giá trị dev = "log". Verify: `go test ./services/appService/config/...`
      không hỏng (không có test nào assert số field cố định nên không cần sửa test), chạy thật qua
      `go run ./services/appService/cmd` xác nhận `sms.New` nhận đúng provider từ YAML. -->
- [x] Bổ sung `validation.NormalizePhone(raw string) string` vào `pkg/validation` (cạnh `PhoneRule` / `IsValidPhone`) — hàm thuần, không I/O
      <!-- design.md flow 1 Step 1 — đã chốt đặt ở pkg/validation, KHÔNG trong usecase, để US-002/US-003 dùng lại đúng một bản. -->
- [x] Test `pkg/validation/phone_test.go` — bảng `NormalizePhone`: `"0901234567"` → `"+84901234567"`, `"+84901234567"` giữ nguyên, `"+14155552671"` giữ nguyên (không ép +84), separator `"090 123-4567"` → `"+84901234567"`
      <!-- Verify: TestNormalizePhone phủ đúng 4 case trên + case từ chối; thêm
      TestNormalizePhoneOutputAlwaysPassesPhoneRule pin hợp đồng "kết quả khác rỗng thì PhoneRule
      luôn nhận". `go test ./pkg/validation/...` pass. -->
- [x] `go mod tidy` + `go mod vendor` sau khi thêm import mới — repo dùng `vendor/` và đã có `go.sum`
      <!-- pkg/sms/repository chỉ thêm import nội bộ module + `pkg/logging` (đã có sẵn trong
      go.mod/vendor) — không mang dependency bên thứ ba MỚI nào. `go mod tidy` chạy vẫn có tác dụng:
      xoá `github.com/DATA-DOG/go-sqlmock` — dependency rác không dùng ở đâu trong repo, có từ trước
      phiên làm việc này (không phải do Phase 3-5 để lại). `go mod vendor` sau đó không đổi gì trong
      `vendor/` (go-sqlmock chưa từng thực sự được vendor). `go build -mod=vendor ./...` vẫn sạch. -->

## Phase 6 — Integration & verification

- [x] `go build ./...` sạch
- [x] `go vet ./...` sạch
- [x] `go test -short ./...` sạch — không cần credential AWS/GCP thật
      <!-- verified 2026-09-21: `go test -short ./...` toàn repo xanh. 3 test `pkg/utils` đỏ ghi dưới đây
      đã được sửa ở `docs/specs/features/json-body-hardening/`. Ghi chú cũ giữ lại làm lịch sử. -->
      <!-- CHƯA sạch, KHÔNG đổi từ trước: `pkg/utils` FAIL 3 test (TestDecodeJSONBody_UnknownField_Disallowed,
      TestDecodeJSONBody_MultipleJSONObjects, TestDecodeJSONBody_WrongContentType —
      http_test.go:153/188/205). Không liên quan feature này (là 3 test có sẵn trước khi Phase 3-5
      của US-001 bắt đầu). Toàn bộ package còn lại — kể cả 4 package mới (`repository`,
      `delivery/http`, `pkg/sms`, `pkg/sms/logsms`) — pass. Không tick vì item nói "sạch" và 3 test
      trên vẫn đỏ. -->
- [x] Chạy end-to-end với Postgres thật khi có sẵn: request-otp → resend-otp → verify-otp
      <!-- 2026-09-17: chạy thật `go run ./services/appService/cmd` trỏ vào Postgres 16 (Docker tạm,
      không lưu trong repo/CI) đã áp `migrations/0001`. Qua `curl`:
      (1) request-otp → 200, OTP log ra "111111" ở `pkg/sms/logsms`;
      (2) resend-otp ngay sau đó → 429 OTP_RESEND_TOO_SOON (đúng cooldown 60s, không phải bug);
      (3) verify-otp với mã sai "000000" → 400 OTP_INVALID;
      (4) verify-otp với mã đúng "111111" → 201, trả access_token/refresh_token/user, `users.email_verified_at`
      = NULL (AC2/AC3), `user_sessions.expires_at` = +168h;
      (5) request-otp lần hai cùng SĐT → 409 PHONE_TAKEN (unique constraint chặn đúng).
      Xác nhận toàn bộ flow AC1/AC2/AC3/AC5(cooldown)/AC6/AC7 chạy đúng trên hạ tầng thật, không chỉ
      qua mock. -->
- [x] Xác nhận chuỗi resend không phá AC12: sau 3 lần sai OTP, `resend-otp` **cũng** bị chặn bởi `locked_until`, không xoá được bộ đếm
      <!-- design.md flow 3 Step 3 — kẻ brute-force không được lấy resend-otp làm cách reset attempts.
      Verify ở tầng usecase: TestResendOTP_Blocked case "locked out after too many wrong codes"
      dựng hàng OTP có LockedUntil tương lai và assert ErrOTPLocked + AssertNotCalled("ReissueOTP").
      Còn phải chạy lại end-to-end khi có Postgres — item "chạy end-to-end" ở trên. -->
- Dọn dẹp định kỳ hàng hết hạn — **chốt 2026-09-21: chấp nhận ở MVP**, không code; chuyển xuống Deferred.
- [x] Ghi vào `changelog.md` concept OKF nào dưới `docs/knowledge/features/` cần `bdd-okf-docs` tạo
      <!-- docs/knowledge/features/ hiện RỖNG (6 concept content đã bị xoá ở commit 0475281).
      Identity là concept MỚI cần tạo, không phải refresh.
      Verify: changelog.md đã ghi ở hai chỗ — dòng 101-102 và dòng 169-170. -->

## Phase 7 — Thay đổi từ `login/`: access token mang `sid`

Nguồn: `docs/specs/features/login/design.md` "Phụ thuộc: sửa US-001". Mục đích: `POST /v1/auth/logout`
(RequireAuth) tìm session qua claim `sid` của access token. **Hành vi ngoài của US-001 không đổi**
(status, body, error code giữ nguyên) nên `openapi.yaml` không cần sửa. Đây là điều kiện tiên quyết của
`login/tasks.md` Phase 0.

- [x] `domain/register.go` — đổi `TokenIssuer` từ `func(userID string)` thành `func(userID, sessionID string) (accessToken, refreshToken string, err error)`; cập nhật doc comment (vì sao `sessionID` phải có trước khi mint)
- [x] `usecase/register_usecase.go` — closure tại `VerifyOTP` (~dòng 369) nhận `sessionID`, gọi `GenerateAccessToken(userID, email, userRole, map[string]any{"sid": sessionID})`; `GenerateRefreshToken(userID, nil)` giữ nguyên
- [x] `repository/postgres_registration_repository.go` — `CreateUserAndSession`: `sessionID := uuid.NewString()` **trước** `issueTokens(userID, sessionID)`, rồi INSERT `user_sessions` với đúng `sessionID` đó (hiện `uuid.NewString()` nằm trong câu INSERT, sau khi mint)
- [x] Sửa test bị ảnh hưởng: `usecase/register_usecase_test.go` (các `f.jwt.On("GenerateAccessToken", ...)` — extra không còn là `nil`), `repository/postgres_registration_repository_test.go` (closure issuer có thêm tham số), mock nếu có
      <!-- `domain/mock_registration_repository.go` không đổi (nó nhận `TokenIssuer` như một giá trị, không gọi). Đã sửa: `stubTokenIssuer` + `failingIssuer` ở repository test, hai lời gọi `issuer(...)` ở usecase test. -->
- [x] Test mới: usecase — access token sinh ra mang `extra["sid"]` bằng `sessionID` issuer nhận; repository — `sessionID` truyền vào issuer **bằng** `user_sessions.session_id` vừa ghi
      <!-- Verify: usecase `TestVerifyOTP_TokenIssuerContract` pin đúng `map[string]any{"sid": testSessionID}` (không dùng mock.Anything nên bỏ claim là đỏ); repository `TestRegistrationRepository_CreateUserAndSession/session_id_handed_to_the_issuer_equals_the_persisted_user_sessions.session_id`. -->
- [x] `go build ./...`, `go vet ./...`, `go test ./services/appService/...` xanh; chạy lại `verify-otp` thật rồi giải mã access token kiểm claim `sid` khớp `session_id` trong DB
      <!-- 2026-09-21: build/vet sạch; `go test -count=1 ./services/appService/...` (Postgres 16 Docker tạm, không lưu trong repo) pass toàn bộ. E2E thật: `go run ./services/appService/cmd` → request-otp → verify-otp (201) → giải mã JWT: `extra.sid` = `4a8d2a72-…6d1f` = `user_sessions.session_id` trong DB. check-architecture.sh + check_dependency_rule.sh sạch. -->

## Phase 8 — Fix: pending hết hạn chặn đăng ký lại (2026-09-21)

Phát hiện khi cập nhật concept OKF `docs/knowledge/features/register.md`: `FindConflicts` đếm mọi hàng
`pending_registration` kể cả hết hạn, và không gì xoá hàng đó → `request-otp` lại trả `409 PHONE_TAKEN`
vĩnh viễn (tái hiện trên binary thật). design.md flow 1 Step 2 (v4.5).

- [x] Repository test đỏ trước: `TestRegistrationRepository_FindConflicts` thêm 3 case — pending hết hạn không trùng; pending hết hạn của SĐT khác không giữ email; pending hết hạn không che user thật
      <!-- 2026-09-21: đỏ trên code cũ (3 FAIL), xanh sau khi sửa; chạy trên Postgres 16 thật (`innerlamp-pg`), không skip. -->
- [x] `repository/postgres_registration_repository.go` `FindConflicts`: hai điều kiện pending thêm `AND expires_at > now()`; doc comment ở repository + `domain.RegistrationRepository.FindConflicts`
- [x] E2E `services/appService/e2e/register_e2e_test.go` `TestRegisterFlow_ExpiredPendingCanRestart_E2E` (OTP/pending TTL override 2s): request-otp → lại ngay → 409 → hết hạn → verify 400 `PENDING_REGISTRATION_EXPIRED` → request-otp email khác 200 → verify 201 với email mới; pending hết hạn của SĐT khác không giữ email *(Phase 10 đã thay bằng `TestRegisterFlow_RestartOverwritesPending_E2E`: pending của chính SĐT nay bị ghi đè thay vì trả `409`.)*
      <!-- 2026-09-21: pass; trả SQL về bản cũ thì cả 2 subtest FAIL (`PHONE_TAKEN` / `EMAIL_TAKEN`), khôi phục diff sạch. Kèm sửa harness: binary build vào `os.MkdirTemp` + `TestMain` dọn (trước đây `t.TempDir()` bị xoá sau test đầu → test thứ hai `fork/exec … no such file`); harness thêm `newE2EServerWithConfig` để override key config có kiểm tồn tại. -->
- [x] `go build ./...`, `go vet`, `go test -short -count=1 ./...` xanh; `go test -count=1 ./services/appService/e2e/` xanh; `check-architecture.sh` (ARCH_ERRORS=0), `check_dependency_rule.sh` sạch

## Phase 9 — Rate-limit gửi OTP: bỏ chặn vĩnh viễn, thêm khoá gửi 30 phút (v5.0, 2026-09-22)

Nguồn: design.md §"OTP send-limit policy". Bug: `otp_codes.resend_count` carry-forward mãi mãi →
SĐT bị chặn vĩnh viễn, không có đường gỡ (known gap `docs/knowledge/features/register.md`). Anh chốt
phương án "khoá 30 phút" ngày 2026-09-22. **Phase này đã xong (2026-09-22)** — xem cột `Verify` của
từng item để biết bằng chứng.

Các item `[x]` ở Phase 1–3 mô tả hành vi v4.5 (`resend_count` carry-forward, trần 3 lần gửi lại,
khoá 15 phút, `pending_registration_ttl` 90s). Phase 9 **thay thế** chúng — không untick, vì chúng đã
verify đúng ở thời điểm viết; đọc kèm để biết cái gì bị đổi. Cụ thể: khối config `OTPConfig` +
`ErrOTPResendExhausted` ở Phase 1, `checkOTPLimits`/`resend_count` ở Phase 2, và phần carry-forward
trong `SavePendingAndIssueOTP`/`ReissueOTP` ở Phase 3. Giá trị đúng hiện tại nằm ở Phase 9/10 và
`design.md` §"OTP send-limit policy".

- [x] `migrations/0003_otp_send_limits.sql` — `CREATE TABLE IF NOT EXISTS otp_send_limits` (`phone_number`, `purpose`, `send_count`, `last_sent_at`, `locked_until`, `created_at`, `updated_at`, `UNIQUE (phone_number, purpose)`) + `ALTER TABLE otp_codes DROP COLUMN IF EXISTS resend_count`. Idempotent như `0001` (applyMigration tự chạy lại được).
- [x] `config/config.go` `OTPConfig` — đổi `ResendCooldown` → `SendCooldown` (60s), bỏ `MaxResend`, thêm `MaxSendAttempts` (3) và `SendLockDuration` (30m), `LockDuration` 15m → 30m; `config.dev.yaml` block `otp:` cập nhật tương ứng.
      <!-- Verify: config_test.go đọc các key mới; không còn field MaxResend/ResendCooldown. -->
- [x] `domain/otp_send_limit.go` — entity `OTPSendLimit` (`PhoneNumber`, `Purpose`, `SendCount`, `LastSentAt *time.Time`, `LockedUntil *time.Time`) + value type `SendLimits` (`Cooldown`, `LockDuration`, `MaxAttempts time.Duration/int`) để usecase truyền policy xuống repository mà `domain` không import `config`.
- [x] `domain/errors.go` — thêm sentinel `ErrOTPSendLimited` + typed `OTPSendLimitedError{LockedUntil time.Time}` (`Unwrap()` về sentinel); **xoá `ErrOTPResendExhausted`** (không còn nhánh nào trả).
- [x] `domain/register.go` — `RegistrationRepository`: `SavePendingAndIssueOTP` và `ReissueOTP` nhận thêm tham số `SendLimits`; thêm `ReleaseSendLimit(ctx, phoneNumber, purpose) error`; cập nhật doc comment (bộ đếm gửi tách khỏi hàng OTP, khoá-gửi không chặn verify).
      <!-- `ResendOTPResult.ResendRemaining` giữ tên field nhưng đổi nghĩa thành "số lượt gửi còn lại trong cycle" (design.md flow 3 Step 5). -->
- [x] `domain/mock_registration_repository.go` — cập nhật chữ ký hai method + `ReleaseSendLimit`.
- [x] `usecase/register_usecase.go` — `RequestOTP` truyền `SendLimits` xuống `SavePendingAndIssueOTP`; `ResendOTP` bỏ nhánh trần cũ, vẫn kiểm khoá-nhập-sai qua `FindActiveOTP` rồi truyền `SendLimits` xuống `ReissueOTP`; **xoá `checkOTPLimits`/`resendsRemaining`/`checkCooldown`** (toàn bộ policy chuyển vào repository, xem design.md §"OTP send-limit policy").
      <!-- AC5 v5.0: request-otp cũng tiêu lượt. AC12: LockDuration 30m. Verify: TestRequestOTP_Cooldown dùng typed error mới; TestRequestOTP_LockedNumber (khoá-nhập-sai) vẫn OTP_LOCKED. -->
- [x] `repository/postgres_registration_repository.go` — trong tx của `SavePendingAndIssueOTP`/`ReissueOTP`: `INSERT ... ON CONFLICT DO NOTHING` + `SELECT ... FOR UPDATE` trên `otp_send_limits`, áp design.md §policy (nhánh (0)–(d): reset cycle / khoá / cooldown / trần, tăng `send_count`, set `locked_until` khi chạm trần); `ReleaseSendLimit` xoá hàng (gọi trong tx `CreateUserAndSession`); **xoá `latestResendCount` và carry-forward**.
      <!-- Verify: TestRegistrationRepository_* cho otp_send_limits — lần đầu send_count=0; lượt 2/3 tăng; lượt 4 set locked_until; lượt 5 trả *OTPSendLimitedError; quá SendLockDuration thì reset; 2 request song song không cùng thấy send_count cũ. -->
- [x] `delivery/http/errors.go` — map `*OTPSendLimitedError` → `429 OTP_SEND_LIMITED` + `locked_until`; bỏ case `ErrOTPResendExhausted`.
- [x] `delivery/http/register_handler.go` + test — cập nhật mô tả `resend_remaining`; thêm test `OTP_SEND_LIMITED` có `locked_until`.
- [x] Tests: usecase (cooldown/trần/khoá-gửi/không-chặn-verify), repository (Postgres thật), handler (mã lỗi mới), và E2E `services/appService/e2e/register_e2e_test.go` — 4 lượt gửi liên tiếp (override `send_cooldown` ngắn), lượt 5 `OTP_SEND_LIMITED`, và `verify-otp` với OTP cuối vẫn 201 dù đang khoá-gửi.
- [x] `docs/specs/api/openapi.yaml` — thêm `OTP_SEND_LIMITED`, bỏ `OTP_RESEND_EXHAUSTED`, cập nhật `locked_until`/`retry_after_seconds`.
- [x] `go build ./...`, `go vet ./...`, `go test -short ./...` xanh; `go test -count=1 ./services/appService/...` (Postgres thật) xanh; `check-architecture.sh` + `check_dependency_rule.sh` sạch.

## Phase 10 — Tách pending TTL khỏi OTP TTL + đường "làm lại" (v5.1, 2026-09-22)

Nguồn: design.md §"Pending payload và đường làm lại" + AC15. Hệ quả của v5.0: payload tạm chỉ sống
90 giây nên `resend-otp` chết trước khoá-gửi rất lâu, còn `request-otp` bị `409 PHONE_TAKEN` bởi
chính lượt dở của mình. Anh chốt: để payload sống 24h (cron dọn sau), cho `request-otp` ghi đè payload
của chính SĐT, và `resend-otp` cấp OTP mới kể cả khi OTP cũ hết hạn. **Phase này đã xong (2026-09-22)**.

- [x] `config/config.go` + `config.dev.yaml` — `pending_registration_ttl` 90s → **24h**, doc comment bỏ ràng buộc "bằng TTL OTP".
      <!-- Verify: config_test.go TestLoad_DevYAMLOTPPolicy kỳ vọng 24h. -->
- [x] `domain/register.go` — `RegistrationConflicts` bỏ `PhoneInPending`; `EmailInPending` chỉ tính hàng pending của SĐT khác; `HasPhoneConflict()` = `PhoneInUsers`; doc `FindConflicts`/`ReissueOTP` cập nhật.
- [x] `domain/errors.go` — cập nhật comment `ErrPhoneTaken`/`ErrEmailTaken`/`ErrOTPExpired`/`ErrPendingRegistrationExpired` cho nghĩa mới.
- [x] `repository/postgres_registration_repository.go` — `FindConflicts` còn 3 `EXISTS`, điều kiện email-pending thêm `phone_number <> $1`; `ReissueOTP` **bỏ** bước gia hạn `pending_registration.expires_at`.
- [x] `usecase/register_usecase.go` — `ResendOTP`: `ErrOTPInvalid` từ `FindActiveOTP` không còn trả `OTP_EXPIRED`; chỉ kiểm khoá-nhập-sai khi có hàng OTP (`otp != nil`).
- [x] `domain/mock_registration_repository.go` — không đổi chữ ký (chỉ hành vi).
- [x] Tests: usecase (`TestResendOTP_AfterCodeExpired` thay `TestResendOTP_NoActiveOTP`; bảng conflict bỏ case pending-cùng-SĐT), repository (`FindConflicts` 3 nhánh mới; `ReissueOTP` không đổi pending expiry), config (24h), E2E (`TestRegisterFlow_RestartOverwritesPending_E2E`, `TestRegisterFlow_ResendAfterCodeExpired_E2E`).
- [x] `docs/specs/api/openapi.yaml` — cập nhật mô tả `PHONE_TAKEN`/`EMAIL_TAKEN`/`OTP_EXPIRED`/`PENDING_REGISTRATION_EXPIRED` cho nghĩa mới.
- [x] `services/appService/domain/README.md` — cập nhật `RegistrationConflicts`, bảng `RegistrationRepository` (10 method, `ReleaseSendLimit`).
- [x] `go build ./...`, `go vet ./...`, `go test -short ./...` xanh; `go test -count=1 ./services/appService/repository/...` + `./services/appService/e2e/...` (Postgres/testcontainers thật) xanh; `check-architecture.sh` + `check_dependency_rule.sh` sạch.

## Phase 11 — Thay `ordination_place` bằng `ordination_temple_id` (v7.0, 2026-10-10)

Nguồn: `requirements.md` AC4/AC8/AC18/AC19 (v7.0) + `design.md` §Layers (dòng `(v7.0)`), flow 1 Step 2,
Step 4(c), flow 2 Step 5(b). Anh chốt 2026-10-10: **thay hẳn** text bằng id, không giữ song song.
**Đã implement (2026-10-10).** Không còn câu hỏi mở (drop cột, không backfill; error code `INVALID_TEMPLE`; không chọn chùa = `NULL`). Verify: `go test -short ./...` xanh; `go test ./services/appService/repository/...` xanh trên Postgres thật; E2E mới `TestRegisterFlow_OrdinationTemple_E2E` xanh; `scripts/check-architecture.sh` ARCH_ERRORS=0.

- [x] `migrations/0023_registration_ordination_temple.sql` — ADD `ordination_temple_id VARCHAR(64) REFERENCES temples(id)` vào `users` + `pending_registration`; DROP `ordination_place` ở cả hai. Idempotent như các file khác.
- [x] `migrations/README.md` — thêm dòng `0023`; `docs/specs/db/erd.mmd` + `erd.svg` (render lại bằng `npx @mermaid-js/mermaid-cli -i erd.mmd -o erd.svg`, 2026-10-10).
- [x] `domain/user.go`, `domain/pending_registration.go`, `domain/register.go` — `OrdinationPlace` → `OrdinationTempleID *string` (entity + `RequestRegisterOTPInput`); thêm `TempleExists(ctx, id) (bool, error)` vào `RegistrationRepository`.
- [x] `domain/errors.go` — thêm `ErrInvalidTemple`.
- [x] `domain/mock_registration_repository.go` — thêm `TempleExists`.
- [x] `usecase/register_usecase.go` — `RequestOTP`: trim id (rỗng → `nil`); sau `FindConflicts` pass gọi `TempleExists` khi id != nil → `ErrInvalidTemple`; truyền `OrdinationTempleID` xuống `pending`.
- [x] `repository/postgres_registration_repository.go` — `TempleExists`; đổi cột trong `SavePendingAndIssueOTP` (c), `FindPendingRegistration`, `CreateUserAndSession` (b); dịch FK violation `23503` (`users_ordination_temple_id_fkey`) → `ErrInvalidTemple`.
- [x] `repository/postgres_login_repository.go` — `userColumns` + Scan: `ordination_place` → `ordination_temple_id` (login đọc cùng bảng `users`, sẽ vỡ nếu không đổi).
- [x] `delivery/http/register_handler.go` + `delivery/http/errors.go` — request DTO `ordination_temple_id`, `userResponse.ordination_temple_id` (omitempty); map `ErrInvalidTemple` → `400 INVALID_TEMPLE` details `{field}`.
- [x] Tests: usecase (id rỗng→NULL, id không tồn tại→`ErrInvalidTemple`, thứ tự phone/email trước chùa), repository trên DB thật (`TempleExists`, FK 23503 ở verify, round-trip `NULL`), handler (field cũ `ordination_place` → 400 `VALIDATION_ERROR`, strict decode), E2E (`register_e2e_test.go`: đăng ký có chùa seed → `user.ordination_temple_id` khớp; id rác → 400).
- [x] `docs/specs/api/openapi.yaml` (+ bản copy `web/docs/public/openapi.yaml`, `README.md` gốc, `web/docs/.../guides/register-and-login.mdx`) — request `request-otp` đổi field; schema `User` (`ordination_temple_id`, dùng chung cho login/verify-otp); thêm `INVALID_TEMPLE`.
- [x] `services/appService/domain/README.md`; `docs/knowledge/features/register.md` — refresh qua `il-okf-writer` (2026-10-10).
- [x] `go build ./...`, `go vet ./...`, `go test -short ./...` xanh; `go test -count=1 ./services/appService/repository/...` xanh với Postgres thật. *(E2E: test mới và 3 test E2E hay timeout đều xanh khi chạy riêng; một lượt chạy cả package `e2e` đồng thời trên máy này có 4 test timeout do tải — chưa chạy lại toàn bộ lần hai.)*

## Deferred (không chặn US-001)

- Job dọn hàng hết hạn (`pending_registration`, `otp_codes`; và `login_attempts`, `user_sessions` đã revoke/hết hạn của US-002) — **chốt 2026-09-21: chấp nhận rác tích tụ ở MVP.** Đúng cho `otp_codes`, `login_attempts`, `user_sessions` (vô hiệu qua `expires_at`/`is_revoked`/cửa sổ reset). `pending_registration` hết hạn từng chặn đăng ký lại (`FindConflicts` không lọc `expires_at`) — **đã sửa ở Phase 8**; nay hàng hết hạn cũng vô hiệu về logic và bị `request-otp` kế tiếp ghi đè. Khi cần: endpoint nội bộ + Cloud Scheduler (Cloud Run không tự chạy cron, goroutine ticker không an toàn khi scale 0 / nhiều instance).
- Xác thực email (gửi mail + set `email_verified_at`) — user story riêng, làm cùng dịp với US-003 vì cùng cần dịch vụ gửi email
- `POST /v1/auth/refresh` + rotate refresh token — ngoài scope US-001 (đã chốt); hệ quả là AC11 vế thứ hai không thoả được ở phạm vi này, xem design.md "Known gaps"
- Bảng `user_devices` + cột `user_sessions.device_id` — contract 2 bước không mang trường device nào; US-002 (đăng nhập, nay ở `docs/specs/features/login/`) định nghĩa payload device rồi thêm bằng migration `0002` riêng
- `resend-otp` phục vụ `purpose` khác `register` — endpoint và hằng `OTPPurpose` đã sẵn sàng, nhưng usecase trả `ErrUnsupportedOTPPurpose` cho tới khi US-002/US-003 wire vào
- Rate limiting cho `request-otp` / `resend-otp` theo IP (khác rate limit gửi lại theo SĐT đã có ở AC5) — cân nhắc ở middleware
- Đổi số điện thoại / email sau khi đã tạo tài khoản — cần luồng xác thực lại, US riêng
- Provider SMS thật (Twilio / SNS / GCP) — chỉ cần thêm subpackage `init()`, không đổi interface
- US-002 (đăng nhập) — **đã chuyển** sang `docs/specs/features/login/`, xem folder đó
- US-004 (hồ sơ) — còn trong `requirements.md` nhưng chưa được `design.md` đặc tả ở version này. US-003 (quên mật khẩu) đã chuyển sang [`docs/specs/features/password-reset/`](../password-reset/requirements.md) (v6.0).
