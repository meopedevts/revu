import { act, renderHook, waitFor } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import type { PRRecord } from "@/lib/types"
import { createQueryWrapper } from "@/test/query-wrapper"

import { usePRs } from "./use-prs"

const listPendingPRs = vi.fn<() => Promise<PRRecord[]>>()
const listHistoryPRs = vi.fn<() => Promise<PRRecord[]>>()

vi.mock("@/bridge", () => ({
  listPendingPRs: () => listPendingPRs(),
  listHistoryPRs: () => listHistoryPRs(),
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
    isDraft: false,
    additions: 0,
    deletions: 0,
    reviewPending: true,
    reviewState: "PENDING",
    firstSeenAt: "2026-01-01T00:00:00Z",
    lastSeenAt: "2026-01-01T00:00:00Z",
  }
}

beforeEach(() => {
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
    const { wrapper } = createQueryWrapper()
    const { result } = renderHook(() => usePRs(), { wrapper })

    await waitFor(() => {
      expect(result.current.pending).toHaveLength(1)
    })
    expect(result.current.pending[0].id).toBe("p1")
    expect(result.current.history[0].id).toBe("h1")
    expect(listPendingPRs).toHaveBeenCalledTimes(1)
    expect(listHistoryPRs).toHaveBeenCalledTimes(1)
  })

  it("reload invalida queries e re-fetcha pending+history", async () => {
    const { wrapper } = createQueryWrapper()
    const { result } = renderHook(() => usePRs(), { wrapper })

    await waitFor(() => {
      expect(listPendingPRs).toHaveBeenCalledTimes(1)
    })

    await act(async () => {
      await result.current.reload()
    })

    await waitFor(() => {
      expect(listPendingPRs).toHaveBeenCalledTimes(2)
      expect(listHistoryPRs).toHaveBeenCalledTimes(2)
    })
  })

  it("lê lastPollAt/lastPollErr do cache poll.meta", async () => {
    const { wrapper, client } = createQueryWrapper()
    const at = new Date("2026-04-30T12:00:00.000Z")
    client.setQueryData(["poll", "meta"], { at, err: "rate limit" })

    const { result } = renderHook(() => usePRs(), { wrapper })

    await waitFor(() => {
      expect(result.current.lastPollAt?.toISOString()).toBe(at.toISOString())
    })
    expect(result.current.lastPollErr).toBe("rate limit")
  })

  it("default poll meta é null quando cache vazio", async () => {
    const { wrapper } = createQueryWrapper()
    const { result } = renderHook(() => usePRs(), { wrapper })

    await waitFor(() => {
      expect(result.current.pending).toHaveLength(1)
    })
    expect(result.current.lastPollAt).toBeNull()
    expect(result.current.lastPollErr).toBeNull()
  })

  it("dedupe: chamadas concorrentes ao bridge resolvem com 1 fetch", async () => {
    let resolvePending: ((v: PRRecord[]) => void) | null = null
    listPendingPRs.mockImplementation(
      () =>
        new Promise<PRRecord[]>((resolve) => {
          resolvePending = resolve
        })
    )
    listHistoryPRs.mockResolvedValue([])

    const { wrapper } = createQueryWrapper()
    const { result } = renderHook(() => usePRs(), { wrapper })

    await waitFor(() => {
      expect(listPendingPRs).toHaveBeenCalledTimes(1)
    })

    // Duas reloads concorrentes enquanto o primeiro fetch ainda está pending —
    // react-query mantém a query inflight e não dispara fetch extra. Disparo
    // fire-and-forget pra não awaitar invalidateQueries (a queryFn travada
    // nunca resolveria).
    await act(async () => {
      void result.current.reload()
      void result.current.reload()
      await new Promise((r) => setTimeout(r, 50))
    })

    expect(listPendingPRs).toHaveBeenCalledTimes(1)
    act(() => {
      resolvePending?.([pr("p1")])
    })
  })
})
