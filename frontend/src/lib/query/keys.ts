export const queryKeys = {
  prs: {
    all: ["prs"] as const,
    pending: ["prs", "pending"] as const,
    history: ["prs", "history"] as const,
    detail: (id: string) => ["prs", "detail", id] as const,
    diff: (id: string) => ["prs", "diff", id] as const,
  },
  poll: {
    meta: ["poll", "meta"] as const,
  },
  tray: {
    acknowledgedAt: ["tray", "acknowledgedAt"] as const,
  },
  profiles: {
    all: ["profiles"] as const,
    list: ["profiles", "list"] as const,
    active: ["profiles", "active"] as const,
  },
  config: ["config"] as const,
} as const
