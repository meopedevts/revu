import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react"
import { toast } from "sonner"

import { getTheme, setTheme as setThemeRemote } from "@/bridge"
import { VALID_THEMES } from "@/generated/constants"
import type { Theme } from "@/lib/types"

const STORAGE_KEY = "revu:theme"
const DARK_QUERY = "(prefers-color-scheme: dark)"

interface ThemeContextValue {
  theme: Theme
  setTheme: (theme: Theme) => Promise<void>
}

const ThemeContext = createContext<ThemeContextValue | null>(null)

function isTheme(value: string | null): value is Theme {
  return value !== null && (VALID_THEMES as readonly string[]).includes(value)
}

function readCache(): Theme {
  if (typeof window === "undefined") return "light"
  try {
    const stored = window.localStorage.getItem(STORAGE_KEY)
    return isTheme(stored) ? stored : "light"
  } catch {
    return "light"
  }
}

function resolveTheme(theme: Theme): "light" | "dark" {
  if (theme !== "auto") return theme
  if (typeof window === "undefined" || !window.matchMedia) return "light"
  return window.matchMedia(DARK_QUERY).matches ? "dark" : "light"
}

function applyTheme(theme: Theme): void {
  const resolved = resolveTheme(theme)
  const root = document.documentElement
  if (resolved === "dark") {
    root.classList.add("dark")
  } else {
    root.classList.remove("dark")
  }
  try {
    window.localStorage.setItem(STORAGE_KEY, theme)
  } catch {
    // ignore quota / private-mode errors — theme still applied in-memory
  }
}

interface ThemeProviderProps {
  children: ReactNode
}

export function ThemeProvider({ children }: ThemeProviderProps) {
  const [theme, setThemeState] = useState<Theme>(() => {
    const cached = readCache()
    if (typeof document !== "undefined") {
      applyTheme(cached)
    }
    return cached
  })
  const reconciledRef = useRef(false)

  // Reconcile cache with config.json (source of truth). Runs once on mount.
  useEffect(() => {
    if (reconciledRef.current) return
    reconciledRef.current = true
    void (async () => {
      try {
        const remote = await getTheme()
        if (remote !== theme) {
          applyTheme(remote)
          setThemeState(remote)
        }
      } catch {
        // Bridge unavailable (smoke build) — cache wins.
      }
    })()
  }, [theme])

  // In "auto" mode, react to OS preference changes at runtime without rerender.
  useEffect(() => {
    if (theme !== "auto") return
    if (typeof window === "undefined" || !window.matchMedia) return
    const mq = window.matchMedia(DARK_QUERY)
    const onChange = (): void => {
      applyTheme("auto")
    }
    mq.addEventListener("change", onChange)
    return () => mq.removeEventListener("change", onChange)
  }, [theme])

  const setTheme = useCallback(
    async (next: Theme): Promise<void> => {
      const previous = theme
      // Preview first — the paint reflects the change even if persistence fails.
      applyTheme(next)
      setThemeState(next)
      try {
        await setThemeRemote(next)
      } catch (err: unknown) {
        applyTheme(previous)
        setThemeState(previous)
        toast.error(err instanceof Error ? err.message : "Falha ao salvar tema")
      }
    },
    [theme]
  )

  return (
    <ThemeContext.Provider value={{ theme, setTheme }}>
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext)
  if (!ctx) throw new Error("useTheme must be used within ThemeProvider")
  return ctx
}
