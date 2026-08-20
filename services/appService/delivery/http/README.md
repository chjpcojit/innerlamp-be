# HTTP Delivery — services/appService/delivery/http

## Mục đích
Tầng "delivery" (interface adapters) của appService: các `http.Handler` thuần `net/http` chuyển request/response HTTP thành lời gọi usecase, không chứa business logic. Mỗi handler định nghĩa một interface usecase tối thiểu (do handler — consumer — khai báo) để dễ unit test bằng mock thay vì usecase thật.

## Khi nào dùng / không dùng
- Dùng khi: thêm/sửa một endpoint REST của appService — parse request, validate hình thức (JSON hợp lệ, field bắt buộc), map input sang `domain.*Input`, gọi usecase, map kết quả/lỗi domain sang response JSON + HTTP status.
- Không dùng khi: cần business rule (điều kiện hợp lệ email/password, tính duy nhất, gửi mail...) — những cái đó thuộc `services/appService/usecase`; cần truy vấn DB/storage trực tiếp — thuộc `services/appService/repository`; cần xác thực JWT — dùng `pkg/jwt/middleware` (xem phần Lưu ý, middleware này **chưa được wire** trong `cmd/main.go`).

## API chính

Mỗi handler có constructor `New*Handler(uc <interface>) *XxxHandler` trả về một `http.Handler` (method `ServeHTTP`), nhận usecase qua interface hẹp khai báo ngay trong file đó.

### `NewRegisterHandler(uc registerUsecase) *RegisterHandler`
`POST /v1/auth/register` (US-001). Decode `registerRequest` (strict — reject field lạ), gắn `Device`/`IPAddress` (từ `X-Forwarded-For` hoặc `RemoteAddr`)/`UserAgent`, gọi `usecase.Register`. Trả `201 Created` + `registerResponse{access_token, refresh_token, user}`. Map lỗi domain: `ErrEmailTaken`/`ErrPhoneTaken` → 409, `ErrInvalidEmail`/`ErrInvalidPhone`/`ErrInvalidPassword` → 400, còn lại → 500.

### `NewLoginHandler(uc loginUsecase) *LoginHandler`
`POST /v1/auth/login` (US-002). Cùng shape response với register. `ErrInvalidCredentials` → 401 `INVALID_CREDENTIALS` (email sai và password sai trả cùng một lỗi, không phân biệt được — theo AC2/AC3).

### `NewForgotPasswordHandler(uc forgotPasswordUsecase) *ForgotPasswordHandler`
`POST /v1/auth/forgot-password` (US-003). Luôn trả cùng message chung chung (`forgotPasswordMessage`) dù email tồn tại hay không (AC1/AC2) — usecase không phân biệt hai trường hợp, chỉ lỗi hạ tầng thật mới rơi vào nhánh 500.

### `NewResetPasswordHandler(uc resetPasswordUsecase) *ResetPasswordHandler`
`POST /v1/auth/reset-password` (US-003). `ErrResetTokenInvalid` → 400, `ErrInvalidPassword` → 400 (field `new_password`), còn lại → 500.

### `NewProfileHandler(uc profileUsecase) *ProfileHandler`
`GET/PATCH /v1/users/me` (US-004). Route theo `r.Method` trong `ServeHTTP` (không phải `net/http.ServeMux` method pattern). **Yêu cầu authenticated user** — dùng `authenticatedUserID(r)`; thiếu/không hợp lệ → 401 `UNAUTHORIZED`. `profileResponse` cố tình không dùng `omitempty` để field trống trả `""` thay vì bị lược khỏi JSON (AC1).

### `NewAvatarHandler(uc avatarUsecase) *AvatarHandler`
`POST /v1/users/me/avatar` (US-004 AC3). Body là raw image bytes (không phải JSON) — tách riêng khỏi PATCH `/v1/users/me`. Giới hạn `maxAvatarBytes` = 5MB qua `http.MaxBytesReader`; content-type xác định bằng `http.DetectContentType` trên byte thật, chỉ chấp nhận `image/jpeg`, `image/png`, `image/webp` (header `Content-Type` của client không được tin). Cũng yêu cầu authenticated user như Profile.

### `NewSutrasHandler(uc listSutrasUsecase) *SutrasHandler`
`GET /v1/content/sutras` (US-005). Public — không cần auth. Query param `sect` (lọc) và `updated_since` (RFC3339, dùng cho delta-sync) — sai format `updated_since` → 400 `INVALID_UPDATED_SINCE`. Response luôn là array (rỗng thay vì `null`).

