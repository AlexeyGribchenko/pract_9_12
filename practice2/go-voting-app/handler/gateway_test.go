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
	"voting-app/models"

	"github.com/go-chi/chi/v5"
)

func TestGatewayHandler_CreatePoll_Success(t *testing.T) {
	handler := NewGatewayHandler("http://poll-manager:8002")

	// Мокаем HTTP-клиент
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// Проверяем, что запрос правильный
			if req.Method != "POST" {
				t.Errorf("expected POST, got %s", req.Method)
			}
			if req.URL.String() != "http://poll-manager:8002/polls" {
				t.Errorf("unexpected URL: %s", req.URL.String())
			}

			// Возвращаем успешный ответ
			resp := models.CreatePollResponse{
				ID:       "test-poll-id",
				AdminKey: "test-admin-key",
				Title:    "Best Language",
			}
			body, _ := json.Marshal(resp)
			return mockJSONResponse(http.StatusCreated, string(body)), nil
		},
	}
	handler.SetHTTPClient(mockClient)

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

	if resp.ID != "test-poll-id" {
		t.Errorf("expected ID 'test-poll-id', got '%s'", resp.ID)
	}
	if resp.AdminKey != "test-admin-key" {
		t.Errorf("expected admin key 'test-admin-key', got '%s'", resp.AdminKey)
	}
}

func TestGatewayHandler_CreatePoll_MissingTitle(t *testing.T) {
	handler := NewGatewayHandler("http://poll-manager:8002")

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return mockResponse(http.StatusBadRequest, "title and options required"), nil
		},
	}
	handler.SetHTTPClient(mockClient)

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

func TestGatewayHandler_GetPoll_NotFound(t *testing.T) {
	handler := NewGatewayHandler("http://poll-manager:8002")

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return mockResponse(http.StatusNotFound, "not found"), nil
		},
	}
	handler.SetHTTPClient(mockClient)

	httpReq := requestWithPollID("GET", "/polls/non-existent", nil, "non-existent")
	w := httptest.NewRecorder()

	handler.GetPoll(w, httpReq)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestGatewayHandler_GetPoll_Success(t *testing.T) {
	handler := NewGatewayHandler("http://poll-manager:8002")

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			poll := models.Poll{
				ID:     "test-poll",
				Title:  "Test Poll",
				Status: "active",
			}
			body, _ := json.Marshal(poll)
			return mockJSONResponse(http.StatusOK, string(body)), nil
		},
	}
	handler.SetHTTPClient(mockClient)

	httpReq := requestWithPollID("GET", "/polls/test-poll", nil, "test-poll")
	w := httptest.NewRecorder()

	handler.GetPoll(w, httpReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var poll models.Poll
	if err := json.NewDecoder(w.Body).Decode(&poll); err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if poll.ID != "test-poll" {
		t.Errorf("expected poll id 'test-poll', got '%s'", poll.ID)
	}
}

func TestGatewayHandler_GetResults_Success(t *testing.T) {
	handler := NewGatewayHandler("http://poll-manager:8002")

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			results := models.PollResults{
				PollID: "test-poll",
				Title:  "Results Poll",
				Status: "active",
				Results: map[string]models.PollOptionCount{
					"opt-1": {Option: "opt-1", Text: "A", Count: 5},
					"opt-2": {Option: "opt-2", Text: "B", Count: 3},
				},
			}
			body, _ := json.Marshal(results)
			return mockJSONResponse(http.StatusOK, string(body)), nil
		},
	}
	handler.SetHTTPClient(mockClient)

	httpReq := requestWithPollID("GET", "/polls/test-poll/results", nil, "test-poll")
	w := httptest.NewRecorder()

	handler.GetResults(w, httpReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var results models.PollResults
	if err := json.NewDecoder(w.Body).Decode(&results); err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if results.PollID != "test-poll" {
		t.Errorf("expected poll id 'test-poll', got '%s'", results.PollID)
	}
}

func TestGatewayHandler_Vote_Success(t *testing.T) {
	handler := NewGatewayHandler("http://poll-manager:8002")

	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			// Мокаем проверку статуса опроса
			poll := models.Poll{
				ID:     "test-poll",
				Title:  "Vote Poll",
				Status: "active",
			}
			body, _ := json.Marshal(poll)
			return mockJSONResponse(http.StatusOK, string(body)), nil
		},
	}
	handler.SetHTTPClient(mockClient)

	publisher := &testVotePublisher{}
	handler.SetVotePublisher(publisher)

	body, _ := json.Marshal(models.VoteRequest{OptionID: "option-1"})
	httpReq := requestWithPollID("POST", "/polls/test-poll/vote", bytes.NewBuffer(body), "test-poll")
	httpReq.RemoteAddr = "192.168.1.50:1234"
	httpReq.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()

	handler.Vote(w, httpReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
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
	if publisher.events[0].PollID != "test-poll" {
		t.Errorf("expected published poll id 'test-poll', got '%s'", publisher.events[0].PollID)
	}
}

