import { Skeleton } from "@/components/ui/skeleton"

interface PRDetailsLoadingSkeletonProps {
  onBack: () => void
}

export function PRDetailsLoadingSkeleton({
  onBack,
}: PRDetailsLoadingSkeletonProps) {
  return (
    <div className="flex h-screen flex-col gap-3 bg-background p-3 text-foreground">
      <button
        type="button"
        onClick={onBack}
        className="self-start text-sm text-muted-foreground underline"
      >
        ← Voltar
      </button>
      <Skeleton className="h-6 w-3/4" />
      <Skeleton className="h-3 w-1/2" />
      <Skeleton className="h-24 w-full" />
      <Skeleton className="h-40 w-full" />
    </div>
  )
}
