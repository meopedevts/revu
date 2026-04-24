# revu

> Vigia silencioso pra PRs de review que te marcam no GitHub.

System tray app Linux que monitora GitHub em background e notifica
quando há Pull Requests aguardando seu review. Mantém histórico das
solicitações com status atualizado, acessível via tray e janela Wails.

Especificação completa em [`SPEC.md`](./SPEC.md).

## O que faz

- Poll periódico de `gh search prs --review-requested=@me` em background.
- Notificação D-Bus `transient` a cada PR novo ou re-request (título, repo,
  autor, diff `+/-`).
- Ícone na tray (SNI) com 3 estados: **idle** (tudo ok, nada pendente),
  **pending** (tem review aguardando), **error** (auth do `gh` falhou).
- Janela React com abas **Pendentes** / **Histórico**, badges de status,
  refresh manual e atualização em tempo real via eventos do poller.
- Config hot-reload: editar `~/.config/revu/config.json` → mudanças aplicam
  sem restart.

## Pré-requisitos

### Runtime

- Go 1.23+ (só pra build)
- [`gh` CLI](https://github.com/cli/cli) autenticado (`gh auth status` OK)

### Sistema

No Arch/CachyOS:

```bash
sudo pacman -S libayatana-appindicator webkit2gtk-4.1 imagemagick
# daemon de notificação (escolha um, se ainda não tiver):
sudo pacman -S mako
```

- `libayatana-appindicator` — publica o ícone via SNI (Waybar, KDE, GNOME
  com extensão AppIndicator consomem isso).
- `webkit2gtk-4.1` — runtime do Wails v2.
- `mako` / `dunst` / `swaync` — qualquer daemon freedesktop-compatível.
- `imagemagick` — só necessário se você quiser regerar os PNGs dos ícones
  (`task icons`).

### Build tools

```bash
sudo pacman -S go nodejs pnpm go-task
```

`go-task` é opcional mas os targets abaixo usam `Taskfile.yml`.

**Ambiente alvo:** Hyprland + Waybar (Wayland). Também roda em KDE
Plasma, Sway + Waybar e GNOME com extensão AppIndicator.

## Instalação

```bash
# 1. clone
git clone https://github.com/meopedevts/revu.git && cd revu

# 2. build (frontend React + binário Go com tags desktop)
task build

# 3. instala em ~/.local/bin e valida deps
task install
revu doctor         # 4 ✓ = tudo pronto
```

### Autostart no Hyprland

Adicione em `~/.config/hypr/hyprland.conf`:

```
exec-once = revu run
```

### Alternativa: systemd user

`~/.config/systemd/user/revu.service`:

```ini
[Unit]
Description=revu — GitHub PR review notifier
After=graphical-session.target

[Service]
ExecStart=%h/.local/bin/revu run
Restart=on-failure
RestartSec=10

[Install]
WantedBy=graphical-session.target
```

```bash
systemctl --user enable --now revu.service
```

## Waybar

Em `~/.config/waybar/config.jsonc` — adicione `tray` aos módulos:

```jsonc
{
  "modules-right": ["tray", "clock", "battery"],
  "tray": {
    "icon-size": 16,
    "spacing": 8
  }
}
```

Se o ícone não aparecer: veja [Troubleshooting](#troubleshooting).

## Configuração

Primeira execução semeia `~/.config/revu/config.json` com os defaults do
SPEC §7. Edite o arquivo e salve — `viper.WatchConfig` detecta e aplica
em tempo real.

| Campo                            | Default | O que faz                                                                 |
| -------------------------------- | ------- | ------------------------------------------------------------------------- |
| `polling_interval_seconds`       | `300`   | Intervalo entre polls do `gh search prs`. Mínimo 30; valores menores são coeridos pro default. |
| `notifications_enabled`          | `true`  | Liga/desliga notificações D-Bus em tempo real.                            |
| `notification_timeout_seconds`   | `5`     | Quanto tempo a notificação fica na tela. `0` = daemon decide.            |
| `status_refresh_every_n_ticks`   | `12`    | Reservado — revisa status de PRs históricos a cada N ticks (pós-MVP).     |
| `history_retention_days`         | `30`    | PRs não-OPEN mais antigos que isso são removidos no próximo Save. `0` = guarda pra sempre. |
| `start_hidden`                   | `true`  | Janela oculta no boot; `false` abre ao iniciar.                           |
| `window.width` / `window.height` | `480` / `640` | Tamanho inicial da janela Wails. Mínimo 240×240.                    |

> Algumas mudanças (dimensões da janela) só se aplicam no próximo `revu
> run`. Polling, notifications, retention são reconfigurados mid-flight.

## CLI

```
revu run         # inicia app (tray + poller + janela oculta)
revu version     # versão + commit + data de build
revu config      # imprime path do config.json
revu doctor      # valida deps runtime (gh, auth, D-Bus, libayatana)
```

## Dev

```bash
task build            # compila frontend + binário
task run              # build + revu run
task test             # go test -race ./...
task icons            # regera PNGs a partir dos SVGs em assets/icons/
task release          # build otimizado (strip + trimpath)
task install          # copia pra ~/.local/bin/revu
```

Sem `go-task`: `task build` equivale a
`cd frontend && pnpm run build && cd .. && go build -trimpath -tags desktop,production,webkit2_41 ./cmd/revu`.

## Troubleshooting

### Ícone não aparece no Waybar

1. `revu doctor` — se `✗ libayatana-appindicator`, instale o pacote.
2. Confirme que o módulo `tray` está nos `modules-right` do Waybar.
3. Veja o item registrado via D-Bus:

   ```bash
   busctl --user call org.kde.StatusNotifierWatcher \
     /StatusNotifierWatcher \
     org.freedesktop.DBus.Properties Get \
     ss org.kde.StatusNotifierWatcher RegisteredStatusNotifierItems
   ```

   Deve listar algo com `StatusNotifierItem-<pid>-1`.
4. Reinicie o Waybar (`pkill -SIGUSR2 waybar`).

### Notificação persiste no histórico do daemon

O revu envia com a hint `transient=true`. Se seu daemon ainda retém:

- **swaync**: adicione `match-app-name = "revu"` com `invisible = true`
  na regra correspondente em `~/.config/swaync/config.json`.
- **mako** / **dunst**: versões atuais respeitam `transient`. Se não,
  atualize o daemon.

### `gh auth status` falha mesmo logado

- `gh auth switch --user <login>` pra mudar conta ativa.
- Em org com SAML SSO: abra <https://github.com/settings/tokens>,
  clique `Configure SSO` no token do `gh`, autorize a org. Ou via CLI:

  ```bash
  gh auth refresh -h github.com -s repo,read:org
  ```

### Lista vazia mesmo com PRs pendentes

- `gh search prs --review-requested=@me --state=open` só retorna PRs com
  review request **formalmente aberta**. Uma vez que você submete
  qualquer review (aprovar, comentar, request changes), o PR sai do
  filtro — mesmo se o nome continua visível no "Reviewers" do GitHub UI.
- "Suggested reviewers" também aparece na mesma seção do UI mas **não**
  é uma review request real — só conta quando alguém confirma.

### Ícone fica grande / esquisito no Waybar

O tray usa PNG 32×32. Waybar redimensiona conforme `tray.icon-size`.
Valores ideais: 16, 18, 20. Se quiser versões dedicadas, regere com
`task icons` editando os tamanhos no `Taskfile.yml`.

## Licença

TBD.
