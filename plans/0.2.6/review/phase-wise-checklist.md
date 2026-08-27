# 0.2.6 review - leftover print CSS architecture and ponytail

> **Parent:** `plans/0.2.6/48-canonical-0.2.6-css-coverage.md` - leftover CSS shipped in `48e06dbc`
> **Status:** review only. Rows stay `[ ]` or `[~]` until a later change proves them.
> **Estimated effort:** 2-4 days if implementing P1 only; ponytail and docs are extra half-days.
> **Date:** 2026-08-28

---

## Overview

Six read-only agents scanned **only** the 61 files in commit
`48e06dbc745dd3ea087bba1e7f007791e847d663`
(`feat(css): ship leftover print CSS from the v0.2.6 ledger`) on
`feature/026-extended-css-support`. Tree was clean. Product ceiling is a
print-oriented report renderer, not Chrome print.

Lenses:

- `/ponytail` full (what to delete or shrink)
- `/improve-codebase-architecture` (deep modules, seams, locality)
- architecture-deepening, extension-seams, go-practices from
  `skills/improve-codebase/`

This folder is the review ledger. It does not replace the canonical 48-56
execution ledger. Do not implement a row until the user names its ID.

## Frozen snapshot

| Field | Value |
|-------|-------|
| Branch | `feature/026-extended-css-support` |
| HEAD | `48e06dbc745dd3ea087bba1e7f007791e847d663` |
| Dirty | none |
| Date | 2026-08-28 |
| Scope | files in that commit only |
| Product | controlled-report renderer. No JS. No CGO. |

Production Go paths in the commit: `internal/css`, `internal/layout`,
`internal/convert`, `internal/imageout`, plus matrix/deferred/catalog/phase
docs.

## Executive summary

The leftover pass shipped real consumers: of-type match, attr `i`, `page:
ident` sibling breaks, named `@page` margins at paint, unnamed margin boxes
into empty CLI HF slots, `ch` from digit-zero advance, elliptical radius
slash, `column-rule`, caption-side left/right.

What the scan found is not "the CSS is fake". It is three kinds of tax:

1. **Tests that stay green if the feature dies.** Caption-side geometry
   stamps styles when cascade misses. `TestZIndexPaintOrder` never compares
   paint order. `flex-wrap: wrap-reverse` and `grid-auto-flow: column` are
   matrix Implemented with intern-only or missing layout proof.
2. **Page geometry has two homes.** Layout `PaintOptions.forPage` applies
   named + side + `:first`. Convert `hfGeom.pageMargins` skips named, so
   link/outline dests can miss the painted origin. Convert's
   `page_named.go` is margin-box chrome, not `page: ident`.
3. **Catalog and phase Status lag the code.** `mapping.json` still marks
   `@import` unsupported while `parseImportRule` + `CollectSheets` fetch.
   Phase 49 Overview still says `:is()` never matches.

No P0 security or public `Convert`/`Run*` panic was proven in this file set.

Counts: about 64 raw findings in, **23 active rows** out, **12 refused**,
**8 parked**. Cap is 25.

## Phase 1: Integrity (tests that lie, cascade that widens)

Correctness of the leftover claims. One row is one proof gap or one
widening parse.

### 1.1 Tests that lie

- [ ] **PRAC-01** Delete the caption-side stamp fallback in
      `internal/layout/caption_side_test.go:97-105`.
      `layoutCaptionSide` re-stamps interned styles when
      `captionSideApplied` is false, so Bottom/Left/Right stay green if
      apply/inherit stops. Expected: geometry tests fail when
      `CaptionSide` is not applied. Proof: `go test ./internal/layout/
      -run 'TestCaptionSide' -count=1` after removing
      `layoutCaptionSideStamped`.

- [ ] **PRAC-02** Make `TestZIndexPaintOrder` assert order.
      `internal/layout/flex_test.go:76-111` records `lowI`/`highI` on
      `res.Ops` and checks `ZIndex == 5`, then never compares indices.
      `Paint` sorts a page index slice, not `res.Ops`. Matrix §2.7 cites
      this as stacking proof. Expected: high fill paints after low, or
      sorted paint indices as in `TestPositionRelativeAbsolute`. Proof:
      `go test ./internal/layout/ -run TestZIndexPaintOrder -count=1`
      with swapped z-index must fail.

