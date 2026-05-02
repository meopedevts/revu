# revu — Brand guide v1

Identidade visual do revu. Bloqueia Fase 2 do rebranding (tokens,
logo+favicon, tray icons, tipografia).

## 1. Mood

> **Calma, focada, "inbox zero".** Não é "playful saas". Dev tool que vive
> no canto da tela e respeita o foco do desenvolvedor.

### Atributos

| Sim | Não |
| --- | --- |
| Densidade média, hierarquia clara | Densidade extrema "Bloomberg" |
| Contraste forte, acentos pontuais | Cores saturadas em tudo |
| Tipografia variável, peso fino-a-médio | Fontes pesadas, "logos animados" |
| Bordas sutis, raios pequenos | Glassmorphism, gradientes fortes |
| Movimento mínimo, transições curtas | Animações lúdicas, micro-interações decorativas |

### Referências

| Produto | O que pegar |
| --- | --- |
| **Linear** | Hierarquia tipográfica, densidade, comando-paleta como UX central, paleta restrita |
| **Raycast** | Tray-first, tom de comando rápido, spacing consistente |
| **GitHub Desktop** | Familiaridade pra dev de PR, badges de estado discretos |
| **Stripe Dashboard** | Status colors precisos sem competir com primária |
| **Arc Browser** | Acento pontual num neutro forte (uso sóbrio de cor) |

### Tom de produto em uma frase

> "revu te avisa quando precisa, fica quieto quando não."

## 2. Paleta

OKLCH em todos os tokens. Light e dark explícitos. Status colors
escolhidos pra **não competir** com a primária (hue 288) — verde 145,
vermelho 27, âmbar 85, azul 240 ficam ortogonais no círculo de matiz.

### 2.1 Primária — Iris

Roxo-azulado equilibrado. Chroma médio (0.18) evita "neon". Hue 288
deixa próximo do indigo, longe do rosa, longe das status colors.

| Token | Light | Dark |
| --- | --- | --- |
| `--primary` | `oklch(0.58 0.18 288)` | `oklch(0.68 0.16 288)` |
| `--primary-foreground` | `oklch(0.98 0.01 288)` | `oklch(0.15 0.02 288)` |

### Escala (derivada do hue 288)

| Step | OKLCH | Uso |
| --- | --- | --- |
| 50 | `oklch(0.97 0.02 288)` | bg highlight muito leve |
| 100 | `oklch(0.94 0.05 288)` | bg highlight |
| 200 | `oklch(0.88 0.09 288)` | bordas de foco |
| 400 | `oklch(0.68 0.16 288)` | dark primary |
| 500 | `oklch(0.58 0.18 288)` | **light primary** |
| 600 | `oklch(0.52 0.19 288)` | hover light |
| 700 | `oklch(0.46 0.19 288)` | active light |
| 800 | `oklch(0.36 0.16 288)` | text on light bg para contraste WCAG AAA |
| 900 | `oklch(0.28 0.14 288)` | bg dark primary deep |

### 2.2 Acento — Coral

Pontual. Highlights raros (notificação nova, badge de "atenção"
não-bloqueante, CTA secundário diferenciado de primária). Nunca em
texto longo.

| Token | Light | Dark |
| --- | --- | --- |
| `--accent` | `oklch(0.72 0.14 30)` | `oklch(0.74 0.13 30)` |
| `--accent-foreground` | `oklch(0.20 0.02 30)` | `oklch(0.18 0.02 30)` |

### 2.3 Neutros

Cinza ligeiramente quente (hue 286) pra harmonizar com Iris.

| Token | Light | Dark |
| --- | --- | --- |
| `--background` | `oklch(1 0 0)` | `oklch(0.14 0.005 286)` |
| `--foreground` | `oklch(0.14 0.005 286)` | `oklch(0.985 0 0)` |
| `--card` | `oklch(1 0 0)` | `oklch(0.21 0.006 286)` |
| `--muted` | `oklch(0.97 0.001 286)` | `oklch(0.27 0.006 286)` |
| `--muted-foreground` | `oklch(0.55 0.016 286)` | `oklch(0.70 0.015 286)` |
| `--border` | `oklch(0.92 0.004 286)` | `oklch(1 0 0 / 10%)` |

### 2.4 Status colors (PR state)

Revisado pra: (a) separar `merged` da primária, (b) baixar saturação geral.

| Token | Light | Dark | Significado |
| --- | --- | --- | --- |
| `--status-open` | `oklch(0.65 0.16 145)` | `oklch(0.62 0.16 145)` | verde — aberto, ativo |
| `--status-draft` | `oklch(0.72 0.01 286)` | `oklch(0.36 0.006 286)` | cinza — rascunho |
| `--status-merged` | `oklch(0.55 0.20 308)` | `oklch(0.50 0.22 308)` | roxo quente — distinto da primária (288) |
| `--status-closed` | `oklch(0.60 0.20 27)` | `oklch(0.55 0.22 27)` | vermelho dessat — fechado sem merge |

### 2.5 Review state

| Token | Light | Dark | Significado |
| --- | --- | --- | --- |
| `--review-pending` | `oklch(0.78 0.14 85)` | `oklch(0.70 0.14 85)` | âmbar — review devido |
| `--review-approved` | `oklch(0.65 0.16 145)` | `oklch(0.62 0.16 145)` | verde — aprovado |
| `--review-changes` | `oklch(0.65 0.18 45)` | `oklch(0.60 0.18 45)` | laranja — bloqueia |
| `--review-commented` | `oklch(0.68 0.12 240)` | `oklch(0.58 0.12 240)` | azul — neutro |

