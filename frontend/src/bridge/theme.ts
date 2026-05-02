import { VALID_THEMES } from "@/generated/constants"
import type { Theme } from "@/lib/types"

import { requireBridge } from "./client"

export interface ThemeBridge {
  GetTheme(): Promise<string>
  SetTheme(theme: string): Promise<void>
}

export async function getTheme(): Promise<Theme> {
  const raw = await requireBridge("GetTheme")()
  return (VALID_THEMES as readonly string[]).includes(raw)
    ? (raw as Theme)
    : "light"
}

export const setTheme = (theme: Theme): Promise<void> =>
  requireBridge("SetTheme")(theme)
