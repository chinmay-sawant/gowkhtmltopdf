# Phase 88.6: WebKit mask aliases

> **Parent:** `../88-canonical-0.2.7-next-100.md`
> **Status:** planned
> **Estimated effort:** S (8 properties)
> **Owner:** `internal/layout` cascade
> **Depends on:** 88.5 bases Implemented
> **Unblocks:** prefixed template CSS
> **Mid-batch gate:** package tests only. No `make lint`. No `make test`.

---

## Overview

Flip eight `-webkit-mask*` aliases only after the unprefixed bases from 88.5 actually paint. Mechanism is `normalizeVendorPrefix`. `-webkit-mask-composite` values differ (`source-over` vs `add`) and need `remapWebkitValue`. Do not include `-webkit-mask-box-image*` (those wait on mask-border).

## Properties (8)

`-webkit-mask`, `-webkit-mask-image`, `-webkit-mask-size`, `-webkit-mask-repeat`, `-webkit-mask-position`, `-webkit-mask-origin`, `-webkit-mask-clip`, `-webkit-mask-composite`

| # | Property | Family | Effort | Demo |
|--:|----------|--------|--------|------|
| 57 | `-webkit-mask` | alias | S | `url(logo.png)` |
| 58 | `-webkit-mask-image` | alias | S | `url(logo.png)` |
| 59 | `-webkit-mask-size` | alias | S | `cover` |
| 60 | `-webkit-mask-repeat` | alias | S | `no-repeat` |
| 61 | `-webkit-mask-position` | alias | S | `center` |
| 62 | `-webkit-mask-origin` | alias | S | `content-box` |
| 63 | `-webkit-mask-clip` | alias | S | `padding-box` |
| 64 | `-webkit-mask-composite` | alias | S | `source-over` |

## Architecture

- Register eight names in `normalizeVendorPrefix`.
- Prefixed tests in `TestWebkitPrefixAliases`.
- Mapping last, one alias per Implemented base.

## Checklist

### 88.6.1 scope lock

- [ ] 88.6.1.1 Confirm the 8 names below against `../next-100-properties.json`. Proof: list in `_proof-88.6.md`.

### 88.6.2 `-webkit-mask`

- [ ] 88.6.2.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_cascade.go`).
- [ ] 88.6.2.2 Consumer reads the field. Demo `url(logo.png)`.
- [ ] 88.6.2.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.6.3 `-webkit-mask-image`

- [ ] 88.6.3.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_cascade.go`).
- [ ] 88.6.3.2 Consumer reads the field. Demo `url(logo.png)`.
- [ ] 88.6.3.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.6.4 `-webkit-mask-size`

- [ ] 88.6.4.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_cascade.go`).
- [ ] 88.6.4.2 Consumer reads the field. Demo `cover`.
- [ ] 88.6.4.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.6.5 `-webkit-mask-repeat`

- [ ] 88.6.5.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_cascade.go`).
- [ ] 88.6.5.2 Consumer reads the field. Demo `no-repeat`.
- [ ] 88.6.5.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.6.6 `-webkit-mask-position`

- [ ] 88.6.6.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_cascade.go`).
- [ ] 88.6.6.2 Consumer reads the field. Demo `center`.
- [ ] 88.6.6.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.6.7 `-webkit-mask-origin`

- [ ] 88.6.7.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_cascade.go`).
- [ ] 88.6.7.2 Consumer reads the field. Demo `content-box`.
- [ ] 88.6.7.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.6.8 `-webkit-mask-clip`

- [ ] 88.6.8.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_cascade.go`).
- [ ] 88.6.8.2 Consumer reads the field. Demo `padding-box`.
- [ ] 88.6.8.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.6.9 `-webkit-mask-composite`

- [ ] 88.6.9.1 Apply arm writes a `ResolvedStyle` field (intended: `internal/layout/style_cascade.go`).
- [ ] 88.6.9.2 Consumer reads the field. Demo `source-over`.
- [ ] 88.6.9.3 Package test exit 0. Flip mapping last with an honesty packet (`HONESTY-GATES.md`).

### 88.6.R batch gate (package only)

- [ ] 88.6.R.1 Targeted `go test ./internal/<pkg> -run '…' -count=1` exit 0 (skip for 88.8 defer-only).
- [ ] 88.6.R.2 `python3 scripts/css-catalog-map.py --check` if apply arms added.
- [ ] 88.6.R.3 Mapping + matrix only for properties with flip packets. **Do not** run `make test` / `make lint` here.

## Out of scope

`-webkit-mask-box-image*` family, animation/transition aliases.
