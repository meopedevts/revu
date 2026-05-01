// Wire types — boundary-only mirrors of Go JSON payloads.
//
// These shapes match the JSON tags emitted by the Go backend verbatim
// (snake_case for store/config/profiles, camelCase for gh-CLI-derived
// data). Only modules inside `src/bridge/` may import from here; the
// rest of the app consumes the camelCase types in `src/lib/types`,
// produced by the mappers in `./mappers`.

import type { Theme } from "@/generated/constants"
import type {
  AuthMethod,
  ChangedFile,
  Label,
  Review,
  StatusCheck,
  WindowConfig,
} from "@/lib/types"

export interface PRRecordWire {
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

export interface AppConfigWire {
  polling_interval_seconds: number
  notifications_enabled: boolean
  notification_timeout_seconds: number
  notification_cooldown_minutes: number
  status_refresh_every_n_ticks: number
  history_retention_days: number
  start_hidden: boolean
  window: WindowConfig
  theme: Theme
}

export interface ProfileWire {
  id: string
  name: string
  auth_method: AuthMethod
  keyring_ref?: string
  github_username?: string
  is_active: boolean
  created_at: string
  last_validated_at?: string
}

export interface CreateProfileWireRequest {
  name: string
  auth_method: AuthMethod
  token: string
  make_active: boolean
}

export interface UpdateProfileWireRequest {
  id: string
  name?: string
  auth_method?: AuthMethod
  token?: string
}

export interface ConfigFieldErrorWire {
  field: string
  msg: string
}

export interface ConfigValidationErrorWire {
  errors: ConfigFieldErrorWire[]
}

// PRFullDetails and its supporting types (Label, Review, StatusCheck,
// ChangedFile) are already camelCase on the Go side (they wrap gh-CLI
// payloads), so the public TS types double as wire types. Re-exported
// here so the bridge layer has a single import surface.
export type {
  ChangedFile as ChangedFileWire,
  Label as LabelWire,
  Review as ReviewWire,
  StatusCheck as StatusCheckWire,
}
export type { PRFullDetails as PRFullDetailsWire } from "@/lib/types"
