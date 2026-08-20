# Session — pkg/session

## Mục đích
Quản lý session người dùng trên MongoDB: tạo/lưu session gắn với access + refresh token (sinh qua `pkg/jwt`), rotate token, revoke, và giới hạn số session đồng thời theo device fingerprint.

## Khi nào dùng / không dùng
- Dùng khi: service cần session có trạng thái (stateful) lưu trên MongoDB — theo dõi thiết bị đăng nhập, cho phép revoke từng session/tất cả session của user, giới hạn số thiết bị đăng nhập cùng lúc, hoặc cần rotate access+refresh token theo cặp gắn với một session cụ thể.
- Không dùng khi: chỉ cần verify JWT stateless không cần lưu trạng thái server-side (dùng thẳng `pkg/jwt` + middleware của nó), hoặc datastore của service không phải MongoDB (package này chỉ có implementation Mongo, chưa có driver khác).

## API chính

### `NewMongoSessionManager(db *mongo.Database, jwtService jwt.Service, config *Config) (Manager, error)`
Khởi tạo session manager, tạo các index MongoDB cần thiết (`session_id` unique, `user_id`, `refresh_token`, `device_id`, TTL index trên `expires_at`, index kép `user_id+is_active`, `user_id+device_id`). `config` truyền `nil` sẽ dùng `DefaultConfig()`.
```go
sessionMgr, err := session.NewMongoSessionManager(mongoDB, jwtService, session.DefaultConfig())
```

### `Manager.CreateSession(ctx, userID, email, role string, deviceInfo *DeviceInfo, ipAddress, userAgent string) (*Session, error)`
Sinh device fingerprint (SHA-256), enforce giới hạn `MaxSessionsPerUser` (revoke session cũ nhất nếu vượt), sinh cặp access/refresh token qua `jwtService`, rồi lưu session vào MongoDB.
```go
sess, err := sessionMgr.CreateSession(ctx, "user-123", "user@example.com", "member", deviceInfo, "1.2.3.4", "Mozilla/5.0...")
```

### `Manager.ValidateSession(ctx, sessionID, accessToken string) error`
Lấy session theo `sessionID`, verify `accessToken` qua `jwtService.ValidateToken`, so khớp với token đang lưu trong session, rồi cập nhật `last_activity_at`.
```go
if err := sessionMgr.ValidateSession(ctx, sessionID, accessToken); err != nil { /* unauthorized */ }
```

### `Manager.RotateTokens(ctx, sessionID string) (*Session, error)`
Validate refresh token đang lưu, sinh cặp access/refresh token mới, tăng `TokenVersion`, cập nhật session.
```go
newSess, err := sessionMgr.RotateTokens(ctx, sessionID)
```

### `Manager.RevokeSession(ctx, sessionID, reason string) error` / `RevokeAllUserSessions(ctx, userID, reason string) (int64, error)` / `RevokeOtherSessions(ctx, userID, currentSessionID, reason string) (int64, error)`
Đánh dấu session `is_revoked=true`, set `revoked_at`/`revoked_reason`. Dùng cho logout (session hiện tại), đổi mật khẩu/nghi ngờ xâm nhập (toàn bộ session của user), hoặc "đăng xuất thiết bị khác" (giữ session hiện tại).

### `Manager.GetSession(ctx, sessionID string) (*Session, error)` / `GetSessionByRefreshToken(ctx, refreshToken string) (*Session, error)`
Trả `ErrSessionNotFound`/`ErrSessionRevoked`/`ErrSessionExpired` tương ứng thay vì session đã hết hạn/bị thu hồi.

### `Manager.GetUserActiveSessions(ctx, userID string) ([]*Session, error)`
Liệt kê session còn hoạt động (chưa hết hạn, chưa revoke) của một user, sắp theo `last_activity_at` giảm dần — dùng cho màn "thiết bị đang đăng nhập".

### `Manager.CleanupExpiredSessions(ctx context.Context) (int64, error)`
Xoá thủ công session đã hết hạn hoặc đã revoke quá 30 ngày. MongoDB TTL index trên `expires_at` đã tự xoá session hết hạn, hàm này chỉ cần thiết nếu muốn dọn thêm session revoked lâu ngày hoặc chạy cleanup job thủ công.

### `DefaultConfig() *Config`
`SessionTTL: 24h`, `AccessTokenTTL: 15m`, `RefreshTokenTTL: 7 ngày`, `MaxSessionsPerUser: 5`.

## Cấu hình / dependency
- `Config` (session.go): `SessionTTL`, `AccessTokenTTL`, `RefreshTokenTTL`, `MaxSessionsPerUser` (`<= 0` nghĩa là không giới hạn số session đồng thời).
- Phụ thuộc nội bộ: chỉ `pkg/jwt` (`jwt.Service` để sinh/validate token). Package **không** dùng `pkg/database` (giao tiếp thẳng qua `go.mongodb.org/mongo-driver/mongo`, không qua interface `Database` của repo) và **không** dùng `pkg/tracing` — session không tự propagate trace context, tự thêm nếu service cần.
- Phụ thuộc thư viện ngoài: `go.mongodb.org/mongo-driver` (bson, mongo, options, primitive).
- Cần một `*mongo.Database` đã kết nối (dùng `mongo.Connect` trực tiếp, package này không đi qua `pkg/database/mongodb`) và một `jwt.Service` đã khởi tạo từ `pkg/jwt`.
- Import path đúng trong repo này: `chjpcoj.com.innerlamp/pkg/session` (README cũ ghi `base_project/pkg/session` — tàn dư template, đừng copy). Xem thêm `QUICK_START.md` và `CLOUD_RUN.md` trong cùng thư mục cho ví dụ triển khai — cả hai cũng cần rà lại import path trước khi copy.

## Lưu ý / lỗi hay gặp
- `NewMongoSessionManager` tạo index đồng bộ lúc khởi tạo (timeout 30s) — gọi 1 lần lúc service start, không gọi lại mỗi request.
- `ValidateSession` so khớp `accessToken` truyền vào với token **đang lưu** trong session — nếu client giữ access token cũ sau khi đã `RotateTokens`, `ValidateSession` sẽ trả `ErrInvalidRefreshToken` (không phải lỗi JWT-level) dù chữ ký token vẫn hợp lệ.
- `RotateTokens` không kiểm tra session đã bị revoke trước khi tạo token mới nếu gọi trực tiếp `GetSession` trả lỗi trước — nhưng nếu `GetSession` thành công (session còn active), hàm luôn sinh token mới, không có cơ chế phát hiện refresh-token-reuse (không lưu lịch sử token cũ để so sánh).
- `MaxSessionsPerUser` chỉ enforce **lúc tạo session mới** (`CreateSession` → `enforceSessionLimit`) — không có job nền tự động thu hồi khi limit thay đổi cấu hình.
- Session hết hạn tự bị MongoDB TTL index xoá khỏi collection — `CleanupExpiredSessions()` không bắt buộc phải chạy định kỳ để giữ collection sạch, chỉ cần nếu muốn dọn sớm session đã revoke quá 30 ngày.
- `DeviceInfo` là con trỏ optional trong `CreateSession` — truyền `nil` vẫn tạo được session, fingerprint khi đó chỉ hash từ `userAgent` (kém phân biệt hơn, dễ đụng fingerprint giữa các thiết bị dùng chung user agent).
- Struct `Session`/`DeviceInfo` dùng tag `bson`, không có tag `json` — nếu serialize thẳng ra API response cần tự thêm struct/DTO khác (đừng trả `AccessToken`/`RefreshToken` ra ngoài log hay response không cần thiết).
