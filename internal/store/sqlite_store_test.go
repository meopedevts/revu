package store

import (
	"context"
	"errors"
	"path/filepath"
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
	novos, _ := s.UpdateFromPoll(context.Background(), prs)
	if len(novos) != 2 {
		t.Fatalf("first poll: want 2 novos, got %d", len(novos))
	}

	*tp = tp.Add(5 * time.Minute)
	novos, _ = s.UpdateFromPoll(context.Background(), prs)
	if len(novos) != 0 {
		t.Fatalf("second poll with same PRs: want 0 novos, got %d", len(novos))
	}
	if len(s.GetPending(context.Background())) != 2 {
		t.Fatalf("want 2 pending, got %d", len(s.GetPending(context.Background())))
	}
}

func TestSQLite_UpdateFromPoll_ReRequestDetected(t *testing.T) {
	clock, tp := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock))

	// Re-request detection now hinges on the IN-clause path: a PR drops
	// because a non-empty subsequent poll omits it (review submitted) and
	// then comes back. Empty polls are no-ops since REV-43 to avoid
	// transient gh-search inconsistency mass-clearing review_pending.
	pr1 := mkSummary("octocat/hello#1", "octocat/hello", 1, "feat: foo", "alice", false)
	pr2 := mkSummary("octocat/hello#2", "octocat/hello", 2, "feat: bar", "bob", false)
	if novos, _ := s.UpdateFromPoll(context.Background(), []github.PRSummary{pr1, pr2}); len(novos) != 2 {
		t.Fatalf("initial insert: want 2 novos, got %d", len(novos))
	}
	if err := s.RefreshPRStatus(context.Background(), pr1.ID, github.PRDetails{State: "OPEN", ReviewState: "APPROVED"}); err != nil {
		t.Fatalf("refresh approved: %v", err)
	}

	*tp = tp.Add(5 * time.Minute)
	novos, vanished := s.UpdateFromPoll(context.Background(), []github.PRSummary{pr2})
	if len(novos) != 0 {
		t.Fatalf("non-empty poll without pr1: want 0 novos, got %d", len(novos))
	}
	if len(vanished) != 1 || vanished[0].ID != pr1.ID {
		t.Fatalf("want pr1 in vanished, got %+v", vanished)
	}
	got, ok := s.GetByID(context.Background(), pr1.ID)
	if !ok || got.ReviewPending {
		t.Fatalf("want pr1 with pending=false after drop, got ok=%v rec=%+v", ok, got)
	}

	*tp = tp.Add(5 * time.Minute)
	novos, _ = s.UpdateFromPoll(context.Background(), []github.PRSummary{pr1, pr2})
	if len(novos) != 1 {
		t.Fatalf("re-request: want 1 novo, got %d", len(novos))
	}
	if novos[0].ID != pr1.ID {
		t.Fatalf("wrong id in novos: %s", novos[0].ID)
	}
	if novos[0].ReviewState != "PENDING" {
		t.Fatalf("re-request should reset review_state to PENDING, got %q", novos[0].ReviewState)
	}
	pending := s.GetPending(context.Background())
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending records after re-request, got %d", len(pending))
	}
}

func TestSQLite_UpdateFromPoll_UpdatesMutableFields(t *testing.T) {
	clock, _ := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock))

	pr := mkSummary("octocat/hello#1", "octocat/hello", 1, "feat: foo", "alice", false)
	s.UpdateFromPoll(context.Background(), []github.PRSummary{pr})

	pr.Title = "feat: foo (edited)"
	pr.IsDraft = true
	s.UpdateFromPoll(context.Background(), []github.PRSummary{pr})

	got := s.GetPending(context.Background())[0]
	if got.Title != "feat: foo (edited)" || !got.IsDraft {
		t.Fatalf("mutable fields not updated: %+v", got)
	}
}

