import { useQuery } from "@tanstack/react-query"

import { getPRDetails, getPRDiff } from "@/bridge"
import { queryKeys } from "@/lib/query/keys"
import type { PRFullDetails } from "@/lib/types"

interface UsePRDetailsResult {
  details: PRFullDetails | null
  diff: string | null
  diffError: Error | null
  loading: boolean
  // diffLoading isola o estado da request de diff pra que o consumer mostre
  // skeleton dedicado em vez de "diff vazio" enganoso enquanto details já
  // chegou mas diff ainda não.
  diffLoading: boolean
  error: string | null
  reload: () => Promise<void>
}

function errorMessage(err: unknown): string {
  if (err instanceof Error) return err.message
  return "erro ao carregar PR"
}

/**
 * usePRDetails busca metadata e diff em paralelo via react-query. A queryKey
 * inclui prID, então trocar de PR isola estado e resultados antigos por chave
 * de cache (sem cancelamento real da request em andamento — o Wails bridge
 * não expõe AbortSignal).
 *
 * O diff é tratado como best-effort: se falhar, `diff` fica null e o consumer
 * pode mostrar o fallback "abrir no GitHub" sem bloquear a tela inteira.
 * Apenas erro do `details` é fatal.
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

  return {
    details: detailsQ.data ?? null,
    diff: diffQ.data ?? null,
    diffError: diffQ.error ?? null,
    loading: detailsQ.isLoading || diffQ.isLoading,
    diffLoading: diffQ.isLoading,
    error: detailsQ.error ? errorMessage(detailsQ.error) : null,
    reload: async () => {
      await Promise.all([detailsQ.refetch(), diffQ.refetch()])
    },
  }
}
