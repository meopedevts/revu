package store

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeStateJSON(t *testing.T, path string, snap jsonImportSnapshot) {
	t.Helper()
	b, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestMigrateJSON_HappyPath(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "state.json")
	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	lastPoll := now.Add(-5 * time.Minute)

	snap := jsonImportSnapshot{
		PRs: map[string]PRRecord{
			"a/b#1": {
				ID: "a/b#1", Number: 1, Repo: "a/b", Title: "A", Author: "x",
				URL: "https://github.com/a/b/pull/1", State: "OPEN",
				ReviewPending: true,
				FirstSeenAt:   now.Add(-time.Hour),
				LastSeenAt:    now.Add(-time.Minute),
			},
			"a/b#2": {
				ID: "a/b#2", Number: 2, Repo: "a/b", Title: "B", Author: "y",
				URL: "https://github.com/a/b/pull/2", State: "CLOSED",
				Additions: 10, Deletions: 5,
				ReviewPending: false,
				FirstSeenAt:   now.Add(-2 * time.Hour),
				LastSeenAt:    now.Add(-time.Hour),
			},
		},
		LastPollAt: &lastPoll,
	}
	writeStateJSON(t, jsonPath, snap)

	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	defer db.Close()

	if err := migrateJSONIfPresent(ctx, db, jsonPath, now, nil); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// PRs migrated.
	s := newSQLiteFromDB(db)
	all := s.GetAll(context.Background())
	if len(all) != 2 {
		t.Fatalf("want 2 PRs migrated, got %d", len(all))
	}

	// Meta set.
	if v, ok, _ := getMeta(ctx, db, metaLastPollAt); !ok || v == "" {
		t.Fatal("last_poll_at not set")
	}
	if v, ok, _ := getMeta(ctx, db, metaJSONMigratedAt); !ok || v == "" {
		t.Fatal("json_migrated_at not set")
	}

	// state.json renamed.
	if _, err := os.Stat(jsonPath); !os.IsNotExist(err) {
		t.Fatalf("state.json should be gone: err=%v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var renamed string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "state.json.migrated-") {
			renamed = e.Name()
		}
	}
	if renamed == "" {
		t.Fatal("no state.json.migrated-<ts> found")
	}
}

func TestMigrateJSON_NoStateFile_NoOp(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "state.json")

	db, err := openDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := migrateJSONIfPresent(ctx, db, jsonPath, time.Now(), nil); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if v, ok, _ := getMeta(ctx, db, metaJSONMigratedAt); ok {
		t.Fatalf("meta should be absent after no-op migration, got %q", v)
	}
}

func TestMigrateJSON_AlreadyMigrated_RenamesOrphan(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "state.json")
	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)

	db, err := openDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Simulate "migration already happened" — set meta and leave state.json
	// in place as an orphan from a previous crash.
	if _, err := db.Exec(qSetMeta, metaJSONMigratedAt, formatTime(now.Add(-time.Hour))); err != nil {
		t.Fatal(err)
	}
	writeStateJSON(t, jsonPath, jsonImportSnapshot{PRs: map[string]PRRecord{}})

	if err := migrateJSONIfPresent(ctx, db, jsonPath, now, nil); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// state.json must be gone; rename happened.
	if _, err := os.Stat(jsonPath); !os.IsNotExist(err) {
		t.Fatalf("orphan state.json should have been renamed: err=%v", err)
	}
	// Nothing re-imported.
	s := newSQLiteFromDB(db)
	if len(s.GetAll(context.Background())) != 0 {
		t.Fatal("orphan rename should not re-import PRs")
	}
}

func TestMigrateJSON_CorruptJSON_ReturnsError(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "state.json")
	if err := os.WriteFile(jsonPath, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	db, err := openDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := migrateJSONIfPresent(ctx, db, jsonPath, time.Now(), nil); err == nil {
		t.Fatal("want decode error, got nil")
	}
	if _, err := os.Stat(jsonPath); err != nil {
		t.Fatalf("state.json should be untouched on error: %v", err)
	}
	s := newSQLiteFromDB(db)
	if len(s.GetAll(context.Background())) != 0 {
		t.Fatal("db should be empty after failed migration")
	}
}

func TestMigrateJSON_Idempotent(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "state.json")
	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)

	snap := jsonImportSnapshot{
		PRs: map[string]PRRecord{
			"a/b#1": {
				ID: "a/b#1", Number: 1, Repo: "a/b", Title: "A", Author: "x",
				URL: "https://github.com/a/b/pull/1", State: "OPEN",
				ReviewPending: true,
				FirstSeenAt:   now, LastSeenAt: now,
			},
		},
	}
	writeStateJSON(t, jsonPath, snap)

	db, err := openDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := migrateJSONIfPresent(ctx, db, jsonPath, now, nil); err != nil {
		t.Fatal(err)
	}
	// Call again on the same DB. No state.json this time.
	if err := migrateJSONIfPresent(ctx, db, jsonPath, now.Add(time.Hour), nil); err != nil {
		t.Fatal(err)
	}
	s := newSQLiteFromDB(db)
	if len(s.GetAll(context.Background())) != 1 {
		t.Fatalf("want 1 PR (no dup), got %d", len(s.GetAll(context.Background())))
	}
}
