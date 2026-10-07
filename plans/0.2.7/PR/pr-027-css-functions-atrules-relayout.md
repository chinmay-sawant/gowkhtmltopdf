## Summary

Integration wave for the v0.2.7 CSS and layout work: 26 commits adding modern color functions and the full named-color table, `calc()`/`min()`/`max()`/`clamp()` math with dynamic viewport units, `@supports`/`@layer`/`@property`, `data:` URL and WOFF2 font loading, conic gradients and basic `clip-path` shapes, text input painting and control UA faces, host-state pseudo-classes, an image resolver on the public layout entry points, `background` shorthand expansion, a per-tree sheet cache, and `css.Relayout` for re-rendering at a new viewport and state. It also exposes element boxes on `Display`, fixes diagonal line strokes in the rasterizer, refreshes four sample PDFs, and adds the 0.2.7 selector, function, and at-rule ledger (89, planned). 72 files, +5948/-241 against `master`. VERSION stays 0.2.6.

---

## Motivation / context

- Plans: `plans/0.2.7/README.md`; new `plans/0.2.7/89-canonical-0.2.7-selectors-functions-atrules.md` (`0cf692c`, status planned, batches 89.1-89.8: recount, checker, selector/function/at-rule waves, fixture-67 closure).
- The 89 ledger records the plan baseline from `plans/0.2.6/catalog/mapping.json`: selectors 158, functions 162, at-rules 55. Those counts are the reason the ledger starts with a recount; this PR implements a large slice of the names behind them and does not edit catalog counts.
- Issues: none open for this wave, see **Related issues**.

---

## Changes

### Colors (`8932226`, merge `5db58c6`, fix `7f8164f`)

- Full CSS Color 4 named table: 148 names including `grey`/`gray` aliases and `rebeccapurple`, replacing the previous 51-name subset (`internal/css/color_names.go`).
- `ParseColor` now resolves `oklab()`, `oklch()`, `color-mix(in srgb, ...)`, and `light-dark()` with number or percentage channels and optional `/ alpha` (`internal/css/color_modern.go:21`, `internal/css/values.go:233`). Oklab uses the Ottosson LMS matrices; `color-mix` interpolates with premultiplied alpha and normalizes weights.
- Cascade fix: `supportedDeclaration` no longer rejects `color-mix(`, `light-dark(`, and `oklch(` values, so they win over earlier fallbacks instead of being filtered out (`internal/layout/style_cascade.go:1187`).

### Math and units (`f7c1279`, merge `23913f8`)

- New `EvalMath` recursive-descent evaluator for `calc()`, `min()`, `max()`, and `clamp()` with `+ - * /`, parentheses, nesting, and mixed units (`internal/css/math_expr.go:29`).
- Layout wires it through `calcLength`/`clampLength`/`mathLength` (`internal/layout/style_values.go:926`, `:933`), and `dvh`/`svh`/`lvh` resolve like `vh` for box lengths (`:801`).
- A clamp declaration now wins the cascade and resolves instead of losing to the earlier fallback. `TestCascadeClampWinsAndResolves` inverts the old fallback expectation (`internal/layout/style_cascade_test.go:127`).

### At-rules (`ba6c297`, merge `eb79731`)

- `@supports`: full condition tree (`and`/`or`/`not`, nested parens, general-enclosed gated false) evaluated against the engine's real property dispatch, so `(display: grid)` matches only with a working apply arm (`internal/css/at_supports.go:30`, `internal/layout/style_cascade.go:1618`).
- `@layer`: per-sheet ranks stamped on rules; unlayered declarations beat ranked ones, and later-declared layers beat earlier ones (`internal/css/at_layer.go:64`, `internal/layout/style_cascade.go:1249`).
- `@property`: registrations provide initial values and the `inherits` flag to custom-property resolution, last registration wins (`internal/css/at_property.go:19`, `internal/layout/style_cascade.go:113`, `:137`).

### Fonts (`eeaa97f`, merge `15797c1`)

