import { useEffect, useState } from 'react'
import { NavLink } from 'react-router-dom'
import GitHubStars from './GitHubStars'

const LINKS = [
  { to: '/', label: 'Overview' },
  { to: '/getting-started', label: 'Getting Started' },
  { to: '/cli', label: 'CLI' },
  { to: '/library-api', label: 'Library API' },
  { to: '/architecture', label: 'Architecture' },
  { to: '/compatibility', label: 'Compatibility' },
  { to: '/fonts', label: 'Fonts' },
  { to: '/security', label: 'Security' },
  { to: '/dossier', label: 'Issue Dossier' },
  { to: '/showcase', label: 'Showcase' },
  { to: '/about', label: 'About' },
]

function useTheme() {
  const [theme, setTheme] = useState(() => {
    if (typeof window === 'undefined') return 'light'
    const saved = window.localStorage.getItem('gowk-theme')
    if (saved === 'dark' || saved === 'light') return saved
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  })

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
    window.localStorage.setItem('gowk-theme', theme)
  }, [theme])

  return [theme, setTheme]
}

export default function SiteNav() {
  const [theme, setTheme] = useTheme()

  return (
    <nav className="site-nav" aria-label="Primary">
      <NavLink to="/" className="brand">
        gowkhtmltopdf
      </NavLink>
      <div className="site-nav-right">
        <div className="site-nav-links">
          {LINKS.map((l) => (
            <NavLink
              key={l.to}
              to={l.to}
              className={({ isActive }) => (isActive ? 'nav-link active' : 'nav-link')}
            >
              {l.label}
            </NavLink>
          ))}
        </div>
          <GitHubStars />
          <button
            type="button"
            className="theme-toggle"
            onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
            aria-label={theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
          >
            {theme === 'dark' ? 'Light' : 'Dark'}
          </button>
      </div>
    </nav>
  )
}
