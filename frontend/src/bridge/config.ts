import type { AppConfig } from "@/lib/types"

import { requireBridge } from "./client"

export interface ConfigBridge {
  GetConfig(): Promise<AppConfig>
  UpdateConfig(c: AppConfig): Promise<void>
}

export const getConfig = (): Promise<AppConfig> => requireBridge("GetConfig")()

export const updateConfig = (c: AppConfig): Promise<void> =>
  requireBridge("UpdateConfig")(c)
