package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"chjpcoj.com.innerlamp/pkg/ratelimit"
	"chjpcoj.com.innerlamp/pkg/ratelimit/middleware"
	"chjpcoj.com.innerlamp/pkg/ratelimit/store"
)

func main() {
	// 1. Configure Rate Limiter
	config := ratelimit.Config{
		Rate:      5, // Strict limit for login: 5 req/min
		Burst:     5,
		Window:    time.Minute,
		Algorithm: ratelimit.TokenBucket,
		Store:     store.NewMemoryStore(),
		KeyFunc: func(ctx context.Context) (string, error) {
			// 1. Check if this is the Login method
			method, _ := ctx.Value(middleware.ContextKeyGRPCMethod).(string)

			// Assuming your proto defines AuthService with Login method
			// Full method name format: /package.Service/Method
			if method == "/auth.AuthService/Login" {
				// 2. For Login, we limit by IP (simulated via metadata)
				// In real gRPC, you get IP from peer.Peer
				md, _ := metadata.FromIncomingContext(ctx)
				ip := "unknown"
				if ips := md.Get("x-forwarded-for"); len(ips) > 0 {
					ip = ips[0]
				}
				return "login_limit:" + ip, nil
			}

			// 3. For other methods, maybe no limit or a different key
			// Returning empty string or specific "skip" key depends on your strategy.
			// If you return a constant, they share a bucket.
			// If you want to SKIP rate limiting for others, you'd need logic in KeyFunc
			// or separate middleware.
			// Here we just give them a huge shared bucket or unique per method.
			return "other_methods:" + method, nil
		},
	}

	// 2. Create Middleware
	mw, err := middleware.NewGRPCMiddleware(config)
	if err != nil {
		log.Fatalf("failed to create middleware: %v", err)
	}

	// 3. Setup Server
	server := grpc.NewServer(
		grpc.UnaryInterceptor(mw.UnaryInterceptor()),
	)

	// Register services...

	lis, err := net.Listen("tcp", ":50052") // #nosec
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	fmt.Println("gRPC Login Server started at :50052")
	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
