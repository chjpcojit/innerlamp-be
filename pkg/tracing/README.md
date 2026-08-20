# Tracing — pkg/tracing

## Mục đích
Bọc OpenTelemetry SDK để khởi tạo distributed tracing cho service (HTTP/gRPC), xuất span sang GCP Cloud Trace, stdout (dev) hoặc no-op (test), dùng chung một `Config` cho toàn bộ pipeline.

## Khi nào dùng / không dùng
- Dùng khi: cần theo dõi latency/luồng gọi xuyên service (HTTP handler → downstream call → DB), muốn export span lên GCP Cloud Trace ở production hoặc in ra stdout khi dev local.
- Không dùng khi: chỉ cần structured logging (dùng `pkg/logging`) hoặc số liệu tổng hợp (request count, latency histogram — dùng `pkg/metrics`). Ba package này bổ trợ nhau, không thay thế nhau.

## API chính

### `InitTracer(config types.Config) (*Tracer, error)`
Khởi tạo `TracerProvider` theo `config.ExporterType`, set làm global TracerProvider (`otel.SetTracerProvider`) và global propagator W3C Trace Context (`otel.SetTextMapPropagator`). Gọi một lần lúc service khởi động.
```go
tracer, err := tracing.InitTracer(types.Config{
    ServiceName:  "user-service",
    Environment:  "production",
    ExporterType: types.ExporterTypeGCP,
    SamplingRate: 0.1,
    ProjectID:    os.Getenv("GCP_PROJECT_ID"),
})
if err != nil {
    log.Fatal(err)
}
defer tracer.Shutdown(context.Background())
```

### `(*Tracer) Shutdown(ctx context.Context) error`
Flush và đóng `TracerProvider`. Với `ExporterTypeNoop`, `tp` là `nil` nên `Shutdown` no-op an toàn. Gọi đúng một lần lúc service tắt (thường qua `defer`).

### `(*Tracer) GetTracer() trace.Tracer`
Trả về `trace.Tracer` của OpenTelemetry gốc — dùng khi cần API span đầy đủ (thay vì chỉ gọi `Start`).

### `(*Tracer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span)`
Tạo span thủ công, trả về `ctx` mới mang span. Phải `defer span.End()`.
```go
ctx, span := tracer.Start(ctx, "process-user")
defer span.End()
span.SetAttributes(attribute.String("user.id", userID))
```

### `types.Config` (pkg/tracing/types/types.go)
Struct cấu hình truyền cho `InitTracer` — xem phần "Cấu hình / dependency".

### `gcp.NewExporter(config types.Config) (trace.SpanExporter, error)` / `gcp.NewBatchSpanProcessor(config types.Config) (trace.SpanProcessor, error)`
Tạo exporter/processor gửi span tới GCP Cloud Trace (dùng `github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace`). `InitTracer` tự gọi `NewBatchSpanProcessor` khi `ExporterType == types.ExporterTypeGCP` — thường không cần gọi trực tiếp trừ khi tự lắp `TracerProvider` tùy biến.
```go
processor, err := gcp.NewBatchSpanProcessor(config) // config.ProjectID bắt buộc
```

### Middleware (pkg/tracing/middleware/http, pkg/tracing/middleware/grpc)
Không nhận tham số `tracer` — dựa vào global `TracerProvider` mà `InitTracer` đã set, nên phải gọi `InitTracer` trước khi dùng.
- `http.NewHTTPMiddleware() func(http.Handler) http.Handler` — bọc `otelhttp.NewHandler`, tự động instrument HTTP request.
- `http.WithRouteTag(handler http.Handler, route string) http.Handler` — gắn tên route cụ thể cho span thay vì tên mặc định `"http_request"`.
- `grpc.NewServerStatsHandler() stats.Handler` — dùng với `grpc.StatsHandler(...)` phía server.
- `grpc.NewClientStatsHandler() stats.Handler` / `grpc.NewClientDialOption() grpc.DialOption` — instrument phía gRPC client.

