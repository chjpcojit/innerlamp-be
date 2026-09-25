# Domain — services/appService/domain

## Mục đích
Tầng entity của Clean Architecture cho appService: struct nghiệp vụ thuần (`User`, `PendingRegistration`, `OTPCode`), interface repository mà `usecase/` phụ thuộc và `repository/` implement, port `OTPSender`, các Input/Result của use-case, và sentinel error dùng chung toàn service.

Package này là **tầng trong cùng** — mọi tầng khác phụ thuộc nó, nó không phụ thuộc tầng nào. Đây không phải quy ước cho vui: `domain` hiện có **zero import ngoài thư viện chuẩn** (cộng `testify/mock` trong các file `mock_*.go`). Ai thêm một import `pkg/*` hay `net/http` vào đây là làm hỏng kiến trúc, không phải "hơi lệch convention".

Phạm vi hiện tại: **US-001 (đăng ký tài khoản, verify OTP qua SMS)**. Chưa có `Device`/`Session` entity — `user_sessions` không có `device_id` ở story này (xem `docs/specs/features/onboarding/design.md` §Known gaps).

## Khi nào dùng / không dùng
- **Dùng khi**: viết use-case mới (cần struct input/result + interface port), viết repository implementation mới (implement interface ở đây), viết unit test (mock nằm sẵn trong package), hoặc cần phân loại lỗi nghiệp vụ bằng `errors.Is`.
- **Không dùng khi**: cần logic nghiệp vụ (đặt ở `usecase/`), cần truy vấn DB thật (đặt ở `repository/`), cần parse/trả HTTP (đặt ở `delivery/http/`).
- **Không** thêm `json:"..."` hay `db:"..."` vào struct ở đây. Hình dạng wire là quyết định của `delivery` (mỗi handler có response struct riêng), hình dạng cột là quyết định của `repository` (SQL). Gắn tag vào entity sẽ ép một hình dạng JSON lên mọi consumer và khiến entity phải sửa mỗi khi một API contract không liên quan đổi.

## API chính

### Entity

- **`User`** (`user.go`) — map bảng `users`. `ID int64`, `Email`, `EmailVerifiedAt *time.Time`, `PhoneNumber`, `PasswordHash`, `DharmaName *string`, `OrdinationPlace *string`, `CreatedAt`, `UpdatedAt`.
  `EmailVerifiedAt` **cố ý NULL** lúc đăng ký: US-001 verify **số điện thoại** bằng OTP, không verify email — cột này ghi nhận một bước chưa tồn tại, không phải bug.
- **`PendingRegistration`** (`pending_registration.go`) — map bảng `pending_registration`. Payload tài khoản (gồm **bcrypt hash** của password, không bao giờ plaintext) nằm tạm giữa bước 1 và bước 2.
  Đây là struct giữ bất biến trung tâm của US-001: **không hàng `users` nào tồn tại ở trạng thái chưa verify**. Một hàng cho mỗi SĐT (`phone_number` UNIQUE — chỗ dựa cho `ON CONFLICT`); `email` cố ý **không** unique ở bảng này, chỉ `users` mới có thẩm quyền về unique email và bước insert cuối tự re-check trong transaction của nó.
  `ExpiresAt` bằng TTL của OTP và được **đẩy lên** mỗi lần resend ⇒ pending hết hạn và OTP hết hạn luôn đi cùng nhau; gặp một cái hết hạn là cả lần đăng ký đã chết, client phải nhập lại email + password.
