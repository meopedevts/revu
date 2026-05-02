// Public TypeScript types consumed by the app. All camelCase — the
// snake_case JSON tags the Go backend uses are confined to the bridge
// boundary (`src/bridge/wire.ts` + `src/bridge/mappers.ts`).

import type { Theme } from "@/generated/constants"

export type { Theme } from "@/generated/constants"
export { DEFAULT_CONFIG } from "@/generated/constants"

export interface PRRecord {
  id: string
  number: number
  repo: string
  title: string
  author: string
  url: string
  state: string
  isDraft: boolean
  additions: number
  deletions: number
  reviewPending: boolean
  reviewState: string
  branch: string
  avatarUrl: string
  firstSeenAt: string
  lastSeenAt: string
  lastNotifiedAt?: string
}

export type PRState = "OPEN" | "DRAFT" | "MERGED" | "CLOSED"

export type ReviewState =
  | "PENDING"
  | "APPROVED"
  | "CHANGES_REQUESTED"
  | "COMMENTED"

export function statusOf(pr: PRRecord): PRState {
  if (pr.state === "MERGED") return "MERGED"
  if (pr.state === "CLOSED") return "CLOSED"
  if (pr.isDraft) return "DRAFT"
  return "OPEN"
}

// reviewStateOf normalizes the raw reviewState string coming off the
// bridge into the closed vocabulary the UI understands. Unknown values
// collapse to PENDING so the badge never disappears.
export function reviewStateOf(pr: PRRecord): ReviewState {
  switch (pr.reviewState) {
    case "APPROVED":
    case "CHANGES_REQUESTED":
    case "COMMENTED":
      return pr.reviewState
    default:
      return "PENDING"
  }
}

export interface WindowConfig {
  width: number
  height: number
}

export interface AppConfig {
  pollingIntervalSeconds: number
  notificationsEnabled: boolean
  notificationTimeoutSeconds: number
  notificationCooldownMinutes: number
  statusRefreshEveryNTicks: number
  historyRetentionDays: number
  startHidden: boolean
  window: WindowConfig
  theme: Theme
}

// Mirrors internal/config.FieldError. The `field` value is normalized to
// the camelCase form path expected by react-hook-form by the bridge
// mappers — backend snake_case never reaches the UI.
export interface ConfigFieldError {
  field: string
  msg: string
}

// Mirrors internal/config.ValidationError after mapping.
export interface ConfigValidationError {
  errors: ConfigFieldError[]
}

// Mirrors internal/profiles.AuthMethod. Tokens stay on the Go side; the
// frontend never inspects keyringRef directly.
export type AuthMethod = "gh-cli" | "keyring"

export interface Profile {
  id: string
  name: string
  authMethod: AuthMethod
  keyringRef?: string
  githubUsername?: string
  isActive: boolean
  createdAt: string
  lastValidatedAt?: string
}

export interface ProfileUpdate {
  name?: string
  authMethod?: AuthMethod
  token?: string
}

// ===== PR details (REV-13) =====
//
// PRFullDetails and its supporting types are camelCase on both sides
// (Go tags match the gh-CLI payload shape), so the public TS types can
// pass through the bridge without translation.

export interface Label {
  name: string
  color: string
}

export interface Review {
  author: string
  state: string
  submittedAt: string
}

export interface StatusCheck {
  name: string
  status: string
  conclusion: string
  url: string
}

export interface ChangedFile {
  path: string
  additions: number
  deletions: number
}

export type MergeableStatus = "MERGEABLE" | "CONFLICTING" | "UNKNOWN"

export type MergeMethod = "squash" | "merge"

export interface PRFullDetails {
  number: number
  title: string
  body: string
  url: string
  state: string
  isDraft: boolean
  author: string
  additions: number
  deletions: number
  changedFiles: number
  labels: Label[]
  reviews: Review[]
  statusChecks: StatusCheck[]
  files: ChangedFile[]
  mergeable: string
  baseRefName: string
  headRefName: string
  createdAt: string
  updatedAt: string
  mergedAt: string | null
}
