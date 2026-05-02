import { act, render, screen } from "@testing-library/react"
import {
  afterEach,
  beforeEach,
  describe,
  expect,
  it,
  vi,
  type MockInstance,
} from "vitest"

import * as bridge from "@/bridge"

import { ThemeProvider, useTheme } from "./theme-provider"

vi.mock("@/bridge", () => ({
  getTheme: vi.fn(() => Promise.resolve("light")),
  setTheme: vi.fn(() => Promise.resolve()),
}))

interface MediaQueryListMock {
  matches: boolean
  media: string
  addEventListener: MockInstance
  removeEventListener: MockInstance
  dispatchChange: (matches: boolean) => void
}

function installMatchMedia(initialMatches: boolean): MediaQueryListMock {
  const listeners = new Set<(ev: { matches: boolean }) => void>()
  const mql: MediaQueryListMock = {
    matches: initialMatches,
    media: "(prefers-color-scheme: dark)",
    addEventListener: vi.fn(
      (_evt: string, cb: (ev: { matches: boolean }) => void) => {
        listeners.add(cb)
      }
    ),
    removeEventListener: vi.fn(
      (_evt: string, cb: (ev: { matches: boolean }) => void) => {
        listeners.delete(cb)
      }
    ),
    dispatchChange(matches: boolean) {
      this.matches = matches
      listeners.forEach((cb) => cb({ matches }))
    },
  }

  Object.defineProperty(window, "matchMedia", {
    writable: true,
    configurable: true,
    value: vi.fn(() => mql),
  })

  return mql
}

function ThemeProbe() {
  const { theme, setTheme } = useTheme()
  return (
    <div>
      <span data-testid="theme">{theme}</span>
      <button type="button" onClick={() => void setTheme("auto")}>
        auto
      </button>
      <button type="button" onClick={() => void setTheme("dark")}>
        dark
      </button>
      <button type="button" onClick={() => void setTheme("light")}>
        light
      </button>
    </div>
  )
}

async function mountProvider(): Promise<void> {
  // act(async) flush dos useEffect que await getTheme(); Promise.resolve()
  // garante que o microtask queue rode antes da expectativa.
  await act(async () => {
    render(
      <ThemeProvider>
        <ThemeProbe />
      </ThemeProvider>
    )
    await Promise.resolve()
  })
}

describe("ThemeProvider", () => {
  beforeEach(() => {
    document.documentElement.classList.remove("dark")
    window.localStorage.clear()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it("auto + system=dark aplica classe .dark", async () => {
    installMatchMedia(true)
    window.localStorage.setItem("revu:theme", "auto")
    vi.mocked(bridge.getTheme).mockResolvedValueOnce("auto")

    await mountProvider()

    expect(document.documentElement.classList.contains("dark")).toBe(true)
    expect(screen.getByTestId("theme").textContent).toBe("auto")
  })

  it("auto + system=light remove classe .dark", async () => {
    installMatchMedia(false)
    document.documentElement.classList.add("dark")
    window.localStorage.setItem("revu:theme", "auto")
    vi.mocked(bridge.getTheme).mockResolvedValueOnce("auto")

    await mountProvider()

    expect(document.documentElement.classList.contains("dark")).toBe(false)
  })

  it("auto: muda OS preference flipa classe .dark sem reload", async () => {
    const mql = installMatchMedia(false)
    window.localStorage.setItem("revu:theme", "auto")
    vi.mocked(bridge.getTheme).mockResolvedValueOnce("auto")

    await mountProvider()

    expect(document.documentElement.classList.contains("dark")).toBe(false)

    act(() => {
      mql.dispatchChange(true)
    })
    expect(document.documentElement.classList.contains("dark")).toBe(true)

    act(() => {
      mql.dispatchChange(false)
    })
    expect(document.documentElement.classList.contains("dark")).toBe(false)
  })

  it("auto persiste literal no localStorage", async () => {
    installMatchMedia(false)
    window.localStorage.setItem("revu:theme", "light")

    await mountProvider()

    await act(async () => {
      screen.getByText("auto").click()
      await Promise.resolve()
    })

    expect(window.localStorage.getItem("revu:theme")).toBe("auto")
    expect(screen.getByTestId("theme").textContent).toBe("auto")
  })

  it("trocar de auto pra dark remove o listener de matchMedia", async () => {
    const mql = installMatchMedia(false)
    window.localStorage.setItem("revu:theme", "auto")
    vi.mocked(bridge.getTheme).mockResolvedValueOnce("auto")

    await mountProvider()

    expect(mql.addEventListener).toHaveBeenCalledTimes(1)

    await act(async () => {
      screen.getByText("dark").click()
      await Promise.resolve()
    })

    expect(mql.removeEventListener).toHaveBeenCalledTimes(1)
    expect(document.documentElement.classList.contains("dark")).toBe(true)
  })
})
