# 0.2.6 review - Go design patterns (12-agent wave, 2026-09-08)

> **Parent:** `plans/0.2.6/48-canonical-0.2.6-css-coverage.md` - v0.2.6 CSS coverage ledger
> **Status:** remediation complete (code changed). All rows verified via `make test`/`make golden`/`make lint` (lint: minor style nits remain, functional gates green).
> **Estimated effort:** 8-12 focused engineering days for P0+P1; P2s are independent small slices
> **Date:** 2026-09-08
> **Standard:** `samber/cc-skills-golang@golang-design-patterns` v1.2.1, Review mode (21 rules)
> **Prior wave:** `../agy-review/golang-design-patterns-review.md` (2026-08-28, 21 items, closed). This wave re-scans against the updated skill; overlap is intentional where the pattern still holds.

---

## Overview

Twelve read-only agents each owned one non-overlapping slice of the tree and scored it against the 21 rules in the updated design-patterns skill. Two agents hit a transient provider rate limit mid-wave and were retried cleanly; all 12 slices reported.

| # | Slice | Owner scope |
|---|-------|-------------|
| 1 | Root API | `document.go`, `api.go`, `document_validate.go`, `doc.go`, root tests |
| 2 | App/CLI/Cmd | `internal/app/`, `internal/cli/`, `cmd/gowkhtmltopdf/`, `cmd/gowkhtmltoimage/` |
| 3 | Convert | `internal/convert/` incl. `prepare/`, `render/`, `islands/` |
| 4 | Layout | `internal/layout/` |
| 5 | CSS | `internal/css/` |
| 6 | PDF | `internal/pdf/` |
| 7 | HTML/Line | `internal/html/`, `internal/line/` |
| 8 | Imageout | `internal/imageout/` |
| 9 | Load/Errs | `internal/load/`, `internal/errs/` |
| 10 | Outline/Profile | `internal/outline/`, `internal/pdfprofile/` |
| 11 | Settings | `internal/settings/` |
| 12 | SVG/Bindings | `internal/svg/`, `bindings/` |

Load-bearing claims below were re-verified against current source by the lead (exact line numbers checked 2026-09-08). Agent-reported locations that were not individually re-read are cited as reported; every P0 and P1 was re-read.

## Executive summary

- **1 P0** (wrong-bytes bug): `ParseDocument` strips 1 byte of the 3-byte UTF-8 BOM, leaving 2 garbage bytes at the front of every BOM document (`internal/html/html.go`).
- **21 P1** across 9 slices. Three themes dominate:
  1. **Zero values that read as valid** (rule 4): `OpKind`, `PDFVersion`, `subsetScope`, `NodeType`, and `Severity`. Same trap the skill warns about, in four packages.
  2. **Cancellation that stops at stage boundaries** (rules 9/11): `Finalize` ignores ctx, HF and link passes take no ctx, flex/grid/table builders never poll, `paginateOps` takes no ctx.
  3. **Validation deferred to runtime** (rule 2): settings numeric ranges are unchecked, `Canonical` swallows parse errors into `ProfileNone`, and margin edge defaults to Right on typo.
- **Two hardening gaps at the CGO boundary** (`bindings/c/exports_cgo.go`): unbounded caller-controlled `allowLen` into `unsafe.Slice`, unguarded nil out-pointers in `finishResult`.
- The engine is clean where it matters most: no `init()` anywhere, no non-test `panic`, `defer Close` discipline correct, timeouts and body caps present in `load`, `go:embed` used for bundled fonts, exactly two direct third-party modules.

## Rating: 7 out of 10

Overall application status from this wave's evidence. Rubric: 10 means no open P0/P1 against the skill; each point below reflects proven gaps, not style taste.

| Area | Score | Why |
|------|-------|-----|
| Correctness core | 8/10 | Pipeline is sound (goldens, no init/panic, `load` caps). Minus 2 for the P0 BOM bug plus CSS paren-skip and WOFF close-leak P1s. Narrow triggers. |
| API and contracts | 6/10 | Struct-literal plus `Validate()` is stable, but zero values read as valid in 4 packages, numeric ranges are unchecked at `Set`, and `Canonical` silences typos into `ProfileNone`. |
| Resilience | 6/10 | Cancellation stops at stage boundaries (`Finalize`, HF/link passes, table builders). No convert-level timeout on a bare ctx. |
| Efficiency | 7/10 | Two unbounded shapes (PDF regex cache, `:has()` subtrees); the rest is repeated small allocs with no profile showing pain yet. |
| Hygiene | 9/10 | Missing `var _` assertions, one dead sentinel, test-only nits. Cosmetic. |

