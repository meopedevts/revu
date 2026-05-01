import { useCallback, useMemo, useState } from "react"
import { toast } from "sonner"

import { useMergePR } from "@/hooks/mutations/use-merge-pr"
import { usePRDetails } from "@/hooks/use-pr-details"
import { type MergeMethod, type PRState, type ReviewState } from "@/lib/types"

import { PRDetailsErrorState } from "./error-state"
import {
  derivePRState,
  deriveReviewState,
  mergeableNow,
  mergeBlockedReason,
} from "./lib/derive-state"
import { PRDetailsLoadingSkeleton } from "./loading-skeleton"
import { PRDetailsBody } from "./pr-details-body"
import { PRDetailsChecks } from "./pr-details-checks"
import { PRDetailsDiffSection } from "./pr-details-diff-section"
import { PRDetailsFiles } from "./pr-details-files"
import { PRDetailsHeader } from "./pr-details-header"
import { PRDetailsMeta } from "./pr-details-meta"
import { PRDetailsReviewers } from "./pr-details-reviewers"
import { PRDetailsStats } from "./pr-details-stats"
import { PRMergeDialog } from "./pr-merge-dialog"

interface PRDetailsViewProps {
  prID: string
  onBack: () => void
}

export function PRDetailsView({ prID, onBack }: PRDetailsViewProps) {
  const { details, diff, diffError, loading, error, reload } =
    usePRDetails(prID)
  const mergeMutation = useMergePR()
  const merging = mergeMutation.isPending

  const [mergeMethod, setMergeMethod] = useState<MergeMethod | null>(null)

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
    try {
      await mergeMutation.mutateAsync({ prID, method: mergeMethod })
      toast.success(
        `PR ${mergeMethod === "squash" ? "squash-merged" : "merged"} com sucesso`
      )
      setMergeMethod(null)
      onBack()
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "erro ao fazer merge")
      void reload()
    }
  }, [mergeMethod, details, prID, mergeMutation, onBack, reload])

  if (loading && !details) {
    return <PRDetailsLoadingSkeleton onBack={onBack} />
  }

  if (error || !details) {
    return (
      <PRDetailsErrorState
        message={error ?? "não foi possível carregar o PR"}
        onBack={onBack}
        onRetry={() => void reload()}
      />
    )
  }

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

      <PRDetailsStats
        additions={details.additions}
        deletions={details.deletions}
        changedFiles={details.changedFiles}
        reviewCount={details.reviews.length}
      />

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

      <PRDetailsDiffSection
        url={details.url}
        additions={details.additions}
        deletions={details.deletions}
        diff={diff}
        diffError={diffError}
      />

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
