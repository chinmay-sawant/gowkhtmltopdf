# Phase 07 — Image Converter (`gowkhtmltoimage`)

> **Parent:** `plans/00-canonical-pure-go-rewrite.md`  
> **Status:** complete (2026-08-03)  
> **Estimated effort:** 4–8 weeks solo after layout exists  
> **Depends on:** Phase 4 layout (+ Phase 2 loader, Phase 1 image settings)  
> **Unblocks:** feature parity with wkhtmltoimage subset

---

## Overview

Port `wkhtmltoimage`: load one page, choose viewport, render to PNG/JPEG (SVG deferred).

## Checklist

### 7.1 CLI
- [x] All image flags from `imagearguments.cc` — `--width/--height/--crop-x/y/w/h/--format/--quality/--transparent/--smart-width/--no-smart-width` registered (`internal/cli/flags.go:345-375`); settings round-trip asserted (`cli_test.go:300-310`)
- [x] Positional `input output` — shared multi-object grammar (`TestMultiObjectGrammar` `cli_test.go:122`); image mode renders the first page object (`firstObject` `imageout.go:438-455`)
- [x] Shared web/load/doc flags — `ModeBoth` table (e.g. `--zoom` `flags.go:225`); `Global`/`Load`/`Web` wired into the loader and `RenderOptions` (`imageout.go:352-400`)

### 7.2 Pipeline (`internal/imageout` + convert)
- [x] Load single resource — `loader.Load` on the first object, skip-policy handled (`imageout.go:362-368`)
- [x] Viewport width = screenWidth (default 1024) — `screenWidthDefault` (`imageout.go:42`), applied in `Render` (`imageout.go:84-88`) and `layoutOptions` (`imageout.go:117-131`)
- [x] Smart width: grow/binary-search until content width fits without horizontal overflow (approximate WebKit scrollbar check) — `layoutSmartWidth` re-lays out at 1.5× growth while the display-list right edge overflows (`imageout.go:137-158`), measured from op extents, links ignored (`contentWidthPx` `imageout.go:162-174`), capped at 4096 px (`maxSmartViewport` `imageout.go:45`); test `TestSmartWidth` (`imageout_test.go:335`: 1536 vs 1024 px)
- [x] Height = screenHeight or content height — `maxHeight` takes the larger of content height and `--height` (`imageout.go:179-185`)
- [x] Crop rect intersection — intersection with canvas, re-origined to (0,0) (`imageout.go:96-114`); `cropRect` from settings (`imageout.go:477-482`); test `TestRenderCrop` (`imageout_test.go:132`)
- [x] Transparent background for PNG when flag set; else white — `rasterize` starts transparent only with `--transparent` (`imageout.go:190-202`); JPEG gets white composite + warning (`imageout.go:410-413`); test `TestRenderTransparent` (`imageout_test.go:116`)
- [x] Encode PNG (`image/png`) / JPEG (`image/jpeg` + quality) — `encode` clamps quality 1-100 (`imageout.go:507-528`); format from flag or output extension (`resolveFormat` `imageout.go:486-503`); test `TestEncodeFormats` (`imageout_test.go:224`)
- [x] Output path / stdout / memory — path or `-`→stdout dispatch (`imageout.go:419-433`); end-to-end via `Run` (`imageout.go:343`); test `TestRunEndToEnd` (`imageout_test.go:289`)

### 7.3 Deferred
- [ ] `[ ]` SVG output — no QSvgGenerator and no stdlib SVG encoder; deferred as optional minimal vector export
- [ ] `[ ]` BMP if needed — deferred

### 7.4 Tests
- [x] Solid color HTML → PNG dimensions — `TestRenderSolidColor` (`imageout_test.go:97`; 200 px wide, exact pixel colors)
- [x] JPEG quality parameter changes size — `TestEncodeFormats` (`imageout_test.go:224`; q10 vs q100 output sizes differ)
- [x] Crop reduces dimensions — `TestRenderCrop` (`imageout_test.go:132`; 200×100 → 100×50)
- [x] Additional coverage landed — `TestRenderTransparent` (`imageout_test.go:116`), `TestRenderText` (`imageout_test.go:160`), `TestRenderImageDataURI` (`imageout_test.go:180`), `TestFontTable` (`imageout_test.go:196`), `TestScaleNearest` (`imageout_test.go:212`), `TestSmartWidth` (`imageout_test.go:335`), `TestRunEndToEnd` (`imageout_test.go:289`)

### 7.5 Closure
- [x] `make test` / `make lint` — `go test ./...` all packages ok, `go vet ./...` exit 0, `gofmt -l .` empty (verified 2026-08-03; Makefile targets `Makefile:5-10`)
- [x] `gowkhtmltoimage` help/version + one golden PNG — help/version/extended-help wired in `cmd/gowkhtmltoimage/main.go:22-34` (`ModeImage`); golden PNG end-to-end in `TestRunEndToEnd` (`imageout_test.go:289`)

---

## Design notes (filled 2026-08-03)

1. **Text is a bitmap font, not a glyph rasterizer.** The stdlib has no text rasterizer, so image mode renders text with the embedded public-domain font5x7 table (Adafruit `font5x7.h`/`glcdfont.c`): 5×8 px glyphs at 12 pt, every other size nearest-neighbour scaled by `size/12`, bold faked by a 1 px double-draw — **no anti-aliasing** (`font.go:9-28,159-204`).
2. **Smart width grows the viewport by op extents.** The layout engine always fills the viewport width, so overflow is measured from the display list (max `op.X+op.W`, link ops ignored); `layoutSmartWidth` re-lays out at 1.5× growth until it fits, capped at 4096 px (`imageout.go:137-174`).
3. **Rasterization scale is fixed at 96 dpi.** `ptToPx = 96/72` maps layout points to output pixels (1 CSS px = 1 output px); images at natural size take a fast `draw.Draw` path, other sizes a local nearest-neighbour scaler (Go 1.26 dropped `image/draw` scalers) (`imageout.go:37,277-284,316-338`).
4. **JPEG quality and media are the only mode-specific knobs.** Quality is clamped 1–100 and PNG is lossless (`imageout.go:507-528`); `--transparent` is composited onto white with a warning for JPEG (`imageout.go:410-413`); media defaults to `"screen"` unless `--print-media-type` (`mediaFor` `imageout.go:459-473`).

## Upstream refs

- `imageconverter.cc` — smart width, crop, formats  
- `imagesettings.*` — defaults  
- `image.h` / examples — API shape for Phase 8  
