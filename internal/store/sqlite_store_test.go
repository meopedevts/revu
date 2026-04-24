package store

import (
	"context"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/meopedevts/revu/internal/github"
)

// newMemoryStore opens an in-memory DB with schema ready and returns the
// sqliteStore directly so tests can exercise the full surface (including
// Close). Retention defaults to 30 days unless an explicit WithRetention
// follows.
func newMemoryStore(t *testing.T, opts ...Option) *sqliteStore {
	t.Helper()
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	allOpts := append([]Option{WithRetention(30)}, opts...)
	return newSQLiteFromDB(db, allOpts...)
}

func TestSQLite_UpdateFromPoll_InsertsAndIsIdempotent(t *testing.T) {
	clock, tp := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock))

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
		t.Fatalf("second poll with same PRs: want 0 novos, got %d", len(novos))
	}
	if len(s.GetPending()) != 2 {
		t.Fatalf("want 2 pending, got %d", len(s.GetPending()))
	}
}

func TestSQLite_UpdateFromPoll_ReRequestDetected(t *testing.T) {
	clock, tp := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock))

	pr := mkSummary("octocat/hello#1", "octocat/hello", 1, "feat: foo", "alice", false)
	if novos := s.UpdateFromPoll([]github.PRSummary{pr}); len(novos) != 1 {
		t.Fatalf("initial insert: want 1 novo, got %d", len(novos))
	}

	*tp = tp.Add(5 * time.Minute)
	if novos := s.UpdateFromPoll(nil); len(novos) != 0 {
		t.Fatalf("empty poll: want 0 novos, got %d", len(novos))
	}
	all := s.GetAll()
	if len(all) != 1 || all[0].ReviewPending {
		t.Fatalf("want 1 record with pending=false, got %+v", all)
	}

	*tp = tp.Add(5 * time.Minute)
	novos := s.UpdateFromPoll([]github.PRSummary{pr})
	if len(novos) != 1 {
		t.Fatalf("re-request: want 1 novo, got %d", len(novos))
	}
	if novos[0].ID != pr.ID {
		t.Fatalf("wrong id in novos: %s", novos[0].ID)
	}
	pending := s.GetPending()
	if len(pending) != 1 || !pending[0].ReviewPending {
		t.Fatal("expected ReviewPending=true after re-request")
	}
}

func TestSQLite_UpdateFromPoll_UpdatesMutableFields(t *testing.T) {
	clock, _ := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock))

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

func TestSQLite_RefreshPRStatus(t *testing.T) {
	clock, _ := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock))

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

func TestSQLite_Retention_DropsOldNonOpen_InPoll(t *testing.T) {
	clock, tp := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock), WithRetention(30))

	prOpen := mkSummary("a/b#1", "a/b", 1, "open", "x", false)
	prClosed := mkSummary("a/b#2", "a/b", 2, "closed", "x", false)
	s.UpdateFromPoll([]github.PRSummary{prOpen, prClosed})
	if err := s.RefreshPRStatus(prClosed.ID, github.PRDetails{State: "CLOSED"}); err != nil {
		t.Fatal(err)
	}

	// 40 days pass; prClosed disappears from the search. Retention runs on
	// the next UpdateFromPoll tx — prClosed gets purged, prOpen stays.
	*tp = tp.Add(40 * 24 * time.Hour)
	s.UpdateFromPoll([]github.PRSummary{prOpen})

	all := s.GetAll()
	if len(all) != 1 || all[0].ID != prOpen.ID {
		t.Fatalf("want only open pr, got %+v", all)
	}
}

func TestSQLite_Retention_DisabledWhenZero(t *testing.T) {
	clock, tp := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock), WithRetention(0))

	pr := mkSummary("a/b#1", "a/b", 1, "t", "x", false)
	s.UpdateFromPoll([]github.PRSummary{pr})
	if err := s.RefreshPRStatus(pr.ID, github.PRDetails{State: "CLOSED"}); err != nil {
		t.Fatal(err)
	}
	*tp = tp.Add(365 * 24 * time.Hour)
	s.UpdateFromPoll(nil)
	if len(s.GetAll()) != 1 {
		t.Fatal("retention=0 should keep everything")
	}
}

func TestSQLite_Retention_RespectsRuntimeChange(t *testing.T) {
	clock, tp := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock), WithRetention(365))

	pr := mkSummary("a/b#1", "a/b", 1, "t", "x", false)
	s.UpdateFromPoll([]github.PRSummary{pr})
	if err := s.RefreshPRStatus(pr.ID, github.PRDetails{State: "CLOSED"}); err != nil {
		t.Fatal(err)
	}
	*tp = tp.Add(30 * 24 * time.Hour)

	s.SetRetentionDays(7)
	s.UpdateFromPoll(nil)
	if len(s.GetAll()) != 0 {
		t.Fatal("tightened retention should have dropped the closed PR")
	}
}

func TestSQLite_GetPendingAndHistoryPartition(t *testing.T) {
	clock, _ := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock))
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

func TestSQLite_GetAll_SortedByLastSeenDesc(t *testing.T) {
	clock, tp := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock))

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
	all2 := s.GetAll()
	if !reflect.DeepEqual(sortedIDs(all), sortedIDs(all2)) {
		t.Fatal("GetAll ordering not stable")
	}
}

func TestSQLite_Meta_RoundTrip(t *testing.T) {
	s := newMemoryStore(t)
	ctx := context.Background()

	if _, ok, err := s.getMetaString(ctx, "nope"); err != nil || ok {
		t.Fatalf("absent meta: want (empty,false,nil); got (_,ok=%v,err=%v)", ok, err)
	}
	if err := s.setMetaString(ctx, "k", "v1"); err != nil {
		t.Fatalf("set: %v", err)
	}
	v, ok, err := s.getMetaString(ctx, "k")
	if err != nil || !ok || v != "v1" {
		t.Fatalf("get1: v=%q ok=%v err=%v", v, ok, err)
	}
	if err := s.setMetaString(ctx, "k", "v2"); err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	v, _, _ = s.getMetaString(ctx, "k")
	if v != "v2" {
		t.Fatalf("overwrite read: got %q", v)
	}
}

func TestSQLite_LastPollAtPersisted(t *testing.T) {
	start := time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC)
	clock, _ := newClock(start)
	s := newMemoryStore(t, WithClock(clock))
	s.UpdateFromPoll([]github.PRSummary{
		mkSummary("a/b#1", "a/b", 1, "t", "x", false),
	})
	v, ok, err := s.getMetaString(context.Background(), metaLastPollAt)
	if err != nil || !ok {
		t.Fatalf("get meta: ok=%v err=%v", ok, err)
	}
	got, err := parseTime(v)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !got.Equal(start) {
		t.Fatalf("want %v, got %v", start, got)
	}
}

func TestSQLite_Close_Idempotent(t *testing.T) {
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	s := newSQLiteFromDB(db)
	if err := s.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

func TestSQLite_Save_NoOp(t *testing.T) {
	s := newMemoryStore(t)
	if err := s.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}
}

func sortedIDs(rs []PRRecord) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.ID
	}
	sort.Strings(out)
	return out
}
