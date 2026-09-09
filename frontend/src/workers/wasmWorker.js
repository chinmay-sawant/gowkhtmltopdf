/* global importScripts */

let runtimePromise

async function instantiateWasm(wasmURL, go) {
  if (WebAssembly.instantiateStreaming) {
    try {
      return await WebAssembly.instantiateStreaming(fetch(wasmURL), go.importObject)
    } catch (error) {
      if (!(error instanceof TypeError)) throw error
    }
  }

  const response = await fetch(wasmURL)
  const bytes = await response.arrayBuffer()
  return WebAssembly.instantiate(bytes, go.importObject)
}

async function startRuntime(runtimeURL, wasmURL) {
  if (!self.Go) importScripts(runtimeURL)
  if (!self.Go) throw new Error('Go WASM runtime did not load')

  const go = new self.Go()
  const result = await instantiateWasm(wasmURL, go)
  go.run(result.instance).catch(() => {
    runtimePromise = undefined
  })

  const deadline = Date.now() + 10000
  while (typeof self.gowkhtmltopdfWASM !== 'function') {
    if (Date.now() >= deadline) throw new Error('WASM conversion bridge did not start')
    await new Promise((resolve) => setTimeout(resolve, 10))
  }
}

function ensureRuntime(runtimeURL, wasmURL) {
  if (!runtimePromise) {
    runtimePromise = startRuntime(runtimeURL, wasmURL).catch((error) => {
      runtimePromise = undefined
      throw error
    })
  }
  return runtimePromise
}

self.onmessage = async ({ data }) => {
  if (!data || data.type !== 'convert') return

  try {
    await ensureRuntime(data.runtimeURL, data.wasmURL)
    const response = self.gowkhtmltopdfWASM(
      JSON.stringify(data.request),
      (phase, percent) => self.postMessage({
        type: 'progress',
        id: data.id,
        phase,
        percent,
      }),
    )

    if (!response.ok) {
      self.postMessage({ type: 'error', id: data.id, error: response.error })
      return
    }

    self.postMessage({
      type: 'result',
      id: data.id,
      mode: response.mode,
      mime: response.mime,
      width: response.width,
      height: response.height,
      bytes: response.bytes,
    }, [response.bytes.buffer])
  } catch (error) {
    self.postMessage({
      type: 'error',
      id: data.id,
      error: {
        code: 'worker_error',
        message: error instanceof Error ? error.message : String(error),
      },
    })
  }
}
