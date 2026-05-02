import { Trash2 } from "lucide-react"
import { useCallback, useState } from "react"
import { toast } from "sonner"

import { clearHistory, refreshNow } from "@/bridge"
import { useSettingsFormContext } from "@/components/settings/settings-form-context"
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
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import { Input } from "@/components/ui/input"
import { Separator } from "@/components/ui/separator"

export function HistorySection() {
  const form = useSettingsFormContext()
  const [clearing, setClearing] = useState(false)

  const onConfirmClearHistory = useCallback(async () => {
    setClearing(true)
    try {
      const n = await clearHistory()
      toast.success(
        `${n} PR${n === 1 ? "" : "s"} removido${n === 1 ? "" : "s"} do histórico`
      )
      await refreshNow()
    } catch (err: unknown) {
      toast.error(
        err instanceof Error ? err.message : "Falha ao limpar histórico"
      )
    } finally {
      setClearing(false)
    }
  }, [])

  return (
    <div className="flex flex-col gap-6">
      <header className="flex flex-col">
        <h2 className="text-sm font-semibold text-foreground">Histórico</h2>
        <p className="mt-0.5 text-xs text-muted-foreground">
          PRs não-OPEN são descartados após este prazo.
        </p>
      </header>

      <FormField
        control={form.control}
        name="historyRetentionDays"
        render={({ field }) => (
          <FormItem className="rounded-md border p-4">
            <FormLabel>Retenção (dias)</FormLabel>
            <FormControl>
              <Input
                type="number"
                min={1}
                max={365}
                {...field}
                onChange={(e) => field.onChange(e.target.valueAsNumber || 0)}
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
              Remove os PRs do histórico que já foram encerrados (merged /
              closed). PRs ainda abertos ficam guardados para detectar novos
              pedidos de review.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancelar</AlertDialogCancel>
            <AlertDialogAction onClick={() => void onConfirmClearHistory()}>
              Confirmar
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
