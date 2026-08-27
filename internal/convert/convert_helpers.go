package convert

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// applyCSSPageMargins applies @page margin shorthand and size to the PDF
// viewport after stylesheets have been loaded. CSS page box properties
// describe the printable page box, so they must be resolved before body layout
// rather than cascaded as ordinary element padding.
//
// Unnamed @page (sheet.Page and Pages with Sel "") sets the default geometry
// for every page. @page :left / :right override even / odd pages (LTR: page 1
// is :right). @page :first then overrides page 1. Named @page ident rules
// store per-name margins for pages that start with that name. Size is
// unnamed-only: the writer paints one page size for the document.
func applyCSSPageMargins(geom hfGeom, sheets []*css.Stylesheet) hfGeom {
	box := collectPageBox(sheets)
	geom = applyPageSize(geom, box.size)
	geom = applyPageMargin(geom, box.margin)
	geom.recomputeContent()
	geom.first = pageMarginOverride(geom, box.firstMargin)
	geom.left = pageMarginOverride(geom, box.leftMargin)
	geom.right = pageMarginOverride(geom, box.rightMargin)
	geom.named = namedPageMarginOverrides(geom, box.named)
	geom.pageBoxes = box.boxes

	return geom
}

func pageMarginOverride(geom hfGeom, raw string) *hfPageMargins {
	applied, ok := tryApplyPageMargin(geom, raw)
	if !ok {
		return nil
	}

	return &hfPageMargins{
		top:    applied.marginTop,
		right:  applied.marginRight,
		bottom: applied.marginBottom,
		left:   applied.marginLeft,
	}
}

