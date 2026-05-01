import { useQuery } from "@tanstack/react-query"

import { getConfig } from "@/bridge"
import { queryKeys } from "@/lib/query/keys"
import { DEFAULT_CONFIG, type AppConfig } from "@/lib/types"

export function useConfig() {
  return useQuery<AppConfig>({
    queryKey: queryKeys.config,
    queryFn: () => getConfig(),
    placeholderData: DEFAULT_CONFIG,
  })
}
