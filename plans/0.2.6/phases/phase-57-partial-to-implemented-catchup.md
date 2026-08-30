# Phase 57: Catalog catch-up + claim lock

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 57
> **Status:** catch-up promotions done; exit criteria locked; remaining Partials owned by 58-66
> **Estimated effort:** S
> **Owner:** catalog + matrix honesty
> **Depends on:** Phase 56
> **Unblocks:** Phase 58

---

## Overview

Start the Partial-to-Implemented program. Promote rows where the compatibility matrix already says Implemented and code/tests match. Lock a one-line exit criterion for every remaining Partial.

Bar: near-browser for print media. No layout behavior change required in this phase beyond catalog honesty.

## Checklist

### 57.1 catch-up promotions

- [x] 57.1.1 Promote matrix-aligned Partials to Implemented in `catalog/mapping.json`: `border`, `border-bottom`, `border-left`, `border-right`, `border-style`, `border-top`, `column-rule`, `column-rule-color`, `column-rule-style`, `column-rule-width`, `font-style`, `max-height`, `text-decoration`, `word-spacing`. Proof: recount below; matrix already Implemented for these names.
- [x] 57.1.2 Regenerate `catalog/coverage-summary.json` and update `property-counts.md`.
- [x] 57.1.3 `python3 scripts/css-catalog-map.py --check` exit 0.

### 57.2 exit criteria lock

- [x] 57.2.1 One-line exit criterion for each remaining Partial (table below). Owned by later phases; do not flip without code+tests.

### 57.3 ledger

- [x] 57.3.1 Canonical ledger amended for phases 57-67 and near-print deepening amendment.

## Remaining Partials and exit criteria

Count after 57.1: **85** (was 99).

