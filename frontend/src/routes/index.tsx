import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { useCallback, useEffect } from "react"

import { refreshNow } from "@/bridge"
import { MainHeader } from "@/components/main-header"
import { PRListTabs } from "@/components/pr-list-tabs"
import type { SettingsSection } from "@/components/settings/settings-sidebar"
import { useAcknowledgeTray, usePRs } from "@/hooks/use-prs"

export const Route = createFileRoute("/")({
  component: MainView,
})

function MainView() {
  const navigate = useNavigate()
  const {
    pending,
    history,
    lastPollAt,
    lastPollErr,
    loading,
    initialLoading,
    reload,
  } = usePRs()
  const ackTray = useAcknowledgeTray()

  // REV-54: ack on mount limpa o estado "novo desde última visualização"
  // sempre que o user abre a janela. Mutate é fire-and-forget — falha de
  // bridge não trava a render. mutate() é estável (referência do react-query).
  useEffect(() => {
    ackTray.mutate()
    // mount-only: ackTray.mutate é estável e re-rodar no efeito
    // sobrescreveria timestamps válidos.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const handleRefresh = useCallback(() => {
    // refreshNow() can throw synchronously (BridgeUnavailableError) before
    // returning a Promise — wrap in async IIFE so sync throws turn into
    // handled rejections instead of crashing the click handler.
    void (async () => {
      try {
        await refreshNow()
      } catch {
        // Bridge unavailable; reload below still picks up any state.
      }
      setTimeout(() => {
        void reload()
      }, 600)
    })()
  }, [reload])

  const openSettings = useCallback(
    (section: SettingsSection = "sync") => {
      void navigate({ to: "/settings/$section", params: { section } })
    },
    [navigate]
  )

  const openPR = useCallback(
    (prID: string) => {
      void navigate({ to: "/pr/$prId", params: { prId: prID } })
    },
    [navigate]
  )

  return (
    <div className="flex h-screen flex-col gap-3 bg-background p-3 text-foreground">
      <MainHeader
        pendingCount={pending.length}
        historyCount={history.length}
        lastPollAt={lastPollAt}
        lastPollErr={lastPollErr}
        loading={loading}
        onRefresh={handleRefresh}
        onOpenSettings={openSettings}
      />
      <PRListTabs
        pending={pending}
        history={history}
        onOpenPR={openPR}
        lastPollErr={lastPollErr}
        onRetry={handleRefresh}
        initialLoading={initialLoading}
      />
    </div>
  )
}
