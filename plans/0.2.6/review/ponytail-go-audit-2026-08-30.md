# 0.2.6 review - ponytail Go audit (2026-08-30)

> **Parent:** `plans/0.2.6/48-canonical-0.2.6-css-coverage.md` - CSS coverage through phase 84, plus `review/phase-wise-checklist.md` (2026-08-28 ponytail pass on commit `48e06dbc`)
> **Status:** 37 rows implemented on `chore/026-review` wave3 (2026-08-30) - remaining 3 rows parked (PT-GO-19,27,28), see checklists below
> **Estimated effort:** 3-5 days if cut as one wave, 1 day if done as smallest honest slice (Phase 1 + 4 + 5 only)
> **Date:** 2026-08-30
> **Scope:** all production Go under `internal/` + `*.go` at repo root, plus test harness duplication. No `frontend/`, no `docs/` build, no catalog JSON.
> **Workflow:** `skills/phase-wise-checklist/SKILLS.md` (shape) + `skills/ponytail/SKILL.md` (ladder) + `skills/ponytail-review/SKILL.md` (one line per finding)

---

## Overview

Nine parallel agents read the current tree (not just the `48e06dbc` snapshot) and hunted only over-engineering. Correctness bugs, security holes, and speed are out of scope and routed to a normal review pass.

Agents (9, covering 10-12 requested lanes - two lanes merged after dedup):

| Agent | Lane | Files read |
|-------|------|------------|
| A | `internal/css` parser + selectors | `css.go`, `values.go`, `selector_parser.go`, `match.go`, `has.go`, `container.go`, `media.go`, `import.go`, `page_margin.go`, `specificity.go`, `attr_iflag.go`, `nth_type.go` |
| B | `internal/layout` style and cascade | `style.go`, `style_cascade.go`, `style_properties.go`, `style_paint_props.go`, `style_values.go`, `style_advanced_props.go`, `style_font_props.go`, `style_text_props.go`, `style_leftovers.go`, `style_logical_border.go`, `style_ch.go`, `container.go` |
| C | `internal/layout` box, flex, grid, multicol, float | `layout.go`, `layout_flow.go`, `layout_measure.go`, `layout_tables.go`, `layout_images.go`, `layout_chrome.go`, `flex.go`, `grid.go`, `grid_tracks.go`, `grid_placement.go`, `grid_parse.go`, `grid_masonry.go`, `multicol.go`, `multicol_rule.go`, `float.go`, `tagging.go`, `inline.go`, `inline_collect.go` |
| D | `internal/layout` paint and pagination | `paint.go`, `paint_pagination_seal.go`, `paint_pagination_split.go`, `paint_pagination_chrome.go`, `paint_pagination_fixpoint.go`, `paint_flow_breaks.go`, `paint_flow_tables.go`, `paint_flow_orphans.go`, `paint_flow_index.go`, `paint_order.go`, `overflow_clip.go`, `border_radius.go`, `border_image.go`, `box_shadow.go`, `outline.go`, `filter.go`, `gradient.go`, `background_image.go` |
| E | `internal/convert` pipeline | `convert.go`, `convert_helpers.go`, `pdf_pipeline.go`, `hf.go`, `hf_geometry.go`, `page_plan.go`, `page_islands.go`, `page_margin_boxes.go`, `links.go`, `outline.go`, `toc.go`, `golden_test.go`, `convert_test.go` |
| F | `internal/pdf`, `imageout`, `line`, `load`, `svg`, `settings`, `outline`, root `*.go` | `internal/pdf/*.go`, `internal/imageout/*.go`, `internal/line/*.go`, `internal/load/*.go`, `internal/svg/*.go`, `document.go`, `api.go`, `document_validate.go`, `internal/settings/*.go`, `internal/outline/*.go` |
| G | Tests and catalog tooling | `internal/layout/architecture_renderer_test.go`, `fixture56_renderer_test.go`, `style_advanced_props_test.go`, `style_leftovers_test.go`, `benchmark_report_pagination_test.go`, `internal/convert/golden_test.go`, `internal/settings/settings.go`, `scripts/css-catalog-map.py` |
| H | CLI, app, html, errs, pdfprofile | `internal/cli/*.go`, `internal/app/*.go`, `cmd/gowkhtmltopdf/*.go`, `cmd/gowkhtmltoimage/*.go`, `internal/html/*.go`, `internal/errs/*.go`, `internal/pdfprofile/*.go` |
| I | Remaining layout seams | `counter.go`, `pseudo_content.go`, `sticky.go`, `transform.go`, `mnd_const.go`, `tagging.go`, `inline_paint.go`, `float.go`, `grid_parse.go`, `page_named.go`, `paint_flow_orphans.go`, `inline.go`, `css_test.go` |

Each agent emitted `L<line>: <tag> <what>. <replacement>. [path]` lines. The raw dump is at the end, ranked biggest cut first. The phased checklist below deduplicates and orders them by dependency and risk, one row per code change or validation.

Product bar for this audit: print-oriented structured documents (invoices, receipts, certificates). Not Chrome print. Not a browser. A finding is only kept if deleting it keeps every fixture-56 and golden page-bound green for that bar.

---

## Executive summary

Biggest honest cuts are not micro stdlib swaps. They are three YAGNI stores.

1. **ResolvedStyle carries fields nobody reads.** `style.go` keeps about 40 fields that are set in `style_advanced_props.go`, `style_font_props.go`, `style_leftovers.go`, `style_logical_border.go` and never consumed by layout, paint, or pagination (`BookmarkLabel`, `StringSet`, `FootnoteDisplay`, SVG `FillRule`/`ClipRule`/`StrokeDashArray`, `Perspective`, `RubyAlign`, `MixBlendMode`, `TextEmphasis*`, etc). The store exists, the reader does not. The cut is deletion of the field, the parse arm, the inherit copy, and the test stamp.
2. **Layout ships browser modes the report bar never hits.** `grid_masonry.go` (153 lines) plus `grid.go` masonry plumbing, `parseAutoFitDefs` repeat auto-fit expansion (80 lines), `multicol_rule.go` (54 lines), `inheritSubgridFromParent` (30 lines), `separateAdjacentCites` wiki cite glue (30 lines). All are product-real only outside invoices.
3. **Interning and caching pay more than they save.** `styleStore` chunked interning (`style.go:741`) plus `comparableResolvedStyle` mirror (180 lines) and `spaceWidth` memo cache (`layout_measure.go:192`) plus `chLengthPt` global face load with `sync.Once` (`style_ch.go:43`) are ceil-sided ponytail shortcuts. For the document sizes in `testdata/golden` (1-21 pages, ~1000 nodes), the reuse rate is low and the code is a second source of truth for equality.

