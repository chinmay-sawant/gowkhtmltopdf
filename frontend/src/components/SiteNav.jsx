import { useEffect, useRef, useState } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import GitHubStars from './GitHubStars'
import useTheme from '../hooks/useTheme'

const LINKS = [
  { to: '/', label: 'Overview' },
  { to: '/getting-started', label: 'Getting Started' },
  { to: '/documentation', label: 'Documentation' },
  { to: '/dossier', label: 'Issue Dossier' },
  { to: '/showcase', label: 'Showcase' },
  { to: '/benchmarks', label: 'Benchmarks' },
]

export default function SiteNav() {
  const [theme, setTheme] = useTheme()
  const [isOpen, setIsOpen] = useState(false)
  const location = useLocation()
  const navRef = useRef(null)

  useEffect(() => {
    setIsOpen(false)
  }, [location.pathname])

  useEffect(() => {
    function onKeyDown(e) {
      if (e.key === 'Escape') {
        setIsOpen(false)
      }
    }
    function onClickOutside(e) {
      if (navRef.current && !navRef.current.contains(e.target)) {
        setIsOpen(false)
      }
    }
    if (isOpen) {
      document.addEventListener('keydown', onKeyDown)
      document.addEventListener('pointerdown', onClickOutside)
      return () => {
        document.removeEventListener('keydown', onKeyDown)
        document.removeEventListener('pointerdown', onClickOutside)
      }
    }
  }, [isOpen])

  return (
    <nav className="site-nav" aria-label="Primary" ref={navRef}>
      <div className="site-nav-bar">
        <NavLink to="/" end className="brand">
          gowkhtmltopdf
        </NavLink>

        <div className="site-nav-desktop">
          <div className="site-nav-links">
            {LINKS.map((l) => (
              <NavLink
                key={l.to}
                end={l.to === '/'}
                to={l.to}
                className={({ isActive }) => {
                  const classes = ['nav-link']
                  if (isActive) classes.push('active')
                  if (l.to === '/getting-started') classes.push('nav-link-cta')
                  return classes.join(' ')
                }}
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

        <div className="site-nav-mobile-bar">
          <NavLink
            to="/getting-started"
            className={({ isActive }) =>
              isActive ? 'nav-link nav-link-cta mobile-bar-cta active' : 'nav-link nav-link-cta mobile-bar-cta'
            }
          >
            Getting Started
          </NavLink>
          <button
            type="button"
            className="mobile-menu-toggle"
            aria-expanded={isOpen}
            aria-controls="mobile-nav-menu"
            aria-label="Toggle navigation menu"
            onClick={() => setIsOpen((prev) => !prev)}
          >
            <svg
              viewBox="0 0 24 24"
              width="20"
              height="20"
              aria-hidden="true"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
            >
              {isOpen ? (
                <path d="M6 18L18 6M6 6l12 12" />
              ) : (
                <path d="M4 6h16M4 12h16M4 18h16" />
              )}
            </svg>
          </button>
        </div>
      </div>

      {isOpen && (
        <div id="mobile-nav-menu" className="mobile-nav-dropdown">
          <div className="mobile-nav-links">
            {LINKS.map((l) => (
              <NavLink
                key={l.to}
                end={l.to === '/'}
                to={l.to}
                className={({ isActive }) => {
                  const classes = ['mobile-nav-link']
                  if (isActive) classes.push('active')
                  if (l.to === '/getting-started') classes.push('mobile-nav-link-cta')
                  return classes.join(' ')
                }}
                onClick={() => setIsOpen(false)}
              >
                {l.label}
              </NavLink>
            ))}
          </div>
          <div className="mobile-nav-footer">
            <GitHubStars />
            <button
              type="button"
              className="theme-toggle mobile-theme-toggle"
              onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
              aria-label={theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
            >
              {theme === 'dark' ? 'Light' : 'Dark'}
            </button>
          </div>
        </div>
      )}
    </nav>
  )
}

