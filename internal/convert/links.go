package convert

import (
	"strings"

	"gowkhtmltopdf/internal/cli"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/layout"
	"gowkhtmltopdf/internal/outline"
	"gowkhtmltopdf/internal/pdf"
)

// linkOpSkip is a sentinel OpKind for neutralized link ops. Paint's switch
// has no case for it, so a skipped op paints nothing; paginateOps and
// isSplittable likewise ignore it. The op keeps its slot in the Ops slice
// because the layout engine's box tree stores op indices (opStart/opEnd)
// that Paint relies on - removing entries would corrupt pagination.
const linkOpSkip = layout.OpKind(255)

// stripLinkURIs neutralizes external (http/https/mailto) link ops in place.
// Same-document fragment links (#id) are left for applyInternalLinks.
func stripLinkURIs(ops []layout.Op) []layout.Op {
	for i := range ops {
		if ops[i].Kind == layout.OpLinkURI && !strings.HasPrefix(ops[i].URI, "#") {
			ops[i].Kind = linkOpSkip
			ops[i].URI = ""
		}
	}
	return ops
}

// pageRect converts an element location (canvas coordinates, y-down) into a
// PDF annotation rect [x1 y1 x2 y2] with y-up coordinates.
func pageRect(loc layout.ElementLocation, g hfGeom) [4]float64 {
	x1 := g.marginLeft + loc.X
	yTop := g.pageH - g.marginTop - (loc.Y - float64(loc.Page)*g.contentH)
	yBot := g.pageH - g.marginTop - (loc.Y + loc.H - float64(loc.Page)*g.contentH)
	return [4]float64{x1, yBot, x1 + loc.W, yTop}
}

// destPoint converts a location's top-left corner into a PDF /XYZ
// destination (x, y-up).
func destPoint(loc layout.ElementLocation, g hfGeom) (float64, float64) {
	x := g.marginLeft + loc.X
	y := g.pageH - g.marginTop - (loc.Y - float64(loc.Page)*g.contentH)
	return x, y
}

// tocAnchorLocations indexes the element boxes of one laid-out TOC document
// by their data-wk-target attribute (the heading anchor of the entry).
func tocAnchorLocations(root *html.Node, res *layout.Result) map[string]layout.ElementLocation {
	out := map[string]layout.ElementLocation{}
	for _, loc := range res.Locations {
		if loc.Node == nil || loc.Node.Type != html.ElementNode {
			continue
		}
		if target := loc.Node.Attribute("data-wk-target"); target != "" {
			out[target] = loc
		}
	}
	return out
}

// applyTOCLinks wires the TOC link annotations once every page exists, using
// the final page indices.
//
// Forward links (toc.forward-links): each TOC entry div becomes a GoTo
// annotation to its heading's page and position. Back links (toc.back-links):
// each body heading becomes a GoTo annotation back to its TOC entry. Both are
// skipped when the anchor side is missing.
//
// Arbitrary same-document <a href="#frag"> links in body HTML are wired by
// applyInternalLinks: layout emits OpLinkURI with a "#frag" URI for inline
// anchors that have a paint box (text runs), and convert resolves them to
// GoTo destinations via element id / heading locations.
func applyTOCLinks(doc *pdf.Document, tocs []*objectState, bodies []*objectState, tocTotal int, headings []*outline.Heading) {
	if len(tocs) == 0 {
		return
	}
	byAnchor := map[string]*outline.Heading{}
	for _, h := range headings {
		if h.Anchor != "" {
			byAnchor[h.Anchor] = h
		}
	}

	for _, tr := range tocs {
		if !tr.toc.ForwardLinks && !tr.toc.BackLinks {
			continue
		}
		entryLocs := tocAnchorLocations(tr.tocRoot, tr.tocRes)
		for anchor, eloc := range entryLocs {
			h := byAnchor[anchor]
			if h == nil {
				continue
			}
			srcPage := doc.PageAt(tr.start + eloc.Page)
			if srcPage == nil {
				continue
			}
			if tr.toc.ForwardLinks {
				// TOC entry → heading
				destX, destY := headingDest(h, bodies)
				srcPage.AddLinkDest(pageRect(eloc, tr.geom), tocTotal+h.Page, destX, destY)
			}
			if tr.toc.BackLinks {
				// heading → TOC entry
				destPage := tr.start + eloc.Page
				destX, destY := destPoint(eloc, tr.geom)
				if st := bodyStateFor(bodies, h.Page); st != nil {
					if page := doc.PageAt(tocTotal + h.Page); page != nil {
						page.AddLinkDest(pageRect(locationOf(st, h.Node), st.geom), destPage, destX, destY)
					}
				}
			}
		}
	}
}

