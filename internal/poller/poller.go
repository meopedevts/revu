package poller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

// Poller ticks the gh search, enriches new PRs, notifies, and persists.
// Life-cycle is owned by Run; callers cancel the ctx to stop.
type Poller struct {
	client   github.Client
	store    store.Store
	notifier notifier.Notifier
	log      *slog.Logger

	interval  time.Duration
	backoff   time.Duration // current backoff (0 when healthy)
	triggerCh chan struct{} // capacity 1; Trigger coalesces bursts
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

// WithLogger injects a logger. Defaults to slog.Default().
func WithLogger(l *slog.Logger) Option {
	return func(p *Poller) {
		if l != nil {
			p.log = l
		}
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

// Run drives the ticker until ctx is cancelled. The first tick fires
// immediately (SPEC §8.2 step 7). On error it applies exponential backoff,
// doubling up to MaxBackoff; on any success it resets to the configured
// interval. Also honors Trigger for out-of-schedule polls.
func (p *Poller) Run(ctx context.Context) error {
	p.tick(ctx)
	for {
		wait := p.interval
		if p.backoff > 0 {
			wait = p.backoff
		}
		timer := time.NewTimer(wait)
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

// tick executes one poll cycle. Errors are logged and applied to backoff but
// never propagated out of Run — we keep polling through transient failures.
func (p *Poller) tick(ctx context.Context) {
	summaries, err := p.client.ListReviewRequested(ctx)
	if err != nil {
		p.handlePollError(err)
		return
	}
	p.resetBackoff()

	novos := p.store.UpdateFromPoll(summaries)
	for _, rec := range novos {
		enriched := p.enrich(ctx, rec)
		if err := p.notifier.Notify(enriched); err != nil {
			p.log.Warn("notify failed", "pr", rec.ID, "err", err)
			continue
		}
	}
	if err := p.store.Save(); err != nil {
		p.log.Warn("save store", "err", err)
	}
}

// enrich fetches additions/deletions for a new PR. On failure it returns
// the record unchanged — SPEC §9 says "notify without diff rather than
// skipping".
func (p *Poller) enrich(ctx context.Context, rec store.PRRecord) store.PRRecord {
	details, err := p.client.GetPRDetails(ctx, rec.URL)
	if err != nil {
		p.log.Warn("enrich failed; notifying without diff", "pr", rec.ID, "err", err)
		return rec
	}
	if err := p.store.RefreshPRStatus(rec.ID, *details); err != nil {
		p.log.Warn("refresh status", "pr", rec.ID, "err", err)
	}
	rec.Additions = details.Additions
	rec.Deletions = details.Deletions
	rec.IsDraft = details.IsDraft
	if details.MergedAt != nil {
		rec.State = "MERGED"
	} else if details.State != "" {
		rec.State = details.State
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
	switch {
	case p.backoff == 0:
		p.backoff = p.interval * 2
	default:
		p.backoff *= 2
	}
	if p.backoff > MaxBackoff {
		p.backoff = MaxBackoff
	}
}

func (p *Poller) resetBackoff() {
	p.backoff = 0
}

// CurrentBackoff exposes the current backoff duration for tests and for
// doctor/status introspection.
func (p *Poller) CurrentBackoff() time.Duration { return p.backoff }

// RunOnce executes a single tick outside of Run. Useful for manual refresh
// triggered from the tray menu (SPEC §5.5 "Atualizar agora").
func (p *Poller) RunOnce(ctx context.Context) error {
	p.tick(ctx)
	if ctx.Err() != nil {
		return fmt.Errorf("run once: %w", ctx.Err())
	}
	return nil
}
