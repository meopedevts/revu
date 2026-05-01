import { useQueryClient } from "@tanstack/react-query"
import { useCallback } from "react"

import { toProfile } from "@/bridge/mappers"
import type { ProfileWire } from "@/bridge/wire"
import { useWailsEvent } from "@/hooks/use-wails-event"
import { queryKeys } from "@/lib/query/keys"
import type { Profile } from "@/lib/types"

interface PollCompletedEvent {
  kind?: string
  at?: string
  err?: string
}

export interface PollMetaCache {
  at: Date | null
  err: string | null
}

/**
 * useGlobalSubscriptions assina eventos do poller e mantém o cache do
 * react-query em sincronia. Montar uma vez no componente raiz — duplicar
 * registra handlers extras e quebra a remoção via EventsOff.
 */
export function useGlobalSubscriptions(): void {
  const qc = useQueryClient()

  useWailsEvent(
    "pr:new",
    useCallback(() => {
      void qc.invalidateQueries({ queryKey: queryKeys.prs.all })
    }, [qc])
  )

  useWailsEvent(
    "pr:status-changed",
    useCallback(() => {
      void qc.invalidateQueries({ queryKey: queryKeys.prs.all })
    }, [qc])
  )

  useWailsEvent<PollCompletedEvent | undefined>(
    "poll:completed",
    useCallback(
      (raw) => {
        qc.setQueryData<PollMetaCache>(queryKeys.poll.meta, {
          at: raw?.at ? new Date(raw.at) : new Date(),
          err: raw?.err ?? null,
        })
      },
      [qc]
    )
  )

  // O evento vem do Go com chaves snake_case (ProfileWire) — convertemos
  // pra camelCase antes de gravar no cache pra alinhar com o que
  // listProfiles/getActiveProfile retornam via bridge.
  useWailsEvent<ProfileWire | undefined>(
    "profiles:active-changed",
    useCallback(
      (p) => {
        if (p) qc.setQueryData<Profile>(queryKeys.profiles.active, toProfile(p))
        else void qc.invalidateQueries({ queryKey: queryKeys.profiles.active })
      },
      [qc]
    )
  )
}