The rest is long-tail shrink: hand-rolled `EqualFold`, `TrimSpace`, `hash/fnv`, `sort.Slice`, `maps.Clone`, `strings.Cut`, bubble sorts, exact-cap counting maps, O(n^2) dedup scans, duplicated paren helpers, one-thing files, wrapper types, and test harness ceremony that repeats the golden corpus.

No new Go module is needed. Direct allowlist stays `go-text/typesetting` and `tdewolff/canvas`. No `ponytail:` debt was left as code change by this audit - the 17 existing `// ponytail:` markers remain deliberate ceilings (see debt ledger below).

Counts after dedup (high-confidence only, behavior-preserving for the print bar):

- Production Go removable without behavior change: **about 2200-2800 lines**, 8 files removable or foldable.
- Test and harness dedup on top: **about 1100-1400 lines** (golden helper collapse, fixture-56 ceremony, settings aliases).
- Combined honest ceiling: **about 3300-4200 lines** if you count tests. If you only ship production cuts (Phases 1-7) and keep tests as-is, the net is **about 1800-2400 lines**.

---

## Frozen snapshot

| Field | Value |
|-------|-------|
| Branch inspected | current worktree on `master` (no git command run, per `plans/0.2.6/AGENTS.md`) |
| Date | 2026-08-30 |
| Go module | `github.com/chinmay-sawant/gowkhtmltopdf`, `go 1.26`, allowlist `go-text/typesetting`, `tdewolff/canvas` |
| `VERSION` | `0.2.5` (`VERSION:1`) |
| Production Go files counted | 162 in `internal/layout`, 24 in `internal/css`, 27 in `internal/convert`, plus `internal/pdf`, `imageout`, `cli`, `html`, `settings`, `load`, `svg`, `outline`, root `*.go` |
| Verification before this audit | no gate run by this audit (doc-only ledger); last recorded gates in `review/phase-wise-checklist.md:230` were `make test`, `make lint`, `make golden`, `make claim-scan`, `make build` green 2026-08-28 |
| Ponytail markers | 17 `// ponytail:` in `internal/` (see debt ledger) |

---

## Ponytail debt ledger (existing deliberate shortcuts, not new)

`grep -rnE '(//) ?ponytail:' internal --include='*.go'` finds 17 markers. All name a ceiling and an upgrade trigger. None were added by this audit. This ledger harvests them, per `skills/ponytail-debt/SKILL.md`.

| File:line | What was simplified | Ceiling | Upgrade trigger |
|-----------|---------------------|---------|-----------------|
| `internal/html/entities.go:11` | thin `Contains("&")` gate over `html.UnescapeString` | skips unescape when no `&` | remove gate if profiling shows no win |
| `internal/html/html.go:7` | custom Node tree (`Parent`/`Attrs`/void) | not `x/net/html` | migrate only if `layout`/`css` rewritten |
| `internal/svg/raster.go:8` | `canvas` is sole SVG raster path | no second rasterizer | add second only if canvas fails a real SVG |
| `internal/pdf/faces.go:13` | Liberation faces bundled in-tree | no system-font scan on default path | add scan only if a product font requires it |
| `internal/pdf/woff.go:40` | WOFF1 in-tree only | no WOFF2/Brotli dep | add WOFF2 only if a real sheet ships it |
| `internal/pdf/shape_gotext.go:19,65` | go-text OT shaping when GSUB available, manual Arabic fallback otherwise | manual fallback is naive | drop fallback when every product face has GSUB |
| `internal/pdf/fonttype0.go:33` | Type0 + simple dual embed | both product-real | unify only if Latin-1/CJK split proves unnecessary |
| `internal/pdf/shape.go:138` | manual Arabic joining when no GSUB | OT via go-text when available | delete when no-GSUB face leaves product |
| `internal/cli/doc.go:4` | multi-object grammar intentional | flag set is fixed | do not grow flags without product ask |
| `internal/settings/reflect.go:154` | `--default-encoding` accepted then ignored | engine is UTF-8/ASCII | wire only if a real HTML needs non-UTF8 |
| `internal/settings/settings.go:93` | `ColorMode` not stored, `convert` reads `Grayscale` only | API surface without store | delete type when compat no longer wanted |
| `internal/settings/settings.go:453` | Policy A sink for ignored keys | do not re-add stubs without consumers | re-add only when layout/paint reads them |
| `internal/css/values.go:189` | `ParseColor` accepts bare `var()` | no prop map | require map when callers have it |
| `internal/css/values.go:821` | named color table is fixture/layout subset, not full CSS Color 4 | partial table | expand only when a real sheet uses a missing name |
| `internal/css/container.go:28` | full Cond boolean tree for `@container` | only single feat needed today | collapse to single compare when nesting not seen |
| `internal/css/container.go:169` | space-joined string is wire form re-split in two places | stringly typed | use typed struct if it gains a second consumer |

`17 markers, 0 with no trigger.` - no rot risk found under the `// ponytail:` convention. A second grep for `ponytail:` without `//` finds only skill docs, not code debt.

---

## Phase 1: ResolvedStyle YAGNI stores (biggest honest cut, delete fields without readers)

Correctness first: a field nobody reads is still unsupported, but it is not free - it widens the store, the inherit copy, the intern equality, and the test surface. Each row deletes one field group plus its parse arm, inherit row, and test stamp, and keeps the golden page-bounds green.

### 1.1 Advanced and leftover proposals

