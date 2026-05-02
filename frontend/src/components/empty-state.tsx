import { CloudOff, Inbox, ScrollText, UserPlus } from "lucide-react"
import type { LucideIcon } from "lucide-react"

import { Button } from "@/components/ui/button"

export type EmptyStateProps =
  | { variant: "pending" }
  | { variant: "history" }
  | { variant: "no-accounts"; onAddAccount: () => void }
  | { variant: "error-sync"; message: string; onRetry: () => void }

interface VariantConfig {
  icon: LucideIcon
  title: string
  hint: string
  decorate?: boolean
}

const VARIANTS: Record<EmptyStateProps["variant"], VariantConfig> = {
  pending: {
    icon: Inbox,
    title: "Tudo em dia ✦",
    hint: "Nenhum PR aguardando seu review agora.",
    decorate: true,
  },
  history: {
    icon: ScrollText,
    title: "Sem histórico ainda",
    hint: "PRs já reviewed ou fechados aparecem aqui.",
  },
  "no-accounts": {
    icon: UserPlus,
    title: "Conecte sua primeira conta",
    hint: "Adicione um perfil GitHub pra começar a receber PRs.",
  },
  "error-sync": {
    icon: CloudOff,
    title: "Falha ao sincronizar",
    hint: "",
  },
}

function Sparkle() {
  return (
    <svg
      viewBox="0 0 16 16"
      className="size-4 text-primary/50"
      aria-hidden="true"
    >
      <path
        d="M8 1l1.5 5L14 8l-4.5 1.5L8 15l-1.5-5L2 8l4.5-1.5z"
        fill="currentColor"
      />
    </svg>
  )
}

export function EmptyState(props: EmptyStateProps) {
  const config = VARIANTS[props.variant]
  const Icon = config.icon
  const hint = props.variant === "error-sync" ? props.message : config.hint

  return (
    <div
      role="status"
      aria-live="polite"
      className="flex h-48 flex-col items-center justify-center gap-3 text-center"
    >
      {config.decorate ? <Sparkle /> : null}
      <Icon
        className="size-12 text-muted-foreground opacity-70"
        aria-hidden="true"
      />
      <div className="text-sm font-semibold text-foreground">
        {config.title}
      </div>
      {hint ? (
        <div className="max-w-xs text-xs text-muted-foreground">{hint}</div>
      ) : null}
      {props.variant === "no-accounts" ? (
        <Button
          type="button"
          size="sm"
          variant="default"
          onClick={props.onAddAccount}
          className="mt-1"
        >
          Adicionar conta
        </Button>
      ) : null}
      {props.variant === "error-sync" ? (
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={props.onRetry}
          className="mt-1"
        >
          Tentar de novo
        </Button>
      ) : null}
    </div>
  )
}
