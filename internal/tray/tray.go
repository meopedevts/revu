// Package tray publishes the app icon and contextual menu via
// StatusNotifierItem (SNI). It consumes external callbacks for Refresh /
// Quit so it does not depend on the poller package directly — dependency
// flows from the caller (revu run) outward.
package tray

import (
	"log/slog"
	"os/exec"

	"fyne.io/systray"

	"github.com/meopedevts/revu/assets"
)

// Tray holds the wiring for a single SNI item. It is not reentrant —
// Start/Stop are designed to be called exactly once per process, per the
// fyne.io/systray lifecycle.
type Tray struct {
	onRefresh  func()
	onQuit     func()
	configPath string
	log        *slog.Logger
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

// New wires up the callbacks. configPath is the file opened by the
// "Configurações" menu item via xdg-open.
func New(onRefresh, onQuit func(), configPath string, opts ...Option) *Tray {
	t := &Tray{
		onRefresh:  onRefresh,
		onQuit:     onQuit,
		configPath: configPath,
		log:        slog.Default(),
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Start blocks the calling goroutine (must be the main one — SNI requires
// the main OS thread on some backends) and runs the systray event loop
// until Stop is called, the user picks "Sair", or the underlying systray
// exits.
func (t *Tray) Start() {
	systray.Run(t.onReady, t.onExit)
}

// Stop asks the systray loop to terminate. Safe to call from any goroutine.
func (t *Tray) Stop() {
	systray.Quit()
}

func (t *Tray) onReady() {
	systray.SetIcon(assets.IconIdle)
	systray.SetTitle("revu")
	systray.SetTooltip("revu — PR review notifier")

	mRefresh := systray.AddMenuItem("Atualizar agora", "Forçar um poll imediato")
	mConfig := systray.AddMenuItem("Configurações", "Abrir config.json no editor padrão")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Sair", "Encerra o revu")

	go t.loop(mRefresh, mConfig, mQuit)
}

func (t *Tray) loop(mRefresh, mConfig, mQuit *systray.MenuItem) {
	for {
		select {
		case <-mRefresh.ClickedCh:
			if t.onRefresh != nil {
				t.onRefresh()
			}
		case <-mConfig.ClickedCh:
			t.openConfig()
		case <-mQuit.ClickedCh:
			if t.onQuit != nil {
				t.onQuit()
			}
			systray.Quit()
			return
		}
	}
}

// openConfig launches xdg-open on the config path. MVP does not block nor
// report failures back to the menu — any error is logged only.
func (t *Tray) openConfig() {
	if t.configPath == "" {
		t.log.Warn("no config path set")
		return
	}
	cmd := exec.Command("xdg-open", t.configPath)
	if err := cmd.Start(); err != nil {
		t.log.Warn("xdg-open failed", "path", t.configPath, "err", err)
		return
	}
	// Let the child outlive us — nothing to wait on.
	go func() { _ = cmd.Wait() }()
}

func (t *Tray) onExit() {
	t.log.Info("tray exited")
}
