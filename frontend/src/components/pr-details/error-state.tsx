interface PRDetailsErrorStateProps {
  message: string
  onBack: () => void
  onRetry: () => void
}

export function PRDetailsErrorState({
  message,
  onBack,
  onRetry,
}: PRDetailsErrorStateProps) {
  return (
    <div className="flex h-screen flex-col gap-3 bg-background p-3 text-foreground">
      <button
        type="button"
        onClick={onBack}
        className="self-start text-sm text-muted-foreground underline"
      >
        ← Voltar
      </button>
      <div className="flex flex-1 flex-col items-center justify-center gap-2 text-center">
        <div className="text-sm text-destructive">{message}</div>
        <button type="button" onClick={onRetry} className="text-xs underline">
          tentar de novo
        </button>
      </div>
    </div>
  )
}
