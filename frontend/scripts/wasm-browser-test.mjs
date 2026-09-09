#!/usr/bin/env node
import { spawn } from 'node:child_process'
import { createRequire } from 'node:module'
import { setTimeout as delay } from 'node:timers/promises'
import { join } from 'node:path'
import assert from 'node:assert/strict'

const require = createRequire(import.meta.url)
const puppeteer = require(join('..', '..', 'scripts', 'puppeteer', 'node_modules', 'puppeteer-core'))
const frontendDir = join(import.meta.dirname, '..')
const viteCLI = join(frontendDir, 'node_modules', 'vite', 'bin', 'vite.js')
const port = 4173
const origin = `http://127.0.0.1:${port}`
const url = `${origin}/gowkhtmltopdf/#/wasm`
const conversionTimeout = 60000

const server = spawn(process.execPath, [viteCLI, 'preview', '--host', '127.0.0.1', '--port', String(port)], {
  cwd: frontendDir,
  stdio: ['ignore', 'pipe', 'pipe'],
})

async function waitForServer() {
  for (let attempt = 0; attempt < 50; attempt += 1) {
    try {
      const response = await fetch(`${origin}/gowkhtmltopdf/`)
      if (response.ok) return
    } catch {
      // Vite is still starting.
    }
    await delay(100)
  }
  throw new Error('frontend preview server did not start')
}

