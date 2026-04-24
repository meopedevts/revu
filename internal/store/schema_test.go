package store

import (
	"testing"
)

func TestSchema_TablesAndIndexesExist(t *testing.T) {
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	defer db.Close()

	want := map[string]string{
		"prs":              "table",
		"meta":             "table",
		"idx_prs_pending":  "index",
		"idx_prs_state":    "index",
		"goose_db_version": "table",
	}
	rows, err := db.Query(`SELECT name, type FROM sqlite_master WHERE name NOT LIKE 'sqlite_%'`)
	if err != nil {
		t.Fatalf("query master: %v", err)
	}
	defer rows.Close()

	got := map[string]string{}
	for rows.Next() {
		var name, typ string
		if err := rows.Scan(&name, &typ); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got[name] = typ
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}

	for name, typ := range want {
		if got[name] != typ {
			t.Errorf("missing %s %s (got=%q)", typ, name, got[name])
		}
	}
}

func TestSchema_PRsColumns(t *testing.T) {
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	defer db.Close()

	rows, err := db.Query(`PRAGMA table_info(prs)`)
	if err != nil {
		t.Fatalf("pragma: %v", err)
	}
	defer rows.Close()
	cols := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan: %v", err)
		}
		cols[name] = true
	}
	for _, name := range []string{
		"id", "number", "repo", "title", "author", "url", "state",
		"is_draft", "additions", "deletions", "review_pending",
		"first_seen_at", "last_seen_at", "last_notified_at",
	} {
		if !cols[name] {
			t.Errorf("column missing: %s", name)
		}
	}
}

func TestSchema_Idempotent(t *testing.T) {
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	defer db.Close()

	// Rodar migrations uma segunda vez contra o mesmo DB não deve falhar.
	if err := runMigrations(db); err != nil {
		t.Fatalf("second runMigrations: %v", err)
	}
}
