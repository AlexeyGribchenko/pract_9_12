package antifraud

import (
	"context"
	"errors"
	"testing"
	"time"
	"voting-app/models"
)

func TestVoteProcessor_ProcessVote_Success(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := newProcessorTestRepo("active")
	cache := NewMockCache()
	processor := newTestVoteProcessor(repo, cache)

	success, message, err := processor.ProcessVote(ctx, &models.VoteEventMessage{
		PollID:   "poll-1",
		OptionID: "option-1",
		IP:       "192.168.1.10",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !success {
		t.Fatalf("expected success, got message %q", message)
	}
	if repo.recordedVotes != 1 {
		t.Errorf("expected one recorded vote, got %d", repo.recordedVotes)
	}

	count, err := cache.GetInt(ctx, "vote:count:option-1")
	if err != nil {
		t.Fatalf("unexpected cache error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected cached vote count 1, got %d", count)
	}
}

func TestVoteProcessor_ProcessVote_RejectsClosedPoll(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := newProcessorTestRepo("closed")
	cache := NewMockCache()
	processor := newTestVoteProcessor(repo, cache)

	success, message, err := processor.ProcessVote(ctx, &models.VoteEventMessage{
		PollID:   "poll-1",
		OptionID: "option-1",
		IP:       "192.168.1.10",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if success {
		t.Fatal("closed poll vote should be rejected")
	}
	if message != "Poll is not active" {
		t.Errorf("unexpected message: %s", message)
	}
	if repo.recordedVotes != 0 {
		t.Errorf("closed poll should not record votes, got %d", repo.recordedVotes)
	}
}

func TestVoteProcessor_ProcessVote_RejectsDatacenterIP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := newProcessorTestRepo("active")
	cache := NewMockCache()
	processor := newTestVoteProcessor(repo, cache)

	success, message, err := processor.ProcessVote(ctx, &models.VoteEventMessage{
		PollID:   "poll-1",
		OptionID: "option-1",
		IP:       "8.8.8.8",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if success {
		t.Fatal("datacenter IP vote should be rejected")
	}
	if message != "IP type not allowed: datacenter" {
		t.Errorf("unexpected message: %s", message)
	}
}

func TestVoteProcessor_ProcessVote_RejectsDuplicateVote(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := newProcessorTestRepo("active")
	cache := NewMockCache()
	processor := newTestVoteProcessor(repo, cache)
	deduplicator := NewDeduplicator(cache)

	if err := deduplicator.RecordVote(ctx, "poll-1", "192.168.1.10"); err != nil {
		t.Fatalf("unexpected error recording initial vote: %v", err)
	}

	success, message, err := processor.ProcessVote(ctx, &models.VoteEventMessage{
		PollID:   "poll-1",
		OptionID: "option-1",
		IP:       "192.168.1.10",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if success {
		t.Fatal("duplicate vote should be rejected")
	}
	if message != "You have already voted in this poll" {
		t.Errorf("unexpected message: %s", message)
	}
}

func TestVoteProcessor_ProcessVote_RejectsRateLimitedVote(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := newProcessorTestRepo("active")
	cache := NewMockCache()
	processor := newTestVoteProcessor(repo, cache)
	limiter := NewRateLimiter(cache)

	for i := 0; i < 5; i++ {
		if err := limiter.RecordVote(ctx, "192.168.1.10", "poll-1"); err != nil {
			t.Fatalf("unexpected rate limiter error: %v", err)
		}
	}

	success, message, err := processor.ProcessVote(ctx, &models.VoteEventMessage{
		PollID:   "poll-1",
		OptionID: "option-1",
		IP:       "192.168.1.10",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if success {
		t.Fatal("rate limited vote should be rejected")
	}
	if message != "Rate limit exceeded" {
		t.Errorf("unexpected message: %s", message)
	}
}

func TestVoteProcessor_ProcessVote_ReturnsRecordVoteError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	expectedErr := errors.New("record vote failed")
	repo := newProcessorTestRepo("active")
	repo.recordVoteErr = expectedErr
	cache := NewMockCache()
	processor := newTestVoteProcessor(repo, cache)

	success, message, err := processor.ProcessVote(ctx, &models.VoteEventMessage{
		PollID:   "poll-1",
		OptionID: "option-1",
		IP:       "192.168.1.10",
	})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected record vote error, got %v", err)
	}
	if success {
		t.Fatal("vote should not succeed when repository write fails")
	}
	if message != "Error recording vote" {
		t.Errorf("unexpected message: %s", message)
	}
}

func TestVoteProcessor_ProcessVote_ReturnsMissingPollError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := &processorTestRepo{}
	cache := NewMockCache()
	processor := newTestVoteProcessor(repo, cache)

	success, message, err := processor.ProcessVote(ctx, &models.VoteEventMessage{
		PollID:   "missing-poll",
		OptionID: "option-1",
		IP:       "192.168.1.10",
	})

	if err == nil {
		t.Fatal("expected missing poll error")
	}
	if success {
		t.Fatal("vote should not succeed for missing poll")
	}
	if message != "Internal error checking poll status" {
		t.Errorf("unexpected message: %s", message)
	}
}

func TestVoteProcessor_GetVoteStatus(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := newProcessorTestRepo("active")
	cache := NewMockCache()
	processor := newTestVoteProcessor(repo, cache)

	if err := NewDeduplicator(cache).RecordVote(ctx, "poll-1", "192.168.1.10"); err != nil {
		t.Fatalf("unexpected error recording vote: %v", err)
	}

	status := processor.GetVoteStatus(ctx, "poll-1", "192.168.1.10")
	if status["poll_id"] != "poll-1" {
		t.Errorf("unexpected poll id: %v", status["poll_id"])
	}
	if status["ip"] != "192.168.1.10" {
		t.Errorf("unexpected ip: %v", status["ip"])
	}
	if status["has_voted"] != true {
		t.Errorf("expected has_voted true, got %v", status["has_voted"])
	}
}

func newTestVoteProcessor(repo *processorTestRepo, cache *MockCache) *VoteProcessor {
	return NewVoteProcessor(
		repo,
		cache,
		NewGeoIPChecker(cache, "http://localhost:8080", "test-key"),
		NewRateLimiter(cache),
		NewDeduplicator(cache),
	)
}

type processorTestRepo struct {
	poll          *models.Poll
	recordedVotes int
	recordVoteErr error
}

func newProcessorTestRepo(status string) *processorTestRepo {
	return &processorTestRepo{
		poll: &models.Poll{
			ID:     "poll-1",
			Title:  "Test Poll",
			Status: status,
		},
	}
}

func (r *processorTestRepo) CreatePoll(title string) (*models.Poll, error) {
	return nil, nil
}

func (r *processorTestRepo) GetPoll(pollID string) (*models.Poll, error) {
	return r.poll, nil
}

func (r *processorTestRepo) GetPollWithOptions(pollID string) (*models.PollWithOptions, error) {
	return nil, nil
}

func (r *processorTestRepo) ClosePoll(pollID string, adminKey string) error {
	return nil
}

func (r *processorTestRepo) AddOption(pollID string, text string, order int) (*models.Option, error) {
	return nil, nil
}

func (r *processorTestRepo) GetOptions(pollID string) ([]*models.Option, error) {
	return nil, nil
}

func (r *processorTestRepo) GetPollConfig(pollID string) (*models.Poll, error) {
	return r.GetPoll(pollID)
}

func (r *processorTestRepo) GetResults(pollID string) (*models.PollResults, error) {
	return nil, nil
}

func (r *processorTestRepo) RecordVote(pollID string, optionID string, ip string) error {
	if r.recordVoteErr != nil {
		return r.recordVoteErr
	}
	r.recordedVotes++
	return nil
}

func (r *processorTestRepo) GetVote(pollID string, ip string) (*models.Vote, error) {
	return nil, nil
}

// Tests for MockCache methods
func TestMockCache_Del(t *testing.T) {
	ctx := context.Background()
	cache := NewMockCache()

	if err := cache.Set(ctx, "key1", "value1", 0); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if err := cache.Set(ctx, "key2", "value2", 0); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	if err := cache.Del(ctx, "key1", "key2"); err != nil {
		t.Fatalf("Del failed: %v", err)
	}

	_, err := cache.Get(ctx, "key1")
	if err == nil {
		t.Fatal("expected error after deleting key1")
	}

	_, err = cache.Get(ctx, "key2")
	if err == nil {
		t.Fatal("expected error after deleting key2")
	}
}

func TestMockCache_IncrBy(t *testing.T) {
	ctx := context.Background()
	cache := NewMockCache()

	val, err := cache.IncrBy(ctx, "counter", 5)
	if err != nil {
		t.Fatalf("IncrBy failed: %v", err)
	}
	if val != 5 {
		t.Errorf("expected 5, got %d", val)
	}

	val, err = cache.IncrBy(ctx, "counter", 3)
	if err != nil {
		t.Fatalf("IncrBy failed: %v", err)
	}
	if val != 8 {
		t.Errorf("expected 8, got %d", val)
	}
}

func TestMockCache_Close(t *testing.T) {
	cache := NewMockCache()
	err := cache.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestMockCache_Expire(t *testing.T) {
	ctx := context.Background()
	cache := NewMockCache()

	if err := cache.Set(ctx, "key", "value", 0); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	err := cache.Expire(ctx, "key", 1*time.Second)
	if err != nil {
		t.Errorf("Expire failed: %v", err)
	}

	val, err := cache.Get(ctx, "key")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if val != "value" {
		t.Errorf("expected 'value', got %s", val)
	}
}

func TestVoteProcessor_LogVoteEvent(t *testing.T) {
	repo := newProcessorTestRepo("active")
	cache := NewMockCache()
	processor := newTestVoteProcessor(repo, cache)

	vote := &models.VoteEventMessage{
		PollID:   "poll-1",
		OptionID: "option-1",
		IP:       "192.168.1.10",
	}
	
	// LogVoteEvent just logs, doesn't return anything, so we just verify it doesn't panic
	processor.LogVoteEvent(vote)
}

// Test recordVoteInCache
func TestVoteProcessor_RecordVoteInCache(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := newProcessorTestRepo("active")
	cache := NewMockCache()
	processor := newTestVoteProcessor(repo, cache)

	// Запись голоса
	err := processor.recordVoteInCache(ctx, "poll-1", "option-1", "192.168.1.10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Проверяем что голос записан в кэш
	count, err := cache.GetInt(ctx, "vote:count:option-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected cached count 1, got %d", count)
	}
}

// Test checkPollStatus
func TestVoteProcessor_CheckPollStatus(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := newProcessorTestRepo("active")
	cache := NewMockCache()
	processor := newTestVoteProcessor(repo, cache)

	status, err := processor.checkPollStatus(ctx, "poll-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status != "active" {
		t.Errorf("expected status 'active', got '%s'", status)
	}

	// Проверяем что статус кэшируется
	cached, err := cache.Get(ctx, "poll:status:poll-1")
	if err != nil {
		t.Fatalf("status should be cached: %v", err)
	}
	if cached != "active" {
		t.Errorf("expected cached status 'active', got '%s'", cached)
	}
}
