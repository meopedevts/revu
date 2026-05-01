import { zodResolver } from "@hookform/resolvers/zod"
import { useCallback, useEffect, useState } from "react"
import { useForm, type UseFormReturn } from "react-hook-form"
import { toast } from "sonner"

import { getConfig, updateConfig } from "@/lib/bridge"
import { toConfigValidationError } from "@/lib/bridge/mappers"
import type { ConfigValidationErrorWire } from "@/lib/bridge/wire"
import { configSchema } from "@/lib/schemas/config-schema"
import {
  DEFAULT_CONFIG,
  type AppConfig,
  type ConfigValidationError,
} from "@/lib/types"

function isValidationErrorWire(err: unknown): err is ConfigValidationErrorWire {
  return (
    typeof err === "object" &&
    err !== null &&
    "errors" in err &&
    Array.isArray(err.errors)
  )
}

// Backend ValidationError comes off the bridge with snake_case `field`
// values (matching internal/config JSON tags). Translate to the
// camelCase form paths used by react-hook-form before reporting back.
function parseBackendError(err: unknown): ConfigValidationError | null {
  if (isValidationErrorWire(err)) return toConfigValidationError(err)
  if (err instanceof Error) {
    try {
      const parsed = JSON.parse(err.message) as unknown
      if (isValidationErrorWire(parsed)) return toConfigValidationError(parsed)
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
    mode: "onChange",
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
      toast.success("Configurações salvas")
    } catch (err: unknown) {
      const ve = parseBackendError(err)
      if (ve) {
        for (const fe of ve.errors) {
          form.setError(fe.field as keyof AppConfig, {
            type: "backend",
            message: fe.msg,
          })
        }
        toast.error("Corrija os campos destacados")
      } else {
        toast.error(
          err instanceof Error ? err.message : "Falha ao salvar configurações"
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
      "pollingIntervalSeconds",
      DEFAULT_CONFIG.pollingIntervalSeconds,
      { shouldDirty: true, shouldValidate: true }
    )
  }, [form])

  return { form, loading, saving, submit, discard, restoreDefaults }
}