- **`OTPCode`** + **`OTPPurpose`** (`otp.go`) — map bảng `otp_codes`, dùng chung mọi flow OTP và phân biệt bằng `Purpose`.
  `OTPHash` là **bcrypt**, không phải mã 6 số. Không gian OTP chỉ 10^6 nên hash nhanh (SHA-256) đổ trong vài giây bằng bảng tra sẵn — giá trị sống ngắn không làm hash nhanh trở nên an toàn.
  Hàng OTP tự mang rate limiting (`Attempts`, `ResendCount`, `LockedUntil`) nên AC5 (giới hạn resend) và AC12 (lock sau nhiều lần sai) không cần store thứ hai và sống sót qua restart.
  Bốn hằng `OTPPurposeRegister/Login/ResetPassword/VerifyEmail` khai **đủ ngay từ đầu** để `chk_otp_codes_purpose` không phải `ALTER` khi US-002/US-003 tới. US-001 **chỉ hiện thực** `register`.

### Input/Output của use-case (`register.go`)
Struct biên của use-case nằm ở `domain` (không phải `usecase`) để `delivery` tham chiếu được mà không phải import concrete type của `usecase`.

- **`RequestRegisterOTPInput`** — `Email`, `Password`, `PhoneNumber`, `DharmaName`, `OrdinationPlace`. Mang **toàn bộ** payload vì bước 1 là bước duy nhất nhận nó; bước 2 chỉ gửi lại SĐT + OTP.
- **`RequestOTPResult`** — `ExpiresInSeconds`, `ResendAfterSeconds`. TTL là quyết định của use-case, response là chỗ duy nhất quan sát được nó; để handler tự viết `90`/`60` thì hai bên lệch nhau ngay khi một bên đổi.
- **`ResendOTPInput`** — `PhoneNumber`, `Purpose`. Một endpoint dùng chung mọi flow OTP, `Purpose` chọn flow.
- **`ResendOTPResult`** — `ExpiresInSeconds`, `ResendAfterSeconds`, `ResendRemaining`. Trả số lần resend còn lại để client biết mình sắp chạm trần (AC5).
- **`VerifyRegisterOTPInput`** — `PhoneNumber`, `OTPCode`. Chỉ hai field: phần còn lại đã nằm trong `PendingRegistration`, client không được đổi email/password ở bước này.
- **`RegisterResult`** — `User *User`, `AccessToken`, `RefreshToken`. Đăng ký xong là đã đăng nhập (AC1/AC7).
- **`RegistrationConflicts`** — `PhoneInUsers`, `EmailInUsers`, `EmailInPending` + helper `HasPhoneConflict()` / `HasEmailConflict()`. Là struct chứ không phải nhiều boolean ở nhiều lần gọi vì bước 1 cần mọi câu trả lời từ **một** query; tách ra thì giữa hai lần đọc có khe hở dữ liệu đã đổi. `EmailInPending` (v5.1) chỉ tính hàng pending của SĐT **khác**; pending của chính SĐT đang gọi không phải trùng (request-otp ghi đè nó).

### Port `RegistrationRepository` (`register.go`)
5 method, thuần register. Mọi thao tác ghi không được phép quan sát ở trạng thái nửa vời đều là
**một method bọc một transaction**, không phải một bộ primitive để use-case tự ghép — crash giữa
hai lời gọi sẽ để lại pending không OTP, hoặc user không session.

Các method OTP dùng chung (`IssueOTP`, `FindActiveOTP`, …) đã chuyển sang port `OTPRepository` bên
dưới (refactor `otp-mechanism`). Method duy nhất còn đụng `otp_codes` ở đây là
`SavePendingAndIssueOTP`, vì nó phải commit pending + OTP trong **một** transaction — atomicity mà
một `OTPRepository.IssueOTP` đứng riêng không thể biểu đạt.

