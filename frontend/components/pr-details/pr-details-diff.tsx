import { useMemo, useState } from 'react'
import ReactDiffViewer from 'react-diff-viewer-continued'
import { ChevronDown, ChevronRight } from 'lucide-react'

import { cn } from '@/lib/utils'
import { type FileDiff, parseDiff } from './parse-diff'

interface PRDetailsDiffProps {
  diff: string
}

export function PRDetailsDiff({ diff }: PRDetailsDiffProps) {
  const files = useMemo(() => parseDiff(diff), [diff])
  if (files.length === 0) {
    return (
      <div className="text-xs text-muted-foreground italic">
        diff vazio — pode ser PR só com mudança de metadata ou binário
      </div>
    )
  }
  return (
    <div className="flex flex-col gap-2">
      {files.map((f) => (
        <FileDiffBlock key={f.path} file={f} />
      ))}
    </div>
  )
}

function FileDiffBlock({ file }: { file: FileDiff }) {
  const [open, setOpen] = useState(false)
  const Chevron = open ? ChevronDown : ChevronRight
  return (
    <div className="rounded-lg border border-border bg-card">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className={cn(
          'flex w-full items-center gap-2 px-3 py-2 text-left text-xs',
          'hover:bg-muted',
        )}
      >
        <Chevron className="size-3 shrink-0" aria-hidden="true" />
        <span className="truncate font-mono">{file.path}</span>
      </button>
      {open && (
        <div className="overflow-x-auto border-t border-border">
          <ReactDiffViewer
            oldValue={file.oldContent}
            newValue={file.newContent}
            splitView={false}
            useDarkTheme
            hideLineNumbers={false}
            disableWordDiff={false}
            styles={{
              contentText: { fontSize: '0.78rem' },
              gutter: { fontSize: '0.7rem', padding: '0 0.5rem' },
            }}
          />
        </div>
      )}
    </div>
  )
}
