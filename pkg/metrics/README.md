# Metrics — pkg/metrics

## Mục đích
Thu thập và xuất metric (counter, gauge, histogram, summary) từ service sang Google Cloud Monitoring, qua một interface `Collector` chung cộng thêm middleware HTTP/gRPC tự động đo request count/duration/size mà không cần chỉnh sửa handler.

## Khi nào dùng / không dùng
- Dùng khi: service chạy trên GCP (Cloud Run/GKE/GCE) và cần custom metric cho Cloud Monitoring, hoặc muốn tự động có `requests_total`/`request_duration_seconds`/... cho HTTP handler hay gRPC service qua middleware có sẵn.
- Không dùng khi: cần backend Prometheus/OpenTelemetry/Datadog — package này **chỉ hỗ trợ backend GCP** (`Config.Validate()` từ chối mọi `Backend` khác `BackendGCP`); cũng không dùng cho tracing (dùng `pkg/tracing`) hay logging (dùng `pkg/logging`).

## API chính

### `NewCollector(config *Config) (Collector, error)` (factory.go)
Tạo collector theo `config.Backend`. Nếu `config.Enabled == false` trả về `NoOpCollector` (mọi thao tác no-op, không lỗi). Nếu `Enabled == true` và backend GCP chưa được đăng ký (chưa blank-import `pkg/metrics/gcp`) sẽ trả lỗi `"GCP backend not initialized"`.
```go
import (
    "chjpcoj.com.innerlamp/pkg/metrics"
    _ "chjpcoj.com.innerlamp/pkg/metrics/gcp" // đăng ký backend GCP qua init()
)

collector, err := metrics.NewCollector(&metrics.Config{
    Backend:   metrics.BackendGCP,
    Namespace: "my_app",
    Enabled:   true,
    GCP: &metrics.GCPConfig{
        ProjectID:             "my-gcp-project",
        UseDefaultCredentials: true,
    },
    FlushInterval: 60 * time.Second,
})
```

### `Collector` interface (metrics.go)
API chính mà mọi code tiêu thụ package dùng: `RegisterMetric(def MetricDefinition) error`, `IncrementCounter/IncrementCounterBy`, `SetGauge/IncrementGauge/DecrementGauge/IncrementGaugeBy/DecrementGaugeBy`, `ObserveHistogram`, `ObserveSummary`, và lifecycle `Start(ctx)/Stop(ctx)/Flush(ctx)`. Phải `RegisterMetric` trước, nếu không các hàm ghi giá trị trả `ErrMetricNotRegistered`.
```go
_ = collector.RegisterMetric(metrics.NewCounterMetric(
    "orders_total", "Total orders", []string{"status"},
))
_ = collector.IncrementCounter("orders_total", metrics.Labels{"status": "paid"})
```

### `DefaultConfig() *Config`
Config mặc định: backend GCP, `FlushInterval` 10s, `MaxBatchSize` 100, `GCP.ResourceType` `"generic_task"`, `GCP.UseDefaultCredentials` true. Dùng làm điểm khởi đầu rồi override field cần thiết.

### `NewCounterMetric/NewGaugeMetric/NewHistogramMetric/NewHistogramMetricWithBuckets/NewSummaryMetric/NewSummaryMetricWithObjectives(...)`
Helper tạo `MetricDefinition` đúng loại (histogram/summary mặc định dùng `DefaultDurationBuckets`/`DefaultSummaryObjectives`).

### `NewTimer(collector, name, labels) *Timer` / `ObserveDuration(collector, name, labels, fn)`
Đo duration rồi ghi vào histogram khi `Stop()` (hoặc tự động khi `fn` chạy xong với `ObserveDuration`).

### `MetricBuilder` — `NewMetricBuilder(collector, name) *MetricBuilder`
Fluent API: `.WithLabel(k, v)`, `.WithLabels(labels)`, `.Increment()`, `.IncrementBy(v)`, `.Set(v)`, `.Observe(v)` (thử histogram trước, fallback summary nếu `ErrMetricNotRegistered`), `.Time(fn)`.

### `NewRegistry() Registry` / biến `DefaultRegistry`
Quản lý nhiều collector cùng lúc theo tên (`Register/Get/Unregister/List/Default/SetDefault`) — chỉ cần khi service có nhiều collector song song; phần lớn trường hợp chỉ cần một `Collector` truyền tay qua constructor, không cần registry.

### `RegisterStandardHTTPMetrics(collector)` / `RegisterStandardGRPCMetrics(collector)` (helpers.go)
Đăng ký sẵn bộ metric chuẩn (`http_requests_total`, `http_request_duration_seconds`, ... / `grpc_requests_total`, ...). Được middleware HTTP/gRPC gọi tự động trong `NewMetrics(...)`, hiếm khi cần gọi tay.

### `gcp.NewCollector(config *metrics.Config) (*gcp.Collector, error)` (gcp/gcp.go)
Implementation `Collector` ghi TimeSeries vào Google Cloud Monitoring. Blank-import `pkg/metrics/gcp` để nó tự `init()` gọi `metrics.RegisterGCPBackend(...)`, sau đó gọi `metrics.NewCollector` như bình thường thay vì gọi `gcp.NewCollector` trực tiếp.
```go
import _ "chjpcoj.com.innerlamp/pkg/metrics/gcp"
```

