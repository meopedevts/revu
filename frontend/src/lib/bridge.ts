import {
  fromAppConfig,
  fromCreateProfile,
  fromProfileUpdate,
  toAppConfig,
  toPRRecord,
  toProfile,
  type CreateProfileInput,
} from "./bridge/mappers"
import type {
  AppConfigWire,
  CreateProfileWireRequest,
  PRFullDetailsWire,
  PRRecordWire,
  ProfileWire,
  UpdateProfileWireRequest,
} from "./bridge/wire"
import type {
  AppConfig,
  MergeMethod,
  PRFullDetails,
  PRRecord,
  Profile,
  ProfileUpdate,
  Theme,
} from "./types"

// Wails v2 injects Go bindings under window.go.<package>.<Struct>. This
// module isolates that runtime contract so the rest of the app does not
// depend on the generated wailsjs files (which require main.go at the
// module root — our entrypoint lives at cmd/revu/main.go instead). The
// bridge speaks wire types (snake_case where Go does); mappers translate
// to/from the camelCase types in `./types` before anything is exposed.
interface WailsBridge {
  ListPendingPRs(): Promise<PRRecordWire[]>
  ListHistoryPRs(): Promise<PRRecordWire[]>
  OpenPRInBrowser(url: string): Promise<void>
  RefreshNow(): Promise<void>
  ShowWindow(): Promise<void>
  HideWindow(): Promise<void>
  GetConfig(): Promise<AppConfigWire>
  UpdateConfig(c: AppConfigWire): Promise<void>
  GetTheme?(): Promise<string>
  SetTheme?(theme: string): Promise<void>
  ClearHistory(): Promise<number>
  GetPRDetails?(prID: string): Promise<PRFullDetailsWire>
  GetPRDiff?(prID: string): Promise<string>
  MergePR?(prID: string, method: MergeMethod): Promise<void>
  ListProfiles?(): Promise<ProfileWire[]>
  GetActiveProfile?(): Promise<ProfileWire>
  CreateProfile?(req: CreateProfileWireRequest): Promise<ProfileWire>
  UpdateProfile?(req: UpdateProfileWireRequest): Promise<ProfileWire>
  DeleteProfile?(id: string): Promise<void>
  SetActiveProfile?(id: string): Promise<void>
  ValidateToken?(token: string): Promise<string>
}

declare global {
  interface Window {
    go?: {
      app?: {
        App?: WailsBridge
      }
    }
  }
}

function bridge(): WailsBridge | undefined {
  return window.go?.app?.App
}

export async function listPendingPRs(): Promise<PRRecord[]> {
  const b = bridge()
  if (!b) return []
  const wire = await b.ListPendingPRs()
  return wire.map(toPRRecord)
}

export async function listHistoryPRs(): Promise<PRRecord[]> {
  const b = bridge()
  if (!b) return []
  const wire = await b.ListHistoryPRs()
  return wire.map(toPRRecord)
}

export async function openPRInBrowser(url: string): Promise<void> {
  await bridge()?.OpenPRInBrowser(url)
}

export async function refreshNow(): Promise<void> {
  await bridge()?.RefreshNow()
}

export async function hideWindow(): Promise<void> {
  await bridge()?.HideWindow()
}

export async function getConfig(): Promise<AppConfig | null> {
  const b = bridge()
  if (!b) return null
  return toAppConfig(await b.GetConfig())
}

export async function updateConfig(c: AppConfig): Promise<void> {
  const b = bridge()
  if (!b) throw new Error("bridge unavailable")
  await b.UpdateConfig(fromAppConfig(c))
}

export async function getTheme(): Promise<Theme> {
  const b = bridge()
  if (!b?.GetTheme) return "light"
  const t = await b.GetTheme()
  return t === "dark" ? "dark" : "light"
}

export async function setTheme(theme: Theme): Promise<void> {
  const b = bridge()
  if (!b?.SetTheme) throw new Error("bridge unavailable")
  await b.SetTheme(theme)
}

export async function clearHistory(): Promise<number> {
  const b = bridge()
  if (!b) return 0
  return b.ClearHistory()
}

// ===== PR details (REV-13) =====

export async function getPRDetails(prID: string): Promise<PRFullDetails> {
  const b = bridge()
  if (!b?.GetPRDetails) throw new Error("bridge unavailable")
  return b.GetPRDetails(prID)
}

export async function getPRDiff(prID: string): Promise<string> {
  const b = bridge()
  if (!b?.GetPRDiff) throw new Error("bridge unavailable")
  return b.GetPRDiff(prID)
}

export async function mergePR(
  prID: string,
  method: MergeMethod
): Promise<void> {
  const b = bridge()
  if (!b?.MergePR) throw new Error("bridge unavailable")
  await b.MergePR(prID, method)
}

// ===== Profiles =====
//
// Every wrapper returns a graceful fallback when the bridge is unavailable
// (vite preview, tests, smoke builds without `app.WithProfiles`). CRUD
// operations throw instead, since the UI needs to know the action did not
// persist.

export async function listProfiles(): Promise<Profile[]> {
  const b = bridge()
  if (!b?.ListProfiles) return []
  const wire = await b.ListProfiles()
  return wire.map(toProfile)
}

export async function getActiveProfile(): Promise<Profile | null> {
  const b = bridge()
  if (!b?.GetActiveProfile) return null
  try {
    return toProfile(await b.GetActiveProfile())
  } catch {
    return null
  }
}

export async function createProfile(
  input: CreateProfileInput
): Promise<Profile> {
  const b = bridge()
  if (!b?.CreateProfile) throw new Error("bridge unavailable")
  return toProfile(await b.CreateProfile(fromCreateProfile(input)))
}

export async function updateProfile(
  id: string,
  patch: ProfileUpdate
): Promise<Profile> {
  const b = bridge()
  if (!b?.UpdateProfile) throw new Error("bridge unavailable")
  return toProfile(await b.UpdateProfile(fromProfileUpdate(id, patch)))
}

export async function deleteProfile(id: string): Promise<void> {
  const b = bridge()
  if (!b?.DeleteProfile) throw new Error("bridge unavailable")
  await b.DeleteProfile(id)
}

export async function setActiveProfile(id: string): Promise<void> {
  const b = bridge()
  if (!b?.SetActiveProfile) throw new Error("bridge unavailable")
  await b.SetActiveProfile(id)
}

export async function validateToken(token: string): Promise<string> {
  const b = bridge()
  if (!b?.ValidateToken) throw new Error("bridge unavailable")
  return b.ValidateToken(token)
}
