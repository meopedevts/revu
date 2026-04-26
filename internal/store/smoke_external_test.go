//go:build smoke

package store

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

// TestSmoke_RealStateJSON runs the JSON → SQLite migration against the
// operator's actual ~/.config/revu/state.json. Gated behind the `smoke`
// build tag so it only runs on demand: `go test -tags smoke -run Smoke
// ./internal/store/`. The test copies state.json to a fresh temp dir —
// the original file under ~/.config/revu is never touched.
func TestSmoke_RealStateJSON(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("home: %v", err)
	}
	src := filepath.Join(home, ".config", "revu", "state.json")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Skipf("no real state.json at %s: %v", src, err)
	}
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "state.json")
	if err := os.WriteFile(jsonPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dir, "revu.db")

	s := New(dbPath,
		WithRetention(30),
		WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		WithJSONMigration(jsonPath),
	)
	if err := s.Load(context.Background()); err != nil {
		t.Fatalf("load: %v", err)
	}
	defer s.Close(context.Background())

	all := s.GetAll(context.Background())
	pending := s.GetPending(context.Background())
	history := s.GetHistory(context.Background())

	t.Logf("migrated: total=%d pending=%d history=%d", len(all), len(pending), len(history))
	for _, pr := range all {
		t.Logf("  [%s] state=%s pending=%v", pr.ID, pr.State, pr.ReviewPending)
	}

	// state.json renamed, revu.db present.
	if _, err := os.Stat(jsonPath); !os.IsNotExist(err) {
		t.Fatalf("state.json should be renamed, stat err=%v", err)
	}
	entries, _ := os.ReadDir(dir)
	var sawMigrated, sawDB bool
	for _, e := range entries {
		if e.Name() == "revu.db" {
			sawDB = true
		}
		if len(e.Name()) > len("state.json.migrated-") &&
			e.Name()[:len("state.json.migrated-")] == "state.json.migrated-" {
			sawMigrated = true
		}
	}
	if !sawDB {
		t.Error("revu.db not created")
	}
	if !sawMigrated {
		t.Error("state.json.migrated-<ts> not found")
	}

	// Second Load must be idempotent (no state.json remains; no-op).
	s2 := New(dbPath,
		WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		WithJSONMigration(jsonPath),
	)
	if err := s2.Load(context.Background()); err != nil {
		t.Fatalf("reload: %v", err)
	}
	defer s2.Close(context.Background())
	if got := len(s2.GetAll(context.Background())); got != len(all) {
		t.Fatalf("reload count drift: first=%d second=%d", len(all), got)
	}
}
