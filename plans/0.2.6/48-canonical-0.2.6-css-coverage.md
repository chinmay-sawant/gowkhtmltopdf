# 48 - v0.2.6 CSS coverage (Canonical Execution Ledger)

> **Parent:** `plans/0.2.5/40-canonical-0.2.5-python-bindings.md` (complete 2026-08-26). Leftover CSS rows under `plans/0.2.0/` move here with `[~]` pointers.
> **Status:** phases 48-79 largely complete (Phase 79 closed Partial→Implemented). Catalog baseline 2026-08-29: **202 Implemented / 0 Partial / 616 Unsupported / 0 Ignored**. Phases **80-84** are the Unsupported triage program (not started). VERSION still 0.2.5.
> **Estimated effort:** several weeks across phases 48-56. Catalog and honesty first. Frequency slices next. Layout deepen last.
> **Constraint:** pure Go, no CGO on the default path, no browser embed, no JavaScript. Direct modules stay `go-text/typesetting` and `tdewolff/canvas` unless an amendment is filed. `catalog/mapping.json` is the CSS name inventory. Agents must follow `HONESTY-GATES.md` before any Implemented flip.
> **Ordering principle:** freeze catalog and honesty docs first, then selectors and cascade, then values and units, then template-visible box/text/paint, then generated content, then paged media, then fixture-driven layout leftovers, then closure. No phase closes on intent.
> **Workflow:** `skills/phase-wise-checklist/SKILLS.md`

---

## Overview

The engine already parses more CSS than it honors. `internal/css` keeps almost any lowercase property name. `applyIgnoredGroup` in `internal/layout/style_properties.go:1341` then drops what layout does not model. About 120 names have apply handlers. Webref lists 818 properties.

v0.2.5 shipped Python bindings. CSS work from v0.2.0 (flex/grid lite, `:has()`, `@container` size, 2D transforms, multicol lite) is done as a *subset*. This ledger is the remaining coverage program: get the rest of print-useful CSS onto `ResolvedStyle` and into layout/paint, or mark it ignored on purpose.

Not a browser. Not Chrome print. Success is more authored-template CSS hitting Implemented, fewer silent drops, goldens green, matrix matching code.

Knowledge base: `knowledge-base/wiki/index.md`, `concepts/css-engine.md`, `compatibility.md`, `syntheses/roadmap.md`. Committed contract: `documentation/compatibility-matrix.md`. Architecture: `documentation/architecture/06-css.md`.

### In scope

1. Frozen catalogs under `plans/0.2.6/catalog/` and a mapping of every webref name to engine status.
2. Selectors and at-rules that real sheets use and currently never match (`:is()`, `:where()`, `@import` under ACL).
3. Values and units that currently drop or lie (`clamp()`, `hsl()`, logical margin/padding, `ex`/`ch` honesty, `pre-wrap`).
4. Template-visible properties: `word-spacing`, `visibility`, `caption-side`, `background-image`, `outline`, radius longhands, overflow clip lite.
5. Generated content beyond quoted strings: counters, `quotes`, `list-style-position`.
6. Paged media leftovers: named `@page`, page selectors, break value honesty.
7. Fixture-driven flex/grid/multicol Partial holes. No Chrome layout tests.
8. Matrix, fidelity, mapping, and KB kept in lockstep.

### Hard non-goals (unless this ledger is amended)

- JavaScript / Phase 22
- Chrome or Wikipedia visual parity / Phase 23
- Pixel-diff goldens as the default gate
- Animation, transition, `@keyframes`, 3D transforms, `perspective`, filter blur/drop-shadow
- Scroll snap, anchor positioning, view transitions, speech
- Full Flexbox / Grid L1/L3 / joint subgrid intrinsic / CSS Grid L3 masonry
- CGO HarfBuzz, bundling Noto CJK, new direct modules without sign-off
- WOFF2 decode: sibling track, still rejected in this worktree (`internal/pdf/woff.go`). Not this ledger unless amended.

