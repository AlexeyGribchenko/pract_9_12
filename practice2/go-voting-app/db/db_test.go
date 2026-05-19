package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
)

func TestInitSchema_Success(t *testing.T) {
	database := openSchemaTestDB(t, nil)
	defer database.Close()

	if err := InitSchema(database); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitSchema_ReturnsExecError(t *testing.T) {
	expectedErr := errors.New("schema exec failed")
	database := openSchemaTestDB(t, expectedErr)
	defer database.Close()

	err := InitSchema(database)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected schema exec error, got %v", err)
	}
}

func openSchemaTestDB(t *testing.T, execErr error) *sql.DB {
	t.Helper()

	driverName := "schema-test-driver"
	schemaTestExecErr = execErr
	if !schemaTestRegistered {
		sql.Register(driverName, schemaTestDriver{})
		schemaTestRegistered = true
	}

	database, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("unexpected open error: %v", err)
	}
	return database
}

var schemaTestRegistered bool
var schemaTestExecErr error

type schemaTestDriver struct{}

func (schemaTestDriver) Open(name string) (driver.Conn, error) {
	return schemaTestConn{}, nil
}

type schemaTestConn struct{}

func (schemaTestConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (schemaTestConn) Close() error {
	return nil
}

func (schemaTestConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not supported")
}

func (schemaTestConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if !strings.Contains(query, "CREATE TABLE IF NOT EXISTS polls") {
		return nil, errors.New("schema does not create polls table")
	}
	if !strings.Contains(query, "CREATE TABLE IF NOT EXISTS votes") {
		return nil, errors.New("schema does not create votes table")
	}
	if schemaTestExecErr != nil {
		return nil, schemaTestExecErr
	}
	return driver.RowsAffected(1), nil
}

var _ driver.ExecerContext = schemaTestConn{}