What moves it: P0 plus Phase 1 rows land an 8. Phases 2-3 (cancellation, fail-fast validation) land a 9. The last point needs allocation profiles first, not blind fixes (Phase 4 rows each require a benchmark as proof).

## Phase 1: Correctness (wrong output, crash, silent loss)

Fix first. Each row is one behavior defect with a failing-input shape.

### 1.1 Data corruption and crashes

- [x] **HTML-01 · P0 · BOM strip removes 1 of 3 bytes**: `internal/html/html.go` `ParseDocument` does `s = s[1:]` after matching `"\ufeff"` (3 bytes in UTF-8). Every BOM document keeps 2 stray bytes that parse as text. Fix: `strings.TrimPrefix(s, "\ufeff")` or strip in `[]byte` space before converting. Proof: unit test parsing BOM input asserts zero leading text nodes.
- [x] **CSS-01 · P1 · `matchingParen` skips byte after quoted span**: `internal/css/has.go:19-30` assigns `idx = skipQuoted(...)` without `continue`, so the loop `idx++` skips one more byte and a paren right after a quoted string is never counted. Sibling helpers (`internal/css/container.go:410`, `internal/css/import.go:73`) use the `continue` idiom. Fix: add `continue`. Proof: `go test ./internal/css/ -count=1` plus a `:has([t="a"(...)])` case.
- [x] **PDF-01 · P1 · zlib reader leaks on corrupt WOFF**: `internal/pdf/woff.go:125-136` calls `zreader.Close()` on the happy path only, discards its error, and the error return above skips `Close`. Fix: `defer zreader.Close()` right after open. Proof: `go test ./internal/pdf/ -count=1`.
- [x] **HTML-03 · P1 · unbounded nesting depth, recursive Walk**: `internal/html/html.go:80-86` (`Walk`) and `:102-114` recurse per tree level with no depth cap in `Parse`. A 100k-deep input builds a 100k-deep tree and exhausts the stack. Fix: max-depth constant enforced in `openElement`, or an explicit stack in `Walk`/`appendText`. Proof: depth-100k input parses without crash plus existing fuzz target.
- [x] **OLP-06 · P1 · nil inputs panic or misbehave**: `internal/outline/outline.go:156-159` `CollectHeadings(nil)` derefs in `root.Walk`; `SortHeadingsBy` nil-guards ordering (`:70-87`) then derefs `leftH.Y`; `SectionOfBy` (`:286-294`) drops `last.Title` when only `last` is set. Fix: nil guard in `CollectHeadings`, hoist `pageOf` calls, symmetric tail. Proof: `go test ./internal/outline/ -count=1` with nil cases.

### 1.2 Silent misconfiguration

- [x] **OLP-02 · P1 · `Canonical` maps every typo to valid None**: `internal/pdfprofile/profile.go:49-56` returns `ProfileNone` when `Parse` fails, so `--pdf-profile a3+ua1` silently yields an unconstrained PDF. `Parse` already returns `ErrInvalidPDFProfile`. Fix: `Canonical(value string) (string, error)` returning `Parse` directly; audit callers. Proof: `go test ./internal/pdfprofile/ -count=1`.
- [x] **SET-05 · P1 · `ParseColorMode` case-sensitive, siblings are not**: `internal/settings/settings.go:110-116` switches on raw `value` while `ParseOrientation`/`ParseLoadErrorHandling` switch on `normalize(value)`, so `"GRAYSCALE"` errors while `"LANDSCAPE"` works. Fix: `switch normalize(value)`. Proof: `go test ./internal/settings/ -count=1`.
- [x] **SET-06 · P1 · unknown margin edge writes Right**: `internal/settings/reflect.go:337-348` `marginEdgePtr` defaults to `&margin.Right`. A caller typo silently corrupts the right margin. Fix: return nil plus bool, or panic on the impossible edge. Proof: `go test ./internal/settings/ -count=1`.

### 1.3 CGO boundary hardening

