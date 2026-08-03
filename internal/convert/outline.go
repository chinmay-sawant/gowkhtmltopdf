package convert

import (
	"fmt"
	"io"
	"sort"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/layout"
	"gowkhtmltopdf/internal/outline"
	"gowkhtmltopdf/internal/pdf"
	"gowkhtmltopdf/internal/settings"
)

// objectState is the per-object state gathered during the loading loop and
// consumed by the TOC, outline, link and header/footer passes. Body objects
// carry headings/layout; TOC objects carry their effective TOC settings and
// the generated page layout.
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

	base     string
	lp       settings.LoadPage
	imagesFn func(src string) ([]byte, error)
	doctitle string // <title> of the object document

	// body objects:
	res      *layout.Result
	pages    int
	offset   int // body-global page index of the first page
	headings []*outline.Heading

	// TOC objects:
	tocPages int
	tocRoot  *html.Node
	tocRes   *layout.Result

	// final document placement (filled after the page reorder):
	start int // document page index of the first page
}

// docTitle extracts the <title> element text of a document, or "".
func docTitle(root *html.Node) string {
	var walk func(n *html.Node) string
	walk = func(n *html.Node) string {
		if n.Type != html.ElementNode {
			return ""
		}
		if n.Name == "title" {
			return collapseText(n.TextContent())
		}
		for _, c := range n.Children {
			if t := walk(c); t != "" {
				return t
			}
		}
		return ""
	}
	return walk(root)
}

func collapseText(s string) string {
	out := make([]byte, 0, len(s))
	prevSpace := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			if !prevSpace {
				out = append(out, ' ')
				prevSpace = true
			}
			continue
		}
		out = append(out, c)
		prevSpace = false
	}
	return string(out)
}

// collectObjectHeadings gathers the h1..h6 elements of one painted object,
// matches them against the layout locations, and rebases their page numbers
// to the document page index given by offset. Objects opted out of the
// outline (UseOutline/IncludeInOutline false) and headings matching
// --exclude-from-outline selectors are dropped.
func collectObjectHeadings(root *html.Node, res *layout.Result, offset int, g settings.PdfGlobal, obj settings.PdfObject, log io.Writer) []*outline.Heading {
	if !obj.UseOutline || !obj.IncludeInOutline {
		return nil
	}
	hs := outline.CollectHeadings(root)
	hs = outline.Lookup(hs, res.Locations)
	if len(hs) == 0 {
		return nil
	}
	sels := parseExcludeSelectors(g.ExcludeFromOutline, log)
	kept := hs[:0]
	for _, h := range hs {
		if matchAnySelector(sels, h.Node) {
			continue
		}
		h.Page += offset
		kept = append(kept, h)
	}
	return kept
}

// parseExcludeSelectors parses the --exclude-from-outline selector strings.
func parseExcludeSelectors(specs []string, log io.Writer) []css.Selector {
	var out []css.Selector
	for _, s := range specs {
		sheet, err := css.Parse(s + "{}")
		if err != nil || sheet == nil || len(sheet.Rules) == 0 || len(sheet.Rules[0].Selectors) == 0 {
			fmt.Fprintf(log, "warning: ignoring invalid --exclude-from-outline selector %q\n", s)
			continue
		}
		out = append(out, sheet.Rules[0].Selectors[0])
	}
	return out
}

func matchAnySelector(sels []css.Selector, n *html.Node) bool {
	for _, s := range sels {
		if css.Match(s, n) {
			return true
		}
	}
	return false
}

// flatHeadings concatenates the per-object heading lists in object order,
// assigns stable synthetic anchors, and sorts by (page, y, x) — the order
// both the outline tree and the [section]/[subsection] lookup use.
func flatHeadings(bodies []*objectState) []*outline.Heading {
	var all []*outline.Heading
	for _, st := range bodies {
		all = append(all, st.headings...)
	}
	outline.AssignAnchors(all)
	sort.SliceStable(all, func(i, j int) bool {
		a, b := all[i], all[j]
		if a.Page != b.Page {
			return a.Page < b.Page
		}
		if a.Y != b.Y {
			return a.Y < b.Y
		}
		return a.X < b.X
	})
	return all
}

// bodyStateFor returns the body object whose page span contains the body
// page index, or nil.
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
func emitOutline(doc *pdf.Document, tree *outline.Node, bodies []*objectState, tocTotal int) *pdf.Outline {
	root := &pdf.Outline{}
	var conv func(n *outline.Node) *pdf.Outline
	conv = func(n *outline.Node) *pdf.Outline {
		h := n.Heading
		o := &pdf.Outline{Title: h.Title}
		if st := bodyStateFor(bodies, h.Page); st != nil {
			locPage := h.Page - st.offset
			o.PageRef = doc.PageRef(tocTotal + h.Page)
			o.X = st.geom.marginLeft + h.X
			// canvas y-down (page-relative offset included) → PDF y-up
			o.Y = st.geom.pageH - st.geom.marginTop - (h.Y - float64(locPage)*st.geom.contentH)
		}
		for _, c := range n.Children {
			o.Children = append(o.Children, conv(c))
		}
		return o
	}
	for _, c := range tree.Children {
		root.Children = append(root.Children, conv(c))
	}
	return root
}
