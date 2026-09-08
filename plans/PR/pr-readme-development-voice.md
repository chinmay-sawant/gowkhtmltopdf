## Summary

- This branch stacks code-review remediation, fixture rendering fixes, and docs/process hygiene on top of master (18 commits, Go-only diff stat via `.gitattributes`).
- Code first: closes all SOLID/Go review rows and all 69 golang-design-patterns rows (validation, cancellation, sentinels, hot-path efficiency), plus shared conversion seams and layout style views.
- Fixtures: fixes fixture 59 form borders and missing textarea, fixture 56 page-10 workflow arrows, and fixture 60 border-image Effect cells.
- Docs tail: README Development rewrite, `RELEASE.md`, review indexes, and a clean `make lint` gate.

---

## Motivation / context

- Plans: `plans/0.2.6/review/golang-design-patterns-2026-09-08.md`, `plans/0.2.6/review/solid-go-review/00-synthesis.md` through `04-output.md`, `plans/0.2.6/review/README.md`, `RELEASE.md`.
- Issues: see **Related issues**.
- Context: the stacked fixture work on this branch predates the lint fixes that later landed on master via PR #66, so this branch also carries the lint cleanup needed for its own CI gate.

---

## Changes

### Engine review remediation (`29fe9cb`, 69 rows)

- Correctness: trims the 3-byte UTF-8 BOM with `strings.TrimPrefix` in `internal/html/html.go`.
- Correctness: bounds element depth at 1024 in `internal/html/html.go` to cap `Walk` recursion.
- Correctness: continues after the quoted span in `matchingParen` in `internal/css/has.go` so a paren after quotes is not skipped.
- Correctness: defers `zreader.Close()` immediately after `zlib.NewReader` in `internal/pdf/woff.go` so corrupt paths still close.
- Correctness: returns `(*float64, bool)` from `marginEdgePtr` in `internal/settings/reflect.go` instead of defaulting to Right.
- Correctness: switches on normalized value for case-insensitive grayscale in `internal/settings/settings.go`.
- Correctness: delegates `Canonical` to `Parse` with error returns in `internal/pdfprofile/profile.go`.
- Correctness: nil-guards outline collection and adds symmetric section helpers in `internal/outline/outline.go`.
- Correctness: caps the CGO allow-list length at 1024 before `unsafe.Slice` in `bindings/c/exports_cgo.go`.
- Correctness: nil-guards `cOutData`, `cOutLen`, and `cErr` before dereference in `bindings/c/exports_cgo.go`.
- Cancellation and bounds: checks `ctx.Err()` before `doc.Write` in `internal/convert/pdf_pipeline.go`.
- Cancellation and bounds: checks `ctx.Err()` at the top of the per-page loop in `internal/convert/hf.go`.
- Cancellation and bounds: threads `ctx` through `applyTOCLinks` and `applyInternalLinks` in `internal/convert/links.go` with per-loop checks.
- Cancellation and bounds: adds `internal/layout/pagination_ctx.go` with 64-interval polling, threaded through flex, grid, tables, and `paginateOps`.
- Cancellation and bounds: validates ranges for copies, quality, timeout, and zoom via `setIntRange` and `setFloatMin` in `internal/settings/reflect.go`.
- Cancellation and bounds: documents caller-owns-timeout for `Run`, `Fetch`, and `ImagesContext` in `internal/convert/convert.go`, `internal/convert/prepare/styles.go`, and `internal/layout/layout_flow.go`.
- Cancellation and bounds: routes loader callers to `NewLoaderWithError` and deprecates `NewLoader` in `internal/load/load.go`.
- Construction and contracts: validates viewport and media in `prepare.Options.validate` and returns `errNilLoader` from `Document` in `internal/convert/prepare/prepare.go`.
- Construction and contracts: validates width, height, crop, and media in `RenderOptions.Validate` in `internal/imageout/imageout.go`.
- Construction and contracts: validates finite values, media, and margins in `Options.validate` and `PaintOptions.validate` in `internal/layout/layout.go` and `internal/layout/paint.go`.
- Construction and contracts: rejects negative `MaxDepth` and returns `(*Node, error)` from outline builders in `internal/outline/outline.go`.
- Construction and contracts: adds `validateImageRange` in `bindings/c/options_image.go` and Python-side quality and crop validation in `bindings/python/src/gowkhtmltopdf/document.py`.
- Construction and contracts: adds zero-value sentinels (`OpUnknown`, `PDFUnknown`, `NodeUnknown`, `MediaUnset`, `subsetUnknown`) across layout, pdf, html, settings, and subset packages.
- Construction and contracts: adds `SetLog`, `SetClient`, `SetMaxBodySize`, and `SetMaxRedirects` setters in `internal/load/load.go`.
- Construction and contracts: removes the duplicate output-missing check in `cmd/gowkhtmltopdf/main.go` in favor of the CLI contract.
- Hot path: avoids per-at-rule `ToLower` with `hasFoldPrefix` in `internal/css/css.go`.
- Hot path: checks the anchor first and ranges `iter.Seq` without append in `internal/css/has.go`.
- Hot path: precomputes `wantLower` in `internal/css/selector_parser.go` and caches selector spec in `internal/css/has.go`.
- Hot path: adds a per-parent sibling cache for prev, next, index, and type totals in `internal/css/match.go`.
- Hot path: uses a copy-free `indexVarFunction` scan in `internal/css/values.go`.
- Hot path: builds one sorted struct-ops index for link collection in `internal/convert/links.go`.
- Hot path: adds `WalkUntil` and `FindFirst` early-exit traversal in `internal/html/html.go`.
- Hot path: replaces the global `sync.Map` plus `MustCompile` with a 64-entry LRU and checked compile in `internal/pdf/semantic.go`.
- Hot path: packs `boxKind` as `uint8` and groups bools in `internal/layout/layout.go`.

