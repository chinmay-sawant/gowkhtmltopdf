# Ponytail packet 2 - CSS and conversion

> **Scope:** `internal/css/`, `internal/convert/`, and conversion preparation helpers
> **Canonical rows:** `PT26-CSS-01` through `PT26-CSS-05`, `PT26-CNV-01` through `PT26-CNV-07`
> **Method:** current caller search and focused source reading. No files changed and no tests ran.

## CSS matcher and parser

`internal/css/match.go:L14-183`: **shrink** replace the global mutex-protected 256-entry sibling LRU with local sibling scans in `getSiblingInfo`, plus direct previous and next scans. The cache builds five maps for each parent and every production use is inside this file. Estimated cut: 135 production lines plus cache tests. This must keep selector behavior and record a before/after CSS benchmark before closure.

`internal/css/match.go:L481-513,L755-782`: **shrink** make `containsWord` delegate to `hasClassToken` after its space guard. The current loops duplicate each other. Estimated cut: 22 lines. Prove with `go test ./internal/css -run 'Test(AttrWordAndSubstring|Match)'`.

`internal/css/values.go:L219,L851-854`: **delete** one-caller `namedColors`; `ParseColor` can read `namedColorTable` directly. Estimated cut: 3 lines. Prove with `go test ./internal/css -run TestParseColor`.

`internal/css/css.go:L231-256`: **stdlib** replace manual ASCII prefix comparison in `hasFoldPrefix` with a length guard and `strings.EqualFold`. The call sites are parser at-rule checks. Estimated cut: 20 lines. Prove with at-rule parser tests and the existing `FuzzParseCSS` corpus.

`internal/css/container.go:L116-125`: **stdlib** remove the uppercase pre-scan before `strings.ToLower` in `LengthToPt`. Estimated cut: 9 lines. Prove with `go test ./internal/css -run 'TestLengthToPt|Test(MediaMatchesSizeFeatures|ContainerCondMatches)'`.

## Conversion and preparation

`internal/convert/page_blocks.go:L13-60`, `internal/convert/convert.go:L642`: **shrink** remove the unused `log io.Writer` argument from `renderIndependentBlocks`. The body ends with `_ = log`. Estimated cut: 5 lines. Prove with independent-block tests.

`internal/convert/hf.go:L500-506,L745-754`: **delete** `drawHeadersFooters` and `hfDrawResult.emitWarnings`. They only call each other, carry unused suppressions, and production uses `drawHeadersFootersResult` at `pdf_pipeline.go:L267`. Estimated cut: 17 lines. Prove with targeted header/footer tests.

`internal/convert/convert.go:L65-69,L123-135,L630-638`, `internal/convert/page_islands.go:L1-190`, `internal/convert/islands/plan.go:L1-154`: **delete** benchmark-only page-island rendering. `NewBenchmarkPDFRequest` is used only by tests and benchmark code; normal requests use generic rendering. Estimated cut: about 330 production lines and 150-200 test or benchmark lines. First compare generic and certified output, then run `go test ./internal/convert` and `make golden`.

`internal/convert/prepare/simplify.go:L27-45`: **delete** test-only `SimplifyDOMEnabled` and `SimplifyDOMProfile`. Production uses `BuildOptions`. Estimated cut: 20 production lines plus about 45 test lines. Prove with `TestBuildOptionsSharedAcrossPDFAndImageLayers` and simplify tests.

`internal/convert/prepare/styles.go:L38-43,L439-444`: **delete** `CollectSheets` and `MergeFontFaces`. They only build a `ResourceContext` then delegate, and tests are their only callers. Estimated cut: 10 lines. Prove with targeted prepare, conversion, and image font-face tests.

`internal/convert/prepare/styles.go:L433-437`: **delete** test-only exported `LinkStylesheet`. Move the sole white-box test beside unexported `linkStylesheet`. Estimated cut: 5 production lines. Prove with prepare tests and `TestLinkStylesheetMediaMatches`.

`internal/convert/convert.go:L162,L270-278`: **delete** `ValidateRenderableObjects`. It forwards to `settings.ValidateRenderableObjects` and has one production caller. Estimated cut: 8 lines. Prove with request validation tests.

## Rejected lookalikes

- `internal/css/container.go:L24-31,L203-381`: retain the `ContainerCond` tree. It is marked `ponytail:` and current tests cover `and`, `or`, and `not`.
- `internal/convert/render/pipeline.go:L14-57`: retain the one-implementation `Pipeline` interface. Its staged cancellation contract is exercised through test fakes.
- `internal/convert/toc.go:L143-147`, `internal/convert/links.go:L353-367`: `cloneResult` and `remapPageForCopies` were completed in the August ledger. They are not active current findings.

Packet estimate: about -540 production lines before test changes, excluding the CSS cache measurement decision.
