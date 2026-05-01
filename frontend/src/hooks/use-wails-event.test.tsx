import { renderHook } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { useWailsEvent } from "./use-wails-event"

type EventHandler = (raw: unknown) => void
const handlers = new Map<string, EventHandler>()
const subscribeCount = new Map<string, number>()

vi.mock("@/wailsjs/runtime/runtime", () => ({
  EventsOn: (name: string, fn: EventHandler) => {
    handlers.set(name, fn)
    subscribeCount.set(name, (subscribeCount.get(name) ?? 0) + 1)
  },
  EventsOff: (name: string) => {
    handlers.delete(name)
  },
}))

beforeEach(() => {
  handlers.clear()
  subscribeCount.clear()
})

afterEach(() => {
  handlers.clear()
  subscribeCount.clear()
})

describe("useWailsEvent", () => {
  it("registra handler ao montar", () => {
    const handler = vi.fn()
    renderHook(() => useWailsEvent("pr:new", handler))
    expect(handlers.has("pr:new")).toBe(true)
  })

  it("propaga payload pro handler", () => {
    const handler = vi.fn()
    renderHook(() => useWailsEvent<{ id: string }>("pr:new", handler))
    handlers.get("pr:new")?.({ id: "abc" })
    expect(handler).toHaveBeenCalledExactlyOnceWith({ id: "abc" })
  })

  it("limpa handler no unmount", () => {
    const { unmount } = renderHook(() =>
      useWailsEvent("pr:new", () => undefined)
    )
    expect(handlers.has("pr:new")).toBe(true)
    unmount()
    expect(handlers.has("pr:new")).toBe(false)
  })

  it("não re-subscreve quando handler é referencialmente estável", () => {
    const handler = vi.fn()
    const { rerender } = renderHook(
      ({ fn }: { fn: () => void }) => useWailsEvent("pr:new", fn),
      { initialProps: { fn: handler } }
    )
    expect(subscribeCount.get("pr:new")).toBe(1)
    rerender({ fn: handler })
    rerender({ fn: handler })
    expect(subscribeCount.get("pr:new")).toBe(1)
  })

  it("aciona handler mais recente após rerender", () => {
    const first = vi.fn()
    const second = vi.fn()
    const { rerender } = renderHook(
      ({ fn }: { fn: (p: unknown) => void }) => useWailsEvent("pr:new", fn),
      { initialProps: { fn: first } }
    )
    rerender({ fn: second })
    handlers.get("pr:new")?.("payload")
    expect(first).not.toHaveBeenCalled()
    expect(second).toHaveBeenCalledExactlyOnceWith("payload")
  })
})
