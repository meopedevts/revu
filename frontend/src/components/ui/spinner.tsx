import { Loader2 } from "lucide-react"

import { cn } from "@/lib/utils"

interface SpinnerProps extends React.SVGAttributes<SVGSVGElement> {
  label?: string
}

function Spinner({ className, label = "carregando", ...props }: SpinnerProps) {
  return (
    <Loader2
      data-slot="spinner"
      role="status"
      aria-label={label}
      className={cn("size-4 animate-spin text-primary", className)}
      {...props}
    />
  )
}

export { Spinner }
