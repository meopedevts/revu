# revu — Especificação Técnica

> **Tagline:** um vigia silencioso pra PRs de review que te marcam no GitHub.

## 1. Visão Geral

Aplicativo de system tray para Linux que monitora o GitHub em background e notifica o usuário quando há Pull Requests aguardando review. Mantém histórico das solicitações com status atualizado.

**Problema:** interrupções frequentes ao longo do dia fazem com que solicitações de review passem despercebidas no chat e no GitHub.

**Solução:** notificações desktop proativas + histórico persistente acessível via tray.

---

## 2. Escopo

### MVP (v0.1.0)

- System tray com ícone e menu contextual
- Poller em background (goroutine com ticker)
- Detecção de novos PRs com review solicitado via `gh`
- Notificações desktop com título, autor, repo e diff (+/-)
- Janela com lista de PRs pendentes + histórico
- Badge de status (Open / Draft / Merged / Closed)
- Click no item abre PR no browser via `xdg-open`
- Configuração via arquivo JSON (Viper) com hot reload
- Health check do `gh auth status` no boot e periódico

### Out of Scope (futuras iterações)

- Configurações editáveis via UI (MVP edita JSON manualmente)
- Ações inline (approve / comment / request changes pelo app)
- Filtros avançados (por repo, org, autor, labels)
- Snooze / mute temporário
- Múltiplas contas GitHub
- Integração com GitLab / Bitbucket
- Suporte Windows / macOS

---

## 3. Stack

| Camada             | Tecnologia                                 |
| ------------------ | ------------------------------------------ |
| Runtime            | Go 1.22+                                   |
| Framework desktop  | Wails v2                                   |
| Frontend           | React + TypeScript + Vite                  |
| System tray        | `fyne.io/systray`                          |
| Notificações       | `github.com/esiqveland/notify` (D-Bus direto com suporte a hints)  |
| Config             | `spf13/viper`                              |
| CLI                | `spf13/cobra`                              |
| GitHub client      | wrapper sobre `gh` CLI via `os/exec`       |
| Persistência       | JSON em disco (`encoding/json`)            |
| Logging            | `log/slog` (stdlib)                        |

**Pré-requisito runtime:** `gh` CLI instalado e autenticado (`gh auth status` deve retornar OK).

**Dependências de sistema (Arch/CachyOS):**

- `libayatana-appindicator` — publicação do ícone via protocolo SNI (necessário pro Waybar)
- `webkit2gtk-4.1` — dependência do Wails v2
- Daemon de notificação freedesktop-compatível (`mako`, `dunst` ou `swaync`) — tipicamente já presente no setup Hyprland

**Ambiente alvo:** Hyprland (Wayland) + Waybar. Deve funcionar também em KDE Plasma, Sway + Waybar, e GNOME com extensão AppIndicator, já que todos consomem SNI.

---

## 4. Arquitetura

```
┌──────────────────────────────────────────────┐
│                  Wails App                   │
│  ┌────────────────┐    ┌──────────────────┐  │
│  │   React UI     │◄──►│  Go Bridge API   │  │
│  └────────────────┘    └────────┬─────────┘  │
└─────────────────────────────────┼────────────┘
                                  │
       ┌──────────────────────────┴──┐
       │                             │
┌──────▼──────┐   ┌────────────┐   ┌─▼──────────┐
│   Poller    │──►│   Store    │◄──│  Systray   │
│ (goroutine) │   │ (JSON fs)  │   │            │
└──────┬──────┘   └────────────┘   └────────────┘
       │
       ├──►┌────────────┐
       │   │ gh client  │
       │   └────────────┘
       │
       └──►┌────────────┐
           │  Notifier  │
           └────────────┘
```

---

## 5. Componentes

### 5.1 Poller

- Goroutine iniciada no boot
- `time.Ticker` com intervalo configurável (default **300s**)
- Respeita `context.Context` para shutdown limpo
- Em cada tick:
    1. Busca PRs com review solicitado ao usuário atual
    2. Para PRs novos (não vistos ou que voltaram a ter review pendente após terem sido reviewed), enriquece com `gh pr view` (additions/deletions) e dispara notificação
    3. Atualiza o store e persiste em disco
