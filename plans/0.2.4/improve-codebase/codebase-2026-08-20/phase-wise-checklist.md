# Improve-codebase - 2026-08-20 architecture, extension, and practices ledger

> **Parent:** [`../README.md`](../README.md) - v0.2.4 improve-codebase index.
> **Status:** open; documentation-only review wave. No implementation started. Current rating **7.8 / 10**.
> **Created:** 2026-08-20
> **Review base:** `1a8ebf84e58ed8a370ed76aa8d61a8f8c71e2b8e` (`master`, post v0.2.4 Document API).
> **Excluded work:** dirty `.gitignore` only. Evidence is the committed snapshot.
> **Method:** `/improve-codebase-architecture` via the project `skills/improve-codebase` pack — six read-only explore agents (Document/CLI DAG, convert/imageout seam, layout/CSS/paint, extension seams, Go practices, closed-ledger cross-check). Finding shape from `skills/improve-codebase/references/finding-schema.md`. This file is the only report.
> **Prior health baseline:** [`codebase-review-2026-08-12`](../../../0.2.0/reviews/improve-codebase/codebase-review-2026-08-12/phase-wise-checklist.md) closed at **8.8 / 10** before the Document hard break.
> **Target:** 10/10 as a controlled-report renderer. Browser/JavaScript parity stays out of scope.
> **Estimated effort:** 10–14 focused engineering days, excluding ARC-03 / ARC-04 deeper follow-up.

---

## Overview

Canonical execution ledger for the 2026-08-20 improve-codebase wave after the
v0.2.4 Document / ImageDocument hard break. Every row is one code change or one
validation result. Check a row only after the named source, test, or command
proof succeeds on current code.

Closed CR-01–CR-08, ARC-01–ARC-13, EXT-01–EXT-07, PRAC-01, and R-01–R-04 from
[`plans/0.2.0/reviews/improve-codebase/codebase-2026-08-13/`](../../../0.2.0/reviews/improve-codebase/codebase-2026-08-13/phase-wise-checklist.md)
were re-checked against current source. **No engine regressions.** One Phase 4.2
docs honesty claim regressed (stale `NewImageRequest` prose) and is re-filed
here as ARC-22.

## Executive Summary

Six agents produced 55 finding blocks. After hard filters and dedupe this ledger
keeps **23 active IDs** (plus two carry-forward rows and closure gates). Cap is
25 active rows; extra P3s sit in Parked.

| Lens | In | Kept | Refused / merged |
|---|---:|---:|---:|
| Architecture (Document/CLI + engine) | 22 | 10 (`ARC-14`–`ARC-23`) | dual public Converter, CLI clone, Pipeline inflation |
| Layout / CSS / paint | 12 | 4 (`EXT-08`, `EXT-11`–`EXT-13`) | EXT-01–03/06, package-per-file, paint visitor |
| Extension seams | 5 | 5 (`EXT-09`, `EXT-10`, `EXT-14`, `EXT-15` + folds) | Policy A Ignored, closed OpKind |
| Go practices | 8 | 4 (`PRAC-02`–`PRAC-05`) | containedctx, wrapcheck, golden-as-pixel |
| Closed-ledger cross-check | 1 regression + holds | 1 (`ARC-22` docs) | all closed CR/ARC/EXT engine rows |

### Scorecard (this wave)

| Band | IDs | Meaning |
|---|---|---|
| P0 integrity | `ARC-14`, `EXT-08` | Cover HF inherits globals against migration prose; fixture accent hex gates chrome stroke width |
| P1 seams | `PRAC-02`, `EXT-09`, `EXT-10`, `ARC-15`–`ARC-17`, `PRAC-03`, `EXT-11`–`EXT-13` | Document ACL/knobs incomplete; dual cover/TOC homes; shallow PDFRequest; prepare forks; HF/thumb/paint forks; missing OnError proof |
| P2 forks | `ARC-18`–`ARC-21`, `EXT-14`, `PRAC-04`, `PRAC-05` | Font registry wrappers, dead app guard, PdfGlobalOptions, CLI ghosts, table-layout, clone/copies proof |
| P3 honesty | `EXT-15`, `ARC-22`, `ARC-23` | Matrix + architecture docs disagree with source |

