package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"voting-app/antifraud"
	"voting-app/models"
	"voting-app/service"

	"github.com/go-chi/chi/v5"
)

func TestGatewayHandler_CreatePoll_Success(t *testing.T) {
	mockRepo := service.NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	pollService := service.NewPollService(mockRepo, mockCache)
	handler := NewGatewayHandler(pollService)

	req := models.CreatePollRequest{
		Title:   "Best Language",
		Options: []string{"Go", "Python", "Rust"},
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/polls", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler.CreatePoll(w, httpReq)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var resp models.CreatePollResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.ID == "" {
		t.Error("response should have poll ID")
	}
	if resp.AdminKey == "" {
		t.Error("response should have admin key")
	}
	if resp.Title != "Best Language" {
		t.Errorf("expected title 'Best Language', got '%s'", resp.Title)
	}
}

func TestGatewayHandler_CreatePoll_MissingTitle(t *testing.T) {
	mockRepo := service.NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	pollService := service.NewPollService(mockRepo, mockCache)
	handler := NewGatewayHandler(pollService)

	req := models.CreatePollRequest{
		Title:   "",
		Options: []string{"Go", "Python"},
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/polls", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler.CreatePoll(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGatewayHandler_CreatePoll_NoOptions(t *testing.T) {
	mockRepo := service.NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	pollService := service.NewPollService(mockRepo, mockCache)
	handler := NewGatewayHandler(pollService)

	req := models.CreatePollRequest{
		Title:   "Test Poll",
		Options: []string{},
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/polls", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler.CreatePoll(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGatewayHandler_CreatePoll_InvalidJSON(t *testing.T) {
	mockRepo := service.NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	pollService := service.NewPollService(mockRepo, mockCache)
	handler := NewGatewayHandler(pollService)

	httpReq := httptest.NewRequest("POST", "/polls", bytes.NewBuffer([]byte("invalid json")))
	w := httptest.NewRecorder()

	handler.CreatePoll(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGatewayHandler_GetPoll_NotFound(t *testing.T) {
	mockRepo := service.NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	pollService := service.NewPollService(mockRepo, mockCache)
	handler := NewGatewayHandler(pollService)

	httpReq := requestWithPollID("GET", "/polls/non-existent", nil, "non-existent")
	w := httptest.NewRecorder()

	handler.GetPoll(w, httpReq)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestGatewayHandler_GetPoll_Success(t *testing.T) {
	mockRepo := service.NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	pollService := service.NewPollService(mockRepo, mockCache)
	handler := NewGatewayHandler(pollService)

	resp, err := pollService.CreatePoll("Handler Poll", []string{"A", "B"})
	if err != nil {
		t.Fatalf("unexpected error creating poll: %v", err)
	}

	httpReq := requestWithPollID("GET", "/polls/test-poll", nil, resp.ID)
	w := httptest.NewRecorder()

	handler.GetPoll(w, httpReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var poll models.PollWithOptions
	if err := json.NewDecoder(w.Body).Decode(&poll); err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if poll.ID != resp.ID {
		t.Errorf("expected poll id '%s', got '%s'", resp.ID, poll.ID)
	}
	if len(poll.Options) != 2 {
		t.Errorf("expected 2 options, got %d", len(poll.Options))
	}
}

func TestGatewayHandler_GetResults_EmptyPoll(t *testing.T) {
	mockRepo := service.NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	pollService := service.NewPollService(mockRepo, mockCache)

	// Создаем опрос
	resp, _ := pollService.CreatePoll("Test Poll", []string{"A", "B", "C"})

	// Получаем результаты
	results, err := pollService.GetResults(resp.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results == nil {
		t.Error("results should not be nil")
	}

	// Все опции должны быть в результатах
	if len(results.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results.Results))
	}

	// Все счетчики должны быть 0
	for _, result := range results.Results {
		if result.Count != 0 {
			t.Errorf("expected 0 votes initially, got %d", result.Count)
		}
	}
}

func TestGatewayHandler_GetResults_Success(t *testing.T) {
	mockRepo := service.NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	pollService := service.NewPollService(mockRepo, mockCache)
	handler := NewGatewayHandler(pollService)

	resp, err := pollService.CreatePoll("Results Poll", []string{"A", "B"})
	if err != nil {
		t.Fatalf("unexpected error creating poll: %v", err)
	}

	httpReq := requestWithPollID("GET", "/polls/test-poll/results", nil, resp.ID)
	w := httptest.NewRecorder()

	handler.GetResults(w, httpReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var results models.PollResults
	if err := json.NewDecoder(w.Body).Decode(&results); err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if results.PollID != resp.ID {
		t.Errorf("expected poll id '%s', got '%s'", resp.ID, results.PollID)
	}
}

func TestGatewayHandler_GetResults_NotFound(t *testing.T) {
	mockRepo := service.NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	pollService := service.NewPollService(mockRepo, mockCache)
	handler := NewGatewayHandler(pollService)

	httpReq := requestWithPollID("GET", "/polls/missing/results", nil, "missing")
	w := httptest.NewRecorder()

	handler.GetResults(w, httpReq)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestGatewayHandler_Vote_Success(t *testing.T) {
	mockRepo := service.NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	pollService := service.NewPollService(mockRepo, mockCache)
	handler := NewGatewayHandler(pollService)
	publisher := &testVotePublisher{}
	handler.SetVotePublisher(publisher)

	createResp, err := pollService.CreatePoll("Vote Poll", []string{"A", "B"})
	if err != nil {
		t.Fatalf("unexpected error creating poll: %v", err)
	}

	body, _ := json.Marshal(models.VoteRequest{OptionID: "option-1"})
	httpReq := requestWithPollID("POST", "/polls/test-poll/vote", bytes.NewBuffer(body), createResp.ID)
	httpReq.RemoteAddr = "192.168.1.50:1234"
	httpReq.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()

	handler.Vote(w, httpReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var voteResp models.VoteResponse
	if err := json.NewDecoder(w.Body).Decode(&voteResp); err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if !voteResp.Success {
		t.Error("vote response should be successful")
	}
	if len(publisher.events) != 1 {
		t.Fatalf("expected one published event, got %d", len(publisher.events))
	}
	if publisher.events[0].PollID != createResp.ID {
		t.Errorf("expected published poll id '%s', got '%s'", createResp.ID, publisher.events[0].PollID)
	}
}

func TestGatewayHandler_Vote_InvalidJSON(t *testing.T) {
	mockRepo := service.NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	pollService := service.NewPollService(mockRepo, mockCache)
	handler := NewGatewayHandler(pollService)

	httpReq := requestWithPollID("POST", "/polls/poll-1/vote", bytes.NewBufferString("{"), "poll-1")
	w := httptest.NewRecorder()

	handler.Vote(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGatewayHandler_Vote_MissingOptionID(t *testing.T) {
	mockRepo := service.NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	pollService := service.NewPollService(mockRepo, mockCache)
	handler := NewGatewayHandler(pollService)

	body, _ := json.Marshal(models.VoteRequest{})
	httpReq := requestWithPollID("POST", "/polls/poll-1/vote", bytes.NewBuffer(body), "poll-1")
	w := httptest.NewRecorder()

	handler.Vote(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGatewayHandler_Vote_PollNotFound(t *testing.T) {
	mockRepo := service.NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	pollService := service.NewPollService(mockRepo, mockCache)
	handler := NewGatewayHandler(pollService)

	body, _ := json.Marshal(models.VoteRequest{OptionID: "option-1"})
	httpReq := requestWithPollID("POST", "/polls/missing/vote", bytes.NewBuffer(body), "missing")
	w := httptest.NewRecorder()

	handler.Vote(w, httpReq)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestGatewayHandler_Vote_InactivePoll(t *testing.T) {
	mockRepo := service.NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	pollService := service.NewPollService(mockRepo, mockCache)
	handler := NewGatewayHandler(pollService)

	resp, err := pollService.CreatePoll("Inactive Poll", []string{"A", "B"})
	if err != nil {
		t.Fatalf("unexpected error creating poll: %v", err)
	}
	if err := pollService.ClosePoll(resp.ID, resp.AdminKey); err != nil {
		t.Fatalf("unexpected error closing poll: %v", err)
	}

	body, _ := json.Marshal(models.VoteRequest{OptionID: "option-1"})
	httpReq := requestWithPollID("POST", "/polls/test-poll/vote", bytes.NewBuffer(body), resp.ID)
	w := httptest.NewRecorder()

	handler.Vote(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGatewayHandler_Vote_PublishError(t *testing.T) {
	mockRepo := service.NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	pollService := service.NewPollService(mockRepo, mockCache)
	handler := NewGatewayHandler(pollService)
	handler.SetVotePublisher(&testVotePublisher{err: errors.New("publish failed")})

	resp, err := pollService.CreatePoll("Publish Error Poll", []string{"A", "B"})
	if err != nil {
		t.Fatalf("unexpected error creating poll: %v", err)
	}

	body, _ := json.Marshal(models.VoteRequest{OptionID: "option-1"})
	httpReq := requestWithPollID("POST", "/polls/test-poll/vote", bytes.NewBuffer(body), resp.ID)
	w := httptest.NewRecorder()

	handler.Vote(w, httpReq)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestGatewayHandler_ClosePoll_Success(t *testing.T) {
	mockRepo := service.NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	pollService := service.NewPollService(mockRepo, mockCache)
	handler := NewGatewayHandler(pollService)

	resp, err := pollService.CreatePoll("Close Poll", []string{"A", "B"})
	if err != nil {
		t.Fatalf("unexpected error creating poll: %v", err)
	}

	body, _ := json.Marshal(models.ClosePollRequest{AdminKey: resp.AdminKey})
	httpReq := requestWithPollID("POST", "/polls/test-poll/close", bytes.NewBuffer(body), resp.ID)
	w := httptest.NewRecorder()

	handler.ClosePoll(w, httpReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if response["success"] != true {
		t.Errorf("expected success true, got %v", response["success"])
	}
}

func TestGatewayHandler_ClosePoll_InvalidJSON(t *testing.T) {
	mockRepo := service.NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	pollService := service.NewPollService(mockRepo, mockCache)
	handler := NewGatewayHandler(pollService)

	httpReq := requestWithPollID("POST", "/polls/poll-1/close", bytes.NewBufferString("{"), "poll-1")
	w := httptest.NewRecorder()

	handler.ClosePoll(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGatewayHandler_ClosePoll_MissingAdminKey(t *testing.T) {
	mockRepo := service.NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	pollService := service.NewPollService(mockRepo, mockCache)
	handler := NewGatewayHandler(pollService)

	body, _ := json.Marshal(models.ClosePollRequest{})
	httpReq := requestWithPollID("POST", "/polls/poll-1/close", bytes.NewBuffer(body), "poll-1")
	w := httptest.NewRecorder()

	handler.ClosePoll(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGatewayHandler_ClosePoll_InvalidAdminKey(t *testing.T) {
	mockRepo := service.NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	pollService := service.NewPollService(mockRepo, mockCache)
	handler := NewGatewayHandler(pollService)

	resp, err := pollService.CreatePoll("Close Poll", []string{"A", "B"})
	if err != nil {
		t.Fatalf("unexpected error creating poll: %v", err)
	}

	body, _ := json.Marshal(models.ClosePollRequest{AdminKey: "wrong-key"})
	httpReq := requestWithPollID("POST", "/polls/test-poll/close", bytes.NewBuffer(body), resp.ID)
	w := httptest.NewRecorder()

	handler.ClosePoll(w, httpReq)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestGatewayHandler_GetVoteStatus(t *testing.T) {
	mockRepo := service.NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	pollService := service.NewPollService(mockRepo, mockCache)
	handler := NewGatewayHandler(pollService)

	httpReq := requestWithPollID("GET", "/polls/poll-1/vote-status", nil, "poll-1")
	httpReq.Header.Set("X-Real-IP", "10.0.0.10")
	w := httptest.NewRecorder()

	handler.GetVoteStatus(w, httpReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var status map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&status); err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if status["poll_id"] != "poll-1" {
		t.Errorf("unexpected poll id: %v", status["poll_id"])
	}
	if status["ip"] != "10.0.0.10" {
		t.Errorf("unexpected ip: %v", status["ip"])
	}
	if status["has_voted"] != false {
		t.Errorf("expected has_voted false, got %v", status["has_voted"])
	}
}

func TestGatewayHandler_VoteRequest_Validation(t *testing.T) {
	tests := []struct {
		name  string
		req   models.VoteRequest
		valid bool
	}{
		{
			name:  "valid vote request",
			req:   models.VoteRequest{OptionID: "opt-1"},
			valid: true,
		},
		{
			name:  "invalid - empty option_id",
			req:   models.VoteRequest{OptionID: ""},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.req.OptionID != ""
			if isValid != tt.valid {
				t.Errorf("expected validation %v, got %v", tt.valid, isValid)
			}
		})
	}
}

func TestGatewayHandler_Health(t *testing.T) {
	mockRepo := service.NewMockPollRepository()
	mockCache := antifraud.NewMockCache()
	pollService := service.NewPollService(mockRepo, mockCache)
	handler := NewGatewayHandler(pollService)

	httpReq := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, httpReq)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%v'", resp["status"])
	}
	if resp["service"] != "gateway" {
		t.Errorf("expected service 'gateway', got '%v'", resp["service"])
	}
}

func TestGatewayHandler_GetClientIP(t *testing.T) {
	tests := []struct {
		name          string
		xForwardedFor string
		remoteAddr    string
		expectedIP    string
	}{
		{
			name:          "X-Forwarded-For header",
			xForwardedFor: "192.168.1.100",
			remoteAddr:    "127.0.0.1:8080",
			expectedIP:    "192.168.1.100",
		},
		{
			name:          "Remote address",
			xForwardedFor: "",
			remoteAddr:    "192.168.1.50:5000",
			expectedIP:    "192.168.1.50",
		},
		{
			name:          "Multiple IPs in X-Forwarded-For",
			xForwardedFor: "192.168.1.100, 10.0.0.1",
			remoteAddr:    "127.0.0.1:8080",
			expectedIP:    "192.168.1.100, 10.0.0.1",
		},
		{
			name:          "X-Real-IP header",
			xForwardedFor: "",
			remoteAddr:    "127.0.0.1:8080",
			expectedIP:    "10.0.0.20",
		},
		{
			name:          "Remote address without port",
			xForwardedFor: "",
			remoteAddr:    "192.168.1.60",
			expectedIP:    "192.168.1.60",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpReq := httptest.NewRequest("GET", "/test", nil)
			if tt.xForwardedFor != "" {
				httpReq.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.name == "X-Real-IP header" {
				httpReq.Header.Set("X-Real-IP", tt.expectedIP)
			}
			httpReq.RemoteAddr = tt.remoteAddr

			ip := GetClientIP(httpReq)
			if ip != tt.expectedIP {
				t.Errorf("expected IP '%s', got '%s'", tt.expectedIP, ip)
			}
		})
	}
}

func requestWithPollID(method string, target string, body io.Reader, pollID string) *http.Request {
	req := httptest.NewRequest(method, target, body)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("pollID", pollID)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
}

type testVotePublisher struct {
	events []models.VoteEventMessage
	err    error
}

func (p *testVotePublisher) PublishVote(event models.VoteEventMessage) error {
	if p.err != nil {
		return p.err
	}
	p.events = append(p.events, event)
	return nil
}