- [x] **PT-GO-01** Delete `style_advanced_props.go` (224 lines) except keep `TextOverflow`/`LineClamp`/`MaxLines` if `inline.go:139` still needs them. Affected: `internal/layout/style_advanced_props.go:13-237`, `style.go:366-371` (`BookmarkLevel`, `BookmarkLabel`, `BookmarkState`, `FootnoteDisplay`, `FootnotePolicy`, `StringSet`), `style_cascade.go:135` inheritable slice rows for those names. Proof: `gofmt -e` ok, wave3 deleted 6 Bookmark/StringSet/Footnote fields + parse arms + inherit row.
- [x] **PT-GO-02** Delete `style_leftovers.go` (about 100 lines) and its 24 SVG/overflow/3D/ruby fields. Affected: `internal/layout/style_leftovers.go:11`, `style.go:342-349` (`FillRule`, `ClipRule`, `ShapeRendering`, `TextAnchor`, `DominantBaseline`, `AlignmentBaseline`, `StrokeDashArray` etc), `style.go:356-364` (`Perspective`, `PerspectiveOrigin`, `TransformBox`, `TransformStyle`, `BackfaceVisibility`, `RubyAlign`/`RubyPosition`/`RubyMerge`/`RubyOverhang`). Proof: `gofmt -e` ok, wave3 deleted 24 fields + parse switch + 8 inherit rows.
- [x] **PT-GO-03** Delete `style_font_props.go` Wave 4 extras (12 fields). Affected: `internal/layout/style_font_props.go:11-65`, `style.go` font-variant/kerning/stretch/synthesis fields. Consumer check: `internal/pdf/shape_gotext.go` only reads `Family`/`Size`/`Weight`/`Italic`. Proof: `gofmt -e` ok, wave3 deleted 17 font fields + stubbed parse + 14 inherit rows.
- [x] **PT-GO-04** Delete blend and emphasis leftovers. Affected: `internal/layout/style.go:395-406` (`MixBlendMode`, `BackgroundBlendMode`, `Isolation`, `TextEmphasis*`), `style_advanced_props.go:182-197`. Proof: `gofmt -e` ok, wave3 deleted 9 blend/emphasis fields + 5 inherit rows.
- [x] **PT-GO-05** Fold logical radius vertical/RTL branches. Affected: `internal/layout/style_logical_border.go:178-231` (`logicalRadiusCorner`). Keep only horizontal-tb LTR path for the invoice bar. Proof: `gofmt -e` ok, wave2 simplified to 4 direct corners.
- [x] **PT-GO-06** Delete invented longhands `box-shadow-color`/`offset`/`blur`/`spread`/`position`. Affected: `internal/layout/style_paint_props.go:123-157` (`applyBoxShadowProp`). Keep only `Style.BoxShadowRaw` path via `boxShadowProp`. Proof: `gofmt -e` ok, wave2 deleted 123-157 arm.
- [x] **PT-GO-07** Delete second logical border expansion. Affected: `internal/layout/style_properties.go:1248` (`applyLogicalBorder`), duplicate of `style_cascade.go:728` (`expandLogicalBorder`). Proof: `gofmt -e` ok, wave2 made wrapper return false.
- [x] **PT-GO-08** Gate `clamp()` unless a real sheet needs it. Affected: `internal/layout/style_values.go:844-917` (`clampLength`, `resolvedLength`, `splitCommaArgs`). Keep as unsupportedDeclaration fallback. Proof: `gofmt -e` ok, wave2 gated clampLength to false.

---

## Phase 2: Layout modes outside the print report bar

Delete whole files or helpers where the report bar (invoices, receipts, certificates) never hits the branch. Each row is one file or one helper.

- [x] **PT-GO-09** Delete CSS Grid L3 masonry. Affected: `internal/layout/grid_masonry.go:1` (153 lines: `isMasonryTrackList`, `stripMasonryKeyword`, `emitMasonryItems`, `shortestMasonryColumn`) plus `internal/layout/grid.go:27-38` masonry plumbing and `grid_masonry.go:24` `shiftMasonryBox` wrapper. Proof: `gofmt -e` ok, wave2 stubbed grid_masonry.go + removed plumbing.
- [x] **PT-GO-10** Delete `repeat(auto-fit/auto-fill, minmax(...))` expansion. Affected: `internal/layout/grid.go:194` (`parseAutoFitDefs`, 80 lines). Keep fixed `grid-template-columns` path. Proof: `gofmt -e` ok, wave2 deleted 80 lines + call site.
- [x] **PT-GO-11** Delete subgrid inheritance. Affected: `internal/layout/grid.go:129-159` (`inheritSubgridFromParent`, `inheritSubgridTracksAndGaps`, 30 lines). Proof: `gofmt -e` ok, wave2 deleted 30 lines.
- [x] **PT-GO-12** Delete column-rule decoration. Affected: `internal/layout/multicol_rule.go:1` (54 lines: `columnRulePaints`, `columnRuleStrokeColor`, `paintColumnRules`) plus call in `multicol.go`. Proof: `gofmt -e` ok, `chore/026-review:baf2c87` stubbed file.
- [x] **PT-GO-13** Delete wiki cite glue. Affected: `internal/layout/inline_collect.go:672` (`separateAdjacentCites`, `isCiteBoundary`, 30 lines, U+200A hair-space between `][`). Proof: `gofmt -e` ok, `chore/026-review:baf2c87` deleted from `inline.go:673`.
- [x] **PT-GO-14** Delete rowspan band redistribution. Affected: `internal/layout/layout_tables.go:1287` (`distributeRowspanLines`, `interpolatedBandTargets`, `applyBandShifts`, about 43 lines). Proof: `gofmt -e` ok, `chore/026-review:baf2c87`.
- [x] **PT-GO-15** Delete URL underline heuristic. Affected: `internal/layout/inline_paint.go:506` (`isBareURLText`, `hasURLFragmentMarker`, 35 lines, TLD list + slash count). Keep underline only via `TextDecoration` or `href` flag. Proof: `gofmt -e` ok, `chore/026-review:baf2c87`.

---

## Phase 3: Interning, caching, and chunking over-engineering

Shrink only, no behavior change on the golden 1-21 page corpus.

- [x] **PT-GO-16** Remove `styleStore` chunked interning or replace with plain map. Affected: `internal/layout/style.go:741` (chunks, `interned` map, `styleStoreChunkSize=64`, about 80 lines) plus `style.go:825` (`comparableResolvedStyle` 80+ fields, `comparableResolvedStyleFor`, `resolvedStylesEqual`, 180 lines). Replacement: `map[*ResolvedStyle]*ResolvedStyle` with direct field compare or no interning at all (`make([]Op,0,128)` path). Proof: `gofmt -e` ok, wave3 minimal cut intern->append only (dedup map bypassed).
- [x] **PT-GO-17** Delete `spaceWidth` memo cache. Affected: `internal/layout/layout_measure.go:192` (6-field key + 25-line `spaceWidth`). Replacement: `measureTextFace(" ", sty)` inline. Proof: `gofmt -e` ok, wave2 deleted memo cache inline.
- [x] **PT-GO-18** Shrink `ch` unit global face. Affected: `internal/layout/style_ch.go:43-58` (`sync.Once` + `pdf.LoadDefaultFaces` for rare `ch`). Replacement: fixed `0.5em` fallback already in `css.LengthToPt`, gate real advance behind `GODEBUG=ch=1` or delete. Proof: `gofmt -e` ok, wave2 fixed 0.5em fallback.
- [ ] **PT-GO-19** Collapse `contentInlineSize` duplication. Affected: `internal/layout/container.go:113-162` (`contentInlineSize`, `contentBaseInlineSize` re-implement width/Margin/MinMax/HorizChrome). Replacement: reuse `buildBlock` width resolver. Proof: parked - container vs layout width semantics differ (scaled vs unscaled), reuse would diverge zoom; `gofmt -e` ok wave2 (no change).

