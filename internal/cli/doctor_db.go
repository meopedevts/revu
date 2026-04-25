package cli

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"

	_ "modernc.org/sqlite" // registra driver "sqlite" via init.
)

// openReadOnlyDB opens the revu SQLite database in read-only mode. Used by
// doctor checks so we never mutate state during diagnostics.
func openReadOnlyDB(ctx context.Context, path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?mode=ro", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// checkDBPath reports on the presence of the SQLite database file. A
// missing file is *not* a failure — fresh installs run `revu doctor` before
// their first `revu run` and we shouldn't yell at them about it.
func checkDBPath(path string) checkResult {
	info, err := os.Stat(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return checkResult{
			Name:   "DB path",
			Detail: path + " (ainda não criado — rode `revu run` pelo menos uma vez)",
			OK:     true,
		}
	case err != nil:
		return checkResult{Name: "DB path", Detail: err.Error()}
	default:
		return checkResult{
			Name:   "DB path",
			Detail: fmt.Sprintf("%s (%d bytes)", path, info.Size()),
			OK:     true,
		}
	}
}

// checkSchemaVersion reports the goose_db_version. Skipped silently (as OK)
// when the DB is absent.
func checkSchemaVersion(ctx context.Context, path string) checkResult {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return checkResult{Name: "schema version", Detail: "DB ausente — skip", OK: true}
	}
	db, err := openReadOnlyDB(ctx, path)
	if err != nil {
		return checkResult{Name: "schema version", Detail: err.Error()}
	}
	defer db.Close()
	var version int64
	err = db.QueryRowContext(ctx, `SELECT MAX(version_id) FROM goose_db_version`).Scan(&version)
	if err != nil {
		return checkResult{Name: "schema version", Detail: err.Error()}
	}
	return checkResult{
		Name:   "schema version",
		Detail: fmt.Sprintf("%d", version),
		OK:     true,
	}
}

// checkPRCounts reports total/pending/history PR counts. Skipped silently
// when the DB is absent.
func checkPRCounts(ctx context.Context, path string) checkResult {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return checkResult{Name: "PR counts", Detail: "DB ausente — skip", OK: true}
	}
	db, err := openReadOnlyDB(ctx, path)
	if err != nil {
		return checkResult{Name: "PR counts", Detail: err.Error()}
	}
	defer db.Close()
	var total, pending, history int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM prs`).Scan(&total); err != nil {
		return checkResult{Name: "PR counts", Detail: err.Error()}
	}
	// REV-16: history is PRs finalized on GitHub (MERGED or CLOSED); anything
	// else remains pending regardless of the review state.
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM prs WHERE state NOT IN ('MERGED', 'CLOSED')`).Scan(&pending); err != nil {
		return checkResult{Name: "PR counts", Detail: err.Error()}
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM prs WHERE state IN ('MERGED', 'CLOSED')`).Scan(&history); err != nil {
		return checkResult{Name: "PR counts", Detail: err.Error()}
	}
	return checkResult{
		Name:   "PR counts",
		Detail: fmt.Sprintf("total=%d pending=%d history=%d", total, pending, history),
		OK:     true,
	}
}
