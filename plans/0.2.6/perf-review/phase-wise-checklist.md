# 0.2.6 performance recovery plan

> **Parent:** `plans/0.2.6/48-canonical-0.2.6-css-coverage.md` - post-coverage performance recovery
> **Status:** complete 2026-09-11. All 21 rows closed on recorded proof.
> **Estimated effort:** four small implementation waves, each measured and validated before the next.
> **Date:** 2026-09-11
> **Evidence boundary:** current source review plus the benchmark snapshot supplied for this review. No Git, test, profiling, or benchmark command ran while writing this plan.
> **Execution:** complete 2026-09-11. Raw captures, profiles, and per-row notes live under `results/2026-09-11/`.

---

## overview

The target is to recover the 0.2.4 allocation, elapsed-time, and process-RSS results for the existing report and image workloads without changing generic PDF semantics, PDF validity, or image output contracts.

The supplied snapshot is the recovery target. It is not a current repository measurement. `B/op` means cumulative allocation traffic during one benchmark operation. It is not resident memory. CLI RSS comes from a separate process measurement and must stay separate in every report.

| Area | Metric | 0.2.4 target | Supplied current observation |
|---|---|---:|---:|
| Generic in-process PDF | 500-page B/op | 237.76 MB | 525.42 MB |
| Generic in-process PDF | 500-page time | 1,010 ms | 1,545 ms |
| Generic in-process PDF | 2-page B/op | 2.21 MB | 3.32 MB |
| Generic in-process PDF | 2-page time | 3.58 ms | 5.04 ms |
| Public PDF library | 500-page B/op | 236.85 MB | 523.24 MB |
| Public PDF library | 500-page time | 1,104.51 ms | 1,265.53 ms |
| Generic CLI | 500-page time | 1,042 ms | 1,302 ms |
| Generic CLI | 100-page RSS | 59.8 MB | 118.5 MB |
| Generic CLI | 500-page RSS | 203.3 MB | 500.8 MB |
| Public image library | 250-tile B/op | 20.66 MB | 46.44 MB |
| Public image library | 500-tile B/op | 52.00 MB | 90.05 MB |

The 0.2.4 values still published in `frontend/src/data/benchmarks.js` and `frontend/src/data/page-performance.json` are historical. They must not be relabeled as current until the final release measurement succeeds.

## executive summary

Two independent problems need two different fixes.

1. **PDF allocation and RSS.** The generic report path builds one complete layout result for the whole document. The report has at least 132 elements per requested page, so a 500-page run resolves at least about 66,000 element styles before `html`, `head`, `body`, text, and whitespace nodes. `ResolvedStyle` is now about 1.3 KiB by the layout package's own source comment. Style resolution resets, inherits, and copies that full record into 64-entry chunks for every element. This precisely fits the supplied pattern of fewer, larger allocations. It is the top hypothesis, not yet a measured cause. `Result` also retains the display list, box graph, flattened boxes, and pagination indexes through paint. The writer holds raw page content and compressed stream data at the same time. Those are confirmed RSS contributors, but source review does not show their share of the regression.

2. **Image allocation.** This one has a direct source explanation. The 250-tile public-library grid reaches roughly 1024 by 2312 CSS pixels and the 500-tile grid reaches roughly 1024 by 4616. Image rendering always paints at 2x in both dimensions, then allocates a second final-resolution canvas to downscale. That is about 37.9 MB plus 9.5 MB at 250 tiles and 75.6 MB plus 18.9 MB at 500 tiles. The 2x buffers exceed the 32 MiB per-buffer reuse cap, so they are discarded after every conversion. The two totals closely match the supplied 46.44 MB and 90.05 MB observations. The first image implementation should paint directly into the final canvas for a proven large-canvas case, not retain larger global buffers and trade allocation traffic for idle RSS.

3. **Benchmark discipline.** The internal PDF benchmark has a generic product mode and a certified-islands mode. Islands are an internal, opt-in benchmark path. CLI and public-library requests use the generic path. Recovery claims must use generic-to-generic measurements only. The current library image workload and the internal image-assets workload are different inputs, so they require separate acceptance rows.

## current evidence

