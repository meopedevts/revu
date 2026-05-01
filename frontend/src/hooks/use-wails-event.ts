import { useEffect } from "react"

import { EventsOff, EventsOn } from "@/wailsjs/runtime/runtime"

/**
 * useWailsEvent assina um evento do runtime Wails e cancela no unmount.
 *
 * ⚠️ Limitação importante: `EventsOff(event)` do Wails desregistra **todos**
 * os handlers do evento — não apenas o registrado por esta instância. Por
 * isso este hook só é seguro em consumers únicos (ex.: useGlobalSubscriptions
 * montado uma vez no __root). Múltiplas instâncias do hook para o mesmo
 * evento vão se canibalizar no unmount.
 *
 * Se precisar de múltiplos consumers para o mesmo evento, introduza um
 * dispatcher singleton com `Set<handler>` antes de chamar este hook em mais
 * de um lugar.
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
