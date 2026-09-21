# Phase 88.2: Text stroke and leftover inline

> **Parent:** `../88-canonical-0.2.7-next-100.md`
> **Status:** planned
> **Estimated effort:** M-L (6 properties)
> **Owner:** `internal/layout`
> **Depends on:** 88.1 preferred
> **Unblocks:** 88.3
> **Mid-batch gate:** package tests only. No `make lint`. No `make test`.

---

## Overview

Last original E_text / E_inline leftovers outside next-72, plus compat text-stroke paint. Text-stroke is not an alias of SVG `stroke`. It needs new fields and a glyph-outline consumer.

## Properties (6)

`word-space-transform`, `baseline-source`, `inline-sizing`, `-webkit-text-stroke`, `-webkit-text-stroke-color`, `-webkit-text-stroke-width`

| # | Property | Family | Effort | Demo |
|--:|----------|--------|--------|------|
| 6 | `word-space-transform` | text | M | `space` |
| 7 | `baseline-source` | inline | M | `first` |
| 8 | `inline-sizing` | inline | M | `stretch` |
| 9 | `-webkit-text-stroke` | text | L | `1px #000` |
| 10 | `-webkit-text-stroke-color` | text | L | `#c00` |
| 11 | `-webkit-text-stroke-width` | text | L | `1px` |

## Architecture

- New `style_text_stroke_props.go` for the three `-webkit-text-stroke*` names.
- `word-space-transform` next to existing text-spacing apply.
- `baseline-source` / `inline-sizing` next to text-box apply.
- Do not grow `style_properties.go` / `layout.go`.

## Checklist

### 88.2.1 scope lock

- [ ] 88.2.1.1 Confirm the 6 names below against `../next-100-properties.json`. Proof: list in `_proof-88.2.md`.

### 88.2.2 `word-space-transform`

- [ ] 88.2.2.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_text_spacing_props.go`).
- [ ] 88.2.2.2 Consumer reads the field. Demo `space`.
- [ ] 88.2.2.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.2.3 `baseline-source`

- [ ] 88.2.3.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_text_box_props.go`).
- [ ] 88.2.3.2 Consumer reads the field. Demo `first`.
- [ ] 88.2.3.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.2.4 `inline-sizing`

- [ ] 88.2.4.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_text_box_props.go`).
- [ ] 88.2.4.2 Consumer reads the field. Demo `stretch`.
- [ ] 88.2.4.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.2.5 `-webkit-text-stroke`

- [ ] 88.2.5.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_text_stroke_props.go`).
- [ ] 88.2.5.2 Consumer reads the field. Demo `1px #000`.
- [ ] 88.2.5.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.2.6 `-webkit-text-stroke-color`

- [ ] 88.2.6.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_text_stroke_props.go`).
- [ ] 88.2.6.2 Consumer reads the field. Demo `#c00`.
- [ ] 88.2.6.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.2.7 `-webkit-text-stroke-width`

- [ ] 88.2.7.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_text_stroke_props.go`).
- [ ] 88.2.7.2 Consumer reads the field. Demo `1px`.
- [ ] 88.2.7.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.2.R batch gate (package only)

- [ ] 88.2.R.1 Targeted `go test ./internal/<pkg> -run '…' -count=1` exit 0 (skip for 88.8 defer-only).
- [ ] 88.2.R.2 `python3 scripts/css-catalog-map.py --check` if apply arms added.
- [ ] 88.2.R.3 Mapping + matrix only for properties with flip packets. **Do not** run `make test` / `make lint` here.

## Out of scope

SVG presentation bake, ruby boxes.
