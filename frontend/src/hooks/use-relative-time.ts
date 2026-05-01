import { useSyncExternalStore } from "react"

interface RelativeTimeOptions {
  idleLabel?: string
  nowLabel?: string
  prefix?: string
}

function subscribe(cb: () => void) {
  const id = setInterval(cb, 60_000)
  return () => clearInterval(id)
}

function getSnapshot() {
  return Math.floor(Date.now() / 60_000)
}

export function useRelativeTime(
  iso: string | Date | null,
  opts: RelativeTimeOptions = {}
): string {
  const { idleLabel = "", nowLabel = "agora", prefix = "" } = opts
  const minute = useSyncExternalStore(subscribe, getSnapshot, getSnapshot)

  if (iso === null) return idleLabel

  const then = iso instanceof Date ? iso.getTime() : new Date(iso).getTime()
  if (Number.isNaN(then)) return idleLabel

  const nowMs = minute * 60_000
  const diff = Math.max(0, nowMs - then)
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
