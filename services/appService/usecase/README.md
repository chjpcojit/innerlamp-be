# Usecase — services/appService/usecase

## Mục đích
Tầng usecase (interactor) của Clean Architecture cho appService: chứa business logic thuần cho từng user story onboarding (đăng ký, đăng nhập, quên/đặt lại mật khẩu, hồ sơ, avatar) và nội dung (danh sách/chi tiết kinh điển), điều phối qua các interface repository định nghĩa ở `domain` — không tự tay chạm DB/storage/mailer cụ thể.

## Khi nào dùng / không dùng
- Dùng khi: cần gọi một hành vi nghiệp vụ đã có (register, login, forgot/reset password, get/update profile, update avatar, list/get sutra) từ tầng delivery (HTTP/gRPC handler) hoặc từ test — usecase là entrypoint chính thức của logic, không phải gọi thẳng repository.
- Không dùng khi: cần thêm field/route mới thuần thuộc tầng transport (parse HTTP body, format response) — việc đó thuộc `delivery/`; hoặc cần logic truy vấn DB cụ thể (SQL, transaction) — việc đó thuộc `repository/` (implement các interface trong `domain`), usecase chỉ được gọi qua interface, không import package `repository` trực tiếp.

## API chính

### `NewRegisterUsecase(users domain.UserRepository, jwtService jwt.Service, logger logging.Logger) *RegisterUsecase`
`RegisterUsecase.Register(ctx, domain.RegisterInput) (*domain.RegisterResult, error)` — US-001. Validate email/số điện thoại (E.164)/độ dài mật khẩu, hash password bằng bcrypt, build `domain.User` + `domain.Device`, rồi gọi `users.Register(...)` với một `TokenIssuer` callback (dùng `jwtService` sinh access + refresh token) để repository có thể mint token bên trong cùng transaction tạo user/device/session. Trả `*domain.RegisterResult` (đã login luôn sau khi đăng ký — AC7).
```go
uc := usecase.NewRegisterUsecase(userRepo, jwtService, logger)
result, err := uc.Register(ctx, domain.RegisterInput{
    Email: "a@b.com", Password: "12345678", PhoneNumber: "+84912345678",
})
```

### `NewLoginUsecase(users domain.UserRepository, jwtService jwt.Service, logger logging.Logger) *LoginUsecase`
`LoginUsecase.Login(ctx, domain.LoginInput) (*domain.LoginResult, error)` — US-002. Tìm user theo email, so khớp password bằng `bcrypt.CompareHashAndPassword`, rồi tạo session mới cho thiết bị (theo device fingerprint) qua `users.CreateSession(...)` với cùng cơ chế `TokenIssuer` như Register.

### `NewPasswordResetUsecase(users domain.UserRepository, m mailer.Mailer, logger logging.Logger) *PasswordResetUsecase`
US-003, gồm 2 method:
- `ForgotPassword(ctx, domain.ForgotPasswordInput) error` — sinh reset token ngẫu nhiên (32 byte, `crypto/rand`), lưu **hash** của token (SHA-256) qua `users.CreatePasswordResetToken`, gửi token thô qua `mailer.SendPasswordResetEmail`. Luôn trả `nil` kể cả khi email không tồn tại hoặc gửi mail thất bại.
- `Reset(ctx, domain.ResetPasswordInput) error` — validate độ dài mật khẩu mới, hash bcrypt, gọi `users.ResetPassword(ctx, hash(token), newHash)` (repository chịu trách nhiệm validate token còn hạn/chưa dùng và revoke mọi session).
```go
uc := usecase.NewPasswordResetUsecase(userRepo, mailerClient, logger)
_ = uc.ForgotPassword(ctx, domain.ForgotPasswordInput{Email: "a@b.com"})
err := uc.Reset(ctx, domain.ResetPasswordInput{Token: rawToken, NewPassword: "newpass123"})
```

### `NewProfileUsecase(users domain.UserRepository, logger logging.Logger) *ProfileUsecase`
US-004 AC1/AC2/AC5:
- `GetProfile(ctx, userID int64) (*domain.User, error)` — proxy thẳng `users.FindByID`.
- `UpdateProfile(ctx, userID int64, input domain.ProfileUpdateInput) (*domain.User, error)` — ghi đè `dharma_name`/`ordination_place` theo đúng giá trị input (không merge với dữ liệu cũ).

### `NewAvatarUsecase(users domain.UserRepository, storage AvatarStorage, logger logging.Logger) *AvatarUsecase`
`AvatarUsecase.UpdateAvatar(ctx, userID int64, reader io.Reader, contentType string) (*domain.User, error)` — US-004 AC3. Upload ảnh lên object storage với key cố định `avatars/{userID}` (upload lại sẽ tự ghi đè, không tạo file rác), rồi lưu URL trả về qua `users.UpdateAvatarURL`. `AvatarStorage` là interface hẹp (`Upload(ctx, objectName, reader, contentType) (url string, err error)`) khai báo ngay trong package `usecase`, không phải `pkg/storage.Client` đầy đủ.

