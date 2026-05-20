package antifraud

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestRateLimiter_CheckLimit_AllowFirstVote(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockCache := NewMockCache()
	limiter := NewRateLimiter(mockCache)

	ip := "192.168.1.100"
	pollID := "poll-1"

	allowed, err := limiter.CheckLimit(ctx, ip, pollID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !allowed {
		t.Errorf("first vote should be allowed")
	}
}

func TestRateLimiter_CheckLimit_EnforceMinuteLimit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockCache := NewMockCache()
	limiter := NewRateLimiter(mockCache)

	ip := "192.168.1.100"
	pollID := "poll-1"

	// Проверяем, что пятый голос разрешен
	for i := 0; i < 5; i++ {
		allowed, err := limiter.CheckLimit(ctx, ip, pollID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Errorf("vote %d should be allowed", i+1)
		}
		limiter.RecordVote(ctx, ip, pollID)
	}

	// Проверяем, что шестой голос запрещен
	allowed, err := limiter.CheckLimit(ctx, ip, pollID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Errorf("vote beyond limit should be denied")
	}
}

func TestRateLimiter_RecordVote_IncrementCounters(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockCache := NewMockCache()
	limiter := NewRateLimiter(mockCache)

	ip := "192.168.1.100"
	pollID := "poll-1"

	// Проверяем начальное значение
	count, _ := mockCache.GetInt(ctx, "ratelimit:minute:"+ip+":"+pollID)
	if count != 0 {
		t.Errorf("expected initial count 0, got %d", count)
	}

	// Записываем голос
	err := limiter.RecordVote(ctx, ip, pollID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Проверяем, что счетчик увеличился
	count, _ = mockCache.GetInt(ctx, "ratelimit:minute:"+ip+":"+pollID)
	if count != 1 {
		t.Errorf("expected count 1 after first vote, got %d", count)
	}
}

func TestRateLimiter_DifferentIPs_IndependentLimits(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockCache := NewMockCache()
	limiter := NewRateLimiter(mockCache)

	pollID := "poll-1"

	ip1 := "192.168.1.100"
	ip2 := "192.168.1.101"

	// Первый IP голосует 5 раз
	for i := 0; i < 5; i++ {
		limiter.CheckLimit(ctx, ip1, pollID)
		limiter.RecordVote(ctx, ip1, pollID)
	}

	// Первый IP достиг лимита
	allowed, _ := limiter.CheckLimit(ctx, ip1, pollID)
	if allowed {
		t.Error("first IP should reach limit after 5 votes")
	}

	// Второй IP должен иметь возможность голосовать
	allowed, _ = limiter.CheckLimit(ctx, ip2, pollID)
	if !allowed {
		t.Error("second IP should not be affected by first IP's votes")
	}
}

func TestRateLimiter_DifferentPolls_IndependentLimits(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockCache := NewMockCache()
	limiter := NewRateLimiter(mockCache)

	ip := "192.168.1.100"
	poll1 := "poll-1"
	poll2 := "poll-2"

	// Голосуем в первом опросе 5 раз
	for i := 0; i < 5; i++ {
		limiter.CheckLimit(ctx, ip, poll1)
		limiter.RecordVote(ctx, ip, poll1)
	}

	// Первый опрос достиг лимита
	allowed, _ := limiter.CheckLimit(ctx, ip, poll1)
	if allowed {
		t.Error("first poll should reach limit")
	}

	// Второй опрос должен быть отдельным
	allowed, _ = limiter.CheckLimit(ctx, ip, poll2)
	if !allowed {
		t.Error("second poll should have independent limit")
	}
}

// ===== Deduplicator Tests =====

func TestDeduplicator_FirstVote_NotRecorded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockCache := NewMockCache()
	dedup := NewDeduplicator(mockCache)

	pollID := "poll-1"
	ip := "192.168.1.100"

	// Проверяем, что IP еще не голосовал
	hasVoted, err := dedup.HasVoted(ctx, pollID, ip)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hasVoted {
		t.Error("IP should not have voted yet")
	}
}

func TestDeduplicator_RecordVote_MarksAsVoted(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockCache := NewMockCache()
	dedup := NewDeduplicator(mockCache)

	pollID := "poll-1"
	ip := "192.168.1.100"

	// Записываем голос
	err := dedup.RecordVote(ctx, pollID, ip)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Проверяем, что IP теперь отмечен как проголосовавший
	hasVoted, err := dedup.HasVoted(ctx, pollID, ip)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !hasVoted {
		t.Error("IP should be marked as voted after RecordVote")
	}
}

func TestDeduplicator_PreventsDuplicate_SameIPAndPoll(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockCache := NewMockCache()
	dedup := NewDeduplicator(mockCache)

	pollID := "poll-1"
	ip := "192.168.1.100"

	// Записываем голос
	dedup.RecordVote(ctx, pollID, ip)

	// Проверяем, что повторный голос запрещен
	hasVoted, _ := dedup.HasVoted(ctx, pollID, ip)
	if !hasVoted {
		t.Error("duplicate vote should be detected")
	}
}

