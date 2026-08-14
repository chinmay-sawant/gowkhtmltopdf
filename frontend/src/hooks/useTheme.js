import { useState, useEffect } from 'react'

/**
 * Custom hook to manage light/dark theme with localStorage persistence,
 * system color scheme detection, document attribute, and theme-color meta tag sync.
 *
 * @returns {[string, (theme: string | ((prev: string) => string)) => void]}
 */
export function useTheme() {
  const [theme, setTheme] = useState(() => {
    if (typeof window === 'undefined') return 'light'
    const saved = window.localStorage.getItem('gowk-theme')
    if (saved === 'dark' || saved === 'light') return saved
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  })

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
    document.documentElement.style.colorScheme = theme
    const metaThemeColor = document.querySelector('meta[name="theme-color"]')
    if (metaThemeColor) {
      metaThemeColor.setAttribute('content', theme === 'dark' ? '#0d1413' : '#f8f9f8')
    }
    window.localStorage.setItem('gowk-theme', theme)
  }, [theme])

  return [theme, setTheme]
}

export default useTheme
