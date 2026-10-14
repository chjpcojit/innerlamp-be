package store

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"chjpcoj.com.innerlamp/pkg/ratelimit"
)

// RedisAlgorithm defines the interface for Redis-based rate limiting algorithms
type RedisAlgorithm interface {
	Allow(ctx context.Context, client redis.UniversalClient, key string, limit int64, window time.Duration, burst int64) (*ratelimit.Result, error)
}

// RedisStore implements a Redis-based rate limit store
type RedisStore struct {
	client     redis.UniversalClient
	algorithms map[ratelimit.Algorithm]RedisAlgorithm
}

// NewRedisStore creates a new Redis store
func NewRedisStore(client redis.UniversalClient) *RedisStore {
	return &RedisStore{
		client: client,
		algorithms: map[ratelimit.Algorithm]RedisAlgorithm{
			ratelimit.TokenBucket:   &tokenBucketRedis{},
			ratelimit.SlidingWindow: &slidingWindowRedis{},
			ratelimit.FixedWindow:   &fixedWindowRedis{},
			ratelimit.LeakyBucket:   &leakyBucketRedis{},
		},
	}
}

// RegisterAlgorithm registers a custom algorithm
func (s *RedisStore) RegisterAlgorithm(name ratelimit.Algorithm, algo RedisAlgorithm) {
	s.algorithms[name] = algo
}

// Allow checks if the request is allowed
func (s *RedisStore) Allow(ctx context.Context, key string, limit int64, window time.Duration, burst int64, algo ratelimit.Algorithm) (*ratelimit.Result, error) {
	implementation, exists := s.algorithms[algo]
	if !exists {
		return nil, ErrNotImplemented
	}
	return implementation.Allow(ctx, s.client, key, limit, window, burst)
}

// --- Algorithm Implementations ---

// Lua scripts
var (
	tokenBucketScript = redis.NewScript(`
		local tokens_key = KEYS[1]
		local timestamp_key = KEYS[2]
		local rate = tonumber(ARGV[1])
		local capacity = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local requested = tonumber(ARGV[4])
		local fill_time = tonumber(ARGV[5])
		local ttl = math.max(1, math.ceil(fill_time * 2))

		local last_tokens = tonumber(redis.call("get", tokens_key))
		if last_tokens == nil then
			last_tokens = capacity
		end

		local last_refreshed = tonumber(redis.call("get", timestamp_key))
		if last_refreshed == nil then
			last_refreshed = 0
		end

		local delta = math.max(0, now - last_refreshed)
		local filled_tokens = math.min(capacity, last_tokens + (delta * rate))
		-- Lua false becomes a nil reply, so allowed is returned as 1/0.
		-- Lua false thành nil reply nên allowed trả về dạng 1/0.
		local allowed = 0
		local new_tokens = filled_tokens
		if filled_tokens >= requested then
			allowed = 1
			new_tokens = filled_tokens - requested
		end

		redis.call("setex", tokens_key, ttl, new_tokens)
		redis.call("setex", timestamp_key, ttl, now)

		-- Floats are truncated to integers on the way out, so retry is sent in ms.
		-- Số thực bị cắt thành số nguyên khi trả về, nên retry tính bằng ms.
		local retry_ms = 0
		if allowed == 0 then
			retry_ms = math.ceil((requested - filled_tokens) / rate * 1000)
		end

		return { allowed, math.floor(new_tokens), retry_ms }
	`)

	// slidingWindowScript keeps one sorted-set entry per accepted request. The
	// clock is Redis's own TIME, so instances with skewed clocks share one
	// window; ARGV[3] is a caller-unique member so two requests in the same
	// millisecond are both counted.
	// slidingWindowScript giữ một entry sorted-set cho mỗi request được nhận.
	// Đồng hồ là TIME của Redis nên các instance lệch giờ vẫn dùng chung một
	// window; ARGV[3] là member duy nhất để hai request cùng mili-giây đều được đếm.
	slidingWindowScript = redis.NewScript(`
		local key = KEYS[1]
		local window = tonumber(ARGV[1]) -- ms
		local limit = tonumber(ARGV[2])
		local member = ARGV[3]

		local t = redis.call("TIME")
		local now = tonumber(t[1]) * 1000 + math.floor(tonumber(t[2]) / 1000)

		redis.call("ZREMRANGEBYSCORE", key, "-inf", now - window)

		local count = redis.call("ZCARD", key)
		local allowed = 0
		if count < limit then
			redis.call("ZADD", key, now, member)
			count = count + 1
			allowed = 1
		end
		redis.call("PEXPIRE", key, window)

		local reset = now + window
		local oldest = redis.call("ZRANGE", key, 0, 0, "WITHSCORES")
		if #oldest > 0 then
			reset = tonumber(oldest[2]) + window
		end

		local retry = 0
		if allowed == 0 then
			retry = reset - now
		end

		return { allowed, math.max(0, limit - count), reset, retry }
	`)

	fixedWindowScript = redis.NewScript(`
		local key = KEYS[1]
		local limit = tonumber(ARGV[1])
		local window_seconds = tonumber(ARGV[2])

		local count = redis.call("INCR", key)
		if count == 1 then
			redis.call("EXPIRE", key, window_seconds)
		end

		local allowed = 0
		if count <= limit then
			allowed = 1
		end
		local remaining = math.max(0, limit - count)
		local ttl = redis.call("TTL", key)

		return { allowed, remaining, ttl }
	`)

	leakyBucketScript = redis.NewScript(`
		local key = KEYS[1]
		local interval = tonumber(ARGV[1]) -- microseconds
		local now = tonumber(ARGV[2]) -- microseconds

		local next_allowed = tonumber(redis.call("GET", key))
		if next_allowed == nil then
			next_allowed = now
		end

		local allowed = 0
		local retry_after = 0
		local reset_time = next_allowed

		if now >= next_allowed then
			allowed = 1
			next_allowed = now + interval
			redis.call("SET", key, next_allowed, "PX", math.ceil(interval/1000) * 2)
			reset_time = next_allowed
		else
			retry_after = math.ceil((next_allowed - now) / 1000) -- ms
		end

		return { allowed, 0, retry_after }
	`)
)