### Rating — **7.8 / 10**

Equal-weight blend under the controlled-report product ceiling (same five
dimensions as the 2026-08-12 health ledger). Scores are the live tree at
`1a8ebf8`, not a forecast after this checklist lands.

| Dimension | Prior (8.8 wave) | Now | Notes |
|---|---:|---:|---|
| Architecture and seams | 9.0 | 7.6 | Document is the right public shape, but Cover/TOC still has two homes (`ARC-14`/`ARC-15`), `PDFRequest` is shallow (`ARC-16`), and prepare options still fork (`ARC-17`). Closed ARC-06 imageout climb still holds. |
| Correctness and API contracts | 9.0 | 7.4 | Cover inherits document HF against migration prose (`ARC-14`). `Collate` silent no-op (`EXT-10`). Missing Document knobs (`EXT-09`). OnError/nil-ctx proof gone with Converter (`PRAC-03`). |
| Rendering fidelity | 8.6 | 7.9 | EXT-01–EXT-07 engine rows hold. New deductions: fixture-56 accent stroke gate (`EXT-08`), HF `PaintBand` fork (`EXT-11`), wiki thumb stripper (`EXT-12`), rounded `OpLine` imageout fork (`EXT-13`). |
| Performance and scalability | 8.4 | 8.4 | Not re-scored this wave. Carry the prior measured baseline; no new perf defects filed. |
| Security and release readiness | 9.0 | 7.8 | CLI `--allow` still works. Library Document cannot express path allowlists the threat model documents (`PRAC-02`). Architecture docs still teach deleted image request APIs (`ARC-22`). |
| **Blended score** | **8.8** | **7.8** | `(7.6 + 7.4 + 7.9 + 8.4 + 7.8) / 5 = 7.82 → 7.8` |

Lens snapshot from the six agents (informational; not re-weighted into the blend):

| Lens | Score | Focus |
|---|---:|---|
| Document / CLI DAG | 7.3 | Cover HF, dual materialize, PdfGlobalOptions, stale Converter docs |
| convert / imageout seam | 8.0 | ARC-06 holds; prepare facade + font-registry residue remain |
| Layout / CSS / paint | 7.5 | Fixture accent, HF band, thumb stripper, OpLine fork |
| Extension seams | 7.4 | Document knobs/Collate; matrix honesty drift |
| Go practices | 7.6 | Allow ACL gap; OnError proof; Validate copies bound |
| Closed-ledger cross-check | 9.0 | No CR/ARC/EXT engine regressions; docs honesty only |

**Projected after this ledger’s open rows close** (same arithmetic, no new
scope): Architecture ~9.0, Correctness ~9.0, Rendering ~8.6, Performance 8.4,
Security ~9.0 → blended **~8.8 / 10** again. 10.0 still needs deeper ARC-03/04
immutability work and committed visual evidence classes outside this pack.

## Evidence baseline

- [x] Review scope frozen at `1a8ebf8`. Proof: this ledger header.
- [x] Six independently scoped read-only reviews completed. Proof: this ledger.
      No `go test` / `make` was run in this documentation-only wave.
- [x] **CR-01–CR-08, ARC-01–ARC-13, EXT-01–EXT-07, PRAC-01, R-01–R-04** confirmed
      closed against current source (cross-check agent). Spot proofs:
      `internal/imageout` imports `prepare`+`render` only (ARC-06);
      `settings.ResolvePDFMedia` / `ResolveImageMedia` (ARC-08 media);
      `writing-mode` inherit + `TestWritingModeInherits` (EXT-03);
      `reportPreflight` still used from `document.go` (PRAC-01 wiring).
- [x] Unrelated dirty worktree file (`.gitignore`) was not used as evidence.

## Phase 0: Preserve scope and carry-forward - P0

### 0.1 Product boundary

- [ ] JavaScript, CGO, Chrome parity, plugin frameworks, and a second public
      settings system stay out of this ledger.
- [ ] Dotted `Set` remains CLI/engine only. Document fields are the typed public
      overlay.

