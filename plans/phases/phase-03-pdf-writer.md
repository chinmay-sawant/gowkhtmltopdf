# Phase 03 — PDF Object Model & Writer

> **Parent:** `plans/00-canonical-pure-go-rewrite.md`  
> **Status:** not started  
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
- [ ] Object store: allocate indirect refs, write body
- [ ] Cross-reference table + trailer + startxref + EOF
- [ ] Catalog + Pages tree + Page dicts (MediaBox, Resources, Contents)
- [ ] Stream objects with optional `/FlateDecode`
- [ ] Writer API: `NewDocument`, `AddPage(width, height pt)`, `Content`, `WriteTo(io.Writer)`

### 3.2 Content streams
- [ ] Graphics state: save/restore, cm transform, colors (RGB + gray)
- [ ] Paths: re, m/l, h, S/f
- [ ] Text: Tf, Td/Tm, Tj/TJ, Tw/Tc basics
- [ ] Clipping optional for later

### 3.3 Fonts
- [ ] Parse TTF: head, hhea, hmtx, cmap (format 4), maxp, loca, glyf or CFF decision (TTF TrueType first)
- [ ] Embed font file as FontFile2 stream
- [ ] Simple font or Type0/CID for Unicode
- [ ] Subset glyphs used on page
- [ ] ToUnicode CMap for copy-paste
- [ ] Ship or load one default Latin font (embed bytes in repo as asset via `//go:embed` — stdlib)

### 3.4 Images
- [ ] JPEG: DCTDecode pass-through when possible
- [ ] PNG: decode with `image/png`, re-encode Flate RGB (+ soft mask for alpha)
- [ ] Scale/DPI helpers for imageDPI setting

### 3.5 Annotations & outline
- [ ] Annots array: Link with `/URI` or `/GoTo`
- [ ] Destinations / named anchors
- [ ] Outline dictionary tree (title, dest, children)
- [ ] Document Info dict: Title, Creator, Producer, CreationDate

### 3.6 Settings wiring hooks
- [ ] Compression on/off
- [ ] Page size helpers (A4 etc. in points)
- [ ] Grayscale: convert colors at paint time

### 3.7 Explicit defer
- [ ] `[~]` Encryption — not in upstream
- [ ] `[~]` AcroForm — Phase 9+ optional
- [ ] `[~]` ICC / PDF/A

### 3.8 Tests & proof
- [ ] Golden: minimal 1-page "Hello" PDF parses (custom structural tests)
- [ ] Multi-page + internal link test
- [ ] Font subset reduces file size vs full embed
- [ ] Benchmark: `go test -bench=BenchmarkWrite50Pages` recorded

### 3.9 Closure
- [ ] `make test` / `make lint` pass
- [ ] API stable enough for layout display-list consumer

---

## Dependencies

None on HTML. Parallelizable with Phase 1–2.

## Risk

Font correctness is the #1 PDF quality risk for selectable text. Latin-only exit criterion for MVP.
