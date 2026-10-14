package middleware

import (
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"chjpcoj.com.innerlamp/pkg/utils"
)

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Started %q %q", r.Method, r.URL.Path) // #nosec
		next.ServeHTTP(w, r)
		log.Printf("Completed in %v", time.Since(start))
	})
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Preflight stops here and goes through the same envelope writer as
		// every other response, so a client parses OPTIONS with the same
		// utils.ApiResponse decoder it uses for real responses. The CORS
		// headers are set above, before WriteHeader, so they survive.
		if r.Method == "OPTIONS" {
			utils.WriteSuccessResponseWithStatus(w, http.StatusOK, nil)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Recovery returns middleware that recovers from panics and returns a 500 error.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				stack := debug.Stack()
				log.Printf("PANIC recovered: %v\n%s", rec, stack) // #nosec

				// Same envelope and code as any other unexpected failure, so a
				// panic is indistinguishable from a 500 to the client.
				utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
