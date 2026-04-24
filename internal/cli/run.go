package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/meopedevts/revu/internal/github"
	"github.com/meopedevts/revu/internal/notifier"
	"github.com/meopedevts/revu/internal/poller"
	"github.com/meopedevts/revu/internal/store"
)

var runCmd = &cobra.Command{
	Use:           "run",
	Short:         "Inicia o app (poller + notifier; tray + UI em fases futuras)",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		log := slog.Default()
		client := github.NewClient(github.DefaultExecutor())

		// Fail fast on auth — SPEC §8.2: if gh auth fails, tray goes to error
		// state and poller does not start. Tray is REV-5+; for now abort the
		// command so systemd can restart and user sees the error.
		if err := client.AuthStatus(ctx); err != nil {
			return fmt.Errorf("pré-requisito falhou: %w", err)
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
		log.Info("revu started", "state", statePath, "interval", poller.DefaultInterval)

		err = p.Run(ctx)
		// Persist one last time so in-flight changes are not lost.
		if saveErr := st.Save(); saveErr != nil {
			log.Warn("save on shutdown", "err", saveErr)
		}
		if err != nil && !errors.Is(err, context.Canceled) {
			return err
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
