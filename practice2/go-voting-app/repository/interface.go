package repository

import (
	"voting-app/models"
)

// IPollRepository определяет интерфейс для работы с опросами
type IPollRepository interface {
	PollServiceRepository
	VoteProcessorRepository
	GetPollWithOptions(pollID string) (*models.PollWithOptions, error)
	GetPollConfig(pollID string) (*models.Poll, error)
	GetResults(pollID string) (*models.PollResults, error)
	GetVote(pollID string, ip string) (*models.Vote, error)
}

type PollServiceRepository interface {
	CreatePoll(title string) (*models.Poll, error)
	GetPoll(pollID string) (*models.Poll, error)
	ClosePoll(pollID string, adminKey string) error
	AddOption(pollID string, text string, order int) (*models.Option, error)
	GetOptions(pollID string) ([]*models.Option, error)
	GetVoteCounts(pollID string) (map[string]int64, error)
}

type VoteProcessorRepository interface {
	GetPoll(pollID string) (*models.Poll, error)
	RecordVote(pollID string, optionID string, ip string) error
}
