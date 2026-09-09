import { useCallback, useEffect, useRef, useState } from 'react'
import * as pdfjsLib from 'pdfjs-dist/build/pdf.mjs'
import pdfjsWorker from 'pdfjs-dist/build/pdf.worker.min.mjs?url'

pdfjsLib.GlobalWorkerOptions.workerSrc = pdfjsWorker

const MIN_SCALE = 0.75
const MAX_SCALE = 2.5
const SCALE_STEP = 0.25

function PdfPage({ pdf, pageNumber, scale }) {
  const canvasRef = useRef(null)

  useEffect(() => {
    let cancelled = false
    let renderTask

    const renderPage = async () => {
      const page = await pdf.getPage(pageNumber)
      if (cancelled) return

      const viewport = page.getViewport({ scale })
      const canvas = canvasRef.current
      if (!canvas) return

      const outputScale = window.devicePixelRatio || 1
      canvas.width = Math.ceil(viewport.width * outputScale)
      canvas.height = Math.ceil(viewport.height * outputScale)
      canvas.style.width = `${viewport.width}px`
      canvas.style.height = `${viewport.height}px`

      const context = canvas.getContext('2d')
      renderTask = page.render({
        canvasContext: context,
        viewport,
        transform: outputScale !== 1 ? [outputScale, 0, 0, outputScale, 0, 0] : undefined,
      })

      try {
        await renderTask.promise
      } catch (error) {
        if (!cancelled && error?.name !== 'RenderingCancelledException') throw error
      }
    }

    void renderPage()
    return () => {
      cancelled = true
      renderTask?.cancel()
    }
  }, [pdf, pageNumber, scale])

  return (
    <div className="pdf-viewer-page" data-page-number={pageNumber}>
      <canvas ref={canvasRef} aria-label={`PDF page ${pageNumber}`} />
    </div>
  )
}

export default function PdfViewer({ src, title, compact = false, showToolbar = false }) {
  const [pdf, setPDF] = useState(null)
  const [pageCount, setPageCount] = useState(0)
  const [scale, setScale] = useState(compact ? 0.72 : 1)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    let loadingTask
    let loadedPDF

    setLoading(true)
    setError('')
    setPDF(null)
    setPageCount(0)

    const loadPDF = async () => {
      try {
        loadingTask = pdfjsLib.getDocument({ url: src })
        loadedPDF = await loadingTask.promise
        if (cancelled) {
          await loadedPDF.destroy()
          return
        }
        setPDF(loadedPDF)
        setPageCount(loadedPDF.numPages)
        setLoading(false)
      } catch (loadError) {
        if (cancelled) return
        setLoading(false)
        setError(loadError.message || 'The PDF could not be rendered.')
      }
    }

    void loadPDF()
    return () => {
      cancelled = true
      void loadingTask?.destroy()
      void loadedPDF?.destroy()
    }
  }, [src])

  const changeScale = useCallback((amount) => {
    setScale((current) => Math.min(MAX_SCALE, Math.max(MIN_SCALE, Number((current + amount).toFixed(2)))))
  }, [])

  useEffect(() => {
    if (!showToolbar) return undefined

    const handleKeyDown = (event) => {
      if (!(event.ctrlKey || event.metaKey)) return
      if (event.key === '+' || event.key === '=') {
        event.preventDefault()
        changeScale(SCALE_STEP)
      } else if (event.key === '-' || event.key === '_') {
        event.preventDefault()
        changeScale(-SCALE_STEP)
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [changeScale, showToolbar])

  return (
    <div className={`pdf-viewer${compact ? ' pdf-viewer-compact' : ''}`} data-testid="pdf-viewer" data-src={src}>
      {showToolbar && (
        <div className="pdf-viewer-toolbar" role="toolbar" aria-label="PDF zoom controls">
          <span className="pdf-viewer-label">Document view</span>
          <button type="button" className="pdf-viewer-control" data-testid="pdf-zoom-out" onClick={() => changeScale(-SCALE_STEP)} disabled={scale <= MIN_SCALE} aria-label="Zoom out of PDF">−</button>
          <output aria-live="polite">{Math.round(scale * 100)}%</output>
          <button type="button" className="pdf-viewer-control" data-testid="pdf-zoom-in" onClick={() => changeScale(SCALE_STEP)} disabled={scale >= MAX_SCALE} aria-label="Zoom in on PDF">+</button>
          <button type="button" className="pdf-viewer-reset" data-testid="pdf-zoom-reset" onClick={() => setScale(1)}>Reset</button>
          {pageCount > 0 && <span className="pdf-viewer-pages">{pageCount} {pageCount === 1 ? 'page' : 'pages'}</span>}
        </div>
      )}
      <div className="pdf-viewer-viewport" data-testid={showToolbar ? 'preview-viewport' : 'pdf-viewport'} aria-label={`${title} pages`}>
        {loading && <p className="pdf-viewer-state">Rendering PDF pages...</p>}
        {error && <p className="pdf-viewer-state pdf-viewer-error" role="alert">{error}</p>}
        {pdf && (
          <div className="pdf-viewer-pages">
            {Array.from({ length: pageCount }, (_, index) => (
              <PdfPage key={index + 1} pdf={pdf} pageNumber={index + 1} scale={scale} />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