func TestSQLite_RefreshPRStatus(t *testing.T) {
	clock, _ := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock))

	pr := mkSummary("octocat/hello#1", "octocat/hello", 1, "feat: foo", "alice", false)
	s.UpdateFromPoll(context.Background(), []github.PRSummary{pr})

	merged := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	err := s.RefreshPRStatus(context.Background(), pr.ID, github.PRDetails{
		Additions: 10, Deletions: 5, State: "CLOSED", MergedAt: &merged, IsDraft: false,
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	got := s.GetAll(context.Background())[0]
	if got.Additions != 10 || got.Deletions != 5 || got.State != "MERGED" {
		t.Fatalf("status not applied: %+v", got)
	}

	if err := s.RefreshPRStatus(context.Background(), "nope", github.PRDetails{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestSQLite_Retention_DropsOldNonOpen_InPoll(t *testing.T) {
	clock, tp := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock), WithRetention(30))

	prOpen := mkSummary("a/b#1", "a/b", 1, "open", "x", false)
	prClosed := mkSummary("a/b#2", "a/b", 2, "closed", "x", false)
	s.UpdateFromPoll(context.Background(), []github.PRSummary{prOpen, prClosed})
	if err := s.RefreshPRStatus(context.Background(), prClosed.ID, github.PRDetails{State: "CLOSED"}); err != nil {
		t.Fatal(err)
	}

	// 40 days pass; prClosed disappears from the search. Retention runs on
	// the next UpdateFromPoll tx — prClosed gets purged, prOpen stays.
	*tp = tp.Add(40 * 24 * time.Hour)
	s.UpdateFromPoll(context.Background(), []github.PRSummary{prOpen})

	all := s.GetAll(context.Background())
	if len(all) != 1 || all[0].ID != prOpen.ID {
		t.Fatalf("want only open pr, got %+v", all)
	}
}

func TestSQLite_Retention_DisabledWhenZero(t *testing.T) {
	clock, tp := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock), WithRetention(0))

	pr := mkSummary("a/b#1", "a/b", 1, "t", "x", false)
	s.UpdateFromPoll(context.Background(), []github.PRSummary{pr})
	if err := s.RefreshPRStatus(context.Background(), pr.ID, github.PRDetails{State: "CLOSED"}); err != nil {
		t.Fatal(err)
	}
	*tp = tp.Add(365 * 24 * time.Hour)
	s.UpdateFromPoll(context.Background(), nil)
	if len(s.GetAll(context.Background())) != 1 {
		t.Fatal("retention=0 should keep everything")
	}
}

func TestSQLite_Retention_RespectsRuntimeChange(t *testing.T) {
	clock, tp := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock), WithRetention(365))

	pr := mkSummary("a/b#1", "a/b", 1, "t", "x", false)
	s.UpdateFromPoll(context.Background(), []github.PRSummary{pr})
	if err := s.RefreshPRStatus(context.Background(), pr.ID, github.PRDetails{State: "CLOSED"}); err != nil {
		t.Fatal(err)
	}
	*tp = tp.Add(30 * 24 * time.Hour)

	s.SetRetentionDays(7)
	s.UpdateFromPoll(context.Background(), nil)
	if len(s.GetAll(context.Background())) != 0 {
		t.Fatal("tightened retention should have dropped the closed PR")
	}
}

func TestSQLite_GetPendingAndHistoryPartition(t *testing.T) {
	clock, _ := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock))
	prA := mkSummary("a/b#1", "a/b", 1, "A", "x", false)
	prB := mkSummary("a/b#2", "a/b", 2, "B", "x", false)
	s.UpdateFromPoll(context.Background(), []github.PRSummary{prA, prB})
	// B is reviewed (approved) and then the author merges it. Under REV-16
	// only the combination "state != OPEN" AND "review submitted" lands in
	// history; apply both so the test actually exercises the history tab.
	if err := s.RefreshPRStatus(context.Background(), prB.ID, github.PRDetails{State: "MERGED", ReviewState: "APPROVED"}); err != nil {
		t.Fatalf("refresh prB: %v", err)
	}
	s.UpdateFromPoll(context.Background(), []github.PRSummary{prA})

	pending := s.GetPending(context.Background())
	history := s.GetHistory(context.Background())
	if len(pending) != 1 || pending[0].ID != prA.ID {
		t.Fatalf("pending mismatch: %+v", pending)
	}
	if len(history) != 1 || history[0].ID != prB.ID {
		t.Fatalf("history mismatch: %+v", history)
	}
}

