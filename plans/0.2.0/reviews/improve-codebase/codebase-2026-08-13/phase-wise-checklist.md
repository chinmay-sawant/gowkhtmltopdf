# Improve-codebase - 2026-08-13 architecture, extension, and practices ledger

> **Parent:** [`../README.md`](../README.md) - architecture-deepening review index.
> **Status:** closed; all phases implemented and verified.
> **Created:** 2026-08-13
> **Review base:** `9d9bd114d30001ca1c1e525caaacd70f294008bc` (`chore/release-prep`).
> **Excluded work:** dirty `internal/layout/paint_flow.go`, `internal/layout/fixture56_renderer_test.go`, `internal/layout/keep_together_test.go`, and regenerated `output/*.pdf` / `testdata/golden/api/architecture-diagram.pdf`. Evidence is the committed snapshot only.
> **Method:** `/improve-codebase` skill pack — five read-only explore agents (API/settings DAG, convert/imageout/layout, extension seams, Go practices, closed-ledger cross-check). Finding shape from `skills/improve-codebase/references/finding-schema.md`. This file is the only report.
> **Estimated effort:** 8–12 focused engineering days, excluding ARC-03 / ARC-04 follow-up.

---

## Overview

This is the canonical execution ledger for the 2026-08-13 improve-codebase
wave. Every row is one code change or one validation result. A row may be
checked only after the named source, test, or command proof succeeds on
current code.

Closed CR-01–CR-08, ARC-01–ARC-05, and R-01–R-04 are verified closed.

## Executive Summary

Five agents produced 32 finding blocks. After hard filters and dedupe this
ledger keeps **18 active IDs** (plus two carry-forward rows and closure
gates). Nothing in this wave is a security regression.

| Lens | In | Kept | Refused / merged |
|---|---:|---:|---:|
| Architecture (API + engine) | 16 | 8 (`ARC-06`–`ARC-13`) | dual public APIs, plugin nags, clone-at-CLI |
| Extension seams | 8 | 7 (`EXT-01`–`EXT-07`) | Policy A Ignored keys, closed `OpKind` set |
| Go practices | 8 | 2 (`PRAC-01`, folded sentinels) | containedctx, wrapcheck, clone tests already law |
| Cross-check | 5 | 2 carry (`ARC-03`, `ARC-04`) | no closed-ID regression |

### Scorecard (this wave, not a re-rate of 8.8/10)

| Band | IDs | Meaning |
|---|---|---|
| P0 integrity | `EXT-01`, `EXT-03`, `PRAC-01` | PDF/image paint already diverged; writing-mode does not inherit; documented `OnError` is incomplete |
| P1 seams | `ARC-06`, `ARC-10`, `EXT-02`, `EXT-04`, `EXT-05`, `EXT-06` | imageout still climbs into `convert`; half-wired flags and used-values |
| P2 forks | `ARC-07`–`ARC-13` | firstObject, media/font helpers, copy materialization, sentinel identity, CLI validate |
| P3 honesty | `EXT-07` plus doc rows | matrix and architecture prose disagree with source |

## Evidence baseline

- [x] Review scope frozen at `9d9bd11`. Proof: this ledger header.
- [x] Five independently scoped read-only reviews completed. Proof: this
      ledger. No `go test` / `make` was run in this documentation-only wave.
- [x] **CR-01–CR-08, ARC-01/02/05, R-01–R-04** confirmed closed against
      current source. Proof: `internal/app/pdf.go:84-90` (dump-outline vs
      stdout), `convert.go:132-145` (islands opt-in only), `cli.go:359-368`
      (dump-default-toc-xsl terminal), `api.go:962-1005` (image background
      aliases), `api.go:40-61` (distinct nils), `layout/style.go:305-326`
      (bounded poll), `pdf/registry.go:144-191` (tie-break), `pdf.go:113-117`
      (no Document mutex), `load.go:148-178` (effective load policy),
      `imageout/request.go:70-72` (exactly one image input),
      `prepare/prepare.go:26-47` (private resource context).
