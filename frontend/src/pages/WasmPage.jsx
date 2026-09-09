import { useEffect, useRef, useState } from 'react'
import { createWasmClient } from '../lib/wasmClient'
import { renderPDFPagesAsImages } from '../lib/renderImagePages'
import PdfViewer from '../components/PdfViewer'

const OUTPUTS = [
  { value: 'pdf', label: 'PDF document', hint: 'Preview pages in the browser' },
  { value: 'png', label: 'PNG image', hint: 'Lossless raster output' },
  { value: 'jpeg', label: 'JPEG image', hint: 'Compact raster output' },
]

const MAX_IMAGE_PADDING = 256

const INITIAL_HTML = '<!DOCTYPE html>\n<html>\n<body>\n  <h1>Hello from WASM</h1>\n</body>\n</html>'
const INITIAL_CSS = `@page { size: A4; margin: 18mm; }
body { color: #20242b; font-family: sans-serif; font-size: 12pt; }
h1 { color: #1f4b99; }`
const GITHUB_TREE_URL = 'https://api.github.com/repos/chinmay-sawant/gowkhtmltopdf/git/trees/master?recursive=1'
const GITHUB_RAW_BASE = 'https://raw.githubusercontent.com/chinmay-sawant/gowkhtmltopdf/master/'

const htmlAttribute = (tag, name) => tag.match(new RegExp(`${name}\\s*=\\s*["']([^"']+)["']`, 'i'))?.[1] || ''

