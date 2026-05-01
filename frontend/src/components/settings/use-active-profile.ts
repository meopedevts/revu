import { useCallback, useEffect, useState } from "react"

import type { Profile } from "@/lib/types"
import { getActiveProfile } from "@/shared/bridge"
import { EventsOff, EventsOn } from "@/wailsjs/runtime/runtime"

const EVENT = "profiles:active-changed"

// useActiveProfile returns the currently-active profile and refreshes it
// when the backend emits profiles:active-changed. refresh() is exposed for
// callers that mutate the state (e.g. the Contas section after a Create).
export function useActiveProfile(): {
  profile: Profile | null
  refresh: () => Promise<void>
} {
  const [profile, setProfile] = useState<Profile | null>(null)

  const refresh = useCallback(async () => {
    try {
      setProfile(await getActiveProfile())
    } catch {
      // Bridge unavailable / no active profile (smoke build, fresh
      // install). REV-33 query layer takes over error semantics.
      setProfile(null)
    }
  }, [])

  useEffect(() => {
    void refresh()
    EventsOn(EVENT, (p: Profile) => setProfile(p))
    return () => {
      EventsOff(EVENT)
    }
  }, [refresh])

  return { profile, refresh }
}
