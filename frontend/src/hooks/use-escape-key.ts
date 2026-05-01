import { useEffect, useRef } from "react"

export function useEscapeKey(handler: () => void, enabled = true) {
  const handlerRef = useRef(handler)

  useEffect(() => {
    handlerRef.current = handler
  })

  useEffect(() => {
    if (!enabled) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== "Escape") return
      handlerRef.current()
    }
    window.addEventListener("keydown", onKey)
    return () => window.removeEventListener("keydown", onKey)
  }, [enabled])
}
