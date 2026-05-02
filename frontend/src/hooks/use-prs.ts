import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import {
  acknowledgeTray,
  getTrayAcknowledgedAt,
  listHistoryPRs,
  listPendingPRs,
} from "@/bridge"
import { POLL_SAFETY_INTERVAL_MS } from "@/generated/constants"
import { queryKeys } from "@/lib/query/keys"
import type { PRRecord } from "@/lib/types"

export interface PollMeta {
  at: Date | null
  err: string | null
}

export function usePendingPRs() {
  return useQuery<PRRecord[]>({
    queryKey: queryKeys.prs.pending,
    queryFn: async () => (await listPendingPRs()) ?? [],
    refetchInterval: POLL_SAFETY_INTERVAL_MS,
  })
}

export function useHistoryPRs() {
  return useQuery<PRRecord[]>({
    queryKey: queryKeys.prs.history,
    queryFn: async () => (await listHistoryPRs()) ?? [],
    refetchInterval: POLL_SAFETY_INTERVAL_MS,
  })
}

export function usePollMeta(): PollMeta {
  const { data } = useQuery<PollMeta>({
    queryKey: queryKeys.poll.meta,
    queryFn: () => Promise.resolve({ at: null, err: null }),
    staleTime: Infinity,
    gcTime: Infinity,
  })
  return data ?? { at: null, err: null }
}

// useTrayAcknowledgedAt resolve o instante do último ack do tray. Backend
// devolve "" antes do primeiro ack — convertemos pra null. Comparado com
// pr.firstSeenAt no card pra decidir o dot "novo desde última visualização".
export function useTrayAcknowledgedAt(): Date | null {
  const { data } = useQuery<Date | null>({
    queryKey: queryKeys.tray.acknowledgedAt,
    queryFn: async () => {
      const iso = await getTrayAcknowledgedAt()
      return iso ? new Date(iso) : null
    },
    staleTime: Infinity,
  })
  return data ?? null
}

// useAcknowledgeTray dispara o ack persistido (REV-54). Disparado on-mount
// pela view principal pra que abrir a janela limpe o "novo". Invalida a query
// do ack pra re-render imediato dos cards perdendo o dot.
export function useAcknowledgeTray() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => acknowledgeTray(),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: queryKeys.tray.acknowledgedAt }),
  })
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
 * usePRs preserva o shape consumido por routes/index.tsx mas delega cache,
 * dedup e refetch ao react-query. Invalidação por eventos do poller fica em
 * useGlobalSubscriptions (montado no __root).
 */
export function usePRs(): UsePRsResult {
  const pendingQ = usePendingPRs()
  const historyQ = useHistoryPRs()
  const meta = usePollMeta()
  const qc = useQueryClient()

  return {
    pending: pendingQ.data ?? [],
    history: historyQ.data ?? [],
    lastPollAt: meta.at,
    lastPollErr: meta.err,
    loading: pendingQ.isFetching || historyQ.isFetching,
    reload: async () => {
      await qc.invalidateQueries({ queryKey: queryKeys.prs.all })
    },
  }
}