- **Status refresh do histórico:** a cada N ticks (configurável, default 12 = 1h com polling 5min), revisa status dos PRs históricos que já não estão com review pendente para manter badge atualizada

### 5.2 GitHub Client

Wrapper sobre `gh` via `os/exec`. Comandos principais:

```bash
# Health check
gh auth status

# Listar PRs com review solicitado
gh search prs \
  --review-requested=@me \
  --state=open \
  --json number,title,url,repository,author,isDraft,updatedAt \
  --limit 100

# Detalhes com diff (PRs novos e refresh de status)
gh pr view <url> --json additions,deletions,state,mergedAt,isDraft
```

Interface Go:

```go
type Client interface {
    AuthStatus(ctx context.Context) error
    ListReviewRequested(ctx context.Context) ([]PRSummary, error)
    GetPRDetails(ctx context.Context, url string) (*PRDetails, error)
}
```

### 5.3 Store

Arquivo JSON em `~/.config/revu/state.json`:

```json
{
  "prs": {
    "owner/repo#123": {
      "id": "owner/repo#123",
      "number": 123,
      "repo": "owner/repo",
      "title": "feat: add foo",
      "author": "meopedevts",
      "url": "https://github.com/owner/repo/pull/123",
      "state": "OPEN",
      "is_draft": false,
      "additions": 142,
      "deletions": 37,
      "review_pending": true,
      "first_seen_at": "2026-04-23T10:00:00Z",
      "last_seen_at": "2026-04-23T14:30:00Z",
      "last_notified_at": "2026-04-23T10:00:00Z"
    }
  },
  "last_poll_at": "2026-04-23T14:30:00Z"
}
```

**Detecção de re-request:** se `review_pending` era `false` e voltou a ser `true` no ciclo atual → conta como novo e notifica.

**Retention:** entradas com `state != OPEN` mais antigas que `history_retention_days` são removidas em cada save.

### 5.4 Notifier

Envia notificações via D-Bus direto pelo `org.freedesktop.Notifications`, com **hint `transient=true`** obrigatória em todas as chamadas. Isso impede que o daemon (`swaync`, `mako`, `dunst`) retenha a notificação no histórico — ela aparece, respeita o timeout e some de verdade. A fonte de verdade pra consulta posterior é o tray do app, não o centro de notificações do compositor.

**Ícone:** embutido no binário via `go:embed` (pasta `assets/`), salvo em diretório temporário no boot e referenciado por path absoluto no campo `AppIcon` da notificação.

```go
n := notify.Notification{
    AppName:       "revu",
    AppIcon:       iconPath, // path absoluto (extraído do embed)
    Summary:       fmt.Sprintf("Review solicitado · %s", pr.Repo),
    Body:          fmt.Sprintf("PR #%d — %s\npor @%s · +%d -%d",
                     pr.Number, pr.Title, pr.Author, pr.Additions, pr.Deletions),
    ExpireTimeout: 5 * time.Second,
    Hints: map[string]dbus.Variant{
        "transient": dbus.MakeVariant(true),
    },
}
notifier.SendNotification(n)
```

Formato final exibido:

```
Título: Review solicitado · owner/repo
Corpo:  PR #123 — feat: add foo
        por @autor · +142 -37
```

### 5.5 System Tray

**Protocolo:** usa **StatusNotifierItem (SNI)** via D-Bus, que é o protocolo consumido pelo módulo `tray` do Waybar. Não depende de XEmbed (legado X11), então funciona nativamente em Wayland/Hyprland.

**Config do Waybar** (`~/.config/waybar/config.jsonc`):

```jsonc
{
  "modules-right": ["tray", /* ... */],
  "tray": {
    "icon-size": 16,
    "spacing": 8
  }
}
```

**Validação via D-Bus** (debug): após iniciar o app, o ícone deve estar registrado em `org.kde.StatusNotifierWatcher`:

```bash
busctl --user call org.kde.StatusNotifierWatcher \
  /StatusNotifierWatcher \
  org.freedesktop.DBus.Properties Get \
  ss org.kde.StatusNotifierWatcher RegisteredStatusNotifierItems
```

**Comportamento:**

