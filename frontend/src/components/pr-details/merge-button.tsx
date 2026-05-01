import { GitMerge, Loader2 } from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { cn } from "@/lib/utils"

interface MergeButtonProps {
  label: string
  method: "squash" | "merge"
  primary: boolean
  canMerge: boolean
  reason: string | null
  merging: boolean
  onClick: (method: "squash" | "merge") => void
}

export function MergeButton({
  label,
  method,
  primary,
  canMerge,
  reason,
  merging,
  onClick,
}: MergeButtonProps) {
  const disabled = !canMerge || merging
  const btn = (
    <Button
      size="sm"
      variant={primary ? "default" : "outline"}
      disabled={disabled}
      onClick={() => onClick(method)}
      className={cn(primary ? "bg-status-merged text-white" : "")}
    >
      {merging ? (
        <Loader2 className="animate-spin" data-icon="inline-start" />
      ) : (
        <GitMerge data-icon="inline-start" />
      )}
      {label}
    </Button>
  )
  if (disabled && reason && !merging) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          {/* eslint-disable-next-line jsx-a11y/no-noninteractive-tabindex -- wrapper precisa receber foco pro Tooltip funcionar com botão disabled (padrão Radix) */}
          <span tabIndex={0}>{btn}</span>
        </TooltipTrigger>
        <TooltipContent>{reason}</TooltipContent>
      </Tooltip>
    )
  }
  return btn
}