---

## Executive Summary

| Fact (current evidence) | Location |
|-------------------------|----------|
| `VERSION` is `0.2.5` | `VERSION:1` |
| CSS parser keeps unknown ident names; layout drops them | `internal/css/values.go` `validPropName`; `applyIgnoredGroup` `style_properties.go:1341` |
| Apply dispatch is 11 groups, not one switch in `style.go` | `style_cascade.go:711-722`, `style_properties.go` |
| About 120 named properties have apply handlers | inventory 2026-08-27 against `style_properties.go` + `applyFontProps` |
| Webref catalog: 818 properties, 55 at-rules, 158 selectors, 162 functions | `catalog/webref-css.json` |
| First mapping: 75 implemented, 45 partial, 488 unsupported, 210 ignored | `catalog/coverage-summary.json` |
| At-rules with any engine effect: `@media`, `@container`, `@page`, `@font-face` (all Partial) | `internal/css/css.go:172-189` |
| `:is()` / `:where()` parse as unknown, never match | `css.go:122`, `css.go:1431-1434` |
| `word-spacing`, `caption-side`, `background-image`, `outline`, `visibility`, `box-shadow` have no apply arms | `style_properties.go` |
| `table-layout: fixed` is consumed lite; matrix still says "auto only" | `layout_tables.go:45` vs `compatibility-matrix.md` §2.5 |
| `ex` / `ch` resolve as 0.5em; matrix says declaration dropped | `internal/css/container.go:133-134` vs matrix §3 |
| `clamp(` / `color-mix(` / `light-dark(` / `oklch(` excluded from cascade | `style_cascade.go:517-529` |
| Product bar remains authored HTML templates, not Chrome print | `documentation/fidelity.md`, `documentation/deferred.md` |
| Direct module allowlist: two names | `internal/pdf/shape_test.go` `TestDirectModuleAllowlist` |

---

## Catalog (done this session)

Primary source is W3C webref, not MDN. MDN is deprecating `mdn/data`. Human indexes:

- https://www.w3.org/Style/CSS/all-properties.en.html
- https://www.w3.org/TR/CSS/ (Snapshot: what counts as CSS)
- https://github.com/w3c/webref
- https://developer.mozilla.org/en-US/docs/Web/CSS/Reference

Pinned files live in `catalog/`. How they were chosen: `catalog/SOURCE.md`.

`mapping.json` fields: `property`, `kind`, `syntax`, `spec`, `spec_href`, `spec_status`, `print_relevant`, `goal` (`implement` or `ignore`), `engine_status` (`implemented` / `partial` / `unsupported` / `ignored`), `code_path`, `mdn_url`.

First-pass `goal: implement` is 608 names. That still includes print-noop UI (`cursor`, `resize`) and SVG fill. Phase 48 reclassifies those before implementation work treats them as must-do.

---

## Phase map

```text
48 Catalog freeze and honesty
  -> 49 Selectors, cascade, at-rules
    -> 50 Values, units, logical properties
      -> 51 Template box, table, text
        -> 52 Paint: backgrounds, radius, outline, overflow clip
          -> 53 Generated content, lists, counters
            -> 54 Paged media and fragmentation
              -> 55 Layout Partial deepen (fixture-driven)
                -> 56 Docs, mapping sync, closure
                  -> 57 Catalog catch-up + claim lock (Partial -> Implemented program)
                    -> 58 Paint finishes
                      -> 59 Logical box
                        -> 60 Text, lists, generated content
                          -> 61 Overflow, visibility, table
                            -> 62 Breaks, orphans/widows, page
                              -> 63 writing-mode vertical
                                -> 64 Flex near-print
                                  -> 65 Grid near-print
                                    -> 66 Position / transform / stacking
                                      -> 67 Mapping + matrix + golden closure
                                        -> 68 Ignored inventory + policy amend
                                          -> 69 Vendor-prefix aliases
                                            -> 70 SVG presentation
                                              -> 71 Mask, clip, filter
                                                -> 72 Scroll UI
                                                  -> 73 Animation / transition
                                                    -> 74 3D transforms
                                                      -> 75 Anchor / view timelines
                                                        -> 76 Pointer / form UI
                                                          -> 77 Speech / aural
                                                            -> 78 Ignored program closure
                                                              -> 79 Partial remaining to Implemented
                                                                -> 80 Implement for print (232)
                                                                  -> 81 Niche / draft (94)
                                                                    -> 82 Vendor aliases when base done (48)
                                                                      -> 83 Hard defer (87)
                                                                        -> 84 Skip print noop (155)
```

