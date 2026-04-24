import { z } from 'zod'

// Mirrors internal/config.validateStrict — keep the bounds in sync with the
// Go source of truth. Client-side validation is for UX; the backend still
// enforces the same rules on UpdateConfig.
export const configSchema = z.object({
  polling_interval_seconds: z
    .number()
    .int('deve ser inteiro')
    .min(30, 'mínimo 30 segundos')
    .max(3600, 'máximo 3600 segundos'),
  notifications_enabled: z.boolean(),
  notification_timeout_seconds: z
    .number()
    .int('deve ser inteiro')
    .min(1, 'mínimo 1 segundo')
    .max(30, 'máximo 30 segundos'),
  status_refresh_every_n_ticks: z
    .number()
    .int('deve ser inteiro')
    .min(1, 'mínimo 1 tick')
    .max(1000, 'máximo 1000 ticks'),
  history_retention_days: z
    .number()
    .int('deve ser inteiro')
    .min(1, 'mínimo 1 dia')
    .max(365, 'máximo 365 dias'),
  start_hidden: z.boolean(),
  window: z.object({
    width: z
      .number()
      .int('deve ser inteiro')
      .min(240, 'mínimo 240 pixels')
      .max(3840, 'máximo 3840 pixels'),
    height: z
      .number()
      .int('deve ser inteiro')
      .min(240, 'mínimo 240 pixels')
      .max(2160, 'máximo 2160 pixels'),
  }),
})
