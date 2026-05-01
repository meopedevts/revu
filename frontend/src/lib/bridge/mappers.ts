// Mappers — boundary translation between Go JSON wire types (snake_case
// where the backend uses snake_case tags) and the camelCase TypeScript
// types consumed by the rest of the app. Field-by-field, no `any`.

import type {
  AppConfig,
  AuthMethod,
  ConfigFieldError,
  ConfigValidationError,
  PRRecord,
  Profile,
  ProfileUpdate,
} from "@/lib/types"

import type {
  AppConfigWire,
  ConfigFieldErrorWire,
  ConfigValidationErrorWire,
  CreateProfileWireRequest,
  PRRecordWire,
  ProfileWire,
  UpdateProfileWireRequest,
} from "./wire"

export function toPRRecord(w: PRRecordWire): PRRecord {
  return {
    id: w.id,
    number: w.number,
    repo: w.repo,
    title: w.title,
    author: w.author,
    url: w.url,
    state: w.state,
    isDraft: w.is_draft,
    additions: w.additions,
    deletions: w.deletions,
    reviewPending: w.review_pending,
    reviewState: w.review_state,
    firstSeenAt: w.first_seen_at,
    lastSeenAt: w.last_seen_at,
    lastNotifiedAt: w.last_notified_at,
  }
}

export function toAppConfig(w: AppConfigWire): AppConfig {
  return {
    pollingIntervalSeconds: w.polling_interval_seconds,
    notificationsEnabled: w.notifications_enabled,
    notificationTimeoutSeconds: w.notification_timeout_seconds,
    statusRefreshEveryNTicks: w.status_refresh_every_n_ticks,
    historyRetentionDays: w.history_retention_days,
    startHidden: w.start_hidden,
    window: w.window,
    theme: w.theme,
  }
}

export function fromAppConfig(c: AppConfig): AppConfigWire {
  return {
    polling_interval_seconds: c.pollingIntervalSeconds,
    notifications_enabled: c.notificationsEnabled,
    notification_timeout_seconds: c.notificationTimeoutSeconds,
    status_refresh_every_n_ticks: c.statusRefreshEveryNTicks,
    history_retention_days: c.historyRetentionDays,
    start_hidden: c.startHidden,
    window: c.window,
    theme: c.theme,
  }
}

export function toProfile(w: ProfileWire): Profile {
  return {
    id: w.id,
    name: w.name,
    authMethod: w.auth_method,
    keyringRef: w.keyring_ref,
    githubUsername: w.github_username,
    isActive: w.is_active,
    createdAt: w.created_at,
    lastValidatedAt: w.last_validated_at,
  }
}

export interface CreateProfileInput {
  name: string
  authMethod: AuthMethod
  token: string
  makeActive: boolean
}

export function fromCreateProfile(
  input: CreateProfileInput
): CreateProfileWireRequest {
  return {
    name: input.name,
    auth_method: input.authMethod,
    token: input.token,
    make_active: input.makeActive,
  }
}

export function fromProfileUpdate(
  id: string,
  p: ProfileUpdate
): UpdateProfileWireRequest {
  return {
    id,
    name: p.name,
    auth_method: p.authMethod,
    token: p.token,
  }
}

// Backend ValidationError carries Go JSON field names (snake_case for
// internal/config). The form binds to camelCase, so we translate the
// field path before reporting it back to react-hook-form.
const CONFIG_FIELD_TO_PATH: Record<string, string> = {
  polling_interval_seconds: "pollingIntervalSeconds",
  notifications_enabled: "notificationsEnabled",
  notification_timeout_seconds: "notificationTimeoutSeconds",
  status_refresh_every_n_ticks: "statusRefreshEveryNTicks",
  history_retention_days: "historyRetentionDays",
  start_hidden: "startHidden",
}

function mapConfigFieldPath(field: string): string {
  return CONFIG_FIELD_TO_PATH[field] ?? field
}

function toConfigValidationError(
  w: ConfigValidationErrorWire
): ConfigValidationError {
  return {
    errors: w.errors.map(
      (e: ConfigFieldErrorWire): ConfigFieldError => ({
        field: mapConfigFieldPath(e.field),
        msg: e.msg,
      })
    ),
  }
}

function isConfigValidationErrorWire(
  err: unknown
): err is ConfigValidationErrorWire {
  return (
    typeof err === "object" &&
    err !== null &&
    "errors" in err &&
    Array.isArray(err.errors)
  )
}

// parseConfigValidationError accepts whatever the bridge throws (a
// raw wire object, an Error whose message is JSON-encoded by Wails, or
// anything else) and returns a public camelCase ConfigValidationError
// when the shape matches. Keeping the JSON-decoding here means callers
// only depend on the public type and never see the wire shape.
export function parseConfigValidationError(
  err: unknown
): ConfigValidationError | null {
  if (isConfigValidationErrorWire(err)) return toConfigValidationError(err)
  if (err instanceof Error) {
    try {
      const parsed = JSON.parse(err.message) as unknown
      if (isConfigValidationErrorWire(parsed))
        return toConfigValidationError(parsed)
    } catch {
      return null
    }
  }
  return null
}
