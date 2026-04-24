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

	"github.com/meopedevts/revu/internal/github"
	"github.com/meopedevts/revu/internal/notifier"
	"github.com/meopedevts/revu/internal/poller"
	"github.com/meopedevts/revu/internal/store"
	"github.com/meopedevts/revu/internal/tray"
)

var runCmd = &cobra.Command{
	Use:           "run",
	Short:         "Inicia o app (tray + poller + notifier; janela Wails em fases futuras)",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		log := slog.Default()
		client := github.NewClient(github.DefaultExecutor())

		// Fail fast on auth — SPEC §8.2. A future session will move this to
		// a degraded-tray state instead of aborting.
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

		p := poller.New(client, st, ntf, poller.WithLogger(log))

		// Poller runs in its own goroutine; tray owns the main thread.
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := p.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				log.Warn("poller exit", "err", err)
			}
		}()

		tr := tray.New(
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

		// Bridge ctx cancellation (SIGINT, SIGTERM, or Sair) into the systray
		// loop so the main-thread tray.Start() returns.
		go func() {
			<-ctx.Done()
			tr.Stop()
		}()

		log.Info("revu started", "state", statePath, "config", cfgPath, "interval", poller.DefaultInterval)

		// Blocks on main thread until the systray loop exits.
		tr.Start()

		// Ensure the ctx is cancelled even if the tray loop exited first for
		// any reason — defensive, cheap.
		stop()
		wg.Wait()

		if saveErr := st.Save(); saveErr != nil {
			log.Warn("save on shutdown", "err", saveErr)
		}
		log.Info("revu stopped")
		return nil
	},
}

// statePath resolves ~/.config/revu/state.json consistently with configPath().
func statePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(dir, "revu", "state.json"), nil
}
