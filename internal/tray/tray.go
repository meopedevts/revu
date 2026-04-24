// Package tray publishes the app icon and contextual menu via
// StatusNotifierItem (SNI). It consumes external callbacks for Refresh /
// Quit so it does not depend on the poller package directly — dependency
// flows from the caller (revu run) outward.
package tray

import (
	"log/slog"
	"sync"

	"fyne.io/systray"

	"github.com/meopedevts/revu/assets"
)

// State is the visual tray mode; the icon byte-buffer is picked from assets
// by state (SPEC REV-10): idle when everything's healthy but no pending PR,
// pending when at least one review is waiting, error on auth failure.
type State int

const (
	StateIdle State = iota
	StatePending
	StateError
)

// Tray holds the wiring for a single SNI item. It is not reentrant —
// Start/Stop are designed to be called exactly once per process, per the
// fyne.io/systray lifecycle.
type Tray struct {
	onOpen     func()
	onRefresh  func()
	onSettings func()
	onQuit     func()
	log        *slog.Logger

	mu      sync.Mutex
	ready   bool
	current State
}

// Option customizes the tray.
type Option func(*Tray)

// WithLogger injects a logger.
func WithLogger(l *slog.Logger) Option {
	return func(t *Tray) {
		if l != nil {
			t.log = l
		}
	}
}

// New wires up the callbacks. onSettings opens the in-app settings view
// (Wails window + ui:navigate event). onOpen/onSettings may be nil when the
// Wails window is not available (e.g. headless smoke).
func New(onOpen, onRefresh, onSettings, onQuit func(), opts ...Option) *Tray {
	t := &Tray{
		onOpen:     onOpen,
		onRefresh:  onRefresh,
		onSettings: onSettings,
		onQuit:     onQuit,
		log:        slog.Default(),
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Start blocks the calling goroutine and runs the systray event loop until
// Stop is called, the user picks "Sair", or the underlying systray exits.
func (t *Tray) Start() {
	systray.Run(t.onReady, t.onExit)
}

// Stop asks the systray loop to terminate. Safe to call from any goroutine.
func (t *Tray) Stop() {
	systray.Quit()
}

// SetOnRefresh replaces the refresh callback. Used by run.go to break the
// cyclic dep between Tray and Poller.
func (t *Tray) SetOnRefresh(fn func()) {
	t.mu.Lock()
	t.onRefresh = fn
	t.mu.Unlock()
}

// SetState swaps the tray icon. Safe to call from any goroutine and from
// before Start — the state is applied when onReady fires. Repeated calls
// with the same state are no-ops (avoids gratuitous SNI churn).
func (t *Tray) SetState(s State) {
	t.mu.Lock()
	if t.current == s && t.ready {
		t.mu.Unlock()
		return
	}
	t.current = s
	ready := t.ready
	t.mu.Unlock()
	if ready {
		systray.SetIcon(iconFor(s))
	}
}

func iconFor(s State) []byte {
	switch s {
	case StatePending:
		return assets.TrayPending
	case StateError:
		return assets.TrayError
	default:
		return assets.TrayIdle
	}
}

func (t *Tray) onReady() {
	t.mu.Lock()
	t.ready = true
	state := t.current
	t.mu.Unlock()

	systray.SetIcon(iconFor(state))
	systray.SetTitle("revu")
	systray.SetTooltip("revu — PR review notifier")

	mOpen := systray.AddMenuItem("Abrir", "Mostra a janela do revu")
	mRefresh := systray.AddMenuItem("Atualizar agora", "Forçar um poll imediato")
	mConfig := systray.AddMenuItem("Configurações", "Abrir tela de configurações")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Sair", "Encerra o revu")

	if t.onOpen == nil {
		mOpen.Disable()
	}
	if t.onSettings == nil {
		mConfig.Disable()
	}

	go t.loop(mOpen, mRefresh, mConfig, mQuit)
}

func (t *Tray) loop(mOpen, mRefresh, mConfig, mQuit *systray.MenuItem) {
	for {
		select {
		case <-mOpen.ClickedCh:
			if fn := t.getOnOpen(); fn != nil {
				fn()
			}
		case <-mRefresh.ClickedCh:
			if fn := t.getOnRefresh(); fn != nil {
				fn()
			}
		case <-mConfig.ClickedCh:
			if fn := t.getOnSettings(); fn != nil {
				fn()
			}
		case <-mQuit.ClickedCh:
			if fn := t.getOnQuit(); fn != nil {
				fn()
			}
			systray.Quit()
			return
		}
	}
}

func (t *Tray) getOnOpen() func() {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.onOpen
}

func (t *Tray) getOnRefresh() func() {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.onRefresh
}

func (t *Tray) getOnQuit() func() {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.onQuit
}

func (t *Tray) getOnSettings() func() {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.onSettings
}

func (t *Tray) onExit() {
	t.log.Info("tray exited")
}
