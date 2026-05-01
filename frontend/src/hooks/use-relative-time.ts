import { useSyncExternalStore } from "react"

interface RelativeTimeOptions {
  idleLabel?: string
  nowLabel?: string
  prefix?: string
}

// Module-scoped store: 1 setInterval shared by all hook consumers.
// Avoids N PRCards spawning N timers. nowMs cached so getSnapshot stays
// referentially stable between renders within the same tick.
const subscribers = new Set<() => void>()
let intervalId: ReturnType<typeof setInterval> | null = null
let nowMs = Date.now()

function tick() {
  nowMs = Date.now()
  subscribers.forEach((cb) => cb())
}

function subscribe(cb: () => void) {
  if (subscribers.size === 0) {
    nowMs = Date.now()
  }
  subscribers.add(cb)
  intervalId ??= setInterval(tick, 60_000)
  return () => {
    subscribers.delete(cb)
    if (subscribers.size === 0 && intervalId !== null) {
      clearInterval(intervalId)
      intervalId = null
    }
  }
}

function getSnapshot() {
  return nowMs
}

export function useRelativeTime(
  iso: string | Date | null,
  opts: RelativeTimeOptions = {}
): string {
  const { idleLabel = "", nowLabel = "agora", prefix = "" } = opts
  const now = useSyncExternalStore(subscribe, getSnapshot, getSnapshot)

  if (iso === null) return idleLabel

  const then = iso instanceof Date ? iso.getTime() : new Date(iso).getTime()
  if (Number.isNaN(then)) return idleLabel

  const diff = Math.max(0, now - then)
  const s = Math.floor(diff / 1000)
  if (s < 10) return `${prefix}${nowLabel}`
  if (s < 60) return `${prefix}há ${s}s`
  const m = Math.floor(s / 60)
  if (m < 60) return `${prefix}há ${m}min`
  const h = Math.floor(m / 60)
  if (h < 24) return `${prefix}há ${h}h`
  const d = Math.floor(h / 24)
  return `${prefix}há ${d}d`
}
