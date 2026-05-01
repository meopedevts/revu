import { act, renderHook, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import type { Profile } from "@/lib/types"
import { createQueryWrapper } from "@/test/query-wrapper"

import { useGlobalSubscriptions } from "./use-global-subscriptions"

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

beforeEach(() => {
  handlers.clear()
})

describe("useGlobalSubscriptions", () => {
  it("registra handlers pra pr:new, pr:status-changed, poll:completed e profiles:active-changed", async () => {
    const { wrapper } = createQueryWrapper()
    renderHook(() => useGlobalSubscriptions(), { wrapper })

    await waitFor(() => {
      expect(handlers.has("pr:new")).toBe(true)
      expect(handlers.has("pr:status-changed")).toBe(true)
      expect(handlers.has("poll:completed")).toBe(true)
      expect(handlers.has("profiles:active-changed")).toBe(true)
    })
  })

  it("'pr:new' invalida queries da família ['prs']", async () => {
    const { wrapper, client } = createQueryWrapper()
    renderHook(() => useGlobalSubscriptions(), { wrapper })

    client.setQueryData(["prs", "pending"], [{ id: "p1" }])
    const spy = vi.spyOn(client, "invalidateQueries")

    await waitFor(() => expect(handlers.has("pr:new")).toBe(true))

    act(() => {
      handlers.get("pr:new")?.(undefined)
    })

    expect(spy).toHaveBeenCalledWith({ queryKey: ["prs"] })
  })

  it("'poll:completed' grava at/err em ['poll','meta']", async () => {
    const { wrapper, client } = createQueryWrapper()
    renderHook(() => useGlobalSubscriptions(), { wrapper })

    await waitFor(() => expect(handlers.has("poll:completed")).toBe(true))

    act(() => {
      handlers.get("poll:completed")?.({
        kind: "scheduled",
        at: "2026-04-30T12:00:00.000Z",
        err: "rate limit",
      })
    })

    const meta = client.getQueryData<{ at: Date; err: string | null }>([
      "poll",
      "meta",
    ])
    expect(meta?.at.toISOString()).toBe("2026-04-30T12:00:00.000Z")
    expect(meta?.err).toBe("rate limit")
  })

  it("'poll:completed' sem at usa Date.now() como fallback", async () => {
    const { wrapper, client } = createQueryWrapper()
    renderHook(() => useGlobalSubscriptions(), { wrapper })

    await waitFor(() => expect(handlers.has("poll:completed")).toBe(true))

    act(() => {
      handlers.get("poll:completed")?.({ kind: "scheduled" })
    })

    const meta = client.getQueryData<{ at: Date | null; err: string | null }>([
      "poll",
      "meta",
    ])
    expect(meta?.at).toBeInstanceOf(Date)
    expect(meta?.err).toBeNull()
  })

  it("'profiles:active-changed' grava perfil em ['profiles','active']", async () => {
    const { wrapper, client } = createQueryWrapper()
    renderHook(() => useGlobalSubscriptions(), { wrapper })

    await waitFor(() =>
      expect(handlers.has("profiles:active-changed")).toBe(true)
    )

    const profile = {
      id: "x",
      name: "trabalho",
      auth_method: "keyring",
      is_active: true,
    } as Profile

    act(() => {
      handlers.get("profiles:active-changed")?.(profile)
    })

    expect(client.getQueryData(["profiles", "active"])).toEqual(profile)
  })

  it("unmount limpa handlers", async () => {
    const { wrapper } = createQueryWrapper()
    const { unmount } = renderHook(() => useGlobalSubscriptions(), { wrapper })

    await waitFor(() => expect(handlers.has("pr:new")).toBe(true))

    unmount()
    expect(handlers.has("pr:new")).toBe(false)
    expect(handlers.has("pr:status-changed")).toBe(false)
    expect(handlers.has("poll:completed")).toBe(false)
    expect(handlers.has("profiles:active-changed")).toBe(false)
  })
})
