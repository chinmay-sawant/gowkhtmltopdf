export function createWasmClient() {
  const worker = new Worker(new URL('../workers/wasmWorker.js', import.meta.url), { type: 'classic' })
  const pending = new Map()
  let nextID = 1

  worker.onmessage = ({ data }) => {
    const request = pending.get(data.id)
    if (!request) return

    if (data.type === 'progress') {
      request.onProgress?.(data.phase, data.percent)
      return
    }

    pending.delete(data.id)
    if (data.type === 'error') {
      request.reject(new Error(data.error?.message || 'WASM conversion failed'))
      return
    }

    request.resolve(data)
  }

  const convert = (request, onProgress) => {
    const id = nextID++
    return new Promise((resolve, reject) => {
      pending.set(id, { resolve, reject, onProgress })
      const baseURL = new URL(import.meta.env.BASE_URL, window.location.origin)
      try {
        worker.postMessage({
          type: 'convert',
          id,
          request,
          runtimeURL: new URL('wasm/wasm_exec.js', baseURL).href,
          wasmURL: new URL('wasm/gowkhtmltopdf.wasm', baseURL).href,
        })
      } catch (error) {
        pending.delete(id)
        reject(error)
      }
    })
  }

  const cancel = () => {
    for (const request of pending.values()) request.reject(new Error('Conversion cancelled'))
    pending.clear()
    worker.terminate()
  }

  return { convert, cancel }
}
