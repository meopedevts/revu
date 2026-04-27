import { createFileRoute, notFound } from "@tanstack/react-router"

import { AccountsSection } from "@/components/settings/sections/accounts-section"
import { AppearanceSection } from "@/components/settings/sections/appearance-section"
import { HistorySection } from "@/components/settings/sections/history-section"
import { NotificationsSection } from "@/components/settings/sections/notifications-section"
import { SyncSection } from "@/components/settings/sections/sync-section"
import { isSettingsSection } from "@/components/settings/settings-sidebar"

export const Route = createFileRoute("/settings/$section")({
  parseParams: ({ section }) => {
    if (!isSettingsSection(section)) {
      // eslint-disable-next-line @typescript-eslint/only-throw-error
      throw notFound()
    }
    return { section }
  },
  component: SectionView,
})

function SectionView() {
  const { section } = Route.useParams()

  switch (section) {
    case "accounts":
      return <AccountsSection />
    case "sync":
      return <SyncSection />
    case "notifications":
      return <NotificationsSection />
    case "history":
      return <HistorySection />
    case "appearance":
      return <AppearanceSection />
  }
}
