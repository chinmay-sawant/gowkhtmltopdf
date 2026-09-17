# Phase 88.7: Zoom, overflow, gap rules, ruby

> **Parent:** `../88-canonical-0.2.7-next-100.md`
> **Status:** planned
> **Estimated effort:** M-L (17 properties)
> **Owner:** `internal/layout`
> **Depends on:** none
> **Unblocks:** CJK print and flex/grid gap chrome
> **Mid-batch gate:** package tests only. No `make lint`. No `make test`.

---

## Overview

CSS `zoom` (operator `--zoom` already scales lengths), overflow-4 longhands `block-ellipsis` / `continue`, core css-gaps-1 row-rule and rule shorthands (not the inset-cap pile), line-padding / line-fit-edge, ruby-position/align (need a real ruby box), min-intrinsic-sizing, and column-rule-break.

## Properties (17)

`zoom`, `block-ellipsis`, `continue`, `row-rule`, `row-rule-width`, `row-rule-style`, `row-rule-color`, `rule`, `rule-width`, `rule-style`, `rule-color`, `line-padding`, `line-fit-edge`, `ruby-position`, `ruby-align`, `min-intrinsic-sizing`, `column-rule-break`

| # | Property | Family | Effort | Demo |
|--:|----------|--------|--------|------|
| 65 | `zoom` | viewport | S | `1.5` |
| 66 | `block-ellipsis` | overflow | M | `auto` |
| 67 | `continue` | overflow | M | `discard` |
| 68 | `row-rule` | gap | S | `2pt solid #c00` |
| 69 | `row-rule-width` | gap | M | `2pt` |
| 70 | `row-rule-style` | gap | M | `solid` |
| 71 | `row-rule-color` | gap | M | `#c00` |
| 72 | `rule` | gap | S | `1pt solid #666` |
| 73 | `rule-width` | gap | S | `2pt` |
| 74 | `rule-style` | gap | S | `dashed` |
| 75 | `rule-color` | gap | S | `#333` |
| 76 | `line-padding` | text | M | `0.5em` |
| 77 | `line-fit-edge` | inline | M | `cap alphabetic` |
| 78 | `ruby-position` | ruby | L | `under` |
| 79 | `ruby-align` | ruby | L | `center` |
| 80 | `min-intrinsic-sizing` | sizing | M | `zero-if-scrollbars` |
| 81 | `column-rule-break` | column | S | `none` |

## Architecture

- `style_zoom_props.go` multiplies into `engine.scale` / `zoomScale`.
- Overflow longhands in `style_advanced_props.go` next to line-clamp.
- Gap rules clone `applyColumnRuleProps` / `emitColumnRules` into flex and grid row gaps.
- New `style_ruby_props.go` + `ruby.go`. UA display for `ruby`/`rt`/`rp`. Do not leave ruby names in the SVG leftover catch-all.
- Do not claim ruby Implemented until `<rt>` sits over/under the base.

## Checklist

### 88.7.1 scope lock

- [ ] 88.7.1.1 Confirm the 17 names below against `../next-100-properties.json`. Proof: list in `_proof-88.7.md`.

### 88.7.2 `zoom`

- [ ] 88.7.2.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_zoom_props.go`).
- [ ] 88.7.2.2 Consumer reads the field. Demo `1.5`.
- [ ] 88.7.2.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.7.3 `block-ellipsis`

- [ ] 88.7.3.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_advanced_props.go; internal/layout/inline.go`).
- [ ] 88.7.3.2 Consumer reads the field. Demo `auto`.
- [ ] 88.7.3.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.7.4 `continue`

- [ ] 88.7.4.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_advanced_props.go; internal/layout/inline.go`).
- [ ] 88.7.4.2 Consumer reads the field. Demo `discard`.
- [ ] 88.7.4.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.7.5 `row-rule`

- [ ] 88.7.5.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_gap_props.go; internal/layout/flex.go`).
- [ ] 88.7.5.2 Consumer reads the field. Demo `2pt solid #c00`.
- [ ] 88.7.5.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.7.6 `row-rule-width`

- [ ] 88.7.6.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_gap_props.go; internal/layout/flex.go`).
- [ ] 88.7.6.2 Consumer reads the field. Demo `2pt`.
- [ ] 88.7.6.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.7.7 `row-rule-style`

- [ ] 88.7.7.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_gap_props.go; internal/layout/flex.go`).
- [ ] 88.7.7.2 Consumer reads the field. Demo `solid`.
- [ ] 88.7.7.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.7.8 `row-rule-color`

- [ ] 88.7.8.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_gap_props.go; internal/layout/flex.go`).
- [ ] 88.7.8.2 Consumer reads the field. Demo `#c00`.
- [ ] 88.7.8.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.7.9 `rule`

- [ ] 88.7.9.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_gap_props.go`).
- [ ] 88.7.9.2 Consumer reads the field. Demo `1pt solid #666`.
- [ ] 88.7.9.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.7.10 `rule-width`

- [ ] 88.7.10.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_gap_props.go`).
- [ ] 88.7.10.2 Consumer reads the field. Demo `2pt`.
- [ ] 88.7.10.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.7.11 `rule-style`

- [ ] 88.7.11.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_gap_props.go`).
- [ ] 88.7.11.2 Consumer reads the field. Demo `dashed`.
- [ ] 88.7.11.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.7.12 `rule-color`

- [ ] 88.7.12.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_gap_props.go`).
- [ ] 88.7.12.2 Consumer reads the field. Demo `#333`.
- [ ] 88.7.12.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.7.13 `line-padding`

- [ ] 88.7.13.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/inline.go`).
- [ ] 88.7.13.2 Consumer reads the field. Demo `0.5em`.
- [ ] 88.7.13.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.7.14 `line-fit-edge`

- [ ] 88.7.14.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_text_box_props.go; internal/layout/inline_text_box.go`).
- [ ] 88.7.14.2 Consumer reads the field. Demo `cap alphabetic`.
- [ ] 88.7.14.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.7.15 `ruby-position`

- [ ] 88.7.15.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_ruby_props.go; internal/layout/ruby.go`).
- [ ] 88.7.15.2 Consumer reads the field. Demo `under`.
- [ ] 88.7.15.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.7.16 `ruby-align`

- [ ] 88.7.16.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_ruby_props.go; internal/layout/ruby.go`).
- [ ] 88.7.16.2 Consumer reads the field. Demo `center`.
- [ ] 88.7.16.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.7.17 `min-intrinsic-sizing`

- [ ] 88.7.17.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_aspect_ratio_props.go`).
- [ ] 88.7.17.2 Consumer reads the field. Demo `zero-if-scrollbars`.
- [ ] 88.7.17.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.7.18 `column-rule-break`

- [ ] 88.7.18.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_multicol_props.go; internal/layout/multicol.go`).
- [ ] 88.7.18.2 Consumer reads the field. Demo `none`.
- [ ] 88.7.18.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.7.R batch gate (package only)

- [ ] 88.7.R.1 Targeted `go test ./internal/<pkg> -run '…' -count=1` exit 0 (skip for 88.8 defer-only).
- [ ] 88.7.R.2 `python3 scripts/css-catalog-map.py --check` if apply arms added.
- [ ] 88.7.R.3 Mapping + matrix only for properties with flip packets. **Do not** run `make test` / `make lint` here.

## Out of scope

ruby-overhang/merge, MathML, copy-into, gap inset-cap/junction drafts.
