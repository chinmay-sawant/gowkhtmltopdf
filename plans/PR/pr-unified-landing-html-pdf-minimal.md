## Summary

Unifies the landing page around one story: HTML to PDF. Rewrites `LandingPage` and its styles to a minimal 5-section layout that states the sole purpose above the fold, shows a single CLI/Go example, and removes the heavy interactive sandbox and benchmark grids.

## Motivation / context

- Direct feedback on the current landing (`frontend/src/pages/LandingPage.jsx:1`): purpose was not immediately clear, page felt fragmented across proof, benchmark, capability and showcase sections.
- Product contract is single-purpose HTML to PDF (`README.md:6`, `documentation/overview.md`) with two surfaces: static binary and Go `Document` library. The landing should make that obvious within 5 seconds.
- Minimalism was requested: warm monochrome, bento cards, no gradients/heavy shadows, typographic contrast. This aligns with the existing design system (`frontend/src/styles/tokens.css:1`) and the dedicated `minimalist-ui` guidance.
- Plans: `plans/0.2.6/README.md` (branch `feature/06-high-fun-extended-CSS-support-Fixing-issues` is the 0.2.6 CSS coverage track). Landing refresh is a frontend-only slice on top of the 38-commits-ahead engine work.
- Issues: no tracked ticket for landing copy; treated as UX improvement driven by user feedback. See **Related issues**.

## Changes

### Area 1 - Frontend landing page (`frontend/src/pages/LandingPage.jsx:1`)

- Replaced the 396-line `LandingPage` with a 214-line minimal version (`git show HEAD --stat`: `159+ 341-`).
- Removed: `useMemo`, `CLI_ROWS`/`HEADLINE`/`LIBRARY_HEADLINE` imports, `HOME_BENCH_PAGES`, `CAPABILITIES[3]`, `OUTPUTS[4]`, `OUTPUT_FLAVOURS[5]`, 5 interactive states (`pageSize`, `margins`, `orientation`, `security`, `outputFlavour`) and two `useMemo` builders for `commandLines`/`rawCommand`.
- Added: static `CLI_CODE` (`frontend/src/pages/LandingPage.jsx:5`) and `GO_CODE` (`frontend/src/pages/LandingPage.jsx:9`), single `tab` state (`cli|go`) at `frontend/src/pages/LandingPage.jsx:20`, `handleCopy` that copies the canonical single command (`gowkhtmltopdf input.html output.pdf`) or the Go snippet (`frontend/src/pages/LandingPage.jsx:24`).
- Root wrapper changed `landing-page` -> `landing-minimal` (`frontend/src/pages/LandingPage.jsx:34`). Kept `PageTitle` at `frontend/src/pages/LandingPage.jsx:35`.
- New 6-section structure sharing one narrative:
  1. Hero minimal (`frontend/src/pages/LandingPage.jsx:37`) - kicker `HTML to PDF · Pure Go · No browser, no cgo` (`frontend/src/pages/LandingPage.jsx:39`), headline `Your HTML, as a print-ready PDF.` (`frontend/src/pages/LandingPage.jsx:40`), lede `One purpose: turn HTML you author into paginated PDFs...` (`frontend/src/pages/LandingPage.jsx:45`), CTAs `Get started -> /getting-started` (`frontend/src/pages/LandingPage.jsx:50`) + `View samples /showcase` (`frontend/src/pages/LandingPage.jsx:53`), micro `Drop-in binary gowkhtmltopdf or native Go library Document. Static build with CGO_ENABLED=0.` (`frontend/src/pages/LandingPage.jsx:55`) at `max-width:72ch` (`frontend/src/styles/landing.css:63`), and code card with tablist CLI/Go (`frontend/src/pages/LandingPage.jsx:63`) + copy (`frontend/src/pages/LandingPage.jsx:83`) + `pre code {code}` (`frontend/src/pages/LandingPage.jsx:92`).
  2. Flow - One pipeline. HTML in, PDF out. (`frontend/src/pages/LandingPage.jsx:102`). `ol.flow-grid` with 3 cards: `01 HTML you control`, `02 Engine Load -> parse -> style -> layout -> paginate -> paint` (`frontend/src/pages/LandingPage.jsx:114`), `03 Print-ready file PDF 1.4 by default...` (`frontend/src/pages/LandingPage.jsx:120`) plus `flow-arrow` `input.html -> engine -> output.pdf` (`frontend/src/pages/LandingPage.jsx:127`).
  3. Fit - Built for documents, not for browsers. (`frontend/src/pages/LandingPage.jsx:132`). `fit-grid` 2x2 bento (`frontend/src/pages/LandingPage.jsx:137`): good for Templates and tables / Structured PDFs, not for JavaScript / Any website as PDF, with links to `/documentation/compatibility` and `/documentation/security` (`frontend/src/pages/LandingPage.jsx:159`).
  4. Proof strip minimal (`frontend/src/pages/LandingPage.jsx:165`): 4 items `HTML -> PDF` / `Static binary` / `Two surfaces` / `Measured 2-page invoice: 17 ms CLI vs 259 ms wkhtmltopdf 0.12.6.1` linking to `/benchmarks` (`frontend/src/pages/LandingPage.jsx:178`).
  5. Samples minimal (`frontend/src/pages/LandingPage.jsx:184`): `See the output before you ship.` (`frontend/src/pages/LandingPage.jsx:186`) + decorative `samples-stack` (`frontend/src/pages/LandingPage.jsx:190`).
  6. Close band (`frontend/src/pages/LandingPage.jsx:197`): `Start with one command.` (`frontend/src/pages/LandingPage.jsx:198`) + `close-code` `gowkhtmltopdf input.html output.pdf` (`frontend/src/pages/LandingPage.jsx:201`) + links `First conversion / Go library / Benchmarks` (`frontend/src/pages/LandingPage.jsx:204`).

