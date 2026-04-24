import { useCallback } from 'react'
import { AlertCircle, RefreshCw } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/ui/tabs'
import { EmptyState } from '@/components/empty-state'
import { PRCard } from '@/components/pr-card'

import { openPRInBrowser, refreshNow } from '@/src/lib/bridge'
import { usePRs } from '@/src/lib/hooks/use-prs'

function formatSince(d: Date | null): string {
  if (!d) return 'ainda não atualizado'
  const diff = Date.now() - d.getTime()
  const s = Math.floor(diff / 1000)
  if (s < 10) return 'atualizado agora'
  if (s < 60) return `atualizado há ${s}s`
  const m = Math.floor(s / 60)
  if (m < 60) return `atualizado há ${m}min`
  const h = Math.floor(m / 60)
  if (h < 24) return `atualizado há ${h}h`
  const days = Math.floor(h / 24)
  return `atualizado há ${days}d`
}

function App() {
  const { pending, history, lastPollAt, lastPollErr, loading, reload } = usePRs()

  const handleRefresh = useCallback(async () => {
    await refreshNow()
    // poll:completed will update lastPollAt; reload() covers the edge case
    // where the user hit Refresh before OnStartup wired the event bus.
    setTimeout(() => {
      void reload()
    }, 600)
  }, [reload])

  return (
    <div className="flex h-screen flex-col gap-3 bg-background p-3 text-foreground">
      <header className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <div className="font-heading text-base font-medium">revu</div>
          <div className="truncate text-xs text-muted-foreground">
            {pending.length} pendente{pending.length === 1 ? '' : 's'} ·{' '}
            {history.length} no histórico · {formatSince(lastPollAt)}
          </div>
          {lastPollErr && (
            <div className="mt-0.5 flex items-center gap-1 text-xs text-destructive">
              <AlertCircle className="size-3" aria-hidden="true" />
              último poll falhou: {lastPollErr}
            </div>
          )}
        </div>
        <Button
          size="sm"
          variant="outline"
          onClick={handleRefresh}
          disabled={loading}
        >
          <RefreshCw data-icon="inline-start" />
          Atualizar
        </Button>
      </header>

      <Tabs
        defaultValue="pending"
        className="flex flex-1 flex-col overflow-hidden"
      >
        <TabsList>
          <TabsTrigger value="pending">
            Pendentes
            {pending.length > 0 && (
              <span className="ml-1.5 rounded-full bg-primary/15 px-1.5 text-[10px] font-medium text-primary">
                {pending.length}
              </span>
            )}
          </TabsTrigger>
          <TabsTrigger value="history">Histórico</TabsTrigger>
        </TabsList>

        <TabsContent value="pending" className="flex-1 overflow-y-auto">
          {pending.length === 0 ? (
            <EmptyState variant="pending" />
          ) : (
            <div className="flex flex-col gap-2">
              {pending.map((pr) => (
                <PRCard key={pr.id} pr={pr} onOpen={openPRInBrowser} />
              ))}
            </div>
          )}
        </TabsContent>

        <TabsContent value="history" className="flex-1 overflow-y-auto">
          {history.length === 0 ? (
            <EmptyState variant="history" />
          ) : (
            <div className="flex flex-col gap-2">
              {history.map((pr) => (
                <PRCard key={pr.id} pr={pr} onOpen={openPRInBrowser} />
              ))}
            </div>
          )}
        </TabsContent>
      </Tabs>
    </div>
  )
}

export default App