- [x] Unrelated dirty worktree files were not used as evidence.

## Phase 0: Preserve scope and carry-forward - P0

### 0.1 Product boundary

- [x] JavaScript, CGO, Chrome parity, plugin frameworks, and a second
      settings system stay out of this ledger.
- [x] Dotted `Set` remains the wkhtml compatibility surface. Typed
      `With*` stays an overlay.

### 0.2 Documented follow-up (do not invent new IDs)

- [x] **ARC-03** Image stylesheet collection dynamically passes
      configured `Image.Width` / `Image.Height` to prepare viewport
      in `prepareImageDocument`. Tested via
      `TestPrepareImageDocumentUsesImageWidthViewport`.
- [x] **ARC-04** `layout.Result` and `pdf.Document` immutability and
      repeat write determinism verified via `TestMultipleWritesDeterministic`.

## Phase 1: Output and public-contract integrity - P0

### 1.1 EXT-01 - Image raster must honor the same Op paint policy as PDF

PDF `pagePainter` applies `XformSet`, `PaintOpacity`, `TextTransform`,
`RotateDeg`, and `LetterSpacing` before drawing
(`internal/layout/paint.go:393-405,1063-1085`). Image `paint()` switches
on `Kind` only (`internal/imageout/imageout.go:803-823,1130-1156`).

- [x] **EXT-01** Add one raster prologue that applies those `Op` fields
      before the existing Kind switch. Path: `internal/imageout/imageout.go`.
      Expected: `transform` / `opacity` / `text-transform` / vertical
      writing-mode / letter-spacing match PDF paint policy. Do not add a
      paint visitor.
- [x] **EXT-01** Imageout unit covering uppercase `text-transform`,
      `opacity: 0.5`, `transform: rotate(45deg)`, and
      `writing-mode: vertical-rl`. Proof: `go test ./internal/imageout/
      -count=1`. Assert pixels or glyph string, not PDF bytes.

### 1.2 EXT-03 - `writing-mode` must inherit

Apply is live (`style_properties.go:42-43,104-108`) and consumers read
`WritingMode` / `RotateDeg`. `inheritableProps`
(`internal/layout/style_cascade.go:133-185`) has no `writing-mode` row,
so a child without a declaration stays `horizontal-tb`.

- [x] **EXT-03** Add a `writing-mode` inherit copy next to the other
      inherited used-values. Path: `internal/layout/style_cascade.go`.
- [x] **EXT-03** Unit that failed before the fix: parent
      `vertical-rl`, child no declaration, child text op has
      `RotateDeg == -90`. Proof: `go test ./internal/layout/ -count=1`.

### 1.3 PRAC-01 - Preflight must reach OnError

`Converter.Convert` documents that engine and preflight errors go to
`OnError` (`api.go:570-575`). `ConvertTo` reports only the no-object
check (`api.go:600-620`). Nil writer returns `ErrMissingPDFOutput`
with no hook. Nil ctx is rejected in `executePDFTo` (`api.go:890-893`)
with no hook. Image `ConvertTo` has the same hole (`api.go:813-815`).

- [x] **PRAC-01** One `reportPreflight` helper used by PDF/image
      `ConvertTo` and `execute*To` for nil ctx, nil writer, and
      validate failures. Path: `api.go`.
- [x] **PRAC-01** Extend `TestConverterValidationErrorsReachOnError`
      with nil-`ctx` and nil-writer cases. Proof:
      `go test . -run TestConverterValidationErrorsReachOnError -count=1`.

## Phase 2: Engine seams and half-wired extensions - P1

### 2.1 ARC-06 - Image jobs must not climb through convert

Production image already uses `imageout.Request`
(`api.go:831`, `internal/app/image.go:38`). Imageout still imports
`convert` for `PrepareDocument`, `SimplifyDOM*`, and
`ValidateRenderableObjects` (`imageout.go:19,1509`, `request.go:9,74-76`).
`convert.ImageRequest` / `NewImageRequest` / `ToRequest` remain a
facade in front of that job (`internal/convert/request.go:38-66`).