- [ ] **PRAC-03** Add a layout proof for `flex-wrap: wrap-reverse`.
      Production flag exists (`internal/layout/flex.go:208-211`). Matrix
      §2.7 line 178 is Implemented. Only intern proof is
      `shorthand_test.go:35`. Fixture-32 class `.wrap-reverse` is
      `flex-direction: row-reverse; flex-wrap: wrap`. Expected: second
      line Y is above first. Proof: new
      `go test ./internal/layout/ -run TestFlexWrapReverse -count=1`.

- [ ] **PRAC-04** Add a placement test for `grid-auto-flow: column`.
      Matrix §2.8 line 205 is Implemented. Repo CSS is intern-only
      (`grid_test.go:715` `.g { grid-auto-flow: column dense }`).
      `TestGridAutoFlowDenseFillsHole` is row vs dense. Expected: second
      auto item under the first (same X, larger Y). Proof:
      `go test ./internal/layout/ -run TestGridAutoFlowColumn -count=1`.

### 1.2 Parse that widens a narrower rule

- [ ] **EXT-01** Reject leftover tokens in `@page` preludes.
      `parsePageSelector` (`internal/css/css.go:294-314`) returns
      `pageIdent(prelude)`, which stops at `:`, so `@page chapter:first`
      stores `Sel: "chapter"`. `applyOnePageRule`
      (`internal/convert/convert_helpers.go:143-152`) then applies that
      margin on every overlapping named page. CSS asked for a narrower
      page. Expected: combined `name:first` and selector lists fail
      closed, or store name plus pseudo. Proof: `go test ./internal/css
      -count=1 -run TestParsePage` plus a convert case that must not
      apply `chapter:first` margin on a later chapter page.

- [ ] **EXT-02** Stop gluing unknown attr flags onto the comparison
      value. `splitAttrIFlag` (`internal/css/attr_iflag.go:5-7`) leaves
      `s` on the value. Tests lock the miss
      (`attr_iflag_test.go:48-50`, `:120`): `[type=foo s]` vs
      `type="foo"` is false. Selectors 4 `s` is case-sensitive exact.
      Expected: strip the one-letter flag without mutating `Value`, or
      reject the selector. Proof: `go test ./internal/css -count=1 -run
      TestAttrIFlag`.

## Phase 2: Seams (page module ownership)

The next named-page or `@page` change already has two owners.

### 2.1 Named page module

- [ ] **ARC-01** Give `page: ident` one owner in layout.
      `internal/layout/page_named.go` owns used-value inherit, sibling
      `page-break-before:always`, `namedPageNames`, and
      `PaintOptions.forPage`. `internal/convert/page_named.go` is a
      different module (unnamed `@page` quoted boxes onto CLI HF).
      Named `@page ident { margin }` lengths resolve in
      `applyCSSPageMargins` (`internal/convert/convert_helpers.go:26-36`).
      Same filename, three verbs. Expected: layout keeps ident, breaks,
      and paint cascade. Convert copies already-resolved boxes into
      `PaintOptions` and keeps HF mapping under a margin-box name.
      Proof: `rg 'page_named|applyNamedPageBreaks|applyPageMarginBoxes'`
      shows one owner per verb; convert `TestPageNamedMargins` still
      goes through `convert.Run`.

- [ ] **ARC-02** One page-box cascade for paint and link geometry.
      Layout `forPage` (`internal/layout/page_named.go:116-140`):
      unnamed, then named ident, then `:left`/`:right`, then `:first`.
      Convert `hfGeom.pageMargins` (`internal/convert/hf_geometry.go:39-60`)
      is unnamed, then side, then `:first`. Comment admits named is
      paint-only. Link/outline `pdfXY`/`pdfRect` and HF clip bands use
      the convert table. Phase 54.1.3 recorded this leftover; the
      matrix `page` row (`documentation/compatibility-matrix.md:136`)
      does not. Expected: one cascade function owned with
      `PaintOptions.forPage`; destinations include named when
      `pageNames` exist. Proof: same inputs yield the same left/top
      from both tables; a `page: chapter` plus `#id` link dest Y tracks
      painted text origin.

