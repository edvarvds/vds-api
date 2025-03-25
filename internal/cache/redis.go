package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"api_vds/internal/config"
)

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(cfg *config.Config) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		Username: cfg.Redis.Username,
		DB:       cfg.Redis.DB,
	})

	// Testar conexão
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	return &RedisClient{client: client}, nil
}

func (c *RedisClient) Get(key string) (string, error) {
	ctx := context.Background()
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("error getting key from Redis: %v", err)
	}
	return val, nil
}

func (c *RedisClient) Set(key string, value string, expiration time.Duration) error {
	ctx := context.Background()
	if err := c.client.Set(ctx, key, value, expiration).Err(); err != nil {
		return fmt.Errorf("error setting key in Redis: %v", err)
	}
	return nil
}

func (c *RedisClient) Delete(key string) error {
	ctx := context.Background()
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("error deleting key from Redis: %v", err)
	}
	return nil
}

func (c *RedisClient) Close() error {
	return c.client.Close()
} 