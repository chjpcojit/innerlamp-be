# Logging — pkg/logging

## Mục đích
Structured logger cho service (level, fields, JSON/text output), tự động gắn correlation ID (request/trace ID) vào từng log entry và cung cấp middleware HTTP/gRPC để inject correlation ID + logger vào `context.Context` xuyên suốt request.

## Khi nào dùng / không dùng
- Dùng khi: cần log có structure (key-value fields) thay vì `fmt.Println`/`log.Printf`, cần correlation ID xuyên suốt một request HTTP hoặc gRPC (log của handler, usecase, repository trong cùng request phải cùng correlation ID), hoặc cần format log khác nhau giữa local dev (text, có màu) và production (JSON cho log aggregator).
- Không dùng khi: cần distributed tracing đầy đủ (span, parent-child) — dùng `pkg/tracing`; chỉ cần in debug tạm thời khi chạy local thì `fmt.Println` vẫn ổn, không cần kéo cả `Logger` vào.

## API chính

### `logging.New(config Config) Logger`
Tạo logger. `Config.Output` mặc định `os.Stdout` nếu để trống, `Config.CorrelationIDField` mặc định `"correlation_id"`.
```go
logger := logging.New(logging.Config{
    Level:       logging.InfoLevel,
    Formatter:   formatter.NewJSON(),
    ServiceName: "app-service",
    Environment: "production",
})
logger.Info("server started", "port", 8080)
```

### `Logger` interface
`Debug/Info/Warn/Error/Fatal/Panic(msg string, keysAndValues ...interface{})` — `keysAndValues` là cặp key-value xen kẽ (kiểu giống `zap.SugaredLogger`). `Fatal` gọi `os.Exit(1)`, `Panic` gọi `panic(msg)` sau khi log.

### `Logger.WithFields(fields Fields) Logger`
Trả logger con có thêm field cố định, dùng cho các log tiếp theo trong cùng scope (ví dụ gắn `user_id` xuyên suốt một usecase).
```go
userLogger := logger.WithFields(logging.Fields{"user_id": userID})
userLogger.Info("profile updated")
```

### `Logger.WithContext(ctx context.Context) Logger`
Trả logger con gắn với `ctx` — logger sẽ tự lấy correlation ID từ context này ở mỗi lần log (xem phần correlation ID bên dưới). Đây là cách chuẩn để lấy logger đã có correlation ID của request trong handler/usecase.

### `Logger.WithError(err error) Logger`
Shortcut cho `WithFields(Fields{"error": err.Error()})`; trả chính logger cũ nếu `err == nil`.

### `formatter.NewJSON() *JSONFormatter` / `formatter.NewText() *TextFormatter`
Hai formatter implement `logging.Formatter`. JSON dùng cho production/log aggregator (mỗi entry một dòng JSON). Text có màu ANSI theo level (tắt bằng `TextFormatter.DisableColors = true`), phù hợp đọc log khi chạy local.
```go
logger := logging.New(logging.Config{Formatter: formatter.NewText()})
```

### Correlation ID helpers (`pkg/logging`, root package)
- `GetOrCreateCorrelationIDFromHTTP(r *http.Request) string` — đọc header (`X-Correlation-ID`, `X-Request-ID`, `X-Trace-ID`, ... theo thứ tự ưu tiên), sinh UUID mới nếu không có.
- `GetOrCreateCorrelationIDFromGRPC(ctx context.Context) string` — tương tự nhưng đọc từ gRPC incoming metadata.
- `AttachCorrelationIDToHTTPContext(r *http.Request) (context.Context, string)` / `AttachCorrelationIDToGRPCContext(ctx) (context.Context, string)` — gộp bước lấy/sinh + gắn vào context, trả context mới và correlation ID.
- `WithCorrelationID(ctx, id) context.Context` / `GetCorrelationIDFromContext(ctx) string` — set/get thủ công (dùng key nội bộ kiểu `correlationIDKeyType`, xem lưu ý bên dưới).

### `pkg/logging/context`: gắn `Logger` vào `context.Context`
- `context.WithLogger(ctx, logger) context.Context` — lưu logger vào context.
- `context.FromContext(ctx) logging.Logger` — lấy logger ra; nếu context chưa có logger, trả về logger mặc định (`InfoLevel`, ghi `os.Stdout`) thay vì panic hay trả `nil`.
```go
ctx = logctx.WithLogger(ctx, logger)
// ... ở tầng khác, cùng request:
logctx.FromContext(ctx).Info("processing")
```

