# 48 - v0.2.6 CSS coverage (Canonical Execution Ledger)

> **Parent:** `plans/0.2.5/40-canonical-0.2.5-python-bindings.md` (complete 2026-08-26). Leftover CSS rows under `plans/0.2.0/` move here with `[~]` pointers.
> **Status:** in progress (catalog + first implementation slice on feature/026-extended-css-support)
> **Estimated effort:** several weeks across phases 48-56. Catalog and honesty first. Frequency slices next. Layout deepen last.
> **Constraint:** pure Go, no CGO on the default path, no browser embed, no JavaScript. Direct modules stay `go-text/typesetting` and `tdewolff/canvas` unless an amendment is filed. `catalog/mapping.json` is the CSS name inventory.
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
- [~] 49.4 Optional of-type: no fixture this session.
- [x] 49.5 `:hover`/`:focus`/`:active` still never-match. `go test ./internal/css` green.

Out: forgiving selector lists, shadow DOM, `@supports` evaluation of the full CSS grammar, `@layer` cascade if a later amendment wants it. `@supports` may land as a tiny parse that treats unknown features as false so nested rules are not dropped blindly. That decision is a 49 row, not a promise.

---

## Phase 50: Values, units, logical properties

- [x] 50.1 `clamp()` computes; removed from `supportedDeclaration`. Proof: `TestClampLength`. Fixture-56 is 21 pages.
- [x] 50.2 `hsl()` / `hsla()` in `ParseColor`. Proof: `TestParseColorHsl`.
- [x] 50.3 Logical box longhands for horizontal-tb. Proof: `TestLogical*`. Mapping `--write`.
- [x] 50.4 `ex`/`ch` Partial 0.5em in the matrix.
- [x] 50.5 Matrix still documents `vw`/`vh` as width/height/min/max only.

Out: `oklch()`, `color-mix()`, `light-dark()` stay cascade-dropped unless a later row takes them. `cq*` units stay out until `@container` used size is wired into length resolve.

---

## Phase 51: Template box, table, text

Invoice and report CSS already writes these. They currently no-op.

- [x] 51.1 `word-spacing`. Proof: `TestWordSpacingInherits`, `TestWordSpacingWidensRuns`.
- [x] 51.2 `visibility: hidden`. Proof: `TestVisibilityHidden`.
- [x] 51.3 `caption-side: bottom`. Proof: `TestCaptionSideBottom`.
- [x] 51.4 `pre-wrap` / `pre-line`. Proof: `TestWhiteSpacePreWrap`.
- [x] 51.5 Matrix table-layout + `TestTableLayoutFixed`.

---

## Phase 52: Paint, backgrounds, outline, overflow

- [x] 52.1 `background-image: url(...)` first layer, no-repeat at box origin. Proof: `TestBackgroundImageLayoutPaints`. No new golden this session.
- [x] 52.2 Outline stroke outside the border edge. Proof: `TestOutlineStroke`.
- [x] 52.3 Radius longhands. Proof: `TestRadiusLonghand`.
- [x] 52.4 Overflow clip for hidden/clip/auto/scroll. Proof: `TestOverflowClip`; `TestStickyOverflow*` green.
- [~] 52.5 `box-shadow` not this session. Permanent skip unless a later amendment.

Gradients are a second slice, not required to close 52. Filter blur stays a non-goal.

---

## Phase 53: Generated content, lists, counters

- [x] 53.1 Counters on `::before`. Proof: `TestCounterInBefore`, `TestCounterResetIncrementLayout`.
- [x] 53.2 Quotes + open/close-quote. Proof: `TestQuotes`.
- [x] 53.3 `list-style-position: inside`. Proof: `TestListStylePositionInside`.
- [~] 53.4 `list-style-image` not this session.

---

## Phase 54: Paged media and fragmentation

- [ ] 54.1 Named `@page` and `:first` / `:left` / `:right` selectors applying margin/size. Proof: convert test with two page sizes.
- [ ] 54.2 Break values documented: `left`/`right`/`recto`/`verso` currently collapse to `always`. Either implement even/odd page or write that collapse in the matrix as Partial with the alias table.
- [ ] 54.3 `@page` margin boxes (`@top-center` and friends) stay `[~]` unless a report fixture proves they beat CLI headers. CLI `--header-*` remains the supported repeating chrome.

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

Each row names the fixture and the proof command. Chrome PDFs are not the gate.

---

## Phase 56: Docs, mapping sync, closure

- [ ] 56.1 `catalog/mapping.json` regenerated; `scripts/css-catalog-map.py --check` green.
- [ ] 56.2 `documentation/compatibility-matrix.md` and `documentation/fidelity.md` match code. `make claim-scan` clean.
- [ ] 56.3 `make lint`, `make test`, `make golden` exit 0. Record tails on the phase file.
- [ ] 56.4 `plans/README.md` version row updated. KB log + roadmap milestone updated.

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

## Out of scope (unless this ledger is amended)

- JavaScript, SPA hydration, `--enable-javascript`
- Chrome print parity, Wikipedia Vector/Minerva clone, pixel goldens
- Animation, transition, 3D, filter blur, scroll-driven animation, view transitions
- Full Grid L3, joint subgrid intrinsic, masonry-as-Chrome
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
