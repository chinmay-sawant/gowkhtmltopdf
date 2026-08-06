package imageout

import (
	"image"
	"image/color"
	"math"
	"sync"

	"gowkhtmltopdf/internal/pdf"
)

// ttfDrawString draws s with face metrics and anti-aliased TTF outlines so
// image mode advances match layout/PDF (same face + AdvanceInPoints).
// basex/basey are the baseline-left position in output pixels (may be
// fractional). pxPerPt converts layout points to pixels (ptToPx, or a
// supersampled multiple of it).
//
// Text is run through pdf.ShapeTextFont first so Arabic/RTL/OT forms match
// PDF emission (Phase 2.4 image shaping parity).
func ttfDrawString(img *image.NRGBA, basex, basey float64, s string, sizePt float64, face *pdf.Font, col color.NRGBA, pxPerPt float64) {
	if face == nil || s == "" || sizePt <= 0 || pxPerPt <= 0 {
		return
	}
	s = pdf.ShapeTextFont(s, face)
	if s == "" {
		return
	}
	pxSize := sizePt * pxPerPt
	upm := float64(face.UnitsPerEm())
	if upm <= 0 {
		upm = 1000
	}
	scale := pxSize / upm
	x := basex
	for _, r := range s {
		adv := face.AdvanceInPoints(r, sizePt) * pxPerPt
		drawGlyphAA(img, x, basey, r, face, scale, col)
		x += adv
	}
}

type glyphKey struct {
	face *pdf.Font
	size int // px size rounded (em size in pixels)
	r    rune
}

type glyphCacheEntry struct {
	img     *image.Alpha
	originX float64 // top-left of alpha relative to baseline-left (font bearing)
	originY float64
}

// maxGlyphCache caps the package-level atlas so a long-lived process re-parsing
// fonts (new *pdf.Font keys) cannot grow RSS without bound (P5-05). When full,
// half the entries are dropped. Full per-Render ownership is a future step.
const maxGlyphCache = 4096

var (
	glyphMu    sync.Mutex
	glyphCache = map[glyphKey]*glyphCacheEntry{}
)

func drawGlyphAA(dst *image.NRGBA, basex, basey float64, r rune, face *pdf.Font, scale float64, col color.NRGBA) {
	if r == ' ' {
		return
	}
	sizeKey := int(math.Round(scale * float64(face.UnitsPerEm())))
	if sizeKey < 1 {
		sizeKey = 1
	}
	key := glyphKey{face: face, size: sizeKey, r: r}

	glyphMu.Lock()
	ent, ok := glyphCache[key]
	if !ok {
		if len(glyphCache) >= maxGlyphCache {
			// Drop about half; map iteration order is unspecified (stdlib).
			n := len(glyphCache) / 2
			for k := range glyphCache {
				delete(glyphCache, k)
				n--
				if n <= 0 {
					break
				}
			}
		}
		ent = rasterGlyph(face, r, scale)
		glyphCache[key] = ent
	}
	glyphMu.Unlock()
	if ent == nil || ent.img == nil {
		return
	}
	// basex/basey is baseline-left in pixels; origin is float offset from that.
	// Round once per glyph so stems share a stable pixel grid (avoids the old
	// Floor(origin) + Round(base) combination that made letters bob up/down).
	ox := int(math.Round(basex + ent.originX))
	oy := int(math.Round(basey + ent.originY))
	bounds := ent.img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			a := ent.img.AlphaAt(x, y).A
			if a == 0 {
				continue
			}
			dx, dy := ox+x, oy+y
			if !image.Pt(dx, dy).In(dst.Bounds()) {
				continue
			}
			srcA := uint32(a) * uint32(col.A) / 255
			if srcA == 0 {
				continue
			}
			i := dst.PixOffset(dx, dy)
			dr, dg, db, da := uint32(dst.Pix[i+0]), uint32(dst.Pix[i+1]), uint32(dst.Pix[i+2]), uint32(dst.Pix[i+3])
			ia := 255 - srcA
			dst.Pix[i+0] = uint8((uint32(col.R)*srcA + dr*ia) / 255)
			dst.Pix[i+1] = uint8((uint32(col.G)*srcA + dg*ia) / 255)
			dst.Pix[i+2] = uint8((uint32(col.B)*srcA + db*ia) / 255)
			dst.Pix[i+3] = uint8(srcA + da*ia/255)
		}
	}
}

