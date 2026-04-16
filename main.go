package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	redisclient "github.com/Wendoom-dev/Go-Rate-Limiter/redisclient"
)

const (
	maxTokens  = 10  // bucket capacity
	refillRate = 1.0 // tokens per second
)

// rateLimiter wraps an HTTP handler and enforces per-API-key rate limits.
func rateLimiter(rc *redisclient.RedisClient, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Identify the caller — use X-API-Key header, fall back to IP
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			apiKey = r.RemoteAddr
		}

		ctx := r.Context()
		result, err := rc.Allow(ctx, apiKey, maxTokens, refillRate)
		if err != nil {
			log.Printf("rate limiter error for key %q: %v", apiKey, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		// Always advertise remaining tokens
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(maxTokens))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%.2f", result.TokensLeft))

		if !result.Allowed {
			retryAfterSec := result.RetryAfterMs / 1000.0
			w.Header().Set("Retry-After", fmt.Sprintf("%.2f", retryAfterSec))
			http.Error(w,
				fmt.Sprintf("rate limit exceeded — retry after %.2fs", retryAfterSec),
				http.StatusTooManyRequests,
			)
			return
		}

		next(w, r)
	}
}

func checkHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK — request allowed")
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rc, err := redisclient.NewRedisClient(ctx)
	if err != nil {
		log.Fatal("Failed to initialize Redis client:", err)
	}
	log.Println("Lua script loaded with SHA:", rc.ScriptSHA)

	http.HandleFunc("/check", rateLimiter(rc, checkHandler))

	addr := ":8080"
	log.Println("Server running on", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
