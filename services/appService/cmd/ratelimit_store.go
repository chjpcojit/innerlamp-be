package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/pkg/ratelimit"
	"chjpcoj.com.innerlamp/pkg/ratelimit/store"
	appConfig "chjpcoj.com.innerlamp/services/appService/config"
)

// rateLimitErrorLogInterval caps how often a failing Redis is logged: during an
// outage every limited request would otherwise write one line.
// rateLimitErrorLogInterval giới hạn tần suất log khi Redis lỗi: nếu không, lúc
// sự cố mỗi request bị limit sẽ ghi một dòng log.
const rateLimitErrorLogInterval = 10 * time.Second

// newRateLimitStore builds the one store every rate limiter shares and the
// prefix their keys start with. For Redis it pings at startup (fail fast, like
// the database) and wraps the client so a Redis error at request time falls
// back to a per-instance memory count instead of the middleware's fail-open.
// The returned close function releases the Redis client.
//
// newRateLimitStore tạo store dùng chung cho mọi rate limiter và prefix của
// key. Với Redis: ping lúc khởi động (fail fast giống database) và bọc client
// để khi Redis lỗi lúc xử lý request thì chuyển sang đếm memory theo instance,
// thay vì fail-open như middleware. Hàm close trả về giải phóng Redis client.
// A-001 (fallback policy), TD-001 (no TLS yet), TD-002 (memory store never evicts):
// docs/assumptions/ratelimit/, docs/tech-debt/ratelimit/.
func newRateLimitStore(ctx context.Context, cfg appConfig.RateLimitStoreConfig, logger logging.Logger) (ratelimit.Store, string, func() error, error) {
	if err := cfg.Validate(); err != nil {
		return nil, "", nil, err
	}
	if cfg.Backend == appConfig.RateLimitStoreMemory {
		logger.Warn("rate limits count per instance (rate_limit_store.backend=memory); N instances allow N times each limit")
		return store.NewMemoryStore(), "rl:", func() error { return nil }, nil
	}

	r := cfg.Redis
	client := redis.NewClient(&redis.Options{
		Addr:         r.Addr,
		Password:     r.Password,
		DB:           r.DB,
		DialTimeout:  r.DialTimeout.Duration(),
		ReadTimeout:  r.OpTimeout.Duration(),
		WriteTimeout: r.OpTimeout.Duration(),
		// A limit check that cannot get a connection quickly is better answered
		// by the fallback than queued behind the pool.
		// Lần kiểm tra không lấy được connection nhanh thì để fallback trả lời,
		// không xếp hàng chờ pool.
		PoolTimeout: r.OpTimeout.Duration(),
	})

	pingCtx, cancel := context.WithTimeout(ctx, r.DialTimeout.Duration())
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, "", nil, fmt.Errorf("ping redis %s: %w", r.Addr, err)
	}

	var lastLogged atomic.Int64
	onError := func(err error) {
		now := time.Now().UnixNano()
		last := lastLogged.Load()
		if now-last < int64(rateLimitErrorLogInterval) || !lastLogged.CompareAndSwap(last, now) {
			return
		}
		logger.Warn("rate limit store unavailable; counting per instance until it recovers",
			"addr", r.Addr, "error", err.Error())
	}
	shared := store.NewFallbackStore(store.NewRedisStore(client), store.NewMemoryStore(), onError)
	logger.Info("rate limits count in redis", "addr", r.Addr, "db", r.DB, "key_prefix", r.KeyPrefix)
	return shared, r.KeyPrefix, client.Close, nil
}