| Phase | File | Goal |
|------:|------|------|
| 48 | [phases/phase-48-catalog-and-honesty.md](phases/phase-48-catalog-and-honesty.md) | Freeze catalogs, reclassify print-noop, fix stale matrix rows, coverage script |
| 49 | [phases/phase-49-selectors-cascade-atrules.md](phases/phase-49-selectors-cascade-atrules.md) | `:is()` / `:where()`, optional of-type, `@import` via FetchSub ACL |
| 50 | [phases/phase-50-values-units-logical.md](phases/phase-50-values-units-logical.md) | `clamp()`, `hsl()`, logical box longhands, unit honesty |
| 51 | [phases/phase-51-template-box-table-text.md](phases/phase-51-template-box-table-text.md) | `word-spacing`, `visibility`, `caption-side`, `pre-wrap`, `table-layout` honesty |
| 52 | [phases/phase-52-paint-backgrounds-outline.md](phases/phase-52-paint-backgrounds-outline.md) | `background-image`, outline, radius longhands, overflow clip lite |
| 53 | [phases/phase-53-generated-content-lists.md](phases/phase-53-generated-content-lists.md) | counters, quotes, `list-style-position` |
| 54 | [phases/phase-54-paged-media-fragmentation.md](phases/phase-54-paged-media-fragmentation.md) | named `@page`, page selectors, break-value honesty |
| 55 | [phases/phase-55-layout-partial-deepen.md](phases/phase-55-layout-partial-deepen.md) | Only flex/grid/multicol/float holes that fail a named fixture |
| 56 | [phases/phase-56-docs-closure.md](phases/phase-56-docs-closure.md) | matrix, fidelity, mapping, claim-scan, lint, test, golden |
| 57 | [phases/phase-57-partial-to-implemented-catchup.md](phases/phase-57-partial-to-implemented-catchup.md) | Promote Partial rows that already match matrix/code; lock exit criteria |
| 58 | [phases/phase-58-paint-finishes.md](phases/phase-58-paint-finishes.md) | outline, radius %, box-shadow, background layers |
| 59 | [phases/phase-59-logical-box-implemented.md](phases/phase-59-logical-box-implemented.md) | logical margin/padding/inset/size for horizontal-tb |
| 60 | [phases/phase-60-text-lists-generated.md](phases/phase-60-text-lists-generated.md) | white-space, font stack, lists, quotes, counters |
| 61 | [phases/phase-61-overflow-visibility-table.md](phases/phase-61-overflow-visibility-table.md) | overflow axes, visibility collapse, table deepen |
| 62 | [phases/phase-62-breaks-page.md](phases/phase-62-breaks-page.md) | break values, orphans/widows, page |
| 63 | [phases/phase-63-writing-mode-vertical.md](phases/phase-63-writing-mode-vertical.md) | true vertical-rl / vertical-lr layout |
| 64 | [phases/phase-64-flex-near-print.md](phases/phase-64-flex-near-print.md) | flex / place / align-content near-print |
| 65 | [phases/phase-65-grid-near-print.md](phases/phase-65-grid-near-print.md) | grid track/placement near-print |
| 66 | [phases/phase-66-position-transform.md](phases/phase-66-position-transform.md) | position, transform, display, container-type |
| 67 | [phases/phase-67-partial-program-closure.md](phases/phase-67-partial-program-closure.md) | mapping, matrix, counts, gates |
| 68 | [phases/phase-68-ignored-inventory-policy.md](phases/phase-68-ignored-inventory-policy.md) | Lock 247 Ignored names; amend policy; move onto work list |
| 69 | [phases/phase-69-vendor-prefix-aliases.md](phases/phase-69-vendor-prefix-aliases.md) | Vendor-prefix aliases (70) |
| 70 | [phases/phase-70-svg-presentation.md](phases/phase-70-svg-presentation.md) | SVG fill/stroke presentation (31) |
| 71 | [phases/phase-71-mask-clip-filter.md](phases/phase-71-mask-clip-filter.md) | Mask, clip, filter (26) |
| 72 | [phases/phase-72-scroll-ui.md](phases/phase-72-scroll-ui.md) | Scroll / overscroll (41) |
| 73 | [phases/phase-73-animation-transition.md](phases/phase-73-animation-transition.md) | Animation / transition (28) |
| 74 | [phases/phase-74-3d-transforms.md](phases/phase-74-3d-transforms.md) | 3D transform props (4) |
| 75 | [phases/phase-75-anchor-view-timeline.md](phases/phase-75-anchor-view-timeline.md) | Anchor / offset / view timelines (21) |
| 76 | [phases/phase-76-pointer-form-ui.md](phases/phase-76-pointer-form-ui.md) | Pointer / form UI (7) |
| 77 | [phases/phase-77-speech-aural.md](phases/phase-77-speech-aural.md) | Speech / aural (19) |
| 78 | [phases/phase-78-ignored-program-closure.md](phases/phase-78-ignored-program-closure.md) | Recount, matrix, gates |
| 79 | [phases/phase-79-partial-remaining-to-implemented.md](phases/phase-79-partial-remaining-to-implemented.md) | Remaining Partial to Implemented (closed at 202/0/616/0) |
| 80 | [phases/phase-80-implement-for-print.md](phases/phase-80-implement-for-print.md) | Tier 5 implement-for-print (**232**) |
| 81 | [phases/phase-81-niche-or-draft.md](phases/phase-81-niche-or-draft.md) | Tier 4 niche/draft honesty (**94**) |
| 82 | [phases/phase-82-vendor-aliases-when-base-done.md](phases/phase-82-vendor-aliases-when-base-done.md) | Tier 3 vendor aliases when base done (**48**) |
| 83 | [phases/phase-83-hard-defer.md](phases/phase-83-hard-defer.md) | Tier 2 hard defer SVG/mask/regions (**87**) |
| 84 | [phases/phase-84-skip-print-noop.md](phases/phase-84-skip-print-noop.md) | Tier 1 skip print noop (**155**) |