- Ícone reflete estado: **idle** / **tem pendências** (badge numérica ou cor diferente) / **erro de auth**
- Menu contextual:
    - "Abrir" → mostra janela principal
    - "Atualizar agora" → dispara poll imediato fora do ciclo
    - "Configurações" → abre `config.json` no editor padrão (`xdg-open`)
    - "Sair"
- Click esquerdo no ícone → toggle da janela

### 5.6 Janela Principal

- Duas abas: **Pendentes** (`review_pending=true`) e **Histórico** (demais)
- Cada item exibe: título, repo, autor, diff (+/-), badge de status, tempo relativo ("há 2h")
- Click no item → abre URL no browser (`xdg-open`)
- Botão "Refresh" dispara poll manual
- Indicador visual de "última atualização há X min"

---

## 6. Badges de Status

| Estado GitHub                   | Badge  | Cor sugerida |
| ------------------------------- | ------ | ------------ |
| `state=OPEN && !isDraft`        | OPEN   | Verde        |
| `state=OPEN && isDraft`         | DRAFT  | Cinza        |
| `state=CLOSED && mergedAt!=nil` | MERGED | Roxo         |
| `state=CLOSED && mergedAt==nil` | CLOSED | Vermelho     |

---

## 7. Configuração

Arquivo em `~/.config/revu/config.json`:

```json
{
  "polling_interval_seconds": 300,
  "notifications_enabled": true,
  "notification_timeout_seconds": 5,
  "status_refresh_every_n_ticks": 12,
  "history_retention_days": 30,
  "start_hidden": true,
  "window": {
    "width": 480,
    "height": 640
  }
}
```

Viper monitora alterações (`viper.WatchConfig`) para aplicação dinâmica sem restart. O poller reinicia o ticker quando `polling_interval_seconds` muda.

---

## 8. CLI, Boot & Instalação

### 8.1 Comandos

| Comando         | Descrição                                          |
| --------------- | -------------------------------------------------- |
| `revu run`      | Inicia o app (tray + poller + janela oculta)       |
| `revu version`  | Imprime versão e info de build                     |
| `revu config`   | Imprime path do arquivo de config                  |
| `revu doctor`   | Valida deps (`gh auth`, libs de sistema, D-Bus)    |

Implementado com `spf13/cobra`. `revu` sem argumentos exibe o help. O comando usado no autostart é `revu run`.

### 8.2 Fluxo de `revu run`

```
1. Carrega config (Viper)
2. Valida gh auth status
   ├─ OK   → prossegue
   └─ FAIL → tray em modo erro, poller não inicia
3. Carrega store do disco (se existir)
4. Inicializa systray
5. Inicia Wails (janela escondida por padrão, conforme start_hidden)
6. Inicia poller em goroutine
7. Executa primeiro poll imediato (não espera o primeiro tick)
```

### 8.3 Instalação & Autostart

Build e instalação local:

```bash
wails build
install -Dm755 build/bin/revu ~/.local/bin/revu
```

**Autostart via Hyprland** (recomendado pela simplicidade) — em `~/.config/hypr/hyprland.conf`:

```conf
exec-once = revu run
```

**Alternativa via systemd user** (permite restart automático em caso de crash) — em `~/.config/systemd/user/revu.service`:

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

Habilitar: `systemctl --user enable --now revu.service`

> Nota: em Hyprland iniciado direto de TTY, `graphical-session.target` pode não estar ativo. Nesses casos, preferir o `exec-once` do próprio Hyprland ou invocar via wrapper script que ative o target.

---

## 9. Tratamento de Erros

| Cenário                           | Comportamento                                                             |
| --------------------------------- | ------------------------------------------------------------------------- |
| `gh` retorna erro transitório     | Log + retry no próximo tick, sem notificar                                |
| `gh auth` expira em runtime       | Ícone da tray vai para estado de erro + notificação one-shot              |
| Falha de escrita no store         | Log + mantém estado em memória, tenta novamente no próximo tick           |
| Rate limit da API                 | Backoff exponencial (dobra intervalo até 30min, reseta após sucesso)      |
| `gh pr view` falha em PR novo     | Notifica sem os dados de diff (+/-) em vez de pular a notificação         |

---

## 10. Estrutura de Pastas