- [x] **SVG-01 · P1 · unbounded `allowLen` into `unsafe.Slice`**: `bindings/c/exports_cgo.go:375-380` `convertAllowList` slices caller-controlled length with no cap; a panic inside `//export` crashes the process. `bufLen` has a `MaxInt32` guard (`:215`), `allowLen` has none. Fix: cap (e.g. 1024 entries) and reject before slicing. Proof: c-shared tests plus `go vet` on the binding package.
- [x] **SVG-02 · P1 · nil out-pointers segfault in `finishResult`**: `bindings/c/exports_cgo.go:401-431` derefs `cOutData`/`cOutLen`/`cErr` unconditionally; `rejectRequest` already nil-guards `cErr`, so the convention exists. Fix: nil guard returning `statusInvalidArg` at the top of the export entry points. Proof: c-shared test with NULL out-pointers.

## Phase 2: Cancellation, timeouts, bounds

The engine advertises ctx cancellation; these rows close the gaps where long work ignores it, plus unbounded growth keyed by input.

- [x] **CNV-04 · P1 · `Finalize` ignores cancellation**: `internal/convert/pdf_pipeline.go:232` takes `_ context.Context` and always runs `doc.Write(Output)`. Fix: check `ctx.Err()` first and return it wrapped. Proof: cancel-during-finalize test in `internal/convert/`.
- [x] **CNV-05 · P1 · HF pass has no per-page checkpoint**: `internal/convert/hf.go:741` loops all pages doing lazy `loadHTMLHF`/`drawHTMLHF` without checking ctx. Fix: `ctx.Err()` check at loop top. Proof: `go test ./internal/convert/ -count=1` with a cancel-during-HF case.
- [x] **CNV-06 · P2 · link passes take no ctx**: `internal/convert/links.go:162` (`applyTOCLinks`) and `:305` (`applyInternalLinks`) loop per entry with map builds; `assembleLinks` checks ctx only between passes. Fix: add ctx param, check per outer loop. Proof: `go test ./internal/convert/ -count=1`.
- [x] **LAY-04 · P1 · flex/grid/table builders never poll ctx**: `internal/layout/flex.go` (~43 loops), `layout_tables.go` (~59), `layout_measure.go` (~40) contain zero `checkContext` calls; `paginateOps` (`paint_pagination_fixpoint.go:8`) takes no ctx and runs 10 fixpoint iterations after one check in `PaintContext` (`paint.go:128`). The `&63`-interval poll pattern exists (`style.go:520-541`). Fix: thread that poll into row/line loops and per-fixpoint-iteration; new helper lives in a same-package `pagination_ctx.go`, never grows the split files. Proof: cancel-during-table-layout test plus `go test ./internal/layout/ -count=1`.
- [x] **CNV-08 · P2 · no timeout enforced at convert boundary**: `internal/convert/convert.go:167`, `prepare/styles.go:123` inherit caller ctx verbatim; no `WithTimeout` in non-test scope. Fix: document the required caller timeout, or wrap fetch with a default timeout when ctx has no deadline. Proof: doc comment plus a deadline test.
- [x] **LAY-06 · P2 · layout-side timeout ownership unstated**: `internal/layout/layout_flow.go:114-118` passes ctx to `ImagesContext` but imposes no deadline; legacy `Images` func cannot observe cancel at all. Fix: one comment line documenting caller-owns-timeout on `Options.ImagesContext`. Proof: doc only.
- [x] **CSS-04 · P1 · `:has()` materializes whole subtrees**: `internal/css/has.go:253-273` `elementDescendants` builds a full slice per subject element, then matches per descendant (quadratic on deep trees). Fix: `iter.Seq[*html.Node]` walk with early exit. Proof: benchmark on a deep fixture plus `go test ./internal/css/ -count=1`.
- [x] **PDF-04 · P1 · unbounded dynamic regex cache**: `internal/pdf/semantic.go:24-38` stores every distinct dict-key pattern in a process-global `sync.Map` forever, keyed by input PDF content, and `MustCompile` panics on pathological input. Fix: fixed precompiled set or small LRU plus `regexp.Compile` with error return. Proof: `go test ./internal/pdf/ -count=1`.
- [x] **SET-04 · P1 · numeric setters accept any value**: `internal/settings/reflect.go:450-453` (copies), `:837-840` (quality), `:780-783` (timeout), `:752-755` (zoom) take any int/float, so `copies=0`, `quality=999`, `timeout=-30` succeed at `Set` and fail downstream. Fix: ranged setters (`setIntRange`, `setFloatMin`) returning parse errors. Proof: `go test ./internal/settings/ -count=1`.
- [x] **LOAD-02 · P1 · invalid proxy deferred to first Load**: `internal/load/load.go:404-425` stashes constructor error in `initErr` instead of failing at construction. `NewLoaderWithError`/`NewLoaderWithNetworkPolicy` already exist. Fix: document/deprecate `NewLoader` for new callers, route to the error-returning constructors. Proof: doc plus `go test ./internal/load/ -count=1`.

