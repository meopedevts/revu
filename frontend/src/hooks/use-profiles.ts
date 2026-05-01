import { useQuery } from "@tanstack/react-query"

import { listProfiles } from "@/bridge"
import { queryKeys } from "@/lib/query/keys"
import type { Profile } from "@/lib/types"

export function useProfiles() {
  return useQuery<Profile[]>({
    queryKey: queryKeys.profiles.list,
    queryFn: async () => (await listProfiles()) ?? [],
  })
}
