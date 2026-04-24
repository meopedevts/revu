import { GitBranch } from 'lucide-react'

import type { PRFullDetails } from '@/src/lib/types'

interface PRDetailsMetaProps {
  details: PRFullDetails
}

export function PRDetailsMeta({ details }: PRDetailsMetaProps) {
  const mergeable = details.mergeable
  const mergeableLabel = mergeableText(mergeable)
  const mergeableTone = mergeableToneClass(mergeable)

  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
      <span>@{details.author}</span>
      <span aria-hidden="true">·</span>
      <span className="inline-flex items-center gap-1">
        <GitBranch className="size-3" aria-hidden="true" />
        <span className="font-mono">{details.baseRefName}</span>
        <span aria-hidden="true">←</span>
        <span className="font-mono">{details.headRefName}</span>
      </span>
      <span aria-hidden="true">·</span>
      <span>atualizado {relTime(details.updatedAt)}</span>
      {mergeableLabel && (
        <>
          <span aria-hidden="true">·</span>
          <span className={mergeableTone}>{mergeableLabel}</span>
        </>
      )}
    </div>
  )
}

function mergeableText(m: string): string {
  switch (m) {
    case 'MERGEABLE':
      return 'pronto pra merge'
    case 'CONFLICTING':
      return 'conflito'
    case 'UNKNOWN':
      return 'checando mergeabilidade'
    default:
      return ''
  }
}

function mergeableToneClass(m: string): string {
  if (m === 'CONFLICTING') return 'text-destructive'
  if (m === 'MERGEABLE') return 'text-status-open'
  return ''
}

function relTime(iso: string): string {
  if (!iso) return ''
  const then = new Date(iso).getTime()
  if (Number.isNaN(then)) return ''
  const diff = Date.now() - then
  const m = Math.floor(diff / 60_000)
  if (m < 1) return 'agora'
  if (m < 60) return `há ${m}min`
  const h = Math.floor(m / 60)
  if (h < 24) return `há ${h}h`
  const d = Math.floor(h / 24)
  return `há ${d}d`
}