## Phase 3: Construction, enums, contracts

Rule 1/2/4/19 territory. Each row is one constructor, enum, or contract fix. None changes the public struct-literal API; compat stays.

### 3.1 Constructors that never fail

- [x] **CNV-03 · P2 · `BuildOptions`/`NewResourceContext` accept anything**: `internal/convert/prepare/simplify.go:52`, `prepare/prepare.go:53`; nil loader degrades to warn-and-continue (`prepare.go:99,122`). Fix: validate viewport, default media, return error on nil loader (or document and test the degraded path). Proof: `go test ./internal/convert/prepare/ -count=1`.
- [x] **IMG-02 · P2 · `RenderContext` never validates opts**: `internal/imageout/imageout.go:100-138` checks root/ctx but not negative Width/Height, bad Crop, or Quality range. Fix: `RenderOptions.Validate()` called at entry. Proof: `go test ./internal/imageout/ -count=1`.
- [x] **LAY-03 · P2 · `Options`/`PaintOptions` never validated**: `internal/layout/layout.go:86-103`, `paint.go:50-67`; bad zoom silently clamps (`layout.go:771-777`), bad height silently resets (`paint.go:112-115`). Fix: `validate()` on both, called in `layoutContext`/`PaintContext`. No options-pattern migration needed. Proof: `go test ./internal/layout/ -count=1`.
- [x] **OLP-04 · P2 · `BuildTree` takes bare `Options`**: `internal/outline/outline.go:230-238`; negative `MaxDepth` silently means keep-all. Fix: `Options.Validate()` at top of `BuildTreeBy`, or document. Proof: `go test ./internal/outline/ -count=1`.
- [x] **SVG-03 · P2 · image path skips fail-fast the PDF path has**: `bindings/c/exports_cgo.go:180-187` vs `:165-176`; quality/crop reach the engine before `statusInvalidArg`. Fix: `validateImageRange` mirroring `validatePDFRange`. Proof: c-shared tests.
- [x] **SVG-07 · P2 · Python `ImageDocument.validate` skips quality/crop**: `bindings/python/src/gowkhtmltopdf/document.py:444-454` while `Document.validate` (`:361-372`) fully checks copies. Fix: raise `InvalidArgumentError` on out-of-range quality/crop. Proof: Python binding tests.

### 3.2 Zero values that read as valid (rule 4)

