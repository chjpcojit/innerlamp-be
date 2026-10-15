# Domain — services/appService/domain

## Mục đích
Tầng entity của Clean Architecture cho appService: struct nghiệp vụ thuần (`User`, `PendingRegistration`, `OTPCode`), interface repository mà `usecase/` phụ thuộc và `repository/` implement, port `AuthMailer` / `PasswordHasher`, các Input/Result của use-case, và sentinel error dùng chung toàn service.

Package này là **tầng trong cùng** — mọi tầng khác phụ thuộc nó, nó không phụ thuộc tầng nào. Đây không phải quy ước cho vui: `domain` hiện có **zero import ngoài thư viện chuẩn** (cộng `testify/mock` trong các file `mock_*.go`). Ai thêm một import `pkg/*` hay `net/http` vào đây là làm hỏng kiến trúc, không phải "hơi lệch convention".

Phạm vi đăng ký: **US-001 v8.0 (đăng ký bằng email, mã 6 số gửi qua email — ADR-0003)**; login (US-002) vẫn theo SĐT. Chưa có `Device`/`Session` entity — `user_sessions` không có `device_id` ở story này (xem `docs/specs/features/onboarding/design.md` §Known gaps).

## Khi nào dùng / không dùng
- **Dùng khi**: viết use-case mới (cần struct input/result + interface port), viết repository implementation mới (implement interface ở đây), viết unit test (mock nằm sẵn trong package), hoặc cần phân loại lỗi nghiệp vụ bằng `errors.Is`.
- **Không dùng khi**: cần logic nghiệp vụ (đặt ở `usecase/`), cần truy vấn DB thật (đặt ở `repository/`), cần parse/trả HTTP (đặt ở `delivery/http/`).
- **Không** thêm `json:"..."` hay `db:"..."` vào struct ở đây. Hình dạng wire là quyết định của `delivery` (mỗi handler có response struct riêng), hình dạng cột là quyết định của `repository` (SQL). Gắn tag vào entity sẽ ép một hình dạng JSON lên mọi consumer và khiến entity phải sửa mỗi khi một API contract không liên quan đổi.

## API chính

### Entity

- **`User`** (`user.go`) — map bảng `users`. `ID int64`, `Email`, `EmailVerifiedAt *time.Time`, `PhoneNumber`, `PasswordHash`, `FullName *string`, `DharmaName *string`, `OrdinationTempleID *string`, `CreatedAt`, `UpdatedAt`.
  `EmailVerifiedAt` = thời điểm tạo tài khoản (đăng ký xác minh email bằng mã, AC14); chỉ NULL với tài khoản tạo trước migration `0037`.
- **`PendingRegistration`** (`pending_registration.go`) — map bảng `pending_registration`. Payload tài khoản (gồm **hash** password, không bao giờ plaintext) nằm tạm giữa bước 1 và bước 2.
  Đây là struct giữ bất biến trung tâm của US-001: **không hàng `users` nào tồn tại ở trạng thái chưa verify**. Một hàng cho mỗi **email** (`email` UNIQUE — chỗ dựa cho `ON CONFLICT`); `phone_number` **không** unique ở bảng này: hai lượt đăng ký dở có thể cùng SĐT, ai xác minh trước giữ số (AC7/AC18).
  `AccountExists` đánh dấu hàng tạo cho email **đã có tài khoản** (AC2): người gọi nhận y hệt email mới, chủ email nhận thư báo thay vì mã, verify-otp coi mọi mã là sai (AC16).
  `ExpiresAt` = `otp.pending_registration_ttl` (24h) sau lần request-otp gần nhất.
- **`OTPCode`** + **`OTPPurpose`** (`otp.go`) — map bảng `otp_codes`, dùng chung mọi flow OTP, scope theo cặp `(Email, Purpose)`.
  `OTPHash` là **bcrypt**, không phải mã 6 số. Không gian mã chỉ 10^6 nên hash nhanh (SHA-256) đổ trong vài giây bằng bảng tra sẵn.
  Hàng mang giới hạn nhập sai (`Attempts`, `LockedUntil`): mỗi lần verify **giữ lượt nguyên tử trước khi so** (`ConsumeOTPAttempt`), lượt chạm trần đặt khoá trong cùng câu lệnh (otp-mechanism AC7). Giới hạn **gửi** nằm ở `OTPSendLimit` (`otp_send_limit.go`), không khoá verify (AC13).
  Bốn hằng `OTPPurposeRegister/Login/ResetPassword/VerifyEmail` khớp `chk_otp_codes_purpose`; đang dùng `register` (và sắp tới `reset_password`).
