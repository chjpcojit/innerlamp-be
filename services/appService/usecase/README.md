# Usecase — services/appService/usecase

## Mục đích
Tầng nghiệp vụ của Clean Architecture cho appService: các quy tắc đăng ký tài khoản US-001 nằm giữa HTTP và database. Cơ chế OTP dùng chung (sinh mã, hash, gửi, verify, đếm sai, khoá) nằm trong `OTPService` (`otp_service.go`) — không biết gì về register — còn `RegisterUsecase` giữ phần nghiệp vụ riêng của register với ba method ứng với ba bước của story: `RequestOTP`, `ResendOTP`, `VerifyOTP`.

Bất biến mà package này tồn tại để bảo vệ: **không hàng `users` nào được tạo trước khi số điện thoại được verify**. Đó là lý do bước 1 chỉ ghi vào `pending_registration` + `otp_codes`, và lý do lệnh insert account nằm *bên trong* transaction của bước 2 — không tồn tại khoảng thời gian nào quan sát được một tài khoản đăng ký dở.

Package import `domain` (port, input/result, sentinel) cộng các abstraction dùng lại được trong `pkg/` (`jwt` để ký token, `logging` để quan sát, `validation` để kiểm hình dạng). **Không** import `repository` hay `delivery`: repository Postgres thật được tiêm vào dưới dạng `domain.RegistrationRepository`, còn hình dạng HTTP của một lỗi được quyết định ở tầng ngoài cùng.

## Khi nào dùng / không dùng
- **Dùng khi**: thêm một bước nghiệp vụ mới cho luồng đăng ký, hoặc một use-case mới (US-002 login, US-003 reset password) — mỗi use-case một file `*_usecase.go` + `*_usecase_test.go` cùng cặp.
- **Không dùng khi**: cần truy vấn SQL (đặt ở `repository/`), cần parse/trả HTTP (đặt ở `delivery/http/`), cần định nghĩa kiểu dữ liệu biên (đặt ở `domain/`).
- **Không** đặt `json`/`db` tag ở đây, và **không** trả thẳng `domain` entity ra ngoài — handler tự khai response struct của nó.

## API chính

### `NewOTPService(repo, sender, logger, policy) *OTPService`
Cơ chế OTP dùng chung (refactor `otp-mechanism`). Dựng một lần ở composition root, inject chung cho mọi
flow OTP hiện tại và tương lai.

| Tham số | Kiểu | Vai trò |
|---|---|---|
| `repo` | `domain.OTPRepository` | port 5 method generic (`IssueOTP`, `FindActiveOTP`, `IncrementOTPAttempts`, `LockOTP`, `ReleaseSendLimit`) |
| `sender` | `domain.OTPSender` | port một method; adapter SMS nằm ở `repository/sms_otp_sender.go` |
| `logger` | `logging.Logger` | quan sát (ghi log lỗi ghi attempt/lock) |
| `policy` | `config.OTPConfig` | TTL, cooldown, trần gửi, số lần sai, thời gian lock |

Method công khai: `Generate()` (mã + hash), `Send()` (gửi, wrap `ErrSMSSendFailed`),
`Issue()` (Generate → `IssueOTP` → Send), `Verify()` (FindActiveOTP → lock-check → so hash → đếm/khoá),
`EnsureNotLocked()` (chặn cấp mã mới khi số đang bị khoá nhập sai), cùng các accessor
`CodeExpiry`/`SendLimits`/`ExpiresInSeconds`/`ResendAfterSeconds` cho caller cần tự persist OTP qua
method riêng (`register` dùng cho `SavePendingAndIssueOTP`).

### `NewRegisterUsecase(registrations, otpService, jwtService, logger, sessionTTL, pendingRegistrationTTL) *RegisterUsecase`
Sáu tham số, không cái nào là concrete type của tầng ngoài:

| Tham số | Kiểu | Vai trò |
|---|---|---|
| `registrations` | `domain.RegistrationRepository` | port 5 method register-only; implementation Postgres nằm ở `repository/` |
| `otpService` | `*OTPService` | cơ chế OTP dùng chung, dựng sẵn ở composition root |
| `jwtService` | `jwt.Service` | mint access/refresh token |
| `logger` | `logging.Logger` | quan sát |
| `sessionTTL` | `time.Duration` | vòng đời hàng `user_sessions` |
| `pendingRegistrationTTL` | `time.Duration` | vòng đời payload `pending_registration` — vẫn là mối quan tâm của register, không chuyển sang `OTPService` |

`sessionTTL` và `pendingRegistrationTTL` **truyền vào chứ không hardcode**: chúng đến từ
`JWTConfig.RefreshTokenExpiration` và `OTPConfig.PendingRegistrationTTL`, nên hạn lưu trong DB và hạn
ghi trong token/luồng là cùng một con số do cấu trúc — hai nguồn riêng sẽ lệch nhau. Policy OTP còn lại
(TTL code, cooldown, trần gửi, lock) đã chuyển sang `OTPService`, không còn là field của `RegisterUsecase`.