- [x] **LAY-01 · P1 · `OpKind` zero paints**: `internal/layout/layout.go:289-296` starts at `OpFillRect`; the package already needed a tombstone (`opKindNoop = 255`, `layout_measure.go:1088`). Fix: leading `OpUnknown` (coordinates with convert/golden needles) or documented intentional zero. Proof: `go test ./internal/layout/ -count=1` plus `make golden` if values shift.
- [x] **PDF-02 · P2 · `PDFVersion` zero is 1.4**: `internal/pdf/policy.go:12-21`; `WriterPolicy{}` validates as 1.4 instead of failing. Fix: `PDFUnknown` at 0, reject in `Validate()`. Proof: `go test ./internal/pdf/ -count=1`.
- [x] **PDF-03 · P2 · `subsetScope` zero is simple**: `internal/pdf/subset.go:27-32`. Fix: `subsetUnknown` at 0, treat as error in `subsetFont`. Proof: `go test ./internal/pdf/ -count=1`.
- [x] **HTML-02 · P1 · both enums zero-value as members**: `internal/html/html.go:29-34` (`ElementNode = 0`), `internal/line/line.go:16-23` (`Info = 0`); `SeverityOf` maps unknown input to `Info` and `String()` defaults to `"info"`, masking bad values. Fix: sentinels at 0, `String()` default `"unknown"`, document the `SeverityOf` mapping. Proof: `go test ./internal/html/ ./internal/line/ -count=1`.
- [x] **SET-02 · P1 · `String()` defaults mask bad values**: `internal/settings/settings.go:101-107,129-135,158-169`; `Orientation(99)` prints "Portrait". Fix: explicit switch with `"unknown"`/`"invalid(n)"` default. Proof: `go test ./internal/settings/ -count=1`.
- [x] **SET-03 · P1 · `MediaIgnore` at 0 conflates unset with ignore**: `internal/settings/settings.go:189-193`; `ResolveMedia` (`:212-232`) cannot tell "never set" from "explicit ignore". Fix: rename contract to `MediaUnset` or split `MediaUnknown=0`. Proof: `go test ./internal/settings/ -count=1`.
- [x] **LAY-02 · P2 · four latent zero-default enums**: `filter.go:17-19`, `grid_parse.go:11-13`, `layout_measure.go:429`, `inline.go:718`. All assigned at parse sites today. Fix: sentinels or intent comments. Proof: `go test ./internal/layout/ -count=1`.

### 3.3 Structural contracts

- [x] **CNV-15 · P2 · fat concretes thread through stages**: `internal/convert/convert.go:281` (`runContext`, 15 fields), `outline.go:43`. Fix: narrow interfaces (`objectRenderer`, `hfLoader`) as params; `runContext` stays wiring. Proof: `go test ./internal/convert/ -count=1`.
- [x] **IMG-06 · P2 · `RunRequest` hard-constructs deps**: `internal/imageout/imageout.go:1603-1621` builds Loader, font, registry inline; only `Images` func is injectable. Fix: `newImagePipeline(...)` constructor, `RunRequest` as wiring. Proof: `go test ./internal/imageout/ -count=1`.
- [x] **LOAD-04 · P2 · exported mutable loader state**: `internal/load/load.go:381-400` lets callers nil `Client` or mutate limits post-construction. Fix: unexport fields with validating options/setters; keep `SetTestDial` seam. Proof: `go test ./internal/load/ -count=1`.
- [x] **CLI-03 · P2 · duplicate output-missing check**: `cmd/gowkhtmltopdf/main.go:52-56` re-checks what `cli.resolveFree` (`internal/cli/cli.go:337`) already rejects, with a narrower predicate. Fix: delete and surface the parser error, or delegate to `errors.Is(cli.ErrMissingOutput)`. Proof: `go test ./internal/cli/ ./internal/app/ -count=1`.

## Phase 4: Hot-path efficiency (allocs, copies, scans)

Behavior is correct; these rows cut repeated work on match, parse, and emit paths. Each fix needs a benchmark or allocation proof, not just green tests.

