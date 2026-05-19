package service

import (
	"context"
	"encoding/json"
	"log"
	"time"
	"voting-app/antifraud"
	"voting-app/models"
	"voting-app/repository"
)

type PollService struct {
	repo  repository.PollServiceRepository
	cache antifraud.CacheInterface
}

func NewPollService(repo repository.PollServiceRepository, cache antifraud.CacheInterface) *PollService {
	return &PollService{repo: repo, cache: cache}
}

func (ps *PollService) CreatePoll(title string, options []string) (*models.CreatePollResponse, error) {
	poll, err := ps.repo.CreatePoll(title)
	if err != nil {
		return nil, err
	}

	// Добавить варианты ответов
	for i, optionText := range options {
		_, err := ps.repo.AddOption(poll.ID, optionText, i+1)
		if err != nil {
			log.Printf("Error adding option: %v", err)
			return nil, err
		}
	}

	// Кэшировать статус опроса в Redis
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cacheKey := "poll:status:" + poll.ID
	err = ps.cache.Set(ctx, cacheKey, "active", 24*time.Hour)
	if err != nil {
		log.Printf("Error caching poll status: %v", err)
	}

	return &models.CreatePollResponse{
		ID:       poll.ID,
		AdminKey: poll.AdminKey,
		Title:    poll.Title,
	}, nil
}

func (ps *PollService) GetPoll(pollID string) (*models.Poll, error) {
	return ps.repo.GetPoll(pollID)
}

func (ps *PollService) GetPollStatus(pollID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cacheKey := "poll:status:" + pollID

	// Пытаемся получить из кэша
	status, err := ps.cache.Get(ctx, cacheKey)
	if err == nil {
		return status, nil
	}

	// Если не в кэше, получаем из БД
	poll, err := ps.repo.GetPoll(pollID)
	if err != nil {
		return "", err
	}

	if poll == nil {
		return "", nil
	}

	if err := ps.cache.Set(ctx, cacheKey, poll.Status, 24*time.Hour); err != nil {
		log.Printf("Error caching poll status: %v", err)
	}

	return poll.Status, nil
}

func (ps *PollService) ClosePoll(pollID string, adminKey string) error {
	err := ps.repo.ClosePoll(pollID, adminKey)
	if err != nil {
		return err
	}

	// Обновить кэш
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cacheKey := "poll:status:" + pollID
	if err := ps.cache.Set(ctx, cacheKey, "closed", 24*time.Hour); err != nil {
		log.Printf("Error caching closed poll status: %v", err)
	}

	return nil
}

func (ps *PollService) GetResults(pollID string) (*models.PollResults, error) {
	poll, err := ps.repo.GetPoll(pollID)
	if err != nil {
		return nil, err
	}

	if poll == nil {
		return nil, nil
	}

	options, err := ps.repo.GetOptions(pollID)
	if err != nil {
		return nil, err
	}

	// Получить счеты из Redis для быстрого ответа
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := make(map[string]models.PollOptionCount)
	var voteCounts map[string]int64

	getVoteCounts := func() map[string]int64 {
		if voteCounts != nil {
			return voteCounts
		}

		counts, err := ps.repo.GetVoteCounts(pollID)
		if err != nil {
			log.Printf("Error getting vote counts from repository: %v", err)
			voteCounts = map[string]int64{}
			return voteCounts
		}

		voteCounts = counts
		return voteCounts
	}

	for _, option := range options {
		countKey := "vote:count:" + option.ID
		count, err := ps.cache.GetInt(ctx, countKey)
		if err != nil {
			log.Printf("Error getting count from cache: %v", err)
			count = getVoteCounts()[option.ID]
		} else if count == 0 {
			if dbCount := getVoteCounts()[option.ID]; dbCount > 0 {
				count = dbCount
			}
		}

		results[option.ID] = models.PollOptionCount{
			Option: option.ID,
			Text:   option.Text,
			Count:  count,
		}
	}

	return &models.PollResults{
		PollID:  poll.ID,
		Title:   poll.Title,
		Status:  poll.Status,
		Results: results,
	}, nil
}

func (ps *PollService) GetPollWithOptions(pollID string) (*models.Poll, []models.Option, error) {
	poll, err := ps.repo.GetPoll(pollID)
	if err != nil {
		return nil, nil, err
	}

	if poll == nil {
		return nil, nil, nil
	}

	optionPtrs, err := ps.repo.GetOptions(pollID)
	if err != nil {
		return nil, nil, err
	}

	// Конвертируем []*models.Option в []models.Option
	options := make([]models.Option, len(optionPtrs))
	for i, opt := range optionPtrs {
		options[i] = *opt
	}

	return poll, options, nil
}

// GetPollConfig возвращает конфигурацию опроса для сервиса Anti-Fraud
func (ps *PollService) GetPollConfig(pollID string) (map[string]interface{}, error) {
	poll, options, err := ps.GetPollWithOptions(pollID)
	if err != nil {
		return nil, err
	}

	if poll == nil {
		return nil, nil
	}

	optionsList := make([]map[string]interface{}, len(options))
	for i, opt := range options {
		optionsList[i] = map[string]interface{}{
			"id":   opt.ID,
			"text": opt.Text,
		}
	}

	return map[string]interface{}{
		"id":      poll.ID,
		"title":   poll.Title,
		"status":  poll.Status,
		"options": optionsList,
		"created": poll.CreatedAt,
	}, nil
}

// CachePollConfig кэширует конфигурацию опроса
func (ps *PollService) CachePollConfig(pollID string) error {
	config, err := ps.GetPollConfig(pollID)
	if err != nil {
		return err
	}
	if config == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	cacheKey := "poll:config:" + pollID
	return ps.cache.Set(ctx, cacheKey, string(data), 24*time.Hour)
}