type tokenBucketRedis struct{}

func (t *tokenBucketRedis) Allow(ctx context.Context, client redis.UniversalClient, key string, limit int64, window time.Duration, burst int64) (*ratelimit.Result, error) {
	rate := float64(limit) / window.Seconds()
	now := float64(time.Now().UnixNano()) / 1e9
	fillTime := float64(burst) / rate

	keys := []string{key + ":tokens", key + ":ts"}
	args := []interface{}{rate, burst, now, 1, fillTime}

	res, err := tokenBucketScript.Run(ctx, client, keys, args...).Result()
	if err != nil {
		return nil, err
	}

	vals, err := scriptInts(res, 3)
	if err != nil {
		return nil, err
	}
	allowed := vals[0] == 1
	retryAfter := time.Duration(vals[2]) * time.Millisecond

	return &ratelimit.Result{
		Allowed:    allowed,
		Remaining:  vals[1],
		ResetTime:  time.Now().Add(retryAfter),
		RetryAfter: retryAfter,
	}, nil
}

type slidingWindowRedis struct{}

func (s *slidingWindowRedis) Allow(ctx context.Context, client redis.UniversalClient, key string, limit int64, window time.Duration, burst int64) (*ratelimit.Result, error) {
	keys := []string{key}
	args := []interface{}{window.Milliseconds(), limit, uniqueMember()}

	res, err := slidingWindowScript.Run(ctx, client, keys, args...).Result()
	if err != nil {
		return nil, err
	}

	vals, err := scriptInts(res, 4)
	if err != nil {
		return nil, err
	}

	return &ratelimit.Result{
		Allowed:    vals[0] == 1,
		Remaining:  vals[1],
		ResetTime:  time.UnixMilli(vals[2]),
		RetryAfter: time.Duration(vals[3]) * time.Millisecond,
	}, nil
}

type fixedWindowRedis struct{}

func (f *fixedWindowRedis) Allow(ctx context.Context, client redis.UniversalClient, key string, limit int64, window time.Duration, burst int64) (*ratelimit.Result, error) {
	keys := []string{key}
	args := []interface{}{limit, window.Seconds()}

	res, err := fixedWindowScript.Run(ctx, client, keys, args...).Result()
	if err != nil {
		return nil, err
	}

	vals, err := scriptInts(res, 3)
	if err != nil {
		return nil, err
	}
	allowed := vals[0] == 1
	resetIn := time.Duration(vals[2]) * time.Second

	result := &ratelimit.Result{
		Allowed:   allowed,
		Remaining: vals[1],
		ResetTime: time.Now().Add(resetIn),
	}
	if !allowed {
		result.RetryAfter = resetIn
	}
	return result, nil
}

type leakyBucketRedis struct{}

func (l *leakyBucketRedis) Allow(ctx context.Context, client redis.UniversalClient, key string, limit int64, window time.Duration, burst int64) (*ratelimit.Result, error) {
	interval := window.Nanoseconds() / limit
	now := time.Now().UnixNano() / 1000 // microseconds

	keys := []string{key}
	args := []interface{}{interval / 1000, now}

	res, err := leakyBucketScript.Run(ctx, client, keys, args...).Result()
	if err != nil {
		return nil, err
	}

	vals, err := scriptInts(res, 3)
	if err != nil {
		return nil, err
	}
	retryAfter := time.Duration(vals[2]) * time.Millisecond

	return &ratelimit.Result{
		Allowed:    vals[0] == 1,
		ResetTime:  time.Now().Add(retryAfter),
		RetryAfter: retryAfter,
	}, nil
}

// scriptInts converts a Lua script reply into n integers. Every script returns
// integers only: a Lua boolean false would arrive as nil and a float would be
// truncated, so a reply of any other shape is an error, not a panic.
// scriptInts đổi reply của Lua script thành n số nguyên. Mọi script chỉ trả số
// nguyên: Lua false sẽ thành nil và số thực bị cắt, nên reply sai dạng là lỗi
// chứ không panic.
func scriptInts(res interface{}, n int) ([]int64, error) {
	vals, ok := res.([]interface{})
	if !ok || len(vals) < n {
		return nil, fmt.Errorf("ratelimit redis: unexpected script reply %v", res)
	}
	out := make([]int64, n)
	for i := 0; i < n; i++ {
		v, ok := vals[i].(int64)
		if !ok {
			return nil, fmt.Errorf("ratelimit redis: reply[%d] is %T, want int64", i, vals[i])
		}
		out[i] = v
	}
	return out, nil
}

// uniqueMember returns a sorted-set member no other request will produce, so
// concurrent requests in the same millisecond are counted separately.
// uniqueMember trả về member không request nào khác tạo ra, để các request
// đồng thời trong cùng mili-giây được đếm riêng.
func uniqueMember() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36) + "-" + strconv.FormatUint(rand.Uint64(), 36)
}
