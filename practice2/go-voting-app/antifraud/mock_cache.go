package antifraud

import (
	"context"
	"errors"
	"sync"
	"time"
)

// CacheInterface определяет интерфейс для кэша
type CacheInterface interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	GetInt(ctx context.Context, key string) (int64, error)
	Incr(ctx context.Context, key string) (int64, error)
	Del(ctx context.Context, keys ...string) error
	SAdd(ctx context.Context, key string, members ...interface{}) error
	SIsMember(ctx context.Context, key string, member interface{}) (bool, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
	IncrBy(ctx context.Context, key string, increment int64) (int64, error)
	Close() error
}

// MockCache - мок Redis для тестирования
type MockCache struct {
	data sync.Map
}

func NewMockCache() *MockCache {
	return &MockCache{
		data: sync.Map{},
	}
}

func (mc *MockCache) Get(ctx context.Context, key string) (string, error) {
	val, ok := mc.data.Load(key)
	if !ok {
		return "", errors.New("key not found")
	}
	return val.(string), nil
}

func (mc *MockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	mc.data.Store(key, value)
	return nil
}

func (mc *MockCache) GetInt(ctx context.Context, key string) (int64, error) {
	val, ok := mc.data.Load(key)
	if !ok {
		return 0, nil
	}
	return val.(int64), nil
}

func (mc *MockCache) Incr(ctx context.Context, key string) (int64, error) {
	val, ok := mc.data.Load(key)
	if !ok {
		mc.data.Store(key, int64(1))
		return 1, nil
	}
	newVal := val.(int64) + 1
	mc.data.Store(key, newVal)
	return newVal, nil
}

func (mc *MockCache) Del(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		mc.data.Delete(key)
	}
	return nil
}

func (mc *MockCache) SAdd(ctx context.Context, key string, members ...interface{}) error {
	set, ok := mc.data.Load(key)
	if !ok {
		set = make(map[interface{}]bool)
	}
	setMap := set.(map[interface{}]bool)
	for _, member := range members {
		setMap[member] = true
	}
	mc.data.Store(key, setMap)
	return nil
}

func (mc *MockCache) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	set, ok := mc.data.Load(key)
	if !ok {
		return false, nil
	}
	return set.(map[interface{}]bool)[member], nil
}

func (mc *MockCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	// Mock doesn't handle expiration, just return nil
	return nil
}

func (mc *MockCache) IncrBy(ctx context.Context, key string, increment int64) (int64, error) {
	val, ok := mc.data.Load(key)
	if !ok {
		mc.data.Store(key, increment)
		return increment, nil
	}
	newVal := val.(int64) + increment
	mc.data.Store(key, newVal)
	return newVal, nil
}

func (mc *MockCache) Close() error {
	return nil
}
