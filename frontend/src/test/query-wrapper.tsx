import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { type ReactNode } from "react"

export function createQueryWrapper(): {
  wrapper: ({ children }: { children: ReactNode }) => JSX.Element
  client: QueryClient
} {
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0, staleTime: 0 },
      mutations: { retry: false },
    },
  })
  function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }
  return { wrapper: Wrapper, client }
}
