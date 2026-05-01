import { useSettingsFormContext } from "@/components/settings/settings-form-context"
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import { Input } from "@/components/ui/input"
import { Slider } from "@/components/ui/slider"

function formatInterval(seconds: number): string {
  if (seconds < 60) return `${seconds}s`
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  if (s === 0) return `${m} min`
  return `${m} min ${s}s`
}

export function SyncSection() {
  const form = useSettingsFormContext()
  const pollingValue = form.watch("pollingIntervalSeconds")

  return (
    <div className="flex flex-col gap-4">
      <header className="flex flex-col gap-0.5">
        <h2 className="text-sm font-semibold">Sincronização</h2>
        <p className="text-xs text-muted-foreground">
          Com que frequência o revu consulta o GitHub.
        </p>
      </header>

      <FormField
        control={form.control}
        name="pollingIntervalSeconds"
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
        name="statusRefreshEveryNTicks"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Revalidar status a cada N polls</FormLabel>
            <FormControl>
              <Input
                type="number"
                min={1}
                max={1000}
                {...field}
                onChange={(e) => field.onChange(e.target.valueAsNumber || 0)}
              />
            </FormControl>
            <FormDescription>
              A cada N ticks, revu revisa status de PRs antigos (merged /
              closed) no histórico.
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />
    </div>
  )
}
