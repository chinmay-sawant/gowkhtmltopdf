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
//
//nolint:cyclop,mnd // glyph drawing with rotation and spacing
func ttfDrawString(
	img *image.NRGBA,
	basex, basey float64,
	text string,
	sizePt float64,
	letterSpacing float64,
	rotateDeg float64,
	face *pdf.Font,
	col color.NRGBA,
	pxPerPt float64,
	atlas *glyphAtlas,
) {
	if face == nil || text == "" || sizePt <= 0 || pxPerPt <= 0 {
		return
	}

	if atlas == nil {
		atlas = newGlyphAtlas()
	}

	run := pdf.ShapeRun(text, face, sizePt)
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
	cursorY := basey

	for i, r := range run.Runes {
		adv := run.Advances[i]*pxPerPt + letterSpacing*pxPerPt

		drawGlyphAA(img, cursorX, cursorY, r, face, scale, col, atlas)

		switch rotateDeg {
		case -90:
			cursorY += adv
		case 90:
			cursorY -= adv
		default:
			cursorX += adv
		}
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
	if ent, ok := a.m[key]; ok {
		a.mu.Unlock()

		return ent
	}
	a.mu.Unlock()

	ent := makeEnt()

	a.mu.Lock()
	defer a.mu.Unlock()

	if existing, ok := a.m[key]; ok {
		return existing
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

	a.m[key] = ent

	return ent
}

func drawGlyphAA(
	dst *image.NRGBA,
	basex, basey float64,
	runeVal rune,
	face *pdf.Font,
	scale float64,
	col color.NRGBA,
	atlas *glyphAtlas,
) {
	if runeVal == ' ' {
		return
	}

	sizeKey := int(math.Round(scale * float64(face.UnitsPerEm())))
	if sizeKey < 1 {
		sizeKey = 1
	}

	key := glyphKey{face: face, size: sizeKey, r: runeVal}

	ent := atlas.get(key, func() *glyphCacheEntry {
		return rasterGlyph(face, runeVal, scale)
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
			dstR := uint32(dst.Pix[pixOff+0])
			dstG := uint32(dst.Pix[pixOff+1])
			dstB := uint32(dst.Pix[pixOff+2])
			dstA := uint32(dst.Pix[pixOff+3])
			invA := channelMax - srcA
			//nolint:gosec // Over blend of byte channels stays in uint8 range
			dst.Pix[pixOff+0] = uint8((uint32(col.R)*srcA + dstR*invA) / channelMax)
			//nolint:gosec // Over blend of byte channels stays in uint8 range
			dst.Pix[pixOff+1] = uint8((uint32(col.G)*srcA + dstG*invA) / channelMax)
			//nolint:gosec // Over blend of byte channels stays in uint8 range
			dst.Pix[pixOff+2] = uint8((uint32(col.B)*srcA + dstB*invA) / channelMax)
			//nolint:gosec // Over blend of byte channels stays in uint8 range
			dst.Pix[pixOff+3] = uint8(srcA + dstA*invA/channelMax)
		}
	}
}

func rasterGlyph(face *pdf.Font, runeVal rune, scale float64) *glyphCacheEntry {
	contours := face.GlyphContours(runeVal)
	if len(contours) == 0 {
		return &glyphCacheEntry{} //nolint:exhaustruct // intentional zero/partial fields
	}

	// More flatten steps at small sizes → smoother curves (less "wobbly" stems).
	steps := 8
	if scale*float64(face.UnitsPerEm()) < minHintScale {
		steps = 12
	}

	flat, minX, minY, maxX, maxY := flattenGlyphContours(contours, steps)
	if len(flat) == 0 {
		return &glyphCacheEntry{} //nolint:exhaustruct // intentional zero/partial fields
	}

	edges := makeGlyphEdgeList(flat)
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

	originX := minXPx - pad
	originY := minYPx - pad
	// Supersample for greyscale AA. Higher factor for body-size text.
	subsample := defaultSubsample
	if scale*float64(face.UnitsPerEm()) < minAAScale {
		subsample = 8
	}

	ss2 := subsample * subsample

	alpha := rasterGlyphAlpha(edges, width, height, subsample, ss2, originX, originY, scale)

	return &glyphCacheEntry{
		img:     alpha,
		originX: originX,
		originY: originY,
	}
}

// makeGlyphEdgeList converts flattened contours into one flat edge list.
func makeGlyphEdgeList(flat [][]pdf.GlyphPoint) []glyphEdge {
	edges := make([]glyphEdge, 0, len(flat)*boxFilterFactor2)

	for _, contour := range flat {
		edges = append(edges, makeGlyphEdges(contour)...)
	}

	return edges
}

// rasterGlyphAlpha supersamples every pixel and writes the coverage into an
// alpha mask (the pixel loop of rasterGlyph, kept separate for scanline
// locality between the active-edge and sampling passes).
//
//nolint:wsl // scanline scratch setup precedes the supersampling loops.
func rasterGlyphAlpha(
	flatEdges []glyphEdge,
	width, height, subsample, ss2 int,
	originX, originY, scale float64,
) *image.Alpha {
	alpha := image.NewAlpha(image.Rect(0, 0, width, height))
	var activeRows [8][]glyphEdge
	for sampleY := range subsample {
		activeRows[sampleY] = make([]glyphEdge, 0, len(flatEdges))
	}

	for pixelY := range height {
		for sampleY := range subsample {
			fontY := -((float64(pixelY) + (float64(sampleY)+pixelCenter)/float64(subsample) + originY) / scale)
			activeRows[sampleY] = activeEdgesInto(activeRows[sampleY][:0], flatEdges, fontY)
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
				alpha.SetAlpha(pixelX, pixelY, color.Alpha{
					//nolint:gosec // cover <= subsample^2, so 255*cover/ss2 <= 255
					A: uint8(255 * cover / ss2),
				})
			}
		}
	}

	return alpha
}

// flattenGlyphContours flattens each contour and returns the union bounds of
// the flattened polygons (min/max in font units).
func flattenGlyphContours(
	contours [][]pdf.GlyphPoint,
	steps int,
) ([][]pdf.GlyphPoint, float64, float64, float64, float64) {
	flat := make([][]pdf.GlyphPoint, 0, len(contours))

	var minX, minY, maxX, maxY float64

	for _, c := range contours {
		pts := pdf.FlattenContour(c, steps)
		if len(pts) < boxFilterFactor2 {
			continue
		}

		flat = append(flat, pts)

		bx0, by0, bx1, by1 := pdf.ContourBounds(pts)
		if len(flat) == 1 {
			minX, minY, maxX, maxY = bx0, by0, bx1, by1
		} else {
			minX = min(minX, bx0)
			minY = min(minY, by0)
			maxX = max(maxX, bx1)
			maxY = max(maxY, by1)
		}
	}

	return flat, minX, minY, maxX, maxY
}

// activeEdgesInto fills a reusable list with edges crossing fontY. Reusing the
// backing array avoids one allocation for every supersampled scanline.
func activeEdgesInto(active, flatEdges []glyphEdge, fontY float64) []glyphEdge {
	for _, edge := range flatEdges {
		if fontY >= edge.yMin && fontY < edge.yMax {
			active = append(active, edge)
		}
	}

	return active
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

func pointInActiveEdges(x, y float64, edges []glyphEdge) bool {
	inside := false

	for _, edge := range edges {
		if x < edge.xAtMin+(y-edge.yMin)*edge.dxdy {
			inside = !inside
		}
	}

	return inside
}