### SOLID and Go findings, style views (`dc4a2d5`, `553afed`, `c2d0651`)

- Shares copy-count validation through `internal/convert/render/plan.go` in `internal/convert/convert.go`.
- Extracts the shared layout-body plus smart-shrink path in `internal/convert/convert.go`.
- Shares resource preparation, cancellation checks, and body policies across `internal/convert/page_plan.go`, `internal/convert/page_islands.go`, `internal/convert/pdf_pipeline.go`, and `internal/convert/prepare/styles.go`.
- Shares PDF and image dimension, margin, and zoom validation in `document_validate.go`.
- Normalizes nil-command errors in `document.go` and `internal/app/pdf.go`.
- Fixes C adapter option contracts in `bindings/c/exports_cgo.go` and `bindings/c/options_image.go`.
- Fixes the CLI adapter contract in `internal/cli/cli.go`.
- Adds cancellable image resolution with `ImagesContext` in `internal/layout/layout_flow.go`.
- Clones document structure for tagged layout in `internal/layout/tagging.go` so clones stop sharing structure.
- Removes the dead style-interning map in `internal/layout/style.go`, keeping append-only stable storage.
- Detects image short writes with `io.ErrShortWrite` in `internal/imageout/imageout.go`.
- Serializes empty PDF streams correctly in `internal/pdf/pdf.go`.
- Reports external PDF byte counts in `internal/pdf/pdf.go`.
- Bounds font input in `internal/pdf/fonts.go` and `internal/pdf/registry.go`.
- Deduplicates JPEG resources in `internal/pdf/images.go`.
- Bounds SVG raster input in `internal/svg/raster.go`.
- Adds a layout-owned `boxModelStyle` view in `internal/layout/style_views.go` and migrates block width, margins, height constraints, and chrome helpers to it.
- Adds paint style view helpers in `internal/layout/paint_style_view.go` and updates `internal/layout/paint_pagination_chrome.go`.
- Classifies link hrefs with `isLinkHref` in `internal/layout/inline_paint.go`, keeping relative refs while rejecting `javascript:`, `data:`, `blob:`, and unknown schemes.
- Resolves protocol-relative `//host/path` links against the base scheme on the links path.
- Adds regression tests: `internal/convert/orchestration_contract_test.go` and `internal/convert/islands_oracle_test.go`.
- Adds regression tests: `internal/imageout/review_regression_test.go`, `internal/pdf/review_regression_test.go`, and `internal/pdf/font_resource_regression_test.go`.
- Adds regression tests: `internal/svg/review_regression_test.go`, `internal/layout/architecture_followup_test.go`, and `internal/layout/style_access_test.go`.
- Rewrites `skills/solid-go-review/SKILLS.md` with a package ownership map, bounded discovery packets, seam inspection, a reuse ladder, severity plus proof fields, and weighted scoring.

