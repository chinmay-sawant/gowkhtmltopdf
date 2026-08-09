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
    <a className="gh-stars" href={REPO_URL} target="_blank" rel="noopener noreferrer">
      <span className="gh-stars-emoji" aria-hidden="true">
        ⭐
      </span>
      <span className="gh-stars-count">{stars === null ? 'Star' : formatStars(stars)}</span>
    </a>
  )
}
