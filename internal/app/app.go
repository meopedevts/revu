// Package app is the Wails v2 bridge between the Go core and the React
// frontend. Methods on App become JS-callable bindings; keep the surface
// small and serializable.
package app

import (
	"context"
	"log/slog"
	"os/exec"
	"sync"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/meopedevts/revu/internal/poller"
	"github.com/meopedevts/revu/internal/store"
)

// App is the exported type Wails generates TS/JS bindings for.
type App struct {
	store     store.Store
	onRefresh func()
	log       *slog.Logger

	mu  sync.RWMutex
	ctx context.Context
}

// Option customizes App.
type Option func(*App)

// WithLogger injects a logger.
func WithLogger(l *slog.Logger) Option {
	return func(a *App) {
		if l != nil {
			a.log = l
		}
	}
}

// New wires the bridge. onRefresh is called by RefreshNow to ping the poller.
// The callback can be rewired post-construction via SetOnRefresh to break the
// cyclic dep between App and Poller (App needs poller.Trigger; Poller needs
// App.OnPollEvent).
func New(s store.Store, onRefresh func(), opts ...Option) *App {
	a := &App{
		store:     s,
		onRefresh: onRefresh,
		log:       slog.Default(),
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// SetOnRefresh replaces the refresh callback. Safe to call after New but
// before Wails runtime is serving requests.
func (a *App) SetOnRefresh(fn func()) {
	a.mu.Lock()
	a.onRefresh = fn
	a.mu.Unlock()
}

// OnStartup is the Wails lifecycle hook fired when the runtime is ready.
// Store the ctx so later methods can call wruntime.WindowShow / Hide.
func (a *App) OnStartup(ctx context.Context) {
	a.mu.Lock()
	a.ctx = ctx
	a.mu.Unlock()
}

// getCtx returns the Wails runtime context, if set.
func (a *App) getCtx() context.Context {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.ctx
}

// WailsCtx exposes the Wails runtime context for callers that need to invoke
// package-level runtime helpers (e.g. runtime.Quit from an out-of-band
// shutdown watcher). Returns nil before OnStartup has fired.
func (a *App) WailsCtx() context.Context { return a.getCtx() }

// ListPendingPRs returns PRs with review_pending=true, most-recent first.
func (a *App) ListPendingPRs() []store.PRRecord {
	return a.store.GetPending()
}

// ListHistoryPRs returns PRs with review_pending=false, most-recent first.
func (a *App) ListHistoryPRs() []store.PRRecord {
	return a.store.GetHistory()
}

// OpenPRInBrowser hands the URL off to xdg-open. Errors are logged, not
// surfaced — the frontend has no recovery path.
func (a *App) OpenPRInBrowser(url string) {
	if url == "" {
		return
	}
	cmd := exec.Command("xdg-open", url)
	if err := cmd.Start(); err != nil {
		a.log.Warn("xdg-open failed", "url", url, "err", err)
		return
	}
	go func() { _ = cmd.Wait() }()
}

// RefreshNow asks the poller to run an out-of-schedule tick.
func (a *App) RefreshNow() {
	a.mu.RLock()
	fn := a.onRefresh
	a.mu.RUnlock()
	if fn != nil {
		fn()
	}
}

// ShowWindow reveals the window (used by the tray "Abrir" item and after
// launch from autostart when user wants to inspect the list).
func (a *App) ShowWindow() {
	ctx := a.getCtx()
	if ctx == nil {
		return
	}
	wruntime.WindowShow(ctx)
}

// HideWindow hides the window without terminating the process — used by the
// frontend when the user clicks the X (HideWindowOnClose handles the native
// close; this method lets the React side trigger a hide from custom UI).
func (a *App) HideWindow() {
	ctx := a.getCtx()
	if ctx == nil {
		return
	}
	wruntime.WindowHide(ctx)
}

// OnPollEvent is the EventHandler wired into the poller. It forwards each
// event to the Wails event bus so the frontend can react live. Events emitted
// before OnStartup (i.e. before the Wails runtime is up) are dropped — the
// frontend will pick up the current state on its initial fetch.
func (a *App) OnPollEvent(e poller.Event) {
	ctx := a.getCtx()
	if ctx == nil {
		return
	}
	wruntime.EventsEmit(ctx, e.Kind, e)
}