- New `internal/fonts` package reconstructs SFNT from null-transform WOFF2: header and table directory parse, Brotli decompression (already an indirect dependency), recomputed checksums, caps of 1024 tables / 16 MiB per table / 32 MiB SFNT (`internal/fonts/woff2.go:65`, `internal/fonts/sfnt.go:16`).
- `data:` URL `@font-face` sources are no longer skipped; fetch bytes pass through `fonts.Decode` before `pdf.ParseFontBytes` (`internal/convert/prepare/styles.go:512`). TTF/OTF/WOFF1 data URLs and null-transform WOFF2 work; transformed or CFF/OTTO fonts log a warning and register no font.

### Conic gradients and clip-path (`b7f694e`, merge `ff2d64c`)

- `conic-gradient()` and `repeating-conic-gradient()` paint with optional `from <angle>` and `at <position>` preludes, rasterized per pixel with `atan2` (`internal/layout/gradient.go:276`, `:561`).
- `clip-path` accepts `inset()`, `circle()`, `ellipse()`, and `polygon()` (both fill rules) and applies as an alpha mask over raster image ops for backgrounds and replaced/inline images (`internal/layout/clip_path.go:70`, `:482`). Invalid or unsupported values leave the previous value or no clip.
- Same commit fixes a missing `border.Transparent` style-interning hash (`internal/layout/style_intern_gen.go:1089`).

### Controls and host state (`1d4024b`, merge `ff2cca6`, `f9df53f`, merge `54a29b6`)

- Text-like inputs (`text`, `password`, `search`, `email`, `tel`, `url`, `number`, and missing type) paint their value, or the placeholder when empty, vertically centered at the 10pt UA control size and cut to content width (`internal/layout/control_text.go:17`, `:50`). Passwords paint one U+2022 bullet per rune, so plaintext never reaches the op list (`:34`).
- UA faces added for `button` and `select`, completed for `textarea`; author declarations override them (`internal/layout/style_values.go:1769`, `:1776`, `:1861`).
- Host state: new `css.Options` fields `Focus`, `Hover`, `Active` (element ids) drive `:focus`/`:focus-visible`, `:hover`, and `:active` matching, threaded through the whole selector walk including `:has()` (`css/css.go:74`, `internal/css/match.go:86`, `internal/css/has.go:211`). `:checked` matches `input[type=checkbox|radio][checked]` and `option[selected]` with no host state (`internal/css/match.go:429`).

### Public layout and display API (`1e1a884`, `274b4d8`, `2111b36`)

- `layout.Display` gains `Boxes []Box`, built from `boxesFrom(PlacedElements(res))` so it matches `Lay().Boxes()` without a raster pass (`layout/displaylist.go:103`, `layout/layout.go:125`). Asserted by `TestDisplayAgreesWithLayOnBoxes`.
- New public `layout.Options` with `Images func(src string) ([]byte, error)`, plus `LayOptions` and `DisplayListOptions` wrappers; nil resolver keeps the old behavior, a resolver paints `<img src>` and `background-image url(...)` bytes (`layout/layout.go:40`, `layout/displaylist.go:134`).
- `background` is expanded into `background-color` and `background-image` at declaration time, so an author shorthand now competes correctly with longhands from other origins and specificities; the UA button face no longer beats an author `background` (`internal/layout/style_cascade.go:850`, `internal/layout/style_properties.go:1318`).

### Sheet cache and Relayout (`1769749`, `040fd14`)

- `prepare.SheetCache` keys parsed sheets on tree nodes and import indices, so later collections over one tree skip every style parse and link/import fetch while still re-evaluating viewport media gates (`internal/convert/prepare/sheet_cache.go:13`, `:28`). `SheetOptions.Cache` is optional and nil runs the old path (`internal/convert/prepare/styles.go:38`).
- New public `css.Relayout(ctx, doc, width, height, focus, hover, active) (*Document, error)` returns a document sharing the tree, sheet cache, and extra sheets, with the font merge skipped when the sheet list is pointer-identical (`css/relayout.go:21`, `css/same_sheets.go:9`). Collection gating re-runs at the new size; the cascade still runs in layout because `@media` and viewport units depend on it. The tree must not change between `Apply` and `Relayout`.