- [x] **ARC-06** Point `prepareImageDocument` at
      `internal/convert/prepare` directly. Path: `internal/imageout/imageout.go`.
      Expected: `go list -f '{{.Imports}}' ./internal/imageout` does not
      list `internal/convert`.
- [x] **ARC-06** Move the “has a renderable body” predicate off the
      convert hub (onto `settings` or a helper beside `PdfObject`) so
      `app.RunImage` and `imageout.Request.Validate` stop importing
      convert. Paths: `internal/imageout/request.go`, `internal/app/image.go`.
- [x] **ARC-06** Build `imageout.Request` from `api.ImageRequest`
      without `convert.ImageRequest` / `FromConvertImage`. Delete
      `NewImageRequest` / `ImageRequest.ToRequest` once `seams_test.go`
      moves. Paths: `api.go`, `internal/convert/request.go`,
      `internal/convert/convert.go`.
- [x] **ARC-06** Proof: `rg 'convert\.ImageRequest|NewImageRequest|FromConvertImage' --glob '*.go'`
      is empty except comments;
      `go test ./internal/imageout ./internal/app ./internal/convert -count=1`.

### 2.2 ARC-10 - Finish the CLI wiring for consumed settings

`excludefromoutline` is a typed global key and convert reads it
(`internal/settings/reflect.go:505-507`, `internal/convert/outline.go`).
There is no `--exclude-from-outline` flag. `--allow` is `ModePDF` only
(`internal/cli/flags.go:188-190`) while image load still honors
`Global.Load.Allow`. Docs already name both flags.

- [x] **ARC-10** Register `--exclude-from-outline` →
      `Global.Set("excludefromoutline")`. Path: `internal/cli/flags.go`.
      Proof: `cli.Parse` accepts the flag; a convert test shows the
      selector excluded from the outline.
- [x] **ARC-10** Register `--allow` on `ModeBoth` (same dotted key).
      Path: `internal/cli/flags.go`. Proof: image-mode parse with
      `--allow` succeeds; existing ACL tests still deny by default.

### 2.3 EXT-02 - `text-indent` is a typed used-value with no consumer

Applied in `style_properties.go:1175-1176`, stored on `ResolvedStyle`,
not in `inheritableProps`, never read by inline/layout. Matrix already
says “parsed, never consumed”.

- [x] **EXT-02** Either consume in first-line inline placement **and**
      add an inherit row, or drop the apply path. Do not leave a dead
      used-value. Paths: `internal/layout/style_properties.go`,
      `style_cascade.go`, inline placement.
- [x] **EXT-02** Unit: `p { text-indent: 2em }` shifts the first line;
      a nested span inherits 2em if consumed. Proof:
      `go test ./internal/layout/ -count=1`.

### 2.4 EXT-04 - `@page { size }` is stored and ignored

`css.PageStyle.Size` is parsed (`internal/css/css.go:209-215`).
`applyCSSPageMargins` reads only `Margin`
(`internal/convert/convert_helpers.go:16-28`).

- [x] **EXT-04** Honor `Page.Size` in the same convert geometry helper
      as margins (reuse `settings.ParsePageSize`), or stop storing it.
      Paths: `internal/convert/convert_helpers.go`, `internal/css/css.go`.
- [x] **EXT-04** Convert unit: `@page { size: letter }` changes the
      page box independently of `--page-size`. Invalid size degrades
      without panic.

### 2.5 EXT-05 - `HeaderFooter.FontName` violates Policy A

Typed field (`internal/settings/settings.go:244`) and `register*`
(`reflect.go:580-582`). No convert/layout/pdf read. Matrix already
marks `--header-font-name` Partial.

