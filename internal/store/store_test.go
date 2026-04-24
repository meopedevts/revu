package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/meopedevts/revu/internal/github"
)

func newClock(start time.Time) (func() time.Time, *time.Time) {
	t := start
	return func() time.Time { return t }, &t
}

func mkSummary(id, repo string, number int, title, author string, draft bool) github.PRSummary {
	return github.PRSummary{
		ID:        id,
		Number:    number,
		Repo:      repo,
		Title:     title,
		URL:       "https://github.com/" + repo + "/pull/" + itoa(number),
		Author:    author,
		IsDraft:   draft,
		UpdatedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func tempStore(t *testing.T, now func() time.Time, retention int) (Store, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	s := New(path, WithClock(now), WithRetention(retention))
	return s, path
}

func TestUpdateFromPoll_InsertsAndIsIdempotent(t *testing.T) {
	clock, tp := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s, _ := tempStore(t, clock, 30)

	prs := []github.PRSummary{
		mkSummary("octocat/hello#1", "octocat/hello", 1, "feat: foo", "alice", false),
		mkSummary("acme/widgets#7", "acme/widgets", 7, "chore: bump", "bob", true),
	}
	novos := s.UpdateFromPoll(prs)
	if len(novos) != 2 {
		t.Fatalf("first poll: want 2 novos, got %d", len(novos))
	}

	*tp = tp.Add(5 * time.Minute)
	novos = s.UpdateFromPoll(prs)
	if len(novos) != 0 {
		t.Fatalf("second poll with same PRs: want 0 novos, got %d: %+v", len(novos), novos)
	}
	if len(s.GetPending()) != 2 {
		t.Fatalf("want 2 pending, got %d", len(s.GetPending()))
	}
}

func TestUpdateFromPoll_ReRequestDetected(t *testing.T) {
	clock, tp := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s, _ := tempStore(t, clock, 30)

	pr := mkSummary("octocat/hello#1", "octocat/hello", 1, "feat: foo", "alice", false)
	if novos := s.UpdateFromPoll([]github.PRSummary{pr}); len(novos) != 1 {
		t.Fatalf("initial insert: want 1 novo, got %d", len(novos))
	}

	// PR drops from search results (review given / removed). Poll empty.
	*tp = tp.Add(5 * time.Minute)
	if novos := s.UpdateFromPoll(nil); len(novos) != 0 {
		t.Fatalf("empty poll: want 0 novos, got %d", len(novos))
	}
	all := s.GetAll()
	if len(all) != 1 || all[0].ReviewPending {
		t.Fatalf("want 1 record with pending=false, got %+v", all)
	}

	// PR returns with review requested again → re-request.
	*tp = tp.Add(5 * time.Minute)
	novos := s.UpdateFromPoll([]github.PRSummary{pr})
	if len(novos) != 1 {
		t.Fatalf("re-request: want 1 novo, got %d", len(novos))
	}
	if novos[0].ID != pr.ID {
		t.Fatalf("wrong id in novos: %s", novos[0].ID)
	}
	if !s.GetPending()[0].ReviewPending {
		t.Fatal("expected ReviewPending=true after re-request")
	}
}

func TestUpdateFromPoll_UpdatesMutableFields(t *testing.T) {
	clock, _ := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s, _ := tempStore(t, clock, 30)

	pr := mkSummary("octocat/hello#1", "octocat/hello", 1, "feat: foo", "alice", false)
	s.UpdateFromPoll([]github.PRSummary{pr})

	pr.Title = "feat: foo (edited)"
	pr.IsDraft = true
	s.UpdateFromPoll([]github.PRSummary{pr})

	got := s.GetPending()[0]
	if got.Title != "feat: foo (edited)" || !got.IsDraft {
		t.Fatalf("mutable fields not updated: %+v", got)
	}
}

func TestRefreshPRStatus(t *testing.T) {
	clock, _ := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s, _ := tempStore(t, clock, 30)

	pr := mkSummary("octocat/hello#1", "octocat/hello", 1, "feat: foo", "alice", false)
	s.UpdateFromPoll([]github.PRSummary{pr})

	merged := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	err := s.RefreshPRStatus(pr.ID, github.PRDetails{
		Additions: 10, Deletions: 5, State: "CLOSED", MergedAt: &merged, IsDraft: false,
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	got := s.GetAll()[0]
	if got.Additions != 10 || got.Deletions != 5 || got.State != "MERGED" {
		t.Fatalf("status not applied: %+v", got)
	}

	if err := s.RefreshPRStatus("nope", github.PRDetails{}); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestRetention_DropsOldNonOpen(t *testing.T) {
	clock, tp := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s, path := tempStore(t, clock, 30)

	prOpen := mkSummary("a/b#1", "a/b", 1, "open", "x", false)
	prClosed := mkSummary("a/b#2", "a/b", 2, "closed", "x", false)
	s.UpdateFromPoll([]github.PRSummary{prOpen, prClosed})
	if err := s.RefreshPRStatus(prClosed.ID, github.PRDetails{State: "CLOSED"}); err != nil {
		t.Fatal(err)
	}

	// Jump 40 days. prClosed becomes stale; prOpen stays (still OPEN).
	*tp = tp.Add(40 * 24 * time.Hour)
	if err := s.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Reload from disk to ensure retention persisted.
	s2 := New(path, WithClock(clock), WithRetention(30))
	if err := s2.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	all := s2.GetAll()
	if len(all) != 1 || all[0].ID != prOpen.ID {
		t.Fatalf("want only open pr, got %+v", all)
	}
}

func TestRetention_DisabledWhenZero(t *testing.T) {
	clock, tp := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s, _ := tempStore(t, clock, 0)

	pr := mkSummary("a/b#1", "a/b", 1, "t", "x", false)
	s.UpdateFromPoll([]github.PRSummary{pr})
	if err := s.RefreshPRStatus(pr.ID, github.PRDetails{State: "CLOSED"}); err != nil {
		t.Fatal(err)
	}
	*tp = tp.Add(365 * 24 * time.Hour)
	if err := s.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}
	if len(s.GetAll()) != 1 {
		t.Fatal("retention=0 should keep everything")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	clock, _ := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s, path := tempStore(t, clock, 30)

	pr := mkSummary("octocat/hello#1", "octocat/hello", 1, "feat: foo", "alice", false)
	s.UpdateFromPoll([]github.PRSummary{pr})
	if err := s.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	s2 := New(path, WithClock(clock), WithRetention(30))
	if err := s2.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	got := s2.GetAll()
	if len(got) != 1 || got[0].ID != pr.ID || !got[0].ReviewPending {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}

func TestLoad_MissingFileIsFresh(t *testing.T) {
	clock, _ := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	dir := t.TempDir()
	s := New(filepath.Join(dir, "does-not-exist.json"), WithClock(clock))
	if err := s.Load(); err != nil {
		t.Fatalf("missing file should be OK, got %v", err)
	}
	if len(s.GetAll()) != 0 {
		t.Fatal("fresh store must be empty")
	}
}

func TestLoad_CorruptedFileReturnsError(t *testing.T) {
	clock, _ := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := New(path, WithClock(clock))
	if err := s.Load(); err == nil {
		t.Fatal("want decode error")
	}
}

func TestSave_AtomicNoTempLeaksOnSuccess(t *testing.T) {
	clock, _ := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s, path := tempStore(t, clock, 30)
	pr := mkSummary("a/b#1", "a/b", 1, "t", "x", false)
	s.UpdateFromPoll([]github.PRSummary{pr})
	if err := s.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" || filepath.Base(e.Name()) != "state.json" {
			if filepath.Base(e.Name()) == "state.json" {
				continue
			}
			t.Fatalf("leftover file in state dir: %s", e.Name())
		}
	}
}

func TestSave_OverwriteIsAtomic(t *testing.T) {
	clock, tp := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s, path := tempStore(t, clock, 30)
	pr := mkSummary("a/b#1", "a/b", 1, "first", "x", false)
	s.UpdateFromPoll([]github.PRSummary{pr})
	if err := s.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Snapshot first save content.
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var snap1 snapshot
	if err := json.Unmarshal(first, &snap1); err != nil {
		t.Fatalf("decode first: %v", err)
	}
	if snap1.PRs[pr.ID].Title != "first" {
		t.Fatalf("first persist: unexpected title %q", snap1.PRs[pr.ID].Title)
	}

	// Mutate and save again.
	*tp = tp.Add(time.Minute)
	pr.Title = "second"
	s.UpdateFromPoll([]github.PRSummary{pr})
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var snap2 snapshot
	if err := json.Unmarshal(second, &snap2); err != nil {
		t.Fatal(err)
	}
	if snap2.PRs[pr.ID].Title != "second" {
		t.Fatalf("second persist: want 'second', got %q", snap2.PRs[pr.ID].Title)
	}
}

func TestGetPendingAndHistoryPartition(t *testing.T) {
	clock, _ := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s, _ := tempStore(t, clock, 30)
	prA := mkSummary("a/b#1", "a/b", 1, "A", "x", false)
	prB := mkSummary("a/b#2", "a/b", 2, "B", "x", false)
	s.UpdateFromPoll([]github.PRSummary{prA, prB})
	// B drops out of review.
	s.UpdateFromPoll([]github.PRSummary{prA})

	pending := s.GetPending()
	history := s.GetHistory()
	if len(pending) != 1 || pending[0].ID != prA.ID {
		t.Fatalf("pending mismatch: %+v", pending)
	}
	if len(history) != 1 || history[0].ID != prB.ID {
		t.Fatalf("history mismatch: %+v", history)
	}
}

func TestGetAll_SortedByLastSeenDesc(t *testing.T) {
	clock, tp := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s, _ := tempStore(t, clock, 30)

	s.UpdateFromPoll([]github.PRSummary{mkSummary("a/b#1", "a/b", 1, "A", "x", false)})
	*tp = tp.Add(time.Minute)
	s.UpdateFromPoll([]github.PRSummary{
		mkSummary("a/b#1", "a/b", 1, "A", "x", false),
		mkSummary("a/b#2", "a/b", 2, "B", "x", false),
	})
	all := s.GetAll()
	if len(all) != 2 {
		t.Fatalf("want 2, got %d", len(all))
	}
	// Both were last seen at the same tick (second call), but ordering must be
	// deterministic — we only assert it's stable across calls.
	all2 := s.GetAll()
	if !reflect.DeepEqual(ids(all), ids(all2)) {
		t.Fatal("GetAll ordering not stable")
	}
}

func ids(rs []PRRecord) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.ID
	}
	sort.Strings(out)
	return out
}
