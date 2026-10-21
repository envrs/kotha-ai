package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/kothagpt/kotha/internal/config"
)

// OpenTestDB connects to a fresh temp SQLite database, applies migrations
// and returns a prepared Queries handle. Exported for use by other packages'
// tests.
func OpenTestDB(t *testing.T) (*Queries, *sql.DB) {
	t.Helper()
	dir := t.TempDir()
	config.SetDefaultProvider(config.NewDefaultProvider(&config.Config{Data: config.Data{Directory: dir}}))

	conn, err := Connect()
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	q, err := Prepare(context.Background(), conn)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	t.Cleanup(func() { _ = q.Close() })
	return q, conn
}