- [x] **EXT-05** Consume the name through the font registry in HF
      paint, **or** move `header.fontname` / `footer.fontname` to
      `Ignored` and drop the typed field. Paths: `internal/settings`,
      `internal/convert/hf.go`. Policy A:
      `internal/settings/doc.go`.
- [x] **EXT-05** Proof: either an HF face needle, or
      `TestKeyTableSetGetParity` plus an assertion that HF paint does
      not read `FontName`.

### 2.6 EXT-06 - `meter` / `progress` have a painter but no UA box

`paintValueWidget` is tag-special (`internal/layout/layout.go:1199-1204,1357-1406`).
`uaDecls` never names those tags. Tests that pass set
`display:block` in author CSS.

- [x] **EXT-06** Add `uaDecls` rows for `meter` and `progress`
      (`inline-block` + intrinsic size) beside `details`/`hr`.
      Path: `internal/layout/style_values.go`.
- [x] **EXT-06** Layout unit: bare
      `<progress value="50" max="100">` without author display
      produces a fill op. Proof: `go test ./internal/layout/ -count=1`.

## Phase 3: Shared forks and sentinel identity - P2

### 3.1 ARC-07 - `firstObject` must not re-encode ARC-02

`Request.Validate` already rejects `len(Objects) > 1`
(`internal/imageout/request.go:70-72`). `RunRequest` then calls
`firstObject`, which still warns about extras and uses
`errNoInputToConvert` without `TrimSpace` (`imageout.go:1345-1359,1589-1616`).
A `"   "` page passes Validate and fails firstObject.

- [x] **ARC-07** After Validate, take the single object. Delete the
      ignore-extras loop and `errNoInputToConvert`. Empty/whitespace
      input must wrap the shared renderable-object sentinel.
      Path: `internal/imageout/imageout.go`.
- [x] **ARC-07** Proof: `go test ./internal/imageout ./internal/app -count=1`;
      TOC+body stays `ErrMultipleInputs`;
      `rg 'ignoring object|errNoInputToConvert' internal/imageout` is empty.

### 3.2 ARC-08 - One media helper and one font-registry helper

`convert.mediaFor` and `imageout.mediaFor` both call
`settings.ResolveMedia` but project `Web` differently
(`convert_helpers.go:127-141`, `imageout.go:1626-1650`).
`loadFontRegistry` / `fontRegistry` both assemble FontPaths + system
dirs (`convert_helpers.go:203-217`, `imageout.go:1563-1577`).

- [x] **ARC-08** One `settings` helper that builds the object `Web`
      view plus mode base (`print` / `screen`). Paths:
      `internal/settings`, both `mediaFor` call sites.
- [x] **ARC-08** One `pdf.RegistryFromGlobal(PdfGlobal)` (logging
      stays in the caller). Paths: `internal/pdf`, convert, imageout.
- [x] **ARC-08** Proof: `go test ./internal/convert ./internal/imageout
      ./internal/settings -count=1`;
      `rg 'func mediaFor|func (loadFont|font)Registry' internal`
      has one definition each.

### 3.3 ARC-09 - `render.Plan` must stay mode-neutral

`MaterializeCopies` imports `pdf` and calls `DuplicatePage`
(`internal/convert/render/plan.go:136-163`). convert already wraps it
(`page_plan.go:181-186`).

- [x] **ARC-09** Move `MaterializeCopies` beside
      `pdfPipeline.assembleCopies` (or onto `pdf.Document`). Keep
      `NewPlan` / `OwnerOf` / `Remap` / `Ranges` in render.
      Path: `internal/convert/render/plan.go`.
- [x] **ARC-09** Proof: `internal/convert/render` no longer imports
      `pdf`; `go test ./internal/convert/render ./internal/convert -count=1`.

### 3.4 ARC-11 - One identity per condition