### 2.6 Justificativa de matiz

Mapa simplificado dos hues escolhidos (graus no círculo OKLCH):

```
   145 (verde open/approved)
        |
   85 ──┼── 240 (azul commented)
   (âmbar pending)
        |
   45 (laranja changes)
        |
   27 (vermelho closed)
        |
   288 (Iris — primária)
        |
   308 (merged — variante quente)
        |
   30  (Coral — acento)
```

Distância entre `primary` (288) e `merged` (308) = 20°. Distância entre
`primary` (288) e nenhuma status color < 50°: roxo nunca confunde com
estado. Coral (30) cai entre `closed` (27) e `changes` (45), mas seu
chroma e lightness diferem o bastante: `closed` é mais escuro e
saturado, `coral` mais claro e dessat — leitura inequívoca.

### 2.7 Acessibilidade

Todos os pares `--*` / `--*-foreground` atendem **WCAG AA** (4.5:1 pra
texto normal, 3:1 pra texto grande). Testar com `culori`/contrast no
ESLint plugin antes de merge dos tokens (REV-49).

## 3. Tipografia

### 3.1 Família

| Uso | Família | Peso típico |
| --- | --- | --- |
| Sans (default) | **Inter Variable** | 400 / 500 / 600 |
| Mono | **JetBrains Mono Variable** | 400 / 500 |

JetBrains Mono escolhido sobre Geist Mono por:
- ligaduras opcionais úteis em diff (`->`, `=>`, `!=`, `==`)
- métricas próximas do que dev já vê em editor (familiaridade)
- variable font = um único arquivo, sem boot de múltiplos pesos

### 3.2 Escala (rem, base 16px)

| Token | Tamanho | Uso |
| --- | --- | --- |
| `text-xs` | 0.75 (12) | labels, badges, metadata |
| `text-sm` | 0.875 (14) | body default no tray e listas |
| `text-base` | 1.0 (16) | body em janela principal |
| `text-lg` | 1.125 (18) | títulos de card |
| `text-xl` | 1.25 (20) | títulos de seção |
| `text-2xl` | 1.5 (24) | header da janela |

### 3.3 Pesos e leading

- **Headings:** 600 (semibold), `leading-tight` (1.25)
- **Body:** 400, `leading-normal` (1.5)
- **UI compacta (badges, metadata):** 500, `leading-snug` (1.375)
- **Mono:** 400, `leading-normal`

### 3.4 Tracking

- `tracking-tight` em headings ≥ `text-xl`
- `tracking-wide uppercase` em labels de status (badges)
- Default em todo o resto

## 4. Tom de voz / copy

### 4.1 Princípios

1. **Direto.** Sem rodeios. Sem "your account has been successfully…".
2. **Português técnico.** Termos do domínio (PR, review, merge) ficam
   em inglês quando é jargão estabelecido.
3. **Verbo no imperativo nas ações.** "Aprovar", "Pular", "Abrir no
   GitHub" — não "Aprove esta requisição".
4. **Estados em uma palavra.** "Aberto", "Aprovado", "Rascunho".
5. **Erro = problema + caminho.** Não só "Falhou ao buscar PRs",
   sim "Falha ao buscar PRs. Confira `gh auth status`."

### 4.2 Vocabulário canônico

| Termo correto | Evitar |
| --- | --- |
| PR | "pull request", "solicitação" |
| Review | "revisão", "code review" (em UI) |
| Aprovar / Solicitar mudanças | "Approve" / "Request changes" |
| Histórico | "Log", "atividade" |
| Pendente | "Aguardando", "TODO" |
| Notificação transient | "Pop-up", "alerta" |

### 4.3 Voz em contextos

- **Notificação D-Bus:** uma frase, ≤ 80 chars. Ex: `omartelo solicitou
  review em revu#42`
- **Tray tooltip:** mais curto ainda. `revu — 3 pendentes`
- **Empty state:** afirmativo, não passivo. `Tudo em dia.` (não
  "Nenhum PR encontrado")
- **Erro recuperável:** problema + ação. `Sem rede. Reconectando…`
- **Erro fatal:** problema + diagnóstico. `gh CLI não autenticado.
  Rode \`gh auth login\`.`

## 5. Logo

3 concepts em `assets/brand/concepts/`:

1. **`r-monogram.svg`** — Monograma `r` minúsculo em Inter Display 600,
   recortado num quadrado roxo Iris. Direto, dev tool clássico.
2. **`r-eye.svg`** — Letra `r` cuja contra-forma sugere uma íris/olho.
   Liga "review" + "ver" + "Iris" (nome da paleta).
3. **`r-check.svg`** — `r` com cauda virando um check. Liga ato de
   revisar/aprovar.

### Regras

- Tamanho mínimo: **16×16** (favicon, tray). Em tray fica monocromático.
- Padding interno: 12.5% do bounding box.
- Em fundo claro: usa `--primary` (light Iris 500).
- Em fundo escuro: usa `--primary` (dark Iris 400).
- Tray icon (monocromático, 24×24): respeita preferência do sistema
  (`prefers-color-scheme`); fornecer SVG pra cada estado: idle,
  pending (com badge), error.

### Decisão final

> **Pendente.** Comparar 3 concepts num PR review e marcar 1 como
> oficial antes de fechar REV-48.

## 6. Versão

`v1` — Maio/2026. Próximas iterações ajustam tokens, não princípios.

Mudanças quebrando paleta exigem RFC e bump pra `v2`.
