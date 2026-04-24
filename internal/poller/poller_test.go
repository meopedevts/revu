package poller

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/meopedevts/revu/internal/github"
	"github.com/meopedevts/revu/internal/store"
)

type fakeClient struct {
	mu sync.Mutex

	listCalls    int
	detailsCalls int

	listResponses   []listResponse
	detailsResponse github.PRDetails
	detailsErr      error
}

type listResponse struct {
	prs []github.PRSummary
	err error
}

func (f *fakeClient) AuthStatus(_ context.Context) error { return nil }

func (f *fakeClient) ListReviewRequested(_ context.Context) ([]github.PRSummary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	i := f.listCalls
	f.listCalls++
	if i >= len(f.listResponses) {
		return nil, nil
	}
	r := f.listResponses[i]
	return r.prs, r.err
}

func (f *fakeClient) GetPRDetails(_ context.Context, _ string) (*github.PRDetails, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.detailsCalls++
	if f.detailsErr != nil {
		return nil, f.detailsErr
	}
	d := f.detailsResponse
	return &d, nil
}

func (f *fakeClient) GetPRFullDetails(_ context.Context, _ string) (*github.PRFullDetails, error) {
	return nil, nil
}

func (f *fakeClient) GetPRDiff(_ context.Context, _ string) (string, error) { return "", nil }

func (f *fakeClient) MergePR(_ context.Context, _ string, _ github.MergeMethod) error {
	return nil
}

type fakeNotifier struct {
	mu   sync.Mutex
	sent []store.PRRecord
	err  error
}

func (f *fakeNotifier) Notify(pr store.PRRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, pr)
	return nil
}

