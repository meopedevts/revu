import { useRouter, type ErrorComponentProps } from "@tanstack/react-router"
import { useEffect } from "react"

import { Button } from "@/components/ui/button"

export function RouteErrorFallback({ error, reset }: ErrorComponentProps) {
  const router = useRouter()

  useEffect(() => {
    console.error("[ErrorBoundary]", error)
  }, [error])

  const isError = error instanceof Error
  const message = isError ? error.message : "Erro inesperado"
  const stack = isError ? error.stack : undefined

  return (
    <div className="flex h-screen flex-col items-center justify-center gap-3 bg-background p-4 text-center">
      <h1 className="text-sm font-semibold text-destructive">
        Algo deu errado
      </h1>
      <p className="max-w-md text-xs text-muted-foreground">{message}</p>
      <div className="flex items-center gap-2">
        <Button size="sm" onClick={reset}>
          Recarregar
        </Button>
        <Button
          size="sm"
          variant="outline"
          onClick={() => router.history.back()}
        >
          Voltar
        </Button>
      </div>
      {import.meta.env.DEV && stack ? (
        <details className="mt-2 max-w-2xl text-left">
          <summary className="cursor-pointer text-xs text-muted-foreground">
            Detalhes
          </summary>
          <pre className="mt-2 overflow-auto rounded-md bg-muted p-2 text-[0.7rem] leading-relaxed text-muted-foreground">
            {stack}
          </pre>
        </details>
      ) : null}
    </div>
  )
}
