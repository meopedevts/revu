import type { AppConfig, PRRecord } from './types'

// Wails v2 injects Go bindings under window.go.<package>.<Struct>. This
// module isolates that runtime contract so the rest of the app does not
// depend on the generated wailsjs files (which require main.go at the
// module root — our entrypoint lives at cmd/revu/main.go instead).
interface WailsBridge {
  ListPendingPRs(): Promise<PRRecord[]>
  ListHistoryPRs(): Promise<PRRecord[]>
  OpenPRInBrowser(url: string): Promise<void>
  RefreshNow(): Promise<void>
  ShowWindow(): Promise<void>
  HideWindow(): Promise<void>
  GetConfig(): Promise<AppConfig>
  UpdateConfig(c: AppConfig): Promise<void>
  ClearHistory(): Promise<number>
}

declare global {
  interface Window {
    go?: {
      app?: {
        App?: WailsBridge
      }
    }
  }
}

function bridge(): WailsBridge | undefined {
  return window.go?.app?.App
}

export async function listPendingPRs(): Promise<PRRecord[]> {
  const b = bridge()
  if (!b) return []
  return b.ListPendingPRs()
}

export async function listHistoryPRs(): Promise<PRRecord[]> {
  const b = bridge()
  if (!b) return []
  return b.ListHistoryPRs()
}

export async function openPRInBrowser(url: string): Promise<void> {
  await bridge()?.OpenPRInBrowser(url)
}

export async function refreshNow(): Promise<void> {
  await bridge()?.RefreshNow()
}

export async function hideWindow(): Promise<void> {
  await bridge()?.HideWindow()
}

export async function getConfig(): Promise<AppConfig | null> {
  const b = bridge()
  if (!b) return null
  return b.GetConfig()
}

export async function updateConfig(c: AppConfig): Promise<void> {
  const b = bridge()
  if (!b) throw new Error('bridge unavailable')
  await b.UpdateConfig(c)
}

export async function clearHistory(): Promise<number> {
  const b = bridge()
  if (!b) return 0
  return b.ClearHistory()
}
