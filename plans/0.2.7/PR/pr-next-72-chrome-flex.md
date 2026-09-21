## Summary

Closes the v0.2.7 next-72 CSS wave and the 40 Chromium Flexbox interaction cases, and cuts Chrome-case PDF allocation traffic from 64.3 MB to 28.0 MB per 40-case pass. 53 of the 72 next-72 names now have an apply arm plus a layout or paint consumer; 19 stay unused on purpose. Catalog after this branch is 407 Implemented / 0 Partial / 411 Unsupported of 818. VERSION stays 0.2.6; 44 commits, 328 files on `feature/027-next-72-with-chrome-test`.

---

## Motivation / context

- Plans: `plans/0.2.7/README.md`, `plans/0.2.7/87-canonical-0.2.7-next-72.md` (batches 87.1-87.8 closed 2026-09-17), `plans/0.2.7/chrome-flex-interactions/00-canonical-chrome-flex-interaction-plan.md` (cases 1-40 completed), `plans/performance/2026-09-21/00-canonical-chrome-case-alloc-reduction.md` (phases 0-2 shipped).
- Catalog source of truth: `plans/0.2.6/catalog/mapping.json` summary (`implemented: 407`, `partial: 0`, `unsupported: 411` of 818).
- Next-100 (`plans/0.2.7/88-canonical-0.2.7-next-100.md`, phases 88.1-88.9) is authored as checklists and JSON. Fixture-66 lands with the first 88.x implementation batch.
- Issues: see **Related issues**.

---

## Changes

### Next-72 CSS (batches 87.1-87.8)

Apply arms register on `styleGroups` in `internal/layout/style_cascade.go`. Consumers live in focused layout files. Catalog flips wait until apply + consumer + package test + matrix row exist (`plans/0.2.6/HONESTY-GATES.md`).

- 87.1 quick wins: `grid-gap` / `grid-row-gap` / `grid-column-gap` aliases, `grid-auto-columns` / `grid-auto-rows`, `overflow-block` / `overflow-inline`, `object-fit` / `object-position`, `counter-set`, `aspect-ratio`.
- 87.2 font features: `font-feature-settings`, `font-kerning`, `font-size-adjust`, `font-stretch` / `font-width`, `font-synthesis*`, `font-variant*`. OT tags reach the shaper through `paint.go` into `internal/pdf` and `internal/imageout`.
- 87.3 VF / palette: `font-optical-sizing`, `font-palette`, `font-variation-settings`. Variable faces instance `opsz` / `fvar` tags; COLR+CPAL uses the first palette color. Static Liberation/DejaVu stay a no-op.
- 87.4 text + hyphen: `text-autospace`, `text-box` / `text-box-edge` / `text-box-trim`, `text-spacing*`, `text-group-align`, `hanging-punctuation`, `hyphenate-limit-*`. `text-fit` stores the keyword and has no scale-search consumer, so it stays Unsupported.
- 87.5 column + initial-letter: `column-height`, `column-wrap`, `initial-letter*`. Drop caps need a real leading element (a `<span>`). `:first-letter` is still rejected by the selector parser.
- 87.6 shape + page float: `shape-outside` lite (`circle()` / `ellipse()` / `inset()`), `shape-margin`, `float-offset`, `float-reference`. Left Unsupported: `shape-inside`, `shape-padding`, `shape-image-threshold`, `float-defer`.
- 87.7 Borders-4 / Round Display: all 14 names stay Unsupported by choice (`border-clip` family, `border-limit`, `border-shape`, `border-boundary`). Matrix §5.5 records them.
- 87.8 closure: fixture-64 envelope, catalog recount, matrix rows. `make lint` was skipped on the 87 ledger on purpose.

Proof fixture: `testdata/golden/fixture-64-next-72-props.html`, envelope 5-8 pages, needle `NEXT-72-PROPS` (`internal/convert/golden_test.go:456-459`). Audit fonts under `testdata/fonts/implemented-audit/` (Cactus Classical Serif subset, `GowkVar-VF.ttf`, `PaletteDemo-Regular.ttf`).

### Chrome Flexbox cases 1-40

`test/chrome/manifest.json` holds 40 cases, all `status: completed`. Split: 24 `layout-unit`, 15 `chrome-reference`, 1 `golden-fixture` (case 29). `TestChromeCasePDFOutputs` only checks that each case converts to a parseable PDF. Box geometry lives in `internal/layout` (`flex_chrome_cases_*_test.go`).