| Method | Ghi chú |
|---|---|
| `FindConflicts(ctx, phoneNumber, email)` | Một query, ba `EXISTS`: `users.phone_number`, `users.email`, và `pending_registration.email` của SĐT **khác** — AC6 |
| `SavePendingAndIssueOTP(ctx, pending, otpHash, expiresAt, limits)` | Áp send-limit → UPSERT pending theo `phone_number` (ghi đè) → vô hiệu OTP cũ → chèn OTP mới. Trả `*SendState`. Không ghi `users` |
| `FindPendingRegistration(ctx, phoneNumber)` | `ErrPendingRegistrationNotFound` nếu không có; `ErrPendingRegistrationExpired` nếu quá hạn |
| `CreateUserAndSession(ctx, pending, otpID, sessionTTL, issueTokens)` | Một transaction: re-check email → INSERT users (`email_verified_at` NULL) → DELETE pending → UPDATE OTP `used_at` → INSERT session. JWT mint **trong** transaction — AC2/AC3/AC7 |
| `DeletePendingRegistration(ctx, pendingID)` | Export theo `design.md`; đường production không cần vì `CreateUserAndSession` xoá pending trong tx của nó |

### Port `OTPRepository` (`otp_repository.go`)
5 method, **không biết gì về `pending_registration`/`users`**. Đây là port cơ chế OTP dùng chung —
feature OTP tương lai (`reset_password`, login bằng OTP, `verify_email`) chỉ inject port này, không
kéo theo `FindConflicts`/`CreateUserAndSession`. Mọi method nhận `purpose` và scope theo cặp
`(phone_number, purpose)`, nên hai purpose cùng một SĐT không bao giờ chung hàng `otp_codes`, chung
bộ đếm gửi, hay chung lock nhập sai.

| Method | Ghi chú |
|---|---|
| `IssueOTP(ctx, phoneNumber, purpose, otpHash, expiresAt, limits)` | Áp send-limit → vô hiệu OTP cũ → chèn OTP mới. **Không** đụng payload pending (v5.1). Trả `*SendState` — AC5 |
| `FindActiveOTP(ctx, phoneNumber, purpose)` | Hàng mới nhất **chưa dùng và chưa lock**; không có → `ErrOTPInvalid` |
| `IncrementOTPAttempts(ctx, otpID)` | Trả tổng mới; ngưỡng lock do use-case quyết, không nhét vào repository — AC4/AC12 |
| `LockOTP(ctx, otpID, lockedUntil)` | Set **cả** `locked_until` **và** `used_at` — AC12 |
| `ReleaseSendLimit(ctx, phoneNumber, purpose)` | Xoá bộ đếm gửi; `CreateUserAndSession` gọi inline trong tx khi đăng ký xong — AC5 |

Concrete `registrationRepository` (`repository/postgres_registration_repository.go`) implement **cả
hai** interface từ một struct; `NewRegistrationRepository(db)` và `NewOTPRepository(db)` chỉ là hai
cách nhìn cùng một store (cùng chia sẻ SQL helper `applySendLimits`/`invalidateActiveOTP`/`insertOTP`).

### Port `OTPSender` (`register.go`)
Một method duy nhất: `SendOTP(ctx, phoneNumber, otpCode string, purpose OTPPurpose) error`. Một method cho mọi purpose vì resend dùng chung đường gửi với lần gửi đầu, và US-002/US-003 sẽ gọi đúng method này với `purpose` khác. Use-case phụ thuộc port này thay vì `pkg/sms` để `domain` không dính hạ tầng cụ thể và test khẳng định được "đã gửi gì" mà không cần mạng.

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
14 sentinel, phân loại bằng `errors.Is`:

```go
ErrPhoneTaken / ErrEmailTaken            // 409, AC6
ErrInvalidPhone / ErrInvalidEmail / ErrInvalidPassword   // 400, AC7
ErrOTPInvalid / ErrOTPExpired            // 400, AC4
ErrOTPLocked / ErrOTPResendTooSoon / ErrOTPResendExhausted  // 429, AC5/AC12
ErrPendingRegistrationNotFound / ErrPendingRegistrationExpired
ErrUnsupportedOTPPurpose                 // 400
ErrSMSSendFailed                         // 500 SMS_SEND_FAILED
```