### Tests, benchmarks, and lint (`8c4c77f`, `9c9a5c6`, `0b58b00`, `0029ea9`)

- 11 relayout tests with zero production changes: sheet pointer reuse for inline/`<link>`/`@import`, re-gating at a `min-width` breakpoint, bad-input errors, stored viewport and state (`css/relayout_*_test.go`); media flips, `vw`/`vh` resize, state pseudo-classes applying and clearing, box-for-box and op-for-op parity with a full `Apply`, no state or sheet leaks across two relayouts, and no memoized styles on the parsed tree (`layout/relayout_*_test.go`).
- `BenchmarkRelayoutStages` isolates collection cost and `BenchmarkRelayoutMedium` measures both paths through `DisplayList`; recorded numbers in `0b58b00` below.
- `0029ea9` is mechanical lint cleanup across the 10 relayout files: renames, wrapped signatures, two removed `//nolint:exhaustruct`, one added `//nolint:unparam`. No behavior change.

### Rasterizer fix, samples, and plans (`1a39183`, `0cf692c`)

- `paintLine` only handled horizontal and vertical segments, so the two diagonal `OpLine` ops behind a checked checkbox tick collapsed into an L-shaped bar pair. Diagonals now stroke their true centerline through `paintStrokeSegment`; axis-aligned border and grid segments keep the rectangle fast path (`internal/imageout/imageout.go:1386`). Test: `internal/imageout/line_diagonal_test.go:13`.
- Four sample PDFs refreshed: `output/fixture-56-architecture-diagram.pdf`, `-57-vanguard-telemetry-audit.pdf`, `-58-unsupported-worklist-audit.pdf`, `-59-apex-digital-landing.pdf`. Fixturetests 56/57/59 update their op counts; `documentation/compatibility-matrix.md:158` documents the background expansion.
- `plans/0.2.7/89-canonical-0.2.7-selectors-functions-atrules.md` added and indexed in `plans/0.2.7/README.md:31`.

Merge hygiene: the seven merge commits (`5db58c6`, `23913f8`, `eb79731`, `15797c1`, `ff2d64c`, `ff2cca6`, `54a29b6`) are dry. Each first-parent diff replays exactly its feature commit; the only differences are blob index lines where both sides edited one file (`style_cascade.go`, `style.go`).

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | `SheetCache` removes repeated parse and fetch work for a second collection over one tree; `css.Relayout` skips the font merge when the sheet list is pointer-identical. Recorded in `0b58b00` (i7-13700HX, `-cpu 1 -benchtime=1000x -count=3`, collection only, not re-run here): apply-only 32.8-38.1us/op, 6368 B/op, 82 allocs/op; relayout-only 8.4-12.5us/op, 1280 B/op, 13 allocs/op. With `DisplayList` both paths run about 3.3-4.1ms/op and cascade/layout dominate. |
| **Memory** | Relayout collection allocates 1280 B/op vs 6368 B/op for a full apply on the benchmark page. The sheet cache retains parsed sheets per tree, with no eviction and no lifetime bound beyond the `Document`. |
| **Behavior / correctness** | Modern colors, conic gradients, clip-path shapes, data-URL fonts, input values, and control faces render where they previously did not. `@supports` gates on the engine's real dispatch, `@layer` orders the cascade, `@property` feeds initial values. `:checked` now matches everywhere, `:focus`/`:hover`/`:active` match when ids are passed. Background shorthand expansion changes origin/specificity outcomes. Element boxes match between `Display` and `Lay`. Checkbox ticks rasterize as a checkmark. |
| **API / CLI** | Additive public API: `layout.Options`, `layout.LayOptions`, `layout.DisplayListOptions`, `layout.Display.Boxes`, `css.Options.Focus/Hover/Active`, `css.Relayout`. CLI flags and `cli.Version` are unchanged. |
| **Dependencies** | None. `go.mod` is untouched; `github.com/andybalholm/brotli v1.2.1` was already an indirect dependency. Allowlist still `go-text/typesetting` and `tdewolff/canvas`. |
| **Binary size / build time** | Not measured. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| Declarations using `oklch()`, `color-mix()`, `light-dark()`, `calc()`/`min()`/`max()`/`clamp()`, conic gradients, or `clip-path` shapes now apply instead of falling through | Re-render documents that already use them; in-repo example is fixture-56. |
| `@supports`/`@layer`/`@property` now affect the cascade | Check stylesheets that relied on those blocks being ignored. |
| `:checked` matches from parsed attributes even with no host state | Expected fix; `checked="false"` still counts as checked, matching HTML boolean attributes. |
| Text inputs paint values and placeholders; `button`/`select`/`textarea` grow UA faces | Forms that assumed empty boxes will show text and default skins; author CSS overrides all of it. |
| `background` shorthand expands at declaration time | Source order now decides between shorthand and longhand; a pure-color shorthand does not reset an earlier `background-image`. |
| `paintLine` diagonals now use the pixel-center rasterizer | Rasterized output changes only for diagonal lines; axis-aligned paths are unchanged. |
| VERSION stays 0.2.6 | Binaries still stamp 0.2.6; cut 0.2.7 later with VERSION, CHANGELOG, and C/Python stamps together. |

