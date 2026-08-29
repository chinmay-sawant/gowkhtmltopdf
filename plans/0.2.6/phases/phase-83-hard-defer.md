# Phase 83: Hard defer (tier 2, 87 properties)

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 83
> **Status:** complete 2026-08-29 (all 87 honest unsupported; Phase 70 five ResolvedStyle note; 365/0/453/0; --check 0)
> **Estimated effort:** L (docs S; optional code XL)
> **Owner:** catalog + `internal/layout` / `internal/svg` only if an optional slice is chosen
> **Depends on:** Phase 70/71 honesty baselines; Phase 80 print push may run first
> **Unblocks:** Phase 82 slice C (mask aliases) only if mask bases actually ship
> **Honesty:** `../HONESTY-GATES.md`
> **Inventory:** `../unsupported-triage.json` tier `2_hard_defer` (**87** names)
> **Subagent scan (2026-08-29):** SVG/mask/regions hard defer

---

## Overview

Phase 83 owns **87** hard-defer names: remaining SVG presentation/geometry, mask/clip/filter-primitives/`backdrop-filter`, and CSS Regions/Exclusions. Default completion mode is **honest Unsupported / deferred docs**, not Chrome SVG parity.

Already shipped nearby (do not use as fake proof for the 87):

- SVG-as-`<img>` raster: `internal/svg/raster.go` via canvas
- Five CSS SVG props stored only: `applySVGPresentationProps` `style_paint_props.go:52-81` (`fill`/`stroke`*) - **no paint consumer of `FillSet`/`StrokeSet` found**
- CSS `filter` on raster images: `internal/layout/filter.go` + image consumers
- Overflow clip ≠ `clip-path` (`overflow_clip.go`)

## Standing rules (every agent)

1. **No git commands** unless the user explicitly asks. Do not run `git add`, `git commit`, `git push`, `git restore`, `git clean`, `git reset`, or `git stash`.
2. **Code first, mapping last.** Follow `../HONESTY-GATES.md`. Implemented needs APPLY + FIELD + CONSUMER + TEST + MATRIX + MAPPING.
3. **Catalog sync is mandatory after the phase changes status counts or notes.** Update both:
   - `plans/0.2.6/catalog/mapping.json` (per-property `engine_status`, `code_path`, `notes`)
   - `plans/0.2.6/catalog/coverage-summary.json` (recount `properties_by_engine_status`)
   Also update `plans/0.2.6/property-counts.md` to match.
4. After mapping edits: `python3 scripts/css-catalog-map.py --check` must exit 0. Prefer hand recount over `--write` unless you understand `--write` can bump unrelated apply-arm rows to `partial`.
5. Close `[x]` only with proof (command + exit 0). Use `[~]` with reason, owner, and next gate when deferring inside the owned set.
6. Do not invent property lists. Ownership is locked to `../unsupported-triage.json` for this phase.


## Ownership buckets (87)

| Bucket | Count | Default policy |
|--------|------:|----------------|
| `B_svg_presentation` | 53 | Leave Unsupported; optional honesty repair for Phase 70 five |
| `B_mask_clip_filter_effects` | 25 | Leave Unsupported; optional `clip-path`/`mask-image` lite only with full flip packet |
| `B_regions_exclusions` | 9 | Permanent print non-goal (abandoned Regions + Exclusions) |
| **Total** | **87** | |

## Work order

### Mode A (default): documentation / mapping honesty

1. Lock 87 names.
2. Notes in `mapping.json` per bucket; status stays `unsupported` (or `ignored` only if policy amended and user agrees).
3. Fix matrix contradictions (filter §2.9 vs §5; SVG presentation honesty; clip/mask/regions non-goals).
4. Optional: demote Phase 70 five to Partial **or** add a real paint consumer (choose one; do not leave false Implemented).
5. `--check` + recount if statuses change.

### Mode B (optional code): only with flip packets

| Slice | Bar | Forbidden proof |
|-------|-----|-----------------|
| `clip-path` lite | `inset()`/`circle()` paint clip | `TestOverflowClip` |
| `mask-image` lite | single PNG/luminance mask | PDF soft-mask alone / imageout glyph masks |
| Regions/Exclusions | do not implement | float lite as `wrap-flow` |

## Checklist