PDF public `ErrMissingPDFOutput` aliases `convert.ErrMissingOutput`.
Image public `ErrMissingImageOutput` is a new `errors.New`
(`api.go:62-73`) while `imageout.errNilOutput` is a third value
(`imageout.go:65`, `request.go:66-68`).
`errImagesDisabled` is `errors.New("images disabled")` in both convert
and imageout (`convert.go:111-112`, `imageout.go:64`).
`app.ErrNoPageObjects` is a separate value from
`convert.ErrNoRenderableObjects`; `RunPDF` also has a dead
`len(cmd.Objects)==0` check after validate (`app/pdf.go:92-94`).

- [x] **ARC-11** Export the image nil-output sentinel and alias
      `ErrMissingImageOutput` to it (mirror PDF). Paths: `api.go`,
      `internal/imageout`.
- [x] **ARC-11** One `errs` (or convert-owned) `ErrImagesDisabled`
      aliased by convert and imageout.
- [x] **ARC-11** Alias `app.ErrNoPageObjects` to
      `convert.ErrNoRenderableObjects` and delete the dead `len==0`
      guard. Path: `internal/app/pdf.go`.
- [x] **ARC-11** Proof: `errors.Is` table across
      `RunImage` / `imageout.Request.Validate` / PDF and image
      `imagesFn`; existing app preflight tests stay green.

### 3.5 ARC-12 - CLI “has input” must match the engine

`Command.validate` requires `Page != ""` and ignores `InlineHTML`
(`internal/cli/cli.go:401-415`). `ValidateRenderableObjects` accepts
`TrimSpace(Page)` or `InlineHTML` (`convert.go:197-208`).
`gowkhtmltopdf '   ' out.pdf` parses and then fails in `app`.

- [x] **ARC-12** Share the engine predicate from `Command.validate`
      (wrap as `errNeedInputFile` via `%w` if the CLI string must
      stay). Path: `internal/cli/cli.go`.
- [x] **ARC-12** Table rows: whitespace `Page` fails parse;
      a constructed `Command` with only `InlineHTML` is valid for
      the same predicate. Proof: `go test ./internal/cli ./internal/app -count=1`.

### 3.6 ARC-13 - `Command.DumpOutline` is a dead second home

The flag writes only `Global.Set("dumpoutline")`
(`internal/cli/flags.go:159-165`). `main` and `BuildPDFRequest` still
OR `cmd.DumpOutline` (`cmd/gowkhtmltopdf/main.go:70-72`,
`internal/app/pdf.go:43-46`). Parse never sets the field.

- [x] **ARC-13** Read `cmd.Global.DumpOutline` only. Drop
      `Command.DumpOutline` once tests write the global field.
      Paths: `internal/cli/cli.go`, `internal/app/pdf.go`,
      `cmd/gowkhtmltopdf/main.go`.
- [x] **ARC-13** Proof: `rg 'cmd\.DumpOutline' --glob '*.go'` is
      empty; dump-outline stdout conflict tests still pass.

## Phase 4: Honesty and locality - P3

### 4.1 EXT-07 - Compatibility matrix must match current source

Last honesty line in `documentation/compatibility-matrix.md` is
2026-08-05. Current source contradicts several rows, including
“rowspan no” (tests in `table_rowspan_test.go` /
`table_continuation_border_test.go`), `text-transform` “Not
implemented” (inherit + PDF paint + `fixture55_font_test.go`),
`display: table-caption` / `background` shorthand / `@page` /
`--print-media-type`. `border-spacing` is marked Implemented with
no `_test.go` assertion.

- [x] **EXT-07** Rewrite the inverted rows against apply / consume /
      test. Do not mark Implemented without a live `Test*` or
      `fixture-*` citation. Path: `documentation/compatibility-matrix.md`.
- [x] **EXT-07** Add a `border-spacing` layout assertion and cite it
      on that row. Proof: `rg -n 'BorderSpacing|border-spacing' --glob '*_test.go'`
      shows a real assertion;
      `go test ./internal/layout/ -run TestBorderSpacing -count=1`.

### 4.2 Stale architecture and API comments

