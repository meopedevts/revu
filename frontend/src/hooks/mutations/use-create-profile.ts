import { useMutation, useQueryClient } from "@tanstack/react-query"

import { createProfile, type CreateProfileInput } from "@/bridge"
import { queryKeys } from "@/lib/query/keys"

export function useCreateProfile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateProfileInput) => createProfile(input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.profiles.all })
    },
  })
}
