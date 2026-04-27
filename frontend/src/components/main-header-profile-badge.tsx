import { UserRound } from "lucide-react"

import { useActiveProfile } from "./settings/use-active-profile"

interface MainHeaderProfileBadgeProps {
  onOpenAccounts: () => void
}

export function MainHeaderProfileBadge({
  onOpenAccounts,
}: MainHeaderProfileBadgeProps) {
  const { profile } = useActiveProfile()
  if (!profile) return null

  return (
    <button
      type="button"
      onClick={onOpenAccounts}
      className="flex items-center gap-1 rounded-md border border-transparent px-1.5 py-0.5 text-[11px] text-muted-foreground transition-colors hover:border-border hover:text-foreground"
      aria-label={`Conta ativa: ${profile.name}`}
    >
      <UserRound className="size-3" aria-hidden="true" />
      <span className="truncate">{profile.name}</span>
    </button>
  )
}