### Middleware (`pkg/logging/middleware`)
- `middleware.HTTPLogger(logger logging.Logger) func(http.Handler) http.Handler` — middleware `net/http`: trích/sinh correlation ID từ header, gắn vào context (`context`, và cả `logctx.WithLogger`), set header `X-Correlation-ID` cho response, log 1 dòng `"HTTP Request"` sau khi handler chạy xong kèm `status`, `duration_ms`, `size`.
```go
mux := http.NewServeMux()
handler := middleware.HTTPLogger(logger)(mux)
```
- `middleware.GRPCUnaryInterceptor(logger logging.Logger) grpc.UnaryServerInterceptor` / `middleware.GRPCStreamInterceptor(logger logging.Logger) grpc.StreamServerInterceptor` — tương tự cho gRPC: đọc correlation ID từ metadata (`x-correlation-id`, `x-request-id`, ...), set lại vào outgoing header, log `"gRPC Request Processed"`/`"gRPC Request Failed"` (unary) hoặc `"gRPC Stream Finished"`/`"gRPC Stream Failed"` (stream) kèm `duration_ms`, `code`.
```go
grpc.NewServer(grpc.ChainUnaryInterceptor(middleware.GRPCUnaryInterceptor(logger)))
```
- `middleware.GetCorrelationID(ctx context.Context) string` — đọc correlation ID đã được middleware gắn vào context (key `middleware.CorrelationIDKey`, kiểu `string`).

### `logging.MockLogger` (pkg/logging/mock.go)
Mock `Logger` bằng `testify/mock`, dùng trong unit test của package tiêu thụ `logging.Logger`.

## Cấu hình / dependency
- `Config` (pkg/logging/logger.go): `Level`, `Formatter` (nil → format đơn giản mặc định), `Output` (mặc định `os.Stdout`), `CorrelationIDField` (mặc định `"correlation_id"` — tên field ghi vào log entry), `ServiceName`, `Environment` (nếu set sẽ tự thêm field `service`/`environment` vào mọi entry).
- Không tự đọc biến môi trường — `Config` truyền vào code khi khởi tạo logger ở `main.go`.
- Phụ thuộc ngoài: `github.com/google/uuid` (sinh correlation ID), `google.golang.org/grpc` + `google.golang.org/grpc/metadata` (chỉ middleware gRPC), `github.com/stretchr/testify/mock` (chỉ `mock.go`).
- Phụ thuộc nội bộ: `pkg/logging/formatter` phụ thuộc ngược lại `pkg/logging` (để dùng `logging.Entry`, `logging.Fields`), `pkg/logging/context` và `pkg/logging/middleware` cũng import `pkg/logging`.
- Import path đúng trong repo này: `chjpcoj.com.innerlamp/pkg/logging` (không phải `base_project/pkg/logging`).

## Lưu ý / lỗi hay gặp
- **Hai hàm `WithCorrelationID` dùng key context khác nhau, không tương thích với nhau:**
  - `logging.WithCorrelationID(ctx, id)` (root package, `correlation_id.go`) dùng context key kiểu `correlationIDKeyType` (type riêng).
  - `logging/context.WithCorrelationID(ctx, key, id)` — cái mà `middleware.HTTPLogger`/`GRPCUnaryInterceptor` thực sự gọi — dùng key kiểu `string` thường (`middleware.CorrelationIDKey = "correlation_id"`).
  - `logger.log()` (bên trong `New()`) chỉ tìm correlation ID bằng `ctx.Value("correlation_id")` (key kiểu `string`) — nghĩa là nó nhận đúng correlation ID do middleware gắn, nhưng **không** nhận được correlation ID nếu bạn tự gắn bằng `logging.WithCorrelationID` ở root package. Muốn logger tự động log correlation ID, phải đi qua middleware hoặc `pkg/logging/context`, không dùng hàm cùng tên ở root package cho mục đích này.
- `logger.WithContext(ctx)` không tự lấy `Logger` đã lưu trong context bằng `logctx.FromContext` — nó chỉ set `ctx` để lấy correlation ID lúc log. Muốn lấy `Logger` đã được middleware/handler khác gắn sẵn, phải gọi `logctx.FromContext(ctx)`, không phải `logging.New(...).WithContext(ctx)`.
- `Fatal`/`Panic` có side effect thật (`os.Exit(1)`/`panic`) — cẩn thận khi gọi trong request handler, sẽ crash cả process/goroutine chứ không chỉ log lỗi.
- `Level` filter theo `level < l.config.Level`, nếu không set `Config.Level` thì mặc định là `DebugLevel` (giá trị zero của `Level`, tức `0`) — log tất cả mọi level, không phải "tắt log".
- `entry` được lấy từ `sync.Pool` trong `log()` — không giữ tham chiếu `*Entry` ra ngoài scope của formatter, vì nó có thể bị tái sử dụng cho lần log kế tiếp.
- `middleware.HTTPLogger`/interceptor gRPC luôn tạo correlation ID mới nếu client không gửi header/metadata tương ứng — không có cách tắt việc sinh UUID tự động qua config.