func TestSQLite_HistoryRule_REV16(t *testing.T) {
	clock, _ := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock))

	// REV-16 refined: history is state IN (MERGED, CLOSED) regardless of
	// review_state. The merged-before-I-reviewed case (co-reviewer merged
	// first) must land in history so the user is not stuck acknowledging
	// something they can no longer act on.
	prOpenApproved := mkSummary("a/b#1", "a/b", 1, "open-approved", "x", false)
	prMergedPending := mkSummary("a/b#2", "a/b", 2, "merged-pending", "x", false)
	prMergedApproved := mkSummary("a/b#3", "a/b", 3, "merged-approved", "x", false)
	prClosedCommented := mkSummary("a/b#4", "a/b", 4, "closed-commented", "x", false)
	s.UpdateFromPoll(context.Background(), []github.PRSummary{
		prOpenApproved, prMergedPending, prMergedApproved, prClosedCommented,
	})

	mustRefresh(t, s, prOpenApproved.ID, "OPEN", "APPROVED")
	mustRefresh(t, s, prMergedPending.ID, "MERGED", "PENDING")
	mustRefresh(t, s, prMergedApproved.ID, "MERGED", "APPROVED")
	mustRefresh(t, s, prClosedCommented.ID, "CLOSED", "COMMENTED")

	pending := sortedIDs(s.GetPending(context.Background()))
	history := sortedIDs(s.GetHistory(context.Background()))
	wantPending := sortedIDs([]PRRecord{{ID: prOpenApproved.ID}})
	wantHistory := sortedIDs([]PRRecord{
		{ID: prMergedPending.ID},
		{ID: prMergedApproved.ID},
		{ID: prClosedCommented.ID},
	})
	if !reflect.DeepEqual(pending, wantPending) {
		t.Fatalf("pending mismatch: got %v, want %v", pending, wantPending)
	}
	if !reflect.DeepEqual(history, wantHistory) {
		t.Fatalf("history mismatch: got %v, want %v", history, wantHistory)
	}
}

func mustRefresh(t *testing.T, s *sqliteStore, id, state, review string) {
	t.Helper()
	if err := s.RefreshPRStatus(context.Background(), id, github.PRDetails{State: state, ReviewState: review}); err != nil {
		t.Fatalf("refresh %s: %v", id, err)
	}
}

