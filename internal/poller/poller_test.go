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

func (f *fakeNotifier) Close() error { return nil }

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func freshStore(t *testing.T) store.Store {
	t.Helper()
	return store.New(filepath.Join(t.TempDir(), "state.json"),
		store.WithClock(func() time.Time { return time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC) }),
	)
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