### `NewSutraDetailHandler(uc getSutraUsecase) *SutraDetailHandler`
`GET /v1/content/sutras/{slug}` (US-006). Public. Đăng ký trong `main.go` bằng pattern có method (`"GET /v1/content/sutras/{slug}"`) để `net/http.ServeMux` (Go 1.22+) tự parse `{slug}` qua `r.PathValue("slug")`. `ErrSutraNotFound` → 404.

### `authenticatedUserID(r *http.Request) (int64, bool)` (auth_context.go)
Helper dùng chung cho Profile/Avatar handler: đọc userID (string) mà middleware auth gắn vào context qua `jwtcontext.GetUserID`, `strconv.ParseInt` sang `int64`. Trả `false` nếu context không có userID hoặc parse lỗi — không panic, không dùng `MustGetUserID`.

## Cấu hình / dependency
- Phụ thuộc `pkg/jwt/context` (`jwtcontext.GetUserID`/`WithUserID`) để đọc/set userID trong `context.Context` — do middleware `pkg/jwt/middleware.NewHTTPAuthMiddleware(...).Authenticate(handler)` gắn vào khi request đi qua. Bản thân package này **không tự xác thực** — nó chỉ đọc context, không verify token.
- Phụ thuộc `pkg/utils` cho toàn bộ hạ tầng handler dùng chung: `DecodeJSONBodyStrict` (decode + reject unknown field), `WriteSuccessResponse`/`WriteSuccessResponseWithStatus`, `WriteValidationError`/`WriteBusinessError`/`WriteSystemError`, `IsRequestTooLarge`, `GetDecodeErrorDetails`.
- Phụ thuộc `services/appService/domain` cho input/output types (`domain.RegisterInput`, `domain.LoginResult`, `domain.User`, `domain.Sutra`, các sentinel error `domain.Err*`...) và `services/appService/usecase` (qua interface hẹp từng handler tự khai báo, không import type `usecase.*Usecase` cụ thể).
- Wiring thật (constructor + đăng ký route) nằm ở `services/appService/cmd/main.go`, dùng `http.NewServeMux()` chuẩn thư viện — không có router bên thứ ba (chi/gorilla/echo...).

## Lưu ý / lỗi hay gặp
- **Middleware xác thực JWT chưa được gắn vào `cmd/main.go`.** `mux.Handle("/v1/users/me", profileHandler)` và `.../avatar` đăng ký handler trực tiếp, không bọc qua `pkg/jwt/middleware.NewHTTPAuthMiddleware(...).Authenticate(...)`. Kết quả: chạy `main.go` như hiện tại, mọi request tới hai route này sẽ luôn thiếu userID trong context → `authenticatedUserID` luôn trả `false` → luôn 401, dù có gửi token hợp lệ. Đây là việc còn thiếu khi wiring production, không phải bug của handler — handler đã viết đúng phần đọc context, chỉ là chưa có ai gắn middleware phía trước nó.
- `ProfileHandler`/`AvatarHandler` tự điều hướng theo `r.Method` bên trong `ServeHTTP` thay vì dùng pattern `"GET /v1/users/me"` của `ServeMux` — khi thêm method mới cho cùng path phải sửa switch trong handler, không sửa ở `main.go`.
- `AvatarHandler` đọc toàn bộ body vào memory (`io.ReadAll` sau `MaxBytesReader`) trước khi detect content-type — giới hạn 5MB là cứng, không cấu hình qua config; muốn đổi phải sửa hằng số `maxAvatarBytes` trong code.
- Các interface usecase (`registerUsecase`, `loginUsecase`, `profileUsecase`...) được khai báo **riêng trong từng file handler**, không tập trung một chỗ — nếu đổi signature usecase thật ở `services/appService/usecase`, phải tự đối chiếu và sửa interface tương ứng ở đây (Go không báo lỗi cho tới khi compile chỗ constructor được gọi).
- Response lỗi mặc định cho mọi lỗi không match nhánh cụ thể là `utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)` — chi tiết lỗi thật không bao giờ lộ ra response, chỉ có trong log của usecase/logger; đừng trông chờ debug được lỗi 500 chỉ từ response HTTP.
- `forgotPasswordMessage`/`resetPasswordMessage` là chuỗi tiếng Việt hardcode trong file — chưa có cơ chế i18n, đổi ngôn ngữ phải sửa trực tiếp code.
