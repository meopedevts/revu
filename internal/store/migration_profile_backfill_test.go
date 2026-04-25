package store

import (
	"database/sql"
	"testing"

	"github.com/pressly/goose/v3"

	"github.com/meopedevts/revu/internal/store/migrations"
)

// TestMigration_ProfileBackfill simulates an existing MVP install (prs rows
// from before profiles existed) and verifies 00003 backfills profile_id with
// the active profile id.
func TestMigration_ProfileBackfill(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("dialect: %v", err)
	}
	goose.SetLogger(goose.NopLogger())

	// Up to 00001 only: we're simulating a legacy DB with prs but no profiles.
	if err := goose.UpTo(db, ".", 1); err != nil {
		t.Fatalf("up to 1: %v", err)
	}

	if _, err := db.Exec(
		`INSERT INTO prs (id, number, repo, title, author, url, state, is_draft, review_pending, first_seen_at, last_seen_at)
		VALUES ('octo/repo#1', 1, 'octo/repo', 'Title', 'author', 'http://x', 'OPEN', 0, 1, '2026-04-24T00:00:00Z', '2026-04-24T00:00:00Z')`,
	); err != nil {
		t.Fatalf("insert legacy pr: %v", err)
	}

	// Apply the remaining migrations (profiles + prs.profile_id backfill).
	if err := goose.Up(db, "."); err != nil {
		t.Fatalf("goose up: %v", err)
	}

	var seedID string
	if err := db.QueryRow(`SELECT id FROM profiles WHERE is_active = 1`).Scan(&seedID); err != nil {
		t.Fatalf("seed select: %v", err)
	}

	var prProfile string
	if err := db.QueryRow(`SELECT profile_id FROM prs WHERE id = 'octo/repo#1'`).Scan(&prProfile); err != nil {
		t.Fatalf("pr profile_id: %v", err)
	}
	if prProfile != seedID {
		t.Errorf("backfill mismatch: prs.profile_id=%q seed.id=%q", prProfile, seedID)
	}
}
