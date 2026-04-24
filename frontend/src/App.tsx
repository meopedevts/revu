import { useCallback, useEffect, useState } from 'react'
import { RefreshCw } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/ui/tabs'
import { EmptyState } from '@/components/empty-state'
import { PRCard } from '@/components/pr-card'

import {
  listHistoryPRs,
  listPendingPRs,
  openPRInBrowser,
  refreshNow,
} from '@/src/lib/bridge'
import type { PRRecord } from '@/src/lib/types'

type Tab = 'pending' | 'history'

function App() {
  const [pending, setPending] = useState<PRRecord[]>([])
  const [history, setHistory] = useState<PRRecord[]>([])
  const [loading, setLoading] = useState(false)
  const [tab, setTab] = useState<Tab>('pending')

  const reload = useCallback(async () => {
    setLoading(true)
    try {
      const [p, h] = await Promise.all([listPendingPRs(), listHistoryPRs()])
      setPending(p ?? [])
      setHistory(h ?? [])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void reload()
    // Periodic re-read of the store. Real reactive updates land in REV-8.
    const iv = setInterval(reload, 30_000)
    return () => clearInterval(iv)
  }, [reload])

  const handleRefresh = useCallback(async () => {
    await refreshNow()
    setTimeout(() => {
      void reload()
    }, 600)
  }, [reload])

  return (
    <div className="flex h-screen flex-col gap-3 bg-background p-3 text-foreground">
      <header className="flex items-center justify-between gap-2">
        <div className="min-w-0">
          <div className="font-heading text-base font-medium">revu</div>
          <div className="truncate text-xs text-muted-foreground">
            {pending.length} pendente{pending.length === 1 ? '' : 's'} ·{' '}
            {history.length} no histórico
          </div>
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
        value={tab}
        onValueChange={(v) => setTab(v as Tab)}
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
