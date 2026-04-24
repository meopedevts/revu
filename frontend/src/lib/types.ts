// Mirrors internal/store.PRRecord — field names match the Go json tags
// (snake_case is converted to camelCase by Wails' binding layer).
export interface PRRecord {
  id: string
  number: number
  repo: string
  title: string
  author: string
  url: string
  state: string
  is_draft: boolean
  additions: number
  deletions: number
  review_pending: boolean
  first_seen_at: string
  last_seen_at: string
  last_notified_at?: string
}

export type PRState = 'OPEN' | 'DRAFT' | 'MERGED' | 'CLOSED'

export function statusOf(pr: PRRecord): PRState {
  if (pr.state === 'MERGED') return 'MERGED'
  if (pr.state === 'CLOSED') return 'CLOSED'
  if (pr.is_draft) return 'DRAFT'
  return 'OPEN'
}