### Area 2 - Styles (`frontend/src/styles/landing.css:1`)

- Rewrote 930 lines -> 680 lines (`405+ 655-`, net -250).
- Root `.landing-minimal` (`frontend/src/styles/landing.css:2`) with `--landing-accent: var(--accent)` and flex column; replaced heavy sandbox styles (`.terminal-card`, `.terminal-sandbox-toolbar`, `.sandbox-chip`, etc.) with flat `.landing-code-card` (`frontend/src/styles/landing.css:80`, `border:1px solid var(--code-border); border-radius:12px; background:var(--code-bg)`), `.landing-code-topbar`/`dots`/`tabs`/`copy` (`frontend/src/styles/landing.css:88`), `pre min-height:148px` (`frontend/src/styles/landing.css:155`), and footer/hint (`frontend/src/styles/landing.css:166`).
- Hero: `.landing-hero-minimal` (`frontend/src/styles/landing.css:10`) `grid 1.05fr/340px`, `gap clamp(32px,5vw,72px)`, `padding:56px 0 48px`, `border-bottom:1px solid var(--line)`; added `.landing-kicker` mono 11px uppercase (`frontend/src/styles/landing.css:19`), `h1 em` serif italic accent (`frontend/src/styles/landing.css:41`), `.landing-lede 52ch` (`frontend/src/styles/landing.css:48`), `.landing-micro 72ch` (`frontend/src/styles/landing.css:63`, widened from 52ch per follow-up).
- New systems: `.landing-flow`/`landing-section-head`/`flow-grid` 3-col / `flow-card`/`flow-index`/`flow-arrow` dashed (`frontend/src/styles/landing.css:188`), `.landing-fit`/`fit-grid` 2-col / `fit-card` / `fit-kicker` pills (`fit-good #edf3ec/#346538`, `fit-limit #fbf3db/#7a5a05`, dark variants at `frontend/src/styles/landing.css:357`) (`frontend/src/styles/landing.css:301`), `.landing-proof-minimal` 4-col grid (`frontend/src/styles/landing.css:376`), `.landing-samples-minimal` `1fr 0.9fr` + `.sample-paper` 176x232 rotated (`frontend/src/styles/landing.css:412`), `.landing-close` centered band (`frontend/src/styles/landing.css:513`).
- Reused tokens from `frontend/src/styles/tokens.css:1` (warm monochrome `--bg #f8f9f8`, `--line #dce3de`, `--accent #176b59`, `--code-bg #111917`) so dark mode (`html[data-theme='dark']` at `frontend/src/styles/tokens.css:32`) stays consistent. No gradients, no heavy shadows, borders `1px solid #EAEAEA` equivalent, radii `6px`/`12px`/`16px`.
- Responsiveness: `@media (max-width:900px)` hero->1fr, `flow-grid`->1fr, `proof-minimal`->2 cols, `samples-minimal`->1fr (`frontend/src/styles/landing.css:624`); `@media (max-width:640px)` hero padding 40/32, `pre` 12px, bento 1fr, `samples-stack scale(0.92)`, buttons full width (`frontend/src/styles/landing.css:647`).
- Routing unchanged: `frontend/src/App.jsx:7` static import `LandingPage` and `frontend/src/App.jsx:68` `<Route path="/" element={<LandingPage />} />` inside `WrapLayout` (still `HashRouter`, `SiteNav`, `Footer`). `frontend/src/App.jsx:10` lazy pages for docs/dossier/showcase/benchmarks remain code-split.

