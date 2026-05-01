import { useMutation, useQueryClient } from "@tanstack/react-query"

import { updateProfile } from "@/bridge"
import { queryKeys } from "@/lib/query/keys"
import type { ProfileUpdate } from "@/lib/types"

interface UpdateProfileArgs {
  id: string
  patch: ProfileUpdate
}

export function useUpdateProfile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, patch }: UpdateProfileArgs) => updateProfile(id, patch),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.profiles.all })
    },
  })
}
