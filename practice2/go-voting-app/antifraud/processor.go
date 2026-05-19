package antifraud

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"voting-app/models"
	"voting-app/repository"
)

// VoteProcessor обрабатывает голоса с проверками
type VoteProcessor struct {
	repo         repository.VoteProcessorRepository
	cache        CacheInterface
	geoipChecker *GeoIPChecker
	rateLimiter  *RateLimiter
	deduplicator *Deduplicator
}

func NewVoteProcessor(
	repo repository.VoteProcessorRepository,
	cache CacheInterface,
	geoipChecker *GeoIPChecker,
	rateLimiter *RateLimiter,
	deduplicator *Deduplicator,
) *VoteProcessor {
	return &VoteProcessor{
		repo:         repo,
		cache:        cache,
		geoipChecker: geoipChecker,
		rateLimiter:  rateLimiter,
		deduplicator: deduplicator,
	}
}

// ProcessVote обрабатывает голос с полной цепочкой проверок
func (vp *VoteProcessor) ProcessVote(ctx context.Context, vote *models.VoteEventMessage) (bool, string, error) {
	log.Printf("Processing vote: Poll=%s, Option=%s, IP=%s", vote.PollID, vote.OptionID, vote.IP)

	// 1. Проверка статуса опроса
	status, err := vp.checkPollStatus(ctx, vote.PollID)
	if err != nil {
		return false, "Internal error checking poll status", err
	}

	if status != "active" {
		log.Printf("Poll %s is not active (status: %s)", vote.PollID, status)
		return false, "Poll is not active", nil
	}

	// 2. Проверка IP (GeoIP)
	ipType, err := vp.geoipChecker.CheckIP(ctx, vote.IP)
	if err != nil {
		log.Printf("Error checking IP: %v", err)
		return false, "Error checking IP", err
	}

	if ipType != "residential" {
		log.Printf("IP %s is not residential (type: %s)", vote.IP, ipType)
		return false, fmt.Sprintf("IP type not allowed: %s", ipType), nil
	}

	// 3. Проверка дедупликации (не голосовал ли уже)
	hasVoted, err := vp.deduplicator.HasVoted(ctx, vote.PollID, vote.IP)
	if err != nil {
		log.Printf("Error checking if voted: %v", err)
		return false, "Internal error", err
	}

	if hasVoted {
		log.Printf("IP %s already voted in poll %s", vote.IP, vote.PollID)
		return false, "You have already voted in this poll", nil
	}

	// 4. Проверка Rate Limiting
	allowed, err := vp.rateLimiter.CheckLimit(ctx, vote.IP, vote.PollID)
	if err != nil {
		log.Printf("Error checking rate limit: %v", err)
		return false, "Internal error", err
	}

	if !allowed {
		log.Printf("Rate limit exceeded for IP %s", vote.IP)
		return false, "Rate limit exceeded", nil
	}

	// 5. Запись голоса
	err = vp.repo.RecordVote(vote.PollID, vote.OptionID, vote.IP)
	if err != nil {
		log.Printf("Error recording vote: %v", err)
		return false, "Error recording vote", err
	}

	// 6. Обновление счетчиков в Redis
	err = vp.recordVoteInCache(ctx, vote.PollID, vote.OptionID, vote.IP)
	if err != nil {
		log.Printf("Error updating cache: %v", err)
		// Не возвращаем ошибку, так как голос уже записан в БД
	}

	// 7. Запись IP как проголосовавшего
	err = vp.deduplicator.RecordVote(ctx, vote.PollID, vote.IP)
	if err != nil {
		log.Printf("Error recording vote in deduplicator: %v", err)
	}

	// 8. Запись в Rate Limiter
	err = vp.rateLimiter.RecordVote(ctx, vote.IP, vote.PollID)
	if err != nil {
		log.Printf("Error recording vote in rate limiter: %v", err)
	}

	log.Printf("Vote successfully processed for poll %s", vote.PollID)
	return true, "Vote recorded", nil
}

// checkPollStatus проверяет статус опроса
func (vp *VoteProcessor) checkPollStatus(ctx context.Context, pollID string) (string, error) {
	// Пытаемся получить статус из кэша
	cacheKey := "poll:status:" + pollID
	status, err := vp.cache.Get(ctx, cacheKey)
	if err == nil {
		return status, nil
	}

	// Если не в кэше, получаем из БД
	poll, err := vp.repo.GetPoll(pollID)
	if err != nil {
		return "", err
	}

	if poll == nil {
		return "", fmt.Errorf("poll not found: %s", pollID)
	}

	if err := vp.cache.Set(ctx, cacheKey, poll.Status, 24*time.Hour); err != nil {
		log.Printf("Error caching poll status: %v", err)
	}

	return poll.Status, nil
}

// recordVoteInCache обновляет счетчики голосов в Redis
func (vp *VoteProcessor) recordVoteInCache(ctx context.Context, pollID string, optionID string, ip string) error {
	// Увеличиваем счетчик голосов для варианта
	countKey := fmt.Sprintf("vote:count:%s", optionID)
	_, err := vp.cache.Incr(ctx, countKey)
	if err != nil {
		return err
	}

	if err := vp.cache.Expire(ctx, countKey, 45*24*time.Hour); err != nil {
		return err
	}

	return nil
}

// GetVoteStatus возвращает статус голоса (для API)
func (vp *VoteProcessor) GetVoteStatus(ctx context.Context, pollID string, ip string) map[string]interface{} {
	hasVoted, _ := vp.deduplicator.HasVoted(ctx, pollID, ip)

	return map[string]interface{}{
		"poll_id":   pollID,
		"ip":        ip,
		"has_voted": hasVoted,
	}
}

// LogVoteEvent логирует событие голоса (для отладки)
func (vp *VoteProcessor) LogVoteEvent(vote *models.VoteEventMessage) {
	data, _ := json.MarshalIndent(vote, "", "  ")
	log.Printf("Vote Event:\n%s", string(data))
}
