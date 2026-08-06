# fix-log/fix-layout.md — wave-1 remediation of `internal/layout`

Agent: fix-layout (owns `internal/layout/*`). Date: 2026-08-07.
Source of truth: `phases/phase-04-layout-engine.md`, phase-03 layout rows, phase-05 layout paint, phase-02 P2-13 layout side.

## Per-CID status

### P4-01 — resolveImage + imageRef cache — DONE
- Added `imageRef` (`src`, `data`, `w/h`, `isJPEG`) and `engine.imgCache map[string]*imageRef`.
- `resolveImage(src)` fetches/decodes once per Layout run (SVG rasterize or `imageDims`); failed fetches cache a nil miss so we do not re-hit the loader.
- `box.img *imageRef` replaces the five image fields; `inlineItem.imgRef` likewise.
- `buildImage` and `measureImageWidth` both call `resolveImage` (measure no longer double-fetches/decodes).

### P4-02 — content-box helpers — DONE
- Added `engine.contentBox(x, w, st) (cx, cw)` as the single scaled content-box rule.
- Call sites switched: `buildBlock`, `emitCell`, `layoutCell`, `buildFlex`, `buildGrid`, `buildMulticol`.
- `resolveUsedWidth` / `resolveUsedHeight` / `resolveContentHeight` left as specialized used-size helpers (already shared by flex/grid/multicol).
- `container.contentInlineSize` still unscaled (measure pass has no `engine`); intentional for pre-layout size containers — zoom divergence noted as residual, not changed (would need zoom threaded into `measureSizeContainers`).

### P4-03 — sizeTableColumns pure function — DONE
- Extracted `tableColumnEnv` + `sizeTableColumns(env) (colW, tableW)` from the ~160-line slab in `buildTable`.
- `buildTable` measures cells, builds env (`chrome`, `availW`, definite `tableW` hint or -1 auto), then sizes columns via the pure function.
- Algorithm is a verbatim move; unit-testable without DOM/ops.

### P4-04 — stored cell start row — DONE
- `box.row` set at placement: `cell.row, cell.rowSpan = p.row, p.rSpan`.
- Removed `cellStartRow` scans; rowspan growth, final height assign, `rowspanCovers`, and `colspanCovers` use `cell.row`.

### P4-05 — one text-wrap policy — PARTIAL
- Added `wordBreakPolicy` + `wordBreakOf(st)` (normal / break-all / break-word / never).
- Wired into `unbreakableMinWidth` and `breakOverflowItem`.
- Full `breakToken` / `minContentWidth` unification and `measureCellMinMax` rewrite deferred (higher risk; fixtures green with policy enum only).
- `// FIX-REVIEW: P4-05` residual: complete token chunker shared by measure + line pack.

### P4-06 — inline CB width into inline-block measure — DONE
- `engine.inlineCBW` set in `layoutInlineFloats` from `contentW`.
- `inlineBlockAvail(n, st, cbW)` resolves `width:%` against `cbW` when > 0, else viewport fallback.
- `collectInlineNode` passes `e.inlineCBW`.

### P4-07 — defer chrome / stop splice — PARTIAL / DEFERRED
- Attempted deferred `chromeEntry` + `finalizeChrome`: broke chrome-only boxes (empty `opStart>opEnd` before merge), sticky clone StickyID stamping, and transform stamping on abspos fills.
- Restored immediate splice in `prependChrome` for correctness.
- Marker in `prependChrome`: `// FIX-REVIEW: P4-07 full deferred-chrome needs re-stamping StickyID/Fixed and chrome-only ranges after merge`.

### P3-01 — one cascade rule-walk for pseudo-content — DONE
- Added `styleContext.matchedRules(n, pe)` with media + `@container` + specificity gates.
- `cascadeRaw` author loop iterates `matchedRules(n, "")`.
- `pseudoContent` builds `styleContext` with `e.containers` and uses `matchedRules(n, pe)` for `content` only.
- Engine stores `containers` from the Layout re-cascade so matching `@container` content still applies; non-matching `@container` content is no longer applied unconditionally.