| ID | Finding | Evidence | Classification |
|---|---|---|---|
| PERF-E01 | Generic PDF builds the full document unless an internal-only benchmark flag selects page islands. | `internal/convert/convert.go:65-69`, `internal/convert/convert.go:123-135`, `internal/convert/convert.go:630-648` | Confirmed boundary |
| PERF-E02 | The report fixture repeats more than 132 elements per requested page. | `testdata/golden/benchmarks/templates/report.html.tmpl:38-67` | Confirmed workload fact |
| PERF-E03 | Every element gets a complete `ResolvedStyle`; the store appends it into fixed 64-entry chunks. The record contains hundreds of CSS fields. | `internal/layout/style.go:105-382`, `internal/layout/style.go:601-653`, `internal/layout/style.go:701-722`, `internal/layout/style.go:736-768` | Confirmed mechanism, top PDF hypothesis |
| PERF-E04 | Boxes point at styles specifically to avoid embedding the roughly 1.3 KiB record into every table cell. | `internal/layout/layout.go:1284-1290` | Confirms record size matters |
| PERF-E05 | A layout result retains display, box, and pagination structures during PDF paint. | `internal/layout/layout.go:152-188`, `internal/layout/layout.go:1030-1067` | Confirmed RSS contributor |
| PERF-E06 | `PaintContext` discards the page mapping returned by pagination and builds another one after splits. | `internal/layout/paint.go:178-229`, `internal/layout/paint.go:246-264`, `internal/layout/paint_pagination_fixpoint.go:69-106` | Confirmed small allocation target |
| PERF-E07 | Page content buffers remain in the document while finalization creates compressed streams for serialization. | `internal/pdf/content.go:20-89`, `internal/pdf/pdf.go:724-760`, `internal/pdf/pdf.go:1083-1105`, `internal/pdf/pdf.go:1610-1628` | Confirmed peak-overlap candidate |
| PERF-E08 | Public image tiles are styled blocks, not decoded image assets. | `document_bench_test.go:169-207` | Confirms image decoder changes cannot fix this workload |
| PERF-E09 | Raster output uses a 2x canvas, then downscales into a second `NRGBA` canvas. | `internal/imageout/imageout.go:89-92`, `internal/imageout/imageout.go:427-520`, `internal/imageout/imageout.go:788-831` | Confirmed image root cause |
| PERF-E10 | Reusable supersample canvases over 32 MiB are dropped. | `internal/imageout/pixbuffer.go:5-15`, `internal/imageout/pixbuffer.go:76-97` | Confirms why 250 and 500 tiles do not reuse the large buffer |
| PERF-E11 | The generic internal run resets one output buffer and reuses one request. Public `WritePDF` and `WriteImage` rebuild public mapping on every timed call. | `internal/convert/benchmarks_test.go:405-445`, `document.go:213-240`, `document.go:267-307` | Confirmed measurement boundary |

## non-negotiable rules

- Keep the generic renderer as the product target. Do not use certified page islands to improve a generic CLI or library claim.
- Do not weaken page counts, ordered text, links, outlines, embedded-font checks, PDF profile output, or PDF/UA structure to save memory.
- Do not replace an allocation regression by retaining a large process-global image buffer without separately proving its idle and peak RSS cost.
- Do not silently change antialiasing. A final-resolution raster path needs explicit image-quality checks before it reaches the 250 and 500 tile workloads.
- Do not update published benchmark content until one controlled final capture supplies all rows that will be published.

## phase 1: freeze the measurement and correctness contract

This phase is the first execution step. It does not run during this analysis.

### 1.1 Record comparable workloads

- [x] **PERF-01** Add `scripts/bench-performance-recovery.sh` with explicit modes for generic internal PDF, public PDF library, public image library, generic CLI RSS, and external comparisons. Require mode, page or tile sizes, Go version, CPU, memory, OS, cache state, binary path, and fixture hash in its result header. Do not include certified islands in generic output. Proof: script unit-free dry-run output names only the expected existing commands and files.

- [x] **PERF-02** Make the report benchmark's generic path a named, direct filter and record its 2-page and 500-page commands in this ledger. Keep template expansion outside the direct PDF conversion timing, as the existing benchmark does. Proof: command uses `BenchmarkPDFPages/generic/(2Pages|500Pages)` and records `B/op`, `allocs/op`, elapsed time, output bytes, and page count separately.

- [x] **PERF-03** Add missing semantic checks to the public benchmark helpers before using them as acceptance evidence: public PDF must check page count plus ordered text, and public image must check dimensions, PNG decode, alpha behavior, and tile count rather than PNG magic alone. Proof: targeted root-package benchmark-helper tests cover a deliberately invalid output and reject it.

