# 0.2.6 review - leftover print CSS architecture and ponytail

> **Parent:** `plans/0.2.6/48-canonical-0.2.6-css-coverage.md` - leftover CSS shipped in `48e06dbc`
> **Status:** implementation complete. All active rows are `[x]`; final gates are recorded below.
> **Estimated effort:** completed in one bounded implementation wave.
> **Date:** 2026-08-28

---

## Overview

Six read-only agents scanned the 61 files from the original
`48e06dbc745dd3ea087bba1e7f007791e847d663` snapshot. This wave implemented
the 24 active review rows against the current tree. Product ceiling remains a
print-oriented report renderer, not Chrome print.

Lenses:

- `/ponytail` full (what to delete or shrink)
- `/improve-codebase-architecture` (deep modules, seams, locality)
- architecture-deepening, extension-seams, go-practices from
  `skills/improve-codebase/`

This folder is the review ledger. It does not replace the canonical 48-56
execution ledger. The user authorized every active row in this file.

## Frozen snapshot

| Field | Value |
|-------|-------|
| Branch | `feature/026-extended-css-support` |
| Snapshot | `48e06dbc745dd3ea087bba1e7f007791e847d663` |
| Current state | review implementation complete; Git state intentionally not inspected |
| Date | 2026-08-28 |
| Scope | files in that commit only |
| Product | controlled-report renderer. No JS. No CGO. |

Production Go paths in the commit: `internal/css`, `internal/layout`,
`internal/convert`, `internal/imageout`, plus matrix/deferred/catalog/phase
docs.

## Executive summary

The review wave now has direct proofs for the previously weak consumers:
of-type matching, attr flags, page selectors, named page margins, unnamed
margin boxes, flex and grid placement, elliptical radii, box-shadow radii,
and caption-side geometry.

What the scan found is not "the CSS is fake". It is three kinds of tax:

1. **Tests that stay green if the feature dies.** Caption-side geometry
   stamps styles when cascade misses. `TestZIndexPaintOrder` never compares
   paint order. `flex-wrap: wrap-reverse` and `grid-auto-flow: column` are
   matrix Implemented with intern-only or missing layout proof.
2. **Page geometry had two homes.** Layout `PaintOptions.forPage` applied
   named plus side plus `:first`; convert `hfGeom.pageMargins` skipped named.
   The current code shares the layout cascade and records page names after
   pagination. Margin-box ownership is now explicit in `page_margin_boxes.go`.
3. **Catalog and phase Status lagged the code.** The mapping and phase
   overviews now describe `@import`, `:is()`, `:where()`, and the six supported
   unnamed margin-box names accurately.

No P0 security or public `Convert`/`Run*` panic was proven in this file set.

Counts: about 64 raw findings in, **24 active rows** closed, **12 refused**,
**8 parked**. Cap is 25.

## Phase 1: Integrity (tests that lie, cascade that widens)

Correctness of the leftover claims. One row is one proof gap or one
widening parse.

### 1.1 Tests that lie

- [x] **PRAC-01** Delete the caption-side stamp fallback in
      `internal/layout/caption_side_test.go:97-105`.
      `layoutCaptionSide` now calls `layoutHTML` directly, so the geometry
      tests depend on the real cascade. Proof: the caption-side test group
      passed after the fallback and its imports were removed.

- [x] **PRAC-02** Make `TestZIndexPaintOrder` assert order.
      The test now compares the low and high fill positions in
      `PaintOrder(res.Ops)`, which is the order the painter consumes. Proof:
      `go test ./internal/layout/ -run TestZIndexPaintOrder -count=1`.

- [x] **PRAC-03** Add a layout proof for `flex-wrap: wrap-reverse`.
      The new test lays out two wrapped items and asserts the second line is
      above the first. Proof: `go test ./internal/layout/ -run
      TestFlexWrapReverse -count=1`.

