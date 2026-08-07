package convert

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/layout"
	"gowkhtmltopdf/internal/line"
	"gowkhtmltopdf/internal/load"
	"gowkhtmltopdf/internal/outline"
	"gowkhtmltopdf/internal/pdf"
	"gowkhtmltopdf/internal/settings"
)

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
func (g hfGeom) pdfY(locPage int, y float64) float64 {
	return g.pageH - g.marginTop - (y - float64(locPage)*g.contentH)
}

// pdfXY converts a y-down element location into PDF (x, y-up) destination.
func (g hfGeom) pdfXY(loc layout.ElementLocation) (float64, float64) {
	return g.marginLeft + loc.X, g.pdfY(loc.Page, loc.Y)
}

// pdfRect converts a y-down element location into a PDF annotation rect
// [x1 y1 x2 y2] with y-up coordinates.
func (g hfGeom) pdfRect(loc layout.ElementLocation) [4]float64 {
	x1 := g.marginLeft + loc.X
	yTop := g.pdfY(loc.Page, loc.Y)
	yBot := g.pageH - g.marginTop - (loc.Y + loc.H - float64(loc.Page)*g.contentH)

	return [4]float64{x1, yBot, x1 + loc.W, yTop}
}

// hfParms is the per-page substitution state for header/footer text. page,
// topage and frompage are 1-based; replaces holds the merged --replace map.
type hfParms struct {
	page, topage, frompage int
	date, clock            string
	title, doctitle        string
	webpage                string
	section, subsection    string
	replaces               map[string]string
}

// knownPlaceholders are the [..] tokens the engine substitutes. Everything
// else passes through literally.
var knownPlaceholders = map[string]bool{
	"page": true, "topage": true, "frompage": true,
	"date": true, "time": true, "title": true, "doctitle": true,
	"webpage": true, "section": true, "subsection": true, "subject": true,
}

// placeholderToken matches one [name] token.
var placeholderToken = regexp.MustCompile(`\[[a-z]+\]`)

// substitute applies the --replace map first, then every known [placeholder]
// token. Unknown placeholders stay literal, matching wkhtmltopdf.
func (p hfParms) substitute(s string) string {
	for k, v := range p.replaces {
		if k == "" {
			continue
		}

		s = strings.ReplaceAll(s, k, v)
	}

	return placeholderToken.ReplaceAllStringFunc(s, func(tok string) string {
		name := tok[1 : len(tok)-1]
		switch name {
		case "page":
			return strconv.Itoa(p.page)
		case "topage":
			return strconv.Itoa(p.topage)
		case "frompage":
			return strconv.Itoa(p.frompage)
		case "date":
			return p.date
		case "time":
			return p.clock
		case "title":
			return p.title
		case "doctitle":
			return p.doctitle
		case "webpage":
			return p.webpage
		case "section":
			return p.section
		case "subsection":
			return p.subsection
		case "subject":
			return "" // no Subject setting in this build; expands to empty
		}

		return tok
	})
}

// knownIn reports whether s contains at least one known placeholder name.
func knownIn(s string) bool {
	for _, m := range placeholderToken.FindAllString(s, -1) {
		name := m[1 : len(m)-1]
		if knownPlaceholders[name] {
			return true
		}
	}

	return false
}

// measureHF returns the width of s in points at size with the embedded font.
func measureHF(font *pdf.Font, s string, size float64) float64 {
	var w float64
	for _, r := range s {
		w += font.AdvanceInPoints(r, size)
	}

	return w
}

// headerHasContent reports whether a header/footer setting draws anything.
func headerHasContent(hf settings.HeaderFooter) bool {
	return hf.Left != "" || hf.Center != "" || hf.Right != "" || hf.Line || hf.HTMLURL != ""
}

