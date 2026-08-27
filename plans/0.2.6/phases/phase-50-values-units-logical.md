# Phase 50: Values, units, logical properties

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 50
> **Status:** not started
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

- [ ] 50.1.1 Implement `clamp(min, pref, max)` next to `calcLength`. Nested calc inside clamp is optional; if skipped, document. Proof: `go test ./internal/layout -run TestClampLength`.
- [ ] 50.1.2 Remove `clamp(` from the `supportedDeclaration` exclude list once it computes, so clamp can win without a dummy fallback. Proof: cascade test where the only value is `clamp(12px, 2vw, 24px)`.
- [ ] 50.1.3 Keep `color-mix(` / `light-dark(` / `oklch(` excluded unless a later row takes them. Proof: those strings still in `supportedDeclaration`.

### 50.2 colors

- [ ] 50.2.1 `hsl()` / `hsla()` in `internal/css/values.go` `ParseColor`. Proof: `go test ./internal/css -run TestParseColorHsl`.
- [ ] 50.2.2 `currentColor` on border/outline when those consumers exist; for this phase, at least `border-*-color: currentColor` uses `style.Color`. Proof: layout test. If outline is Phase 52, land border only here.

### 50.3 logical box

- [ ] 50.3.1 Map `margin-inline` / `margin-inline-start` / `margin-inline-end` / `margin-block*` onto physical margin fields for `horizontal-tb`. Proof: `go test ./internal/layout -run TestLogicalMargin`.
- [ ] 50.3.2 Same for `padding-inline*` / `padding-block*`. Proof: `TestLogicalPadding`.
- [ ] 50.3.3 `inset` / `inset-block` / `inset-inline` map onto `top`/`right`/`bottom`/`left`. Proof: `TestLogicalInset`.
- [ ] 50.3.4 `inline-size` / `block-size` / `min-inline-size` / `max-inline-size` map onto width/height for horizontal-tb. Proof: `TestLogicalSize`.
- [ ] 50.3.5 Vertical-rl/lr: keep physical fallbacks, document Partial. Do not pretend logical works in vertical until Phase 55 or a later amendment.

### 50.4 units honesty

- [ ] 50.4.1 Matrix §3: `ex`/`ch` Partial, 0.5em at `container.go:133-134`. Proof: matrix text.
- [ ] 50.4.2 Optional: `ch` from the used font's `0` advance if `Measure` is already available. Otherwise `[~]`.
- [ ] 50.4.3 `vw`/`vh` on margin/padding via `marginLen`, or a matrix sentence that they remain width/height/min/max only. Pick one. Proof: test or matrix.
- [ ] 50.4.4 `vmin`/`vmax` either resolve as min/max of vw/vh or stay unsupported in mapping. Do not leave them as silent parse-fail without a matrix row.

### 50.5 gates

- [ ] 50.5.1 Flip mapping property rows for everything landed. Proof: `scripts/css-catalog-map.py --check` if Phase 48 script exists, else manual mapping edit.
- [ ] 50.5.2 `make lint` and `make test`. Golden if pagination or box sizes of existing fixtures change. Record tails.

## Dependencies

`lengthBox` / `marginLen` / `ParseColor` as they exist today.

## Evidence

Package tests listed above. Matrix §3.

## Out of scope

Color 4 `oklch` / `color-mix`. `cq*` units. Full `calc` grammar trees.

## Handoff

Next is Phase 51.
