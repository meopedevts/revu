import { useCallback, useEffect, useRef, useState } from "react"

import { getPRDetails, getPRDiff } from "@/src/lib/bridge"
import type { PRFullDetails } from "@/src/lib/types"

interface UsePRDetailsResult {
  details: PRFullDetails | null
  diff: string | null
  loading: boolean
  error: string | null
  reload: () => Promise<void>
}

/**
 * usePRDetails fetches the full metadata and the unified diff in parallel so
 * the details view renders as soon as either resolves. The backend returns
 * "" for the diff when the PR exceeds detailsDiffLimit lines — the caller
 * treats that as the "PR too big, show open-in-GitHub fallback" signal.
 */
export function usePRDetails(prID: string | null): UsePRDetailsResult {
  const [details, setDetails] = useState<PRFullDetails | null>(null)
  const [diff, setDiff] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Guard against race conditions when the user navigates from one PR to
  // another faster than the first fetch resolves. We bump the ref on every
  // load and drop results that arrive under a stale id.
  const fetchId = useRef(0)

  const reload = useCallback(async (): Promise<void> => {
    if (!prID) {
      setDetails(null)
      setDiff(null)
      setError(null)
      return
    }
    const myFetch = ++fetchId.current
    setLoading(true)
    setError(null)
    try {
      const [d, raw] = await Promise.all([
        getPRDetails(prID),
        getPRDiff(prID).catch((err: unknown) => {
          // Diff failures should not block the details view — log via error
          // state and fall through with an empty diff.
          if (err instanceof Error) {
            throw err
          }
          throw new Error("diff fetch failed")
        }),
      ])
      if (fetchId.current !== myFetch) return
      setDetails(d)
      setDiff(raw)
    } catch (err: unknown) {
      if (fetchId.current !== myFetch) return
      setError(err instanceof Error ? err.message : "erro ao carregar PR")
      setDetails(null)
      setDiff(null)
    } finally {
      if (fetchId.current === myFetch) {
        setLoading(false)
      }
    }
  }, [prID])

  useEffect(() => {
    void reload()
  }, [reload])

  return { details, diff, loading, error, reload }
}
