import { useCallback, useEffect, useMemo, useState } from "react"
import { toast } from "sonner"

import { Skeleton } from "@/components/ui/skeleton"
import { mergePR, openPRInBrowser } from "@/src/lib/bridge"
import { usePRDetails } from "@/src/lib/hooks/use-pr-details"
import {
  type MergeMethod,
  type PRFullDetails,
  type PRState,
  type ReviewState,
} from "@/src/lib/types"
import { DETAILS_DIFF_LIMIT } from "@/src/shared/generated/constants"

import { PRDetailsBody } from "./pr-details/pr-details-body"
import { PRDetailsChecks } from "./pr-details/pr-details-checks"
import { PRDetailsDiff } from "./pr-details/pr-details-diff"
import { PRDetailsFiles } from "./pr-details/pr-details-files"
import { PRDetailsHeader } from "./pr-details/pr-details-header"
import { PRDetailsMeta } from "./pr-details/pr-details-meta"
import { PRDetailsReviewers } from "./pr-details/pr-details-reviewers"
import { PRMergeDialog } from "./pr-details/pr-merge-dialog"

interface PRDetailsViewProps {
  prID: string
  onBack: () => void
}

export function PRDetailsView({ prID, onBack }: PRDetailsViewProps) {
  const { details, diff, loading, error, reload } = usePRDetails(prID)

  // Esc → back. Scoped to the component so the listener unmounts when we
  // leave the details view.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onBack()
    }
    window.addEventListener("keydown", onKey)
    return () => window.removeEventListener("keydown", onKey)
  }, [onBack])

  const [mergeMethod, setMergeMethod] = useState<MergeMethod | null>(null)
  const [merging, setMerging] = useState(false)

  const canMerge = useMemo(() => mergeableNow(details), [details])
  const blockReason = useMemo(() => mergeBlockedReason(details), [details])

  const prState = useMemo<PRState>(() => derivePRState(details), [details])
  const reviewState = useMemo<ReviewState>(
    () => deriveReviewState(details),
    [details]
  )

  const handleRequestMerge = useCallback((method: MergeMethod) => {
    setMergeMethod(method)
  }, [])

  const handleConfirmMerge = useCallback(async () => {
    if (!mergeMethod || !details) return
    setMerging(true)
    try {
      await mergePR(prID, mergeMethod)
      toast.success(
        `PR ${mergeMethod === "squash" ? "squash-merged" : "merged"} com sucesso`
      )
      setMergeMethod(null)
      onBack()
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "erro ao fazer merge")
    } finally {
      setMerging(false)
      void reload()
    }
  }, [mergeMethod, details, prID, onBack, reload])

  if (loading && !details) {
    return <LoadingSkeleton onBack={onBack} />
  }

  if (error || !details) {
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
          <div className="text-sm text-destructive">
            {error ?? "não foi possível carregar o PR"}
          </div>
          <button
            type="button"
            onClick={() => void reload()}
            className="text-xs underline"
          >
            tentar de novo
          </button>
        </div>
      </div>
    )
  }

  const totalLines = details.additions + details.deletions
  const diffTooBig = totalLines > DETAILS_DIFF_LIMIT
  const diffEmpty = !diffTooBig && (diff === null || diff === "")

  return (
    <div className="flex h-screen flex-col gap-3 overflow-y-auto bg-background p-3 text-foreground">
      <PRDetailsHeader
        details={details}
        prState={prState}
        reviewState={reviewState}
        onBack={onBack}
        canMerge={canMerge}
        mergeBlockReason={blockReason}
        merging={merging}
        onRequestMerge={handleRequestMerge}
      />

      <PRDetailsMeta details={details} />

      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
        <span>
          <span className="font-mono text-emerald-600 dark:text-emerald-400">
            +{details.additions}
          </span>{" "}
          <span className="font-mono text-rose-600 dark:text-rose-400">
            −{details.deletions}
          </span>
        </span>
        <span aria-hidden="true">·</span>
        <span>{details.changedFiles} arquivos</span>
        <span aria-hidden="true">·</span>
        <span>{details.reviews.length} reviews</span>
      </div>

      <PRDetailsChecks checks={details.statusChecks} />

      <section className="space-y-1">
        <h2 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          Reviewers
        </h2>
        <PRDetailsReviewers reviews={details.reviews} />
      </section>

      <section className="space-y-1">
        <h2 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          Descrição
        </h2>
        <PRDetailsBody body={details.body} />
      </section>

      <section className="space-y-1">
        <h2 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          Arquivos ({details.changedFiles})
        </h2>
        <PRDetailsFiles files={details.files} />
      </section>

      <section className="space-y-1">
        <h2 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          Diff
        </h2>
        {diffTooBig ? (
          <BigPRPlaceholder url={details.url} totalLines={totalLines} />
        ) : diffEmpty ? (
          <div className="text-xs text-muted-foreground italic">diff vazio</div>
        ) : (
          <PRDetailsDiff diff={diff ?? ""} />
        )}
      </section>

      <PRMergeDialog
        open={mergeMethod !== null}
        onOpenChange={(open) => {
          if (!open && !merging) setMergeMethod(null)
        }}
        prNumber={details.number}
        prTitle={details.title}
        method={mergeMethod}
        onConfirm={() => void handleConfirmMerge()}
        busy={merging}
      />
    </div>
  )
}

