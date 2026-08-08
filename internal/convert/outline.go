package convert

import (
	"io"
	"sort"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/layout"
	"gowkhtmltopdf/internal/line"
	"gowkhtmltopdf/internal/outline"
	"gowkhtmltopdf/internal/pdf"
	"gowkhtmltopdf/internal/settings"
)

// objectPlacement holds document page indices filled after paint / TOC
// assembly. Load-time fields stay on objectState; these mutate as the
// pipeline places the object into the final page set.
//
// Lifecycle:
//   - pages, offset: set in renderObject after body paint (pre-TOC reorder)
//   - start: set after TOC reorder (tocs: assembly positions; bodies: tocTotal+offset)
//   - TOC objects use tocPages for count; start is set at paint then rewritten
//     when TOC pages move to the front.
type objectPlacement struct {
	pages  int // body: number of painted pages
	offset int // body: body-global index of the first page (pre-TOC reorder)
	start  int // final document page index of the first page (post-reorder)
}

// objectState is the per-object state gathered during the loading loop and
// consumed by the TOC, outline, link and header/footer passes. Body objects
// carry headings/layout; TOC objects carry their effective TOC settings and
// the generated page layout.
//
// Fields are grouped by lifecycle:
//   - identity / settings: obj, idx, isTOC, header, footer, repl, toc, media, …
//   - load-time resources: registry, base, lp, imagesFn, geom, headerHTML/footerHTML
//   - body content: res, headings
//   - TOC content: tocPages, tocRoot, tocRes
//   - placement (post-paint): embedded objectPlacement (pages, offset, start)
type objectState struct {
	obj   *settings.PdfObject
	idx   int // 0-based object index (messages use idx+1)
	isTOC bool

	geom   hfGeom
	header settings.HeaderFooter
	footer settings.HeaderFooter
	repl   map[string]string // merged --replace map (header+footer)
	toc    settings.TableOfContent

	headerHTML *htmlHFLayout
	footerHTML *htmlHFLayout

	// registry is the opt-in font registry (--font-path / body @font-face)
	// shared with nested HTML HF layout so HF can resolve the same faces.
	// effectiveMargins returns the HF-extended registry; callers assign it
	// explicitly (see renderObject handshake).
	registry      *pdf.Registry
	resources     ResourceContext
	imagesEnabled bool

	base     string
	lp       settings.LoadPage
	media    string // layout CSS media ("print", "screen", …)
	imagesFn func(src string) ([]byte, error)
	doctitle string // <title> of the object document

	// body objects:
	res      *layout.Result
	headings []*outline.Heading

	// TOC objects:
	tocPages int
	tocRoot  *html.Node
	tocRes   *layout.Result

	// Document placement (pages/offset/start); see objectPlacement.
	objectPlacement
}

// docTitle extracts the <title> element text of a document, or "".
func docTitle(root *html.Node) string {
	if root == nil {
		return ""
	}

	if t := root.TextContentOf("title"); t != "" {
		return outline.CollapseWS(t)
	}

	return ""
}

// collectObjectHeadings gathers the h1..h6 elements of one painted object and
// matches them against the layout locations. Page stays object-local (from
// Lookup); DocPage is filled once in flatHeadings. Objects opted out of the
// outline (UseOutline/IncludeInOutline false) are dropped here;
// --exclude-from-outline is applied later via outline.BuildTree Options.Exclude.
func collectObjectHeadings(root *html.Node, res *layout.Result, _ int, _ settings.PdfGlobal, obj settings.PdfObject, _ io.Writer) []*outline.Heading { //nolint:lll // shared heading collection signature
	if !obj.UseOutline || !obj.IncludeInOutline {
		return nil
	}

	hsVal := outline.CollectHeadings(root)
	hsVal = outline.Lookup(hsVal, res.Locations)

	if len(hsVal) == 0 {
		return nil
	}

	return hsVal
}

// parseExcludeSelectors parses --exclude-from-outline selector strings into
// css.Selector values for outline.Options.Exclude (matching lives in outline).
func parseExcludeSelectors(specs []string, log io.Writer) []css.Selector {
	var out []css.Selector

	for _, s := range specs {
		sels, ok := css.ParseSelectors(s)
		if !ok || len(sels) == 0 {
			line.Emit(log, line.Warn, "ignoring invalid --exclude-from-outline selector %q", s)

			continue
		}

		out = append(out, sels...)
	}

	return out
}

// flatHeadings concatenates the per-object heading lists in object order,
// sets DocPage once (body-global), assigns stable synthetic anchors, and
// sorts by document page order.
func flatHeadings(bodies []*objectState) []*outline.Heading {
	var all []*outline.Heading

	for _, st := range bodies {
		for _, h := range st.headings {
			h.DocPage = st.offset + h.Page
			all = append(all, h)
		}
	}

	outline.AssignAnchors(all)
	// Document order by the explicit DocPage field. The object-local Page field
	// remains unchanged for page-local layout and header/footer semantics.
	sort.SliceStable(all, func(i, j int) bool {
		flag, buf := all[i], all[j]
		if flag.DocPage != buf.DocPage {
			return flag.DocPage < buf.DocPage
		}

		if flag.Y != buf.Y {
			return flag.Y < buf.Y
		}

		return flag.X < buf.X
	})

	return all
}

// bodyStateFor returns the body object whose page span contains the body
// page index (body-global / DocPage), or nil.
func bodyStateFor(bodies []*objectState, page int) *objectState {
	for _, st := range bodies {
		if page >= st.offset && page < st.offset+st.pages {
			return st
		}
	}

	return nil
}

// emitOutline converts the outline tree (canvas coordinates) into pdf.Outline
// nodes with final page refs and PDF (y-up) coordinates. The root is a
// container for the top-level headings, as pdf.Document.SetOutline expects.
// Tree headings retain object-local Page and carry document-global DocPage.
func emitOutline(doc *pdf.Document, tree *outline.Node, bodies []*objectState, tocTotal int) *pdf.Outline {
	root := &pdf.Outline{} //nolint:exhaustruct // intentional zero-value fields

	var conv func(n *outline.Node) *pdf.Outline

	conv = func(num *outline.Node) *pdf.Outline {
		hVal := num.Heading
		obj := &pdf.Outline{Title: hVal.Title} //nolint:exhaustruct // intentional zero-value fields

		docPage := hVal.DocPage
		if stVal := bodyStateFor(bodies, docPage); stVal != nil {
			locPage := docPage - stVal.offset
			obj.PageRef = doc.PageRef(tocTotal + docPage)
			loc := layout.ElementLocation{ //nolint:exhaustruct // intentional zero-value fields
				Page: locPage, X: hVal.X, Y: hVal.Y, W: hVal.W, H: hVal.H,
			}
			obj.X, obj.Y = stVal.geom.pdfXY(loc)
		}

		for _, c := range num.Children {
			obj.Children = append(obj.Children, conv(c))
		}

		return obj
	}
	for _, c := range tree.Children {
		root.Children = append(root.Children, conv(c))
	}

	return root
}
