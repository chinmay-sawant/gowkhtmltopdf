import { useCallback, useEffect, useRef, useState } from 'react'
import { githubPdfUrl, githubTemplateUrl } from '../data/showcase'

const IMAGES = import.meta.glob('../assets/showcase/*.png', { eager: true, query: '?url', import: 'default' })

function pageUrl(item, page) {
  if (page === 1) return IMAGES[`../assets/showcase/${item.name}.png`]
  return IMAGES[`../assets/showcase/${item.name}-${page}.png`]
}

const MIN_ZOOM = 1
const MAX_ZOOM = 4

export default function Cardbox({ item, onClose }) {
  const total = item.pages ?? 1
  const compactPager = total > 10
  const [page, setPage] = useState(1)
  const [zoom, setZoom] = useState(1)
  const [pan, setPan] = useState({ x: 0, y: 0 })
  const [isDragging, setIsDragging] = useState(false)
  const [isFullscreen, setIsFullscreen] = useState(false)

  const closeRef = useRef(null)
  const innerRef = useRef(null)
  const openerRef = useRef(document.activeElement)
  const dragStartRef = useRef({ startX: 0, startY: 0, panX: 0, panY: 0 })

  const resetZoom = useCallback(() => {
    setZoom(1)
    setPan({ x: 0, y: 0 })
  }, [])

  const zoomIn = useCallback(() => {
    setZoom((z) => Math.min(MAX_ZOOM, Math.round((z + 0.5) * 10) / 10))
  }, [])

  const zoomOut = useCallback(() => {
    setZoom((z) => {
      const next = Math.max(MIN_ZOOM, Math.round((z - 0.5) * 10) / 10)
      if (next <= 1) setPan({ x: 0, y: 0 })
      return next
    })
  }, [])

  const toggleFullscreen = useCallback(() => {
    if (!document.fullscreenElement) {
      if (innerRef.current?.requestFullscreen) {
        innerRef.current.requestFullscreen().catch(() => {})
      }
    } else {
      if (document.exitFullscreen) {
        document.exitFullscreen().catch(() => {})
      }
    }
  }, [])

  const handlePageChange = useCallback((newPage) => {
    setPage(newPage)
    resetZoom()
  }, [resetZoom])

  const prev = useCallback(() => {
    setPage((p) => (p > 1 ? p - 1 : total))
    resetZoom()
  }, [total, resetZoom])

  const next = useCallback(() => {
    setPage((p) => (p < total ? p + 1 : 1))
    resetZoom()
  }, [total, resetZoom])

  // Mouse pan handling
  const handleMouseDown = (e) => {
    if (zoom <= 1) return
    if (e.button !== 0) return
    e.preventDefault()
    setIsDragging(true)
    dragStartRef.current = {
      startX: e.clientX,
      startY: e.clientY,
      panX: pan.x,
      panY: pan.y,
    }
  }

  // Touch pan handling
  const handleTouchStart = (e) => {
    if (zoom <= 1 || e.touches.length !== 1) return
    const touch = e.touches[0]
    setIsDragging(true)
    dragStartRef.current = {
      startX: touch.clientX,
      startY: touch.clientY,
      panX: pan.x,
      panY: pan.y,
    }
  }

  const handleTouchMove = (e) => {
    if (!isDragging || e.touches.length !== 1) return
    const touch = e.touches[0]
    const dx = touch.clientX - dragStartRef.current.startX
    const dy = touch.clientY - dragStartRef.current.startY
    setPan({
      x: dragStartRef.current.panX + dx,
      y: dragStartRef.current.panY + dy,
    })
  }

  const handleTouchEnd = () => {
    setIsDragging(false)
  }

  // Double click toggles between 1x and 2x
  const handleDoubleClick = () => {
    if (zoom > 1) {
      resetZoom()
    } else {
      setZoom(2)
      setPan({ x: 0, y: 0 })
    }
  }

  useEffect(() => {
    if (!isDragging) return
    const handleMouseMove = (e) => {
      const dx = e.clientX - dragStartRef.current.startX
      const dy = e.clientY - dragStartRef.current.startY
      setPan({
        x: dragStartRef.current.panX + dx,
        y: dragStartRef.current.panY + dy,
      })
    }
    const handleMouseUp = () => {
      setIsDragging(false)
    }
    window.addEventListener('mousemove', handleMouseMove)
    window.addEventListener('mouseup', handleMouseUp)
    return () => {
      window.removeEventListener('mousemove', handleMouseMove)
      window.removeEventListener('mouseup', handleMouseUp)
    }
  }, [isDragging])

  useEffect(() => {
    const onFsChange = () => {
      setIsFullscreen(Boolean(document.fullscreenElement))
    }
    document.addEventListener('fullscreenchange', onFsChange)
    return () => document.removeEventListener('fullscreenchange', onFsChange)
  }, [])

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
        if (document.fullscreenElement) {
          document.exitFullscreen().catch(() => {})
        } else {
          onClose()
        }
      } else if (e.key === 'ArrowLeft') {
        prev()
      } else if (e.key === 'ArrowRight') {
        next()
      } else if (e.key === '+' || e.key === '=') {
        e.preventDefault()
        zoomIn()
      } else if (e.key === '-' || e.key === '_') {
        e.preventDefault()
        zoomOut()
      } else if (e.key === '0') {
        e.preventDefault()
        resetZoom()
      } else if (e.key === 'f' || e.key === 'F') {
        e.preventDefault()
        toggleFullscreen()
      } else if (e.key === 'Tab') {
        if (!innerRef.current) return
        const focusable = Array.from(
          innerRef.current.querySelectorAll(
            'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
          ),
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
  }, [onClose, prev, next, zoomIn, zoomOut, resetZoom, toggleFullscreen])

  const pageButtons = Array.from({ length: total }, (_, i) => i + 1).map((n) => (
    <button
      type="button"
      key={n}
      className={n === page ? 'pager-dot active' : 'pager-dot'}
      onClick={() => handlePageChange(n)}
      aria-label={`Page ${n}`}
      aria-current={n === page ? 'page' : undefined}
    >
      {n}
    </button>
  ))

  return (
    <div className={`cardbox${isFullscreen ? ' is-fullscreen' : ''}`} role="dialog" aria-modal="true" aria-label={item.title} onClick={onClose}>
      <div className="cardbox-inner" ref={innerRef} onClick={(e) => e.stopPropagation()}>
        <div className="cardbox-head">
          <div className="cardbox-title-wrap">
            <h2>{item.title}</h2>
            <span className="cardbox-file">{item.file}</span>
          </div>

          <div className="cardbox-header-controls">
            {/* Zoom Controls */}
            <div className="cardbox-zoom-group" role="toolbar" aria-label="Image zoom controls">
              <button
                type="button"
                className="cardbox-ctrl-btn"
                onClick={zoomOut}
                disabled={zoom <= MIN_ZOOM}
                title="Zoom Out (-)"
                aria-label="Zoom Out"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                  <line x1="5" y1="12" x2="19" y2="12" />
                </svg>
              </button>
              <button
                type="button"
                className={`cardbox-ctrl-btn zoom-indicator${zoom > 1 ? ' active' : ''}`}
                onClick={resetZoom}
                title="Reset Zoom to 100% (0)"
                aria-label={`Zoom level ${Math.round(zoom * 100)}%. Click to reset`}
              >
                {Math.round(zoom * 100)}%
              </button>
              <button
                type="button"
                className="cardbox-ctrl-btn"
                onClick={zoomIn}
                disabled={zoom >= MAX_ZOOM}
                title="Zoom In (+)"
                aria-label="Zoom In"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                  <line x1="12" y1="5" x2="12" y2="19" />
                  <line x1="5" y1="12" x2="19" y2="12" />
                </svg>
              </button>
            </div>

            {/* Fullscreen Toggle */}
            <button
              type="button"
              className={`cardbox-ctrl-btn cardbox-fs-btn${isFullscreen ? ' active' : ''}`}
              onClick={toggleFullscreen}
              title={isFullscreen ? 'Exit Fullscreen (F)' : 'Fullscreen (F)'}
              aria-label={isFullscreen ? 'Exit Fullscreen' : 'Fullscreen'}
            >
              {isFullscreen ? (
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M8 3v3a2 2 0 0 1-2 2H3m18 0h-3a2 2 0 0 1-2-2V3m0 18v-3a2 2 0 0 1 2-2h3M3 16h3a2 2 0 0 1 2 2v3" />
                </svg>
              ) : (
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7" />
                </svg>
              )}
            </button>

            <button ref={closeRef} type="button" className="cardbox-close" onClick={onClose} aria-label="Close" title="Close (Esc)">
              ×
            </button>
          </div>
        </div>

        <div className="cardbox-stage">
          {total > 1 && (
            <button type="button" className="cardbox-arrow left" onClick={prev} aria-label="Previous page" title="Previous page (←)">
              ←
            </button>
          )}
          <div
            className={`cardbox-page${zoom > 1 ? ' is-zoomed' : ''}${isDragging ? ' is-dragging' : ''}`}
            onMouseDown={handleMouseDown}
            onTouchStart={handleTouchStart}
            onTouchMove={handleTouchMove}
            onTouchEnd={handleTouchEnd}
            onDoubleClick={handleDoubleClick}
            style={{
              cursor: zoom > 1 ? (isDragging ? 'grabbing' : 'grab') : 'default',
            }}
          >
            {pageUrl(item, page) ? (
              <img
                src={pageUrl(item, page)}
                alt={`${item.title} page ${page}`}
                style={{
                  transform: `translate3d(${pan.x}px, ${pan.y}px, 0) scale(${zoom})`,
                  transformOrigin: 'center center',
                  transition: isDragging ? 'none' : 'transform 0.15s cubic-bezier(0.2, 0, 0, 1)',
                  userSelect: 'none',
                  pointerEvents: zoom > 1 ? 'none' : 'auto',
                }}
                draggable={false}
              />
            ) : null}
          </div>
          {total > 1 && (
            <button type="button" className="cardbox-arrow right" onClick={next} aria-label="Next page" title="Next page (→)">
              →
            </button>
          )}
        </div>

        {/* Visual Keyboard Navigation Hints */}
        <div className="cardbox-hints" aria-hidden="true">
          <span className="hint-item"><kbd>←</kbd> <kbd>→</kbd> Page</span>
          <span className="hint-sep">•</span>
          <span className="hint-item"><kbd>+</kbd> <kbd>−</kbd> Zoom</span>
          <span className="hint-sep">•</span>
          <span className="hint-item"><kbd>0</kbd> Reset</span>
          {zoom > 1 && (
            <>
              <span className="hint-sep">•</span>
              <span className="hint-item hint-pan">Drag to Pan</span>
            </>
          )}
          <span className="hint-sep">•</span>
          <span className="hint-item"><kbd>F</kbd> Fullscreen</span>
          <span className="hint-sep">•</span>
          <span className="hint-item"><kbd>Esc</kbd> Close</span>
        </div>

        <div className="cardbox-foot">
          <div className="cardbox-pager">
            {compactPager ? (
              <label className="cardbox-page-picker">
                <span>Page</span>
                <select value={page} onChange={(e) => handlePageChange(Number(e.target.value))} aria-label={`Go to page of ${total}`}>
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


