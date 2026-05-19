package antifraud

import (
	"context"
	"fmt"
	"log"
	"time"
)

// RateLimiter ограничивает частоту голосов с одного IP
type RateLimiter struct {
	cache             CacheInterface
	maxVotesPerMinute int
	maxVotesPerHour   int
	maxVotesPerDay    int
}

func NewRateLimiter(cache CacheInterface) *RateLimiter {
	return &RateLimiter{
		cache:             cache,
		maxVotesPerMinute: 5,
		maxVotesPerHour:   50,
		maxVotesPerDay:    200,
	}
}

// CheckLimit проверяет, не превышен ли лимит голосов с данного IP
func (rl *RateLimiter) CheckLimit(ctx context.Context, ip string, pollID string) (bool, error) {
	// Проверяем лимит за минуту
	minuteKey := fmt.Sprintf("ratelimit:minute:%s:%s", ip, pollID)
	minuteCount, err := rl.cache.GetInt(ctx, minuteKey)
	if err != nil {
		return false, err
	}

	if minuteCount >= int64(rl.maxVotesPerMinute) {
		log.Printf("Rate limit exceeded for %s (minute limit)", ip)
		return false, nil
	}

	// Проверяем лимит за час
	hourKey := fmt.Sprintf("ratelimit:hour:%s:%s", ip, pollID)
	hourCount, err := rl.cache.GetInt(ctx, hourKey)
	if err != nil {
		return false, err
	}

	if hourCount >= int64(rl.maxVotesPerHour) {
		log.Printf("Rate limit exceeded for %s (hour limit)", ip)
		return false, nil
	}

	// Проверяем лимит за день
	dayKey := fmt.Sprintf("ratelimit:day:%s:%s", ip, pollID)
	dayCount, err := rl.cache.GetInt(ctx, dayKey)
	if err != nil {
		return false, err
	}

	if dayCount >= int64(rl.maxVotesPerDay) {
		log.Printf("Rate limit exceeded for %s (day limit)", ip)
		return false, nil
	}

	return true, nil
}

// RecordVote записывает голос для ограничения частоты
func (rl *RateLimiter) RecordVote(ctx context.Context, ip string, pollID string) error {
	minuteKey := fmt.Sprintf("ratelimit:minute:%s:%s", ip, pollID)
	hourKey := fmt.Sprintf("ratelimit:hour:%s:%s", ip, pollID)
	dayKey := fmt.Sprintf("ratelimit:day:%s:%s", ip, pollID)

	if _, err := rl.cache.Incr(ctx, minuteKey); err != nil {
		return err
	}
	if _, err := rl.cache.Incr(ctx, hourKey); err != nil {
		return err
	}
	if _, err := rl.cache.Incr(ctx, dayKey); err != nil {
		return err
	}

	if err := rl.cache.Expire(ctx, minuteKey, 1*time.Minute); err != nil {
		return err
	}
	if err := rl.cache.Expire(ctx, hourKey, 1*time.Hour); err != nil {
		return err
	}
	if err := rl.cache.Expire(ctx, dayKey, 24*time.Hour); err != nil {
		return err
	}

	return nil
}

// ===================================

// Deduplicator проверяет дублирующиеся голоса
type Deduplicator struct {
	cache CacheInterface
}

func NewDeduplicator(cache CacheInterface) *Deduplicator {
	return &Deduplicator{cache: cache}
}

// HasVoted проверяет, голосовал ли данный IP в опросе
func (d *Deduplicator) HasVoted(ctx context.Context, pollID string, ip string) (bool, error) {
	setKey := fmt.Sprintf("votes:ips:%s", pollID)
	return d.cache.SIsMember(ctx, setKey, ip)
}

// RecordVote записывает IP как проголосовавший
func (d *Deduplicator) RecordVote(ctx context.Context, pollID string, ip string) error {
	setKey := fmt.Sprintf("votes:ips:%s", pollID)
	err := d.cache.SAdd(ctx, setKey, ip)
	if err != nil {
		return err
	}

	// Устанавливаем TTL для множества (полтора месяца)
	return d.cache.Expire(ctx, setKey, 45*24*time.Hour)
}
