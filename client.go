package redisclient

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Client    *redis.Client
	ScriptSHA string
}

func NewRedisClient(ctx context.Context) (*RedisClient, error) {
	// 1. Create Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// 2. Read Lua script from file
	script, err := os.ReadFile("script.lua")
	if err != nil {
		return nil, err
	}

	// 3. Load script into Redis (returns SHA)
	sha, err := rdb.ScriptLoad(ctx, string(script)).Result()
	if err != nil {
		return nil, err
	}

	return &RedisClient{
		Client:    rdb,
		ScriptSHA: sha,
	}, nil
}
