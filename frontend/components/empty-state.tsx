import { Inbox, ScrollText } from 'lucide-react'

interface EmptyStateProps {
  variant: 'pending' | 'history'
}

const COPY: Record<
  EmptyStateProps['variant'],
  { title: string; hint: string }
> = {
  pending: {
    title: 'Nenhum PR aguardando review.',
    hint: 'Quando alguém te marcar como reviewer, ele aparece aqui.',
  },
  history: {
    title: 'Histórico vazio.',
    hint: 'PRs já reviewed ou fechados aparecem aqui.',
  },
}

export function EmptyState({ variant }: EmptyStateProps) {
  const { title, hint } = COPY[variant]
  const Icon = variant === 'pending' ? Inbox : ScrollText
  return (
    <div className="flex h-48 flex-col items-center justify-center gap-2 text-center text-muted-foreground">
      <Icon className="size-8 opacity-60" aria-hidden="true" />
      <div className="text-sm font-medium text-foreground">{title}</div>
      <div className="max-w-xs text-xs">{hint}</div>
    </div>
  )
}
