// Time formatters used by the PR list and details views. Output is in
// pt-BR — these strings show up directly in the UI.

export function relTime(iso: string): string {
  if (!iso) return ""
  const then = new Date(iso).getTime()
  if (Number.isNaN(then)) return ""
  const diff = Date.now() - then
  const m = Math.floor(diff / 60_000)
  if (m < 1) return "agora"
  if (m < 60) return `há ${m}min`
  const h = Math.floor(m / 60)
  if (h < 24) return `há ${h}h`
  const d = Math.floor(h / 24)
  return `há ${d}d`
}

export function formatSince(d: Date | null): string {
  if (!d) return "ainda não atualizado"
  const diff = Date.now() - d.getTime()
  const s = Math.floor(diff / 1000)
  if (s < 10) return "atualizado agora"
  if (s < 60) return `atualizado há ${s}s`
  const m = Math.floor(s / 60)
  if (m < 60) return `atualizado há ${m}min`
  const h = Math.floor(m / 60)
  if (h < 24) return `atualizado há ${h}h`
  const days = Math.floor(h / 24)
  return `atualizado há ${days}d`
}
