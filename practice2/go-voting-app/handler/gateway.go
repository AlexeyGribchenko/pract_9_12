package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"voting-app/models"
	"voting-app/service"

	"github.com/go-chi/chi/v5"
)

type GatewayHandler struct {
	pollService *service.PollService
	publisher   VotePublisher
}

type VotePublisher interface {
	PublishVote(event models.VoteEventMessage) error
}

func NewGatewayHandler(pollService *service.PollService) *GatewayHandler {
	return &GatewayHandler{
		pollService: pollService,
	}
}

func (gh *GatewayHandler) SetVotePublisher(publisher VotePublisher) {
	gh.publisher = publisher
}

// CreatePoll обработчик для создания опроса
func (gh *GatewayHandler) CreatePoll(w http.ResponseWriter, r *http.Request) {
	var req models.CreatePollRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Title == "" || len(req.Options) == 0 {
		http.Error(w, "Title and options are required", http.StatusBadRequest)
		return
	}

	resp, err := gh.pollService.CreatePoll(req.Title, req.Options)
	if err != nil {
		log.Printf("Error creating poll: %v", err)
		http.Error(w, "Error creating poll", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// GetPoll обработчик для получения информации об опросе
func (gh *GatewayHandler) GetPoll(w http.ResponseWriter, r *http.Request) {
	pollID := chi.URLParam(r, "pollID")

	poll, options, err := gh.pollService.GetPollWithOptions(pollID)
	if err != nil {
		log.Printf("Error getting poll: %v", err)
		http.Error(w, "Error getting poll", http.StatusInternalServerError)
		return
	}

	if poll == nil {
		http.Error(w, "Poll not found", http.StatusNotFound)
		return
	}

	// Возвращаем опрос с вариантами ответов
	pollWithOptions := models.PollWithOptions{
		ID:        poll.ID,
		Title:     poll.Title,
		Status:    poll.Status,
		CreatedAt: poll.CreatedAt,
		ClosedAt:  poll.ClosedAt,
		Options:   options,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pollWithOptions)
}

// Vote обработчик для голосования
func (gh *GatewayHandler) Vote(w http.ResponseWriter, r *http.Request) {
	pollID := chi.URLParam(r, "pollID")

	var req models.VoteRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.OptionID == "" {
		http.Error(w, "option_id is required", http.StatusBadRequest)
		return
	}

	// Получить IP адрес клиента
	ip := GetClientIP(r)
	log.Printf("Vote request from IP: %s for poll: %s, option: %s", ip, pollID, req.OptionID)

	poll, err := gh.pollService.GetPoll(pollID)
	if err != nil {
		log.Printf("Error getting poll: %v", err)
		http.Error(w, "Error checking poll", http.StatusInternalServerError)
		return
	}
	if poll == nil {
		http.Error(w, "Poll not found", http.StatusNotFound)
		return
	}
	if poll.Status != "active" {
		http.Error(w, "Poll is not active", http.StatusBadRequest)
		return
	}

	voteEvent := models.VoteEventMessage{
		PollID:    pollID,
		OptionID:  req.OptionID,
		IP:        ip,
		UserAgent: r.UserAgent(),
	}
	if gh.publisher != nil {
		if err := gh.publisher.PublishVote(voteEvent); err != nil {
			log.Printf("Error publishing vote event: %v", err)
			http.Error(w, "Error publishing vote", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.VoteResponse{
		Success: true,
		Message: "Vote submitted for processing",
	})
}

// GetResults обработчик для получения результатов
func (gh *GatewayHandler) GetResults(w http.ResponseWriter, r *http.Request) {
	pollID := chi.URLParam(r, "pollID")

	results, err := gh.pollService.GetResults(pollID)
	if err != nil {
		log.Printf("Error getting results: %v", err)
		http.Error(w, "Error getting results", http.StatusInternalServerError)
		return
	}

	if results == nil {
		http.Error(w, "Poll not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// ClosePoll обработчик для закрытия опроса
func (gh *GatewayHandler) ClosePoll(w http.ResponseWriter, r *http.Request) {
	pollID := chi.URLParam(r, "pollID")

	var req models.ClosePollRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.AdminKey == "" {
		http.Error(w, "admin_key is required", http.StatusBadRequest)
		return
	}

	err = gh.pollService.ClosePoll(pollID, req.AdminKey)
	if err != nil {
		log.Printf("Error closing poll: %v", err)
		http.Error(w, "Invalid admin key", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Poll closed",
	})
}

// GetVoteStatus обработчик для проверки статуса голоса
func (gh *GatewayHandler) GetVoteStatus(w http.ResponseWriter, r *http.Request) {
	pollID := chi.URLParam(r, "pollID")
	ip := GetClientIP(r)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"poll_id":   pollID,
		"ip":        ip,
		"has_voted": false, // Будет получаться из Anti-Fraud Service
	})
}

// Health обработчик для проверки здоровья сервиса
func (gh *GatewayHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"service": "gateway",
	})
}