Name inventory for 68-78: [ignored-inventory.json](ignored-inventory.json). Unsupported triage for 80-84: [unsupported-triage.md](unsupported-triage.md) / [unsupported-triage.json](unsupported-triage.json) (232+94+48+87+155=616).

Older CSS ledgers: `plans/0.2.0/phases/pending-phase-items/`, `tier-2-pending-3/`, `phase-17-broader-css.md`. Those rows move here with `[~]` pointers. Do not copy the checklists.

---

## Phase 48: Catalog freeze and honesty

See [phases/phase-48-catalog-and-honesty.md](phases/phase-48-catalog-and-honesty.md).

### 48.1 Vendor catalogs
- [x] 48.1.1 Store webref `css.json` at `catalog/webref-css.json`. Proof: file present, sha256 in `catalog/SOURCE.md`.
- [x] 48.1.2 Store W3C all-properties JSON at `catalog/w3c-all-properties.json`. Proof: `SOURCE.md`.
- [x] 48.1.3 Store mdn units + properties overlays. Proof: `catalog/mdn-units.json`, `catalog/mdn-properties.json`.
- [x] 48.1.4 Generate `catalog/mapping.json` from webref plus apply-handler inventory. Proof: `coverage-summary.json` counts 818 properties, 75/45/488/210.