func TestGatewayHandler_Vote_InvalidJSON(t *testing.T) {
	handler := NewGatewayHandler("http://poll-manager:8002")

	// Не настраиваем HTTP-клиент, так как до него не должно дойти

	httpReq := requestWithPollID("POST", "/polls/poll-1/vote", bytes.NewBufferString("{"), "poll-1")
	w := httptest.NewRecorder()

	handler.Vote(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGatewayHandler_Vote_MissingOptionID(t *testing.T) {
	handler := NewGatewayHandler("http://poll-manager:8002")

	body, _ := json.Marshal(models.VoteRequest{})
	httpReq := requestWithPollID("POST", "/polls/poll-1/vote", bytes.NewBuffer(body), "poll-1")
	w := httptest.NewRecorder()

	handler.Vote(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGatewayHandler_Vote_PollNotFound(t *testing.T) {
	handler := NewGatewayHandler("http://poll-manager:8002")

	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			// Возвращаем 404 - опрос не найден
			return mockResponse(http.StatusNotFound, "not found"), nil
		},
	}
	handler.SetHTTPClient(mockClient)

	body, _ := json.Marshal(models.VoteRequest{OptionID: "option-1"})
	httpReq := requestWithPollID("POST", "/polls/missing/vote", bytes.NewBuffer(body), "missing")
	w := httptest.NewRecorder()

	handler.Vote(w, httpReq)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGatewayHandler_Vote_InactivePoll(t *testing.T) {
	handler := NewGatewayHandler("http://poll-manager:8002")

	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			poll := models.Poll{
				ID:     "test-poll",
				Title:  "Inactive Poll",
				Status: "closed",
			}
			body, _ := json.Marshal(poll)
			return mockJSONResponse(http.StatusOK, string(body)), nil
		},
	}
	handler.SetHTTPClient(mockClient)

	body, _ := json.Marshal(models.VoteRequest{OptionID: "option-1"})
	httpReq := requestWithPollID("POST", "/polls/test-poll/vote", bytes.NewBuffer(body), "test-poll")
	w := httptest.NewRecorder()

	handler.Vote(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGatewayHandler_Vote_PublishError(t *testing.T) {
	handler := NewGatewayHandler("http://poll-manager:8002")

	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			poll := models.Poll{
				ID:     "test-poll",
				Title:  "Publish Error Poll",
				Status: "active",
			}
			body, _ := json.Marshal(poll)
			return mockJSONResponse(http.StatusOK, string(body)), nil
		},
	}
	handler.SetHTTPClient(mockClient)

	publisher := &testVotePublisher{err: errors.New("publish failed")}
	handler.SetVotePublisher(publisher)

	body, _ := json.Marshal(models.VoteRequest{OptionID: "option-1"})
	httpReq := requestWithPollID("POST", "/polls/test-poll/vote", bytes.NewBuffer(body), "test-poll")
	w := httptest.NewRecorder()

	handler.Vote(w, httpReq)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGatewayHandler_Vote_PollManagerError(t *testing.T) {
	handler := NewGatewayHandler("http://poll-manager:8002")

	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			// Симулируем ошибку соединения
			return nil, errors.New("connection refused")
		},
	}
	handler.SetHTTPClient(mockClient)

	body, _ := json.Marshal(models.VoteRequest{OptionID: "option-1"})
	httpReq := requestWithPollID("POST", "/polls/test-poll/vote", bytes.NewBuffer(body), "test-poll")
	w := httptest.NewRecorder()

	handler.Vote(w, httpReq)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGatewayHandler_ClosePoll_Success(t *testing.T) {
	handler := NewGatewayHandler("http://poll-manager:8002")

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			resp := map[string]interface{}{
				"success": true,
				"message": "Poll closed",
			}
			body, _ := json.Marshal(resp)
			return mockJSONResponse(http.StatusOK, string(body)), nil
		},
	}
	handler.SetHTTPClient(mockClient)

	body, _ := json.Marshal(models.ClosePollRequest{AdminKey: "valid-key"})
	httpReq := requestWithPollID("POST", "/polls/test-poll/close", bytes.NewBuffer(body), "test-poll")
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

func TestGatewayHandler_ClosePoll_InvalidAdminKey(t *testing.T) {
	handler := NewGatewayHandler("http://poll-manager:8002")

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return mockResponse(http.StatusUnauthorized, "invalid admin key"), nil
		},
	}
	handler.SetHTTPClient(mockClient)

	body, _ := json.Marshal(models.ClosePollRequest{AdminKey: "wrong-key"})
	httpReq := requestWithPollID("POST", "/polls/test-poll/close", bytes.NewBuffer(body), "test-poll")
	w := httptest.NewRecorder()

	handler.ClosePoll(w, httpReq)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestGatewayHandler_GetVoteStatus(t *testing.T) {
	handler := NewGatewayHandler("http://poll-manager:8002")

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
}

func TestGatewayHandler_Health(t *testing.T) {
	handler := NewGatewayHandler("http://poll-manager:8002")

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
		xRealIP       string
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
			name:       "X-Real-IP header",
			xRealIP:    "10.0.0.20",
			remoteAddr: "127.0.0.1:8080",
			expectedIP: "10.0.0.20",
		},
		{
			name:       "Remote address",
			remoteAddr: "192.168.1.50:5000",
			expectedIP: "192.168.1.50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpReq := httptest.NewRequest("GET", "/test", nil)
			if tt.xForwardedFor != "" {
				httpReq.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				httpReq.Header.Set("X-Real-IP", tt.xRealIP)
			}
			httpReq.RemoteAddr = tt.remoteAddr

			ip := GetClientIP(httpReq)
			if ip != tt.expectedIP {
				t.Errorf("expected IP '%s', got '%s'", tt.expectedIP, ip)
			}
		})
	}
}

func TestGatewayHandler_ProxyRequest_ServiceUnavailable(t *testing.T) {
	handler := NewGatewayHandler("http://poll-manager:8002")

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("connection refused")
		},
	}
	handler.SetHTTPClient(mockClient)

	httpReq := requestWithPollID("GET", "/polls/test", nil, "test")
	w := httptest.NewRecorder()

	handler.GetPoll(w, httpReq)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}
}

// Вспомогательные функции
func requestWithPollID(method string, target string, body io.Reader, pollID string) *http.Request {
	req := httptest.NewRequest(method, target, body)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("pollID", pollID)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
}

// Mock publisher для тестов
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
