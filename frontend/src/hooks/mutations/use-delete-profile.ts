import { useMutation, useQueryClient } from "@tanstack/react-query"

import { deleteProfile } from "@/bridge"
import { queryKeys } from "@/lib/query/keys"

export function useDeleteProfile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteProfile(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.profiles.all })
    },
  })
}