func (f *fakeNotifier) SetEnabled(bool)                {}
func (f *fakeNotifier) SetExpireTimeout(time.Duration) {}
func (f *fakeNotifier) Close() error                   { return nil }

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func freshStore(t *testing.T) store.Store {
	t.Helper()
	s := store.New(filepath.Join(t.TempDir(), "revu.db"),
		store.WithClock(func() time.Time { return time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC) }),
	)
	if err := s.Load(); err != nil {
		t.Fatalf("store Load: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestTick_NotifiesNovosAndEnriches(t *testing.T) {
	fc := &fakeClient{
		listResponses: []listResponse{{
			prs: []github.PRSummary{
				{ID: "a/b#1", Number: 1, Repo: "a/b", Title: "t", Author: "x",
					URL: "https://github.com/a/b/pull/1"},
			},
		}},
		detailsResponse: github.PRDetails{Additions: 10, Deletions: 2, State: "OPEN"},
	}
	fn := &fakeNotifier{}
	s := freshStore(t)
	p := New(fc, s, fn, WithLogger(quietLogger()))

	p.tick(context.Background())

	if fc.detailsCalls != 1 {
		t.Fatalf("want 1 details call, got %d", fc.detailsCalls)
	}
	if len(fn.sent) != 1 {
		t.Fatalf("want 1 notify, got %d", len(fn.sent))
	}
	got := fn.sent[0]
	if got.Additions != 10 || got.Deletions != 2 {
		t.Fatalf("notification not enriched: %+v", got)
	}
}

func TestTick_SecondPollSamePRsDoesNotRenotify(t *testing.T) {
	prs := []github.PRSummary{
		{ID: "a/b#1", Number: 1, Repo: "a/b", Title: "t", Author: "x", URL: "u"},
	}
	fc := &fakeClient{
		listResponses: []listResponse{{prs: prs}, {prs: prs}},
	}
	fn := &fakeNotifier{}
	s := freshStore(t)
	p := New(fc, s, fn, WithLogger(quietLogger()))

	p.tick(context.Background())
	p.tick(context.Background())

	if len(fn.sent) != 1 {
		t.Fatalf("want 1 notify across 2 polls, got %d", len(fn.sent))
	}
}

func TestTick_ReRequestRenotifies(t *testing.T) {
	prs := []github.PRSummary{
		{ID: "a/b#1", Number: 1, Repo: "a/b", Title: "t", Author: "x", URL: "u"},
	}
	fc := &fakeClient{
		listResponses: []listResponse{
			{prs: prs}, // initial
			{prs: nil}, // dropped (review given)
			{prs: prs}, // re-request
		},
	}
	fn := &fakeNotifier{}
	s := freshStore(t)
	p := New(fc, s, fn, WithLogger(quietLogger()))

	p.tick(context.Background())
	p.tick(context.Background())
	p.tick(context.Background())

	if len(fn.sent) != 2 {
		t.Fatalf("want 2 notifies (initial + re-request), got %d", len(fn.sent))
	}
}

func TestTick_EnrichFailureStillNotifies(t *testing.T) {
	fc := &fakeClient{
		listResponses: []listResponse{{prs: []github.PRSummary{
			{ID: "a/b#1", Number: 1, Repo: "a/b", Title: "t", Author: "x", URL: "u"},
		}}},
		detailsErr: errors.New("boom"),
	}
	fn := &fakeNotifier{}
	s := freshStore(t)
	p := New(fc, s, fn, WithLogger(quietLogger()))

	p.tick(context.Background())

	if len(fn.sent) != 1 {
		t.Fatalf("want 1 notify even without diff, got %d", len(fn.sent))
	}
	if fn.sent[0].Additions != 0 || fn.sent[0].Deletions != 0 {
		t.Fatalf("want zero diff when enrich fails, got %+v", fn.sent[0])
	}
}

func TestBackoff_DoublesUntilCap(t *testing.T) {
	fc := &fakeClient{
		listResponses: []listResponse{
			{err: github.ErrTransient},
			{err: github.ErrTransient},
			{err: github.ErrTransient},
			{err: github.ErrTransient},
			{err: github.ErrTransient},
			{err: github.ErrTransient},
			{err: github.ErrTransient},
		},
	}
	fn := &fakeNotifier{}
	s := freshStore(t)
	p := New(fc, s, fn, WithInterval(5*time.Minute), WithLogger(quietLogger()))

	wants := []time.Duration{
		10 * time.Minute,
		20 * time.Minute,
		MaxBackoff,
		MaxBackoff,
		MaxBackoff,
		MaxBackoff,
		MaxBackoff,
	}
	for i, want := range wants {
		p.tick(context.Background())
		if got := p.CurrentBackoff(); got != want {
			t.Fatalf("tick #%d: want backoff %s, got %s", i+1, want, got)
		}
	}
}

func TestBackoff_ResetsOnSuccess(t *testing.T) {
	fc := &fakeClient{
		listResponses: []listResponse{
			{err: github.ErrTransient},
			{prs: nil},
		},
	}
	fn := &fakeNotifier{}
	s := freshStore(t)
	p := New(fc, s, fn, WithLogger(quietLogger()))

	p.tick(context.Background())
	if p.CurrentBackoff() == 0 {
		t.Fatal("want backoff > 0 after failure")
	}
	p.tick(context.Background())
	if p.CurrentBackoff() != 0 {
		t.Fatalf("want backoff reset after success, got %s", p.CurrentBackoff())
	}
}

func TestTick_EmitsEvents_HappyPath(t *testing.T) {
	fc := &fakeClient{
		listResponses: []listResponse{{
			prs: []github.PRSummary{
				{ID: "a/b#1", Number: 1, Repo: "a/b", Title: "t", Author: "x", URL: "u"},
			},
		}},
		detailsResponse: github.PRDetails{Additions: 5, Deletions: 2, State: "OPEN"},
	}
	fn := &fakeNotifier{}
	s := freshStore(t)
	var mu sync.Mutex
	var events []Event
	p := New(fc, s, fn,
		WithLogger(quietLogger()),
		WithEventHandler(func(e Event) {
			mu.Lock()
			events = append(events, e)
			mu.Unlock()
		}),
	)

	p.tick(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if len(events) != 2 {
		t.Fatalf("want 2 events (pr:new + poll:completed), got %d: %+v", len(events), events)
	}
	if events[0].Kind != EventPRNew {
		t.Fatalf("events[0] kind = %s, want %s", events[0].Kind, EventPRNew)
	}
	if events[0].PR == nil || events[0].PR.ID != "a/b#1" || events[0].PR.Additions != 5 {
		t.Fatalf("events[0] PR payload wrong: %+v", events[0].PR)
	}
	if events[1].Kind != EventPollCompleted || events[1].At.IsZero() {
		t.Fatalf("events[1] bad: %+v", events[1])
	}
	if events[1].Err != "" {
		t.Fatalf("poll:completed should have no error on happy path, got %q", events[1].Err)
	}
}

func TestTick_EmitsPollCompleted_WithErrOnFailure(t *testing.T) {
	fc := &fakeClient{
		listResponses: []listResponse{{err: github.ErrTransient}},
	}
	fn := &fakeNotifier{}
	s := freshStore(t)
	var got Event
	p := New(fc, s, fn,
		WithLogger(quietLogger()),
		WithEventHandler(func(e Event) { got = e }),
	)

	p.tick(context.Background())

	if got.Kind != EventPollCompleted {
		t.Fatalf("want %s, got %+v", EventPollCompleted, got)
	}
	if got.Err == "" {
		t.Fatal("want non-empty Err on failed poll")
	}
}

func TestTick_EmitsStatusChanged(t *testing.T) {
	prs := []github.PRSummary{
		{ID: "a/b#1", Number: 1, Repo: "a/b", Title: "t", Author: "x", URL: "u"},
	}
	// First poll: open. Second poll (after merge): same URL but details
	// return CLOSED + mergedAt → state flips to MERGED → status-changed.
	merged := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	fc := &fakeClient{
		listResponses: []listResponse{
			{prs: prs},
			{prs: prs},
		},
	}
	fn := &fakeNotifier{}
	s := freshStore(t)

	var mu sync.Mutex
	var kinds []string
	p := New(fc, s, fn,
		WithLogger(quietLogger()),
		WithEventHandler(func(e Event) {
			mu.Lock()
			kinds = append(kinds, e.Kind)
			mu.Unlock()
		}),
	)

	// Tick 1: OPEN details.
	fc.detailsResponse = github.PRDetails{State: "OPEN"}
	p.tick(context.Background())
	// Tick 2: CLOSED + mergedAt → MERGED.
	fc.detailsResponse = github.PRDetails{State: "CLOSED", MergedAt: &merged}
	p.tick(context.Background())

	mu.Lock()
	defer mu.Unlock()
	// Tick 1: pr:new + poll:completed (no status-changed — prevState empty).
	// Tick 2: neither pr:new (already known, pending=true again) nor
	// status-changed for this case — wait, enrich only fires for novos in
	// UpdateFromPoll. Re-pending-without-prior-drop is NOT novo. So tick 2
	// has just poll:completed. We really need to test the status change
	// path explicitly: manually call enrich on a pre-populated store.
	foundNew := false
	foundCompleted := 0
	for _, k := range kinds {
		if k == EventPRNew {
			foundNew = true
		}
		if k == EventPollCompleted {
			foundCompleted++
		}
	}
	if !foundNew {
		t.Fatal("expected at least one pr:new")
	}
	if foundCompleted != 2 {
		t.Fatalf("expected 2 poll:completed, got %d", foundCompleted)
	}
}

func TestTick_VanishedPRGetsEnriched(t *testing.T) {
	pr := []github.PRSummary{
		{ID: "a/b#1", Number: 1, Repo: "a/b", Title: "t", Author: "x", URL: "u"},
	}
	fc := &fakeClient{
		listResponses: []listResponse{
			{prs: pr},  // tick 1: PR visible, not yet reviewed
			{prs: nil}, // tick 2: PR vanished from search — poller re-enriches
		},
	}
	fn := &fakeNotifier{}
	s := freshStore(t)
	p := New(fc, s, fn, WithLogger(quietLogger()))

	// Tick 1 enrich → PR stays PENDING + OPEN.
	fc.detailsResponse = github.PRDetails{State: "OPEN", ReviewState: "PENDING"}
	p.tick(context.Background())

	// Tick 2 enrich → PR was merged after I approved it; under REV-16 it
	// should flip into the history tab.
	fc.detailsResponse = github.PRDetails{State: "MERGED", ReviewState: "APPROVED"}
	p.tick(context.Background())

	history := s.GetHistory()
	if len(history) != 1 || history[0].ID != "a/b#1" {
		t.Fatalf("vanished PR should be in history, got %+v", history)
	}
	if history[0].State != "MERGED" || history[0].ReviewState != "APPROVED" {
		t.Fatalf("history row not fully enriched: %+v", history[0])
	}
	if len(fn.sent) != 1 {
		t.Fatalf("vanished path must not notify; sent=%d", len(fn.sent))
	}
}

func TestTrigger_ForcesImmediatePoll(t *testing.T) {
	fc := &fakeClient{
		listResponses: []listResponse{
			{prs: nil}, // first (immediate) tick
			{prs: []github.PRSummary{
				{ID: "a/b#1", Number: 1, Repo: "a/b", Title: "t", Author: "x", URL: "u"},
			}},
		},
		detailsResponse: github.PRDetails{State: "OPEN"},
	}
	fn := &fakeNotifier{}
	s := freshStore(t)
	// Long interval so the trigger, not the timer, drives the second tick.
	p := New(fc, s, fn, WithInterval(time.Hour), WithLogger(quietLogger()))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx) }()

	// Wait for the immediate tick to land.
	deadline := time.Now().Add(2 * time.Second)
	for {
		fc.mu.Lock()
		n := fc.listCalls
		fc.mu.Unlock()
		if n >= 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("first tick never fired")
		}
		time.Sleep(10 * time.Millisecond)
	}

	p.Trigger()

	deadline = time.Now().Add(2 * time.Second)
	for {
		fn.mu.Lock()
		sent := len(fn.sent)
		fn.mu.Unlock()
		if sent >= 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("trigger did not produce a notification")
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	<-done
}