| Property | Exit criterion for Implemented |
|----------|--------------------------------|
| `align-content` | Flex line packing including stretch and claimed values. |
| `background` | Multi-layer url + color shorthand parity with longhands you claim; or lock Implemented to color+first url only and match code. |
| `background-image` | Layers, repeat, size, position as claimed for print; gradients if claimed. |
| `block-size` | Maps to height for horizontal-tb; used width/height paths honor it. |
| `border-bottom-left-radius` | Same percent/elliptical rules as border-radius longhands. |
| `border-bottom-right-radius` | Same percent/elliptical rules as border-radius longhands. |
| `border-collapse` | Collapsed border conflict resolution, not only zero spacing. |
| `border-radius` | Percent corners resolve per CSS (width/height axes), not uniform fallback only. |
| `border-top-left-radius` | Same percent/elliptical rules as border-radius longhands. |
| `border-top-right-radius` | Same percent/elliptical rules as border-radius longhands. |
| `box-shadow` | Multi-layer + spread + inset as claimed, or narrow Implemented claim to shipped subset and match tests. |
| `break-after` | Print-relevant values honored or aliased honestly. |
| `break-before` | Print-relevant values honored or aliased honestly. |
| `break-inside` | avoid/avoid-page (and claimed set) honored in paginator. |
| `caption-side` | top/bottom/left/right caption placement as claimed. |
| `container-type` | Size containment for @container as claimed. |
| `content` | Quoted strings, attr, counter/counters, quotes open/close as claimed. |
| `counter-increment` | Increments named counters at the right tree point. |
| `counter-reset` | Creates/resets named counters used by counter()/counters(). |
| `display` | flex/grid/inline-* modes match deepened engines; core values already done. |
| `flex-flow` | Shorthand equals flex-direction + flex-wrap. |
| `font` | Shorthand sets size/family/style/weight/line-height used by layout. |
| `font-family` | Author stack + generics resolve; missing faces fall through as claimed. |
| `grid` | Shorthand expands to template/auto tracks as claimed. |
| `grid-column` | Shorthand placement equals start/end. |
| `grid-column-end` | Line/area placement as claimed. |
| `grid-column-start` | Line/area placement as claimed. |
| `grid-row` | Shorthand placement equals start/end. |
| `grid-row-end` | Line/area placement as claimed. |
| `grid-row-start` | Line/area placement as claimed. |
| `grid-template` | Shorthand expands to areas/columns/rows as claimed. |
| `grid-template-columns` | Track list sizing near-print as claimed. |
| `grid-template-rows` | Track list sizing near-print as claimed. |
| `inline-size` | Maps to width for horizontal-tb. |
| `inset` | Shorthand sets top/right/bottom/left. |
| `inset-block` | Maps to top/bottom for horizontal-tb. |
| `inset-block-end` | Equals bottom for horizontal-tb. |
| `inset-block-start` | Equals top for horizontal-tb. |
| `inset-inline` | Maps to left/right for horizontal-tb LTR. |
| `inset-inline-end` | Equals right for horizontal-tb LTR. |
| `inset-inline-start` | Equals left for horizontal-tb LTR. |
| `list-style` | Shorthand expands to type/position/image. |
| `list-style-image` | url marker paints when set; none falls back to type. |
| `list-style-position` | inside/outside marker placement as claimed. |
| `margin-block` | Shorthand expands to block-start/end; physical margins match horizontal-tb. |
| `margin-block-end` | Equals margin-bottom for horizontal-tb. |
| `margin-block-start` | Equals margin-top for horizontal-tb. |
| `margin-inline` | Shorthand expands to inline-start/end for horizontal-tb. |
| `margin-inline-end` | Equals margin-right for horizontal-tb LTR. |
| `margin-inline-start` | Equals margin-left for horizontal-tb LTR. |
| `max-block-size` | Maps to max-height for horizontal-tb. |
| `max-inline-size` | Maps to max-width for horizontal-tb. |
| `min-block-size` | Maps to min-height for horizontal-tb. |
| `min-inline-size` | Maps to min-width for horizontal-tb. |
| `orphans` | Fragmentation Rule 3 when line boxes exist; no silent no-op. |
| `outline` | Documented outline style/width/color/offset set paints outside border without affecting layout. |
| `outline-color` | Parity with outline shorthand color. |
| `outline-offset` | Offset shifts outline without layout size change. |
| `outline-style` | Claimed style keywords paint. |
| `outline-width` | Claimed widths paint. |
| `overflow` | visible/hidden/auto/scroll/clip behavior as claimed for print clip + sticky scrollport. |
| `overflow-x` | Independent horizontal overflow when claimed. |
| `overflow-y` | Independent vertical overflow when claimed. |
| `padding-block` | Shorthand expands; physical padding matches horizontal-tb. |
| `padding-block-end` | Equals padding-bottom for horizontal-tb. |
| `padding-block-start` | Equals padding-top for horizontal-tb. |
| `padding-inline` | Shorthand expands for horizontal-tb. |
| `padding-inline-end` | Equals padding-right for horizontal-tb LTR. |
| `padding-inline-start` | Equals padding-left for horizontal-tb LTR. |
| `page` | Named page used-value + breaks + @page ident margins as claimed. |
| `page-break-after` | Alias parity with break-after. |
| `page-break-before` | Alias parity with break-before. |
| `page-break-inside` | Alias parity with break-inside. |
| `place-content` | Shorthand equals align-content + justify-content. |
| `place-items` | Shorthand equals align-items + justify-items. |
| `place-self` | Shorthand equals align-self + justify-self. |
| `position` | relative/absolute/fixed/sticky containing blocks and print sticky policy as claimed. |
| `quotes` | Inherited pairs + nesting depth for open/close-quote. |
| `table-layout` | fixed uses column hints/first-row widths per claim; auto path unchanged. |
| `transform` | Static 2D functions + origin percentages/nesting as claimed; 3D stays ignored. |
| `vertical-align` | Claimed keywords for cells and inline replaced. |
| `visibility` | hidden keeps layout; collapse hides table rows as claimed. |
| `white-space` | All claimed values (incl. pre-line/pre-wrap/break-spaces if claimed) match CSS white-space processing. |
| `widows` | Fragmentation Rule 3 when line boxes exist; no silent no-op. |
| `writing-mode` | vertical-rl/lr use real vertical flow, not glyph-rotate-only. |

## Property counts after 57.1

| Status | Count |
|--------|------:|
| implemented | 89 |
| partial | 85 |
| unsupported | 397 |
| ignored | 247 |

## Out of scope

Engine behavior changes (those start at Phase 58). Chrome pixel goldens. The 397 unsupported names except as dependencies of a Partial exit criterion.

## Handoff

Next is Phase 58 (paint finishes: outline, radius %, box-shadow, background).