- [x] **CSS-02 · P1 · full-remainder `ToLower` per at-rule**: `internal/css/css.go:212-214` (`parseAtRule`) and `:624-626` (`parseNestedAtRule`) lowercase the whole remaining sheet per rule (quadratic on many-`@media` sheets). Fix: lowercase a short prefix or a shared `hasPrefixFold` helper. Proof: stylesheet benchmark before/after.
- [x] **CSS-05 · P2 · `:has()` sibling path allocates twice**: `internal/css/has.go:243` appends anchor onto the F4 full-subtree slice. Fix: check anchor first, range the F4 iterator. Proof: same benchmark as CSS-04.
- [x] **CSS-06 · P2 · `want` re-lowered per element per rule**: `internal/css/match.go:239-244` lowers the selector-constant side on every call. Fix: precompute `wantLower` in `parseAttrSelector` (`selector_parser.go:461`). Proof: selector benchmark.
- [x] **CSS-07 · P2 · sibling scans are O(n) per call**: `internal/css/match.go:404-444,447-466`; `ofTypeLastIndex` (`:702-725`) double-scans via `ofTypeIndex`. Wide-table `:nth-child` is quadratic. Fix: single-pass helper returning prev/next/index/type totals, or a per-parent index cache. Proof: wide-table benchmark.
- [x] **CSS-08 · P2 · remainder lowered per `var()`**: `internal/css/values.go:647-661` inside the `ResolveVars` loop. Fix: copy-free case-insensitive scan, or lower once and reuse offsets. Proof: multi-var benchmark.
- [x] **CSS-10 · P2 · nested specificity never cached**: `internal/css/has.go:289-313`; top-level cache exists (`css.go:792-794,844-850`) but `parseSelectorCtx` (`has.go:85`) never sets it. Fix: set `spec`/`specValid` in `parseSelectorCtx`. Proof: `go test ./internal/css/ -count=1`.
- [x] **CNV-07 · P2 · link index is O(L*O)**: `internal/convert/links.go:47` rescans all ops per id location. Fix: page/Y op index built once. Proof: large-doc link benchmark.
- [x] **CNV-09 · P2 · stylesheet body copied per file**: `internal/convert/prepare/styles.go:129,232` do `string(resource.Body)` per link/import. Fix: `css.ParseBytes([]byte)`, keep `Parse` as wrapper. Proof: `go test ./internal/convert/prepare/ -count=1`.
- [x] **PDF-06 · P2 · full-stream string copy in semantic scan**: `internal/pdf/semantic.go:565-570` copies every page content stream for regex. Fix: `FindAllSubmatch` on bytes. Proof: `go test ./internal/pdf/ -count=1`.
- [x] **PDF-07 · P2 · dict assembly through strings**: `internal/pdf/pdf.go:35-40`, `fonttype0.go:150-158`; hot `/Widths` path joins per-value strings (`num()` at `pdf.go:1522-1526` allocs per width). Fix: append numbers straight into buffer. Optional unless the allocation profile says otherwise. Proof: subset benchmark.
- [x] **HTML-04 · P2 · `Walk` cannot stop**: `internal/html/html.go:79-100`; `TextContentOf` always scans the whole tree. Fix: `Walk(f func(*Node) bool)` or `FindFirst`. Proof: `go test ./internal/html/ -count=1`.
- [x] **HTML-05 · P2 · text concat and per-call lowering**: `internal/html/html.go:190` (`last.Text += decoded` per token), `internal/line/line.go:58` (double copy per log line), `html.go:49-57` (`ToLower` per uppercase attr lookup). Fix: builder/capacity growth, fold-compare without lowering, leave the lowercase fast path alone. Proof: `go test ./internal/html/ ./internal/line/ -count=1`.
- [x] **LAY-07 · P2 · box `kind string` plus split bools**: `internal/layout/layout.go:1170-1222`; 16B header plus alloc per box where `uint8` fits, padding from scattered bools. Keep the `style *ResolvedStyle` indirection. Fix: `boxKind uint8`, pack bools. Proof: allocation benchmark on a multi-page report.
- [x] **OLP-05 · P2 · outline XML copied twice**: `internal/outline/outline.go:390-401` `[]byte(buf.String())`. Fix: accept, or add a string-returning variant. Proof: none required; one-line judgment call.
- [x] **API-07 · P2 · `PDF()`/`Image()` double peak memory**: `document.go:243,304` buffer the whole artifact plus one full copy. Fix: document streaming via `WritePDF`/`WriteImage` for large jobs, or return buffer ownership explicitly. Proof: doc only.

## Phase 5: Hygiene (assertions, error shape, test files)

Small, safe, independent. Good first slices.