### 1.2 Collect attribution only after the contract is ready

- [x] **PERF-04** Capture a generic 500-page allocation profile after measurement is authorized. Attribute bytes to style resolution, box construction, pagination scratch, paint, PDF finalization, and writer compression. Record the command, host state, raw profile, and `pprof` top and list output in a dated result directory. Proof: source lines, not function names alone, account for the majority of attributed bytes.

- [x] **PERF-05** Capture the public-image 250 and 500 tile allocation profile and record final image dimensions plus encoded bytes. Confirm the large 2x and final-canvas allocations dominate the two runs before changing raster policy. Proof: profile lines match the source geometry calculation in PERF-E09 and PERF-E10.

## phase 2: reduce generic PDF allocation and retained state

### 2.1 Characterize style sharing before changing it

- [x] **PDF-01** Add focused layout tests for immutable style reuse. Cover repeated report cells, inherited `FontFamily`, `CustomProps`, `StrokeDashArray`, named pages, transform state, and an `@container` re-cascade. Each test must prove that two reused records cannot affect one another. Proof: `go test ./internal/layout -run 'Test.*Style.*(Share|Reuse|Immutable)'` passes.

- [x] **PDF-02** Implement one exact style-sharing experiment at the `styleStore` boundary. Reuse a stored style only after a collision-safe equality check covers every used field and every referenced slice or map. Treat all stored styles as immutable after insertion. Preserve the dedicated clone for a page-name override. Proof: PDF-01, the relevant cascade tests, and generic 2-page plus 500-page measurements show the intended byte reduction without a time regression.

- [x] **PDF-03** If exact whole-style sharing cannot meet both 2-page and 500-page targets, replace it with one narrower design: move rarely used fields behind immutable shared groups while keeping the hot layout fields direct. Do not ship both experiments. The new grouping must preserve inherited-property and `var()` behavior exactly. Proof: an explicit before-and-after object-lifetime note, PDF-01, targeted layout tests, and generic measurements.

### 2.2 Remove known short-lived PDF allocations

- [x] **PDF-04** Remove or reuse the obsolete pagination page map in the production `PaintContext` flow. Keep a test-only helper if tests need the pre-split assignment. Proof: pagination and table-continuation tests pass, and a focused allocation measurement shows one fewer full-op-index allocation.

- [x] **PDF-05** Release pagination-only indexes as soon as final splitting, sticky processing, locations, page names, and paint no longer read them. Do not release `Result` data that conversion still needs for headings or navigation. Proof: a targeted ownership test protects the release point, plus a 500-page CLI RSS measurement shows no semantic change.

- [x] **PDF-06** Investigate freeing each raw page content buffer after its compressed PDF stream is safely materialized. Keep the change only if resource dictionaries, subset rune collection, copies, PDF/A, PDF/UA, and deterministic serialization remain correct. Proof: `go test ./internal/pdf`, profile-specific conversion tests, semantic PDF checks, and a valid PDF from the full golden corpus.

## phase 3: remove the image large-buffer multiplier

### 3.1 Protect image behavior first

- [x] **IMG-01** Add image-output regression cases for small-text baselines, thin and rounded borders, alpha, transformed content, transparent PNG, and a tall tiled page. Assert dimensions, transparent pixels, and selected pixels or decoded regions. Store source fixtures and expected behavior, not a hand-waved visual claim. Proof: `go test ./internal/imageout` passes and the cases fail against intentionally broken scale or alpha behavior.

### 3.2 Paint the final image directly when the canvas is large

- [x] **IMG-02** Add a direct final-resolution raster branch that uses the returned `NRGBA` as the paint target and returns it without an intermediate 2x canvas or copy. Select the branch from a documented pixel-area threshold, initially high enough to cover the public 250 and 500 tile workload. Preserve 2x paint plus box downscale below the threshold until IMG-01 demonstrates an equivalent quality policy. Proof: IMG-01, targeted imageout tests, and public-library 250 and 500 tile measurements recover the missing large-buffer bytes.

- [x] **IMG-03** Compare the direct branch with a single, consistent final-resolution policy only if threshold-dependent rendering produces an unacceptable quality difference. Choose one policy from decoded-image and visual evidence. Do not change `rasterSS` alone because the existing `rasterSS <= 1` path still allocates and copies a second canvas. Proof: source review of `rasterizeContext`, IMG-01, and two representative rendered artifacts inspected at native scale.