func namedPageMarginOverrides(geom hfGeom, raw map[string]string) map[string]*hfPageMargins {
	if len(raw) == 0 {
		return nil
	}

	out := make(map[string]*hfPageMargins, len(raw))

	for name, margin := range raw {
		if over := pageMarginOverride(geom, margin); over != nil {
			out[name] = over
		}
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

type pageBoxRaw struct {
	margin, size, firstMargin, leftMargin, rightMargin string
	named                                              map[string]string
	boxes                                              css.PageMarginBoxes
}

func collectPageBox(sheets []*css.Stylesheet) pageBoxRaw {
	var box pageBoxRaw

	for _, sheet := range sheets {
		if sheet == nil {
			continue
		}

		box = applyUnnamedPageBox(box, sheet.Page)
		for _, rule := range sheet.Pages {
			box = applyOnePageRule(box, rule)
		}
	}

	return box
}

func applyUnnamedPageBox(box pageBoxRaw, page *css.PageStyle) pageBoxRaw {
	if page == nil {
		return box
	}

	if margin := strings.TrimSpace(page.Margin); margin != "" {
		box.margin = margin
	}

	if size := strings.TrimSpace(page.Size); size != "" {
		box.size = size
	}

	return box
}

func applyOnePageRule(box pageBoxRaw, rule css.PageRule) pageBoxRaw {
	margin := strings.TrimSpace(rule.Margin)
	size := strings.TrimSpace(rule.Size)
	sel := strings.ToLower(strings.TrimSpace(rule.Sel))

	switch sel {
	case "":
		return applyUnnamedPageRule(box, margin, size, rule.Boxes)
	case ":first":
		box.firstMargin = firstNonEmpty(margin, box.firstMargin)
	case ":left":
		box.leftMargin = firstNonEmpty(margin, box.leftMargin)
	case ":right":
		box.rightMargin = firstNonEmpty(margin, box.rightMargin)
	default:
		box = applyNamedPageRule(box, sel, margin)
	}

	return box
}

func applyUnnamedPageRule(box pageBoxRaw, margin, size string, boxes css.PageMarginBoxes) pageBoxRaw {
	box.margin = firstNonEmpty(margin, box.margin)
	box.size = firstNonEmpty(size, box.size)
	box.boxes = mergePageMarginBoxes(box.boxes, boxes)

	return box
}

func applyNamedPageRule(box pageBoxRaw, sel, margin string) pageBoxRaw {
	if strings.HasPrefix(sel, ":") || margin == "" {
		return box
	}

	if box.named == nil {
		box.named = map[string]string{}
	}

	box.named[sel] = margin

	return box
}

//nolint:cyclop,wsl,varnamelen,mnd,nestif // compact CSS size expansion
func applyPageSize(geom hfGeom, rawSize string) hfGeom {
	if rawSize == "" {
		return geom
	}

	parts := strings.Fields(rawSize)
	switch len(parts) {
	case 1:
		if w, h, err := settings.ParsePageSize(parts[0]); err == nil && w > 0 && h > 0 {
			geom.pageW = w
			geom.pageH = h
		}
	case 2:
		if w, h, err := settings.ParsePageSize(parts[0]); err == nil && w > 0 && h > 0 {
			if strings.EqualFold(parts[1], "landscape") {
				geom.pageW = h
				geom.pageH = w
			} else {
				geom.pageW = w
				geom.pageH = h
			}
		} else {
			v1, u1, ok1 := css.ParseLength(parts[0])
			v2, u2, ok2 := css.ParseLength(parts[1])
			if ok1 && ok2 {
				pt1, okPt1 := css.LengthToPt(v1, u1, 12)
				pt2, okPt2 := css.LengthToPt(v2, u2, 12)
				if okPt1 && okPt2 && pt1 > 0 && pt2 > 0 {
					geom.pageW = pt1
					geom.pageH = pt2
				}
			}
		}
	}

	return geom
}

func applyPageMargin(geom hfGeom, rawMargin string) hfGeom {
	out, _ := tryApplyPageMargin(geom, rawMargin)

	return out
}

//nolint:cyclop,varnamelen,mnd // compact CSS margin shorthand expansion
func tryApplyPageMargin(geom hfGeom, rawMargin string) (hfGeom, bool) {
	if rawMargin == "" {
		return geom, false
	}

	parts := strings.Fields(rawMargin)
	if len(parts) == 0 || len(parts) > 4 {
		return geom, false
	}

	vals := make([]float64, len(parts))

	for idx, part := range parts {
		value, unit, ok := css.ParseLength(part)
		if !ok {
			return geom, false
		}

		pt, ok := css.LengthToPt(value, unit, 12)
		if !ok || pt < 0 {
			return geom, false
		}

		vals[idx] = pt
	}

	var top, right, bottom, left float64

	switch len(vals) {
	case 1:
		top, right, bottom, left = vals[0], vals[0], vals[0], vals[0]
	case 2:
		top, bottom = vals[0], vals[0]
		right, left = vals[1], vals[1]
	case 3:
		top, right, left, bottom = vals[0], vals[1], vals[1], vals[2]
	case 4:
		top, right, left, bottom = vals[0], vals[1], vals[2], vals[3]
	}

	geom.marginTop = top
	geom.marginRight = right
	geom.marginBottom = bottom
	geom.marginLeft = left

	return geom, true
}

// measuredWidth returns the effective content width of a layout result: the
// reported Result.Width, raised to the widest visual op extent when the
// report only mirrors the viewport (layout currently sets Result.Width to
// Options.Width - see internal/layout/layout.go - so over-wide fixed-width
// boxes show up only as op extents). Text and link ops never force a page
// wider, so they are ignored; rects and images are what push content out.
func measuredWidth(res *layout.Result) float64 {
	width := res.Width

	for _, op := range res.Ops {
		switch op.Kind {
		case layout.OpFillRect, layout.OpStrokeRect, layout.OpImage:
			if ext := op.X + op.W; ext > width {
				width = ext
			}
		case layout.OpLine, layout.OpText, layout.OpLinkURI, layout.OpBullet:
			// Text and link ops never force a page wider; ignore.
			continue
		}
	}

	return width
}

// pageGeometry resolves the page size in points from the single size model:
// Size.Width/Height (mm) override a named PageSize.
// Landscape swaps the pair. Legacy PageWidth/PageHeight fields are gone.
func pageGeometry(glob settings.PdfGlobal) (float64, float64, error) {
	var width, height float64

	if glob.Size.Width > 0 && glob.Size.Height > 0 {
		width, height = glob.Size.Width*mmToPt, glob.Size.Height*mmToPt
	} else {
		name := glob.PageSize

		var err error

		width, height, err = settings.ParsePageSize(name)
		if err != nil {
			return 0, 0, fmt.Errorf("parse page size %q: %w", name, err)
		}
	}

	if glob.Orientation == settings.OrientationLandscape {
		width, height = height, width
	}

	return width, height, nil
}

// mediaFor resolves layout CSS media for PDF mode via settings.ResolvePDFMedia.
func mediaFor(glob settings.PdfGlobal, obj *settings.PdfObject) string {
	return settings.ResolvePDFMedia(glob, obj)
}

// DefaultTOCXSL returns the default TOC stylesheet. In pure Go the default
// TOC look is a built-in Go template; this returns a description of it for
// --dump-default-toc-xsl compatibility.
func DefaultTOCXSL() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!-- gowkhtmltopdf default TOC stylesheet.
     Upstream ships an XSLT here; the pure-Go implementation uses an
     equivalent built-in template (see internal/convert/toc.go). -->
<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="html" indent="yes"/>
  <xsl:template match="/">
    <h1>Table of Contents</h1>
    <ul id="toc"/>
  </xsl:template>
</xsl:stylesheet>
`
}

// resolveRelativeLinkURIs rewrites non-absolute, non-fragment OpLinkURI
// values against the page base URL when --resolve-relative-links is on
// (default). Fragments (#id) and scheme URLs are left unchanged.
func resolveRelativeLinkURIs(ops []layout.Op, base string) {
	if base == "" {
		return
	}

	bufU, err := url.Parse(base)
	if err != nil || bufU == nil {
		return
	}

	for idx := range ops {
		if newURI, ok := resolveRelativeLinkURI(ops[idx], bufU); ok {
			ops[idx].URI = newURI
		}
	}
}

// resolveRelativeLinkURI rewrites one OpLinkURI against base, reporting
// whether the op's URI should be replaced. Fragments (#id), scheme URLs,
// mailto links and unparsable references are left unchanged.
func resolveRelativeLinkURI(op layout.Op, base *url.URL) (string, bool) {
	if op.Kind != layout.OpLinkURI || op.URI == "" {
		return "", false
	}

	u := op.URI
	if strings.HasPrefix(u, "#") || strings.Contains(u, "://") || strings.HasPrefix(strings.ToLower(u), "mailto:") {
		return "", false
	}

	ref, err := url.Parse(u)
	if err != nil {
		return "", false
	}

	return base.ResolveReference(ref).String(), true
}
