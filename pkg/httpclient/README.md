# HTTP Client — pkg/httpclient

## Mục đích
Client HTTP dùng lại được, tự động retry với exponential backoff và có circuit breaker tuỳ chọn — dùng khi service này gọi ra một service/API HTTP khác và cần chống chịu lỗi tạm thời (mạng chập chờn, 5xx thoáng qua) mà không cần tự viết logic retry mỗi nơi.

## Khi nào dùng / không dùng
- Dùng khi: gọi ra một HTTP downstream (service nội bộ, API bên thứ ba) và muốn có retry + backoff chuẩn hoá, kèm circuit breaker để tránh dồn request vào một downstream đang chết.
- Không dùng khi: chỉ cần một request HTTP đơn giản, không cần retry (dùng thẳng `net/http`); hoặc cần giới hạn tốc độ gọi *đến* service này (đó là việc của `pkg/ratelimit`, không phải package này — `pkg/httpclient` hiện tại không tích hợp với `pkg/ratelimit`, `pkg/logging`, hay `pkg/metrics`).

## API chính

### `New(cfg Config) *Client`
Tạo `Client` mới, áp default cho các field chưa set trong `cfg` (xem phần Cấu hình). Tạo `*http.Client` nội bộ với `cfg.Timeout`.
```go
client := httpclient.New(httpclient.Config{
    Timeout:          10 * time.Second,
    MaxRetries:       3,
    InitialBackoff:   500 * time.Millisecond,
    FailureThreshold: 5,
    ResetTimeout:     30 * time.Second,
})
```

### `NewWithHTTPClient(httpClient *http.Client, cfg Config) *Client`
Giống `New` nhưng dùng `*http.Client` do caller truyền vào thay vì tạo mới — dùng trong test để trỏ vào `httptest.Server`.
```go
client := httpclient.NewWithHTTPClient(server.Client(), httpclient.Config{MaxRetries: 2})
```

### `(*Client) Do(ctx context.Context, req *http.Request) (*http.Response, error)`
Thực thi request, tự áp policy retry + circuit breaker.
```go
req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
resp, err := client.Do(ctx, req)
if err != nil { /* lỗi mạng, hết retry, hoặc circuit breaker đang open */ }
defer resp.Body.Close()
```
Hành vi retry:
- Retry khi lỗi network/transport hoặc response 5xx; response 4xx trả về ngay, không retry.
- Backoff bắt đầu từ `InitialBackoff`, nhân đôi sau mỗi lần retry.
- Request có body phải "replay" được qua các lần retry — `req.GetBody` phải khác nil (tự động đúng nếu dựng request bằng `bytes.NewReader`).

Hành vi circuit breaker (khi `FailureThreshold > 0`):
- Mở circuit sau `FailureThreshold` lần lỗi liên tiếp (network error, 4xx, hoặc 5xx).
- Khi đang mở, `Do` trả `ErrCircuitOpen` ngay, không gọi mạng.
- Sau `ResetTimeout`, chuyển sang half-open — request tiếp theo là probe; probe thành công thì đóng lại circuit, thất bại thì mở lại.

### `ErrCircuitOpen`
Sentinel error trả về khi circuit breaker đang ở trạng thái open — dùng `errors.Is` để phân biệt với lỗi request thật.
```go
if errors.Is(err, httpclient.ErrCircuitOpen) { /* downstream đang bị breaker chặn, không gọi được */ }
```

## Cấu hình / dependency
- `Config` (pkg/httpclient/httpclient.go):
  - `Timeout` — timeout mỗi request. Default: `30s`.
  - `MaxRetries` — số lần thử lại sau lần đầu. `0` = không retry. Default: `0`.
  - `InitialBackoff` — backoff cơ sở, nhân đôi mỗi lần retry. Default: `1s`.
  - `FailureThreshold` — số lỗi liên tiếp để mở circuit breaker. `0` = tắt circuit breaker. Default: `0`.
  - `ResetTimeout` — thời gian circuit ở trạng thái open trước khi cho probe. Default: `60s`.
- Không phụ thuộc package nội bộ nào khác trong repo (chỉ dùng `net/http`, `context`, `sync`, `time` của Go standard library) — package hoàn toàn tự chứa.
- Import path đúng trong repo này: `chjpcoj.com.innerlamp/pkg/httpclient`.

## Lưu ý / lỗi hay gặp
- Body của request phải replay được nếu `MaxRetries > 0`: nếu `req.Body != nil` và `req.GetBody == nil`, `Do` trả lỗi ngay khi cần retry lần đầu — luôn dựng request bằng `bytes.NewReader`/`strings.NewReader` (tự set `GetBody`), không dùng io.Reader một lần (ví dụ pipe) nếu có bật retry.
- 4xx không kích hoạt retry nhưng **vẫn tính vào** `FailureThreshold` của circuit breaker — nếu downstream trả 4xx liên tục (ví dụ do request sai) circuit vẫn có thể bị mở dù bản chất không phải lỗi tạm thời.
- Caller phải tự đóng `resp.Body` khi `Do` trả về response không lỗi — package không tự đóng giúp.
- `Client` an toàn dùng đồng thời từ nhiều goroutine (circuit breaker dùng `sync.Mutex`) — tạo một `Client` dùng chung cho cả service thay vì tạo mới mỗi request.
- Package này **chưa** tích hợp `pkg/ratelimit`, `pkg/logging`, hay `pkg/metrics` dù mô tả kiến trúc tổng ở `CLAUDE.md` gốc có nhắc — nếu cần log/metrics cho request, tự wrap `Do` ở tầng gọi.
