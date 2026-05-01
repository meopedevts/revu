import { act, renderHook, waitFor } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { POLL_SAFETY_INTERVAL_MS } from "@/generated/constants"
import type { PRRecord } from "@/lib/types"

import { usePRs } from "./use-prs"

const listPendingPRs = vi.fn<() => Promise<PRRecord[]>>()
const listHistoryPRs = vi.fn<() => Promise<PRRecord[]>>()

vi.mock("@/bridge", () => ({
  listPendingPRs: () => listPendingPRs(),
  listHistoryPRs: () => listHistoryPRs(),
}))

type EventHandler = (raw: unknown) => void
const handlers = new Map<string, EventHandler>()

vi.mock("@/wailsjs/runtime/runtime", () => ({
  EventsOn: (name: string, fn: EventHandler) => {
    handlers.set(name, fn)
  },
  EventsOff: (name: string) => {
    handlers.delete(name)
  },
}))

function pr(id: string): PRRecord {
  return {
    id,
    number: 1,
    repo: "owner/repo",
    title: id,
    author: "alice",
    url: `https://github.com/owner/repo/pull/${id}`,
    state: "OPEN",
    is_draft: false,
    additions: 0,
    deletions: 0,
    review_pending: true,
    review_state: "PENDING",
    first_seen_at: "2026-01-01T00:00:00Z",
    last_seen_at: "2026-01-01T00:00:00Z",
  }
}

beforeEach(() => {
  handlers.clear()
  listPendingPRs.mockReset()
  listHistoryPRs.mockReset()
  listPendingPRs.mockResolvedValue([pr("p1")])
  listHistoryPRs.mockResolvedValue([pr("h1")])
})

afterEach(() => {
  vi.useRealTimers()
})

describe("usePRs", () => {
  it("popula pending/history no load inicial", async () => {
    const { result } = renderHook(() => usePRs())

    await waitFor(() => {
      expect(result.current.pending).toHaveLength(1)
    })
    expect(result.current.pending[0].id).toBe("p1")
    expect(result.current.history[0].id).toBe("h1")
    expect(listPendingPRs).toHaveBeenCalledTimes(1)
  })

  it("registra handlers pra pr:new, pr:status-changed e poll:completed", async () => {
    renderHook(() => usePRs())
    await waitFor(() => {
      expect(handlers.has("pr:new")).toBe(true)
      expect(handlers.has("pr:status-changed")).toBe(true)
      expect(handlers.has("poll:completed")).toBe(true)
    })
  })

  it("dispara reload quando 'pr:new' chega", async () => {
    renderHook(() => usePRs())
    await waitFor(() => {
      expect(listPendingPRs).toHaveBeenCalledTimes(1)
    })

    act(() => {
      handlers.get("pr:new")?.(undefined)
    })

    await waitFor(() => {
      expect(listPendingPRs).toHaveBeenCalledTimes(2)
    })
  })

  it("seta lastPollAt e lastPollErr a partir de 'poll:completed'", async () => {
    const { result } = renderHook(() => usePRs())
    await waitFor(() => {
      expect(handlers.has("poll:completed")).toBe(true)
    })

    act(() => {
      handlers.get("poll:completed")?.({
        kind: "scheduled",
        at: "2026-04-30T12:00:00.000Z",
        err: "rate limit",
      })
    })

    await waitFor(() => {
      expect(result.current.lastPollAt?.toISOString()).toBe(
        "2026-04-30T12:00:00.000Z"
      )
    })
    expect(result.current.lastPollErr).toBe("rate limit")
  })

  it("'poll:completed' sem 'at' usa Date.now() como fallback", async () => {
    const { result } = renderHook(() => usePRs())
    await waitFor(() => {
      expect(handlers.has("poll:completed")).toBe(true)
    })

    act(() => {
      handlers.get("poll:completed")?.({ kind: "scheduled", at: "" })
    })

    await waitFor(() => {
      expect(result.current.lastPollAt).toBeInstanceOf(Date)
    })
    expect(result.current.lastPollErr).toBeNull()
  })

  it("dedupe: reloads concorrentes compartilham promessa inflight", async () => {
    let resolvePending: ((v: PRRecord[]) => void) | null = null
    listPendingPRs.mockImplementation(
      () =>
        new Promise<PRRecord[]>((resolve) => {
          resolvePending = resolve
        })
    )
    listHistoryPRs.mockResolvedValue([])

    const { result } = renderHook(() => usePRs())
    await waitFor(() => {
      expect(listPendingPRs).toHaveBeenCalledTimes(1)
    })

    act(() => {
      void result.current.reload()
      void result.current.reload()
    })

    expect(listPendingPRs).toHaveBeenCalledTimes(1)
    act(() => {
      resolvePending?.([pr("p1")])
    })
  })

  it("safety interval dispara reload após POLL_SAFETY_INTERVAL_MS", async () => {
    vi.useFakeTimers({ toFake: ["setInterval", "clearInterval"] })
    renderHook(() => usePRs())

    await waitFor(() => {
      expect(listPendingPRs).toHaveBeenCalledTimes(1)
    })

    const before = listPendingPRs.mock.calls.length

    act(() => {
      vi.advanceTimersByTime(POLL_SAFETY_INTERVAL_MS)
    })

    await waitFor(() => {
      expect(listPendingPRs.mock.calls.length).toBeGreaterThan(before)
    })
  })

  it("unmount limpa handlers de eventos", async () => {
    const { unmount } = renderHook(() => usePRs())
    await waitFor(() => {
      expect(handlers.has("pr:new")).toBe(true)
    })

    unmount()
    expect(handlers.has("pr:new")).toBe(false)
    expect(handlers.has("pr:status-changed")).toBe(false)
    expect(handlers.has("poll:completed")).toBe(false)
  })
})