func rasterGlyph(face *pdf.Font, r rune, scale float64) *glyphCacheEntry {
	contours := face.GlyphContours(r)
	if len(contours) == 0 {
		return &glyphCacheEntry{}
	}
	var flat [][]pdf.GlyphPoint
	var minX, minY, maxX, maxY float64
	first := true
	// More flatten steps at small sizes → smoother curves (less "wobbly" stems).
	steps := 8
	if scale*float64(face.UnitsPerEm()) < 18 {
		steps = 12
	}
	for _, c := range contours {
		f := pdf.FlattenContour(c, steps)
		if len(f) < 2 {
			continue
		}
		flat = append(flat, f)
		bx0, by0, bx1, by1 := pdf.ContourBounds(f)
		if first {
			minX, minY, maxX, maxY = bx0, by0, bx1, by1
			first = false
		} else {
			if bx0 < minX {
				minX = bx0
			}
			if by0 < minY {
				minY = by0
			}
			if bx1 > maxX {
				maxX = bx1
			}
			if by1 > maxY {
				maxY = by1
			}
		}
	}
	if first {
		return &glyphCacheEntry{}
	}
	// scale font units -> pixels; y flips (font y-up, image y-down)
	pad := 1.5
	x0 := minX * scale
	y0 := -maxY * scale // top in image space relative to baseline
	x1 := maxX * scale
	y1 := -minY * scale
	w := int(math.Ceil(x1-x0)) + int(2*pad) + 2
	h := int(math.Ceil(y1-y0)) + int(2*pad) + 2
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if w > 2048 || h > 2048 {
		return &glyphCacheEntry{}
	}
	alpha := image.NewAlpha(image.Rect(0, 0, w, h))
	ox := x0 - pad
	oy := y0 - pad
	// Supersample for greyscale AA. Higher factor for body-size text.
	ss := 6
	if scale*float64(face.UnitsPerEm()) < 16 {
		ss = 8
	}
	ss2 := ss * ss
	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			var cover int
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					// sample in font units
					ix := (float64(px) + (float64(sx)+0.5)/float64(ss) + ox) / scale
					iy := -((float64(py) + (float64(sy)+0.5)/float64(ss) + oy) / scale)
					if pointInGlyph(ix, iy, flat) {
						cover++
					}
				}
			}
			if cover > 0 {
				alpha.SetAlpha(px, py, color.Alpha{A: uint8(255 * cover / ss2)})
			}
		}
	}
	return &glyphCacheEntry{
		img:     alpha,
		originX: ox,
		originY: oy,
	}
}

// pointInGlyph uses even-odd fill over all contours.
func pointInGlyph(x, y float64, contours [][]pdf.GlyphPoint) bool {
	inside := false
	for _, c := range contours {
		if pointInPoly(x, y, c) {
			inside = !inside
		}
	}
	return inside
}

func pointInPoly(x, y float64, poly []pdf.GlyphPoint) bool {
	n := len(poly)
	if n < 3 {
		return false
	}
	inside := false
	j := n - 1
	for i := 0; i < n; i++ {
		yi, yj := poly[i].Y, poly[j].Y
		xi, xj := poly[i].X, poly[j].X
		if (yi > y) != (yj > y) {
			xint := (xj-xi)*(y-yi)/(yj-yi) + xi
			if x < xint {
				inside = !inside
			}
		}
		j = i
	}
	return inside
}
