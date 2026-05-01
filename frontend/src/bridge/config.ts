import type { AppConfig } from "@/lib/types"

import { requireBridge } from "./client"
import { fromAppConfig, toAppConfig } from "./mappers"
import type { AppConfigWire } from "./wire"

export interface ConfigBridge {
  GetConfig(): Promise<AppConfigWire>
  UpdateConfig(c: AppConfigWire): Promise<void>
}

export async function getConfig(): Promise<AppConfig> {
  return toAppConfig(await requireBridge("GetConfig")())
}

export const updateConfig = (c: AppConfig): Promise<void> =>
  requireBridge("UpdateConfig")(fromAppConfig(c))
