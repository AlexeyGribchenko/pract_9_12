package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
	"voting-app/models"
)

func TestPollRepository_CreatePoll(t *testing.T) {
	repo := newRepositoryTestRepo(t, repositoryTestData{})

	poll, err := repo.CreatePoll("Test Poll")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if poll.ID == "" {
		t.Error("poll ID should not be empty")
	}
	if poll.AdminKey == "" {
		t.Error("admin key should not be empty")
	}
	if poll.Title != "Test Poll" {
		t.Errorf("expected title 'Test Poll', got '%s'", poll.Title)
	}
	if poll.Status != "active" {
		t.Errorf("expected status active, got %s", poll.Status)
	}
}

func TestPollRepository_GetPoll_Found(t *testing.T) {
	createdAt := time.Now()
	repo := newRepositoryTestRepo(t, repositoryTestData{
		poll: &models.Poll{
			ID:        "poll-1",
			Title:     "Test Poll",
			AdminKey:  "admin-key",
			Status:    "active",
			CreatedAt: createdAt,
		},
	})

	poll, err := repo.GetPoll("poll-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if poll == nil {
		t.Fatal("poll should not be nil")
	}
	if poll.ID != "poll-1" || poll.Title != "Test Poll" {
		t.Errorf("unexpected poll: %+v", poll)
	}
}

func TestPollRepository_GetPoll_NotFound(t *testing.T) {
	repo := newRepositoryTestRepo(t, repositoryTestData{})

	poll, err := repo.GetPoll("missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if poll != nil {
		t.Error("poll should be nil")
	}
}

func TestPollRepository_GetPollWithOptions(t *testing.T) {
	repo := newRepositoryTestRepo(t, repositoryTestData{
		poll: &models.Poll{
			ID:       "poll-1",
			Title:    "Test Poll",
			AdminKey: "admin-key",
			Status:   "active",
		},
		options: []*models.Option{
			{ID: "opt-1", PollID: "poll-1", Text: "A", Order: 1},
			{ID: "opt-2", PollID: "poll-1", Text: "B", Order: 2},
		},
	})

	poll, err := repo.GetPollWithOptions("poll-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if poll == nil {
		t.Fatal("poll should not be nil")
	}
	if len(poll.Options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(poll.Options))
	}
	if poll.Options[0].Text != "A" || poll.Options[1].Text != "B" {
		t.Errorf("unexpected options: %+v", poll.Options)
	}
}

