import { useMutation, useQueryClient } from "@tanstack/react-query"

import { setActiveProfile } from "@/bridge"
import { queryKeys } from "@/lib/query/keys"

export function useSetActiveProfile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => setActiveProfile(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.profiles.all })
    },
  })
}
