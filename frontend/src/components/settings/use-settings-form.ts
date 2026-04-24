import { useCallback, useEffect, useState } from 'react'
import { useForm, type UseFormReturn } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { toast } from 'sonner'

import { getConfig, updateConfig } from '@/src/lib/bridge'
import { configSchema } from '@/src/lib/schemas/config-schema'
import {
  DEFAULT_CONFIG,
  type AppConfig,
  type ConfigFieldError,
  type ConfigValidationError,
} from '@/src/lib/types'

function isValidationError(err: unknown): err is ConfigValidationError {
  return (
    typeof err === 'object' &&
    err !== null &&
    'errors' in err &&
    Array.isArray((err as { errors: unknown }).errors)
  )
}

function parseBackendError(err: unknown): ConfigValidationError | null {
  if (isValidationError(err)) return err
  if (err instanceof Error) {
    try {
      const parsed = JSON.parse(err.message) as unknown
      if (isValidationError(parsed)) return parsed
    } catch {
      return null
    }
  }
  return null
}

export interface SettingsFormBag {
  form: UseFormReturn<AppConfig>
  loading: boolean
  saving: boolean
  submit: () => Promise<void>
  discard: () => Promise<void>
  restoreDefaults: () => void
}

export function useSettingsForm(): SettingsFormBag {
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  const form = useForm<AppConfig>({
    resolver: zodResolver(configSchema),
    defaultValues: DEFAULT_CONFIG,
    mode: 'onChange',
  })

  const loadConfig = useCallback(async (): Promise<void> => {
    const cfg = await getConfig()
    if (cfg) form.reset(cfg)
    setLoading(false)
  }, [form])

  useEffect(() => {
    void loadConfig()
  }, [loadConfig])

  const submit = form.handleSubmit(async (values) => {
    setSaving(true)
    try {
      await updateConfig(values)
      form.reset(values)
      toast.success('Configurações salvas')
    } catch (err: unknown) {
      const ve = parseBackendError(err)
      if (ve) {
        for (const fe of ve.errors as ConfigFieldError[]) {
          form.setError(fe.field as keyof AppConfig, {
            type: 'backend',
            message: fe.msg,
          })
        }
        toast.error('Corrija os campos destacados')
      } else {
        toast.error(
          err instanceof Error ? err.message : 'Falha ao salvar configurações',
        )
      }
    } finally {
      setSaving(false)
    }
  })

  const discard = loadConfig

  const restoreDefaults = useCallback(() => {
    form.reset(DEFAULT_CONFIG, { keepDefaultValues: true })
    form.setValue(
      'polling_interval_seconds',
      DEFAULT_CONFIG.polling_interval_seconds,
      { shouldDirty: true, shouldValidate: true },
    )
  }, [form])

  return { form, loading, saving, submit, discard, restoreDefaults }
}