### `RequestOTP(ctx, domain.RequestRegisterOTPInput) (*domain.RequestOTPResult, error)`
Bước 1. Thứ tự thực thi, và lý do từng chỗ:

1. `NormalizeEmail` → `NormalizePhone` → kiểm email hợp lệ, phone hợp lệ, `len(password) >= 8`. Chạy trên giá trị **đã chuẩn hoá** (đúng thứ sẽ được lưu) và chạy **trước truy vấn đầu tiên** — request sai hình dạng không tốn round-trip nào. Sai → `ErrInvalidEmail` / `ErrInvalidPhone` / `ErrInvalidPassword`.
2. `FindConflicts` → **SĐT trước, email sau**. SĐT là định danh chính của tài khoản, nên khi cả hai đều bị chiếm thì người gọi được nghe về SĐT.
3. `OTPService.EnsureNotLocked` → chặn nếu số đang bị lock nhập sai (AC12). Cooldown/trần gửi (AC5) do repository áp trong `SavePendingAndIssueOTP` — lock nhập sai phải chặn verify, lock gửi thì không, nên hai lock nằm tách nhau.
4. `bcrypt.GenerateFromPassword` (cost mặc định) cho **password**. Hash OTP do `OTPService.Generate` làm. Chỉ hash rời khỏi hàm này; plaintext chỉ còn trong stack frame hiện tại.
5. `SavePendingAndIssueOTP` — một transaction (pending + OTP).
6. `OTPService.Send` — **sau khi** transaction commit.

### `ResendOTP(ctx, domain.ResendOTPInput) (*domain.ResendOTPResult, error)`
`purpose != register` → `ErrUnsupportedOTPPurpose` ngay, không chạm DB. Ngược lại: `FindPendingRegistration`
→ `OTPService.EnsureNotLocked` (lock nhập sai) → `OTPService.Issue(register)` (= Generate → `IssueOTP` →
Send). Cooldown/trần gửi do `IssueOTP` áp trong transaction của nó, nên một lần gửi bị chặn không ghi gì.

Một lần resend **không** được đổi payload tài khoản: email và password đã chốt ở bước 1, người gọi muốn giá trị khác thì làm lại từ đầu. Hạn của pending được đẩy lên cùng TTL với code mới, giữ nguyên tính chất "hai deadline luôn hết hạn cùng nhau".

### `VerifyOTP(ctx, domain.VerifyRegisterOTPInput) (*domain.RegisterResult, error)`
Bước 2. Mọi nhánh trước lệnh insert account đều là ngõ cụt — code sai, code hết hạn, số bị lock, không có pending — tất cả đều để `users` nguyên vẹn.

Thứ tự: `FindPendingRegistration` → `OTPService.Verify(ctx, phone, register, code)`, trong đó cơ chế dùng
chung làm `FindActiveOTP(register)` → kiểm lock (**trước khi so hash**, để kẻ đoán không kéo dài được lock
cũng không đoán miễn phí) → `bcrypt.CompareHashAndPassword`:

- **sai** → cơ chế này đếm: `IncrementOTPAttempts`, và khi `attempts >= MaxAttempts` thì
  `LockOTP(now + LockDuration)`. Luôn trả `ErrOTPInvalid` và **không** nói còn mấy lần thử — chính con số đó
  là thứ người brute-force 10^6 keyspace dùng để điều nhịp.
- **đúng** → trả `*OTPCode` đã verify; register completion `CreateUserAndSession(ctx, pending, otp.ID,
  sessionTTL, tokenIssuer(pending))`.

### `tokenIssuer(pending) domain.TokenIssuer`
Closure mà repository gọi **bên trong** transaction của nó: sau khi `INSERT users` sinh ID, trước khi `INSERT user_sessions` cần token. Sign sau transaction sẽ để lại một account đã commit mà không có session dùng được nếu bước hai fail. Repository gọi nó mà **không** import `pkg/jwt` — nó chỉ thấy function type `domain.TokenIssuer`.

Access token mang `(userID, email, "user", {"sid": sessionID})`; refresh token mang `(userID, nil)`.

### Hằng nội bộ
```go
userRole          = "user"
minPasswordLength = 8
```

Policy OTP (TTL code, cooldown, trần gửi, số lần sai, thời gian lock, TTL pending) là **cấu hình**
`config.OTPConfig`, do `OTPService` nắm và truyền vào qua constructor — không phải hằng ở đây. Các con số này
là thứ dev environment rút ngắn để test hết hạn và production kéo dài dưới tải, nên chúng là cấu hình chứ
không phải "quyết định nghiệp vụ". Delivery đọc lại TTL qua `RequestOTPResult`/`ResendOTPResult` thay vì tự
viết `90`/`60` — handler hardcode sẽ lệch ngay khi policy đổi.

