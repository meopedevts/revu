import { useEffect } from "react"

import { EventsOff, EventsOn } from "@/wailsjs/runtime/runtime"

/**
 * useWailsEvent assina um evento do runtime Wails e cancela no unmount. O
 * handler é tipado por payload — runtime entrega `unknown` mas a maioria dos
 * eventos do backend tem shape estável (definidos em internal/poller/events.go).
 */
export function useWailsEvent<T = unknown>(
  event: string,
  handler: (payload: T) => void
): void {
  useEffect(() => {
    EventsOn(event, (raw: unknown) => {
      handler(raw as T)
    })
    return () => {
      EventsOff(event)
    }
  }, [event, handler])
}
