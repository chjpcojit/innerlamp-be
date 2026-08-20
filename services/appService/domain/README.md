# Domain — services/appService/domain

## Mục đích
Tầng entity của Clean Architecture cho appService: định nghĩa các struct nghiệp vụ thuần (User, Session, Sutra, ...), interface repository (`UserRepository`, `SutraRepository`) mà `usecase/` phụ thuộc vào và `repository/` implement, cùng các domain error sentinel dùng chung toàn service.

## Khi nào dùng / không dùng
- Dùng khi: cần tham chiếu struct nghiệp vụ (User, Device, Session, Sutra, PasswordResetToken, các Input/Result của use-case) hoặc interface repository để viết use-case mới, viết implementation repository mới, hoặc mock trong unit test.
- Không dùng khi: cần logic nghiệp vụ (đặt trong `usecase/`), cần truy vấn DB thật (đặt trong `repository/`), hoặc cần xử lý HTTP/gRPC request-response (đặt trong `delivery/`) — package này không được import bất kỳ tầng nào trong số đó.

## API chính

### Entity struct
- **`User`** (user.go) — map bảng `users`: `ID`, `Email`, `PasswordHash`, `PhoneNumber`, `DharmaName`, `OrdinationPlace`, `AvatarURL`, `CreatedAt`, `UpdatedAt`.
- **`Device`** (user.go) — map bảng `user_devices`, một dòng cho mỗi thiết bị user đã đăng nhập/đăng ký.
- **`Session`** (user.go) — map bảng `user_sessions`, giữ cặp access/refresh token đã cấp cho một thiết bị (`TokenVersion`, `IsActive`, `IsRevoked`, `ExpiresAt`, ...).
- **`Sutra`** (content.go) — map bảng `sutras` (docs/db/content-erd.md): `Slug`, `Title`, `Sect`, `Language`, `Body`, `AudioURL`, `Version`, ...
- **`PasswordResetToken`** (password_reset.go) — map bảng `password_reset_tokens`; chỉ lưu `TokenHash` (SHA-256), không bao giờ lưu token gốc.

### Input/Result cho use-case
- **`RegisterInput`**, **`RegisterResult`**, **`DeviceInfo`**, **`TokenIssuer`** (register.go) — input đăng ký (US-001) và kết quả trả về (user + access/refresh token, vì đăng ký xong tự động login theo AC1/AC7). `TokenIssuer` là callback do use-case truyền xuống repository để sinh token *sau khi* biết user ID, trong cùng transaction tạo device/session.
- **`LoginInput`**, **`LoginResult`** (login.go) — input đăng nhập (US-002); `LoginResult` là alias (`= RegisterResult`) vì spec yêu cầu response shape giống hệt register.
- **`ForgotPasswordInput`**, **`ResetPasswordInput`** (password_reset.go) — input cho flow quên/đặt lại mật khẩu (US-003).
- **`ProfileUpdateInput`** (profile.go) — input cập nhật hồ sơ (US-004 AC2/AC5); field rỗng nghĩa là xoá giá trị cột đó, không có sentinel "không đổi".
- **`ListSutrasInput`** (content.go) — input liệt kê kinh (US-005); `UpdatedSince` zero-value = full sync, non-zero = delta sync; `Sect` rỗng = không lọc tông phái.

### `UserRepository` (register.go)
Interface mà `usecase/` phụ thuộc và `repository/` (Postgres) implement, bao gồm cả nghiệp vụ đăng ký/đăng nhập/reset password/hồ sơ:
```go
type UserRepository interface {
    Register(ctx context.Context, user *User, device *Device, issueTokens TokenIssuer) (*Session, error)
    FindByEmail(ctx context.Context, email string) (*User, error)
    CreateSession(ctx context.Context, userID int64, device *Device, issueTokens TokenIssuer) (*Session, error)
    CreatePasswordResetToken(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) (*PasswordResetToken, error)
    ResetPassword(ctx context.Context, tokenHash, newPasswordHash string) error
    FindByID(ctx context.Context, userID int64) (*User, error)
    UpdateProfile(ctx context.Context, userID int64, input ProfileUpdateInput) (*User, error)
    UpdateAvatarURL(ctx context.Context, userID int64, avatarURL string) (*User, error)
}
```
`FindByEmail`/`FindByID` trả `ErrUserNotFound` nếu không có user. `ResetPassword` trả `ErrResetTokenInvalid` nếu token thiếu/hết hạn/đã dùng, và phải atomically revoke mọi session đang active của account (US-003 AC3).