- [x] **PRAC-04** Add a placement test for `grid-auto-flow: column`.
      The new test uses explicit rows and asserts that the second auto item
      shares the first item's X coordinate and has a larger Y. Proof:
      `go test ./internal/layout/ -run TestGridAutoFlowColumn -count=1`.

### 1.2 Parse that widens a narrower rule

- [x] **EXT-01** Reject leftover tokens in `@page` preludes.
      `parsePageSelector` now accepts only one named ident or one supported
      page pseudo. Combined names, pseudo suffixes, and selector lists are
      ignored. Proof: CSS parser tests and `TestPageSelectorLeftoverDoesNotApply`.

- [x] **EXT-02** Stop gluing unknown attr flags onto the comparison
      `splitAttrIFlag` strips both `i` and `s` while preserving the comparison
      value. The `s` flag selects the existing exact comparison path. Proof:
      `go test ./internal/css -count=1 -run TestAttrIFlag`.

## Phase 2: Seams (page module ownership)

The next named-page or `@page` change already has two owners.

### 2.1 Named page module

- [x] **ARC-01** Give `page: ident` one owner in layout.
      `internal/layout/page_named.go` owns used-value inherit, sibling
      `page-break-before:always`, `namedPageNames`, and
      `PaintOptions.forPage`. `internal/convert/page_margin_boxes.go` owns
      unnamed `@page` quoted boxes onto CLI HF.
      Named `@page ident { margin }` lengths resolve in
      `applyCSSPageMargins` (`internal/convert/convert_helpers.go:26-36`).
      Each verb now has one named owner. Proof: `TestPageNamedMargins` still
      goes through `convert.Run`, and the source paths separate layout page
      names from convert margin-box mapping.

- [x] **ARC-02** One page-box cascade for paint and link geometry.
      `layout.PaintOptions.PageMarginsForPage` resolves unnamed, named,
      side, and first-page margins. `hfGeom.pageMargins` calls that method
      with the page names recorded after pagination, so link and outline
      coordinates use the paint cascade. Proof: `TestPageMarginsSharePaintCascade`.

- [x] **EXT-03** Ignore named and page-pseudo `@page` margin boxes without
      misleading storage. Closure choice: keep the lite CLI header/footer
      path, capture boxes only for unnamed rules, and reject named or
      pseudo-page boxes before storage. Proof: `TestParsePageMarginBoxesNamedPage`
      and the page-selector rejection test.

## Phase 3: Shared forks (radius, helpers, adapters)

Locality. Behavior already works on the happy path.

### 3.1 Used radius

- [x] **ARC-03** Collapse X and Y used-radius into one helper in
      `border_radius.go`. `usedBorderRadiiXY` now resolves and clamps both
      axes, and inline and block chrome consume its result. Proof:
      `go test ./internal/layout/ -run 'TestRadius' -count=1`.

- [x] **ARC-04** Stamp used radii on box-shadow fills.
      `appendBoxShadow` now receives both used-radius axes and stamps them on
      core and blur fills. Proof: `TestBoxShadowPaints` includes a rounded
      shadow case and `go test ./internal/layout/ -run 'TestBoxShadow' -count=1`.

- [x] **PRAC-05** Put elliptical longhand onto a paint op in tests.
      `TestRadiusEllipticalLonghand` now lays out a longhand slash rule and
      checks `RadiusTopLeftY` on the fill op. Proof: `go test
      ./internal/layout/ -run TestRadiusEllipticalLonghand -count=1`.

### 3.2 Convert helpers and PDF/image adapters

- [x] **ARC-05** Resolve `@page` boxes once, next to `css.PageRule`.
      `collectPageBox` now walks each `Stylesheet.Pages` slice once. The
      typed `PageMarginBoxes` table is folded into the same raw page-box
      record, and `paintOptions` lives beside `hfGeom` in `hf_geometry.go`.
      Proof: there is no `collectUnnamedPageBoxes`, and convert page tests pass.

