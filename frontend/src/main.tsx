import { QueryClientProvider } from "@tanstack/react-query"
import {
  RouterProvider,
  createHashHistory,
  createRouter,
} from "@tanstack/react-router"
import React, { lazy, Suspense } from "react"
import { createRoot } from "react-dom/client"

import "./style.css"
import { queryClient } from "@/lib/query/client"
import { ThemeProvider } from "@/lib/theme/theme-provider"
import { routeTree } from "@/routeTree.gen"

const ReactQueryDevtools = import.meta.env.DEV
  ? lazy(() =>
      import("@tanstack/react-query-devtools").then((m) => ({
        default: m.ReactQueryDevtools,
      }))
    )
  : null

const router = createRouter({
  routeTree,
  history: createHashHistory(),
  defaultPreload: "intent",
})

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router
  }
}

const container = document.getElementById("root")

const root = createRoot(container!)

root.render(
  <React.StrictMode>
    <ThemeProvider>
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
        {ReactQueryDevtools ? (
          <Suspense fallback={null}>
            <ReactQueryDevtools initialIsOpen={false} />
          </Suspense>
        ) : null}
      </QueryClientProvider>
    </ThemeProvider>
  </React.StrictMode>
)
