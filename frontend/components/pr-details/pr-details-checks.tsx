import { Check, ExternalLink, Loader2, X } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { cn } from "@/lib/utils"
import { openPRInBrowser } from "@/src/lib/bridge"
import type { StatusCheck } from "@/src/lib/types"

interface PRDetailsChecksProps {
  checks: StatusCheck[]
}

export function PRDetailsChecks({ checks }: PRDetailsChecksProps) {
  if (checks.length === 0) return null
  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {checks.map((check, i) => (
        <CheckBadge key={`${check.name}-${i}`} check={check} />
      ))}
    </div>
  )
}

function CheckBadge({ check }: { check: StatusCheck }) {
  const tone = toneClass(check)
  const Icon = iconFor(check)
  const content = (
    <Badge variant="outline" className={cn("gap-1 border-transparent", tone)}>
      <Icon className="size-3" aria-hidden="true" />
      <span className="truncate max-w-[160px]">{check.name}</span>
      {check.url && <ExternalLink className="size-2.5" aria-hidden="true" />}
    </Badge>
  )
  const wrapped = check.url ? (
    <a
      href={check.url}
      onClick={(e) => {
        e.preventDefault()
        void openPRInBrowser(check.url)
      }}
      className="no-underline"
    >
      {content}
    </a>
  ) : (
    content
  )
  return (
    <Tooltip>
      <TooltipTrigger asChild>{wrapped}</TooltipTrigger>
      <TooltipContent>
        {check.conclusion || check.status || "status desconhecido"}
      </TooltipContent>
    </Tooltip>
  )
}

function iconFor(check: StatusCheck) {
  const c = check.conclusion.toUpperCase()
  if (c === "SUCCESS") return Check
  if (c === "FAILURE" || c === "TIMED_OUT" || c === "CANCELLED") return X
  return Loader2
}

function toneClass(check: StatusCheck): string {
  const c = check.conclusion.toUpperCase()
  if (c === "SUCCESS")
    return "bg-status-open/15 text-status-open border-status-open/40"
  if (c === "FAILURE" || c === "TIMED_OUT" || c === "CANCELLED")
    return "bg-status-closed/15 text-status-closed border-status-closed/40"
  return "bg-muted text-muted-foreground"
}
