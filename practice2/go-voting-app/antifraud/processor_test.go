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
