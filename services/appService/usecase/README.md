# Usecase — services/appService/usecase

## Mục đích
Tầng nghiệp vụ của Clean Architecture cho appService. Phần đăng ký (onboarding v8.0, ADR-0003): cơ chế OTP dùng chung (sinh mã `crypto/rand`, hash, áp send-limit, verify với giữ lượt nguyên tử, khoá) nằm trong `OTPService` (`otp_service.go`) — không biết gì về register và **không gửi thư** — còn `RegisterUsecase` giữ phần riêng của register với ba method ứng với ba bước: `RequestOTP`, `ResendOTP`, `VerifyOTP`.

Bất biến mà luồng đăng ký bảo vệ: **không hàng `users` nào được tạo trước khi email được xác minh**. Bước 1 chỉ ghi `pending_registration` + `otp_codes`; lệnh insert account nằm *bên trong* transaction của bước 2.

Package import `domain` (port, input/result, sentinel), `config` (chỉ đặt tên kiểu), cộng các abstraction trong `pkg/` (`jwt`, `logging`, `validation`). **Không** import `repository` hay `delivery`.

## Khi nào dùng / không dùng
- **Dùng khi**: thêm một bước nghiệp vụ mới cho luồng đăng ký, hoặc một use-case mới (US-002 login, US-003 reset password) — mỗi use-case một file `*_usecase.go` + `*_usecase_test.go` cùng cặp.
- **Không dùng khi**: cần truy vấn SQL (đặt ở `repository/`), cần parse/trả HTTP (đặt ở `delivery/http/`), cần định nghĩa kiểu dữ liệu biên (đặt ở `domain/`).
- **Không** đặt `json`/`db` tag ở đây, và **không** trả thẳng `domain` entity ra ngoài — handler tự khai response struct của nó.

## API chính

### `NewOTPService(repo, logger, policy) *OTPService`
Cơ chế OTP dùng chung (otp-mechanism v2.0). Dựng một lần ở composition root, inject chung cho đăng ký và
(sắp tới) đặt lại mật khẩu.

| Tham số | Kiểu | Vai trò |
|---|---|---|
| `repo` | `domain.OTPRepository` | `IssueOTP`, `ApplySendLimit`, `FindActiveOTP`, `ConsumeOTPAttempt`, `ReleaseSendLimit` — scope `(email, purpose)` |
| `logger` | `logging.Logger` | quan sát |
| `policy` | `config.OTPConfig` | TTL (600s), cooldown, trần gửi, số lần sai, thời gian khoá, `dev_fixed_code` |

Method công khai: `Generate()` (6 chữ số `crypto/rand`, giữ số 0 đầu, + bcrypt hash; dùng
`policy.DevFixedCode` khi có — `AppConfig.Validate` đã chặn nó ngoài `dev`), `Issue()` (Generate →
`IssueOTP`, trả cả mã thô để **caller** gửi thư), `ApplySendLimit()`, `ReleaseSendLimit()` (caller gọi khi
gửi thư lỗi), `EnsureNotLocked()`, `Verify()` (Step V-1…V-4: tìm mã → đang khoá/hết hạn? →
`ConsumeOTPAttempt` giữ lượt **trước** khi so hash → so bcrypt; sai luôn là `ErrOTPInvalid`, không nói còn
mấy lượt), cùng accessor `CodeExpiry`/`TTL`/`SendLimits`/`ExpiresInSeconds`/`ResendAfterSeconds`.

Đồng hồ là field `now` (mặc định `time.Now`); test cùng package thay nó để ghim hạn mã/khoá.

### `NewRegisterUsecase(registrations, otpService, mailer, passwords, jwtService, logger, sessionTTL, pendingRegistrationTTL, dharmaNameMaxChars, fullNameMaxChars) *RegisterUsecase`

| Tham số | Kiểu | Vai trò |
|---|---|---|
| `registrations` | `domain.RegistrationRepository` | `PhoneTaken`, `EmailHasAccount`, `TempleExists`, `SavePendingAndIssueOTP`, `FindPendingRegistration`, `CreateUserAndSession`, `DeletePendingRegistration` |
| `otpService` | `*OTPService` | cơ chế OTP dùng chung |
| `mailer` | `domain.AuthMailer` | thư mã đăng ký / thư báo "email đã có tài khoản" |
| `passwords` | `domain.PasswordHasher` | `BcryptPasswordHasher` — `bcrypt(base64(SHA-256(NFC(pw))))`, dùng chung với login |
| `jwtService` | `jwt.Service` | mint access/refresh token |
| `logger` | `logging.Logger` | quan sát (không log email, mã, mật khẩu) |
| `sessionTTL` | `time.Duration` | từ `JWTConfig.RefreshTokenExpiration` |
| `pendingRegistrationTTL` | `time.Duration` | từ `OTPConfig.PendingRegistrationTTL` (24h) |
| `dharmaNameMaxChars`, `fullNameMaxChars` | `int` | từ `ProfileConfig` |