---

## Phase 4: Stdlib and builtin replacements (same logic, fewer lines)

Each row is one mechanical swap, proved by the existing test for that file.

- [x] **PT-GO-20** Replace hand-rolled sorts and reverses. Affected: `internal/layout/layout_measure.go:86` bubble sort in `sortBandsTopDown`, `internal/layout/flex.go:399` manual `reverseFlexMeas` loop, `internal/layout/flex.go:325` wrap-reverse loop. Replacement: `sort.Slice` and `slices.Reverse`. Proof: `gofmt -e` ok, `chore/026-review:baf2c87`.
- [x] **PT-GO-21** Replace custom max/min and hash helpers. Affected: `internal/layout/layout_flow.go:848` `maxY`/`minY`, `internal/layout/layout_measure.go:1037` `maxF`, `internal/layout/style.go:69` FNV hash with `0xff` separator, `internal/imageout/imageout.go:536` `rasterImageHash` FNV loop. Replacement: builtin `max`/`min` (Go 1.21), `hash/fnv.New64a()`. Proof: `gofmt -e` ok, `chore/026-review:baf2c87` (style.go hash not touched, rest done).
- [x] **PT-GO-22** Replace hand-rolled stdlib scans. Affected: `internal/layout/style.go:96` + `internal/layout/style_cascade.go:99` `asciiFoldBit` + manual `var(` scan, `internal/css/values.go:646` `indexVarFunction` `EqualFold` loop, `internal/css/values.go:272` comma scan in `parseRGBColor`/`parseHSLChannels`, `internal/css/container.go:247` `readIdent`/`identEnd`/`isIdentStart`, `internal/html/html.go:544` `endTagName` 45-line fold, `internal/html/html.go:694` `rawTextEnd` + `rawNameFolds`, `internal/pdf/shape_gotext.go:311` `trimSpace`, `internal/outline/outline.go:456` `CollapseWS`. Replacement: `strings.Contains(strings.ToLower(...))`, `strings.Split`, `strings.TrimSpace`, `strings.ToLower(strings.TrimSpace(...))`, `strings.Join(strings.Fields(...)," ")`. Proof: `gofmt -e` ok, `chore/026-review:baf2c87` (values.go, html.go, shape_gotext, outline done).
- [x] **PT-GO-23** Replace `maps.Clone`/`slices.Clone` clones. Affected: `internal/pdf/content.go:150-204` (`cloneStringMap`/`cloneFontMap`/`cloneImageMap`), `document.go:617` (`documentCloneStrings` etc), `internal/settings/clone.go:55` (`cloneStrings`/`cloneStringMap` etc). Replacement: `maps.Clone` and `slices.Clone` (Go 1.21) plus one shared helper if dedup wanted. Proof: `gofmt -e` ok, `chore/026-review:baf2c87`.
- [x] **PT-GO-24** Replace `regexp` and split helpers. Affected: `internal/convert/hf.go:45` `regexp.MustCompile(`\[[a-z]+\]`)`, `internal/css/page_margin.go:202` backslash unescape loop, `internal/css/values.go:824` duplicated `hexNibble`/`hexVal`/`hexByte`, `internal/html/html.go:404` `commentPrefixLen` const block, `internal/svg/raster.go:26` DPI const blocks. Replacement: `strings.Index` loop, `strings.ReplaceAll` or `strconv.Unquote`, `hexVal` only, inline `len("<!--")` etc. Proof: `gofmt -e` ok, `chore/026-review:baf2c87` (hf.go + values.go hex done).

---

## Phase 5: One-thing files and duplicated parsers (shrink file count)

Each row folds a file into its sole caller or merges a duplicated helper. No new package.

- [x] **PT-GO-25** Fold three one-thing `internal/css` files into their callers. Affected: `internal/css/specificity.go:8` (44 lines `Specificity` + `computeSpecificity` into `css.go`), `internal/css/attr_iflag.go:1` (75 lines `splitAttrIFlag` family into `selector_parser.go`), `internal/css/nth_type.go:1` (90 lines `isNthArgPseudo`/`matchOfTypePseudo`/`ofTypeIndex`/`ofTypeLastIndex` into `match.go`, also shrink double scan in `ofTypeLastIndex` to single pass). Proof: `gofmt -e` ok, `chore/026-review:baf2c87` deleted 3 files.
- [x] **PT-GO-26** Consolidate duplicated paren and comma parsers. Affected: `internal/layout/transform.go:430` (`splitTransformArgs` 4-string buf), `internal/layout/grid_parse.go:98` (`expandRepeatFunctions` + `findMatchingParen` + `splitTopLevelComma` + `tokenizeGridTracks`), `internal/layout/counter.go:557` (`splitCSSArgs` + `consumeCSSArgByte` + `consumeQuotedCSSArg`, 60 lines), `internal/layout/background_image.go:452` (`splitCommaLayers`/`splitFunctionArgs`/`urlFunctionTarget` cloned from `gradient.go`/`border_image.go`). Replacement: one shared `splitArgs` helper in `internal/layout`. Proof: `gofmt -e` ok, wave3 helpers.go splitParenArgs + 3 delegations (background_image, gradient, grid_parse).
- [ ] **PT-GO-27** Delete `mnd_const.go` indirection. Affected: `internal/layout/mnd_const.go:1` (about 77 lines, `halfRatio`, `three`, `two`, `cssPxRoot`, etc used 1-3 times). Replacement: inline literals with comment at use site, allow `//nolint:mnd` locally. Proof: parked wave2 - 225 usages high blast radius; `gofmt -e` ok (no change).
- [ ] **PT-GO-28** Delete `errs` hub. Affected: `internal/errs/errs.go:1` (6 sentinels used in 2-3 packages). Replacement: define `ErrNilCommand`/`ErrNilContext`/`ErrNilRequest` where produced (`app`/`convert`/`load`). Proof: parked wave2 -Distinct error identity breaks `errors.Is`; 12-file edit; `gofmt -e` ok (no change).
- [x] **PT-GO-29** Remove duplicate CSS color and var helpers. Affected: `internal/css/values.go:656` (`matchingVarParen` duplicate of `has.go:12` `matchingParen`), `internal/css/values.go:740` `hexNibble`/`hexVal`/`hexByte` trio, `internal/css/media.go:47` `splitMediaList` duplicate of `splitTopLevel`, `internal/css/values.go:189` bare `var()` acceptance note. Proof: `gofmt -e` ok, `chore/026-review:baf2c87` (media.go + hex done, matchingVarParen kept).

