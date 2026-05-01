import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import type { PRRecord } from "@/lib/types"

import { PRCard } from "./pr-card"

function makePR(overrides: Partial<PRRecord> = {}): PRRecord {
  return {
    id: "pr-42",
    number: 42,
    repo: "owner/repo",
    title: "fix: bug crítico",
    author: "alice",
    url: "https://github.com/owner/repo/pull/42",
    state: "OPEN",
    is_draft: false,
    additions: 12,
    deletions: 3,
    review_pending: true,
    review_state: "PENDING",
    first_seen_at: "2026-04-30T11:55:00Z",
    last_seen_at: "2026-04-30T11:55:00Z",
    ...overrides,
  }
}

beforeEach(() => {
  vi.useFakeTimers()
  vi.setSystemTime(new Date("2026-04-30T12:00:00.000Z"))
})

afterEach(() => {
  vi.useRealTimers()
})

describe("PRCard", () => {
  it("renderiza número, título, repo, autor e contadores de diff", () => {
    render(<PRCard pr={makePR()} onOpen={vi.fn()} />)

    expect(screen.getByText("#42")).toBeInTheDocument()
    expect(screen.getByText("fix: bug crítico")).toBeInTheDocument()
    expect(screen.getByText("owner/repo")).toBeInTheDocument()
    expect(screen.getByText("@alice")).toBeInTheDocument()
    expect(screen.getByText("+12")).toBeInTheDocument()
    expect(screen.getByText("−3")).toBeInTheDocument()
  })

  it("renderiza relTime baseado em last_seen_at", () => {
    render(<PRCard pr={makePR()} onOpen={vi.fn()} />)
    expect(screen.getByText(/visto há 5min/)).toBeInTheDocument()
  })

  it("renderiza badges StatusBadge e ReviewBadge", () => {
    render(<PRCard pr={makePR()} onOpen={vi.fn()} />)
    expect(screen.getByText("OPEN")).toBeInTheDocument()
    expect(screen.getByText("revisar")).toBeInTheDocument()
  })

  it("usa role=button e tabIndex=0 pra acessibilidade", () => {
    render(<PRCard pr={makePR()} onOpen={vi.fn()} />)
    const card = screen.getByRole("button")
    expect(card).toHaveAttribute("tabIndex", "0")
  })

  it("dispara onOpen com pr.id ao clicar", async () => {
    vi.useRealTimers()
    const onOpen = vi.fn()
    const user = userEvent.setup()
    render(<PRCard pr={makePR({ id: "pr-99" })} onOpen={onOpen} />)

    await user.click(screen.getByRole("button"))
    expect(onOpen).toHaveBeenCalledExactlyOnceWith("pr-99")
  })

  it("dispara onOpen quando Enter é pressionado", async () => {
    vi.useRealTimers()
    const onOpen = vi.fn()
    const user = userEvent.setup()
    render(<PRCard pr={makePR({ id: "pr-99" })} onOpen={onOpen} />)

    const card = screen.getByRole("button")
    card.focus()
    await user.keyboard("{Enter}")
    expect(onOpen).toHaveBeenCalledExactlyOnceWith("pr-99")
  })

  it("dispara onOpen quando Space é pressionado", async () => {
    vi.useRealTimers()
    const onOpen = vi.fn()
    const user = userEvent.setup()
    render(<PRCard pr={makePR({ id: "pr-99" })} onOpen={onOpen} />)

    const card = screen.getByRole("button")
    card.focus()
    await user.keyboard(" ")
    expect(onOpen).toHaveBeenCalledExactlyOnceWith("pr-99")
  })

  it("não dispara onOpen pra outras teclas", async () => {
    vi.useRealTimers()
    const onOpen = vi.fn()
    const user = userEvent.setup()
    render(<PRCard pr={makePR()} onOpen={onOpen} />)

    const card = screen.getByRole("button")
    card.focus()
    await user.keyboard("a")
    expect(onOpen).not.toHaveBeenCalled()
  })

  it("renderiza badge DRAFT quando is_draft=true", () => {
    render(<PRCard pr={makePR({ is_draft: true })} onOpen={vi.fn()} />)
    expect(screen.getByText("DRAFT")).toBeInTheDocument()
  })

  it("renderiza badge APPROVED quando review_state=APPROVED", () => {
    render(
      <PRCard pr={makePR({ review_state: "APPROVED" })} onOpen={vi.fn()} />
    )
    expect(screen.getByText("aprovado")).toBeInTheDocument()
  })
})
