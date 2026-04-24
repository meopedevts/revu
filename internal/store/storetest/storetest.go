// Package storetest exposes migration-aware test helpers for packages that
// need a throwaway *sql.DB pre-populated with the revu schema (profiles,
// prs, meta, etc). Keep this tiny — production code never imports it.
package storetest

import (
	"database/sql"
	"sync"
	"testing"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"

	"github.com/meopedevts/revu/internal/store/migrations"
)

var (
	gooseOnce sync.Once
	gooseErr  error
)

// OpenMem opens an in-memory SQLite DB and applies all pending migrations.
// The DB is closed via t.Cleanup. Use for tests in sibling packages that
// need the store schema without booting the full Store façade.
func OpenMem(t testing.TB) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Ping(); err != nil {
		t.Fatalf("ping sqlite: %v", err)
	}
	gooseOnce.Do(func() {
		goose.SetBaseFS(migrations.FS)
		gooseErr = goose.SetDialect("sqlite3")
		goose.SetLogger(goose.NopLogger())
	})
	if gooseErr != nil {
		t.Fatalf("goose setup: %v", gooseErr)
	}
	if err := goose.Up(db, "."); err != nil {
		t.Fatalf("goose up: %v", err)
	}
	return db
}
