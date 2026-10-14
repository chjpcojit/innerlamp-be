package store

import (
	"context"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/ratelimit"
)

// redisTestClient connects to REDIS_TEST_ADDR (default localhost:6379, the
// docker compose Redis) and skips when it is not reachable or under -short.
// redisTestClient kết nối REDIS_TEST_ADDR (mặc định localhost:6379, Redis của
// docker compose) và skip khi không kết nối được hoặc chạy -short.
func redisTestClient(t *testing.T) *redis.Client {
	t.Helper()
	if testing.Short() {
		t.Skip("needs a real Redis; skipped under -short")
	}
	addr := os.Getenv("REDIS_TEST_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	client := redis.NewClient(&redis.Options{Addr: addr, Password: os.Getenv("REDIS_TEST_PASSWORD")})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Skipf("no Redis at %s: %v", addr, err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

// testKey isolates each test's keys and removes them afterwards.
// testKey tách key của từng test và xoá chúng sau khi chạy.
func testKey(t *testing.T, client *redis.Client) string {
	t.Helper()
	key := "rl:test:" + t.Name() + ":" + strconv.FormatInt(time.Now().UnixNano(), 36)
	t.Cleanup(func() {
		_ = client.Del(context.Background(), key, key+":tokens", key+":ts").Err()
	})
	return key
}

func TestRedisStore_DeniedRequestDoesNotPanic(t *testing.T) {
	client := redisTestClient(t)
	s := NewRedisStore(client)
	ctx := context.Background()

	algos := []ratelimit.Algorithm{ratelimit.SlidingWindow, ratelimit.FixedWindow, ratelimit.TokenBucket, ratelimit.LeakyBucket}
	for _, algo := range algos {
		t.Run(string(algo), func(t *testing.T) {
			key := testKey(t, client)
			var denied *ratelimit.Result
			for i := 0; i < 3; i++ {
				res, err := s.Allow(ctx, key, 2, time.Minute, 2, algo)
				require.NoError(t, err)
				if !res.Allowed {
					denied = res
				}
			}
			require.NotNil(t, denied, "a request over the limit must be denied")
			assert.Greater(t, denied.RetryAfter, time.Duration(0), "a denial must say when to retry")
		})
	}
}

func TestRedisStore_SlidingWindow(t *testing.T) {
	client := redisTestClient(t)
	s := NewRedisStore(client)
	ctx := context.Background()
	key := testKey(t, client)

	for i := 0; i < 3; i++ {
		res, err := s.Allow(ctx, key, 3, time.Minute, 3, ratelimit.SlidingWindow)
		require.NoError(t, err)
		assert.True(t, res.Allowed, "request %d", i+1)
		assert.Equal(t, int64(2-i), res.Remaining)
	}

	res, err := s.Allow(ctx, key, 3, time.Minute, 3, ratelimit.SlidingWindow)
	require.NoError(t, err)
	assert.False(t, res.Allowed)
	assert.Equal(t, int64(0), res.Remaining)
	assert.InDelta(t, time.Minute.Seconds(), res.RetryAfter.Seconds(), 2)

	ttl, err := client.PTTL(ctx, key).Result()
	require.NoError(t, err)
	assert.Greater(t, ttl, time.Duration(0), "the key must expire on its own")
	assert.LessOrEqual(t, ttl, time.Minute)
}

func TestRedisStore_SlidingWindowExpires(t *testing.T) {
	client := redisTestClient(t)
	s := NewRedisStore(client)
	ctx := context.Background()
	key := testKey(t, client)

	_, err := s.Allow(ctx, key, 1, 200*time.Millisecond, 1, ratelimit.SlidingWindow)
	require.NoError(t, err)
	res, err := s.Allow(ctx, key, 1, 200*time.Millisecond, 1, ratelimit.SlidingWindow)
	require.NoError(t, err)
	assert.False(t, res.Allowed)

	time.Sleep(250 * time.Millisecond)
	res, err = s.Allow(ctx, key, 1, 200*time.Millisecond, 1, ratelimit.SlidingWindow)
	require.NoError(t, err)
	assert.True(t, res.Allowed, "the window must slide past the old request")
}

// Concurrent callers, as from several instances, must never get more than the
// limit through, and same-millisecond requests must each be counted.
// Nhiều caller đồng thời (như nhiều instance) không bao giờ vượt limit, và
// request cùng mili-giây phải được đếm riêng.
func TestRedisStore_SlidingWindowConcurrent(t *testing.T) {
	client := redisTestClient(t)
	s := NewRedisStore(client)
	ctx := context.Background()
	key := testKey(t, client)

	const limit, callers = 20, 100
	var allowed atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := s.Allow(ctx, key, limit, time.Minute, limit, ratelimit.SlidingWindow)
			if err == nil && res.Allowed {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, int64(limit), allowed.Load())
}