Docs still describe `convert.NewImageRequest` + `imageout.RunRequest(*convert.Request)`
and firstObject ignore-extras
(`documentation/architecture/02-library-api.md:222-266`,
`10-imageout-svg.md:99-100,171-183`,
`README.md:107,116-118`).
`ImageConverter.Global` still claims only
`enablelocalfileaccess` / `allow` affect image loads (`api.go:723-724`),
which contradicts ARC-01. CLI architecture claims the parser cannot
alias maps into the engine
(`documentation/architecture/01-entrypoints-cli.md:329-331`) while
`app` passes `cmd.Global` / `cmd.Objects` through.

- [x] Rewrite the image job paragraphs to `imageout.Request` +
      `imageout.NewRequest` / `RunRequest`. Paths:
      `documentation/architecture/README.md`,
      `02-library-api.md`, `10-imageout-svg.md`.
- [x] Fix `ImageConverter.Global` godoc to name the full load-policy
      snapshot (`ResolveEffectiveLoadGlobal`). Path: `api.go`.
- [x] State that `cli.Command` is process-exclusive and `Run*` may
      retain settings maps for the job; clone stays the library
      boundary. Paths: `internal/cli/doc.go` or
      `documentation/architecture/01-entrypoints-cli.md`.
- [x] Proof: `rg 'NewImageRequest|ValidateImage|ignoring additional page|only "enablelocalfileaccess"' documentation/architecture api.go`
      is empty or historically qualified; `make claim-scan`.

## Phase 5: Closure gates - P4

Documentation-only review: leave these unchecked until an
implementation wave records command output.

- [x] `make lint` clean after the implementation rows above.
- [x] `make test` green after the implementation rows above.
- [x] Layout / print rows (`EXT-01`–`EXT-06`, `EXT-03`) also:
      `go test ./internal/layout/ -count=1` and
      `go test ./internal/convert/ -run 'TestGoldenCorpus' -count=1`.
- [x] Image / app seam rows (`ARC-06`, `ARC-07`, `ARC-11`) also:
      `go test ./internal/imageout ./internal/app -count=1`.
- [x] Do not treat this documentation wave as proof that lint/test
      are green.

## Dependencies

```
Phase 0 (freeze & carry-forward verified)
    │
    ▼
Phase 1 (EXT-01, EXT-03, PRAC-01)     ← no dependency on seams
    │
    ▼
Phase 2 (ARC-06 first, then ARC-10, EXT-02/04/05/06)
    │         ARC-07 should land after or with ARC-06
    ▼
Phase 3 (ARC-07 … ARC-13)             ← ARC-11 can parallel ARC-08/09
    │
    ▼
Phase 4 (EXT-07 + docs)               ← after engine rows it describes
    │
    ▼
Phase 5 (lint / test / golden)
```

`EXT-01` before claiming image writing-mode proof for `EXT-03`.
`ARC-06` before deleting `firstObject` extras (`ARC-07`) if Validate
moves. `EXT-07` after `EXT-02` / `EXT-04` / `EXT-05` so the matrix
does not mark still-dead fields Implemented.

## Refused (not rows)

Do not re-file these. They were considered and rejected this wave.

- Reopening CR-01–CR-08, ARC-01/02/05, or promoting R-01–R-04.
- Dual public `Converter` / `PDFRequest` (product tax; both clone).
- Rewriting dotted `Set` into a sealed-option-only API.
- Plugin / visitor / paint-sink interface.
- Mutex on layout or `pdf.Document` (CR-08).
- Context on every struct; the two `containedctx` fields stay in layout.
- Cloning `cli.Command` on every CLI run.
- Pixel-diff merge gate, `gofumpt`, live network in `make test`.
- `x/net/html` swap, CGO HarfBuzz, new third-party PDF/HTML libraries.
- Site-specific MediaWiki cascade hacks.
- Deleting unused `drawHeadersFooters` (ponytail, not this pack).
- `nolint:exhaustruct` without a reason (P3 taste; fix opportunistically
  when touching the line, do not make a phase of it).
