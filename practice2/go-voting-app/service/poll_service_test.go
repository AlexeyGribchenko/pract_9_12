package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
	"voting-app/antifraud"
	"voting-app/models"
)

func TestPollService_CreatePoll_Success(t *testing.T) {
	mockRepo := NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	service := NewPollService(mockRepo, mockCache)

	title := "Best Programming Language"
	options := []string{"Go", "Python", "Rust"}

	resp, err := service.CreatePoll(title, options)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.ID == "" {
		t.Error("poll ID should not be empty")
	}
	if resp.AdminKey == "" {
		t.Error("admin key should not be empty")
	}
	if resp.Title != title {
		t.Errorf("expected title '%s', got '%s'", title, resp.Title)
	}
}

func TestPollService_CreatePoll_AddsOptions(t *testing.T) {
	mockRepo := NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	service := NewPollService(mockRepo, mockCache)

	title := "Test Poll"
	options := []string{"Option A", "Option B", "Option C"}

	resp, err := service.CreatePoll(title, options)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Проверяем что опции добавлены
	opts, _ := mockRepo.GetOptions(resp.ID)
	if len(opts) != len(options) {
		t.Errorf("expected %d options, got %d", len(options), len(opts))
	}

	for i, opt := range opts {
		if opt.Text != options[i] {
			t.Errorf("expected option '%s', got '%s'", options[i], opt.Text)
		}
	}
}

func TestPollService_CreatePoll_CachesStatus(t *testing.T) {
	mockRepo := NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	service := NewPollService(mockRepo, mockCache)

	resp, err := service.CreatePoll("Cached Poll", []string{"A", "B"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status, err := mockCache.Get(ctx, "poll:status:"+resp.ID)
	if err != nil {
		t.Fatalf("expected cached status: %v", err)
	}
	if status != "active" {
		t.Errorf("expected cached status 'active', got '%s'", status)
	}
}

func TestPollService_CreatePoll_ReturnsCreateError(t *testing.T) {
	expectedErr := errors.New("create failed")
	repo := &errorPollRepository{createErr: expectedErr}
	mockCache := antifraud.NewMockCache()
	service := NewPollService(repo, mockCache)

	resp, err := service.CreatePoll("Broken Poll", []string{"A", "B"})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected create error, got %v", err)
	}
	if resp != nil {
		t.Error("response should be nil on create error")
	}
}

func TestPollService_CreatePoll_ReturnsAddOptionError(t *testing.T) {
	expectedErr := errors.New("add option failed")
	repo := &errorPollRepository{addOptionErr: expectedErr}
	mockCache := antifraud.NewMockCache()
	service := NewPollService(repo, mockCache)

	resp, err := service.CreatePoll("Broken Poll", []string{"A", "B"})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected add option error, got %v", err)
	}
	if resp != nil {
		t.Error("response should be nil on add option error")
	}
}

func TestPollService_GetPoll_NotFound(t *testing.T) {
	mockRepo := NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	service := NewPollService(mockRepo, mockCache)

	poll, err := service.GetPoll("non-existent-poll")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if poll != nil {
		t.Error("poll should be nil when not found")
	}
}