### 0.2 Documented follow-up (do not invent new IDs)

- [~] **ARC-03** Narrow claim holds (`TestPrepareImageDocumentUsesImageWidthViewport`).
      Deeper responsive stylesheet eligibility remains follow-up.
- [~] **ARC-04** Narrow claim holds (`TestMultipleWritesDeterministic`). Deeper
      `layout.Result` / `pdf.Document` immutability state-machine remains follow-up.

## Phase 1: Output and public-contract integrity - P0

### 1.1 ARC-14 - Cover must not inherit document headers/footers

`Document.mapPage` sets `HeaderSet`/`FooterSet` only when the Cover carries HF
pointers (`document.go:408-426`). With `Cover.Header == nil`, `HeaderFor` falls
through to `PdfGlobal.Header` (`internal/settings/settings.go:504-509`). CLI
forces the opposite for `--cover`: empty HF + `HeaderSet`/`FooterSet` true
(`internal/cli/cli.go:370-378`). Migration and library docs require no inherit
unless configured (`documentation/library-api.md:230`,
`documentation/MIGRATION-0.2.4.md`).

- [ ] **ARC-14** When `cover == true` and page HF is nil, stamp empty
      `Header`/`Footer` with `HeaderSet`/`FooterSet` true (match CLI). Path:
      `document.go` `mapPage`.
- [ ] **ARC-14** Unit: `Document{Cover, Header: Left "X", Pages…}` → cover object
      has `HeaderSet` and empty HF; body still inherits document HF. Proof:
      `go test . ./internal/cli -count=1`.

### 1.2 EXT-08 - Chrome stroke width must not hardcode fixture-56 accent `#2563eb`

`mixedLeftBorderPaintWidth` scales only when `side.Color == {37/255,99/255,235/255}`
(`internal/layout/layout_chrome.go:464-470`). That RGB is the fixture-56 accent.
Production stroke policy now depends on one report's hex.

- [ ] **EXT-08** Drop the color gate; express thickness via CSS / used border
      width only. Path: `internal/layout/layout_chrome.go`.
- [ ] **EXT-08** Unit: a non-`#2563eb` thick left rail asserts the same paint
      width rule. Proof: `go test ./internal/layout/ -run 'TestRounded|Fixture56|Architecture' -count=1`.

## Phase 2: Engine seams and half-wired extensions - P1

### 2.1 PRAC-02 - Document must expose path allowlists the threat model already documents

Public model has only `AllowLocalFiles bool` (`document.go:124`). Engine and CLI
still honor `Load.Allow` prefixes (`internal/settings/settings.go:327`,
`internal/cli/flags.go`). Threat docs tell operators to use `--allow /var/...`
(or API equivalent) (`documentation/THREAT-MODEL.md`,
`documentation/integration-security.md`).

- [ ] **PRAC-02** Add `Document.Allow` / `ImageDocument.Allow` (`[]string`), clone
      via `documentCloneStrings` into `global.Load.Allow`. Paths: `document.go`,
      docs beside `AllowLocalFiles`.
- [ ] **PRAC-02** Proof: allow-and-deny unit + mutate-after-write isolation on the
      slice. `go test . -run 'TestDocumentAllow|TestLoadAllow' -count=1`.

### 2.2 EXT-09 - Document mapper omits engine-consumed PdfGlobal knobs

CLI still registers `--grayscale`, `--page-offset`, `--exclude-from-outline`,
`--replace`, `--zoom`. `Document` has no fields; `pdfGlobal` never sets
`Grayscale` / `PageOffset` / `ExcludeFromOutline` (convert reads grayscale at
`internal/convert/pdf_pipeline.go:147`). After the hard break there is no public
`Set` escape hatch.

- [ ] **EXT-09** Add Document / Page / HeaderFooter fields only for Policy A
      consumed knobs; map in `pdfGlobal` / `mapPage` / `mapHeaderFooter`. Path:
      `document.go`, `internal/settings` types as needed.
- [ ] **EXT-09** Units: grayscale paint fold; page-offset HF `[page]`; replace
      substitution. Proof: `go test . ./internal/convert -count=1`.

