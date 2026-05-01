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
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/meopedevts/revu/assets"
	"github.com/meopedevts/revu/frontend"
	"github.com/meopedevts/revu/internal/app"
	"github.com/meopedevts/revu/internal/auth"
	appconfig "github.com/meopedevts/revu/internal/config"
	"github.com/meopedevts/revu/internal/github"
	"github.com/meopedevts/revu/internal/notifier"
	"github.com/meopedevts/revu/internal/poller"
	"github.com/meopedevts/revu/internal/profiles"
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

	paths, err := resolveRuntimePaths()
	if err != nil {
		return err
	}

	svc, err := bootstrapServices(ctx, paths)
	if err != nil {
		return err
	}
	defer svc.shutdown()

	bridge := buildBridge(svc)
	tr := buildTray(ctx, svc, bridge, stop)
	p := buildPoller(svc, bridge, tr)

	wireProfileChange(ctx, svc, bridge, tr, p)
	wireConfigChanges(svc, p)

	return launchRuntime(ctx, svc, bridge, tr, p, paths)
}

// runtimePaths agrupa os 3 caminhos canônicos resolvidos pelo runtime
// (config + db + state.json legado) pra evitar list-of-strings entre
// helpers.
type runtimePaths struct {
	config string
	db     string
	legacy string
}

// runtimeServices carrega o set de colaboradores construídos durante o
// bootstrap. Vive apenas dentro de runApp; nunca é exportado.
type runtimeServices struct {
	log    *slog.Logger
	cfg    appconfig.Config
	cfgMgr *appconfig.Manager
	store  store.Store
	profs  *profiles.Service
	client github.Client
	ntf    notifier.Notifier
}

// shutdown encerra recursos na ordem inversa de inicialização. Idempotente
// — chamado via defer no runApp.
//
// Close usa [context.Background] porque shutdown roda no defer de runApp,
// depois que `wails.Run` retornou — momento em que o ctx raiz já está
// cancelado. Passar esse ctx abortaria o `PRAGMA wal_checkpoint(TRUNCATE)`
// e deixaria o WAL gordo no disco.
func (r *runtimeServices) shutdown() {
	if err := r.store.Close(context.Background()); err != nil {
		r.log.Warn("close store", "err", err)
	}
	_ = r.ntf.Close()
}

// resolveRuntimePaths centraliza os 3 lookups em UserConfigDir. Erros são
// retornados sem wrapping pra preservar o sentinel original.
func resolveRuntimePaths() (runtimePaths, error) {
	cfgPath, err := configPath()
	if err != nil {
		return runtimePaths{}, err
	}
	db, err := dbPath()
	if err != nil {
		return runtimePaths{}, err
	}
	legacy, err := legacyStateJSONPath()
	if err != nil {
		return runtimePaths{}, err
	}
	return runtimePaths{config: cfgPath, db: db, legacy: legacy}, nil
}

// bootstrapServices monta cfgMgr → store → profiles → client → notifier
// nessa ordem (cada um depende do anterior). Aborta no primeiro erro,
// retornando-os com wrapping pra ficar claro qual estágio falhou.
func bootstrapServices(ctx context.Context, paths runtimePaths) (*runtimeServices, error) {
	log := slog.Default()
	executor := github.DefaultExecutor()

	cfgMgr, err := appconfig.Load(paths.config, appconfig.WithLogger(log))
	if err != nil {
		return nil, fmt.Errorf("carregar config: %w", err)
	}
	cfg := cfgMgr.Current()

	st := store.New(paths.db,
		store.WithRetention(cfg.HistoryRetentionDays),
		store.WithLogger(log),
		store.WithJSONMigration(paths.legacy),
	)
	if err := st.Load(ctx); err != nil {
		return nil, fmt.Errorf("carregar store: %w", err)
	}

	profSvc := profiles.NewService(
		profiles.NewRepository(st.DB()),
		auth.New(),
		executor,
		profiles.WithLogger(log),
	)

	client := github.NewClient(executor, profSvc)

	// Fail fast on auth using the active profile. gh-cli profiles defer to
	// the ambient session (checked via AuthStatus); keyring profiles must
	// have a valid token — validated via a cheap ValidateToken round-trip.
	if err := preflightAuth(ctx, client, profSvc); err != nil {
		_ = st.Close(context.Background())
		return nil, fmt.Errorf("pré-requisito falhou: %w", err)
	}

	ntf, err := notifier.New(notifier.WithExpireTimeout(time.Duration(cfg.NotificationTimeoutSeconds) * time.Second))
	if err != nil {
		_ = st.Close(context.Background())
		return nil, fmt.Errorf("inicializar notifier: %w", err)
	}
	ntf.SetEnabled(cfg.NotificationsEnabled)

	return &runtimeServices{
		log:    log,
		cfg:    cfg,
		cfgMgr: cfgMgr,
		store:  st,
		profs:  profSvc,
		client: client,
		ntf:    ntf,
	}, nil
}

