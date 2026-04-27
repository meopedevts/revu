import { useCallback, useEffect, useRef, useState } from "react"

import { listHistoryPRs, listPendingPRs } from "@/src/lib/bridge"
import type { PRRecord } from "@/src/lib/types"
import { EventsOff, EventsOn } from "@/wailsjs/runtime/runtime"

interface PollCompletedEvent {
  kind: string
  at: string
  err?: string
}

interface UsePRsResult {
  pending: PRRecord[]
  history: PRRecord[]
  lastPollAt: Date | null
  lastPollErr: string | null
  loading: boolean
  reload: () => Promise<void>
}

/**
 * usePRs subscribes to poller events (pr:new, pr:status-changed,
 * poll:completed) and keeps the pending/history lists in sync. Falls back to
 * a slow interval in case events drop on the floor (Wails reconnect edge
 * cases). Returns a manual reload too — used by the Refresh button after it
 * triggers the poller.
 */
export function usePRs(): UsePRsResult {
  const [pending, setPending] = useState<PRRecord[]>([])
  const [history, setHistory] = useState<PRRecord[]>([])
  const [lastPollAt, setLastPollAt] = useState<Date | null>(null)
  const [lastPollErr, setLastPollErr] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  // Guard against overlapping reloads turning a burst of pr:new into a pile
  // of concurrent fetches.
  const inflight = useRef<Promise<void> | null>(null)

  const reload = useCallback(async (): Promise<void> => {
    if (inflight.current) {
      await inflight.current
      return
    }
    setLoading(true)
    const job = (async () => {
      try {
        const [p, h] = await Promise.all([listPendingPRs(), listHistoryPRs()])
        setPending(p ?? [])
        setHistory(h ?? [])
      } finally {
        setLoading(false)
        inflight.current = null
      }
    })()
    inflight.current = job
    return job
  }, [])

  useEffect(() => {
    void reload()

    EventsOn("pr:new", () => {
      void reload()
    })
    EventsOn("pr:status-changed", () => {
      void reload()
    })
    EventsOn("poll:completed", (raw: PollCompletedEvent | undefined) => {
      if (raw?.at) {
        setLastPollAt(new Date(raw.at))
      } else {
        setLastPollAt(new Date())
      }
      setLastPollErr(raw?.err ?? null)
    })

    // Safety net: if events are lost (dev reloads, Wails reconnect), re-read
    // the store every couple minutes regardless.
    const safety = setInterval(() => {
      void reload()
    }, 120_000)

    return () => {
      EventsOff("pr:new")
      EventsOff("pr:status-changed")
      EventsOff("poll:completed")
      clearInterval(safety)
    }
  }, [reload])

  return { pending, history, lastPollAt, lastPollErr, loading, reload }
}