Hai typed error mang thêm dữ liệu mà sentinel trần không chở được (sentinel là giá trị dùng chung, bất biến; deadline/ số giây là của từng hàng):
```go
type OTPLockedError struct{ LockedUntil time.Time }
type OTPResendTooSoonError struct{ RetryAfterSeconds int }
```
Cả hai `Unwrap()` về sentinel tương ứng ⇒ `errors.Is(err, ErrOTPLocked)` vẫn chạy, `errors.As` lấy được con số.

Cố ý **không** có cặp "email không tồn tại" / "SĐT không tồn tại": verify OTP cho SĐT không có pending trả `ErrOTPInvalid`, không trả lỗi not-found riêng — lỗi phân biệt được sẽ cho phép dò xem SĐT nào đang đăng ký dở (cùng lý do `ErrInvalidCredentials` tồn tại ở luồng login).

`ErrPendingRegistrationNotFound` cũng **không** được map thành 404 ở delivery, mà thành 400 `OTP_INVALID`.

### Mock cho unit test
- **`MockRegistrationRepository`** (`mock_registration_repository.go`) — mock tay `RegistrationRepository` bằng `testify/mock`.
- **`MockOTPRepository`** (`mock_otp_repository.go`) — mock tay `OTPRepository`, phục vụ test `OTPService`.
- **`MockOTPSender`** (`mock_otp_sender.go`) — mock tay `OTPSender`; trả error ở đây là cách test nhánh `ErrSMSSendFailed`.

Cả ba khai `var _ Interface = (*MockX)(nil)` ở đầu file: interface đổi mà mock không cập nhật thì **fail lúc compile**, không đợi tới lúc test tình cờ chạm method mới.

## Cấu hình / dependency
- **Không có config, không đọc biến môi trường.** Package chỉ chứa định nghĩa kiểu dữ liệu và interface.
- Import: thư viện chuẩn (`context`, `errors`, `fmt`, `time`) + `github.com/stretchr/testify/mock` **chỉ trong 2 file `mock_*.go`** (code hỗ trợ test, không phải logic domain).
- Import path: `chjpcoj.com.innerlamp/services/appService/domain`.
- Consumer: `usecase/` (dùng interface + input/result + sentinel), `repository/` (implement `RegistrationRepository` + `OTPRepository`, adapter `OTPSender`), `delivery/http/` (dùng input/result + sentinel để map status).
- Kiểm tra chiều phụ thuộc: `bash .claude/skills/golang-clean-arch-tdd/scripts/check_dependency_rule.sh services/appService`.

## Lưu ý / lỗi hay gặp
- **Mock nằm cùng package `domain`**, không tách package `mocks/` riêng, và repo **không có** `go:generate`/mockgen/mockery. Thêm/xoá/đổi chữ ký method trong `RegistrationRepository` thì phải sửa `mock_registration_repository.go` **bằng tay trong cùng thay đổi** — không có bước codegen nào để chạy thay.
- **Không tự tạo `TokenIssuer` mặc định trong `domain`.** Nó là closure do use-case cấp; mục đích là để repository sinh token *sau khi* biết user ID, trong cùng transaction với session.
- **`ErrPendingRegistrationNotFound` không được rò ra ngoài dưới dạng 404.** Map thành `OTP_INVALID` ở delivery, nếu không sẽ lộ SĐT nào đang đăng ký dở.
- **`OTPHash`/`PasswordHash` không bao giờ được log.** Hash bcrypt của OTP cũng không: mã 6 số + hash thời gian ngắn vẫn đủ để brute-force ngoài luồng.
- **Không thêm `Device`/`Session` vào đây cho US-001.** Session được tạo trong `CreateUserAndSession`, `RegisterResult` chỉ trả token; bảng `user_sessions` ở story này không có `device_id`.
- `ExpiresAt` của `PendingRegistration` **bằng** TTL của OTP là chủ ý, không phải trùng lặp thừa: hai deadline luôn khớp nhau nên caller gặp cái nào hết hạn cũng kết luận được cả lần đăng ký đã chết.
