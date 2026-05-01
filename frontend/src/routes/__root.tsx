import { Outlet, createRootRoute, useRouter } from "@tanstack/react-router"
import { lazy, Suspense } from "react"

import { RouteErrorFallback } from "@/components/route-error-fallback"
import { isSettingsSection } from "@/components/settings/settings-sidebar"
import { Toaster } from "@/components/ui/sonner"
import { TooltipProvider } from "@/components/ui/tooltip"
import { useEscapeKey } from "@/hooks/use-escape-key"
import { useWailsEvent } from "@/hooks/use-wails-event"

interface NavigatePayload {
  target: string
  section?: string
}

const TanStackRouterDevtools = import.meta.env.DEV
  ? lazy(() =>
      import("@tanstack/react-router-devtools").then((m) => ({
        default: m.TanStackRouterDevtools,
      }))
    )
  : null

export const Route = createRootRoute({
  component: RootComponent,
  errorComponent: RouteErrorFallback,
  notFoundComponent: () => (
    <div className="flex h-screen flex-col items-center justify-center gap-2 bg-background p-4 text-center">
      <h1 className="text-sm font-semibold">Rota não encontrada</h1>
      <p className="text-xs text-muted-foreground">
        Volte para a tela principal.
      </p>
    </div>
  ),
})

function RootComponent() {
  const router = useRouter()

  useWailsEvent<NavigatePayload | undefined>("ui:navigate", (payload) => {
    if (!payload) return
    if (payload.target === "settings") {
      const raw = payload.section?.trim() ?? ""
      const section = isSettingsSection(raw) ? raw : "sync"
      void router.navigate({
        to: "/settings/$section",
        params: { section },
      })
    } else if (payload.target === "main") {
      void router.navigate({ to: "/" })
    }
  })

  useEscapeKey(
    () => router.history.back(),
    router.state.location.pathname !== "/"
  )

  return (
    <TooltipProvider delayDuration={300}>
      <Outlet />
      <Toaster />
      {TanStackRouterDevtools ? (
        <Suspense fallback={null}>
          <TanStackRouterDevtools />
        </Suspense>
      ) : null}
    </TooltipProvider>
  )
}
