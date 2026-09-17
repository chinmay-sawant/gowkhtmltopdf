# 87 - Canonical next-72 CSS coverage (v0.2.7)

> **Parent:** `plans/0.2.7/README.md`
> **Status:** planned (ledger open; no batch `[x]` until code + package proof)
> **Estimated effort:** XL (8 batches: quick wins S–M, font M–L, VF honesty S/L, text/hyphen M–L, column/initial M–L, shape/float L–XL, border defer S, closure M)
> **Owner:** `internal/layout` (+ `internal/pdf` for font feature shaping, `internal/css` only if selector/parse gaps)
> **Depends on:** v0.2.6 catalog honesty (354 Implemented), fixture-64
> **Unblocks:** mapping recount toward ~426 Implemented; fixture-64 Effect cells that actually paint
> **Honesty:** `plans/0.2.6/HONESTY-GATES.md` flip packet required per Implemented name
> **Scan evidence:** five explore agents (font, border, text/hyphen/initial-letter, grid/object/overflow, shape/float+architecture), 2026-09-17

---

## Overview

Implement the 72 properties in `next-72-properties.json` in dependency-ordered batches.
Validation bar is the **gowk PDF** Effect cell on `fixture-64-next-72-props.html`, not Chrome.
Chrome labels in the JSON are advisory (`Chrome yes` = optional browser sanity; `Chrome no` = expect no Chrome paint).

## Effort scale (same as 0.2.6 phase-79)

| Scale | Meaning |
|-------|---------|
| **S** | Alias, thin apply, or consumer already nearly present; package tests in one sitting |
| **M** | New fields + apply + one consumer path; focused new Go file |
| **L** | Cross-cutting layout/paint (inline metrics, shapes, drop caps, multicol wrap) |
| **XL** | Pagination-coupled or multi-backend chrome (page floats, full Borders-4) |

## Architecture rules

1. **Pipeline ownership:** cascade apply (`style_*_props.go`) → layout/paint consumer → package test → matrix → mapping last.
2. **Deep modules:** small apply interface (`applyFooProps(sty, prop, raw, …) bool`) registered on `styleGroups` in `style_cascade.go`; hide parse complexity inside the new file.
3. **Do not grow** `style_properties.go` or `layout.go` (allowlist soft cap ~2000; both already over). Extract before adding cases.
4. **New fields:** `ResolvedStyle` + `initialStyle` in `style.go`, inherit only if needed (`inheritableProps` is nearly full: coalesce or widen mask before adding many inherit rows), then `go generate` style intern.
5. **Features off hot `Op`:** put font feature payloads on `opExtra` (same pattern as text language), not on every `Op`.
6. **Gates:** mid-batch package tests only. **No `make lint` in this ledger.** Full `make test` / `make golden` only in batch 87.8.

## Executive summary (batches)

| Batch | Phase file | Props | Effort | Primary seams |
|-------|------------|------:|--------|---------------|
| 87.1 Quick wins | [phase-87.1-quick-wins.md](phases/phase-87.1-quick-wins.md) | 11 | S–M | gap aliases, overflow logical, object-fit/position, counter-set, aspect-ratio, grid-auto verify |
| 87.2 Font features | [phase-87.2-font-features.md](phases/phase-87.2-font-features.md) | 18 | M–L | `style_font_feature_props.go`, synthesis, width, size-adjust → shaper |
| 87.3 Font VF honesty | [phase-87.3-font-vf-palette.md](phases/phase-87.3-font-vf-palette.md) | 3 | S / L real | optical-sizing, palette, variation-settings (honest Partial unless PDF instances) |
| 87.4 Text + hyphen | [phase-87.4-text-hyphen.md](phases/phase-87.4-text-hyphen.md) | 13 | M–L | text-box/spacing, hyphenate-limit-*, hanging-punctuation |
| 87.5 Column + initial-letter | [phase-87.5-column-initial-letter.md](phases/phase-87.5-column-initial-letter.md) | 5 | M–L | `multicol.go`, `inline_initial_letter.go` |
| 87.6 Shape + float page | [phase-87.6-shape-float.md](phases/phase-87.6-shape-float.md) | 8 | L–XL | `shape_exclusion.go`, float exclusion / page-float lite |
| 87.7 Border drafts | [phase-87.7-border-drafts.md](phases/phase-87.7-border-drafts.md) | 14 | S defer / L paint | Borders-4 / Round Display; prefer stay Unsupported |
| 87.8 Closure | [phase-87.8-closure-integration.md](phases/phase-87.8-closure-integration.md) | — | M | **`make test` + `make golden` only here**; mapping recount; matrix; fixture-64 envelope |

Property math: 11+18+3+13+5+8+14 = **72**.

## Recommended new Go files (do not grow allowlisted giants)

| File | Owns |
|------|------|
| `style_font_feature_props.go` | feature-settings, kerning, variant*, synthesis*, width/stretch, size-adjust |
| `style_font_size_adjust_props.go` | optional split if feature file grows |
| `style_text_box_props.go` | text-box / edge / trim |
| `style_text_spacing_props.go` | autospace, text-spacing*, text-fit, text-group-align |
| `style_hyphenation_props.go` | hyphenate-limit-*, hanging-punctuation |
| `style_overflow_logical.go` | overflow-block / overflow-inline → OverflowX/Y |
| `style_shape_props.go` + `shape_exclusion.go` | shape-* apply + per-line exclusion |
| `style_float_page_props.go` | float-defer / offset / reference |
| `style_border_partial_props.go` | border-clip family (only if not deferred) |
| `style_initial_letter_props.go` + `inline_initial_letter.go` | drop-cap |
| `object_fit.go` | fit/position geometry helpers extracted from `layout_images.go` if needed |

Prefer extending existing focused files when small: `style_image_adjust_props.go` (object-*), `style_paint_props.go` + `counter.go` (counter-set), `applyGapProps` home after extract (grid-gap aliases).

## Agent scan summary (2026-09-17)

- **Font:** Wave-4 apply is a no-op stub; OT feature shaper exists in `pdf` but paint passes `nil` features. Optical/palette/variation fields exist with intentional static no-op consumer.
- **Border:** No apply/paint for clip/limit/shape/boundary; Borders-4 "not ready"; 13/14 Chrome no. Prefer defer.
- **Text/hyphen/initial-letter:** `hyphens` stored but unread; text-box/spacing absent; `:first-letter` rejected in CSS parser.
- **Grid/object/overflow:** `gap`/`row-gap`/`column-gap` live; `grid-gap*` aliases missing; `grid-auto-*` apply+partial consume already; object-fit/position missing; overflow-block/inline missing; counter-set missing; aspect-ratio CSS prop missing.
- **Shape/float:** CSS2 rectangular floats only; no CSS Shapes; page-float props absent.

## Out of scope

- JavaScript, animation, scroll UI, speech, 3D, mask/clip-path hard-defer families outside these 72
- Growing `style_properties.go` / `layout.go` without a compensating extract
- `make lint` inside any batch (owner runs after)
- Claiming Implemented from apply-only or fixture authorship alone

## Handoff

Start at [phase-87.1-quick-wins.md](phases/phase-87.1-quick-wins.md). Close the program only via [phase-87.8-closure-integration.md](phases/phase-87.8-closure-integration.md).
