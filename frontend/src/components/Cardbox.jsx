import { useCallback, useEffect, useRef, useState } from 'react'
import { githubPdfUrl, githubTemplateUrl } from '../data/showcase'

const IMAGES = import.meta.glob('../assets/showcase/*.png', { eager: true, query: '?url', import: 'default' })

function pageUrl(item, page) {
  if (page === 1) return IMAGES[`../assets/showcase/${item.name}.png`]
  return IMAGES[`../assets/showcase/${item.name}-${page}.png`]
}

export default function Cardbox({ item, onClose }) {
  const total = item.pages ?? 1
  const compactPager = total > 10
  const [page, setPage] = useState(1)
  const closeRef = useRef(null)
  const innerRef = useRef(null)
  const openerRef = useRef(document.activeElement)

  const prev = useCallback(() => setPage((p) => (p > 1 ? p - 1 : total)), [total])
  const next = useCallback(() => setPage((p) => (p < total ? p + 1 : 1)), [total])

  const pageButtons = Array.from({ length: total }, (_, i) => i + 1).map((n) => (
    <button
      type="button"
      key={n}
      className={n === page ? 'pager-dot active' : 'pager-dot'}
      onClick={() => setPage(n)}
      aria-label={`Page ${n}`}
      aria-current={n === page ? 'page' : undefined}
    >
      {n}
    </button>
  ))

  useEffect(() => {
    const prevOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    closeRef.current?.focus()

    return () => {
      document.body.style.overflow = prevOverflow
      if (openerRef.current && typeof openerRef.current.focus === 'function') {
        openerRef.current.focus()
      }
    }
  }, [])

  useEffect(() => {
    function onKey(e) {
      if (e.key === 'Escape') {
        e.preventDefault()
        onClose()
      } else if (e.key === 'ArrowLeft') {
        prev()
      } else if (e.key === 'ArrowRight') {
        next()
      } else if (e.key === 'Tab') {
        if (!innerRef.current) return
        const focusable = Array.from(
          innerRef.current.querySelectorAll(
            'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
          )
        ).filter((el) => el.offsetParent !== null || el.offsetWidth > 0 || el.offsetHeight > 0)

        if (focusable.length === 0) {
          e.preventDefault()
          return
        }

        const first = focusable[0]
        const last = focusable[focusable.length - 1]

        if (e.shiftKey) {
          if (document.activeElement === first || !innerRef.current.contains(document.activeElement)) {
            e.preventDefault()
            last.focus()
          }
        } else {
          if (document.activeElement === last || !innerRef.current.contains(document.activeElement)) {
            e.preventDefault()
            first.focus()
          }
        }
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose, prev, next])

  return (
    <div className="cardbox" role="dialog" aria-modal="true" aria-label={item.title} onClick={onClose}>
      <div className="cardbox-inner" ref={innerRef} onClick={(e) => e.stopPropagation()}>
        <div className="cardbox-head">
          <div>
            <h2>{item.title}</h2>
            <span className="cardbox-file">{item.file}</span>
          </div>
          <button ref={closeRef} type="button" className="cardbox-close" onClick={onClose} aria-label="Close">
            ×
          </button>
        </div>

        <div className="cardbox-stage">
          {total > 1 && (
            <button type="button" className="cardbox-arrow left" onClick={prev} aria-label="Previous page">
              ←
            </button>
          )}
          <div className="cardbox-page">
            {pageUrl(item, page) ? (
              <img src={pageUrl(item, page)} alt={`${item.title} page ${page}`} />
            ) : null}
          </div>
          {total > 1 && (
            <button type="button" className="cardbox-arrow right" onClick={next} aria-label="Next page">
              →
            </button>
          )}
        </div>

        <div className="cardbox-foot">
          <div className="cardbox-pager">
            {compactPager ? (
              <label className="cardbox-page-picker">
                <span>Page</span>
                <select value={page} onChange={(e) => setPage(Number(e.target.value))} aria-label={`Go to page of ${total}`}>
                  {Array.from({ length: total }, (_, i) => i + 1).map((n) => (
                    <option key={n} value={n}>
                      {n} of {total}
                    </option>
                  ))}
                </select>
              </label>
            ) : (
              total > 1 ? pageButtons : null
            )}
          </div>
          <div className="cardbox-actions">
            <span className="cardbox-pagestate">
              {total > 1 ? `page ${page} of ${total}` : 'single page'}
            </span>
            {githubTemplateUrl(item.name) && (
              <a className="cardbox-btn" href={githubTemplateUrl(item.name)} target="_blank" rel="noopener noreferrer">
                View template ↗
              </a>
            )}
            <a className="cardbox-btn primary" href={githubPdfUrl(item.file)} target="_blank" rel="noopener noreferrer">
              Open PDF ↗
            </a>
          </div>
        </div>
      </div>
    </div>
  )
}

