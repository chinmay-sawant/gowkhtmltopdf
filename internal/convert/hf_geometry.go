package convert

import "github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"

// hfGeom is the page geometry of one object, in points. contentW/contentH
// are the content-area dimensions the object's layout was paginated with.
type hfGeom struct {
	pageW, pageH            float64
	marginTop, marginBottom float64
	marginLeft, marginRight float64
	contentW, contentH      float64
	// first is the @page :first margin box. Nil means page 1 uses unnamed or
	// :right. Size stays unnamed: the writer paints one page size.
	first *hfPageMargins `exhaustruct:"optional"`
	// left / right are @page :left / :right. LTR: page 1 is :right, even
	// pages :left, odd pages :right. :first wins on page 1.
	left  *hfPageMargins `exhaustruct:"optional"`
	right *hfPageMargins `exhaustruct:"optional"`
	// named maps lower-case page names from @page ident { margin }.
	named map[string]*hfPageMargins `exhaustruct:"optional"`
}

// hfPageMargins is one page-box margin set in points.
type hfPageMargins struct {
	top, right, bottom, left float64
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

// pageMargins returns the page-box left and top margins for locPage.
// Cascade: unnamed, then :left/:right by page side, then :first on page 1.
// Named @page margins are applied at paint (layout.PaintOptions); link
// destinations here use the side/:first cascade only.
func (g *hfGeom) pageMargins(locPage int) (float64, float64) {
	left, top := g.marginLeft, g.marginTop
	pageNum := locPage + 1

	if pageNum%2 == 0 && g.left != nil {
		left, top = g.left.left, g.left.top
	}

	if pageNum%2 == 1 && g.right != nil {
		left, top = g.right.left, g.right.top
	}

	if locPage == 0 && g.first != nil {
		return g.first.left, g.first.top
	}

	return left, top
}

// pdfY converts a y-down canvas coordinate on object-local page locPage into
// PDF y-up coordinates (top of the box).
func (g *hfGeom) pdfY(locPage int, y float64) float64 {
	_, marginTop := g.pageMargins(locPage)

	return g.pageH - marginTop - (y - float64(locPage)*g.contentH)
}

// pdfXY converts a y-down element location into PDF (x, y-up) destination.
func (g *hfGeom) pdfXY(loc layout.ElementLocation) (float64, float64) {
	marginLeft, _ := g.pageMargins(loc.Page)

	return marginLeft + loc.X, g.pdfY(loc.Page, loc.Y)
}

// pdfRect converts a y-down element location into a PDF annotation rect
// [x1 y1 x2 y2] with y-up coordinates.
func (g *hfGeom) pdfRect(loc layout.ElementLocation) [4]float64 {
	marginLeft, marginTop := g.pageMargins(loc.Page)
	x1 := marginLeft + loc.X
	yTop := g.pdfY(loc.Page, loc.Y)
	yBot := g.pageH - marginTop - (loc.Y + loc.H - float64(loc.Page)*g.contentH)

	return [4]float64{x1, yBot, x1 + loc.W, yTop}
}
