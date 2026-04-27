import path from "node:path"
import { fileURLToPath } from "node:url"

import { tanstackRouter } from "@tanstack/router-plugin/vite"
import { defineConfig } from "vite"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"

const dirname = path.dirname(fileURLToPath(import.meta.url))

export default defineConfig({
  plugins: [
    tanstackRouter({
      target: "react",
      autoCodeSplitting: true,
      routesDirectory: "./src/routes",
      generatedRouteTree: "./src/routeTree.gen.ts",
    }),
    react(),
    tailwindcss(),
  ],
  resolve: {
    alias: [
      {
        find: /^@\/wailsjs\/(.*)$/,
        replacement: path.resolve(dirname, "wailsjs/$1"),
      },
      { find: "@", replacement: path.resolve(dirname, "src") },
    ],
  },
})
