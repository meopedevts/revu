import { Monitor, Moon, Sun } from "lucide-react"

import { PRCard } from "@/components/pr-card"
import { useSettingsFormContext } from "@/components/settings/settings-form-context"
import { Button } from "@/components/ui/button"
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import { Switch } from "@/components/ui/switch"
import { useTheme } from "@/lib/theme/theme-provider"
import { type PRRecord, type Theme } from "@/lib/types"
import { cn } from "@/lib/utils"

interface ThemeOption {
  value: Theme
  label: string
  hint: string
  icon: typeof Sun
}

const THEME_OPTIONS: readonly ThemeOption[] = [
  { value: "light", label: "Claro", hint: "Sempre claro", icon: Sun },
  { value: "dark", label: "Escuro", hint: "Sempre escuro", icon: Moon },
  { value: "auto", label: "Auto", hint: "Segue o sistema", icon: Monitor },
]

export function AppearanceSection() {
  const form = useSettingsFormContext()
  const { theme, setTheme } = useTheme()

  return (
    <div className="flex flex-col gap-6">
      <header className="flex flex-col gap-0.5">
        <h2 className="text-sm font-semibold text-foreground">Aparência</h2>
        <p className="mt-0.5 text-xs text-muted-foreground">
          Tema, comportamento inicial e tamanho da janela.
        </p>
      </header>

      <section className="flex flex-col gap-3">
        <div className="space-y-0.5">
          <div className="text-sm font-medium">Tema</div>
          <p className="text-xs text-muted-foreground">
            Aplicado imediatamente e persistido em config.json.
          </p>
        </div>

        <RadioGroup
          value={theme}
          onValueChange={(v) => void setTheme(v as Theme)}
          className="grid grid-cols-3 gap-2"
        >
          {THEME_OPTIONS.map(({ value, label, hint, icon: Icon }) => (
            <Label
              key={value}
              htmlFor={`theme-${value}`}
              className={cn(
                "flex cursor-pointer flex-col gap-2 rounded-md border bg-card p-3 text-left transition-colors duration-150",
                "hover:bg-muted/40",
                "has-[[data-state=checked]]:border-primary has-[[data-state=checked]]:bg-primary/5"
              )}
            >
              <div className="flex items-center justify-between">
                <Icon className="size-4 text-foreground" aria-hidden="true" />
                <RadioGroupItem
                  id={`theme-${value}`}
                  value={value}
                  className="size-3.5"
                />
              </div>
              <div className="flex flex-col gap-0.5">
                <span className="text-sm font-medium">{label}</span>
                <span className="text-[0.7rem] text-muted-foreground">
                  {hint}
                </span>
              </div>
            </Label>
          ))}
        </RadioGroup>

        <ThemePreview />
      </section>

      <FormField
        control={form.control}
        name="startHidden"
        render={({ field }) => (
          <FormItem className="flex flex-row items-center justify-between gap-4 rounded-md border p-4">
            <div className="space-y-0.5">
              <FormLabel>Iniciar minimizado</FormLabel>
              <FormDescription>
                Abre só no tray; janela aparece ao clicar em &quot;Abrir&quot;.
              </FormDescription>
            </div>
            <FormControl>
              <Switch checked={field.value} onCheckedChange={field.onChange} />
            </FormControl>
          </FormItem>
        )}
      />

      <div className="grid grid-cols-2 gap-3 rounded-md border p-4">
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
                  onChange={(e) => field.onChange(e.target.valueAsNumber || 0)}
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
                  onChange={(e) => field.onChange(e.target.valueAsNumber || 0)}
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

// Timestamps fixos garantem render determinístico do preview e evitam o lint
// de pureza (Date.now impuro dentro de hooks). Os valores não chegam ao
// backend — só alimentam a label "visto Xmin atrás".
const PREVIEW_PR: PRRecord = {
  id: "preview-1",
  number: 42,
  repo: "meopedevts/revu",
  title: "feat: dark mode com tokens OKLCH",
  author: "octocat",
  url: "",
  state: "OPEN",
  isDraft: false,
  additions: 248,
  deletions: 31,
  reviewPending: true,
  reviewState: "PENDING",
  branch: "feat/dark-mode",
  avatarUrl: "",
  firstSeenAt: "2026-05-02T17:30:00Z",
  lastSeenAt: "2026-05-02T17:50:00Z",
}

function previewNoop(): void {
  // preview é estático; handlers existem só pra satisfazer prop types.
}

function ThemePreview() {
  return (
    <div
      aria-label="Preview de tema"
      className="flex flex-col gap-3 rounded-lg border bg-background p-3"
    >
      <span className="text-[0.65rem] font-semibold tracking-wide text-muted-foreground uppercase">
        Preview
      </span>
      <div aria-hidden="true" className="pointer-events-none select-none">
        <PRCard pr={PREVIEW_PR} onOpen={previewNoop} isNew />
      </div>
      <div className="flex flex-wrap items-center gap-3 border-t pt-3">
        <Button size="sm">Primário</Button>
        <Button size="sm" variant="outline">
          Outline
        </Button>
        <div className="flex items-center gap-2">
          <Switch
            id="preview-switch"
            checked
            onCheckedChange={previewNoop}
            aria-label="Switch sample"
          />
          <Label htmlFor="preview-switch" className="text-xs">
            Toggle
          </Label>
        </div>
        <Input
          value="campo de exemplo"
          readOnly
          aria-label="Input sample"
          className="h-8 max-w-[10rem] text-xs"
        />
      </div>
    </div>
  )
}
