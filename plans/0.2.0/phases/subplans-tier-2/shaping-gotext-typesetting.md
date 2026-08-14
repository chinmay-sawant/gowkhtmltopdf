# Tier 2 Subplan - OpenType shaping via `go-text/typesetting`

> **Parent:** [`plans/phases/phase-19-fonts-i18n.md`](../phase-19-fonts-i18n.md)  
> **Amendment:** [`plans/amendments/2026-08-05-gotext-typesetting.md`](../../amendments/2026-08-05-gotext-typesetting.md)  
> **Status:** done — wired on `feature/tier-2-pending-2`  
> **Estimated effort:** 3–8 weeks once started  
> **Constraint exception:** allow **only** `github.com/go-text/typesetting` (+ its module graph); still `CGO_ENABLED=0`

---

## Overview

Replace / augment in-tree presentation-form Arabic and NFC Indic with real
OpenType shaping through **`go-text/typesetting`**. This subplan is the
execution ledger for the 2026-08-05 amendment.

## Executive Summary

| Topic | Decision |
|-------|----------|
| Module | `github.com/go-text/typesetting` only |
| CGO HarfBuzz | Still forbidden |
| Font supply | Unchanged: `--font-path`, system fonts, local TTF/OTF `@font-face` |
| Full Noto CJK bundle | **Not required** — font-path is enough |
| WOFF/WOFF2 | **Not required** for this shaping work (see clarification below) |

---

## WOFF clarification (point 4 — do we need it?)

**Short answer: No — not for adopting `go-text/typesetting`.**

| Concern | Reality |
|---------|---------|
| What typesetting does | Shapes **glyph runs** from an already-loaded font face (GSUB/GPOS, scripts) |
| What WOFF does | **File format** wrapper around SFNT (TTF/OTF) for the web |
| Our load path today | `ParseTTF` / TrueType-flavored OTF; `@font-face` skips `.woff`/`.woff2` |
| typesetting & WOFF1 | Library can read WOFF1 internally; we do not need it unless we accept `.woff` URLs |
| WOFF2 | Needs **Brotli** (not in Go stdlib). typesetting does **not** remove that need |

**Recommendation:** keep serving faces as **TTF/OTF via `--font-path` / local `@font-face`**. Defer WOFF1/WOFF2 to a future amendment if product wants webfont URLs.

- [~] WOFF1 decode (stdlib zlib **or** typesetting reader) — optional later  
- [~] WOFF2 + Brotli module — requires a **new** constraint amendment (not this one)

---

## Alternatives (document only — do not implement)

See amendment table. Chosen: **A. go-text/typesetting**. Fallbacks if evaluation fails: in-tree OT (B), `boxesandglue/textshape` (C). Never CGO HarfBuzz (E).

---

## Phase 1: Constraint & CI (before any feature code)

### 1.1 Product docs

- [x] Land amendment language in `plans/10-canonical-post-mvp-roadmap.md` header
      Constraint: stdlib **except** allowlisted `go-text/typesetting`
- [x] README “Built from scratch / no third-party” → honesty exception sentence
- [x] Update `plans/amendments/2026-08-04-shaping-stdlib.md` pointer (done in repo)

### 1.2 Module allowlist gate

- [x] When implementing: `go get github.com/go-text/typesetting@v0.3.4`
- [x] CI: `TestDirectModuleAllowlist` in `internal/pdf` (`go list` direct ⊆ typesetting)
- [x] Prove `CGO_ENABLED=0 go test ./internal/pdf`

---

## Phase 2: Integration design (still design until scheduled)

### 2.1 Seam

- [x] Design replace point: `ShapeTextFont` / `ShapeRun` in `shape_gotext.go` →
      typesetting shaper → reverse-cmap Unicode → existing PDF Type0/CID;
      `TextShow` calls `ShapeTextFont`
- [x] Face bridge: `pdf.Font` TTF bytes → typesetting `font.Face` (cached)
- [x] Keep presentation-form path as **fallback** if face lacks OT tables

### 2.2 Scripts & features

- [x] Arabic: OT features instead of (or verified vs) presentation forms
- [x] Indic: claim **Partial** only; combining marks preserved in fallback
- [~] Optional: `halt` / `palt` if face exposes and API allows feature tags
- [x] Hangul: still requires capable face via font-path (unchanged)

### 2.3 Image mode

- [x] Same shaping seam (`ShapeTextFont`) — image mode benefits if/when it
      calls the shared API (no imageout registry rewrite in this subplan)

---

## Phase 3: Validation (when implementing)

### 3.1 Fixtures

- [x] Arabic OT tests (joining + Lam-Alef) with DejaVu GSUB face
- [x] Indic honesty (Partial; existing combining-mark test kept)
- [~] Existing fixture-27 CJK/Hangul must not regress (not re-run here; no layout change)

### 3.2 Gates

- [x] `make lint` →  
- [x] `CGO_ENABLED=0 go test ./internal/pdf` →  
- [x] Docs: fonts.md / matrix RTL/CJK rows refresh  
- [x] Flip Phase 19 shaping / amendment acceptance boxes  

---

## Phase 4: Explicit non-work

- [~] Bundling full Noto CJK — **not required** (font-path policy stands)
- [~] WOFF2 / Brotli — separate amendment if ever needed
- [~] Other Go modules beyond typesetting allowlist

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Amendment 2026-08-05 | Legal go.mod exception |
| Phase 19 registry / Type0 | Faces + PDF embed |
| Image-mode `@font-face` subplan | Optional parity for CSS-loaded faces |

---

## Out of scope

- General dependency expansion
- HarfBuzz CGO
- WOFF1/WOFF2 decode
