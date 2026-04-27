import { z } from "zod"

import { CONFIG_BOUNDS, VALID_THEMES } from "@/shared/generated/constants"

// Bounds come from internal/config.Limits via cmd/gentsconst — single source
// of truth for both server-side validation and the Zod schema below.
// Client-side validation is for UX; the backend still enforces the same
// rules on UpdateConfig.
export const configSchema = z.object({
  polling_interval_seconds: z
    .number()
    .int("deve ser inteiro")
    .min(
      CONFIG_BOUNDS.pollingIntervalSeconds.min,
      `mínimo ${CONFIG_BOUNDS.pollingIntervalSeconds.min} segundos`
    )
    .max(
      CONFIG_BOUNDS.pollingIntervalSeconds.max,
      `máximo ${CONFIG_BOUNDS.pollingIntervalSeconds.max} segundos`
    ),
  notifications_enabled: z.boolean(),
  notification_timeout_seconds: z
    .number()
    .int("deve ser inteiro")
    .min(
      CONFIG_BOUNDS.notificationTimeoutSeconds.min,
      `mínimo ${CONFIG_BOUNDS.notificationTimeoutSeconds.min} segundo`
    )
    .max(
      CONFIG_BOUNDS.notificationTimeoutSeconds.max,
      `máximo ${CONFIG_BOUNDS.notificationTimeoutSeconds.max} segundos`
    ),
  status_refresh_every_n_ticks: z
    .number()
    .int("deve ser inteiro")
    .min(
      CONFIG_BOUNDS.statusRefreshEveryNTicks.min,
      `mínimo ${CONFIG_BOUNDS.statusRefreshEveryNTicks.min} tick`
    )
    .max(
      CONFIG_BOUNDS.statusRefreshEveryNTicks.max,
      `máximo ${CONFIG_BOUNDS.statusRefreshEveryNTicks.max} ticks`
    ),
  history_retention_days: z
    .number()
    .int("deve ser inteiro")
    .min(
      CONFIG_BOUNDS.historyRetentionDays.min,
      `mínimo ${CONFIG_BOUNDS.historyRetentionDays.min} dia`
    )
    .max(
      CONFIG_BOUNDS.historyRetentionDays.max,
      `máximo ${CONFIG_BOUNDS.historyRetentionDays.max} dias`
    ),
  start_hidden: z.boolean(),
  window: z.object({
    width: z
      .number()
      .int("deve ser inteiro")
      .min(
        CONFIG_BOUNDS.windowWidth.min,
        `mínimo ${CONFIG_BOUNDS.windowWidth.min} pixels`
      )
      .max(
        CONFIG_BOUNDS.windowWidth.max,
        `máximo ${CONFIG_BOUNDS.windowWidth.max} pixels`
      ),
    height: z
      .number()
      .int("deve ser inteiro")
      .min(
        CONFIG_BOUNDS.windowHeight.min,
        `mínimo ${CONFIG_BOUNDS.windowHeight.min} pixels`
      )
      .max(
        CONFIG_BOUNDS.windowHeight.max,
        `máximo ${CONFIG_BOUNDS.windowHeight.max} pixels`
      ),
  }),
  theme: z.enum(VALID_THEMES),
})