### 48.2 Reclassify and scripts
- [x] 48.2.1 Print-noop UI `goal: ignore`. Proof: `python3 scripts/css-catalog-map.py --check`.
- [x] 48.2.2 `scripts/css-catalog-map.py --check` exit 0 (135 apply arms).
- [x] 48.2.3 Matrix stale rows fixed. Proof: `documentation/compatibility-matrix.md`; `make claim-scan` clean.
- [x] 48.2.4 Point `plans/0.2.0/phases/pending-phase-items/README.md` at this ledger for remaining CSS. Proof: banner paragraph exists 2026-08-27.

---

## Phase 49: Selectors, cascade, at-rules

Highest frequency gap: utility sheets wrap rules in `:is()` / `:where()`. Those currently never match, so the whole rule vanishes.

- [x] 49.1 `:is()` matches, specificity is the most specific argument. Proof: `go test ./internal/css -run TestIsPseudo`.
- [x] 49.2 `:where()` matches, specificity 0. Proof: `go test ./internal/css -run TestWherePseudo`.
- [x] 49.3 `@import` fetches under the same ACL as `<link>`. Proof: `TestImportStylesheet`.
- [x] 49.4 `:first-of-type` / `:nth-of-type` / `:nth-last-of-type` and attr `i` flag. Proof: `TestFirstOfType`, `TestNthOfType`, `TestAttrIFlag`.
- [x] 49.5 `:hover`/`:focus`/`:active` still never-match. `go test ./internal/css` green.

Out: forgiving selector lists, shadow DOM, `@supports` evaluation of the full CSS grammar, `@layer` cascade if a later amendment wants it. `@supports` may land as a tiny parse that treats unknown features as false so nested rules are not dropped blindly. That decision is a 49 row, not a promise.

---

## Phase 50: Values, units, logical properties

- [x] 50.1 `clamp()` computes; removed from `supportedDeclaration`. Proof: `TestClampLength`. Fixture-56 is 21 pages.
- [x] 50.2 `hsl()` / `hsla()` in `ParseColor`. Proof: `TestParseColorHsl`.
- [x] 50.3 Logical box longhands for horizontal-tb. Proof: `TestLogical*`. Mapping `--write`.
- [x] 50.4 `ex` Partial 0.5em; `ch` default-face U+0030 advance (`style_ch.go`). Proof: `TestChUsesZeroGlyphAdvance`.
- [x] 50.5 Matrix still documents `vw`/`vh` as width/height/min/max only.

Out: `oklch()`, `color-mix()`, `light-dark()` stay cascade-dropped unless a later row takes them. `cq*` units stay out until `@container` used size is wired into length resolve.

---

## Phase 51: Template box, table, text

Invoice and report CSS already writes these. They currently no-op.

- [x] 51.1 `word-spacing`. Proof: `TestWordSpacingInherits`, `TestWordSpacingWidensRuns`.
- [x] 51.2 `visibility: hidden`. Proof: `TestVisibilityHidden`.
- [x] 51.3 `caption-side: top`/`bottom`/`left`/`right`. Proof: `TestCaptionSideBottom`, `TestCaptionSideLeft`, `TestCaptionSideRight`.
- [x] 51.4 `pre-wrap` / `pre-line`. Proof: `TestWhiteSpacePreWrap`.
- [x] 51.5 Matrix table-layout + `TestTableLayoutFixed`.

---

## Phase 52: Paint, backgrounds, outline, overflow