func TestSQLite_GetAll_SortedByLastSeenDesc(t *testing.T) {
	clock, tp := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock))

	s.UpdateFromPoll(context.Background(), []github.PRSummary{mkSummary("a/b#1", "a/b", 1, "A", "x", false)})
	*tp = tp.Add(time.Minute)
	s.UpdateFromPoll(context.Background(), []github.PRSummary{
		mkSummary("a/b#1", "a/b", 1, "A", "x", false),
		mkSummary("a/b#2", "a/b", 2, "B", "x", false),
	})
	all := s.GetAll(context.Background())
	if len(all) != 2 {
		t.Fatalf("want 2, got %d", len(all))
	}
	all2 := s.GetAll(context.Background())
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
	s.UpdateFromPoll(context.Background(), []github.PRSummary{
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
	if err := s.Close(context.Background()); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := s.Close(context.Background()); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

func TestSQLite_Save_NoOp(t *testing.T) {
	s := newMemoryStore(t)
	if err := s.Save(context.Background()); err != nil {
		t.Fatalf("save: %v", err)
	}
}

// TestSQLite_Load_Idempotent cobre o branch `if s.db != nil` em Load.
// Segunda chamada de Load() retorna nil sem reabrir/migrar o DB.
func TestSQLite_Load_Idempotent(t *testing.T) {
	st := New(":memory:").(*sqliteStore)
	if err := st.Load(context.Background()); err != nil {
		t.Fatalf("first Load: %v", err)
	}
	t.Cleanup(func() { _ = st.Close(context.Background()) })

	first := st.db
	if first == nil {
		t.Fatal("first Load did not set db handle")
	}

	if err := st.Load(context.Background()); err != nil {
		t.Fatalf("second Load: %v", err)
	}
	if st.db != first {
		t.Fatal("second Load reopened the db handle (must be no-op)")
	}
}

// TestSQLite_GetByID_NotFound cobre o branch `if err != nil || !ok` em
// GetByID. ID inexistente → (PRRecord{}, false).
func TestSQLite_GetByID_NotFound(t *testing.T) {
	s := newMemoryStore(t)
	rec, ok := s.GetByID(context.Background(), "does-not-exist")
	if ok {
		t.Fatal("missing id must return ok=false")
	}
	if rec != (PRRecord{}) {
		t.Fatalf("missing id must return zero record, got %+v", rec)
	}
}

func TestSQLite_ClearHistory_DropsEveryHistoryRow(t *testing.T) {
	clock, tp := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock))

	prStillPending := mkSummary("a/b#1", "a/b", 1, "pending", "x", false)
	prMerged := mkSummary("a/b#2", "a/b", 2, "merged", "x", false)
	prClosed := mkSummary("a/b#3", "a/b", 3, "closed", "x", false)
	prDropped := mkSummary("a/b#4", "a/b", 4, "dropped-from-search", "x", false)
	s.UpdateFromPoll(context.Background(), []github.PRSummary{prStillPending, prMerged, prClosed, prDropped})
	// Finalize two PRs on GitHub — under REV-16 the state alone decides the
	// tab, so both end up in history regardless of review state.
	if err := s.RefreshPRStatus(context.Background(), prMerged.ID, github.PRDetails{State: "MERGED", ReviewState: "APPROVED"}); err != nil {
		t.Fatalf("refresh merged: %v", err)
	}
	if err := s.RefreshPRStatus(
		context.Background(),
		prClosed.ID,
		github.PRDetails{State: "CLOSED", ReviewState: "COMMENTED"},
	); err != nil {
		t.Fatalf("refresh closed: %v", err)
	}

	// Next poll keeps only prStillPending — the other three drop from the
	// search and flip to review_pending=0. prDropped stays state='OPEN'
	// (never enriched) so still belongs to the pending tab; merged/closed
	// rows land in history.
	*tp = tp.Add(1 * time.Minute)
	s.UpdateFromPoll(context.Background(), []github.PRSummary{prStillPending})

	pending := sortedIDs(s.GetPending(context.Background()))
	wantPending := sortedIDs([]PRRecord{{ID: prStillPending.ID}, {ID: prDropped.ID}})
	if !reflect.DeepEqual(pending, wantPending) {
		t.Fatalf("want %v in pending, got %v", wantPending, pending)
	}
	if history := s.GetHistory(context.Background()); len(history) != 2 {
		t.Fatalf("want 2 rows in history, got %d: %+v", len(history), history)
	}

	// ClearHistory now wipes every merged/closed row; OPEN survivors stay
	// regardless of their review_state.
	n, err := s.ClearHistory(context.Background())
	if err != nil {
		t.Fatalf("clear history: %v", err)
	}
	if n != 2 {
		t.Fatalf("want 2 finalized rows cleared, got %d", n)
	}

	all := s.GetAll(context.Background())
	ids := sortedIDs(all)
	want := sortedIDs([]PRRecord{{ID: prStillPending.ID}, {ID: prDropped.ID}})
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("want [prStillPending, prDropped] left, got %+v", ids)
	}

	// Idempotent: nothing else to finalize.
	n, err = s.ClearHistory(context.Background())
	if err != nil {
		t.Fatalf("clear history second: %v", err)
	}
	if n != 0 {
		t.Fatalf("second clear should remove 0 rows, got %d", n)
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

// REV-43 — empty polls must not flip review_pending. Transient gh-search
// `[]` results were causing burst notifications on the next non-empty poll.
func TestSQLite_UpdateFromPoll_EmptyPollIsNoOp(t *testing.T) {
	clock, tp := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock))

	prs := []github.PRSummary{
		mkSummary("a/b#1", "a/b", 1, "t1", "x", false),
		mkSummary("a/b#2", "a/b", 2, "t2", "y", false),
	}
	if novos, _ := s.UpdateFromPoll(context.Background(), prs); len(novos) != 2 {
		t.Fatalf("seed: want 2 novos, got %d", len(novos))
	}

	*tp = tp.Add(5 * time.Minute)
	novos, vanished := s.UpdateFromPoll(context.Background(), nil)
	if len(novos) != 0 {
		t.Fatalf("empty poll: want 0 novos, got %d", len(novos))
	}
	if len(vanished) != 0 {
		t.Fatalf("empty poll: want 0 vanished (no-op contract), got %d", len(vanished))
	}
	pending := s.GetPending(context.Background())
	if len(pending) != 2 {
		t.Fatalf("empty poll must not flip review_pending: pending=%d", len(pending))
	}
	for _, rec := range pending {
		if !rec.ReviewPending {
			t.Fatalf("review_pending flipped on empty poll: %+v", rec)
		}
	}
}

