package models

import "time"

// Poll представляет опрос
type Poll struct {
	ID        string     `db:"id" json:"id"`
	Title     string     `db:"title" json:"title"`
	AdminKey  string     `db:"admin_key" json:"admin_key,omitempty"`
	Status    string     `db:"status" json:"status"` // active, closed
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	ClosedAt  *time.Time `db:"closed_at" json:"closed_at,omitempty"`
}

// PollWithOptions представляет опрос с вариантами ответов
type PollWithOptions struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	ClosedAt  *time.Time `json:"closed_at,omitempty"`
	Options   []Option   `json:"options"`
}

// Option представляет вариант ответа
type Option struct {
	ID     string `db:"id" json:"id"`
	PollID string `db:"poll_id" json:"poll_id"`
	Text   string `db:"text" json:"text"`
	Order  int    `db:"order" json:"order"`
}

// Vote представляет голос (для истории)
type Vote struct {
	ID       string    `db:"id" json:"id"`
	PollID   string    `db:"poll_id" json:"poll_id"`
	OptionID string    `db:"option_id" json:"option_id"`
	IP       string    `db:"ip" json:"ip"`
	VotedAt  time.Time `db:"voted_at" json:"voted_at"`
}

// VoteRequest - запрос на голосование (pollID передается в URL)
type VoteRequest struct {
	OptionID string `json:"option_id"`
}

// VoteResponse - ответ на голосование
type VoteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// PollResults - результаты опроса
type PollResults struct {
	PollID  string                     `json:"poll_id"`
	Title   string                     `json:"title"`
	Status  string                     `json:"status"`
	Results map[string]PollOptionCount `json:"results"`
}

type PollOptionCount struct {
	Option string `json:"option"`
	Text   string `json:"text"`
	Count  int64  `json:"count"`
}

// CreatePollRequest - запрос на создание опроса
type CreatePollRequest struct {
	Title   string   `json:"title"`
	Options []string `json:"options"`
}

// CreatePollResponse - ответ после создания опроса
type CreatePollResponse struct {
	ID       string `json:"id"`
	AdminKey string `json:"admin_key"`
	Title    string `json:"title"`
}

// ClosePollRequest - запрос на закрытие опроса
type ClosePollRequest struct {
	AdminKey string `json:"admin_key"`
}

// VoteEventMessage - сообщение о голосе для очереди NATS
type VoteEventMessage struct {
	PollID    string `json:"poll_id"`
	OptionID  string `json:"option_id"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
}

// IPCheckResponse - ответ от GeoIP API
type IPCheckResponse struct {
	IP      string `json:"ip"`
	Type    string `json:"type"` // residential, datacenter, vpn
	Country string `json:"country"`
	City    string `json:"city"`
	ISP     string `json:"isp"`
}
