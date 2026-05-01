import { useMutation, useQueryClient } from "@tanstack/react-query"

import { createProfile, type CreateProfileRequest } from "@/bridge"
import { queryKeys } from "@/lib/query/keys"

export function useCreateProfile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateProfileRequest) => createProfile(input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.profiles.all })
    },
  })
}
