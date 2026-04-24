package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

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
	statePath, err := statePath()
	if err != nil {
		return err
	}

	cfgMgr, err := appconfig.Load(cfgPath, appconfig.WithLogger(log))
	if err != nil {
		return fmt.Errorf("carregar config: %w", err)
	}
	cfg := cfgMgr.Current()

	st := store.New(statePath, store.WithRetention(cfg.HistoryRetentionDays))
	if err := st.Load(); err != nil {
		return fmt.Errorf("carregar store: %w", err)
	}

	ntf, err := notifier.New(notifier.WithExpireTimeout(time.Duration(cfg.NotificationTimeoutSeconds) * time.Second))
	if err != nil {
		return fmt.Errorf("inicializar notifier: %w", err)
	}
	ntf.SetEnabled(cfg.NotificationsEnabled)
	defer ntf.Close()

	bridge := app.New(st, func() {}, app.WithLogger(log))
	p := poller.New(client, st, ntf,
		poller.WithLogger(log),
		poller.WithInterval(time.Duration(cfg.PollingIntervalSeconds)*time.Second),
		poller.WithEventHandler(bridge.OnPollEvent),
	)
	bridge.SetOnRefresh(p.Trigger)

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

	// fyne.io/systray on Linux SNI does not touch the GTK main thread, so
	// it coexists fine with Wails (which owns main).
	tr := tray.New(
		bridge.ShowWindow,
		func() {
			log.Info("tray: refresh requested")
			p.Trigger()
		},
		func() {
			log.Info("tray: quit requested")
			stop()
		},
		cfgPath,
		tray.WithLogger(log),
	)
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
		"state", statePath,
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
		OnStartup: bridge.OnStartup,
		Bind:      []interface{}{bridge},
	})

	stop()
	wg.Wait()

	if saveErr := st.Save(); saveErr != nil {
		log.Warn("save on shutdown", "err", saveErr)
	}
	log.Info("revu stopped")

	if runErr != nil {
		return fmt.Errorf("wails: %w", runErr)
	}
	return nil
}

// statePath resolves ~/.config/revu/state.json consistently with configPath().
func statePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(dir, "revu", "state.json"), nil
}