### `SutraRepository` (content.go)
```go
type SutraRepository interface {
    ListSutras(ctx context.Context, input ListSutrasInput) ([]*Sutra, error)
    GetSutraBySlug(ctx context.Context, slug string) (*Sutra, error)
}
```
`ListSutras` trả kết quả sắp theo `updated_at` tăng dần để caller dùng dòng cuối làm cursor cho lần delta-sync kế tiếp. `GetSutraBySlug` trả `ErrSutraNotFound` nếu slug không tồn tại.

### Mock cho unit test
- **`MockUserRepository`** (mock_user_repository.go) — mock `UserRepository` bằng `github.com/stretchr/testify/mock`, dùng trong test của `usecase/` (register/login/forgot-reset-password/profile).
- **`MockSutraRepository`** (mock_sutra_repository.go) — mock `SutraRepository`, cùng kiểu testify/mock, dùng trong test use-case liên quan nội dung kinh.

### Domain error sentinel (errors.go)
```go
var (
    ErrEmailTaken         = errors.New("email already in use")
    ErrPhoneTaken         = errors.New("phone number already in use")
    ErrInvalidEmail       = errors.New("invalid email format")
    ErrInvalidPhone       = errors.New("invalid phone number format, expected E.164 e.g. +84...")
    ErrInvalidPassword    = errors.New("password must be at least 8 characters")
    ErrUserNotFound       = errors.New("user not found")
    ErrInvalidCredentials = errors.New("email hoặc mật khẩu không đúng")
    ErrResetTokenInvalid  = errors.New("reset token is invalid or has expired")
    ErrSutraNotFound      = errors.New("sutra not found")
)
```
Dùng `errors.Is` để phân biệt. Lưu ý `ErrUserNotFound` là lỗi nội bộ repository — use-case login phải map nó thành `ErrInvalidCredentials` trước khi trả ra ngoài, để không lộ email nào tồn tại hay không (US-002 AC3, chống user-enumeration).

## Cấu hình / dependency
- Package này **không phụ thuộc** vào bất kỳ tầng nào khác của appService (`usecase/`, `repository/`, `delivery/`) — đã kiểm tra import ở đầu từng file: chỉ `context`, `time`, `errors` (thư viện chuẩn) và `github.com/stretchr/testify/mock` (chỉ trong 2 file `mock_*.go`, phục vụ test). Đây là nguyên tắc bắt buộc của tầng entity trong Clean Architecture — dependency luôn hướng vào trong (`usecase`/`repository`/`delivery` phụ thuộc `domain`, chiều ngược lại không được xảy ra).
- Không có config, không đọc biến môi trường — package chỉ chứa định nghĩa kiểu dữ liệu và interface.

## Lưu ý / lỗi hay gặp
- `TokenIssuer` là closure do use-case cung cấp, không phải field cấu hình — không tự tạo `TokenIssuer` mặc định trong `domain`, vì mục đích của nó là để repository sinh token *sau khi* có user ID, trong cùng transaction với việc tạo device/session (register.go).
- `LoginResult` chỉ là type alias của `RegisterResult` (`= RegisterResult`, không phải `RegisterResult` mới) — sửa field của `RegisterResult` sẽ ảnh hưởng cả login lẫn register, cân nhắc trước khi đổi shape.
- `ProfileUpdateInput` không có sentinel "giữ nguyên giá trị cũ" — field rỗng luôn được ghi đè, kể cả khi ý định là "không đổi field này". Use-case gọi `UpdateProfile` phải tự đảm bảo giá trị truyền vào đúng ý định trước khi gọi.
- `ErrUserNotFound` không được để lộ ra khỏi `usecase/` login — phải map thành `ErrInvalidCredentials` ở tầng use-case, không phải ở `domain` hay `repository`.
- `PasswordResetToken.TokenHash` là SHA-256 của token gốc — `domain`/`repository` không bao giờ nhận hay lưu token dạng plaintext, việc hash phải làm ở use-case trước khi gọi `CreatePasswordResetToken`/`ResetPassword`.
- 2 file `mock_*.go` nằm cùng package `domain` (không tách package `mocks` riêng) — khi thêm method mới vào `UserRepository`/`SutraRepository`, phải cập nhật mock tương ứng thủ công, không có `go:generate` tự sinh trong các file đã đọc.
