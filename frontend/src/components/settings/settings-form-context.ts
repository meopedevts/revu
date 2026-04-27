import { createContext, useContext } from "react"
import type { UseFormReturn } from "react-hook-form"

import type { AppConfig } from "@/lib/types"

export const SettingsFormContext =
  createContext<UseFormReturn<AppConfig> | null>(null)

export function useSettingsFormContext(): UseFormReturn<AppConfig> {
  const ctx = useContext(SettingsFormContext)
  if (!ctx) {
    throw new Error(
      "useSettingsFormContext must be used inside SettingsFormContext.Provider"
    )
  }
  return ctx
}