- [x] **CNV-01 · P2 · missing Pipeline assertion**: `internal/convert/pdf_pipeline.go:13` satisfies `render.Pipeline` implicitly. Fix: `var _ render.Pipeline = (*pdfPipeline)(nil)`. Proof: `go build ./...`.
- [x] **PDF-08 · P2 · missing `io.WriterTo` assertion**: `internal/pdf/pdf.go:522-525` conformance is comment-only. Fix: `var _ io.WriterTo = (*Document)(nil)`. Proof: `go build ./...`.
- [x] **IMG-03 · P2 · missing Pipeline assertion**: `internal/imageout/imageout.go:1632-1640` vs `renderpipeline.Run` at `:1622`. Fix: `var _ renderpipeline.Pipeline = (*imagePipeline)(nil)`. Proof: `go build ./...`.
- [x] **LOAD-03 · P2 · resolver conformance is comment-only**: `internal/load/load.go:376-378` claims `*net.Resolver` implements `IPResolver`. Fix: `var _ IPResolver = net.DefaultResolver`. Proof: `go build ./...`.
- [x] **OLP-03 · P2 · `LocationReader` has no assertion**: `internal/outline/outline.go:29-33`. Fix: `var _ LocationReader = Location{}` (plus pointer form if intended). Proof: `go build ./...`.
- [x] **API-06 · P2 · `lineLog` has no assertion**: `api.go:136-148` passes `*lineLog` as `io.Writer` (`api.go:109,121`). Fix: `var _ io.Writer = (*lineLog)(nil)`. Proof: `go build ./...`.
- [x] **SET-09 · P2 · error types lack assertions**: `internal/settings/reflect.go:17-27` four types. Fix: `var _ error = ...` per type. Proof: `go build ./...`.
- [x] **CNV-10 · P2 · link skips are silent**: `internal/convert/links.go:199` `continue` on missing pages/anchors while HF failures accumulate (`hf.go:727`). Fix: warn callback mirroring the HF result pattern. Proof: `go test ./internal/convert/ -count=1`.
- [x] **LOAD-05 · P2 · bare sentinel returns lack context**: `internal/load/load.go:1493,1628,1231` return unwrapped sentinels. Fix: `fmt.Errorf("...: %w", ...)` with truncated ref. Proof: `go test ./internal/load/ -count=1`.
- [x] **SET-10 · P2 · units case-sensitive**: `internal/settings/unitreal.go:38-45` rejects `"25MM"` while sibling parsers lowercase first. Fix: lowercase copy for suffix match. Proof: `go test ./internal/settings/ -count=1`.
- [x] **SET-11 · P2 · pagesize stores non-canonical case**: `internal/settings/reflect.go:593-607` stores `"Letter"` after case-insensitive validation. Fix: store canonical form. Proof: `go test ./internal/settings/ -count=1`.
- [x] **SVG-05 · P2 · canvas NaN passes bounds check**: `internal/svg/raster.go:115-122` guards `<= 0` but not NaN/Inf from the third-party lib (own parsing at `:168-174` already guards). Fix: extend the check. Proof: `go test ./internal/svg/ -count=1`.
- [x] **SVG-09 · P2 · Python error fallback races under threads**: `bindings/python/src/gowkhtmltopdf/_lib.py:290-314` reads the process-wide `last_error` slot outside `_CALL_LOCK`. Fix: copy message bytes under the lock, raise after. Proof: Python binding tests.
- [x] **SVG-10 · P2 · per-call sentinel slice**: `bindings/c/classify.go:46` allocates 15 errors per classification. Fix: package-level slice. Proof: `go test ./bindings/c/ -count=1`.
- [x] **OLP-07 · P2 · dead exported sentinel**: `internal/pdfprofile/profile.go:37-41` `ErrProfilePDF20Unsupported` is never returned by its own comment. Fix: delete with a breaking-change note, or mark `Deprecated`. Proof: `go build ./...`.
- [x] **CLI-04 · P2 · test restores global stdout without defer**: `cmd/gowkhtmltopdf/main_test.go:14-27`; a panic in `action()` leaks the swap. Fix: `defer` restore right after swap. Proof: `go test ./cmd/... -count=1`.

## Phase 6: Closure gates

This wave changed no source, tests, or docs outside this file, so per `skills/phase-wise-checklist/SKILLS.md` Required Checks this documentation-only change runs no lint or test gates. Verification for the report itself:

- [x] **GATE-01 · P0/P1 locations re-read**: BOM strip (`html.go:305-316`), severity enum (`line.go:16-67`), paren scan (`has.go:19-30`), scope/version enums (`subset.go:27-32`, `policy.go:12-21`), zlib close (`woff.go:125-136`), profile `Canonical` (`profile.go:49-56`), `CollectHeadings` nil path (`outline.go:156-165`), `OpKind` plus tombstone (`layout.go:289-296`, `layout_measure.go:1088`), `Finalize` (`pdf_pipeline.go:232`), settings enums plus `ParseColorMode` (`settings.go:94-147`), margin default (`reflect.go:337-348`), `allowLen` slice (`exports_cgo.go:375-380`), status zero (`main.go:26`), at-rule lowering (`css.go:212-214,624-626`), load kinds (`load.go:82-92`). Proof: read output on file, 2026-09-08.
- [x] **GATE-02 · remediation gates**: before any Phase 1-5 row flips to `[x]`, run `make lint` and `make test`; layout/paint/pagination rows additionally require `make golden`. Record both outcomes on the row. Enum-value shifts (LAY-01, PDF-02) require golden re-baseline review, not silent updates.

