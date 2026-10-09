package database

import (
	"context"
	"os"
	"strings"
	"testing"
)

// Runs against a real test database (make test passes the settings).
func TestRequireColumns(t *testing.T) {
	name := os.Getenv("TEST_MYSQL_DATABASE")
	if !strings.HasSuffix(name, "_test") {
		t.Skip("set TEST_MYSQL_DATABASE to a *_test database to run")
	}
	ctx := context.Background()
	db, err := Open(ctx, Config{Host: os.Getenv("MYSQL_HOST"), Port: os.Getenv("MYSQL_PORT"),
		User: os.Getenv("MYSQL_USER"), Password: os.Getenv("MYSQL_PASSWORD"), Name: name})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := RequireColumns(ctx, db, Requirement{"users", "tags", "002"}); err != nil {
		t.Fatalf("present column: %v", err)
	}
	err = RequireColumns(ctx, db, Requirement{"users", "no_such_column", "migrations/009_x.sql"})
	if err == nil || !strings.Contains(err.Error(), "migrations/009_x.sql") {
		t.Fatalf("missing column must name the migration to apply: %v", err)
	}
}