---

## Test plan

Run on HEAD `0cf692c` on 2026-10-08. The five gates below the summary exited 0; the lint and run items were not run in this pass.

- [x] `make test` (exit 0; `go test -p 2 -parallel 2 ./...`; `test/chrome` fresh at 0.179s, rest cached)
- [x] `make golden` (exit 0; every corpus fixture PASS, including fixture-56 0.39s, fixture-57 1.17s, fixture-58 1.25s, fixture-59 0.69s)
- [x] `make claim-scan` (exit 0; `claim-scan: clean`)
- [x] `python3 scripts/css-catalog-map.py --check` (exit 0; `check ok (253 apply arms mapped)`)
- [x] `make build` (exit 0; `bin/gowkhtmltopdf` and `bin/gowkhtmltoimage` stamped 0.2.6)
- [ ] `make lint` not run in this pass; the 0.2.7 gate policy leaves lint to the owner, and `0029ea9` is the lint cleanup commit
- [ ] `make run` not run in this pass

### Commands

```sh
make test
make golden
make claim-scan
python3 scripts/css-catalog-map.py --check
make build

go test ./css/ -run TestRelayout -count=1
go test ./layout/ -run 'TestRelayout|TestLayoutResolvesStylesPerDocument' -count=1
go test ./internal/css -run 'TestParseColor|TestSupports|TestLayer|TestProperty|TestMatchState' -count=1
go test ./internal/layout -run 'TestMathLength|TestConicGradient|TestParseClipPathShapes|TestAuthorBackgroundShorthand' -count=1
go test ./internal/imageout -run TestPaintLineDiagonalStrokesCenterline -count=1
go test ./internal/convert/prepare -run 'TestMergeFontFaces|TestFetchFontFace' -count=1

# optional, recorded in 0b58b00, not re-run here:
go test ./layout -run '^$' -bench '^BenchmarkRelayout' -benchmem
```

---

## Screenshots / sample output