`minPasswordLength` đếm **byte, không phải rune**: quy tắc nói về lượng kẻ tấn công phải đoán, mà một ký tự đa byte khiến việc đoán tốn hơn một byte — đếm rune sẽ cho lọt secret ngắn hơn.

### `generateOTP() string` (trong `otp_service.go`)
Trả hằng `"111111"` (US-001 technical notes: OTP cố định ở dev). Đây là **placeholder có chủ ý, không phải
thiếu sót**, và được cô lập trong đúng một hàm (nay nằm ở `OTPService.Generate`) để khi chuyển sang mã 6 số
ngẫu nhiên (`crypto/rand`) thì không phải sửa chỗ nào khác. Mọi caller coi giá trị trả về là opaque.

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
- Import: `domain` (port/input/sentinel), `pkg/jwt`, `pkg/logging`, `pkg/validation`, `config` (chỉ để *đặt tên* kiểu `OTPConfig`/`AttestationConfig`), `golang.org/x/crypto/bcrypt`, thư viện chuẩn.
- Import path: `chjpcoj.com.innerlamp/services/appService/usecase`.
- Consumer: `delivery/http/` (khai interface consumer-side là subset của use-case) và `cmd/main.go` (dựng concrete).
- Kiểm tra chiều phụ thuộc: `bash .claude/skills/golang-clean-arch-tdd/scripts/check_dependency_rule.sh services/appService`.

## Test
`otp_service_test.go` — test cơ chế OTP dùng chung (AC1, AC2, AC8), mock `domain.MockOTPRepository` + `domain.MockOTPSender`, **không** DB, không mạng.

`register_usecase_test.go` — table-driven, `testify/assert` + `require`, mock là `domain.MockRegistrationRepository` + `domain.MockOTPRepository` + `domain.MockOTPSender` + `jwt.MockService` + `logging.MockLogger`. **Không** DB, không mạng.

`attestation_usecase_test.go` — table-driven, mock `domain.MockAttestationRepository` + `domain.MockAttestationVerifier`, phủ challenge/verify (android pass/fail, upstream, ios attest pass/fail) và assertion (counter tăng/không tăng/unknown key/challenge invalid/chữ ký sai). **Không** DB, không mạng.

Test ghim cả những thứ dễ trôi mà không phải hằng số: thứ tự `save` trước `send` (OTP gửi sau commit), OTP đi ra khỏi tầng này dưới dạng bcrypt hash chứ không phải mã, `IncrementOTPAttempts` cộng dồn đúng và lock chỉ bật ở lần sai thứ ba, và `resend_count` **+1** chứ không reset.

Comment đầu file test liệt kê mapping AC → tên test; đọc nó trước khi thêm test mới để không tạo khoảng trống hoặc trùng lặp.

## Lưu ý / lỗi hay gặp
- **Trần resend không áp lên `RequestOTP`.** `resend_count` kế thừa từ hàng cũ, nên nếu chặn ở đây thì một số đã dùng hết 3 lần resend sẽ **vĩnh viễn không đăng ký được** — không còn đường quay lại code hoạt động. `request-otp` là bắt đầu lại từ bước 1, và một lần bắt đầu lại thì được phép bắt đầu lại; chỉ cooldown (theo thời gian) còn hiệu lực. Lock thì **có** áp — AC12 không có ngoại lệ cho endpoint này.
- **SMS gửi sau commit, và lỗi SMS trả về lỗi.** Hàng OTP lúc đó đã commit, nhưng một đăng ký mà code không bao giờ tới không phải là thành công — trả 200 sẽ để người dùng chờ một tin nhắn không đến. Hệ quả đã chấp nhận: lần thử lại ngay sẽ **đụng cooldown** thay vì âm thầm thành công.
- **`ErrPendingRegistrationNotFound` không được rò ra ngoài dạng 404.** Cả "không có pending" lẫn "pending hết hạn" đều map về cùng một phản hồi không tiết lộ gì ở delivery — lỗi phân biệt được sẽ cho phép dò xem SĐT nào đang đăng ký dở.
- **Không log `PasswordHash`/`OTPHash`, và cũng không log mã OTP hay plaintext password.** Hash bcrypt của một mã 6 số sống ngắn vẫn đủ để brute-force ngoài luồng.
- **Đừng thêm nil-guard cho `result.User`** ở `VerifyOTP` để đối phó một mock trả `RegisterResult{}` rỗng — đó là mock vi phạm contract của port (`CreateUserAndSession` luôn set `User`). Sửa stub, không nới code.
- Test có thể panic với `mock: I don't know what to return` khi impl gọi một method chưa được stub. Đó thường là **test thiếu kỳ vọng**, không phải bug impl — nhưng cũng có lần là bug thật (`checkRequestThrottle` từng quên nhánh lock, và test bắt đúng). Đọc method bị gọi trước khi kết luận.
