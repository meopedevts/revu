package poller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/meopedevts/revu/internal/github"
	"github.com/meopedevts/revu/internal/notifier"
	"github.com/meopedevts/revu/internal/store"
)

const (
	// DefaultInterval matches SPEC §5.1 (300s).
	DefaultInterval = 5 * time.Minute
	// MaxBackoff caps exponential backoff per SPEC §9.
	MaxBackoff = 30 * time.Minute
)

// ActiveProfileProvider is the subset of profiles.Service the poller needs.
// Kept as a local interface so the poller package does not import profiles.
type ActiveProfileProvider interface {
	ActiveProfileID(ctx context.Context) (string, error)
}

// Poller ticks the gh search, enriches new PRs, notifies, and persists.
// Life-cycle is owned by Run; callers cancel the ctx to stop.
type Poller struct {
	client   github.Client
	store    store.Store
	notifier notifier.Notifier
	log      *slog.Logger
	profiles ActiveProfileProvider

	mu           sync.RWMutex
	interval     time.Duration
	backoff      time.Duration // current backoff (0 when healthy)
	triggerCh    chan struct{} // capacity 1; Trigger coalesces bursts
	eventHandler EventHandler  // nil = no emission
}

// Option customizes the poller.
type Option func(*Poller)

// WithInterval overrides DefaultInterval. Non-positive values fall back to
// the default.
func WithInterval(d time.Duration) Option {
	return func(p *Poller) {
		if d > 0 {
			p.interval = d
		}
	}
}

// WithLogger injects a logger. Defaults to [slog.Default].
func WithLogger(l *slog.Logger) Option {
	return func(p *Poller) {
		if l != nil {
			p.log = l
		}
	}
}

// WithActiveProfile injects the profile provider. When set, every tick tags
// inserts with the current active profile id via store.SetActiveProfileID.
// Passing nil disables the feature.
func WithActiveProfile(pp ActiveProfileProvider) Option {
	return func(p *Poller) {
		p.profiles = pp
	}
}

// New builds a Poller with the given dependencies.
func New(c github.Client, s store.Store, n notifier.Notifier, opts ...Option) *Poller {
	p := &Poller{
		client:    c,
		store:     s,
		notifier:  n,
		log:       slog.Default(),
		interval:  DefaultInterval,
		triggerCh: make(chan struct{}, 1),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Trigger requests an immediate poll. Safe to call from any goroutine; if a
// trigger is already queued, extra calls are coalesced (no unbounded queue
// even under rapid clicking of "Atualizar agora").
func (p *Poller) Trigger() {
	select {
	case p.triggerCh <- struct{}{}:
	default:
	}
}

// Run drives the ticker until ctx is canceled. The first tick fires
// immediately (SPEC §8.2 step 7). On error it applies exponential backoff,
// doubling up to MaxBackoff; on any success it resets to the configured
// interval. Also honors Trigger for out-of-schedule polls and picks up
// interval changes landed via SetInterval on the next timer scheduling.
func (p *Poller) Run(ctx context.Context) error {
	p.tick(ctx)
	for {
		timer := time.NewTimer(p.waitDuration())
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			p.tick(ctx)
		case <-p.triggerCh:
			timer.Stop()
			p.tick(ctx)
		}
	}
}

// waitDuration returns backoff (if active) or interval under lock so
// concurrent SetInterval calls are race-free.
func (p *Poller) waitDuration() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.backoff > 0 {
		return p.backoff
	}
	return p.interval
}

// SetInterval replaces the scheduled polling interval. Non-positive values
// are ignored. The change takes effect on the next wait (current timer runs
// to completion) — this matches the "próximo poll respeita novo intervalo"
// expectation from SPEC §7.
func (p *Poller) SetInterval(d time.Duration) {
	if d <= 0 {
		return
	}
	p.mu.Lock()
	p.interval = d
	p.mu.Unlock()
}

