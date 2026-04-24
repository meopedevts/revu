import { SettingsLayout } from './settings/settings-layout'
import type { SettingsSection } from './settings/settings-sidebar'

interface SettingsViewProps {
  onBack: () => void
  initialSection?: SettingsSection
}

export function SettingsView({ onBack, initialSection }: SettingsViewProps) {
  return <SettingsLayout onBack={onBack} initialSection={initialSection} />
}
