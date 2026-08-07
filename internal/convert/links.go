package convert

import (
	"strings"

	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/layout"
	"gowkhtmltopdf/internal/outline"
	"gowkhtmltopdf/internal/pdf"
)

// stripLinkURIs neutralizes external (http/https/mailto) link ops in place.
// Same-document fragment links (#id) are left for applyInternalLinks.
// Neutralization uses layout.DeactivateOp so every painter skips the op
// while keeping its slot (box tree op indices stay valid).
func stripLinkURIs(ops []layout.Op) []layout.Op {
	for i := range ops {
		if ops[i].Kind == layout.OpLinkURI && !strings.HasPrefix(ops[i].URI, "#") {
			layout.DeactivateOp(&ops[i])
		}
	}

	return ops
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

	for _, trVal := range tocs {
		if !trVal.toc.ForwardLinks && !trVal.toc.BackLinks {
			continue
		}

		entryLocs := tocAnchorLocations(trVal.tocRoot, trVal.tocRes)
		for anchor, eloc := range entryLocs {
			hVal := byAnchor[anchor]
			if hVal == nil {
				continue
			}

			srcPage := doc.PageAt(trVal.start + eloc.Page)
			if srcPage == nil {
				continue
			}

			docPage := hVal.DocPage

			if trVal.toc.ForwardLinks {
				// TOC entry → heading
				destX, destY := headingDest(hVal, bodies)
				srcPage.AddLinkDest(trVal.geom.pdfRect(eloc), tocTotal+docPage, destX, destY)
			}

			if trVal.toc.BackLinks {
				// heading → TOC entry
				destPage := trVal.start + eloc.Page
				destX, destY := trVal.geom.pdfXY(eloc)

				if st := bodyStateFor(bodies, docPage); st != nil {
					if page := doc.PageAt(tocTotal + docPage); page != nil {
						locPage := docPage - st.offset
						hLoc := layout.ElementLocation{Page: locPage, X: hVal.X, Y: hVal.Y, W: hVal.W, H: hVal.H} //nolint:exhaustruct // intentional zero-value fields
						page.AddLinkDest(st.geom.pdfRect(hLoc), destPage, destX, destY)
					}
				}
			}
		}
	}
}

// headingDest returns the PDF destination (x, y-up) of a heading's top-left
// corner using the heading's own geometry (no location scan).
func headingDest(hVal *outline.Heading, bodies []*objectState) (float64, float64) {
	docPage := hVal.DocPage

	state := bodyStateFor(bodies, docPage)
	if state == nil {
		return 0, 0
	}

	locPage := docPage - state.offset

	return state.geom.pdfXY(layout.ElementLocation{Page: locPage, X: hVal.X, Y: hVal.Y, W: hVal.W, H: hVal.H}) //nolint:exhaustruct // intentional zero-value fields
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

	for _, state := range bodies {
		if state == nil || state.res == nil {
			continue
		}

		for _, loc := range state.res.Locations {
			if loc.Node == nil {
				continue
			}

			if id := loc.Node.Attribute("id"); id != "" {
				idLoc[id] = bodyIDDest{state, loc}
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
// document page index in the same copy group as srcPage. Thin wrapper over
// pagePlan.Remap kept for unit tests.
func remapPageForCopies(logicalDest, srcPage, logicalN, copies int, collate bool) int {
	pp := &pagePlan{copies: copies, collate: collate, owners: make([]pageOwner, logicalN)} //nolint:exhaustruct // intentional zero-value fields

	return pp.Remap(logicalDest, srcPage)
}

// applyInternalLinks turns OpLinkURI ops whose URI is a same-document
// fragment (#id) into GoTo annotations. Destinations are element boxes with
// a matching id attribute. When LocalLinks is false, fragment ops are skipped.
func applyInternalLinks(doc *pdf.Document, bodies []*objectState, tocTotal int) {
	if doc == nil {
		return
	}

	idLoc := buildBodyIDIndex(bodies)

	for _, state := range bodies {
		if state == nil || state.res == nil || state.geom.contentH <= 0 {
			continue
		}

		useLocal := state.obj.LocalLinks

		for i := range state.res.Ops {
			oper := &state.res.Ops[i]
			if oper.Kind != layout.OpLinkURI || !strings.HasPrefix(oper.URI, "#") {
				continue
			}

			frag := strings.TrimPrefix(oper.URI, "#")
			layout.DeactivateOp(oper)

			if !useLocal || frag == "" {
				continue
			}

			dest, ok := idLoc[frag]
			if !ok {
				continue
			}

			srcPageIdx := tocTotal + state.offset + int(oper.Y/state.geom.contentH)

			srcPage := doc.PageAt(srcPageIdx)
			if srcPage == nil {
				continue
			}

			srcLoc := layout.ElementLocation{ //nolint:exhaustruct // intentional zero-value fields
				Page: int(oper.Y / state.geom.contentH),
				X:    oper.X, Y: oper.Y, W: oper.W, H: oper.H,
			}
			if srcLoc.H <= 0 {
				srcLoc.H = 10
			}

			if srcLoc.W <= 0 {
				srcLoc.W = 10
			}

			destPage := logicalDestPage(dest, tocTotal)
			dx, dy := dest.st.geom.pdfXY(dest.loc)
			srcPage.AddLinkDest(state.geom.pdfRect(srcLoc), destPage, dx, dy)
		}
	}
}