### `RequestOTP(ctx, domain.RequestRegisterOTPInput) (*domain.RequestOTPResult, error)`
Bước 1 (design Step R-2…R-8):

1. Validate, không chạm DB, theo thứ tự email (`validation.NormalizeEmail`, AC3) → SĐT theo nước
   (`validation.NormalizePhoneForCountry`, AC4) → mật khẩu 8–64 ký tự sau NFC (`ValidatePassword`, AC5) →
   họ tên / pháp danh (AC6). Trả lỗi của trường sai **đầu tiên**.
2. `PhoneTaken` → `ErrPhoneTaken` (AC7). Chạy trước kiểm email nên 409 không nói gì về email.
3. `TempleExists` khi có `ordination_temple_id` (AC8).
4. `EmailHasAccount` — chỉ chọn thư ở bước 8.
5. `OTPService.EnsureNotLocked` — mã đang bị khoá nhập sai thì không cấp mã mới (cấp mới = phát lượt đoán mới).
6. Hash mật khẩu + `OTPService.Generate()` — **cả hai nhánh** làm cùng việc (AC2).
7. `SavePendingAndIssueOTP` — một transaction, send-limit áp trước (AC9).
8. Sau commit: `SendRegisterCode` hoặc `SendAccountExistsNotice`. Lỗi → `ReleaseSendLimit` rồi
   `ErrEmailSendFailed` (AC10).

### `ResendOTP(ctx, domain.ResendOTPInput) (*domain.ResendOTPResult, error)`
Step S-2…S-5: email hợp lệ → `FindPendingRegistration` (không có / hết hạn → `ErrPendingRegistrationExpired`,
AC12) → `EnsureNotLocked` → `OTPService.Issue(register)` → gửi thư theo `AccountExists` (như bước 8 ở
trên). Payload không đổi.

### `VerifyOTP(ctx, domain.VerifyRegisterOTPInput) (*domain.RegisterResult, error)`
Step V-2…V-5: email hợp lệ → `FindPendingRegistration` (AC17) → `OTPService.Verify` (AC15) → nếu
`pending.AccountExists` thì `ErrOTPInvalid` dù mã đúng (AC16; lượt vẫn bị tính) →
`CreateUserAndSession(ctx, pending, otp.ID, sessionTTL, tokenIssuer(pending))` (AC14, AC18).

### `tokenIssuer(pending) domain.TokenIssuer`
Closure mà repository gọi **bên trong** transaction của nó: sau khi `INSERT users` sinh ID, trước khi `INSERT user_sessions` cần token. Sign sau transaction sẽ để lại một account đã commit mà không có session dùng được nếu bước hai fail. Repository gọi nó mà **không** import `pkg/jwt` — nó chỉ thấy function type `domain.TokenIssuer`.

Access token mang `(userID, email, "user", {"sid": sessionID})`; refresh token mang `(userID, nil)`.

### Hằng nội bộ
```go
userRole    = "user"
otpKeyspace = 1_000_000 // mã 6 chữ số — định dạng cố định, không phải cấu hình
```

Policy OTP (TTL, cooldown, trần gửi, số lần sai, thời gian khoá, TTL pending) là **cấu hình**
`config.OTPConfig`, truyền qua constructor. Độ dài mật khẩu 8–64 là luật sản phẩm BR-ONB-06 nên là hằng
ở `domain/password.go`.

### `NewAttestationUsecase(repo, verifier, logger, cfg) *AttestationUsecase`
Use-case cho feature attestation (`attestation_usecase.go`) — Play Integrity (android) + App Attest (iOS).

| Tham số | Kiểu | Vai trò |
|---|---|---|
| `repo` | `domain.AttestationRepository` | port 6 method (challenge + key); implementation Postgres ở `repository/` |
| `verifier` | `domain.AttestationVerifier` | seam tới `pkg/attestation`; adapter ở `repository/attestation_verifier.go` |
| `logger` | `logging.Logger` | quan sát |
| `cfg` | `config.AttestationConfig` | `challenge_ttl`, `freshness_window`, `android.package_name` + ngưỡng verdict, `ios.app_id` |

