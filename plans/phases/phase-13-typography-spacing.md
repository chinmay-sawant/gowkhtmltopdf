# Phase 13 - Typography: Spacing Stability

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** not started  
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

- [ ] Trace one line of fixture-01 from layout runs → PDF content ops; document advance formula
- [ ] List all sites that compute text width: inline layout, paint, HF, bullets
- [ ] Record whether layout and PDF use the same `Font.Advance` / scale path
- [ ] Capture before screenshots/PDFs of fixture-01 and fixture-16 for visual compare

### 13.2 Layout fixes

- [ ] Consistent space width: use font space glyph advance (not ad-hoc constant) unless justified
- [ ] Trailing space on runs: do not double-count or drop inconsistently across line breaks
- [ ] Optional: coalesce adjacent same-style words on one baseline into one text op (fewer gaps, smaller streams)
- [ ] `letter-spacing` already implemented - regression test remains green
- [ ] `[~]` `word-spacing` CSS - implement only if invoice fixtures need it; else leave matrix Not implemented
- [ ] Path: `internal/layout/inline.go`, `paint.go`

### 13.3 PDF paint alignment

- [ ] `drawText` positions match layout run X for Regular and Bold faces
- [ ] After phase 12: bold face advances used in both measure and paint
- [ ] Guard: no regression of double letter-spacing (Widths must stay 1000 units/em)
- [ ] Path: `internal/pdf/{content.go,fontpdf.go,fonts.go}`

### 13.4 Tests

- [ ] Unit: known string width within tolerance for Liberation Regular at 12pt
- [ ] Unit: bold width ≥ regular width for same string when bold face present
- [ ] Regression: content-stream heuristic or layout golden for fixture-01 meta line / table cells
- [ ] `TestGoldenCorpus` still green

### 13.5 Docs

- [ ] Matrix §2.3: remaining limits (no kerning, justify still left)
- [ ] Fidelity guide: “stable spacing for Latin reports” claim only after visual gate

### 13.6 Visual gate

- [ ] fixture-01 PDF: even word spacing in viewer
- [ ] fixture-16 invoice CSS PDF: no “A c m e”-style letter stretch; natural words
- [ ] `make samples` updated if needed

### 13.7 Closure gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Parent Phase 13 checked
- [ ] Next: **Phase 14** (PDF images) or **15** if images already solid

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
