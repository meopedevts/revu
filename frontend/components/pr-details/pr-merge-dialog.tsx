import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import type { MergeMethod } from "@/src/lib/types"

interface PRMergeDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  prNumber: number
  prTitle: string
  method: MergeMethod | null
  onConfirm: () => void
  busy: boolean
}

export function PRMergeDialog({
  open,
  onOpenChange,
  prNumber,
  prTitle,
  method,
  onConfirm,
  busy,
}: PRMergeDialogProps) {
  const label = method === "squash" ? "Squash & merge" : "Merge commit"
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Confirmar {label}?</AlertDialogTitle>
          <AlertDialogDescription asChild>
            <div className="space-y-1">
              <div className="font-mono text-xs text-muted-foreground">
                #{prNumber}
              </div>
              <div className="text-sm">{prTitle}</div>
              <div className="pt-2 text-xs text-muted-foreground">
                Método: <span className="font-medium">{label}</span>
              </div>
            </div>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={busy}>Cancelar</AlertDialogCancel>
          <AlertDialogAction
            onClick={(e) => {
              e.preventDefault()
              onConfirm()
            }}
            disabled={busy}
          >
            {busy ? "Executando…" : "Confirmar"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