### Area 3 - Built docs (`docs/`)

- Rebuilt via `frontend/package.json:9` `vite build && node scripts/copy-to-docs.mjs` (`frontend/vite.config.js:11` `outDir:dist`). Copied to `docs/` (`frontend/scripts/copy-to-docs.mjs:19`).
- Hashed chunks rotated (`git show HEAD -- docs/`): `BenchmarksPage-MKqi2Jw5.js -> BErojGA3.js`, `DocumentationPage-DS7yggaw.js -> CDqD5FN9.js`, `DossierPage-CPIbGTwg.js -> B8nQiJtH.js`, `ShowcasePage-QMumJZgp.js -> Di5JpTqe.js`, `index-41iEqrQc.js (136 lines) -> index-BUuSB4SA.js (145 lines)`, `index-L07kejqp.css -> index-25J-iRC3.css`, plus `docs/index.html:22` pointer updates (`/assets/index-BUuSB4SA.js`, `/assets/index-25J-iRC3.css`). Verified `make claim-scan` scans `frontend/src/data/content` + `internal/cli/help.go` and now passes.

Note on branch scope: `git rev-list --count master..HEAD` = 38 commits, `git diff master...HEAD --stat` = 264 files (`internal/css`, `internal/layout`, `internal/convert`, `plans/0.2.6/`, `testdata/golden/fixture-57..62`, `output/*.pdf`, plus this landing slice). The PR diff therefore includes the 0.2.6 CSS coverage engine work (356 Implemented props, fixtures 60-62, Type0 cmap fix, table break seals, etc.) on top of the landing focus. Latest commit `09d5399` is the landing-only slice (`12 files, 719+ 1142-`); earlier 37 commits are described in `git log --oneline -20` and `plans/0.2.6/` ledgers.

## Impact

| Area | Impact |
|------|--------|
| **Performance** | No runtime engine impact. Landing is static/pre-rendered Vite chunk; main bundle hash rotated but size stable: `index-BUuSB4SA.js 287.26 kB (gzip 92.52 kB)`, `index-25J-iRC3.css 73.13 kB (gzip 12.83 kB)` at last `vite build`. |
| **Memory** | No change. Landing uses one `useState` vs prior 5 states + 2 `useMemo`; lighter client memory. |
| **Behavior / correctness** | Landing behavior changed by design (see Changes). Functional behavior of the PDF engine, CLI, library, golden corpus and `output/` samples not touched by `09d5399`. |
| **API / CLI** | No CLI/library API change. Docs routing (`frontend/src/App.jsx:68`) and content pages unchanged. |
| **Dependencies** | No dependency addition; stays on pinned `frontend` stack (`react 18.3`, `react-router-dom 6.30`, `vite 6.0`). Allowlist still `go-text/typesetting`, `tdewolff/canvas` per `Makefile:4`. |
| **Binary size / build time** | No Go binary impact. Frontend build `1.58s` (Vite) + `copy-to-docs`. `docs/` dirty check now passes (CI will fail if not rebuilt). |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

Landing class rename `landing-page` -> `landing-minimal` is internal frontend only; no public API or CLI contract change. No URL changes (`/` still `LandingPage`). Old CSS classes removed are not part of any external contract.

## Test plan

