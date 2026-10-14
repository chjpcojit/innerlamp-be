package store

import (
	"context"
	"time"

	"chjpcoj.com.innerlamp/pkg/ratelimit"
)

// FallbackStore asks primary first and, when primary returns an error (Redis
// unreachable, timeout), answers from secondary instead. It keeps a limit in
// force while the shared store is down: the count is then per instance, which
// is weaker than the shared one but far better than the fail-open the HTTP
// middleware applies to a store error.
//
// FallbackStore hỏi primary trước; khi primary lỗi (Redis không kết nối được,
// timeout) thì trả lời bằng secondary. Nhờ vậy limit vẫn còn hiệu lực khi store
// dùng chung bị down: lúc đó đếm theo từng instance, yếu hơn đếm chung nhưng tốt
// hơn nhiều so với fail-open mà HTTP middleware áp dụng khi store lỗi.
type FallbackStore struct {
	primary   ratelimit.Store
	secondary ratelimit.Store
	onError   func(error)
}

// NewFallbackStore builds a FallbackStore. onError, when not nil, is called
// with every primary error, so the caller can log or count it.
// NewFallbackStore tạo FallbackStore. onError (nếu khác nil) được gọi với mỗi
// lỗi của primary để caller log hoặc đếm.
func NewFallbackStore(primary, secondary ratelimit.Store, onError func(error)) *FallbackStore {
	return &FallbackStore{primary: primary, secondary: secondary, onError: onError}
}

// Allow implements ratelimit.Store.
// Allow hiện thực ratelimit.Store.
func (s *FallbackStore) Allow(ctx context.Context, key string, limit int64, window time.Duration, burst int64, algo ratelimit.Algorithm) (*ratelimit.Result, error) {
	res, err := s.primary.Allow(ctx, key, limit, window, burst, algo)
	if err == nil {
		return res, nil
	}
	if s.onError != nil {
		s.onError(err)
	}
	return s.secondary.Allow(ctx, key, limit, window, burst, algo)
}
