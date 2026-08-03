# Phase 03 — PDF Object Model & Writer

> **Parent:** `plans/00-canonical-pure-go-rewrite.md`  
> **Status:** complete (2026-08-03)  
> **Estimated effort:** 8–14 weeks solo (with fonts) · 5–8 weeks pair  
> **Depends on:** Phase 0  
> **Unblocks:** Phase 4 paint backend; Phase 5–6 composition

---

## Overview

Implement a **stdlib-only PDF writer**. Upstream does **not** implement PDF — it uses `QPrinter`. We must own the entire PDF object model.

## Executive Summary

Minimal PDF 1.4 with text, images, compression, links, and outlines is ~2–4 PM; production font subsetting and Unicode maps add substantial cost. Prefer vector operators over full-page raster.

---

## Checklist

### 3.1 Core structure (`internal/pdf`)
- [x] Object store: allocate indirect refs, write body
- [x] Cross-reference table + trailer + startxref + EOF
- [x] Catalog + Pages tree + Page dicts (MediaBox, Resources, Contents)
- [x] Stream objects with optional `/FlateDecode`
- [x] Writer API: `NewDocument`, `AddPage(width, height pt)`, `Content`, `WriteTo(io.Writer)`

### 3.2 Content streams
- [x] Graphics state: save/restore, cm transform, colors (RGB + gray)
- [x] Paths: re, m/l, h, S/f
- [x] Text: Tf, Td/Tm, Tj/TJ, Tw/Tc basics
- [x] Clipping optional for later

### 3.3 Fonts
- [x] Parse TTF: head, hhea, hmtx, cmap (format 4), maxp, loca, glyf or CFF decision (TTF TrueType first)
- [x] Embed font file as FontFile2 stream
- [x] Simple font or Type0/CID for Unicode
- [x] Subset glyphs used on page
- [x] ToUnicode CMap for copy-paste
- [x] Ship or load one default Latin font (embed bytes in repo as asset via `//go:embed` — stdlib)

### 3.4 Images
- [x] JPEG: DCTDecode pass-through when possible
- [x] PNG: decode with `image/png`, re-encode Flate RGB (+ soft mask for alpha)
- [~] Scale/DPI helpers for imageDPI setting — layout computes target rect; writer paints into it

### 3.5 Annotations & outline
- [x] Annots array: Link with `/URI` or `/GoTo`
- [x] Destinations / named anchors
- [x] Outline dictionary tree (title, dest, children)
- [x] Document Info dict: Title, Creator, Producer, CreationDate

### 3.6 Settings wiring hooks
- [x] Compression on/off
- [~] Page size helpers (A4 etc. in points) — `internal/settings` owns sizes; layout converts
- [x] Grayscale: convert colors at paint time

### 3.7 Explicit defer
- [x] `[~]` Encryption — not in upstream
- [x] `[~]` AcroForm — Phase 9+ optional
- [x] `[~]` ICC / PDF/A

### 3.8 Tests & proof
- [x] Golden: minimal 1-page "Hello" PDF parses (custom structural tests)
- [x] Multi-page + internal link test
- [x] Font subset reduces file size vs full embed
- [x] Benchmark: `go test -bench=BenchmarkWrite50Pages` recorded

### 3.9 Closure
- [x] `make test` / `make lint` pass
- [x] API stable enough for layout display-list consumer

---

## Dependencies

None on HTML. Parallelizable with Phase 1–2.

## Risk

Font correctness is the #1 PDF quality risk for selectable text. Latin-only exit criterion for MVP.