- [x] 52.1 `background-image: url(...)` first layer, no-repeat at box origin. Proof: `TestBackgroundImageLayoutPaints`. No new golden this session.
- [x] 52.2 Outline stroke outside the border edge. Proof: `TestOutlineStroke`.
- [x] 52.3 Radius longhands and `rx / ry` slash. Proof: `TestRadiusLonghand`, `TestRadiusSlash`.
- [x] 52.4 Overflow clip for hidden/clip/auto/scroll. Proof: `TestOverflowClip`; `TestStickyOverflow*` green.
- [x] 52.5 Lite un-inset `box-shadow` offset fill plus stacked-rect blur. Proof: `TestBoxShadowParse`, `TestBoxShadowPaints`, `TestBoxShadowBlurPaints`.

Gradients are a second slice, not required to close 52. Filter blur stays a non-goal.

---

## Phase 53: Generated content, lists, counters

- [x] 53.1 Counters on `::before`. Proof: `TestCounterInBefore`, `TestCounterResetIncrementLayout`.
- [x] 53.2 Quotes + open/close-quote. Proof: `TestQuotes`.
- [x] 53.3 `list-style-position: inside`. Proof: `TestListStylePositionInside`.
- [x] 53.4 `list-style-image` paints via resolveImage, falls back to type marker. Proof: `TestListStyleImage`.

---

## Phase 54: Paged media and fragmentation

- [x] 54.1 Named `@page` parse; `:first` margins on page 1; `:left`/`:right` even/odd; `page: ident` sibling break plus named margin. Size unnamed-only. Proof: `TestParsePageSelectors`, `TestPageFirstMargins`, `TestPageLeftRightMargins`, `TestPageNamedMargins`.
- [x] 54.2 Break values documented in matrix §2.6: `left`/`right`/`page`/`column` -> `always`; `avoid-page` -> `avoid`; `avoid-column` ignored. Writer has no even/odd or left/right page side; alias kept, not faked. `recto`/`verso` are ignored (not in the apply switch). Proof: `documentation/compatibility-matrix.md` §2.6.
- [x] 54.3 `@page` margin boxes lite: unnamed quoted `@top-*`/`@bottom-*` fill empty CLI header/footer slots. Proof: `TestPageMarginBoxes`. Corners / `running()` still out.

GCPM `running()` / named strings stay out. That is browser header territory.

---

## Phase 55: Layout Partial deepen (fixture-driven)

Do not "finish flexbox." Open a row only when a *named* fixture or golden fails after 49-54.

Candidates already known:

- `align-content: stretch` packing at start
- `flex-flow` / `place-content` / `place-items` shorthands
- `grid` / `grid-template` / `grid-auto-columns` shorthands
- `column-rule`
- `display: inline-grid` not inline-level (`layout_flow.go:85-87`)
- float infobox wrap from `02-openweb-css-residuals.md`

- [x] 55.1 `flex-flow`, `place-*`, `grid`/`grid-template` shorthands. Proof: `TestFlexFlowShorthand`, `TestPlaceShorthands`, `TestGridTemplateShorthand`. `vmin`/`vmax` Partial via `vminVmaxPt`. Proof: `TestVminVmax`.
- [x] 55.2 `align-content: stretch`, `column-rule*`, `display: inline-grid` inline-level. Proof: `TestAlignContentStretch`, `TestColumnRulePaints`, `TestInlineGridIsInlineLevel`. Float wrap still `[~]` (no unit-test hole; live Ana skipped).
- [~] 55.3 `paint_flow.go` (2367) and `paint_pagination.go` (2244) already over the 2000-line soft cap. This slice did not grow them.
- [x] 55.4 Mapping Partial for new shorthands. Proof: `python3 scripts/css-catalog-map.py --check`.

---

## Phase 56: Docs, mapping sync, closure

- [x] 56.1 mapping `--check` green (147 apply arms).
- [x] 56.2 matrix + fidelity. `make claim-scan` clean.
- [x] 56.3 `make lint`, `make test`, `make golden` exit 0 2026-08-27. VERSION 0.2.5.
- [x] 56.4 plans/README.md + KB updated.

---

## Phase 57: Catalog catch-up + claim lock

See [phases/phase-57-partial-to-implemented-catchup.md](phases/phase-57-partial-to-implemented-catchup.md).