### `NewGetSutraUsecase(sutras domain.SutraRepository, logger logging.Logger) *GetSutraUsecase`
`GetSutraBySlug(ctx, slug string) (*domain.Sutra, error)` — US-006, proxy thẳng `sutras.GetSutraBySlug`, luôn lấy bản mới nhất từ server (không cache).

### `NewListSutrasUsecase(sutras domain.SutraRepository, logger logging.Logger) *ListSutrasUsecase`
`ListSutras(ctx, domain.ListSutrasInput) ([]*domain.Sutra, error)` — US-005, proxy thẳng `sutras.ListSutras`. `input.UpdatedSince` rỗng = full sync, khác rỗng = delta sync; `input.Sect` lọc theo tông phái nếu có.

## Cấu hình / dependency
- Không phụ thuộc trực tiếp vào `services/appService/repository` (implementation cụ thể) — mọi usecase nhận vào interface định nghĩa ở `services/appService/domain` (`UserRepository`, `SutraRepository`) qua constructor injection, đúng chiều phụ thuộc Clean Architecture (usecase → domain interface, repository impl phụ thuộc ngược lại vào domain).
- Phụ thuộc package nội bộ khác: `pkg/jwt` (sinh access/refresh token — chỉ dùng ở Register/Login), `pkg/logging` (log có context, mọi usecase đều nhận `logging.Logger`), `pkg/validation` (chuẩn hoá/validate email), `pkg/mailer` (gửi email reset password, chỉ dùng ở `PasswordResetUsecase`).
- Phụ thuộc ngoài: `golang.org/x/crypto/bcrypt` (hash password và reset-token-adjacent flows).
- `AvatarUsecase` không phụ thuộc `pkg/storage.Client` cụ thể, chỉ cần một implementation thoả `AvatarStorage` (interface hẹp khai báo trong file `avatar_usecase.go`) — bất kỳ storage driver nào ở `pkg/storage/*` cũng dùng được miễn implement đúng chữ ký `Upload`.

## Lưu ý / lỗi hay gặp
- `tokenRole` (hằng số `"user"`) là role JWT mặc định gán cho mọi user mới — chưa có hệ thống phân quyền thật ở MVP, đừng nhầm là cấu hình có thể đổi theo user.
- Login/Register cùng dùng `deviceFingerprint` (SHA-256 của Platform|OSVersion|DeviceModel|UserAgent) để định danh thiết bị — đăng nhập từ "thiết bị" có cùng 4 field này sẽ được coi là cùng một device, tái sử dụng session row thay vì tạo mới.
- `LoginUsecase.Login` cố ý gộp lỗi "không tìm thấy email" và "sai mật khẩu" thành cùng một `domain.ErrInvalidCredentials` để chống user-enumeration — đừng "sửa" cho rõ ràng hơn ở tầng usecase, đây là quyết định bảo mật có chủ đích (US-002 AC2/AC3).
- Rate limiting (AC6 của US-002) **không** nằm trong usecase — nó thuộc tầng middleware/gateway, đừng đi tìm hoặc thêm ở đây.
- `PasswordResetUsecase.ForgotPassword` luôn trả `nil` (kể cả khi email không tồn tại, kể cả khi gửi mail thất bại — lỗi gửi mail chỉ được log) — cùng lý do chống enumeration; test không nên assert lỗi trả về khi mail service down.
- Reset token: chỉ hash SHA-256 của token được lưu DB, token thô chỉ tồn tại trong email — muốn test phải tự tính hash giống `hashResetToken` hoặc override `newResetToken` (field không export, chỉ đổi được trong cùng package, dùng cho test xác định).
- `ProfileUsecase.UpdateProfile` ghi đè toàn bộ field trong `input`, không merge với giá trị cũ — để trống một field và gọi update sẽ xoá dữ liệu cột đó (AC5), không phải bug.
- Password tối thiểu 8 ký tự (`minPasswordLength`, khai báo trong `register_usecase.go`, dùng chung cho cả Register và Reset) — không có kiểm tra độ phức tạp (chữ hoa/số/ký tự đặc biệt) ở tầng này.
- Không usecase nào tự mở transaction — mọi tính atomic (ví dụ Register tạo user+device+session cùng lúc, hay ResetPassword vừa đổi password vừa revoke session) đều được đẩy xuống cho repository implement, usecase chỉ gọi một method duy nhất và tin tưởng nó atomic.
