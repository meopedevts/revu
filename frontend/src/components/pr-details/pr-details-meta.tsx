import { GitBranch } from "lucide-react"

import { useRelativeTime } from "@/hooks/use-relative-time"
import type { PRFullDetails } from "@/lib/types"

interface PRDetailsMetaProps {
  details: PRFullDetails
}

export function PRDetailsMeta({ details }: PRDetailsMetaProps) {
  const mergeable = details.mergeable
  const mergeableLabel = mergeableText(mergeable)
  const mergeableTone = mergeableToneClass(mergeable)
  const updated = useRelativeTime(details.updatedAt, { prefix: "atualizado " })

  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs leading-snug text-muted-foreground">
      <span>@{details.author}</span>
      <span aria-hidden="true">·</span>
      <span className="inline-flex items-center gap-1">
        <GitBranch className="size-3" aria-hidden="true" />
        <span className="font-mono">{details.baseRefName}</span>
        <span aria-hidden="true">←</span>
        <span className="font-mono">{details.headRefName}</span>
      </span>
      <span aria-hidden="true">·</span>
      <span>{updated}</span>
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
    case "MERGEABLE":
      return "pronto pra merge"
    case "CONFLICTING":
      return "conflito"
    case "UNKNOWN":
      return "checando mergeabilidade"
    default:
      return ""
  }
}

function mergeableToneClass(m: string): string {
  if (m === "CONFLICTING") return "text-destructive"
  if (m === "MERGEABLE") return "text-status-open"
  return ""
}
