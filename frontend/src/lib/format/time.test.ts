import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { formatSince, relTime } from "./time"

const NOW = new Date("2026-04-30T12:00:00.000Z")

describe("relTime", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(NOW)
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it("retorna string vazia quando ISO é vazio", () => {
    expect(relTime("")).toBe("")
  })

  it("retorna string vazia quando ISO é inválido", () => {
    expect(relTime("not-a-date")).toBe("")
  })

  it("retorna 'agora' pra menos de 1 minuto", () => {
    const iso = new Date(NOW.getTime() - 30_000).toISOString()
    expect(relTime(iso)).toBe("agora")
  })

  it("retorna minutos pra entre 1min e 60min", () => {
    const iso = new Date(NOW.getTime() - 5 * 60_000).toISOString()
    expect(relTime(iso)).toBe("há 5min")
  })

  it("retorna horas pra entre 1h e 24h", () => {
    const iso = new Date(NOW.getTime() - 3 * 3600_000).toISOString()
    expect(relTime(iso)).toBe("há 3h")
  })

  it("retorna dias pra ≥24h", () => {
    const iso = new Date(NOW.getTime() - 5 * 24 * 3600_000).toISOString()
    expect(relTime(iso)).toBe("há 5d")
  })

  it("boundary: exatamente 60min vira 1h", () => {
    const iso = new Date(NOW.getTime() - 60 * 60_000).toISOString()
    expect(relTime(iso)).toBe("há 1h")
  })
})

describe("formatSince", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(NOW)
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it("retorna placeholder quando data é null", () => {
    expect(formatSince(null)).toBe("ainda não atualizado")
  })

  it("retorna 'atualizado agora' pra menos de 10s", () => {
    const d = new Date(NOW.getTime() - 5_000)
    expect(formatSince(d)).toBe("atualizado agora")
  })

  it("retorna segundos pra entre 10s e 60s", () => {
    const d = new Date(NOW.getTime() - 30_000)
    expect(formatSince(d)).toBe("atualizado há 30s")
  })

  it("retorna minutos pra entre 1min e 60min", () => {
    const d = new Date(NOW.getTime() - 7 * 60_000)
    expect(formatSince(d)).toBe("atualizado há 7min")
  })

  it("retorna horas pra entre 1h e 24h", () => {
    const d = new Date(NOW.getTime() - 4 * 3600_000)
    expect(formatSince(d)).toBe("atualizado há 4h")
  })

  it("retorna dias pra ≥24h", () => {
    const d = new Date(NOW.getTime() - 2 * 24 * 3600_000)
    expect(formatSince(d)).toBe("atualizado há 2d")
  })
})
