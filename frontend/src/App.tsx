import { useCallback, useEffect, useState } from 'react'
import { RefreshCw } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import {
  listHistoryPRs,
  listPendingPRs,
  openPRInBrowser,
  refreshNow,
} from '@/src/lib/bridge'
import { type PRRecord, type PRState, statusOf } from '@/src/lib/types'

type Tab = 'pending' | 'history'

const stateVariant: Record<
  PRState,
  'default' | 'secondary' | 'destructive' | 'outline'
> = {
  OPEN: 'default',
  DRAFT: 'secondary',
  MERGED: 'secondary',
  CLOSED: 'destructive',
}

function relTime(iso: string): string {
  if (!iso) return ''
  const then = new Date(iso).getTime()
  if (Number.isNaN(then)) return ''
  const diff = Date.now() - then
  const m = Math.floor(diff / 60_000)
  if (m < 1) return 'agora'
  if (m < 60) return `há ${m}min`
  const h = Math.floor(m / 60)
  if (h < 24) return `há ${h}h`
  const d = Math.floor(h / 24)
  return `há ${d}d`
}

function PRCard({ pr }: { pr: PRRecord }) {
  const status = statusOf(pr)
  return (
    <Card
      size="sm"
      className="cursor-pointer transition hover:ring-ring/60"
      onClick={() => openPRInBrowser(pr.url)}
    >
      <CardHeader>
        <CardTitle className="truncate">
          #{pr.number} · {pr.title}
        </CardTitle>
        <CardDescription className="truncate">
          {pr.repo} · @{pr.author} · +{pr.additions} −{pr.deletions}
        </CardDescription>
        <CardAction>
          <Badge variant={stateVariant[status]}>{status}</Badge>
        </CardAction>
      </CardHeader>
      <CardContent className="text-xs text-muted-foreground">
        visto {relTime(pr.last_seen_at)}
      </CardContent>
    </Card>
  )
}

function EmptyState({ tab }: { tab: Tab }) {
  return (
    <div className="flex h-40 flex-col items-center justify-center gap-1 text-sm text-muted-foreground">
      <div>
        {tab === 'pending'
          ? 'Nenhum PR aguardando review.'
          : 'Histórico vazio.'}
      </div>
      <div className="text-xs">Aguardando próximo poll.</div>
    </div>
  )
}

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
    const iv = setInterval(reload, 30_000)
    return () => clearInterval(iv)
  }, [reload])

  const handleRefresh = useCallback(async () => {
    await refreshNow()
    // Give the poller a beat to update the store.
    setTimeout(() => {
      void reload()
    }, 600)
  }, [reload])

  return (
    <div className="flex h-screen flex-col gap-3 bg-background p-3 text-foreground">
      <header className="flex items-center justify-between">
        <div>
          <div className="font-heading text-base font-medium">revu</div>
          <div className="text-xs text-muted-foreground">
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
        className="flex-1 overflow-hidden"
      >
        <TabsList>
          <TabsTrigger value="pending">Pendentes</TabsTrigger>
          <TabsTrigger value="history">Histórico</TabsTrigger>
        </TabsList>

        <TabsContent
          value="pending"
          className="flex-1 overflow-y-auto"
        >
          {pending.length === 0 ? (
            <EmptyState tab="pending" />
          ) : (
            <div className="flex flex-col gap-2">
              {pending.map((pr) => (
                <PRCard key={pr.id} pr={pr} />
              ))}
            </div>
          )}
        </TabsContent>

        <TabsContent
          value="history"
          className="flex-1 overflow-y-auto"
        >
          {history.length === 0 ? (
            <EmptyState tab="history" />
          ) : (
            <div className="flex flex-col gap-2">
              {history.map((pr) => (
                <PRCard key={pr.id} pr={pr} />
              ))}
            </div>
          )}
        </TabsContent>
      </Tabs>
    </div>
  )
}

export default App
