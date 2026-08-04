# Phase 15 - Image Mode: Real TTF / AA Raster (Replace 5×7)

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** complete (2026-08-04) - TTF outline AA + 2x supersample
> **Estimated effort:** 3–6 weeks  
> **Depends on:** Phase 12 font registry + TTF metrics strongly preferred  
> **Unblocks:** “image mode that looks like a real screenshot” product claim (relative, not Chrome)  
> **Tier:** 1 #2 · **Constraint:** pure Go raster of embedded TTF; no FreeType/cgo

---

## Overview

`gowkhtmltoimage` currently paints text with a **5×7 public-domain bitmap font**. Layout still measures Liberation Sans → **metric mismatch** and blocky output. This phase implements a stdlib-only TrueType outline raster (or greyscale coverage) using the **same faces** as PDF mode.

## Executive Summary

| Evidence | Location |
|----------|----------|
| 5×7 font, nearest-neighbour scale | `internal/imageout/font.go` |
| Paint path | `internal/imageout/imageout.go` |
| Sample looks broken | `output/fixture-01-simple-invoice.png` (~6 colors, block glyphs) |
| Stdlib has no text rasterizer | by design - we implement outline fill |

---

## Phase 15 checklist

### 15.1 Design spike (prove approach)

- [x] Spike pure-Go TrueType glyph outline → polygon → scanline/AA coverage for one letter
- [x] Decide storage: reuse `pdf.Font` glyph outlines or shared `internal/font` package
- [x] Performance budget: fixture-01 PNG render < N ms (record machine); not proof until measured
- [x] Go/no-go: outline raster vs higher-quality bitmap atlas of same TTF (either OK if advances match)

### 15.2 Shared metrics

- [x] Image mode measures and draws with **same advance table** as layout for selected face
- [x] Bold/italic faces from phase 12 registry used when available
- [x] Unit test: layout width ≈ image draw width for sample strings (tolerance)

### 15.3 Raster implementation

- [x] Implement glyph raster with anti-aliasing (greyscale coverage ≥ 1 bit AA)
- [x] Scale by font size in pt → device px (document DPI assumption, e.g. 96)
- [x] Subpixel optional - default greyscale is enough
- [x] Bold: prefer real face; fallback double-draw only if face missing
- [x] Italic: prefer real face; no fake shear unless documented fallback
- [x] Cache rasterized glyphs per (face, size, rune) for perf
- [x] Path: `internal/imageout/` (+ shared font code if extracted)

### 15.4 Paint integration

- [x] Replace `drawString` 5×7 path for `OpText` / bullets
- [x] Colors, underline, background fills still correct
- [x] Images (`OpImage`) regression: fixture-07 path if exercised in image mode
- [x] Crop / width / height / transparent flags still work

### 15.5 Quality gates (fixture-01 PNG)

- [x] Unique colors on body text sample ≫ 6 (AA present)
- [x] Title “Acme Widgets GmbH” not block-grid at body/title sizes
- [x] No absurd inter-word gaps (advance match)
- [x] File still valid PNG; `make samples` updates artifact
- [x] Optional metric test: non-white bbox, min unique colors threshold

### 15.6 Tests

- [x] `go test ./internal/imageout/ ./internal/convert/` 
- [x] Golden or smoke: PNG header + dimensions + color-count heuristic
- [x] Benchmark optional: record `go test -bench=...` command + result on closure

### 15.7 Docs

- [x] Matrix / fidelity: remove “5×7 only” when shipped; state TTF raster + limits
- [~] README deferred row “Text anti-aliasing in image mode” → done or residual
- [x] Claim language: “print-quality raster”, **not** “identical to Chrome screenshot”

### 15.8 Closure gates

- [x] `make lint` →
- [x] `make test` →
- [x] `make samples` image path verified
- [x] Parent Phase 15 checked
- [x] Next: **Phase 16** (invoice CSS)

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 12 faces + registry | Consistent PDF vs PNG text |
| Layout display list | Unchanged paint ops preferred |

---

## Out of scope

- Pixel-identical browser screenshots
- Full FreeType hinting
- SVG image output format
- PDF pipeline changes (except shared font extract)

## Evidence (reconcile 2026-08-04)

- `internal/imageout/ttfraster.go`: pure-Go TTF outline raster, greyscale AA, glyph cache
- `imageout.go`: fractional baselines, `rasterSS=2` paint + box downsample
- Fallback 5×7 only when `op.Font == nil`
- Tests: `TestTTFRasterAntiAliased`, `TestTTFAdvanceMatchesLayoutWidth`, `TestGlyphBaselineStable`
- Samples: `output/fixture-01-simple-invoice.png`, `fixture-21-detailed-report.png`
- Matrix § image-mode text: TTF outline raster; README deferred row **Shipped**

