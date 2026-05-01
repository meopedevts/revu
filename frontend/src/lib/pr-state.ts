import type { PRFullDetails, PRState, ReviewState } from "@/lib/types"

export function derivePRState(details: PRFullDetails | null): PRState {
  if (!details) return "OPEN"
  if (details.state === "MERGED") return "MERGED"
  if (details.state === "CLOSED") return "CLOSED"
  if (details.isDraft) return "DRAFT"
  return "OPEN"
}

// Mirror what the PR card on the list shows. We have no viewer-id on the
// details payload, so pick the most actionable state across reviewers
// instead of trying to figure out which review is "mine".
export function deriveReviewState(details: PRFullDetails | null): ReviewState {
  if (!details) return "PENDING"
  const order = ["CHANGES_REQUESTED", "APPROVED", "COMMENTED"] as const
  for (const want of order) {
    if (details.reviews.some((r) => r.state === want)) return want
  }
  return "PENDING"
}

export function mergeableNow(details: PRFullDetails | null): boolean {
  if (!details) return false
  if (details.state !== "OPEN") return false
  if (details.isDraft) return false
  if (details.mergeable !== "MERGEABLE") return false
  const failing = details.statusChecks.some((c) => {
    const k = c.conclusion.toUpperCase()
    return k === "FAILURE" || k === "TIMED_OUT" || k === "CANCELLED"
  })
  return !failing
}

export function mergeBlockedReason(
  details: PRFullDetails | null
): string | null {
  if (!details) return null
  if (details.state === "MERGED") return "PR já foi merged"
  if (details.state === "CLOSED") return "PR fechado"
  if (details.isDraft) return "PR está em draft"
  if (details.mergeable === "CONFLICTING")
    return "conflitos — resolva pelo GitHub"
  if (details.mergeable === "UNKNOWN")
    return "GitHub ainda está checando se pode merge"
  const failing = details.statusChecks.some((c) => {
    const k = c.conclusion.toUpperCase()
    return k === "FAILURE" || k === "TIMED_OUT" || k === "CANCELLED"
  })
  if (failing) return "algum check falhou"
  return null
}