### 2.3 EXT-10 - `Collate` plain bool is a silent no-op unless `Copies != 0`

```go
if d.Copies != 0 {
    global.Copies = d.Copies
    global.Collate = d.Collate
}
```
(`document.go:362-365`). Siblings use `*bool` for unset. `Document{Collate: false}`
alone does nothing; docs say a plain bool is explicit (`library-api.md`).

- [ ] **EXT-10** Make `Collate *bool` (match `Outline` / `Compression`) or always
      map when non-default. Path: `document.go`, validate, migration note.
- [ ] **EXT-10** Proof: `{Collate: boolPtr(false)}` with default copies →
      `PdfGlobal.Collate == false`. `go test . -run TestDocument -count=1`.

### 2.4 ARC-15 - Cover/TOC PdfObject materialization has two homes

Library `mapPage`/`mapTOC` and CLI `resolveFree` independently stamp `IsCover`,
outline exclusion, and HF overrides (`document.go:408-447`,
`internal/cli/cli.go:369-385`). ARC-14-class invariants will fork again.

- [ ] **ARC-15** One internal materialize helper for cover/TOC/page objects used
      by Document mapper and CLI final resolution (no new package; CLI must not
      import root `Document`). Paths: small helper beside settings or app.
- [ ] **ARC-15** Shared table test for cover HF + outline defaults. Proof:
      `go test . ./internal/cli -count=1`.

### 2.5 PRAC-03 - Document OnError / nil-context proof vanished with Converter

`reportPreflight` still wires hooks (`api.go:125-130`, `document.go:195-259`).
`TestConverterValidationErrorsReachOnError` is gone; root tests do not assert
`OnError` or `errors.Is(..., ErrNilContext)` for Document.

- [ ] **PRAC-03** Add `TestDocumentValidationErrorsReachOnError` (+ image twin)
      for nil ctx, nil writer, and Validate failures. Path: `document_render_test.go`.
- [ ] **PRAC-03** Proof: `go test . -run 'TestDocument.*OnError|TestImageDocument.*OnError' -count=1`.

### 2.6 ARC-16 - convert still exposes two PDF job types

`PDFRequest` + `ToRequest` + `RunTypedPDF` sit beside PDF-only `Request`
(`internal/convert/request.go:11-46`). Document builds `PDFRequest` then
`ToRequest()`; app builds `*Request` directly. Unused image-union sentinels
remain (`ErrUnexpectedImageSettings`, `ErrMissingImageSettings`).

- [ ] **ARC-16** Document/app both construct `*convert.Request` (or one
      constructor). Delete or unexport `PDFRequest` / `RunTypedPDF` / dead
      image-union sentinels once seams/fuzz tests move.
- [ ] **ARC-16** Proof: `rg 'PDFRequest|RunTypedPDF|ErrUnexpectedImageSettings' --glob '*.go'`
      limited to intentional remnants; `go test ./internal/convert ./internal/app . -count=1`.

### 2.7 ARC-17 - prepare options assembly still forks PDF vs image

convert hub re-exports `PrepareDocument` / `CollectSheets` / `MergeFontFaces`
(`internal/convert/prepare.go`) while image calls `prepare.Document` directly.
SimplifyDOM / media option assembly differs between
`convert.go:485-494` and `imageout.go:1731-1758`.

- [ ] **ARC-17** One prepare-options builder (in `prepare` or `settings`) shared
      by PDF and image. Convert internals import `prepare` directly; drop shallow
      hub re-exports once call sites move.
- [ ] **ARC-17** Table test: identical Global+Object yield the same `Options` in
      both modes. Proof: `go test ./internal/convert/prepare ./internal/convert ./internal/imageout -count=1`.

### 2.8 EXT-11 - HF `PaintBand` simple path forks body paint policy

HF passes only origin (`internal/convert/hf.go`) → `useSimple` → `bandText` /
`bandStrokeRect` ignore `RotateDeg`, radius, and stroke mask that body `draw*`
honors (`internal/layout/paint.go:607-783`, `1210-1221`).

- [ ] **EXT-11** One op prologue + draw table behind `Paint` and `PaintBand`
      (BandOptions stay for origin only). Path: `internal/layout/paint.go`.
