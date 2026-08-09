import { cpSync, rmSync, readdirSync, writeFileSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
const frontend = join(here, '..')
const dist = join(frontend, 'dist')
const docs = join(frontend, '..', 'docs')

rmSync(docs, { recursive: true, force: true })
cpSync(dist, docs, { recursive: true })

// Tell GitHub Pages to skip Jekyll processing so filenames starting with _
// (e.g. assets/_noop) are published unchanged.
writeFileSync(join(docs, '.nojekyll'), '')

const files = readdirSync(docs)
console.log(`copied build output → ${docs} (${files.join(', ')})`)