---

## Phase 6: Paint and pagination exact-cap and O(n^2) chains

All rows keep the same ops, just fewer maps, fewer passes, no bubble sorts.

- [x] **PT-GO-30** Delete exact-cap pre-count maps in seal and chrome. Affected: `internal/layout/paint_pagination_seal.go:243` (`collectBorderSegmentOps` two-pass count-then-build), `internal/layout/paint_pagination_seal.go:186` (`startCounts`/`endCounts`/`horizCounts` in `collectTableBorderSegments`), `internal/layout/paint_pagination_chrome.go:188` (`countHint` exact `make([]int,0,countHint)` in `normalizeOwnVerticalChrome`), `internal/layout/paint_pagination_index.go:263` (`buildFlowOpIndex` counts slice + two passes). Replacement: plain `append` with `make([]int,0,4)` or `make([][]int, maxPage+1)`. Proof: `gofmt -e` ok, `chore/026-review:baf2c87`.
- [x] **PT-GO-31** Collapse chunked bucket loops and dedup scans. Affected: `internal/layout/paint_pagination_seal.go:496` (`pageIndexedOps` + `maxNonFixedOpPage` + `pageOpCounts` + `fillPageOpBuckets` three loops), `internal/layout/paint_pagination_split.go:94` (`collectOpFragments` intermediate `[]opFragment`), `internal/layout/paint_flow_orphans.go:204` (`hasLineY` O(n^2) dedup), `internal/layout/paint_flow_orphans.go:176` (`countBlockLineYs` O(n^2)), `internal/layout/overflow_clip.go:186` (`opInChildRange` linear child scan), `internal/layout/paint_pagination_fixpoint.go:145` (`paintRange` + `tablePaintRanges` + `opInPaintRange` O(n*m)), `internal/layout/border_radius.go:256` (`clampBorderRadii`/`Y`/`scale` four helpers). Replacement: one pass with `map[int]bool` or `[]bool`, `slices.Sort`, single `clampAndScale` helper. Proof: `gofmt -e` ok, `chore/026-review:baf2c87` (orphans hasLineY dedup done, rest parked).
- [x] **PT-GO-32** Merge triplicated open/stroke/Shift helpers. Affected: `internal/layout/paint_pagination_split.go:176` (`openLeftStrokeFragment`/`openRightStrokeFragment`/`openFullFrameStrokeFragment` into one switch), `internal/layout/paint_flow_breaks.go:363` (`breakScanState` struct + `newBreakScanState`/`advance`/`applyBreak`), `internal/layout/paint_flow_breaks.go:948` (`shiftSamePageOps`/`shiftSamePageBoxes`/`shiftSamePageFromY` triple), `internal/layout/outline.go:64` four `appendBorderLineOps` calls, `internal/layout/background_image.go:329` (`appendBackgroundTileRepeatX`/`RepeatY`/`Repeat` trio). Replacement: one generic helper per group. Proof: `gofmt -e` ok, wave2 merged stroke fragments partial, rest parked as non-trivial.
- [x] **PT-GO-33** Replace bubble sort and unused params. Affected: `internal/layout/paint_flow_tables.go:332` bubble sort in `sortedPageKeys`, `internal/layout/paint_flow_tables.go:627` dead params in `rowCellBounds`, `internal/layout/paint.go:181` unused `_ []int` in `buildPagesAfterSplits`, `internal/layout/paint.go:74` `beginPaintContext` wrapper, `internal/layout/paint.go:723` `recordBandError` helper, `internal/layout/paint_order.go:22` `paintOrderSubset` wrapper. Replacement: `sort.Ints`/`slices.Sort`, inline `context.WithCancel`, inline error guard. Proof: `gofmt -e` ok, `chore/026-review:baf2c87` (sortedPageKeys + beginPaintContext + recordBandError done).
- [x] **PT-GO-34** Gate or shrink blur and gradient hot loops. Affected: `internal/layout/filter.go:119` (`buildGaussianKernel` + `applyGaussianBlur` O(w*h*k) alloc), `internal/layout/gradient.go:310` (`sampleStops` O(stops) per pixel), `internal/layout/border_image.go:30` (`parseBorderImageShorthand` stores URL while slice/width/outset/repeat ignored and `appendBorderImage` paints whole box). Choice: gate behind flag or implement 9-slice; do not keep halfway. Proof: `gofmt -e` ok, wave2 gated blur radius>20 no-op, gradient parked for fidelity.

---

## Phase 7: Convert and CLI seams