```
revu/
├── cmd/
│   └── revu/
│       └── main.go
├── internal/
│   ├── app/          # composição Wails
│   ├── cli/          # comandos cobra (root, run, version, config, doctor)
│   ├── config/       # Viper
│   ├── github/       # client gh
│   ├── notifier/     # beeep wrapper
│   ├── poller/       # goroutine + ticker
│   ├── store/        # JSON persistence
│   └── tray/         # systray
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   ├── hooks/
│   │   └── App.tsx
│   ├── package.json
│   └── vite.config.ts
├── build/
├── wails.json
├── go.mod
└── README.md
```

---

## 11. Fases de Implementação

### Fase 1 — Core (sem UI)

- Wrapper do `gh`, poller, store, notificações
- Main sem janela, logs no stdout
- **Critério de aceite:** ao abrir um PR solicitando review do usuário, notificação dispara com os dados corretos

### Fase 2 — Systray

- **Spike inicial:** tray "hello world" com `fyne.io/systray`, validar visibilidade no Waybar antes de seguir
- Integra `fyne.io/systray`
- Menu contextual funcional
- Ícone reflete estados (idle / pendente / erro)
- **Critério de aceite:** app roda em background sem janela aberta, ícone visível no Waybar, interação 100% via tray

### Fase 3 — Janela (Wails + React)

- Lista de PRs pendentes
- Abas pendentes/histórico
- Badges de status
- Click abre no browser
- **Critério de aceite:** usuário consulta PRs pendentes e histórico sem precisar abrir o GitHub

### Fase 4 — Polimento

- Config via Viper + hot reload
- Retention policy no histórico
- Ícones por estado na tray (SVG/PNG customizados)
- Build `.AppImage` _(descartado)_ → install em `~/.local/bin` + autostart via Hyprland `exec-once` (ver seção 8.3)

---

## 12. Riscos & Considerações

- **`gh` CLI é runtime dependency** — se o usuário deslogar (`gh auth logout`), o app precisa detectar. Mitigado pelo health check periódico.
- **System tray via SNI** — `fyne.io/systray` depende de `libayatana-appindicator` para publicar via SNI. Sem a lib instalada, o ícone não aparece no Waybar. Validação precoce recomendada na Fase 2 (rodar tray "hello world" e confirmar visibilidade antes de investir no menu completo).
- **Waybar recarrega tray ao reiniciar** — alterações no binário requerem reiniciar o app, mas não o Waybar. Se o ícone "sumir" após crash, checar se o app ainda está registrado via `busctl` antes de assumir bug no Waybar.
- **Hint `transient` em daemons antigos** — daemons muito antigos (libnotify < 0.7.9) podem ignorar a hint `transient` e ainda reter no histórico. Na prática, `swaync`/`mako`/`dunst` atuais respeitam. Se isso falhar em runtime, alternativa é configurar regra de ignore no próprio daemon (ex: `match-app-name = "revu"` no `swaync`).
- **Ícones embutidos** — assets ficam em `assets/` e são incluídos no binário via `go:embed`. Para trocar, substituir arquivo e rebuildar. Não há override via config no MVP.
- **Rate limit** do GitHub (5000 req/h autenticado) — com polling de 5min e `gh pr view` por PR novo, fica muito abaixo do limite.
- **Wails + Linux** — garantir que o binário final dependa só de `webkit2gtk` (já é padrão).
- **Clique em notificação** — ações em notificações via libnotify variam por daemon. MVP não depende disso (usuário abre a tray para ver a lista).

---

## 13. Evoluções Planejadas (backlog)

Lista viva para capturar as "ideias mirabolantes" — a serem priorizadas pós-MVP:

- [ ] Tela de configurações dentro da janela (em vez de editar JSON)
- [ ] Aprovar / comentar / request changes inline via `gh pr review`
- [ ] Filtros e grupos por repo/org
- [ ] Snooze de PRs específicos
- [ ] Resumo diário (ex: "você tem 3 PRs pendentes há mais de 24h")
- [ ] Empacotamento pra distribuição (AUR, AppImage ou Flatpak) — caso o app ganhe usuários além do autor
- [ ] Integração com calendário (bloqueio de foco silencia notificações)
- [ ] Estatísticas pessoais (tempo médio de review, PRs/semana)
- [ ] Suporte a múltiplas contas `gh` via perfis