- Cases 1-10: grow/shrink algorithm, minmax freeze, auto margins, fractional column shrink (8.5pt / 15.5pt), definite main size, justify-content, column cross-axis center (issue #75), vertical-rl alignment, flow orientations, RTL row-reverse.
- Cases 11-20: multiline wrap, column `align-content`, logical auto margins, baseline alignment across writing modes, per-line `align-self`, stretch, `flex-basis: 100%` in a nested column (Flexbox 9.4 step 5), RTL column wrap-reverse, row auto margins, auto min-size with overflow clip.
- Cases 21-40: row wrap + gaps, aspect-ratio cross size, writing-mode matrix, percentage abspos child, definite min-height, column percentage height, transferred min-width, nested-float print (3 pages), column auto margins including vertical-writing, reverse wrap, fractional grow, compressible items, max-content intrinsic width, border-box stretch, max-width freeze, replaced SVG ratio, wrapped gaps, row-reverse + vertical-rl overflow, min-content contribution.

`make chrome-cases-pdf` renders every `test/chrome/cases/case-*.html` through the built CLI into `test/chrome/cases/pdf/` (gitignored).

### Shared layout and paint (landed because of a case, applies to every document)

- Containing-block height, logical axes, and flex margins (`248c5b3`).
- Outline paint order via `opExtra.IsOutline`; `layout.Op` stays 256 bytes (`3485775`, `28d17fc`).
- Measure passes use `noEmit` so deferred chrome is not spliced at a stale index (`8b74c47`).
- Inline-block shrink-to-fit stops double-counting specified-width child margins; BFC keeps the last child's bottom margin (`6841d77`).
- Measure-pass floats no longer leak into the real BFC (`bfe4dfa`, case 29). Twin golden: `testdata/golden/fixture-29-wpt-break-nested-float-print.html`, envelope 3-3 pages.
- Vertical-writing column items center on the cross axis (`2678d72`).
- `flexIntrinsicWidth` returns border-box, padding and border included (`b3101f1`).
- Specified-height and transformed flex frames stay out of chrome stretch (`a416914`, `2aefa27`).
- `input[type=range]` paints an 11.25pt accent thumb; `input[type=button]` paints `value` at 10pt (`a158c1f`, `layout_widgets.go`).
- `border: solid transparent` keeps layout width and skips the stroke (`04914c0`). `hr`, collapsed table borders, and `column-rule-color` still drop alpha.
- Dashed and dotted fragments sit inside the border strip; a zero-length dotted fragment paints a square so MuPDF shows the dot (`7496277`).
- Continuation-page table body rows move up under a repeated thead when only pinned header clones sit above them (`9fe9b51`, `paint_flow_tables.go` `closeContinuationHeaderGap`). Fixture-64 page 7 lost a 63.65pt blank band.
- Fixture-56 dropped `aspect-ratio: 16 / 7` on `figure.d07-flow` because the engine now honors it and reserved about 173pt of blank space (`1b6672d`).

### Allocation and PDF writer

Measured on `BenchmarkChromeCasePDFs` (40 cases × 10 warm renders) at `e1ef59a`. Ledger: `plans/performance/2026-09-21/00-canonical-chrome-case-alloc-reduction.md`.

- Serial `/FlateDecode` keeps one zlib writer in an `atomic.Pointer` across GC (`internal/pdf/pdf.go` `flateSerial`). The old `sync.Pool` dropped an ~817 KiB state within two GC cycles. `flate.NewWriter` cumulative 245.15 MB -> ~22.5 MB. Pin: `TestFlateStateRetainedAcrossGC`.
- Convert resolves CSS once and shares the map with the independent-block probe, body layout, and the fit-to-page-width pass (`internal/convert/convert.go`, `layout.ContextWithStyles`). `IndependentBlocksForOptions` is deleted. Sheets with `@container` remount. Warm bytes-per-pass 64.3 MB -> 28.0 MB; pprof alloc_space 713.50 MB -> 326.98 MB.
- Hot PDF formatting leaves `fmt`: xref `appendPadZero`, ToUnicode bfchar 7,957 -> 2 allocs per 4,096 lines, CID `/W` 9,986 -> 3 allocs per 2,048 rows (`90e5fdd`).
- Style-store chunk tail was measured at about 1% waste vs a 5% gate and was left alone.

### Plans, skills, samples

- `plans/0.2.7/` ledgers for next-72, next-100, and Chrome flex. `documentation/compatibility-matrix.md` catalog line is 407 / 0 / 411, next-72 rows use full property names in backticks, and §5.5 lists the 14 Borders-4 deferrals.
- `documentation/deferred.md` records the 19 next-72 leftovers. `documentation/fonts.md` and `documentation/architecture/07-layout.md` describe vertical-rl/lr lite instead of parse-only. `documentation/samples.md` lists fixtures 57-64 and 29-wpt.
- Frontend compatibility and about pages now quote 407 Implemented / 0 Partial / 411 Unsupported. The old "font-feature-settings Unsupported" row is gone; next-72 Implemented groups and the 19 leftovers are in the property table. `npm --prefix frontend run build` refreshed `docs/`.
- Skills: `skills/chrome-debug-v2` (budgeted one-fixture Chrome-vs-Go PDF loop), `skills/perf-patterns` (measured Go hot-path catalog), `skills/chrome-flex-pdf-closure`, `skills/chrome-debug` kept as the long-form reference.
- `make samples` regenerated `output/` including `output/fixture-64-next-72-props.pdf` and `output/fixture-29-wpt-break-nested-float-print.pdf` (`1c449b1`, `9fe9b51`, `1b6672d`).
- `make run` times fixture-01 through the CLI and fails at 400ms.

No `go.mod` change. No Document / CLI / C ABI / Python stamp change.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Chrome 40-case warm bytes allocated per pass 64.3 MB -> 28.0 MB (-56.4%); allocs/op 70,091 -> 63,699 (-9.1%); pprof alloc_space 713.50 MB -> 326.98 MB. Mean per-case alloc -57%. Serial flate keep-alive plus one shared CSS resolve. `fmt` off xref / ToUnicode / CID `/W`. Wall time on that harness moved 1.43 -> 1.20 ms then recaptured at 1.47 ms; treat ns/op as host noise. 500-page golden alloc row was not recaptured. |
| **Memory** | Process now pins one serial zlib state (~817 KiB, buffer cap 16 MiB) instead of reallocating after GC. Multi-page jobs still pin up to 8 parallel workers. `ResolvedStyle` stays 4056 B; after sharing, style-store is still the top site (~62 records/case). `layout.Op` stays ≤ 256 B. RSS and binary size were not measured. |
| **Behavior / correctness** | 53 next-72 names now size or paint. Flex geometry matches the 40 Chromium-shaped cases on the layout tests. Transparent borders skip paint. Range and button faces draw. Continuation thead gap closes. Nested floats stay on the left edge through measure. Templates that already declared `aspect-ratio`, `object-fit`, font-variant, `shape-outside`, or `initial-letter` will layout differently. |
| **API / CLI** | None. `Document` / `ImageDocument` / CLI flags / `cli.Version` are unchanged. `IndependentBlocksForOptions` was internal. New Makefile targets: `chrome-cases-pdf`, `run`. |
| **Dependencies** | None. Allowlist still `go-text/typesetting` and `tdewolff/canvas`. |
| **Binary size / build time** | Not measured. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| CSS names that used to be ignored now size or paint (`aspect-ratio`, `object-fit` / `object-position`, `shape-outside`, font-variant / feature-settings, `column-height` / `column-wrap`, `initial-letter`, `counter-set`, `text-autospace`, `text-box-trim`) | Re-render documents that already declare those properties. In-repo example: fixture-56 dropped `aspect-ratio: 16 / 7` on `figure.d07-flow`. |
| Flex containers follow Chrome stretch, specified height, intrinsic chrome, writing-mode, and gap rules more closely | Visual check of flex layouts. No flag restores the old stretch. |
| Continuation-page table body rows sit under the repeated thead | Expected bugfix. Only surprises a template that used the blank band. |
| `input[type=range]` and `input[type=button]` grow native faces | Forms that assumed empty boxes will show a thumb / label. |
| `test/Chrome` import path renamed to `test/chrome` | Update any local invocation of the old path. |
| VERSION stays 0.2.6 | Binaries still stamp 0.2.6. Cut 0.2.7 later with VERSION + CHANGELOG + C/Python stamps together. |

Public library, CLI flags, and C/Python ABI: none.

---

## Test plan

Re-run on HEAD `1b6672d` (2026-09-21). Every listed gate exited 0.

- [x] `make test` (exit 0, 13s; `go test -p 2 -parallel 2 ./...`; convert 11.557s, pdf 2.785s, rest cached after the `-count=1` package runs)
- [x] `make golden` (exit 0, 7s; `TestGoldenCorpusAllFixtures` PASS including fixture-64 0.32s, fixture-29-wpt 0.07s, fixture-56 0.25s)
- [x] `make lint` (exit 0, 4s; golangci-lint v1.64.8, size-check clean with 3 allowlisted over-limit files, frontend eslint + data lint clean)
- [x] `make build` (exit 0, 2s; `bin/gowkhtmltopdf` and `bin/gowkhtmltoimage` stamped 0.2.6)
- [x] `make samples` (committed `output/` on `1c449b1`, `9fe9b51`, `1b6672d`; not re-run here so `output/` stays as committed)
- [x] `python3 scripts/css-catalog-map.py --check` (exit 0; `check ok (253 apply arms mapped)`)
- [x] `make claim-scan` (exit 0; `claim-scan: clean` after the documentation/frontend honesty pass)
- [x] `go test ./test/chrome -count=1` (exit 0, 0.095s)
- [x] `go test ./internal/layout -count=1` (exit 0, 3.722s)
- [x] `go test ./internal/convert -run 'TestGoldenCorpusAllFixtures/fixture-64-next-72-props.html' -count=1` (exit 0, 0.506s)
- [x] `go test ./internal/convert -run 'TestGoldenCorpusAllFixtures/fixture-29-wpt-break-nested-float-print.html' -count=1` (exit 0, 0.099s)
- [x] `go test ./internal/convert -run 'TestGoldenCorpusAllFixtures/fixture-56-architecture-diagram.html' -count=1` (exit 0, 0.282s)
- [x] `go test ./internal/pdf -run 'TestFlateStateRetainedAcrossGC' -count=1` (exit 0, 0.007s)
- [x] `go test ./internal/imageout -run TestRunRawHTMLColumnAlignItemsCenterPNG -count=1` (exit 0, 0.013s)

- [x] `make run` (exit 0, 12ms for fixture-01; hard cap 400ms)

### Commands

```sh
make test
make golden
make lint
make claim-scan
make build
make run
python3 scripts/css-catalog-map.py --check

go test ./test/chrome -count=1
go test ./internal/layout -run 'TestChrome' -count=1
go test ./internal/convert -run 'TestGoldenCorpusAllFixtures/fixture-64-next-72-props.html' -count=1
go test ./internal/convert -run 'TestGoldenCorpusAllFixtures/fixture-29-wpt-break-nested-float-print.html' -count=1

# optional visual PDFs (gitignored)
make chrome-cases-pdf

# optional alloc profile (not a Makefile target)
go test ./test/chrome -run '^$' -bench '^BenchmarkChromeCasePDFs$' \
  -benchtime=1x -count=1 -benchmem
```

Do not run `python3 scripts/generate_chrome_flex_cases.py` on this tree; it rewrites the hand-ported fixtures.

---

## Screenshots / sample output

```
Catalog (plans/0.2.6/catalog/mapping.json):
  implemented: 407
  partial: 0
  unsupported: 411
  ignored: 0
  webref_properties: 818

Next-72 slice (plans/0.2.7/next-72-properties.json):
  implemented: 53
  partial: 0
  unsupported: 19  (Borders-4 x14, text-fit, shape-inside,
                    shape-padding, shape-image-threshold, float-defer)

Golden envelopes:
  fixture-64-next-72-props.html                 5-8 pages, needle NEXT-72-PROPS
  fixture-29-wpt-break-nested-float-print.html  3-3 pages
  fixture-56-architecture-diagram.html          21-21 pages (aspect-ratio dropped)

Chrome flex:
  test/chrome/manifest.json  40 cases, all status: completed

Alloc (BenchmarkChromeCasePDFs, e1ef59a):
  warm bytes/pass  64.3 MB -> 28.0 MB
  allocs/op        70,091 -> 63,699
  pprof alloc_space 713.50 MB -> 326.98 MB

VERSION: 0.2.6
CHANGELOG: no 0.2.7 section yet

HEAD 1b6672d gates (2026-09-21, all exit 0):
  make build                         2s
  css-catalog-map.py --check         check ok (253 apply arms)
  make claim-scan                    clean
  go test ./test/chrome -count=1     0.095s
  go test ./internal/pdf -run TestFlateStateRetainedAcrossGC -count=1   0.007s
  go test ./internal/imageout -run TestRunRawHTMLColumnAlignItemsCenterPNG -count=1  0.013s
  convert fixture-64 -count=1        0.506s
  convert fixture-29-wpt -count=1    0.099s
  convert fixture-56 -count=1        0.282s
  go test ./internal/layout -count=1 3.722s
  make test                          13s
  make golden                        7s (corpus PASS)
  make lint                          4s (golangci-lint v1.64.8 + size-check + frontend)
  make run                           12ms fixture-01 (hard < 400ms)
```

Inspect `output/fixture-64-next-72-props.pdf` and `output/fixture-29-wpt-break-nested-float-print.pdf`.

---

## Related issues

- Closes #75 (`4c1c374` centered auto-width column flex items; cases 1-40 are the interaction coverage that issue asked for)
- Relates to `plans/0.2.7/87-canonical-0.2.7-next-72.md` (87.1-87.8 closed; fixture-64)
- Relates to `plans/0.2.7/chrome-flex-interactions/00-canonical-chrome-flex-interaction-plan.md` (40 cases completed)
- Relates to `plans/performance/2026-09-21/00-canonical-chrome-case-alloc-reduction.md` (phases 0-2 shipped; phase 3 deferred)
- Next-100 remains in `plans/0.2.7/88-canonical-0.2.7-next-100.md`. Fixture-66 HTML lands with the first 88.x implementation batch.

---

## PR metadata checklist (author)

- [x] Self-assigned (`chinmay-sawant`)
- [x] Labels applied (`enhancement`, `documentation`, `bug`, `performance`)
- [x] Related issues filled with real ticket IDs (`Closes #75`)
- [x] Filled body under `plans/0.2.7/PR/pr-next-72-chrome-flex.md`

GitHub PR: https://github.com/chinmay-sawant/gowkhtmltopdf/pull/76

---

## Follow-ups (out of scope)

- Author and implement next-100 (88.1-88.9) with fixture-66.
- Bump VERSION, add a CHANGELOG 0.2.7 section, and stamp C/Python together when cutting the release.
- Flip or keep the 19 next-72 Unsupported names (Borders-4, `text-fit`, shape-inside family, `float-defer`).
- Reconcile stale 393 / 14 Partial strings in `plans/0.2.6/property-counts.md` and `coverage-summary.json` `sources.engine` with mapping.json 407 / 0 / 411.
- Update root README and `documentation/deferred.md` to the 407 count.
- Chrome leftovers: `html, body` / `@page` reset on cases 16-19; abspos containing block vs initial containing block; inline-block vertical margins on the line box; `translateY` unscaled under smart-shrink; transparent color on `hr` / collapsed table borders / `column-rule-color`; solid border stroke still centered on the box edge.
- Alloc phase 3 style-store chunk tail (~1% waste).
- Phase 5 interaction-coverage registry in the Chrome parent plan.

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented (none)
- [ ] New rules have fixture coverage (fixture-64, fixture-29-wpt, 40 chrome cases)
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords (`Closes #75`)
- [ ] No secrets or generated artifacts committed (`output/` samples are regenerated on purpose; chrome PDF dirs are gitignored)
- [ ] Diff-stat-by-extension table pasted at the bottom
- [ ] Catalog claim in the body is 407 / 0 / 411, matching `mapping.json`, not the stale 393 / 14 markdown tables
- [ ] VERSION 0.2.6 and missing CHANGELOG 0.2.7 section are accepted as follow-ups

---

## Diff stat by extension

Generated from the staged tree vs `master` (same grouping as `bash scripts/pr-diff-stat.sh master`) on `feature/027-next-72-with-chrome-test`.

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.css` | 1 | 2 | 1 |
| `.go` | 151 | 21777 | 1466 |
| `.html` | 43 | 2958 | 1 |
| `.js` | 6 | 19 | 19 |
| `.json` | 7 | 4550 | 205 |
| `.md` | 43 | 5066 | 63 |
| `.pdf` | 76 | Binary | Binary |
| `.png` | 131 | Binary | Binary |
| `.py` | 3 | 1092 | 0 |
| `.sh` | 1 | 34 | 0 |
| `.ttf` | 3 | Binary | Binary |
| `.txt` | 2 | 17 | 3 |
| `.webp` | 129 | Binary | Binary |
| `.yaml` | 2 | 6 | 0 |
| No extension | 2 | 40 | 1 |
| **Total** | **600** | **35561** | **1759** |
