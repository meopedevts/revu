interface PRDetailsStatsProps {
  additions: number
  deletions: number
  changedFiles: number
  reviewCount: number
}

export function PRDetailsStats({
  additions,
  deletions,
  changedFiles,
  reviewCount,
}: PRDetailsStatsProps) {
  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs leading-snug text-muted-foreground">
      <span>
        <span className="font-mono font-medium text-emerald-600 dark:text-emerald-400">
          +{additions}
        </span>{" "}
        <span className="font-mono font-medium text-rose-600 dark:text-rose-400">
          −{deletions}
        </span>
      </span>
      <span aria-hidden="true">·</span>
      <span>{changedFiles} arquivos</span>
      <span aria-hidden="true">·</span>
      <span>{reviewCount} reviews</span>
    </div>
  )
}