### P3-03 layout — LengthToPt callers — DONE
- `fontSize`, `lineHeight`, `lengthBox`, `marginLen`, `parseTransformLength` call `css.LengthToPt(val, unit, basePt)`.
- Property policy kept locally (`%`, rem-root via `pxToPt(16)`, line-height unitless, unknown line-height units inherit instead of mm-default).

### P3-04 layout — ResolveCustomProps — DONE
- `mergeCustomProps` gathers `--*` declared props and returns `css.ResolveCustomProps(declared, parentProps)` (or parent map when none declared).
- `resolveRawVars` unchanged (plain property pass over resolved custom props).

### P3-05 — collapse style surfaces — DONE (thin wrappers)
- Single entry `resolveStylesWith(root, opts, containers)`.
- Prior four names remain as thin adapters for tests/callers.
- Layout container re-cascade uses `resolveStylesWith` directly.

### P5-01 layout paint semantics — DONE (export side)
- Exported `PaintStyle`, `StyleOf(op)`, `FakeBoldFor(op)`.
- Exported `BandOptions` + `PaintBand` (shared fill/stroke/line/text/image dispatch; links skipped for convert HF).
- `drawFill` / `drawText` use StyleOf/FakeBoldFor.
- convert `hf.go` / imageout consumption = fix-convert / fix-imageout-wave2.

### P5-07 layout image embed errors — DONE
- `drawImage` returns `AddJPEGImage`/`AddPNGImage` errors (no `_ =`).
- `Paint` accumulates first embed error and returns it; `PaintBand` likewise.

### P2-13 layout DeactivateOp — DONE
- Exported `DeactivateOp(op *Op)` + unexported `opKindNoop OpKind = 255`.
- `Paint` / `PaintBand` skip `opKindNoop`.
- convert `stripLinkURIs` consumer = fix-convert.

## Files changed
- `internal/layout/layout.go` — imageRef/cache, contentBox, sizeTableColumns, cell.row, wordBreakPolicy, DeactivateOp, Layout containers, buildImage/measure paths
- `internal/layout/inline.go` — imgRef, inlineCBW, inlineBlockAvail(cbW), wordBreakOf in breakOverflow
- `internal/layout/flex.go`, `grid.go`, `multicol.go` — contentBox
- `internal/layout/style.go` — resolveStylesWith, matchedRules, mergeCustomProps → ResolveCustomProps, LengthToPt callers
- `internal/layout/pseudo_content.go` — matchedRules + containers
- `internal/layout/transform.go` — LengthToPt in parseTransformLength
- `internal/layout/paint.go` — PaintStyle/StyleOf/FakeBoldFor/PaintBand, image embed error prop, noop skip

## Validation
```
gofmt -w $(find internal/layout -name '*.go')
go build ./internal/layout/...
go vet ./internal/layout/...
go test ./internal/layout/... -count=1
```
All green (`ok gowkhtmltopdf/internal/layout`).

## Remaining markers / handoffs
| Marker | Owner |
|---|---|
| `// FIX-REVIEW: P4-07` deferred chrome | fix-layout later wave |
| `// FIX-REVIEW: P4-05` full breakToken/minContentWidth | fix-layout later wave |
| PaintBand/StyleOf consumers | fix-convert, fix-imageout-wave2 |
| `layout.DeactivateOp` consumers | fix-convert |
| container.go zoom in contentInlineSize | optional follow-up |

## Summary for parent
- **done:** P4-01, P4-02, P4-03, P4-04, P4-06, P3-01, P3-03 layout, P3-04 layout, P3-05, P5-01 layout, P5-07 layout, P2-13 layout
- **partial:** P4-05 (policy enum wired), P4-07 (deferred chrome rolled back; marker left)
- **blocked:** none
- **CI:** `go build/vet/test ./internal/layout/...` all pass