- [ ] **EXT-11** HF HTML with `border-radius` + `writing-mode: vertical-rl`
      matches body draw policy. Proof: `go test ./internal/layout ./internal/convert -count=1`.

### 2.9 EXT-12 - `stripThumbImageHairlines` is a wiki post-pass inside Paint

Geometry heuristic no-ops hairlines under large images on every Paint
(`internal/layout/paint_pagination.go:1230-1244`, called from `paint.go:130`).
Comment names wiki thumbs.

- [ ] **EXT-12** Fix link/underline emission for collapsed figure frames in
      chrome/inline paint; delete the Paint stripper.
- [ ] **EXT-12** Proof: `go test ./internal/layout/ -run 'Thumb|Hairline|Figure' -count=1`.

### 2.10 EXT-13 - Rounded accent `OpLine` policy has a second home in imageout

Chrome still emits raw `OpLine` for some mixed sides; imageout rewrites into
`StrokeMask*` overlays (`internal/imageout/imageout.go:373-429`). PDF paints the
line as a straight segment.

- [ ] **EXT-13** Emit only `OpStrokeRect` + `StrokeMask*` from chrome; delete
      imageout overlay reconstruction. Depends-on: EXT-08.
- [ ] **EXT-13** Proof: `go test ./internal/layout/ ./internal/imageout/ -run 'Rounded|RasterPaint' -count=1`.

## Phase 3: Shared forks and sentinel identity - P2

### 3.1 ARC-18 - ARC-08 font-registry collapse did not finish

`loadFontRegistry` and `fontRegistry` still wrap `pdf.RegistryFromPaths`
separately (`internal/convert/convert_helpers.go:234+`,
`internal/imageout/imageout.go:1806+`). `RegistryFromGlobal` was never added.
Media half of ARC-08 holds.

- [ ] **ARC-18** One `pdf.RegistryFromGlobal(PdfGlobal)` (or equivalent);
      convert/imageout only log. Proof: `rg 'func (loadFont|font)Registry' internal`
      empty; `go test ./internal/pdf ./internal/convert ./internal/imageout -count=1`.

### 3.2 ARC-19 - Dead post-validate `len(cmd.Objects)==0` guard remains

`BuildPDFRequest` already validates (`internal/app/pdf.go:46-48`). `RunPDF` still
rechecks `len == 0` with a weaker predicate (`pdf.go:90-92`). ARC-11 required
deleting it; alias is done, guard remains.

- [ ] **ARC-19** Delete the redundant branch. Proof: `rg 'len\(cmd\.Objects\) == 0' internal/app`
      empty; `go test ./internal/app -count=1`.

### 3.3 EXT-14 - `table-layout` is a typed used-value with no consumer

Applied in `style_properties.go:1211-1213`, stored on `ResolvedStyle`, never read
by `buildTable`. Same half-wired shape as closed EXT-02 before it was fixed.

- [ ] **EXT-14** Consume in column used-width resolution **or** drop apply + field
      (matrix stays Not implemented). Paths: `internal/layout/style_properties.go`,
      tables.
- [ ] **EXT-14** If consumed: unit for `table-layout: fixed`. Proof:
      `go test ./internal/layout/ -count=1`.

### 3.4 ARC-20 - `PdfGlobalOptions` remains a second typed settings system

`internal/settings/options.go` still calls itself the “typed library-side
builder”. Production Go callers are tests only. Phase 35 removed the public
builder in favor of Document fields.

- [ ] **ARC-20** Delete or demote to clearly test-only helpers; stop documenting
      `With*` as the library API. Paths: `options.go`, architecture/settings docs,
      matrix library rows.
- [ ] **ARC-20** Proof: `rg 'NewPdfGlobalOptions|WithPDFVersion' --glob '*.go'`
      tests-only; docs cite `Document.PDFVersion` / `PDFProfile`.

### 3.5 PRAC-04 - HTML ownership test never calls WritePDF

`TestDocumentAdapterCopiesHTMLAtExecutionBoundary` mutates then only `Validate`s
(`document_render_test.go:67-79`). That proves intake clones, not mapper
isolation at write.

