import path from "node:path"
import { fileURLToPath } from "node:url"

import { defineConfig } from "vitest/config"

const dirname = path.dirname(fileURLToPath(import.meta.url))

export default defineConfig({
  resolve: {
    alias: [
      {
        find: /^@\/wailsjs\/(.*)$/,
        replacement: path.resolve(dirname, "wailsjs/$1"),
      },
      { find: "@", replacement: path.resolve(dirname, "src") },
    ],
  },
  test: {
    environment: "jsdom",
    globals: false,
    setupFiles: ["./src/test/setup.ts"],
    css: false,
    include: ["src/**/*.test.{ts,tsx}"],
    coverage: {
      provider: "v8",
      reporter: ["text", "html"],
      // Cobertura inicial (REV-37). O escopo cresce conforme novos testes
      // chegam — espelha a estratégia incremental do REV-29 no Go.
      include: [
        "src/components/pr-card.tsx",
        "src/hooks/use-prs.ts",
        "src/lib/format/**/*.ts",
        "src/lib/parse-diff.ts",
        "src/lib/pr-state.ts",
        "src/lib/types.ts",
      ],
      exclude: [
        "src/wailsjs/**",
        "src/generated/**",
        "src/routeTree.gen.ts",
        "src/test/**",
        "src/**/*.test.{ts,tsx}",
        "src/main.tsx",
        "src/components/ui/**",
      ],
      thresholds: {
        lines: 60,
      },
    },
  },
})
