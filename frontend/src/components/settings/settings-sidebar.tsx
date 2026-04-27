import { Bell, Clock, Palette, RefreshCw, User } from "lucide-react"

import type { Profile } from "@/lib/types"
import { cn } from "@/lib/utils"

export const SETTINGS_SECTIONS = [
  "accounts",
  "sync",
  "notifications",
  "history",
  "appearance",
] as const

export type SettingsSection = (typeof SETTINGS_SECTIONS)[number]

export function isSettingsSection(s: string): s is SettingsSection {
  return (SETTINGS_SECTIONS as readonly string[]).includes(s)
}

interface SidebarItem {
  id: SettingsSection
  label: string
  icon: typeof User
}

const ITEMS: SidebarItem[] = [
  { id: "accounts", label: "Contas", icon: User },
  { id: "sync", label: "Sincronização", icon: RefreshCw },
  { id: "notifications", label: "Notificações", icon: Bell },
  { id: "history", label: "Histórico", icon: Clock },
  { id: "appearance", label: "Aparência", icon: Palette },
]

interface SettingsSidebarProps {
  active: SettingsSection
  onSelect: (section: SettingsSection) => void
  activeProfile: Profile | null
}

export function SettingsSidebar({
  active,
  onSelect,
  activeProfile,
}: SettingsSidebarProps) {
  return (
    <nav
      aria-label="Seções das configurações"
      className="flex w-40 flex-col gap-0.5 border-r bg-muted/30 p-2"
    >
      {ITEMS.map((item) => {
        const Icon = item.icon
        const isActive = active === item.id
        return (
          <button
            key={item.id}
            type="button"
            onClick={() => onSelect(item.id)}
            className={cn(
              "flex items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs font-medium",
              "transition-colors",
              isActive
                ? "bg-primary/10 text-primary"
                : "text-muted-foreground hover:bg-muted hover:text-foreground"
            )}
          >
            <Icon className="size-3.5" aria-hidden="true" />
            <span className="flex-1 truncate">{item.label}</span>
            {item.id === "accounts" && activeProfile ? (
              <span className="truncate text-[10px] font-normal text-muted-foreground">
                {activeProfile.name}
              </span>
            ) : null}
          </button>
        )
      })}
    </nav>
  )
}
