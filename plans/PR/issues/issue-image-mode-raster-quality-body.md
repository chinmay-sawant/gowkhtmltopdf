## Context

**Parent epic:** #2 - [epic: post-MVP rendering quality - image mode, fonts, CSS for real sites](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/2)

**Siblings under #2:** #3 (image-mode PNG) · #4 (font spacing) · #5 (Wikipedia CSS) · #6 (multi-font)

`make samples` writes `output/fixture-01-simple-invoice.png` via `gowkhtmltoimage`. Visual inspection of that PNG shows the invoice **layout is roughly correct** (columns, colors, header hierarchy) but the **raster result looks broken / non-production**: blocky text, odd letter/word spacing, and a “chunky pixel” look that users describe as images not displaying properly.

Note: fixture-01 HTML is **text-only** (no `<img>` logo). The defect is **image-mode rendering of the page as a PNG**, not a missing PNG logo asset. (PDF path for fixtures with real images is separate; see references.)

### Analysis of `output/fixture-01-simple-invoice.png`

| Observation | Measurement / evidence |
|-------------|------------------------|
| Canvas | **1024×409** RGB, file ~5.7 KB |
| Color depth | Only **~6 unique colors** (white, blackish greys, brand blue `#1a3d6d`, light grey rule) → no anti-aliased edges |
| Content | Non-white bbox spans almost full width; text is readable only as **nearest-neighbour scaled blocks** |
| Title | “Acme Widgets GmbH” appears as large **pixel-block** glyphs, not smooth vector/TrueType raster |
| Body | Meta line and table cells show **fixed-cell bitmap** shapes; OCR-like misreads (`I te m`, wide gaps) match low-res font |
| Word gaps | Large horizontal gaps between words (layout places each word; raster font advances ≠ layout font metrics) |

### Root cause (code)

Image mode intentionally uses a **5×7 public-domain bitmap font** because the Go stdlib has no text rasterizer:

- `internal/imageout/font.go` - `font5x7` / Adafruit-style table; `glyphCols=5`, `glyphRows=8`, `glyphAdvance=6`
- Scale: `size/12` nearest-neighbour (`drawString`); **no anti-aliasing**
- Bold: draw glyph twice with 1 px offset (fake bold)
- Layout still measures text with **Liberation Sans** metrics (`pdf.Font` / layout engine); image paint uses **fixed bitmap advances** → **metric mismatch** → spacing looks wrong even when PDF looks better
- Pipeline: `internal/imageout/imageout.go` `paint` → `drawString` for `OpText` / `OpBullet`

So “images not displaying properly” for this sample is primarily:

1. **Bitmap text quality** (blocky, no AA)  
2. **Advance mismatch** vs layout  
3. **Word-at-a-time ops** amplifying gaps  

Not: missing JPEG decoder for fixture-01 (there is no image resource in that HTML).

## Scope (in)

1. Document current image-mode limits in `documentation/` / compatibility matrix if not already clear.
2. Design and implement a better text raster path under stdlib-only constraints, e.g.:
   - Rasterize the **same embedded TTF** used by PDF (pure-Go outline fill), or
   - Higher-quality bitmap with AA / greyscale coverage, or
   - Measure text in image mode with the **same advances** used for drawing.
3. Align layout measurement with image paint advances to fix word/letter spacing on PNG.
4. Golden/regression: fixture-01 PNG smoke (visual or metric: unique colors, bbox, no absurd inter-word gaps).
5. Optional: verify `<img>` fixtures (`fixture-07`, `fixture-20`) still paint correctly after text changes.

## Out of scope

- Full FreeType/HarfBuzz / cgo font stack  
- Pixel-identical browser screenshots  
- PDF text pipeline (tracked in sibling spacing/font issues)  
- CSS for Wikipedia (sibling issue)

## Success criteria

- [ ] `output/fixture-01-simple-invoice.png` (or regenerable equivalent) shows **smooth or clearly improved** text (not 5×7 block grid at body sizes)
- [ ] Image-mode text advances **match** layout advances for the same font/size (no large spurious word gaps)
- [ ] `go test ./internal/imageout/ ./internal/convert/` pass; `make samples` updates PNG
- [ ] Docs state image-mode font strategy and limits

## Plan

- Parent epic: #2  
- Code: `internal/imageout/{font.go,imageout.go}`, layout paint bridge  
- Checklist: intermediate roadmap “image mode quality” under phase-07 notes  

## References

- Relates to #2 (parent epic)
- Artifact: `output/fixture-01-simple-invoice.png`  
- Code: `internal/imageout/font.go`, `internal/imageout/imageout.go`  
- Docs: `documentation/samples.md`, `documentation/cli.md` (image mode)  
- Related: PDF path embeds images for `fixture-07` / `fixture-20` (`/Subtype /Image` present); this issue is PNG **raster** quality

