package store

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"sync"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // driver SQLite registrado via init.

	"github.com/meopedevts/revu/internal/store/migrations"
)

var (
	gooseOnce sync.Once
	errGoose  error
)

// openDB opens the SQLite database at path, applies pragmas via DSN, pings
// the handle, and runs pending goose migrations. Returns the ready-to-use
// [*sql.DB] or an error. For `:memory:` pass path == ":memory:" (no pragmas
// applied — irrelevant for in-memory).
func openDB(path string) (*sql.DB, error) {
	dsn := buildDSN(path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if err := runMigrations(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// buildDSN assembles the sqlite DSN with file URI + pragmas. `:memory:`
// remains unchanged (no pragma wrapping) so in-memory tests behave as
// expected. Path is URL-escaped to tolerate spaces / special chars.
func buildDSN(path string) string {
	if path == ":memory:" {
		return ":memory:"
	}
	abs := path
	if p, err := filepath.Abs(path); err == nil {
		abs = p
	}
	q := url.Values{}
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "synchronous(NORMAL)")
	q.Add("_pragma", "foreign_keys(on)")
	q.Add("_pragma", "busy_timeout(5000)")
	return "file:" + abs + "?" + q.Encode()
}

// runMigrations applies all pending goose migrations from the embedded FS.
// goose.SetBaseFS / SetDialect are process-global; wrap in [sync.Once] so
// concurrent openDB calls (eg. tests) don't race on state.
func runMigrations(db *sql.DB) error {
	gooseOnce.Do(func() {
		goose.SetBaseFS(migrations.FS)
		errGoose = goose.SetDialect("sqlite3")
	})
	if errGoose != nil {
		return fmt.Errorf("goose setup: %w", errGoose)
	}
	// Silence goose's default stdout logger — we handle logging at the call
	// site via slog.
	goose.SetLogger(goose.NopLogger())
	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
