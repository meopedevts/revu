import type { ConfigBridge } from "./config"
import { BridgeMethodMissingError, BridgeUnavailableError } from "./errors"
import type { ProfilesBridge } from "./profiles"
import type { PRsBridge } from "./prs"
import type { SystemBridge } from "./system"
import type { ThemeBridge } from "./theme"

// Wails v2 injects Go bindings under window.go.<package>.<Struct>. This
// module isolates that runtime contract so the rest of the app does not
// depend on the generated wailsjs files (which require main.go at the
// module root — our entrypoint lives at cmd/revu/main.go instead).
export type WailsBridge = PRsBridge &
  ConfigBridge &
  ThemeBridge &
  ProfilesBridge &
  SystemBridge

declare global {
  interface Window {
    go?: {
      app?: {
        App?: Partial<WailsBridge>
      }
    }
  }
}

export function requireBridge<K extends keyof WailsBridge>(
  method: K
): NonNullable<WailsBridge[K]> {
  const b = window.go?.app?.App
  if (!b) throw new BridgeUnavailableError()
  const fn = b[method]
  if (typeof fn !== "function") {
    throw new BridgeMethodMissingError(String(method))
  }
  return fn as NonNullable<WailsBridge[K]>
}