func TestSQLite_MarkNotified_PersistsTimestamp(t *testing.T) {
	clock, _ := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock))

	pr := mkSummary("a/b#1", "a/b", 1, "t", "x", false)
	if novos, _ := s.UpdateFromPoll(context.Background(), []github.PRSummary{pr}); len(novos) != 1 {
		t.Fatalf("seed: want 1 novo")
	}

	when := time.Date(2026, 4, 23, 11, 30, 0, 0, time.UTC)
	if err := s.MarkNotified(context.Background(), pr.ID, when); err != nil {
		t.Fatalf("MarkNotified: %v", err)
	}

	got, ok := s.GetByID(context.Background(), pr.ID)
	if !ok {
		t.Fatal("rec missing after MarkNotified")
	}
	if got.LastNotifiedAt == nil {
		t.Fatal("LastNotifiedAt still nil after MarkNotified")
	}
	if !got.LastNotifiedAt.Equal(when) {
		t.Fatalf("LastNotifiedAt mismatch: want %s got %s", when, got.LastNotifiedAt)
	}
}

func TestSQLite_MarkNotified_UnknownIDReturnsErrNotFound(t *testing.T) {
	clock, _ := newClock(time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC))
	s := newMemoryStore(t, WithClock(clock))

	err := s.MarkNotified(context.Background(), "missing/repo#999", time.Now())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

// REV-43 — Load backfills last_notified_at = first_seen_at on the first
// run after upgrade so legacy NULL rows don't burst-notify when the
// throttle activates. The meta key gates idempotency.
func TestSQLite_Load_BackfillNotifiedAt_OneShotIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "revu.db")
	now := time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC)

	s := New(dbPath, WithClock(func() time.Time { return now }))
	if err := s.Load(context.Background()); err != nil {
		t.Fatalf("first Load: %v", err)
	}

	// Insert a legacy row with last_notified_at NULL via the underlying handle.
	firstSeen := time.Date(2026, 4, 22, 9, 0, 0, 0, time.UTC)
	if _, err := s.DB().ExecContext(context.Background(),
		`INSERT INTO prs (id, number, repo, title, author, url, state, is_draft,
			additions, deletions, review_pending, review_state, first_seen_at, last_seen_at,
			last_notified_at, profile_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, '')`,
		"a/b#1", 1, "a/b", "t", "x", "u", "OPEN", 0,
		0, 0, 1, "PENDING", formatTime(firstSeen), formatTime(firstSeen),
	); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	// Manually nuke the meta key so the next Load re-runs the backfill.
	if _, err := s.DB().ExecContext(context.Background(),
		`DELETE FROM meta WHERE key = ?`, metaNotifyBackfillDone); err != nil {
		t.Fatalf("clear meta: %v", err)
	}
	if err := s.Close(context.Background()); err != nil {
		t.Fatalf("close: %v", err)
	}

	s2 := New(dbPath, WithClock(func() time.Time { return now }))
	if err := s2.Load(context.Background()); err != nil {
		t.Fatalf("second Load: %v", err)
	}
	t.Cleanup(func() { _ = s2.Close(context.Background()) })

	got, ok := s2.GetByID(context.Background(), "a/b#1")
	if !ok {
		t.Fatal("seeded row missing after second Load")
	}
	if got.LastNotifiedAt == nil {
		t.Fatal("backfill did not populate LastNotifiedAt")
	}
	if !got.LastNotifiedAt.Equal(firstSeen) {
		t.Fatalf("want LastNotifiedAt == FirstSeenAt (%s), got %s", firstSeen, got.LastNotifiedAt)
	}

	// Idempotency: rewrite to a sentinel and Load again — backfill must NOT
	// run twice (meta key gates it).
	sentinel := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := s2.DB().ExecContext(context.Background(),
		`UPDATE prs SET last_notified_at = ? WHERE id = ?`, formatTime(sentinel), "a/b#1"); err != nil {
		t.Fatalf("rewrite sentinel: %v", err)
	}
	if err := s2.Close(context.Background()); err != nil {
		t.Fatalf("close: %v", err)
	}
	s3 := New(dbPath, WithClock(func() time.Time { return now }))
	if err := s3.Load(context.Background()); err != nil {
		t.Fatalf("third Load: %v", err)
	}
	t.Cleanup(func() { _ = s3.Close(context.Background()) })
	got3, _ := s3.GetByID(context.Background(), "a/b#1")
	if got3.LastNotifiedAt == nil || !got3.LastNotifiedAt.Equal(sentinel) {
		t.Fatalf("backfill ran twice: LastNotifiedAt=%v want sentinel %s", got3.LastNotifiedAt, sentinel)
	}
}

