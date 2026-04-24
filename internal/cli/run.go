package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/meopedevts/revu/assets"
	"github.com/meopedevts/revu/frontend"
	"github.com/meopedevts/revu/internal/app"
	appconfig "github.com/meopedevts/revu/internal/config"
	"github.com/meopedevts/revu/internal/github"
	"github.com/meopedevts/revu/internal/notifier"
	"github.com/meopedevts/revu/internal/poller"
	"github.com/meopedevts/revu/internal/store"
	"github.com/meopedevts/revu/internal/tray"
)

var runCmd = &cobra.Command{
	Use:           "run",
	Short:         "Inicia o app (Wails + tray + poller + notifier)",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runApp,
}

func runApp(cmd *cobra.Command, _ []string) error {
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log := slog.Default()
	client := github.NewClient(github.DefaultExecutor())

	// Fail fast on auth — later sessions can degrade gracefully into a
	// tray error state. For now abort so systemd can restart / user sees
	// the message.
	if err := client.AuthStatus(ctx); err != nil {
		return fmt.Errorf("pré-requisito falhou: %w", err)
	}

	cfgPath, err := configPath()
	if err != nil {
		return err
	}
	dbFile, err := dbPath()
	if err != nil {
		return err
	}
	legacyJSON, err := legacyStateJSONPath()
	if err != nil {
		return err
	}

	cfgMgr, err := appconfig.Load(cfgPath, appconfig.WithLogger(log))
	if err != nil {
		return fmt.Errorf("carregar config: %w", err)
	}
	cfg := cfgMgr.Current()

	st := store.New(dbFile,
		store.WithRetention(cfg.HistoryRetentionDays),
		store.WithLogger(log),
		store.WithJSONMigration(legacyJSON),
	)
	if err := st.Load(); err != nil {
		return fmt.Errorf("carregar store: %w", err)
	}
	defer func() {
		if err := st.Close(); err != nil {
			log.Warn("close store", "err", err)
		}
	}()

	ntf, err := notifier.New(notifier.WithExpireTimeout(time.Duration(cfg.NotificationTimeoutSeconds) * time.Second))
	if err != nil {
		return fmt.Errorf("inicializar notifier: %w", err)
	}
	ntf.SetEnabled(cfg.NotificationsEnabled)
	defer ntf.Close()

	bridge := app.New(st, cfgMgr, func() {}, app.WithLogger(log))

	// fyne.io/systray on Linux SNI does not touch the GTK main thread, so
	// it coexists fine with Wails (which owns main).
	tr := tray.New(
		bridge.ShowWindow,
		nil, // refresh wired after poller exists
		bridge.ShowSettings,
		func() {
			log.Info("tray: quit requested")
			stop()
		},
		tray.WithLogger(log),
	)
	// Seed initial state from the already-loaded store; saves one icon
	// flicker on startup when there are pending PRs waiting.
	tr.SetState(stateFromStore(st))

	p := poller.New(client, st, ntf,
		poller.WithLogger(log),
		poller.WithInterval(time.Duration(cfg.PollingIntervalSeconds)*time.Second),
		poller.WithEventHandler(func(e poller.Event) {
			bridge.OnPollEvent(e)
			syncTrayState(tr, st, e)
		}),
	)
	bridge.SetOnRefresh(p.Trigger)
	tr.SetOnRefresh(func() {
		log.Info("tray: refresh requested")
		p.Trigger()
	})

	// Apply every validated config change to live components.
	cfgMgr.Subscribe(func(c appconfig.Config) {
		log.Info("config reloaded",
			"polling_s", c.PollingIntervalSeconds,
			"notifications", c.NotificationsEnabled,
			"retention_d", c.HistoryRetentionDays,
		)
		p.SetInterval(time.Duration(c.PollingIntervalSeconds) * time.Second)
		ntf.SetEnabled(c.NotificationsEnabled)
		ntf.SetExpireTimeout(time.Duration(c.NotificationTimeoutSeconds) * time.Second)
		st.SetRetentionDays(c.HistoryRetentionDays)
	})

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := p.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Warn("poller exit", "err", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		tr.Start()
	}()

	// Bridge ctx cancellation into both the systray loop and the Wails
	// runtime so Wails.Run (blocking on the main thread) returns.
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		tr.Stop()
		// WailsCtx may be nil if shutdown fires before OnStartup.
		if wc := bridge.WailsCtx(); wc != nil {
			wruntime.Quit(wc)
		}
	}()

	log.Info("revu started",
		"db", dbFile,
		"config", cfgPath,
		"interval", p.Interval(),
		"retention_d", cfg.HistoryRetentionDays,
	)

	runErr := wails.Run(&options.App{
		Title:             "revu",
		Width:             cfg.Window.Width,
		Height:            cfg.Window.Height,
		StartHidden:       cfg.StartHidden,
		HideWindowOnClose: true,
		AssetServer: &assetserver.Options{
			Assets: frontend.AssetsFS(),
		},
		Linux: &linux.Options{
			Icon:        assets.WindowIcon,
			ProgramName: "revu",
			// Wails defaults this to Never only when options.Linux is nil
			// (https://github.com/wailsapp/wails/issues/2977). Passing a
			// non-nil Options flips it to "Always" — that path crashes
			// webkit2gtk on Wayland/Hyprland with "Error 71 (Protocol
			// error)". Keep it Never.
			WebviewGpuPolicy: linux.WebviewGpuPolicyNever,
		},
		OnStartup: bridge.OnStartup,
		Bind:      []interface{}{bridge},
	})

	stop()
	wg.Wait()
	log.Info("revu stopped")

	if runErr != nil {
		return fmt.Errorf("wails: %w", runErr)
	}
	return nil
}

// dbPath resolves ~/.config/revu/revu.db consistently with configPath().
func dbPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(dir, "revu", "revu.db"), nil
}

// legacyStateJSONPath points at the pre-SQLite persistence file. Handed to
// the store so the first boot after upgrade imports it into SQLite.
func legacyStateJSONPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(dir, "revu", "state.json"), nil
}

// stateFromStore derives the initial tray state from the loaded store.
func stateFromStore(s store.Store) tray.State {
	if len(s.GetPending()) > 0 {
		return tray.StatePending
	}
	return tray.StateIdle
}

// syncTrayState reacts to poller events. An auth-expired tick flips to
// error; any other completed poll re-reads the store and picks idle vs
// pending.
func syncTrayState(tr *tray.Tray, s store.Store, e poller.Event) {
	if e.Kind != poller.EventPollCompleted {
		return
	}
	if isAuthError(e.Err) {
		tr.SetState(tray.StateError)
		return
	}
	tr.SetState(stateFromStore(s))
}

// isAuthError looks for the sentinel string emitted by poller.handlePollError
// → classify() path. Substring match is fragile but contained to one place.
func isAuthError(msg string) bool {
	if msg == "" {
		return false
	}
	// matches github.ErrAuthExpired.Error()
	return strings.Contains(msg, "gh auth expired")
}
