package http

import "chjpcoj.com.innerlamp/pkg/ratelimit"

// RateLimitOption configures ClientIPRateLimit and UserRateLimit.
// RateLimitOption cấu hình ClientIPRateLimit và UserRateLimit.
type RateLimitOption func(*rateLimitOptions)

type rateLimitOptions struct {
	store ratelimit.Store
	scope string
}

// WithSharedStore keeps the counters in store under keys "<scope>:<ip or user
// id>". A shared store (Redis) holds every limiter's counters side by side, so
// each limiter needs its own scope: without one, two limiters keyed by the same
// IP would count into one bucket.
// WithSharedStore lưu counter trong store với key "<scope>:<ip hoặc user id>".
// Store dùng chung (Redis) chứa counter của mọi limiter cạnh nhau nên mỗi
// limiter cần scope riêng: thiếu scope, hai limiter cùng key theo IP sẽ đếm
// chung một bucket.
func WithSharedStore(store ratelimit.Store, scope string) RateLimitOption {
	return func(o *rateLimitOptions) {
		o.store = store
		o.scope = scope
	}
}

func applyRateLimitOptions(opts []RateLimitOption) rateLimitOptions {
	var o rateLimitOptions
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// key prefixes id with the scope. An empty id stays empty so the limiter keeps
// its "global" fallback bucket, scoped like every other key.
// key gắn scope trước id. id rỗng vẫn tạo key có scope, tránh trộn bucket
// "global" giữa các limiter.
func (o rateLimitOptions) key(id string) string {
	if o.scope == "" {
		return id
	}
	return o.scope + ":" + id
}
