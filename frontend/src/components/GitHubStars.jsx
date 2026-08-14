import { useEffect, useState } from 'react'

const REPO = 'chinmay-sawant/gowkhtmltopdf'
const REPO_URL = `https://github.com/${REPO}`
const CACHE_KEY = 'gowk-gh-stars'
const CACHE_TTL_MS = 2 * 60 * 1000

function formatStars(n) {
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`
  return String(n)
}

function readCache() {
  try {
    const raw = window.localStorage.getItem(CACHE_KEY)
    if (!raw) return null
    const { at, stars } = JSON.parse(raw)
    if (typeof stars !== 'number') return null
    const fresh = Date.now() - at < CACHE_TTL_MS
    return { stars, fresh }
  } catch {
    return null
  }
}

function writeCache(stars) {
  try {
    window.localStorage.setItem(CACHE_KEY, JSON.stringify({ at: Date.now(), stars }))
  } catch {
    // storage unavailable; ignore
  }
}

export default function GitHubStars() {
  const [stars, setStars] = useState(null)

  useEffect(() => {
    let cancelled = false
    const cached = readCache()

    if (cached && cached.fresh) {
      setStars(cached.stars)
      return () => {
        cancelled = true
      }
    }

    if (cached && !cached.fresh) {
      setStars(cached.stars)
    }

    fetch(`https://api.github.com/repos/${REPO}`)
      .then((r) => (r.ok ? r.json() : Promise.reject(new Error('bad response'))))
      .then((data) => {
        if (!cancelled && typeof data.stargazers_count === 'number') {
          setStars(data.stargazers_count)
          writeCache(data.stargazers_count)
        }
      })
      .catch(() => {})

    return () => {
      cancelled = true
    }
  }, [])

  return (
    <a
      className="gh-stars"
      href={REPO_URL}
      target="_blank"
      rel="noopener noreferrer"
      aria-label={stars !== null ? `gowkhtmltopdf on GitHub (${stars} stars)` : 'gowkhtmltopdf on GitHub'}
    >
      <svg className="gh-stars-icon" viewBox="0 0 16 16" aria-hidden="true" focusable="false">
        <path d="M8 1.1a6.9 6.9 0 0 0-2.18 13.45c.35.06.47-.15.47-.34v-1.2c-1.92.42-2.33-.82-2.33-.82-.32-.82-.78-1.04-.78-1.04-.63-.43.05-.42.05-.42.7.05 1.07.72 1.07.72.62 1.06 1.63.75 2.03.57.06-.45.24-.75.44-.92-1.54-.18-3.16-.77-3.16-3.44 0-.76.27-1.38.72-1.87-.07-.18-.31-.88.07-1.84 0 0 .59-.19 1.9.71A6.6 6.6 0 0 1 8 4.45a6.6 6.6 0 0 1 1.73.23c1.31-.9 1.9-.71 1.9-.71.38.96.14 1.66.07 1.84.45.49.72 1.11.72 1.87 0 2.68-1.63 3.26-3.18 3.43.25.22.47.64.47 1.29v1.92c0 .19.13.41.48.34A6.9 6.9 0 0 0 8 1.1Z" />
      </svg>
      <span className="gh-stars-count">{stars === null ? 'Star' : formatStars(stars)}</span>
    </a>
  )
}