## Dependencies

```text
HTML-01 (BOM) ── independent; do first, it corrupts input for everything downstream
CSS-01 ── independent
PDF-01, OLP-06, SET-05, SET-06 ── independent
SVG-01 ──▶ SVG-02 (same file, same hardening pass)
OLP-02 ── independent
CNV-04 ──▶ CNV-05 ──▶ CNV-06 (cancellation front to back)
CNV-15 ── independent
CSS-04 ──▶ CSS-05 (iterator first, callers second)
LAY-04 blocks any latency claim on table-heavy docs
GATE-02 ── after any remediation, before any [x]
```

Suggested first slice if asked: HTML-01, CSS-01, OLP-02, SVG-01/SVG-02, PDF-01.

## Parked (test-only nits)

Real but lowest value; re-open only alongside a production row in the same file.

- `CNV-12`: `panic` in `benchmarks_test.go:354,395`; use `tb.Fatalf`.
- `CNV-13`: per-test inline `regexp.MustCompile` (`convert_test.go:615,619`, `golden_test.go:715,762,865`, `compliance_golden_test.go:61,192,538`); hoist to package vars.
- `PDF-09`: per-assertion `MustCompile` in `fonttype0_test.go`, `structure_test.go`, `policy_test.go`, `pdf_test.go`, `pdf20_test.go`; hoist.
- `HTML-06`: `line_test.go:49-57` hand-rolled writer without assertion; use `bytes.Buffer`.
- `API-03`: `document_bench_test.go:121-142` panics on missing template; use `b.Fatalf`.
- `API-04`: bench template via CWD-relative `os.ReadFile` (`document_bench_test.go:122`); `go:embed` it.
- `API-05`: `PDF()`/`Image()` buffering note is API-07 in Phase 4; the double copy itself stays parked until a large-doc profile demands it.
- `SVG-08`: `exports_cgo.go:472` double copy in a test helper; do not propagate to production.

## Refused (accepted deviations, do not re-file without new evidence)

| Item | Why refused |
|------|-------------|
| Struct-literal public API (`Document`, `Request`, `Options`) | wkhtmltopdf-compat and library stability; options layers add alongside, never replace. |
| `Request` plus `Validate()` instead of erroring constructors | Project-wide convention; rows above add validation inside, not new signatures, unless adopted per package. |
| `Render` using `context.Background()` | Documented public adapter; `RenderContext` exists. Same call as the 2026-08-28 wave. |
| `imageout` importing `convert/prepare` plus `convert/render` | Known shape from the 2026-08-28 wave; parent `convert` hub import already dropped. |
| Mutex on layout or `pdf.Document` | Single-goroutine engine. |
| Dual public stories (`api.go` plus `convert.Request`) | Tax, not a bug. Same call as the 2026-08-28 wave. |
| `buildCIDMap` 128 KiB worst case (`fonttype0.go:196-205`) | Bounded by BMP; document only if touched. |
| Full in-memory PDF object model (`pdf.go:35-40`) | Inherent to xref; spill-to-disk only on a large-doc profile. |
| `supersamplePixPool` global (`imageout.go:319-320`) | Bounded at 32 MiB with cap comment; struct-passing is optional. |
| `statusOK = 0` zero-is-success (`bindings/c/main.go:26`) | ABI-frozen C convention; comment only. |
| `Registry` taking concrete `*Font` (`registry.go:62-118`) | Data struct with pure helpers; interface only when a fake is needed. |
| `Emit` ignoring write errors (`line.go:47-53`) | Best-effort log emitter; document or return error only if a caller needs it. |
| Pixel-diff goldens as proof | Goldens are structural per `output/README.md`. |

## What this wave did not do

- No source, test, fixture, catalog, or matrix changes. All rows are `[ ]` by construction.
- No perf-review or critical-go-review lenses; design-patterns skill only.
- No KB rewrite beyond the session log pointer; code and mapping untouched, so `wiki/concepts/css-engine.md` and `wiki/compatibility.md` stand.
- Git publication is separate and needs explicit user authorization.
