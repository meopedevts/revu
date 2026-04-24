package cli

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// writeTestDB creates a minimal SQLite file at path with the columns the
// doctor queries read. Pending rows get state='OPEN' and PENDING review;
// history rows get state='MERGED' and a submitted review so they satisfy
// the REV-16 history predicate.
func writeTestDB(t *testing.T, path string, prs int, pending int) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	stmts := []string{
		`CREATE TABLE prs (
			id TEXT PRIMARY KEY,
			review_pending INTEGER NOT NULL,
			state TEXT NOT NULL,
			review_state TEXT NOT NULL
		)`,
		`CREATE TABLE goose_db_version (version_id INTEGER NOT NULL)`,
		`INSERT INTO goose_db_version (version_id) VALUES (0), (1)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < prs; i++ {
		reviewPending := 0
		state := "MERGED"
		review := "APPROVED"
		if i < pending {
			reviewPending = 1
			state = "OPEN"
			review = "PENDING"
		}
		if _, err := db.Exec(
			`INSERT INTO prs (id, review_pending, state, review_state) VALUES (?, ?, ?, ?)`,
			fmt.Sprintf("id-%d", i), reviewPending, state, review); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCheckDBPath(t *testing.T) {
	dir := t.TempDir()

	// Missing file → OK with "ainda não criado" detail.
	r := checkDBPath(filepath.Join(dir, "nope.db"))
	if !r.OK {
		t.Fatalf("missing file should be OK: %+v", r)
	}
	if !strings.Contains(r.Detail, "ainda não criado") {
		t.Fatalf("unexpected detail: %q", r.Detail)
	}

	// Present file → OK with size.
	path := filepath.Join(dir, "revu.db")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	r = checkDBPath(path)
	if !r.OK || !strings.Contains(r.Detail, "5 bytes") {
		t.Fatalf("present file should be OK with size: %+v", r)
	}
}

func TestCheckSchemaVersion(t *testing.T) {
	dir := t.TempDir()

	// Missing file → OK skip.
	r := checkSchemaVersion(filepath.Join(dir, "nope.db"))
	if !r.OK || !strings.Contains(r.Detail, "skip") {
		t.Fatalf("missing file should skip: %+v", r)
	}

	// Present file with goose table → reports version.
	path := filepath.Join(dir, "revu.db")
	writeTestDB(t, path, 0, 0)
	r = checkSchemaVersion(path)
	if !r.OK || r.Detail != "1" {
		t.Fatalf("schema version: %+v", r)
	}
}

func TestCheckPRCounts(t *testing.T) {
	dir := t.TempDir()

	// Missing file → OK skip.
	r := checkPRCounts(filepath.Join(dir, "nope.db"))
	if !r.OK {
		t.Fatalf("missing file should skip: %+v", r)
	}

	path := filepath.Join(dir, "revu.db")
	writeTestDB(t, path, 5, 2)
	r = checkPRCounts(path)
	if !r.OK {
		t.Fatalf("counts failed: %+v", r)
	}
	for _, want := range []string{"total=5", "pending=2", "history=3"} {
		if !strings.Contains(r.Detail, want) {
			t.Fatalf("detail missing %q: %q", want, r.Detail)
		}
	}
}
