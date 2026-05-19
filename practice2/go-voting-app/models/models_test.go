package models

import (
	"testing"
	"time"
)

func TestVoteRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     VoteRequest
		isValid bool
	}{
		{
			name:    "valid request with option_id",
			req:     VoteRequest{OptionID: "550e8400-e29b-41d4-a716-446655440000"},
			isValid: true,
		},
		{
			name:    "invalid request with empty option_id",
			req:     VoteRequest{OptionID: ""},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.req.OptionID != ""
			if isValid != tt.isValid {
				t.Errorf("expected %v, got %v", tt.isValid, isValid)
			}
		})
	}
}

func TestCreatePollRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     CreatePollRequest
		isValid bool
	}{
		{
			name:    "valid request",
			req:     CreatePollRequest{Title: "Best Language", Options: []string{"Go", "Python", "Rust"}},
			isValid: true,
		},
		{
			name:    "invalid - empty title",
			req:     CreatePollRequest{Title: "", Options: []string{"Go", "Python"}},
			isValid: false,
		},
		{
			name:    "invalid - no options",
			req:     CreatePollRequest{Title: "Best Language", Options: []string{}},
			isValid: false,
		},
		{
			name:    "invalid - single option",
			req:     CreatePollRequest{Title: "Best Language", Options: []string{"Go"}},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.req.Title != "" && len(tt.req.Options) >= 2
			if isValid != tt.isValid {
				t.Errorf("expected %v, got %v", tt.isValid, isValid)
			}
		})
	}
}

func TestPoll_Creation(t *testing.T) {
	poll := &Poll{
		ID:       "550e8400-e29b-41d4-a716-446655440000",
		Title:    "Test Poll",
		AdminKey: "secret-key",
		Status:   "active",
		CreatedAt: time.Now(),
	}

	if poll.ID == "" {
		t.Error("poll ID should not be empty")
	}
	if poll.Title == "" {
		t.Error("poll Title should not be empty")
	}
	if poll.Status != "active" {
		t.Errorf("expected status 'active', got '%s'", poll.Status)
	}
}

func TestOption_Creation(t *testing.T) {
	option := &Option{
		ID:     "opt-1",
		PollID: "550e8400-e29b-41d4-a716-446655440000",
		Text:   "Go",
		Order:  1,
	}

	if option.ID == "" {
		t.Error("option ID should not be empty")
	}
	if option.Text == "" {
		t.Error("option Text should not be empty")
	}
	if option.Order != 1 {
		t.Errorf("expected order 1, got %d", option.Order)
	}
}

func TestVoteEventMessage_Creation(t *testing.T) {
	voteEvent := &VoteEventMessage{
		PollID:    "550e8400-e29b-41d4-a716-446655440000",
		OptionID:  "opt-1",
		IP:        "192.168.1.100",
		UserAgent: "Mozilla/5.0",
	}

	if voteEvent.PollID == "" {
		t.Error("PollID should not be empty")
	}
	if voteEvent.OptionID == "" {
		t.Error("OptionID should not be empty")
	}
	if voteEvent.IP == "" {
		t.Error("IP should not be empty")
	}
}

func TestPollResults_Creation(t *testing.T) {
	results := &PollResults{
		PollID: "550e8400-e29b-41d4-a716-446655440000",
		Title:  "Best Language",
		Status: "closed",
		Results: map[string]PollOptionCount{
			"opt-1": {Option: "opt-1", Text: "Go", Count: 100},
			"opt-2": {Option: "opt-2", Text: "Python", Count: 80},
		},
	}

	if len(results.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results.Results))
	}

	if results.Results["opt-1"].Count != 100 {
		t.Errorf("expected count 100 for Go, got %d", results.Results["opt-1"].Count)
	}
}