// headingDest returns the PDF destination (x, y-up) of a heading's top-left
// corner.
func headingDest(h *outline.Heading, bodies []*objectState) (float64, float64) {
	st := bodyStateFor(bodies, h.Page)
	if st == nil {
		return 0, 0
	}
	return destPoint(locationOf(st, h.Node), st.geom)
}

// locationOf finds an element's location in the object's layout result.
func locationOf(st *objectState, node *html.Node) layout.ElementLocation {
	for _, loc := range st.res.Locations {
		if loc.Node == node {
			return loc
		}
	}
	return layout.ElementLocation{}
}

// bodyIDDest is one body element destination keyed by id attribute.
type bodyIDDest struct {
	st  *objectState
	loc layout.ElementLocation
}

// buildBodyIDIndex maps element id attributes across body objects to their
// layout locations. Later occurrences of the same id overwrite earlier ones
// (matches the prior applyInternalLinks loop).
func buildBodyIDIndex(bodies []*objectState) map[string]bodyIDDest {
	idLoc := map[string]bodyIDDest{}
	for _, st := range bodies {
		if st == nil || st.res == nil {
			continue
		}
		for _, loc := range st.res.Locations {
			if loc.Node == nil {
				continue
			}
			if id := loc.Node.Attribute("id"); id != "" {
				idLoc[id] = bodyIDDest{st, loc}
			}
		}
	}
	return idLoc
}

// logicalDestPage is the pre-copy document page index of a body id dest.
func logicalDestPage(dest bodyIDDest, tocTotal int) int {
	return tocTotal + dest.st.offset + dest.loc.Page
}

// remapPageForCopies maps a pre-copy (logical) page index onto the final
// document page index in the same copy group as srcPage. Matches the
// ownership rules used by drawHeadersFooters after materializeCopies.
func remapPageForCopies(logicalDest, srcPage, logicalN, copies int, collate bool) int {
	if copies <= 1 || logicalN <= 0 {
		return logicalDest
	}
	if collate {
		copyIdx := srcPage / logicalN
		return copyIdx*logicalN + logicalDest
	}
	copyIdx := srcPage % copies
	return logicalDest*copies + copyIdx
}

// applyInternalLinks turns OpLinkURI ops whose URI is a same-document
// fragment (#id) into GoTo annotations. Destinations are element boxes with
// a matching id attribute. When LocalLinks is false, fragment ops are skipped.
func applyInternalLinks(doc *pdf.Document, bodies []*objectState, tocTotal int, cmd *cli.Command) {
	if doc == nil {
		return
	}
	_ = cmd
	idLoc := buildBodyIDIndex(bodies)
	for _, st := range bodies {
		if st == nil || st.res == nil || st.geom.contentH <= 0 {
			continue
		}
		useLocal := st.obj.LocalLinks
		for i := range st.res.Ops {
			op := &st.res.Ops[i]
			if op.Kind != layout.OpLinkURI || !strings.HasPrefix(op.URI, "#") {
				continue
			}
			frag := strings.TrimPrefix(op.URI, "#")
			op.Kind = linkOpSkip
			op.URI = ""
			if !useLocal || frag == "" {
				continue
			}
			dest, ok := idLoc[frag]
			if !ok {
				continue
			}
			srcPageIdx := tocTotal + st.offset + int(op.Y/st.geom.contentH)
			srcPage := doc.PageAt(srcPageIdx)
			if srcPage == nil {
				continue
			}
			srcLoc := layout.ElementLocation{
				Page: int(op.Y / st.geom.contentH),
				X:    op.X, Y: op.Y, W: op.W, H: op.H,
			}
			if srcLoc.H <= 0 {
				srcLoc.H = 10
			}
			if srcLoc.W <= 0 {
				srcLoc.W = 10
			}
			destPage := logicalDestPage(dest, tocTotal)
			dx, dy := destPoint(dest.loc, dest.st.geom)
			srcPage.AddLinkDest(pageRect(srcLoc, st.geom), destPage, dx, dy)
		}
	}
}
