import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import type { Review } from '@/src/lib/types'

interface PRDetailsReviewersProps {
  reviews: Review[]
}

// Collapse multiple reviews per author to the latest entry — gh returns
// every review, we only care about "where does this person stand right now".
export function PRDetailsReviewers({ reviews }: PRDetailsReviewersProps) {
  const latest = new Map<string, Review>()
  for (const r of reviews) {
    const prev = latest.get(r.author)
    if (!prev || new Date(r.submittedAt) > new Date(prev.submittedAt)) {
      latest.set(r.author, r)
    }
  }
  const entries = Array.from(latest.values())
  if (entries.length === 0) {
    return <div className="text-xs text-muted-foreground">sem reviewers</div>
  }
  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {entries.map((r) => (
        <ReviewerBadge key={r.author} review={r} />
      ))}
    </div>
  )
}

function ReviewerBadge({ review }: { review: Review }) {
  return (
    <Badge
      variant="outline"
      className={cn('gap-1.5 border-transparent', toneClass(review.state))}
    >
      <span className="inline-flex size-4 items-center justify-center rounded-full bg-muted-foreground/15 text-[10px] font-medium uppercase">
        {review.author.slice(0, 2)}
      </span>
      <span className="truncate">@{review.author}</span>
      <span className="text-[10px] opacity-80">{labelFor(review.state)}</span>
    </Badge>
  )
}

function labelFor(state: string): string {
  switch (state) {
    case 'APPROVED':
      return 'aprovou'
    case 'CHANGES_REQUESTED':
      return 'mudanças'
    case 'COMMENTED':
      return 'comentou'
    default:
      return 'pendente'
  }
}

function toneClass(state: string): string {
  switch (state) {
    case 'APPROVED':
      return 'bg-review-approved/15 text-review-approved'
    case 'CHANGES_REQUESTED':
      return 'bg-review-changes/15 text-review-changes'
    case 'COMMENTED':
      return 'bg-review-commented/15 text-review-commented'
    default:
      return 'bg-review-pending/15 text-review-pending'
  }
}