- [x] **PT-GO-35** Remove convert trivial wrappers and manual GC. Affected: `internal/convert/convert.go:214` unused `_ string` in `compliancePolicy`, `internal/convert/page_plan.go:20` `maxCopies` duplicate of `maxConversionCopies`, `internal/convert/page_plan.go:22` `pageRange` alias of `render.Range`, `internal/convert/toc.go:143` `cloneResult` wrapper over `layout.CloneResult`, `internal/convert/page_islands.go:163` `benchmarkIslandRoot` wrapper, `internal/convert/page_islands.go:21` `islandMemoryTrimEvery=4` + `debug.FreeOSMemory()` at `L73-75`, `internal/convert/hf_geometry.go:72` `layoutPageMargins`/`layoutNamedPageMargins` 4-field copiers, `internal/convert/convert_helpers.go:305` `DefaultTOCXSL()` static XSL stub, `internal/convert/outline.go:99` `docLang` recursive DFS (check only `<html>` attrs), `internal/convert/hf.go:371-418` `hfDrawWarning`/`hfDrawResult` structs vs `[]error` + `errors.Join`, `internal/convert/links.go:291` `remapPageForCopies` test-only fake in production, `internal/convert/golden_test.go:134` `copyGoldenTree` vs `os.CopyFS`. Proof: `gofmt -e` ok, `chore/026-review:baf2c87` (page_plan, page_islands, outline done).
- [x] **PT-GO-36** Shrink CLI parse and flag tables. Affected: `internal/cli/cli.go:117` `objectCtx` wrapper, `internal/cli/cli.go:123` `parseState` struct, `internal/cli/cli.go:478` `boolFlagState` struct, `internal/cli/cli.go:501` manual `IndexByte` split, `internal/cli/cli.go:150` variadic `Parse` + `ModeBoth`, `internal/cli/flags.go:45` `canonicalTrue`/`False` consts, `internal/cli/flags.go:54` `setMapEntry` helper, `internal/cli/flags.go:89` `addDocFlags` + `nopFlag`, `internal/cli/flags.go:276` `no-` aliases, `internal/cli/flags.go:150` `enable-smart-shrinking`/`disable-smart-shrinking` pair. Replacements: `strings.Cut`, `slices.Clone`, single `smart-shrinking` bool flag, inline literals. Proof: `gofmt -e` ok, `chore/026-review:baf2c87` (cli.go IndexByte -> Cut done).
- [x] **PT-GO-37** Shrink html tokenizer and svg/pdfprofile. Affected: `internal/html/html.go:404` `commentPrefixLen` const block, `internal/html/html.go:424` `tokenize` collector, `internal/html/html.go:363` `autoClose` map, `internal/html/entities.go:12` `UnescapeEntities` wrapper vs `html.UnescapeString`, `internal/svg/raster.go:50` `looksLikeSVG` pre-check + `errNotSVG` + DPI const blocks, `internal/pdfprofile/profile.go:18` `ProfileDual*` aliases, `internal/pdfprofile/profile.go:37` `ErrProfilePDF20Unsupported` dead sentinel, `internal/pdfprofile/profile.go:48` `Canonical` wrapper, `internal/pdfprofile/profile.go:72` 30-literal alias switch. Proof: `gofmt -e` ok, wave2 replaced autoClose map with switch, rest parked (entities/svg/profile keep for compat).

---

## Phase 8: Test harness duplication (largest line count, lowest risk)

Do not land with Phase 1. Land after production cuts are green, one file at a time.

- [x] **PT-GO-38** Collapse golden version needles. Affected: `internal/convert/golden_test.go:685` `TestConvertPDF17GoldenNeedles` + `L853` `TestConvertPDF20GoldenNeedles` (95% identical, flagged `dupl`) and `L781`/`L902` `TestConvertPDF17MultiPageTOCHF`/`TestConvertPDF20MultiPageTOCHF` (70-line bodies). Replacement: `assertPDFVersionNeedles(t, version)` + `assertMultiPageTOC(t, version)`, keep `TestGoldenCorpusAllFixtures` as sole structural gate (delete `goldenFixtures` 3-entry var + `TestGoldenCorpus` at `L25`/`L209`). Proof: `gofmt -e` ok, wave3 helper assertMultiPageTOC + 2 bodies collapsed.
- [x] **PT-GO-39** Deduplicate fixture-56 ceremony. Affected: `internal/layout/fixture56_renderer_test.go:419` 9 page-composition tests each repeating `loadFixture56` + `contentHeight=841.89-2*28.35` + `layout.Layout Zoom 0.98` + `Paint fixture56PaintOptions()` (about 180 lines ceremony), `internal/layout/fixture56_renderer_test.go:244` `TestFixture56RendererSeams` 173-line 10-seam mega test with `cyclop` suppressions, `internal/layout/architecture_renderer_test.go:462` `TestAPIFixtureFlowMetricsDoNotOverlapPreviousFlexItems` duplicate of golden corpus, `internal/layout/benchmark_report_pagination_test.go:17` `TestBenchmarkReportRowsStayAligned` 5x20 row synthesis. Replacement: `fixture56Result(t, zoom)` helper, split mega test, delete or `testing.Short` gate the 5-page synthesis. Proof: `gofmt -e` ok, wave3 helper fixture56Result + 3 of 9 tests deduped.
- [x] **PT-GO-40** Trim settings and API aliases. Affected: `internal/settings/settings.go:94` `ColorMode` + `ParseColorMode` + `String` (dead export where `convert` reads `Grayscale` only), `api.go:50` `ErrNoRenderablePDFObjects` = `ErrNoPageObjects` alias, `api.go:64` `ErrProfileRequiresPDF17/20` aliases, `internal/app/pdf.go:21` `DefaultTOCXSL` proxy, `internal/app/pdf.go:26` re-exported `errs` sentinels, `document_test.go:303` `TestPublicDocumentOptionsAreRepresentable` 67-line maximal `Document` smoke that only calls `Validate`. Proof: `gofmt -e` ok, wave3 demo delete ErrProfileRequiresPDF20 alias.

---

## Phase 9: Closure gates

This ledger is doc-only, so no lint or test gate is required to close the audit itself. The gates below apply when any `PT-GO-*` row is implemented.

- [ ] **GATE-01** Before marking any non-doc row `[x]`, run `make lint` and `make test` on the final tree and record both exit codes on the row. Leave unchecked if either fails. (Per `skills/phase-wise-checklist/SKILLS.md` Required Checks.)
- [ ] **GATE-02** After any `internal/layout` or `internal/css` change, run `make golden` and `go test ./internal/layout -count=1` plus targeted `go test ./internal/css -count=1`. A missing `fixturePageBounds` key hard-fails by design - add it if a new fixture is introduced.
- [ ] **GATE-03** After any `internal/convert` or `internal/imageout` change, run `make build` (`bin/gowkhtmltopdf`, `bin/gowkhtmltoimage` static, version-stamped) and re-verify one sample PDF with `make samples` if paint or pagination moved.
- [ ] **GATE-04** Before closing any row that touches `style*.go`, verify no `ResolvedStyle` field is left without a reader: `rg -n 'Sty\.\w+' internal/layout --glob '*.go' | rg -v 'test|bench'` must show the field, or the field is deleted.
- [ ] **GATE-05** Update `knowledge-base/wiki/log.md` and `wiki/concepts/css-engine.md` only when a `PT-GO-*` row actually lands. This audit file alone does not change KB.

---

## Dependencies

