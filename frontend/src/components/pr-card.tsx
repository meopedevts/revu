import { ChevronRight } from "lucide-react"

import { ReviewBadge } from "@/components/review-badge"
import { StatusBadge } from "@/components/status-badge"
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { relTime } from "@/lib/format/time"
import { type PRRecord, reviewStateOf, statusOf } from "@/lib/types"

interface PRCardProps {
  pr: PRRecord
  onOpen: (prID: string) => void
}

export function PRCard({ pr, onOpen }: PRCardProps) {
  const status = statusOf(pr)
  const review = reviewStateOf(pr)

  function handleClick() {
    onOpen(pr.id)
  }

  function handleKey(e: React.KeyboardEvent<HTMLDivElement>) {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault()
      onOpen(pr.id)
    }
  }

  return (
    <Card
      size="sm"
      role="button"
      tabIndex={0}
      onClick={handleClick}
      onKeyDown={handleKey}
      className="cursor-pointer outline-none transition hover:ring-ring/60 focus-visible:ring-ring"
    >
      <CardHeader>
        <CardTitle className="flex items-center gap-2 truncate">
          <span className="text-muted-foreground">#{pr.number}</span>
          <span className="truncate">{pr.title}</span>
        </CardTitle>
        <CardDescription className="flex flex-wrap items-center gap-x-2 gap-y-1 truncate">
          <span className="truncate">{pr.repo}</span>
          <span aria-hidden="true">·</span>
          <span>@{pr.author}</span>
        </CardDescription>
        <CardAction className="flex flex-col items-end gap-1">
          <StatusBadge status={status} />
          <ReviewBadge state={review} />
        </CardAction>
      </CardHeader>
      <CardContent className="flex items-center justify-between gap-2 text-xs text-muted-foreground">
        <div className="flex items-center gap-2">
          <span className="font-mono text-emerald-600 dark:text-emerald-400">
            +{pr.additions}
          </span>
          <span className="font-mono text-rose-600 dark:text-rose-400">
            −{pr.deletions}
          </span>
        </div>
        <div className="flex items-center gap-1">
          <span>visto {relTime(pr.lastSeenAt)}</span>
          <ChevronRight className="size-3" aria-hidden="true" />
        </div>
      </CardContent>
    </Card>
  )
}