// buildBridge instancia o app.App (Wails bridge) com os colaboradores
// já bootstrappados. onRefresh inicia vazio — é wired depois que o
// poller existe.
func buildBridge(svc *runtimeServices) *app.App {
	return app.New(
		svc.store,
		svc.cfgMgr,
		func() {},
		app.WithLogger(svc.log),
		app.WithProfiles(svc.profs),
		app.WithGitHubClient(svc.client),
	)
}

// buildTray monta o systray com callbacks completos (open/refresh/quit/
// settings/profile-list/profile-select). Refresh fica nil até o poller
// existir — o caller religa via tr.SetOnRefresh.
//
// fyne.io/systray on Linux SNI does not touch the GTK main thread, so
// it coexists fine with Wails (which owns main).
func buildTray(ctx context.Context, svc *runtimeServices, bridge *app.App, stop context.CancelFunc) *tray.Tray {
	tr := tray.New(
		bridge.ShowWindow,
		nil, // refresh wired after poller exists
		bridge.ShowSettings,
		func() {
			svc.log.Info("tray: quit requested")
			stop()
		},
		tray.WithLogger(svc.log),
		tray.WithProfileList(
			func() []tray.ProfileItem {
				list, err := svc.profs.List(ctx)
				if err != nil {
					svc.log.Warn("tray list profiles", "err", err)
					return nil
				}
				out := make([]tray.ProfileItem, 0, len(list))
				for _, pr := range list {
					out = append(out, tray.ProfileItem{
						ID: pr.ID, Name: pr.Name, IsActive: pr.IsActive,
					})
				}
				return out
			},
			func(id string) {
				if err := svc.profs.SetActive(ctx, id); err != nil {
					svc.log.Warn("tray set active", "err", err)
				}
			},
		),
	)
	// Seed initial state from the already-loaded store; saves one icon
	// flicker on startup when there are pending PRs waiting.
	tr.SetState(stateFromStore(ctx, svc.store))
	return tr
}

// buildPoller cria o poller com handler que faz fan-out pra bridge e
// tray. Não roda ainda — Run é disparado por launchRuntime.
func buildPoller(svc *runtimeServices, bridge *app.App, tr *tray.Tray) *poller.Poller {
	return poller.New(svc.client, svc.store, svc.ntf,
		poller.WithLogger(svc.log),
		poller.WithInterval(time.Duration(svc.cfg.PollingIntervalSeconds)*time.Second),
		poller.WithNotifyCooldown(time.Duration(svc.cfg.NotificationCooldownMinutes)*time.Minute),
		poller.WithActiveProfile(svc.profs),
		poller.WithEventHandler(func(e poller.Event) {
			bridge.OnPollEvent(e)
			syncTrayState(tr, svc.store, e)
		}),
	)
}

// wireProfileChange instala o subscriber de active-profile e finaliza os
// onRefresh callbacks que dependem do poller. Também faz o seed inicial
// do activeProfileID no store.
func wireProfileChange(
	ctx context.Context,
	svc *runtimeServices,
	bridge *app.App,
	tr *tray.Tray,
	p *poller.Poller,
) {
	// Wire the active-profile change fan-out: update store tagging, trigger
	// an immediate poll so the UI reflects the new account, emit on the
	// Wails bus for the frontend, and let the tray rebuild its submenu.
	svc.profs.SubscribeActive(func(pr profiles.Profile) {
		svc.log.Info("active profile changed",
			slog.String("id", pr.ID), slog.String("name", pr.Name))
		svc.store.SetActiveProfileID(pr.ID)
		bridge.EmitActiveProfileChanged(pr)
		tr.OnProfilesChanged()
		p.Trigger()
	})
	// Seed the store with the boot-time active id so the first tick tags
	// inserts correctly.
	if active, err := svc.profs.GetActive(ctx); err == nil {
		svc.store.SetActiveProfileID(active.ID)
	}
	bridge.SetOnRefresh(p.Trigger)
	tr.SetOnRefresh(func() {
		svc.log.Info("tray: refresh requested")
		p.Trigger()
	})
}

