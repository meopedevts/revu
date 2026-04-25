# revu

System tray Linux pra PRs com review solicitado no GitHub. Veja `SPEC.md` pra especificação completa.

## Stack

- Go 1.23+ / Wails v2
- React + TypeScript + Vite
- Tray: `fyne.io/systray` (SNI via `libayatana-appindicator`)
- Notificações: `github.com/esiqveland/notify` (D-Bus direto, hint `transient=true` obrigatória)
- Config: `spf13/viper` · CLI: `spf13/cobra`
- GitHub: wrapper sobre `gh` CLI via `os/exec`
- Store: JSON em `~/.config/revu/state.json`
- Config: `~/.config/revu/config.json`

## Pré-requisitos runtime

- `gh` CLI autenticado (`gh auth status` OK)
- `libayatana-appindicator`, `webkit2gtk-4.1`
- Daemon notify freedesktop (`mako`/`dunst`/`swaync`)

## Pré-requisitos dev

- `golangci-lint` ≥ 2.11 (config em `.golangci.yml`, schema v2). Instalar via
  `mise use -g golangci-lint@latest` ou pacote da distro.

## Estrutura (SPEC §10)

```
cmd/revu/main.go          # entrypoint, passa ldflags pra cli
internal/
  app/                    # Wails wiring (fase 3)
  cli/                    # cobra: root, run, version, config, doctor
  config/                 # viper (fase 9)
  github/                 # wrapper gh CLI
  notifier/               # notify (D-Bus)
  poller/                 # goroutine + ticker
  store/                  # JSON persistence
  tray/                   # fyne.io/systray
frontend/src/             # React TS (Wails)
assets/                   # ícones embed via go:embed
```

## Build & Run

```bash
task build                # build/bin/revu
task run                  # build + revu run
task install              # ~/.local/bin/revu
task test                 # go test -race ./...
task fmt                  # gofmt + goimports + golines via golangci-lint
task lint                 # task fmt + golangci-lint run -v
task check                # task lint + go vet + go test -race
task release              # wails build (app completo com UI)
```

## Política de qualidade — obrigatório após qualquer mudança de código

Toda alteração em `*.go` exige `task check` verde **antes** de commit/push.
A task encadeia, em ordem:

1. `golangci-lint fmt ./...` — aplica gofmt + goimports + golines.
2. `golangci-lint run -v ./...` — roda o set completo do `.golangci.yml`.
3. `go vet ./...`.
4. `go test -race ./...`.

Qualquer falha bloqueia o commit. Se um lint novo aparecer, corrija no
código — **não** desabilite o linter sem justificativa explícita no
`.golangci.yml` ou via `//nolint:<linter> // <motivo>` direcionado.

CLI: `revu run|version|config|doctor` (cobra). Ldflags injetam
`main.version`/`main.commit`/`main.date` — forwarded pro pacote
`internal/cli` via `SetBuildInfo`.

## Fases (SPEC §11)

1. Core — gh wrapper + poller + store + notifier (sem UI)
2. Systray — spike `fyne.io/systray` antes de menu completo, validar no Waybar
3. Janela Wails — abas pendentes/histórico, badges, `xdg-open`
4. Polimento — Viper hot reload, retention, ícones por estado

## Decisões chave

- **Notificações transient** — fonte de verdade é o tray, não histórico do daemon
- **Re-request detection** — `review_pending` false → true conta como novo
- **Status refresh** — a cada N ticks (default 12 = 1h) revisa histórico
- **Rate limit** — backoff exponencial até 30min, reseta no sucesso
- **Ícones** — embed via `go:embed` (pasta `assets/`), extraídos pra tmp no boot

## Target

Hyprland + Waybar (Wayland). Deve rodar em KDE/Sway/GNOME com AppIndicator. Sem suporte Win/macOS no MVP.
