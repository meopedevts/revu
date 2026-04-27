import {
  RouterProvider,
  createHashHistory,
  createRouter,
} from "@tanstack/react-router"
import React from "react"
import { createRoot } from "react-dom/client"

import "./style.css"
import { ThemeProvider } from "@/lib/theme/theme-provider"
import { routeTree } from "@/routeTree.gen"

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
      <RouterProvider router={router} />
    </ThemeProvider>
  </React.StrictMode>
)