try {
  await waitForServer()
  const browser = await puppeteer.launch({
    executablePath: process.env.PUPPETEER_EXECUTABLE_PATH || '/usr/bin/google-chrome',
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage', '--disable-gpu'],
  })

  try {
    const page = await browser.newPage()
    await page.setViewport({ width: 1440, height: 900 })
    await page.setRequestInterception(true)
    page.on('request', (request) => {
      const requestURL = new URL(request.url())
      const localRequest = requestURL.origin === origin
      const githubRequest = requestURL.origin === 'https://api.github.com' || requestURL.origin === 'https://raw.githubusercontent.com'
      const embeddedRequest = requestURL.protocol === 'data:' || requestURL.protocol === 'blob:' || requestURL.protocol === 'about:' || requestURL.protocol === 'chrome-extension:'
      if (localRequest || githubRequest || embeddedRequest) request.continue()
      else request.abort()
    })
    await page.goto(url, { waitUntil: 'networkidle0' })
    await page.waitForSelector('#wasm-html')
    const convertButton = await page.$eval('[data-testid="convert"]', (element) => {
      const rect = element.getBoundingClientRect()
      const style = getComputedStyle(element)
      return {
        top: rect.top,
        bottom: rect.bottom,
        inActionGroup: Boolean(element.closest('.wasm-actions')),
        backgroundColor: style.backgroundColor,
      }
    })
    assert.ok(convertButton.top < 900 && convertButton.bottom > 0, `Convert button should be visible in the initial viewport: ${JSON.stringify(convertButton)}`)
    assert.equal(convertButton.inActionGroup, true, 'Convert button should share the visible input action group')
    assert.notEqual(convertButton.backgroundColor, 'rgba(0, 0, 0, 0)', 'Convert button should have a visible background')
    const layout = await page.evaluate(() => ({
      pageWidth: document.querySelector('.wasm-page').getBoundingClientRect().width,
      viewportWidth: window.innerWidth,
      navLabel: document.querySelector('.site-nav-desktop .site-nav-links a:last-child')?.textContent.trim(),
      navWidth: document.querySelector('.site-nav-desktop .site-nav-links a:last-child')?.getBoundingClientRect().width,
      sourceWidths: [...document.querySelectorAll('.wasm-source-field')].map((element) => element.getBoundingClientRect().width),
    }))
    assert.equal(layout.navLabel, 'Try Live Demo', 'the WASM navigation label should describe the live demo')
    assert.ok(layout.navWidth >= 124, `the live-demo navigation target should have a wider hit area: ${JSON.stringify(layout)}`)
    assert.ok(layout.pageWidth >= layout.viewportWidth * 0.84 && layout.pageWidth <= layout.viewportWidth * 0.86, `the desktop WASM page should use an 85vw frame: ${JSON.stringify(layout)}`)
    assert.ok(Math.abs(layout.sourceWidths[0] - layout.sourceWidths[1]) <= 2, `HTML and CSS editors should have equal widths: ${JSON.stringify(layout)}`)
    assert.equal(await page.$eval('.wasm-output-options + .wasm-editor-actions', (element) => Boolean(element.querySelector('[data-testid="convert"]'))), true, 'conversion buttons should follow the output options')
    assert.equal(await page.$eval('.wasm-editor-actions + .wasm-source-grid', () => true), true, 'conversion buttons should precede the source editors')
    assert.equal(await page.$('.wasm-status'), null, 'the WASM page should not render a status label')
    assert.equal(await page.$('[data-testid="cancel"]'), null, 'the WASM page should not render a cancel control')
    assert.doesNotMatch(await page.$eval('.wasm-panel', (element) => element.innerText), /Conversion complete|Starting|Rendering|Cancel/, 'the WASM panel should not show conversion status copy')
    await page.emulateMediaFeatures([{ name: 'prefers-reduced-motion', value: 'reduce' }])
    assert.equal(await page.$eval('.wasm-output-option', (element) => getComputedStyle(element).transitionDuration), '0s', 'reduced motion should disable transitions')
    await page.click('.theme-toggle')
    await page.waitForFunction(() => document.documentElement.dataset.theme === 'dark')
    await page.click('.theme-toggle')
    await page.waitForFunction(() => document.documentElement.dataset.theme === 'light')

    const sampleCatalog = await page.evaluate(async () => {
      const response = await fetch(new URL('wasm/samples/manifest.json', document.baseURI))
      return response.json()
    })
    const curatedSamples = sampleCatalog.samples
    assert.equal(curatedSamples.length, 5, 'sample catalog should preserve five curated templates')
    await page.waitForSelector('option[value="golden-fixture-60-implemented-props-a"]', { timeout: conversionTimeout })
    const goldenCount = await page.$$eval('#wasm-sample option[value^="golden-"]', (options) => options.length)
    assert.ok(goldenCount >= 60, `sample catalog should expose the GitHub golden fixture corpus: ${goldenCount}`)

    const defaultSample = curatedSamples[0]
    await page.waitForFunction((sample) => document.querySelector('#wasm-html')?.value.includes(sample.htmlNeedle) && document.querySelector('#wasm-css')?.value.includes(sample.cssNeedle), {}, defaultSample)
    try {
      await page.waitForFunction(() => document.querySelector('.pdf-viewer') || document.querySelector('.wasm-error'), { timeout: conversionTimeout })
    } catch (conversionWaitError) {
      const state = await page.$eval('.wasm-panel', (element) => element.innerText)
      throw new Error(`default sample conversion timed out. Current panel state: ${state}`, { cause: conversionWaitError })
    }
    assert.equal(await page.$eval('.wasm-error', (element) => element.textContent, { timeout: 1000 }).catch(() => ''), '', 'the default sample should convert without an error')
    assert.equal(await page.$eval('input[name="wasm-output"][value="pdf"]', (element) => element.checked), true, 'the default sample should select PDF output')
    assert.ok(await page.$('.pdf-viewer'), 'the default sample should render a PDF preview')

    const autoSample = curatedSamples[1]
    await page.select('#wasm-sample', autoSample.id)
    await page.waitForFunction((sample) => document.querySelector('#wasm-html')?.value.includes(sample.htmlNeedle) && document.querySelector('#wasm-css')?.value.includes(sample.cssNeedle), {}, autoSample)
    try {
      await page.waitForFunction(() => document.querySelector('.pdf-viewer') || document.querySelector('.wasm-error'), { timeout: conversionTimeout })
    } catch (conversionWaitError) {
      const state = await page.$eval('.wasm-panel', (element) => element.innerText)
      throw new Error(`automatic ${autoSample.id} conversion timed out. Current panel state: ${state}`, { cause: conversionWaitError })
    }
    assert.equal(await page.$eval('.wasm-error', (element) => element.textContent, { timeout: 1000 }).catch(() => ''), '', `${autoSample.id} should convert without an error`)
    assert.equal(await page.$eval('input[name="wasm-output"][value="pdf"]', (element) => element.checked), true, 'selecting a sample should keep PDF as the default output')
    await page.click('#wasm-html')
    assert.equal(await page.$eval('#wasm-html', (element) => document.activeElement === element), true, 'HTML editor should receive keyboard focus')

    for (const sample of curatedSamples) {
      await page.select('#wasm-sample', sample.id)
      await page.waitForFunction((currentSample) => document.querySelector('#wasm-html')?.value.includes(currentSample.htmlNeedle) && document.querySelector('#wasm-css')?.value.includes(currentSample.cssNeedle), {}, sample)
      await page.waitForFunction(() => document.querySelector('.pdf-viewer') || document.querySelector('.wasm-error'), { timeout: conversionTimeout })
      await page.click('[data-testid="convert"]')
      try {
        await page.waitForFunction(() => document.querySelector('.pdf-viewer') || document.querySelector('.wasm-error'), { timeout: conversionTimeout })
      } catch (conversionWaitError) {
        const state = await page.$eval('.wasm-panel', (element) => element.innerText)
        throw new Error(`${sample.id} conversion timed out. Current panel state: ${state}`, { cause: conversionWaitError })
      }
      assert.equal(await page.$eval('.wasm-error', (element) => element.textContent, { timeout: 1000 }).catch(() => ''), '', `${sample.id} should convert without an error`)
      const samplePDF = await page.evaluate(async () => {
        const response = await fetch(document.querySelector('.pdf-viewer').dataset.src)
        const bytes = new Uint8Array(await response.arrayBuffer())
        const text = new TextDecoder().decode(bytes)
        return { bytes: bytes.length, pages: (text.match(/\/Type \/Page\b/g) || []).length }
      })
      assert.ok(samplePDF.bytes > 1000, `${sample.id} PDF should contain rendered bytes`)
      assert.ok(samplePDF.pages >= 2 && samplePDF.pages <= 3, `${sample.id} should render two or three pages, got ${samplePDF.pages}`)
    }

    const goldenSample = { id: 'golden-fixture-60-implemented-props-a', htmlNeedle: 'fixture-60-implemented-props-a' }
    await page.select('#wasm-sample', goldenSample.id)
    await page.waitForFunction((sample) => document.querySelector('#wasm-html')?.value.includes(sample.htmlNeedle) && document.querySelector('#wasm-css')?.value.includes('@page'), {}, goldenSample)
    assert.doesNotMatch(await page.$eval('#wasm-html', (element) => element.value), /<style\b/i, 'golden fixture CSS should be extracted from the HTML editor')
    await page.waitForFunction(() => document.querySelector('.pdf-viewer') || document.querySelector('.wasm-error'), { timeout: conversionTimeout })
    assert.equal(await page.$eval('.wasm-error', (element) => element.textContent, { timeout: 1000 }).catch(() => ''), '', 'a golden fixture should convert without an error')

    await page.select('#wasm-sample', curatedSamples[0].id)
    await page.waitForFunction((sample) => document.querySelector('#wasm-html')?.value.includes(sample.htmlNeedle), {}, curatedSamples[0])
    await page.waitForFunction(() => document.querySelector('.pdf-viewer') || document.querySelector('.wasm-error'), { timeout: conversionTimeout })
    await page.click('[data-testid="load-sample"]')

    try {
      await page.waitForFunction((sample) => document.querySelector('#wasm-html')?.value.includes(sample.htmlNeedle), {}, curatedSamples[0])
    } catch (sampleWaitError) {
      const state = await page.$eval('.wasm-panel', (element) => element.innerText)
      throw new Error(`manual sample load timed out. Current panel state: ${state}`, { cause: sampleWaitError })
    }

    let previousPreviewURL = null
    for (const mode of ['pdf', 'png', 'jpeg']) {
      if (mode === 'pdf') await page.click('[data-testid="convert"]')
      else await page.click(`input[name="wasm-output"][value="${mode}"]`)
      const previewSelector = mode === 'pdf' ? '.pdf-viewer' : '.wasm-image-preview'
      try {
        await page.waitForSelector(previewSelector, { timeout: conversionTimeout })
      } catch (conversionWaitError) {
        const state = await page.$eval('.wasm-panel', (element) => element.innerText)
        throw new Error(`${mode} conversion timed out. Current panel state: ${state}`, { cause: conversionWaitError })
      }
      assert.equal(await page.$eval('.wasm-error', (element) => element.textContent, { timeout: 1000 }).catch(() => ''), '', `${mode} should convert without an error`)

      const result = await page.evaluate(async (currentMode) => {
        const target = currentMode === 'pdf' ? document.querySelector('.pdf-viewer') : document.querySelector('.wasm-image-preview')
        const response = await fetch(currentMode === 'pdf' ? target.dataset.src : target.src)
        const bytes = new Uint8Array(await response.arrayBuffer())
        return {
          bytes: Array.from(bytes.slice(0, 8)),
          byteLength: bytes.length,
          tail: Array.from(bytes.slice(-5)),
          mime: response.headers.get('content-type'),
          width: target.naturalWidth || 0,
          height: target.naturalHeight || 0,
        }
      }, mode)

      assert.ok(result.bytes.length > 0, `${mode} preview should contain bytes`)
      if (mode === 'pdf') {
        assert.deepEqual(result.bytes.slice(0, 5), [37, 80, 68, 70, 45], 'PDF signature')
        assert.deepEqual(result.tail, [37, 69, 79, 70, 10], 'PDF trailer')
        assert.match(result.mime || '', /^application\/pdf/, 'PDF MIME type')
        assert.ok(result.byteLength > 1000, 'PDF preview should contain rendered bytes')
      } else {
        assert.ok(result.width > 0 && result.height > 0, `${mode} preview should have dimensions`)
        const imagePages = await page.$$('[data-testid="image-page"]')
        assert.ok(imagePages.length >= 2, `${mode} preview should display every rendered page, got ${imagePages.length}`)
        assert.ok(await page.$('[data-testid="download-zip"]'), `${mode} should offer a ZIP download for multiple pages`)
        assert.ok(await page.$('[data-testid="open-pages"]'), `${mode} should offer an Open action`)
        assert.equal(await page.$eval('[data-testid="open-pages"]', (element) => element.textContent.trim()), 'Open', `${mode} should label the image gallery action Open`)
        const archive = await page.evaluate(async () => {
          const response = await fetch(document.querySelector('[data-testid="download-zip"]').href)
          const bytes = new Uint8Array(await response.arrayBuffer())
          const read16 = (offset) => bytes[offset] | (bytes[offset + 1] << 8)
          const read32 = (offset) => (bytes[offset] | (bytes[offset + 1] << 8) | (bytes[offset + 2] << 16) | (bytes[offset + 3] << 24)) >>> 0
          const hash = (data) => data.reduce((value, byte) => ((value * 31) + byte) >>> 0, 7)
          let endOfCentralDirectory = -1
          for (let offset = bytes.length - 22; offset >= 0; offset -= 1) {
            if (read32(offset) === 0x06054b50) {
              endOfCentralDirectory = offset
              break
            }
          }
          if (endOfCentralDirectory < 0) throw new Error('ZIP end-of-central-directory record is missing')
          const centralEntries = read16(endOfCentralDirectory + 10)
          const centralOffset = read32(endOfCentralDirectory + 16)
          const entries = []
          let central = centralOffset
          for (let index = 0; index < centralEntries; index += 1) {
            if (read32(central) !== 0x02014b50) throw new Error(`ZIP central entry ${index + 1} is missing`)
            const nameLength = read16(central + 28)
            const extraLength = read16(central + 30)
            const commentLength = read16(central + 32)
            const localOffset = read32(central + 42)
            if (read32(localOffset) !== 0x04034b50) throw new Error(`ZIP entry ${index + 1} points to an invalid local header`)
            const localNameLength = read16(localOffset + 26)
            const localExtraLength = read16(localOffset + 28)
            const dataLength = read32(localOffset + 18)
            const dataStart = localOffset + 30 + localNameLength + localExtraLength
            entries.push({ dataHash: hash(bytes.slice(dataStart, dataStart + dataLength)), localOffset })
            central += 46 + nameLength + extraLength + commentLength
          }
          return { signature: Array.from(bytes.slice(0, 4)), centralEntries, entries }
        })
        assert.deepEqual(archive.signature, [80, 75, 3, 4], `${mode} ZIP should have a local-file header`)
        assert.equal(archive.centralEntries, imagePages.length, `${mode} ZIP should contain one file per rendered page`)
        assert.equal(new Set(archive.entries.map((entry) => entry.localOffset)).size, imagePages.length, `${mode} ZIP should point each entry at a different local file`)
        assert.equal(new Set(archive.entries.map((entry) => entry.dataHash)).size, imagePages.length, `${mode} ZIP should contain distinct rendered page data`)
        const galleryImageCount = await page.evaluate(async () => {
          const response = await fetch(document.querySelector('[data-testid="open-pages"]').href)
          const html = await response.text()
          return (html.match(/<img /g) || []).length
        })
        assert.equal(galleryImageCount, imagePages.length, `${mode} Open view should contain one image per rendered page`)
        if (mode === 'png') assert.deepEqual(result.bytes, [137, 80, 78, 71, 13, 10, 26, 10], 'PNG signature')
        if (mode === 'jpeg') assert.deepEqual(result.bytes.slice(0, 3), [255, 216, 255], 'JPEG signature')
        assert.match(result.mime || '', new RegExp(`^image/${mode}`), `${mode} MIME type`)
        if (mode === 'png') assert.equal(await page.$eval('#wasm-padding', (input) => input.value), '20', 'image padding should default to 20 pixels')

        if (mode === 'png') {
          await page.click('#wasm-padding')
          await page.keyboard.down('Control')
          await page.keyboard.press('A')
          await page.keyboard.up('Control')
          await page.keyboard.type('12')
          await page.click('[data-testid="convert"]')
          try {
            await page.waitForFunction((dimensions) => {
              const image = document.querySelector('.wasm-image-preview')
              return image?.naturalWidth === dimensions.width - 16 && image?.naturalHeight === dimensions.height - 16
            }, result)
          } catch (paddingWaitError) {
            const state = await page.$eval('.wasm-panel', (element) => element.innerText)
            const dimensions = await page.$eval('.wasm-image-preview', (image) => ({ width: image.naturalWidth, height: image.naturalHeight })).catch(() => null)
            const value = await page.$eval('#wasm-padding', (input) => input.value).catch(() => '')
            throw new Error(`padded PNG conversion timed out. input=${value} dimensions=${JSON.stringify(dimensions)} state=${state}`, { cause: paddingWaitError })
          }
          const padded = await page.$eval('.wasm-image-preview', (image) => ({ width: image.naturalWidth, height: image.naturalHeight }))
          assert.equal(padded.width, result.width - 16, 'changing padding from 20px to 12px should remove 8 pixels from both horizontal edges')
          assert.equal(padded.height, result.height - 16, 'changing padding from 20px to 12px should remove 8 pixels from both vertical edges')

          await page.click('#wasm-padding')
          await page.keyboard.down('Control')
          await page.keyboard.press('A')
          await page.keyboard.up('Control')
          await page.keyboard.type('0')
        }
      }

      assert.equal(await page.$('[data-testid="open-preview"]'), null, 'the duplicate full-preview trigger should be removed')
      const openLink = await page.$eval('.wasm-actions a[target="_blank"]', (element) => ({
        href: element.href,
        target: element.target,
      }))
      assert.equal(openLink.target, '_blank', `${mode} Open action should use a new tab`)
      assert.match(openLink.href, /^blob:/, `${mode} Open action should point to the generated Blob URL`)

      const viewportSelector = mode === 'pdf' ? '[data-testid="pdf-viewport"]' : '.wasm-preview'
      if (mode === 'pdf') {
        await page.waitForFunction(() => document.querySelector('.pdf-viewer-page canvas')?.width > 0)
      }
      const viewport = await page.$eval(viewportSelector, (element) => ({
        scrollWidth: element.scrollWidth,
        clientWidth: element.clientWidth,
        scrollHeight: element.scrollHeight,
        clientHeight: element.clientHeight,
      }))
      if (mode === 'pdf') {
        assert.ok(viewport.scrollHeight > viewport.clientHeight, `inline PDF preview should be scrollable by default: ${JSON.stringify(viewport)}`)
      }
      await page.$eval(viewportSelector, (element) => element.scrollIntoView({ block: 'center' }))
      const viewportBounds = await page.$eval(viewportSelector, (element) => {
        const bounds = element.getBoundingClientRect()
        return { left: bounds.left, top: bounds.top, width: bounds.width, height: bounds.height }
      })
      const scrollBeforeWheel = await page.$eval(viewportSelector, (element) => ({
        top: element.scrollTop,
        left: element.scrollLeft,
      }))
      await page.mouse.move(viewportBounds.left + viewportBounds.width / 2, viewportBounds.top + viewportBounds.height / 2)
      await page.mouse.wheel({ deltaY: 240 })
      await delay(100)
      const scrollAfterWheel = await page.$eval(viewportSelector, (element) => ({
        top: element.scrollTop,
        left: element.scrollLeft,
      }))
      if (mode === 'pdf') assert.ok(scrollAfterWheel.top > scrollBeforeWheel.top || scrollAfterWheel.left !== scrollBeforeWheel.left, `inline PDF preview should respond to mouse-wheel scrolling: ${JSON.stringify({ viewport, viewportBounds, scrollBeforeWheel, scrollAfterWheel })}`)

      if (mode !== 'pdf') {
        await page.click('.wasm-image-preview')
        assert.equal(await page.$('.wasm-lightbox'), null, `${mode} click should not open a second preview layer`)
        if (mode === 'png') {
          assert.match(await page.$eval('.wasm-preview', (element) => getComputedStyle(element).backgroundImage), /linear-gradient/, 'PNG preview should show a transparency checkerboard')
        }
        const corner = await page.evaluate(async () => {
          const image = document.querySelector('.wasm-image-preview')
          await image.decode()
          const canvas = document.createElement('canvas')
          canvas.width = image.naturalWidth
          canvas.height = image.naturalHeight
          canvas.getContext('2d').drawImage(image, 0, 0)
          return [...canvas.getContext('2d').getImageData(image.naturalWidth - 1, image.naturalHeight - 1, 1, 1).data]
        })
        if (mode === 'png') assert.equal(corner[3], 0, 'PNG preview should preserve transparent canvas pixels')
        if (mode === 'jpeg') assert.ok(corner.slice(0, 3).every((channel) => channel >= 250), `JPEG preview should composite transparent pixels onto white, got ${corner.slice(0, 3)}`)
      }

      const currentPreviewURL = await page.$eval(
        mode === 'pdf' ? '.pdf-viewer' : '.wasm-image-preview',
        (element, isPDF) => isPDF ? element.dataset.src : element.src,
        mode === 'pdf',
      )
      if (previousPreviewURL) {
        await assert.rejects(fetch(previousPreviewURL), 'previous preview URL should be revoked')
      }
      previousPreviewURL = currentPreviewURL
    }

    await page.screenshot({ path: '/tmp/gowkhtmltopdf-wasm.png', fullPage: true })

    await page.click('#wasm-html')
    await page.keyboard.down('Control')
    await page.keyboard.press('A')
    await page.keyboard.up('Control')
    await page.keyboard.press('Backspace')
    await page.click('[data-testid="convert"]')
    await page.waitForSelector('.wasm-error')
    assert.match(await page.$eval('.wasm-error', (element) => element.textContent), /Enter some HTML/)

    await page.setViewport({ width: 375, height: 800 })
    const overflow = await page.evaluate(() => ({
      scrollWidth: document.documentElement.scrollWidth,
      innerWidth: window.innerWidth,
      offenders: [...document.querySelectorAll('body *')]
        .filter((element) => element.getBoundingClientRect().right > window.innerWidth + 1)
        .slice(0, 5)
        .map((element) => ({ tag: element.tagName, className: element.className, right: element.getBoundingClientRect().right })),
    }))
    assert.equal(overflow.scrollWidth <= overflow.innerWidth, true, `mobile layout should not overflow horizontally: ${JSON.stringify(overflow)}`)
    console.log('WASM browser smoke passed for PDF, PNG, JPEG, validation, and mobile layout.')
  } finally {
    await browser.close()
  }
} finally {
  server.kill('SIGTERM')
}
