# Phase 07 — Image Converter (`gowkhtmltoimage`)

> **Parent:** `plans/00-canonical-pure-go-rewrite.md`  
> **Status:** not started  
> **Estimated effort:** 4–8 weeks solo after layout exists  
> **Depends on:** Phase 4 layout (+ Phase 2 loader, Phase 1 image settings)  
> **Unblocks:** feature parity with wkhtmltoimage subset

---

## Overview

Port `wkhtmltoimage`: load one page, choose viewport, render to PNG/JPEG (SVG deferred).

## Checklist

### 7.1 CLI
- [ ] All image flags from `imagearguments.cc`
- [ ] Positional `input output`
- [ ] Shared web/load/doc flags

### 7.2 Pipeline (`internal/imageout` + convert)
- [ ] Load single resource
- [ ] Viewport width = screenWidth (default 1024)
- [ ] Smart width: grow/binary-search until content width fits without horizontal overflow (approximate WebKit scrollbar check)
- [ ] Height = screenHeight or content height
- [ ] Crop rect intersection
- [ ] Transparent background for PNG when flag set; else white
- [ ] Encode PNG (`image/png`) / JPEG (`image/jpeg` + quality)
- [ ] Output path / stdout / memory

### 7.3 Deferred
- [ ] `[~]` SVG output — no QSvgGenerator; optional minimal later
- [ ] `[~]` BMP if needed

### 7.4 Tests
- [ ] Solid color HTML → PNG dimensions
- [ ] JPEG quality parameter changes size
- [ ] Crop reduces dimensions

### 7.5 Closure
- [ ] `make test` / `make lint`
- [ ] `gowkhtmltoimage` help/version + one golden PNG

---

## Upstream refs

- `imageconverter.cc` — smart width, crop, formats  
- `imagesettings.*` — defaults  
- `image.h` / examples — API shape for Phase 8  
