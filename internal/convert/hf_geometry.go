package convert

import "github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"

// hfGeom is the page geometry of one object, in points. contentW/contentH
// are the content-area dimensions the object's layout was paginated with.
type hfGeom struct {
	pageW, pageH            float64
	marginTop, marginBottom float64
	marginLeft, marginRight float64
	contentW, contentH      float64
}

// recomputeContent refreshes contentW/contentH from page size and margins.
// contentH falls back to pageH when margins would leave a non-positive band.
func (g *hfGeom) recomputeContent() {
	g.contentW = g.pageW - g.marginLeft - g.marginRight
	g.contentH = g.pageH - g.marginTop - g.marginBottom

	if g.contentH <= 0 {
		g.contentH = g.pageH
	}
}

// pdfY converts a y-down canvas coordinate on object-local page locPage into
// PDF y-up coordinates (top of the box).
func (g *hfGeom) pdfY(locPage int, y float64) float64 {
	return g.pageH - g.marginTop - (y - float64(locPage)*g.contentH)
}

// pdfXY converts a y-down element location into PDF (x, y-up) destination.
func (g *hfGeom) pdfXY(loc layout.ElementLocation) (float64, float64) {
	return g.marginLeft + loc.X, g.pdfY(loc.Page, loc.Y)
}

// pdfRect converts a y-down element location into a PDF annotation rect
// [x1 y1 x2 y2] with y-up coordinates.
func (g *hfGeom) pdfRect(loc layout.ElementLocation) [4]float64 {
	x1 := g.marginLeft + loc.X
	yTop := g.pdfY(loc.Page, loc.Y)
	yBot := g.pageH - g.marginTop - (loc.Y + loc.H - float64(loc.Page)*g.contentH)

	return [4]float64{x1, yBot, x1 + loc.W, yTop}
}