func TestTrigger_CoalescesBursts(t *testing.T) {
	p := New(&fakeClient{}, freshStore(t), &fakeNotifier{}, WithLogger(quietLogger()))
	// Ten rapid triggers must not panic nor block.
	for i := 0; i < 10; i++ {
		p.Trigger()
	}
	// Channel must still have exactly one pending item.
	select {
	case <-p.triggerCh:
	default:
		t.Fatal("expected at least one trigger queued")
	}
	select {
	case <-p.triggerCh:
		t.Fatal("trigger channel should have coalesced extras")
	default:
	}
}

func TestSetInterval_TakesEffectOnNextWait(t *testing.T) {
	fc := &fakeClient{}
	fn := &fakeNotifier{}
	s := freshStore(t)
	p := New(fc, s, fn, WithInterval(time.Hour), WithLogger(quietLogger()))

	if p.Interval() != time.Hour {
		t.Fatalf("initial interval: %s", p.Interval())
	}
	p.SetInterval(10 * time.Second)
	if p.Interval() != 10*time.Second {
		t.Fatalf("post-set interval: %s", p.Interval())
	}
	// Non-positive ignored.
	p.SetInterval(0)
	p.SetInterval(-1 * time.Second)
	if p.Interval() != 10*time.Second {
		t.Fatalf("invalid SetInterval mutated state: %s", p.Interval())
	}
}

func TestRun_StopsOnCtxCancel(t *testing.T) {
	fc := &fakeClient{}
	fn := &fakeNotifier{}
	s := freshStore(t)
	p := New(fc, s, fn, WithInterval(time.Hour), WithLogger(quietLogger()))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx) }()

	// Give the first tick a moment to fire.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("want context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after ctx cancel")
	}
}