// tick executes one poll cycle. Errors are logged and applied to backoff but
// never propagated out of Run — we keep polling through transient failures.
// Every call ends with a poll:completed emission so the frontend can refresh
// a "last updated" indicator even on empty polls.
func (p *Poller) tick(ctx context.Context) {
	if p.profiles != nil {
		if id, err := p.profiles.ActiveProfileID(ctx); err == nil && id != "" {
			p.store.SetActiveProfileID(id)
		} else if err != nil {
			p.log.WarnContext(ctx, "resolve active profile", "err", err)
		}
	}
	summaries, err := p.client.ListReviewRequested(ctx)
	if err != nil {
		p.handlePollError(err)
		p.emit(Event{Kind: EventPollCompleted, Err: err.Error()})
		return
	}
	p.resetBackoff()

	novos, vanished := p.store.UpdateFromPoll(summaries)
	for i := range novos {
		rec := novos[i]
		enriched := p.enrich(ctx, rec)
		if err := p.notifier.Notify(enriched); err != nil {
			p.log.WarnContext(ctx, "notify failed", "pr", rec.ID, "err", err)
		}
		prCopy := enriched
		p.emit(Event{Kind: EventPRNew, PR: &prCopy})
	}
	// REV-16: PRs that dropped from the search need a fresh enrich so PR state
	// + review_state converge with GitHub. These are not new work — no notify,
	// no pr:new event. The enrich itself emits pr:status-changed when the
	// stored state flips (e.g. OPEN → MERGED).
	if len(vanished) > 0 {
		p.log.DebugContext(ctx, "enriching vanished PRs", "count", len(vanished))
	}
	for i := range vanished {
		p.enrich(ctx, vanished[i])
	}
	if err := p.store.Save(); err != nil {
		p.log.WarnContext(ctx, "save store", "err", err)
	}
	p.emit(Event{Kind: EventPollCompleted})
}

// enrich fetches additions/deletions for a new PR. On failure it returns
// the record unchanged — SPEC §9 says "notify without diff rather than
// skipping". If the state of a non-OPEN PR changes (e.g. CLOSED → MERGED
// after a merge), emits pr:status-changed.
func (p *Poller) enrich(ctx context.Context, rec store.PRRecord) store.PRRecord {
	details, err := p.client.GetPRDetails(ctx, rec.URL)
	if err != nil {
		p.log.WarnContext(ctx, "enrich failed; notifying without diff", "pr", rec.ID, "err", err)
		return rec
	}
	prevState := rec.State
	if err := p.store.RefreshPRStatus(rec.ID, *details); err != nil {
		p.log.WarnContext(ctx, "refresh status", "pr", rec.ID, "err", err)
	}
	rec.Additions = details.Additions
	rec.Deletions = details.Deletions
	rec.IsDraft = details.IsDraft
	if details.MergedAt != nil {
		rec.State = "MERGED"
	} else if details.State != "" {
		rec.State = details.State
	}
	if details.ReviewState != "" {
		rec.ReviewState = details.ReviewState
	}
	if prevState != "" && prevState != rec.State {
		prCopy := rec
		p.emit(Event{Kind: EventPRStatusChanged, PR: &prCopy})
	}
	return rec
}

// handlePollError logs and advances the backoff. Classified errors (auth
// expired, rate limited, transient) are differentiated for the logs but all
// feed the same exponential schedule — the tray/notifier will surface auth
// state separately in a later session.
func (p *Poller) handlePollError(err error) {
	switch {
	case errors.Is(err, github.ErrAuthExpired):
		p.log.Error("gh auth expired", "err", err)
	case errors.Is(err, github.ErrRateLimited):
		p.log.Warn("github rate limited", "err", err)
	case errors.Is(err, github.ErrTransient):
		p.log.Warn("transient gh failure", "err", err)
	default:
		p.log.Warn("poll failed", "err", err)
	}
	p.advanceBackoff()
}

func (p *Poller) advanceBackoff() {
	p.mu.Lock()
	defer p.mu.Unlock()
	switch p.backoff {
	case 0:
		p.backoff = p.interval * 2
	default:
		p.backoff *= 2
	}
	if p.backoff > MaxBackoff {
		p.backoff = MaxBackoff
	}
}

func (p *Poller) resetBackoff() {
	p.mu.Lock()
	p.backoff = 0
	p.mu.Unlock()
}

// CurrentBackoff exposes the current backoff duration for tests and for
// doctor/status introspection.
func (p *Poller) CurrentBackoff() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.backoff
}

// Interval returns the currently-scheduled polling interval.
func (p *Poller) Interval() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.interval
}

// RunOnce executes a single tick outside of Run. Useful for manual refresh
// triggered from the tray menu (SPEC §5.5 "Atualizar agora").
func (p *Poller) RunOnce(ctx context.Context) error {
	p.tick(ctx)
	if ctx.Err() != nil {
		return fmt.Errorf("run once: %w", ctx.Err())
	}
	return nil
}
