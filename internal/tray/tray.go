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

// ProfileItem is the minimal shape the tray needs to render the "Trocar
// conta" submenu without importing the profiles package. Populated by the
// caller via WithProfileList.
type ProfileItem struct {
	ID       string
	Name     string
	IsActive bool
}

// ProfileLister returns the current list of profiles, with IsActive set on
// the selected row. Called on menu build — should be cheap.
type ProfileLister func() []ProfileItem

// ProfileSelector is called when the user clicks a profile in the submenu.
type ProfileSelector func(id string)

// Tray holds the wiring for a single SNI item. It is not reentrant —
// Start/Stop are designed to be called exactly once per process, per the
// fyne.io/systray lifecycle.
type Tray struct {
	onOpen      func()
	onRefresh   func()
	onSettings  func()
	onQuit      func()
	onSelectPrf ProfileSelector
	listPrfs    ProfileLister
	log         *slog.Logger

	mu      sync.Mutex
	ready   bool
	current State

	// buildCh signals onReady's loop goroutine to rebuild the menu. Capacity
	// 1 + non-blocking send: bursts are coalesced.
	buildCh chan struct{}
	// stopCh is closed by Stop to unblock the loop when Quit is pending.
	stopCh chan struct{}
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

// WithProfileList enables the "Trocar conta" submenu. lister is called on
// every menu rebuild to render the current profile list; selector is called
// when the user picks a row.
func WithProfileList(lister ProfileLister, selector ProfileSelector) Option {
	return func(t *Tray) {
		t.listPrfs = lister
		t.onSelectPrf = selector
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
		buildCh:    make(chan struct{}, 1),
		stopCh:     make(chan struct{}),
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
	select {
	case <-t.stopCh:
	default:
		close(t.stopCh)
	}
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

// OnProfilesChanged requests a menu rebuild so the "Trocar conta" submenu
// reflects the latest list (new profile added, removed, or active flipped).
// Safe to call before Start — the rebuild runs on the next onReady.
func (t *Tray) OnProfilesChanged() {
	select {
	case t.buildCh <- struct{}{}:
	default:
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

	// onReady must return promptly so the systray library can pump D-Bus
	// events; the rebuild/dispatch loop runs on its own goroutine.
	go t.buildMenuAndLoop()
}

// buildMenuAndLoop constructs the menu, dispatches click events, and rebuilds
// on OnProfilesChanged. The loop exits when Stop is called or the user picks
// "Sair".
func (t *Tray) buildMenuAndLoop() {
	for {
		items := t.buildMenu()
		done := make(chan struct{})
		dispatchDone := make(chan struct{})
		go func() {
			t.dispatch(items, done)
			close(dispatchDone)
		}()
		select {
		case <-t.buildCh:
			// Rebuild requested: tear down the current dispatcher, reset the
			// menu, and loop to re-build.
			close(done)
			<-dispatchDone
			systray.ResetMenu()
		case <-t.stopCh:
			close(done)
			<-dispatchDone
			return
		case <-dispatchDone:
			// dispatch exited on its own (user picked Sair).
			return
		}
	}
}

type menuItems struct {
	open     *systray.MenuItem
	refresh  *systray.MenuItem
	config   *systray.MenuItem
	quit     *systray.MenuItem
	switchTo map[string]*systray.MenuItem // profile id → menu item
}

// buildMenu assembles every menu row from scratch. Must be called after
// ResetMenu (or on first onReady).
func (t *Tray) buildMenu() menuItems {
	mOpen := systray.AddMenuItem("Abrir", "Mostra a janela do revu")
	mRefresh := systray.AddMenuItem("Atualizar agora", "Forçar um poll imediato")
	mConfig := systray.AddMenuItem("Configurações", "Abrir tela de configurações")

	switchTo := map[string]*systray.MenuItem{}
	if t.listPrfs != nil && t.onSelectPrf != nil {
		profs := t.listPrfs()
		if len(profs) > 0 {
			mSwitch := systray.AddMenuItem("Trocar conta", "Seleciona a conta GitHub ativa")
			for _, pr := range profs {
				label := pr.Name
				if pr.IsActive {
					label = "● " + label
				} else {
					label = "○ " + label
				}
				item := mSwitch.AddSubMenuItem(label, "")
				if pr.IsActive {
					item.Disable()
				}
				switchTo[pr.ID] = item
			}
		}
	}

	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Sair", "Encerra o revu")

	if t.onOpen == nil {
		mOpen.Disable()
	}
	if t.onSettings == nil {
		mConfig.Disable()
	}
	return menuItems{
		open:     mOpen,
		refresh:  mRefresh,
		config:   mConfig,
		quit:     mQuit,
		switchTo: switchTo,
	}
}

// dispatch forwards click events from the built menu to the registered
// callbacks until done is closed. Each menu row is watched by its own
// goroutine that loops until shutdown — this keeps the select-case count
// static regardless of how many profiles we render.
func (t *Tray) dispatch(items menuItems, done <-chan struct{}) {
	watch := func(ch <-chan struct{}, run func()) {
		for {
			select {
			case <-ch:
				run()
			case <-done:
				return
			}
		}
	}

	go watch(items.open.ClickedCh, func() {
		if fn := t.getOnOpen(); fn != nil {
			fn()
		}
	})
	go watch(items.refresh.ClickedCh, func() {
		if fn := t.getOnRefresh(); fn != nil {
			fn()
		}
	})
	go watch(items.config.ClickedCh, func() {
		if fn := t.getOnSettings(); fn != nil {
			fn()
		}
	})

	for id, mi := range items.switchTo {
		id := id
		mi := mi
		go watch(mi.ClickedCh, func() {
			if fn := t.getOnSelectProfile(); fn != nil {
				fn(id)
			}
		})
	}

	// Quit returns from dispatch so buildMenuAndLoop can exit the outer loop.
	select {
	case <-items.quit.ClickedCh:
		if fn := t.getOnQuit(); fn != nil {
			fn()
		}
		systray.Quit()
		return
	case <-done:
		return
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

func (t *Tray) getOnSelectProfile() ProfileSelector {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.onSelectPrf
}

func (t *Tray) onExit() {
	t.log.Info("tray exited")
}