func TestDeduplicator_DifferentPolls_IndependentTracking(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockCache := NewMockCache()
	dedup := NewDeduplicator(mockCache)

	ip := "192.168.1.100"
	poll1 := "poll-1"
	poll2 := "poll-2"

	// Записываем голос в первом опросе
	dedup.RecordVote(ctx, poll1, ip)

	// Проверяем, что IP отмечен в первом опросе
	hasVoted, _ := dedup.HasVoted(ctx, poll1, ip)
	if !hasVoted {
		t.Error("IP should be marked as voted in first poll")
	}

	// Проверяем, что IP может голосовать во втором опросе
	hasVoted, _ = dedup.HasVoted(ctx, poll2, ip)
	if hasVoted {
		t.Error("IP should not be marked as voted in second poll")
	}

	// Записываем голос во втором опросе
	dedup.RecordVote(ctx, poll2, ip)

	// Проверяем, что IP теперь отмечен во втором опросе
	hasVoted, _ = dedup.HasVoted(ctx, poll2, ip)
	if !hasVoted {
		t.Error("IP should be marked as voted in second poll after recording")
	}
}

func TestDeduplicator_DifferentIPs_IndependentTracking(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockCache := NewMockCache()
	dedup := NewDeduplicator(mockCache)

	pollID := "poll-1"
	ip1 := "192.168.1.100"
	ip2 := "192.168.1.101"

	// Записываем голос первого IP
	dedup.RecordVote(ctx, pollID, ip1)

	// Проверяем, что первый IP отмечен
	hasVoted1, _ := dedup.HasVoted(ctx, pollID, ip1)
	if !hasVoted1 {
		t.Error("first IP should be marked as voted")
	}

	// Проверяем, что второй IP может голосовать
	hasVoted2, _ := dedup.HasVoted(ctx, pollID, ip2)
	if hasVoted2 {
		t.Error("second IP should not be marked as voted")
	}
}

// Additional tests for better coverage

func TestRateLimiter_CheckLimit_ThirdVote(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockCache := NewMockCache()
	limiter := NewRateLimiter(mockCache)

	ip := "192.168.1.100"
	pollID := "poll-1"

	// Проверяем третий голос
	limiter.CheckLimit(ctx, ip, pollID)
	limiter.RecordVote(ctx, ip, pollID)

	limiter.CheckLimit(ctx, ip, pollID)
	limiter.RecordVote(ctx, ip, pollID)

	allowed, err := limiter.CheckLimit(ctx, ip, pollID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Errorf("third vote should be allowed")
	}
}

func TestRateLimiter_RecordVote_CacheIncrement(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockCache := NewMockCache()
	limiter := NewRateLimiter(mockCache)

	ip := "192.168.1.100"
	pollID := "poll-1"

	// Запись нескольких голосов
	for i := 0; i < 3; i++ {
		err := limiter.RecordVote(ctx, ip, pollID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// Проверяем итоговое значение
	count, _ := mockCache.GetInt(ctx, "ratelimit:minute:"+ip+":"+pollID)
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}
}

// Hour limit tests
func TestRateLimiter_CheckLimit_HourLimit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockCache := NewMockCache()
	limiter := NewRateLimiter(mockCache)

	ip := "192.168.1.100"
	pollID := "poll-1"

	// Set hour limit to max
	hourKey := fmt.Sprintf("ratelimit:hour:%s:%s", ip, pollID)
	mockCache.IncrBy(ctx, hourKey, 60) // Max per hour

	allowed, err := limiter.CheckLimit(ctx, ip, pollID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("should deny vote when hour limit reached")
	}
}

// Day limit tests
func TestRateLimiter_CheckLimit_DayLimit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockCache := NewMockCache()
	limiter := NewRateLimiter(mockCache)

	ip := "192.168.1.100"
	pollID := "poll-1"

	// Set day limit to max
	dayKey := fmt.Sprintf("ratelimit:day:%s:%s", ip, pollID)
	mockCache.IncrBy(ctx, dayKey, 300) // Max per day

	allowed, err := limiter.CheckLimit(ctx, ip, pollID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("should deny vote when day limit reached")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Deduplicator edge case tests
func TestDeduplicator_SAdd_And_SIsMember(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockCache := NewMockCache()

	setKey := "test:set"
	members := []interface{}{"member1", "member2", "member3"}

	err := mockCache.SAdd(ctx, setKey, members...)
	if err != nil {
		t.Fatalf("SAdd failed: %v", err)
	}

	for _, member := range members {
		isMember, err := mockCache.SIsMember(ctx, setKey, member)
		if err != nil {
			t.Fatalf("SIsMember failed: %v", err)
		}
		if !isMember {
			t.Errorf("member %v should be in set", member)
		}
	}

	notMember, err := mockCache.SIsMember(ctx, setKey, "nonexistent")
	if err != nil {
		t.Fatalf("SIsMember failed: %v", err)
	}
	if notMember {
		t.Error("nonexistent member should not be in set")
	}
}