## Cấu hình / dependency
`types.Config` (pkg/tracing/types/types.go) là struct phẳng, không có struct con `GCPConfig`:
- `ServiceName string` — tên service, dùng làm resource attribute và tên tracer.
- `ProjectID string` — GCP project ID, **bắt buộc** khi `ExporterType == types.ExporterTypeGCP` (nếu rỗng, `gcp.NewExporter` trả lỗi ngay).
- `SamplingRate float64` — 0.0–1.0, dùng `sdktrace.TraceIDRatioBased`.
- `ExporterType types.ExporterType` — một trong `types.ExporterTypeGCP`, `types.ExporterTypeStdout`, `types.ExporterTypeNoop`.
- `Environment string` — set vào resource attribute `deployment.environment`.

Package không tự đọc biến môi trường (không có auto-detect `K_SERVICE`/`KUBERNETES_SERVICE_HOST` như một số package khác trong repo) — `ProjectID`/`Environment` phải truyền vào từ code gọi (thường lấy từ env var ở tầng `main.go`).

Với exporter GCP: `texporter.New` (thư viện `opentelemetry-operations-go`) chỉ nhận `WithProjectID` trong `gcp.go` — không truyền credentials tường minh, nên thư viện dùng Application Default Credentials (ADC) mặc định của Google Cloud client library.

Phụ thuộc ngoài chính: `go.opentelemetry.io/otel` (+ `sdk/trace`, `sdk/resource`, `semconv`, `propagation`), `go.opentelemetry.io/otel/exporters/stdout/stdouttrace`, `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`, `github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace`.

Import path đúng trong repo này: `chjpcoj.com.innerlamp/pkg/tracing`, `.../pkg/tracing/gcp`, `.../pkg/tracing/types`, `.../pkg/tracing/middleware/http`, `.../pkg/tracing/middleware/grpc`.

**Chưa có consumer nào trong repo thực sự dùng package này** — `grep` cho thấy chỉ `tracing.go` và `gcp/gcp.go` tự tham chiếu lẫn nhau; `services/appService` và `pkg/session` hiện không import `pkg/tracing`. Muốn dùng phải tự gọi `InitTracer` trong `main.go` của service tiêu thụ.

## Lưu ý / lỗi hay gặp
- Middleware HTTP/gRPC (`NewHTTPMiddleware`, `NewServerStatsHandler`, ...) không nhận `*Tracer` — chúng lấy tracer qua global `TracerProvider`. Nếu quên gọi `InitTracer` trước, middleware vẫn chạy nhưng dùng no-op tracer mặc định của OpenTelemetry (không có lỗi, nhưng không có span thật nào được export).
- `types.Option` / `WithServiceName`, `WithProjectID`, `WithSamplingRate`, `WithExporterType`, `WithEnvironment` tồn tại trong `types.go` nhưng **không được `InitTracer` sử dụng** (`InitTracer` nhận `types.Config` trực tiếp, không nhận `...Option`) — đừng dựa vào các hàm này, có vẻ là API chưa hoàn thiện.
- `ExporterTypeGCP` dùng `sdktrace.NewBatchSpanProcessor` (bất đồng bộ, batch); `ExporterTypeStdout` dùng `sdktrace.NewSimpleSpanProcessor` (đồng bộ, in ngay từng span) — khác nhau về độ trễ khi debug local so với production.
- `ExporterTypeNoop` trả về `*Tracer` với `tp == nil`; `GetTracer()`/`Start()` vẫn hoạt động (dùng `otel.Tracer(config.ServiceName)`) nhưng không export đi đâu — dùng cho test.
- `gcp.NewExporter` chỉ validate `ProjectID` rỗng — không validate quyền/ADC hợp lệ tại thời điểm gọi; lỗi credential (nếu có) sẽ xuất hiện muộn hơn khi thực sự gửi span, không phải lúc `InitTracer` trả về.
- `Shutdown` nên gọi đúng một lần lúc service tắt (qua `defer`); gọi lại nhiều lần trên cùng một `TracerProvider` không được đảm bảo an toàn bởi OpenTelemetry SDK.
