package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/ratelimit"
)

type stubStore struct {
	res *ratelimit.Result
	err error
}

func (s stubStore) Allow(context.Context, string, int64, time.Duration, int64, ratelimit.Algorithm) (*ratelimit.Result, error) {
	return s.res, s.err
}

func TestFallbackStore(t *testing.T) {
	down := errors.New("redis down")
	tests := []struct {
		name        string
		primary     stubStore
		wantAllowed bool
		wantErrSeen error
	}{
		{name: "primary answers", primary: stubStore{res: &ratelimit.Result{Allowed: true}}, wantAllowed: true},
		{name: "primary denies", primary: stubStore{res: &ratelimit.Result{Allowed: false}}, wantAllowed: false},
		{name: "primary fails, secondary answers", primary: stubStore{err: down}, wantAllowed: false, wantErrSeen: down},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var seen error
			secondary := stubStore{res: &ratelimit.Result{Allowed: false}}
			s := NewFallbackStore(tc.primary, secondary, func(err error) { seen = err })

			res, err := s.Allow(context.Background(), "k", 1, time.Minute, 1, ratelimit.SlidingWindow)
			require.NoError(t, err)
			assert.Equal(t, tc.wantAllowed, res.Allowed)
			assert.Equal(t, tc.wantErrSeen, seen)
		})
	}
}

// With a real memory secondary the limit still holds while the primary is down.
// Với secondary là memory thật, limit vẫn giữ khi primary down.
func TestFallbackStore_LimitHoldsWhilePrimaryDown(t *testing.T) {
	s := NewFallbackStore(stubStore{err: errors.New("down")}, NewMemoryStore(), nil)
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		res, err := s.Allow(ctx, "k", 2, time.Minute, 2, ratelimit.SlidingWindow)
		require.NoError(t, err)
		assert.True(t, res.Allowed)
	}
	res, err := s.Allow(ctx, "k", 2, time.Minute, 2, ratelimit.SlidingWindow)
	require.NoError(t, err)
	assert.False(t, res.Allowed)
}
