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
const port = 4174
const origin = `http://127.0.0.1:${port}`
const url = `${origin}/gowkhtmltopdf/#/showcase`

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
    await page.setViewport({ width: 1280, height: 900 })
    await page.goto(url, { waitUntil: 'networkidle0' })
    await page.waitForSelector('.showcase-card')
    assert.equal(await page.$('.cardbox'), null, 'showcase should not open a cardbox before a user click')

    await page.click('.showcase-card')
    await page.waitForSelector('.cardbox')
    assert.equal(await page.$eval('.cardbox-page', (element) => getComputedStyle(element).overflow), 'auto', 'image preview should own its scroll region')

    await page.click('button[aria-label="Zoom In"]')
    assert.equal(await page.$eval('.zoom-indicator', (element) => element.textContent), '150%', 'image zoom button should increase the zoom')
    const zoomedImage = await page.$eval('.cardbox-page img', (element) => ({
      width: element.getBoundingClientRect().width,
      containerWidth: element.parentElement.clientWidth,
      scrollWidth: element.parentElement.scrollWidth,
      clientWidth: element.parentElement.clientWidth,
    }))
    assert.ok(zoomedImage.width > zoomedImage.containerWidth, `zoomed image should be wider than its viewport: ${JSON.stringify(zoomedImage)}`)
    assert.ok(zoomedImage.scrollWidth > zoomedImage.clientWidth, `zoomed image should make the viewport scrollable: ${JSON.stringify(zoomedImage)}`)

    await page.keyboard.down('Control')
    await page.keyboard.press('=')
    await page.keyboard.up('Control')
    assert.equal(await page.$eval('.zoom-indicator', (element) => element.textContent), '200%', 'Control plus should zoom the image in')
    await page.keyboard.down('Control')
    await page.keyboard.press('-')
    await page.keyboard.up('Control')
    assert.equal(await page.$eval('.zoom-indicator', (element) => element.textContent), '150%', 'Control minus should zoom the image out')

    const bounds = await page.$eval('.cardbox-page', (element) => {
      const rect = element.getBoundingClientRect()
      return { left: rect.left, top: rect.top, width: rect.width, height: rect.height }
    })
    const scrollBefore = await page.$eval('.cardbox-page', (element) => ({ top: element.scrollTop, left: element.scrollLeft }))
    await page.mouse.move(bounds.left + bounds.width / 2, bounds.top + bounds.height / 2)
    await page.mouse.wheel({ deltaY: 260 })
    await delay(100)
    const scrollAfter = await page.$eval('.cardbox-page', (element) => ({ top: element.scrollTop, left: element.scrollLeft }))
    assert.ok(scrollAfter.top > scrollBefore.top || scrollAfter.left > scrollBefore.left, `image preview should respond to wheel scrolling: ${JSON.stringify({ scrollBefore, scrollAfter })}`)

    console.log('Showcase interaction smoke passed for click-only opening, image zoom, Control shortcuts, and scrolling.')
  } finally {
    await browser.close()
  }
} finally {
  server.kill('SIGTERM')
}
