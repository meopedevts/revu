import { Outlet, createRootRoute, useRouter } from "@tanstack/react-router"
import { lazy, Suspense, useEffect } from "react"

import { RouteErrorFallback } from "@/components/route-error-fallback"
import { isSettingsSection } from "@/components/settings/settings-sidebar"
import { Toaster } from "@/components/ui/sonner"
import { TooltipProvider } from "@/components/ui/tooltip"
import { EventsOff, EventsOn } from "@/wailsjs/runtime/runtime"

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

  useEffect(() => {
    EventsOn("ui:navigate", (payload: NavigatePayload | undefined) => {
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
    return () => {
      EventsOff("ui:navigate")
    }
  }, [router])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== "Escape") return
      if (router.state.location.pathname === "/") return
      router.history.back()
    }
    window.addEventListener("keydown", onKey)
    return () => window.removeEventListener("keydown", onKey)
  }, [router])

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
