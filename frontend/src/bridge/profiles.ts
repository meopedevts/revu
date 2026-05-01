import type { AuthMethod, Profile, ProfileUpdate } from "@/lib/types"

import { requireBridge } from "./client"

export interface CreateProfileRequest {
  name: string
  auth_method: AuthMethod
  token: string
  make_active: boolean
}

export interface UpdateProfileRequest {
  id: string
  name?: string
  auth_method?: AuthMethod
  token?: string
}

export interface ProfilesBridge {
  ListProfiles(): Promise<Profile[]>
  GetActiveProfile(): Promise<Profile>
  CreateProfile(req: CreateProfileRequest): Promise<Profile>
  UpdateProfile(req: UpdateProfileRequest): Promise<Profile>
  DeleteProfile(id: string): Promise<void>
  SetActiveProfile(id: string): Promise<void>
  ValidateToken(token: string): Promise<string>
}

export const listProfiles = (): Promise<Profile[]> =>
  requireBridge("ListProfiles")()

export const getActiveProfile = (): Promise<Profile> =>
  requireBridge("GetActiveProfile")()

export const createProfile = (input: CreateProfileRequest): Promise<Profile> =>
  requireBridge("CreateProfile")(input)

export const updateProfile = (
  id: string,
  patch: ProfileUpdate
): Promise<Profile> => requireBridge("UpdateProfile")({ id, ...patch })

export const deleteProfile = (id: string): Promise<void> =>
  requireBridge("DeleteProfile")(id)

export const setActiveProfile = (id: string): Promise<void> =>
  requireBridge("SetActiveProfile")(id)

export const validateToken = (token: string): Promise<string> =>
  requireBridge("ValidateToken")(token)
