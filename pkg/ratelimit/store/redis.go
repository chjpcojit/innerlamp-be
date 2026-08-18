package store

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"base_project/pkg/ratelimit"
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
		local ttl = math.floor(fill_time * 2)

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
		local allowed = filled_tokens >= requested
		local new_tokens = filled_tokens
		if allowed then
			new_tokens = filled_tokens - requested
		end

		redis.call("setex", tokens_key, ttl, new_tokens)
		redis.call("setex", timestamp_key, ttl, now)

		local retry_after = 0
		if not allowed then
			retry_after = (requested - filled_tokens) / rate
		end

		return { allowed, new_tokens, retry_after }
	`)

	slidingWindowScript = redis.NewScript(`
		local key = KEYS[1]
		local window_start = tonumber(ARGV[1])
		local now = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		local window = tonumber(ARGV[4])

		-- Remove old requests
		redis.call("ZREMRANGEBYSCORE", key, "-inf", window_start)
		
		-- Count current requests
		local count = redis.call("ZCARD", key)
		local allowed = count < limit

		if allowed then
			redis.call("ZADD", key, now, now)
			redis.call("PEXPIRE", key, window * 1000)
			count = count + 1
		end

		-- Get oldest request time for reset calculation
		local oldest = redis.call("ZRANGE", key, 0, 0, "WITHSCORES")
		local reset_time = now + window
		if #oldest > 0 then
			reset_time = tonumber(oldest[2]) + window
		end

		return { allowed, limit - count, reset_time }
	`)

	fixedWindowScript = redis.NewScript(`
		local key = KEYS[1]
		local limit = tonumber(ARGV[1])
		local window_seconds = tonumber(ARGV[2])

		local count = redis.call("INCR", key)
		if count == 1 then
			redis.call("EXPIRE", key, window_seconds)
		end

		local allowed = count <= limit
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

		local allowed = false
		local retry_after = 0
		local reset_time = next_allowed

		if now >= next_allowed then
			allowed = true
			next_allowed = now + interval
			redis.call("SET", key, next_allowed, "PX", math.ceil(interval/1000) * 2)
			reset_time = next_allowed
		else
			allowed = false
			retry_after = (next_allowed - now) / 1000000 -- seconds
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

	vals := res.([]interface{})
	allowed := vals[0].(int64) == 1
	retryAfter := 0.0
	if v, ok := vals[2].(float64); ok {
		retryAfter = v
	} else if v, ok := vals[2].(string); ok {
		_ = v
	}

	resetTime := time.Now()
	if !allowed {
		resetTime = resetTime.Add(time.Duration(retryAfter * float64(time.Second)))
	}

	return &ratelimit.Result{
		Allowed:    allowed,
		Remaining:  0,
		ResetTime:  resetTime,
		RetryAfter: time.Duration(retryAfter * float64(time.Second)),
	}, nil
}

type slidingWindowRedis struct{}

func (s *slidingWindowRedis) Allow(ctx context.Context, client redis.UniversalClient, key string, limit int64, window time.Duration, burst int64) (*ratelimit.Result, error) {
	now := float64(time.Now().UnixNano()) / 1e9
	windowStart := now - window.Seconds()

	keys := []string{key}
	args := []interface{}{windowStart, now, limit, window.Seconds()}

	res, err := slidingWindowScript.Run(ctx, client, keys, args...).Result()
	if err != nil {
		return nil, err
	}

	vals := res.([]interface{})
	allowed := vals[0].(int64) == 1
	remaining := vals[1].(int64)

	return &ratelimit.Result{
		Allowed:   allowed,
		Remaining: remaining,
		ResetTime: time.Now().Add(window),
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

	vals := res.([]interface{})
	allowed := vals[0].(int64) == 1
	remaining := vals[1].(int64)
	ttl := vals[2].(int64)

	return &ratelimit.Result{
		Allowed:   allowed,
		Remaining: remaining,
		ResetTime: time.Now().Add(time.Duration(ttl) * time.Second),
	}, nil
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

	vals := res.([]interface{})
	allowed := vals[0].(int64) == 1

	return &ratelimit.Result{
		Allowed: allowed,
	}, nil
}
