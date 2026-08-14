package convert

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/layout"
	"gowkhtmltopdf/internal/line"
	"gowkhtmltopdf/internal/load"
	"gowkhtmltopdf/internal/outline"
	"gowkhtmltopdf/internal/pdf"
	"gowkhtmltopdf/internal/settings"
)

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

func isKnownPlaceholder(token string) bool {
	switch token {
	case "page", "topage", "frompage", "date", "time", "title",
		"doctitle", "webpage", htmlSectionName, "subsection", "subject":
		return true
	default:
		return false
	}
}

// placeholderToken matches one [name] token.
var placeholderToken = regexp.MustCompile(`\[[a-z]+\]`)

// substitute applies the --replace map first, then every known [placeholder]
// token. Unknown placeholders stay literal, matching wkhtmltopdf.
func (p hfParms) substitute(src string) string { //nolint:cyclop // per-token switch over known placeholders
	for k, v := range p.replaces {
		if k == "" {
			continue
		}

		src = strings.ReplaceAll(src, k, v)
	}

	return placeholderToken.ReplaceAllStringFunc(src, func(tok string) string {
		name := tok[1 : len(tok)-1]

		switch name {
		case "page":
			return strconv.Itoa(p.page)
		case "frompage":
			return strconv.Itoa(p.frompage)
		case "topage":
			return strconv.Itoa(p.topage)
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
		if isKnownPlaceholder(name) {
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

//nolint:mnd // 400 is default normal font weight
func resolveHFFont(name string, reg *pdf.Registry, fallback *pdf.Font) *pdf.Font {
	if strings.TrimSpace(name) == "" || reg == nil {
		return fallback
	}

	if f := reg.Lookup([]string{strings.TrimSpace(name)}, 400, false); f != nil {
		return f
	}

	return fallback
}

// drawTextHF paints the text header (isHeader) or footer into the page's
// margin band: left/center/right sides at the band's vertical middle and the
// separator line at the content edge. HeaderFooter.FontName is resolved
// through the font registry with fallback to the default base font.
func drawTextHF(page *pdf.Page, hfVal settings.HeaderFooter, geom hfGeom, parms hfParms, font *pdf.Font, reg *pdf.Registry, isHeader bool) { //nolint:cyclop,funlen,lll // left/center/right bands with line flag
	if !headerHasContent(hfVal) {
		return
	}

	font = resolveHFFont(hfVal.FontName, reg, font)

	cur := page.Content()
	cur.UseEmbeddedFont("F0", font)

	size := hfVal.FontSize
	if size <= 0 {
		size = 12
	}

	spacing := hfVal.Spacing * mmToPt
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

	left := parms.substitute(hfVal.Left)
	center := parms.substitute(hfVal.Center)
	right := parms.substitute(hfVal.Right)

	if !hfVal.Line && left == "" && center == "" && right == "" {
		return
	}

	isUA1 := page.Doc() != nil && page.Doc().Policy().IsPDFUA1()
	if isUA1 {
		cur.BeginArtifact("Pagination")
	}

	cur.SetFillColor(0, 0, 0)

	if hfVal.Line {
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
		if text == "" {
			return
		}

		cur.SetFont("F0", size)
		cur.BeginText()
		cur.TextAt(posX, baseY)
		cur.TextShow(text)
		cur.EndText()
	}
	if left != "" {
		draw(left, geom.marginLeft)
	}

	if center != "" {
		const centerDivisor = 2

		draw(center, (page.Width()-measureHF(font, center, size))/centerDivisor)
	}

	if right != "" {
		draw(right, page.Width()-geom.marginRight-measureHF(font, right, size))
	}

	if isUA1 {
		cur.EndArtifact()
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
func loadHTMLHF(ctx context.Context, loader *load.Loader, font *pdf.Font, state *objectState, rawOrURL string, log io.Writer) (*htmlHFLayout, *pdf.Registry, error) { //nolint:cyclop,funlen,lll // per-kind resolution/merge branches
	if load.IsHTML(rawOrURL) {
		// upstream looksLikeHtmlAndNotAUrl: raw markup is not a URL
		line.Emit(log, line.Warn, "object %d: header/footer html value looks like markup, not a URL; ignoring", state.idx)

		return &htmlHFLayout{skip: true}, state.registry, nil //nolint:exhaustruct // intentional zero-value fields
	}

	if state.resources.Loader != nil {
		loader = state.resources.Loader
	}

	lineP := state.lp
	if state.resources.Loader != nil {
		lineP = state.resources.Load
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

	for k, v := range state.repl {
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

	media := state.media
	if media == "" {
		media = mediaPrint
	}

	resources := NewResourceContext(loader, res.Base, lineP)
	sheets := resources.CollectSheets(ctx, root, SheetOptions{
		ViewportW:   state.geom.contentW,
		ViewportH:   state.geom.contentH,
		MediaType:   media,
		ObjectIndex: state.idx + 1,
	}, log)

	reg := resources.MergeFontFaces(ctx, state.registry, sheets, state.idx+1, log)
	if reg == nil {
		reg = state.registry
	}

	imagesFn := func(src string) ([]byte, error) {
		if !state.imagesEnabled {
			return nil, errImagesDisabled
		}

		r, err := resources.Fetch(ctx, src)
		if err != nil {
			return nil, fmt.Errorf("fetch header/footer resource: %w", err)
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
		width:     state.geom.contentW,
		height:    state.geom.contentH,
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

// hfDrawWarning records a recoverable header/footer failure with enough
// locality to explain which band was omitted. Header/footer failures do not
// invalidate an already-painted body document, so the compatibility adapter
// below reports every warning instead of dropping the error or aborting the
// conversion after body output exists.
type hfDrawWarning struct {
	object int
	page   int
	band   string
	err    error
}

type hfDrawResult struct {
	warnings []hfDrawWarning
}

// Err returns the aggregate failure for strict conversion callers. The
// compatibility adapter may still emit warnings, but the primary PDF engine
// must not report success when required header/footer content was omitted.
// Err returns the aggregate failure for strict conversion callers. The
// compatibility adapter may still emit warnings, but the primary PDF engine
// must not report success when required header/footer content was omitted.
func (r *hfDrawResult) Err() error {
	if len(r.warnings) == 0 {
		return nil
	}

	errs := make([]error, 0, len(r.warnings))
	for _, warning := range r.warnings {
		errs = append(errs, fmt.Errorf("object %d page %d %s: %w",
			warning.object, warning.page+1, warning.band, warning.err))
	}

	return errors.Join(errs...)
}

func (r *hfDrawResult) warn(object, page int, band string, err error) {
	if err == nil {
		return
	}

	r.warnings = append(r.warnings, hfDrawWarning{
		object: object,
		page:   page,
		band:   band,
		err:    err,
	})
}

//nolint:unused // warning emitter helper for compatibility adapter
func (r *hfDrawResult) emitWarnings(log io.Writer) {
	for _, warning := range r.warnings {
		line.Emit(log, line.Warn, "object %d page %d: %s header/footer: %v",
			warning.object, warning.page+1, warning.band, warning.err)
	}
}

// drawHTMLHF paints a cached HTML header/footer onto page, clipped to the
// margin band. The HF document's canvas origin maps to `spacing` points from
// the page top (header) or from the page bottom (footer); x is aligned with
// the content area's left margin. Images are embedded per page.
//
// Body pages use layout.Paint (pagination + sticky); HF is a single-band
// clamp that shares the same canvas→PDF op dispatch via paintLayoutOps.
func drawHTMLHF(ctx context.Context, page *pdf.Page, hfL *htmlHFLayout, hfVal settings.HeaderFooter, geom hfGeom, parms hfParms, isHeader bool, links hfLinkContext) error { //nolint:cyclop,funlen,lll // header/footer draw signature
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
			return fmt.Errorf("header/footer html: parse: %w", err)
		}

		res, err = layout.LayoutContext(ctx, root, layout.Options{ //nolint:exhaustruct // intentional zero-value fields
			Width: hfL.width, Height: hfL.height, Font: hfL.font, Registry: hfL.registry,
			Sheets: hfL.sheets, Media: media, Images: hfL.imagesFn,
		})
		if err != nil {
			return fmt.Errorf("header/footer html: layout: %w", err)
		}
	}

	if links.resolve {
		resolveRelativeLinkURIs(res.Ops, hfL.base)
	}

	spacing := hfVal.Spacing * mmToPt
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

	pageContent := page.Content()
	pageContent.Save()

	isUA1 := page.Doc() != nil && page.Doc().Policy().IsPDFUA1()
	if isUA1 {
		pageContent.BeginArtifact("Pagination")
	}

	pageContent.Rect(0, bandTop, page.Width(), bandH)
	pageContent.Clip()

	err := paintLayoutOps(ctx, page, pageContent, res.Ops, geom.marginLeft, yTop, links)

	if isUA1 {
		pageContent.EndArtifact()
	}

	pageContent.Restore()

	return err
}

// paintLayoutOps paints visual ops via layout.PaintBand (shared fake-bold /
// alpha / stroke policy with body paint), then wires link annotations that
// need document context (fragment GoTo / external URI).
func paintLayoutOps(ctx context.Context, page *pdf.Page, c *pdf.Content, ops []layout.Op, originX, yTop float64, links hfLinkContext) error { //nolint:cyclop,funlen,lll // band paint plus per-op link wiring
	if err := layout.PaintBandContext(ctx, page, c, ops, layout.BandOptions{ //nolint:exhaustruct,lll // intentional zero-value fields
		OriginX: originX,
		OriginY: yTop,
	}); err != nil {
		return fmt.Errorf("convert: paint HF layout band: %w", err)
	}

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
			annotRef := page.AddLinkDest(rect, destPage, dx, dy)
			attachLinkStructElem(page.Doc(), page, oper.StructElem, annotRef)

			continue
		}

		if !links.useExt {
			continue
		}

		annotRef := page.AddLinkURI(rect, uri)
		attachLinkStructElem(page.Doc(), page, oper.StructElem, annotRef)
	}

	return nil
}

// hfHeightFor returns the drawn height of a text or HTML header/footer in
// points (0 when nothing is drawn). Text uses the font's line box; HTML uses
// the laid-out document height. Any HF @font-face merge mutates st.registry
// (the returned registry from loadHTMLHF is assigned there, not returned).
func hfHeightFor(ctx context.Context, loader *load.Loader, font *pdf.Font, state *objectState, hfVal settings.HeaderFooter, isHeader bool, log io.Writer) (float64, error) { //nolint:lll // hf band signature
	if !headerHasContent(hfVal) {
		return 0, nil
	}

	if hfVal.HTMLURL != "" { //nolint:nestif // load/cache/height branches for one band
		lst, reg, err := loadHTMLHF(ctx, loader, font, state, hfVal.HTMLURL, log)
		if err != nil {
			return 0, err
		}

		if reg != nil {
			state.registry = reg // next HF band (footer) sees header faces
		}

		if isHeader {
			state.headerHTML = lst
		} else {
			state.footerHTML = lst
		}

		if lst.skip || lst.res == nil {
			return 0, nil
		}

		return lst.res.Height, nil
	}

	size := hfVal.FontSize
	if size <= 0 {
		size = 12
	}

	if state != nil {
		font = resolveHFFont(hfVal.FontName, state.registry, font)
	}

	h := (float64(font.Ascent()) - float64(font.Descent())) / float64(font.UnitsPerEm()) * size

	return h, nil
}

// effectiveMargins resolves the object's top/bottom margins in points,
// replacing auto (-1) margins with the measured header/footer height plus the
// header/footer spacing, so the body layout reserves the bands. HTML
// headers/footers are loaded and laid out here (once, cached on st).
//
// The returned registry is st.registry after any HF MergeFontFaces extensions;
// callers assign it for the body layout handshake (do not rely on mutation alone).
func effectiveMargins(ctx context.Context, loader *load.Loader, font *pdf.Font, glob settings.PdfGlobal, state *objectState, log io.Writer) (*pdf.Registry, error) { //nolint:lll // margin resolution signature
	if glob.Margin.Top < 0 {
		h, err := hfHeightFor(ctx, loader, font, state, state.header, true, log)
		if err != nil {
			return nil, err
		}

		state.geom.marginTop = h + state.header.Spacing*mmToPt
	}

	if glob.Margin.Bottom < 0 {
		h, err := hfHeightFor(ctx, loader, font, state, state.footer, false, log)
		if err != nil {
			return nil, err
		}

		state.geom.marginBottom = h + state.footer.Spacing*mmToPt
	}

	state.geom.recomputeContent()

	return state.registry, nil
}

// drawHeadersFooters is the compatibility adapter for the existing caller.
// The result-producing implementation below keeps the failure policy
// explicit: body output remains usable, every recoverable HF error is
// collected, and the adapter emits one warning per failed band.
//
//nolint:lll,unused // compatibility adapter for existing caller
func drawHeadersFooters(ctx context.Context, loader *load.Loader, font *pdf.Font, doc *pdf.Document, req *Request, plan *pagePlan, headings []*outline.Heading, log io.Writer) {
	res := drawHeadersFootersResult(ctx, loader, font, doc, req, plan, headings, log)
	res.emitWarnings(log)
}

// drawHeadersFootersResult is the final pass that paints the effective
// text/HTML header and footer of every page once the whole document exists
// (so [topage] and the page indices are final). Cover pages are skipped.
// Returning a result keeps failure handling testable and gives a future
// caller a precise integration point for a strict policy without changing
// the current convert.Run signature.
func drawHeadersFootersResult(ctx context.Context, loader *load.Loader, font *pdf.Font, doc *pdf.Document, req *Request, plan *pagePlan, headings []*outline.Heading, log io.Writer) hfDrawResult { //nolint:gocognit,cyclop,funlen,lll // per-page draw dispatch with lazy HF load
	var result hfDrawResult

	total := doc.PageCount()
	now := req.now()
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

			band := "footer"
			if isHeader {
				band = "header"
			}

			if hfVal.HTMLURL != "" { //nolint:nestif // lazy load, cache and draw for one band
				lst := own.st.headerHTML
				if !isHeader {
					lst = own.st.footerHTML
				}

				if lst == nil {
					var err error

					var reg *pdf.Registry

					lst, reg, err = loadHTMLHF(ctx, loader, font, own.st, hfVal.HTMLURL, log)
					if err != nil {
						result.warn(own.st.idx, pVal, band, fmt.Errorf("html load: %w", err))

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
					result.warn(own.st.idx, pVal, band, fmt.Errorf("html draw: %w", err))
				}

				return
			}

			drawTextHF(page, hfVal, own.st.geom, parms, font, own.st.registry, isHeader)
		}
		draw(own.st.header, true)
		draw(own.st.footer, false)
	}

	return result
}

// planBodyStates returns unique body objectStates from a page plan (for id index).
func planBodyStates(plan *pagePlan) []*objectState {
	if plan == nil {
		return nil
	}

	seen := map[*objectState]bool{}

	out := make([]*objectState, 0, len(plan.owners))

	for _, own := range plan.owners {
		if own.st == nil || own.st.isTOC || seen[own.st] {
			continue
		}

		seen[own.st] = true

		out = append(out, own.st)
	}

	return out
}
