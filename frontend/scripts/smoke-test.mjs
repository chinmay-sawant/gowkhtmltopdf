#!/usr/bin/env node
import { readFileSync, existsSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import assert from 'node:assert/strict'

const __dirname = dirname(fileURLToPath(import.meta.url))
const root = join(__dirname, '..')
const docs = join(root, '..', 'docs')
const dist = join(root, 'dist')

console.log('🧪 Running frontend smoke tests...\n')

// 1. Check build files
console.log('1. Checking build outputs...')
assert.ok(existsSync(join(dist, 'index.html')), 'dist/index.html should exist')
assert.ok(existsSync(join(dist, 'favicon.svg')), 'dist/favicon.svg should exist')
assert.ok(existsSync(join(dist, 'data', 'issues.json')), 'dist/data/issues.json should exist')
assert.ok(existsSync(join(docs, 'index.html')), 'docs/index.html should exist')
assert.ok(existsSync(join(docs, '.nojekyll')), 'docs/.nojekyll should exist')
assert.ok(existsSync(join(docs, 'favicon.svg')), 'docs/favicon.svg should exist')
assert.ok(existsSync(join(docs, 'data', 'issues.json')), 'docs/data/issues.json should exist')
console.log('  ✓ Static build outputs and public assets exist.')

// 2. Validate issues data payload
console.log('\n2. Validating static issues payload...')
const issuesRaw = readFileSync(join(dist, 'data', 'issues.json'), 'utf8')
const issues = JSON.parse(issuesRaw)
assert.equal(Array.isArray(issues), true, 'issues should be an array')
assert.equal(issues.length, 1329, `issues should have 1329 records, found ${issues.length}`)

for (const it of issues) {
  assert.ok(typeof it.number === 'number', `issue #${it.number} missing valid number`)
  assert.ok(typeof it.title === 'string' && it.title.length > 0, `issue #${it.number} missing title`)
  assert.ok(['implemented', 'partial', 'not-implemented'].includes(it.status), `issue #${it.number} invalid status: ${it.status}`)
  assert.ok(['High', 'Medium', 'Low'].includes(it.severity), `issue #${it.number} invalid severity: ${it.severity}`)
  assert.ok(typeof it.category === 'string', `issue #${it.number} missing category`)
}
console.log(`  ✓ Successfully validated ${issues.length} issues data integrity.`)

// 3. Test slugify helper
console.log('\n3. Testing slugify helper...')
const { slugify } = await import('../src/components/blocks/slugify.js')
assert.equal(slugify('Hello World!'), 'hello-world')
assert.equal(slugify('`gowkhtmltopdf` --enable-local-file-access'), 'gowkhtmltopdf-enable-local-file-access')
assert.equal(slugify('Why JavaScript-heavy issues are not implemented'), 'why-javascript-heavy-issues-are-not-implemented')
console.log('  ✓ slugify utility passed contract assertions.')

// 4. Test showcase data categories
console.log('\n4. Validating showcase dataset...')
const { SHOWCASE, SHOWCASE_SPECIAL, SHOWCASE_CATEGORIES } = await import('../src/data/showcase.js')
assert.ok(SHOWCASE.length > 0, 'SHOWCASE should contain items')
for (const item of [...SHOWCASE, ...SHOWCASE_SPECIAL]) {
  assert.ok(item.category, `Showcase item ${item.name} missing category`)
  assert.ok(SHOWCASE_CATEGORIES.includes(item.category), `Showcase item ${item.name} has invalid category: ${item.category}`)
}
console.log(`  ✓ All ${SHOWCASE.length + SHOWCASE_SPECIAL.length} showcase items have valid curated categories.`)

// 5. Check HTML shell for title, theme-color, favicon, and OpenGraph tags
console.log('\n5. Checking HTML shell headers...')
const indexHtml = readFileSync(join(dist, 'index.html'), 'utf8')
assert.match(indexHtml, /<meta\s+name="theme-color"/i, 'HTML missing theme-color')
assert.match(indexHtml, /<link\s+rel="icon"/i, 'HTML missing favicon link')
assert.match(indexHtml, /<meta\s+property="og:title"/i, 'HTML missing og:title')
assert.match(indexHtml, /<meta\s+property="og:description"/i, 'HTML missing og:description')
assert.match(indexHtml, /<meta\s+property="og:image"/i, 'HTML missing og:image')
console.log('  ✓ HTML metadata and open graph tags verified.')

// 6. Test benchmarks data configuration
console.log('\n6. Validating benchmarks data configuration...')
const { CLI_ROWS, CHART_PAGES } = await import('../src/data/benchmarks.js')
assert.ok(CLI_ROWS.length > 0, 'CLI_ROWS should contain rows')
assert.ok(CHART_PAGES.length > 0, 'CHART_PAGES should contain sample pages')
for (const p of [2, 10, 100, 500]) {
  assert.ok(CLI_ROWS.some((r) => r.pages === p), `CLI_ROWS should contain ${p} pages data`)
}
console.log('  ✓ Benchmarks dataset verified.')

console.log('\n🎉 All frontend smoke tests passed successfully!\n')