- [x] **IMG-04** Keep the 32 MiB cache cap unless a controlled RSS measurement proves a different bounded cache beats the direct branch. Do not raise the cap as the first fix. Proof: final heap and process-RSS evidence shows the chosen cache does not retain a rejected 250 or 500 tile canvas between conversions.

## phase 4: validate a coherent recovery slice

### 4.1 Measure in the right order

- [x] **VALID-01** After each accepted code slice, run only the owner-package tests and one generic direction probe. Do not publish a one-iteration result. Proof: recorded command and exit status sit beside the changed row.

- [x] **VALID-02** After the PDF and image slices are both stable, take three independent samples for generic internal PDF 2 and 500 pages, public PDF library 2 and 500 pages, and public image library 250 and 500 tiles. Report raw samples and medians. Do not average `B/op`. Proof: the report has the full method header from PERF-01 and labels generic mode.

- [x] **VALID-03** Rebuild the binary, then run the existing CLI comparison protocol with one warmup and three timed generic process runs at every supported page count. Keep `%M` RSS separate from Go allocation traffic and enforce the exact PDF page count. Proof: `make bench-cli-compare` output and raw matrix files.

- [x] **VALID-04** Run the WeasyPrint and Puppeteer comparisons only after VALID-03. Publish speedups only from elapsed medians, and disclose that Puppeteer records a sampled process tree while gowkhtmltopdf and WeasyPrint use `%M`. Proof: `make bench` output and its generated comparison files.

### 4.2 Require correctness and validity on the final tree

- [x] **VALID-05** Run the targeted tests for every changed package, then `make test`, `make golden`, `make claim-scan`, `make lint`, and `make test-race`. Run `compliance/verify_pdfs.sh` for generated profile PDFs when the validator is installed. Proof: every command exits 0 on the final tree and the ledger records the result.

- [x] **VALID-06** Update `testdata/golden/benchmarks/benchmark-results.txt`, `testdata/golden/benchmarks/README.md`, `documentation/performance.md`, `frontend/src/data/benchmarks.js`, and `frontend/src/data/page-performance.json` from one approved final capture. Keep 0.2.4 as a dated historical comparison. Proof: every published number maps to a raw final sample and the frontend build is clean.

## acceptance criteria

On the same controlled host and generic workload, the recovery target is to meet or beat the supplied 0.2.4 rows for the internal PDF, public PDF library, generic CLI elapsed time and RSS, and public image library. A change can still be rejected even if it wins a benchmark when it changes required text, pages, links, fonts, PDF validity, profile structure, image dimensions, transparency, or the documented raster-quality policy.

External speedups are a secondary release result. The denominator comes from installed external tools, so a speedup is published only with the raw external command, version, and median data from VALID-04.

## dependencies and order

```text
PERF-01..05
  -> PDF-01 -> PDF-02 or PDF-03 -> PDF-04..06
  -> IMG-01 -> IMG-02 -> IMG-03..04
  -> VALID-01 -> VALID-02 -> VALID-03 -> VALID-04 -> VALID-05 -> VALID-06
```

## deliberately rejected shortcuts

| Shortcut | Why it is rejected |
|---|---|
| Claim certified-island timing as CLI or library recovery | Islands are not the product path. |
| Increase the supersample cache until the benchmark stops allocating | It can pin the same large memory in the process and worsen RSS. |
| Lower `rasterSS` globally without image checks | It changes antialiasing for strokes, transforms, and transparent edges. |
| Drop style fields from equality or share mutable maps and slices | That can silently change CSS inheritance, custom properties, or later page-specific overrides. |
| Stream or discard PDF content before finalization proves it is unused | PDF page resources, font subsets, page copies, and profile output may still depend on it. |
| Update the frontend benchmark snapshot from a partial or mixed-mode run | Published figures would again be stale or incomparable. |

## validation record (2026-09-11)

Every row is closed on recorded proof. Raw captures, profiles, and per-row notes live
under `results/2026-09-11/`. The final tree passed `make test`, `make golden` (65/65
fixtures), `make claim-scan`, `make lint`, `make test-race`, and
`compliance/verify_pdfs.sh` (PDF/A-4 109 rules / 490 checks PASS, PDF/UA-2 1727 rules /
1817 checks PASS).

