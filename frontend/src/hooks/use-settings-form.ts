import { zodResolver } from "@hookform/resolvers/zod"
import { useCallback, useEffect, useState } from "react"
import { useForm, type FieldPath, type UseFormReturn } from "react-hook-form"
import { toast } from "sonner"

import { getConfig, parseConfigValidationError, updateConfig } from "@/bridge"
import { configSchema } from "@/lib/schemas/config-schema"
import { DEFAULT_CONFIG, type AppConfig } from "@/lib/types"

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
    try {
      form.reset(await getConfig())
    } catch {
      // Bridge unavailable (smoke build / preview). Form keeps
      // DEFAULT_CONFIG; REV-33 query layer surfaces errors uniformly.
    } finally {
      setLoading(false)
    }
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
      const ve = parseConfigValidationError(err)
      if (ve) {
        for (const fe of ve.errors) {
          form.setError(fe.field as FieldPath<AppConfig>, {
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
