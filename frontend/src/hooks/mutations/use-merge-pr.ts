import { useMutation, useQueryClient } from "@tanstack/react-query"

import { mergePR } from "@/bridge"
import { queryKeys } from "@/lib/query/keys"
import type { MergeMethod } from "@/lib/types"

interface MergePRArgs {
  prID: string
  method: MergeMethod
}

export function useMergePR() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ prID, method }: MergePRArgs) => mergePR(prID, method),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.prs.all })
    },
  })
}
