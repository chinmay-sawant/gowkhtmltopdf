## Summary

This integration PR brings the current renderer architecture, public request API, page-island/resource-lifecycle work, performance wave, fixture repairs, benchmark evidence, and review-ledger updates together on `master`. It also records the validation boundary for the full branch so the architectural and performance claims remain tied to executable tests, golden fixtures, and documented benchmark snapshots.

The branch is intentionally broad: it contains the completed refactor sequence from the typed request/API seams through the certified page-island workspace and the associated layout, PDF, raster, benchmark, and documentation work.

## Motivation / context

- The conversion path had accumulated large orchestration seams, duplicated lifecycle state, mutable cross-stage ownership, and expensive whole-document/page-island retention.
- Public callers needed typed request/options entry points, explicit errors, defensive copying, and cancellation-aware execution without removing the existing wkhtmltopdf-compatible settings surface.
- Performance work needed to preserve page counts, layout/PDF semantics, image correctness, and fixture fidelity while reducing allocation traffic and retained page-island state.
- The branch’s benchmark evidence distinguishes in-process Go `B/op` from process RSS and labels historical measurements separately from the current certified-workspace snapshot.
- The repository PR template is retained as the structural basis for this body; its stale Goslop/`main` examples are corrected here for `chinmay-sawant/gowkhtmltopdf` and the actual `master` integration branch.

## Changes

### Public API and request contracts

- Add typed `PDFRequest`, `ImageRequest`, `PdfGlobalOptions`, and `ImageSettings` flows while retaining the compatible string-based settings API.
- Export stable conversion errors and add nil-safe behavior for public settings/converter methods where callers previously could panic.
- Clone settings, byte slices, maps, headers/footers, post items, and allowlists at ownership boundaries so later caller mutation cannot alter an in-flight conversion.
- Add explicit internal request validation and typed request-to-runtime adapters for PDF and image modes.
- Preserve context cancellation, progress reporting, output ownership, outline output, and library/CLI behavior across the adapters.

### Conversion pipeline and package seams

- Separate document preparation, simplification, style preparation, render planning, page planning, rendering, page-island planning, and PDF output lifecycle responsibilities.
- Add focused `internal/convert/prepare`, `internal/convert/render`, and `internal/convert/islands` packages and their tests.
- Make page-island ownership/certification explicit, reuse certified workspace storage where safe, and release completed islands instead of retaining the complete expanded document unnecessarily.
- Isolate header/footer geometry, links/navigation, outline projection, TOC handling, and page-plan responsibilities behind narrower helpers.
- Keep failure propagation and resource-limit validation explicit at loader, conversion, image, and PDF boundaries.

### Layout, paint, pagination, and rendering correctness

- Split layout and paint responsibilities into focused files for flow, tables, images, chrome, measurement, paint order, pagination, and style properties.
- Centralize paint ordering and route cross-page operation movement through indexed pagination helpers so `flowPages`, `flowPageOf`, and `flowPos` stay synchronized.
- Repair forced-break/table continuation behavior and retain regression coverage for text, borders, repeated headers, sticky content, relative offsets, transforms, floats, flex/grid, multicolumn, and orphan/widow cases.
- Extend CSS/HTML value and container handling while retaining the existing pure-Go rendering boundary.
- Add the complex dossier fixture and golden-corpus coverage for the combined five-page layout surface.

### Performance and resource work

- Reduce allocation and retained-memory pressure in page-island rendering, style/cascade handling, inline text collection, text measurement, table/display-list construction, PDF content/resource ownership, image rasterization, and TTF raster buffers.
- Add/reuse pool-backed and allocation-conscious paths for raster and font work, including lock-free/zero-allocation lookup paths where the measured seam supports them.
- Retain semantic gates: exact/expected page envelopes, PDF validity, layout and image tests, fixture rendering, and no semantic weakening for performance wins.
- Record the current certified-workspace 500-page PDF snapshot as approximately `0.500s / 157.6MB B/op / 529.4K allocs/op` in the branch documentation; treat this as in-process Go allocation traffic, not RSS.
- Retain the controlled direct CLI comparison against installed `wkhtmltopdf 0.12.6.1`: at 500 pages the documented matrix records approximately `890ms` versus `1,720ms` and `50,888 KiB` versus `116,512 KiB` peak RSS for Go versus wkhtmltopdf. This is workload-specific evidence, not a universal browser-engine ranking.
- Keep the one-pass 43-fixture installed-binary PDF run as a local ignored artifact/report workflow; generated local comparison PDFs are not required for the source/benchmark snapshot commit.

### Benchmark and fixture evidence

