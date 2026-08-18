// ⚠️ MOCK DATA - NOT FOR PRODUCTION
// This file is an example only. It uses an in-memory store for rate limiting.
//
// Before using in production:
//   - Replace MemoryStore with RedisStore backed by GCP Memorystore (Redis)
//   - Configure Redis client via pkg/secrets/ (do not hardcode connection strings)
//
// MOCK_DATA_MARKER: MemoryStore used instead of Redis
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"base_project/pkg/ratelimit"
	"base_project/pkg/ratelimit/middleware"
	"base_project/pkg/ratelimit/store"
)

func main() {
	// Shared store (e.g., Redis)
	// In production: client := redis.NewClient(...)
	// store := store.NewRedisStore(client)
	memStore := store.NewMemoryStore()

	// 1. Global Rate Limit
	// Protects the server from total overload
	globalConfig := ratelimit.Config{
		Rate:      100, // 100 req/sec total for the whole server
		Burst:     200,
		Window:    time.Second,
		Algorithm: ratelimit.TokenBucket,
		Store:     memStore,
		KeyFunc: func(ctx context.Context) (string, error) {
			return "global_limit", nil // Constant key for everyone
		},
	}
	globalMW, _ := middleware.NewHTTPMiddleware(globalConfig)

	// 2. Per-IP Rate Limit
	// Prevents individual users from hogging resources
	ipConfig := ratelimit.Config{
		Rate:      10, // 10 req/sec per IP
		Burst:     20,
		Window:    time.Second,
		Algorithm: ratelimit.TokenBucket,
		Store:     memStore,
		KeyFunc: func(ctx context.Context) (string, error) {
			// In a real app, you'd extract IP from request
			// Since we don't have request here, we simulate it or rely on upstream
			// For this example, let's assume a "X-Forwarded-For" or similar is handled
			// by a helper, or we just use a mock IP.
			return "ip:127.0.0.1", nil
		},
	}
	ipMW, _ := middleware.NewHTTPMiddleware(ipConfig)

	// 3. Setup Router
	mux := http.NewServeMux()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Request allowed!")) // #nosec
	})

	// Chain: Request -> Global Limit -> IP Limit -> Handler
	// If Global limit is hit, it returns 429 immediately.
	// If Global passes but IP limit is hit, it returns 429.
	mux.Handle("/", globalMW.Handler(ipMW.Handler(handler)))

	fmt.Println("Server started at :8081")
	http.ListenAndServe(":8081", mux) // #nosec
}