- [x] **ARC-06** One font-registry scan notice.
      `pdf.LogFontRegistryScan` is the single helper used by PDF and image
      requests. Proof: `rg 'func .*FontRegistryScan' internal` finds one
      implementation in `internal/pdf/registry.go`.

- [x] **ARC-07** One flex justify helper for row and column.
      Column distribution now calls the row-side `justifyDistributed` helper;
      the duplicate function is gone. Proof: `go test ./internal/layout/
      -run 'TestFlex' -count=1`.

## Phase 4: Ponytail cuts

Deletion and shrink only. Ranked biggest honest cut first. Do not delete
features this commit just shipped (`ch`, elliptical Y, caption-side
left/right). Those are refused below.

- [x] **PT-01** One walk of `@page` blocks.
      `parsePageBlock` scans one page block, records supported quoted boxes,
      and returns declarations with nested at-rules removed. Convert folds
      those boxes into its single page-box walk. Proof: the CSS and convert
      margin-box tests pass.

- [x] **PT-02** `ofTypeLastIndex` from `ofTypeIndex`.
      The implementation counts same-tag siblings once, then returns
      `total - ofTypeIndex(node) + 1`. Proof: the last-of-type selector tests
      pass.

- [x] **PT-03** One width/style/color token loop for `column-rule` and
      `outline` and `column-rule` now consume `parseRuleShorthand` in
      `style_paint_props.go`. Proof: `go test ./internal/layout -run
      'TestColumnRuleParse|TestOutline' -count=1`.

## Phase 5: Docs honesty

Docs follow code. No engine change.

- [x] **DOC-01** Flip catalog rows that already have consumers.
      `@import` is partial, `:is()` and `:where()` are implemented, and the
      six unnamed margin-box names are partial with explicit lite notes.
      Proof: `python3 scripts/css-catalog-map.py --check` and the catalog
      entries match the matrix.

- [x] **DOC-02** Rewrite phase 49-52 Status and Overview to match `[x]`
      Phase 49-52 now describe the live selector, import, value, template,
      background, radius, outline, overflow, and shadow implementations.
      Their status lines are complete and their file references name current
      code.

- [x] **DOC-03** Fix the ghost `TestGridIndependentGaps` citation.
      Matrix §2.8 line 202. No such function. Real test is
      `TestGridRowGapVsColumnGap` (`grid_test.go:360`), which does not
      pin row-gap to 8pt. Proof: grep `TestGridIndependentGaps` empty;
      matrix names the live test.

- [x] **DOC-04** State the named-page link behavior on the matrix
      `page` row. Link and outline destinations now use the same named-page
      cascade as painting after page names are recorded. Proof: the matrix
      sentence and `TestPageMarginsSharePaintCascade` match the code.

## Phase 6: Closure gates

This wave included source changes, tests, and documentation. The final gates
below were run after the implementation and documentation edits.

- [x] **GATE-01** Session-end gates on the review implementation are now
      2026-08-28 evidence. `make test`, `make lint`, `make golden`,
      `make claim-scan`, and `make build` all exited 0 after the source and
      documentation changes.

- [x] **GATE-02** Before closing any non-doc row above, record
      `make lint` and `make test` exit 0 on the final implementation. Layout
      and paint rows also passed `go test ./internal/layout/ -count=1` and
      the golden corpus command after the implementation.

## Dependencies

```text
EXT-01 (page selector) ── independent
EXT-02 (attr flag)     ── independent
PRAC-01..04 (tests)    ── independent; do first, they are proof
ARC-01 (owner names) ──▶ ARC-02 (shared cascade) ──▶ ARC-05 (one walker)
ARC-03 (radius helper) ──▶ ARC-04 (shadow radii) ──▶ PRAC-05 (longhand op)
ARC-06, ARC-07, PT-*   ── independent
DOC-*                  ── can land without engine changes
GATE-01                ── after leftover work, before claiming 56 on this HEAD
```

