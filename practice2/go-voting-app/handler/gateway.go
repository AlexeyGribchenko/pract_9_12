package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"voting-app/models"

	"github.com/go-chi/chi/v5"
)

// HTTPClient интерфейс для мокания HTTP-запросов
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
	Get(url string) (*http.Response, error)
}

type GatewayHandler struct {
	pollManagerURL string
	publisher      VotePublisher
	httpClient     HTTPClient
}

type VotePublisher interface {
	PublishVote(event models.VoteEventMessage) error
}

func NewGatewayHandler(pollManagerURL string) *GatewayHandler {
	return &GatewayHandler{
		pollManagerURL: pollManagerURL,
		httpClient:     &http.Client{},
	}
}

// SetHTTPClient позволяет установить мок-клиент для тестов
func (gh *GatewayHandler) SetHTTPClient(client HTTPClient) {
	gh.httpClient = client
}

func (gh *GatewayHandler) SetVotePublisher(publisher VotePublisher) {
	gh.publisher = publisher
}

// CreatePoll - проксируем запрос в Poll Manager
func (gh *GatewayHandler) CreatePoll(w http.ResponseWriter, r *http.Request) {
	gh.proxyRequest(w, r, "POST", gh.pollManagerURL+"/polls")
}

// GetPoll - проксируем запрос в Poll Manager
func (gh *GatewayHandler) GetPoll(w http.ResponseWriter, r *http.Request) {
	pollID := chi.URLParam(r, "pollID")
	gh.proxyRequest(w, r, "GET", fmt.Sprintf("%s/polls/%s", gh.pollManagerURL, pollID))
}

// Vote - особая логика: публикуем в NATS
func (gh *GatewayHandler) Vote(w http.ResponseWriter, r *http.Request) {
	pollID := chi.URLParam(r, "pollID")

	var req models.VoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.OptionID == "" {
		http.Error(w, "option_id is required", http.StatusBadRequest)
		return
	}

	// Проверяем статус опроса через Poll Manager
	poll, err := gh.getPollStatus(pollID)
	if err != nil {
		log.Printf("Error checking poll status: %v", err)
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

	// Публикуем голос в NATS
	voteEvent := models.VoteEventMessage{
		PollID:    pollID,
		OptionID:  req.OptionID,
		IP:        GetClientIP(r),
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

// GetResults - проксируем в Poll Manager
func (gh *GatewayHandler) GetResults(w http.ResponseWriter, r *http.Request) {
	pollID := chi.URLParam(r, "pollID")
	gh.proxyRequest(w, r, "GET", fmt.Sprintf("%s/polls/%s/results", gh.pollManagerURL, pollID))
}

// ClosePoll - проксируем в Poll Manager
func (gh *GatewayHandler) ClosePoll(w http.ResponseWriter, r *http.Request) {
	pollID := chi.URLParam(r, "pollID")
	gh.proxyRequest(w, r, "POST", fmt.Sprintf("%s/polls/%s/close", gh.pollManagerURL, pollID))
}

// GetVoteStatus - оставляем в Gateway
func (gh *GatewayHandler) GetVoteStatus(w http.ResponseWriter, r *http.Request) {
	pollID := chi.URLParam(r, "pollID")
	ip := GetClientIP(r)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"poll_id":   pollID,
		"ip":        ip,
		"has_voted": false,
	})
}

// Health check
func (gh *GatewayHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"service": "gateway",
	})
}

// proxyRequest - общий метод для проксирования запросов
func (gh *GatewayHandler) proxyRequest(w http.ResponseWriter, r *http.Request, method, url string) {
	// Читаем тело запроса
	var body io.Reader
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading request body: %v", err)
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		body = bytes.NewReader(bodyBytes)
	}

	// Создаем новый запрос к Poll Manager
	proxyReq, err := http.NewRequestWithContext(r.Context(), method, url, body)
	if err != nil {
		log.Printf("Error creating proxy request: %v", err)
		http.Error(w, "Error creating proxy request", http.StatusInternalServerError)
		return
	}

	// Копируем заголовки
	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Выполняем запрос через HTTP клиент (может быть моком)
	resp, err := gh.httpClient.Do(proxyReq)
	if err != nil {
		log.Printf("Error proxying request to %s: %v", url, err)
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	// Копируем заголовки ответа
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Копируем статус и тело ответа
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// getPollStatus - получает статус опроса из Poll Manager
func (gh *GatewayHandler) getPollStatus(pollID string) (*models.Poll, error) {
	resp, err := gh.httpClient.Get(fmt.Sprintf("%s/polls/%s", gh.pollManagerURL, pollID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var poll models.Poll
	if err := json.NewDecoder(resp.Body).Decode(&poll); err != nil {
		return nil, err
	}

	return &poll, nil
}
