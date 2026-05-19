package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func Connect(host, port, user, password, dbname string) (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	log.Println("✓ Connected to PostgreSQL")
	return db, nil
}

// InitSchema создает таблицы
func InitSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS polls (
		id VARCHAR(36) PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		admin_key VARCHAR(36) NOT NULL UNIQUE,
		status VARCHAR(20) NOT NULL DEFAULT 'active',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		closed_at TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS options (
		id VARCHAR(36) PRIMARY KEY,
		poll_id VARCHAR(36) NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
		text VARCHAR(255) NOT NULL,
		"order" INT NOT NULL,
		UNIQUE(poll_id, "order")
	);

	CREATE TABLE IF NOT EXISTS votes (
		id VARCHAR(36) PRIMARY KEY,
		poll_id VARCHAR(36) NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
		option_id VARCHAR(36) NOT NULL REFERENCES options(id) ON DELETE CASCADE,
		ip VARCHAR(45) NOT NULL,
		voted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_polls_status ON polls(status);
	CREATE INDEX IF NOT EXISTS idx_options_poll_id ON options(poll_id);
	CREATE INDEX IF NOT EXISTS idx_votes_poll_id ON votes(poll_id);
	CREATE INDEX IF NOT EXISTS idx_votes_ip ON votes(ip);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_votes_poll_ip ON votes(poll_id, ip);
	`

	_, err := db.Exec(schema)
	return err
}
