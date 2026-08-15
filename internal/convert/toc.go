package convert

import (
	"context"
	"fmt"
	stdlibhtml "html"
	"io"
	"strconv"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/line"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/outline"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// effectiveTOC overlays the object-level TOC settings on the global ones.
// Scalars replace when set; booleans OR (a false object flag cannot be
// distinguished from "unset" - documented).
func effectiveTOC(obj settings.PdfObject, g settings.PdfGlobal) settings.TableOfContent {
	tbl := g.TOC
	if obj.TOC.FontScale != 0 {
		tbl.FontScale = obj.TOC.FontScale
	}

	if obj.TOC.Indentation != "" {
		tbl.Indentation = obj.TOC.Indentation
	}

	if obj.TOC.CaptionText != "" {
		tbl.CaptionText = obj.TOC.CaptionText
	}

	if obj.TOC.XSLStyleSheet != "" {
		tbl.XSLStyleSheet = obj.TOC.XSLStyleSheet
	}

	tbl.DottedLines = obj.TOC.DottedLines || tbl.DottedLines
	tbl.ForwardLinks = obj.TOC.ForwardLinks || tbl.ForwardLinks
	tbl.BackLinks = obj.TOC.BackLinks || tbl.BackLinks

	return tbl
}

// lengthToPt converts a CSS length to points. em/rem resolve against the
// base font size; px at 96 dpi; % is rejected (-1). Unparsable lengths return
// -1 so callers can fall back.
func lengthToPt(v string, baseSize float64) float64 {
	found, unit, okVal := css.ParseLength(v)
	if !okVal {
		return -1
	}

	pt, okVal := css.LengthToPt(found, unit, baseSize)
	if !okVal {
		return -1
	}

	return pt
}

// genTOCHTML renders the default-look table of contents document: a caption
// <h1> plus one block <div> per outline entry, indented by level, with a
// dotted leader and the page number inline. Each entry div carries its target
// anchor in a data-wk-target attribute so the link pass can locate it after
// layout (the <a> wrappers are inline and produce no boxes).
func genTOCHTML(toc settings.TableOfContent, entries []*outline.Node, pageOf func(*outline.Heading) int, font *pdf.Font, contentW float64) string { //nolint:funlen,lll // template rendering with leader dots and anchors
	if toc.CaptionText == "" {
		toc.CaptionText = "Table of Contents"
	}

	const (
		// defaultTOCFontSize is the layout engine's default body size used for TOC entries.
		defaultTOCFontSize = 12.0
		// tocDotGapFactor leaves a small gap on each side of the dotted leader (in ems).
		tocDotGapFactor = 2
		// tocDotAdvanceFactor is the approximate advance of "." relative to the font size.
		tocDotAdvanceFactor = 0.4
		// defaultTOCFontScale is used when TOC.FontScale is unset/non-positive.
		defaultTOCFontScale = 0.8
	)

	scale := toc.FontScale
	if scale <= 0 {
		scale = defaultTOCFontScale
	}

	indentPt := lengthToPt(toc.Indentation, defaultTOCFontSize)
	if indentPt < 0 {
		indentPt = defaultTOCFontSize
	}

	baseSize := defaultTOCFontSize

	var buf strings.Builder

	buf.WriteString("<html><body>")
	buf.WriteString("<h1>")
	buf.WriteString(stdlibhtml.EscapeString(toc.CaptionText))
	buf.WriteString("</h1>")

	for _, n := range entries {
		hVal := n.Heading
		if hVal == nil || hVal.Title == "" {
			continue
		}

		size := baseSize * scale
		pad := indentPt * float64(hVal.Level-1)
		pageNum := strconv.Itoa(pageOf(hVal))
		entry := hVal.Title + "  " + pageNum

		if toc.DottedLines {
			titleW := measureHF(font, hVal.Title, size)
			pageW := measureHF(font, pageNum, size)

			avail := contentW - pad - titleW - pageW - float64(tocDotGapFactor)*size
			if dots := int(avail / (tocDotAdvanceFactor * size)); dots > 0 {
				entry = hVal.Title + " " + strings.Repeat(".", dots) + " " + pageNum
			}
		}

		start, end := "", ""
		if toc.ForwardLinks {
			start = `<a href="#` + hVal.Anchor + `">`
			end = "</a>"
		}

		fmt.Fprintf(&buf,
			`<div data-wk-target="%s" style="padding-left:%gpt;font-size:%gpt;">%s%s%s</div>`,
			hVal.Anchor, pad, size, start, entry, end)
	}

	buf.WriteString("</body></html>")

	return buf.String()
}

// cloneResult deep-copies a layout result so it can be painted more than
// once (Paint splits rects and mutates op positions in place).
func cloneResult(res *layout.Result) *layout.Result {
	return layout.CloneResult(res)
}

// paintOptions converts an object geometry into layout paint options.
func paintOptions(geom hfGeom) layout.PaintOptions {
	return layout.PaintOptions{
		PageWidth:    geom.pageW,
		PageHeight:   geom.pageH,
		MarginTop:    geom.marginTop,
		MarginBottom: geom.marginBottom,
		MarginLeft:   geom.marginLeft,
		MarginRight:  geom.marginRight,
	}
}

// paintCount lays the result out into a scratch document and returns its
// page count, leaving res untouched.
func paintCount(ctx context.Context, policy pdf.WriterPolicy, res *layout.Result, geom hfGeom) (int, error) {
	scratchPolicy := pdf.WriterPolicy{Version: policy.Version} //nolint:exhaustruct // estimation scratch policy

	scratch, err := pdf.NewDocumentWithPolicy(scratchPolicy)
	if err != nil {
		return 0, fmt.Errorf("toc: paintCount: %w", err)
	}

	if err := layout.PaintContext(ctx, scratch, cloneResult(res), paintOptions(geom)); err != nil {
		return 0, fmt.Errorf("toc: paintCount: %w", err)
	}

	return scratch.PageCount(), nil
}

// layoutTOC generates and lays out the TOC document for one TOC object.
// Entry page numbers are offset by tocGuess - the assumed number of pages the
// TOC objects will occupy at the front of the document.
// Headings typically come from BuildTree on a DocPage view, so h.Page is body-global.
func layoutTOC(ctx context.Context, font *pdf.Font, state *objectState, entries []*outline.Node, tocGuess int, glob settings.PdfGlobal, log io.Writer) (*html.Node, *layout.Result, error) { //nolint:lll // toc pipeline signature
	toc := state.toc
	if toc.XSLStyleSheet != "" {
		line.Emit(log, line.Warn, "object %d: --xsl-style-sheet is not supported; using the built-in TOC template", state.idx)
	}

	contentW := state.geom.contentW
	pageOf := func(h *outline.Heading) int {
		p := h.DocPage
		if h.DocPage == 0 {
			p = h.Page // view copies put DocPage into Page (including 0)
		}

		return tocGuess + p + 1 + glob.PageOffset
	}
	htmlDoc := genTOCHTML(toc, entries, pageOf, font, contentW)
	// TOC HTML is a generated string template, not loaded document bytes.
	root, err := html.Parse(htmlDoc)
	if err != nil {
		return nil, nil, fmt.Errorf("object %d: toc: parse: %w", state.idx+1, err)
	}

	media := state.media
	if media == "" {
		media = mediaPrint
	}

	res, err := layout.LayoutContext(ctx, root, layout.Options{ //nolint:exhaustruct // intentional zero-value fields
		Width:  contentW,
		Height: state.geom.contentH,
		Font:   font,
		Media:  media,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("object %d: toc: layout: %w", state.idx+1, err)
	}

	return root, res, nil
}

// renderTOCObjects runs the TOC fixed-point: iteration 1 measures each TOC
// object's page count with entry numbers computed as if the TOC took no
// pages; iteration 2 renumbers with the measured total and paints the final
// layouts into doc. It returns the total number of TOC pages. The renumbering
// is done twice at most; if the second measurement changed the count the
// entry numbers may be off by the delta (documented, rare in practice).
func renderTOCObjects(ctx context.Context, font *pdf.Font, doc *pdf.Document, req *Request, tocs []*objectState, entries []*outline.Node, log io.Writer) (int, error) { //nolint:cyclop,lll // fixed-point iterations over TOC objects
	if len(tocs) == 0 {
		return 0, nil
	}

	glob := req.Global

	// Iteration 1: measure with tocGuess = 0.
	guess := 0
	for _, state := range tocs {
		root, res, err := layoutTOC(ctx, font, state, entries, guess, glob, log)
		if err != nil {
			return 0, err
		}

		state.tocRoot, state.tocRes = root, res

		n, err := paintCount(ctx, doc.Policy(), res, state.geom)
		if err != nil {
			return 0, fmt.Errorf("object %d: toc: paintCount: %w", state.idx+1, err)
		}

		state.tocPages = n
		guess += state.tocPages
	}
	// Iteration 2: renumber with the measured total, measure again, and keep
	// the final layout for painting.
	if guess > 0 {
		for _, state := range tocs {
			root, res, err := layoutTOC(ctx, font, state, entries, guess, glob, log)
			if err != nil {
				return 0, err
			}

			state.tocRoot, state.tocRes = root, res

			n, err := paintCount(ctx, doc.Policy(), res, state.geom)
			if err != nil {
				return 0, fmt.Errorf("object %d: toc: paintCount: %w", state.idx+1, err)
			}

			state.tocPages = n
		}
	}
	// Paint the final TOC pages into doc, keeping the painted result (its
	// Locations feed the link pass).
	total := 0

	for _, state := range tocs {
		state.start = doc.PageCount()
		painted := cloneResult(state.tocRes)

		if err := layout.PaintContext(ctx, doc, painted, paintOptions(state.geom)); err != nil {
			return 0, fmt.Errorf("object %d: toc: paint: %w", state.idx+1, err)
		}

		state.tocRes = painted
		total += state.tocPages
	}

	return total, nil
}
