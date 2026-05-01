import { renderHook } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { useEscapeKey } from "./use-escape-key"

function dispatchKey(key: string) {
  window.dispatchEvent(new KeyboardEvent("keydown", { key }))
}

describe("useEscapeKey", () => {
  it("chama handler ao pressionar Escape", () => {
    const handler = vi.fn()
    renderHook(() => useEscapeKey(handler))
    dispatchKey("Escape")
    expect(handler).toHaveBeenCalledOnce()
  })

  it("ignora teclas diferentes de Escape", () => {
    const handler = vi.fn()
    renderHook(() => useEscapeKey(handler))
    dispatchKey("Enter")
    dispatchKey("a")
    expect(handler).not.toHaveBeenCalled()
  })

  it("não registra listener quando enabled=false", () => {
    const handler = vi.fn()
    renderHook(() => useEscapeKey(handler, false))
    dispatchKey("Escape")
    expect(handler).not.toHaveBeenCalled()
  })

  it("remove listener no unmount", () => {
    const handler = vi.fn()
    const { unmount } = renderHook(() => useEscapeKey(handler))
    unmount()
    dispatchKey("Escape")
    expect(handler).not.toHaveBeenCalled()
  })

  it("usa handler atualizado sem re-subscrever", () => {
    const first = vi.fn()
    const second = vi.fn()
    const { rerender } = renderHook(
      ({ h }: { h: () => void }) => useEscapeKey(h),
      { initialProps: { h: first } }
    )
    dispatchKey("Escape")
    expect(first).toHaveBeenCalledOnce()
    rerender({ h: second })
    dispatchKey("Escape")
    expect(second).toHaveBeenCalledOnce()
    expect(first).toHaveBeenCalledOnce()
  })

  it("alterna registro quando enabled muda", () => {
    const handler = vi.fn()
    const { rerender } = renderHook(
      ({ on }: { on: boolean }) => useEscapeKey(handler, on),
      { initialProps: { on: false } }
    )
    dispatchKey("Escape")
    expect(handler).not.toHaveBeenCalled()
    rerender({ on: true })
    dispatchKey("Escape")
    expect(handler).toHaveBeenCalledOnce()
  })
})
