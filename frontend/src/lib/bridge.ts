import type {
  AppConfig,
  AuthMethod,
  MergeMethod,
  PRFullDetails,
  PRRecord,
  Profile,
  ProfileUpdate,
  Theme,
} from './types'

interface CreateProfileRequest {
  name: string
  auth_method: AuthMethod
  token: string
  make_active: boolean
}

interface UpdateProfileRequest {
  id: string
  name?: string
  auth_method?: AuthMethod
  token?: string
}

// Wails v2 injects Go bindings under window.go.<package>.<Struct>. This
// module isolates that runtime contract so the rest of the app does not
// depend on the generated wailsjs files (which require main.go at the
// module root — our entrypoint lives at cmd/revu/main.go instead).
interface WailsBridge {
  ListPendingPRs(): Promise<PRRecord[]>
  ListHistoryPRs(): Promise<PRRecord[]>
  OpenPRInBrowser(url: string): Promise<void>
  RefreshNow(): Promise<void>
  ShowWindow(): Promise<void>
  HideWindow(): Promise<void>
  GetConfig(): Promise<AppConfig>
  UpdateConfig(c: AppConfig): Promise<void>
  GetTheme?(): Promise<string>
  SetTheme?(theme: string): Promise<void>
  ClearHistory(): Promise<number>
  GetPRDetails?(prID: string): Promise<PRFullDetails>
  GetPRDiff?(prID: string): Promise<string>
  MergePR?(prID: string, method: MergeMethod): Promise<void>
  ListProfiles?(): Promise<Profile[]>
  GetActiveProfile?(): Promise<Profile>
  CreateProfile?(req: CreateProfileRequest): Promise<Profile>
  UpdateProfile?(req: UpdateProfileRequest): Promise<Profile>
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
  return b.ListPendingPRs()
}

export async function listHistoryPRs(): Promise<PRRecord[]> {
  const b = bridge()
  if (!b) return []
  return b.ListHistoryPRs()
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
  return b.GetConfig()
}

export async function updateConfig(c: AppConfig): Promise<void> {
  const b = bridge()
  if (!b) throw new Error('bridge unavailable')
  await b.UpdateConfig(c)
}

export async function getTheme(): Promise<Theme> {
  const b = bridge()
  if (!b?.GetTheme) return 'light'
  const t = await b.GetTheme()
  return t === 'dark' ? 'dark' : 'light'
}

export async function setTheme(theme: Theme): Promise<void> {
  const b = bridge()
  if (!b?.SetTheme) throw new Error('bridge unavailable')
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
  if (!b?.GetPRDetails) throw new Error('bridge unavailable')
  return b.GetPRDetails(prID)
}

export async function getPRDiff(prID: string): Promise<string> {
  const b = bridge()
  if (!b?.GetPRDiff) throw new Error('bridge unavailable')
  return b.GetPRDiff(prID)
}

export async function mergePR(
  prID: string,
  method: MergeMethod
): Promise<void> {
  const b = bridge()
  if (!b?.MergePR) throw new Error('bridge unavailable')
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
  return b.ListProfiles()
}

export async function getActiveProfile(): Promise<Profile | null> {
  const b = bridge()
  if (!b?.GetActiveProfile) return null
  try {
    return await b.GetActiveProfile()
  } catch {
    return null
  }
}

export async function createProfile(input: {
  name: string
  auth_method: AuthMethod
  token: string
  make_active: boolean
}): Promise<Profile> {
  const b = bridge()
  if (!b?.CreateProfile) throw new Error('bridge unavailable')
  return b.CreateProfile(input)
}

export async function updateProfile(
  id: string,
  patch: ProfileUpdate
): Promise<Profile> {
  const b = bridge()
  if (!b?.UpdateProfile) throw new Error('bridge unavailable')
  return b.UpdateProfile({ id, ...patch })
}

export async function deleteProfile(id: string): Promise<void> {
  const b = bridge()
  if (!b?.DeleteProfile) throw new Error('bridge unavailable')
  await b.DeleteProfile(id)
}

export async function setActiveProfile(id: string): Promise<void> {
  const b = bridge()
  if (!b?.SetActiveProfile) throw new Error('bridge unavailable')
  await b.SetActiveProfile(id)
}

export async function validateToken(token: string): Promise<string> {
  const b = bridge()
  if (!b?.ValidateToken) throw new Error('bridge unavailable')
  return b.ValidateToken(token)
}