### Fixture rendering fixes (`572218f`, `cd5ada0`, `62377a9`)

- Fixes fixture 59 thick dark double borders: `outline: none` was defaulted to solid when width or color was set, now fixed in `internal/layout/outline.go` to respect explicit `none`.
- Fixes fixture 59 missing Message textarea on page 7: auto-height textareas had no intrinsic height.
- Fixes it with rows-based intrinsic height plus inline-block UA in `internal/layout/layout.go` and `pre-wrap` wrapping in `internal/layout/style_values.go`.
- Regenerates `output/fixture-59-apex-digital-landing.pdf` with thin 1px `#cbd5e1` borders and a rows=5 Message box matching Chrome print.
- Fixes fixture 56 page-10 workflow arrows: rotated border chevrons drew a long line over the css and layout pills.
- Replaces the chevrons with text arrows in `testdata/golden/fixture-56-architecture-diagram.css` and `testdata/golden/fixture-56-architecture-diagram.html`.
- Regenerates `output/fixture-56-architecture-diagram.pdf` so page 10 reads `html -> css -> layout -> pdf / image` with no overwritten line.
- Fixes fixture 60 props 81-86 Effect cells: all cells used solid `logo.png` frames so the six effects looked alike.
- Rebuilds those cells with shared `bi-box` demos over the new `testdata/golden/assets/border-slice.png` 9-slice source in `testdata/golden/fixture-60-implemented-props-a.html`.
- Records the new asset convention in `plans/0.2.6/catalog/implemented-fixture-authoring.md` and regenerates `output/fixture-60-implemented-props-a.pdf`.

### Test renames (`5babef0`)

- Renames `internal/convert/phase6_test.go` to `internal/convert/headers_toc_outline_links_test.go` with identical content.
- Renames `internal/convert/phase79_test.go` to `internal/convert/css_partial_remaining_test.go` with test names updated.
- Updates references in `documentation/architecture/08-convert-pipeline.md` and `documentation/compatibility-matrix.md`.

### README Development rewrite (`f33ac69`, `94cbba4`)

- Replaces the dense clean-room and pairing paragraph in `README.md` with short plain sentences.
- States the human and AI split directly: humans own architecture and the pipeline, AI tools help with drafts, golden-fixture checks, and tests.
- Points performance proof at `documentation/performance.md`, `testdata/golden/benchmarks/README.md`, and the CI perf budget test.
- Fixes grammar in the follow-up sentence about manual validation of the 50+ sample templates.

### Lint cleanup (`c84e2c1`, `988b1e4`, `.golangci.yml`)

- Adds explicit `OpUnknown`, `NodeUnknown`, and `line.Unknown` arms on flagged switches, grouped with existing no-op arms.
- Uses static sentinel errors plus `%w` wraps in layout, paint, prepare, outline, and load, with byte-identical rendered messages.
- Deletes dead code: `(*runContext).renderObjects`, the unused `log` parameter of `drawHeadersFootersResult`, `oldCollectBodyNavigation`, and four unused css index helpers.
- Extracts behavior-neutral helpers: `settleBeforeAlways` plus `assignFlowPages` in pagination.
- Extracts behavior-neutral helpers: `decodeImageBytes` in filters and `applyBodyLink` in internal links.
- Extracts behavior-neutral helpers: `matchSubstringOp` plus `attrWantValue` in attribute matching.
- Extracts behavior-neutral helpers: `buildAttrSelector` in the selector parser and `paintWidgetControl` in block building.
- Centralizes `testpackage` and `exhaustruct` test exclusions in `.golangci.yml` via `_test.go` exclude-rules and drops the per-file suppressions.
- Applies test hygiene: `t.Parallel` additions, short-name renames, line wraps, and focused-fixture `//nolint:exhaustruct` with reasons.

### Release and review docs (`efddd3d`, `9ade9fc`, `523e7b9`, `54684de`)

