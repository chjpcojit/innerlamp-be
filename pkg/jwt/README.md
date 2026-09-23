# JWT — pkg/jwt

## Mục đích
Sinh và xác thực JWT (access + refresh token) cho các service, hỗ trợ 2 thuật toán ký HS256 (HMAC, khoá đối xứng) và RS256 (RSA, khoá bất đối xứng). Chống algorithm-confusion attack và không cho phép alg `none`.

## Khi nào dùng / không dùng
- Dùng khi: cần phát hành/kiểm tra access token, refresh token cho HTTP API hoặc gRPC service trong hệ thống (auth service, service tiêu thụ token qua middleware).
- Không dùng khi: cần session cookie stateful thuần (dùng `pkg/session`), hoặc chỉ cần xác thực nội bộ giữa các service tin cậy lẫn nhau (mTLS/service mesh có thể phù hợp hơn JWT).

## API chính

### `New(config *Config) (Service, error)`
Tạo JWT service từ config đã validate. Trả lỗi nếu config sai (thiếu key, alg không hỗ trợ, thời hạn refresh ≤ access...).
```go
jwtService, err := jwt.New(&jwt.Config{
    Algorithm:              jwt.AlgorithmHS256,
    SecretKey:              "at-least-32-characters-secret-key",
    AccessTokenExpiration:  15 * time.Minute,
    RefreshTokenExpiration: 7 * 24 * time.Hour,
    Issuer:                 "innerlamp",
    Audience:               "innerlamp_users",
})
```

### `Service.GenerateAccessToken(userID, email, role string, extra map[string]any) (string, error)`
Sinh access token. `userID` bắt buộc, `email`/`role`/`extra` optional.

Mọi token (access lẫn refresh, HS256 lẫn RS256) mang claim **`jti`** (`claims.CustomClaims.ID`): 128 bit ngẫu nhiên từ `crypto/rand`, dạng 32 ký tự hex. Nhờ đó hai token sinh liên tiếp cho cùng user **luôn khác nhau** — trước đây `iat`/`exp` chỉ chính xác 1 giây nên hai token cùng user trong cùng giây giống hệt từng byte, làm vỡ mọi ràng buộc `UNIQUE` trên hash token. `jti` nằm ngoài `extra`: `extra` không ghi đè được nó. Nếu `crypto/rand` lỗi, hàm trả error (`generate token id: ...`) chứ không ký token thiếu `jti`.
```go
token, err := jwtService.GenerateAccessToken("user-123", "user@example.com", "member", nil)
```

### `Service.GenerateRefreshToken(userID string, extra map[string]any) (string, error)`
Sinh refresh token (thời hạn dài hơn access token — bắt buộc theo `Config.Validate`).

Cũng mang `jti` riêng như access token (xem trên).

### `Service.ValidateToken(tokenString string) (*claims.CustomClaims, error)`
Verify chữ ký + hạn dùng + thuật toán. `jti` **không bắt buộc** khi verify: token phát hành trước khi có `jti` vẫn hợp lệ (`claims.ID == ""`), để không làm vô hiệu hàng loạt token đang sống. Lỗi trả về là các sentinel error trong `errors.go` (`ErrTokenExpired`, `ErrInvalidSignature`, `ErrAlgorithmMismatch`, `ErrNoneAlgorithm`, ...), dùng `errors.Is` để phân biệt.
```go
claims, err := jwtService.ValidateToken(tokenString)
if errors.Is(err, jwt.ErrTokenExpired) { /* yêu cầu refresh */ }
```

### `Service.ParseToken(tokenString string) (*claims.CustomClaims, error)`
Parse claims **không** verify chữ ký/hạn dùng (vẫn check alg để chặn tấn công). Dùng khi cần đọc claims từ token đã hết hạn — không dùng để cấp quyền.

### `Service.RefreshAccessToken(refreshToken string, email, role string) (string, error)`
Validate refresh token, kiểm tra đúng `TokenType == TokenTypeRefresh`, rồi sinh access token mới cho cùng `userID`.

