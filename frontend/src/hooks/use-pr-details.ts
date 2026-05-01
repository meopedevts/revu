import { useQuery } from "@tanstack/react-query"

import { getPRDetails, getPRDiff } from "@/bridge"
import { queryKeys } from "@/lib/query/keys"
import type { PRFullDetails } from "@/lib/types"

interface UsePRDetailsResult {
  details: PRFullDetails | null
  diff: string | null
  loading: boolean
  error: string | null
  reload: () => Promise<void>
}

function errorMessage(err: unknown): string {
  if (err instanceof Error) return err.message
  return "erro ao carregar PR"
}

/**
 * usePRDetails busca metadata e diff em paralelo via react-query. A queryKey
 * inclui prID, então trocar de PR cancela requests antigos automaticamente —
 * substitui o fetchId ref que ficava na implementação manual.
 */
export function usePRDetails(prID: string | null): UsePRDetailsResult {
  const enabled = !!prID
  const detailsQ = useQuery<PRFullDetails>({
    queryKey: queryKeys.prs.detail(prID ?? "__none__"),
    queryFn: () => getPRDetails(prID!),
    enabled,
  })
  const diffQ = useQuery<string>({
    queryKey: queryKeys.prs.diff(prID ?? "__none__"),
    queryFn: () => getPRDiff(prID!),
    enabled,
  })

  const error = detailsQ.error ?? diffQ.error
  return {
    details: detailsQ.data ?? null,
    diff: diffQ.data ?? null,
    loading: detailsQ.isLoading || diffQ.isLoading,
    error: error ? errorMessage(error) : null,
    reload: async () => {
      await Promise.all([detailsQ.refetch(), diffQ.refetch()])
    },
  }
}