- Extend the Go benchmark matrix for PDF pages, template-plus-PDF pages, deterministic web-fetch image tiles, and inline image assets across 2, 5, 10, 20, 50, 100, 200, 250, and 500 workloads.
- Add the wkhtmltopdf comparison benchmark and retain explicit binary-path/version, wall-time, RSS, PDF-size, and metric-compatibility notes.
- Refresh checked-in benchmark result snapshots and output fixtures without conflating generated artifacts with reproducible source evidence.
- Add fixture corpus hygiene and structural checks covering PDF signatures, EOF/xref validity, page envelopes, embedded fonts, images, annotations, and feature expectations.

### Review, skill, and durable planning records

- Add the critical Go review and performance-review skills used to structure the branch’s audits and validation handoffs.
- Preserve the architecture, code-style, design-pattern, RSS, allocation, performance, and phase-wise review records as repository-local evidence.
- Synchronize the golden corpus and benchmark README guidance with the executable commands and current snapshot boundaries.
- Update the Ponytail skill metadata/help records used by the review workflow.

## Impact

| Area | Impact |
|---|---|
| **Performance** | Lower measured allocation traffic and retained page-island state; current certified-workspace evidence is recorded separately from historical snapshots. The direct CLI matrix documents Go versus wkhtmltopdf at controlled page sizes. |
| **Memory** | Page-island workspace reuse/release and raster/font pooling reduce retained or repeated allocation pressure while keeping resource limits and PDF/layout gates active. |
| **Behavior / correctness** | Preserves PDF/page validity, fixture page envelopes, layout/paint semantics, navigation, outlines, images, table borders, repeated headers, and pagination-index invariants under regression coverage. |
| **API / CLI** | Adds typed public PDF/image request/options paths and explicit errors while retaining the compatible CLI/settings surface. |
| **Dependencies** | No new runtime dependency beyond the existing allowlisted pure-Go shaping/SVG dependencies; no CGO/browser dependency is introduced. |
| **Binary size / build time** | Package decomposition and additional tests increase source/test surface; both CLI binaries still build successfully. |

## Breaking changes / migration

| Item | Migration |
|---|---|
| None intended | Existing `GlobalSettings`, `ObjectSettings`, CLI-compatible settings, and converter entry points remain available. New typed request/options APIs are additive. |

## Test plan

- [x] `GOCACHE=/tmp/gowk-go-cache go test ./...` with loopback access for `httptest`-based tests.
- [x] `GOLANGCI_LINT_CACHE=/tmp/gowk-golangci-cache GOCACHE=/tmp/gowk-go-cache make lint`.
- [x] `GOCACHE=/tmp/gowk-go-cache make golden`.
- [x] `GOCACHE=/tmp/gowk-go-cache make build`.
- [x] `git diff --check`.
- [x] Golden corpus converted through the full load/parse/style/layout/paint/PDF pipeline; all fixture assertions passed.
- [x] Installed `wkhtmltopdf 0.12.6.1` converted all 43 standalone golden fixtures in a local one-pass run; generated PDFs were non-empty and Ghostscript validated 43 PDFs with 67 total pages.
- [x] Current benchmark docs distinguish Go `B/op` from process RSS and preserve historical/current labels.

### Commands

```sh
GOCACHE=/tmp/gowk-go-cache go test ./...
GOLANGCI_LINT_CACHE=/tmp/gowk-golangci-cache GOCACHE=/tmp/gowk-go-cache make lint
GOCACHE=/tmp/gowk-go-cache make golden
GOCACHE=/tmp/gowk-go-cache make build
git diff --check
```

Installed-binary fixture capture:

```sh
/usr/local/bin/wkhtmltopdf --enable-local-file-access --quiet \
  testdata/golden/fixture-01-simple-invoice.html \
  output/wkhtmltopdf/fixture-01-simple-invoice.pdf
```

The complete local timing report from the 43-fixture run is generated as
`output/wkhtmltopdf/benchmark-summary.md` and
`output/wkhtmltopdf/benchmark-results.csv`; that directory is ignored by the
repository’s existing broad `wkhtmltopdf` ignore rule and is intentionally
kept out of the source PR.

## Screenshots / sample output

Representative checked-in outputs and evidence:

- `output/fixture-01-simple-invoice.pdf` through `output/fixture-43-complex-dossier.pdf` cover the refreshed golden fixture output set.
- `output/showcase-toc-hf-outline.pdf` covers TOC, headers/footers, outline, and section output.
- `testdata/golden/benchmarks/benchmark-results.txt` contains the raw checked-in Go benchmark snapshot.
- `testdata/golden/benchmarks/README.md` contains the current matrix, direct CLI comparison, metric caveat, reproduction commands, and artifact policy.
- `plans/performance/2026-08-09/fresh-go-architecture-review-2026-08-09/` contains the companion HTML executive-review pages in addition to the Markdown review records listed below.

