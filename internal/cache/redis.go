package cache

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

const CacheTTL = 24 * time.Hour

// RedisCache wraps a Redis client for URL caching.
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache connects to Redis with retry logic.
func NewRedisCache() (*RedisCache, error) {
	addr := fmt.Sprintf("%s:%s",
		getEnv("REDIS_HOST", "redis"),
		getEnv("REDIS_PORT", "6379"),
	)

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     getEnv("REDIS_PASSWORD", ""),
		DB:           0,
		PoolSize:     50,
		MinIdleConns: 10,
	})

	ctx := context.Background()
	var err error
	for i := 0; i < 10; i++ {
		err = client.Ping(ctx).Err()
		if err == nil {
			break
		}
		log.Printf("Waiting for Redis... attempt %d/10", i+1)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("Connected to Redis")
	return &RedisCache{client: client}, nil
}

// Get retrieves a cached original URL by short code.
func (c *RedisCache) Get(ctx context.Context, shortCode string) (string, error) {
	return c.client.Get(ctx, cacheKey(shortCode)).Result()
}

// Set caches a short code -> original URL mapping.
func (c *RedisCache) Set(ctx context.Context, shortCode, originalURL string) error {
	return c.client.Set(ctx, cacheKey(shortCode), originalURL, CacheTTL).Err()
}

// Delete removes a cached entry.
func (c *RedisCache) Delete(ctx context.Context, shortCode string) error {
	return c.client.Del(ctx, cacheKey(shortCode)).Err()
}

// Ping checks Redis connectivity.
func (c *RedisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func cacheKey(shortCode string) string {
	return "url:" + shortCode
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
