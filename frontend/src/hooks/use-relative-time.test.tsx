import { act, renderHook } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { useRelativeTime } from "./use-relative-time"

const NOW = new Date("2026-04-30T12:00:00.000Z")

beforeEach(() => {
  vi.useFakeTimers()
  vi.setSystemTime(NOW)
})

afterEach(() => {
  vi.useRealTimers()
})

describe("useRelativeTime", () => {
  it("retorna idleLabel quando iso é null", () => {
    const { result } = renderHook(() =>
      useRelativeTime(null, { idleLabel: "ainda não atualizado" })
    )
    expect(result.current).toBe("ainda não atualizado")
  })

  it("retorna idleLabel default vazio quando iso é null sem opts", () => {
    const { result } = renderHook(() => useRelativeTime(null))
    expect(result.current).toBe("")
  })

  it("retorna 'agora' pra menos de 10s", () => {
    const iso = new Date(NOW.getTime() - 5_000).toISOString()
    const { result } = renderHook(() => useRelativeTime(iso))
    expect(result.current).toBe("agora")
  })

  it("aplica nowLabel customizado", () => {
    const iso = new Date(NOW.getTime() - 5_000).toISOString()
    const { result } = renderHook(() =>
      useRelativeTime(iso, { nowLabel: "atualizado agora" })
    )
    expect(result.current).toBe("atualizado agora")
  })

  it("retorna segundos pra entre 10s e 60s", () => {
    const iso = new Date(NOW.getTime() - 30_000).toISOString()
    const { result } = renderHook(() => useRelativeTime(iso))
    expect(result.current).toBe("há 30s")
  })

  it("retorna minutos pra entre 1min e 60min", () => {
    const iso = new Date(NOW.getTime() - 5 * 60_000).toISOString()
    const { result } = renderHook(() => useRelativeTime(iso))
    expect(result.current).toBe("há 5min")
  })

  it("retorna horas pra entre 1h e 24h", () => {
    const iso = new Date(NOW.getTime() - 3 * 3600_000).toISOString()
    const { result } = renderHook(() => useRelativeTime(iso))
    expect(result.current).toBe("há 3h")
  })

  it("retorna dias pra ≥24h", () => {
    const iso = new Date(NOW.getTime() - 5 * 24 * 3600_000).toISOString()
    const { result } = renderHook(() => useRelativeTime(iso))
    expect(result.current).toBe("há 5d")
  })

  it("aplica prefix customizado", () => {
    const iso = new Date(NOW.getTime() - 5 * 60_000).toISOString()
    const { result } = renderHook(() =>
      useRelativeTime(iso, { prefix: "visto " })
    )
    expect(result.current).toBe("visto há 5min")
  })

  it("aceita Date além de string ISO", () => {
    const d = new Date(NOW.getTime() - 5 * 60_000)
    const { result } = renderHook(() => useRelativeTime(d))
    expect(result.current).toBe("há 5min")
  })

  it("retorna idleLabel quando iso inválido", () => {
    const { result } = renderHook(() =>
      useRelativeTime("not-a-date", { idleLabel: "—" })
    )
    expect(result.current).toBe("—")
  })

  it("re-renderiza após 60s avançarem", () => {
    const iso = new Date(NOW.getTime() - 30_000).toISOString()
    const { result } = renderHook(() => useRelativeTime(iso))
    expect(result.current).toBe("há 30s")
    act(() => {
      vi.advanceTimersByTime(60_000)
    })
    expect(result.current).toBe("há 1min")
  })

  it("limpa interval no unmount", () => {
    const clearSpy = vi.spyOn(globalThis, "clearInterval")
    const { unmount } = renderHook(() => useRelativeTime(null))
    unmount()
    expect(clearSpy).toHaveBeenCalled()
    clearSpy.mockRestore()
  })

  it("retorna segundos corretos quando Date.now está mid-minute", () => {
    // Regression: anchored-minute snapshot quebrava precisão sub-minuto.
    // Date.now às 12:00:30, iso 30s antes (12:00:00) deve render "há 30s",
    // não "agora".
    vi.setSystemTime(new Date(NOW.getTime() + 30_000))
    const iso = NOW.toISOString()
    const { result } = renderHook(() => useRelativeTime(iso))
    expect(result.current).toBe("há 30s")
  })

  it("compartilha 1 setInterval entre múltiplas instâncias do hook", () => {
    const setSpy = vi.spyOn(globalThis, "setInterval")
    const before = setSpy.mock.calls.length
    const a = renderHook(() => useRelativeTime(null))
    const b = renderHook(() => useRelativeTime(null))
    const c = renderHook(() => useRelativeTime(null))
    expect(setSpy.mock.calls.length - before).toBe(1)
    a.unmount()
    b.unmount()
    c.unmount()
    setSpy.mockRestore()
  })
})