function LoadingSkeleton({ onBack }: { onBack: () => void }) {
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

function BigPRPlaceholder({
  url,
  totalLines,
}: {
  url: string
  totalLines: number
}) {
  return (
    <div className="flex flex-col items-start gap-2 rounded-lg border border-dashed border-border bg-muted/40 p-3 text-xs">
      <div className="text-muted-foreground">
        PR grande — {totalLines} linhas alteradas, acima do limite de{" "}
        {DETAILS_DIFF_LIMIT}. Abra no GitHub para revisar o diff completo.
      </div>
      <button
        type="button"
        onClick={() => void openPRInBrowser(url)}
        className="text-primary underline"
      >
        Abrir no GitHub →
      </button>
    </div>
  )
}

function derivePRState(details: PRFullDetails | null): PRState {
  if (!details) return "OPEN"
  if (details.state === "MERGED") return "MERGED"
  if (details.state === "CLOSED") return "CLOSED"
  if (details.isDraft) return "DRAFT"
  return "OPEN"
}

function deriveReviewState(details: PRFullDetails | null): ReviewState {
  if (!details) return "PENDING"
  // Don't try to figure out which review is "mine" here — the details view
  // mirrors what the PR card on the list already shows, but we have no
  // viewer-id on this payload. Pick the most actionable state across
  // reviewers so the badge stays informative.
  const order = ["CHANGES_REQUESTED", "APPROVED", "COMMENTED"] as const
  for (const want of order) {
    if (details.reviews.some((r) => r.state === want)) return want
  }
  return "PENDING"
}

function mergeableNow(details: PRFullDetails | null): boolean {
  if (!details) return false
  if (details.state !== "OPEN") return false
  if (details.isDraft) return false
  if (details.mergeable !== "MERGEABLE") return false
  const failing = details.statusChecks.some((c) => {
    const k = c.conclusion.toUpperCase()
    return k === "FAILURE" || k === "TIMED_OUT" || k === "CANCELLED"
  })
  return !failing
}

function mergeBlockedReason(details: PRFullDetails | null): string | null {
  if (!details) return null
  if (details.state === "MERGED") return "PR já foi merged"
  if (details.state === "CLOSED") return "PR fechado"
  if (details.isDraft) return "PR está em draft"
  if (details.mergeable === "CONFLICTING")
    return "conflitos — resolva pelo GitHub"
  if (details.mergeable === "UNKNOWN")
    return "GitHub ainda está checando se pode merge"
  const failing = details.statusChecks.some((c) => {
    const k = c.conclusion.toUpperCase()
    return k === "FAILURE" || k === "TIMED_OUT" || k === "CANCELLED"
  })
  if (failing) return "algum check falhou"
  return null
}