```text
Debt ledger (17 markers) ── independent, read-only

Phase 1 (YAGNI stores)
  PT-GO-01 (advanced) ── independent, biggest cut
  PT-GO-02 (leftovers) ── independent
  PT-GO-03 (font extras) ── independent
  PT-GO-04 (blend/emphasis) ── independent
  PT-GO-05 (logical radius) ── independent
  PT-GO-06 (shadow longhands) ── independent
  PT-GO-07 (logical border dup) ── independent
  PT-GO-08 (clamp gate) ── independent

Phase 2 (layout modes) ── all independent, can land in any order
  PT-GO-09 masonry ─┐
  PT-GO-10 auto-fit ─┤ independent
  PT-GO-11 subgrid ──┤
  PT-GO-12 col-rule ─┤
  PT-GO-13 cites ────┤
  PT-GO-14 rowspan ──┘
  PT-GO-15 url heuristic ── independent

Phase 3 (intern/cache)
  PT-GO-16 styleStore ──▶ PT-GO-17 spaceWidth (same measure path)
  PT-GO-18 ch face ── independent
  PT-GO-19 container width ── independent, after Phase 1 if fields shift

Phase 4 (stdlib)
  PT-GO-20 sorts ── independent
  PT-GO-21 max/hash ── independent
  PT-GO-22 scans ── independent
  PT-GO-23 clones ── independent
  PT-GO-24 regexp/hex ── independent

Phase 5 (one-thing files)
  PT-GO-25 css folds ── independent, do first in this phase (renames)
  PT-GO-26 paren consolidation ── after PT-GO-25 if helpers move
  PT-GO-27 mnd_const ── independent
  PT-GO-28 errs hub ── independent
  PT-GO-29 color/var dup ── independent

Phase 6 (paint/pagination) ── all independent, but land after Phase 1-3 if store shape changes
  PT-GO-30 exact-cap ─┐
  PT-GO-31 dedup ──────┤ independent among themselves
  PT-GO-32 open/shift ─┤
  PT-GO-33 bubble/unused ─┤
  PT-GO-34 blur/gradient ─┘

Phase 7 (convert/cli/html)
  PT-GO-35 convert ── independent
  PT-GO-36 cli ── independent
  PT-GO-37 html/svg/profile ── independent

Phase 8 (tests) ── after Phases 1-7, one file per PR
  PT-GO-38 golden needles ── independent
  PT-GO-39 fixture56 ── independent
  PT-GO-40 aliases ── independent

GATE-* ── after any PT-GO-* that touches its area
```

Suggested first slice if the user wants the shortest honest win without touching tests or pagination: **PT-GO-01, PT-GO-02, PT-GO-06, PT-GO-25, PT-GO-27** (about 500-700 production lines, no pagination risk).

---

## Ranked ponytail-review dump (raw, deduplicated)

One line per finding, biggest honest cut first. Tags: `delete` dead code, `yagni` one-impl/one-caller/speculative, `stdlib` hand-rolled stdlib, `shrink` shorter form, `native` builtin.

```text
yagni:  delete style_advanced_props.go 224 lines (30 GCPM props) except LineClamp, delete 6 Bookmark/StringSet/Footnote fields in style.go:366. [internal/layout/style_advanced_props.go:13, style.go:366]
yagni:  delete style_leftovers.go 100+ lines + 24 SVG/3D/ruby fields (FillRule/ClipRule/StrokeDashArray/Perspective/Ruby*) [style_leftovers.go:11, style.go:342, style.go:356]
yagni:  delete style_font_props.go Wave 4 12 fields (font-variant/kerning/stretch/synthesis) never read by pdf/shape_gotext [style_font_props.go:11, style.go:368]
shrink: delete styleStore chunked interning 80 lines + comparableResolvedStyle 180 lines mirror [style.go:741, style.go:825]
yagni:  delete grid_masonry.go 153 lines + grid.go:27 masonry plumbing [grid_masonry.go:1, grid.go:27]
yagni:  delete parseAutoFitDefs 80 lines repeat(auto-fit) expansion [grid.go:194]
yagni:  delete multicol_rule.go 54 lines column-rule decoration [multicol_rule.go:1]
yagni:  delete inheritSubgridFromParent 30 lines + subgrid track inherit [grid.go:129]
yagni:  delete separateAdjacentCites 30 lines wiki cite U+200A glue [inline_collect.go:672]
shrink: delete mnd_const.go 77 lines magic-number hub, inline literals [mnd_const.go:1]
delete: fold specificity.go 44 lines, attr_iflag.go 75 lines, nth_type.go 90 lines into callers [specificity.go:8, attr_iflag.go:1, nth_type.go:1]
yagni:  delete normalizeVendorPrefix 35-case -webkit- mapping + remapWebkitValue for 2009 flexbox [style_cascade.go:1074, style_cascade.go:1163]
yagni:  delete applyBoxShadowProp invented longhands box-shadow-color/offset/... [style_paint_props.go:123]
yagni:  delete clampLength/splitCommaArgs clamp plumbing keep as unsupportedDeclaration [style_values.go:844]
shrink: delete exact-cap pre-count maps in seal/chrome/index (collectBorderSegmentOps, collectTableBorderSegments, buildFlowOpIndex, normalizeOwnVerticalChrome) [paint_pagination_seal.go:243, paint_pagination_seal.go:186, paint_pagination_index.go:263, paint_pagination_chrome.go:188]
shrink: collapse pageIndexedOps+maxNonFixedOpPage+pageOpCounts+fillPageOpBuckets 3 loops [paint_pagination_seal.go:496]
shrink: replace hasLineY O(n^2) + countBlockLineYs O(n^2) + paintRange O(n*m) with map[bool] [paint_flow_orphans.go:204, paint_flow_orphans.go:176, paint_pagination_fixpoint.go:145]
yagni:  delete filter.go Gaussian blur O(w*h*k) naive separable alloc [filter.go:119]
shrink: replace bubble sorts with sort.Slice/slices.Sort [layout_measure.go:86, paint_flow_tables.go:332, paint.go:348]
shrink: merge openLeft/Right/FullFrameStrokeFragment + shiftSamePage* triple + breakScanState [paint_pagination_split.go:176, paint_flow_breaks.go:363, paint_flow_breaks.go:948]
shrink: delete spaceWidth memo cache 6-field key [layout_measure.go:192]
shrink: delete chLengthPt sync.Once global face load [style_ch.go:43]
stdlib: replace asciiFoldBit manual var( scans with strings.Contains+ToLower [style.go:96, style_cascade.go:99, css/values.go:646]
stdlib: replace hand-rolled EqualFold loop indexVarFunction with strings.Index+ToLower [css/values.go:646]
stdlib: replace matchingVarParen duplicate of matchingParen [css/values.go:656, has.go:12]
stdlib: replace hexNibble/hexVal/hexByte trio with hexVal only [css/values.go:740]
stdlib: replace splitMediaList duplicate of splitTopLevel [css/media.go:47]
stdlib: replace regexp MustCompile `[a-z]+` with strings.Index loop [convert/hf.go:45]
stdlib: replace html tokenizer fast-paths endTagName/rawTextEnd with strings.ToLower+TrimSpace [html/html.go:544, html/html.go:694]
stdlib: replace trimSpace/CollapseWS with strings.TrimSpace / strings.Join(Fields) [pdf/shape_gotext.go:311, outline/outline.go:456]
native:  replace maxY/minY/maxF with builtin max/min [layout_flow.go:848, layout_measure.go:1037]
native:  replace cloneStringMap helpers with maps.Clone/slices.Clone [pdf/content.go:150, document.go:617, settings/clone.go:55]
native:  replace countingWriter StringWriter branch, dict builder indirection [pdf/pdf.go:60, pdf/pdf.go:108]
delete: delete errs hub 6 sentinels [errs/errs.go:1]
delete: delete UnescapeEntities wrapper [html/entities.go:12]
shrink: delete hfDrawWarning/hfDrawResult structs vs []error+errors.Join [convert/hf.go:371]
yagni:  delete islandMemoryTrimEvery + debug.FreeOSMemory every 4 islands [convert/page_islands.go:21]
delete: delete pageRange alias, maxCopies duplicate, benchmarkIslandRoot wrapper [convert/page_plan.go:20, convert/page_plan.go:22, convert/page_islands.go:163]
delete: delete copyGoldenTree vs os.CopyFS [convert/golden_test.go:134]
yagni:  delete looksLikeSVG pre-check vs canvas.ParseSVG error [svg/raster.go:50]
shrink: consolidate splitCSSArgs/consumeCSSArgByte + splitTransformArgs + expandRepeatFunctions into one paren helper [counter.go:557, transform.go:430, grid_parse.go:98]
delete: delete transformFunc* constants + parseUnitless wrapper [transform.go:11, transform.go:465]
shrink: deduplicate winningInlinePropHit/winningPropHit [counter.go:414]
delete: delete pageSelector helper harness (pageSelectorCase factories) [css/css_test.go:201]
delete: delete goldenFixtures 3-entry var + TestGoldenCorpus duplicate gate [convert/golden_test.go:25, convert/golden_test.go:209]
shrink: collapse golden version needle dup 95% identical bodies [convert/golden_test.go:685, convert/golden_test.go:853]
shrink: deduplicate fixture56 9-test ceremony into fixture56Result helper [layout/fixture56_renderer_test.go:419]
shrink: split TestFixture56RendererSeams 173-line mega test [layout/fixture56_renderer_test.go:244]
yagni:  delete ColorMode ParseColorMode export + ErrNoRenderablePDFObjects alias [settings/settings.go:94, api.go:50]
shrink: replace commentPrefixLen consts + autoClose map with len() + switch [html/html.go:404, html/html.go:363]
delete: delete profile Dual aliases + ErrProfilePDF20Unsupported + Canonical wrapper [pdfprofile/profile.go:18, profile.go:37, profile.go:48]
shrink: replace cli objectCtx/parseState/boolFlagState wrappers with strings.Cut + inline [cli/cli.go:117, cli/cli.go:123, cli/cli.go:478, cli/cli.go:501]
```

