package imageout

import (
	"image"
	"image/color"
	"math"
	"sync"

	"gowkhtmltopdf/internal/pdf"
)

const (
	minHintScale     = 18
	minAAScale       = 16
	minPolygonVerts  = 3
	defaultSubsample = 6
)

// ttfDrawString draws s with face metrics and anti-aliased TTF outlines so
// image mode advances match layout/PDF (same face + AdvanceInPoints).
// basex/basey are the baseline-left position in output pixels (may be
// fractional). pxPerPt converts layout points to pixels (ptToPx, or a
// supersampled multiple of it). atlas caches glyph bitmaps for one raster
// run; nil creates a private atlas (tests / one-shot callers).
//
// Text is run through pdf.ShapeTextFont first so Arabic/RTL/OT forms match
// PDF emission (Phase 2.4 image shaping parity).
func ttfDrawString(img *image.NRGBA, basex, basey float64, s string, sizePt float64, face *pdf.Font, col color.NRGBA, pxPerPt float64, atlas *glyphAtlas) {
	if face == nil || s == "" || sizePt <= 0 || pxPerPt <= 0 {
		return
	}

	if atlas == nil {
		atlas = newGlyphAtlas()
	}

	run := pdf.ShapeRun(s, face, sizePt)
	if run.Text == "" {
		return
	}

	pxSize := sizePt * pxPerPt

	upm := float64(face.UnitsPerEm())
	if upm <= 0 {
		upm = 1000
	}

	scale := pxSize / upm
	cursorX := basex

	for i, r := range run.Runes {
		adv := run.Advances[i] * pxPerPt

		drawGlyphAA(img, cursorX, basey, r, face, scale, col, atlas)
		cursorX += adv
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

// maxGlyphCache caps one atlas so a long display list (or tests that reuse an
// atlas across many sizes) cannot grow RSS without bound (P5-05). When full,
// half the entries are dropped.
const maxGlyphCache = 4096

// glyphAtlas holds rasterized glyph bitmaps for one Render/rasterize run.
// Per-run ownership avoids concurrent Renders contending on a package map
// and keeps eviction local to that run's working set.
type glyphAtlas struct {
	mu sync.Mutex
	m  map[glyphKey]*glyphCacheEntry
}

func newGlyphAtlas() *glyphAtlas {
	return &glyphAtlas{m: make(map[glyphKey]*glyphCacheEntry)} //nolint:exhaustruct // intentional zero/partial fields
}

func (a *glyphAtlas) get(key glyphKey, makeEnt func() *glyphCacheEntry) *glyphCacheEntry {
	if a == nil {
		return makeEnt()
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if ent, ok := a.m[key]; ok {
		return ent
	}

	if len(a.m) >= maxGlyphCache {
		// Drop about half; map iteration order is unspecified (stdlib).
		count := len(a.m) / boxFilterFactor2

		for k := range a.m {
			delete(a.m, k)

			count--
			if count <= 0 {
				break
			}
		}
	}

	ent := makeEnt()
	a.m[key] = ent

	return ent
}

func drawGlyphAA(dst *image.NRGBA, basex, basey float64, r rune, face *pdf.Font, scale float64, col color.NRGBA, atlas *glyphAtlas) {
	if r == ' ' {
		return
	}

	sizeKey := int(math.Round(scale * float64(face.UnitsPerEm())))
	if sizeKey < 1 {
		sizeKey = 1
	}

	key := glyphKey{face: face, size: sizeKey, r: r}

	ent := atlas.get(key, func() *glyphCacheEntry {
		return rasterGlyph(face, r, scale)
	})
	if ent == nil || ent.img == nil {
		return
	}
	// basex/basey is baseline-left in pixels; origin is float offset from that.
	// Round once per glyph so stems share a stable pixel grid (avoids the old
	// Floor(origin) + Round(base) combination that made letters bob up/down).
	originX := int(math.Round(basex + ent.originX))
	originY := int(math.Round(basey + ent.originY))

	bounds := ent.img.Bounds()
	for row := bounds.Min.Y; row < bounds.Max.Y; row++ {
		for pixelX := bounds.Min.X; pixelX < bounds.Max.X; pixelX++ {
			alpha := ent.img.AlphaAt(pixelX, row).A
			if alpha == 0 {
				continue
			}

			dstX, dstY := originX+pixelX, originY+row
			if !image.Pt(dstX, dstY).In(dst.Bounds()) {
				continue
			}

			srcA := uint32(alpha) * uint32(col.A) / channelMax
			if srcA == 0 {
				continue
			}

			pixOff := dst.PixOffset(dstX, dstY)
			dr, dg, db, da := uint32(dst.Pix[pixOff+0]), uint32(dst.Pix[pixOff+1]), uint32(dst.Pix[pixOff+2]), uint32(dst.Pix[pixOff+3])
			ia := channelMax - srcA
			dst.Pix[pixOff+0] = uint8((uint32(col.R)*srcA + dr*ia) / channelMax)
			dst.Pix[pixOff+1] = uint8((uint32(col.G)*srcA + dg*ia) / channelMax)
			dst.Pix[pixOff+2] = uint8((uint32(col.B)*srcA + db*ia) / channelMax)
			dst.Pix[pixOff+3] = uint8(srcA + da*ia/channelMax)
		}
	}
}

func rasterGlyph(face *pdf.Font, r rune, scale float64) *glyphCacheEntry {
	contours := face.GlyphContours(r)
	if len(contours) == 0 {
		return &glyphCacheEntry{} //nolint:exhaustruct // intentional zero/partial fields
	}

	var flat [][]pdf.GlyphPoint

	var minX, minY, maxX, maxY float64

	first := true
	// More flatten steps at small sizes → smoother curves (less "wobbly" stems).
	steps := 8
	if scale*float64(face.UnitsPerEm()) < minHintScale {
		steps = 12
	}

	for _, c := range contours {
		pts := pdf.FlattenContour(c, steps)
		if len(pts) < boxFilterFactor2 {
			continue
		}

		flat = append(flat, pts)

		bx0, by0, bx1, by1 := pdf.ContourBounds(pts)
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
		return &glyphCacheEntry{} //nolint:exhaustruct // intentional zero/partial fields
	}

	edges := make([][]glyphEdge, 0, len(flat))
	for _, contour := range flat {
		edges = append(edges, makeGlyphEdges(contour))
	}
	// scale font units -> pixels; y flips (font y-up, image y-down)
	pad := 1.5
	minXPx := minX * scale
	minYPx := -maxY * scale // top in image space relative to baseline
	x1 := maxX * scale
	y1 := -minY * scale
	width := int(math.Ceil(x1-minXPx)) + int(boxFilterFactor2*pad) + boxFilterFactor2
	height := int(math.Ceil(y1-minYPx)) + int(boxFilterFactor2*pad) + boxFilterFactor2

	if width < 1 {
		width = 1
	}

	if height < 1 {
		height = 1
	}

	if width > 2048 || height > 2048 {
		return &glyphCacheEntry{} //nolint:exhaustruct // intentional zero/partial fields
	}

	alpha := image.NewAlpha(image.Rect(0, 0, width, height))
	originX := minXPx - pad
	originY := minYPx - pad
	// Supersample for greyscale AA. Higher factor for body-size text.
	subsample := defaultSubsample
	if scale*float64(face.UnitsPerEm()) < minAAScale {
		subsample = 8
	}

	ss2 := subsample * subsample

	flatEdges := make([]glyphEdge, 0)
	for _, contour := range edges {
		flatEdges = append(flatEdges, contour...)
	}

	for pixelY := range height {
		var activeStorage [8][64]glyphEdge

		var activeRows [8][]glyphEdge

		for sampleY := range subsample {
			fontY := -((float64(pixelY) + (float64(sampleY)+pixelCenter)/float64(subsample) + originY) / scale)
			active := activeStorage[sampleY][:0]

			var overflow []glyphEdge

			for _, edge := range flatEdges {
				if fontY >= edge.yMin && fontY < edge.yMax {
					if overflow != nil {
						overflow = append(overflow, edge)
					} else if len(active) < len(activeStorage[sampleY]) {
						active = append(active, edge)
					} else {
						overflow = append(append([]glyphEdge(nil), active...), edge)
					}
				}
			}

			if overflow != nil {
				activeRows[sampleY] = overflow
			} else {
				activeRows[sampleY] = active
			}
		}

		for pixelX := range width {
			var cover int

			for sampleY := range subsample {
				fontY := -((float64(pixelY) + (float64(sampleY)+pixelCenter)/float64(subsample) + originY) / scale)
				active := activeRows[sampleY]

				for sampleX := range subsample {
					// sample in font units
					ix := (float64(pixelX) + (float64(sampleX)+pixelCenter)/float64(subsample) + originX) / scale
					if pointInActiveEdges(ix, fontY, active) {
						cover++
					}
				}
			}

			if cover > 0 {
				alpha.SetAlpha(pixelX, pixelY, color.Alpha{A: uint8(255 * cover / ss2)})
			}
		}
	}

	return &glyphCacheEntry{
		img:     alpha,
		originX: originX,
		originY: originY,
	}
}

type glyphEdge struct {
	yMin, yMax float64
	xAtMin     float64
	dxdy       float64
}

func makeGlyphEdges(poly []pdf.GlyphPoint) []glyphEdge {
	if len(poly) < minPolygonVerts {
		return nil
	}

	edges := make([]glyphEdge, 0, len(poly))

	for idx := range poly {
		j := idx - 1
		if j < 0 {
			j = len(poly) - 1
		}

		a, b := poly[idx], poly[j]
		if a.Y == b.Y {
			continue
		}

		low, high := a, b
		if low.Y > high.Y {
			low, high = high, low
		}

		edges = append(edges, glyphEdge{
			yMin:   low.Y,
			yMax:   high.Y,
			xAtMin: low.X,
			dxdy:   (high.X - low.X) / (high.Y - low.Y),
		})
	}

	return edges
}

func pointInGlyphEdges(x, y float64, contours [][]glyphEdge) bool {
	inside := false

	for _, edges := range contours {
		if pointInEdges(x, y, edges) {
			inside = !inside
		}
	}

	return inside
}

func pointInEdges(x, y float64, edges []glyphEdge) bool {
	inside := false

	for _, edge := range edges {
		if y >= edge.yMin && y < edge.yMax && x < edge.xAtMin+(y-edge.yMin)*edge.dxdy {
			inside = !inside
		}
	}

	return inside
}

func pointInActiveEdges(x, y float64, edges []glyphEdge) bool {
	inside := false

	for _, edge := range edges {
		if x < edge.xAtMin+(y-edge.yMin)*edge.dxdy {
			inside = !inside
		}
	}

	return inside
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
	nVerts := len(poly)
	if nVerts < minPolygonVerts {
		return false
	}

	inside := false

	prev := nVerts - 1
	for idx := range nVerts {
		yi, yj := poly[idx].Y, poly[prev].Y
		xi, xj := poly[idx].X, poly[prev].X

		if (yi > y) != (yj > y) {
			xint := (xj-xi)*(y-yi)/(yj-yi) + xi
			if x < xint {
				inside = !inside
			}
		}

		prev = idx
	}

	return inside
}
