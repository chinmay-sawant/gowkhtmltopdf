# Phase 50: Values, units, logical properties

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 50
> **Status:** in progress (clamp, hsl, logical box landed; currentColor and vmin/vmax still open)
> **Estimated effort:** 5-8 days
> **Owner:** `internal/css/values.go`, `internal/layout/style_values.go`, `style_properties.go`
> **Depends on:** Phase 49 not strictly required; can overlap after 48
> **Unblocks:** Phase 51-52 (lengths and colors used by new properties)

---

## Overview

`calcLength` only accepts `calc(A + B)`, `calc(A - B)`, `calc(A * N)` with exactly three tokens (`internal/layout/style_values.go:593-636`). `clamp(` is stripped in `supportedDeclaration` (`style_cascade.go:517-529`) so a fallback can win. `ParseColor` has hex, rgb/rgba, transparent, and a short name table. No `hsl()`. Logical properties parse then ignore.

Modern report CSS uses `clamp()` for type scale and `margin-inline` for horizontal page padding. Those should resolve for `writing-mode: horizontal-tb`. Vertical logical mapping waits on real vertical layout, which stays Partial.

## Goals

- `clamp()` computes on lengths
- `hsl()` / `hsla()` parse
- Logical box longhands map to physical fields in horizontal-tb
- Unit rows in the matrix match `LengthToPt`

## Checklist

### 50.1 clamp and calc

- [x] 50.1.1 `clampLength` next to `calcLength`. Proof: `go test ./internal/layout -run TestClampLength`.
- [x] 50.1.2 `clamp(` removed from `supportedDeclaration`. Fixture-56 page envelope is now 21. Proof: `TestClampLength` last-wins case.
- [x] 50.1.3 `color-mix(` / `light-dark(` / `oklch(` still excluded. Proof: `TestClampLength` mix case.

### 50.2 colors

- [x] 50.2.1 `hsl()` / `hsla()` in `ParseColor`. Proof: `go test ./internal/css -run TestParseColorHsl`.
- [x] 50.2.2 `currentColor` on color, border-*-color, and outline-color. Proof: `TestCurrentColor`.

### 50.3 logical box

- [x] 50.3.1 Logical margins for horizontal-tb. Proof: `TestLogicalMargin`.
- [x] 50.3.2 Logical padding. Proof: `TestLogicalPadding`.
- [x] 50.3.3 Logical inset. Proof: `TestLogicalInset`.
- [x] 50.3.4 Logical inline/block size. Proof: `TestLogicalSize`.
- [x] 50.3.5 Vertical still physical-only (Partial). No vertical mapping added.

### 50.4 units honesty

- [x] 50.4.1 Matrix §3: `ex`/`ch` Partial, 0.5em. Proof: matrix text.
- [~] 50.4.2 `ch` from glyph advance of `0` not done.
- [x] 50.4.3 Matrix still says `vw`/`vh` are width/height/min/max only.
- [ ] 50.4.4 `vmin`/`vmax` still not a matrix row of their own beyond the "Not implemented" catch-all.

### 50.5 gates

- [x] 50.5.1 Mapping flipped via `--write`. Proof: `--check` exit 0.
- [~] 50.5.2 Full `make lint`/`make test` not run. `go test ./internal/layout` exit 0. Fixture-56 golden 21 pages.

## Dependencies

`lengthBox` / `marginLen` / `ParseColor` as they exist today.

## Evidence

Package tests listed above. Matrix §3.

## Out of scope

Color 4 `oklch` / `color-mix`. `cq*` units. Full `calc` grammar trees.

## Handoff

Next is Phase 51.
