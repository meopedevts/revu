import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"

// PRCardSkeleton espelha o layout do PRCard (avatar, title row, meta row,
// badges column, footer stats) pra evitar CLS quando dados chegam no 1º
// fetch da PR list.
export function PRCardSkeleton() {
  return (
    <Card size="sm" aria-hidden="true" className="ring-1 ring-foreground/10">
      <div className="flex items-start gap-3 px-3">
        <Skeleton className="size-8 shrink-0 rounded-full" />

        <div className="flex min-w-0 flex-1 flex-col gap-1.5">
          <div className="flex items-baseline gap-2">
            <Skeleton className="h-3 w-8" />
            <Skeleton className="h-3.5 w-2/3" />
          </div>
          <div className="flex flex-wrap items-center gap-x-2 gap-y-0.5">
            <Skeleton className="h-3 w-20" />
            <Skeleton className="h-3 w-16" />
            <Skeleton className="h-4 w-24 rounded" />
          </div>
        </div>

        <div className="flex shrink-0 flex-col items-end gap-1">
          <Skeleton className="h-4 w-14 rounded-full" />
          <Skeleton className="h-4 w-16 rounded-full" />
        </div>
      </div>

      <CardContent className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Skeleton className="h-3 w-8" />
          <Skeleton className="h-3 w-8" />
        </div>
        <Skeleton className="h-3 w-20" />
      </CardContent>
    </Card>
  )
}
