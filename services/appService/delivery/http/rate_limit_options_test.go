package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/jwt/claims"
	jwtcontext "chjpcoj.com.innerlamp/pkg/jwt/context"
	"chjpcoj.com.innerlamp/pkg/ratelimit"
	"chjpcoj.com.innerlamp/pkg/ratelimit/store"
)

// recordingStore wraps a MemoryStore and remembers every key it was asked about.
// recordingStore bọc MemoryStore và ghi lại mọi key được hỏi.
type recordingStore struct {
	inner *store.MemoryStore
	keys  []string
}

func (s *recordingStore) Allow(ctx context.Context, key string, limit int64, window time.Duration, burst int64, algo ratelimit.Algorithm) (*ratelimit.Result, error) {
	s.keys = append(s.keys, key)
	return s.inner.Allow(ctx, key, limit, window, burst, algo)
}

// Two limiters on one shared store, both keyed by the same IP, must count
// separately: each one's scope keeps its keys apart.
// Hai limiter trên cùng store, cùng key theo IP, phải đếm riêng nhờ scope.
func TestWithSharedStore_ScopesKeepLimitersApart(t *testing.T) {
	shared := &recordingStore{inner: store.NewMemoryStore()}
	strict, err := NewClientIPRateLimit(1, time.Hour, false, WithSharedStore(shared, "rl:strict:ip"))
	require.NoError(t, err)
	loose, err := NewClientIPRateLimit(5, time.Hour, false, WithSharedStore(shared, "rl:loose:ip"))
	require.NoError(t, err)

	var calls int
	strictH, looseH := strict.WrapFunc(okHandler(&calls)), loose.WrapFunc(okHandler(&calls))

	w := httptest.NewRecorder()
	strictH(w, fromIP("10.0.0.1", ""))
	require.Equal(t, http.StatusNoContent, w.Code)
	w = httptest.NewRecorder()
	strictH(w, fromIP("10.0.0.1", ""))
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "the strict limiter is spent")

	w = httptest.NewRecorder()
	looseH(w, fromIP("10.0.0.1", ""))
	assert.Equal(t, http.StatusNoContent, w.Code, "the strict limiter's count must not spill into the loose one")

	assert.Equal(t, []string{"rl:strict:ip:10.0.0.1", "rl:strict:ip:10.0.0.1", "rl:loose:ip:10.0.0.1"}, shared.keys)
}

func TestWithSharedStore_UserLimitKeysByAccount(t *testing.T) {
	shared := &recordingStore{inner: store.NewMemoryStore()}
	rl, err := NewUserRateLimit(1, time.Hour, WithSharedStore(shared, "rl:profile:user"))
	require.NoError(t, err)

	var calls int
	h := rl.WrapFunc(okHandler(&calls))
	r := httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
	r = r.WithContext(jwtcontext.WithClaims(r.Context(), &claims.CustomClaims{UserID: "42"}))
	h(httptest.NewRecorder(), r)

	assert.Equal(t, []string{"rl:profile:user:42"}, shared.keys)
}