func TestPollService_GetPoll_Found(t *testing.T) {
	mockRepo := NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	service := NewPollService(mockRepo, mockCache)

	// Создаем опрос
	title := "Test Poll"
	resp, _ := service.CreatePoll(title, []string{"A", "B"})

	// Получаем опрос
	poll, err := service.GetPoll(resp.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if poll == nil {
		t.Error("poll should not be nil")
	}
	if poll.Title != title {
		t.Errorf("expected title '%s', got '%s'", title, poll.Title)
	}
}

func TestPollService_GetPoll_ReturnsRepositoryError(t *testing.T) {
	expectedErr := errors.New("get poll failed")
	repo := &errorPollRepository{getPollErr: expectedErr}
	mockCache := antifraud.NewMockCache()
	service := NewPollService(repo, mockCache)

	poll, err := service.GetPoll("poll-id")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
	if poll != nil {
		t.Error("poll should be nil on repository error")
	}
}

func TestPollService_ClosePoll_Success(t *testing.T) {
	mockRepo := NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	service := NewPollService(mockRepo, mockCache)

	// Создаем опрос
	resp, _ := service.CreatePoll("Test Poll", []string{"A", "B"})

	// Закрываем опрос
	err := service.ClosePoll(resp.ID, resp.AdminKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Проверяем что опрос закрыт
	poll, _ := service.GetPoll(resp.ID)
	if poll.Status != "closed" {
		t.Errorf("expected status 'closed', got '%s'", poll.Status)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cached, err := mockCache.Get(ctx, "poll:status:"+resp.ID)
	if err != nil {
		t.Fatalf("expected closed status in cache: %v", err)
	}
	if cached != "closed" {
		t.Errorf("expected cached status 'closed', got '%s'", cached)
	}
}

func TestPollService_ClosePoll_InvalidAdminKey(t *testing.T) {
	mockRepo := NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	service := NewPollService(mockRepo, mockCache)

	// Создаем опрос
	resp, _ := service.CreatePoll("Test Poll", []string{"A", "B"})

	// Пытаемся закрыть с неправильным ключом
	err := service.ClosePoll(resp.ID, "wrong-key")
	if err == nil {
		t.Error("should get error with wrong admin key")
	}
}

func TestPollService_ClosePoll_ReturnsRepositoryError(t *testing.T) {
	expectedErr := errors.New("close failed")
	repo := &errorPollRepository{closeErr: expectedErr}
	mockCache := antifraud.NewMockCache()
	service := NewPollService(repo, mockCache)

	err := service.ClosePoll("poll-id", "admin-key")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected close error, got %v", err)
	}
}

func TestPollService_GetResults_Empty(t *testing.T) {
	mockRepo := NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	service := NewPollService(mockRepo, mockCache)

	// Создаем опрос
	resp, _ := service.CreatePoll("Test Poll", []string{"A", "B"})

	// Получаем результаты
	results, err := service.GetResults(resp.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results == nil {
		t.Error("results should not be nil")
	}

	// Все опции должны иметь 0 голосов
	for _, result := range results.Results {
		if result.Count != 0 {
			t.Errorf("expected 0 votes for option %s, got %d", result.Option, result.Count)
		}
	}
}

func TestPollService_GetResults_NotFound(t *testing.T) {
	mockRepo := NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	service := NewPollService(mockRepo, mockCache)

	results, err := service.GetResults("missing-poll")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Error("results should be nil for missing poll")
	}
}

func TestPollService_GetResults_ReturnsPollMetadata(t *testing.T) {
	mockRepo := NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	service := NewPollService(mockRepo, mockCache)

	resp, err := service.CreatePoll("Metadata Poll", []string{"A", "B"})
	if err != nil {
		t.Fatalf("unexpected error creating poll: %v", err)
	}

	results, err := service.GetResults(resp.ID)
	if err != nil {
		t.Fatalf("unexpected error getting results: %v", err)
	}
	if results.PollID != resp.ID {
		t.Errorf("expected poll id '%s', got '%s'", resp.ID, results.PollID)
	}
	if results.Title != "Metadata Poll" {
		t.Errorf("expected title 'Metadata Poll', got '%s'", results.Title)
	}
	if results.Status != "active" {
		t.Errorf("expected status 'active', got '%s'", results.Status)
	}
	if len(results.Results) != 2 {
		t.Errorf("expected 2 result rows, got %d", len(results.Results))
	}
}

func TestPollService_GetResults_ReturnsRepositoryError(t *testing.T) {
	expectedErr := errors.New("get poll failed")
	repo := &errorPollRepository{getPollErr: expectedErr}
	mockCache := antifraud.NewMockCache()
	service := NewPollService(repo, mockCache)

	results, err := service.GetResults("poll-id")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
	if results != nil {
		t.Error("results should be nil on repository error")
	}
}

func TestPollService_GetResults_ReturnsOptionsError(t *testing.T) {
	expectedErr := errors.New("get options failed")
	repo := &errorPollRepository{getOptionsErr: expectedErr}
	mockCache := antifraud.NewMockCache()
	service := NewPollService(repo, mockCache)

	results, err := service.GetResults("poll-id")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected options error, got %v", err)
	}
	if results != nil {
		t.Error("results should be nil on options error")
	}
}

func TestPollService_GetResults_WithVotes(t *testing.T) {
	mockRepo := NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	service := NewPollService(mockRepo, mockCache)

	// Создаем опрос
	resp, _ := service.CreatePoll("Test Poll", []string{"Go", "Python", "Rust"})

	// Получаем опции
	opts, _ := mockRepo.GetOptions(resp.ID)

	// Убеждаемся что опции созданы
	if len(opts) != 3 {
		t.Fatalf("expected 3 options, got %d", len(opts))
	}

	// Записываем голоса
	err1 := mockRepo.RecordVote(resp.ID, opts[0].ID, "192.168.1.1")
	if err1 != nil {
		t.Fatalf("error recording first vote: %v", err1)
	}

	err2 := mockRepo.RecordVote(resp.ID, opts[0].ID, "192.168.1.2")
	if err2 != nil {
		t.Fatalf("error recording second vote: %v", err2)
	}

	err3 := mockRepo.RecordVote(resp.ID, opts[1].ID, "192.168.1.3")
	if err3 != nil {
		t.Fatalf("error recording third vote: %v", err3)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := mockCache.IncrBy(ctx, "vote:count:"+opts[0].ID, 2); err != nil {
		t.Fatalf("error updating first option cache count: %v", err)
	}
	if _, err := mockCache.IncrBy(ctx, "vote:count:"+opts[1].ID, 1); err != nil {
		t.Fatalf("error updating second option cache count: %v", err)
	}

	// Получаем результаты
	results, err := service.GetResults(resp.ID)
	if err != nil {
		t.Fatalf("error getting results: %v", err)
	}

	if results == nil {
		t.Fatal("results should not be nil")
	}

	firstCount := results.Results[opts[0].ID].Count
	secondCount := results.Results[opts[1].ID].Count

	if firstCount != 2 {
		t.Errorf("expected 2 votes for first option, got %d", firstCount)
	}
	if secondCount != 1 {
		t.Errorf("expected 1 vote for second option, got %d", secondCount)
	}
}

func TestPollService_GetPollStatus_ReturnsRepositoryError(t *testing.T) {
	expectedErr := errors.New("status lookup failed")
	repo := &errorPollRepository{getPollErr: expectedErr}
	mockCache := antifraud.NewMockCache()
	service := NewPollService(repo, mockCache)

	status, err := service.GetPollStatus("poll-id")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
	if status != "" {
		t.Errorf("expected empty status on error, got '%s'", status)
	}
}

func TestPollService_GetPollStatus_FallsBackToRepository(t *testing.T) {
	mockRepo := NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	service := NewPollService(mockRepo, mockCache)

	resp, err := service.CreatePoll("Fallback Poll", []string{"A", "B"})
	if err != nil {
		t.Fatalf("unexpected error creating poll: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cacheKey := "poll:status:" + resp.ID
	if err := mockCache.Del(ctx, cacheKey); err != nil {
		t.Fatalf("unexpected error clearing cache: %v", err)
	}

	status, err := service.GetPollStatus(resp.ID)
	if err != nil {
		t.Fatalf("unexpected error getting status: %v", err)
	}
	if status != "active" {
		t.Errorf("expected status 'active', got '%s'", status)
	}

	cached, err := mockCache.Get(ctx, cacheKey)
	if err != nil {
		t.Fatalf("expected status to be cached after repository fallback: %v", err)
	}
	if cached != "active" {
		t.Errorf("expected cached status 'active', got '%s'", cached)
	}
}

func TestPollService_GetPollStatus_NotFound(t *testing.T) {
	mockRepo := NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	service := NewPollService(mockRepo, mockCache)

	status, err := service.GetPollStatus("missing-poll")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "" {
		t.Errorf("expected empty status for missing poll, got '%s'", status)
	}
}

func TestPollService_GetPollWithOptions_Success(t *testing.T) {
	mockRepo := NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	service := NewPollService(mockRepo, mockCache)

	resp, err := service.CreatePoll("Options Poll", []string{"First", "Second"})
	if err != nil {
		t.Fatalf("unexpected error creating poll: %v", err)
	}

	poll, options, err := service.GetPollWithOptions(resp.ID)
	if err != nil {
		t.Fatalf("unexpected error getting poll with options: %v", err)
	}
	if poll == nil {
		t.Fatal("poll should not be nil")
	}
	if len(options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(options))
	}
	if options[0].Text != "First" || options[1].Text != "Second" {
		t.Errorf("unexpected option order: %+v", options)
	}
}

func TestPollService_GetPollWithOptions_NotFound(t *testing.T) {
	mockRepo := NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	service := NewPollService(mockRepo, mockCache)

	poll, options, err := service.GetPollWithOptions("missing-poll")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if poll != nil {
		t.Error("poll should be nil")
	}
	if options != nil {
		t.Error("options should be nil")
	}
}

func TestPollService_GetPollWithOptions_ReturnsPollError(t *testing.T) {
	expectedErr := errors.New("get poll failed")
	repo := &errorPollRepository{getPollErr: expectedErr}
	mockCache := antifraud.NewMockCache()
	service := NewPollService(repo, mockCache)

	poll, options, err := service.GetPollWithOptions("poll-id")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected poll error, got %v", err)
	}
	if poll != nil || options != nil {
		t.Error("poll and options should be nil on error")
	}
}

func TestPollService_GetPollWithOptions_ReturnsOptionsError(t *testing.T) {
	expectedErr := errors.New("get options failed")
	repo := &errorPollRepository{getOptionsErr: expectedErr}
	mockCache := antifraud.NewMockCache()
	service := NewPollService(repo, mockCache)

	poll, options, err := service.GetPollWithOptions("poll-id")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected options error, got %v", err)
	}
	if poll != nil {
		t.Error("poll should be nil when options lookup fails")
	}
	if options != nil {
		t.Error("options should be nil when options lookup fails")
	}
}

func TestPollService_GetPollConfig_Success(t *testing.T) {
	mockRepo := NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	service := NewPollService(mockRepo, mockCache)

	resp, err := service.CreatePoll("Config Poll", []string{"Yes", "No"})
	if err != nil {
		t.Fatalf("unexpected error creating poll: %v", err)
	}

	config, err := service.GetPollConfig(resp.ID)
	if err != nil {
		t.Fatalf("unexpected error getting poll config: %v", err)
	}
	if config == nil {
		t.Fatal("config should not be nil")
	}
	if config["id"] != resp.ID {
		t.Errorf("expected poll id '%s', got '%v'", resp.ID, config["id"])
	}
	if config["status"] != "active" {
		t.Errorf("expected status 'active', got '%v'", config["status"])
	}

	options, ok := config["options"].([]map[string]interface{})
	if !ok {
		t.Fatalf("expected options to be []map[string]interface{}, got %T", config["options"])
	}
	if len(options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(options))
	}
	if options[0]["text"] != "Yes" || options[1]["text"] != "No" {
		t.Errorf("unexpected config options: %+v", options)
	}
}