- Adds the audit-only Go design patterns review at `plans/0.2.6/review/golang-design-patterns-2026-09-08.md`.
- Adds the release checklist at `RELEASE.md` with the version-source table, Python-ships and WASM-does-not-ship status, and hard gates.
- Links the `RELEASE.md` hard gates and version table from `AGENTS.md`.
- Indexes the new review in `plans/0.2.6/review/README.md` and deletes the scoped rules at `plans/0.2.6/AGENTS.md`.
- Moves 4 PR bodies from `plans/PR/` to `plans/0.2.6/PR/` with zero content changes.
- Moves 5 solid-go-review files from `plans/phase/review/` to `plans/0.2.6/review/solid-go-review/` with zero content changes.

### Diff-stat hygiene (`5bcf10e`)

- Marks PDFs, images, fonts, and archives as `linguist-vendored` in `.gitattributes` so binaries skip diff stats.
- Marks `*.md`, `*.markdown`, `*.rst`, `/testdata/`, `/output/`, `/docs/`, and `/frontend/` as vendored in `.gitattributes`.
- Adds a fallback vendored rule with `*.go` and `go.mod` opt-out so PR stats count only Go.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Hot-path only: less per-rule string work in css matching, cached selector specs, one link index build, byte-slice semantic extraction; no perf regression claimed beyond existing CI budget |
| **Memory** | Small: 64-entry semantic LRU replaces an unbounded global map, sibling cache avoids repeated walks, box packing in layout |
| **Behavior / correctness** | Real fixes: explicit `outline: none` honored, textarea height restored, fixture 56 arrows and fixture 60 cells corrected, validation errors surfaced |
| **API / CLI** | Narrow: stricter `Validate` errors for bad options, canonical nil-command errors, fixed C and CLI adapter contracts; no new public API |
| **Dependencies** | None |
| **Binary size / build time** | Negligible |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| Stricter option validation (out-of-range copies, quality, zoom, bad crop or media now error) | Pass valid values; error strings name the bad field |
| `outline.BuildTree` and `BuildTreeBy` return `(*Node, error)`; `profile.Canonical` returns `(string, error)` | Handle the new error return |
| `NewLoader` deprecated for `NewLoaderWithError` | Migrate to the new constructor |
| None other | - |

---

## Test plan

- [x] `make build` (pass, per remediation commit gate)
- [x] `make test-quick -p 2` (pass, all packages, per remediation commit gate)
- [x] `make golden` (all 61 fixtures pass, per remediation commit gate)
- [x] `make claim-scan` (clean, exit 0)
- [x] `make lint` (clean after `c84e2c1` plus `988b1e4`, including frontend)
- [ ] Full `make test` plus `make golden` plus `make lint` re-run at merge time (branch stacks work that predates PR #66 lint fixes)

### Commands

```sh
make build
make test-quick -p 2
make golden
make claim-scan
make lint
```

---

## Screenshots / sample output

```text
output/fixture-56-architecture-diagram.pdf: page 10 workflow reads html -> css -> layout -> pdf / image
output/fixture-59-apex-digital-landing.pdf: thin 1px #cbd5e1 input borders, Message textarea present (rows=5)
output/fixture-60-implemented-props-a.pdf: props 81-86 show distinct stretch, outset, round, slice, source, width
claim-scan: clean
```

---

## Related issues

- None (review remediation closes rows in `plans/0.2.6/review/golang-design-patterns-2026-09-08.md` and the solid-go-review synthesis, not GitHub issues)

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`documentation`; suggest adding `enhancement` for the engine work and `bug` for the fixture fixes)
- [x] Related issues filled (none exist for this cleanup)
- [x] Filled body committed under `plans/PR/pr-readme-development-voice.md`

---

## Follow-ups (out of scope)

- Consider retitling the PR: the current `docs(readme)` title understates a branch that is mostly engine remediation plus fixture fixes.
- Full gate re-run (`make test`, `make golden`, `make lint`) right before merge since the stack spans the PR #66 lint landing.
- None other.

---

## Reviewer checklist

- [ ] Spot-check one remediation per phase (correctness, cancellation, contracts, hot path, hygiene) against the cited file.
- [ ] Confirm fixture PDFs show the described rendering (56 page 10, 59 page 7, 60 props 81-86).
- [ ] Confirm no unrelated changes beyond the listed commits.
- [ ] PR has assignee and labels.
