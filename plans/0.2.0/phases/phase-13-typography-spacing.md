# Phase 13 - Typography: Spacing Stability

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** complete (2026-08-04) - coalesce runs + shared advances + spacing tests
> **Estimated effort:** 1–2 weeks  
> **Depends on:** Phase 12 preferred (real bold changes advances); can start audit before 12 lands  
> **Unblocks:** cleaner invoice PDFs; image-mode advance alignment (15)  
> **Tier:** 1 #1 (spacing) · **Constraint:** stdlib-only

---

## Overview

After the 1000-unit `/Widths` fix, Latin text is readable but residual **word/letter spacing** issues remain (word-by-word `Tj`, layout vs paint advance drift, trailing space handling). This phase stabilizes spacing for report fixtures without claiming full kerning.

## Executive Summary

| Symptom | Likely site |
|---------|-------------|
| Uneven inter-word gaps | `internal/layout/inline.go` Fields split + space width |
| Fake bold width drift | paint stroke vs layout measure (fixed by 12 when bold face lands) |
| Image mode worse gaps | 5×7 advances ≠ layout (phase 15) |
| Missing/tight joins | trailing space trim on runs |

---

## Phase 13 checklist

### 13.1 Audit (evidence first)

- [x] Trace one line of fixture-01 from layout runs → PDF content ops; document advance formula
- [x] List all sites that compute text width: inline layout, paint, HF, bullets
- [x] Record whether layout and PDF use the same `Font.Advance` / scale path
- [x] Capture before screenshots/PDFs of fixture-01 and fixture-16 for visual compare

### 13.2 Layout fixes

- [x] Consistent space width: use font space glyph advance (not ad-hoc constant) unless justified
- [x] Trailing space on runs: do not double-count or drop inconsistently across line breaks
- [x] Optional: coalesce adjacent same-style words on one baseline into one text op (fewer gaps, smaller streams)
- [x] `letter-spacing` already implemented - regression test remains green
- [~] `word-spacing` CSS - implement only if invoice fixtures need it; else leave matrix Not implemented
- [x] Path: `internal/layout/inline.go`, `paint.go`

### 13.3 PDF paint alignment

- [x] `drawText` positions match layout run X for Regular and Bold faces
- [x] After phase 12: bold face advances used in both measure and paint
- [x] Guard: no regression of double letter-spacing (Widths must stay 1000 units/em)
- [x] Path: `internal/pdf/{content.go,fontpdf.go,fonts.go}`

### 13.4 Tests

- [x] Unit: known string width within tolerance for Liberation Regular at 12pt
- [x] Unit: bold width ≥ regular width for same string when bold face present
- [x] Regression: content-stream heuristic or layout golden for fixture-01 meta line / table cells
- [x] `TestGoldenCorpus` still green

### 13.5 Docs

- [x] Matrix §2.3: remaining limits (no kerning, justify still left)
- [x] Fidelity guide: “stable spacing for Latin reports” claim only after visual gate

### 13.6 Visual gate

- [x] fixture-01 PDF: even word spacing in viewer
- [x] fixture-16 invoice CSS PDF: no “A c m e”-style letter stretch; natural words
- [x] `make samples` updated if needed

### 13.7 Closure gates

- [x] `make lint` →
- [x] `make test` →
- [x] Parent Phase 13 checked
- [x] Next: **Phase 14** (PDF images) or **15** if images already solid

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 12 faces (preferred) | Phase 15 advance match |
| Prior Widths fix | Must not regress |

---

## Out of scope

- OpenType kerning
- `text-align: justify` full algorithm (optional later)
- Image-mode bitmap (15)

## Evidence (reconcile 2026-08-04)

- `coalesceTextItems` in `internal/layout/inline.go` (same-style runs → one op)
- Advances: `measureWith` / `Font.AdvanceInPoints` shared with PDF paint
- PDF `/Widths` in 1000 units/em (`fontpdf.go`) - no double letter-spacing
- Tests: `spacing_fix_test.go`, golden corpus, font width tests
- Residual: no kerning / word-spacing CSS (matrix); image-mode residual handled in phase 15

