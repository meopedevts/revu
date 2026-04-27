import { useCallback, useEffect, useState } from 'react'

import { getActiveProfile } from '@/src/lib/bridge'
import type { Profile } from '@/src/lib/types'
import { EventsOff, EventsOn } from '@/wailsjs/runtime/runtime'

const EVENT = 'profiles:active-changed'

// useActiveProfile returns the currently-active profile and refreshes it
// when the backend emits profiles:active-changed. refresh() is exposed for
// callers that mutate the state (e.g. the Contas section after a Create).
export function useActiveProfile(): {
  profile: Profile | null
  refresh: () => Promise<void>
} {
  const [profile, setProfile] = useState<Profile | null>(null)

  const refresh = useCallback(async () => {
    setProfile(await getActiveProfile())
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