- [ ] **PRAC-04** Assert after `WritePDF`/`WriteImage` starts (mutate
      `Pages[0].Source.HTML` / `FontPaths` against request isolation), or add a
      dedicated mapper-boundary test. Align Write* comments with actual clones
      (see ARC-23).
- [ ] **PRAC-04** Proof: `go test . -run TestDocumentAdapterCopiesHTMLAtExecutionBoundary -count=1`.

### 3.6 PRAC-05 - Document.Validate accepts copy counts the engine later rejects

Public validate rejects only `Copies < 0` (`document_validate.go:108-110`).
`convert.Request.Validate` enforces `Copies > maxConversionCopies` (1000).

- [ ] **PRAC-05** Teach `validatePDFOptions` the same upper bound; alias/wrap to
      the public copies sentinel. Fix `library-api.md` Copies row (`Copies < 0`,
      not `< 1`).
- [ ] **PRAC-05** Proof: `go test . -run 'TestDocumentValidate/too_many_copies' -count=1`.

### 3.7 ARC-21 - CLI Command keeps dead dual homes after ARC-13

`OutlineWriter` is declared and never assigned (`internal/cli/cli.go:63-71`).
`DumpDefaultTOCXSL` on `Command` is never written; live path reads
`cmd.Global.DumpDefaultTOCXSL`. Docs still mention `Command.DumpOutline`.

- [ ] **ARC-21** Drop unused `Command` fields; docs match `PdfGlobal` + outline
      writer into `app.RunPDF`. Proof: `rg 'OutlineWriter|cmd\.DumpDefaultTOCXSL|Command\.DumpOutline' --glob '*.go'`
      clean; dump-outline conflict tests still pass.

## Phase 4: Honesty and locality - P3

### 4.1 EXT-15 - Compatibility matrix disagrees with post-EXT-07 source

`text-indent` still “parsed, never consumed” while `inline.go` consumes it.
`writing-mode` notes say horizontal-only while `TestWritingModeInherits` requires
`RotateDeg == -90`. Library rows still cite `WithPDFVersion`. Audit stamp
2026-08-13.

- [ ] **EXT-15** Rewrite inverted rows against apply / consume / test. Cite
      `Test*` or fixtures. Path: `documentation/compatibility-matrix.md` (+ deferred
      `@page size` if still wrong).
- [ ] **EXT-15** Proof: `rg -n 'text-indent|writing-mode|WithPDFVersion' documentation/compatibility-matrix.md`
      matches source; bump honesty date.

### 4.2 ARC-22 - Architecture docs still teach deleted convert image APIs

Phase 4.2 of the 2026-08-13 ledger required those strings gone.
`documentation/architecture/10-imageout-svg.md:176-187`,
`01-entrypoints-cli.md:214`, `08-convert-pipeline.md:139-141`, and top-level
`architecture.md` DAG still describe `NewImageRequest` / `ValidateImage` /
`firstObject` ignore-extras. Production uses `imageout.NewRequest`.

- [ ] **ARC-22** Rewrite to `imageout.Request` / `prepare` direct / Document.
      Paths: `documentation/architecture/{README,01,02,08,10}.md`,
      `architecture.md`.
- [ ] **ARC-22** Proof: `rg 'NewImageRequest|ValidateImage|firstObject|imageout ──► convert' documentation/architecture`
      empty or historically qualified; `make claim-scan`.

### 4.3 ARC-23 - Settings / Document docs overclaim library Set and clone depth

`internal/settings/doc.go` still says dotted Set is used by the library API.
`02-library-api.md` / Write* comments claim full map clone while mapper builds
fresh structs and clones selected slices only.

- [ ] **ARC-23** Doc honesty: CLI + engine own `Set`; Document is the typed
      overlay; state the single-goroutine / snapshot rule that is actually true.
- [ ] **ARC-23** Proof: comment/`03-settings.md` match import graph; no
      “library Set” claim without a root caller.

## Phase 5: Closure gates - P4

Documentation-only review: leave these unchecked until an implementation wave
records command output.

