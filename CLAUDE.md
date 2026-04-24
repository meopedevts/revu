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

## Estrutura alvo (SPEC §10)

```
cmd/revu/main.go
internal/{app,cli,config,github,notifier,poller,store,tray}/
frontend/src/
assets/            # ícones embed via go:embed
```

Scaffold atual = Wails default (`app.go` + `main.go` na raiz). Refatorar pra layout `cmd/` + `internal/` antes da fase 1.

## Comandos

```bash
wails dev                  # dev server
wails build                # build release → build/bin/revu
install -Dm755 build/bin/revu ~/.local/bin/revu
```

CLI alvo: `revu run|version|config|doctor` (cobra).

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
