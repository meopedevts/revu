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

	"github.com/spf13/cobra"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/meopedevts/revu/frontend"
	"github.com/meopedevts/revu/internal/app"
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
	st := store.New(statePath)
	if err := st.Load(); err != nil {
		return fmt.Errorf("carregar store: %w", err)
	}

	ntf, err := notifier.New()
	if err != nil {
		return fmt.Errorf("inicializar notifier: %w", err)
	}
	defer ntf.Close()

	bridge := app.New(st, func() {}, app.WithLogger(log))
	p := poller.New(client, st, ntf,
		poller.WithLogger(log),
		poller.WithEventHandler(bridge.OnPollEvent),
	)
	// Re-wire the refresh callback now that we have the poller.
	bridge.SetOnRefresh(p.Trigger)

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

	log.Info("revu started", "state", statePath, "config", cfgPath, "interval", poller.DefaultInterval)

	runErr := wails.Run(&options.App{
		Title:             "revu",
		Width:             480,
		Height:            640,
		StartHidden:       true,
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
