package convert

import (
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

// stripLinkURIs neutralizes external link ops in place. Called before
// painting when the object's --disable-external-links (ExternalLinks=false)
// flag is set. Internal link ops are never emitted by the layout engine, so
// --disable-local-links has nothing to filter (documented in applyTOCLinks).
func stripLinkURIs(ops []layout.Op) []layout.Op {
	for i := range ops {
		if ops[i].Kind == layout.OpLinkURI {
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
// Arbitrary same-document <a href="#frag"> links in body HTML are NOT wired:
// the layout engine emits link ops only for external hrefs and inline <a>
// elements produce no boxes, so the source rect is unavailable. Named
// destinations for every heading still exist via the outline and the TOC
// links, which covers the report use case.
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
