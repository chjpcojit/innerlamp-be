# Middleware — pkg/middleware

## Mục đích
Tập hợp middleware/interceptor generic, không phụ thuộc domain: log request, CORS, và recover panic — cho cả HTTP (`net/http`) lẫn gRPC. Khác với các thư mục `middleware` nằm trong từng package khác (`pkg/jwt/middleware`, `pkg/logging/middleware`, `pkg/ratelimit/middleware`, `pkg/metrics/middleware/...`) vốn gắn chặt với package chủ của nó, `pkg/middleware` là package độc lập, chỉ chứa các middleware cơ bản dùng chung.

## Khi nào dùng / không dùng
- Dùng khi: cần nhanh một middleware `net/http` để log request, set CORS header, hoặc recover panic thành response lỗi thay vì crash service; hoặc cần interceptor gRPC tương đương để recover panic trong handler unary/stream.
- Không dùng khi: cần structured logging có context/trace (dùng `pkg/logging/middleware` thay vì `Logger` ở đây — `Logger` chỉ `log.Printf` đơn giản, không cấu trúc); cần auth/JWT (dùng `pkg/jwt/middleware`); cần rate limit (dùng `pkg/ratelimit/middleware`); cần thu metrics (dùng `pkg/metrics/middleware/...`); cần CORS cấu hình được origin/method cụ thể (`CORS` ở đây hardcode `Access-Control-Allow-Origin: *` và không nhận tham số cấu hình).

## API chính

### `Logger(next http.Handler) http.Handler`
Middleware `net/http` chuẩn (dạng `func(http.Handler) http.Handler`), log method + path lúc bắt đầu và thời gian xử lý lúc kết thúc bằng `log.Printf` (không structured, không dùng `pkg/logging`).
```go
mux := http.NewServeMux()
handler := middleware.Logger(mux)
```

### `CORS(next http.Handler) http.Handler`
Set cứng `Access-Control-Allow-Origin: *`, `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`, `Access-Control-Allow-Headers: Content-Type, Authorization`; trả `200 OK` ngay cho request `OPTIONS` (preflight) mà không gọi tiếp `next`.
```go
handler := middleware.CORS(mux)
```

### `Recovery(next http.Handler) http.Handler`
Recover panic trong handler, log stack trace bằng `log.Printf`, trả response `500` dạng JSON `{"status":"error","error":{"code":"ERR_INTERNAL","message":"..."}}`.
```go
handler := middleware.Recovery(mux)
```

### `RecoveryInterceptor() grpc.UnaryServerInterceptor`
Recover panic trong unary gRPC handler, trả về `status.Errorf(codes.Internal, ...)` kèm stack trace trong message lỗi (không log ra ngoài — stack trace nằm trong error trả về client).
```go
srv := grpc.NewServer(grpc.UnaryInterceptor(middleware.RecoveryInterceptor()))
```

### `StreamRecoveryInterceptor() grpc.StreamServerInterceptor`
Bản tương đương `RecoveryInterceptor` cho stream gRPC handler.

### `RecoveryWithLoggerInterceptor(logFunc func(msg string, keysAndValues ...interface{})) grpc.UnaryServerInterceptor`
Giống `RecoveryInterceptor` nhưng nhận thêm hàm log ngoài (ví dụ log có cấu trúc từ `pkg/logging`) để ghi `method`, `panic`, `stack` — thay vì nhét stack trace vào error message trả cho client.
```go
srv := grpc.NewServer(grpc.UnaryInterceptor(
    middleware.RecoveryWithLoggerInterceptor(logger.Errorw),
))
```

### `StreamRecoveryWithLoggerInterceptor(logFunc func(msg string, keysAndValues ...interface{})) grpc.StreamServerInterceptor`
Bản tương đương `RecoveryWithLoggerInterceptor` cho stream gRPC handler.

## Cấu hình / dependency
- Không có config struct, không đọc biến môi trường — mọi hàm đều nhận tham số trực tiếp (handler kế tiếp, hoặc `logFunc`).
- Phụ thuộc thư viện ngoài: `google.golang.org/grpc` (chỉ `grpc.go` cần).
- Không phụ thuộc package nội bộ nào khác trong repo (không import `pkg/logging`, `pkg/jwt`, ...).
- Import path đúng: `chjpcoj.com.innerlamp/pkg/middleware`.
- **Hiện chưa có nơi nào trong repo import package này** (`grep -rn "pkg/middleware\"" --include="*.go" .` không ra kết quả) — `services/appService` chưa dùng các middleware ở đây.

## Lưu ý / lỗi hay gặp
- `Logger` và `Recovery` (bản HTTP) dùng `log.Printf` thẳng ra stdout/stderr — không có log level, không structured, không gắn trace ID. Nếu service đã dùng `pkg/logging`, hai middleware này sẽ tạo log không đồng nhất định dạng với phần còn lại của service.
- `CORS` hardcode `Access-Control-Allow-Origin: *` — không phù hợp nếu cần whitelist origin cụ thể hoặc bật `Access-Control-Allow-Credentials` (không set `*` cùng credentials được, vi phạm CORS spec).
- `RecoveryInterceptor`/`StreamRecoveryInterceptor` nhét cả stack trace vào `status.Errorf` trả về **client** — rò rỉ thông tin nội bộ (đường dẫn file, số dòng) ra ngoài nếu client là bên thứ ba. Cân nhắc dùng bản `WithLogger` (log stack ra logger nội bộ, chỉ trả `"panic recovered: %v"` không kèm stack cho client) khi expose ra ngoài.
- Không có middleware nào ở đây set request ID / trace ID — muốn correlate log request với trace phải tự thêm hoặc dùng `pkg/tracing`/`pkg/logging/middleware`.
- Thứ tự chain middleware quan trọng: `Recovery`/`RecoveryInterceptor` nên là middleware ngoài cùng (chạy đầu tiên, tức bọc các middleware khác) để bắt được panic xảy ra trong các middleware nằm sâu hơn (`Logger`, `CORS`, handler thật).
