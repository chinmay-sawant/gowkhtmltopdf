import { cpSync, existsSync, readFileSync, rmSync, readdirSync, writeFileSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
const frontend = join(here, '..')
const dist = join(frontend, 'dist')
const docs = join(frontend, '..', 'docs')
const docsGoMod = join(docs, 'go.mod')
const stubGoMod = `// Separate module so docs/ (GitHub Pages build) is not packed into the
// parent module zip (go install of the CLIs does not need the site).
module github.com/chinmay-sawant/gowkhtmltopdf/docs

go 1.26
`

const keepGoMod = existsSync(docsGoMod) ? readFileSync(docsGoMod, 'utf8') : stubGoMod

rmSync(docs, { recursive: true, force: true, maxRetries: 5, retryDelay: 50 })
cpSync(dist, docs, { recursive: true })

// Tell GitHub Pages to skip Jekyll processing so filenames starting with _
// (e.g. assets/_noop) are published unchanged.
writeFileSync(join(docs, '.nojekyll'), '')
writeFileSync(docsGoMod, keepGoMod)

const files = readdirSync(docs)
console.log(`copied build output → ${docs} (${files.join(', ')})`)