// drawTextHF paints the text header (isHeader) or footer into the page's
// margin band: left/center/right sides at the band's vertical middle and the
// separator line at the content edge. The embedded Liberation Sans is used
// for every requested font name (the engine embeds a single font; font-size
// and spacing are honored).
func drawTextHF(page *pdf.Page, hf settings.HeaderFooter, geom hfGeom, parms hfParms, font *pdf.Font, isHeader bool) {
	if !headerHasContent(hf) {
		return
	}

	cur := page.Content()
	cur.UseEmbeddedFont("F0", font)

	size := hf.FontSize
	if size <= 0 {
		size = 12
	}

	spacing := hf.Spacing * mmToPt
	ascent := float64(font.Ascent()) * size / float64(font.UnitsPerEm())
	descent := float64(-font.Descent()) * size / float64(font.UnitsPerEm())

	// Header text sits with its top edge `spacing` below the page top;
	// footer text sits with its bottom edge `spacing` above the page bottom.
	var baseY float64
	if isHeader {
		baseY = page.Height() - spacing - ascent
	} else {
		baseY = spacing + descent
	}

	cur.SetFillColor(0, 0, 0)

	if hf.Line {
		lyVal := geom.marginBottom
		if isHeader {
			lyVal = page.Height() - geom.marginTop
		}

		cur.SetStrokeColor(0, 0, 0)
		cur.SetLineWidth(1)
		cur.MoveTo(geom.marginLeft, lyVal)
		cur.LineTo(page.Width()-geom.marginRight, lyVal)
		cur.Stroke()
	}

	draw := func(text string, posX float64) {
		text = parms.substitute(text)
		if text == "" {
			return
		}

		cur.SetFont("F0", size)
		cur.BeginText()
		cur.TextAt(posX, baseY)
		cur.TextShow(text)
		cur.EndText()
	}
	if hf.Left != "" {
		draw(hf.Left, geom.marginLeft)
	}

	if hf.Center != "" {
		const centerDivisor = 2

		draw(hf.Center, (page.Width()-measureHF(font, parms.substitute(hf.Center), size))/centerDivisor)
	}

	if hf.Right != "" {
		draw(hf.Right, page.Width()-geom.marginRight-measureHF(font, parms.substitute(hf.Right), size))
	}
}

// htmlHFLayout is a cached, laid-out HTML header/footer child document. When
// the source contains placeholders (perPage) the layout is redone per page;
// the placeholder-free case lays out once and reuses the display list.
// Fonts use the same Registry + MergeFontFaces path as the body.
type htmlHFLayout struct {
	raw       string
	skip      bool // value looked like raw markup, not a URL
	perPage   bool
	base      string
	lp        settings.LoadPage
	sheets    []*css.Stylesheet
	res       *layout.Result
	imagesFn  func(src string) ([]byte, error)
	font      *pdf.Font
	registry  *pdf.Registry
	resources ResourceContext
	width     float64
	height    float64
	media     string
}