- [ ] `make lint` clean after the implementation rows above.
- [ ] `make test` green after the implementation rows above.
- [ ] Layout / print rows (`EXT-08`, `EXT-11`–`EXT-14`) also:
      `go test ./internal/layout/ -count=1` and
      `go test ./internal/convert/ -run 'TestGoldenCorpus' -count=1`.
- [ ] Document / ACL rows (`ARC-14`, `PRAC-02`, `EXT-09`, `EXT-10`, `PRAC-03`) also:
      `go test . ./internal/cli ./internal/app -count=1`.
- [ ] Image / prepare rows (`ARC-17`, `ARC-18`, `EXT-13`) also:
      `go test ./internal/imageout ./internal/convert/prepare -count=1`.
- [ ] Do not treat this documentation wave as proof that lint/test are green.

## Parked (over the 25-row cap; not fake phases)

- Layout tests assert through private `Result.root` / `*box` (interface-is-test-surface friction). Migrate sticky/orphans tests off `res.root` when touching those files.
- `beforeAlways` god-function (`paint_flow.go`) and `preferSplitOverBlank` wiki-gap tuning — deepen when pagination work resumes.
- `pagePlan.Remap` nil-model fallback; islands benchmark marker duplicated in convert vs islands.
- `prepare.ResourceContext` deprecated Loader/Base/Load snapshots.
- ImageDocument progress hooks are bookend-only while PDF streams real progress — document or plumb, do not invent a hook bus.
- convert vs imageout nil-ctx / Validate ordering disagreement.
- Root `document_render_test.go` file-level `usetesting` nolint with leftover `context.Background()`.
- Public `ErrNoPageObjects` is a distinct root sentinel from `settings.ErrNoRenderableObjects` (cross-check note; alias only if product wants `errors.Is` across both).

## Refused

- Re-filing closed CR-01–CR-08, ARC-01–ARC-13 (except ARC-18 font residue and ARC-19 dead guard), EXT-01–EXT-07 engine rows, PRAC-01 wiring.
- Plugin / visitor / paint-sink interface / second public dotted `Set` on Document.
- Package-per-file split of `internal/layout`.
- Mutex on Document or layout; context on every struct; pixel-diff merge gate; live network in `make test`.
- Clone-at-CLI for process-exclusive `Command`.
- Ponytail deletion-only of unused helpers without a locality win.
- Perf micro-allocation claims (perf-review).

## Dependencies

```
Phase 0 (freeze & carry-forward)
    │
    ▼
Phase 1 (ARC-14, EXT-08)          ← no dependency on seams
    │
    ▼
Phase 2 (PRAC-02, EXT-09/10, ARC-15–17, PRAC-03, EXT-11–13)
    │         ARC-15 after or with ARC-14
    │         EXT-13 after EXT-08
    │         ARC-17 before more prepare work
    ▼
Phase 3 (ARC-18–21, EXT-14, PRAC-04/05)
    │
    ▼
Phase 4 (EXT-15, ARC-22, ARC-23)  ← docs after engine truth
    │
    ▼
Phase 5 (lint / test / golden gates)
```

## Handoff

1. Ledger: `plans/0.2.4/improve-codebase/codebase-2026-08-20/phase-wise-checklist.md`
2. Rating: **7.8 / 10** now (was **8.8 / 10** after the 2026-08-12 health wave). Projected **~8.8 / 10** once this ledger’s open rows close.
3. Counts: ~55 findings in → 23 active rows out → large refuse set (closed prior IDs + product ceiling).
4. P0 titles: **ARC-14** Cover HF inherit; **EXT-08** fixture accent stroke gate.
5. P1 titles: **PRAC-02** Allow prefixes; **EXT-09** missing Document knobs; **EXT-10** Collate; **ARC-15** dual cover/TOC; **PRAC-03** OnError proof; **ARC-16** PDFRequest; **ARC-17** prepare fork; **EXT-11** HF paint; **EXT-12** thumb hairlines; **EXT-13** rounded OpLine.
6. Not done: no implementation, no perf-review, no ponytail, no HTML report (ledger is the report for this pack).
7. Next: implement named IDs, or re-run one lens on a narrower slice (`document`, `layout`, `prepare`).