- [x] `npm --prefix frontend run lint` - `eslint . && node scripts/lint-data.mjs` -> `src/data lint clean (11 content pages, 61 showcase items)`
- [x] `npm --prefix frontend run build` - `vite build` `1.58s`, `copied build output -> docs` (`index-25J-iRC3.css 73.13 kB`, `index-BUuSB4SA.js 287.26 kB`)
- [x] `make claim-scan` - `claim-scan: clean` (no forbidden phrases in `doc.go`, `README.md`, `documentation/*.md`, `frontend/src/data/content`, `internal/cli/help.go`)
- [x] Manual visual check requested: Hero now reads `HTML to PDF` above the fold; code card toggles CLI/Go, copy works, `landing-micro` at `72ch` sits on one line on desktop, wraps on mobile; bento sections respect `1px solid var(--line)` and dark mode
- [ ] `make test` - deferred for this frontend-only slice (Go engine tests not affected by `09d5399`; full `master...HEAD` scope is 264 files and is covered by 0.2.6 engine golden corpus on the feature branch)
- [ ] `CGO_ENABLED=0 go build` - not required (no Go code in `09d5399`)

### Commands

```sh
npm --prefix frontend run lint
npm --prefix frontend run build
make claim-scan
# full gate when including prior 0.2.6 engine commits:
make test
make golden
```

## Screenshots / sample output

```
[vite build]
dist/assets/index-25J-iRC3.css  73.13 kB | gzip: 12.83 kB
dist/assets/index-BUuSB4SA.js  287.26 kB | gzip: 92.52 kB
✓ built in 1.58s
copied build output → docs (.nojekyll, assets, data, favicon.svg, go.mod, gopher.gif, gopher.png, index.html, logo.png, og-preview.png)

[lint]
> eslint . && node scripts/lint-data.mjs
src/data lint clean (11 content pages, 61 showcase items)

[claim-scan]
claim-scan: clean

[commit]
09d5399 feat(frontend): unify landing around HTML to PDF with minimal layout
12 files changed, 719 insertions(+), 1142 deletions(-)
```

Visual: Hero shows kicker `HTML to PDF · Pure Go · No browser, no cgo`, title `Your HTML, as a print-ready PDF.`, two CTAs, micro `Drop-in binary gowkhtmltopdf or native Go library Document. Static build with CGO_ENABLED=0.` at 72ch, code card with CLI `gowkhtmltopdf input.html output.pdf` / Go `Document.PDF(ctx)`. Remaining sections: pipeline flow, fit bento 2x2, proof strip 4 items, samples stack, close band `gowkhtmltopdf input.html output.pdf`.

## Related issues

- Relates to no prior GitHub issue - landing feedback was direct user report (HTML to PDF purpose unclear, need minimalism). Treated as UX improvement on top of `feature/06-high-fun-extended-CSS-support-Fixing-issues` (0.2.6 CSS coverage track).
- Branch tracks 0.2.6 ledgers `plans/0.2.6/` and CSS catalog; no `Closes` keyword used to avoid auto-closing unrelated engine tickets.

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me` -> `chinmay-sawant`)
- [x] Labels applied (`enhancement`, `documentation`)
- [x] Related issues filled
- [x] Filled body committed under `plans/PR/pr-unified-landing-html-pdf-minimal.md`

## Follow-ups (out of scope)

- Consider a hero input->PDF animated arrow or live `input.html` preview thumbnail once asset pipeline for preview images exists.
- Revisit whether the 38-commit breadth of this branch should be split into stacked PRs (engine/catalog vs docs/landing) for reviewer focus; currently shipped as single integration branch per `skills/PR/PR_TEMPLATE.md` epic guidance.
- Add a `frontend` smoke test asserting `LandingPage` still renders `PageTitle` and `/` route (regression guard for earlier `PageTitle` drop noted in `plans/0.2.0/frontend-improves/phase-wise-checklist.md:148`).

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff (note: 37 prior engine commits are intentional branch breadth; review focus may be last commit `09d5399` for landing intent)
- [ ] Public API / CLI changes documented (none in `09d5399`)
- [ ] New rules have fixture coverage when applicable (landing has no engine rules; `make golden` still required for prior CSS commits)
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed (`docs/assets` are required built output per `global.css` - `docs is the built static website generated from frontend`)
