# revu

> Vigia silencioso pra PRs de review que te marcam no GitHub.

**Status:** 🚧 WIP — em desenvolvimento ativo (fase 1 / 4).

System tray app Linux que monitora GitHub em background e notifica
quando há Pull Requests aguardando seu review. Mantém histórico das
solicitações com status atualizado, acessível via tray.

Especificação completa em [`SPEC.md`](./SPEC.md).

## Pré-requisitos

**Runtime:**

- Go 1.23+
- `gh` CLI autenticado (`gh auth status` OK)

**Sistema (Arch/CachyOS):**

- `libayatana-appindicator` — publicação SNI pro Waybar
- `webkit2gtk-4.1` — Wails v2
- Daemon freedesktop: `mako` / `dunst` / `swaync`

**Ambiente alvo:** Hyprland + Waybar (Wayland). Roda também em KDE
Plasma, Sway + Waybar, GNOME com extensão AppIndicator.

## Build

```bash
task build            # build/bin/revu
task run              # build + revu run
task install          # ~/.local/bin/revu
task test             # go test -race ./...
task release          # wails build (app completo com UI)
```

Sem [go-task](https://taskfile.dev): `go build -o build/bin/revu ./cmd/revu`.

## CLI

```
revu run         # inicia app (tray + poller + janela)
revu version     # versão + commit + data de build
revu config      # imprime path do config.json
revu doctor      # valida deps runtime
```

## Licença

TBD.