// loadHTMLHF loads an HTML header/footer as a nested child document: fetch
// under ACL, collect stylesheets, MergeFontFaces, layout at content width.
// Placeholder substitution happens per page at draw time. Output is always
// clipped to the margin band (no independent multi-page HF).
//
// The HF URL is resolved like a top-level page (CWD-relative / absolute /
// http(s)), not as a subresource of the body document. Resolving against
// st.base would break CLI paths such as
// `--header-html testdata/golden/fixture-36-header.html` when the page is
// already under that directory (path doubling).
// loadHTMLHF returns the layout and the font registry after any HF @font-face
// merge. Callers must assign the returned registry onto st (and any outer
// registry handoff); this function does not mutate st.registry.
func loadHTMLHF(ctx context.Context, loader *load.Loader, font *pdf.Font, st *objectState, rawOrURL string, log io.Writer) (*htmlHFLayout, *pdf.Registry, error) {
	if load.IsHTML(rawOrURL) {
		// upstream looksLikeHtmlAndNotAUrl: raw markup is not a URL
		line.Emit(log, line.Warn, "object %d: header/footer html value looks like markup, not a URL; ignoring", st.idx)

		return &htmlHFLayout{skip: true}, st.registry, nil //nolint:exhaustruct // intentional zero-value fields
	}

	if st.resources.Loader != nil {
		loader = st.resources.Loader
	}

	lineP := st.lp
	if st.resources.Loader != nil {
		lineP = st.resources.Load
	}

	res, err := loader.Load(ctx, rawOrURL, lineP)
	if err != nil {
		return nil, nil, fmt.Errorf("header/footer html: %w", err)
	}

	raw := string(res.Body)
	// Detect placeholders on the pristine document. Applying full substitute
	// here with zero page counters used to rewrite [page]/[topage] to "0"
	// and clear perPage, so draw never re-expanded them.
	perPage := knownIn(raw)

	for k, v := range st.repl {
		if k == "" {
			continue
		}

		raw = strings.ReplaceAll(raw, k, v)
	}

	measureRaw := raw
	if perPage {
		measureRaw = (hfParms{page: 1, topage: 1}).substitute(raw) //nolint:exhaustruct // intentional zero-value fields
	}

	root, err := html.Parse(measureRaw)
	if err != nil {
		return nil, nil, fmt.Errorf("header/footer html: parse: %w", err)
	}

	media := st.media
	if media == "" {
		media = mediaPrint
	}

	resources := NewResourceContext(loader, res.Base, lineP)
	sheets := resources.CollectSheets(ctx, root, SheetOptions{
		ViewportW:   st.geom.contentW,
		ViewportH:   st.geom.contentH,
		MediaType:   media,
		ObjectIndex: st.idx + 1,
	}, log)

	reg := resources.MergeFontFaces(ctx, st.registry, sheets, st.idx+1, log)
	if reg == nil {
		reg = st.registry
	}

	imagesFn := func(src string) ([]byte, error) {
		if !st.imagesEnabled {
			return nil, errors.New("images disabled")
		}

		r, err := resources.Fetch(ctx, src)
		if err != nil {
			return nil, err
		}

		return r.Body, nil
	}
	lst := &htmlHFLayout{ //nolint:exhaustruct // intentional zero-value fields
		raw:       raw,
		perPage:   perPage,
		base:      res.Base,
		lp:        lineP,
		sheets:    sheets,
		imagesFn:  imagesFn,
		font:      font,
		registry:  reg,
		resources: resources,
		width:     st.geom.contentW,
		height:    st.geom.contentH,
		media:     media,
	}
	// Lay out once regardless: placeholder-free docs reuse this display list
	// for every page; placeholder docs use it only for the natural height
	// (auto margins) and re-layout per page at draw time.
	lst.res, err = layout.LayoutContext(ctx, root, layout.Options{ //nolint:exhaustruct // intentional zero-value fields
		Width: lst.width, Height: lst.height, Font: font, Registry: lst.registry,
		Sheets: sheets, Media: media, Images: imagesFn,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("header/footer html: layout: %w", err)
	}

	return lst, reg, nil
}

// hfLinkContext carries body id destinations and link flags into HTML HF
// drawing so fragment hrefs become GoTo annotations on the final page set.
type hfLinkContext struct {
	idIndex  map[string]bodyIDDest
	tocTotal int
	plan     *pagePlan
	srcPage  int // final (post-copy) page index being drawn
	useLocal bool
	useExt   bool
	resolve  bool
}

// drawHTMLHF paints a cached HTML header/footer onto page, clipped to the
// margin band. The HF document's canvas origin maps to `spacing` points from
// the page top (header) or from the page bottom (footer); x is aligned with
// the content area's left margin. Images are embedded per page.
//
// Body pages use layout.Paint (pagination + sticky); HF is a single-band
// clamp that shares the same canvas→PDF op dispatch via paintLayoutOps.
func drawHTMLHF(ctx context.Context, page *pdf.Page, hfL *htmlHFLayout, hf settings.HeaderFooter, geom hfGeom, parms hfParms, isHeader bool, links hfLinkContext) error {
	if hfL == nil || hfL.skip {
		return nil
	}

	res := hfL.res

	media := hfL.media
	if media == "" {
		media = mediaPrint
	}

	if hfL.perPage {
		raw := parms.substitute(hfL.raw)

		root, err := html.Parse(raw)
		if err != nil {
			return err
		}

		res, err = layout.LayoutContext(ctx, root, layout.Options{ //nolint:exhaustruct // intentional zero-value fields
			Width: hfL.width, Height: hfL.height, Font: hfL.font, Registry: hfL.registry,
			Sheets: hfL.sheets, Media: media, Images: hfL.imagesFn,
		})
		if err != nil {
			return err
		}
	}

	if links.resolve {
		resolveRelativeLinkURIs(res.Ops, hfL.base)
	}

	spacing := hf.Spacing * mmToPt
	pageH := page.Height()
	// Clip to the reserved margin band. Footer band is the bottom strip
	// [0, marginBottom]; header band is the top strip
	// [pageH-marginTop, pageH]. (Using marginBottom as the footer's bandTop
	// incorrectly clipped footer ink out of the page.)
	bandTop := 0.0
	bandH := geom.marginBottom

	if isHeader {
		bandTop = pageH - geom.marginTop
		bandH = geom.marginTop
	}
	// Nested HF is a single-page clamp: taller content is clipped to the band
	// (height was reserved via hfHeightFor / effectiveMargins).
	// Canvas y=0 (top of the HF document) maps to:
	//   header: spacing below the page top; footer: spacing + doc height
	yTop := pageH - spacing
	if !isHeader {
		yTop = spacing + res.Height
	}

	c := page.Content()
	c.Save()
	c.Rect(0, bandTop, page.Width(), bandH)
	c.Clip()
	paintLayoutOps(ctx, page, c, res.Ops, geom.marginLeft, yTop, links)
	c.Restore()

	return nil
}

// paintLayoutOps paints visual ops via layout.PaintBand (shared fake-bold /
// alpha / stroke policy with body paint), then wires link annotations that
// need document context (fragment GoTo / external URI).
func paintLayoutOps(ctx context.Context, page *pdf.Page, c *pdf.Content, ops []layout.Op, originX, yTop float64, links hfLinkContext) {
	_ = layout.PaintBandContext(ctx, page, c, ops, layout.BandOptions{ //nolint:exhaustruct // intentional zero-value fields
		OriginX: originX,
		OriginY: yTop,
	})

	for i := range ops {
		oper := &ops[i]
		if oper.Kind != layout.OpLinkURI {
			continue
		}

		posX := originX + oper.X
		y1Val := yTop - (oper.Y + oper.H)
		y2Val := yTop - oper.Y

		if y2Val < y1Val {
			y1Val, y2Val = y2Val, y1Val
		}

		// minHFLinkExtent is a fallback hit-box side when layout reports zero width/height.
		const minHFLinkExtent = 10

		rect := [4]float64{posX, y1Val, posX + oper.W, y2Val}
		if oper.W <= 0 {
			rect[2] = posX + minHFLinkExtent
		}

		if oper.H <= 0 {
			rect[1] = yTop - oper.Y - minHFLinkExtent
		}

		uri := oper.URI
		if uri == "" {
			continue
		}

		if len(uri) > 0 && uri[0] == '#' {
			frag := uri[1:]
			if !links.useLocal || frag == "" || links.idIndex == nil {
				continue
			}

			dest, ok := links.idIndex[frag]
			if !ok {
				continue
			}

			logical := logicalDestPage(dest, links.tocTotal)
			destPage := logical

			if links.plan != nil {
				destPage = links.plan.Remap(logical, links.srcPage)
			}

			dx, dy := dest.st.geom.pdfXY(dest.loc)
			page.AddLinkDest(rect, destPage, dx, dy)

			continue
		}

		if !links.useExt {
			continue
		}

		page.AddLinkURI(rect, uri)
	}
}

// hfHeightFor returns the drawn height of a text or HTML header/footer in
// points (0 when nothing is drawn) and the font registry after any HTML HF
// @font-face merge. Text uses the font's line box; HTML uses the laid-out
// document height.
func hfHeightFor(ctx context.Context, loader *load.Loader, font *pdf.Font, st *objectState, hf settings.HeaderFooter, isHeader bool, log io.Writer) (float64, *pdf.Registry, error) {
	if !headerHasContent(hf) {
		return 0, st.registry, nil
	}

	if hf.HTMLURL != "" {
		lst, reg, err := loadHTMLHF(ctx, loader, font, st, hf.HTMLURL, log)
		if err != nil {
			return 0, nil, err
		}

		if reg != nil {
			st.registry = reg // next HF band (footer) sees header faces
		}

		if isHeader {
			st.headerHTML = lst
		} else {
			st.footerHTML = lst
		}

		if lst.skip || lst.res == nil {
			return 0, st.registry, nil
		}

		return lst.res.Height, st.registry, nil
	}

	size := hf.FontSize
	if size <= 0 {
		size = 12
	}

	h := (float64(font.Ascent()) - float64(font.Descent())) / float64(font.UnitsPerEm()) * size

	return h, st.registry, nil
}

// effectiveMargins resolves the object's top/bottom margins in points,
// replacing auto (-1) margins with the measured header/footer height plus the
// header/footer spacing, so the body layout reserves the bands. HTML
// headers/footers are loaded and laid out here (once, cached on st).
//
// The returned registry is st.registry after any HF MergeFontFaces extensions;
// callers assign it for the body layout handshake (do not rely on mutation alone).
func effectiveMargins(ctx context.Context, loader *load.Loader, font *pdf.Font, g settings.PdfGlobal, st *objectState, log io.Writer) (*pdf.Registry, error) {
	if g.Margin.Top < 0 {
		h, _, err := hfHeightFor(ctx, loader, font, st, st.header, true, log)
		if err != nil {
			return nil, err
		}

		st.geom.marginTop = h + st.header.Spacing*mmToPt
	}

	if g.Margin.Bottom < 0 {
		h, _, err := hfHeightFor(ctx, loader, font, st, st.footer, false, log)
		if err != nil {
			return nil, err
		}

		st.geom.marginBottom = h + st.footer.Spacing*mmToPt
	}

	st.geom.recomputeContent()

	return st.registry, nil
}

// drawHeadersFooters is the final pass that paints the effective text/HTML
// header and footer of every page once the whole document exists (so [topage]
// and the page indices are final). Cover pages are skipped. Errors loading an
// HTML header/footer here only warn - the body content is already painted.
func drawHeadersFooters(ctx context.Context, loader *load.Loader, font *pdf.Font, doc *pdf.Document, req *Request, plan *pagePlan, headings []*outline.Heading, log io.Writer) {
	total := doc.PageCount()
	now := time.Now()
	date := now.Format("2006-01-02")
	clock := now.Format("15:04:05")
	idIndex := buildBodyIDIndex(planBodyStates(plan))

	tocTotal := 0
	if plan != nil {
		tocTotal = plan.tocTotal
	}

	for pVal := range total {
		own, ok := plan.OwnerOf(pVal)
		if !ok || own.st == nil {
			continue
		}

		if own.st.obj.IsCover {
			continue
		}

		parms := hfParms{ //nolint:exhaustruct // intentional zero-value fields
			page:     pVal + 1 + req.Global.PageOffset,
			topage:   total,
			frompage: own.local + 1,
			date:     date,
			clock:    clock,
			title:    req.Global.Title,
			doctitle: own.st.doctitle,
			webpage:  own.st.obj.Page,
			replaces: own.st.repl,
		}
		if !own.st.isTOC {
			parms.section, parms.subsection = outline.SectionOfBy(headings, pVal-tocTotal, outline.DocumentPage)
		}

		page := doc.PageAt(pVal)
		if page == nil {
			continue
		}

		links := hfLinkContext{
			idIndex:  idIndex,
			tocTotal: tocTotal,
			plan:     plan,
			srcPage:  pVal,
			useLocal: own.st.obj.LocalLinks,
			useExt:   own.st.obj.ExternalLinks,
			resolve:  req.Global.ResolveRelativeLinks,
		}
		draw := func(hfVal settings.HeaderFooter, isHeader bool) {
			if !headerHasContent(hfVal) {
				return
			}

			if hfVal.HTMLURL != "" {
				lst := own.st.headerHTML
				if !isHeader {
					lst = own.st.footerHTML
				}

				if lst == nil {
					var err error

					var reg *pdf.Registry

					lst, reg, err = loadHTMLHF(ctx, loader, font, own.st, hfVal.HTMLURL, log)
					if err != nil {
						line.Emit(log, line.Warn, "object %d: header/footer html: %v", own.st.idx, err)

						return
					}

					if reg != nil {
						own.st.registry = reg
					}

					if isHeader {
						own.st.headerHTML = lst
					} else {
						own.st.footerHTML = lst
					}
				}

				if err := drawHTMLHF(ctx, page, lst, hfVal, own.st.geom, parms, isHeader, links); err != nil {
					line.Emit(log, line.Warn, "object %d: header/footer html draw: %v", own.st.idx, err)
				}

				return
			}

			drawTextHF(page, hfVal, own.st.geom, parms, font, isHeader)
		}
		draw(own.st.header, true)
		draw(own.st.footer, false)
	}
}

// planBodyStates returns unique body objectStates from a page plan (for id index).
func planBodyStates(plan *pagePlan) []*objectState {
	if plan == nil {
		return nil
	}

	seen := map[*objectState]bool{}

	var out []*objectState

	for _, own := range plan.owners {
		if own.st == nil || own.st.isTOC || seen[own.st] {
			continue
		}

		seen[own.st] = true

		out = append(out, own.st)
	}

	return out
}