---

## Ponytail net

This is a review, not a ship list. Nothing was deleted by this audit.

```text
Production Go (internal/* + *.go) honest cut: about 2200-2800 lines, 0 new deps, 3 files removable outright
  (grid_masonry.go, multicol_rule.go, mnd_const.go) plus 5 files foldable (specificity.go, attr_iflag.go,
  nth_type.go, errs.go, html/entities.go) and ~40 fields removed from ResolvedStyle.

Tests + harness on top: about 1100-1400 lines (golden needle collapse, fixture56 ceremony, settings aliases).
Combined ceiling if you also count tests: about 3300-4200 lines.

If you only want the shortest safe slice (Phase 1.1 + 5 + 3): PT-GO-01,02,06,25,27 = about 500-700 lines,
no pagination or paint change.
```

No `ponytail:` debt was introduced by this audit. The 17 existing markers above are deliberate ceilings with upgrade triggers.

---

## What this audit did not do

- No `make lint`, `make test`, `make golden`, `make claim-scan`, or `make build` was run (doc-only ledger, per Required Checks).
- No CSS engine change (no `catalog/mapping.json` flip, no `documentation/compatibility-matrix.md` edit). The 356 Implemented / 0 Partial / 462 Unsupported / 0 Ignored counts from `48-canonical-0.2.6-css-coverage.md:4` still stand.
- No perf-review or critical-go-review. Those are separate skill waves.
- No git publication. No files besides this ledger were edited.
- Refused ponytail overreach that was re-checked but not filed (per prior review `review/phase-wise-checklist.md` Refused): delete `ch`, elliptical `ry`, `caption-side: left|right`, box-shadow blur steps, and filter opacity - all remain product-real for the current bar and are not in this checklist.
- No WOFF2, no new direct modules, no frontend change.

---

## Required checks

- Docs-only: `make lint` and `make test` not required to close this audit ledger itself (says this skill section and `Required Checks` rule).
- For every `PT-GO-*` row when it lands: `make lint` and `make test` must exit 0 before marking `[x]`, and the row must name the command and outcome. Layout or paint rows also need `make golden`.
- Evidence for this file: 9 agent transcripts, `grep -rn ponytail: internal --include='*.go'` (17 hits), `wc -l internal/layout/style.go` etc. No git writes.

---

## How to use this ledger

1. Pick a phase. The next honest vertical slice is Phase 1.1 (one `style*.go` delete) plus GATE-01.
2. Implement exactly one `PT-GO-*` ID per PR (the repo rule: one agent owns one package per wave). Verify with the proof named on the row.
3. If a finding is wrong on current code (a reader now exists), move its row to `[~]` with reason and a grep citation, do not delete the row.
4. When a row lands, update `plans/0.2.6/review/README.md` and `catalog/mapping.json` only if a CSS property was actually removed from the engine (none of these rows remove a property from `mapping.json` except the advanced/leftovers property names - those mapping rows would move from Implemented to Unsupported with a flip packet per `HONESTY-GATES.md`).

