# Rate Limit — pkg/ratelimit

## Mục đích
Rate limiting cho HTTP handler và gRPC service, hỗ trợ 4 thuật toán (token bucket, sliding window, fixed window, leaky bucket) trên 2 backend lưu trạng thái: in-memory (một instance) hoặc Redis (nhiều instance dùng chung).

## Khi nào dùng / không dùng
- Dùng khi: cần giới hạn số request/gRPC call theo IP, user ID, API key, hoặc endpoint cụ thể (chống brute-force login, chống lạm dụng API).
- Không dùng khi: cần throttle ở tầng network/gateway (Cloud Armor, Envoy, API Gateway) — package này chạy trong process của service, không thay thế WAF/gateway; hoặc chỉ cần giới hạn tài nguyên nội bộ (dùng `sync.Pool`/semaphore thay vì rate limiter theo key).

## API chính

### `ratelimit.Config` (pkg/ratelimit/ratelimit.go)
Cấu hình cho một rate limiter: `Rate` (số request/window), `Burst` (số request tối đa dồn lại — chủ yếu dùng cho token bucket), `Window` (khoảng thời gian), `Algorithm` (`ratelimit.TokenBucket` / `SlidingWindow` / `FixedWindow` / `LeakyBucket`), `Store` (backend lưu trạng thái), `KeyFunc func(ctx context.Context) (string, error)` (hàm trích key — theo IP, user ID, API key...).

### `middleware.NewLimiter(config ratelimit.Config) (*Limiter, error)`
Tạo limiter thuần (không gắn HTTP/gRPC), dùng khi cần gọi `Allow` trực tiếp trong logic nghiệp vụ. Tự điền default: `Store` → `store.NewMemoryStore()` nếu nil, `Algorithm` → `TokenBucket` nếu rỗng, `Burst` → bằng `Rate` nếu ≤ 0. Trả lỗi nếu `Rate` hoặc `Window` ≤ 0.
```go
limiter, err := middleware.NewLimiter(ratelimit.Config{
    Rate:      100,
    Window:    time.Minute,
    Algorithm: ratelimit.TokenBucket,
    Store:     store.NewMemoryStore(),
})
result, err := limiter.Allow(ctx) // result.Allowed, result.Remaining, result.RetryAfter
```

### `middleware.NewHTTPMiddleware(config ratelimit.Config) (*HTTPMiddleware, error)`
Middleware `net/http`. `.Handler(next http.Handler) http.Handler` bọc handler, set header `X-RateLimit-Limit`/`X-RateLimit-Remaining`/`X-RateLimit-Reset`, khi vượt hạn mức trả `429` + header `Retry-After` và body envelope chuẩn `{"success":false,"error":{"code":"RATE_LIMITED","message":"too many requests","type":"business"},"details":{"retry_after_seconds":N}}` (qua `utils.WriteBusinessError`, xem `pkg/utils/README.md` §Envelope).
```go
mw, err := middleware.NewHTTPMiddleware(ratelimit.Config{
    Rate: 5, Burst: 5, Window: time.Minute,
    Algorithm: ratelimit.TokenBucket,
    Store:     store.NewMemoryStore(),
    KeyFunc: func(ctx context.Context) (string, error) {
        path, _ := ctx.Value(middleware.ContextKeyHTTPPath).(string)
        return "login_limit:" + path, nil
    },
})
mux.Handle("/login", mw.Handler(loginHandler))
```

### `middleware.NewGRPCMiddleware(config ratelimit.Config) (*GRPCMiddleware, error)`
Interceptor gRPC. `.UnaryInterceptor() grpc.UnaryServerInterceptor` và `.StreamInterceptor() grpc.StreamServerInterceptor`, trả lỗi `codes.ResourceExhausted` khi vượt hạn mức.
```go
mw, _ := middleware.NewGRPCMiddleware(config)
server := grpc.NewServer(
    grpc.UnaryInterceptor(mw.UnaryInterceptor()),
    grpc.StreamInterceptor(mw.StreamInterceptor()),
)
```
Context key sẵn có để dùng trong `KeyFunc`: `middleware.ContextKeyHTTPPath`, `middleware.ContextKeyHTTPMethod` (HTTP), `middleware.ContextKeyGRPCMethod` (gRPC) — middleware set các key này vào context trước khi gọi `KeyFunc`.

### `store.NewMemoryStore() *MemoryStore`
Backend in-memory, `map[string]Limiter` bảo vệ bằng `sync.RWMutex`. Trạng thái chỉ tồn tại trong process hiện tại.

