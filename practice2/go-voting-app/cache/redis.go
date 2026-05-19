package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(host, port string) (*RedisCache, error) {
	addr := fmt.Sprintf("%s:%s", host, port)
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Ping(ctx).Err()
	if err != nil {
		return nil, err
	}

	log.Println("✓ Connected to Redis")
	return &RedisCache{client: client}, nil
}

func (rc *RedisCache) Get(ctx context.Context, key string) (string, error) {
	return rc.client.Get(ctx, key).Result()
}

func (rc *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return rc.client.Set(ctx, key, value, expiration).Err()
}

func (rc *RedisCache) Del(ctx context.Context, keys ...string) error {
	return rc.client.Del(ctx, keys...).Err()
}

func (rc *RedisCache) Incr(ctx context.Context, key string) (int64, error) {
	return rc.client.Incr(ctx, key).Result()
}

func (rc *RedisCache) IncrBy(ctx context.Context, key string, increment int64) (int64, error) {
	return rc.client.IncrBy(ctx, key, increment).Result()
}

func (rc *RedisCache) GetInt(ctx context.Context, key string) (int64, error) {
	val, err := rc.Get(ctx, key)
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, err
	}

	var result int64
	_, err = fmt.Sscanf(val, "%d", &result)
	return result, err
}

func (rc *RedisCache) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return rc.client.SAdd(ctx, key, members...).Err()
}

func (rc *RedisCache) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return rc.client.SIsMember(ctx, key, member).Result()
}

func (rc *RedisCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return rc.client.Expire(ctx, key, expiration).Err()
}

func (rc *RedisCache) Close() error {
	return rc.client.Close()
}