```
Catalog (not changed by this PR):
  selectors 158, functions 162, at-rules 55 (plans/0.2.6/catalog/mapping.json)
  css-catalog-map.py --check: ok (253 apply arms mapped)

HEAD 0cf692c gates (2026-10-08, all exit 0):
  make test                              go test -p 2 -parallel 2 ./... (test/chrome 0.179s)
  make golden                            all fixtures PASS (56 0.39s, 57 1.17s, 58 1.25s, 59 0.69s)
  make claim-scan                        claim-scan: clean
  css-catalog-map.py --check            check ok (253 apply arms mapped)
  make build                             both binaries stamped 0.2.6

Recorded in 0b58b00 (i7-13700HX, -cpu 1, collection only, not re-run here):
  apply-only    32.8-38.1us/op   6368 B/op   82 allocs/op
  relayout-only  8.4-12.5us/op   1280 B/op   13 allocs/op
  with DisplayList: both paths about 3.3-4.1ms/op

VERSION: 0.2.6 (no bump in this PR)
Changed samples: output/fixture-56-architecture-diagram.pdf, -57-vanguard-telemetry-audit.pdf,
                 -58-unsupported-worklist-audit.pdf, -59-apex-digital-landing.pdf
```

---

## Related issues

- No tracking issue exists for this wave.
- Relates to `plans/0.2.7/89-canonical-0.2.7-selectors-functions-atrules.md` (new in `0cf692c`; the ledger says 89.1 replaces its baseline counts).
- Relates to `plans/0.2.7/87-canonical-0.2.7-next-72.md` and `plans/0.2.7/88-canonical-0.2.7-next-100.md` for the surrounding 0.2.7 plan.

---

## PR metadata checklist (author)

- [ ] Self-assigned (GitHub PR #87 currently has no assignee)
- [ ] Labels applied (none yet; suggest `enhancement`, `documentation`)
- [x] Related issues filled (none; plan references above)
- [x] Filled body saved under `plans/0.2.7/PR/pr-027-css-functions-atrules-relayout.md`
- Suggested title: `feat(css): modern colors, math, at-rules, fonts, and relayout`
- GitHub PR: https://github.com/chinmay-sawant/gowkhtmltopdf/pull/87

---

## Follow-ups (out of scope)

- 89.1-89.8 per the new ledger: recount plus checker, structural pseudos and pseudo-elements, functional/attribute/state selectors, math and color/image functions, conditional and paged at-rules, fixture-67 closure. Phase files do not exist yet.
- Known limits left in this wave: `light-dark()` ignores color scheme; `color-mix()` supports only `in srgb`; transformed and CFF/OTTO WOFF2 are rejected; remote `.woff2`/`.eot` URL sources remain skipped; `path()`/`url()`/`shape()`/box keywords in `clip-path` are unsupported; `@property` `syntax` is stored but not validated, and `@property` nested in `@media`/`@layer` is skipped; a nested `@layer` block has its rank overwritten by the outer layer; `:not()`/`:is()`/`:where()` accept but ignore host state; `:target` and form-state pseudos still never match; `background` expands only `background-color` and `background-image`; an image resolver is synchronous and skips errors silently; `supportedDeclaration` is now a no-op filter.
- VERSION bump, `CHANGELOG` 0.2.7 section, and C/Python/site stamps when cutting the release.
- Owner runs `make lint` after the merge, per the 0.2.7 gate policy.

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff (the 26 commits map to the areas above; 7 merges are dry)
- [ ] Public API changes documented (`layout.Options`, `LayOptions`, `DisplayListOptions`, `layout.Display.Boxes`, `css.Options.Focus/Hover/Active`, `css.Relayout`)
- [ ] New rules have test coverage (color, math, at-rules, fonts, clip-path, controls, state, relayout, rasters)
- [ ] PR has assignee and labels (currently missing)
- [ ] Related issues use correct Closes/Relates keywords (none open)
- [ ] No secrets committed; `output/` samples are regenerated on purpose (4 PDFs)
- [ ] Diff-stat-by-extension table pasted at the bottom
- [ ] VERSION 0.2.6 and missing CHANGELOG 0.2.7 section are accepted as follow-ups

---

## Diff stat by extension

Generated with `bash scripts/pr-diff-stat.sh master` on `chore/changes` at `0cf692c`.

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.go` | 65 | 5832 | 240 |
| `.md` | 3 | 116 | 1 |
| `.pdf` | 4 | Binary | Binary |
| **Total** | **72** | **5948** | **241** |