// TestSQLite_Acknowledge_RoundTrip cobre o ciclo set → get do ack de tray
// (REV-51). Antes do primeiro Acknowledge, AcknowledgedAt deve devolver
// ok=false; depois, devolver o instante exato (preservando UTC + nanos).
func TestSQLite_Acknowledge_RoundTrip(t *testing.T) {
	s := newMemoryStore(t)
	ctx := context.Background()

	if _, ok, err := s.AcknowledgedAt(ctx); err != nil || ok {
		t.Fatalf("pre-ack: want (zero, false, nil), got ok=%v err=%v", ok, err)
	}

	when := time.Date(2026, 5, 2, 14, 56, 17, 123_000_000, time.UTC)
	if err := s.Acknowledge(ctx, when); err != nil {
		t.Fatalf("Acknowledge: %v", err)
	}

	got, ok, err := s.AcknowledgedAt(ctx)
	if err != nil {
		t.Fatalf("AcknowledgedAt: %v", err)
	}
	if !ok {
		t.Fatal("post-ack: ok=false, want true")
	}
	if !got.Equal(when) {
		t.Fatalf("ack timestamp mismatch: got %s, want %s", got, when)
	}
}

// TestSQLite_Acknowledge_Overwrites verifica que chamadas sucessivas
// substituem o ack anterior (semântica "última visualização vence").
func TestSQLite_Acknowledge_Overwrites(t *testing.T) {
	s := newMemoryStore(t)
	ctx := context.Background()

	first := time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)
	second := first.Add(2 * time.Hour)

	if err := s.Acknowledge(ctx, first); err != nil {
		t.Fatalf("first ack: %v", err)
	}
	if err := s.Acknowledge(ctx, second); err != nil {
		t.Fatalf("second ack: %v", err)
	}

	got, ok, err := s.AcknowledgedAt(ctx)
	if err != nil || !ok {
		t.Fatalf("AcknowledgedAt: ok=%v err=%v", ok, err)
	}
	if !got.Equal(second) {
		t.Fatalf("want last ack (%s), got %s", second, got)
	}
}

// TestSQLite_Acknowledge_NormalizesToUTC garante que um ack passado em
// fuso local volta normalizado em UTC — store grava sempre RFC3339Nano UTC,
// e o caller (tray) compara com LastSeenAt que também é UTC.
func TestSQLite_Acknowledge_NormalizesToUTC(t *testing.T) {
	s := newMemoryStore(t)
	ctx := context.Background()

	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Skipf("tz not available: %v", err)
	}
	local := time.Date(2026, 5, 2, 11, 0, 0, 0, loc)

	if err := s.Acknowledge(ctx, local); err != nil {
		t.Fatalf("Acknowledge: %v", err)
	}
	got, ok, err := s.AcknowledgedAt(ctx)
	if err != nil || !ok {
		t.Fatalf("AcknowledgedAt: ok=%v err=%v", ok, err)
	}
	if got.Location() != time.UTC {
		t.Fatalf("want UTC, got %s", got.Location())
	}
	if !got.Equal(local) {
		t.Fatalf("instant changed: got %s want %s", got, local)
	}
}
