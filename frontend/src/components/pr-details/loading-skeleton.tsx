import { ArrowLeft, GitBranch } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"

import { DiffLoadingSkeleton } from "./diff-loading-skeleton"

interface PRDetailsLoadingSkeletonProps {
  onBack: () => void
}

// PRDetailsLoadingSkeleton espelha o layout real do PRDetailsView (header,
// meta, stats, checks, reviewers, body, files, diff section) pra eliminar CLS
// quando os dados chegam.
export function PRDetailsLoadingSkeleton({
  onBack,
}: PRDetailsLoadingSkeletonProps) {
  return (
    <div className="flex h-screen flex-col gap-3 overflow-y-auto bg-background p-3 text-foreground">
      <div className="flex flex-col gap-2">
        <div className="flex items-center justify-between gap-2">
          <Button
            size="sm"
            variant="ghost"
            onClick={onBack}
            aria-label="Voltar"
          >
            <ArrowLeft data-icon="inline-start" />
            Voltar
          </Button>
          <div className="flex items-center gap-1" aria-hidden="true">
            <Skeleton className="h-8 w-32 rounded-md" />
            <Skeleton className="h-8 w-20 rounded-md" />
            <Skeleton className="h-8 w-16 rounded-md" />
          </div>
        </div>

        <div className="flex flex-wrap items-start gap-2" aria-hidden="true">
          <div className="flex min-w-0 flex-1 items-center gap-2">
            <Skeleton className="h-4 w-10" />
            <Skeleton className="h-5 w-3/5" />
          </div>
          <div className="flex shrink-0 items-center gap-1">
            <Skeleton className="h-5 w-16 rounded-full" />
            <Skeleton className="h-5 w-20 rounded-full" />
          </div>
        </div>
      </div>

      <div
        className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground"
        aria-hidden="true"
      >
        <Skeleton className="h-3 w-20" />
        <span aria-hidden="true">·</span>
        <span className="inline-flex items-center gap-1">
          <GitBranch className="size-3 text-muted-foreground/40" />
          <Skeleton className="h-3 w-16" />
          <span aria-hidden="true">←</span>
          <Skeleton className="h-3 w-20" />
        </span>
        <span aria-hidden="true">·</span>
        <Skeleton className="h-3 w-24" />
      </div>

      <div
        className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs"
        aria-hidden="true"
      >
        <Skeleton className="h-3 w-12" />
        <span aria-hidden="true">·</span>
        <Skeleton className="h-3 w-16" />
        <span aria-hidden="true">·</span>
        <Skeleton className="h-3 w-16" />
      </div>

      <div className="flex flex-wrap gap-1" aria-hidden="true">
        <Skeleton className="h-5 w-24 rounded-full" />
        <Skeleton className="h-5 w-20 rounded-full" />
      </div>

      <section className="space-y-1" aria-hidden="true">
        <Skeleton className="h-3 w-24" />
        <div className="flex flex-wrap gap-2">
          <Skeleton className="h-6 w-28 rounded-full" />
          <Skeleton className="h-6 w-32 rounded-full" />
          <Skeleton className="h-6 w-24 rounded-full" />
        </div>
      </section>

      <section className="space-y-1" aria-hidden="true">
        <Skeleton className="h-3 w-24" />
        <Skeleton className="h-24 w-full rounded-md" />
      </section>

      <section className="space-y-1" aria-hidden="true">
        <Skeleton className="h-3 w-32" />
        <div className="flex flex-col gap-1.5">
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-11/12" />
          <Skeleton className="h-4 w-9/12" />
          <Skeleton className="h-4 w-10/12" />
        </div>
      </section>

      <section className="space-y-1">
        <Skeleton className="h-3 w-12" aria-hidden="true" />
        <DiffLoadingSkeleton />
      </section>
    </div>
  )
}
