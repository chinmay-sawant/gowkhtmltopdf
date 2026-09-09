import * as pdfjsLib from 'pdfjs-dist/build/pdf.mjs'
import pdfjsWorker from 'pdfjs-dist/build/pdf.worker.min.mjs?url'

pdfjsLib.GlobalWorkerOptions.workerSrc = pdfjsWorker

const IMAGE_WIDTH = 1024
const DATA_CHUNK_SIZE = 0x8000

const canvasBlob = (canvas, type, quality) => new Promise((resolve, reject) => {
  canvas.toBlob((blob) => {
    if (blob) resolve(blob)
    else reject(new Error(`Could not encode ${type} output.`))
  }, type, quality || undefined)
})

const blobDataURL = async (blob) => {
  const bytes = new Uint8Array(await blob.arrayBuffer())
  let binary = ''
  for (let offset = 0; offset < bytes.length; offset += DATA_CHUNK_SIZE) {
    binary += String.fromCharCode(...bytes.subarray(offset, offset + DATA_CHUNK_SIZE))
  }
  return `data:${blob.type};base64,${btoa(binary)}`
}

const crcTable = (() => {
  const table = new Uint32Array(256)
  for (let index = 0; index < table.length; index += 1) {
    let value = index
    for (let bit = 0; bit < 8; bit += 1) value = value & 1 ? 0xedb88320 ^ (value >>> 1) : value >>> 1
    table[index] = value >>> 0
  }
  return table
})()

const crc32 = (bytes) => {
  let value = 0xffffffff
  for (const byte of bytes) value = crcTable[(value ^ byte) & 0xff] ^ (value >>> 8)
  return (value ^ 0xffffffff) >>> 0
}

const write16 = (target, offset, value) => {
  target[offset] = value & 0xff
  target[offset + 1] = (value >>> 8) & 0xff
}

const write32 = (target, offset, value) => {
  target[offset] = value & 0xff
  target[offset + 1] = (value >>> 8) & 0xff
  target[offset + 2] = (value >>> 16) & 0xff
  target[offset + 3] = (value >>> 24) & 0xff
}

const zipImages = async (pages) => {
  const encoder = new TextEncoder()
  const files = await Promise.all(pages.map(async (page) => ({
    name: encoder.encode(page.name),
    data: new Uint8Array(await page.blob.arrayBuffer()),
  })))
  const localSize = files.reduce((size, file) => size + 30 + file.name.length + file.data.length, 0)
  const centralSize = files.reduce((size, file) => size + 46 + file.name.length, 0)
  const output = new Uint8Array(localSize + centralSize + 22)
  const centralOffset = localSize
  const localOffsets = []
  let offset = 0

  for (const file of files) {
    localOffsets.push(offset)
    const checksum = crc32(file.data)
    write32(output, offset, 0x04034b50)
    write16(output, offset + 4, 20)
    write16(output, offset + 6, 0)
    write16(output, offset + 8, 0)
    write16(output, offset + 10, 0)
    write16(output, offset + 12, 0)
    write32(output, offset + 14, checksum)
    write32(output, offset + 18, file.data.length)
    write32(output, offset + 22, file.data.length)
    write16(output, offset + 26, file.name.length)
    write16(output, offset + 28, 0)
    output.set(file.name, offset + 30)
    output.set(file.data, offset + 30 + file.name.length)
    offset += 30 + file.name.length + file.data.length
  }

  let central = centralOffset
  for (const [index, file] of files.entries()) {
    const checksum = crc32(file.data)
    write32(output, central, 0x02014b50)
    write16(output, central + 4, 20)
    write16(output, central + 6, 20)
    write16(output, central + 8, 0)
    write16(output, central + 10, 0)
    write16(output, central + 12, 0)
    write16(output, central + 14, 0)
    write32(output, central + 16, checksum)
    write32(output, central + 20, file.data.length)
    write32(output, central + 24, file.data.length)
    write16(output, central + 28, file.name.length)
    write16(output, central + 30, 0)
    write16(output, central + 32, 0)
    write16(output, central + 34, 0)
    write16(output, central + 36, 0)
    write32(output, central + 38, 0)
    write32(output, central + 42, localOffsets[index])
    output.set(file.name, central + 46)
    central += 46 + file.name.length
  }

  write32(output, central, 0x06054b50)
  write16(output, central + 4, 0)
  write16(output, central + 6, 0)
  write16(output, central + 8, files.length)
  write16(output, central + 10, files.length)
  write32(output, central + 12, centralSize)
  write32(output, central + 16, centralOffset)
  write16(output, central + 20, 0)

  return new Blob([output], { type: 'application/zip' })
}

const openGallery = async (pages, format) => {
  const images = await Promise.all(pages.map(async (page) => `
    <img src="${await blobDataURL(page.blob)}" alt="Generated ${format.toUpperCase()} page ${page.number}">
  `))
  const markup = `<!doctype html><html><head><meta charset="utf-8"><title>Generated ${format.toUpperCase()} pages</title><style>body{margin:0;padding:32px;background:#18201d;color:#edf4f0;font:16px system-ui,sans-serif}main{display:grid;gap:28px;justify-items:center}img{display:block;max-width:100%;height:auto;background:white;box-shadow:0 10px 28px rgb(0 0 0 / 28%)}</style></head><body><main>${images.join('')}</main></body></html>`
  return URL.createObjectURL(new Blob([markup], { type: 'text/html' }))
}

export async function renderPDFPagesAsImages(bytes, { format, padding, isCurrent }) {
  const mime = format === 'jpeg' ? 'image/jpeg' : 'image/png'
  const loadingTask = pdfjsLib.getDocument({ data: bytes })
  const pdf = await loadingTask.promise
  const pages = []

  try {
    for (let number = 1; number <= pdf.numPages; number += 1) {
      if (!isCurrent()) throw new Error('Conversion cancelled')
      const page = await pdf.getPage(number)
      const baseViewport = page.getViewport({ scale: 1 })
      const viewport = page.getViewport({ scale: IMAGE_WIDTH / baseViewport.width })
      const canvas = document.createElement('canvas')
      canvas.width = Math.ceil(viewport.width) + padding * 2
      canvas.height = Math.ceil(viewport.height) + padding * 2
      const context = canvas.getContext('2d', { alpha: true })
      if (!context) throw new Error('Could not create an image canvas.')
      if (format === 'jpeg') {
        context.fillStyle = '#fff'
        context.fillRect(0, 0, canvas.width, canvas.height)
      }
      await page.render({
        canvasContext: context,
        viewport,
        background: format === 'jpeg' ? '#fff' : 'rgba(0, 0, 0, 0)',
        transform: [1, 0, 0, 1, padding, padding],
      }).promise
      const blob = await canvasBlob(canvas, mime)
      pages.push({ number, name: `gowkhtmltopdf-page-${number}.${format}`, blob, url: URL.createObjectURL(blob), width: canvas.width, height: canvas.height })
    }

    const zipBlob = pages.length > 1 ? await zipImages(pages) : null
    const openUrl = pages.length > 1 ? await openGallery(pages, format) : pages[0]?.url
    return { mime, pages, zipUrl: zipBlob ? URL.createObjectURL(zipBlob) : null, openUrl, pageCount: pages.length }
  } catch (error) {
    for (const page of pages) URL.revokeObjectURL(page.url)
    throw error
  } finally {
    await pdf.destroy()
  }
}
