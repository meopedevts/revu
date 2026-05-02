import type { ChangedFile } from "@/lib/types"

interface PRDetailsFilesProps {
  files: ChangedFile[]
}

export function PRDetailsFiles({ files }: PRDetailsFilesProps) {
  if (files.length === 0) {
    return (
      <div className="text-xs text-muted-foreground italic">sem arquivos</div>
    )
  }
  const sorted = [...files].sort(
    (a, b) => b.additions + b.deletions - (a.additions + a.deletions)
  )
  return (
    <ul className="flex flex-col divide-y divide-border rounded-lg border border-border bg-card">
      {sorted.map((f) => (
        <li
          key={f.path}
          className="flex items-center justify-between gap-2 px-3 py-1.5 text-xs leading-snug"
        >
          <span className="truncate font-mono">{f.path}</span>
          <span className="flex shrink-0 items-center gap-2 font-mono font-medium">
            <span className="text-emerald-600 dark:text-emerald-400">
              +{f.additions}
            </span>
            <span className="text-rose-600 dark:text-rose-400">
              −{f.deletions}
            </span>
          </span>
        </li>
      ))}
    </ul>
  )
}
