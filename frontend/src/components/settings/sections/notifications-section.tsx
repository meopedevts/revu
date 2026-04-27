import type { UseFormReturn } from 'react-hook-form'

import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import type { AppConfig } from '@/src/lib/types'

interface NotificationsSectionProps {
  form: UseFormReturn<AppConfig>
}

export function NotificationsSection({ form }: NotificationsSectionProps) {
  return (
    <div className="flex flex-col gap-4">
      <header className="flex flex-col gap-0.5">
        <h2 className="text-sm font-semibold">Notificações</h2>
        <p className="text-xs text-muted-foreground">
          Alertas via D-Bus quando um review novo aparece.
        </p>
      </header>

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
              <Switch checked={field.value} onCheckedChange={field.onChange} />
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
                onChange={(e) => field.onChange(e.target.valueAsNumber || 0)}
              />
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />
    </div>
  )
}
