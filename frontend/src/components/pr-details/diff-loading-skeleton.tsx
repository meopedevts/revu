import { ChevronRight } from "lucide-react"

import { Skeleton } from "@/components/ui/skeleton"

// DiffLoadingSkeleton mimica 2 file blocks colapsados (header + chevron + path
// mono). Mantém o layout do diff section estável enquanto a request do diff
// está em voo, evitando o flash "diff vazio" enganoso.
export function DiffLoadingSkeleton() {
  return (
    <div aria-hidden="true" className="flex flex-col gap-2">
      <FileBlockPlaceholder widthClass="w-3/5" />
      <FileBlockPlaceholder widthClass="w-2/5" />
    </div>
  )
}

function FileBlockPlaceholder({ widthClass }: { widthClass: string }) {
  return (
    <div className="rounded-lg border border-border bg-card">
      <div className="flex w-full items-center gap-2 px-3 py-2">
        <ChevronRight
          className="size-3 shrink-0 text-muted-foreground/40"
          aria-hidden="true"
        />
        <Skeleton className={`h-3 ${widthClass}`} />
      </div>
    </div>
  )
}