## Markdown references on this branch

These are all 19 Markdown files changed relative to `origin/master` and are the durable documentation/review records included in the branch diff:

- `README.md` — product/API, fixture, benchmark, CLI comparison, and repository guidance.
- `plans/deferred/0.0.3/500-page-allocation-and-latency-optimization-plan.md` — closed allocation/latency wave, gates, cycle log, residual boundary, and wkhtmltopdf history.
- `plans/deferred/0.0.3/500-page-one-second-performance-target.md` — one-second objective, candidate architecture, acceptance criteria, and wkhtmltopdf reference.
- `plans/performance/2026-08-07/performance-profile-and-improvement-plan.md` — profiling evidence, optimization checklist, validation gates, and current comparison addendum.
- `plans/performance/2026-08-09/500-page-allocation-architecture.md` — canonical allocation architecture ledger and certified page-island direction.
- `plans/performance/2026-08-09/fresh-go-architecture-review-2026-08-09/critical-golang-architecture-review.md` — critical review scorecard, findings, validation matrix, and roadmap.
- `plans/performance/2026-08-09/fresh-go-architecture-review-2026-08-09/golang-10-out-of-10-architecture-review.md` — consolidated 10/10 architecture/code-quality roadmap and certification checklist.
- `plans/performance/2026-08-09/fresh-go-architecture-review-2026-08-09/golang-code-style-fresh-review.md` — fresh code-style findings, remediation order, and validation boundary.
- `plans/performance/2026-08-09/fresh-go-architecture-review-2026-08-09/golang-design-patterns-fresh-review.md` — fresh Go design-pattern findings, dependencies, and phase-wise closure criteria.
- `plans/performance/2026-08-09/fresh-go-architecture-review-2026-08-09/phase-wise-implementation-checklist.md` — phase closure record for API, page islands, context, pagination, package decomposition, and release gates.
- `plans/performance/2026-08-09/golang-code-style-architecture-review.md` — architecture-level code-style findings and checklist.
- `plans/performance/2026-08-09/golang-design-patterns-architecture-review.md` — architecture-level design-pattern findings and checklist.
- `plans/performance/2026-08-09/rss-reduction-phase-wise-checklist.md` — RSS reduction phases, evidence, hard gates, and residual follow-ups.
- `skills/critical-go-review/SKILLS.md` — critical review workflow and subagent prompt contract.
- `skills/perf-review/SKILLS.md` — performance review wave workflow and handoff requirements.
- `skills/ponytail-help/SKILL.md` — Ponytail help/configuration reference updated with the branch’s review workflow.
- `skills/ponytail/SKILL.md` — Ponytail persistence/intensity/output guidance updated for the review process.
- `testdata/golden/README.md` — golden corpus layout, fixture inventory, pass criteria, and reproduction commands.
- `testdata/golden/benchmarks/README.md` — benchmark templates, historical/current snapshots, direct wkhtmltopdf comparison, and artifact policy.

## Related issues

- No related issue number was supplied for this user-requested integration PR.

## PR metadata checklist (author)

- [x] Self-assigned as `chinmay-sawant` through the GitHub connector.
- [x] Labels applied: `enhancement`, `documentation`.
- [x] Related issues explicitly recorded as unavailable rather than inventing ticket IDs.
- [x] Filled body committed under `plans/PR/pr-optimization-with-refactors.md`.
- [x] Base branch verified as `master` from repository metadata.

## Follow-ups (out of scope)

- Complete the remaining allocation-architecture follow-up toward the documented `100 MB/op` stretch target without weakening layout/PDF semantics.
- Finish the planned `internal/convert` package decomposition and reduce remaining context/ownership complexity.
- Add CI automation for the benchmark certification matrix and stronger visual artifact publication.
- Decide whether ignored installed-wkhtmltopdf per-fixture PDFs should be published as release artifacts rather than committed source outputs.

## Reviewer checklist

- [ ] Behavior matches summary and test plan.
- [ ] No unrelated changes in diff.
- [ ] Public API / CLI changes documented.
- [ ] New or repaired rendering surfaces have fixture coverage.
- [ ] PR has assignee and labels.
- [ ] Related issues use correct keywords or explicitly state that none were supplied.
- [ ] No secrets are committed; generated artifacts are included only where the branch’s existing output policy intentionally tracks them.
