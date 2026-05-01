import { useEffect, useRef } from "react"

import { EventsOff, EventsOn } from "@/wailsjs/runtime/runtime"

export function useWailsEvent<T = unknown>(
  event: string,
  handler: (payload: T) => void
) {
  const handlerRef = useRef(handler)

  useEffect(() => {
    handlerRef.current = handler
  })

  useEffect(() => {
    EventsOn(event, (payload: T) => {
      handlerRef.current(payload)
    })
    return () => {
      EventsOff(event)
    }
  }, [event])
}