const humanizeGoldenPath = (path) => path
  .replace(/^testdata\/golden\//, '')
  .replace(/\.html$/i, '')
  .replace(/[-_]+/g, ' ')
  .replace(/\b\w/g, (letter) => letter.toUpperCase())

const goldenRawURL = (path) => `${GITHUB_RAW_BASE}${path.split('/').map((segment) => encodeURIComponent(segment)).join('/')}`

const resolveGoldenPath = (htmlPath, reference) => {
  if (!reference || /^(?:[a-z]+:|\/\/|\/|#)/i.test(reference)) return ''
  const pathParts = [...htmlPath.split('/').slice(0, -1), ...reference.split(/[?#]/, 1)[0].split('/')]
  const normalized = []
  for (const part of pathParts) {
    if (!part || part === '.') continue
    if (part === '..') normalized.pop()
    else normalized.push(part)
  }
  const resolved = normalized.join('/')
  return resolved.startsWith('testdata/golden/') ? resolved : ''
}

const fetchGoldenText = async (path) => {
  const response = await fetch(goldenRawURL(path))
  if (!response.ok) throw new Error(`GitHub fixture failed to load (${response.status})`)
  return response.text()
}

const prepareGoldenSample = async (sourceHTML, htmlPath) => {
  const cssParts = []
  let html = sourceHTML.replace(/<style\b[^>]*>[\s\S]*?<\/style>/gi, (tag) => {
    cssParts.push(tag.replace(/^<style\b[^>]*>|<\/style>$/gi, ''))
    return ''
  })
  const stylesheetPaths = []
  html = html.replace(/<link\b[^>]*>/gi, (tag) => {
    const rel = htmlAttribute(tag, 'rel')
    const href = htmlAttribute(tag, 'href')
    const path = resolveGoldenPath(htmlPath, href)
    if (!rel.toLowerCase().split(/\s+/).includes('stylesheet') || !path || !path.endsWith('.css')) return tag
    stylesheetPaths.push(path)
    return ''
  })
  const linkedCSS = await Promise.all(stylesheetPaths.map((path) => fetchGoldenText(path).catch(() => '')))
  cssParts.push(...linkedCSS)
  return { html, css: cssParts.filter(Boolean).join('\n\n') }
}

const fetchGoldenSamples = async () => {
  const response = await fetch(GITHUB_TREE_URL, { headers: { Accept: 'application/vnd.github+json' } })
  if (!response.ok) throw new Error(`GitHub fixture catalog failed to load (${response.status})`)
  const tree = await response.json()
  return (tree.tree || [])
    .filter((entry) => entry.type === 'blob' && /^testdata\/golden\/.*\.html$/i.test(entry.path))
    .sort((left, right) => left.path.localeCompare(right.path))
    .map((entry) => ({
      id: `golden-${entry.path.replace(/^testdata\/golden\//, '').replace(/\.html$/i, '').replace(/[^a-z0-9]+/gi, '-').replace(/^-|-$/g, '').toLowerCase()}`,
      title: `Golden: ${humanizeGoldenPath(entry.path)}`,
      description: `GitHub fixture: ${entry.path}`,
      htmlNeedle: entry.path.split('/').pop().replace(/\.html$/i, ''),
      path: entry.path,
      source: 'golden',
    }))
}

const composeDocument = (sourceHTML, sourceCSS) => {
  const style = `<style>\n${sourceCSS}\n</style>`
  if (/<\/head>/i.test(sourceHTML)) return sourceHTML.replace(/<\/head>/i, `${style}\n</head>`)
  return `${style}\n${sourceHTML}`
}

const revokeResultURLs = (value) => {
  const urls = [value?.url, value?.zipUrl, value?.openUrl, ...(value?.pages || []).map((page) => page.url)]
  for (const url of new Set(urls.filter(Boolean))) URL.revokeObjectURL(url)
}

export default function WasmPage() {
  const [html, setHTML] = useState(INITIAL_HTML)
  const [css, setCSS] = useState(INITIAL_CSS)
  const [samples, setSamples] = useState([])
  const [selectedSample, setSelectedSample] = useState('')
  const [mode, setMode] = useState('pdf')
  const [padding, setPadding] = useState('20')
  const [status, setStatus] = useState('idle')
  const [error, setError] = useState('')
  const [result, setResult] = useState(null)
  const clientRef = useRef(null)
  const requestRef = useRef(0)
  const loadSampleRef = useRef(null)

  useEffect(() => () => {
    clientRef.current?.cancel()
    clientRef.current = null
  }, [])

  useEffect(() => () => {
    revokeResultURLs(result)
  }, [result])

  useEffect(() => {
    let cancelled = false

    const loadSampleCatalog = async () => {
      try {
        const baseURL = new URL(import.meta.env.BASE_URL, window.location.origin)
        const response = await fetch(new URL('wasm/samples/manifest.json', baseURL))
        if (!response.ok) throw new Error(`Sample catalog failed to load (${response.status})`)
        const catalog = await response.json()
        if (cancelled) return
        const catalogSamples = catalog.samples || []
        setSamples(catalogSamples)
        setSelectedSample(catalogSamples[0]?.id || '')
        if (catalogSamples[0]) void loadSampleRef.current?.(catalogSamples[0].id, true, catalogSamples)
        try {
          const goldenSamples = await fetchGoldenSamples()
          if (!cancelled) setSamples([...catalogSamples, ...goldenSamples])
        } catch (goldenError) {
          if (!cancelled) setError(goldenError.message)
        }
      } catch (catalogError) {
        if (!cancelled) setError(catalogError.message)
      }
    }

    loadSampleCatalog()
    return () => { cancelled = true }
  }, [])

  const convertDocument = async (sourceHTML, sourceCSS, outputMode, outputPadding = padding) => {
    if (!sourceHTML.trim()) {
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

    try {
      const imagePadding = Number(outputPadding)
      if (outputMode !== 'pdf' && (!Number.isInteger(imagePadding) || imagePadding < 0 || imagePadding > MAX_IMAGE_PADDING)) {
        throw new Error(`Image padding must be a whole number from 0 to ${MAX_IMAGE_PADDING}.`)
      }
      const request = { html: composeDocument(sourceHTML, sourceCSS), mode: 'pdf' }
      const response = await client.convert(request)
      if (requestRef.current !== requestID) return
      if (outputMode === 'pdf') {
        const bytes = new Uint8Array(response.bytes)
        const url = URL.createObjectURL(new Blob([bytes], { type: response.mime }))
        setResult({ ...response, url })
      } else {
        const rendered = await renderPDFPagesAsImages(new Uint8Array(response.bytes), {
          format: outputMode,
          padding: imagePadding,
          isCurrent: () => requestRef.current === requestID,
        })
        if (requestRef.current !== requestID) {
          revokeResultURLs(rendered)
          return
        }
        const firstPage = rendered.pages[0]
        setResult({ ...rendered, mode: outputMode, url: firstPage.url, width: firstPage.width, height: firstPage.height })
      }
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

  const loadSample = async (sampleID = selectedSample, autoConvert = false, sampleList = samples) => {
    const sample = sampleList.find((entry) => entry.id === sampleID)
    if (!sample) return

    try {
      let nextHTML
      let nextCSS
      if (sample.source === 'golden') {
        const sourceHTML = await fetchGoldenText(sample.path)
        const prepared = await prepareGoldenSample(sourceHTML, sample.path)
        nextHTML = prepared.html
        nextCSS = prepared.css
      } else {
        const baseURL = new URL(import.meta.env.BASE_URL, window.location.origin)
        const responses = await Promise.all([
          fetch(new URL(sample.html, baseURL)),
          fetch(new URL(sample.css, baseURL)),
        ])
        for (const response of responses) {
          if (!response.ok) throw new Error(`Sample asset failed to load (${response.status})`)
        }
        [nextHTML, nextCSS] = await Promise.all(responses.map((response) => response.text()))
      }
      setSelectedSample(sampleID)
      setHTML(nextHTML)
      setCSS(nextCSS)
      setMode('pdf')
      setResult(null)
      setStatus('idle')
      setError('')
      if (autoConvert) await convertDocument(nextHTML, nextCSS, 'pdf')
    } catch (sampleError) {
      setError(sampleError.message)
      setStatus('error')
    }
  }

  loadSampleRef.current = loadSample

  const selectSample = (sampleID) => {
    setSelectedSample(sampleID)
    setStatus('loading')
    setError('')
    setResult(null)
    void loadSample(sampleID, true)
  }

  const reset = () => {
    requestRef.current += 1
    clientRef.current?.cancel()
    clientRef.current = null
    setHTML(INITIAL_HTML)
    setCSS(INITIAL_CSS)
    setSelectedSample(samples[0]?.id || '')
    setMode('pdf')
    setPadding('20')
    setStatus('idle')
    setError('')
    setResult(null)
  }

  const convert = (event) => {
    event.preventDefault()
    void convertDocument(html, css, mode)
  }

  const selectOutput = (outputMode) => {
    setMode(outputMode)
    void convertDocument(html, css, outputMode)
  }

  return (
    <div className="wasm-page">
      <section className="wasm-hero" aria-labelledby="wasm-title">
        <p className="wasm-kicker">Browser tool</p>
        <h1 id="wasm-title">Convert HTML in your browser.</h1>
        <p className="lede">The Go engine runs locally through WebAssembly. Choose a document or image output, then preview the result without uploading your HTML.</p>
      </section>

      <form className="wasm-workspace" onSubmit={convert} aria-busy={status === 'loading'}>
        <section className="wasm-panel wasm-editor-panel" aria-labelledby="editor-title">
          <div className="wasm-panel-heading">
            <div>
              <p className="wasm-kicker">01 / Input</p>
              <h2 id="editor-title">HTML source</h2>
            </div>
          </div>

          <div className="wasm-sample-picker">
            <label className="wasm-label" htmlFor="wasm-sample">Sample template</label>
            <select id="wasm-sample" value={selectedSample} onChange={(event) => selectSample(event.target.value)} disabled={!samples.length || status === 'loading'}>
              {!samples.length && <option value="">Loading samples...</option>}
              {[
                { label: 'Curated templates', entries: samples.filter((sample) => sample.source !== 'golden') },
                { label: 'Golden fixtures', entries: samples.filter((sample) => sample.source === 'golden') },
              ].map((group) => group.entries.length > 0 && (
                <optgroup label={group.label} key={group.label}>
                  {group.entries.map((sample) => <option value={sample.id} key={sample.id}>{sample.title}</option>)}
                </optgroup>
              ))}
            </select>
            {samples.find((sample) => sample.id === selectedSample)?.description && <p className="wasm-sample-description">{samples.find((sample) => sample.id === selectedSample).description}</p>}
          </div>

          <fieldset className="wasm-output-options">
            <legend>Output format</legend>
            <div className="wasm-output-grid">
              {OUTPUTS.map((output) => (
                <label className={`wasm-output-option${mode === output.value ? ' selected' : ''}`} key={output.value}>
                  <input type="radio" name="wasm-output" value={output.value} checked={mode === output.value} onChange={() => selectOutput(output.value)} disabled={status === 'loading'} />
                  <span>
                    <strong>{output.label}</strong>
                    <small>{output.hint}</small>
                  </span>
                </label>
              ))}
            </div>
          </fieldset>

          {mode !== 'pdf' && (
            <div className="wasm-image-options">
              <label className="wasm-label" htmlFor="wasm-padding">Image padding (px)</label>
              <input
                id="wasm-padding"
                data-testid="image-padding"
                className="wasm-number-input"
                type="number"
                min="0"
                max={MAX_IMAGE_PADDING}
                step="1"
                value={padding}
                onChange={(event) => setPadding(event.target.value)}
                disabled={status === 'loading'}
                aria-describedby="wasm-padding-help"
              />
              <p id="wasm-padding-help" className="wasm-field-help">Adds the same transparent or white space to all four edges of PNG and JPEG output.</p>
            </div>
          )}

          <div className="wasm-actions wasm-editor-actions">
            <button type="submit" className="button button-primary" data-testid="convert" disabled={status === 'loading'}>Convert locally</button>
            <button type="button" className="button button-secondary" data-testid="load-sample" onClick={() => loadSample()} disabled={!selectedSample || status === 'loading'}>Load sample</button>
            <button type="button" className="button button-secondary" data-testid="reset" onClick={reset}>Reset</button>
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

          {error && <p className="wasm-error" role="alert">{error}</p>}
        </section>

        <section className="wasm-panel wasm-preview-panel" aria-labelledby="preview-title">
          <div className="wasm-panel-heading">
            <div>
              <p className="wasm-kicker">02 / Preview</p>
              <h2 id="preview-title">Rendered output</h2>
            </div>
            {result && <div className="wasm-actions">
              {result.pages?.length > 1 ? (
                <a className="button button-secondary" data-testid="download-zip" href={result.zipUrl} download="gowkhtmltopdf-pages.zip">Download ZIP</a>
              ) : (
                <a className="button button-secondary" href={result.url} download={`gowkhtmltopdf.${result.mode}`}>Download</a>
              )}
              {result.pages?.length > 1 ? (
                <a className="button button-secondary" data-testid="open-pages" href={result.openUrl} target="_blank" rel="noreferrer">Open</a>
              ) : (
                <a className="button button-secondary" href={result.url} target="_blank" rel="noreferrer">Open</a>
              )}
            </div>}
          </div>
          <div className={`wasm-preview${result?.mode === 'png' ? ' wasm-transparent-preview' : ''}`} aria-live="polite">
            {!result && <p className="wasm-empty">Your preview will appear here.</p>}
            {result?.mode === 'pdf' && <PdfViewer src={result.url} title="Generated PDF" compact />}
            {result && result.mode !== 'pdf' && (
              <div className="wasm-image-pages" data-testid="image-pages" aria-label={`${result.mode.toUpperCase()} output pages`}>
                {result.pages.map((page) => (
                  <div className="wasm-image-page" data-testid="image-page" key={page.number}>
                    <img className="wasm-image-preview" src={page.url} alt={`Generated HTML ${result.mode.toUpperCase()} page ${page.number}`} />
                  </div>
                ))}
              </div>
            )}
          </div>
          {result && result.mode !== 'pdf' && <p className="wasm-meta">{result.pageCount} {result.pageCount === 1 ? 'page' : 'pages'} · {result.width} × {result.height}px · {result.mime}</p>}
        </section>
      </form>
    </div>
  )
}