// wireConfigChanges aplica cada nova versão do config nos componentes
// vivos (poller interval, notifier toggle, store retention).
func wireConfigChanges(svc *runtimeServices, p *poller.Poller) {
	svc.cfgMgr.Subscribe(func(c appconfig.Config) {
		svc.log.Info("config reloaded",
			"polling_s", c.PollingIntervalSeconds,
			"notifications", c.NotificationsEnabled,
			"cooldown_min", c.NotificationCooldownMinutes,
			"retention_d", c.HistoryRetentionDays,
		)
		p.SetInterval(time.Duration(c.PollingIntervalSeconds) * time.Second)
		p.SetNotifyCooldown(time.Duration(c.NotificationCooldownMinutes) * time.Minute)
		svc.ntf.SetEnabled(c.NotificationsEnabled)
		svc.ntf.SetExpireTimeout(time.Duration(c.NotificationTimeoutSeconds) * time.Second)
		svc.store.SetRetentionDays(c.HistoryRetentionDays)
	})
}

// launchRuntime dispara as 3 goroutines (poller, systray dispatcher,
// shutdown bridge) e bloqueia em wails.Run. Drena o WaitGroup antes de
// retornar pra garantir que ninguém escreve em recursos já fechados.
func launchRuntime(
	ctx context.Context,
	svc *runtimeServices,
	bridge *app.App,
	tr *tray.Tray,
	p *poller.Poller,
	paths runtimePaths,
) error {
	var wg sync.WaitGroup

	wg.Go(func() {
		if err := p.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			svc.log.Warn("poller exit", "err", err)
		}
	})

	wg.Go(func() {
		tr.Start()
	})

	// Bridge ctx cancellation into both the systray loop and the Wails
	// runtime so Wails.Run (blocking on the main thread) returns.
	wg.Go(func() {
		<-ctx.Done()
		tr.Stop()
		// WailsCtx may be nil if shutdown fires before OnStartup.
		if wc := bridge.WailsCtx(); wc != nil {
			//nolint:contextcheck // wc é o contexto gerenciado pela Wails runtime; Quit precisa dele.
			wruntime.Quit(wc)
		}
	})

	svc.log.InfoContext(ctx, "revu started",
		"db", paths.db,
		"config", paths.config,
		"interval", p.Interval(),
		"retention_d", svc.cfg.HistoryRetentionDays,
	)

	runErr := wails.Run(&options.App{
		Title:             "revu",
		Width:             svc.cfg.Window.Width,
		Height:            svc.cfg.Window.Height,
		StartHidden:       svc.cfg.StartHidden,
		HideWindowOnClose: true,
		BackgroundColour:  themeBackgroundColour(svc.cfg.Theme),
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
		Bind:      []any{bridge},
	})

	wg.Wait()
	svc.log.InfoContext(ctx, "revu stopped")

	if runErr != nil {
		return fmt.Errorf("wails: %w", runErr)
	}
	return nil
}

// themeBackgroundColour maps the persisted theme to the webview's initial
// chrome color so the window does not flash white while the bundle parses.
// Values are pulled from frontend/src/style.css (`--background`) and
// converted to sRGB. Unknown themes fall back to light.
func themeBackgroundColour(theme string) *options.RGBA {
	if theme == "dark" {
		return &options.RGBA{R: 0x13, G: 0x13, B: 0x16, A: 255}
	}
	return &options.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 255}
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
func stateFromStore(ctx context.Context, s store.Store) tray.State {
	if len(s.GetPending(ctx)) > 0 {
		return tray.StatePending
	}
	return tray.StateIdle
}

// syncTrayState reacts to poller events. Any completed poll re-reads the
// store and picks idle vs pending. Usa Background() porque o callback é
// disparado fora do escopo do ctx do bootstrap; a query é curta e
// fire-and-forget — WithTimeout aqui seria over-engineering.
func syncTrayState(tr *tray.Tray, s store.Store, e poller.Event) {
	if e.Kind != poller.EventPollCompleted {
		return
	}
	tr.SetState(stateFromStore(context.Background(), s))
}

// preflightAuth verifies the active profile can talk to GitHub before the
// poller starts. gh-cli profiles → AuthStatus (ambient gh auth). Keyring
// profiles → read the token from the keyring and run a token validation.
// Token is never logged on any branch.
func preflightAuth(ctx context.Context, client github.Client, svc *profiles.Service) error {
	active, err := svc.GetActive(ctx)
	if err != nil {
		return fmt.Errorf("resolve active profile: %w", err)
	}
	switch active.AuthMethod {
	case profiles.AuthKeyring:
		token, err := svc.TokenFor(ctx, active)
		if err != nil {
			return fmt.Errorf("read token for profile %s: %w", active.Name, err)
		}
		if _, err := svc.ValidateToken(ctx, token); err != nil {
			return fmt.Errorf("validate token for profile %s: %w", active.Name, err)
		}
		return nil
	default:
		if err := client.AuthStatus(ctx); err != nil {
			return err
		}
		return nil
	}
}