func TestPollService_GetPollConfig_NotFound(t *testing.T) {
	mockRepo := NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	service := NewPollService(mockRepo, mockCache)

	config, err := service.GetPollConfig("missing-poll")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config != nil {
		t.Error("config should be nil for missing poll")
	}
}

func TestPollService_GetPollConfig_ReturnsRepositoryError(t *testing.T) {
	expectedErr := errors.New("config failed")
	repo := &errorPollRepository{getPollErr: expectedErr}
	mockCache := antifraud.NewMockCache()
	service := NewPollService(repo, mockCache)

	config, err := service.GetPollConfig("poll-id")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
	if config != nil {
		t.Error("config should be nil on error")
	}
}

func TestPollService_CachePollConfig(t *testing.T) {
	mockRepo := NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	service := NewPollService(mockRepo, mockCache)

	resp, err := service.CreatePoll("Cached Config Poll", []string{"Left", "Right"})
	if err != nil {
		t.Fatalf("unexpected error creating poll: %v", err)
	}

	if err := service.CachePollConfig(resp.ID); err != nil {
		t.Fatalf("unexpected error caching poll config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cached, err := mockCache.Get(ctx, "poll:config:"+resp.ID)
	if err != nil {
		t.Fatalf("expected cached config: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(cached), &config); err != nil {
		t.Fatalf("cached config should be valid json: %v", err)
	}
	if config["id"] != resp.ID {
		t.Errorf("expected cached poll id '%s', got '%v'", resp.ID, config["id"])
	}
	if config["title"] != "Cached Config Poll" {
		t.Errorf("expected cached title, got '%v'", config["title"])
	}
}

func TestPollService_CachePollConfig_ReturnsConfigError(t *testing.T) {
	expectedErr := errors.New("config failed")
	repo := &errorPollRepository{getPollErr: expectedErr}
	mockCache := antifraud.NewMockCache()
	service := NewPollService(repo, mockCache)

	err := service.CachePollConfig("poll-id")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected config error, got %v", err)
	}
}

func TestPollService_GetPollStatus_Cached(t *testing.T) {
	mockRepo := NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	service := NewPollService(mockRepo, mockCache)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Создаем опрос
	resp, _ := service.CreatePoll("Test Poll", []string{"A", "B"})

	// Получаем статус
	status, err := service.GetPollStatus(resp.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status != "active" {
		t.Errorf("expected status 'active', got '%s'", status)
	}

	// Проверяем что статус в кэше
	cacheKey := "poll:status:" + resp.ID
	cached, _ := mockCache.Get(ctx, cacheKey)
	if cached != "active" {
		t.Errorf("status not cached, got '%s'", cached)
	}
}

func TestPollService_CreatePoll_WithEmptyOptions(t *testing.T) {
	mockRepo := NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	service := NewPollService(mockRepo, mockCache)

	// Пытаемся создать опрос без опций
	resp, _ := service.CreatePoll("Test Poll", []string{})

	// Должна быть ошибка или пустой результат
	if resp != nil && len(resp.ID) > 0 {
		// Проверяем что опции пусты
		opts, _ := mockRepo.GetOptions(resp.ID)
		if len(opts) != 0 {
			t.Error("poll should have no options")
		}
	}
}

type errorPollRepository struct {
	createErr     error
	addOptionErr  error
	getPollErr    error
	getOptionsErr error
	getCountsErr  error
	closeErr      error
}

func (e *errorPollRepository) CreatePoll(title string) (*models.Poll, error) {
	if e.createErr != nil {
		return nil, e.createErr
	}

	return &models.Poll{
		ID:       "poll-id",
		Title:    title,
		AdminKey: "admin-key",
		Status:   "active",
	}, nil
}

func (e *errorPollRepository) GetPoll(pollID string) (*models.Poll, error) {
	if e.getPollErr != nil {
		return nil, e.getPollErr
	}

	return &models.Poll{
		ID:       pollID,
		Title:    "Test Poll",
		AdminKey: "admin-key",
		Status:   "active",
	}, nil
}

func (e *errorPollRepository) GetPollWithOptions(pollID string) (*models.PollWithOptions, error) {
	return nil, nil
}

func (e *errorPollRepository) ClosePoll(pollID string, adminKey string) error {
	return e.closeErr
}

func (e *errorPollRepository) AddOption(pollID string, text string, order int) (*models.Option, error) {
	if e.addOptionErr != nil {
		return nil, e.addOptionErr
	}

	return &models.Option{
		ID:     "option-id",
		PollID: pollID,
		Text:   text,
		Order:  order,
	}, nil
}

func (e *errorPollRepository) GetOptions(pollID string) ([]*models.Option, error) {
	if e.getOptionsErr != nil {
		return nil, e.getOptionsErr
	}

	return []*models.Option{
		{
			ID:     "option-id",
			PollID: pollID,
			Text:   "Option",
			Order:  1,
		},
	}, nil
}

func (e *errorPollRepository) GetPollConfig(pollID string) (*models.Poll, error) {
	return e.GetPoll(pollID)
}

func (e *errorPollRepository) GetResults(pollID string) (*models.PollResults, error) {
	return nil, nil
}

func (e *errorPollRepository) GetVoteCounts(pollID string) (map[string]int64, error) {
	if e.getCountsErr != nil {
		return nil, e.getCountsErr
	}

	return map[string]int64{"option-id": 1}, nil
}

func (e *errorPollRepository) RecordVote(pollID string, optionID string, ip string) error {
	return nil
}

func (e *errorPollRepository) GetVote(pollID string, ip string) (*models.Vote, error) {
	return nil, nil
}
