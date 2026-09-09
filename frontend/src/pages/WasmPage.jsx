import { useEffect, useRef, useState } from 'react'
import { createWasmClient } from '../lib/wasmClient'

const OUTPUTS = [
  { value: 'pdf', label: 'PDF document', hint: 'Preview pages in the browser' },
  { value: 'png', label: 'PNG image', hint: 'Lossless raster output' },
  { value: 'jpeg', label: 'JPEG image', hint: 'Compact raster output' },
]

const INITIAL_HTML = '<!DOCTYPE html>\n<html>\n<body>\n  <h1>Hello from WASM</h1>\n</body>\n</html>'

export default function WasmPage() {
  const [html, setHTML] = useState(INITIAL_HTML)
  const [mode, setMode] = useState('pdf')
  const [status, setStatus] = useState('idle')
  const [progress, setProgress] = useState({ phase: '', percent: 0 })
  const [error, setError] = useState('')
  const [result, setResult] = useState(null)
  const clientRef = useRef(null)
  const requestRef = useRef(0)

  useEffect(() => () => clientRef.current?.cancel(), [])

  useEffect(() => () => {
    if (result?.url) URL.revokeObjectURL(result.url)
  }, [result])

  const loadSample = async () => {
    try {
      const baseURL = new URL(import.meta.env.BASE_URL, window.location.origin)
      const response = await fetch(new URL('wasm/sample.html', baseURL))
      if (!response.ok) throw new Error(`Sample fixture failed to load (${response.status})`)
      setHTML(await response.text())
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
    setResult(null)
    setProgress({ phase: 'Starting', percent: 0 })

    try {
      const response = await client.convert({ html, mode }, (phase, percent) => {
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
              <button type="button" className="button button-secondary" data-testid="load-sample" onClick={loadSample}>Load sample</button>
              <button type="button" className="button button-secondary" data-testid="reset" onClick={reset}>Reset</button>
            </div>
          </div>

          <label className="wasm-label" htmlFor="wasm-html">HTML to convert</label>
          <textarea id="wasm-html" className="wasm-editor" value={html} onChange={(event) => setHTML(event.target.value)} spellCheck="false" />

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

          <div className="wasm-submit-row">
            <button type="submit" className="button button-primary" data-testid="convert" disabled={status === 'loading'}>{status === 'loading' ? 'Converting...' : 'Convert locally'}</button>
            {status === 'loading' && <button type="button" className="button button-secondary" data-testid="cancel" onClick={cancel}>Cancel</button>}
            <span className="wasm-status" role="status" aria-live="polite">
              {status === 'loading' && `${progress.phase} ${progress.percent}%`}
              {status === 'success' && 'Conversion complete'}
              {status === 'cancelled' && 'Conversion cancelled'}
            </span>
          </div>
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
          <div className="wasm-preview" aria-live="polite">
            {!result && status !== 'loading' && <p className="wasm-empty">Your preview will appear here.</p>}
            {status === 'loading' && <p className="wasm-empty">Rendering {mode.toUpperCase()} locally...</p>}
            {result?.mode === 'pdf' && <iframe className="wasm-pdf-preview" src={result.url} title="Generated PDF preview" />}
            {result && result.mode !== 'pdf' && <img className="wasm-image-preview" src={result.url} alt="Generated HTML image preview" />}
          </div>
          {result && result.mode !== 'pdf' && <p className="wasm-meta">{result.width} × {result.height}px · {result.mime}</p>}
        </section>
      </form>
    </div>
  )
}
