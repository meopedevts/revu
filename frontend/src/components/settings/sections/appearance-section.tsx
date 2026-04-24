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

interface AppearanceSectionProps {
  form: UseFormReturn<AppConfig>
}

export function AppearanceSection({ form }: AppearanceSectionProps) {
  return (
    <div className="flex flex-col gap-4">
      <header className="flex flex-col gap-0.5">
        <h2 className="text-sm font-semibold">Aparência</h2>
        <p className="text-xs text-muted-foreground">
          Comportamento inicial e tamanho da janela.
        </p>
      </header>

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
    </div>
  )
}