func TestPollRepository_ClosePoll(t *testing.T) {
	repo := newRepositoryTestRepo(t, repositoryTestData{rowsAffected: 1})

	if err := repo.ClosePoll("poll-1", "admin-key"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPollRepository_ClosePoll_NoRows(t *testing.T) {
	repo := newRepositoryTestRepo(t, repositoryTestData{rowsAffected: 0})

	err := repo.ClosePoll("poll-1", "wrong-key")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestPollRepository_AddOption(t *testing.T) {
	repo := newRepositoryTestRepo(t, repositoryTestData{})

	option, err := repo.AddOption("poll-1", "Option A", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if option.ID == "" {
		t.Error("option ID should not be empty")
	}
	if option.PollID != "poll-1" || option.Text != "Option A" || option.Order != 1 {
		t.Errorf("unexpected option: %+v", option)
	}
}

func TestPollRepository_GetOptions(t *testing.T) {
	repo := newRepositoryTestRepo(t, repositoryTestData{
		options: []*models.Option{
			{ID: "opt-1", PollID: "poll-1", Text: "A", Order: 1},
			{ID: "opt-2", PollID: "poll-1", Text: "B", Order: 2},
		},
	})

	options, err := repo.GetOptions("poll-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(options))
	}
}

func TestPollRepository_RecordVote(t *testing.T) {
	repo := newRepositoryTestRepo(t, repositoryTestData{rowsAffected: 1})

	if err := repo.RecordVote("poll-1", "opt-1", "192.168.1.1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPollRepository_RecordVote_OptionMismatch(t *testing.T) {
	repo := newRepositoryTestRepo(t, repositoryTestData{rowsAffected: 0})

	err := repo.RecordVote("poll-1", "opt-from-other-poll", "192.168.1.1")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestPollRepository_GetResults(t *testing.T) {
	repo := newRepositoryTestRepo(t, repositoryTestData{
		poll: &models.Poll{
			ID:     "poll-1",
			Title:  "Test Poll",
			Status: "active",
		},
		options: []*models.Option{
			{ID: "opt-1", PollID: "poll-1", Text: "A", Order: 1},
			{ID: "opt-2", PollID: "poll-1", Text: "B", Order: 2},
		},
		voteCounts: map[string]int64{"opt-1": 3, "opt-2": 1},
	})

	results, err := repo.GetResults("poll-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results == nil {
		t.Fatal("results should not be nil")
	}
	if results.Results["opt-1"].Count != 3 {
		t.Errorf("expected opt-1 count 3, got %d", results.Results["opt-1"].Count)
	}
	if results.Results["opt-2"].Count != 1 {
		t.Errorf("expected opt-2 count 1, got %d", results.Results["opt-2"].Count)
	}
}

func TestPollRepository_GetVote_Found(t *testing.T) {
	repo := newRepositoryTestRepo(t, repositoryTestData{
		vote: &models.Vote{
			ID:       "vote-1",
			PollID:   "poll-1",
			OptionID: "opt-1",
			IP:       "192.168.1.1",
			VotedAt:  time.Now(),
		},
	})

	vote, err := repo.GetVote("poll-1", "192.168.1.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vote == nil {
		t.Fatal("vote should not be nil")
	}
	if vote.ID != "vote-1" {
		t.Errorf("unexpected vote: %+v", vote)
	}
}

func TestPollRepository_GetVote_NotFound(t *testing.T) {
	repo := newRepositoryTestRepo(t, repositoryTestData{})

	vote, err := repo.GetVote("poll-1", "192.168.1.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vote != nil {
		t.Error("vote should be nil")
	}
}

func newRepositoryTestRepo(t *testing.T, data repositoryTestData) *PollRepository {
	t.Helper()

	repositoryTestCurrentData = data
	if !repositoryTestRegistered {
		sql.Register("repository-test-driver", repositoryTestDriver{})
		repositoryTestRegistered = true
	}

	database, err := sql.Open("repository-test-driver", "")
	if err != nil {
		t.Fatalf("unexpected open error: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return NewPollRepository(database)
}

var repositoryTestRegistered bool
var repositoryTestCurrentData repositoryTestData

type repositoryTestData struct {
	poll         *models.Poll
	options      []*models.Option
	voteCounts   map[string]int64
	vote         *models.Vote
	rowsAffected int64
}

type repositoryTestDriver struct{}

func (repositoryTestDriver) Open(name string) (driver.Conn, error) {
	return repositoryTestConn{}, nil
}

type repositoryTestConn struct{}

func (repositoryTestConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (repositoryTestConn) Close() error {
	return nil
}

func (repositoryTestConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not supported")
}

func (repositoryTestConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return driver.RowsAffected(repositoryTestCurrentData.rowsAffected), nil
}

func (repositoryTestConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	switch {
	case strings.Contains(query, "INSERT INTO polls"):
		return repositoryTestRowsFromValues([]string{"id", "title", "admin_key", "status", "created_at"}, [][]driver.Value{
			{args[0].Value, args[1].Value, args[2].Value, "active", time.Now()},
		}), nil
	case strings.Contains(query, "INSERT INTO options"):
		return repositoryTestRowsFromValues([]string{"id", "poll_id", "text", "order"}, [][]driver.Value{
			{args[0].Value, args[1].Value, args[2].Value, args[3].Value},
		}), nil
	case strings.Contains(query, "SELECT id, title, admin_key, status, created_at, closed_at"):
		poll := repositoryTestCurrentData.poll
		if poll == nil {
			return repositoryTestRowsFromValues([]string{"id", "title", "admin_key", "status", "created_at", "closed_at"}, nil), nil
		}
		var closedAt driver.Value
		if poll.ClosedAt != nil {
			closedAt = *poll.ClosedAt
		}
		return repositoryTestRowsFromValues([]string{"id", "title", "admin_key", "status", "created_at", "closed_at"}, [][]driver.Value{
			{poll.ID, poll.Title, poll.AdminKey, poll.Status, poll.CreatedAt, closedAt},
		}), nil
	case strings.Contains(query, "SELECT id, poll_id, text"):
		values := make([][]driver.Value, 0, len(repositoryTestCurrentData.options))
		for _, option := range repositoryTestCurrentData.options {
			values = append(values, []driver.Value{option.ID, option.PollID, option.Text, option.Order})
		}
		return repositoryTestRowsFromValues([]string{"id", "poll_id", "text", "order"}, values), nil
	case strings.Contains(query, "SELECT option_id, COUNT(*)"):
		values := make([][]driver.Value, 0, len(repositoryTestCurrentData.voteCounts))
		for optionID, count := range repositoryTestCurrentData.voteCounts {
			values = append(values, []driver.Value{optionID, count})
		}
		return repositoryTestRowsFromValues([]string{"option_id", "count"}, values), nil
	case strings.Contains(query, "SELECT id, poll_id, option_id, ip, voted_at"):
		vote := repositoryTestCurrentData.vote
		if vote == nil {
			return repositoryTestRowsFromValues([]string{"id", "poll_id", "option_id", "ip", "voted_at"}, nil), nil
		}
		return repositoryTestRowsFromValues([]string{"id", "poll_id", "option_id", "ip", "voted_at"}, [][]driver.Value{
			{vote.ID, vote.PollID, vote.OptionID, vote.IP, vote.VotedAt},
		}), nil
	default:
		return nil, errors.New("unexpected query: " + query)
	}
}

var _ driver.ExecerContext = repositoryTestConn{}
var _ driver.QueryerContext = repositoryTestConn{}

type repositoryTestRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func repositoryTestRowsFromValues(columns []string, values [][]driver.Value) *repositoryTestRows {
	return &repositoryTestRows{columns: columns, values: values}
}

func (r *repositoryTestRows) Columns() []string {
	return r.columns
}

func (r *repositoryTestRows) Close() error {
	return nil
}

func (r *repositoryTestRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}
