# gowkhtmltopdf product site (React)

The static product site for **gowkhtmltopdf**, a print-oriented HTML-to-PDF
engine written in Go: no browser process, no cgo in the native build, and a
clean-room [wkhtmltopdf](https://github.com/wkhtmltopdf/wkhtmltopdf) work-alike.
The site covers the CLI (`gowkhtmltopdf`, `gowkhtmltoimage`), the Go `Document`
/ `ImageDocument` library, the Python bindings, and the browser WASM adapter.

What the site ships:

- **Landing** (`/`): CLI and Go examples, the load -> parse -> style -> layout
  -> paginate -> paint pipeline, scope notes, and benchmark proof points.
- **Documentation** (`/documentation/<docId>`): eight pages (`getting-started`,
  `cli`, `library-api`, `architecture`, `compatibility`, `fonts`, `security`,
  `performance`) rendered from `src/data/content/page-*.json`, with a sidebar,
  reading time, scrollspy table of contents, previous/next links, and an
  "Edit this page on GitHub" link. `/about` uses the same block format and
  renders through `ContentPage.jsx`.
- **Showcase** (`/showcase`): 61 committed samples (58 golden outputs plus 3
  specials) shown as page images, with category filters, a per-card page
  stepper, and a zoomable modal viewer.
- **Benchmarks** (`/benchmarks`): gowkhtmltopdf versus wkhtmltopdf 0.12.6.1,
  WeasyPrint, and Puppeteer, with time, speedup, and RSS views plus dated
  history snapshots.
- **Live demo** (`/live-demo`): inline HTML/CSS converted in a Web Worker
  through the WASM adapter, with a pdfjs-dist PDF preview and PNG/JPEG output
  (single-file download, or a ZIP when the image output has multiple pages).
  The sample picker loads five curated templates from `public/wasm/samples/`
  and, when GitHub is reachable, the golden fixture corpus from the repository.
  The WASM adapter is in flight for 0.2.6
  (`plans/0.2.6/86-canonical-0.2.6-wasm.md`); `VERSION` in this tree is 0.2.5.
- **Issue dossier** (`/dossier`): all 1,329 open `wkhtmltopdf/wkhtmltopdf`
  issues classified 469 implemented / 298 partial / 562 not implemented, with
  cited evidence on every row. The 100 newest rows carry hand-written
  `key_detail` notes (98 of them); the other 1,229 keep raw label/comment
  counts there, and 15 rows list a workaround. The dossier is filterable by
  status, category, and severity, searchable, sortable, and paginated
  (10/25/50/100 per page).

Site-wide: a command palette on Cmd/Ctrl+K, a light/dark theme persisted in
`localStorage` (`gowk-theme`, default from `prefers-color-scheme`), a GitHub
star count, and `HashRouter` routing so the static GitHub Pages build deep-links
without server rewrites.

## Routes

| Route | Renders |
|-------|---------|
| `/` | Landing page |
| `/about` | Single content page (`ContentPage.jsx`) |
| `/documentation/<docId>` | `getting-started`, `cli`, `library-api`, `architecture`, `compatibility`, `fonts`, `security`, `performance` |
| `/documentation` | Redirect to `/documentation/getting-started` |
| `/dossier` | Issue dossier |
| `/showcase` | Golden-fixture sample gallery |
| `/benchmarks` | Benchmark comparisons |
| `/live-demo` | Browser WASM demo |
| `/wasm` | Redirect to `/live-demo` |
| `/getting-started`, `/cli`, `/library-api`, `/architecture`, `/compatibility`, `/fonts`, `/security`, `/performance` | Redirect to `/documentation/<id>` |
| anything else | 404 page |

`HashRouter` serves all of these as `#/dossier`-style URLs.

Built to deploy as a static site on **GitHub Pages** from the repo's `docs/`
folder (base path `/gowkhtmltopdf/`).

## Stack

- [Vite](https://vite.dev) + [React](https://react.dev) + [React Router](https://reactrouter.com) (`HashRouter`)
- [pdfjs-dist](https://mozilla.github.io/pdf.js/) for PDF previews and page rasterization in the live demo
- Plain CSS in `src/styles/` (tokens, reset, one file per surface): editorial warm-monochrome style with a `[data-theme=dark]` variable block, no UI framework
- A Web Worker (`src/workers/wasmWorker.js`) runs the Go WASM runtime; `src/lib/wasmClient.js` posts requests, and `src/lib/renderImagePages.js` turns PDF pages into PNG/JPEG blobs and ZIPs

## Folder structure

```
frontend/
|-- index.html                 # Vite shell (title, meta, OpenGraph, fonts)
|-- vite.config.js             # base path + build config
|-- package.json
|-- scripts/
|   |-- copy-to-docs.mjs       # build ../docs: rm -rf, copy dist/, .nojekyll, go.mod stub
|   |-- lint-data.mjs          # validates content JSON, showcase, benchmarks, constants
|   |-- smoke-test.mjs         # npm test: build outputs + data integrity
|   |-- interaction-test.mjs   # npm run test:interactions: Puppeteer showcase run
|   `-- live-demo-browser-test.mjs  # npm run test:live-demo: Puppeteer WASM run
|-- public/
|   |-- data/issues.json       # 1,329-row dossier payload (1.1 MB, fetched at runtime)
|   |-- wasm/                  # gowkhtmltopdf.wasm, wasm_exec.js, sample.html, samples/ (make wasm)
|   |-- favicon.svg
|   `-- og-preview.png
`-- src/
    |-- main.jsx               # React bootstrap
    |-- App.jsx                # HashRouter route table, lazy pages, legacy redirects
    |-- components/
    |   |-- SiteNav.jsx / Footer.jsx
    |   |-- CommandPalette.jsx # Cmd/Ctrl+K search over docs, CLI flags, samples, actions
    |   |-- Cardbox.jsx        # showcase modal: page stepper, zoom, fullscreen
    |   |-- PdfViewer.jsx      # pdfjs-dist canvas viewer
    |   |-- GitHubStars.jsx    # cached repo star count
    |   |-- FilterChips.jsx / StatsSidebar.jsx / IssueCard.jsx / Pagination.jsx
    |   |-- RichText.jsx / highlightText.jsx / PageTitle.jsx
    |   `-- blocks/            # ContentBlocks dispatch + hero/stats/cards/prose/code/table/bullets/callout/toc
    |-- hooks/                 # useIssues, useTheme, useDebounce
    |-- lib/                   # wasmClient.js, renderImagePages.js
    |-- workers/wasmWorker.js  # Go WASM runtime + conversion bridge
    |-- pages/
    |   |-- LandingPage.jsx
    |   |-- DocumentationPage.jsx  # /documentation/:docId (8 docs)
    |   |-- ContentPage.jsx        # /about
    |   |-- DossierPage.jsx
    |   |-- ShowcasePage.jsx
    |   |-- BenchmarksPage.jsx
    |   |-- LiveDemoPage.jsx
    |   `-- NotFoundPage.jsx
    |-- data/
    |   |-- issues.js          # fetch + sort/count helpers for issues.json
    |   |-- benchmarks.js      # dated benchmark snapshots (current + history)
    |   |-- constants.js       # status / severity / category metadata
    |   |-- showcase.js        # sample catalog + GitHub PDF/template links
    |   |-- types.js           # JSDoc typedefs
    |   `-- content/           # page-*.json content (11 files)
    |-- styles/                # tokens + reset + one CSS file per surface
    `-- assets/showcase/       # committed PNG pages and thumbs/*.webp
```

The dossier dataset lives in `frontend/public/data/issues.json` (fetched at
runtime from `${BASE_URL}data/issues.json`). `copy-to-docs.mjs` ships the built
site and the dataset to `../docs` for GitHub Pages.

## Content block schema

Each `src/data/content/page-*.json` is `{ id, nav, content: [...] }`.
`scripts/lint-data.mjs` (`npm run lint`) enforces:

- `id` matches the filename (`page-cli.json` -> `cli`).
- `nav` is a non-empty string and `content` is a non-empty array.
- Block shapes:

| `type` | Required | Optional |
|--------|----------|----------|
| `hero` | `title` | `lede` |
| `stats` | `items` (non-empty), each with `value` and `label` | per-item `percent` |
| `cards` | `items` (non-empty), each with `title` and `body` | `heading` |
| `prose` | `sections` (non-empty); each section needs `body` or non-empty `bullets` | section `heading` |
| `code` | `code` (string) | `language`, `heading` |
| `table` | `headers` and `rows` (non-empty); row length must match `headers` | `heading` |
| `bullets` | `items` (non-empty strings) | `heading` |
| `callout` | `title` or `body` | `variant`: `info`, `warn`, or `tip` |
| `toc` | none | `title`, `items` (array; strings or `{ label, href }`) |

The same script validates `src/data/showcase.js` (categories, positive page
counts), `src/data/benchmarks.js` (`CLI_ROWS` non-empty, `CHART_PAGES` present
in `CLI_ROWS`), and `src/data/constants.js` (metadata for every status,
severity, and category). Adding a doc page means adding `page-<id>.json` and an
entry in the `DOCS` list in `DocumentationPage.jsx`.

## Commands

```sh
npm install
npm run dev                  # dev server (http://localhost:5173/gowkhtmltopdf/)
npm run lint                 # ESLint (src, scripts, Vite config) + scripts/lint-data.mjs
npm run build                # vite build -> dist/, then copy-to-docs.mjs -> ../docs
npm run preview              # serve the production build
npm test                     # scripts/smoke-test.mjs: dist/ + ../docs outputs and data checks
npm run test:interactions    # scripts/interaction-test.mjs: Puppeteer showcase interactions
npm run test:live-demo       # scripts/live-demo-browser-test.mjs: Puppeteer WASM conversion run
```

`npm test` and the two Puppeteer tests read the production build, so run
`npm run build` first. The Puppeteer tests need Chrome at
`PUPPETEER_EXECUTABLE_PATH` (default `/usr/bin/google-chrome`) and the
`scripts/puppeteer` dependencies (`npm ci --prefix scripts/puppeteer`). The
live demo tests need `public/wasm/` built from the repo root with `make wasm`,
which builds `bindings/wasm` and copies `wasm_exec.js`, `sample.html`, and
`manifest.json` from the Go toolchain and `testdata/wasm/`.

## Deploying to GitHub Pages

`npm run build` runs `vite build` and then `scripts/copy-to-docs.mjs`, which
removes `../docs`, copies `dist/` into it, writes `.nojekyll`, and preserves the
`docs/go.mod` module stub so the built site stays out of the parent module zip.
`docs/` is committed; CI (`.github/workflows/ci.yml`) runs the same build and
fails if `docs/` or `frontend/dist/` comes out dirty.

1. Run `npm run build`.
2. On GitHub, enable Pages with Source: Deploy from a branch, branch `master`,
   folder `/docs`.
3. The site is served at `https://<user>.github.io/gowkhtmltopdf/`. Override
   the base path with `VITE_BASE_PATH=/whatever/`; the default is
   `/gowkhtmltopdf/` (`vite.config.js`).

## Updating the data

- **Dossier:** replace the array in `public/data/issues.json` and rebuild.
  Rows have the shape `number`, `title`, `summary`, `category`, `severity`,
  `status`, `evidence`, `url`, `labels`, `author`, `created_at`, `updated_at`,
  and `comments`. The 100 newest rows carry hand-written `key_detail` notes
  (98 of them); the other 1,229 keep raw label/comment counts there. Fifteen
  rows carry a `workaround`. `scripts/smoke-test.mjs` asserts 1,329 records, so
  update that count when the dataset changes.
- **Product content:** edit the matching `src/data/content/page-<id>.json`, run
  `npm run lint`, then rebuild. Keep content ASCII-only (no em dashes, no
  emojis); the linter does not check this.
- **Showcase samples:** add the output PDF to `output/`, regenerate the page
  PNGs and `thumbs/*.webp` in `src/assets/showcase/` with `make samples` and
  `make screenshots` from the repo root, then add an entry to
  `src/data/showcase.js`.
- **Benchmarks:** edit `src/data/benchmarks.js`; the file labels the current
  capture and keeps older samples dated.
