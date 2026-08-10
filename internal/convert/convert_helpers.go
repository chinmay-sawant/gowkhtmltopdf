package convert

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/layout"
	"gowkhtmltopdf/internal/line"
	"gowkhtmltopdf/internal/pdf"
	"gowkhtmltopdf/internal/settings"
)

// applyCSSPageMargins applies the unnamed @page margin shorthand to the PDF
// viewport after stylesheets have been loaded. CSS page margins describe the
// printable page box, so they must be resolved before body layout rather than
// cascaded as ordinary element padding.
func applyCSSPageMargins(geom hfGeom, sheets []*css.Stylesheet) hfGeom {
	var raw string

	for _, sheet := range sheets {
		if sheet != nil && sheet.Page != nil && strings.TrimSpace(sheet.Page.Margin) != "" {
			raw = sheet.Page.Margin
		}
	}

	if raw == "" {
		return geom
	}

	parts := strings.Fields(raw)
	if len(parts) < 1 || len(parts) > 4 {
		return geom
	}

	vals := make([]float64, len(parts))
	for idx, part := range parts {
		value, unit, ok := css.ParseLength(part)
		if !ok {
			return geom
		}

		pt, ok := css.LengthToPt(value, unit, 12)
		if !ok || pt < 0 {
			return geom
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
		top, right, bottom, left = vals[0], vals[1], vals[2], vals[3]
	}

	geom.marginTop = top
	geom.marginRight = right
	geom.marginBottom = bottom
	geom.marginLeft = left
	geom.recomputeContent()

	return geom
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
// Size.Width/Height (mm) override a named PageSize / Size.PageSize.
// Landscape swaps the pair. Legacy PageWidth/PageHeight fields are gone.
func pageGeometry(glob settings.PdfGlobal) (float64, float64, error) {
	var width, height float64

	if glob.Size.Width > 0 && glob.Size.Height > 0 {
		width, height = glob.Size.Width*mmToPt, glob.Size.Height*mmToPt
	} else {
		name := glob.PageSize
		if name == "" {
			name = glob.Size.PageSize
		}

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

// mediaFor resolves layout CSS media for PDF mode via settings.ResolveMedia.
// Object media lives on Load (CLI --media-type / print-media-type object flags);
// it is projected onto a temporary Web for the shared resolver. PDF default is "print".
func mediaFor(glob settings.PdfGlobal, obj *settings.PdfObject) string {
	var objWeb *settings.Web

	if obj != nil {
		w := settings.Web{ //nolint:exhaustruct // intentional zero-value fields
			PrintMediaType: obj.Load.PrintMediaType,
			MediaType:      obj.Load.MediaType,
		}
		objWeb = &w
	}

	return settings.ResolveMedia(mediaPrint, glob.Web, objWeb)
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

// loadFontRegistry builds the opt-in font registry from --font-path and
// optional --use-system-fonts. Returns nil when nothing was configured.
func loadFontRegistry(glob settings.PdfGlobal, log io.Writer) *pdf.Registry {
	var dirs []string

	dirs = append(dirs, glob.FontPaths...)
	if glob.UseSystemFonts {
		dirs = append(dirs, pdf.DefaultSystemFontDirs()...)
	}

	if len(dirs) == 0 {
		return nil
	}

	reg := pdf.ScanFontDirs(dirs)

	if log != nil && log != io.Discard && !glob.Quiet {
		line.Emit(log, line.Info, "scanned %d font path(s)", len(dirs))
	}

	return reg
}
