# Phase 50: Values, units, logical properties

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 50
> **Status:** complete (clamp, hsl, logical box, currentColor, and viewport-unit proofs verified 2026-08-28)
> **Estimated effort:** 5-8 days
> **Owner:** `internal/css/values.go`, `internal/layout/style_values.go`, `style_properties.go`
> **Depends on:** Phase 49 not strictly required; can overlap after 48
> **Unblocks:** Phase 51-52 (lengths and colors used by new properties)

---

## Overview

`calcLength` handles the supported arithmetic forms, and `clampLength` resolves a bounded length before layout (`internal/layout/style_values.go`). `ParseColor` handles `hsl()` and `hsla()` alongside the existing RGB forms. Logical properties map to physical fields for horizontal writing mode; vertical writing remains partial.

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

- [x] 50.4.1 Matrix §3: `ex` Partial 0.5em. Proof: matrix text.
- [x] 50.4.2 `ch` from glyph advance of U+0030 on the default Liberation face. Proof: `TestChUsesZeroGlyphAdvance`. `ex` stays 0.5em.
- [x] 50.4.3 Matrix still says `vw`/`vh` are width/height/min/max only.
- [x] 50.4.4 `vmin`/`vmax` Partial via `vminVmaxPt`. Matrix units row. Proof: `TestVminVmax`.

### 50.5 gates

- [x] 50.5.1 Mapping flipped via `--write`. Proof: `--check` exit 0.
- [x] 50.5.2 Full `make lint`/`make test`/`make golden`/`make claim-scan` and `make build` exit 0 on 2026-08-28. Fixture-56 golden remains 21 pages.

## Dependencies

`lengthBox` / `marginLen` / `ParseColor` as they exist today.

## Evidence

Package tests listed above. Matrix §3.


## Agent guard

Read `../HONESTY-GATES.md` before any mapping flip. This phase is marked complete only for the **print subset documented in the matrix**. Do not broaden to Chrome-complete or re-fake Implemented rows with empty `code_path`.

## Out of scope

Color 4 `oklch` / `color-mix`. `cq*` units. Full `calc` grammar trees.

## Handoff

Next is Phase 51.