Implement named IDs only. Suggested first slice if asked: PRAC-01, PRAC-02,
EXT-01, ARC-01/ARC-02, DOC-01/DOC-02.

## Refused

Lookalikes that are decided tradeoffs or ponytail overreach. Do not re-file
without a current-source regression.

| ID | Why refused |
|----|-------------|
| imageout imports `convert/prepare` + `convert/render` | Known live leak. This commit dropped the parent `convert` hub import. Not a new invention. |
| Paint-sink interface for `pagePainter` vs band paint | Decided. Extract a helper (not filed as a new interface). Two adapters that do not share a paint loop. |
| `internal/layout/flex` or `layout/table` packages | Package-per-file DAG refused. ARC-07 stays same-package. |
| Delete `ch` / `style_ch.go` | Just shipped. `TestChUsesZeroGlyphAdvance` proves glyph advance, not 0.5em. |
| Delete elliptical Y radii | Just shipped. `TestRadiusSlash` paints slash syntax. |
| Delete `caption-side: left\|right` | Just shipped. Geometry tests exist (once PRAC-01 removes the stamp). |
| Mutex on layout or `pdf.Document` | Single-goroutine engine. |
| Plugin / visitor for CSS properties | `styleGroups` is the apply table. |
| Second settings hierarchy | Product. |
| Pixel-diff goldens as wrap-reverse proof | Goldens are structural. |
| Dual public stories (`api.go` + `convert.Request`) | Tax, not a bug. |
| `Render` using `context.Background()` | Documented public adapter; `RenderContext` exists. |

## Parked (over the 25-row cap)

P3 or weaker P2. Re-open only if a named ID is in progress in the same file.

- `TestFlexRowLayout` counts text ops only (`flex_test.go:113-140`).
- `TestFlexShorthandParsing` throws away `layoutHTML` and re-resolves intern fields.
- `*SurvivesPaint` tests re-read `res.Ops` after `Paint` without checking painted pages.
- Logical margin/padding/inset/size tests intern only (`css_apply_test.go:180-278`).
- Of-type and attr `i` are matrix Implemented with css `Match` tests, no
  `TestIsPseudoStyle`-style layout consume (fixture-55 of-type equals nth-child
  on same-tag rows).
- `ch` on `@page` / media still uses `css.LengthToPt` 0.5em
  (`convert_helpers.go:184,222`; `container.go:137`). Matrix already notes
  query 0.5em. Catalog `units.ch` is partial.
- Percent `border-radius` collapses to a uniform min-side circle
  (`border_radius.go:48-51`). Matrix Partial. Next change: drop percent decls
  the engine cannot honor, or store per-corner percent.
- Box-shadow blur test reads `engine.deferredChrome` (`box_shadow_test.go:72`).
- `applyRelativeOffset` lives at the tail of `flex.go`.
- `styleContext` stores ctx besides `engine` (`style.go:323`). Document as the
  same engine's style pass, do not add a third.
- Image `Finalize` writes then `render.Run` can still return cancel
  (`imageout.go:1673`). Hypothesis; needs a cancel-during-write test.
- Page vs band `Save`/CTM/opacity wrap copied in `paint.go:410` and `:664`.
- Caption attach in `buildTable` vs `placeTableCaption`.
- `comparableResolvedStyle` grows by every new field (intern projection).

## Ponytail net (not a ship list)

Ponytail-review on this commit estimated about **1100 lines** if you also
deleted `ch`, elliptical Y, caption-side sides, box-shadow blur steps, and
filter opacity. After the refuse list, honest cuts left in PT-01..PT-03 plus
ARC-06/ARC-07 are on the order of **100-200 lines**, no deps.

No `ponytail:` debt comments in the scoped files.

## What this wave did not do

- No perf-review or critical-go-review was requested by this checklist.
- Git publication is handled separately after the final gates and explicit user authorization.
- Refused and parked items remain outside the 24 active rows.