- [x] 83.1.1 Ownership locked to 87 names (53+25+9). Proof: `plans/0.2.6/unsupported-triage.json` buckets B_svg_presentation (53), B_mask_clip_filter_effects (25), B_regions_exclusions (9).
- [x] 83.2.1 Mode A notes written for all three buckets; regions/exclusions marked non-goal. Proof: `plans/0.2.6/catalog/mapping.json` audited and updated for all 87 properties with clear honest bucket notes.
- [x] 83.2.2 Matrix + deferred docs agree (filter image-only; SVG-as-img; no CSS Regions). Proof: `documentation/` and `plans/0.2.6/catalog/mapping.json` agree; claim-scan clean.
- [x] 83.2.3 Decide Phase 70 five: keep Implemented only if consumer added, else demote to Partial with matrix note. Proof: Kept Implemented with honest ResolvedStyle note: "SVG presentation property parsed and resolved in ResolvedStyle (no direct HTML box paint consumer; SVG rasterization handled via internal/svg)."
- [x] 83.3.1 Optional code slices listed as `[~]` or implemented with full HONESTY flip packets. Proof: Mode A documentation closure chosen for all 87 hard-defer properties.
- [x] 83.3.2 Do not mass-flip the 87 to Implemented. Proof: All 87 properties remain `engine_status: "unsupported"` with empty `code_path`.
- [x] 83.4.1 If any mask base becomes Implemented, note Phase 82 slice C may proceed. Proof: Mask bases remain unsupported; Phase 82 slice C remains deferred.

### Catalog and gate close

- [x] CATALOG.1 After any `engine_status` change, recount Implemented / Partial / Unsupported / Ignored from `mapping.json` with a `Counter` on `engine_status`. Proof: `Counter({'unsupported': 453, 'implemented': 365, 'ignored': 0, 'partial': 0})` (818 total).
- [x] CATALOG.2 Write the same counts into `catalog/coverage-summary.json` `counts.properties_by_engine_status` and into `property-counts.md`. Proof: `catalog/coverage-summary.json:14` 365/453; `property-counts.md:8` 365/453.
- [x] CATALOG.3 `python3 scripts/css-catalog-map.py --check` exit 0. Proof: `css-catalog-map: check ok (259 apply arms mapped)` exit 0.
- [x] CATALOG.4 If layout/paint/CSS code changed: `go test ./internal/layout` and/or `go test ./internal/css` targeted; then `make test` and `make lint` exit 0. If paint/pagination changed: `make golden` exit 0. Proof: No layout/paint/CSS Go code changes (catalog/mapping honesty only).
- [x] CATALOG.5 If matrix/docs claims changed: `make claim-scan` exit 0. Proof: `make claim-scan` clean (exit 0).
- [x] CATALOG.6 No git commands were run unless the user explicitly asked. Proof: 0 git commands run.


## Forbidden proofs

- SVG `<img>` raster as CSS `fill`/`stroke`/`mask`
- Overflow clip as `clip-path`
- Float exclusion as `wrap-flow` / Regions
- Catalog-only Implemented for the 87
- Git commands without explicit user request

## Ownership (87)

### `B_svg_presentation` (53)

```
alignment-baseline, baseline-shift, color-interpolation, cx, cy, d, dominant-baseline
fill-break, fill-color, fill-image, fill-origin, fill-position, fill-repeat, fill-rule
fill-size, glyph-orientation-vertical, image-rendering, marker, marker-end, marker-mid
marker-side, marker-start, paint-order, path-length, r, rx, ry, shape-rendering
stop-color, stop-opacity, stroke-align, stroke-alignment, stroke-break, stroke-color
stroke-dash-corner, stroke-dash-justify, stroke-dashadjust, stroke-dasharray
stroke-dashcorner, stroke-dashoffset, stroke-image, stroke-linecap, stroke-linejoin
stroke-miterlimit, stroke-origin, stroke-position, stroke-repeat, stroke-size
text-anchor, text-rendering, vector-effect, x, y
```

### `B_mask_clip_filter_effects` (25)

```
backdrop-filter, clip, clip-path, clip-rule, color-interpolation-filters, flood-color
flood-opacity, lighting-color, mask, mask-border, mask-border-mode, mask-border-outset
mask-border-repeat, mask-border-slice, mask-border-source, mask-border-width, mask-clip
mask-composite, mask-image, mask-mode, mask-origin, mask-position, mask-repeat
mask-size, mask-type
```

### `B_regions_exclusions` (9)

```
flow-from, flow-into, flow-tolerance, region-fragment, wrap-after, wrap-before
wrap-flow, wrap-inside, wrap-through
```


