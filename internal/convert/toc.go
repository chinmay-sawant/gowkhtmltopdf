package convert

import (
	"fmt"
	stdlibhtml "html"
	"io"
	"strconv"
	"strings"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/layout"
	"gowkhtmltopdf/internal/line"
	"gowkhtmltopdf/internal/outline"
	"gowkhtmltopdf/internal/pdf"
	"gowkhtmltopdf/internal/settings"
)

// effectiveTOC overlays the object-level TOC settings on the global ones.
// Scalars replace when set; booleans OR (a false object flag cannot be
// distinguished from "unset" - documented).
func effectiveTOC(o settings.PdfObject, g settings.PdfGlobal) settings.TableOfContent {
	t := g.TOC
	if o.TOC.FontScale != 0 {
		t.FontScale = o.TOC.FontScale
	}
	if o.TOC.Indentation != "" {
		t.Indentation = o.TOC.Indentation
	}
	if o.TOC.CaptionText != "" {
		t.CaptionText = o.TOC.CaptionText
	}
	if o.TOC.XSLStyleSheet != "" {
		t.XSLStyleSheet = o.TOC.XSLStyleSheet
	}
	t.DottedLines = o.TOC.DottedLines || t.DottedLines
	t.ForwardLinks = o.TOC.ForwardLinks || t.ForwardLinks
	t.BackLinks = o.TOC.BackLinks || t.BackLinks
	return t
}

// lengthToPt converts a CSS length to points. em/rem resolve against the
// base font size; px at 96 dpi; % is rejected (-1). Unparsable lengths return
// -1 so callers can fall back.
func lengthToPt(v string, baseSize float64) float64 {
	val, unit, ok := css.ParseLength(v)
	if !ok {
		return -1
	}
	pt, ok := css.LengthToPt(val, unit, baseSize)
	if !ok {
		return -1
	}
	return pt
}

// genTOCHTML renders the default-look table of contents document: a caption
// <h1> plus one block <div> per outline entry, indented by level, with a
// dotted leader and the page number inline. Each entry div carries its target
// anchor in a data-wk-target attribute so the link pass can locate it after
// layout (the <a> wrappers are inline and produce no boxes).
func genTOCHTML(toc settings.TableOfContent, entries []*outline.Node, pageOf func(*outline.Heading) int, font *pdf.Font, contentW float64) string {
	if toc.CaptionText == "" {
		toc.CaptionText = "Table of Contents"
	}
	scale := toc.FontScale
	if scale <= 0 {
		scale = 0.8
	}
	indentPt := lengthToPt(toc.Indentation, 12)
	if indentPt < 0 {
		indentPt = 12
	}
	const baseSize = 12.0 // the layout engine's default font size

	var b strings.Builder
	b.WriteString("<html><body>")
	b.WriteString("<h1>" + stdlibhtml.EscapeString(toc.CaptionText) + "</h1>")
	for _, n := range entries {
		h := n.Heading
		if h == nil || h.Title == "" {
			continue
		}
		size := baseSize * scale
		pad := indentPt * float64(h.Level-1)
		pageNum := strconv.Itoa(pageOf(h))
		entry := h.Title + "  " + pageNum
		if toc.DottedLines {
			titleW := measureHF(font, h.Title, size)
			pageW := measureHF(font, pageNum, size)
			avail := contentW - pad - titleW - pageW - 2*size
			if dots := int(avail / (0.4 * size)); dots > 0 {
				entry = h.Title + " " + strings.Repeat(".", dots) + " " + pageNum
			}
		}
		start, end := "", ""
		if toc.ForwardLinks {
			start = `<a href="#` + h.Anchor + `">`
			end = "</a>"
		}
		fmt.Fprintf(&b,
			`<div data-wk-target="%s" style="padding-left:%gpt;font-size:%gpt;">%s%s%s</div>`,
			h.Anchor, pad, size, start, entry, end)
	}
	b.WriteString("</body></html>")
	return b.String()
}

// cloneResult deep-copies a layout result so it can be painted more than
// once (Paint splits rects and mutates op positions in place).
func cloneResult(res *layout.Result) *layout.Result {
	c := *res
	c.Ops = append([]layout.Op(nil), res.Ops...)
	return &c
}

