import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import type { ReviewState } from '@/src/lib/types'

const REVIEW_CLASS: Record<ReviewState, string> = {
  PENDING: 'bg-review-pending text-review-pending-foreground',
  APPROVED: 'bg-review-approved text-review-approved-foreground',
  CHANGES_REQUESTED: 'bg-review-changes text-review-changes-foreground',
  COMMENTED: 'bg-review-commented text-review-commented-foreground',
}

const REVIEW_LABEL: Record<ReviewState, string> = {
  PENDING: 'revisar',
  APPROVED: 'aprovado',
  CHANGES_REQUESTED: 'mudanças',
  COMMENTED: 'comentado',
}

interface ReviewBadgeProps {
  state: ReviewState
  className?: string
}

export function ReviewBadge({ state, className }: ReviewBadgeProps) {
  return (
    <Badge
      variant="outline"
      className={cn(
        'border-transparent uppercase tracking-wide',
        REVIEW_CLASS[state],
        className,
      )}
    >
      {REVIEW_LABEL[state]}
    </Badge>
  )
}
