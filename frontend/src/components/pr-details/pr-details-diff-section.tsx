import { DETAILS_DIFF_LIMIT } from "@/generated/constants"

import { PRDetailsBigPRPlaceholder } from "./big-pr-placeholder"
import { DiffLoadingSkeleton } from "./diff-loading-skeleton"
import { PRDetailsDiff } from "./pr-details-diff"

interface PRDetailsDiffSectionProps {
  url: string
  additions: number
  deletions: number
  diff: string | null
  diffError: Error | null
  diffLoading: boolean
}

export function PRDetailsDiffSection({
  url,
  additions,
  deletions,
  diff,
  diffError,
  diffLoading,
}: PRDetailsDiffSectionProps) {
  const totalLines = additions + deletions
  const diffTooBig = totalLines > DETAILS_DIFF_LIMIT
  const diffFailed = !diffTooBig && diffError !== null
  // Loading tem prioridade sobre "vazio" pra evitar o flash enganoso entre
  // details chegar e diff terminar de carregar.
  const showLoading = !diffTooBig && !diffFailed && diffLoading
  const diffEmpty =
    !diffTooBig && !diffFailed && !showLoading && (diff === null || diff === "")

  return (
    <section className="space-y-1">
      <h2 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
        Diff
      </h2>
      {diffTooBig ? (
        <PRDetailsBigPRPlaceholder url={url} totalLines={totalLines} />
      ) : diffFailed ? (
        <div className="text-xs text-destructive">
          falha ao carregar diff: {diffError?.message ?? "erro desconhecido"}
        </div>
      ) : showLoading ? (
        <DiffLoadingSkeleton />
      ) : diffEmpty ? (
        <div className="text-xs text-muted-foreground italic">diff vazio</div>
      ) : (
        <PRDetailsDiff diff={diff ?? ""} />
      )}
    </section>
  )
}
