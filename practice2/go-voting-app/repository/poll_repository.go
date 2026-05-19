package repository

import (
	"database/sql"
	"voting-app/models"

	"github.com/google/uuid"
)

type PollRepository struct {
	db *sql.DB
}

func NewPollRepository(db *sql.DB) *PollRepository {
	return &PollRepository{db: db}
}

func (pr *PollRepository) CreatePoll(title string) (*models.Poll, error) {
	pollID := uuid.New().String()
	adminKey := uuid.New().String()

	query := `
		INSERT INTO polls (id, title, admin_key, status) 
		VALUES ($1, $2, $3, 'active')
		RETURNING id, title, admin_key, status, created_at
	`

	var poll models.Poll
	err := pr.db.QueryRow(query, pollID, title, adminKey).
		Scan(&poll.ID, &poll.Title, &poll.AdminKey, &poll.Status, &poll.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &poll, nil
}

func (pr *PollRepository) GetPoll(pollID string) (*models.Poll, error) {
	query := `
		SELECT id, title, admin_key, status, created_at, closed_at 
		FROM polls 
		WHERE id = $1
	`

	var poll models.Poll
	err := pr.db.QueryRow(query, pollID).
		Scan(&poll.ID, &poll.Title, &poll.AdminKey, &poll.Status, &poll.CreatedAt, &poll.ClosedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &poll, nil
}

func (pr *PollRepository) ClosePoll(pollID string, adminKey string) error {
	query := `
		UPDATE polls 
		SET status = 'closed', closed_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND admin_key = $2
	`

	result, err := pr.db.Exec(query, pollID, adminKey)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (pr *PollRepository) AddOption(pollID string, text string, order int) (*models.Option, error) {
	optionID := uuid.New().String()

	query := `
		INSERT INTO options (id, poll_id, text, "order") 
		VALUES ($1, $2, $3, $4)
		RETURNING id, poll_id, text, "order"
	`

	var option models.Option
	err := pr.db.QueryRow(query, optionID, pollID, text, order).
		Scan(&option.ID, &option.PollID, &option.Text, &option.Order)

	if err != nil {
		return nil, err
	}

	return &option, nil
}

func (pr *PollRepository) GetOptions(pollID string) ([]*models.Option, error) {
	query := `
		SELECT id, poll_id, text, "order" 
		FROM options 
		WHERE poll_id = $1 
		ORDER BY "order"
	`

	rows, err := pr.db.Query(query, pollID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var options []*models.Option
	for rows.Next() {
		var option models.Option
		err := rows.Scan(&option.ID, &option.PollID, &option.Text, &option.Order)
		if err != nil {
			return nil, err
		}
		options = append(options, &option)
	}

	return options, rows.Err()
}

func (pr *PollRepository) GetPollWithOptions(pollID string) (*models.PollWithOptions, error) {
	poll, err := pr.GetPoll(pollID)
	if err != nil {
		return nil, err
	}

	if poll == nil {
		return nil, nil
	}

	optionPtrs, err := pr.GetOptions(pollID)
	if err != nil {
		return nil, err
	}

	options := make([]models.Option, len(optionPtrs))
	for i, opt := range optionPtrs {
		options[i] = *opt
	}

	return &models.PollWithOptions{
		ID:        poll.ID,
		Title:     poll.Title,
		Status:    poll.Status,
		CreatedAt: poll.CreatedAt,
		ClosedAt:  poll.ClosedAt,
		Options:   options,
	}, nil
}

func (pr *PollRepository) GetPollConfig(pollID string) (*models.Poll, error) {
	return pr.GetPoll(pollID)
}

func (pr *PollRepository) GetResults(pollID string) (*models.PollResults, error) {
	poll, err := pr.GetPoll(pollID)
	if err != nil {
		return nil, err
	}

	if poll == nil {
		return nil, nil
	}

	options, err := pr.GetOptions(pollID)
	if err != nil {
		return nil, err
	}

	voteCounts, err := pr.GetVoteCounts(pollID)
	if err != nil {
		return nil, err
	}

	results := make(map[string]models.PollOptionCount)
	for _, option := range options {
		results[option.ID] = models.PollOptionCount{
			Option: option.ID,
			Text:   option.Text,
			Count:  voteCounts[option.ID],
		}
	}

	return &models.PollResults{
		PollID:  poll.ID,
		Title:   poll.Title,
		Status:  poll.Status,
		Results: results,
	}, nil
}

func (pr *PollRepository) RecordVote(pollID string, optionID string, ip string) error {
	voteID := uuid.New().String()

	query := `
		INSERT INTO votes (id, poll_id, option_id, ip)
		SELECT $1::VARCHAR(36), $2::VARCHAR(36), $3::VARCHAR(36), $4::VARCHAR(45)
		WHERE EXISTS (
			SELECT 1
			FROM options
			WHERE id = $3::VARCHAR(36) AND poll_id = $2::VARCHAR(36)
		)
	`

	result, err := pr.db.Exec(query, voteID, pollID, optionID, ip)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (pr *PollRepository) GetVoteCounts(pollID string) (map[string]int64, error) {
	query := `
		SELECT option_id, COUNT(*) as count 
		FROM votes 
		WHERE poll_id = $1 
		GROUP BY option_id
	`

	rows, err := pr.db.Query(query, pollID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int64)
	for rows.Next() {
		var optionID string
		var count int64
		err := rows.Scan(&optionID, &count)
		if err != nil {
			return nil, err
		}
		counts[optionID] = count
	}

	return counts, rows.Err()
}

func (pr *PollRepository) GetVote(pollID string, ip string) (*models.Vote, error) {
	query := `
		SELECT id, poll_id, option_id, ip, voted_at
		FROM votes
		WHERE poll_id = $1 AND ip = $2
		ORDER BY voted_at DESC
		LIMIT 1
	`

	var vote models.Vote
	err := pr.db.QueryRow(query, pollID, ip).
		Scan(&vote.ID, &vote.PollID, &vote.OptionID, &vote.IP, &vote.VotedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &vote, nil
}
