import { AlertCircle, RefreshCw, Settings } from "lucide-react"

import { MainHeaderProfileBadge } from "@/components/main-header-profile-badge"
import type { SettingsSection } from "@/components/settings/settings-sidebar"
import { Button } from "@/components/ui/button"
import { useRelativeTime } from "@/hooks/use-relative-time"

interface MainHeaderProps {
  pendingCount: number
  historyCount: number
  lastPollAt: Date | null
  lastPollErr: string | null
  loading: boolean
  onRefresh: () => void
  onOpenSettings: (section?: SettingsSection) => void
}

export function MainHeader({
  pendingCount,
  historyCount,
  lastPollAt,
  lastPollErr,
  loading,
  onRefresh,
  onOpenSettings,
}: MainHeaderProps) {
  const since = useRelativeTime(lastPollAt, {
    idleLabel: "ainda não atualizado",
    prefix: "atualizado ",
  })

  return (
    <header className="flex items-start justify-between gap-2">
      <div className="min-w-0">
        <div className="flex items-center gap-2">
          <div className="font-heading text-base font-medium">revu</div>
          <MainHeaderProfileBadge
            onOpenAccounts={() => onOpenSettings("accounts")}
          />
        </div>
        <div className="truncate text-xs text-muted-foreground">
          {pendingCount} pendente{pendingCount === 1 ? "" : "s"} ·{" "}
          {historyCount} no histórico · {since}
        </div>
        {lastPollErr && (
          <div className="mt-0.5 flex items-center gap-1 text-xs text-destructive">
            <AlertCircle className="size-3" aria-hidden="true" />
            último poll falhou: {lastPollErr}
          </div>
        )}
      </div>
      <div className="flex items-center gap-1">
        <Button
          size="sm"
          variant="ghost"
          onClick={() => onOpenSettings()}
          aria-label="Configurações"
        >
          <Settings />
        </Button>
        <Button
          size="sm"
          variant="outline"
          onClick={onRefresh}
          disabled={loading}
        >
          <RefreshCw data-icon="inline-start" />
          Atualizar
        </Button>
      </div>
    </header>
  )
}
