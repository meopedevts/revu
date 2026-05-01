import { zodResolver } from "@hookform/resolvers/zod"
import { useCallback, useEffect, useRef } from "react"
import { useForm, type FieldPath, type UseFormReturn } from "react-hook-form"
import { toast } from "sonner"

import { parseConfigValidationError } from "@/bridge"
import { useUpdateConfig } from "@/hooks/mutations/use-update-config"
import { useConfig } from "@/hooks/use-config"
import { configSchema } from "@/lib/schemas/config-schema"
import { DEFAULT_CONFIG, type AppConfig } from "@/lib/types"

export interface SettingsFormBag {
  form: UseFormReturn<AppConfig>
  loading: boolean
  saving: boolean
  loadError: Error | null
  submit: () => Promise<void>
  discard: () => Promise<void>
  restoreDefaults: () => void
}

export function useSettingsForm(): SettingsFormBag {
  const configQ = useConfig()
  const update = useUpdateConfig()
  const loading = configQ.isLoading
  const saving = update.isPending
  const loadError = configQ.error ?? null

  const form = useForm<AppConfig>({
    resolver: zodResolver(configSchema),
    defaultValues: DEFAULT_CONFIG,
    mode: "onChange",
  })

  // Só faz reset com dado real do backend (não com placeholderData), pra evitar
  // sobrescrever edição em curso quando o estado vai placeholder → success.
  useEffect(() => {
    if (configQ.data && !configQ.isPlaceholderData) {
      form.reset(configQ.data)
    }
  }, [configQ.data, configQ.isPlaceholderData, form])

  // Toast 1x por sessão de erro do load — useEffect com ref evita spam em
  // re-render. Reseta a sinalização quando o erro some.
  const toastedLoadErr = useRef(false)
  useEffect(() => {
    if (loadError && !toastedLoadErr.current) {
      toastedLoadErr.current = true
      toast.error(
        loadError instanceof Error
          ? `Falha ao carregar configurações: ${loadError.message}`
          : "Falha ao carregar configurações"
      )
    } else if (!loadError) {
      toastedLoadErr.current = false
    }
  }, [loadError])

  const submit = form.handleSubmit(async (values) => {
    try {
      await update.mutateAsync(values)
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
    }
  })

  const discard = useCallback(async () => {
    const fresh = await configQ.refetch()
    if (fresh.data) form.reset(fresh.data)
  }, [configQ, form])

  const restoreDefaults = useCallback(() => {
    form.reset(DEFAULT_CONFIG, { keepDefaultValues: true })
    form.setValue(
      "pollingIntervalSeconds",
      DEFAULT_CONFIG.pollingIntervalSeconds,
      { shouldDirty: true, shouldValidate: true }
    )
  }, [form])

  return { form, loading, saving, loadError, submit, discard, restoreDefaults }
}
