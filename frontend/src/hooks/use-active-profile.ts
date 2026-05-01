import { useQuery, useQueryClient } from "@tanstack/react-query"

import { getActiveProfile } from "@/bridge"
import { queryKeys } from "@/lib/query/keys"
import type { Profile } from "@/lib/types"

interface UseActiveProfileResult {
  profile: Profile | null
  refresh: () => Promise<void>
}

export function useActiveProfile(): UseActiveProfileResult {
  const qc = useQueryClient()
  const query = useQuery<Profile | null>({
    queryKey: queryKeys.profiles.active,
    queryFn: async () => {
      try {
        return await getActiveProfile()
      } catch {
        return null
      }
    },
  })

  return {
    profile: query.data ?? null,
    refresh: async () => {
      await qc.invalidateQueries({ queryKey: queryKeys.profiles.active })
    },
  }
}