### `httpMetrics.NewMetrics(collector, namespace, subsystem) (*Metrics, error)` (middleware/http/http.go)
Tự động gọi `RegisterStandardHTTPMetrics`. Dùng `.Middleware(next http.Handler) http.Handler` để wrap handler, tự đo method/path/status/duration/size + in-flight gauge. Có thêm `.HandlerWithConfig(HandlerConfig{PathPattern, Labels, SkipMetrics}, handler)` khi cần gắn label tuỳ chỉnh hoặc bỏ qua route, và `.NewTimer(method, path)` để đo thủ công.
```go
httpMet, err := httpMetrics.NewMetrics(collector, "my_app", "http")
handler := httpMet.Middleware(mux)
http.ListenAndServe(":8080", handler)
```

### `grpcMetrics.NewMetrics(collector, namespace, subsystem) (*Metrics, error)` (middleware/grpc/grpc.go)
Tự động gọi `RegisterStandardGRPCMetrics`. Cung cấp `.UnaryServerInterceptor()`, `.StreamServerInterceptor()` cho server, `.UnaryClientInterceptor()`, `.StreamClientInterceptor()` cho client — đo service/method/code/duration + đếm stream message (sent/received).
```go
grpcMet, err := grpcMetrics.NewMetrics(collector, "my_app", "grpc")
server := grpc.NewServer(
    grpc.UnaryInterceptor(grpcMet.UnaryServerInterceptor()),
    grpc.StreamInterceptor(grpcMet.StreamServerInterceptor()),
)
```

## Cấu hình / dependency
- `Config` (metrics.go): `Backend` (chỉ `BackendGCP`), `Namespace`/`Subsystem` (ghép thành tiền tố tên metric qua `Config.MetricName`), `Enabled`, `ConstLabels`, `GCP *GCPConfig`, `FlushInterval`, `MaxBatchSize`.
- `GCPConfig`: `ProjectID` (bắt buộc), một trong `UseDefaultCredentials`/`CredentialsFile`/`CredentialsJSON` (bắt buộc khi `Enabled`), `MetricPrefix` (mặc định `custom.googleapis.com`), `Timeout`, `BatchSize`, `ResourceType` (mặc định `generic_task`), `ResourceLabels`.
- Không tự động đọc biến môi trường (`GCP_PROJECT_ID`, `K_SERVICE`, `KUBERNETES_SERVICE_HOST`, ...) trong code package — việc đọc env var và suy ra `ResourceType`/`ResourceLabels` chỉ được minh hoạ trong `examples/complete_example.go`, caller phải tự làm tương tự nếu cần detect Cloud Run/GKE.
- Phụ thuộc thư viện ngoài: `cloud.google.com/go/monitoring/apiv3/v2`, `google.golang.org/grpc` (chỉ cần khi dùng `middleware/grpc`).
- Import path đúng trong repo này: `chjpcoj.com.innerlamp/pkg/metrics`, `.../pkg/metrics/gcp`, `.../pkg/metrics/middleware/http`, `.../pkg/metrics/middleware/grpc` (README/USAGE.md cũ trong thư mục này còn ghi `base_project/pkg/metrics...` — tàn dư từ template, đừng copy).

## Lưu ý / lỗi hay gặp
- Quên blank-import `pkg/metrics/gcp` → `metrics.NewCollector` báo lỗi `"GCP backend not initialized"` dù config hợp lệ.
- Quên `RegisterMetric` trước khi `IncrementCounter`/`SetGauge`/`ObserveHistogram`/`ObserveSummary` → trả `ErrMetricNotRegistered` (không panic, nhưng bị bỏ qua âm thầm nếu không check lỗi).
- `gcp.NewCollector` (được gọi gián tiếp qua `metrics.NewCollector`) tạo `monitoring.MetricClient` **ngay trong constructor**, không tách riêng bước connect như convention `Connect(ctx)` của các package khác trong repo (`pkg/database`, `pkg/storage`, ...) — service vẫn cần gọi thêm `collector.Start(ctx)`, nhưng `Start` chỉ khởi động goroutine flush định kỳ theo `FlushInterval`, không mở kết nối.
- Nếu không bao giờ gọi `collector.Start(ctx)`, gọi `collector.Stop(ctx)` sẽ **block vô thời hạn** — `Stop` chờ trên `doneCh`, vốn chỉ được đóng khi goroutine `flushWorker` (chạy bởi `Start`) kết thúc.
- `IncrementGaugeBy`/`DecrementGaugeBy` trên GCP backend **không thực sự cộng dồn** — implementation hiện tại gọi thẳng `SetGauge` với giá trị truyền vào, tức là ghi đè chứ không đọc giá trị hiện tại rồi cộng thêm; đừng dùng để tăng/giảm gauge tương đối nhiều lần liên tiếp mà kỳ vọng cộng dồn đúng.
- Metric tự flush khi `pendingData` đạt `GCP.BatchSize`, hoặc theo chu kỳ `FlushInterval` nếu đã `Start(ctx)`, hoặc khi gọi `Flush`/`Stop` thủ công — nếu tắt service đột ngột mà không gọi `Stop`/`Flush`, dữ liệu batch chưa đủ ngưỡng sẽ mất.
- `Config.Enabled == false` khiến `Validate()` bỏ qua mọi kiểm tra khác và `NewCollector` trả về `NoOpCollector` — hữu ích để tắt metrics ở môi trường dev mà không cần đổi code, nhưng cũng nghĩa là config sai (thiếu `ProjectID`, ...) sẽ không bị phát hiện cho tới khi `Enabled: true`.
- Tên package con `middleware/http` trùng tên với `net/http` — luôn import alias, ví dụ `httpMetrics "chjpcoj.com.innerlamp/pkg/metrics/middleware/http"`.
- `pkg/metrics` chỉ hỗ trợ backend GCP: truyền `Backend` khác `BackendGCP` (khi `Enabled: true`) khiến `Config.Validate()` trả lỗi ngay khi tạo collector — fail fast lúc khởi tạo, không phải lúc gửi metric.