### `context.WithClaims/GetClaims/WithUserID/GetUserID` (pkg/jwt/context)
Đưa/lấy claims hoặc userID vào/từ `context.Context` — dùng trong middleware và handler downstream. Có bản `MustGet*` sẽ panic nếu thiếu (chỉ dùng sau middleware đã set).

### `LoadRSAPrivateKeyFromFile(path) / LoadRSAPublicKeyFromFile(path)` (pkg/jwt/keys.go)
Đọc key RSA từ file PEM cho RS256. Path phải do admin cấu hình, không nhận trực tiếp từ user input.

### `MockService` (pkg/jwt/mock.go)
Mock `Service` bằng `testify/mock`, dùng trong unit test của service tiêu thụ `jwt.Service`.

### Middleware (pkg/jwt/middleware)
- `middleware.NewHTTPAuthMiddleware(jwtService, logger)` → `.Authenticate(handler)` cho `net/http`.
- Lỗi HTTP mặc định dùng envelope chuẩn `utils.ApiResponse` (xem `pkg/utils/README.md` §Envelope): token thiếu/sai/hết hạn → `401` `UNAUTHORIZED`; `RequireRole` không có claims → `401` `UNAUTHORIZED`, sai role → `403` `FORBIDDEN`. `HTTPAuthOptions.ErrorHandler` ghi đè được nhánh `401` của `Authenticate` — nếu ghi đè, vẫn phải trả cùng envelope (appService dùng `deliveryhttp.WriteAuthError`).
- `middleware.NewGRPCAuthInterceptor(jwtService, logger)` → `UnaryServerInterceptor()`/`StreamServerInterceptor()` cho gRPC.

## Cấu hình / dependency
- `Config` (pkg/jwt/config.go): `Algorithm` (`AlgorithmHS256`/`AlgorithmRS256`), `SecretKey` (HS256, ≥32 ký tự), `PrivateKey`/`PublicKey` (RS256, `*rsa.PrivateKey`/`*rsa.PublicKey`), `AccessTokenExpiration`, `RefreshTokenExpiration` (phải > access), `Issuer`, `Audience`.
- Không đọc biến môi trường trực tiếp — config truyền vào code (secret/key thường lấy qua `pkg/secrets` rồi build `Config`).
- Phụ thuộc thư viện ngoài: `github.com/golang-jwt/jwt/v5`.
- Phụ thuộc nội bộ: `pkg/jwt/claims` (định nghĩa `CustomClaims`, `TokenType`), `pkg/jwt/context` (context helper), `pkg/jwt/middleware` (HTTP/gRPC). `pkg/jwt/middleware` import thêm `pkg/logging` và `pkg/utils` (envelope response lỗi).
- Import path đúng trong repo này: `chjpcoj.com.innerlamp/pkg/jwt` (một số doc cũ như QUICK_START.md/examples/README.md còn ghi `base_project/pkg/jwt` — đó là tàn dư từ template, đừng copy).

## Lưu ý / lỗi hay gặp
- `RefreshTokenExpiration` phải lớn hơn `AccessTokenExpiration`, nếu không `New()` trả lỗi ngay — fail fast lúc khởi tạo, không phải lúc runtime.
- HS256 secret key < 32 ký tự → `New()` từ chối luôn.
- `ParseToken` không verify chữ ký/hạn dùng — không dùng kết quả của nó để authorize request, chỉ dùng để debug/đọc claims token cũ.
- `ValidateToken`/`ParseToken` đều check alg khớp với `Config.Algorithm` và chặn `none` — không tự ý nới lỏng check này khi thêm driver mới.
- `RefreshAccessToken` không tự rotate refresh token — muốn refresh-token-rotation phải tự gọi thêm `GenerateRefreshToken` ở tầng gọi.
- `MustGetClaims`/`MustGetUserID` trong `pkg/jwt/context` panic nếu context chưa có claims — chỉ gọi sau khi middleware auth đã chạy, không gọi ở handler public.
