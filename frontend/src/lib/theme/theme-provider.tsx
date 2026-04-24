import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import { toast } from 'sonner'

import { getTheme, setTheme as setThemeRemote } from '@/src/lib/bridge'
import type { Theme } from '@/src/lib/types'

const STORAGE_KEY = 'revu:theme'

interface ThemeContextValue {
  theme: Theme
  setTheme: (theme: Theme) => Promise<void>
}

const ThemeContext = createContext<ThemeContextValue | null>(null)

function readCache(): Theme {
  if (typeof window === 'undefined') return 'light'
  try {
    return window.localStorage.getItem(STORAGE_KEY) === 'dark' ? 'dark' : 'light'
  } catch {
    return 'light'
  }
}

function applyTheme(theme: Theme): void {
  const root = document.documentElement
  if (theme === 'dark') {
    root.classList.add('dark')
  } else {
    root.classList.remove('dark')
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
  const [theme, setThemeState] = useState<Theme>(() => readCache())
  const reconciledRef = useRef(false)

  // Reconcile cache with config.json (source of truth). Runs once on mount.
  // If the backend says "dark" but cache said "light" (or user edited
  // config.json by hand), this corrects the UI without a reload.
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
        toast.error(
          err instanceof Error ? err.message : 'Falha ao salvar tema',
        )
      }
    },
    [theme],
  )

  return (
    <ThemeContext.Provider value={{ theme, setTheme }}>
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext)
  if (!ctx) throw new Error('useTheme must be used within ThemeProvider')
  return ctx
}