- **`PasswordHasher`** + `ValidatePasswordLength` (`password.go`) — port hash mật khẩu (8–64 ký tự sau NFC, BR-ONB-06). Hiện thực: `usecase.BcryptPasswordHasher` (`bcrypt(base64(SHA-256(NFC(pw))))`), dùng chung đăng ký và login.
- **`AuthMailer`** (`auth_mailer.go`) — port gửi thư auth: `SendRegisterCode`, `SendAccountExistsNotice`, `SendPasswordResetCode`. Adapter: `repository/email_auth_mailer.go` trên `pkg/mailer`. Không method nào nhận mật khẩu.

### Input/Output của use-case (`register.go`)
Struct biên của use-case nằm ở `domain` (không phải `usecase`) để `delivery` tham chiếu được mà không phải import concrete type của `usecase`.

- **`RequestRegisterOTPInput`** — `Email`, `Password`, `PhoneCountry`, `PhoneNumber`, `FullName`, `DharmaName`, `OrdinationTempleID`. Mang **toàn bộ** payload vì bước 1 là bước duy nhất nhận nó; bước 2 chỉ gửi lại email + mã.
- **`RequestOTPResult`** — `ExpiresInSeconds`, `ResendAfterSeconds` (từ `config.OTPConfig` qua `OTPService`).
- **`ResendOTPInput`** — `Email`.
- **`ResendOTPResult`** — `ExpiresInSeconds`, `ResendAfterSeconds`, `ResendRemaining` (AC11).
- **`VerifyRegisterOTPInput`** — `Email`, `OTPCode`. Phần còn lại đã nằm trong `PendingRegistration`.
- **`RegisterResult`** — `User *User`, `AccessToken`, `RefreshToken`. Đăng ký xong là đã đăng nhập (AC14).

### Port `RegistrationRepository` (`register.go`)
6 method, thuần register (v7.0 thêm `TempleExists`). Mọi thao tác ghi không được phép quan sát ở trạng thái nửa vời đều là
**một method bọc một transaction**, không phải một bộ primitive để use-case tự ghép — crash giữa
hai lời gọi sẽ để lại pending không OTP, hoặc user không session.

Các method OTP dùng chung (`IssueOTP`, `FindActiveOTP`, …) đã chuyển sang port `OTPRepository` bên
dưới (refactor `otp-mechanism`). Method duy nhất còn đụng `otp_codes` ở đây là
`SavePendingAndIssueOTP`, vì nó phải commit pending + OTP trong **một** transaction — atomicity mà
một `OTPRepository.IssueOTP` đứng riêng không thể biểu đạt.

| Method | Ghi chú |
|---|---|
| `PhoneTaken(ctx, phone)` | `users.phone_number` đã có tài khoản? (AC7). Pending cùng SĐT **không** tính |
| `EmailHasAccount(ctx, email)` | Chỉ để chọn thư ở Step R-8 (AC2), không bao giờ lộ ra response |
| `TempleExists(ctx, id)` | `SELECT EXISTS` trên `temples` — trả `ErrInvalidTemple` sớm (AC8); FK là chốt chặn cuối |
| `SavePendingAndIssueOTP(ctx, pending, otpHash, expiresAt, limits, canReplace)` | Áp send-limit `(email, register)` → UPSERT pending theo `email` (ghi đè, kể cả `account_exists`) → vô hiệu mã cũ → chèn mã mới. Trả `*SendState`. Không ghi `users` |
| `FindPendingRegistration(ctx, email)` | `ErrPendingRegistrationNotFound` / `ErrPendingRegistrationExpired` (use-case gộp cả hai thành Expired, AC12/AC17) |
| `CreateUserAndSession(ctx, pending, otpID, sessionTTL, issueTokens)` | Một transaction: đánh `used_at` cho mã (`WHERE used_at IS NULL`, không được → `ErrOTPExpired`) → INSERT users (`email_verified_at = now()`) → DELETE pending + bộ đếm gửi → INSERT session. UNIQUE SĐT → `ErrPhoneTaken`, FK chùa → `ErrInvalidTemple` (AC18) |
| `DeletePendingRegistration(ctx, pendingID)` | Đường production không cần vì `CreateUserAndSession` xoá pending trong tx của nó |

