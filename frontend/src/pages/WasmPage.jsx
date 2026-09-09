import { useEffect, useRef, useState } from 'react'
import { createWasmClient } from '../lib/wasmClient'

const OUTPUTS = [
  { value: 'pdf', label: 'PDF document', hint: 'Preview pages in the browser' },
  { value: 'png', label: 'PNG image', hint: 'Lossless raster output' },
  { value: 'jpeg', label: 'JPEG image', hint: 'Compact raster output' },
]

const INITIAL_HTML = '<!DOCTYPE html>\n<html>\n<body>\n  <h1>Hello from WASM</h1>\n</body>\n</html>'
const INITIAL_CSS = `@page { size: A4; margin: 18mm; }
body { color: #20242b; font-family: sans-serif; font-size: 12pt; }
h1 { color: #1f4b99; }`

const composeDocument = (sourceHTML, sourceCSS) => {
  const style = `<style>\n${sourceCSS}\n</style>`
  if (/<\/head>/i.test(sourceHTML)) return sourceHTML.replace(/<\/head>/i, `${style}\n</head>`)
  return `${style}\n${sourceHTML}`
}

export default function WasmPage() {
  const [html, setHTML] = useState(INITIAL_HTML)
  const [css, setCSS] = useState(INITIAL_CSS)
  const [samples, setSamples] = useState([])
  const [selectedSample, setSelectedSample] = useState('')
  const [mode, setMode] = useState('pdf')
  const [status, setStatus] = useState('idle')
  const [progress, setProgress] = useState({ phase: '', percent: 0 })
  const [error, setError] = useState('')
  const [result, setResult] = useState(null)
  const [previewOpen, setPreviewOpen] = useState(false)
  const [previewZoom, setPreviewZoom] = useState(1)
  const clientRef = useRef(null)
  const requestRef = useRef(0)
  const previewCloseRef = useRef(null)
  const previewTriggerRef = useRef(null)

  useEffect(() => () => clientRef.current?.cancel(), [])

  useEffect(() => () => {
    if (result?.url) URL.revokeObjectURL(result.url)
  }, [result])

  const closePreview = () => {
    setPreviewOpen(false)
    setPreviewZoom(1)
    previewTriggerRef.current?.focus()
  }

  useEffect(() => {
    if (!previewOpen) return undefined

    const previousOverflow = document.body.style.overflow
    const handleKeyDown = (event) => {
      if (event.key === 'Escape') closePreview()
    }
    document.body.style.overflow = 'hidden'
    document.addEventListener('keydown', handleKeyDown)
    previewCloseRef.current?.focus()

    return () => {
      document.body.style.overflow = previousOverflow
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [previewOpen])

  useEffect(() => {
    let cancelled = false

    const loadSampleCatalog = async () => {
      try {
        const baseURL = new URL(import.meta.env.BASE_URL, window.location.origin)
        const response = await fetch(new URL('wasm/samples/manifest.json', baseURL))
        if (!response.ok) throw new Error(`Sample catalog failed to load (${response.status})`)
        const catalog = await response.json()
        if (cancelled) return
        setSamples(catalog.samples || [])
        setSelectedSample(catalog.samples?.[0]?.id || '')
      } catch (catalogError) {
        if (!cancelled) setError(catalogError.message)
      }
    }

    loadSampleCatalog()
    return () => { cancelled = true }
  }, [])

  const loadSample = async () => {
    const sample = samples.find((entry) => entry.id === selectedSample)
    if (!sample) return

    try {
      const baseURL = new URL(import.meta.env.BASE_URL, window.location.origin)
      const responses = await Promise.all([
        fetch(new URL(sample.html, baseURL)),
        fetch(new URL(sample.css, baseURL)),
      ])
      for (const response of responses) {
        if (!response.ok) throw new Error(`Sample asset failed to load (${response.status})`)
      }
      const [nextHTML, nextCSS] = await Promise.all(responses.map((response) => response.text()))
      setHTML(nextHTML)
      setCSS(nextCSS)
      setPreviewOpen(false)
      setResult(null)
      setStatus('idle')
      setProgress({ phase: '', percent: 0 })
      setError('')
    } catch (sampleError) {
      setError(sampleError.message)
      setStatus('error')
    }
  }

  const reset = () => {
    requestRef.current += 1
    clientRef.current?.cancel()
    clientRef.current = null
    setHTML(INITIAL_HTML)
    setCSS(INITIAL_CSS)
    setSelectedSample(samples[0]?.id || '')
    setPreviewOpen(false)
    setMode('pdf')
    setStatus('idle')
    setProgress({ phase: '', percent: 0 })
    setError('')
    setResult(null)
  }

  const convert = async (event) => {
    event.preventDefault()
    if (!html.trim()) {
      setStatus('error')
      setError('Enter some HTML before converting.')
      return
    }

    const requestID = requestRef.current + 1
    requestRef.current = requestID
    const client = clientRef.current || createWasmClient()
    clientRef.current = client
    setStatus('loading')
    setError('')
    setPreviewOpen(false)
    setResult(null)
    setProgress({ phase: 'Starting', percent: 0 })

    try {
      const response = await client.convert({ html: composeDocument(html, css), mode }, (phase, percent) => {
        if (requestRef.current === requestID) setProgress({ phase: phase || 'Converting', percent })
      })
      if (requestRef.current !== requestID) return
      const bytes = new Uint8Array(response.bytes)
      const url = URL.createObjectURL(new Blob([bytes], { type: response.mime }))
      setResult({ ...response, url })
      setStatus('success')
    } catch (conversionError) {
      if (requestRef.current !== requestID) return
      if (conversionError.message === 'Conversion cancelled') {
        setStatus('cancelled')
        return
      }
      setStatus('error')
      setError(conversionError.message)
    }
  }

  const cancel = () => {
    requestRef.current += 1
    clientRef.current?.cancel()
    clientRef.current = null
    setStatus('cancelled')
  }

  const openPreview = () => {
    if (!result) return
    setPreviewZoom(1)
    setPreviewOpen(true)
  }

  const changePreviewZoom = (amount) => {
    setPreviewZoom((current) => Math.min(3, Math.max(0.5, Number((current + amount).toFixed(2)))))
  }

  return (
    <div className="wasm-page">
      <section className="wasm-hero" aria-labelledby="wasm-title">
        <p className="wasm-kicker">Browser tool</p>
        <h1 id="wasm-title">Convert HTML in your browser.</h1>
        <p className="lede">The Go engine runs locally through WebAssembly. Choose a document or image output, then preview the result without uploading your HTML.</p>
      </section>

      <form className="wasm-workspace" onSubmit={convert}>
        <section className="wasm-panel wasm-editor-panel" aria-labelledby="editor-title">
          <div className="wasm-panel-heading">
            <div>
              <p className="wasm-kicker">01 / Input</p>
              <h2 id="editor-title">HTML source</h2>
            </div>
            <div className="wasm-actions">
              <button type="submit" className="button button-primary" data-testid="convert" disabled={status === 'loading'}>{status === 'loading' ? 'Converting...' : 'Convert locally'}</button>
              {status === 'loading' && <button type="button" className="button button-secondary" data-testid="cancel" onClick={cancel}>Cancel</button>}
              <button type="button" className="button button-secondary" data-testid="load-sample" onClick={loadSample} disabled={!selectedSample || status === 'loading'}>Load sample</button>
              <button type="button" className="button button-secondary" data-testid="reset" onClick={reset}>Reset</button>
              <span className="wasm-status" role="status" aria-live="polite">
                {status === 'loading' && `${progress.phase} ${progress.percent}%`}
                {status === 'success' && 'Conversion complete'}
                {status === 'cancelled' && 'Conversion cancelled'}
              </span>
            </div>
          </div>

          <div className="wasm-sample-picker">
            <label className="wasm-label" htmlFor="wasm-sample">Sample template</label>
            <select id="wasm-sample" value={selectedSample} onChange={(event) => setSelectedSample(event.target.value)} disabled={!samples.length || status === 'loading'}>
              {!samples.length && <option value="">Loading samples...</option>}
              {samples.map((sample) => <option value={sample.id} key={sample.id}>{sample.title}</option>)}
            </select>
            {samples.find((sample) => sample.id === selectedSample)?.description && <p className="wasm-sample-description">{samples.find((sample) => sample.id === selectedSample).description}</p>}
          </div>

          <div className="wasm-source-grid">
            <div className="wasm-source-field">
              <label className="wasm-label" htmlFor="wasm-html">HTML to convert</label>
              <textarea id="wasm-html" className="wasm-editor" value={html} onChange={(event) => setHTML(event.target.value)} spellCheck="false" />
            </div>
            <div className="wasm-source-field">
              <label className="wasm-label" htmlFor="wasm-css">CSS to convert</label>
              <textarea id="wasm-css" className="wasm-editor" value={css} onChange={(event) => setCSS(event.target.value)} spellCheck="false" />
            </div>
          </div>

          <fieldset className="wasm-output-options">
            <legend>Output format</legend>
            <div className="wasm-output-grid">
              {OUTPUTS.map((output) => (
                <label className={`wasm-output-option${mode === output.value ? ' selected' : ''}`} key={output.value}>
                  <input type="radio" name="wasm-output" value={output.value} checked={mode === output.value} onChange={() => setMode(output.value)} />
                  <span>
                    <strong>{output.label}</strong>
                    <small>{output.hint}</small>
                  </span>
                </label>
              ))}
            </div>
          </fieldset>
          {error && <p className="wasm-error" role="alert">{error}</p>}
        </section>

        <section className="wasm-panel wasm-preview-panel" aria-labelledby="preview-title">
          <div className="wasm-panel-heading">
            <div>
              <p className="wasm-kicker">02 / Preview</p>
              <h2 id="preview-title">Rendered output</h2>
            </div>
            {result && <div className="wasm-actions">
              <a className="button button-secondary" href={result.url} download={`gowkhtmltopdf.${result.mode}`}>Download</a>
              <a className="button button-secondary" href={result.url} target="_blank" rel="noreferrer">Open</a>
            </div>}
          </div>
          <div
            className={`wasm-preview${result ? ' wasm-preview-clickable' : ''}${result?.mode === 'png' ? ' wasm-transparent-preview' : ''}`}
            aria-live="polite"
            aria-label={result ? 'Open rendered output in expanded preview' : undefined}
            role={result ? 'button' : undefined}
            tabIndex={result ? 0 : undefined}
            onClick={result ? openPreview : undefined}
            onKeyDown={result ? (event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault()
                openPreview()
              }
            } : undefined}
          >
            {!result && status !== 'loading' && <p className="wasm-empty">Your preview will appear here.</p>}
            {status === 'loading' && <p className="wasm-empty">Rendering {mode.toUpperCase()} locally...</p>}
            {result?.mode === 'pdf' && <iframe className="wasm-pdf-preview" src={result.url} title="Generated PDF preview" />}
            {result && result.mode !== 'pdf' && <img className="wasm-image-preview" src={result.url} alt="Generated HTML image preview" />}
            {result && <span className="wasm-preview-hint">Click to open full preview</span>}
          </div>
          {result && result.mode !== 'pdf' && <p className="wasm-meta">{result.width} × {result.height}px · {result.mime}</p>}
        </section>
      </form>

      {previewOpen && result && <div className="wasm-lightbox" role="presentation" onMouseDown={(event) => {
        if (event.target === event.currentTarget) closePreview()
      }}>
        <section className="wasm-lightbox-card" role="dialog" aria-modal="true" aria-labelledby="expanded-preview-title">
          <header className="wasm-lightbox-heading">
            <div>
              <p className="wasm-kicker">Expanded preview</p>
              <h2 id="expanded-preview-title">{result.mode.toUpperCase()} output</h2>
            </div>
            <button ref={previewCloseRef} type="button" className="button button-secondary" data-testid="preview-close" onClick={closePreview}>Close</button>
          </header>
          <div className="wasm-lightbox-toolbar" role="toolbar" aria-label="Preview zoom controls">
            <button type="button" className="button button-secondary" data-testid="zoom-out" onClick={() => changePreviewZoom(-0.25)} disabled={previewZoom <= 0.5}>-</button>
            <output aria-live="polite">{Math.round(previewZoom * 100)}%</output>
            <button type="button" className="button button-secondary" data-testid="zoom-in" onClick={() => changePreviewZoom(0.25)} disabled={previewZoom >= 3}>+</button>
            <button type="button" className="button button-secondary" data-testid="zoom-reset" onClick={() => setPreviewZoom(1)}>Reset zoom</button>
            <span className="wasm-lightbox-help">Scroll inside the card to pan</span>
          </div>
          <div className={`wasm-lightbox-viewport${result.mode === 'png' ? ' wasm-transparent-preview' : ''}`} data-testid="preview-viewport">
            <div className="wasm-lightbox-canvas" style={{ width: `${previewZoom * 100}%`, minHeight: `${previewZoom * 100}%` }}>
              {result.mode === 'pdf' && <iframe className="wasm-lightbox-pdf" src={result.url} title="Expanded generated PDF preview" />}
              {result.mode !== 'pdf' && <img className="wasm-lightbox-image" src={result.url} alt="Expanded generated HTML image preview" />}
            </div>
          </div>
        </section>
      </div>}
    </div>
  )
}