## Phase 58: Paint finishes

See [phases/phase-58-paint-finishes.md](phases/phase-58-paint-finishes.md).

## Phase 59: Logical box Implemented

See [phases/phase-59-logical-box-implemented.md](phases/phase-59-logical-box-implemented.md).

## Phase 60: Text, lists, generated content

See [phases/phase-60-text-lists-generated.md](phases/phase-60-text-lists-generated.md).

## Phase 61: Overflow, visibility, table

See [phases/phase-61-overflow-visibility-table.md](phases/phase-61-overflow-visibility-table.md).

## Phase 62: Breaks and page

See [phases/phase-62-breaks-page.md](phases/phase-62-breaks-page.md).

## Phase 63: writing-mode vertical

See [phases/phase-63-writing-mode-vertical.md](phases/phase-63-writing-mode-vertical.md).

## Phase 64: Flex near-print

See [phases/phase-64-flex-near-print.md](phases/phase-64-flex-near-print.md).

## Phase 65: Grid near-print

See [phases/phase-65-grid-near-print.md](phases/phase-65-grid-near-print.md).

## Phase 66: Position / transform / stacking

See [phases/phase-66-position-transform.md](phases/phase-66-position-transform.md).

## Phase 67: Partial program closure

See [phases/phase-67-partial-program-closure.md](phases/phase-67-partial-program-closure.md).

## Phase 68: Ignored inventory + policy amend

See [phases/phase-68-ignored-inventory-policy.md](phases/phase-68-ignored-inventory-policy.md).

## Phase 69: Vendor-prefix aliases

See [phases/phase-69-vendor-prefix-aliases.md](phases/phase-69-vendor-prefix-aliases.md).

## Phase 70: SVG presentation

See [phases/phase-70-svg-presentation.md](phases/phase-70-svg-presentation.md).

## Phase 71: Mask, clip, filter

See [phases/phase-71-mask-clip-filter.md](phases/phase-71-mask-clip-filter.md).

## Phase 72: Scroll UI

See [phases/phase-72-scroll-ui.md](phases/phase-72-scroll-ui.md).

## Phase 73: Animation / transition

See [phases/phase-73-animation-transition.md](phases/phase-73-animation-transition.md).

## Phase 74: 3D transforms

See [phases/phase-74-3d-transforms.md](phases/phase-74-3d-transforms.md).

## Phase 75: Anchor / view timelines

See [phases/phase-75-anchor-view-timeline.md](phases/phase-75-anchor-view-timeline.md).

## Phase 76: Pointer / form UI

See [phases/phase-76-pointer-form-ui.md](phases/phase-76-pointer-form-ui.md).

## Phase 77: Speech / aural

See [phases/phase-77-speech-aural.md](phases/phase-77-speech-aural.md).

## Phase 78: Ignored program closure

See [phases/phase-78-ignored-program-closure.md](phases/phase-78-ignored-program-closure.md).

## Phase 79: Partial remaining to Implemented

See [phases/phase-79-partial-remaining-to-implemented.md](phases/phase-79-partial-remaining-to-implemented.md).

## Phase 80: Implement for print (232)

See [phases/phase-80-implement-for-print.md](phases/phase-80-implement-for-print.md).

## Phase 81: Niche or draft (94)

See [phases/phase-81-niche-or-draft.md](phases/phase-81-niche-or-draft.md).

## Phase 82: Vendor aliases when base done (48)

See [phases/phase-82-vendor-aliases-when-base-done.md](phases/phase-82-vendor-aliases-when-base-done.md).

## Phase 83: Hard defer (87)

See [phases/phase-83-hard-defer.md](phases/phase-83-hard-defer.md).

## Phase 84: Skip print noop (155)