### Port `OTPRepository` (`otp_repository.go`)
5 method, **không biết gì về `pending_registration`/`users`** (otp-mechanism AC11). Đây là port cơ chế
OTP dùng chung — đăng ký và đặt lại mật khẩu cùng inject. Mọi method scope theo cặp `(email, purpose)`
(email đã chuẩn hoá bởi caller), nên hai purpose của cùng một email không bao giờ chung mã, chung bộ
đếm gửi, hay chung khoá nhập sai.

| Method | Ghi chú |
|---|---|
| `IssueOTP(ctx, email, purpose, otpHash, expiresAt, limits)` | Áp send-limit → vô hiệu mã cũ → chèn mã mới, một transaction. **Không** đụng pending. Trả `*SendState` |
| `ApplySendLimit(ctx, email, purpose, limits)` | Đúng send-limit của `IssueOTP` nhưng không tạo mã (đặt lại mật khẩu cho email không có tài khoản, otp-mechanism AC6) |
| `FindActiveOTP(ctx, email, purpose)` | Hàng đang khoá (để báo hạn khoá), hoặc hàng mới nhất chưa dùng, chưa hết hạn, chưa từng khoá; không có → `ErrOTPInvalid` |
| `ConsumeOTPAttempt(ctx, otpID, max, lockUntil)` | Một `UPDATE … WHERE attempts < max RETURNING attempts`: giữ lượt **trước** khi so hash; lượt chạm `max` đặt `locked_until`. `ok=false` = hết lượt |
| `ReleaseSendLimit(ctx, email, purpose)` | Xoá bộ đếm gửi khi gửi thư lỗi (otp-mechanism AC10) |

Concrete `registrationRepository` (`repository/postgres_registration_repository.go`) implement **cả
hai** interface từ một struct; `NewRegistrationRepository(db)` và `NewOTPRepository(db)` chỉ là hai
cách nhìn cùng một store (cùng chia sẻ SQL helper `applySendLimits`/`invalidateActiveOTP`/`insertOTP`).

### Attestation (`attestation.go`)
Entity + port cho feature attestation (Play Integrity + App Attest), cùng pattern thuần-domain:

- **Entity** `AttestationChallenge` (nonce dùng-một-lần), `AttestationKey` (khoá iOS bền), `Verdict`
  (`Attested`, `Signals`, `Reason` — verdict âm tính là kết quả, **không** phải error), enum
  `Platform` (`android`/`ios`) và `ChallengePurpose` (`attest`/`assert`).
- **Port `AttestationRepository`** (`SaveChallenge`, `FindChallenge`, `MarkChallengeUsed`,
  `SaveKey`, `FindKey`, `UpdateKeyCounter`) — `SaveKey`/`UpdateKeyCounter` nhận thêm `challenge` để
  đóng nonce cùng MỘT tx (atomicity). Implement ở `repository/postgres_attestation_repository.go`.
- **Port `AttestationVerifier`** (`VerifyAndroid`, `VerifyIOSAttestation`, `VerifyIOSAssertion`) —
  seam tới `pkg/attestation`; use-case không import `pkg/attestation` trực tiếp. Adapter ở
  `repository/attestation_verifier.go`.
- **Mock** `MockAttestationRepository` (`mock_attestation_repository.go`) +
  `MockAttestationVerifier` (`mock_attestation_verifier.go`) cho test use-case.

### Callback-port `TokenIssuer` (`register.go`)
```go
type TokenIssuer func(userID, sessionID string) (accessToken, refreshToken string, err error)
```
`sessionID` do repository sinh **trước** khi gọi issuer và chính là giá trị nó lưu vào `user_sessions.session_id`; access token mang nó ở claim `sid` để `logout` tìm đúng session.

JWT phải được mint **giữa transaction** — sau khi `INSERT users` sinh ID, trước khi `INSERT user_sessions` cần token. Tách thành hai round-trip sẽ phá atomicity. Nên `domain` khai một function type, use-case truyền closure bọc `jwtService`, repository gọi nó đúng chỗ trong `tx` mà **không bao giờ import `pkg/jwt`**.

Chỉ dùng pattern này khi có ràng buộc thứ tự/atomicity thật sự buộc logic tầng ngoài chạy *trong lúc* tầng trong đang ghi — không phải để thay thế repository interface nói chung.

### Sentinel error (`errors.go`)
Phần đăng ký, phân loại bằng `errors.Is` (map HTTP ở `delivery/http/errors.go`):

