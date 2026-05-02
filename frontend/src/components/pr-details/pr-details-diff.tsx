import { ChevronDown, ChevronRight } from "lucide-react"
import { useMemo, useState } from "react"
import ReactDiffViewer from "react-diff-viewer-continued"

import { type FileDiff, parseDiff } from "@/lib/parse-diff"
import { useTheme } from "@/lib/theme/theme-provider"
import { cn } from "@/lib/utils"

// Tokens semânticos (style.css §REV-56). CSS vars resolvem no tema corrente
// do <html>, então o mesmo conjunto serve pra variables.{light,dark} — o
// dark-mode toggle da lib só decide qual block é injetado, mas as vars
// reagem ao tema do app via cascade.
const DIFF_VARS = {
  diffViewerBackground: "var(--card)",
  diffViewerColor: "var(--card-foreground)",
  addedBackground: "var(--diff-added-bg)",
  addedColor: "var(--diff-added)",
  removedBackground: "var(--diff-removed-bg)",
  removedColor: "var(--diff-removed)",
  wordAddedBackground: "var(--diff-added-line)",
  wordRemovedBackground: "var(--diff-removed-line)",
  addedGutterBackground: "var(--diff-added-line)",
  removedGutterBackground: "var(--diff-removed-line)",
  gutterBackground: "var(--muted)",
  gutterBackgroundDark: "var(--muted)",
  gutterColor: "var(--muted-foreground)",
  addedGutterColor: "var(--diff-added)",
  removedGutterColor: "var(--diff-removed)",
  codeFoldBackground: "var(--muted)",
  codeFoldGutterBackground: "var(--muted)",
  codeFoldContentColor: "var(--muted-foreground)",
  emptyLineBackground: "var(--card)",
  highlightBackground: "var(--accent)",
  highlightGutterBackground: "var(--accent)",
}

const DIFF_STYLES = {
  variables: { dark: DIFF_VARS, light: DIFF_VARS },
  contentText: {
    fontSize: "0.78rem",
    fontFamily: "var(--font-mono)",
  },
  gutter: { fontSize: "0.7rem", padding: "0 0.5rem" },
  line: { fontFamily: "var(--font-mono)" },
}

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
  const { theme } = useTheme()
  const Chevron = open ? ChevronDown : ChevronRight
  return (
    <div className="rounded-lg border border-border bg-card">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className={cn(
          "flex w-full items-center gap-2 px-3 py-2 text-left text-xs",
          "hover:bg-muted"
        )}
      >
        <Chevron className="size-3 shrink-0" aria-hidden="true" />
        <span className="truncate font-mono">{file.path}</span>
      </button>
      {open && (
        <div className="overflow-x-auto border-t border-border">
          <ReactDiffViewer
            key={theme}
            oldValue={file.oldContent}
            newValue={file.newContent}
            splitView={false}
            useDarkTheme={theme === "dark"}
            hideLineNumbers={false}
            disableWordDiff={false}
            styles={DIFF_STYLES}
          />
        </div>
      )}
    </div>
  )
}
