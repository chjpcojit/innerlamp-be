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
	// Import your generated proto package here
	// pb "path/to/your/proto"
)

func main() {
	// 1. Configure Rate Limiter
	config := ratelimit.Config{
		Rate:      100,
		Burst:     10,
		Window:    time.Minute,
		Algorithm: ratelimit.TokenBucket,
		Store:     store.NewMemoryStore(),
		KeyFunc: func(ctx context.Context) (string, error) {
			// Extract client ID from metadata (headers)
			md, ok := metadata.FromIncomingContext(ctx)
			if !ok {
				return "anonymous", nil
			}

			// Example: Limit by "client-id" header
			if ids := md.Get("client-id"); len(ids) > 0 {
				return "grpc_client:" + ids[0], nil
			}

			return "anonymous", nil
		},
	}

	// 2. Create gRPC Middleware
	mw, err := middleware.NewGRPCMiddleware(config)
	if err != nil {
		log.Fatalf("failed to create middleware: %v", err)
	}

	// 3. Setup gRPC Server with Interceptors
	server := grpc.NewServer(
		// Unary Interceptor (for normal RPCs)
		grpc.UnaryInterceptor(mw.UnaryInterceptor()),

		// Stream Interceptor (for streaming RPCs)
		grpc.StreamInterceptor(mw.StreamInterceptor()),
	)

	// Register your service implementation
	// pb.RegisterMyServiceServer(server, &myServiceImpl{})

	lis, err := net.Listen("tcp", ":50051") // #nosec
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	fmt.Println("gRPC Server started at :50051")
	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
