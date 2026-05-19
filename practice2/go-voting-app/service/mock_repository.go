package service

import (
	"database/sql"
	"errors"
	"sync"
	"voting-app/models"
)

// MockPollRepository - мок для PollRepository
type MockPollRepository struct {
	polls   map[string]*models.Poll
	options map[string][]*models.Option
	votes   map[string][]*models.Vote
	mu      sync.RWMutex
}

func NewMockPollRepository() *MockPollRepository {
	return &MockPollRepository{
		polls:   make(map[string]*models.Poll),
		options: make(map[string][]*models.Option),
		votes:   make(map[string][]*models.Vote),
	}
}

func (mpr *MockPollRepository) CreatePoll(title string) (*models.Poll, error) {
	mpr.mu.Lock()
	defer mpr.mu.Unlock()

	poll := &models.Poll{
		ID:       "test-poll-" + title,
		Title:    title,
		AdminKey: "admin-key",
		Status:   "active",
	}

	mpr.polls[poll.ID] = poll
	return poll, nil
}

func (mpr *MockPollRepository) GetPoll(pollID string) (*models.Poll, error) {
	mpr.mu.RLock()
	defer mpr.mu.RUnlock()

	poll, ok := mpr.polls[pollID]
	if !ok {
		return nil, nil
	}
	return poll, nil
}

func (mpr *MockPollRepository) GetPollWithOptions(pollID string) (*models.PollWithOptions, error) {
	mpr.mu.RLock()
	defer mpr.mu.RUnlock()

	poll, ok := mpr.polls[pollID]
	if !ok {
		return nil, nil
	}

	optionPtrs := mpr.options[pollID]
	// Конвертируем *models.Option в models.Option
	options := make([]models.Option, len(optionPtrs))
	for i, opt := range optionPtrs {
		options[i] = *opt
	}

	return &models.PollWithOptions{
		ID:        poll.ID,
		Title:     poll.Title,
		Status:    poll.Status,
		CreatedAt: poll.CreatedAt,
		ClosedAt:  poll.ClosedAt,
		Options:   options,
	}, nil
}

func (mpr *MockPollRepository) ClosePoll(pollID string, adminKey string) error {
	mpr.mu.Lock()
	defer mpr.mu.Unlock()

	poll, ok := mpr.polls[pollID]
	if !ok {
		return sql.ErrNoRows
	}

	if poll.AdminKey != adminKey {
		return sql.ErrNoRows
	}

	poll.Status = "closed"
	return nil
}

func (mpr *MockPollRepository) AddOption(pollID string, text string, order int) (*models.Option, error) {
	mpr.mu.Lock()
	defer mpr.mu.Unlock()

	option := &models.Option{
		ID:     "opt-" + text,
		PollID: pollID,
		Text:   text,
		Order:  order,
	}

	mpr.options[pollID] = append(mpr.options[pollID], option)
	return option, nil
}

func (mpr *MockPollRepository) GetOptions(pollID string) ([]*models.Option, error) {
	mpr.mu.RLock()
	defer mpr.mu.RUnlock()

	opts, ok := mpr.options[pollID]
	if !ok {
		return []*models.Option{}, nil
	}
	return opts, nil
}

func (mpr *MockPollRepository) GetPollConfig(pollID string) (*models.Poll, error) {
	return mpr.GetPoll(pollID)
}

func (mpr *MockPollRepository) GetResults(pollID string) (*models.PollResults, error) {
	mpr.mu.RLock()
	defer mpr.mu.RUnlock()

	poll, ok := mpr.polls[pollID]
	if !ok {
		return nil, nil
	}

	results := &models.PollResults{
		PollID:  poll.ID,
		Title:   poll.Title,
		Status:  poll.Status,
		Results: make(map[string]models.PollOptionCount),
	}

	// Подсчитываем голоса
	voteCount := make(map[string]int64)
	for _, vote := range mpr.votes[pollID] {
		voteCount[vote.OptionID]++
	}

	// Добавляем результаты
	for _, option := range mpr.options[pollID] {
		results.Results[option.ID] = models.PollOptionCount{
			Option: option.ID,
			Text:   option.Text,
			Count:  voteCount[option.ID],
		}
	}

	return results, nil
}

func (mpr *MockPollRepository) GetVoteCounts(pollID string) (map[string]int64, error) {
	mpr.mu.RLock()
	defer mpr.mu.RUnlock()

	voteCount := make(map[string]int64)
	for _, vote := range mpr.votes[pollID] {
		voteCount[vote.OptionID]++
	}

	return voteCount, nil
}

func (mpr *MockPollRepository) RecordVote(pollID string, optionID string, ip string) error {
	mpr.mu.Lock()
	defer mpr.mu.Unlock()

	vote := &models.Vote{
		ID:       "vote-" + ip + "-" + optionID,
		PollID:   pollID,
		OptionID: optionID,
		IP:       ip,
	}

	mpr.votes[pollID] = append(mpr.votes[pollID], vote)
	return nil
}

func (mpr *MockPollRepository) GetVote(pollID string, ip string) (*models.Vote, error) {
	mpr.mu.RLock()
	defer mpr.mu.RUnlock()

	for _, vote := range mpr.votes[pollID] {
		if vote.IP == ip {
			return vote, nil
		}
	}
	return nil, errors.New("vote not found")
}