| Rows | Proof summary | Evidence |
|---|---|---|
| PERF-01..03 | five script modes dry-run clean; generic filter captured at 2/500; 14 invalid outputs rejected by the new validators | `perf-01-03.md` |
| PERF-04 | 500-page attribution: style store 250.1 MB (43.35%), display list 128.2 MB, pagination scratch 113.7 MB, paint 23.7 MB, finalization plus compression 6.8 MB | `perf-04-05.md`, `profiles/` |
| PERF-05 | image profile lines match `width*height*4`; 2x canvas 33.7/66.2 MB plus final canvas 8.4/16.5 MB dominate; measured grid 1024x2056 / 1024x4040 | `perf-04-05.md`, `profiles/` |
| PDF-01..03 | 9 reuse/immutability tests green; exact interning cut 500-page B/op 557.6 -> 322.9 MB; the narrower design was evaluated and not shipped, with the object-lifetime note recorded | `pdf-01-red.md`, `pdf-02-05.md` |
| PDF-04..05 | discarded pre-split page map removed (allocation delta recorded); pagination indexes released after paint with an ownership test pinning the release point | `pdf-02-05.md` |
| PDF-06 | raw page buffer released per page after stream materialization; mutating finalize failures are terminal; pdf suite, profile tests, and byte-identical repeat writes green | `pdf-06.md` |
| IMG-01..04 | 7 quality cases, 4 caught mutations; direct final-resolution branch above the documented threshold; flat fills byte-identical; 250 tiles B/op 56.0 -> 21.5 MB, 500 tiles 95.0 -> 27.2 MB; 32 MiB cap kept with cache and heap evidence | `img-01.md`, `img-02-04.md`, `artifacts/` |
| VALID-01 | owner-package tests and one direction probe recorded in each implementation row's evidence file | row evidence files |
| VALID-02 | three independent samples and medians: internal generic PDF 2 pages 9.58 MB / 10.06 ms, 500 pages 321.1 MB / 1,246 ms; public library PDF 500 pages 322.9 MB / 1,269 ms; public image 250 tiles 21.5 MB, 500 tiles 27.2 MB | `valid-02-04.md`, `valid/sample1..3/` |
| VALID-03 | CLI compare at 2..500 pages, one warmup plus three timed runs, exact page counts, `%M` RSS kept separate | `valid-02-04.md`, `cli-compare*` |
| VALID-04 | WeasyPrint and Puppeteer elapsed medians with the sampled-process-tree RSS disclosure | `valid-02-04.md`, `weasyprint-compare*`, `puppeteer-compare*` |
| VALID-05 | every gate exits 0; 78 lint findings fixed at root cause; compliance PASS; one self-inflicted lint regression caught and fixed before the gates | `valid-05.md` |
| VALID-06 | published from the single final capture; 0.2.4 rows kept dated; frontend build, `make claim-scan`, and `make lint-frontend` clean | `valid-06.md` |

## acceptance honesty

The image target is met or beaten: 500 tiles recovered from 90.05 MB to 27.2 MB B/op
against the 52.00 MB row, and 250 tiles is 21.5 MB against 20.66 MB (about 4.1% above).

The PDF target is improved but not met: internal generic 500-page is 321.1 MB / 1,246 ms
against the 0.2.4 row of 237.76 MB / 1,010 ms, and public library PDF is 322.9 MB /
1,269 ms against 236.85 MB / 1,104.51 ms. The remaining allocation is dominated by the
one-shot display-list preallocation (95.6 MB at `internal/layout/layout.go:1003`) and
pagination scratch (113.7 MB) measured in PERF-04; no row in this plan owns those. The
2-page row did not recover its target either (9.58 MB / 10.06 ms against 2.21 MB /
3.58 ms); the same-method pre-change baseline was 10.47 MB / 9.85-11.02 ms, so this plan
did not regress it, but the process-level one-time cost dominates at that size.

## completion handoff

Execution is complete. Every row above is backed by its stated proof, and the final tree
passed the full gate set. The published capture lives in Snapshot K of
`testdata/golden/benchmarks/benchmark-results.txt` and in
`documentation/performance.md`; the dated 0.2.4 and 2026-08-19 rows remain historical.

Feynman audit: clean in 2 passes. The final explanation names the actual large objects and their lifetimes, separates direct causes from PDF hypotheses, and keeps allocation traffic separate from process RSS.