```go
ErrPhoneTaken                                        // 409 PHONE_TAKEN (AC7, AC18)
ErrInvalidEmail / ErrInvalidPhone / ErrInvalidPassword / ErrInvalidFullName / ErrInvalidDharmaName / ErrInvalidTemple  // 400 (AC3–AC8)
ErrOTPInvalid / ErrOTPExpired                        // 400 (AC15)
ErrOTPLocked / ErrOTPResendTooSoon / ErrOTPSendLimited  // 429 (AC9, AC15)
ErrPendingRegistrationNotFound / ErrPendingRegistrationExpired  // 400 PENDING_REGISTRATION_EXPIRED (AC12, AC17)
ErrEmailSendFailed                                   // 500 EMAIL_SEND_FAILED (AC10)
```

Typed error mang thêm dữ liệu mà sentinel trần không chở được: `OTPLockedError{LockedUntil}`,
`OTPResendTooSoonError{RetryAfterSeconds}`, `OTPSendLimitedError{LockedUntil}`. Tất cả `Unwrap()` về
sentinel tương ứng ⇒ `errors.Is` vẫn chạy, `errors.As` lấy được con số.

Cố ý **không** có "email đã được dùng": email đã có tài khoản nhận y hệt email mới (AC2), nên không
sentinel nào phân biệt được hai trường hợp. Ngoại lệ đã chấp nhận duy nhất là `409 PHONE_TAKEN`.

### Mock cho unit test
- **`MockRegistrationRepository`** (`mock_registration_repository.go`) — mock tay `RegistrationRepository` bằng `testify/mock`.
- **`MockOTPRepository`** (`mock_otp_repository.go`) — mock tay `OTPRepository`, phục vụ test `OTPService`.
- **`MockAuthMailer`** (`mock_auth_mailer.go`) — mock tay `AuthMailer`; trả error ở đây là cách test nhánh `ErrEmailSendFailed`.

Cả ba khai `var _ Interface = (*MockX)(nil)` ở đầu file: interface đổi mà mock không cập nhật thì **fail lúc compile**, không đợi tới lúc test tình cờ chạm method mới.

## Cấu hình / dependency
- **Không có config, không đọc biến môi trường.** Package chỉ chứa định nghĩa kiểu dữ liệu và interface.
- Import: thư viện chuẩn (`context`, `errors`, `fmt`, `time`) + `github.com/stretchr/testify/mock` **chỉ trong 2 file `mock_*.go`** (code hỗ trợ test, không phải logic domain).
- Import path: `chjpcoj.com.innerlamp/services/appService/domain`.
- Consumer: `usecase/` (dùng interface + input/result + sentinel), `repository/` (implement `RegistrationRepository` + `OTPRepository`, adapter `AuthMailer`), `delivery/http/` (dùng input/result + sentinel để map status).
- Kiểm tra chiều phụ thuộc: `bash .claude/skills/golang-clean-arch-tdd/scripts/check_dependency_rule.sh services/appService`.

## Lưu ý / lỗi hay gặp
- **Mock nằm cùng package `domain`**, không tách package `mocks/` riêng, và repo **không có** `go:generate`/mockgen/mockery. Thêm/xoá/đổi chữ ký method trong `RegistrationRepository` thì phải sửa `mock_registration_repository.go` **bằng tay trong cùng thay đổi** — không có bước codegen nào để chạy thay.
- **Không tự tạo `TokenIssuer` mặc định trong `domain`.** Nó là closure do use-case cấp; mục đích là để repository sinh token *sau khi* biết user ID, trong cùng transaction với session.
- **`ErrPendingRegistrationNotFound` không được rò ra ngoài dưới dạng 404.** Map thành `OTP_INVALID` ở delivery, nếu không sẽ lộ SĐT nào đang đăng ký dở.
- **`OTPHash`/`PasswordHash` không bao giờ được log.** Hash bcrypt của OTP cũng không: mã 6 số + hash thời gian ngắn vẫn đủ để brute-force ngoài luồng.
- **Không thêm `Device`/`Session` vào đây cho US-001.** Session được tạo trong `CreateUserAndSession`, `RegisterResult` chỉ trả token; bảng `user_sessions` ở story này không có `device_id`.
- `ExpiresAt` của `PendingRegistration` **bằng** TTL của OTP là chủ ý, không phải trùng lặp thừa: hai deadline luôn khớp nhau nên caller gặp cái nào hết hạn cũng kết luận được cả lần đăng ký đã chết.
