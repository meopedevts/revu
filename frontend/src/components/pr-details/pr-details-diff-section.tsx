import { DETAILS_DIFF_LIMIT } from "@/generated/constants"

import { PRDetailsBigPRPlaceholder } from "./big-pr-placeholder"
import { PRDetailsDiff } from "./pr-details-diff"

interface PRDetailsDiffSectionProps {
  url: string
  additions: number
  deletions: number
  diff: string | null
}

export function PRDetailsDiffSection({
  url,
  additions,
  deletions,
  diff,
}: PRDetailsDiffSectionProps) {
  const totalLines = additions + deletions
  const diffTooBig = totalLines > DETAILS_DIFF_LIMIT
  const diffEmpty = !diffTooBig && (diff === null || diff === "")

  return (
    <section className="space-y-1">
      <h2 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
        Diff
      </h2>
      {diffTooBig ? (
        <PRDetailsBigPRPlaceholder url={url} totalLines={totalLines} />
      ) : diffEmpty ? (
        <div className="text-xs text-muted-foreground italic">diff vazio</div>
      ) : (
        <PRDetailsDiff diff={diff ?? ""} />
      )}
    </section>
  )
}
