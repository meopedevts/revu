import { useCallback, useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { ArrowLeft, Loader2, RotateCcw, Trash2 } from 'lucide-react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { Slider } from '@/components/ui/slider'
import { Switch } from '@/components/ui/switch'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'

import {
  clearHistory,
  getConfig,
  refreshNow,
  updateConfig,
} from '@/src/lib/bridge'
import { configSchema } from '@/src/lib/schemas/config-schema'
import {
  DEFAULT_CONFIG,
  type AppConfig,
  type ConfigFieldError,
  type ConfigValidationError,
} from '@/src/lib/types'

interface SettingsViewProps {
  onBack: () => void
}

function formatInterval(seconds: number): string {
  if (seconds < 60) return `${seconds}s`
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  if (s === 0) return `${m} min`
  return `${m} min ${s}s`
}

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

export function SettingsView({ onBack }: SettingsViewProps) {
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [clearing, setClearing] = useState(false)

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

  const onSubmit = form.handleSubmit(async (values) => {
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

  const onDiscard = useCallback(() => {
    void loadConfig()
  }, [loadConfig])

  const onRestoreDefaults = useCallback(() => {
    form.reset(DEFAULT_CONFIG, { keepDefaultValues: true })
    // Force dirty so Salvar lights up even when on-disk config already equals
    // defaults — user still has to confirm the action.
    form.setValue('polling_interval_seconds', DEFAULT_CONFIG.polling_interval_seconds, {
      shouldDirty: true,
      shouldValidate: true,
    })
  }, [form])

  const onConfirmClearHistory = useCallback(async () => {
    setClearing(true)
    try {
      const n = await clearHistory()
      toast.success(`${n} PR${n === 1 ? '' : 's'} removido${n === 1 ? '' : 's'} do histórico`)
      // Nudge the poller so the history list in the main view refreshes.
      await refreshNow()
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : 'Falha ao limpar histórico')
    } finally {
      setClearing(false)
    }
  }, [])

  const pollingValue = form.watch('polling_interval_seconds')
  const canSave = form.formState.isDirty && form.formState.isValid && !saving

  if (loading) {
    return (
      <div className="flex h-screen items-center justify-center bg-background text-muted-foreground">
        <Loader2 className="size-4 animate-spin" />
      </div>
    )
  }

  return (
    <div className="flex h-screen flex-col bg-background text-foreground">
      <header className="flex items-center gap-2 border-b px-3 py-2">
        <Button
          size="sm"
          variant="ghost"
          onClick={onBack}
          aria-label="Voltar"
        >
          <ArrowLeft />
        </Button>
        <div className="font-heading text-base font-medium">Configurações</div>
      </header>

      <Form {...form}>
        <form
          onSubmit={onSubmit}
          className="flex flex-1 flex-col overflow-hidden"
        >
          <div className="flex-1 overflow-y-auto px-3 py-3">
            <div className="flex flex-col gap-3">
              <Card size="sm">
                <CardHeader>
                  <CardTitle>Polling</CardTitle>
                  <CardDescription>
                    Com que frequência o revu consulta o GitHub.
                  </CardDescription>
                </CardHeader>
                <CardContent className="flex flex-col gap-4">
                  <FormField
                    control={form.control}
                    name="polling_interval_seconds"
                    render={({ field }) => (
                      <FormItem>
                        <div className="flex items-center justify-between">
                          <FormLabel>Intervalo</FormLabel>
                          <span className="text-xs text-muted-foreground tabular-nums">
                            {formatInterval(pollingValue)}
                          </span>
                        </div>
                        <FormControl>
                          <Slider
                            min={30}
                            max={3600}
                            step={30}
                            value={[field.value]}
                            onValueChange={(v) => field.onChange(v[0])}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name="status_refresh_every_n_ticks"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Revalidar status a cada N polls</FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            min={1}
                            max={1000}
                            {...field}
                            onChange={(e) =>
                              field.onChange(e.target.valueAsNumber || 0)
                            }
                          />
                        </FormControl>
                        <FormDescription>
                          A cada N ticks, revu revisa status de PRs antigos (merged / closed) no histórico.
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </CardContent>
              </Card>

              <Card size="sm">
                <CardHeader>
                  <CardTitle>Notificações</CardTitle>
                  <CardDescription>
                    Alertas via D-Bus quando um review novo aparece.
                  </CardDescription>
                </CardHeader>
                <CardContent className="flex flex-col gap-4">
                  <FormField
                    control={form.control}
                    name="notifications_enabled"
                    render={({ field }) => (
                      <FormItem className="flex flex-row items-center justify-between gap-4">
                        <div className="space-y-0.5">
                          <FormLabel>Habilitar notificações</FormLabel>
                          <FormDescription>
                            Mostra toast do sistema quando há um review pendente novo.
                          </FormDescription>
                        </div>
                        <FormControl>
                          <Switch
                            checked={field.value}
                            onCheckedChange={field.onChange}
                          />
                        </FormControl>
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name="notification_timeout_seconds"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Timeout da notificação (s)</FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            min={1}
                            max={30}
                            {...field}
                            onChange={(e) =>
                              field.onChange(e.target.valueAsNumber || 0)
                            }
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </CardContent>
              </Card>

              <Card size="sm">
                <CardHeader>
                  <CardTitle>Histórico</CardTitle>
                  <CardDescription>
                    PRs não-OPEN são descartados após este prazo.
                  </CardDescription>
                </CardHeader>
                <CardContent className="flex flex-col gap-4">
                  <FormField
                    control={form.control}
                    name="history_retention_days"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Retenção (dias)</FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            min={1}
                            max={365}
                            {...field}
                            onChange={(e) =>
                              field.onChange(e.target.valueAsNumber || 0)
                            }
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <Separator />

                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        className="w-fit"
                        disabled={clearing}
                      >
                        <Trash2 />
                        Limpar finalizados agora
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>Limpar PRs finalizados?</AlertDialogTitle>
                        <AlertDialogDescription>
                          Remove os PRs do histórico que já foram encerrados (merged / closed). PRs ainda abertos ficam guardados para detectar novos pedidos de review.
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>Cancelar</AlertDialogCancel>
                        <AlertDialogAction onClick={onConfirmClearHistory}>
                          Confirmar
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </CardContent>
              </Card>

              <Card size="sm">
                <CardHeader>
                  <CardTitle>Aparência</CardTitle>
                  <CardDescription>
                    Comportamento inicial e tamanho da janela.
                  </CardDescription>
                </CardHeader>
                <CardContent className="flex flex-col gap-4">
                  <FormField
                    control={form.control}
                    name="start_hidden"
                    render={({ field }) => (
                      <FormItem className="flex flex-row items-center justify-between gap-4">
                        <div className="space-y-0.5">
                          <FormLabel>Iniciar minimizado</FormLabel>
                          <FormDescription>
                            Abre só no tray; janela aparece ao clicar em "Abrir".
                          </FormDescription>
                        </div>
                        <FormControl>
                          <Switch
                            checked={field.value}
                            onCheckedChange={field.onChange}
                          />
                        </FormControl>
                      </FormItem>
                    )}
                  />

                  <div className="grid grid-cols-2 gap-3">
                    <FormField
                      control={form.control}
                      name="window.width"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Largura (px)</FormLabel>
                          <FormControl>
                            <Input
                              type="number"
                              min={240}
                              max={3840}
                              {...field}
                              onChange={(e) =>
                                field.onChange(e.target.valueAsNumber || 0)
                              }
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="window.height"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Altura (px)</FormLabel>
                          <FormControl>
                            <Input
                              type="number"
                              min={240}
                              max={2160}
                              {...field}
                              onChange={(e) =>
                                field.onChange(e.target.valueAsNumber || 0)
                              }
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>
                </CardContent>
              </Card>
            </div>
          </div>

          <footer className="flex items-center justify-between gap-2 border-t bg-muted/40 px-3 py-2">
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={onRestoreDefaults}
            >
              <RotateCcw />
              Restaurar padrões
            </Button>
            <div className="flex items-center gap-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={onDiscard}
                disabled={!form.formState.isDirty || saving}
              >
                Descartar
              </Button>
              <Button type="submit" size="sm" disabled={!canSave}>
                {saving ? <Loader2 className="animate-spin" /> : null}
                Salvar
              </Button>
            </div>
          </footer>
        </form>
      </Form>
    </div>
  )
}