// paintOptions converts an object geometry into layout paint options.
func paintOptions(g hfGeom) layout.PaintOptions {
	return layout.PaintOptions{
		PageWidth:    g.pageW,
		PageHeight:   g.pageH,
		MarginTop:    g.marginTop,
		MarginBottom: g.marginBottom,
		MarginLeft:   g.marginLeft,
		MarginRight:  g.marginRight,
	}
}

// paintCount lays the result out into a scratch document and returns its
// page count, leaving res untouched.
func paintCount(res *layout.Result, g hfGeom) (int, error) {
	scratch := pdf.NewDocument()
	if err := layout.Paint(scratch, cloneResult(res), paintOptions(g)); err != nil {
		return 0, err
	}
	return scratch.PageCount(), nil
}

// layoutTOC generates and lays out the TOC document for one TOC object.
// Entry page numbers are offset by tocGuess - the assumed number of pages the
// TOC objects will occupy at the front of the document.
// Headings typically come from BuildTree on a DocPage view, so h.Page is body-global.
func layoutTOC(font *pdf.Font, st *objectState, entries []*outline.Node, tocGuess int, g settings.PdfGlobal, log io.Writer) (*html.Node, *layout.Result, error) {
	toc := st.toc
	if toc.XSLStyleSheet != "" {
		line.Emit(log, line.Warn, "object %d: --xsl-style-sheet is not supported; using the built-in TOC template", st.idx)
	}
	contentW := st.geom.contentW
	pageOf := func(h *outline.Heading) int {
		p := h.DocPage
		if h.DocPage == 0 {
			p = h.Page // view copies put DocPage into Page (including 0)
		}
		return tocGuess + p + 1 + g.PageOffset
	}
	htmlDoc := genTOCHTML(toc, entries, pageOf, font, contentW)
	// TOC HTML is a generated string template, not loaded document bytes.
	root, err := html.Parse(htmlDoc)
	if err != nil {
		return nil, nil, fmt.Errorf("object %d: toc: parse: %w", st.idx+1, err)
	}
	media := st.media
	if media == "" {
		media = "print"
	}
	res, err := layout.Layout(root, layout.Options{
		Width:  contentW,
		Height: st.geom.contentH,
		Font:   font,
		Media:  media,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("object %d: toc: layout: %w", st.idx+1, err)
	}
	return root, res, nil
}

// renderTOCObjects runs the TOC fixed-point: iteration 1 measures each TOC
// object's page count with entry numbers computed as if the TOC took no
// pages; iteration 2 renumbers with the measured total and paints the final
// layouts into doc. It returns the total number of TOC pages. The renumbering
// is done twice at most; if the second measurement changed the count the
// entry numbers may be off by the delta (documented, rare in practice).
func renderTOCObjects(font *pdf.Font, doc *pdf.Document, req *Request, tocs []*objectState, entries []*outline.Node, log io.Writer) (int, error) {
	if len(tocs) == 0 {
		return 0, nil
	}
	g := req.Global

	// Iteration 1: measure with tocGuess = 0.
	guess := 0
	for _, st := range tocs {
		root, res, err := layoutTOC(font, st, entries, guess, g, log)
		if err != nil {
			return 0, err
		}
		st.tocRoot, st.tocRes = root, res
		n, err := paintCount(res, st.geom)
		if err != nil {
			return 0, fmt.Errorf("object %d: toc: paintCount: %w", st.idx+1, err)
		}
		st.tocPages = n
		guess += st.tocPages
	}
	// Iteration 2: renumber with the measured total, measure again, and keep
	// the final layout for painting.
	if guess > 0 {
		for _, st := range tocs {
			root, res, err := layoutTOC(font, st, entries, guess, g, log)
			if err != nil {
				return 0, err
			}
			st.tocRoot, st.tocRes = root, res
			n, err := paintCount(res, st.geom)
			if err != nil {
				return 0, fmt.Errorf("object %d: toc: paintCount: %w", st.idx+1, err)
			}
			st.tocPages = n
		}
	}
	// Paint the final TOC pages into doc, keeping the painted result (its
	// Locations feed the link pass).
	total := 0
	for _, st := range tocs {
		st.start = doc.PageCount()
		painted := cloneResult(st.tocRes)
		if err := layout.Paint(doc, painted, paintOptions(st.geom)); err != nil {
			return 0, fmt.Errorf("object %d: toc: paint: %w", st.idx+1, err)
		}
		st.tocRes = painted
		total += st.tocPages
	}
	return total, nil
}