See [phases/phase-84-skip-print-noop.md](phases/phase-84-skip-print-noop.md).

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| v0.2.0 CSS subset (flex/grid/multicol/sticky/transforms/:has/@container) | This ledger does not rebuild those engines |
| `internal/css` parse-keep of unknown names | New properties start in layout, not the parser |
| Image fetch + paint ops | `background-image` reuses them |
| Catalog freeze (48.1, done) | Every later phase flips mapping rows instead of guessing names |
| This ledger | Higher Implemented frequency on authored templates |

---

## Amendment (2026-08-28): Partial to Implemented program

Phases **57-67** promote the remaining **Partial** property rows to **Implemented** at a near-browser **print media** bar. Closed at **174 Implemented / 0 Partial / 397 Unsupported / 247 Ignored**. See [property-counts.md](property-counts.md).

## Amendment (2026-08-28): Ignored to browser-level print (phases 68-78)

Phases **68-78** reopen **all 247 former Ignored** properties for **browser-level print**. Phase 68 inventory/policy stands. Phases **69-77 were falsely marked complete** (catalog Implemented without engine code); a 2026-08-28 four-agent audit reverted those rows to **unsupported** (`filter` opacity-only **partial**). Checklists for 69-77 are **not started** again. Do not flip `engine_status` to Implemented without apply arms + tests + matrix agreement.

Inventory: [ignored-inventory.json](ignored-inventory.json). Counts: [property-counts.md](property-counts.md).

Browser-print golden harness: `testdata/golden/fixture-57-vanguard-telemetry-audit.html` covers the pre-68 **571** print-relevant set (not the reopened Ignored names yet).

## Amendment (2026-08-29): Unsupported triage phases 80-84

After Phase 79 closed at **202 / 0 / 616 / 0**, the remaining **616 Unsupported** names were triaged in [unsupported-triage.md](unsupported-triage.md):

| Phase | Tier | Count | Mode |
|------:|------|------:|------|
| 80 | implement for print | 232 | Code + tests + mapping (waves); hard rows may stay Unsupported honestly |
| 81 | niche / draft | 94 | Catalog notes; no Implemented flips |
| 82 | alias when base done | 48 | Prefixed aliases only after unprefixed base is Implemented |
| 83 | hard defer | 87 | Default honesty/docs; optional tiny clip/mask only with flip packets |
| 84 | skip print noop | 155 | Catalog notes; print non-goals |

Standing rules in every 80-84 checklist: **no git commands unless the user explicitly asks**; after status changes update both `catalog/mapping.json` and `catalog/coverage-summary.json` (plus `property-counts.md`); mapping last per `HONESTY-GATES.md`.

## Out of scope (unless this ledger is amended)

- JavaScript, SPA hydration, `--enable-javascript`
- Chrome print parity, Wikipedia Vector/Minerva clone, pixel goldens
- Animation, transition, 3D, filter blur, scroll-driven animation, view transitions
- Full Grid L3 / masonry-as-Chrome beyond what Phase 65 claims for print
- `@supports` as a real feature-query engine (tiny false-unknown parse is a Phase 49 maybe)
- WOFF2 / metric font aliases (sidecar; code still rejects WOFF2 here)
- New direct Go modules
- Growing any Go file past ~2000 lines. Split by responsibility.

---

## Evidence rules

- Prefer current code, tests, and `catalog/mapping.json` over historical notes.
- Negative results are precise: "no apply arm for `word-spacing` in `style_properties.go`", never "no bugs".
- Close a row only after the matching `make test` / `make lint` / `make golden` / `make claim-scan` exits 0.
- First mapping statuses are a snapshot. Code wins if they drift.

---

## Body record and branch

- Body: `plans/PR/pr-0.2.6-css-coverage.md` when a PR opens
- Suggested branch: `feature/0.2.6-css-coverage`

---

## Completion handoff

Confirm rows, run the smallest validation, `[x]` only after evidence, name the next unchecked phase. Phase 48 remaining work is 48.2. Implementation starts at 49.

## Required checks

- Docs-only: skip lint/test.
- Otherwise: `make lint` and `make test` before marking the phase complete. Leave unchecked if either fails.
