# Phase 88.1: Quick aliases and cascade all

> **Parent:** `../88-canonical-0.2.7-next-100.md`
> **Status:** planned
> **Estimated effort:** S-M (5 properties)
> **Owner:** `internal/layout` cascade
> **Depends on:** none
> **Unblocks:** 88.2+
> **Mid-batch gate:** package tests only. No `make lint`. No `make test`.

---

## Overview

Ship the five names whose unprefixed bases already paint: three `-webkit-background-*` remaps, `-webkit-line-clamp` (apply and consumer already exist), and the `all` cascade shorthand subset `initial | inherit | unset`.

## Properties (5)

`all`, `-webkit-background-clip`, `-webkit-background-origin`, `-webkit-background-size`, `-webkit-line-clamp`

| # | Property | Family | Effort | Demo |
|--:|----------|--------|--------|------|
| 1 | `all` | cascade | M | `unset` |
| 2 | `-webkit-background-clip` | alias | S | `padding-box` |
| 3 | `-webkit-background-origin` | alias | S | `content-box` |
| 4 | `-webkit-background-size` | alias | S | `cover` |
| 5 | `-webkit-line-clamp` | overflow | S | `2` |

## Architecture

- Add the three background aliases to `normalizeVendorPrefix` in `internal/layout/style_cascade.go` next to `-webkit-text-fill-color`.
- Extend `TestWebkitPrefixAliases`.
- `-webkit-line-clamp` already hits `applyAdvancedProps`; add a prefixed alias test and flip mapping.
- Expand `all` before apply. Skip `revert` / `revert-layer`.
- Do not grow `style_properties.go`.

## Checklist

### 88.1.1 scope lock

- [ ] 88.1.1.1 Confirm the 5 names below against `../next-100-properties.json`. Proof: list in `_proof-88.1.md`.

### 88.1.2 `all`

- [ ] 88.1.2.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_cascade.go`).
- [ ] 88.1.2.2 Consumer reads the field. Demo `unset`.
- [ ] 88.1.2.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.1.3 `-webkit-background-clip`

- [ ] 88.1.3.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_cascade.go`).
- [ ] 88.1.3.2 Consumer reads the field. Demo `padding-box`.
- [ ] 88.1.3.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.1.4 `-webkit-background-origin`

- [ ] 88.1.4.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_cascade.go`).
- [ ] 88.1.4.2 Consumer reads the field. Demo `content-box`.
- [ ] 88.1.4.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.1.5 `-webkit-background-size`

- [ ] 88.1.5.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_cascade.go`).
- [ ] 88.1.5.2 Consumer reads the field. Demo `cover`.
- [ ] 88.1.5.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.1.6 `-webkit-line-clamp`

- [ ] 88.1.6.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_advanced_props.go`).
- [ ] 88.1.6.2 Consumer reads the field. Demo `2`.
- [ ] 88.1.6.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.1.R batch gate (package only)

- [ ] 88.1.R.1 Targeted `go test ./internal/<pkg> -run '…' -count=1` exit 0 (skip for 88.8 defer-only).
- [ ] 88.1.R.2 `python3 scripts/css-catalog-map.py --check` if apply arms added.
- [ ] 88.1.R.3 Mapping + matrix only for properties with flip packets. **Do not** run `make test` / `make lint` here.

## Out of scope

Text-stroke paint, SVG bake, clip-path, ruby.
