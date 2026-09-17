# Phase 88.8: Honesty defer (mask-border, math, filters)

> **Parent:** `../88-canonical-0.2.7-next-100.md`
> **Status:** planned
> **Estimated effort:** S defer (19 properties)
> **Owner:** catalog
> **Depends on:** none
> **Unblocks:** honest fixture-65 Effect cells
> **Mid-batch gate:** package tests only. No `make lint`. No `make test`.

---

## Overview

Nineteen names stay Unsupported this wave, same product choice as next-72 batch 87.7 (Borders-4). No apply/paint stubs. Fixture Effect cells are honesty panels, not fake live demos. Do not flip mapping.

## Properties (19)

`ruby-overhang`, `ruby-merge`, `math-style`, `math-depth`, `math-shift`, `copy-into`, `mask-border`, `mask-border-source`, `mask-border-slice`, `mask-border-width`, `mask-border-outset`, `mask-border-repeat`, `mask-border-mode`, `flood-color`, `flood-opacity`, `lighting-color`, `color-interpolation-filters`, `backdrop-filter`, `path-length`

| # | Property | Family | Effort | Demo |
|--:|----------|--------|--------|------|
| 82 | `ruby-overhang` | ruby | L | `auto` |
| 83 | `ruby-merge` | ruby | L | `merge` |
| 84 | `math-style` | math | L | `compact` |
| 85 | `math-depth` | math | L | `auto-add` |
| 86 | `math-shift` | math | L | `compact` |
| 87 | `copy-into` | gcpm | XL | `none` |
| 88 | `mask-border` | mask | XL | `url(logo.png) 30 / 10px` |
| 89 | `mask-border-source` | mask | L | `url(logo.png)` |
| 90 | `mask-border-slice` | mask | L | `30` |
| 91 | `mask-border-width` | mask | M | `10px` |
| 92 | `mask-border-outset` | mask | M | `4px` |
| 93 | `mask-border-repeat` | mask | M | `round` |
| 94 | `mask-border-mode` | mask | S | `luminance` |
| 95 | `flood-color` | filter | L | `red` |
| 96 | `flood-opacity` | filter | L | `0.4` |
| 97 | `lighting-color` | filter | XL | `white` |
| 98 | `color-interpolation-filters` | filter | L | `sRGB` |
| 99 | `backdrop-filter` | filter | XL | `blur(8px)` |
| 100 | `path-length` | svg | M | `100` |

## Architecture

- Record the defer in this file and in `next-100-properties.json` (`honest_defer: true`).
- Matrix / deferred docs if the defer is newly explicit.
- Leave `mapping.json` `engine_status: unsupported`.

## Checklist

### 88.8.1 scope lock

- [ ] 88.8.1.1 Confirm the 19 names below against `../next-100-properties.json`. Proof: list in `_proof-88.8.md`.
- [ ] 88.8.1.2 Record **defer all 19**. No apply/paint stubs. Effect cells are honesty panels.
- [ ] 88.8.1.3 Leave `mapping.json` `engine_status: unsupported` for all 19.

### 88.8.R batch gate (package only)

- [ ] 88.8.R.1 Targeted `go test ./internal/<pkg> -run '…' -count=1` exit 0 (skip for 88.8 defer-only).
- [ ] 88.8.R.2 `python3 scripts/css-catalog-map.py --check` if apply arms added.
- [ ] 88.8.R.3 Mapping + matrix only for properties with flip packets. **Do not** run `make test` / `make lint` here.

## Out of scope

Implementing any of these 19 in this wave without a full honesty packet.
