import { zodResolver } from "@hookform/resolvers/zod"
import { useCallback, useEffect } from "react"
import { useForm, type UseFormReturn } from "react-hook-form"
import { toast } from "sonner"

import { useUpdateConfig } from "@/hooks/mutations/use-update-config"
import { useConfig } from "@/hooks/use-config"
import { configSchema } from "@/lib/schemas/config-schema"
import {
  DEFAULT_CONFIG,
  type AppConfig,
  type ConfigValidationError,
} from "@/lib/types"

function isValidationError(err: unknown): err is ConfigValidationError {
  return (
    typeof err === "object" &&
    err !== null &&
    "errors" in err &&
    Array.isArray(err.errors)
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
  const configQ = useConfig()
  const update = useUpdateConfig()
  const loading = configQ.isLoading
  const saving = update.isPending

  const form = useForm<AppConfig>({
    resolver: zodResolver(configSchema),
    defaultValues: DEFAULT_CONFIG,
    mode: "onChange",
  })

  useEffect(() => {
    if (configQ.data) {
      form.reset(configQ.data)
    }
  }, [configQ.data, form])

  const submit = form.handleSubmit(async (values) => {
    try {
      await update.mutateAsync(values)
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
    }
  })

  const discard = useCallback(async () => {
    const fresh = await configQ.refetch()
    if (fresh.data) form.reset(fresh.data)
  }, [configQ, form])

  const restoreDefaults = useCallback(() => {
    form.reset(DEFAULT_CONFIG, { keepDefaultValues: true })
    form.setValue(
      "polling_interval_seconds",
      DEFAULT_CONFIG.polling_interval_seconds,
      { shouldDirty: true, shouldValidate: true }
    )
  }, [form])

  return { form, loading, saving, submit, discard, restoreDefaults }
}
