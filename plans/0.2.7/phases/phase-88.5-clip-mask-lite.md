# Phase 88.5: Clip and mask lite

> **Parent:** `../88-canonical-0.2.7-next-100.md`
> **Status:** planned
> **Estimated effort:** L (13 properties)
> **Owner:** `internal/layout` + `internal/pdf`
> **Depends on:** none
> **Unblocks:** 88.6
> **Mid-batch gate:** package tests only. No `make lint`. No `make test`.

---

## Overview

Paint clip and a single PNG/luminance mask. `clip-path` lite is `none | inset() | circle()`. `clip` is the obsolete abspos rect. Mask longhands clone background size/repeat/position/origin/clip. Forbidden proofs: `TestOverflowClip`, PNG `/SMask` alone.

## Properties (13)

`clip-path`, `clip`, `clip-rule`, `mask-image`, `mask-mode`, `mask-size`, `mask-repeat`, `mask-position`, `mask-origin`, `mask-clip`, `mask`, `mask-composite`, `mask-type`

| # | Property | Family | Effort | Demo |
|--:|----------|--------|--------|------|
| 44 | `clip-path` | clip | L | `inset(12px)` |
| 45 | `clip` | clip | S | `rect(10px, 90px, 90px, 10px)` |
| 46 | `clip-rule` | clip | S | `evenodd` |
| 47 | `mask-image` | mask | L | `url(logo.png)` |
| 48 | `mask-mode` | mask | M | `alpha` |
| 49 | `mask-size` | mask | M | `cover` |
| 50 | `mask-repeat` | mask | M | `no-repeat` |
| 51 | `mask-position` | mask | M | `center` |
| 52 | `mask-origin` | mask | S | `content-box` |
| 53 | `mask-clip` | mask | S | `padding-box` |
| 54 | `mask` | mask | M | `url(logo.png) center / contain no-repeat` |
| 55 | `mask-composite` | mask | M | `add` |
| 56 | `mask-type` | mask | M | `alpha` |

## Architecture

- New `style_mask_clip_props.go`, `clip_path.go`, `mask_image.go`.
- Reuse `canonicalShapeInset` / `canonicalShapeCircle` from `style_shape_props.go`.
- Clip wraps ops in Save / path / Clip / paint / Restore. PDF needs `W*` for evenodd.
- Mask uses a Form group times luminance/alpha. Geometry clones `background_image.go`.
- `mask-type` needs a real SVG `<mask>` or mask-image consumer, or it moves to 88.8.

## Checklist

### 88.5.1 scope lock

- [ ] 88.5.1.1 Confirm the 13 names below against `../next-100-properties.json`. Proof: list in `_proof-88.5.md`.

### 88.5.2 `clip-path`

- [ ] 88.5.2.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_mask_clip_props.go; internal/layout/clip_path.go`).
- [ ] 88.5.2.2 Consumer reads the field. Demo `inset(12px)`.
- [ ] 88.5.2.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.5.3 `clip`

- [ ] 88.5.3.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_mask_clip_props.go; internal/layout/clip_path.go`).
- [ ] 88.5.3.2 Consumer reads the field. Demo `rect(10px, 90px, 90px, 10px)`.
- [ ] 88.5.3.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.5.4 `clip-rule`

- [ ] 88.5.4.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_mask_clip_props.go; internal/pdf/content.go`).
- [ ] 88.5.4.2 Consumer reads the field. Demo `evenodd`.
- [ ] 88.5.4.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.5.5 `mask-image`

- [ ] 88.5.5.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_mask_clip_props.go; internal/layout/mask_image.go`).
- [ ] 88.5.5.2 Consumer reads the field. Demo `url(logo.png)`.
- [ ] 88.5.5.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.5.6 `mask-mode`

- [ ] 88.5.6.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_mask_clip_props.go; internal/layout/mask_image.go`).
- [ ] 88.5.6.2 Consumer reads the field. Demo `alpha`.
- [ ] 88.5.6.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.5.7 `mask-size`

- [ ] 88.5.7.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_mask_clip_props.go; internal/layout/mask_image.go`).
- [ ] 88.5.7.2 Consumer reads the field. Demo `cover`.
- [ ] 88.5.7.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.5.8 `mask-repeat`

- [ ] 88.5.8.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_mask_clip_props.go; internal/layout/mask_image.go`).
- [ ] 88.5.8.2 Consumer reads the field. Demo `no-repeat`.
- [ ] 88.5.8.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.5.9 `mask-position`

- [ ] 88.5.9.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_mask_clip_props.go; internal/layout/mask_image.go`).
- [ ] 88.5.9.2 Consumer reads the field. Demo `center`.
- [ ] 88.5.9.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.5.10 `mask-origin`

- [ ] 88.5.10.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_mask_clip_props.go; internal/layout/mask_image.go`).
- [ ] 88.5.10.2 Consumer reads the field. Demo `content-box`.
- [ ] 88.5.10.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.5.11 `mask-clip`

- [ ] 88.5.11.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_mask_clip_props.go; internal/layout/mask_image.go`).
- [ ] 88.5.11.2 Consumer reads the field. Demo `padding-box`.
- [ ] 88.5.11.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.5.12 `mask`

- [ ] 88.5.12.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_mask_clip_props.go`).
- [ ] 88.5.12.2 Consumer reads the field. Demo `url(logo.png) center / contain no-repeat`.
- [ ] 88.5.12.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.5.13 `mask-composite`

- [ ] 88.5.13.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_mask_clip_props.go; internal/layout/mask_image.go`).
- [ ] 88.5.13.2 Consumer reads the field. Demo `add`.
- [ ] 88.5.13.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.5.14 `mask-type`

- [ ] 88.5.14.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_mask_clip_props.go; internal/svg`).
- [ ] 88.5.14.2 Consumer reads the field. Demo `alpha`.
- [ ] 88.5.14.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.5.R batch gate (package only)

- [ ] 88.5.R.1 Targeted `go test ./internal/<pkg> -run '…' -count=1` exit 0 (skip for 88.8 defer-only).
- [ ] 88.5.R.2 `python3 scripts/css-catalog-map.py --check` if apply arms added.
- [ ] 88.5.R.3 Mapping + matrix only for properties with flip packets. **Do not** run `make test` / `make lint` here.

## Out of scope

polygon/path/url clip-path, mask-border, backdrop-filter, `-webkit-mask*` (88.6).
