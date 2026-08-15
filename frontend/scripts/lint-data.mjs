#!/usr/bin/env node
import { readdirSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
const dataDir = join(here, '..', 'src', 'data')
const contentDir = join(dataDir, 'content')

const BLOCK_TYPES = new Set([
  'hero',
  'stats',
  'cards',
  'prose',
  'code',
  'table',
  'bullets',
  'callout',
  'toc',
])
const CALLOUT_VARIANTS = new Set(['info', 'warn', 'tip'])

const errors = []

function fail(msg) {
  errors.push(msg)
}

function isNonEmptyString(value) {
  return typeof value === 'string' && value.length > 0
}

function checkBlock(pageId, index, block) {
  const loc = `${pageId} content[${index}]`

  if (!block || typeof block !== 'object' || Array.isArray(block)) {
    fail(`${loc}: expected a block object`)
    return
  }

  if (!BLOCK_TYPES.has(block.type)) {
    fail(`${loc}: unknown type ${JSON.stringify(block.type)}`)
    return
  }

  switch (block.type) {
    case 'hero':
      if (!isNonEmptyString(block.title)) fail(`${loc}: hero.title is required`)
      break
    case 'stats':
      if (!Array.isArray(block.items) || block.items.length === 0) {
        fail(`${loc}: stats.items must be a non-empty array`)
        break
      }
      block.items.forEach((item, i) => {
        if (!item || typeof item !== 'object') {
          fail(`${loc}.items[${i}]: expected object`)
          return
        }
        if (!isNonEmptyString(item.value)) fail(`${loc}.items[${i}].value is required`)
        if (!isNonEmptyString(item.label)) fail(`${loc}.items[${i}].label is required`)
      })
      break
    case 'cards':
      if (!Array.isArray(block.items) || block.items.length === 0) {
        fail(`${loc}: cards.items must be a non-empty array`)
        break
      }
      block.items.forEach((item, i) => {
        if (!item || typeof item !== 'object') {
          fail(`${loc}.items[${i}]: expected object`)
          return
        }
        if (!isNonEmptyString(item.title)) fail(`${loc}.items[${i}].title is required`)
        if (!isNonEmptyString(item.body)) fail(`${loc}.items[${i}].body is required`)
      })
      break
    case 'prose':
      if (!Array.isArray(block.sections) || block.sections.length === 0) {
        fail(`${loc}: prose.sections must be a non-empty array`)
        break
      }
      block.sections.forEach((section, i) => {
        if (!section || typeof section !== 'object') {
          fail(`${loc}.sections[${i}]: expected object`)
          return
        }
        const hasBody = isNonEmptyString(section.body)
        const hasBullets = Array.isArray(section.bullets) && section.bullets.length > 0
        if (!hasBody && !hasBullets) {
          fail(`${loc}.sections[${i}]: needs body or bullets`)
        }
        if (hasBullets) {
          section.bullets.forEach((bullet, j) => {
            if (!isNonEmptyString(bullet)) {
              fail(`${loc}.sections[${i}].bullets[${j}]: expected string`)
            }
          })
        }
      })
      break
    case 'code':
      if (typeof block.code !== 'string') fail(`${loc}: code.code is required`)
      break
    case 'table':
      if (!Array.isArray(block.headers) || block.headers.length === 0) {
        fail(`${loc}: table.headers must be a non-empty array`)
        break
      }
      if (!Array.isArray(block.rows) || block.rows.length === 0) {
        fail(`${loc}: table.rows must be a non-empty array`)
        break
      }
      block.headers.forEach((header, i) => {
        if (!isNonEmptyString(header)) fail(`${loc}.headers[${i}]: expected string`)
      })
      block.rows.forEach((row, i) => {
        if (!Array.isArray(row)) {
          fail(`${loc}.rows[${i}]: expected array`)
          return
        }
        if (row.length !== block.headers.length) {
          fail(`${loc}.rows[${i}]: expected ${block.headers.length} cells, got ${row.length}`)
        }
        row.forEach((cell, j) => {
          if (typeof cell !== 'string') fail(`${loc}.rows[${i}][${j}]: expected string`)
        })
      })
      break
    case 'bullets':
      if (!Array.isArray(block.items) || block.items.length === 0) {
        fail(`${loc}: bullets.items must be a non-empty array`)
        break
      }
      block.items.forEach((item, i) => {
        if (!isNonEmptyString(item)) fail(`${loc}.items[${i}]: expected string`)
      })
      break
    case 'callout':
      if (block.variant && !CALLOUT_VARIANTS.has(block.variant)) {
        fail(`${loc}: unknown callout variant ${JSON.stringify(block.variant)}`)
      }
      if (!isNonEmptyString(block.title) && !isNonEmptyString(block.body)) {
        fail(`${loc}: callout needs title or body`)
      }
      break
    case 'toc':
      if (block.items != null && !Array.isArray(block.items)) {
        fail(`${loc}: toc.items must be an array when set`)
      }
      break
    default:
      break
  }
}

function checkContentPage(fileName, page) {
  const expectedId = fileName.replace(/^page-/, '').replace(/\.json$/, '')

  if (!page || typeof page !== 'object') {
    fail(`${fileName}: expected a JSON object`)
    return
  }

  if (page.id !== expectedId) {
    fail(`${fileName}: id ${JSON.stringify(page.id)} must match ${JSON.stringify(expectedId)}`)
  }

  if (!isNonEmptyString(page.nav)) fail(`${fileName}: nav is required`)

  if (!Array.isArray(page.content) || page.content.length === 0) {
    fail(`${fileName}: content must be a non-empty array`)
    return
  }

  page.content.forEach((block, i) => checkBlock(page.id || fileName, i, block))
}

function loadJSON(path) {
  try {
    return JSON.parse(readFileSync(path, 'utf8'))
  } catch (err) {
    fail(`${path}: ${err.message}`)
    return null
  }
}

const contentFiles = readdirSync(contentDir)
  .filter((name) => name.startsWith('page-') && name.endsWith('.json'))
  .sort()

if (contentFiles.length === 0) {
  fail('src/data/content: no page-*.json files found')
}

for (const name of contentFiles) {
  const page = loadJSON(join(contentDir, name))
  if (page) checkContentPage(name, page)
}

const { SHOWCASE, SHOWCASE_SPECIAL, SHOWCASE_CATEGORIES } = await import(
  pathToFileURL(join(dataDir, 'showcase.js')).href
)

if (!Array.isArray(SHOWCASE) || SHOWCASE.length === 0) {
  fail('showcase.js: SHOWCASE must be a non-empty array')
}

if (!Array.isArray(SHOWCASE_SPECIAL)) {
  fail('showcase.js: SHOWCASE_SPECIAL must be an array')
}

if (!Array.isArray(SHOWCASE_CATEGORIES) || SHOWCASE_CATEGORIES.length === 0) {
  fail('showcase.js: SHOWCASE_CATEGORIES must be a non-empty array')
}

for (const item of [...(SHOWCASE || []), ...(SHOWCASE_SPECIAL || [])]) {
  if (!item || typeof item !== 'object') {
    fail('showcase.js: item must be an object')
    continue
  }
  if (!isNonEmptyString(item.name)) fail('showcase.js: item.name is required')
  if (!isNonEmptyString(item.file)) fail(`showcase.js: ${item.name} missing file`)
  if (!Number.isInteger(item.pages) || item.pages < 1) {
    fail(`showcase.js: ${item.name} pages must be a positive integer`)
  }
  if (!isNonEmptyString(item.title)) fail(`showcase.js: ${item.name} missing title`)
  if (!isNonEmptyString(item.desc)) fail(`showcase.js: ${item.name} missing desc`)
  if (!SHOWCASE_CATEGORIES.includes(item.category)) {
    fail(`showcase.js: ${item.name} has invalid category ${JSON.stringify(item.category)}`)
  }
}

const { CLI_ROWS, CHART_PAGES } = await import(pathToFileURL(join(dataDir, 'benchmarks.js')).href)

if (!Array.isArray(CLI_ROWS) || CLI_ROWS.length === 0) {
  fail('benchmarks.js: CLI_ROWS must be a non-empty array')
}

if (!Array.isArray(CHART_PAGES) || CHART_PAGES.length === 0) {
  fail('benchmarks.js: CHART_PAGES must be a non-empty array')
}

for (const row of CLI_ROWS || []) {
  if (!Number.isFinite(row.pages) || row.pages <= 0) {
    fail(`benchmarks.js: CLI_ROWS entry missing pages (${JSON.stringify(row)})`)
  }
}

for (const pages of CHART_PAGES || []) {
  if (!(CLI_ROWS || []).some((row) => row.pages === pages)) {
    fail(`benchmarks.js: CHART_PAGES value ${pages} is missing from CLI_ROWS`)
  }
}

const { STATUS_ORDER, STATUS_META, SEVERITY_ORDER, SEVERITY_META, CATEGORY_ORDER, CATEGORY_COLOR } =
  await import(pathToFileURL(join(dataDir, 'constants.js')).href)

for (const status of STATUS_ORDER || []) {
  if (!STATUS_META[status]) fail(`constants.js: STATUS_META missing ${status}`)
}

for (const severity of SEVERITY_ORDER || []) {
  if (!SEVERITY_META[severity]) fail(`constants.js: SEVERITY_META missing ${severity}`)
}

for (const category of CATEGORY_ORDER || []) {
  if (!CATEGORY_COLOR[category]) fail(`constants.js: CATEGORY_COLOR missing ${category}`)
}

if (errors.length > 0) {
  console.error(`src/data lint failed (${errors.length}):`)
  for (const err of errors) console.error(`  ${err}`)
  process.exit(1)
}

console.log(
  `src/data lint clean (${contentFiles.length} content pages, ${SHOWCASE.length + SHOWCASE_SPECIAL.length} showcase items)`,
)
