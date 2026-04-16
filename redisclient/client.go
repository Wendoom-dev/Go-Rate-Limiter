package redisclient

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Client    *redis.Client
	ScriptSHA string
}

type RateLimitResult struct {
	Allowed      bool
	TokensLeft   float64
	RetryAfterMs float64
}

func NewRedisClient(ctx context.Context) (*RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Ping to verify connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Read Lua script from file (relative to where the binary is run)
	script, err := os.ReadFile("script.lua")
	if err != nil {
		return nil, fmt.Errorf("failed to read script.lua: %w", err)
	}

	// Load script into Redis — returns SHA1 for future EVALSHA calls
	sha, err := rdb.ScriptLoad(ctx, string(script)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load Lua script: %w", err)
	}

	return &RedisClient{
		Client:    rdb,
		ScriptSHA: sha,
	}, nil
}

// Allow checks whether the given API key is within its rate limit.
// maxTokens = bucket capacity, refillRate = tokens per second.
func (rc *RedisClient) Allow(ctx context.Context, apiKey string, maxTokens int, refillRate float64) (*RateLimitResult, error) {
	key := fmt.Sprintf("rate_limit:%s", apiKey)
	nowMs := time.Now().UnixMilli()

	// EVALSHA runs the pre-loaded Lua script by its SHA
	res, err := rc.Client.EvalSha(ctx, rc.ScriptSHA,
		[]string{key}, // KEYS[1]
		maxTokens,     // ARGV[1]
		refillRate,    // ARGV[2]
		nowMs,         // ARGV[3]
	).Slice()
	if err != nil {
		return nil, fmt.Errorf("evalsha error: %w", err)
	}

	// Lua returns: {allowed (0/1), tokens_left, retry_after_ms}
	allowed, _ := res[0].(int64)
	tokens, _ := res[1].(float64) // may come back as int64 if whole number
	retryMs, _ := res[2].(float64)

	// Redis Lua integers come back as int64, not float64 — handle both
	if tokens == 0 {
		if t, ok := res[1].(int64); ok {
			tokens = float64(t)
		}
	}
	if retryMs == 0 {
		if t, ok := res[2].(int64); ok {
			retryMs = float64(t)
		}
	}

	return &RateLimitResult{
		Allowed:      allowed == 1,
		TokensLeft:   tokens,
		RetryAfterMs: retryMs,
	}, nil
}
