#!/usr/bin/env node
import { spawn } from 'node:child_process'
import { createRequire } from 'node:module'
import { setTimeout as delay } from 'node:timers/promises'
import { join } from 'node:path'
import assert from 'node:assert/strict'

const require = createRequire(import.meta.url)
const puppeteer = require(join('..', '..', 'scripts', 'puppeteer', 'node_modules', 'puppeteer-core'))
const frontendDir = join(import.meta.dirname, '..')
const port = 4173
const origin = `http://127.0.0.1:${port}`
const url = `${origin}/gowkhtmltopdf/#/wasm`

const server = spawn('npm', ['run', 'preview', '--', '--host', '127.0.0.1', '--port', String(port)], {
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
      const embeddedRequest = requestURL.protocol === 'data:' || requestURL.protocol === 'blob:' || requestURL.protocol === 'about:'
      if (localRequest || embeddedRequest) request.continue()
      else request.abort()
    })
    await page.goto(url, { waitUntil: 'networkidle0' })
    await page.waitForSelector('#wasm-html')
    await page.emulateMediaFeatures([{ name: 'prefers-reduced-motion', value: 'reduce' }])
    assert.equal(await page.$eval('.wasm-output-option', (element) => getComputedStyle(element).transitionDuration), '0s', 'reduced motion should disable transitions')
    await page.click('.theme-toggle')
    await page.waitForFunction(() => document.documentElement.dataset.theme === 'dark')
    await page.click('.theme-toggle')
    await page.waitForFunction(() => document.documentElement.dataset.theme === 'light')

    await page.click('[data-testid="load-sample"]')
    await page.waitForFunction(() => document.querySelector('#wasm-html')?.value.includes('Browser WASM conversion'))
    await page.click('#wasm-html')
    assert.equal(await page.$eval('#wasm-html', (element) => document.activeElement === element), true, 'HTML editor should receive keyboard focus')
    const manifest = await page.evaluate(async () => {
      const response = await fetch(new URL('wasm/manifest.json', document.baseURI))
      return response.json()
    })

    await page.$eval('#wasm-html', (element) => {
      const setter = Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, 'value').set
      setter.call(element, '<!doctype html><html><body>' + '<p>cancel test</p>'.repeat(50000) + '</body></html>')
      element.dispatchEvent(new Event('input', { bubbles: true }))
    })
    await page.click('[data-testid="convert"]')
    await page.waitForSelector('[data-testid="cancel"]', { timeout: 10000 })
    await page.click('[data-testid="cancel"]')
    await page.waitForFunction(() => document.querySelector('.wasm-status')?.textContent.includes('Conversion cancelled'))
    await page.click('[data-testid="load-sample"]')

    let previousPreviewURL = null
    for (const mode of ['pdf', 'png', 'jpeg']) {
      await page.click(`input[name="wasm-output"][value="${mode}"]`)
      await page.click('[data-testid="convert"]')
      await page.waitForFunction(() => document.querySelector('.wasm-status')?.textContent.includes('Conversion complete'), { timeout: 30000 })

      const result = await page.evaluate(async (currentMode) => {
        const target = currentMode === 'pdf' ? document.querySelector('.wasm-pdf-preview') : document.querySelector('.wasm-image-preview')
        const response = await fetch(target.src)
        const bytes = new Uint8Array(await response.arrayBuffer())
        return {
          bytes: Array.from(bytes.slice(0, 8)),
          tail: Array.from(bytes.slice(-5)),
          text: new TextDecoder().decode(bytes),
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
        for (const needle of manifest.outputs.pdf.textNeedles) {
          assert.match(result.text, new RegExp(needle), `PDF text needle: ${needle}`)
        }
      } else {
        assert.ok(result.width > 0 && result.height > 0, `${mode} preview should have dimensions`)
        if (mode === 'png') assert.deepEqual(result.bytes, [137, 80, 78, 71, 13, 10, 26, 10], 'PNG signature')
        if (mode === 'jpeg') assert.deepEqual(result.bytes.slice(0, 3), [255, 216, 255], 'JPEG signature')
        assert.match(result.mime || '', new RegExp(`^image/${mode}`), `${mode} MIME type`)
      }

      const currentPreviewURL = await page.$eval(
        mode === 'pdf' ? '.wasm-pdf-preview' : '.wasm-image-preview',
        (element) => element.src,
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
