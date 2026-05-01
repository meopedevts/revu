import type { Theme } from "@/lib/types"

import { requireBridge } from "./client"

export interface ThemeBridge {
  GetTheme(): Promise<string>
  SetTheme(theme: string): Promise<void>
}

export async function getTheme(): Promise<Theme> {
  const raw = await requireBridge("GetTheme")()
  return raw === "dark" ? "dark" : "light"
}

export const setTheme = (theme: Theme): Promise<void> =>
  requireBridge("SetTheme")(theme)