### `store.NewRedisStore(client redis.UniversalClient) *RedisStore`
Backend Redis, mỗi thuật toán chạy bằng Lua script (`redis.NewScript`) để đảm bảo atomic khi nhiều instance cùng ghi. Sliding window dùng `TIME` của Redis (không lệch đồng hồ giữa instance), member duy nhất cho mỗi request và `PEXPIRE` bằng window. `.RegisterAlgorithm(name ratelimit.Algorithm, algo RedisAlgorithm)` cho phép đăng ký thuật toán tuỳ biến.
```go
rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
config.Store = store.NewRedisStore(rdb)
```

## Cấu hình / dependency
- Import path đúng trong repo này: `chjpcoj.com.innerlamp/pkg/ratelimit`, `.../store`, `.../middleware`.
- `store.NewMemoryStore()`: chỉ cần `sync`, không phụ thuộc thư viện ngoài — phù hợp một instance duy nhất hoặc test.
- `store.NewRedisStore(client)`: cần `github.com/redis/go-redis/v9` và một `redis.UniversalClient` đã kết nối — dùng khi service chạy nhiều instance (Cloud Run/GKE scale >1) và cần đếm request thống nhất giữa các instance.
- `middleware.NewGRPCMiddleware`: phụ thuộc `google.golang.org/grpc`.
- `middleware.NewHTTPMiddleware`: import `pkg/utils` để ghi response `429` theo envelope chuẩn.
- Nếu không set `Config.Store`, `middleware.NewLimiter` tự dùng `MemoryStore` — không tự chọn Redis, phải truyền tay khi cần distributed.

## Lưu ý / lỗi hay gặp
- **Fail-open khi Store lỗi**: cả `HTTPMiddleware.Handler` và `GRPCMiddleware` đều cho request đi qua nếu `Store.Allow` trả lỗi (Redis mất kết nối chẳng hạn) — ưu tiên availability hơn strict limiting, cần lưu ý khi audit bảo mật (brute-force login có thể lọt qua lúc Redis down). Muốn giữ limit khi Redis down thì bọc store bằng `store.NewFallbackStore(redisStore, store.NewMemoryStore(), onError)`: lỗi của Redis được báo qua `onError` và lần kiểm tra đó được đếm bằng memory của instance (appService dùng cách này).
- **Store dùng chung cần scope cho key**: một `RedisStore` (hoặc một `MemoryStore`) dùng cho nhiều limiter thì mỗi limiter phải có prefix key riêng trong `KeyFunc`, nếu không hai limiter cùng key theo IP sẽ đếm chung một bucket.
- **Reply của Lua script chỉ là số nguyên**: Lua `false` về tới Go thành `nil` và số thực bị cắt, nên mọi script trả `allowed` dạng `1/0` và thời gian dạng ms nguyên; `scriptInts` báo lỗi (không panic) nếu reply sai dạng. Trước 2026-10-14 request bị từ chối làm panic.
- **`KeyFunc` rỗng hoặc trả `""`** → `middleware.Limiter.Allow` dùng key `"global"`, nghĩa là mọi caller dùng chung một bucket — dễ gây lock-out chéo giữa user nếu quên set `KeyFunc`.
- **`MemoryStore` không chia sẻ được giữa nhiều instance** — chạy rate limit theo IP/user trên nhiều pod/container sẽ mỗi instance đếm riêng, hạn mức thực tế nhân lên theo số instance. Dùng `RedisStore` nếu cần đúng con số toàn cục.
- **`Burst` chỉ có ý nghĩa với `TokenBucket`** (và một phần `LeakyBucket` dùng `Rate` để tính interval) — `SlidingWindow`/`FixedWindow` bỏ qua `Burst`, dùng thẳng `Rate`.
- **`LeakyBucket`**: trường `Result.Remaining` không phản ánh đúng ý nghĩa (comment ngay trong code: "Not really applicable for leaky bucket") — đừng dựa vào `Remaining` để hiển thị UI khi dùng thuật toán này; với bản Redis, `leakyBucketRedis.Allow` còn không set `ResetTime`.
- Mỗi `middleware.Limiter`/`HTTPMiddleware`/`GRPCMiddleware` gắn với **một** `ratelimit.Config` (một rate/algorithm/key strategy) — muốn áp nhiều tầng giới hạn khác nhau (vd. giới hạn theo IP + giới hạn theo endpoint) phải tạo nhiều middleware và chain lại (xem `examples/chained_limit`).
- `store.RedisStore.Allow` trả `store.ErrNotImplemented` nếu `algo` không có trong map thuật toán đã đăng ký — kiểm tra lỗi này khi thêm `Algorithm` mới mà quên gọi `RegisterAlgorithm`.