- [~] **EXT-03** Named and page-pseudo `@page` margin boxes parse with
      no chrome consumer. Reason: lite, documented in
      `applyPageMarginBoxes` (`internal/convert/page_named.go:10-13`)
      and matrix §4 `@page` (unnamed quoted `@top-*` / `@bottom-*`
      only). `TestParsePageMarginBoxesNamedPage` is parse-only. Owner:
      convert HF, not layout paint. Next gate: either keep `[~]` and
      stop storing `Boxes` on named/pseudo rules, or paint per-page
      chrome keyed by `forPage`. Proof if promoted: `@page chapter {
      @top-center { content: "Ch" } }` with `page: chapter` emits `Ch`
      on the named page only.

## Phase 3: Shared forks (radius, helpers, adapters)

Locality. Behavior already works on the happy path.

### 3.1 Used radius

- [ ] **ARC-03** Collapse X and Y used-radius into one helper in
      `border_radius.go`. X used values, clamp, and scale live in
      `layout_chrome.go:360-412`. Y lives in
      `border_radius.go:187-236`. Inline chrome stamps both
      (`inline_paint.go`). PDF emit clamps a third time
      (`paint.go` `clampEllipseRadii`). Imageout re-reads Op radii
      (`internal/imageout/imageout.go:1173-1194`). Expected: one
      same-package helper returns both axes; paint paths consume
      stamped `Op` radii only. No new package. Proof: `go test
      ./internal/layout/ -run 'TestRadius' -count=1`.

- [ ] **ARC-04** Stamp used radii on box-shadow fills.
      `appendBoxShadow` (`internal/layout/box_shadow.go:105-114`) emits
      `OpFillRect` with no `Radius*` fields, so rounded boxes keep
      square shadows. `stampOneOpRadiiY` only copies Y when X is
      already on the op. Expected: same used radii as `prependChrome`.
      Proof: `go test ./internal/layout/ -run 'TestBoxShadow' -count=1`
      plus a rounded-box case whose shadow ops match fill radii.

- [ ] **PRAC-05** Put elliptical longhand onto a paint op in tests.
      `TestRadiusEllipticalLonghand` (`radius_test.go:48-75`) reads
      used values only. Matrix claims `10pt / 5pt` paints elliptical
      arcs. `TestRadiusSlash` covers slash shorthand paint;
      `TestBorderRadiusReachesRoundedPaintOps` is circular
      `op.Radius`. Expected: `border-top-left-radius: 10pt / 5pt`
      fill has `RadiusTopLeftY`. Proof: `go test ./internal/layout/
      -run TestRadiusEllipticalLonghand -count=1`.

### 3.2 Convert helpers and PDF/image adapters

- [ ] **ARC-05** Resolve `@page` boxes once, next to `css.PageRule`.
      `applyOnePageRule` (`internal/convert/convert_helpers.go:117-156`)
      re-classifies raw `Sel`/`Margin`/`Size` strings. `collectUnnamedPageBoxes`
      walks `sheet.Pages` again with a different skip rule.
      `paintOptions` that feeds layout lives in `toc.go:148`. Expected:
      one walker of `Stylesheet.Pages`; typed table consumed by
      `PaintOptions`; `paintOptions` sits beside geometry, not TOC.
      Proof: `rg 'func applyOnePageRule|func collectUnnamedPageBoxes'`
      after the move; convert page tests stay green.

- [ ] **ARC-06** One font-registry scan notice.
      `logFontRegistryScan` is copied in
      `internal/imageout/imageout.go:1798` and
      `internal/convert/convert_helpers.go:366`. Convert still says it
      is used by PDF and image callers; imageout no longer imports
      package `convert`. Expected: `pdf.RegistryFromGlobal` (or a tiny
      pdf helper that takes `io.Writer`) emits once. Proof: `rg 'func
      logFontRegistryScan'` is one hit.

- [ ] **ARC-07** One flex justify helper for row and column.
      `justifyDistributed` (`flex.go:943`) and
      `justifyColumnDistributed` (`flex.go:1443`) copy remainder math.
      Ponytail shrink, not a new interface. Proof: `go test
      ./internal/layout/ -run 'TestFlex' -count=1`.

## Phase 4: Ponytail cuts

Deletion and shrink only. Ranked biggest honest cut first. Do not delete
features this commit just shipped (`ch`, elliptical Y, caption-side
left/right). Those are refused below.

