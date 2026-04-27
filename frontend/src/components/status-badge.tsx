import { Badge } from "@/components/ui/badge"
import type { PRState } from "@/lib/types"
import { cn } from "@/lib/utils"

const STATUS_CLASS: Record<PRState, string> = {
  OPEN: "bg-status-open text-status-open-foreground",
  DRAFT: "bg-status-draft text-status-draft-foreground",
  MERGED: "bg-status-merged text-status-merged-foreground",
  CLOSED: "bg-status-closed text-status-closed-foreground",
}

interface StatusBadgeProps {
  status: PRState
  className?: string
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  return (
    <Badge
      variant="outline"
      className={cn(
        "border-transparent uppercase tracking-wide",
        STATUS_CLASS[status],
        className
      )}
    >
      {status}
    </Badge>
  )
}
