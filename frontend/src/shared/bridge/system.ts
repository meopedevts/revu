import { requireBridge } from "./client"

export interface SystemBridge {
  ShowWindow(): Promise<void>
  HideWindow(): Promise<void>
  ClearHistory(): Promise<number>
}

export const showWindow = (): Promise<void> => requireBridge("ShowWindow")()

export const hideWindow = (): Promise<void> => requireBridge("HideWindow")()

export const clearHistory = (): Promise<number> =>
  requireBridge("ClearHistory")()
