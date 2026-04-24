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
  review_state: string
  first_seen_at: string
  last_seen_at: string
  last_notified_at?: string
}

export type PRState = 'OPEN' | 'DRAFT' | 'MERGED' | 'CLOSED'

export type ReviewState =
  | 'PENDING'
  | 'APPROVED'
  | 'CHANGES_REQUESTED'
  | 'COMMENTED'

export function statusOf(pr: PRRecord): PRState {
  if (pr.state === 'MERGED') return 'MERGED'
  if (pr.state === 'CLOSED') return 'CLOSED'
  if (pr.is_draft) return 'DRAFT'
  return 'OPEN'
}

// reviewStateOf normalizes the raw review_state string coming off the bridge
// into the closed vocabulary the UI understands. Unknown values collapse to
// PENDING so the badge never disappears.
export function reviewStateOf(pr: PRRecord): ReviewState {
  switch (pr.review_state) {
    case 'APPROVED':
    case 'CHANGES_REQUESTED':
    case 'COMMENTED':
      return pr.review_state
    default:
      return 'PENDING'
  }
}

// Mirrors internal/config.Config 1:1 (snake_case JSON tags on the Go side).
export interface WindowConfig {
  width: number
  height: number
}

export interface AppConfig {
  polling_interval_seconds: number
  notifications_enabled: boolean
  notification_timeout_seconds: number
  status_refresh_every_n_ticks: number
  history_retention_days: number
  start_hidden: boolean
  window: WindowConfig
}

// Mirrors internal/config.FieldError.
export interface ConfigFieldError {
  field: string
  msg: string
}

// Mirrors internal/config.ValidationError.
export interface ConfigValidationError {
  errors: ConfigFieldError[]
}

// Defaults echo internal/config.Defaults — used by the settings view's
// "Restaurar padrões" button.
export const DEFAULT_CONFIG: AppConfig = {
  polling_interval_seconds: 300,
  notifications_enabled: true,
  notification_timeout_seconds: 5,
  status_refresh_every_n_ticks: 12,
  history_retention_days: 30,
  start_hidden: true,
  window: { width: 480, height: 640 },
}

// Mirrors internal/profiles.AuthMethod. The frontend never needs to look at
// keyring_ref directly — tokens stay on the Go side.
export type AuthMethod = 'gh-cli' | 'keyring'

// Mirrors internal/profiles.Profile — the JSON tags on Go flatten to
// snake_case, so we mirror them verbatim.
export interface Profile {
  id: string
  name: string
  auth_method: AuthMethod
  keyring_ref?: string
  github_username?: string
  is_active: boolean
  created_at: string
  last_validated_at?: string
}

export interface ProfileUpdate {
  name?: string
  auth_method?: AuthMethod
  token?: string
}
