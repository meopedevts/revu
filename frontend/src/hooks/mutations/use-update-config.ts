import { useMutation, useQueryClient } from "@tanstack/react-query"

import { updateConfig } from "@/bridge"
import { queryKeys } from "@/lib/query/keys"
import type { AppConfig } from "@/lib/types"

export function useUpdateConfig() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (cfg: AppConfig) => updateConfig(cfg),
    onSuccess: (_data, cfg) => {
      qc.setQueryData<AppConfig>(queryKeys.config, cfg)
    },
  })
}