Method: `RequestChallenge` (sinh nonce `crypto/rand` → `SaveChallenge`), `Verify` (rẽ nhánh android/ios →
trả `Verdict` — verdict âm tính là kết quả, không phải error), `VerifyAssertion` (cho middleware
`RequireAttestation`: `FindChallenge` → `FindKey` → `VerifyIOSAssertion` → `UpdateKeyCounter` +
`MarkChallengeUsed`). Upstream error (android) **không** đánh dấu nonce đã dùng để caller retry được.

## Cấu hình / dependency
- Không đọc biến môi trường, không đọc package-level config singleton. Policy OTP là `config.OTPConfig`, do `OTPService` nắm (constructor `NewOTPService(..., policy)`); `sessionTTL`/`pendingRegistrationTTL` là các giá trị riêng do composition root truyền vào `NewRegisterUsecase`; policy attestation là `config.AttestationConfig`, truyền vào `NewAttestationUsecase`.
- Import: `domain` (port/input/sentinel), `pkg/jwt`, `pkg/logging`, `pkg/validation`, `config` (chỉ để *đặt tên* kiểu `OTPConfig`/`AttestationConfig`), `golang.org/x/crypto/bcrypt`, `golang.org/x/text/unicode/norm`, thư viện chuẩn.
- Import path: `chjpcoj.com.innerlamp/services/appService/usecase`.
- Consumer: `delivery/http/` (khai interface consumer-side là subset của use-case) và `cmd/main.go` (dựng concrete).
- Kiểm tra chiều phụ thuộc: `bash .claude/skills/golang-clean-arch-tdd/scripts/check_dependency_rule.sh services/appService`.

## Test
`otp_service_test.go` — cơ chế OTP dùng chung (otp-mechanism AC1–AC11: mã ngẫu nhiên có số 0 đầu, phân phối đủ
6 vị trí × 10 chữ số, mã dev, giữ lượt trước khi so, không lộ số lượt còn lại), mock `domain.MockOTPRepository`,
đồng hồ giả. **Không** DB, không mạng.

`register_usecase_test.go` — onboarding AC1–AC20, mock `domain.MockRegistrationRepository` +
`domain.MockOTPRepository` + `domain.MockAuthMailer` + `jwt.MockService` + `logging.MockLogger`, `OTPService`
và `BcryptPasswordHasher` thật, đồng hồ giả. Mock là strict: lời gọi không stub sẽ panic — đó là cách test
chứng minh "không ghi gì / không gửi gì".

`attestation_usecase_test.go` — table-driven, mock `domain.MockAttestationRepository` + `domain.MockAttestationVerifier`, phủ challenge/verify (android pass/fail, upstream, ios attest pass/fail) và assertion (counter tăng/không tăng/unknown key/challenge invalid/chữ ký sai). **Không** DB, không mạng.

Comment đầu file test liệt kê mapping AC → tên test; đọc nó trước khi thêm test mới để không tạo khoảng trống hoặc trùng lặp.

## Lưu ý / lỗi hay gặp
- **Hai nhánh của request-otp phải làm cùng việc** (AC2): hash mật khẩu, sinh + hash mã, một lần ghi, một lần gọi mailer. Thêm một bước chỉ ở một nhánh là mở lại kênh dò email qua thời gian phản hồi.
- **Thư gửi sau commit, lỗi thư trả về lỗi** (AC10), và trước khi trả lỗi thì nhả slot gửi (`ReleaseSendLimit`) để lần thử lại không dính cooldown vì một thư chưa từng đi.
- **`ErrPendingRegistrationNotFound` không rò ra ngoài**: use-case gộp nó vào `ErrPendingRegistrationExpired` (AC12/AC17).
- **Không log email, mã, mật khẩu hay hash.** Hash bcrypt của một mã 6 số sống ngắn vẫn đủ để brute-force ngoài luồng.
- **Đừng thêm nil-guard cho `result.User`** ở `VerifyOTP` để đối phó một mock trả `RegisterResult{}` rỗng — đó là mock vi phạm contract của port. Sửa stub, không nới code.
- Test có thể panic với `mock: I don't know what to return` khi impl gọi một method chưa được stub. Đó thường là **test thiếu kỳ vọng**, không phải bug impl — đọc method bị gọi trước khi kết luận.
