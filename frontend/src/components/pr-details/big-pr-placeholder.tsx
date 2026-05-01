import { openPRInBrowser } from "@/bridge"
import { DETAILS_DIFF_LIMIT } from "@/generated/constants"

interface PRDetailsBigPRPlaceholderProps {
  url: string
  totalLines: number
}

export function PRDetailsBigPRPlaceholder({
  url,
  totalLines,
}: PRDetailsBigPRPlaceholderProps) {
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
