import type { Profile, ProfileUpdate } from "@/lib/types"

import { requireBridge } from "./client"
import {
  fromCreateProfile,
  fromProfileUpdate,
  toProfile,
  type CreateProfileInput,
} from "./mappers"
import type {
  CreateProfileWireRequest,
  ProfileWire,
  UpdateProfileWireRequest,
} from "./wire"

export type { CreateProfileInput } from "./mappers"

export interface ProfilesBridge {
  ListProfiles(): Promise<ProfileWire[]>
  GetActiveProfile(): Promise<ProfileWire>
  CreateProfile(req: CreateProfileWireRequest): Promise<ProfileWire>
  UpdateProfile(req: UpdateProfileWireRequest): Promise<ProfileWire>
  DeleteProfile(id: string): Promise<void>
  SetActiveProfile(id: string): Promise<void>
  ValidateToken(token: string): Promise<string>
}

export async function listProfiles(): Promise<Profile[]> {
  return (await requireBridge("ListProfiles")()).map(toProfile)
}

export async function getActiveProfile(): Promise<Profile> {
  return toProfile(await requireBridge("GetActiveProfile")())
}

export async function createProfile(
  input: CreateProfileInput
): Promise<Profile> {
  return toProfile(
    await requireBridge("CreateProfile")(fromCreateProfile(input))
  )
}

export async function updateProfile(
  id: string,
  patch: ProfileUpdate
): Promise<Profile> {
  return toProfile(
    await requireBridge("UpdateProfile")(fromProfileUpdate(id, patch))
  )
}

export const deleteProfile = (id: string): Promise<void> =>
  requireBridge("DeleteProfile")(id)

export const setActiveProfile = (id: string): Promise<void> =>
  requireBridge("SetActiveProfile")(id)

export const validateToken = (token: string): Promise<string> =>
  requireBridge("ValidateToken")(token)
