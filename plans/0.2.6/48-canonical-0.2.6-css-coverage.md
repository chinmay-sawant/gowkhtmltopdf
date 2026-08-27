# 48 - v0.2.6 CSS coverage (Canonical Execution Ledger)

> **Parent:** `plans/0.2.5/40-canonical-0.2.5-python-bindings.md` (complete 2026-08-26). Leftover CSS rows under `plans/0.2.0/` move here with `[~]` pointers.
> **Status:** not started (catalog freeze done 2026-08-27; implementation phases 49-56 open)
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
- [ ] 48.2.1 Mark print-noop UI (`cursor`, `caret-color`, `resize`, `user-select`, `pointer-events`, `touch-action`, `appearance`) `goal: ignore` in `mapping.json`. Proof: grep those names in mapping, `goal` is `ignore`.
- [ ] 48.2.2 Commit a regenerator under `scripts/` that rebuilds mapping from code + frozen catalogs. Proof: `scripts/css-catalog-map.py --check` exits 0 against current mapping.
- [ ] 48.2.3 Fix matrix stale rows: `table-layout` consumed lite (`layout_tables.go:45`); `ex`/`ch` are 0.5em (`container.go:133`); `applyRestProps` lives in `style_cascade.go:666` not `style.go:340`. Proof: those sentences in `documentation/compatibility-matrix.md`.
- [x] 48.2.4 Point `plans/0.2.0/phases/pending-phase-items/README.md` at this ledger for remaining CSS. Proof: banner paragraph exists 2026-08-27.

---

## Phase 49: Selectors, cascade, at-rules

Highest frequency gap: utility sheets wrap rules in `:is()` / `:where()`. Those currently never match, so the whole rule vanishes.

- [ ] 49.1 `:is()` matches, specificity is the most specific argument. Proof: `go test ./internal/css -run TestIsPseudo`.
- [ ] 49.2 `:where()` matches, specificity 0. Proof: `go test ./internal/css -run TestWherePseudo`.
- [ ] 49.3 `@import` fetches via `FetchSub` under the same ACL as `<link>`. Cycles and failed fetches skip. Proof: convert test with local imported sheet.
- [ ] 49.4 Optional, fixture-gated: `:first-of-type` / `:nth-of-type` only if a golden or wiki-like fixture needs them.
- [ ] 49.5 Keep `:hover`/`:focus`/`:active` as never-match. Print has no pointer. Do not "fix" them into matching.

Out: forgiving selector lists, shadow DOM, `@supports` evaluation of the full CSS grammar, `@layer` cascade if a later amendment wants it. `@supports` may land as a tiny parse that treats unknown features as false so nested rules are not dropped blindly. That decision is a 49 row, not a promise.

---

## Phase 50: Values, units, logical properties

- [ ] 50.1 `clamp(min, pref, max)` on lengths. Remove `clamp(` from `supportedDeclaration` once it computes. Proof: `go test ./internal/layout -run TestClampLength`.
- [ ] 50.2 `hsl()` / `hsla()` in `ParseColor`. Proof: `go test ./internal/css -run TestParseColorHsl`.
- [ ] 50.3 Logical box longhands used in reports: `margin-inline`, `margin-block`, `padding-inline`, `padding-block`, `inset`, `inline-size`, `block-size` mapping onto physical fields for `horizontal-tb`. Proof: layout tests plus mapping status flip.
- [ ] 50.4 Document `ex`/`ch` as 0.5em Partial, not "dropped". Optional: `ch` from actual glyph width of `0` if cheap.
- [ ] 50.5 `vw`/`vh` on margin/padding, or document that they stay width/height-only.

Out: `oklch()`, `color-mix()`, `light-dark()` stay cascade-dropped unless a later row takes them. `cq*` units stay out until `@container` used size is wired into length resolve.

---

## Phase 51: Template box, table, text

Invoice and report CSS already writes these. They currently no-op.

- [ ] 51.1 `word-spacing`. Proof: inherit + run-width test like `letter-spacing`.
- [ ] 51.2 `visibility: hidden` hides paint, still occupies space. `collapse` may alias hide for non-tables. Proof: layout test.
- [ ] 51.3 `caption-side: bottom` paints caption below the table. Proof: table test + matrix row.
- [ ] 51.4 `white-space: pre-wrap` and `pre-line` no longer collapse to `pre`. Proof: `style_properties.go:1086-1092` behavior change + test.
- [ ] 51.5 Matrix `table-layout` row matches `layout_tables.go:45`. Add a golden if fixed layout is still under-tested.

---

## Phase 52: Paint, backgrounds, outline, overflow

- [ ] 52.1 `background-image: url(...)` using the existing image fetch + paint path. No-repeat, default positioning, first layer only is an acceptable Partial. Proof: golden with a PNG background.
- [ ] 52.2 `outline` / `outline-width` / `outline-style` / `outline-color` as a paint stroke outside the border edge. Proof: layout chrome test.
- [ ] 52.3 `border-top-left-radius` and the other three longhands. Proof: `rounded_border_test.go` extended.
- [ ] 52.4 `overflow: hidden|clip` paint clip lite for the box. Sticky already uses overflow as scrollport. Proof: child ink does not draw outside the clip in a unit test.
- [ ] 52.5 `box-shadow` lite: one un-inset shadow, no blur if blur is expensive, or blur as a soft fill if the paint path already can. If this blows the file-size limit, extract. If it is not cheap, `[~]` with reason.

Gradients are a second slice, not required to close 52. Filter blur stays a non-goal.

---

## Phase 53: Generated content, lists, counters

- [ ] 53.1 `counter-reset` / `counter-increment` plus `content: counter(name)` / `counters()`. Proof: nested `ol` fixture.
- [ ] 53.2 `quotes` + `content: open-quote` / `close-quote` subset. Proof: unit test.
- [ ] 53.3 `list-style-position: inside`. Proof: marker in the first line box, not hanging outside.
- [ ] 53.4 `list-style-image` may stay `[~]` if images-as-markers need extra paint ops.

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
