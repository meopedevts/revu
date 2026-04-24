import type { PRRecord } from './types'

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