- [ ] **PT-01** One walk of `@page` blocks.
      `extractPageMarginBoxes` then `stripNestedAtRules` walk the same
      block (`internal/css/css.go` around the page-rule parse). Convert
      then walks `sheet.Pages` twice (`collectPageBox` vs
      `collectUnnamedPageBoxes`). Replacement: record quoted boxes while
      dropping nested `@`; fold unnamed boxes into `collectPageBox`.
      Proof: `go test ./internal/css ./internal/convert -run
      'TestParsePageMarginBoxes|TestPageMarginBoxes' -count=1`.

- [ ] **PT-02** `ofTypeLastIndex` from `ofTypeIndex`.
      `internal/css/nth_type.go:63-96` re-walks siblings.
      Replacement: `n - ofTypeIndex(node) + 1` with the same-tag count.
      Proof: `go test ./internal/css -run 'TestLastOfType|TestNthLastOfType'
      -count=1`.

- [ ] **PT-03** One width/style/color token loop for `column-rule` and
      `outline` shorthands. `applyColumnRuleShorthand` clones
      `applyOutlineShorthand` (`style_properties.go`). Proof: `go test
      ./internal/layout -run 'TestColumnRuleParse|TestOutline' -count=1`.

## Phase 5: Docs honesty

Docs follow code. No engine change.

- [ ] **DOC-01** Flip catalog rows that already have consumers.
      `mapping.json` `@import` is `unsupported` with `code_path`
      `internal/css/css.go:187` (line 19310). Current `parseAtRule`
      routes `@import` to `parseImportRule`. Matrix §4 is Partial.
      `:is()` / `:where()` rows still unsupported while
      `appendIsWherePseudo` matches. Six `@top-*` / `@bottom-*` names
      are unsupported with empty `code_path` while
      `internal/css/page_margin.go` plus `applyPageMarginBoxes` consume
      unnamed quoted content. Expected: `@import` partial; `:is`/`:where`
      implemented; six margin-box names partial with notes matching the
      matrix. Proof: `python3 scripts/css-catalog-map.py --check`; grep
      those names vs matrix.

- [ ] **DOC-02** Rewrite phase 49-52 Status and Overview to match `[x]`
      rows. `plans/0.2.6/phases/phase-49-selectors-cascade-atrules.md:4`
      still says layout consume for `:is` is open. Overview still cites
      `css.go:122` / `css.go:1431-1434` as never-matching `:is`, and
      `css.go:187` as `@import` `skipAtRule`. Those line numbers are
      dead. Phase 52 Status is still `in progress` with every 52.x row
      `[x]`. Proof: Status line matches the last `[x]`/`[~]` row; cited
      `file:line` still names the feature.

- [ ] **DOC-03** Fix the ghost `TestGridIndependentGaps` citation.
      Matrix §2.8 line 202. No such function. Real test is
      `TestGridRowGapVsColumnGap` (`grid_test.go:360`), which does not
      pin row-gap to 8pt. Proof: grep `TestGridIndependentGaps` empty;
      matrix names the live test.

- [ ] **DOC-04** State the named-page link leftover on the matrix
      `page` row. Paint uses `geom.named`. `hfGeom.pageMargins` does
      not (`hf_geometry.go:41-42`). Phase 54.1.3 already says so.
      Proof: matrix sentence matches that comment.

## Phase 6: Closure gates

This wave is documentation only. Lint and test were not run here.

- [~] **GATE-01** Session-end gates on HEAD `48e06dbc` are still the
      2026-08-27 numbers. Reason: phase 56.2.1-56.2.4 `[x]` dated
      2026-08-27. Phase 54.4.1 already notes they were not re-run in
      the leftover pass. This commit added convert/css/layout tests
      after that date. Owner: whoever next marks 56 closed for this
      HEAD. Next gate: `make test`, `make lint`, `make golden`,
      `make claim-scan` exit 0. Do not uncheck 56 from intent.

- [ ] **GATE-02** Before closing any non-doc row above, record
      `make lint` and `make test` exit 0 on that change. Layout/paint
      rows also need `go test ./internal/layout/ -count=1` and
      `go test ./internal/convert/ -run TestGoldenCorpus -count=1`.

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

- No implementation.
- No `make lint` / `make test` / `make golden` (documentation-only review).
- No perf-review and no critical-go-review.
- No files outside commit `48e06dbc`.
